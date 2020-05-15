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
	trackMap      map[uint]*trackInfo
	videoCodec    string
	file          *os.File
}

func NewMFileProducer(name string, offset int, ts TrackSelect) *WebMProducer {
	r, err := os.Open(name)
	if err != nil {
		log.Fatal("unable to open file", name)
	}
	var w webm.WebM
	reader, err := webm.Parse(r, &w)
	if err != nil {
		panic(err)
	}

	fileReader := &WebMProducer{
		name:          name,
		offsetSeconds: offset,
		reader:        reader,
		webm:          w,
		file:          r,
	}

	fileReader.buildTracks(ts)

	return fileReader
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
	t.file.Close()
}

func (t *WebMProducer) Start() {
	go t.readLoop()
}

func (t *WebMProducer) VideoCodec() string {
	return t.videoCodec
}

type trackInfo struct {
	track     *webrtc.Track
	rate      int
	lastFrame time.Duration
}

func (t *WebMProducer) buildTracks(ts TrackSelect) {
	pc, err := webrtc.NewPeerConnection(webrtc.Configuration{})
	if err != nil {
		log.Fatal(err)
	}

	trackMap := make(map[uint]*trackInfo)

	if ts.Video {
		if vidTrack := t.webm.FindFirstVideoTrack(); vidTrack != nil {
			log.Println("Video codec", vidTrack.CodecID)

			var vidCodedID uint8
			switch vidTrack.CodecID {
			case "V_VP8":
				vidCodedID = webrtc.DefaultPayloadTypeVP8
				t.videoCodec = "VP8"
			case "V_VP9":
				vidCodedID = webrtc.DefaultPayloadTypeVP9
				t.videoCodec = "VP9"
			default:
				log.Fatal("Unsupported video codec", vidTrack.CodecID)
			}

			videoTrack, err := pc.NewTrack(vidCodedID, rand.Uint32(), "video", "video")
			if err != nil {
				panic(err)
			}

			trackMap[vidTrack.TrackNumber] = &trackInfo{track: videoTrack, rate: 90000}
			t.videoTrack = videoTrack
		}
	}

	if ts.Audio {
		if audTrack := t.webm.FindFirstAudioTrack(); audTrack != nil {
			audioTrack, err := pc.NewTrack(webrtc.DefaultPayloadTypeOpus, rand.Uint32(), "audio", "video")
			if err != nil {
				panic(err)
			}

			trackMap[audTrack.TrackNumber] = &trackInfo{
				track: audioTrack,
				rate:  int(audTrack.Audio.OutputSamplingFrequency),
			}
			t.audioTrack = audioTrack
		}
	}

	t.trackMap = trackMap
}

func (t *WebMProducer) readLoop() {
	startDuration := time.Duration(t.offsetSeconds)
	skipDuration := startDuration * time.Second

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

		if track, ok := t.trackMap[pck.TrackNumber]; ok {
			// Calc frame time diff per track
			diff := pck.Timecode - track.lastFrame
			ms := float64(diff.Milliseconds()) / 1000.0
			samps := uint32(float64(track.rate) * ms)
			track.lastFrame = pck.Timecode

			// Send samples
			if ivfErr := track.track.WriteSample(media.Sample{Data: pck.Data, Samples: samps}); ivfErr != nil {
				log.Println("Track write error", ivfErr)
			}
		}
	}
	log.Println("Exiting webm producer")
}
