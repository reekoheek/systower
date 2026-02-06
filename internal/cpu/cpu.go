package cpu

import (
	"bufio"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type Stats struct {
	Idle  uint64
	Total uint64
	Temp  int
}

func (s Stats) Percent() int {
	if s.Total == 0 {
		return 0
	}
	return int(math.Round(100 - float64(s.Idle)*100/float64(s.Total)))
}

type Reader struct {
	prevIdle  uint64
	prevTotal uint64
	tempPath  string
}

func New() *Reader {
	return &Reader{tempPath: detectThermalZone()}
}

func (r *Reader) Read() Stats {
	idle, total := r.readProcStat()

	deltaIdle := idle - r.prevIdle
	deltaTotal := total - r.prevTotal

	r.prevIdle = idle
	r.prevTotal = total

	return Stats{
		Idle:  deltaIdle,
		Total: deltaTotal,
		Temp:  r.readTemp(),
	}
}

func (r *Reader) readProcStat() (idle, total uint64) {
	f, err := os.Open("/proc/stat")
	if err != nil {
		return 0, 0
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	if !scanner.Scan() {
		return 0, 0
	}

	// cpu  user nice system idle iowait irq softirq steal guest guest_nice
	fields := strings.Fields(scanner.Text())
	if len(fields) < 5 || fields[0] != "cpu" {
		return 0, 0
	}

	// user nice system idle iowait irq softirq steal | guest guest_nice
	// guest/guest_nice already included in user/nice, skip them
	for i, field := range fields[1:] {
		if i >= 8 {
			break
		}
		val, _ := strconv.ParseUint(field, 10, 64)
		total += val
		if i == 3 || i == 4 { // idle + iowait
			idle += val
		}
	}

	return idle, total
}

var cpuThermalTypes = []string{"x86_pkg_temp", "TCPU", "acpitz"}

func detectThermalZone() string {
	zones, _ := filepath.Glob("/sys/class/thermal/thermal_zone*/type")

	for _, target := range cpuThermalTypes {
		for _, zoneType := range zones {
			data, err := os.ReadFile(zoneType)
			if err != nil {
				continue
			}
			if strings.TrimSpace(string(data)) == target {
				return filepath.Join(filepath.Dir(zoneType), "temp")
			}
		}
	}

	if _, err := os.Stat("/sys/class/thermal/thermal_zone0/temp"); err == nil {
		return "/sys/class/thermal/thermal_zone0/temp"
	}

	return ""
}

func (r *Reader) readTemp() int {
	if r.tempPath == "" {
		return 0
	}

	data, err := os.ReadFile(r.tempPath)
	if err != nil {
		return 0
	}

	millideg, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return 0
	}

	return int(math.Round(float64(millideg) / 1000))
}
