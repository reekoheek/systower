package caffeine

import (
	"os/exec"
	"strings"
)

type X11Adapter struct{}

func NewX11Adapter() *X11Adapter {
	return &X11Adapter{}
}

func (a *X11Adapter) On() {
	exec.Command("xset", "s", "0").Run()
	exec.Command("xset", "dpms", "0", "0", "0").Run()
}

func (a *X11Adapter) Off() {
	exec.Command("xset", "s", "180").Run()
	exec.Command("xset", "dpms", "0", "0", "180").Run()
}

func (a *X11Adapter) Status() string {
	out, err := exec.Command("xset", "q").Output()
	if err != nil {
		return "off"
	}

	for _, line := range strings.Split(string(out), "\n") {
		if strings.Contains(line, "Off:") {
			fields := strings.Fields(line)
			for i, f := range fields {
				if f == "Off:" && i+1 < len(fields) {
					if fields[i+1] == "0" {
						return "on"
					}
					return "off"
				}
			}
		}
	}

	return "off"
}
