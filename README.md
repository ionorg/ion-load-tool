## Ion load testing tool

### Requirements

Uses VP8 playback based on [play-from-disk](https://github.com/pion/webrtc/tree/master/examples/play-from-disk)

First encode your test file
```
ffmpeg -i $INPUT_FILE -g 30 output.ivf
```
See the [ffmpeg VP8 docs](https://trac.ffmpeg.org/wiki/Encode/VP8) for more encoding options

### Run

`ion-load -clients <num clients>`


#### Producer

Pass `-produce -container-path <encoded video>`

Each client starts stream playback in an offset position from the last with a different track and SSRC ID to simulate independent content.


#### Consumers

Pass `-consume`

Each client subscribes to all published streams in the provided room. A basic consumer with simple out-of-order detection is implemented.


### Test Configurations

#### N to N fully connected

Run both produce and consume on the same command

`-produce -consume -container-file <file> -clients N`

This creates N clients publishing a stream, each of which will subscribe to the other N-1 client streams.

#### 1 to N fanout

Run separate instances of the load tool.

##### Producer

`-produce -container-file <file> -clients 1`

##### Consumer
`-consume -clients N`
