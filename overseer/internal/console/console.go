// SPDX-License-Identifier: AGPL-3.0-only
package console

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"sunder/overseer/internal/whisper"
)

// Registry is the shard store the console reads from
type Registry interface {
	ListShards() []whisper.Shard
	ShardByID(id string) (whisper.Shard, bool)
}

var errExit = errors.New("exit")
var errInterrupted = errors.New("interrupted")

// Console is the Overseer REPL
type Console struct {
	reg     Registry
	in      *bufio.Reader
	out     io.Writer
	events  <-chan string
	user    string
	voice   string
	grasped string
	editor  *termEditor
}

// New builds a console over a registry and an input stream
func New(reg Registry, in io.Reader, out io.Writer, events <-chan string) *Console {
	user := osUser()
	c := &Console{
		reg:    reg,
		in:     bufio.NewReader(in),
		out:    out,
		events: events,
		user:   user,
		voice:  "verse",
	}
	if f, ok := in.(*os.File); ok && isCharDevice(f) {
		if e, err := newTermEditor(f, out); err == nil {
			c.editor = e
		}
	}
	return c
}

// Run reads lines until exit or EOF
func (c *Console) Run() error {
	if c.editor != nil {
		defer c.editor.Close()
	}
	for {
		c.drain()
		line, err := c.readLine()
		if err != nil {
			c.drain()
			if errors.Is(err, io.EOF) {
				return nil
			}
			if errors.Is(err, errInterrupted) {
				continue
			}
			return err
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if err := c.exec(line); err != nil {
			if errors.Is(err, errExit) {
				return nil
			}
			c.errln(err)
		}
	}
}

func (c *Console) readLine() (string, error) {
	if c.editor != nil {
		return c.editor.ReadLine(c.prompt())
	}
	fmt.Fprintf(c.out, "%s", c.prompt())
	line, err := c.in.ReadString('\n')
	if err != nil {
		return "", err
	}
	return line, nil
}

func (c *Console) prompt() string {
	return fmt.Sprintf("%s@sunder:~$ ", c.user)
}

func (c *Console) exec(line string) error {
	if strings.HasPrefix(line, "!") {
		return c.passthrough(strings.TrimSpace(strings.TrimPrefix(line, "!")))
	}
	fields := strings.Fields(line)
	if len(fields) == 0 {
		return nil
	}
	word, args := fields[0], fields[1:]
	switch word {
	case "help", "?":
		c.printHelp()
	case "clear":
		c.outln("\x1b[2J\x1b[H")
	case "exit", "quit":
		return errExit
	case "tone":
		return c.doVoice(args)
	case "shards":
		return c.doShards(args)
	case "grasp":
		return c.doGrasp(args)
	case "breath":
		return c.doBreath(args)
	default:
		return fmt.Errorf("unknown word: %s (try help)", word)
	}
	return nil
}

func (c *Console) doShards(args []string) error {
	shards := c.reg.ListShards()
	if len(shards) == 0 {
		if c.voice == "verse" {
			c.outln("No shards. The blade is whole, for now.")
		} else if c.voice == "plain" {
			c.outln("no shards")
		}
		return nil
	}
	c.outln(fmt.Sprintf("%-16s %-10s %-6s %-8s %-12s %s", "id", "user", "os", "arch", "state", "last breath"))
	for _, s := range shards {
		state, _ := describe(s, time.Now())
		c.outln(fmt.Sprintf("%-16s %-10s %-6s %-8s %-12s %s",
			s.ID, s.User, s.OS, s.Arch, state, ago(time.Since(s.LastBreath))))
	}
	return nil
}

func (c *Console) doGrasp(args []string) error {
	if len(args) == 0 {
		if c.grasped == "" {
			c.outln("usage: grasp <id|host>")
			return nil
		}
		c.outln(fmt.Sprintf("hooked: %s", c.grasped))
		return nil
	}
	target := args[0]
	for _, s := range c.reg.ListShards() {
		if strings.HasPrefix(s.ID, target) || strings.HasPrefix(s.Hostname, target) {
			c.grasped = s.ID
			if c.voice == "verse" {
				c.outln("You have its ear.")
			} else if c.voice == "plain" {
				c.outln(fmt.Sprintf("grasped %s", s.ID))
			}
			return nil
		}
	}
	return fmt.Errorf("no shard answers to that name")
}

func (c *Console) doBreath(args []string) error {
	id := c.grasped
	if len(args) > 0 {
		id = args[0]
	}
	if id == "" {
		return fmt.Errorf("grasp a shard first, or name one")
	}
	s, ok := c.reg.ShardByID(id)
	if !ok {
		return fmt.Errorf("no shard answers to that name")
	}
	state, flavor := describe(s, time.Now())
	if c.voice == "verse" {
		c.outln(flavor)
	} else if c.voice == "plain" {
		c.outln(state)
	}
	c.outln(fmt.Sprintf("last breath %s ago (cadence %ds)", ago(time.Since(s.LastBreath)), s.CadenceS))
	return nil
}

func (c *Console) doVoice(args []string) error {
	if len(args) != 1 {
		c.outln("usage: tone verse|plain|quiet")
		return nil
	}
	switch args[0] {
	case "verse", "plain", "quiet":
		c.voice = args[0]
	default:
		return fmt.Errorf("unknown voice: %s (verse, plain, quiet)", args[0])
	}
	return nil
}

func (c *Console) passthrough(cmdline string) error {
	if cmdline == "" {
		return fmt.Errorf("say what to run: !<command>")
	}
	cmd := exec.Command("sh", "-c", cmdline)
	cmd.Stdout = c.out
	cmd.Stderr = c.out
	return cmd.Run()
}

func (c *Console) printHelp() {
	c.outln("words of this console:")
	c.outln("  help, ?     this reference")
	c.outln("  shards      list deployed shards and their breath")
	c.outln("  grasp       hook into a shard by id or host")
	c.outln("  breath      heartbeat check of the grasped shard")
	c.outln("  tone        console voice: verse, plain, quiet")
	c.outln("  clear       clear the console")
	c.outln("  exit        leave the console")
	c.outln("  !<command>  run locally through the real shell")
}

// describe maps registry age onto the state ladder from the design
func describe(s whisper.Shard, now time.Time) (state, flavor string) {
	cad := time.Duration(s.CadenceS) * time.Second
	if cad <= 0 {
		cad = 5 * time.Second
	}
	age := now.Sub(s.LastBreath)
	switch {
	case age <= 2*cad:
		return "breathing", "It breathes, shallow and patient."
	case age <= 5*cad:
		return "silent", "It holds its breath."
	default:
		return "dark", "Silence. The wire is cold."
	}
}

func (c *Console) drain() {
	if c.events == nil {
		return
	}
	for {
		select {
		case msg := <-c.events:
			if c.voice != "quiet" {
				c.outln(msg)
			}
		default:
			return
		}
	}
}

func (c *Console) outln(s string) {
	fmt.Fprintln(c.out, s)
}

func (c *Console) errln(err error) {
	fmt.Fprintln(c.out, err)
}

func ago(d time.Duration) string {
	if d < time.Second {
		return "0s"
	}
	return d.Round(time.Second).String()
}

func osUser() string {
	if u := os.Getenv("USER"); u != "" {
		return u
	}
	return "you"
}

func isCharDevice(f *os.File) bool {
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}
