package main

import (
	"fmt"
	"log"
	"math/rand"
	"net/url"
	"sync"
	"time"

	"github.com/beowulflab/rtcbase/utils"
	signal "github.com/beowulflab/signal/livestream"
	"github.com/mitchellh/mapstructure"
	"github.com/pion/rtcp"
	"github.com/pion/sdp/v2"
	"github.com/pion/webrtc/v3"
)

// GetSignalURL linter
func GetSignalURL(id string) string {
	return fmt.Sprintf("wss://signal-test.dechen.app?id=%s", id)
}

// Node linter
type Node struct {
	iceCache        *utils.AdvanceMap // save all ice before set remote description
	localVideoTrack *webrtc.Track
	localAudioTrack *webrtc.Track
	signl           *signal.Signal
	conn            *webrtc.PeerConnection
	mutex           sync.RWMutex
}

// NewNode linter
func NewNode(id string) *Node {
	n := &Node{
		iceCache: utils.NewAdvanceMap(),
	}
	signal := signal.NewSignal(id, GetSignalURL(id), n.handleMsg)
	signal.Start()
	n.signl = signal
	return n
}

func (n *Node) handleMsg(values []interface{}) {
	signalID, hasSignalID := values[0].(string)
	if !hasSignalID {
		return
	}

	event, isEvent := values[1].(string)
	if !isEvent {
		return
	}

	switch event {
	case "ok":
		if s := n.getSignal(); s != nil {
			s.Send(signalID, "ok")
		}
		break
	case "sdp":
		// conn := n.getConn()
		// if conn == nil {
		// 	n.NewConn(signalID)
		// 	conn = n.getConn()
		// }
		// n.handlePeerEvent(signalID, conn)
		// if err := n.AddSDP(values[2], signalID); err != nil {
		// 	log.Println(fmt.Sprintf("Wss sdp err: %s", err.Error()))
		// }
		if err := n.handleSDPEvent(signalID, values[2]); err != nil {
			log.Println(fmt.Sprintf("Wss sdp err: %s", err.Error()))
		}
		break
	case "reconnect":
		break
	case "candidate":
		if err := n.AddCandidate(values[2]); err != nil {
			log.Println(fmt.Sprintf("Wss candidate err: %s", err.Error()))
		}
		break
	default:
		errStr := fmt.Sprintf("[ProcessSignal] receive not processing event: %s", event)
		fmt.Println(errStr)
	}
}

// NewConn create new peer connectiom
func (n *Node) NewConn(signalID string) error {

	mediaEngine := &webrtc.MediaEngine{}
	mediaEngine.RegisterCodec(webrtc.NewRTPOpusCodec(webrtc.DefaultPayloadTypeOpus, 48000))
	mediaEngine.RegisterCodec(webrtc.NewRTPVP8Codec(webrtc.DefaultPayloadTypeVP8, 90000))

	settingEngine := &webrtc.SettingEngine{}
	settingEngine.SetEphemeralUDPPortRange(20000, 60000)
	settingEngine.SetICETimeouts(10*time.Second, 20*time.Second, 1*time.Second)

	api := webrtc.NewAPI(webrtc.WithMediaEngine(*mediaEngine), webrtc.WithSettingEngine(*settingEngine))

	conn, err := api.NewPeerConnection(*getTurns())
	if err != nil {
		return err
	}

	n.setConn(conn)

	if err := n.CreateAudioTrack("audio"); err != nil {
		return err
	}

	if err := n.CreateVideoTrack("video"); err != nil {
		return err
	}

	return nil
}

// CreateAudioTrack linter
func (n *Node) CreateAudioTrack(trackID string) error {
	if conn := n.getConn(); conn != nil {
		localTrack, err := conn.NewTrack(webrtc.DefaultPayloadTypeOpus, rand.Uint32(), trackID, trackID)
		if err != nil {
			return err
		}
		// Add this newly created track to the PeerConnection
		_, err = conn.AddTrack(localTrack)
		if err != nil {
			return err
		}
		n.setLocalAudioTrack(localTrack)
		return nil
	}
	return fmt.Errorf("cannot create audio track because rtc connection is nil")
}

