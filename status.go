package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

type AppStatus struct {
	State   string    `json:"state"` // idle | running | ok | error
	Message string    `json:"message"`
	At      time.Time `json:"at"`
}

func statusPath(cfg Config) string {
	return filepath.Join(filepath.Dir(cfg.StateFilePath), "status.json")
}

func writeStatus(cfg Config, state, msg string) {
	p := statusPath(cfg)
	_ = os.MkdirAll(filepath.Dir(p), 0o755)
	s := AppStatus{State: state, Message: msg, At: time.Now()}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return
	}
	tmp := p + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return
	}
	_ = os.Rename(tmp, p)
}

func readStatus(cfg Config) AppStatus {
	data, err := os.ReadFile(statusPath(cfg))
	if err != nil {
		return AppStatus{State: "idle", Message: "waiting", At: time.Now()}
	}
	var s AppStatus
	if json.Unmarshal(data, &s) != nil || s.State == "" {
		return AppStatus{State: "idle", Message: "waiting", At: time.Now()}
	}
	return s
}
