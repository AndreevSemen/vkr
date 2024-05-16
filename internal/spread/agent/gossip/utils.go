package gossip

import (
	"container/list"

	"github.com/google/uuid"

	"gossip_emulation/internal/spread/types"
)

func newEventQueue() (chan<- types.Event, <-chan types.Event) {
	send := make(chan types.Event, 1)
	receive := make(chan types.Event, 1)
	go manageQueue(send, receive)
	return send, receive
}

func manageQueue(send <-chan types.Event, receive chan<- types.Event) {
	sent := make(map[uuid.UUID]struct{})
	queue := list.New()
	for {
		if front := queue.Front(); front == nil {
			if send == nil {
				close(receive)
				return
			}
			value, ok := <-send
			if !ok {
				close(receive)
				return
			}
			queue.PushBack(value)

		} else {
			e := front.Value.(types.Event)
			if _, exists := sent[e.UUID]; exists {
				queue.Remove(front)

			} else {
				select {
				case receive <- e:
					queue.Remove(front)
					sent[e.UUID] = struct{}{}
				case value, ok := <-send:
					if ok {
						queue.PushBack(value)
					} else {
						send = nil
					}
				}
			}
		}
	}
}