// CreateVideoTrack linter
func (n *Node) CreateVideoTrack(seatID string) error {
	if conn := n.getConn(); conn != nil {
		localTrack, err := conn.NewTrack(webrtc.DefaultPayloadTypeVP8, rand.Uint32(), seatID, seatID)
		if err != nil {
			return err
		}
		// Add this newly created track to the PeerConnection
		_, err = conn.AddTrack(localTrack)
		if err != nil {
			return err
		}
		n.setLocalVideoTrack(localTrack)
		return nil
	}
	return fmt.Errorf("cannot create video track because rtc connection is nil")
}

func (n *Node) getLocalAudioTrack() *webrtc.Track {
	return n.localAudioTrack
}

func (n *Node) setLocalAudioTrack(t *webrtc.Track) {
	n.mutex.Lock()
	defer n.mutex.Unlock()
	n.localAudioTrack = t
}

func (n *Node) getLocalVideoTrack() *webrtc.Track {
	n.mutex.RLock()
	defer n.mutex.RUnlock()
	return n.localVideoTrack
}

func (n *Node) setLocalVideoTrack(t *webrtc.Track) {
	n.mutex.Lock()
	defer n.mutex.Unlock()
	n.localVideoTrack = t
}

func (n *Node) handlePeerEvent(signalID string, conn *webrtc.PeerConnection) {
	conn.OnICECandidate(func(i *webrtc.ICECandidate) {
		if i == nil {
			return
		}

		// send to signal
		// ["id", "event", "data"]

		if wss := n.getSignal(); wss != nil {
			wss.Send(signalID, "candidate", i.ToJSON())
		}
	})

	conn.OnICEConnectionStateChange(func(is webrtc.ICEConnectionState) {
		log.Println(fmt.Sprintf("Current connection states: %s", is.String()))
	})

	conn.OnTrack(func(remoteTrack *webrtc.Track, r *webrtc.RTPReceiver) {
		var localTrack *webrtc.Track

		fmt.Println(remoteTrack.Codec())
		fmt.Println("RID: ", remoteTrack.RID())
		if remoteTrack.Kind().String() == "video" {
			go func() {
				ticker := time.NewTicker(time.Millisecond * 500)
				for range ticker.C {
					conn := n.getConn()
					if conn == nil {
						return
					}
					errSend := conn.WriteRTCP([]rtcp.Packet{&rtcp.PictureLossIndication{MediaSSRC: remoteTrack.SSRC()}})
					if errSend != nil {
						return
					}

					if routineErr := conn.WriteRTCP([]rtcp.Packet{&rtcp.RapidResynchronizationRequest{SenderSSRC: remoteTrack.SSRC(), MediaSSRC: remoteTrack.SSRC()}}); routineErr != nil {
						return
					}
				}
			}()

			localTrack = n.getLocalVideoTrack()
		} else {
			localTrack = n.getLocalAudioTrack()
		}

		for {
			rtp, err := remoteTrack.ReadRTP()
			if err != nil {
				log.Println(err.Error())
				return
			}

			// rtp.PayloadType = localTrack.PayloadType()
			rtp.SSRC = localTrack.SSRC()
			err = localTrack.WriteRTP(rtp)
			if err != nil {
				log.Println(fmt.Sprintf("%s localtrack write rtp err: %s", localTrack.Kind().String(), err.Error()))
			}
		}
	})
}

