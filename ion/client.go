package ion

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/cloudwebrtc/go-protoo/client"
	"github.com/cloudwebrtc/go-protoo/logger"
	"github.com/cloudwebrtc/go-protoo/peer"
	"github.com/cloudwebrtc/go-protoo/transport"
	"github.com/pion/webrtc/v2"
)

type RoomClient struct {
	peerCon    *webrtc.PeerConnection
	room       RoomInfo
	name       string
	audioTrack *webrtc.Track
	videoTrack *webrtc.Track
	paused     bool
	ionPath    string
}

func NewClient(name, room, path string) RoomClient {
	pc, err := webrtc.NewPeerConnection(webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{
				URLs: []string{"stun:stun.l.google.com:19302"},
			},
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	return RoomClient{
		peerCon: pc,
		room: RoomInfo{
			Uid: name,
			Rid: room,
		},
		name:    name,
		ionPath: path,
	}
}

func (t *RoomClient) Init() {
	wsClient := client.NewClient(t.ionPath+"?peer="+t.room.Uid, t.handleWebSocketOpen)
	wsClient.ReadMessage()
}

func (t *RoomClient) handleWebSocketOpen(transport *transport.WebSocketTransport) {
	logger.Infof("handleWebSocketOpen")

	pr := peer.NewPeer(t.room.Uid, transport)
	pr.On("close", func(code int, err string) {
		logger.Infof("peer close [%d] %s", code, err)
	})

	joinMsg := JoinMsg{RoomInfo: t.room, Info: UserInfo{Name: t.name}}

	pr.Request("join", joinMsg,
		func(result json.RawMessage) {
			logger.Infof("login success: =>  %s", result)
			// Add media stream
			t.publish(pr)
		},
		func(code int, err string) {
			logger.Infof("login reject: %d => %s", code, err)
		})
}

func (t *RoomClient) publish(peer *peer.Peer) {
	// Get code from rtwatch and gstreamer
	if t.audioTrack != nil {
		if _, err := t.peerCon.AddTrack(t.audioTrack); err != nil {
			log.Print(err)
			panic(err)
		}
	}
	if t.videoTrack != nil {
		if _, err := t.peerCon.AddTrack(t.videoTrack); err != nil {
			log.Print(err)
			panic(err)
		}
	}

	t.peerCon.OnICEConnectionStateChange(func(connectionState webrtc.ICEConnectionState) {
		fmt.Printf("Connection State has changed %s \n", connectionState.String())
	})

	// Create an offer to send to the browser
	offer, err := t.peerCon.CreateOffer(nil)
	if err != nil {
		panic(err)
	}

	// Sets the LocalDescription, and starts our UDP listeners
	err = t.peerCon.SetLocalDescription(offer)
	if err != nil {
		panic(err)
	}
	log.Println(offer)

	pubMsg := PublishMsg{RoomInfo: t.room, Jsep: offer, Options: newPublishOptions()}

	peer.Request("publish", pubMsg, t.finalizeConnect,
		func(code int, err string) {
			logger.Infof("publish reject: %d => %s", code, err)
		})
}

func (t *RoomClient) finalizeConnect(result json.RawMessage) {
	logger.Infof("publish success: =>  %s", result)

	var msg connectMsg
	err := json.Unmarshal(result, &msg)
	if err != nil {
		log.Println(err)
		return
	}

	// Set the remote SessionDescription
	err = t.peerCon.SetRemoteDescription(msg.Ans)
	if err != nil {
		panic(err)
	}
}
