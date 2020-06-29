## Ion load testing tool

### Test Data
Publishing of files in the following formats are supported.

|Container|Video Codecs|Audio|
|---|---|---|
IVF| VP8 | none
WEBM| VP8 and VP9 | ogg



If your data is not in this format you can transcode with:
```
ffmpeg -i $INPUT_FILE -g 30 output.(ivf|webm)
```

See the ffmpeg docs on [VP8](https://trac.ffmpeg.org/wiki/Encode/VP8) or [VP9](https://trac.ffmpeg.org/wiki/Encode/VP9) for encoding options

### Run

`ion-load-tool -input input.ivf -clients 2 -produce -consume`

#### Input

Pass `-input`

File inputs and stream inputs are supported. For a files, the path can be a `ivf` or `webm` file. For streams, we support an `mid` published to the specified `sfu`

#### Room

Pass `-room`

Specify a room to pubuce/consume from. Specify multiple rooms with `-room room1 -room room2`.

#### Producer

Pass `-produce`

Each client starts stream playback in an offset position from the last with a different track and SSRC ID to simulate independent content.


#### Consumers

Pass `-consume`

Each client subscribes to all published streams in the provided room. A basic consumer with simple out-of-order detection is implemented.


### Test Configurations

#### N to N fully connected

Run produce with multiple clients.

`-input <file> -produce -consume -clients N`

This creates N clients publishing a stream, each of which will subscribe to the other N-1 client streams.

#### 1 to N fanout

Run separate instances of the load tool.

##### Producer

`-input <file> -produce -clients 1`

##### Consumer
`-consume -clients N`
