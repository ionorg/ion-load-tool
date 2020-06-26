package ion

import (
	"log"

	"github.com/pion/webrtc/v2"
)

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

func newConsumerPeerCon(clientId string, consumerId int) *webrtc.PeerConnection {
	// Create a MediaEngine object to configure the supported codec
	m := webrtc.MediaEngine{}
	m.RegisterDefaultCodecs()

	// Create the API object with the MediaEngine
	api := webrtc.NewAPI(webrtc.WithMediaEngine(m))

	// Prepare the configuration
	config := webrtc.Configuration{
		ICEServers: IceServers,
	}

	// Create a new RTCPeerConnection
	peerConnection, err := api.NewPeerConnection(config)
	if err != nil {
		panic(err)
	}

	// Allow us to receive 1 audio track, and 1 video track
	if _, err = peerConnection.AddTransceiverFromKind(webrtc.RTPCodecTypeVideo); err != nil {
		panic(err)
	}

	peerConnection.OnICEConnectionStateChange(func(connectionState webrtc.ICEConnectionState) {
		log.Printf("Client %v Consumer %d Connection State has changed %s \n", clientId, consumerId, connectionState.String())
	})

	peerConnection.OnTrack(func(track *webrtc.Track, receiver *webrtc.RTPReceiver) {
		go discardConsumeLoop(track)
	})

	return peerConnection
}
