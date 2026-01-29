package main

import (
	"fmt"
	"os"
	"time"

	"github.com/reekoheek/systower/internal/battery"
	"github.com/reekoheek/systower/internal/caffeine"
	"github.com/reekoheek/systower/internal/notification"
	"github.com/reekoheek/systower/internal/sys"
)

type Listen struct {
	caff   *caffeine.Caffeine
	notif  *notification.Notification
	batMon *battery.BatteryMonitor
	sysMgr *sys.Sys
}

func NewListen(caff *caffeine.Caffeine, notif *notification.Notification, batMon *battery.BatteryMonitor, sysMgr *sys.Sys) *Listen {
	return &Listen{caff: caff, notif: notif, batMon: batMon, sysMgr: sysMgr}
}

func (u *Listen) Execute() {
	batCh, err := u.batMon.Listen()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	go func() {
		for info := range batCh {
			if info.Status == "charging" {
				continue
			}

			if u.caff.Status() == "on" && info.Capacity < 15 {
				u.caff.Off()
				u.notif.Send("Systower", "Low battery, disable systower")
			}

			if info.Capacity < 5 {
				u.notif.Send("Systower", "Battery almost drained, have a nice sleep")
				time.Sleep(5 * time.Second)
				u.sysMgr.Poweroff()
			}
		}
	}()

	if err := u.caff.Listen(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
