package main

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/lamhai1401/gologs/logs"
	log "github.com/lamhai1401/gologs/logs"
	"github.com/pion/rtp"
)

type action struct {
	id      *string
	action  *string
	handler func(wrapper *Wrapper) error
	wg      *sync.WaitGroup
}

// Wrapper linter
type Wrapper struct {
	Pkg    rtp.Packet // save rtp packet
	Data   []byte     `json:"rtp"`    // packet to write
	Kind   string     `json:"kind"`   // audio or video
	SeatID int        `json:"seatID"` // stream id number 1-2-3-4
	Type   string     `json:"type"`   // type off wrapper data - ok - ping - pong
}

// Forwarder linter
type Forwarder struct {
	id          string                                  // stream id
	isClosed    bool                                    // makesure is closed
	clients     *AdvanceMap                             // clientID - channel
	handlers    map[string]func(wrapper *Wrapper) error // to save handler
	actionChann chan *action                            // handle action add and remove, close
	msgChann    chan *Wrapper
	mutex       sync.RWMutex
}

// NewForwarder return new forwarder
func NewForwarder(id string) *Forwarder {
	f := &Forwarder{
		id:          id,
		actionChann: make(chan *action, 100),
		clients:     NewAdvanceMap(),
		handlers:    make(map[string]func(wrapper *Wrapper) error),
		isClosed:    false,
		msgChann:    make(chan *Wrapper),
	}

	f.serve()
	return f
}

// Close linter
func (f *Forwarder) Close() {
	if f.checkClose() {
		return
	}
	if chann := f.getActionChann(); chann != nil {
		close := "close"
		a := &action{
			action: &close,
		}
		chann <- a
	}
}

// Push new wrapper to server chan
func (f *Forwarder) Push(wrapper *Wrapper) {
	if f.checkClose() {
		f.info("fwd was closed")
		return
	}
	if chann := f.getMsgChann(); chann != nil {
		chann <- wrapper
	}
}

// Register new client
func (f *Forwarder) Register(clientID string, handler func(wrapper *Wrapper) error) {
	if chann := f.getActionChann(); chann != nil && !f.checkClose() {
		add := "add"
		chann <- &action{
			id:      &clientID,
			action:  &add,
			handler: handler,
		}
	}
}

func (f *Forwarder) addNewClient(clientID string, handler func(wrapper *Wrapper) error) {
	if f.checkClose() {
		f.info("fwd was closed")
		return
	}

	// remove client if exist
	if chann := f.getClient(clientID); chann != nil {
		f.UnRegister(clientID)
	}

	f.setClient(clientID, make(chan *Wrapper, 1000))
	f.setHandler(clientID, handler)

	go f.collectData(clientID)
}

func (f *Forwarder) collectData(clientID string) {
	var handler func(w *Wrapper) error
	var err error
	chann := f.getData(clientID)

	for {
		if f.checkClose() {
			f.info("fwd was closed")
			return
		}
		w, open := <-chann
		if !open {
			// fmt.Println("out collectData")
			return
		}

		handler = f.getHandler(clientID)
		if handler == nil {
			f.info(fmt.Sprintf("%s handler is nil. Close for loop", clientID))
			return
		}

		if err = handler(&w); err != nil {
			log.Error(fmt.Sprintf("%s handler err: %v", clientID, err))
			return
		}

		w = Wrapper{} // clear mem
		handler = nil
		err = nil
	}
}

func (f *Forwarder) getData(clientID string) <-chan Wrapper {
	c := make(chan Wrapper, 100)
	var dumpWrapper *Wrapper

	// dumpWrapper := &Wrapper{
	// 	Kind: "string",
	// 	Type: "string",
	// }

	parent := context.Background()
	timeout := 3 * time.Second
	var ctx context.Context
	var cancel context.CancelFunc

	chann := f.getClient(clientID)

	go func() {
		defer close(c)
		for {
			ctx, cancel = context.WithTimeout(parent, timeout)
			select {
			case w, open := <-chann:
				if !open {
					// fmt.Println("out getData")
					return
				}
				c <- *w
				dumpWrapper = w
				break
			case <-ctx.Done():
				if dumpWrapper != nil {
					c <- *dumpWrapper
				}
				break
			}
			cancel()
			ctx = nil
			cancel = nil
		}
	}()

	return c
}

// UnRegister linter
func (f *Forwarder) UnRegister(clientID string) {
	if f.checkClose() {
		return
	}

	if chann := f.getActionChann(); chann != nil {
		remove := "remove"
		a := &action{
			action: &remove,
			id:     &clientID,
		}
		chann <- a
	}
}

