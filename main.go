package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/cloudwebrtc/go-protoo/logger"
	"github.com/pion/ion-load-tool/ion"
	"github.com/pion/ion-load-tool/producer"
)

var (
	waitGroup      sync.WaitGroup
	clientNameTmpl = "client_%v"
)

func init() {
	logger.SetLevel(logger.InfoLevel)
}

type testRun struct {
	client      ion.RoomClient
	consume     bool
	produce     bool
	mediaSource producer.IFileProducer
	doneCh      chan interface{}
	index       int
}

func (t *testRun) runClient() {
	defer waitGroup.Done()

	t.client.Init()
	t.client.Join()

	// Start producer
	if t.produce {
		t.client.Publish(t.mediaSource.VideoCodec())
	}

	// Wire consumers
	// Wait for the end of the test then shutdown
	done := false
	for !done {
		select {
		case msg := <-t.client.OnStreamAdd:
			if t.consume {
				t.client.Subscribe(msg.MediaInfo)
			}
		case msg := <-t.client.OnStreamRemove:
			if t.consume {
				t.client.UnSubscribe(msg.MediaInfo)
			}
		case <-t.client.OnBroadcast:
		case <-t.doneCh:
			done = true
			break
		}
	}
	log.Printf("Begin client %v shutdown", t.index)

	// Close producer and sender
	if t.produce {
		t.mediaSource.Stop()
		t.client.UnPublish()
	}

	// Close client
	t.client.Leave()
	t.client.Close()
}

func (t *testRun) setupClient(room, path, vidFile, fileType string, audio bool) {
	name := fmt.Sprintf(clientNameTmpl, t.index)
	t.client = ion.NewClient(name, room, path)
	t.doneCh = make(chan interface{})

	if t.produce {
		// Configure sender tracks
		offset := t.index * 5
		if fileType == "webm" {
			t.mediaSource = producer.NewMFileProducer(vidFile, offset, producer.TrackSelect{
				Audio: audio,
				Video: true,
			})
		} else if fileType == "ivf" {
			audio = false
			t.mediaSource = producer.NewIVFProducer(vidFile, offset)
		}
		t.client.VideoTrack = t.mediaSource.VideoTrack()
		if audio {
			t.client.AudioTrack = t.mediaSource.AudioTrack()
		}
		t.mediaSource.Start()
	}

	go t.runClient()
}

func main() {
	var containerPath, containerType string
	var ionPath, roomName string
	var numClients, runSeconds int
	var consume, produce bool
	var staggerSeconds float64
	var audio bool

	flag.StringVar(&containerPath, "produce", "", "path to the media file you want to playback")
	flag.StringVar(&ionPath, "ion-url", "ws://localhost:8443/ws", "websocket url for ion biz system")
	flag.StringVar(&roomName, "room", "video-demo", "Room name for Ion")
	flag.IntVar(&numClients, "clients", 1, "Number of clients to start")
	flag.Float64Var(&staggerSeconds, "stagger", 1.0, "Number of seconds to stagger client start and stop")
	flag.IntVar(&runSeconds, "seconds", 60, "Number of seconds to run test for")
	flag.BoolVar(&consume, "consume", false, "Run subscribe to all streams and consume data")
	flag.BoolVar(&audio, "audio", false, "Publish audio stream from webm file")

	flag.Parse()

	produce = containerPath != ""

	// Validate type
	if produce {
		ext, ok := producer.ValidateVPFile(containerPath)
		log.Println(ext)
		if !ok {
			panic("Only IVF and WEBM containers are supported.")
		}
		containerType = ext
	}

	clients := make([]*testRun, numClients)
	staggerDur := time.Duration(staggerSeconds) * time.Second
	waitGroup.Add(numClients)

	for i := 0; i < numClients; i++ {
		cfg := &testRun{consume: consume, produce: produce, index: i}
		cfg.setupClient(roomName, ionPath, containerPath, containerType, audio)
		clients[i] = cfg
		time.Sleep(staggerDur)
	}

	// Setup shutdown
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	timer := time.NewTimer(time.Duration(runSeconds) * time.Second)

	select {
	case <-sigs:
	case <-timer.C:
	}

	for i, a := range clients {
		// Signal shutdown
		close(a.doneCh)
		// Staggered shutdown.
		if len(clients) > 1 && i < len(clients)-1 {
			time.Sleep(staggerDur)
		}
	}

	log.Println("Wait for client shutdown")
	waitGroup.Wait()
	log.Println("All clients shut down")
}
