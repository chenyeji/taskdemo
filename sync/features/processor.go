package features

type Processor interface {
	Loop(shutdown chan struct{})
}
