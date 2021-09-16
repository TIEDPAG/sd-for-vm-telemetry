package handler

import "sd-for-vm-telemetry/event"

type Handler interface {
	Start()
	Stop()
}

type handler struct {
	eventChan <-chan event.Event
	stop      chan struct{}
}

func NewHandler(eventChan <-chan event.Event) Handler {
	return &handler{
		eventChan: eventChan,
		stop:      make(chan struct{}),
	}
}

func (h *handler) Start() {
	go func() {}()
}

func (h *handler) Stop() {
	h.stop <- struct{}{}
}
