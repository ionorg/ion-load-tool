package ion

import (
	"encoding/json"
	"log"

	"github.com/cloudwebrtc/go-protoo/client"
	"github.com/cloudwebrtc/go-protoo/logger"
	"github.com/cloudwebrtc/go-protoo/peer"
	"github.com/cloudwebrtc/go-protoo/transport"
	"github.com/google/uuid"
	"github.com/pion/ion/pkg/node/biz"
	"github.com/pion/ion/pkg/proto"
	"github.com/pion/webrtc/v2"
)

var (
	IceServers = []webrtc.ICEServer{
		{
			URLs: []string{"stun:stun.l.google.com:19302"},
		},
	}
)

type Consumer struct {
	Pc   *webrtc.PeerConnection
	Info biz.MediaInfo
}

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
	consumers  []*Consumer
}

func newPeerCon() *webrtc.PeerConnection {
	pc, err := webrtc.NewPeerConnection(webrtc.Configuration{
		ICEServers: IceServers,
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
		consumers: make([]*Consumer, 0),
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
	res := <-t.WsPeer.Request(proto.ClientJoin, joinMsg, nil, nil)

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
		log.Printf("Client %v producer State has changed %s \n", t.name, connectionState.String())
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

	res := <-t.WsPeer.Request(proto.ClientPublish, pubMsg, nil, nil)
	if res.Err != nil {
		logger.Infof("publish reject: %d => %s", res.Err.Code, res.Err.Text)
		return
	}

	var msg biz.PublishResponseMsg
	err = json.Unmarshal(res.Result, &msg)
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
	go t.Subscribe(msgData.MediaInfo)
}

func (t *RoomClient) handleStreamRemove(msg json.RawMessage) {
	var msgData biz.StreamRemoveMsg
	if err := json.Unmarshal(msg, &msgData); err != nil {
		log.Println("Marshal error", err)
		return
	}
	log.Println("Remove stream", msgData)
	t.UnSubscribe(msgData.MediaInfo)
}

func (t *RoomClient) subcribe(mid string) {

}

func (t *RoomClient) UnPublish() {
	msg := biz.UnpublishMsg{
		MediaInfo: t.MediaInfo,
		RoomInfo:  t.room,
	}
	res := <-t.WsPeer.Request(proto.ClientUnPublish, msg, nil, nil)
	if res.Err != nil {
		logger.Infof("unpublish reject: %d => %s", res.Err.Code, res.Err.Text)
		return
	}

	// Stop producer peer connection
	t.pubPeerCon.Close()
}

func (t *RoomClient) Subscribe(info biz.MediaInfo) {
	log.Println("Subscribing to ", info)
	// Create peer connection
	pc := newConsumerPeerCon(t.name, len(t.consumers))
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

	// Send subscribe requestv
	req := biz.SubscribeMsg{
		MediaInfo: info,
		RoomInfo:  t.room,
		RTCInfo:   biz.RTCInfo{Jsep: offer},
	}
	res := <-t.WsPeer.Request(proto.ClientSubscribe, req, nil, nil)
	if res.Err != nil {
		logger.Infof("unpublish reject: %d => %s", res.Err.Code, res.Err.Text)
		return
	}

	var msg biz.SubscribeResponseMsg
	err = json.Unmarshal(res.Result, &msg)
	if err != nil {
		log.Println(err)
		return
	}

	// Set the remote SessionDescription
	err = pc.SetRemoteDescription(msg.Jsep)
	if err != nil {
		panic(err)
	}

	// Create consumer
	consumer := &Consumer{pc, info}
	t.consumers = append(t.consumers, consumer)

	log.Println("Subscribe complete")
}

func (t *RoomClient) UnSubscribe(info biz.MediaInfo) {
	// Send upsubscribe request
	// Shut down peerConnection
	var sub *Consumer
	for _, a := range t.consumers {
		if a.Info.Mid == info.Mid {
			sub = a
			break
		}
	}
	if sub != nil && sub.Pc != nil {
		log.Println("Closing subscription peerConnection")
		sub.Pc.Close()
	}
}

func (t *RoomClient) Leave() {

}

// Shutdown client and websocket transport
func (t *RoomClient) Close() {
	t.client.Close()

	// Close any remaining consumers
	for _, sub := range t.consumers {
		sub.Pc.Close()
	}
}
