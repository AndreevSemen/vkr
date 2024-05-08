package types

type Detector interface {
	DetectorEvents() <-chan Event
	Close() error
}

type AppState int

const (
	AppState_AppAvailable AppState = iota
	AppState_PortAvailable
	AppState_HostAvailable
	AppState_Unavailable
)

type Event struct {
	Hostname string
	State    AppState
}
