package notifier

import "time"

type Notifier struct {
	ch      chan struct{}
	trigger chan struct{}
}

func New(debounce time.Duration) *Notifier {
	n := &Notifier{
		ch:      make(chan struct{}, 1),
		trigger: make(chan struct{}, 1),
	}
	go n.run(debounce)
	return n
}

func (n *Notifier) run(debounce time.Duration) {
	var timer *time.Timer
	var timerC <-chan time.Time

	for {
		select {
		case <-n.trigger:
			if timer == nil {
				timer = time.NewTimer(debounce)
				timerC = timer.C
			} else {
				if !timer.Stop() {
					select {
					case <-timerC:
					default:
					}
				}
				timer.Reset(debounce)
			}
		case <-timerC:
			select {
			case n.ch <- struct{}{}:
			default:
			}
			timer = nil
			timerC = nil
		}
	}
}

func (n *Notifier) Notify() {
	select {
	case n.trigger <- struct{}{}:
	default:
	}
}

func (n *Notifier) Notified() <-chan struct{} {
	return n.ch
}
