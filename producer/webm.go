package producer

import (
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/ebml-go/webm"
	"github.com/pion/webrtc/v2"
	"github.com/pion/webrtc/v2/pkg/media"
)

type IFileProducer interface {
	VideoTrack() *webrtc.Track
	AudioTrack() *webrtc.Track
	Stop()
	Start()
}

type WebMProducer struct {
	name          string
	stop          bool
	videoTrack    *webrtc.Track
	audioTrack    *webrtc.Track
	offsetSeconds int
	reader        *webm.Reader
	webm          webm.WebM
}

func NewMFileProducer(name string, offset int) *WebMProducer {
	pc, err := webrtc.NewPeerConnection(webrtc.Configuration{})
	if err != nil {
		log.Fatal(err)
	}

	// Create track
	videoTrack, err := pc.NewTrack(webrtc.DefaultPayloadTypeVP8, rand.Uint32(), "video", "video")
	if err != nil {
		panic(err)
	}

	audioTrack, err := pc.NewTrack(webrtc.DefaultPayloadTypeOpus, rand.Uint32(), "video", "video")
	if err != nil {
		panic(err)
	}

	r, err := os.Open(name)
	if err != nil {
		log.Fatal("unable to open file", name)
	}
	var w webm.WebM
	reader, err := webm.Parse(r, &w)
	if err != nil {
		panic(err)
	}

	return &WebMProducer{
		name:          name,
		videoTrack:    videoTrack,
		audioTrack:    audioTrack,
		offsetSeconds: offset,
		reader:        reader,
		webm:          w,
	}
}

func (t *WebMProducer) AudioTrack() *webrtc.Track {
	return t.audioTrack
}

func (t *WebMProducer) VideoTrack() *webrtc.Track {
	return t.videoTrack
}

func (t *WebMProducer) Stop() {
	t.stop = true
	t.reader.Shutdown()
}

func (t *WebMProducer) Start() {
	go t.ReadLoop()
}

func (t *WebMProducer) ReadLoop() {
	startDuration := time.Duration(t.offsetSeconds)
	skipDuration := startDuration * time.Second

	vidTrack := t.webm.FindFirstVideoTrack()
	vidNum := vidTrack.TrackNumber

	setStartTime := func() time.Time {
		return time.Now().Add(-startDuration * time.Second)
	}
	startTime := setStartTime()
	first := true

	for pck := range t.reader.Chan {
		if pck.Timecode < 0 {
			if !t.stop {
				log.Println("Restart media")
				t.reader.Seek(0)
				first = false
				startTime = time.Now()
			}
			continue
		} else if first && pck.Timecode < skipDuration {
			startTime = setStartTime()
			continue
		}

		timeDiff := pck.Timecode - time.Since(startTime)
		// TODO if less than some min just send now
		if timeDiff > 0 {
			time.Sleep(timeDiff)
		}

		// TODO send audio tracks
		// GET if from first of each
		if pck.TrackNumber == vidNum {
			if ivfErr := t.videoTrack.WriteSample(media.Sample{Data: pck.Data, Samples: 1}); ivfErr != nil {
				log.Println("Track write error", ivfErr)
			}
		}

	}
	log.Println("Exiting webm producer")
}
