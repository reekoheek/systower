package caffeine

import (
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
	inotifyFd  int
	buf        []byte
}

func New(adapter Adapter) *Caffeine {
	return NewWithStatusFile(adapter, filepath.Join(getRuntimeDir(), "caffeine.status"))
}

func NewWithStatusFile(adapter Adapter, statusFile string) *Caffeine {
	return &Caffeine{
		adapter:    adapter,
		statusFile: statusFile,
		inotifyFd:  -1,
		buf:        make([]byte, 1024),
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

// InitFd creates and returns the inotify fd for status file changes
// Caller is responsible for closing the fd
func (c *Caffeine) InitFd() (int, error) {
	// Initialize status file if not exists
	if _, err := os.Stat(c.statusFile); os.IsNotExist(err) {
		os.WriteFile(c.statusFile, []byte("0"), 0644)
	}

	fd, err := unix.InotifyInit1(unix.IN_NONBLOCK | unix.IN_CLOEXEC)
	if err != nil {
		return -1, err
	}

	_, err = unix.InotifyAddWatch(fd, c.statusFile, unix.IN_MODIFY|unix.IN_CREATE)
	if err != nil {
		unix.Close(fd)
		return -1, err
	}

	c.inotifyFd = fd
	return fd, nil
}

// Drain reads and clears pending inotify events
func (c *Caffeine) Drain() {
	if c.inotifyFd < 0 {
		return
	}
	// Read all pending events to clear the fd
	for {
		_, err := unix.Read(c.inotifyFd, c.buf)
		if err != nil {
			break
		}
	}
}

// Close closes the inotify fd
func (c *Caffeine) Close() {
	if c.inotifyFd >= 0 {
		unix.Close(c.inotifyFd)
		c.inotifyFd = -1
	}
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
