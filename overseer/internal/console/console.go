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

const taskWait = 90 * time.Second

// Registry is the shard store the console reads from
type Registry interface {
	ListShards() []whisper.Shard
	ShardByID(id string) (whisper.Shard, bool)
}

// Tasker queues Words for a Shard and waits on their answers
type Tasker interface {
	QueueTask(shardID, word string, args map[string]string) (string, error)
	WaitTaskResult(taskID string, timeout time.Duration) (whisper.TaskResult, error)
}

// Controller is everything the console reads and speaks through
type Controller interface {
	Registry
	Tasker
}

var errExit = errors.New("exit")
var errInterrupted = errors.New("interrupted")

// Console is the Overseer REPL
type Console struct {
	reg     Registry
	tasks   Tasker
	in      *bufio.Reader
	out     io.Writer
	events  <-chan string
	user    string
	voice   string
	grasped string
	editor  *termEditor
}

// New builds a console over a controller and an input stream
func New(ctrl Controller, in io.Reader, out io.Writer, events <-chan string) *Console {
	user := osUser()
	c := &Console{
		reg:    ctrl,
		tasks:  ctrl,
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
	case "utter", "gaze", "unfold", "pulse", "anatomy":
		return c.doWord(word, args)
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

// doWord queues a Word for the grasped shard and prints its answer
func (c *Console) doWord(word string, args []string) error {
	if c.grasped == "" {
		return errors.New("grasp a shard first, then speak")
	}
	a, err := wordArgs(word, args)
	if err != nil {
		return err
	}
	taskID, err := c.tasks.QueueTask(c.grasped, word, a)
	if err != nil {
		return err
	}
	res, err := c.tasks.WaitTaskResult(taskID, taskWait)
	if err != nil {
		if c.voice == "verse" {
			c.outln("It does not answer.")
			return nil
		}
		return err
	}
	if !res.OK {
		if res.Result == "" {
			c.outln("failed")
		} else {
			c.outln("failed: " + res.Result)
		}
		return nil
	}
	if res.Result == "" {
		c.outln("(no output)")
	} else {
		c.outln(res.Result)
	}
	return nil
}

// wordArgs maps a Word's command line onto the wire args
func wordArgs(word string, args []string) (map[string]string, error) {
	a := make(map[string]string)
	switch word {
	case "utter":
		var cmd []string
		for i := 0; i < len(args); i++ {
			switch args[i] {
			case "-t":
				if i+1 >= len(args) {
					return nil, errors.New("usage: utter [-t seconds] <command>")
				}
				i++
				a["t"] = args[i]
			default:
				cmd = append(cmd, args[i])
			}
		}
		if len(cmd) == 0 {
			return nil, errors.New("usage: utter [-t seconds] <command>")
		}
		a["command"] = strings.Join(cmd, " ")
	case "gaze":
		if len(args) > 0 {
			a["path"] = args[0]
		}
	case "unfold":
		var pos []string
		for i := 0; i < len(args); i++ {
			switch args[i] {
			case "-o", "-n", "-t":
				if i+1 >= len(args) {
					return nil, errors.New("usage: unfold <path> [-o offset] [-n bytes] [-t text|hex|base64]")
				}
				i++
				a[strings.TrimPrefix(args[i-1], "-")] = args[i]
			default:
				pos = append(pos, args[i])
			}
		}
		if len(pos) == 0 {
			return nil, errors.New("usage: unfold <path> [-o offset] [-n bytes] [-t text|hex|base64]")
		}
		a["path"] = pos[0]
	case "pulse", "anatomy":
		// nothing to parse yet
	}
	return a, nil
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
	c.outln("  utter       run a command on the grasped shard")
	c.outln("  gaze        list a directory on the grasped shard")
	c.outln("  unfold      read a file on the grasped shard")
	c.outln("  pulse       list processes on the grasped shard")
	c.outln("  anatomy     system information on the grasped shard")
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
