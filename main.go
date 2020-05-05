package main

import (
	"flag"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/jbrady42/ion-load/ion"
	"github.com/jbrady42/ion-load/producer"
)

var (
	mediaSource *producer.FileProducer
	waitGroup   sync.WaitGroup
	doneChan    = make(chan interface{})
)

func runClient(client ion.RoomClient, index int, doneCh <-chan interface{}) {
	defer waitGroup.Done()

	// Configure sender tracks
	client.VideoTrack = mediaSource.Track

	client.Init()

	ready := <-client.ReadyChan
	if !ready {
		log.Println("Client initialization error")
		return
	}

	// Start producer
	client.Publish()

	// Comfigure auto subscribe in room

	// Wire consumers

	// Wait for the end of the test then shutdown
	<-doneCh
	log.Printf("Begin client %v shutdown", index)

	// Unsubscribe Consumers

	// Unpublish producer

	// Close client
	client.Leave()

}

func newClient(name, room, path string, index int) ion.RoomClient {
	client := ion.NewClient(name, room, path)
	go runClient(client, index, doneChan)
	return client
}

func main() {
	var containerPath string
	var ionPath string
	var roomName string

	flag.StringVar(&containerPath, "container-path", "", "path to the media file you want to playback")
	flag.StringVar(&ionPath, "ion-url", "ws://localhost:8443/ws", "websocket url for ion biz system")
	flag.StringVar(&roomName, "room", "video-demo", "Room name for Ion")
	flag.Parse()

	if containerPath == "" {
		panic("-container-path must be specified")
	}

	mediaSource = producer.NewFileProducer(containerPath)
	go mediaSource.ReadLoop(500)

	maxClients := 1

	for i := 0; i < maxClients; i++ {
		clientName := fmt.Sprintf("client_%v", i)
		_ = newClient(clientName, roomName, ionPath, i)
	}
	waitGroup.Add(maxClients)

	// Run test
	// Create X rooms
	// Create y clients per room
	// Each client
	//// publishes 1 streams
	//// Consumes all streams in the room
	//// Measure quality of delivered streams in some way

	time.Sleep(60 * time.Second)
	close(doneChan)

	//Wait for client shutdown
	waitGroup.Wait()
	log.Println("All clients shut down")
}
