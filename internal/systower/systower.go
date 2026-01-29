package systower

import (
	"fmt"
	"os"
	"time"

	"github.com/reekoheek/systower/internal/battery"
	"github.com/reekoheek/systower/internal/caffeine"
	"github.com/reekoheek/systower/internal/notification"
	"github.com/reekoheek/systower/internal/sys"
)

type Systower struct {
	caff   *caffeine.Caffeine
	notif  *notification.Notification
	batMon *battery.BatteryMonitor
	sysMgr *sys.Sys
}

func New(caff *caffeine.Caffeine, notif *notification.Notification, batMon *battery.BatteryMonitor, sysMgr *sys.Sys) *Systower {
	return &Systower{caff: caff, notif: notif, batMon: batMon, sysMgr: sysMgr}
}

func (s *Systower) Daemon() {
	batCh, err := s.batMon.Listen()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	caffCh, err := s.caff.Listen()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	for {
		select {
		case info, ok := <-batCh:
			if !ok {
				return
			}
			if info.Status == "charging" {
				continue
			}
			if s.caff.Status() == "on" && info.Capacity < 15 {
				s.caff.Off()
				s.notif.Send("Low battery, disable caffeine")
			}
			if info.Capacity < 5 {
				s.notif.Send("Battery almost drained, have a nice sleep")
				time.Sleep(5 * time.Second)
				s.sysMgr.Poweroff()
			}
		case status, ok := <-caffCh:
			if !ok {
				return
			}
			fmt.Printf("status|string|%s\n\n", status)
		}
	}
}
