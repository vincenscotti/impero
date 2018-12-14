package engine

import (
	"fmt"
	"time"
)

type eventThread struct {
	e                 *Engine
	nextRunValid      bool
	nextRun           time.Time
	pendingRequests   []tokenRequest
	eventTimerExpired <-chan time.Time
	tokenChan         chan tokenRequest
	incomingEvents    chan time.Time
}

type tokenRequest struct {
	timestamp time.Time
	ack       chan bool
}

func NewEventThread(e *Engine) *eventThread {
	et := &eventThread{e: e, nextRunValid: false,
		pendingRequests:   make([]tokenRequest, 0, 100),
		eventTimerExpired: make(<-chan time.Time, 1),
		tokenChan:         make(chan tokenRequest, 100),
		incomingEvents:    make(chan time.Time, 100)}

	go et.run()

	return et
}

func (et *eventThread) RequestToken(timestamp time.Time) {
	req := tokenRequest{timestamp: timestamp, ack: make(chan bool)}

	fmt.Println("REQUESTING TOKEN")
	et.tokenChan <- req
	<-req.ack
}

func (et *eventThread) RegisterEvent(timestamp time.Time) {
	et.incomingEvents <- timestamp
}

func (et *eventThread) runTimer() {
	fmt.Println("RUNNING TIMER")
	et.eventTimerExpired = time.After(time.Until(et.nextRun))
}

func (et *eventThread) run() {
	for {
		select {
		case <-et.eventTimerExpired:
			// FIXME: review this approach
			fmt.Println("PROCESSING EVENTS")
			tx := et.e.openSessionUnsafe()
			et.nextRunValid, et.nextRun = tx.processEvents()
			tx.Commit()
			tx.Close()

			if et.nextRunValid {
				fmt.Println("RESCHEDULING AT", et.nextRun)
				et.runTimer()
			}
			fmt.Println("PROCESSING DONE")

			// copy before launching the goroutine to ensure atomicity
			pendingRequestsCopy := make([]tokenRequest, len(et.pendingRequests))
			fmt.Println("GOT", len(et.pendingRequests), "PENDING REQUESTS")
			copy(pendingRequestsCopy, et.pendingRequests)

			// the send can block so we do it on another goroutine
			go func() {
				for i, tr := range pendingRequestsCopy {
					fmt.Println("enqueueing request", i)
					et.tokenChan <- tr
				}
			}()

		case ts := <-et.incomingEvents:
			fmt.Println("INCOMING EVENT AT", ts)
			if !et.nextRunValid || ts.Before(et.nextRun) {
				et.nextRunValid, et.nextRun = true, ts
				et.runTimer()
			}

		case req := <-et.tokenChan:
			fmt.Println("got req", req)
			if req.ack != nil {
				if et.nextRunValid && req.timestamp.After(et.nextRun) {
					fmt.Println("ADDING PENDING REQUEST")
					et.pendingRequests = append(et.pendingRequests, req)
				} else {
					fmt.Println("sending ack")
					req.ack <- true
				}
			}
			fmt.Println("end req")
		}
	}
}
