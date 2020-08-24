package main

import (
	"fmt"
	"math/rand"
	"net/url"
	"time"

	"github.com/beowulflab/rtcbase/utils"
	log "github.com/lamhai1401/gologs/logs"
	"github.com/mitchellh/mapstructure"
	"github.com/pion/rtcp"
	"github.com/pion/rtp"
	"github.com/pion/sdp/v2"
	"github.com/pion/webrtc/v3"
)

func (p *Peer) addAPIWithSDP(values interface{}) (*webrtc.API, error) {
	// parse sdp
	var data utils.SDPTemp
	err := mapstructure.Decode(values, &data)
	if err != nil {
		return nil, err
	}

	offer := &webrtc.SessionDescription{
		Type: utils.NewSDPType(data.Type),
		SDP:  data.SDP,
	}

	mediaEngine := &webrtc.MediaEngine{}

	err = mediaEngine.PopulateFromSDP(*offer)
	if err != nil {
		panic(err)
	}

	videoCodecs := mediaEngine.GetCodecsByKind(webrtc.RTPCodecTypeVideo)
	if len(videoCodecs) == 0 {
		panic("Offer contained no video codecs")
	}

	audioCodecs := mediaEngine.GetCodecsByKind(webrtc.RTPCodecTypeAudio)
	if len(audioCodecs) == 0 {
		panic("Offer contained no audio codecs")
	}

	p.setAudioCodecs(audioCodecs[0].PayloadType)
	p.setVideoCodecs(videoCodecs[0].PayloadType)

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

	se := p.initSettingEngine()
	se.AddSDPExtensions(webrtc.SDPSectionVideo, exts)
	se.AddSDPExtensions(webrtc.SDPSectionAudio, exts)

	api := webrtc.NewAPI(webrtc.WithMediaEngine(*mediaEngine), webrtc.WithSettingEngine((*se)))

	return api, nil
}

func (p *Peer) newConnection(sdp interface{}, config *webrtc.Configuration) (*webrtc.PeerConnection, error) {
	if sdp == nil {
		api := p.addAPI()
		conn, err := api.NewPeerConnection(*config)
		if err != nil {
			return nil, err
		}
		p.setConn(conn)

		if err := p.createAudioTrack(p.getSessionID(), p.getAudioCodecs()); err != nil {
			return nil, err
		}

		if err := p.createVideoTrack(p.getSessionID(), p.getVideoCodecs()); err != nil {
			return nil, err
		}

		return conn, nil
	}
	api, err := p.addAPIWithSDP(sdp)
	if err != nil {
		return nil, err
	}

	conn, err := api.NewPeerConnection(*config)
	p.setConn(conn)

	if err := p.createAudioTrack(p.getSessionID(), webrtc.DefaultPayloadTypeOpus); err != nil {
		return nil, err
	}

	if err := p.createVideoTrack(p.getSessionID(), webrtc.DefaultPayloadTypeVP8); err != nil {
		return nil, err
	}

	return conn, nil
}

// NewAPI linter
func (p *Peer) addAPI() *webrtc.API {
	mediaEngine := &webrtc.MediaEngine{}
	mediaEngine.RegisterCodec(webrtc.NewRTPOpusCodec(webrtc.DefaultPayloadTypeOpus, 48000))
	mediaEngine.RegisterCodec(webrtc.NewRTPVP8Codec(webrtc.DefaultPayloadTypeVP8, 90000))

	settingEngine := &webrtc.SettingEngine{}
	// settingEngine.SetEphemeralUDPPortRange(20000, 60000)
	settingEngine.SetICETimeouts(10*time.Second, 20*time.Second, 1*time.Second)

	api := webrtc.NewAPI(webrtc.WithMediaEngine(*mediaEngine), webrtc.WithSettingEngine(*settingEngine))

	return api
}

func (p *Peer) initMediaEngine() *webrtc.MediaEngine {
	mediaEngine := &webrtc.MediaEngine{}
	mediaEngine.RegisterCodec(webrtc.NewRTPOpusCodec(webrtc.DefaultPayloadTypeOpus, 48000))
	mediaEngine.RegisterCodec(webrtc.NewRTPVP8Codec(webrtc.DefaultPayloadTypeVP8, 90000))
	return mediaEngine
}

func (p *Peer) initSettingEngine() *webrtc.SettingEngine {
	settingEngine := &webrtc.SettingEngine{}
	// settingEngine.SetEphemeralUDPPortRange(20000, 60000)
	// settingEngine.SetICETimeouts(10*time.Second, 20*time.Second, 1*time.Second)
	return settingEngine
}

func (p *Peer) setVideoCodecs(codecs uint8) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.videoCodecs = codecs
}

func (p *Peer) setAudioCodecs(codecs uint8) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.audioCodecs = codecs
}

func (p *Peer) getVideoCodecs() uint8 {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	return p.videoCodecs
}

func (p *Peer) getAudioCodecs() uint8 {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	return p.audioCodecs
}

func (p *Peer) writeRTP(packet *rtp.Packet, track *webrtc.Track) error {
	// packet.PayloadType = track.PayloadType()
	packet.SSRC = track.SSRC()
	packet.Header.PayloadType = track.PayloadType()
	return track.WriteRTP(packet)
}

// CreateAudioTrack linter
func (p *Peer) createAudioTrack(trackID string, codesc uint8) error {
	if conn := p.getConn(); conn != nil {
		localTrack, err := conn.NewTrack(codesc, rand.Uint32(), trackID, trackID)
		if err != nil {
			return err
		}
		// Add this newly created track to the PeerConnection
		_, err = conn.AddTrack(localTrack)
		if err != nil {
			return err
		}
		p.setLocalAudioTrack(localTrack)
		return nil
	}
	return fmt.Errorf("cannot create audio track because rtc connection is nil")
}

