package main

import (
	"flag"
	"time"

	log "github.com/pion/ion-log"
	engine "github.com/pion/ion-sdk-go"
	"github.com/pion/webrtc/v3"
)

func run(sfu *engine.SFU, room, url, input, role string, total, duration, cycle int) {
	log.Infof("run room=%v url=%v input=%v role=%v total=%v duration=%v cycle=%v\n", room, url, input, role, total, duration, cycle)
	timer := time.NewTimer(time.Duration(duration) * time.Second)

	go sfu.Stats(3)
	for i := 0; i < total; i++ {
		switch role {
		case "pubsub":
			// create trans and join
			log.Infof("sfu.GetTransport %v", room)
			t, err := sfu.GetTransport(room, input)
			if err != nil {
				log.Errorf("err=%v", err)
				break
			}

			// log.Infof("t.AddProducer %v", input)
			// err = t.AddProducer(input)
			// if err != nil {
			// log.Errorf("err=%v", err)
			// break
			// }

			log.Infof("sfu.Subscribe")
			err = t.Subscribe(nil)
			if err != nil {
				log.Errorf("err=%v", err)
			}

			// log.Infof("sfu.Publish")
			// err = t.Publish()
			// if err != nil {
			// log.Errorf("err=%v", err)
			// break
			// }
		case "pub":
			_, err := sfu.GetTransport(room, input)
			if err != nil {
				log.Errorf("err=%v", err)
				break
			}
			// err = t.AddProducer(input)
			// if err != nil {
			// log.Errorf("err=%v", err)
			// break
			// }
			// err = t.Publish()
			// if err != nil {
			// log.Errorf("err=%v", err)
			// }
		case "sub":
			t, err := sfu.GetTransport(room, "")
			if err != nil {
				log.Errorf("err=%v", err)
				break
			}
			log.Infof("t.Subscribe")
			err = t.Subscribe(nil)
			if err != nil {
				log.Errorf("err=%v", err)
				break
			}
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
	flag.StringVar(&loglevel, "log", "info", "Log level")
	// flag.BoolVar(&video, "video", true, "Publish video stream from webm file")
	// flag.BoolVar(&audio, "audio", true, "Publish audio stream from webm file")
	flag.Parse()
	log.Init(loglevel, fixByFile, fixByFunc)

	config := engine.Config{
		Log: log.Config{
			Level: loglevel,
		},
		WebRTC: engine.WebRTCConf{
			ICEServers: []engine.ICEConf{
				engine.ICEConf{
					URLs:           []string{"stun:stun.stunprotocol.org:3478"},
					Username:       "",
					Credential:     "",
					CredentialType: webrtc.ICECredentialTypePassword,
				},
			},
			ICEPortRange: []uint16{5000, 6000},
			// ICELite:      true,
		},
	}
	sfu, err := engine.NewSFU(url, config)
	if err != nil {
		log.Errorf("err=%v", err)
		return
	}

	run(sfu, room, url, input, role, total, duration, cycle)
}
