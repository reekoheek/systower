package clock

import "time"

type Stats struct {
	Unix int64
}

func (s Stats) Day() string {
	return time.Unix(s.Unix, 0).Format("Mon")
}

func (s Stats) Date() string {
	return time.Unix(s.Unix, 0).Format("2006-01-02")
}

func (s Stats) Time() string {
	return time.Unix(s.Unix, 0).Format("15:04")
}

type Reader struct{}

func New() *Reader {
	return &Reader{}
}

func (r *Reader) Read() Stats {
	return Stats{Unix: time.Now().Unix()}
}
