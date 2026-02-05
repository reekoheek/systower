package poller

import (
	"time"

	"golang.org/x/sys/unix"
)

// EventType identifies the source of an event
type EventType int

const (
	EventTicker EventType = iota
	EventBacklight
	EventCaffeine
	EventVolume
)

// Event represents a polling event
type Event struct {
	Type EventType
}

// FdSource represents a file descriptor source for polling
type FdSource struct {
	Fd   int
	Type EventType
}

// Poller consolidates multiple fd sources into a single poll call
type Poller struct {
	sources  []FdSource
	timerFd  int
	cancelFd int
	pollFds  []unix.PollFd
	fdToType map[int]EventType
	events   []Event // reusable buffer for events
}

// New creates a unified poller with the given timer interval
func New(interval time.Duration) (*Poller, error) {
	// Create timerfd for polling interval
	timerFd, err := unix.TimerfdCreate(unix.CLOCK_MONOTONIC, unix.TFD_NONBLOCK|unix.TFD_CLOEXEC)
	if err != nil {
		return nil, err
	}

	// Create eventfd for cancellation
	cancelFd, err := unix.Eventfd(0, unix.EFD_NONBLOCK|unix.EFD_CLOEXEC)
	if err != nil {
		unix.Close(timerFd)
		return nil, err
	}

	// Set timer interval
	spec := unix.ItimerSpec{
		Interval: unix.NsecToTimespec(int64(interval)),
		Value:    unix.NsecToTimespec(int64(interval)),
	}
	if err := unix.TimerfdSettime(timerFd, 0, &spec, nil); err != nil {
		unix.Close(timerFd)
		unix.Close(cancelFd)
		return nil, err
	}

	return &Poller{
		timerFd:  timerFd,
		cancelFd: cancelFd,
		fdToType: make(map[int]EventType),
		events:   make([]Event, 0, 8),
	}, nil
}

// AddSource adds a file descriptor source to poll
func (p *Poller) AddSource(fd int, eventType EventType) {
	p.sources = append(p.sources, FdSource{Fd: fd, Type: eventType})
	p.fdToType[fd] = eventType
}

// Init builds the pollFds array. Must be called after all AddSource calls.
func (p *Poller) Init() {
	// Build poll fds array: [timerfd, cancelfd, ...sources]
	p.pollFds = make([]unix.PollFd, 2+len(p.sources))
	p.pollFds[0] = unix.PollFd{Fd: int32(p.timerFd), Events: unix.POLLIN}
	p.pollFds[1] = unix.PollFd{Fd: int32(p.cancelFd), Events: unix.POLLIN}
	for i, src := range p.sources {
		p.pollFds[2+i] = unix.PollFd{Fd: int32(src.Fd), Events: unix.POLLIN}
	}
}

// Cancel signals the poller to stop
func (p *Poller) Cancel() {
	unix.Write(p.cancelFd, []byte{1, 0, 0, 0, 0, 0, 0, 0})
}

// Close releases resources
func (p *Poller) Close() {
	unix.Close(p.timerFd)
	unix.Close(p.cancelFd)
}

// Wait blocks until events are available, returns events slice and cancelled flag
// The returned slice is reused between calls, copy if needed.
func (p *Poller) Wait() ([]Event, bool) {
	buf := make([]byte, 8) // For reading timerfd expirations

	for {
		n, err := unix.Poll(p.pollFds, -1)
		if err != nil {
			if err == unix.EINTR {
				continue
			}
			return nil, true // Error, treat as cancelled
		}
		if n == 0 {
			continue
		}

		// Check cancel fd
		if p.pollFds[1].Revents&unix.POLLIN != 0 {
			return nil, true
		}

		// Collect events
		p.events = p.events[:0]

		// Check timer fd
		if p.pollFds[0].Revents&unix.POLLIN != 0 {
			unix.Read(p.timerFd, buf) // Clear the timer
			p.events = append(p.events, Event{Type: EventTicker})
		}

		// Check source fds
		for i := range p.sources {
			if p.pollFds[2+i].Revents&unix.POLLIN != 0 {
				p.events = append(p.events, Event{Type: p.sources[i].Type})
			}
		}

		if len(p.events) > 0 {
			return p.events, false
		}
	}
}
