package webm

import (
	"log"
	"math"
	"math/rand"
	"os"
	"time"

	"github.com/ebml-go/webm"
	"github.com/pion/ion-load-tool/producer"
	"github.com/pion/webrtc/v2"
	"github.com/pion/webrtc/v2/pkg/media"
)

type WebMProducer struct {
	name          string
	stop          bool
	paused        bool
	pauseChan     chan bool
	seekChan      chan time.Duration
	videoTrack    *webrtc.Track
	audioTrack    *webrtc.Track
	offsetSeconds int
	reader        *webm.Reader
	webm          webm.WebM
	trackMap      map[uint]*trackInfo
	videoCodec    string
	file          *os.File
}

func NewMFileProducer(name string, offset int, ts producer.TrackSelect) *WebMProducer {
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
		pauseChan:     make(chan bool),
		seekChan:      make(chan time.Duration, 1),
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
}

func (t *WebMProducer) Start() {
	go t.readLoop()
}

func (t *WebMProducer) SeekP(ts int) {
	seekDuration := time.Duration(ts) * time.Second
	t.seekChan <- seekDuration
}

func (t *WebMProducer) Pause(pause bool) {
	t.pauseChan <- pause
}

func (t *WebMProducer) VideoCodec() string {
	return t.videoCodec
}

type trackInfo struct {
	track         *webrtc.Track
	rate          int
	lastFrameTime time.Duration
}

func (t *WebMProducer) buildTracks(ts producer.TrackSelect) {
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
				t.videoCodec = webrtc.VP8
			case "V_VP9":
				vidCodedID = webrtc.DefaultPayloadTypeVP9
				t.videoCodec = webrtc.VP9
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
	startTime := time.Now()
	timeEps := 5 * time.Millisecond

	seekDuration := time.Duration(-1)

	if t.offsetSeconds > 0 {
		t.SeekP(t.offsetSeconds)
	}

	startSeek := func(seekTime time.Duration) {
		t.reader.Seek(seekTime)
		seekDuration = seekTime
	}

	for pck := range t.reader.Chan {
		if t.paused {
			log.Println("Paused")
			// Wait for unpause
			for pause := range t.pauseChan {
				if !pause {
					t.paused = false
					break
				}
			}
			log.Println("Unpaused")
			startTime = time.Now().Add(-pck.Timecode)
		}

		// Restart when track runs out
		if pck.Timecode < 0 {
			if !t.stop {
				log.Println("Restart media")
				startSeek(0)
			}
			continue
		}

		// Handle seek and pause
		select {
		case dur := <-t.seekChan:
			log.Println("Seek duration", dur)
			startSeek(dur)
			continue
		case pause := <-t.pauseChan:
			t.paused = pause
			if pause {
				continue
			}
		default:
		}

		// Handle actual seek
		if seekDuration > -1 && math.Abs(float64((pck.Timecode-seekDuration).Milliseconds())) < 30.0 {
			log.Println("Seek happened!!!!")
			startTime = time.Now().Add(-seekDuration)
			seekDuration = time.Duration(-1)
			// Clear frame count tracking
			for _, t := range t.trackMap {
				t.lastFrameTime = 0
			}
			continue
		}

		// Find sender
		if track, ok := t.trackMap[pck.TrackNumber]; ok {
			// Only delay frames we care about
			timeDiff := pck.Timecode - time.Since(startTime)
			if timeDiff > timeEps {
				time.Sleep(timeDiff - time.Millisecond)
			}

			// Calc frame time diff per track
			diff := pck.Timecode - track.lastFrameTime
			ms := float64(diff.Milliseconds()) / 1000.0
			samps := uint32(float64(track.rate) * ms)
			track.lastFrameTime = pck.Timecode

			// Send samples
			if ivfErr := track.track.WriteSample(media.Sample{Data: pck.Data, Samples: samps}); ivfErr != nil {
				log.Println("Track write error", ivfErr)
			}
		}
	}
	log.Println("Exiting webm producer")
}
