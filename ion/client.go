package ion

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"path/filepath"

	"github.com/pion/ion-load-tool/webm"
	sfu "github.com/pion/ion-sfu/cmd/server/grpc/proto"
	"github.com/pion/webrtc/v3"
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
	room       string
	pc         *webrtc.PeerConnection
	AudioTrack *webrtc.Track
	VideoTrack *webrtc.Track
	conn       *grpc.ClientConn
	c          sfu.SFUClient
	media      *webm.WebMProducer
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
		name: name,
		room: room,
		conn: conn,
		c:    c,
		pc:   pc,
	}

	if input != "" {
		ext := filepath.Ext(input)
		if ext == ".webm" {
			lc.media = webm.NewMFileProducer(input, 0, webm.TrackSelect{
				Audio: true,
				Video: true,
			})
		} else {
			panic("unsupported input type")
		}
	}

	return &lc
}

// Publish a stream with load client
func (lc *LoadClient) Publish() {
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

	lc.pc.OnTrack(func(track *webrtc.Track, recv *webrtc.RTPReceiver, s []*webrtc.Stream) {
		log.Printf("Got on track: %v", track)
		go discardConsumeLoop(track)
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
	stream, err := lc.c.Signal(ctx)

	if err != nil {
		log.Fatalf("Error publishing response: %v", err)
	}

	stream.Send(&sfu.SignalRequest{
		Payload: &sfu.SignalRequest_Join{
			Join: &sfu.JoinRequest{
				Sid: lc.room,
				Offer: &sfu.SessionDescription{
					Type: offer.Type.String(),
					Sdp:  []byte(offer.SDP),
				},
			},
		},
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
				log.Fatalf("Error receiving publish response: %v", err)
			}

			switch payload := res.Payload.(type) {
			case *sfu.SignalReply_Join:
				pid := res.GetJoin().Pid
				// Set the remote SessionDescription
				if err = lc.pc.SetRemoteDescription(webrtc.SessionDescription{
					Type: webrtc.SDPTypeAnswer,
					SDP:  string(res.GetJoin().Answer.Sdp),
				}); err != nil {
					panic(err)
				}
				log.Printf("Published %s", pid)
			case *sfu.SignalReply_Negotiate:
				if payload.Negotiate.Type == webrtc.SDPTypeOffer.String() {
					offer := webrtc.SessionDescription{
						Type: webrtc.SDPTypeOffer,
						SDP:  string(payload.Negotiate.Sdp),
					}

					// Peer exists, renegotiating existing peer
					err = lc.pc.SetRemoteDescription(offer)
					if err != nil {
						log.Printf("negotiate error %s", err)
						continue
					}

					answer, err := lc.pc.CreateAnswer(nil)
					if err != nil {
						log.Printf("negotiate error %s", err)
						continue
					}

					err = stream.Send(&sfu.SignalRequest{
						Payload: &sfu.SignalRequest_Negotiate{
							Negotiate: &sfu.SessionDescription{
								Type: answer.Type.String(),
								Sdp:  []byte(answer.SDP),
							},
						},
					})
					if err != nil {
						log.Printf("negotiate error %s", err)
						continue
					}
				} else if payload.Negotiate.Type == webrtc.SDPTypeAnswer.String() {
					err = lc.pc.SetRemoteDescription(webrtc.SessionDescription{
						Type: webrtc.SDPTypeAnswer,
						SDP:  string(payload.Negotiate.Sdp),
					})

					if err != nil {
						log.Printf("negotiate error %s", err)
						continue
					}
				}
			case *sfu.SignalReply_Trickle:
				var candidate webrtc.ICECandidateInit
				_ = json.Unmarshal([]byte(payload.Trickle.Init), &candidate)
				lc.pc.AddICECandidate(candidate)
			}
		}
	}()
}

// Close client and websocket transport
func (lc *LoadClient) Close() {
	log.Printf("Closing load client %s", lc.name)
	lc.conn.Close()
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
