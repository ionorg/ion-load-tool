package gst

import (
	"log"
	"math/rand"

	"github.com/pion/rtwatch/gst"
	"github.com/pion/webrtc/v2"
)

type GSTProducer struct {
	name       string
	audioTrack *webrtc.Track
	videoTrack *webrtc.Track
	pipeline   *gst.Pipeline
	paused     bool
}

func NewGSTProducer(path string) *GSTProducer {

	pc, err := webrtc.NewPeerConnection(webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{
				URLs: []string{"stun:stun.l.google.com:19302"},
			},
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	videoTrack, err := pc.NewTrack(webrtc.DefaultPayloadTypeH264, rand.Uint32(), "synced-video", "synced-video")
	if err != nil {
		log.Fatal(err)
	}

	audioTrack, err := pc.NewTrack(webrtc.DefaultPayloadTypeOpus, rand.Uint32(), "synced-audio", "synced-video")
	if err != nil {
		log.Fatal(err)
	}

	pipeline := gst.CreatePipeline(path, audioTrack, videoTrack)

	return &GSTProducer{
		videoTrack: videoTrack,
		audioTrack: audioTrack,
		pipeline:   pipeline,
	}
}

func (t *GSTProducer) AudioTrack() *webrtc.Track {
	return t.audioTrack
}

func (t *GSTProducer) VideoTrack() *webrtc.Track {
	return t.videoTrack
}

func (t *GSTProducer) SeekP(ts int) {
	t.pipeline.SeekToTime(int64(ts))
}

func (t *GSTProducer) Pause(pause bool) {
	if pause {
		t.pipeline.Pause()
	} else {
		t.pipeline.Play()
	}
}

func (t *GSTProducer) Stop() {
}

func (t *GSTProducer) Start() {
	t.pipeline.Start()
}

func (t *GSTProducer) VideoCodec() string {
	return webrtc.H264
}
