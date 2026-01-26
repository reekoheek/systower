package caffeine

type Adapter interface {
	On()
	Off()
	Status() string
}
