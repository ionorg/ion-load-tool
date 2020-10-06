package main

import (
	"flag"
	"fmt"
	"time"

	log "github.com/pion/ion-log"
	engine "github.com/pion/ion-sdk-go/pkg"
	"github.com/pion/webrtc/v3"
)

func run(sfu *engine.SFU, room, url, input, role string, total, duration, cycle int) {
	log.Infof("run room=%v url=%v input=%v role=%v total=%v duration=%v cycle=%v\n", room, url, input, role, total, duration, cycle)
	timer := time.NewTimer(time.Duration(duration) * time.Second)

	go sfu.Stats(3)
	for i := 0; i < total; i++ {
		tid := fmt.Sprintf("%s_%d", room, i)
		// tid := room
		switch role {
		case "pubsub":
			t := sfu.GetTransport(room, tid)
			err := t.AddProducer(input)
			if err != nil {
				log.Errorf("err=%v", err)
				break
			}
			t.Subscribe()
			sfu.Join(room, t)
		case "pub":
			t := sfu.GetTransport(room, tid)
			err := t.AddProducer(input)
			if err != nil {
				log.Errorf("err=%v", err)
				break
			}
			sfu.Join(room, t)
		case "sub":
			t := sfu.GetTransport(room, tid)
			t.Subscribe()
			sfu.Join(room, t)
		default:
			log.Errorf("invalid role! should be pub/sub/pubsub")
		}

		time.Sleep(time.Millisecond * time.Duration(cycle))
	}

	select {
	case <-timer.C:
	}
}

func main() {
	//init log
	fixByFile := []string{"asm_amd64.s", "proc.go", "icegatherer.go"}
	fixByFunc := []string{"AddProducer"}

	//get args
	var room string
	var url, input string
	var total, cycle, duration int
	var role string
	var loglevel string
	// var video, audio bool

	flag.StringVar(&input, "input", "./input.webm", "Path to the input media")
	flag.StringVar(&url, "url", "localhost:50051", "Ion-sfu grpc url")
	flag.StringVar(&room, "room", "room", "Room to join")
	flag.IntVar(&total, "clients", 1, "Number of clients to start")
	flag.IntVar(&cycle, "cycle", 300, "Run new client cycle in ms")
	flag.IntVar(&duration, "duration", 3600, "Running duration in sencond")
	flag.StringVar(&role, "role", "pubsub", "Run as pub/sub/pubsub  (sender/receiver/both)")
	flag.StringVar(&loglevel, "loglevel", "info", "Log level")
	// flag.BoolVar(&video, "video", true, "Publish video stream from webm file")
	// flag.BoolVar(&audio, "audio", true, "Publish audio stream from webm file")
	flag.Parse()
	log.Init(loglevel, fixByFile, fixByFunc)

	config := engine.WebRTCTransportConfig{
		Configuration: webrtc.Configuration{
			SDPSemantics: webrtc.SDPSemanticsUnifiedPlan,
			ICEServers: []webrtc.ICEServer{
				{URLs: []string{"stun:stun.l.google.com:19302"}},
				// {URLs: []string{"stun:stun.stunprotocol.org:3478"}},
			},
			ICETransportPolicy: webrtc.NewICETransportPolicy("all"),
		},
		Setting: webrtc.SettingEngine{},
	}
	sfu := engine.NewSFU(url, config)
	run(sfu, room, url, input, role, total, duration, cycle)
}
