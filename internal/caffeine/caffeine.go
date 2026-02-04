package caffeine

import (
	"context"
	"os"
	"path/filepath"

	"golang.org/x/sys/unix"
)

type Stats struct {
	Active bool
}

func (s Stats) String() string {
	if s.Active {
		return "on"
	}
	return "off"
}

type Caffeine struct {
	adapter    Adapter
	statusFile string
}

func New(adapter Adapter) *Caffeine {
	return &Caffeine{
		adapter:    adapter,
		statusFile: filepath.Join(getRuntimeDir(), "caffeine.status"),
	}
}

func (c *Caffeine) Read() Stats {
	data, err := os.ReadFile(c.statusFile)
	if err != nil {
		return Stats{}
	}
	return Stats{Active: string(data) == "1"}
}

func (c *Caffeine) On() {
	c.adapter.On()
	os.WriteFile(c.statusFile, []byte("1"), 0644)
}

func (c *Caffeine) Off() {
	c.adapter.Off()
	os.WriteFile(c.statusFile, []byte("0"), 0644)
}

func (c *Caffeine) Toggle() {
	if c.Read().Active {
		c.Off()
	} else {
		c.On()
	}
}

func (c *Caffeine) Listen(ctx context.Context) (<-chan struct{}, error) {
	// Initialize status file if not exists
	if _, err := os.Stat(c.statusFile); os.IsNotExist(err) {
		os.WriteFile(c.statusFile, []byte("0"), 0644)
	}

	fd, err := unix.InotifyInit1(unix.IN_CLOEXEC)
	if err != nil {
		return nil, err
	}

	_, err = unix.InotifyAddWatch(fd, c.statusFile, unix.IN_MODIFY|unix.IN_CREATE)
	if err != nil {
		unix.Close(fd)
		return nil, err
	}

	ch := make(chan struct{}, 1)

	go func() {
		defer close(ch)
		defer unix.Close(fd)

		cancelFd, err := unix.Eventfd(0, unix.EFD_NONBLOCK)
		if err != nil {
			return
		}
		defer unix.Close(cancelFd)

		context.AfterFunc(ctx, func() {
			unix.Write(cancelFd, []byte{1, 0, 0, 0, 0, 0, 0, 0})
		})

		buf := make([]byte, 1024)
		fds := []unix.PollFd{
			{Fd: int32(fd), Events: unix.POLLIN},
			{Fd: int32(cancelFd), Events: unix.POLLIN},
		}

		for {
			n, err := unix.Poll(fds, -1)
			if err != nil {
				if err == unix.EINTR {
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
				unix.Read(fd, buf)
				select {
				case ch <- struct{}{}:
				default:
				}
			}
		}
	}()

	return ch, nil
}

func DetectAdapter() Adapter {
	if !isWayland() {
		return NewX11Adapter()
	}
	lockfile := filepath.Join(getRuntimeDir(), "swayidle.lock")
	return NewWaylandAdapter(lockfile)
}

func isWayland() bool {
	return os.Getenv("WAYLAND_DISPLAY") != "" || os.Getenv("XDG_SESSION_TYPE") == "wayland"
}

func getRuntimeDir() string {
	if dir := os.Getenv("XDG_RUNTIME_DIR"); dir != "" {
		return dir
	}
	return "/tmp"
}
