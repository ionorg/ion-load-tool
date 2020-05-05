package producer

import (
	"io"
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/pion/webrtc/v2"
	"github.com/pion/webrtc/v2/pkg/media"
	"github.com/pion/webrtc/v2/pkg/media/ivfreader"
)

// TODO
// Loop at end of track
// Random open position

type FileProducer struct {
	name    string
	stop    bool
	Samples chan media.Sample
	Track   *webrtc.Track
}

func NewFileProducer(name string) *FileProducer {
	pc, err := webrtc.NewPeerConnection(webrtc.Configuration{})
	if err != nil {
		log.Fatal(err)
	}

	// Create track
	videoTrack, err := pc.NewTrack(webrtc.DefaultPayloadTypeVP8, rand.Uint32(), "video", "video")
	if err != nil {
		panic(err)
	}

	return &FileProducer{
		name:    name,
		Samples: make(chan media.Sample),
		Track:   videoTrack,
	}
}

func (t *FileProducer) Stop() {
	t.stop = true
}

func (t *FileProducer) ReadLoop(startSeekFrames int) {
	file, err := os.Open(t.name)
	if err != nil {
		panic(err)
	}
	defer file.Close()
	ivf, header, ivfErr := ivfreader.NewWith(file)
	if ivfErr != nil {
		panic(ivfErr)
	}

	// Discard frames
	for i := 0; i < startSeekFrames; i++ {
		// TODO check for errors
		ivf.ParseNextFrame()
	}

	// Send our video file frame at a time. Pace our sending so we send it at the same speed it should be played back as.
	// This isn't required since the video is timestamped, but we will such much higher loss if we send all at once.
	sleepTime := time.Millisecond * time.Duration((float32(header.TimebaseNumerator)/float32(header.TimebaseDenominator))*1000)
	log.Println("Sleep time", sleepTime)
	for !t.stop {
		// Push sample
		frame, _, ivfErr := ivf.ParseNextFrame()
		if ivfErr == io.EOF {
			log.Println("All frames parsed and sent. Restart file")
			// TODO cleanup
			file.Seek(0, 0)
			ivf, header, ivfErr = ivfreader.NewWith(file)
			if ivfErr != nil {
				panic(ivfErr)
			}
			continue
		}

		if ivfErr != nil {
			log.Println("IVF error", ivfErr)
		}

		time.Sleep(sleepTime)
		if ivfErr = t.Track.WriteSample(media.Sample{Data: frame, Samples: 90000}); ivfErr != nil {
			log.Println("Track write error", ivfErr)
		}
	}
}
