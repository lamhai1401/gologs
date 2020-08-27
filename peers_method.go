package main

import "github.com/beowulflab/signal/signal-wss"

func (ps *Peers) getSignal() *signal.NotifySignal {
	ps.mutex.RLock()
	defer ps.mutex.RUnlock()
	return ps.signal
}

func (ps *Peers) sendOk(id, session string) {
	if signal := ps.getSignal(); signal != nil {
		signal.Send(id, session, "ok")
	}
}

func (ps *Peers) sendSDP(id, session string, sdp interface{}) {
	if signal := ps.getSignal(); signal != nil {
		signal.Send(id, session, "sdp", sdp)
	}
}

func (ps *Peers) sendCandidate(id, session string, candidate interface{}) {
	if signal := ps.getSignal(); signal != nil {
		signal.Send(id, session, "candidate", candidate)
	}
}

func (ps *Peers) sendError(id, session string, reason interface{}) {
	if signal := ps.getSignal(); signal != nil {
		signal.Send(id, session, "error", reason)
	}
}

func (ps *Peers) getConns() *AdvanceMap {
	ps.mutex.RLock()
	defer ps.mutex.RUnlock()
	return ps.conns
}

func (ps *Peers) getConn(id string) *Peer {
	if conns := ps.getConns(); conns != nil {
		conn, has := conns.Get(id)
		if has {
			peer, ok := conn.(*Peer)
			if ok {
				return peer
			}
		}
	}
	return nil
}

func (ps *Peers) getAudioFwd() *Forwarder {
	ps.mutex.RLock()
	defer ps.mutex.RUnlock()
	return ps.audioFwd
}

func (ps *Peers) register(id string, handler func(wrapper *Wrapper) error) {
	if fwd := ps.getAudioFwd(); fwd != nil {
		fwd.Register(id, handler)
	}
}

func (ps *Peers) unRegister(id string) {
	if fwd := ps.getAudioFwd(); fwd != nil {
		fwd.UnRegister(id)
	}
}
