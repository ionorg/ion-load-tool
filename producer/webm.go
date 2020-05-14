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

type WebMProducer struct {
	name          string
	stop          bool
	videoTrack    *webrtc.Track
	audioTrack    *webrtc.Track
	offsetSeconds int
	reader        *webm.Reader
	webm          webm.WebM
}

func NewMFileProducer(name string, offset int, ts TrackSelect) *WebMProducer {
	pc, err := webrtc.NewPeerConnection(webrtc.Configuration{})
	if err != nil {
		log.Fatal(err)
	}

	var videoTrack, audioTrack *webrtc.Track

	// Create track
	if ts.Video {
		videoTrack, err = pc.NewTrack(webrtc.DefaultPayloadTypeVP8, rand.Uint32(), "video", "video")
		if err != nil {
			panic(err)
		}
	}

	if ts.Audio {
		audioTrack, err = pc.NewTrack(webrtc.DefaultPayloadTypeOpus, rand.Uint32(), "video", "video")
		if err != nil {
			panic(err)
		}
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
	go t.readLoop()
}

func (t *WebMProducer) buildTracks() map[uint]*webrtc.Track {
	trackMap := make(map[uint]*webrtc.Track)

	if t.videoTrack != nil {
		if vidTrack := t.webm.FindFirstVideoTrack(); vidTrack != nil {
			trackMap[vidTrack.TrackNumber] = t.videoTrack
		}
	}

	if t.audioTrack != nil {
		if audTrack := t.webm.FindFirstAudioTrack(); audTrack != nil {
			trackMap[audTrack.TrackNumber] = t.audioTrack
		}
	}

	return trackMap
}

func (t *WebMProducer) readLoop() {
	startDuration := time.Duration(t.offsetSeconds)
	skipDuration := startDuration * time.Second

	trackMap := t.buildTracks()

	setStartTime := func() time.Time {
		return time.Now().Add(-startDuration * time.Second)
	}
	startTime := setStartTime()
	first := true

	timeEps := 5 * time.Millisecond

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
		if timeDiff > timeEps {
			time.Sleep(timeDiff - time.Millisecond)
		}

		if track, ok := trackMap[pck.TrackNumber]; ok {
			if ivfErr := track.WriteSample(media.Sample{Data: pck.Data, Samples: 1}); ivfErr != nil {
				log.Println("Track write error", ivfErr)
			}
		}
	}
	log.Println("Exiting webm producer")
}
