package config

import (
	"fmt"
	"os"
	"path/filepath"
)

// writeFileAtomic follows existing symlinks, preserving their target's mode.
// Errors after rename explicitly report that replacement has already occurred.
func writeFileAtomic(path string, data []byte, perm os.FileMode) error {
	if info, err := os.Lstat(path); err == nil {
		if info.Mode()&os.ModeSymlink != 0 {
			resolved, err := filepath.EvalSymlinks(path)
			if err != nil {
				return fmt.Errorf("resolve config symlink: %w", err)
			}
			path = resolved
		}
		info, err = os.Stat(path)
		if err != nil {
			return fmt.Errorf("stat config target: %w", err)
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("config target must be a regular file")
		}
		perm = info.Mode().Perm()
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("stat config: %w", err)
	}
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".config-*.tmp")
	if err != nil {
		return fmt.Errorf("create temp config file: %w", err)
	}
	defer os.Remove(tmp.Name())
	defer tmp.Close()
	if _, err := tmp.Write(data); err != nil {
		return fmt.Errorf("write temp config file: %w", err)
	}
	if err := tmp.Chmod(perm); err != nil {
		return fmt.Errorf("chmod temp config file: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		return fmt.Errorf("sync temp config file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp config file: %w", err)
	}
	if err := os.Rename(tmp.Name(), path); err != nil {
		return fmt.Errorf("rename temp config file: %w", err)
	}
	d, err := os.Open(dir)
	if err != nil {
		return fmt.Errorf("config replaced; open directory for sync: %w", err)
	}
	defer d.Close()
	if err := d.Sync(); err != nil {
		return fmt.Errorf("config replaced; sync directory: %w", err)
	}
	if err := d.Close(); err != nil {
		return fmt.Errorf("config replaced; close directory: %w", err)
	}
	return nil
}
