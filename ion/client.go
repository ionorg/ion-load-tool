package ion

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/cloudwebrtc/go-protoo/client"
	"github.com/cloudwebrtc/go-protoo/logger"
	"github.com/cloudwebrtc/go-protoo/peer"
	"github.com/cloudwebrtc/go-protoo/transport"
	"github.com/google/uuid"
	"github.com/pion/ion/pkg/node/biz"
	"github.com/pion/webrtc/v2"
)

type RoomClient struct {
	biz.MediaInfo
	pubPeerCon *webrtc.PeerConnection
	WsPeer     *peer.Peer
	room       biz.RoomInfo
	name       string
	AudioTrack *webrtc.Track
	VideoTrack *webrtc.Track
	paused     bool
	ionPath    string
	ReadyChan  chan bool
	client     *client.WebSocketClient
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
	uidStr := name
	uuid, err := uuid.NewRandom()
	if err != nil {
		log.Println("Can't make new uuid??", err)
	} else {
		uidStr = uuid.String()
	}

	return RoomClient{
		pubPeerCon: pc,
		room: biz.RoomInfo{
			Uid: uidStr,
			Rid: room,
		},
		name:      name,
		ionPath:   path,
		ReadyChan: make(chan bool),
	}
}

func (t *RoomClient) Init() {
	t.client = client.NewClient(t.ionPath+"?peer="+t.room.Uid, t.handleWebSocketOpen)
}

func (t *RoomClient) handleWebSocketOpen(transport *transport.WebSocketTransport) {
	logger.Infof("handleWebSocketOpen")

	t.WsPeer = peer.NewPeer(t.room.Uid, transport)

	go func() {
		for {
			select {
			case msg := <-t.WsPeer.OnNotification:
				t.handleNotification(msg)
			case msg := <-t.WsPeer.OnRequest:
				log.Println("Got request", msg)
			case msg := <-t.WsPeer.OnClose:
				log.Println("Peer close msg", msg)
			}
		}
	}()

}

func (t *RoomClient) Join() {
	joinMsg := biz.JoinMsg{RoomInfo: t.room, Info: biz.UserInfo{Name: t.name}}
	res := <-t.WsPeer.Request("join", joinMsg, nil, nil)

	if res.Err != nil {
		logger.Infof("login reject: %d => %s", res.Err.Code, res.Err.Text)
	} else {
		logger.Infof("login success: =>  %s", res.Result)
	}
}

func (t *RoomClient) Publish() {
	if t.AudioTrack != nil {
		if _, err := t.pubPeerCon.AddTrack(t.AudioTrack); err != nil {
			log.Print(err)
			panic(err)
		}
	}
	if t.VideoTrack != nil {
		if _, err := t.pubPeerCon.AddTrack(t.VideoTrack); err != nil {
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

	pubMsg := biz.PublishMsg{
		RoomInfo: t.room,
		RTCInfo:  biz.RTCInfo{Jsep: offer},
		Options:  newPublishOptions(),
	}

	res := <-t.WsPeer.Request("publish", pubMsg, nil, nil)
	if res.Err != nil {
		logger.Infof("publish reject: %d => %s", res.Err.Code, res.Err.Text)
		return
	}
	t.finalizeConnect(res.Result)
}

func (t *RoomClient) finalizeConnect(result json.RawMessage) {
	logger.Infof("publish success")

	var msg biz.PublishResponseMsg
	err := json.Unmarshal(result, &msg)
	if err != nil {
		log.Println(err)
		return
	}

	t.MediaInfo = msg.MediaInfo

	// Set the remote SessionDescription
	err = t.pubPeerCon.SetRemoteDescription(msg.Jsep)
	if err != nil {
		panic(err)
	}
}

func (t *RoomClient) handleNotification(msg peer.Notification) {
	switch msg.Method {
	case "stream-add":
		t.handleStreamAdd(msg.Data)
	case "stream-remove":
		t.handleStreamRemove(msg.Data)
	}
}

func (t *RoomClient) handleStreamAdd(msg json.RawMessage) {
	var msgData biz.StreamAddMsg
	if err := json.Unmarshal(msg, &msgData); err != nil {
		log.Println("Marshal error", err)
		return
	}
	log.Println("New stream", msgData)
}

func (t *RoomClient) handleStreamRemove(msg json.RawMessage) {
	var msgData biz.StreamRemoveMsg
	if err := json.Unmarshal(msg, &msgData); err != nil {
		log.Println("Marshal error", err)
		return
	}
	log.Println("Remove stream", msgData)
}

func (t *RoomClient) subcribe(mid string) {

}

func (t *RoomClient) UnPublish() {
	msg := biz.UnpublishMsg{
		MediaInfo: t.MediaInfo,
		RoomInfo:  t.room,
	}
	res := <-t.WsPeer.Request("unpublish", msg, nil, nil)
	if res.Err != nil {
		logger.Infof("unpublish reject: %d => %s", res.Err.Code, res.Err.Text)
		return
	}
}

func (t *RoomClient) Leave() {

}

// Shutdown client and websocket transport
func (t *RoomClient) Close() {
	t.client.Close()
}
