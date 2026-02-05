//go:build linux

package backlight

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"golang.org/x/sys/unix"
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
	maxBrightness int
	netlinkFd     int
	buf           []byte
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

	return &Monitor{
		device:        device,
		maxBrightness: maxBrightness,
		netlinkFd:     -1,
		buf:           make([]byte, 4096),
	}, nil
}

func (m *Monitor) Read() Stats {
	brightness, _ := readInt(filepath.Join(sysBacklight, m.device, "brightness"))
	return Stats{
		Brightness:    brightness,
		MaxBrightness: m.maxBrightness,
	}
}

// InitFd creates and returns the netlink fd for udev events
// Caller is responsible for closing the fd
func (m *Monitor) InitFd() (int, error) {
	fd, err := syscall.Socket(syscall.AF_NETLINK, syscall.SOCK_RAW|syscall.SOCK_NONBLOCK|syscall.SOCK_CLOEXEC, syscall.NETLINK_KOBJECT_UEVENT)
	if err != nil {
		return -1, err
	}

	addr := syscall.SockaddrNetlink{
		Family: syscall.AF_NETLINK,
		Groups: 2, // UDEV_MONITOR_UDEV (processed events)
	}
	if err := syscall.Bind(fd, &addr); err != nil {
		syscall.Close(fd)
		return -1, err
	}

	m.netlinkFd = fd
	return fd, nil
}

// Drain reads and processes pending events, returns true if backlight changed
func (m *Monitor) Drain() bool {
	if m.netlinkFd < 0 {
		return false
	}

	changed := false
	for {
		n, _, err := syscall.Recvfrom(m.netlinkFd, m.buf, unix.MSG_DONTWAIT)
		if err != nil {
			break
		}
		if m.isBacklightEvent(m.buf[:n]) {
			changed = true
		}
	}
	return changed
}

// Close closes the netlink fd
func (m *Monitor) Close() {
	if m.netlinkFd >= 0 {
		syscall.Close(m.netlinkFd)
		m.netlinkFd = -1
	}
}

// isBacklightEvent checks if the udev event is for our backlight device
func (m *Monitor) isBacklightEvent(data []byte) bool {
	hasBacklight := indexOf(data, []byte("SUBSYSTEM=backlight")) >= 0
	hasDevice := indexOf(data, []byte("/"+m.device+"\x00")) >= 0 ||
		indexOf(data, []byte("/"+m.device+"@")) >= 0
	return hasBacklight && hasDevice
}

func indexOf(data, pattern []byte) int {
	for i := 0; i <= len(data)-len(pattern); i++ {
		match := true
		for j := 0; j < len(pattern); j++ {
			if data[i+j] != pattern[j] {
				match = false
				break
			}
		}
		if match {
			return i
		}
	}
	return -1
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
