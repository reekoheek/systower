package event

type Kind int

const (
	BacklightUpdated Kind = iota
	BatteryUpdated
	CaffeineUpdated
	ClockUpdated
	CPUUpdated
	MemUpdated
	StorageUpdated
	VolumeUpdated
)

type Event struct {
	Kind    Kind
	Payload any
}
