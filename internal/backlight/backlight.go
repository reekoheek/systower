//go:build linux

package backlight

import (
	"bytes"
	"context"
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
	}, nil
}

func (m *Monitor) Read() Stats {
	brightness, _ := readInt(filepath.Join(sysBacklight, m.device, "brightness"))
	return Stats{
		Brightness:    brightness,
		MaxBrightness: m.maxBrightness,
	}
}

func (m *Monitor) Listen(ctx context.Context) <-chan Stats {
	ch := make(chan Stats, 1)

	go func() {
		defer close(ch)

		// Create netlink socket for udev events
		fd, err := syscall.Socket(syscall.AF_NETLINK, syscall.SOCK_RAW, syscall.NETLINK_KOBJECT_UEVENT)
		if err != nil {
			return
		}
		defer syscall.Close(fd)

		addr := syscall.SockaddrNetlink{
			Family: syscall.AF_NETLINK,
			Groups: 2, // UDEV_MONITOR_UDEV (processed events)
		}
		if err := syscall.Bind(fd, &addr); err != nil {
			return
		}

		// Create eventfd for cancellation
		cancelFd, err := unix.Eventfd(0, unix.EFD_NONBLOCK)
		if err != nil {
			return
		}
		defer syscall.Close(cancelFd)

		context.AfterFunc(ctx, func() {
			unix.Write(cancelFd, []byte{1, 0, 0, 0, 0, 0, 0, 0})
		})

		ch <- m.Read()

		buf := make([]byte, 4096)
		fds := []unix.PollFd{
			{Fd: int32(fd), Events: unix.POLLIN},
			{Fd: int32(cancelFd), Events: unix.POLLIN},
		}

		for {
			n, err := unix.Poll(fds, -1)
			if err != nil {
				if err == syscall.EINTR {
					continue
				}
				return
			}
			if n == 0 {
				continue
			}

			if fds[1].Revents&unix.POLLIN != 0 {
				return
			}

			if fds[0].Revents&unix.POLLIN != 0 {
				n, _, err := syscall.Recvfrom(fd, buf, 0)
				if err != nil {
					continue
				}
				if m.isBacklightEvent(buf[:n]) {
					ch <- m.Read()
				}
			}
		}
	}()

	return ch
}


// isBacklightEvent checks if the udev event is for our backlight device
func (m *Monitor) isBacklightEvent(data []byte) bool {
	// Udev event format: key=value pairs separated by null bytes
	// We look for SUBSYSTEM=backlight and our device path
	hasBacklight := bytes.Contains(data, []byte("SUBSYSTEM=backlight"))
	hasDevice := bytes.Contains(data, []byte("/"+m.device+"\x00")) ||
		bytes.Contains(data, []byte("/"+m.device+"@"))
	return hasBacklight && hasDevice
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
