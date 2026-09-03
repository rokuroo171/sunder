// SPDX-License-Identifier: AGPL-3.0-only
//go:build linux

package console

import (
	"fmt"
	"io"
	"os"
	"strings"
	"syscall"
	"unsafe"
)

type termEditor struct {
	in     *os.File
	out    io.Writer
	old    *syscall.Termios
	hist   []string
	histAt int
	draft  string
}

func newTermEditor(in *os.File, out io.Writer) (*termEditor, error) {
	old, err := rawMode(int(in.Fd()))
	if err != nil {
		return nil, err
	}
	return &termEditor{in: in, out: out, old: old}, nil
}

func (e *termEditor) Close() {
	if e.old != nil {
		restoreMode(int(e.in.Fd()), e.old)
		e.old = nil
	}
}

// ReadLine edits one line on the terminal and returns it on Enter
func (e *termEditor) ReadLine(prompt string) (string, error) {
	line := make([]byte, 0, 64)
	pos := 0
	e.histAt = len(e.hist)
	e.draft = ""
	io.WriteString(e.out, prompt)
	for {
		b, err := e.readByte()
		if err != nil {
			return "", err
		}
		switch b {
		case '\r', '\n':
			io.WriteString(e.out, "\r\n")
			if s := strings.TrimSpace(string(line)); s != "" {
				e.hist = append(e.hist, s)
			}
			return string(line), nil
		case 3: // ^C cancels the line
			io.WriteString(e.out, "^C\r\n")
			e.draft = ""
			return "", errInterrupted
		case 4: // ^D on an empty line ends the session
			if len(line) == 0 {
				return "", io.EOF
			}
		case 127, 8: // backspace
			if pos > 0 {
				line = append(line[:pos-1], line[pos:]...)
				pos--
				e.redraw(prompt, line, pos)
			}
		case '\x1b':
			seq, err := e.readEscape()
			if err != nil {
				return "", err
			}
			switch seq {
			case "up":
				if e.histAt == len(e.hist) {
					e.draft = string(line)
				}
				if e.histAt > 0 {
					e.histAt--
					line = []byte(e.hist[e.histAt])
					pos = len(line)
					e.redraw(prompt, line, pos)
				}
			case "down":
				if e.histAt < len(e.hist) {
					e.histAt++
					if e.histAt == len(e.hist) {
						line = []byte(e.draft)
					} else {
						line = []byte(e.hist[e.histAt])
					}
					pos = len(line)
					e.redraw(prompt, line, pos)
				}
			case "left":
				if pos > 0 {
					pos--
					io.WriteString(e.out, "\x1b[D")
				}
			case "right":
				if pos < len(line) {
					pos++
					io.WriteString(e.out, "\x1b[C")
				}
			}
		default:
			if b >= 0x20 {
				line = append(line, 0)
				copy(line[pos+1:], line[pos:])
				line[pos] = b
				pos++
				if pos == len(line) {
					e.out.Write([]byte{b})
				} else {
					e.redraw(prompt, line, pos)
				}
			}
		}
	}
}

func (e *termEditor) readByte() (byte, error) {
	var buf [1]byte
	n, err := e.in.Read(buf[:])
	if err != nil {
		return 0, err
	}
	if n == 0 {
		return 0, io.ErrUnexpectedEOF
	}
	return buf[0], nil
}

// readEscape consumes an escape sequence after ESC and names it
func (e *termEditor) readEscape() (string, error) {
	b, err := e.readByte()
	if err != nil {
		return "", err
	}
	if b != '[' && b != 'O' {
		return "", nil
	}
	b, err = e.readByte()
	if err != nil {
		return "", err
	}
	switch b {
	case 'A':
		return "up", nil
	case 'B':
		return "down", nil
	case 'C':
		return "right", nil
	case 'D':
		return "left", nil
	}
	return "", nil
}

func (e *termEditor) redraw(prompt string, line []byte, pos int) {
	io.WriteString(e.out, "\r\x1b[K"+prompt+string(line))
	if pos < len(line) {
		fmt.Fprintf(e.out, "\x1b[%dD", len(line)-pos)
	}
}

func rawMode(fd int) (*syscall.Termios, error) {
	var old syscall.Termios
	if _, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), syscall.TCGETS, uintptr(unsafe.Pointer(&old))); errno != 0 {
		return nil, errno
	}
	raw := old
	raw.Lflag &^= syscall.ECHO | syscall.ICANON | syscall.ISIG
	raw.Iflag &^= syscall.IXON
	raw.Cc[syscall.VMIN] = 1
	raw.Cc[syscall.VTIME] = 0
	if _, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), syscall.TCSETS, uintptr(unsafe.Pointer(&raw))); errno != 0 {
		return nil, errno
	}
	return &old, nil
}

func restoreMode(fd int, old *syscall.Termios) {
	_, _, _ = syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), syscall.TCSETS, uintptr(unsafe.Pointer(old)))
}
