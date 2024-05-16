package types

import (
	detect_types "AndreevSemen/vkr/internal/detect/types"
	"time"

	"github.com/google/uuid"
)

type Event struct {
	UUID        uuid.UUID
	Source      string
	DetectEvent detect_types.Event
}

type Message struct {
	SrcAddress string
	DstAddress string
	Event      Event
}

type Cluster interface {
	Join(a Agent) error
	Send(msg Message, timeout time.Duration) error
	SentMessages() int
	MaxAgentMsgs() int
}

type Agent interface {
	Address() string
	Receive(m Message) error
	GetEvents() <-chan Event
	PublishEvent(Event) error
	Close() error
}
