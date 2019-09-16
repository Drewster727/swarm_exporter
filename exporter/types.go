package exporter

type Service struct {
	Mode     string
	Replicas uint64
	Name     string
	Tasks    []Task
}

type Task struct {
	DesiredState string
	State        string
	Count        int32
}
