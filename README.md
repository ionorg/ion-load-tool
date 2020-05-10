## Ion load testing tool

### Requirements

Uses VP8 playback based on [play-from-disk](https://github.com/pion/webrtc/tree/master/examples/play-from-disk)

First encode your test file
```
ffmpeg -i $INPUT_FILE -g 30 output.ivf
```
See the [ffmpeg VP8 docs](https://trac.ffmpeg.org/wiki/Encode/VP8) for more encoding options

### Run
`ion-load -container-path <encoded video>`

Set client count with `-clients` default 1.

### Status

Opens requested number of clients in the provided room name.

Each client starts stream playback in an offset position from the last with a different track and SSRC ID to simulate independent content.

No consuming is currently implemented, but the room can be viewed in the browser when using a limited number of clients.