// CreateVideoTrack linter
func (p *Peer) createVideoTrack(trackID string, codesc uint8) error {
	if conn := p.getConn(); conn != nil {
		localTrack, err := conn.NewTrack(codesc, rand.Uint32(), trackID, trackID)
		if err != nil {
			return err
		}
		// Add this newly created track to the PeerConnection
		_, err = conn.AddTrack(localTrack)
		if err != nil {
			return err
		}
		p.setLocalVideoTrack(localTrack)
		return nil
	}
	return fmt.Errorf("cannot create video track because rtc connection is nil")
}

func (p *Peer) getConn() *webrtc.PeerConnection {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	return p.conn
}

func (p *Peer) setConn(c *webrtc.PeerConnection) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.conn = c
}

func (p *Peer) getSignalID() string {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	return p.signalID
}

func (p *Peer) closeConn() {
	if conn := p.getConn(); conn != nil {
		p.setConn(nil)
		conn.Close()
		conn = nil
	}
}

func (p *Peer) getLocalAudioTrack() *webrtc.Track {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	return p.localAudioTrack
}

func (p *Peer) setLocalAudioTrack(t *webrtc.Track) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.localAudioTrack = t
}

func (p *Peer) getLocalVideoTrack() *webrtc.Track {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	return p.localVideoTrack
}

func (p *Peer) setLocalVideoTrack(t *webrtc.Track) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.localVideoTrack = t
}

func (p *Peer) getSessionID() string {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	return p.sessionID
}

func (p *Peer) getBitrate() *int {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	return p.bitrate
}

// ModifyBitrate so set bitrate when datachannel has signal
// Use this only for video not audio track
func (p *Peer) modifyBitrate(remoteTrack *webrtc.Track) {
	ticker := time.NewTicker(time.Millisecond * 500)
	for range ticker.C {
		bitrate := p.getBitrate()
		if p.checkClose() || bitrate == nil {
			return
		}

		numbers := (*bitrate) * 1024
		if conn := p.getConn(); conn != nil {
			errSend := conn.WriteRTCP([]rtcp.Packet{&rtcp.ReceiverEstimatedMaximumBitrate{
				SenderSSRC: remoteTrack.SSRC(),
				Bitrate:    uint64(numbers),
				// SSRCs:      []uint32{rand.Uint32()},
			}})

			if errSend != nil {
				log.Error("Modify bitrate write rtcp err: ", errSend.Error())
				// return
			}
		}
	}
}

// PictureLossIndication packet informs the encoder about the loss of an undefined amount of coded video data belonging to one or more pictures
func (p *Peer) pictureLossIndication(remoteTrack *webrtc.Track) {
	ticker := time.NewTicker(time.Millisecond * 500)
	for range ticker.C {
		if p.checkClose() {
			return
		}

		conn := p.getConn()
		if conn == nil {
			return
		}
		errSend := conn.WriteRTCP([]rtcp.Packet{&rtcp.PictureLossIndication{MediaSSRC: remoteTrack.SSRC()}})
		if errSend != nil {
			log.Error("Picture loss indication write rtcp err: ", errSend.Error())
			// return
		}
	}
}

// RapidResynchronizationRequest packet informs the encoder about the loss of an undefined amount of coded video data belonging to one or more pictures
func (p *Peer) rapidResynchronizationRequest(remoteTrack *webrtc.Track) {
	ticker := time.NewTicker(time.Millisecond * 100)
	for range ticker.C {
		if p.checkClose() {
			return
		}

		conn := p.getConn()
		if conn == nil {
			return
		}
		if routineErr := conn.WriteRTCP([]rtcp.Packet{&rtcp.RapidResynchronizationRequest{SenderSSRC: remoteTrack.SSRC(), MediaSSRC: remoteTrack.SSRC()}}); routineErr != nil {
			log.Error("rapidResynchronizationRequest write rtcp err: ", routineErr.Error())
			// return
		}
	}
}

func (p *Peer) checkClose() bool {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	return p.isClosed
}

func (p *Peer) setClose(state bool) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.isClosed = state
}

func (p *Peer) getIceCache() *utils.AdvanceMap {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	return p.iceCache
}

func (p *Peer) addIceCache(ice *webrtc.ICECandidateInit) {
	if cache := p.getIceCache(); cache != nil {
		cache.Set(ice.Candidate, ice)
	}
}

// setCacheIce add ice save in cache
func (p *Peer) setCacheIce() error {
	cache := p.getIceCache()
	if cache == nil {
		return fmt.Errorf("ICE cache map is nil")
	}
	conn := p.getConn()
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

// GetLocalDescription get current peer local description
func (p *Peer) getLocalDescription() (*webrtc.SessionDescription, error) {
	conn := p.getConn()
	if conn == nil {
		return nil, fmt.Errorf("rtc connection is nil")
	}
	return conn.LocalDescription(), nil
}

// AddOffer add client offer and return answer
func (p *Peer) addOffer(offer *webrtc.SessionDescription) error {
	conn := p.getConn()
	if conn == nil {
		return fmt.Errorf("rtc connection is nil")
	}

	//set remote desc
	err := conn.SetRemoteDescription(*offer)
	if err != nil {
		return err
	}

	err = p.setCacheIce()
	if err != nil {
		return err
	}

	err = p.CreateAnswer()
	if err != nil {
		return err
	}

	return nil
}

// AddAnswer add client answer and set remote desc
func (p *Peer) addAnswer(answer *webrtc.SessionDescription) error {
	conn := p.getConn()
	if conn == nil {
		return fmt.Errorf("Peer connection is nil")
	}

	//set remote desc
	err := conn.SetRemoteDescription(*answer)
	if err != nil {
		return err
	}
	return p.setCacheIce()
}
