package producer

import (
	"strings"

	"github.com/pion/webrtc/v2"
)

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

type IFilePlayer interface {
	IFileProducer
	SeekP(int)
	Pause(bool)
}

func ValidateVPFile(name string) (string, bool) {
	list := strings.Split(name, ".")
	if len(list) < 2 {
		return "", false
	}
	ext := strings.ToLower(list[len(list)-1])
	var valid bool
	// Validate is ivf|webm
	for _, a := range []string{"ivf", "webm"} {
		if a == ext {
			valid = true
		}
	}

	return ext, valid
}
