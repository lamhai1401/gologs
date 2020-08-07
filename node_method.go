package main

import (
	"fmt"

	"github.com/beowulflab/rtcbase/utils"
	signal "github.com/beowulflab/signal/livestream"
	"github.com/pion/webrtc/v3"
)

func getTurns() *webrtc.Configuration {
	return &webrtc.Configuration{
		SDPSemantics: webrtc.SDPSemanticsUnifiedPlan,
		ICEServers: []webrtc.ICEServer{
			webrtc.ICEServer{
				URLs: []string{"stun:14.225.239.138:3478"},
			},
			webrtc.ICEServer{
				URLs: []string{"stun:14.225.239.137:3478"},
			},
			// Asia
			webrtc.ICEServer{
				URLs:           []string{"turn:14.225.239.138:5349"},
				Username:       "username",
				Credential:     "password",
				CredentialType: webrtc.ICECredentialTypePassword,
			},
			webrtc.ICEServer{
				URLs:           []string{"turn:14.225.239.137:5349"},
				Username:       "username",
				Credential:     "password",
				CredentialType: webrtc.ICECredentialTypePassword,
			},
		},
	}
}

func (n *Node) getSignal() *signal.Signal {
	n.mutex.RLock()
	defer n.mutex.RUnlock()
	return n.signl
}

func (n *Node) getConn() *webrtc.PeerConnection {
	n.mutex.RLock()
	defer n.mutex.RUnlock()
	return n.conn
}

func (n *Node) setConn(conn *webrtc.PeerConnection) {
	n.mutex.Lock()
	defer n.mutex.Unlock()
	n.conn = conn
}

func (n *Node) getIceCache() *utils.AdvanceMap {
	n.mutex.RLock()
	defer n.mutex.RUnlock()
	return n.iceCache
}

func (n *Node) addIceCache(ice *webrtc.ICECandidateInit) {
	if cache := n.getIceCache(); cache != nil {
		cache.Set(ice.Candidate, ice)
	}
}

// setCacheIce add ice save in cache
func (n *Node) setCacheIce() error {
	cache := n.getIceCache()
	if cache == nil {
		return fmt.Errorf("ICE cache map is nil")
	}
	conn := n.getConn()
	if conn == nil {
		return fmt.Errorf("Peer connection is nil")
	}

	captureCache := cache.Capture()
	for _, value := range captureCache {
		ice, ok := value.(*webrtc.ICECandidateInit)
		if ok {
			if err := conn.AddICECandidate(*ice); err != nil {
				return err
			}
		}
	}
	return nil
}
