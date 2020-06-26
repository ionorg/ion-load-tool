package ion

import (
	"context"
	"io"
	"log"
	"path/filepath"

	sfu "github.com/pion/ion-sfu/pkg/proto/sfu"
	"github.com/pion/producer"
	"github.com/pion/producer/ivf"
	"github.com/pion/producer/webm"
	"github.com/pion/webrtc/v2"
	"google.golang.org/grpc"
)

var (
	IceServers = []webrtc.ICEServer{
		{
			URLs: []string{"stun:stun.l.google.com:19302"},
		},
	}
)

type Consumer struct {
	Pc  *webrtc.PeerConnection
	Mid string
}

type LoadClient struct {
	name       string
	mid        string
	pc         *webrtc.PeerConnection
	AudioTrack *webrtc.Track
	VideoTrack *webrtc.Track
	conn       *grpc.ClientConn
	c          sfu.SFUClient
	consumers  []*Consumer
	media      producer.IFileProducer
}

func NewLoadClient(name, room, address, input string) *LoadClient {
	log.Printf("Creating load client => name: %s room: %s input: %s", name, room, input)

	// Set up a connection to the sfu server.
	conn, err := grpc.Dial(address, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	c := sfu.NewSFUClient(conn)

	// Create peer connection
	pc, err := webrtc.NewPeerConnection(webrtc.Configuration{
		ICEServers: IceServers,
	})

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
			lc.media.Start()
		} else {
			panic("unsupported input type")
		}
	}

	return &lc
}

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
	stream, err := lc.c.Publish(ctx, &sfu.PublishRequest{
		Rid: "default",
		Options: &sfu.PublishOptions{
			Codec: "VP8",
		},
		Description: &sfu.SessionDescription{
			Type: offer.Type.String(),
			Sdp:  offer.SDP,
		},
	})

	if err != nil {
		log.Printf("Error publishing stream: %v", err)
		return ""
	}

	answer, err := stream.Recv()
	if err != nil {
		log.Fatalf("Error receving publish response: %v", err)
	}

	// Set the remote SessionDescription
	if err = lc.pc.SetRemoteDescription(webrtc.SessionDescription{
		Type: webrtc.SDPTypeAnswer,
		SDP:  answer.Description.Sdp,
	}); err != nil {
		panic(err)
	}

	go func() {
		answer, err = stream.Recv()
		if err == io.EOF {
			// WebRTC Transport closed
			log.Printf("WebRTC Transport Closed")
		}
	}()

	log.Printf("Published %s", answer.Mediainfo.Mid)
	return answer.Mediainfo.Mid
}

func (lc *LoadClient) Unpublish() {
	ctx := context.Background()
	_, err := lc.c.Unpublish(ctx, &sfu.UnpublishRequest{Mid: lc.mid})

	if err != nil {
		log.Fatalf("Error unpublishing: %v", err)
	}

	// Stop producer peer connection
	lc.pc.Close()
}

func (lc *LoadClient) Subscribe(mid string) {
	log.Println("Subscribing to ", mid)
	id := len(lc.consumers) // broken make better

	// Create peer connection
	pc := newConsumerPeerCon(lc.name, id)
	// Create an offer to send to the browser
	offer, err := pc.CreateOffer(nil)
	if err != nil {
		panic(err)
	}

	// Sets the LocalDescription, and starts our UDP listeners
	err = pc.SetLocalDescription(offer)
	if err != nil {
		panic(err)
	}

	ctx := context.Background()
	answer, err := lc.c.Subscribe(ctx, &sfu.SubscribeRequest{Mid: mid, Description: &sfu.SessionDescription{
		Type: offer.Type.String(),
		Sdp:  offer.SDP,
	}})

	if err != nil {
		log.Printf("Error subscribing to stream: %v", err)
		return
	}

	// Set the remote SessionDescription
	err = pc.SetRemoteDescription(webrtc.SessionDescription{
		Type: webrtc.SDPTypeAnswer,
		SDP:  answer.Description.Sdp,
	})

	if err != nil {
		panic(err)
	}

	// Create consumer
	consumer := &Consumer{pc, mid}
	lc.consumers = append(lc.consumers, consumer)

	log.Println("Subscribe complete")
}

func (lc *LoadClient) Unsubscribe(mid string) {
	// Send upsubscribe request
	// Shut down peerConnection
	var sub *Consumer
	for _, a := range lc.consumers {
		if a.Mid == mid {
			sub = a
			break
		}
	}

	if sub != nil && sub.Pc != nil {
		log.Println("Closing subscription peerConnection")
		sub.Pc.Close()
	}
}

// Close client and websocket transport
func (lc *LoadClient) Close() {
	lc.conn.Close()

	// Close any remaining consumers
	for _, sub := range lc.consumers {
		sub.Pc.Close()
	}
}
