package main

import (
	"fmt"
	"strings"
	"sync"

	"github.com/beowulflab/rtcbase-v2/utils"
	"github.com/beowulflab/signal/signal-wss"
	"github.com/lamhai1401/gologs/logs"
	log "github.com/lamhai1401/gologs/logs"
	"github.com/lamhai1401/gologs/mixing"
	"github.com/pion/webrtc/v2"
)

func isStudent(id string) bool {
	return strings.HasPrefix(id, "student")
}

func isTeacher(id string) bool {
	return strings.HasPrefix(id, "teacher")
}

func checkRole(id string) string {
	var role string
	if isStudent(id) {
		role = "student"
	}
	if isTeacher(id) {
		role = "teacher"
	}
	return role
}

// Peers linter
type Peers struct {
	id        string
	signal    *signal.NotifySignal // send socket
	conns     *AdvanceMap          // save peer connection with id
	mixer     mixing.Mixer         // mixing stream
	audioFwdm utils.Fwdm
	bitrate   int
	configs   *webrtc.Configuration
	mutex     sync.RWMutex
}

// NewPeers litner
func NewPeers() (*Peers, error) {
	p := &Peers{
		id:        "mixedStreamID",
		conns:     NewAdvanceMap(),
		bitrate:   1000,
		configs:   utils.GetTurns(),
		audioFwdm: utils.NewForwarderMannager("id"),
	}

	mixer := mixing.NewMixer(3, "mixedStreamID")
	if err := mixer.Start(); err != nil {
		return nil, err
	}
	p.mixer = mixer
	p.initAudioMixed()

	sig := signal.NewNotifySignal("123", p.processNotifySignal)
	go sig.Start()
	p.signal = sig
	return p, nil
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
	case "sdp":
		log.Debug(fmt.Sprintf("Receive sdp from id: %s_%s", signalID, sessionID))
		err = ps.handleSDPEvent(signalID, sessionID, values[3])
		break
	case "candidate":
		err = ps.handCandidateEvent(signalID, sessionID, values[3])
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
	return ps.addSDP(signalID, sessionID, value)
}

func (ps *Peers) addSDP(id, session string, values interface{}) error {
	var err error
	peer := ps.getConn(id)

	if peer != nil {
		ps.closeConn(id)
	}

	peer, err = ps.addConn(id, session)
	if err != nil {
		return err
	}

	_, err = peer.NewConnection(values, ps.getConfig())
	if err != nil {
		return err
	}
	ps.handleConnEvent(peer)

	err = peer.AddSDP(values)
	if err != nil {
		return err
	}

	answer, err := peer.GetLocalDescription()
	if err != nil {
		return err
	}

	ps.sendSDP(id, session, answer)
	return err
}

func (ps *Peers) handCandidateEvent(signalID string, sessionID string, value interface{}) error {
	return ps.addCandidate(signalID, sessionID, value)
}

func (ps *Peers) addCandidate(id, session string, values interface{}) error {
	if conn := ps.getConn(id); conn != nil {
		return conn.AddICECandidate(values)
	}
	return fmt.Errorf("Connection with id %s is nil", id)
}

func (ps *Peers) initAudioMixed() {
	if mixer := ps.getMixedAudio(); mixer != nil {
		chann := mixer.GetMixedAudioOutput()

		go func() {
			var fwd *utils.Forwarder
			for {
				rtp, open := <-chann
				if !open {
					return
				}
				if fwdm := ps.getAudioFwd(); fwdm != nil {
					fwd = fwdm.GetForwarder(ps.getID())
					if fwd == nil {
						fwd = fwdm.AddNewForwarder(ps.getID())
					}
					logs.Stack("[Mixer] Push mixed audio to fwdm")
					fwd.Push(&utils.Wrapper{
						Pkg: *rtp,
					})
				}
			}
		}()
	}
}
