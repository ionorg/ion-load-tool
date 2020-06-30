package ion

import (
	"context"
	"fmt"
	"io"
	"log"
	"path/filepath"

	sfu "github.com/pion/ion-sfu/pkg/proto"
	"github.com/pion/producer"
	"github.com/pion/producer/ivf"
	"github.com/pion/producer/webm"
	"github.com/pion/webrtc/v2"
	"google.golang.org/grpc"
)

var (
	conf = webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{
				URLs: []string{"stun:stun.l.google.com:19302"},
			},
		},
	}
)

// LoadClient can be used for producing and consuming sfu streams
type LoadClient struct {
	name       string
	pc         *webrtc.PeerConnection
	AudioTrack *webrtc.Track
	VideoTrack *webrtc.Track
	conn       *grpc.ClientConn
	c          sfu.SFUClient
	consumers  []*Consumer
	media      producer.IFileProducer
}

// NewLoadClient creates a new LoadClient instance
func NewLoadClient(name, room, address, input string) *LoadClient {
	log.Printf("Creating load client => name: %s room: %s input: %s", name, room, input)

	// Set up a connection to the sfu server.
	conn, err := grpc.Dial(address, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	c := sfu.NewSFUClient(conn)

	// Create peer connection
	pc, err := webrtc.NewPeerConnection(conf)

	if err != nil {
		log.Fatal(err)
	}

	lc := LoadClient{
		name:      name,
		conn:      conn,
		c:         c,
		pc:        pc,
		consumers: make([]*Consumer, 0),
	}

	if input != "" {
		ext := filepath.Ext(input)
		if ext == ".webm" {
			lc.media = webm.NewMFileProducer(input, 0, producer.TrackSelect{
				Audio: true,
				Video: true,
			})
		} else if ext == ".ivf" {
			lc.media = ivf.NewIVFProducer(input, 1)
		} else {
			panic("unsupported input type")
		}
	}

	return &lc
}

// Publish a stream with load client
func (lc *LoadClient) Publish() string {
	log.Printf("Publishing stream for client: %s", lc.name)
	if lc.media.AudioTrack() != nil {
		if _, err := lc.pc.AddTrack(lc.media.AudioTrack()); err != nil {
			log.Print(err)
			panic(err)
		}

	}

	if lc.media.VideoTrack() != nil {
		if _, err := lc.pc.AddTrack(lc.media.VideoTrack()); err != nil {
			log.Print(err)
			panic(err)
		}
	}

	lc.pc.OnICEConnectionStateChange(func(connectionState webrtc.ICEConnectionState) {
		log.Printf("Client %v producer State has changed %s \n", lc.name, connectionState.String())
	})

	// Create an offer to send to the browser
	offer, err := lc.pc.CreateOffer(nil)
	if err != nil {
		panic(err)
	}

	// Sets the LocalDescription, and starts our UDP listeners
	err = lc.pc.SetLocalDescription(offer)
	if err != nil {
		panic(err)
	}

	lc.media.Start()

	ctx := context.Background()
	stream, err := lc.c.Publish(ctx)

	if err != nil {
		log.Fatalf("Error publishing response: %v", err)
	}

	stream.Send(&sfu.PublishRequest{
		Rid: "default",
		Payload: &sfu.PublishRequest_Connect{
			Connect: &sfu.Connect{
				Options: &sfu.Options{
					Codec: "VP8",
				},
				Description: &sfu.SessionDescription{
					Type: offer.Type.String(),
					Sdp:  offer.SDP,
				},
			},
		},
	})

	// First response is always connect
	res, err := stream.Recv()
	if err != nil {
		log.Fatalf("Error receiving publish->connect response: %v", err)
	}

	// Set the remote SessionDescription
	if err = lc.pc.SetRemoteDescription(webrtc.SessionDescription{
		Type: webrtc.SDPTypeAnswer,
		SDP:  res.GetConnect().Description.GetSdp(),
	}); err != nil {
		panic(err)
	}

	go func() {
		for {
			res, err := stream.Recv()
			if err == io.EOF {
				// WebRTC Transport closed
				fmt.Println("WebRTC Transport Closed")
				lc.Close()
				stream.CloseSend()
				return
			}

			if err == grpc.ErrClientConnClosing {
				// Client connection closed
				stream.CloseSend()
				return
			}

			if err != nil {
				log.Fatalf("Error receiving publish response: %v", err)
			}

			switch payload := res.Payload.(type) {
			case *sfu.PublishReply_Trickle:
				lc.pc.AddICECandidate(webrtc.ICECandidateInit{Candidate: payload.Trickle.Candidate})
			}
		}
	}()

	log.Printf("Published %s", res.Mid)
	return res.Mid
}

// Subscribe to a stream with load client
func (lc *LoadClient) Subscribe(mid string) {
	log.Println("Subscribing to ", mid)
	id := len(lc.consumers) // broken make better

	// Create new consumer
	consumer := NewConsumer(lc.name, id)

	// Create an offer to send to the browser
	offer, err := consumer.Pc.CreateOffer(nil)
	if err != nil {
		panic(err)
	}

	// Sets the LocalDescription, and starts our UDP listeners
	err = consumer.Pc.SetLocalDescription(offer)
	if err != nil {
		panic(err)
	}

	ctx := context.Background()
	stream, err := lc.c.Subscribe(ctx)
	if err != nil {
		log.Fatalf("Error receiving subscribe response: %v", err)
	}

	err = stream.Send(&sfu.SubscribeRequest{
		Mid: mid,
		Payload: &sfu.SubscribeRequest_Connect{
			Connect: &sfu.Connect{
				Description: &sfu.SessionDescription{
					Type: offer.Type.String(),
					Sdp:  offer.SDP,
				},
			},
		},
	})

	if err != nil {
		log.Fatalf("Error sending connect request: %v", err)
	}

	// First response is always connect
	res, err := stream.Recv()

	if err != nil {
		log.Printf("Error subscribing to stream: %v", err)
		return
	}

	// Set the remote SessionDescription
	err = consumer.Pc.SetRemoteDescription(webrtc.SessionDescription{
		Type: webrtc.SDPTypeAnswer,
		SDP:  res.GetConnect().Description.Sdp,
	})

	go func() {
		for {
			res, err := stream.Recv()
			if err == io.EOF {
				// WebRTC Transport closed
				fmt.Println("WebRTC Transport Closed")
				lc.Close()
				stream.CloseSend()
				return
			}

			if err == grpc.ErrClientConnClosing {
				// Client connection closed
				stream.CloseSend()
				return
			}

			if err != nil {
				log.Fatalf("Error receiving subscribe response: %v", err)
			}

			switch payload := res.Payload.(type) {
			case *sfu.SubscribeReply_Trickle:
				lc.pc.AddICECandidate(webrtc.ICECandidateInit{Candidate: payload.Trickle.Candidate})
			}
		}
	}()

	if err != nil {
		panic(err)
	}

	lc.consumers = append(lc.consumers, consumer)

	log.Println("Subscribe complete")
}

// Close client and websocket transport
func (lc *LoadClient) Close() {
	log.Printf("Closing load client %s", lc.name)
	lc.conn.Close()

	// Close any remaining consumers
	for _, sub := range lc.consumers {
		sub.Pc.Close()
	}
}
