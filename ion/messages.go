package ion

import (
	"github.com/pion/ion/pkg/node/biz"
)

func newPublishOptions(codec string) biz.PublishOptions {
	return biz.PublishOptions{
		Codec:      codec,
		Resolution: "hd",
		Bandwidth:  1024,
		Audio:      true,
		Video:      true,
		Screen:     false,
	}
}