// CreateAnswer add answer
func (n *Node) CreateAnswer() error {
	conn := n.getConn()
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

// CreateOffer add offer
func (n *Node) CreateOffer() error {
	conn := n.getConn()
	if conn == nil {
		return fmt.Errorf("webrtc connection is nil")
	}

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

// AddSDP linter
func (n *Node) AddSDP(values interface{}, signalID string) error {
	conns := n.getConn()
	if conns == nil {
		return fmt.Errorf("peer connection is nil")
	}

	var data utils.SDPTemp
	err := mapstructure.Decode(values, &data)
	if err != nil {
		return err
	}

	sdp := &webrtc.SessionDescription{
		Type: utils.NewSDPType(data.Type),
		SDP:  data.SDP,
	}

	switch data.Type {
	case "offer":
		if err := n.AddOffer(sdp); err != nil {
			return err
		}
		// send sdp back
		answer, err := n.GetLocalDescription()
		if err != nil {
			return err
		}

		if signal := n.getSignal(); signal != nil {
			signal.Send(signalID, "sdp", answer)
		}
		break
	case "answer":
		if err := n.AddAnswer(sdp); err != nil {
			return err
		}
		break
	default:
		return fmt.Errorf("[AddSDP ]Invalid sdp type: %s", data.Type)
	}
	return nil
}

// AddCandidate to add streamer candidate
func (n *Node) AddCandidate(icecandidate interface{}) error {
	var candidateInit webrtc.ICECandidateInit
	err := mapstructure.Decode(icecandidate, &candidateInit)
	if err != nil {
		return err
	}

	conn := n.getConn()
	if conn == nil {
		n.addIceCache(&candidateInit)
		return fmt.Errorf("Peer connection is nil")
	}

	if conn.RemoteDescription() == nil {
		n.addIceCache(&candidateInit)
	}
	return conn.AddICECandidate(candidateInit)
}

// AddOffer add client offer and return answer
func (n *Node) AddOffer(offer *webrtc.SessionDescription) error {
	conn := n.getConn()
	if conn == nil {
		return fmt.Errorf("rtc connection is nil")
	}

	//set remote desc
	err := conn.SetRemoteDescription(*offer)
	if err != nil {
		return err
	}

	err = n.setCacheIce()
	if err != nil {
		return err
	}

	err = n.CreateAnswer()
	if err != nil {
		return err
	}

	return nil
}

// AddAnswer add client answer and set remote desc
func (n *Node) AddAnswer(answer *webrtc.SessionDescription) error {
	conn := n.getConn()
	if conn == nil {
		return fmt.Errorf("Peer connection is nil")
	}

	//set remote desc
	err := conn.SetRemoteDescription(*answer)
	if err != nil {
		return err
	}
	return n.setCacheIce()
}

// GetLocalDescription get current peer local description
func (n *Node) GetLocalDescription() (*webrtc.SessionDescription, error) {
	conn := n.getConn()
	if conn == nil {
		return nil, fmt.Errorf("rtc connection is nil")
	}
	return conn.LocalDescription(), nil
}

func (n *Node) handleSDPEvent(signalID string, values interface{}) error {
	// parse sdp
	var data utils.SDPTemp
	err := mapstructure.Decode(values, &data)
	if err != nil {
		return err
	}

	offer := &webrtc.SessionDescription{
		Type: utils.NewSDPType(data.Type),
		SDP:  data.SDP,
	}

	mediaEngine := &webrtc.MediaEngine{}
	// mediaEngine.RegisterCodec(webrtc.NewRTPOpusCodec(webrtc.DefaultPayloadTypeOpus, 48000))
	// mediaEngine.RegisterCodec(webrtc.NewRTPVP8Codec(webrtc.DefaultPayloadTypeVP8, 90000))

	// parse sdp
	err = mediaEngine.PopulateFromSDP(*offer)
	if err != nil {
		panic(err)
	}

	videoCodecs := mediaEngine.GetCodecsByKind(webrtc.RTPCodecTypeVideo)
	if len(videoCodecs) == 0 {
		panic("Offer contained no video codecs")
	}

	//Configure required extensions

	sdes, _ := url.Parse(sdp.SDESRTPStreamIDURI)
	sdedMid, _ := url.Parse(sdp.SDESMidURI)
	exts := []sdp.ExtMap{
		{
			URI: sdes,
		},
		{
			URI: sdedMid,
		},
	}

	se := webrtc.SettingEngine{}
	se.AddSDPExtensions(webrtc.SDPSectionVideo, exts)
	se.AddSDPExtensions(webrtc.SDPSectionAudio, exts)

	// add setting engine
	settingEngine := &webrtc.SettingEngine{}
	// settingEngine.SetEphemeralUDPPortRange(20000, 60000)
	settingEngine.SetICETimeouts(10*time.Second, 20*time.Second, 1*time.Second)

	api := webrtc.NewAPI(webrtc.WithMediaEngine(*mediaEngine), webrtc.WithSettingEngine((se)))

	// Create a new RTCPeerConnection

	conn := n.getConn()
	if conn == nil {
		conn, err = api.NewPeerConnection(*getTurns())
		if err != nil {
			panic(err)
		}
		n.setConn(conn)

		if err := n.CreateAudioTrack("audio"); err != nil {
			return err
		}

		if err := n.CreateVideoTrack("video"); err != nil {
			return err
		}

	}

	n.handlePeerEvent(signalID, conn)
	n.AddSDP(values, signalID)
	return nil
}