// transfer old fwd to new fwd
func (f *Forwarder) transfer(fw *Forwarder) {
	if f.checkClose() {
		f.info("fwd was closed")
		return
	}

	if clients := f.getClients(); clients != nil {
		tmp := clients.Capture()

		for k, v := range tmp {
			handler, ok := v.(func(wrapper *Wrapper) error)
			if ok {
				fw.setClient(k, make(chan *Wrapper, 1000))
				fw.setHandler(k, handler)
				fw.collectData(k)
			}
		}
	}
}

func (f *Forwarder) getID() string {
	f.mutex.RLock()
	defer f.mutex.RUnlock()
	return f.id
}

func (f *Forwarder) checkClose() bool {
	f.mutex.RLock()
	defer f.mutex.RUnlock()
	return f.isClosed
}

func (f *Forwarder) setClose(state bool) {
	f.mutex.Lock()
	defer f.mutex.Unlock()
	f.isClosed = state
}

// close to close all serve
func (f *Forwarder) close() {
	if !f.checkClose() {
		f.setClose(true)
		f.closeClients()
		f.info(fmt.Sprintf("%s forwader was closed", f.getID()))
	}
}

// info to export log info
func (f *Forwarder) info(v ...interface{}) {
	log.Info(fmt.Sprintf("[%s] ", f.id), v)
}

// error to export error info
func (f *Forwarder) error(v ...interface{}) {
	log.Error(fmt.Sprintf("[%s] ", f.id), v)
}

func (f *Forwarder) getClient(clientID string) chan *Wrapper {
	if clients := f.getClients(); clients != nil {
		client, ok1 := clients.Get(clientID)
		if !ok1 {
			return nil
		}
		chann, ok2 := client.(chan *Wrapper)
		if ok2 {
			return chann
		}
	}
	return nil
}

func (f *Forwarder) setClient(clientID string, chann chan *Wrapper) {
	if clients := f.getClients(); clients != nil {
		clients.Set(clientID, chann)
	}
}

func (f *Forwarder) deleteClient(clientID string) {
	if clients := f.getClients(); clients != nil {
		clients.Delete(clientID)
	}
}

func (f *Forwarder) closeClient(clientID string) {
	if client := f.getClient(clientID); client != nil {
		f.deleteClient(clientID)
		f.deleteHandler(clientID)
		close(client)
		client = nil
		log.Info(fmt.Sprintf("Remove id %s from Forwarder id: %s done", clientID, f.getID()))
	}
}

func (f *Forwarder) closeClients() {
	if clients := f.getClients(); clients != nil {
		keys := clients.GetKeys()
		for _, key := range keys {
			f.closeClient(key)
		}
	}
}

func (f *Forwarder) getClients() *AdvanceMap {
	f.mutex.RLock()
	defer f.mutex.RUnlock()
	return f.clients
}

func (f *Forwarder) setHandler(id string, handler func(w *Wrapper) error) {
	f.mutex.Lock()
	defer f.mutex.Unlock()
	f.handlers[id] = handler
}

func (f *Forwarder) getHandler(id string) func(w *Wrapper) error {
	f.mutex.RLock()
	defer f.mutex.RUnlock()
	return f.handlers[id]
}

func (f *Forwarder) deleteHandler(id string) {
	f.mutex.Lock()
	defer f.mutex.Unlock()
	delete(f.handlers, id)
}

func (f *Forwarder) getActionChann() chan *action {
	f.mutex.Lock()
	defer f.mutex.Unlock()
	return f.actionChann
}

// Serve to run
func (f *Forwarder) serve() {
	go func() {
		for {
			msg, open := <-f.msgChann
			if !open || f.checkClose() {
				return
			}
			f.forward(msg)
			msg = nil
		}
	}()

	go func() {
		for {
			action, open := <-f.actionChann
			if !open || f.checkClose() {
				return
			}
			switch *action.action {
			case "remove":
				f.closeClient(*action.id)
				break
			case "add":
				f.addNewClient(*action.id, action.handler)
				break
			case "close":
				f.close()
				return
			default:
				logs.Info("Nothing to do with this action: ", *action.action)
			}
		}
	}()
}

func (f *Forwarder) forward(wrapper *Wrapper) {
	if f.checkClose() {
		f.info(f.getID(), " fwd was closed")
		return
	}

	if clients := f.getClients(); clients != nil {
		clients.Iter(func(key, value interface{}) bool {
			chann, ok := value.(chan *Wrapper)
			if ok {
				chann <- wrapper
			}
			return true
		})
	}
}

func (f *Forwarder) getMsgChann() chan *Wrapper {
	f.mutex.RLock()
	defer f.mutex.RUnlock()
	return f.msgChann
}
