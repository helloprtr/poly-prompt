package app

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
)

type SubmitConfirmer interface {
	ConfirmSubmit(target string) (bool, error)
}

type TTYConfirmer struct {
	output  io.Writer
	openTTY func() (io.ReadCloser, error)
}

func NewTTYConfirmer(output io.Writer) *TTYConfirmer {
	return &TTYConfirmer{
		output: output,
		openTTY: func() (io.ReadCloser, error) {
			return os.Open("/dev/tty")
		},
	}
}

func NewTTYConfirmerForTesting(output io.Writer, openTTY func() (io.ReadCloser, error)) *TTYConfirmer {
	return &TTYConfirmer{
		output:  output,
		openTTY: openTTY,
	}
}

func (c *TTYConfirmer) ConfirmSubmit(target string) (bool, error) {
	if c.openTTY == nil {
		return false, fmt.Errorf("submit confirmation is unavailable")
	}
	tty, err := c.openTTY()
	if err != nil {
		return false, fmt.Errorf("open tty for submit confirmation: %w", err)
	}
	defer tty.Close()

	_, _ = fmt.Fprintf(c.output, "Submit pasted prompt to %s now? [y/N]: ", target)
	reader := bufio.NewReader(tty)
	line, err := reader.ReadString('\n')
	if err != nil && err != io.EOF {
		return false, fmt.Errorf("read submit confirmation: %w", err)
	}
	answer := strings.ToLower(strings.TrimSpace(line))
	return answer == "y" || answer == "yes", nil
}
