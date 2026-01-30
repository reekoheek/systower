//go:build linux

package backlight

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/pilebones/go-udev/netlink"
)

const sysBacklight = "/sys/class/backlight"

type Stats struct {
	Brightness    int
	MaxBrightness int
}

func (s Stats) Percent() int {
	if s.MaxBrightness == 0 {
		return 0
	}
	return s.Brightness * 100 / s.MaxBrightness
}

type Monitor struct {
	device        string
	conn          *netlink.UEventConn
	maxBrightness int
}

func New(device string) (*Monitor, error) {
	if device == "" {
		var err error
		device, err = detectDevice()
		if err != nil {
			return nil, err
		}
	}

	maxBrightness, err := readInt(filepath.Join(sysBacklight, device, "max_brightness"))
	if err != nil {
		return nil, err
	}

	conn := &netlink.UEventConn{}
	if err := conn.Connect(netlink.UdevEvent); err != nil {
		return nil, err
	}

	return &Monitor{
		device:        device,
		conn:          conn,
		maxBrightness: maxBrightness,
	}, nil
}

func (m *Monitor) Read() Stats {
	brightness, _ := readInt(filepath.Join(sysBacklight, m.device, "brightness"))
	return Stats{
		Brightness:    brightness,
		MaxBrightness: m.maxBrightness,
	}
}

func (m *Monitor) Listen(ctx context.Context, callback func(Stats)) error {
	// Send initial state
	callback(m.Read())

	events := make(chan netlink.UEvent)
	errs := make(chan error)
	quit := m.conn.Monitor(events, errs, nil)

	go func() {
		defer close(quit)
		for {
			select {
			case <-ctx.Done():
				return
			case ev := <-events:
				if m.isBacklightEvent(ev) {
					callback(m.Read())
				}
			case <-errs:
				// Ignore errors, continue monitoring
			}
		}
	}()

	return nil
}

func (m *Monitor) Close() {
	if m.conn != nil {
		m.conn.Close()
	}
}

func (m *Monitor) isBacklightEvent(ev netlink.UEvent) bool {
	if ev.Env["SUBSYSTEM"] != "backlight" {
		return false
	}
	// Check if it's our device
	return strings.HasSuffix(ev.KObj, "/"+m.device)
}

func detectDevice() (string, error) {
	entries, err := os.ReadDir(sysBacklight)
	if err != nil {
		return "", err
	}
	if len(entries) == 0 {
		return "", os.ErrNotExist
	}
	return entries[0].Name(), nil
}

func readInt(path string) (int, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(strings.TrimSpace(string(data)))
}
