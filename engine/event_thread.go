package engine

import (
	"time"
)

type eventThread struct {
	e                 *Engine
	nextRunValid      bool
	nextRun           time.Time
	pendingRequests   []*tokenRequest
	eventTimerExpired <-chan time.Time
	tokenChan         chan *tokenRequest
	incomingEvents    chan time.Time
}

type tokenRequest struct {
	timestamp time.Time
	ack       chan bool
}

func NewEventThread(e *Engine) *eventThread {
	et := &eventThread{e: e, nextRunValid: false,
		pendingRequests:   make([]*tokenRequest, 1),
		eventTimerExpired: make(<-chan time.Time, 1),
		tokenChan:         make(chan *tokenRequest, 100),
		incomingEvents:    make(chan time.Time, 100)}

	go et.run()

	return et
}

func (et *eventThread) RequestToken(timestamp time.Time) {
	req := &tokenRequest{timestamp: timestamp, ack: make(chan bool)}

	et.tokenChan <- req
	<-req.ack
}

func (et *eventThread) RegisterEvent(timestamp time.Time) {
	et.incomingEvents <- timestamp
}

func (et *eventThread) run() {
	for {
		select {
		case <-et.eventTimerExpired:
			// FIXME: review this approach
			et.nextRunValid, et.nextRun = et.e.processEvents()

			// copy before launching the goroutine to ensure atomicity
			pendingRequestsCopy := make([]*tokenRequest, len(et.pendingRequests))
			copy(et.pendingRequests, pendingRequestsCopy)

			// the send can block so we do it on another goroutine
			go func() {
				for _, tr := range pendingRequestsCopy {
					et.tokenChan <- tr
				}
			}()

		case ts := <-et.incomingEvents:
			if !et.nextRunValid || ts.Before(et.nextRun) {
				et.nextRunValid, et.nextRun = true, ts

				et.eventTimerExpired = time.After(time.Until(ts))
			}

		case req := <-et.tokenChan:
			if et.nextRunValid && req.timestamp.After(et.nextRun) {
				et.pendingRequests = append(et.pendingRequests, req)
			} else {
				req.ack <- true
			}
		}
	}
}
