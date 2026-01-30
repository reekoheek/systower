# Optimization Roadmap

Target: lightweight system app with battery optimization.

## Critical

- [ ] Cache X11 caffeine status (avoid `xset q` spawn on every read)
- [ ] Graceful shutdown for goroutines (battery, caffeine, poller listeners)
- [ ] Configurable polling intervals via flags/config

## High Priority

- [ ] Cache GCD calculation in poller (calculate once at init, not every tick)
- [ ] Increase event channel buffer (10 → 64)
- [ ] Fix EventBus mutex pattern (copy handlers before calling, release lock earlier)

## Medium Priority

- [ ] Reduce file I/O allocations (reuse scanner/buffer for `/proc/*` reads)
- [ ] Extract duplicated battery parsing logic
- [ ] Reuse timer instead of creating new one per event

## Low Priority

- [ ] Use `sync.Pool` for `strings.Builder` in output()
- [ ] Cache formatted clock strings (avoid redundant `time.Unix()` calls)
- [ ] Increase storage polling interval (60s → 300s)
