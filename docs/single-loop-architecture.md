# Single Blocking Loop Architecture

## Problem Statement

Dengan arsitektur saat ini (multi-goroutine + channels), systower menghasilkan ~10 wakeups/s bahkan dengan `GOMAXPROCS=1`. Ini disebabkan oleh overhead Go runtime:

| Setting | Context Switches/s |
|---------|-------------------|
| GOMAXPROCS=default | ~15 |
| GOMAXPROCS=1 | ~10 |
| Target | ~1-2 |

### Sumber Wakeup Saat Ini

1. **Go runtime sysmon** - monitoring goroutine setiap 10-20ms
2. **Channel operations** - scheduler involvement tiap send/receive
3. **time.Timer/Ticker** - Go timer heap management
4. **Multiple goroutines** - scheduling overhead

## Solution: Single `unix.Poll()` Loop

Ganti semua goroutines dan channels dengan single blocking `unix.Poll()` call yang menunggu semua event sources sekaligus.

### Architecture Diagram

```
Current (Multi-goroutine):

  +----------+  +----------+  +----------+
  | backlight|  | battery  |  | volume   |  ... (goroutines)
  | goroutine|  | goroutine|  | goroutine|
  +----+-----+  +----+-----+  +----+-----+
       |             |             |
       v             v             v
  +--------------------------------+
  |     chan Event (buffered)      |
  +---------------+----------------+
                  |
                  v
  +---------------+----------------+
  |      main loop (select)        |
  |      time.Timer (throttle)     |
  +--------------------------------+


New (Single Poll Loop):

  +------------------------------------------------+
  |              unix.Poll(fds, -1)                 |
  |                                                 |
  |  fds[] = {                                      |
  |    netlink_fd,    // backlight                  |
  |    dbus_eventfd,  // battery/caffeine notify    |
  |    pactl_fd,      // volume (stdout pipe)       |
  |    timerfd_1s,    // clock                      |
  |    timerfd_5s,    // cpu/mem                    |
  |    timerfd_300s,  // storage                    |
  |    throttle_fd,   // output batching            |
  |    cancel_fd,     // shutdown                   |
  |  }                                              |
  |                                                 |
  |  // Single thread, no channels, no Go timers   |
  +------------------------------------------------+
```

## Implementation Details

### 1. timerfd Package

Replace Go `time.Ticker` dengan Linux `timerfd_create()`:

```go
// internal/timerfd/timerfd.go
package timerfd

import (
    "time"
    "golang.org/x/sys/unix"
)

type Timer struct {
    fd       int
    callback func()
}

func New(interval time.Duration, callback func()) (*Timer, error) {
    fd, err := unix.TimerfdCreate(unix.CLOCK_MONOTONIC, unix.TFD_CLOEXEC)
    if err != nil {
        return nil, err
    }

    spec := unix.ItimerSpec{
        Interval: unix.NsecToTimespec(interval.Nanoseconds()),
        Value:    unix.NsecToTimespec(interval.Nanoseconds()),
    }
    unix.TimerfdSettime(fd, 0, &spec, nil)

    return &Timer{fd: fd, callback: callback}, nil
}

func (t *Timer) Fd() int { return t.fd }

func (t *Timer) Handle() {
    var buf [8]byte
    unix.Read(t.fd, buf[:]) // clear timer
    t.callback()
}
```

### 2. eventloop Package

Central poll loop manager:

```go
// internal/eventloop/eventloop.go
package eventloop

import (
    "golang.org/x/sys/unix"
    "syscall"
)

type Handler interface {
    Fd() int
    Handle()
}

type Loop struct {
    fds      []unix.PollFd
    handlers []Handler
}

func New() *Loop {
    return &Loop{}
}

func (l *Loop) Register(h Handler) {
    l.fds = append(l.fds, unix.PollFd{
        Fd:     int32(h.Fd()),
        Events: unix.POLLIN,
    })
    l.handlers = append(l.handlers, h)
}

func (l *Loop) Run() error {
    for {
        n, err := unix.Poll(l.fds, -1) // blocking, zero CPU
        if err != nil {
            if err == syscall.EINTR {
                continue
            }
            return err
        }

        for i := range l.fds {
            if l.fds[i].Revents&unix.POLLIN != 0 {
                l.handlers[i].Handle()
                l.fds[i].Revents = 0
            }
        }
    }
}
```

### 3. Throttle dengan timerfd

One-shot timer untuk output batching:

```go
// internal/eventloop/throttle.go
type Throttle struct {
    fd       int
    armed    bool
    callback func()
}

func (t *Throttle) Arm(duration time.Duration) {
    if t.armed {
        return
    }
    t.armed = true

    spec := unix.ItimerSpec{
        Value: unix.NsecToTimespec(duration.Nanoseconds()),
        // Interval = 0 means one-shot
    }
    unix.TimerfdSettime(t.fd, 0, &spec, nil)
}

func (t *Throttle) Handle() {
    var buf [8]byte
    unix.Read(t.fd, buf[:])
    t.armed = false
    t.callback()
}
```

### 4. Volume Handler (extract fd dari pipe)

