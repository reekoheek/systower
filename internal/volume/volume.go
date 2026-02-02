package volume

import (
	"context"

	"github.com/lawl/pulseaudio"
)

type Stats struct {
	Percent int
	Muted   bool
}

type Monitor struct {
	client *pulseaudio.Client
}

func New() (*Monitor, error) {
	client, err := pulseaudio.NewClient()
	if err != nil {
		return nil, err
	}

	return &Monitor{client: client}, nil
}

func (m *Monitor) Close() {
	if m.client != nil {
		m.client.Close()
	}
}

func (m *Monitor) Read() Stats {
	vol, err := m.client.Volume()
	if err != nil {
		return Stats{}
	}

	muted, err := m.client.Mute()
	if err != nil {
		return Stats{}
	}

	return Stats{
		Percent: int(vol * 100),
		Muted:   muted,
	}
}

func (m *Monitor) Listen(ctx context.Context, callback func(Stats)) error {
	updates, err := m.client.Updates()
	if err != nil {
		return err
	}

	go func() {
		// Send initial state
		lastStats := m.Read()
		callback(lastStats)

		for {
			select {
			case <-ctx.Done():
				return
			case <-updates:
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
