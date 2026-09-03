// SPDX-License-Identifier: AGPL-3.0-only
//go:build !linux

package console

import (
	"errors"
	"io"
	"os"
)

type termEditor struct{}

func (e *termEditor) Close() {}

func (e *termEditor) ReadLine(prompt string) (string, error) {
	return "", io.EOF
}

func newTermEditor(in *os.File, out io.Writer) (*termEditor, error) {
	return nil, errors.New("terminal editor only on linux")
}
