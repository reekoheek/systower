package volume

import (
	"bufio"
	"context"
	"os/exec"
	"strconv"
	"strings"
)

type Stats struct {
	Percent int
	Muted   bool
}

type Monitor struct {
	cmd *exec.Cmd
}

func New() (*Monitor, error) {
	return &Monitor{}, nil
}

func (m *Monitor) Close() {
	if m.cmd != nil && m.cmd.Process != nil {
		m.cmd.Process.Kill()
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

func (m *Monitor) Listen(ctx context.Context, callback func(Stats)) error {
	// Start pactl subscribe to listen for events
	m.cmd = exec.CommandContext(ctx, "pactl", "subscribe")
	stdout, err := m.cmd.StdoutPipe()
	if err != nil {
		return err
	}

	if err := m.cmd.Start(); err != nil {
		return err
	}

	go func() {
		// Send initial state
		lastStats := m.Read()
		callback(lastStats)

		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()
			// Only react to sink events (volume/mute changes)
			// Format: "Event 'change' on sink #123"
			if strings.Contains(line, " on sink ") {
				stats := m.Read()
				if stats != lastStats {
					lastStats = stats
					callback(stats)
				}
			}
		}
	}()

	return nil
}
