## Ion load testing tool
Ion load tool support publish or subscribe streaming

### Install
```
go get -u github.com/pion/ion-load-tool
```

### Test Data
Publishing of files in the following formats are supported.

|Container|Video Codecs|Audio|
|---|---|---|
WEBM|VP8|OPUS

If your data is not webm, you can transcode
This show how to make a 0.5Mbps webm:
```
ffmpeg -i djrm480p.mp4 -strict -2 -b:v 0.4M -vcodec libvpx -acodec opus djrm480p.webm
```

See the ffmpeg docs on [VP8](https://trac.ffmpeg.org/wiki/Encode/VP8) or [VP9](https://trac.ffmpeg.org/wiki/Encode/VP9) for encoding options

### Quick Start
You need another host in the same LAN with the ion-sfu.
You can make a script and run:

```
#!/bin/bash
ulimit -c unlimited
ulimit -SHn 1000000
sysctl -w net.ipv4.tcp_keepalive_time=60
sysctl -w net.ipv4.tcp_timestamps=0
sysctl -w net.ipv4.tcp_tw_reuse=1
#sysctl -w net.ipv4.tcp_tw_recycle=0
sysctl -w net.core.somaxconn=65535
sysctl -w net.ipv4.tcp_max_syn_backlog=65535
sysctl -w net.ipv4.tcp_syncookies=1

#publish
ion-load-tool -input ./djrm480p.webm -clients 1 -role pub -url "ion-sfu-ip:grpc-port"

#subscribe
#ion-load-tool -clients 100 -role sub -cycle 1000 -url "ion-sfu-ip:grpc-port"
```

### Command Line

```
./ion-load-tool --help 
Usage of ./ion-load-tool:
  -clients int
    	Number of clients to start (default 1)
  -cycle int
    	Run new client cycle in ms (default 300)
  -duration int
    	Running duration in sencond (default 3600)
  -input string
    	Path to the input media (default "./input.webm")
  -loglevel string
    	Log level (default "info")
  -role string
    	Run as pub/sub/pubsub  (sender/receiver/both) (default "pubsub")
  -room string
    	Room to join (default "room")
  -url string
    	Ion-sfu grpc url (default "localhost:50051")
```

