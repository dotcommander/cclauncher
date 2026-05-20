package config

import (
	"fmt"
	"os"
	"path/filepath"
)

// writeFileAtomic writes data to path durably and atomically: it creates a temp
// file in the same directory, writes, syncs, and closes it, then os.Rename's it
// into place. The temp file is fsync'd before the rename so a kernel or power
// crash cannot leave a zero-length or partially-written config file. The parent
// directory is also synced after the rename so the directory entry itself is
// durable. The final file mode is set to perm.
func writeFileAtomic(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".config-*.tmp")
	if err != nil {
		return fmt.Errorf("create temp config file: %w", err)
	}
	tmpName := tmp.Name()

	// On any failure, remove the temp file.
	cleanup := func() { _ = os.Remove(tmpName) }

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		cleanup()
		return fmt.Errorf("write temp config file: %w", err)
	}

	if err := tmp.Chmod(perm); err != nil {
		_ = tmp.Close()
		cleanup()
		return fmt.Errorf("chmod temp config file: %w", err)
	}

	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		cleanup()
		return fmt.Errorf("sync temp config file: %w", err)
	}

	if err := tmp.Close(); err != nil {
		cleanup()
		return fmt.Errorf("close temp config file: %w", err)
	}

	if err := os.Rename(tmpName, path); err != nil {
		cleanup()
		return fmt.Errorf("rename temp config file: %w", err)
	}

	// Sync the directory so the rename entry is durable. The file is already in
	// place at this point, so a dir-sync failure is not fatal.
	if d, err := os.Open(dir); err == nil {
		_ = d.Sync()
		_ = d.Close()
	}

	return nil
}
