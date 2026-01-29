# Systower Development Plan

## Overview
Systower - unified system stats + keep-awake tool untuk yambar.

## Current State
- Module: `github.com/reekoheek/systower`
- Binary output: `systower`
- Features: caffeine (keep awake) + battery monitor via DBus
- Output format: single tag `status|string|on/off`

## Target State
- Features: cpu, mem, disk, battery, caffeine
- Output format: multiple tags untuk yambar

## Tasks

### 1. Rename Module & Packages
- [x] `go.mod`: module `github.com/reekoheek/systower`
- [x] Rename package `internal/caffeine/` → `internal/systower/`
- [x] Update semua import paths
- [x] `Makefile`: output `systower`, cmd path `./cmd/systower`
- [ ] Refactor `internal/systower/` → `internal/caffeine/` (subfeature)

### 2. Poller Module (`internal/poller/poller.go`)
```go
type Stats struct {
    CPU  int  // 0-100%
    Mem  int  // used GB
    Disk int  // 0-100%
}

type Poller struct {}

func New(diskPath string) *Poller
func (p *Poller) Listen(interval time.Duration) <-chan Stats
```
- CPU: baca /proc/stat, hitung delta
- Mem: baca /proc/meminfo, hitung MemTotal - MemAvailable
- Disk: syscall.Statfs() pada diskPath

### 3. Battery Module (`internal/battery/battery.go`)
```go
type Info struct {
    Capacity int
    Status   string  // "charging", "discharging", "full"
}

type Monitor struct {}

func New(conn *dbus.Conn) (*Monitor, error)
func (m *Monitor) Listen() <-chan Info
```
- Listen DBus events dari UPower
- Emit ke channel saat battery state berubah

### 4. Caffeine Module (`internal/caffeine/caffeine.go`)
```go
type Monitor struct {}

func New(conn *dbus.Conn, adapter Adapter) *Monitor
func (m *Monitor) Listen() <-chan string  // "on" / "off"
func (m *Monitor) On()
func (m *Monitor) Off()
func (m *Monitor) Toggle()
func (m *Monitor) Status() string
```
- Listen DBus signals untuk status change
- Emit ke channel saat caffeine status berubah

### 5. Main CLI (`cmd/systower/main.go`)

Commands:
```
systower listen             # persistent mode, output tags ke stdout
systower caffeine on            # enable keep awake
systower caffeine off           # disable keep awake
systower caffeine toggle        # toggle keep awake
systower caffeine status        # print caffeine status
```

Listen Mode Output (yambar format):
```
cpu|int|45
mem|int|12
disk|int|67
battery|int|85
battery_status|string|discharging
caffeine|string|off

```
(empty line = commit transaction)

### 6. Listen Mode Logic
```go
func listen() {
    pollerCh := poller.New("/").Listen(5 * time.Second)
    batteryCh := battery.New(sysConn).Listen()
    caffeineCh := caffeine.New(conn, adapter).Listen()

    state := &State{}

    for {
        select {
        case s := <-pollerCh:
            state.CPU = s.CPU
            state.Mem = s.Mem
            state.Disk = s.Disk
        case b := <-batteryCh:
            state.Battery = b.Capacity
            state.BatteryStatus = b.Status
            // low battery logic here
        case c := <-caffeineCh:
            state.Caffeine = c
        }
        state.Print()
    }
}
```

### 7. Update Battery Logic
- [x] Keep existing low battery auto-off caffeine
- [x] Keep existing critical battery poweroff
- [x] Rename notification dari "Caffeine" → "Systower"

## File Structure (Target)
```
systower/
├── cmd/systower/main.go
├── internal/
│   ├── systower/
│   │   └── systower.go      # unified state manager
│   ├── cpu/
│   │   └── cpu.go
│   ├── mem/
│   │   └── mem.go
│   ├── disk/
│   │   └── disk.go
│   ├── battery/
│   │   └── battery.go       # existing
│   └── caffeine/
│       ├── caffeine.go
│       ├── adapter.go
│       ├── x11_adapter.go
│       └── wayland_adapter.go
├── go.mod
├── go.sum
└── Makefile
```

## Testing
```bash
# Build
make build

# Test one-shot
./systower stats

# Test listen mode
./systower listen

# Test caffeine commands
./systower caffeine on
./systower caffeine status
./systower caffeine toggle
./systower caffeine off
```
