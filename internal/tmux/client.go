package tmux

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

type Client struct{}

func New() *Client { return &Client{} }

func (c *Client) Output(args ...string) (string, error) {
	cmd := exec.Command("tmux", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		if stderr.Len() > 0 {
			return "", fmt.Errorf("tmux %s: %s", strings.Join(args, " "), strings.TrimSpace(stderr.String()))
		}
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func (c *Client) Run(args ...string) error {
	cmd := exec.Command("tmux", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if stderr.Len() > 0 {
			return fmt.Errorf("tmux %s: %s", strings.Join(args, " "), strings.TrimSpace(stderr.String()))
		}
		return err
	}
	return nil
}
