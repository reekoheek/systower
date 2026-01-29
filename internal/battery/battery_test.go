//go:build linux

package battery

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMonitor_Info(t *testing.T) {
	// skip if no battery present
	batPath := filepath.Join(sysfsPath, "BAT0")
	if _, err := os.Stat(batPath); os.IsNotExist(err) {
		t.Skip("no battery present, skipping test")
	}

	m := &Monitor{}
	stats, err := m.readFromSysfs()
	if err != nil {
		t.Fatalf("failed to read battery info: %v", err)
	}

	t.Logf("status:   %s", stats.Status)
	t.Logf("capacity: %d%%", stats.Percent)

	validStatuses := map[string]bool{
		"charging":     true,
		"discharging":  true,
		"full":         true,
		"not charging": true,
	}
	if !validStatuses[stats.Status] {
		t.Errorf("unexpected status: %s", stats.Status)
	}

	if stats.Percent < 0 || stats.Percent > 100 {
		t.Errorf("capacity should be 0-100, got %d", stats.Percent)
	}
}
