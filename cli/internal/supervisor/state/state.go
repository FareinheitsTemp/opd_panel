package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// State persists which servers were running so the daemon can restore them
// on restart (e.g. after a system reboot with systemd).

const stateFile = "/var/lib/opd/daemon-state.json"

type RunningState struct {
	Servers []string `json:"servers"`
}

var mu sync.Mutex

func Load() ([]string, error) {
	mu.Lock()
	defer mu.Unlock()

	f, err := os.Open(stateFile)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var s RunningState
	if err := json.NewDecoder(f).Decode(&s); err != nil {
		return nil, err
	}
	return s.Servers, nil
}

// Save writes state atomically: write to a temp file, then rename.
// BUG FIX #7: the previous os.Create approach truncated the file first —
// if the daemon crashed mid-write, the state file would be empty/corrupt.
// os.Rename is atomic on POSIX systems, so the old state is always valid
// until the new one is fully written.
func Save(ids []string) error {
	mu.Lock()
	defer mu.Unlock()

	dir := filepath.Dir(stateFile)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	tmp := stateFile + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}

	if err := json.NewEncoder(f).Encode(RunningState{Servers: ids}); err != nil {
		f.Close()
		os.Remove(tmp)
		return err
	}
	if err := f.Sync(); err != nil {
		f.Close()
		os.Remove(tmp)
		return err
	}
	f.Close()

	if err := os.Rename(tmp, stateFile); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("atomic rename state: %w", err)
	}
	return nil
}
