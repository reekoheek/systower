package caffeine

import (
	"os"
	"os/exec"
)

type WaylandAdapter struct {
	lockfile string
}

func NewWaylandAdapter(lockfile string) *WaylandAdapter {
	return &WaylandAdapter{lockfile: lockfile}
}

func (a *WaylandAdapter) On() {
	exec.Command("pkill", "swayidle").Run()
	os.WriteFile(a.lockfile, []byte{}, 0644)
}

func (a *WaylandAdapter) Off() {
	exec.Command("pkill", "swayidle").Run()
	a.startSwayidle()
	os.Remove(a.lockfile)
}


func (a *WaylandAdapter) startSwayidle() {
	cmd := exec.Command("swayidle", "-w",
		"timeout", "180", "swaylock -f",
		"timeout", "600", "systemctl suspend",
		"before-sleep", "swaylock -f",
	)
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.Start()
}
