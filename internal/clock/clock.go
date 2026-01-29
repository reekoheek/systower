package clock

import "time"

type Stats struct {
	Time time.Time
}

func (s Stats) Date() string {
	return s.Time.Format("Mon, 2006-01-02")
}

func (s Stats) TimeStr() string {
	return s.Time.Format("15:04:05")
}

type Monitor struct{}

func New() *Monitor {
	return &Monitor{}
}

func (r *Monitor) Read() Stats {
	return Stats{Time: time.Now()}
}

func (r *Monitor) Listen(interval time.Duration) <-chan Stats {
	ch := make(chan Stats)
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		defer close(ch)

		ch <- r.Read()

		for range ticker.C {
			ch <- r.Read()
		}
	}()
	return ch
}
