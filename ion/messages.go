package ion

import "github.com/pion/webrtc/v2"

type RoomInfo struct {
	Rid string `json:"rid"`
	Uid string `json:"uid"`
}

type ChatInfo struct {
	Msg        string `json:"msg"`
	SenderName string `json:"senderName"`
}

type UserInfo struct {
	Name string `json:"name"`
}

type PublishOptions struct {
	Codec      string `json:"codec"`
	Resolution string `json:"resolution"`
	Bandwidth  int    `json:"bandwidth"`
	Audio      bool   `json:"audio"`
	Video      bool   `json:"video"`
	Screen     bool   `json:"screen"`
}

type JoinMsg struct {
	RoomInfo
	Info UserInfo `json:"info"`
}

type ChatMsg struct {
	RoomInfo
	Info ChatInfo `json:"info"`
}

type PublishMsg struct {
	RoomInfo
	Jsep    webrtc.SessionDescription `json:"jsep"`
	Options PublishOptions            `json:"options"`
}

type connectMsg struct {
	Ans webrtc.SessionDescription `json:"jsep"`
}

func newPublishOptions() PublishOptions {
	return PublishOptions{
		Codec:      "h264",
		Resolution: "hd",
		Bandwidth:  1024,
		Audio:      true,
		Video:      true,
		Screen:     false,
	}
}
