package main

import (
	"flag"
	"log"
	"sync"
	"time"

	"github.com/jbrady42/ion-load/ion"
)

var (
	shutdown  = false
	waitGroup sync.WaitGroup
)

func runClient(client ion.RoomClient, index int) {
	waitGroup.Add(1)
	defer waitGroup.Done()

	// Configure sender tracks

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

	for !shutdown {
		// wait
	}

	log.Printf("Begin client %v shutdown", index)

	// Unsubscribe Consumers

	// Unpublish producer

	// Close client

}

func newClient(name, room, path string, index int) ion.RoomClient {
	client := ion.NewClient(name, room, path)
	go runClient(client, index)
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

	name := "testname"

	_ = newClient(name, roomName, ionPath, 0)

	// Run test
	// Create X rooms
	// Create y clients per room
	// Each client
	//// publishes 1 streams
	//// Consumes all streams in the room
	//// Measure quality of delivered streams in some way

	time.Sleep(10 * time.Second)
	shutdown = true

	//Wait for client shutdown
	waitGroup.Wait()
	log.Println("All clients shut down")
}
