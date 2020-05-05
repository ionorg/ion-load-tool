package ion

import (
	"github.com/pion/ion/pkg/node/biz"
)

func newPublishOptions() biz.PublishOptions {
	return biz.PublishOptions{
		Codec:      "vp8",
		Resolution: "hd",
		Bandwidth:  1024,
		Audio:      false,
		Video:      true,
		Screen:     false,
	}
}
