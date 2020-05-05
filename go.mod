module github.com/jbrady42/ion-load

go 1.13

require (
	github.com/cloudwebrtc/go-protoo v0.0.0-20200503150352-84211f461572
	github.com/gorilla/websocket v1.4.2 // indirect
	github.com/pion/ion v0.3.0
	github.com/pion/webrtc/v2 v2.2.8
)

replace github.com/pion/ion => github.com/jbrady42/ion v0.3.1
