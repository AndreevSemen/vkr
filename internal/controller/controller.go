package controller

import (
	spread_types "AndreevSemen/vkr/internal/spread/types"
	"log"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
	"github.com/google/uuid"
)

type EventsProvider interface {
	WatchEvents() <-chan spread_types.Event
}

type Controller struct {
	p EventsProvider

	// source -> hostname -> event
	mx       *sync.RWMutex
	events   map[string]map[string]spread_types.Event
	watchers map[uuid.UUID]chan spread_types.Event
}

func NewController(p EventsProvider) *Controller {
	c := &Controller{
		p:        p,
		mx:       &sync.RWMutex{},
		events:   make(map[string]map[string]spread_types.Event),
		watchers: make(map[uuid.UUID]chan spread_types.Event),
	}

	go c.manage()

	return c
}

func (c *Controller) Route(app *fiber.App) {
	app.Use("/ws", func(ctx *fiber.Ctx) error {
		if websocket.IsWebSocketUpgrade(ctx) {
			ctx.Locals("allowed", true)
			return ctx.Next()
		}
		return fiber.ErrUpgradeRequired
	}).Get("/ws", websocket.New(c.HandleWS))
}

func (c *Controller) manage() {
	for e := range c.p.WatchEvents() {
		c.mx.Lock()
		hosts, exists := c.events[e.Source]
		if !exists {
			hosts = make(map[string]spread_types.Event)
			c.events[e.Source] = hosts
		}

		hosts[e.DetectEvent.Hostname] = e

		for _, w := range c.watchers {
			w <- e
		}
		c.mx.Unlock()
	}
}

func (c *Controller) startWatch(id uuid.UUID) ([]spread_types.Event, <-chan spread_types.Event) {
	c.mx.Lock()
	defer c.mx.Unlock()

	var initial []spread_types.Event
	for _, hosts := range c.events {
		for _, e := range hosts {
			initial = append(initial, e)
		}
	}

	watcher := make(chan spread_types.Event, 100)
	c.watchers[id] = watcher

	return initial, watcher
}

func (c *Controller) stopWatch(id uuid.UUID) {
	c.mx.Lock()
	defer c.mx.Unlock()

	close(c.watchers[id])
	delete(c.watchers, id)
}

func (c *Controller) HandleWS(conn *websocket.Conn) {
	var resErr error
	defer func() {
		if resErr != nil {
			log.Printf("result error: %v", resErr)
		}
		if closeErr := conn.WriteControl(websocket.CloseMessage, []byte(resErr.Error()), time.Now().Add(5*time.Second)); closeErr != nil {
			log.Printf("write close error: %v", closeErr)
		}
		if closeErr := conn.Close(); closeErr != nil {
			log.Printf("close error: %v", closeErr)
		}
	}()

	id := uuid.New()

	initial, watcher := c.startWatch(id)
	defer c.stopWatch(id)

	for _, e := range initial {
		if err := conn.WriteJSON(e); err != nil {
			resErr = err
			return
		}
	}

	for e := range watcher {
		if err := conn.WriteJSON(e); err != nil {
			resErr = err
			return
		}
	}
}
