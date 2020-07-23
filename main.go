package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/pion/ion-load-tool/ion"
	"github.com/pion/producer"
)

type roomFlags []string

func (i *roomFlags) String() string {
	return "default-room"
}

func (i *roomFlags) Set(value string) error {
	*i = append(*i, value)
	return nil
}

func run(room, sfu, input string, produce, consume bool, n, duration int, stagger time.Duration) {
	var clients []*ion.LoadClient
	var streams []string
	timer := time.NewTimer(time.Duration(duration) * time.Second)

	for i := 0; i < n; i++ {
		client := ion.NewLoadClient(fmt.Sprintf("client_%s_%d", room, i), room, sfu, input)

		if produce {
			// Validate type
			if input != "" {
				ext, ok := producer.ValidateVPFile(input)
				log.Println(ext)
				if !ok {
					panic("Only IVF and WEBM containers are supported.")
				}
			}
			mid := client.Publish()

			// if consume {
			// 	// Subscribe to existing pubs
			// 	for _, pub := range streams {
			// 		client.Subscribe(pub)
			// 	}

			// 	// Subscribe existing clients to new pub
			// 	for _, c := range clients {
			// 		c.Subscribe(mid)
			// 	}
			// }

			streams = append(streams, mid)
			// } else if consume && input != "" {
			// 	client.Subscribe(input)
		} else {
			panic("unsupported configuration. must produce or consume")
		}

		clients = append(clients, client)

		time.Sleep(stagger)
	}

	select {
	case <-timer.C:
	}

	for i, a := range clients {
		// Signal shutdown
		a.Close()
		// Staggered shutdown.
		if len(clients) > 1 && i < len(clients)-1 {
			time.Sleep(stagger)
		}
	}
}

func main() {
	var rooms roomFlags
	var sfu, input string
	var n, duration int
	var audio, produce, consume bool
	var stagger float64

	flag.StringVar(&input, "input", "", "path to the input media")
	flag.StringVar(&sfu, "sfu", "localhost:50051", "ion-sfu grpc url")
	flag.Var(&rooms, "room", "Rooms to join.")
	flag.IntVar(&n, "clients", 1, "Number of clients to start")
	flag.Float64Var(&stagger, "stagger", 1.0, "Number of seconds to stagger client start and stop")
	flag.IntVar(&duration, "seconds", 60, "Number of seconds to run test for")
	flag.BoolVar(&audio, "audio", false, "Publish audio stream from webm file")
	flag.BoolVar(&produce, "produce", false, "path to the media file you want to playback")
	flag.BoolVar(&consume, "consume", false, "Run subscribe to all streams and consume data")

	flag.Parse()

	staggerDur := time.Duration(stagger*1000) * time.Millisecond

	if len(rooms) == 0 {
		rooms = append(rooms, "default")
	}

	for _, room := range rooms {
		go run(room, sfu, input, produce, consume, n, duration, staggerDur)
	}

	// Setup shutdown
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-sigs:
	}

	log.Println("Wait for client shutdown")
	log.Println("All clients shut down")
}
