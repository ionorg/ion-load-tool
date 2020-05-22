package ion

import (
	"github.com/pion/ion/pkg/proto"
)

func newPublishOptions(codec string) proto.PublishOptions {
	return proto.PublishOptions{
		Codec:      codec,
		Resolution: "hd",
		Bandwidth:  1024,
		Audio:      true,
		Video:      true,
		Screen:     false,
	}
}
