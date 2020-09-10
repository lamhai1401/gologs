package main

import (
	"fmt"

	"github.com/beowulflab/rtcbase-v2/utils"
	"github.com/beowulflab/signal/signal-wss"
	"github.com/lamhai1401/gologs/logs"
	"github.com/lamhai1401/gologs/mixing"
	"github.com/lamhai1401/gologs/peer"
	"github.com/pion/webrtc/v2"
)

func (ps *Peers) getID() string {
	ps.mutex.RLock()
	defer ps.mutex.RUnlock()
	return ps.id
}

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

func (ps *Peers) getConn(id string) *peer.Peer {
	if conns := ps.getConns(); conns != nil {
		conn, has := conns.Get(id)
		if has {
			peer, ok := conn.(*peer.Peer)
			if ok {
				return peer
			}
		}
	}
	return nil
}

func (ps *Peers) setConn(id string, peer *peer.Peer) {
	if conns := ps.getConns(); conns != nil {
		conns.Set(id, peer)
	}
}

func (ps *Peers) getAudioFwd() utils.Fwdm {
	ps.mutex.RLock()
	defer ps.mutex.RUnlock()
	return ps.audioFwdm
}

func (ps *Peers) register(id string, clientID string, handler func(wrapper *utils.Wrapper) error) {
	if fwdm := ps.getAudioFwd(); fwdm != nil {
		fwdm.Register(id, clientID, handler)
	}
}

func (ps *Peers) unRegister(id, clientID string) {
	if fwdm := ps.getAudioFwd(); fwdm != nil {
		fwdm.Unregister(id, clientID)
	}
}

func (ps *Peers) getConfig() *webrtc.Configuration {
	ps.mutex.RLock()
	defer ps.mutex.RUnlock()
	return ps.configs
}

func (ps *Peers) addConn(id, session string) (*peer.Peer, error) {
	peer := peer.NewPeer(&ps.bitrate, session, id)
	ps.setConn(id, peer)
	return peer, nil
}

func (ps *Peers) deleteConn(id string) {
	if conns := ps.getConns(); conns != nil {
		conns.Delete(id)
	}
}

func (ps *Peers) closeConn(id string) {
	if conn := ps.getConn(id); conn != nil {
		ps.deleteConn(id)
		conn.Close()

		role := checkRole(conn.GetSignalID())
		switch role {
		case "teacher":
			// call remove mixer
			if mixer := ps.getMixedAudio(); mixer != nil {
				index, err := mixer.FindIndex(id)
				if err == nil {
					mixer.RemoveAudioStream(id)
					mixer.UnRegister(index, id)
				}
			}
			break
		case "student":
			// call remove from fwd
			ps.unRegister(ps.getID(), conn.GetSignalID())
			break
		default:
			break
		}
		conn = nil
	}
}

func (ps *Peers) handleConnEvent(peer *peer.Peer) {
	role := checkRole(peer.GetSignalID())
	conn := peer.GetConn()

	conn.OnICECandidate(func(i *webrtc.ICECandidate) {
		if i == nil {
			return
		}
		ps.sendCandidate(peer.GetSignalID(), peer.GetSessionID(), i.ToJSON())
	})

	conn.OnICEConnectionStateChange(func(is webrtc.ICEConnectionState) {
		state := is.String()
		logs.Info(fmt.Sprintf("Connection %s has states %s", peer.GetSignalID(), state))
		switch state {
		case "connected":

			if !peer.CheckConnected() {
				if role == "student" {
					ps.register(ps.getID(), peer.GetSignalID(), func(wrapper *utils.Wrapper) error {
						err := peer.AddAudioRTP(&wrapper.Pkg)
						if err != nil {
							logs.Error(err.Error())
						}
						logs.Stack(fmt.Sprintf("[Mixer] Add mixed audio rtp to %s", peer.GetSignalID()))
						return nil
					})
				}

				if role == "teacher" {
					if mixer := ps.getMixedAudio(); mixer != nil {
						index, err := mixer.FindIndex(peer.GetSignalID())
						if err != nil {
							logs.Error(err.Error())
						}
						mixer.Register(index, peer)
						// ps.register(index, peer.GetSignalID(), func(wrapper *utils.Wrapper) error {
						// 	err := peer.AddAudioRTP(&wrapper.Pkg)
						// 	if err != nil {
						// 		logs.Error(err.Error())
						// 	}
						// 	logs.Stack(fmt.Sprintf("[Mixer] Add %s mixed audio rtp to teacher %s", index, peer.GetSignalID()))
						// 	return nil
						// })
					}
				}
			}
			peer.SetConnected()
			break
		case "closed":
			if role == "student" {
				ps.closeConn(peer.GetSignalID())
			}
			break
		case "failed":
			ps.closeConn(peer.GetSignalID())
			break
		default:
			break
		}
	})

	conn.OnTrack(func(remoteTrack *webrtc.Track, r *webrtc.RTPReceiver) {
		logs.Info(fmt.Sprintf("Has remote %s track of ID %s", remoteTrack.Kind().String(), peer.GetSignalID()))
		kind := remoteTrack.Kind().String()

		if role == "teacher" && kind == "audio" {
			// call register to mixer
			if mixer := ps.getMixedAudio(); mixer != nil {
				index, err := mixer.FindIndex(peer.GetSignalID())
				logs.Info("[Mixer]", fmt.Sprintf("%s index is: %s", peer.GetSignalID(), index))
				if err != nil {
					logs.Error("ontrack err: ", err.Error())
					return
				}
				mixer.AddAudioStream(index, peer.GetSignalID(), remoteTrack)
			}
		}
	})
}

func (ps *Peers) getMixedAudio() mixing.Mixer {
	ps.mutex.RLock()
	defer ps.mutex.RUnlock()
	return ps.mixer
}
