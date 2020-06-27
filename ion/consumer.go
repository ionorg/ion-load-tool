package ion

import (
	"log"

	"github.com/pion/webrtc/v2"
)

// Consumer subscribes to a sfu pub and consumes its output
type Consumer struct {
	Pc *webrtc.PeerConnection
}

// NewConsumer creates a new consumer instance
// name is the client name, id is the consumer instance id
func NewConsumer(name string, id int) *Consumer {
	m := webrtc.MediaEngine{}
	m.RegisterDefaultCodecs()

	// Create the API object with the MediaEngine
	api := webrtc.NewAPI(webrtc.WithMediaEngine(m))

	// Create a new RTCPeerConnection
	pc, err := api.NewPeerConnection(conf)
	if err != nil {
		panic(err)
	}

	// Allow us to receive 1 audio track, and 1 video track
	if _, err = pc.AddTransceiverFromKind(webrtc.RTPCodecTypeVideo); err != nil {
		panic(err)
	}

	pc.OnICEConnectionStateChange(func(connectionState webrtc.ICEConnectionState) {
		log.Printf("Client %v Consumer %d Connection State has changed %s \n", name, id, connectionState.String())
	})

	pc.OnTrack(func(track *webrtc.Track, receiver *webrtc.RTPReceiver) {
		go discardConsumeLoop(track)
	})

	return &Consumer{
		Pc: pc,
	}
}

func discardConsumeLoop(track *webrtc.Track) {
	log.Println("Start discard consumer")
	var lastNum uint16
	for {
		// Discard packet
		// Do nothing
		packet, err := track.ReadRTP()
		if err != nil {
			log.Println("Error reading RTP packet", err)
			return
		}
		seq := packet.Header.SequenceNumber
		if seq != lastNum+1 {
			log.Printf("Packet out of order! prev %d current %d", lastNum, seq)
		}
		lastNum = seq
	}
}
