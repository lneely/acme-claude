package acme

import (
	"fmt"

	"9fans.net/go/acme"
)

// WindowExists returns true if a window with a given name exists, false if not
func WindowExists(name string) (bool, error) {
	wis, err := acme.Windows()
	if err != nil {
		return false, err
	}
	for _, wi := range wis {
		if wi.Name == name {
			return true, nil
		}
	}
	return false, nil
}

// WindowOpen shows the window with a given name if it exists, otherwise creates it with the given tag.
func WindowOpen(name string) (*acme.Win, error) {
	if w := acme.Show(name); w != nil {
		return w, nil
	}
	w, err := acme.New()
	if err != nil {
		return nil, fmt.Errorf("failed to open prompt window: %w", err)
	}

	if err := w.Name(name); err != nil {
		w.Del(true)
		return nil, fmt.Errorf("failed to set prompt window name: %w", err)
	}
	return w, nil
}

// TagSet sets the tag of the window with the given name.
func TagSet(name string, tag string) error {
	w, err := WindowOpen(name)
	if err != nil {
		return fmt.Errorf("failed to get window [%s]: %w", name, err)
	}
	if _, err := w.Write("tag", []byte(tag)); err != nil {
		w.Del(true)
		return fmt.Errorf("failed to set tag on [%s]: %w", name, err)
	}
	return nil
}

// BodyWrite writes the bytes s to the window body at address addr.
// addr is what follows the ":" in a file address, e.g., "$" for EOF,
// "1" for BOF, "," for the full file, "1,20" for lines 1-20.
func BodyWrite(name string, addr string, s []byte) error {
	w, err := WindowOpen(name)
	if err != nil {
		return fmt.Errorf("failed to append to body, window=[%s]: %w", name, err)
	}

	if err = w.Addr(addr); err != nil {
		return fmt.Errorf("failed to seek to end of body, window=[%s]: %w", name, err)
	}

	_, err = w.Write("data", s)
	if err != nil {
		return fmt.Errorf("failed to append data, window=[%s]: %w", name, err)
	}

	return nil
}

// BodyRead reads the full contents from the window of the given name.
func BodyRead(name string) ([]byte, error) {
	w, err := WindowOpen(name)
	if err != nil {
		return []byte(""), err
	}

	data, err := w.ReadAll("body")
	if err != nil {
		return []byte(""), fmt.Errorf("failed to read from window [%s]: %w", name, err)
	}

	return data, nil
}
