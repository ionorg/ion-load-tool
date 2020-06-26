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

	"github.com/pion/ion-load-tool/ion"
	"github.com/pion/producer"
)

var (
	clients   []*ion.LoadClient
	streams   []string
	waitGroup sync.WaitGroup
)

func main() {
	var sfu, room, input string
	var n, duration int
	var audio, consume bool
	var stagger float64

	flag.StringVar(&input, "produce", "", "path to the media file you want to playback")
	flag.StringVar(&sfu, "sfu", "localhost:50051", "ion-sfu grpc url")
	flag.StringVar(&room, "room", "video-demo", "Room name for Ion")
	flag.IntVar(&n, "clients", 1, "Number of clients to start")
	flag.Float64Var(&stagger, "stagger", 1.0, "Number of seconds to stagger client start and stop")
	flag.IntVar(&duration, "seconds", 60, "Number of seconds to run test for")
	flag.BoolVar(&consume, "consume", false, "Run subscribe to all streams and consume data")
	flag.BoolVar(&audio, "audio", false, "Publish audio stream from webm file")

	flag.Parse()

	// Validate type
	if input != "" {
		ext, ok := producer.ValidateVPFile(input)
		log.Println(ext)
		if !ok {
			panic("Only IVF and WEBM containers are supported.")
		}
	}

	staggerDur := time.Duration(stagger*1000) * time.Millisecond

	for i := 0; i < n; i++ {
		client := ion.NewLoadClient(fmt.Sprintf("client_%d", i), room, sfu, input)
		mid := client.Publish()

		// Subscribe to existing pubs
		for _, pub := range streams {
			client.Subscribe(pub)
		}

		// Subscribe existing clients to new pub
		for _, c := range clients {
			c.Subscribe(mid)
		}

		streams = append(streams, mid)
		clients = append(clients, client)

		time.Sleep(staggerDur)
	}

	// Setup shutdown
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	timer := time.NewTimer(time.Duration(duration) * time.Second)

	select {
	case <-sigs:
	case <-timer.C:
	}

	for i, a := range clients {
		// Signal shutdown
		a.Close()
		// Staggered shutdown.
		if len(clients) > 1 && i < len(clients)-1 {
			time.Sleep(staggerDur)
		}
	}

	log.Println("Wait for client shutdown")
	log.Println("All clients shut down")
}
