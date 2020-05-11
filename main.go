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
	"github.com/jbrady42/ion-load/ion"
	"github.com/jbrady42/ion-load/producer"
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
		t.client.Publish()
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

func (t *testRun) setupClient(room, path, vidFile string) {
	name := fmt.Sprintf(clientNameTmpl, t.index)
	t.client = ion.NewClient(name, room, path)
	t.doneCh = make(chan interface{})

	if t.produce {
		// Configure sender tracks
		offset := t.index * 5
		t.mediaSource = producer.NewMFileProducer(vidFile, offset)
		t.client.VideoTrack = t.mediaSource.VideoTrack()
		t.mediaSource.Start()
	}

	go t.runClient()
}

func main() {
	var containerPath string
	var ionPath, roomName string
	var numClients, runSeconds int
	var consume, produce bool

	flag.StringVar(&containerPath, "container-path", "", "path to the media file you want to playback")
	flag.StringVar(&ionPath, "ion-url", "ws://localhost:8443/ws", "websocket url for ion biz system")
	flag.StringVar(&roomName, "room", "video-demo", "Room name for Ion")
	flag.IntVar(&numClients, "clients", 1, "Number of clients to start")
	flag.IntVar(&runSeconds, "seconds", 60, "Number of seconds to run test for")
	flag.BoolVar(&consume, "consume", false, "Run subscribe to all streams and consume data")
	flag.BoolVar(&produce, "produce", false, "Produce stream to room")

	flag.Parse()

	if produce && containerPath == "" {
		panic("-container-path must be specified")
	}

	clients := make([]*testRun, numClients)

	for i := 0; i < numClients; i++ {
		cfg := &testRun{consume: consume, produce: produce, index: i}
		cfg.setupClient(roomName, ionPath, containerPath)
		clients[i] = cfg
		time.Sleep(2 * time.Second)
	}
	waitGroup.Add(numClients)

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
			time.Sleep(2 * time.Second)
		}
	}

	log.Println("Wait for client shutdown")
	waitGroup.Wait()
	log.Println("All clients shut down")
}
