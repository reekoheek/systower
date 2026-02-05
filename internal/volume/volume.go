package volume

import (
	"os"
	"os/exec"
	"strconv"
	"strings"

	"golang.org/x/sys/unix"
)

type Stats struct {
	Percent int
	Muted   bool
}

type Monitor struct {
	cmd       *exec.Cmd
	fd        int
	buf       []byte
	lineBuf   []byte
	lastStats Stats
}

func New() *Monitor {
	return &Monitor{
		fd:  -1,
		buf: make([]byte, 1024),
	}
}

func (m *Monitor) Read() Stats {
	// Use wpctl to get volume (works with PipeWire and PulseAudio)
	out, err := exec.Command("wpctl", "get-volume", "@DEFAULT_AUDIO_SINK@").Output()
	if err != nil {
		return Stats{}
	}

	return parseWpctlVolume(string(out))
}

func parseWpctlVolume(output string) Stats {
	// Format: "Volume: 0.82" or "Volume: 0.82 [MUTED]"
	output = strings.TrimSpace(output)
	parts := strings.Fields(output)
	if len(parts) < 2 {
		return Stats{}
	}

	vol, err := strconv.ParseFloat(parts[1], 64)
	if err != nil {
		return Stats{}
	}

	muted := strings.Contains(output, "[MUTED]")

	return Stats{
		Percent: int(vol * 100),
		Muted:   muted,
	}
}

// InitFd starts pactl subscribe and returns the stdout fd
// Caller is responsible for calling Close()
func (m *Monitor) InitFd() (int, error) {
	m.cmd = exec.Command("pactl", "subscribe")
	stdout, err := m.cmd.StdoutPipe()
	if err != nil {
		return -1, err
	}

	if err := m.cmd.Start(); err != nil {
		return -1, err
	}

	// Get raw fd from stdout pipe
	file := stdout.(*os.File)
	m.fd = int(file.Fd())

	// Set non-blocking
	if err := unix.SetNonblock(m.fd, true); err != nil {
		m.cmd.Process.Kill()
		m.cmd.Wait()
		return -1, err
	}

	// Read initial state
	m.lastStats = m.Read()

	return m.fd, nil
}

// Drain reads available data and returns true if volume changed
func (m *Monitor) Drain() bool {
	if m.fd < 0 {
		return false
	}

	// Read all available data
	for {
		n, err := unix.Read(m.fd, m.buf)
		if err != nil || n == 0 {
			break // EAGAIN or no data
		}
		m.lineBuf = append(m.lineBuf, m.buf[:n]...)
	}

	// Process complete lines
	changed := false
	for {
		idx := indexByte(m.lineBuf, '\n')
		if idx < 0 {
			break
		}

		line := string(m.lineBuf[:idx])
		m.lineBuf = m.lineBuf[idx+1:]

		// Only react to sink events (volume/mute changes)
		// Format: "Event 'change' on sink #123"
		if strings.Contains(line, " on sink ") {
			stats := m.Read()
			if stats != m.lastStats {
				m.lastStats = stats
				changed = true
			}
		}
	}

	return changed
}

// Stats returns the last known stats
func (m *Monitor) Stats() Stats {
	return m.lastStats
}

// Close stops the pactl process
func (m *Monitor) Close() {
	if m.cmd != nil && m.cmd.Process != nil {
		m.cmd.Process.Kill()
		m.cmd.Wait()
	}
	m.fd = -1
}

func indexByte(b []byte, c byte) int {
	for i, v := range b {
		if v == c {
			return i
		}
	}
	return -1
}
