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

	"github.com/jbrady42/ion-load/ion"
	"github.com/jbrady42/ion-load/producer"
)

var (
	waitGroup sync.WaitGroup
)

func runClient(client ion.RoomClient, index int, doneCh <-chan interface{}, mediaSource *producer.FileProducer) {
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

	client.Close()
}

func newClient(name, room, path, vidFile string, index int) (ion.RoomClient, chan interface{}) {
	client := ion.NewClient(name, room, path)
	doneChan := make(chan interface{})

	mediaSource := producer.NewFileProducer(vidFile)
	offset := index * 100
	go mediaSource.ReadLoop(offset)

	go runClient(client, index, doneChan, mediaSource)
	return client, doneChan
}

func main() {
	var containerPath string
	var ionPath string
	var roomName string
	var numClients int

	flag.StringVar(&containerPath, "container-path", "", "path to the media file you want to playback")
	flag.StringVar(&ionPath, "ion-url", "ws://localhost:8443/ws", "websocket url for ion biz system")
	flag.StringVar(&roomName, "room", "video-demo", "Room name for Ion")
	flag.IntVar(&numClients, "clients", 1, "Number of clients to start")
	flag.Parse()

	if containerPath == "" {
		panic("-container-path must be specified")
	}

	clients := make([]chan interface{}, numClients)

	for i := 0; i < numClients; i++ {
		clientName := fmt.Sprintf("client_%v", i)
		_, closeCh := newClient(clientName, roomName, ionPath, containerPath, i)
		clients[i] = closeCh
		time.Sleep(2 * time.Second)
	}
	waitGroup.Add(numClients)

	// Setup shutdown
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	timer := time.NewTimer(1 * time.Minute)

	select {
	case <-sigs:
	case <-timer.C:
	}

	// Staggered shutdown.
	// 10s per client shutdown
	for _, a := range clients {
		// Signal shutdown
		close(a)
		time.Sleep(2 * time.Second)
	}

	//Wait for client shutdown
	waitGroup.Wait()
	log.Println("All clients shut down")
}
