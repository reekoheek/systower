package caffeine

import "os/exec"

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
