package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestWorkerUsesExistingSharingSetting(t *testing.T) {
	before := *dataDir
	*dataDir = t.TempDir()
	defer func() { *dataDir = before }()
	if !defaultConfig().IsPublic {
		t.Fatal("privacy default changed")
	}
	for _, value := range []bool{false, true} {
		bytes, _ := json.Marshal(map[string]any{"Enable": false, "IsPublic": value})
		os.WriteFile(filepath.Join(*dataDir, "config.json"), bytes, 0600)
		cfg := loadDaemonConfig()
		m, err := measurementForConfig([]string{"4", "6"}, cfg)
		if err != nil {
			t.Fatal(err)
		}
		if m.SessionRequest().IsPublic == nil || *m.SessionRequest().IsPublic != value {
			t.Fatal("existing config not propagated")
		}
	}
}
