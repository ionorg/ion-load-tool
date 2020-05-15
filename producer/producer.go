package producer

import "github.com/pion/webrtc/v2"

type TrackSelect struct {
	Audio bool
	Video bool
}

type IFileProducer interface {
	VideoTrack() *webrtc.Track
	VideoCodec() string
	AudioTrack() *webrtc.Track
	Stop()
	Start()
}
