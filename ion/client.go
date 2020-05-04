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
	pubPeerCon *webrtc.PeerConnection
	wsPeer     *peer.Peer
	room       RoomInfo
	name       string
	audioTrack *webrtc.Track
	videoTrack *webrtc.Track
	paused     bool
	ionPath    string
	ReadyChan  chan bool
	Connected  bool
	OnLogin    func(json.RawMessage)
}

func newPeerCon() *webrtc.PeerConnection {
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
	return pc
}

func NewClient(name, room, path string) RoomClient {
	pc := newPeerCon()

	return RoomClient{
		pubPeerCon: pc,
		room: RoomInfo{
			Uid: name,
			Rid: room,
		},
		name:      name,
		ionPath:   path,
		ReadyChan: make(chan bool),
	}
}

func (t *RoomClient) Init() {
	wsClient := client.NewClient(t.ionPath+"?peer="+t.room.Uid, t.handleWebSocketOpen)
	go wsClient.ReadMessage()
}

func (t *RoomClient) handleWebSocketOpen(transport *transport.WebSocketTransport) {
	logger.Infof("handleWebSocketOpen")

	t.wsPeer = peer.NewPeer(t.room.Uid, transport)
	t.wsPeer.On("close", func(code int, err string) {
		logger.Infof("peer close [%d] %s", code, err)
	})

	joinMsg := JoinMsg{RoomInfo: t.room, Info: UserInfo{Name: t.name}}
	t.wsPeer.Request("join", joinMsg,
		func(result json.RawMessage) {
			logger.Infof("login success: =>  %s", result)
			// Add media stream
			// // t.publish()
			// t.Connected = true
			t.ReadyChan <- true
			if t.OnLogin != nil {
				t.OnLogin(result)
			}
		},
		func(code int, err string) {
			logger.Infof("login reject: %d => %s", code, err)
			t.ReadyChan <- false
		})
}

func (t *RoomClient) Publish() {
	if t.audioTrack != nil {
		if _, err := t.pubPeerCon.AddTrack(t.audioTrack); err != nil {
			log.Print(err)
			panic(err)
		}
	}
	if t.videoTrack != nil {
		if _, err := t.pubPeerCon.AddTrack(t.videoTrack); err != nil {
			log.Print(err)
			panic(err)
		}
	}

	t.pubPeerCon.OnICEConnectionStateChange(func(connectionState webrtc.ICEConnectionState) {
		fmt.Printf("Connection State has changed %s \n", connectionState.String())
	})

	// Create an offer to send to the browser
	offer, err := t.pubPeerCon.CreateOffer(nil)
	if err != nil {
		panic(err)
	}

	// Sets the LocalDescription, and starts our UDP listeners
	err = t.pubPeerCon.SetLocalDescription(offer)
	if err != nil {
		panic(err)
	}
	log.Println(offer)

	pubMsg := PublishMsg{RoomInfo: t.room, Jsep: offer, Options: newPublishOptions()}

	t.wsPeer.Request("publish", pubMsg, t.finalizeConnect,
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
	err = t.pubPeerCon.SetRemoteDescription(msg.Ans)
	if err != nil {
		panic(err)
	}
}

func (t *RoomClient) subcribe(mid string) {

}
