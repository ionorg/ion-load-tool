package main

import (
	"flag"

	"github.com/jbrady42/ion-load/ion"
)

func main() {
	var containerPath string
	var ionPath string
	var roomName string

	flag.StringVar(&containerPath, "container-path", "", "path to the media file you want to playback")
	flag.StringVar(&ionPath, "ion-url", "ws://localhost:8443/ws", "websocket url for ion biz system")
	flag.StringVar(&roomName, "room", "video-demo", "Room name for Ion")
	flag.Parse()

	client := ion.NewClient("test", roomName, ionPath)
	client.Init()
}
