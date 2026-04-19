package util

import (
	"os"
	"path/filepath"
)

const (
	stateEnv = "TMXLOG_STATE_FILE"
)

func StateFilePath() (string, error) {
	if p := os.Getenv(stateEnv); p != "" {
		return p, nil
	}
	base := os.Getenv("XDG_STATE_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		base = filepath.Join(home, ".local", "state")
	}
	return filepath.Join(base, "tmxlog", "state.json"), nil
}
