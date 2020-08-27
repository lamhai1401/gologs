package main

import (
	"fmt"
	"strings"
	"sync"

	"github.com/beowulflab/signal/signal-wss"
	log "github.com/lamhai1401/gologs/logs"
)

func isStudent(id string) bool {
	return strings.HasPrefix(id, "student")
}

func isTeacher(id string) bool {
	return strings.HasPrefix(id, "teacher")
}

// Peers linter
type Peers struct {
	signal   *signal.NotifySignal // send socket
	conns    *AdvanceMap          // save peer connection with id
	audioFwd *Forwarder           // studentID
	mutex    sync.RWMutex
}

// NewPeers litner
func NewPeers() *Peers {
	p := &Peers{
		conns:    NewAdvanceMap(),
		audioFwd: NewForwarder("peers"),
	}

	sig := signal.NewNotifySignal("123", p.processNotifySignal)
	go sig.Start()
	p.signal = sig
	return p
}

func (ps *Peers) addSDP(id, session string, values interface{}) error {

}

func (ps *Peers) processNotifySignal(values []interface{}) {
	if len(values) < 3 {
		log.Error("Len of msg < 4")
		return
	}

	signalID, hasSignalID := values[0].(string)
	if !hasSignalID {
		log.Error(fmt.Sprintf("[ProcessSignal] Invalid signal ID: %v", signalID))
		return
	}

	sessionID, hasSessionID := values[1].(string)
	if !hasSessionID {
		log.Error(fmt.Sprintf("[ProcessSignal] Invalid session ID: %v", sessionID))
		return
	}

	event, isEvent := values[2].(string)
	if !isEvent {
		log.Error(fmt.Sprintf("[ProcessSignal] Invalid event: %v", event))
		return
	}

	var err error
	switch event {
	case "ok":
		log.Debug(fmt.Sprintf("Receive ok from id: %s_%s", signalID, sessionID))
		err = ps.handleOkEvent(signalID, sessionID)
		break
	default:
		err = fmt.Errorf("[ProcessSignal] receive not processing event: %s", event)
	}

	if err != nil {
		log.Error(err.Error())
		ps.sendError(signalID, sessionID, err.Error())
	}
}

func (ps *Peers) handleOkEvent(signalID string, sessionID string) error {
	ps.sendOk(signalID, sessionID)
	return nil
}

func (ps *Peers) handleSDPEvent(signalID, sessionID string, value interface{}) error {

	return nil
}