```go
// internal/volume/handler.go
type Handler struct {
    cmd      *exec.Cmd
    fd       int
    reader   *bufio.Reader
    callback func(Stats)
}

func NewHandler(callback func(Stats)) (*Handler, error) {
    cmd := exec.Command("pactl", "subscribe")
    stdout, _ := cmd.StdoutPipe()
    cmd.Start()

    file := stdout.(*os.File)

    return &Handler{
        cmd:      cmd,
        fd:       int(file.Fd()),
        reader:   bufio.NewReader(file),
        callback: callback,
    }, nil
}

func (h *Handler) Fd() int { return h.fd }

func (h *Handler) Handle() {
    line, _ := h.reader.ReadString('\n')
    if strings.Contains(line, " on sink ") {
        h.callback(h.Read())
    }
}
```

### 5. DBus Handling (Hybrid Approach)

Karena godbus tidak expose fd secara mudah, gunakan pendekatan hybrid:
- Tetap pakai godbus untuk DBus protocol handling
- Gunakan `eventfd` untuk notify poll loop

```go
// internal/dbus/notifier.go
type Notifier struct {
    eventfd int
    pending []interface{}
    mu      sync.Mutex
}

func (n *Notifier) Notify(data interface{}) {
    n.mu.Lock()
    n.pending = append(n.pending, data)
    n.mu.Unlock()

    // Wake up poll loop
    var one uint64 = 1
    unix.Write(n.eventfd, (*[8]byte)(unsafe.Pointer(&one))[:])
}

func (n *Notifier) Fd() int { return n.eventfd }

func (n *Notifier) Handle() {
    var buf [8]byte
    unix.Read(n.eventfd, buf[:])

    n.mu.Lock()
    pending := n.pending
    n.pending = nil
    n.mu.Unlock()

    for _, data := range pending {
        // process
    }
}
```

**Trade-off**: godbus masih punya 2 internal goroutines (session + system bus), tapi ini acceptable karena:
- Mereka blocking pada socket read
- Jauh lebih simple daripada reimplement DBus protocol

### 6. New Watch() Implementation

```go
func (s *Systower) Watch(ctx context.Context) {
    loop := eventloop.New()

    // Timers (timerfd)
    loop.Register(timerfd.New(s.intervals.Clock, func() {
        s.stats.Clock = s.clockReader.Read()
        s.throttle.Arm(s.intervals.Throttle)
    }))

    loop.Register(timerfd.New(s.intervals.CPU, func() {
        s.stats.CPU = s.cpuReader.Read()
        s.throttle.Arm(s.intervals.Throttle)
    }))

    // ... mem, storage timers

    // Backlight (netlink)
    if s.blMon != nil {
        loop.Register(s.blMon.AsHandler(func(st backlight.Stats) {
            s.stats.Backlight = st
            s.throttle.Arm(s.intervals.Throttle)
        }))
    }

    // Volume (pactl pipe)
    loop.Register(s.volHandler)

    // DBus notifier (battery + caffeine)
    loop.Register(s.dbusNotifier)

    // Throttle output
    s.throttle = eventloop.NewThrottle(func() {
        if s.stats != s.lastStats {
            s.lastStats = s.stats
            os.Stdout.WriteString(s.output())
        }
    })
    loop.Register(s.throttle)

    // Cancellation
    loop.RegisterCancel(ctx)

    loop.Run()
}
```

## Expected Results

| Metric | Before | After |
|--------|--------|-------|
| Context switches/s | ~10 | ~1-2 |
| Goroutines | ~10 | ~3 (main + 2 dbus) |
| Go timers | 2 (ticker + throttle) | 0 |
| Channels | 3 (events + 2 signal) | 0 |

## Files Changed

| File | Action |
|------|--------|
| `internal/timerfd/timerfd.go` | NEW |
| `internal/eventloop/eventloop.go` | NEW |
| `internal/eventloop/throttle.go` | NEW |
| `internal/backlight/backlight.go` | MODIFY - add Handler interface |
| `internal/volume/volume.go` | MODIFY - extract fd |
| `internal/systower/systower.go` | MODIFY - rewrite Watch() |
| `internal/poller/poller.go` | DELETE - replaced by timerfd |

## Verification

```bash
# Build
go build -o /tmp/systower-new ./cmd/systower

# Measure context switches
/tmp/systower-new watch >/dev/null &
PID=$!
sleep 2
INITIAL=$(grep nr_voluntary_switches /proc/$PID/sched | awk '{print $3}')
sleep 10
FINAL=$(grep nr_voluntary_switches /proc/$PID/sched | awk '{print $3}')
echo "Switches/s: $(( (FINAL - INITIAL) / 10 ))"
kill $PID

# Expected: 1-2 switches/s

# Functional test
/tmp/systower-new watch | head -20
# Verify semua stats terisi dengan benar
```

## Risks & Mitigations

1. **DBus complexity** - Mitigasi: Keep godbus, accept 2 goroutines
2. **Linux-only timerfd** - Already Linux-only (netlink, etc)
3. **Breaking changes** - Mitigasi: Extensive testing before merge
