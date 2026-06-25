package state

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

// State persists which servers were running so the daemon can restore them
// on restart (e.g. after a system reboot with systemd).

const stateFile = "/var/lib/opd/daemon-state.json"

type RunningState struct {
	Servers []string `json:"servers"` // list of server IDs that were running
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

func Save(ids []string) error {
	mu.Lock()
	defer mu.Unlock()

	_ = os.MkdirAll(filepath.Dir(stateFile), 0755)

	f, err := os.Create(stateFile)
	if err != nil {
		return err
	}
	defer f.Close()

	return json.NewEncoder(f).Encode(RunningState{Servers: ids})
}
