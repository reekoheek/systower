# Systower

A lightweight system monitor daemon for Linux status bars. Designed to work with [yambar](https://codeberg.org/dnkl/yambar) but outputs a simple key-value format that can be adapted for other status bars.

## Features

- **Event-driven** - Uses D-Bus signals, udev netlink, and PulseAudio events (no polling for most monitors)
- **Efficient polling** - Clock, CPU, memory, and storage use configurable intervals with GCD-based ticker optimization
- **Low wakeups** - Dynamic ticker interval calculated from GCD of all polling intervals minimizes CPU wakeups
- **Smart power management**:
  - Auto-disables caffeine when battery drops below 15%
  - Sends notification and powers off when battery drops below 5%
- **Cross-platform** - Supports both X11 and Wayland for screen saver inhibition

## Installation

```bash
go install github.com/reekoheek/systower/cmd/systower@latest
```

Or build from source:

```bash
go build -o systower ./cmd/systower
```

## Usage

### Watch Mode

Outputs system stats to stdout in yambar-compatible format:

```bash
systower watch
```

Default polling intervals:
- Clock: 1s
- CPU: 5s
- Memory: 5s
- Storage: 60s

The ticker interval is automatically optimized using GCD (Greatest Common Divisor) of all polling intervals. For example, with intervals of 1s, 5s, 5s, and 60s, the GCD is 1s, resulting in 1 wakeup/second. To reduce wakeups further, increase the clock interval (e.g., Clock: 5s would give GCD=5s = 0.2 wakeups/second).

Output format:

```
backlight_percent|int|50
clock_day|string|Fri
clock_date|string|31 Jan
clock_time|string|14:30
caffeine|string|off
bat_status|string|discharging
bat_percent|int|75
bat_estimate|string|02:30
cpu_percent|float|12.34567
mem_used|float|8.50000
mem_percent|float|53.12500
storage_percent|float|45.00000
vol_percent|int|85
vol_muted|bool|false
```

### Caffeine Control

Inhibit screen saver / idle sleep:

```bash
systower caffeine on       # Enable caffeine
systower caffeine off      # Disable caffeine
systower caffeine toggle   # Toggle state
systower caffeine status   # Print current state (on/off)
```

On X11, uses `xset` to disable DPMS and screen saver.
On Wayland, creates a lock file at `$XDG_RUNTIME_DIR/swayidle.lock` (configure swayidle to check this file).

## Yambar Configuration

Example yambar configuration using systower:

```yaml
bar:
  left:
    - script:
        path: /path/to/systower
        args: [watch]
        content:
          map:
            conditions:
              caffeine == "on":
                string: { text: "☕" }
              caffeine == "off":
                string: { text: "" }
    - script:
        path: /path/to/systower
        args: [watch]
        content:
          string: { text: "🔋 {bat_percent}% {bat_estimate}" }
```

## Monitors

| Monitor | Source | Update Method |
|---------|--------|---------------|
| Backlight | `/sys/class/backlight`, udev netlink | Event-driven |
| Clock | System time | Polling (configurable, default 1s) |
| Caffeine | D-Bus signal | Event-driven |
| Battery | UPower D-Bus | Event-driven |
| CPU | `/proc/stat` | Polling (configurable, default 5s) |
| Memory | `/proc/meminfo`, `/proc/swaps` | Polling (configurable, default 5s) |
| Storage | `syscall.Statfs` | Polling (configurable, default 60s) |
| Volume | PulseAudio (`pactl subscribe`) | Event-driven |

## Dependencies

- D-Bus (for battery monitoring and notifications)
- UPower (for battery information)
- PulseAudio/PipeWire (`pactl`, `wpctl` for volume monitoring)
- X11: `xset` command (for caffeine on X11)
- Wayland: swayidle (for caffeine on Wayland)

## License

MIT
