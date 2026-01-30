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

type Reader struct{}

func New() *Reader {
	return &Reader{}
}

func (r *Reader) Read() Stats {
	return Stats{Time: time.Now()}
}
