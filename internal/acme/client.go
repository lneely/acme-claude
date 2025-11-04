package acme

import (
	"fmt"
	"os"
	"path/filepath"

	"9fans.net/go/acme"
)

type Client struct {
	windows map[string]*acme.Win
}

func NewClient() *Client {
	return &Client{
		windows: make(map[string]*acme.Win),
	}
}

func (c *Client) CreateWindow(name string) (*acme.Win, error) {
	if win, exists := c.windows[name]; exists {
		return win, nil
	}

	win, err := acme.New()
	if err != nil {
		return nil, fmt.Errorf("failed to create acme window: %w", err)
	}

	err = win.Name(name)
	if err != nil {
		win.Del(true)
		return nil, fmt.Errorf("failed to set window name: %w", err)
	}

	c.windows[name] = win
	return win, nil
}

func (c *Client) SetWindowTag(name string, tag string) error {
	win, err := c.OpenWindow(name)
	if err != nil {
		return err
	}

	_, err = win.Write("tag", []byte(tag))
	if err != nil {
		return fmt.Errorf("failed to write to window tag: %w", err)
	}

	return nil
}

func (c *Client) OpenWindow(name string) (*acme.Win, error) {
	if win, exists := c.windows[name]; exists {
		return win, nil
	}

	wins, err := acme.Windows()
	if err != nil {
		return nil, fmt.Errorf("failed to list acme windows: %w", err)
	}

	for _, info := range wins {
		if info.Name == name {
			win, err := acme.Open(info.ID, nil)
			if err != nil {
				return nil, fmt.Errorf("failed to open window %s: %w", name, err)
			}
			c.windows[name] = win
			return win, nil
		}
	}

	return c.CreateWindow(name)
}

func (c *Client) WriteToWindow(name string, content string) error {
	win, err := c.OpenWindow(name)
	if err != nil {
		return err
	}

	_, err = win.Write("body", []byte(content))
	if err != nil {
		return fmt.Errorf("failed to write to window %s: %w", name, err)
	}

	return nil
}

func (c *Client) AppendToWindow(name string, content string) error {
	win, err := c.OpenWindow(name)
	if err != nil {
		return err
	}

	err = win.Addr("$")
	if err != nil {
		return fmt.Errorf("failed to seek to end of window %s: %w", name, err)
	}

	_, err = win.Write("data", []byte(content))
	if err != nil {
		return fmt.Errorf("failed to append to window %s: %w", name, err)
	}

	return nil
}

func (c *Client) ReadFromWindow(name string) (string, error) {
	win, err := c.OpenWindow(name)
	if err != nil {
		return "", err
	}

	data, err := win.ReadAll("body")
	if err != nil {
		return "", fmt.Errorf("failed to read from window %s: %w", name, err)
	}

	return string(data), nil
}

func (c *Client) ClearWindow(name string) error {
	win, err := c.OpenWindow(name)
	if err != nil {
		return err
	}

	err = win.Addr(",")
	if err != nil {
		return fmt.Errorf("failed to select all in window %s: %w", name, err)
	}

	_, err = win.Write("data", []byte(""))
	if err != nil {
		return fmt.Errorf("failed to clear window %s: %w", name, err)
	}

	return nil
}

func GetCurrentWorkingDir() string {
	pwd, err := os.Getwd()
	if err != nil {
		return ""
	}
	return pwd
}

func GetWindowName(suffix string) string {
	pwd := GetCurrentWorkingDir()
	return filepath.Join(pwd, suffix)
}

func (c *Client) Close() {
	for _, win := range c.windows {
		win.CloseFiles()
	}
}