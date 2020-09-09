package peer

import (
	"fmt"
	"sync"

	"github.com/beowulflab/rtcbase/utils"
	"github.com/mitchellh/mapstructure"
	"github.com/pion/rtp"
	"github.com/pion/webrtc/v2"
)

// Peer linter
type Peer struct {
	bitrate           *int
	iceCache          *utils.AdvanceMap
	conn              *webrtc.PeerConnection
	mutex             sync.RWMutex
	sessionID         string
	signalID          string
	localVideoTrack   *webrtc.Track
	localAudioTrack   *webrtc.Track
	remotelVideoTrack *webrtc.Track
	remoteVideoTrack  *webrtc.Track
	videoCodecs       uint8
	audioCodecs       uint8
	isConnected       bool
	isClosed          bool
}

// NewPeer linter
func NewPeer(
	bitrate *int,
	sessionID string,
	signalID string,
) *Peer {
	p := &Peer{
		bitrate:     bitrate,
		iceCache:    utils.NewAdvanceMap(),
		sessionID:   sessionID,
		signalID:    signalID,
		isClosed:    false,
		isConnected: false,
	}

	return p
}

// Close linter
func (p *Peer) Close() {
	if !p.checkClose() {
		p.setClose(true)
		p.closeConn()
	}
}

// AddVideoRTP write rtp to local video track
func (p *Peer) AddVideoRTP(packet *rtp.Packet) error {
	track := p.getLocalVideoTrack()
	if track == nil {
		return fmt.Errorf("ErrNilVideoTrack")
	}
	return p.writeRTP(packet, track)
}

// AddAudioRTP write rtp to local audio track
func (p *Peer) AddAudioRTP(packet *rtp.Packet) error {
	track := p.getLocalAudioTrack()
	if track == nil {
		return fmt.Errorf("ErrNilAudioTrack")
	}
	return p.writeRTP(packet, track)
}

// AddICECandidate to add candidate
func (p *Peer) AddICECandidate(icecandidate interface{}) error {
	var candidateInit webrtc.ICECandidateInit
	err := mapstructure.Decode(icecandidate, &candidateInit)
	if err != nil {
		return err
	}

	conn := p.getConn()
	if conn == nil {
		p.addIceCache(&candidateInit)
		return fmt.Errorf("ErrNilPeerconnection")
	}

	if conn.RemoteDescription() == nil {
		p.addIceCache(&candidateInit)
	}

	return conn.AddICECandidate(candidateInit)
}

// CreateOffer add offer
func (p *Peer) CreateOffer(iceRestart bool) error {
	conn := p.getConn()
	if conn == nil {
		return fmt.Errorf("webrtc connection is nil")
	}

	// opt := &webrtc.OfferOptions{}

	// if iceRestart {
	// 	opt.ICERestart = iceRestart
	// }

	// set local desc
	offer, err := conn.CreateOffer(nil)
	if err != nil {
		return err
	}

	err = conn.SetLocalDescription(offer)
	if err != nil {
		return err
	}
	return nil
}

// CreateAnswer add answer
func (p *Peer) CreateAnswer() error {
	conn := p.getConn()
	if conn == nil {
		return fmt.Errorf("webrtc connection is nil")
	}
	// set local desc
	answer, err := conn.CreateAnswer(nil)
	if err != nil {
		return err
	}

	err = conn.SetLocalDescription(answer)
	if err != nil {
		return err
	}
	return nil
}

// AddSDP add sdp
func (p *Peer) AddSDP(values interface{}) error {
	conns := p.getConn()
	if conns == nil {
		return fmt.Errorf("ErrNilPeerconnection")
	}

	var data utils.SDPTemp
	err := mapstructure.Decode(values, &data)
	if err != nil {
		return err
	}

	sdp := &webrtc.SessionDescription{
		Type: NewSDPType(data.Type),
		SDP:  data.SDP,
	}

	switch data.Type {
	case "offer":
		if err := p.addOffer(sdp); err != nil {
			return err
		}
		break
	case "answer":
		if err := p.addAnswer(sdp); err != nil {
			return err
		}
		break
	default:
		return fmt.Errorf("Invalid sdp type: %s", data.Type)
	}
	return nil
}
