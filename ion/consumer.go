package ion

import (
	"fmt"
	"log"

	"github.com/pion/ion/pkg/node/biz"
	"github.com/pion/webrtc/v2"
)

type Consumer struct {
	Pc   *webrtc.PeerConnection
	Info biz.MediaInfo
}

func discardConsumeLoop(track *webrtc.Track) {
	b := make([]byte, 1460)
	for {
		// Discard packet
		// Do nothing
		_, err := track.Read(b)
		if err != nil {
			log.Println("Error reading RTP packet", err)
			return
		}
	}
}

func newConsumerPeerCon() *webrtc.PeerConnection {
	// Create a MediaEngine object to configure the supported codec
	m := webrtc.MediaEngine{}

	// Setup the codecs you want to use.
	// We'll use a VP8 codec but you can also define your own
	// m.RegisterCodec(webrtc.NewRTPOpusCodec(webrtc.DefaultPayloadTypeOpus, 48000))
	m.RegisterCodec(webrtc.NewRTPVP8Codec(webrtc.DefaultPayloadTypeVP8, 90000))

	// Create the API object with the MediaEngine
	api := webrtc.NewAPI(webrtc.WithMediaEngine(m))

	// Everything below is the Pion WebRTC API! Thanks for using it ❤️.

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
		fmt.Printf("Consumer Connection State has changed %s \n", connectionState.String())
	})

	peerConnection.OnTrack(func(track *webrtc.Track, receiver *webrtc.RTPReceiver) {
		go discardConsumeLoop(track)
	})

	return peerConnection
}
