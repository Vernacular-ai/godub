package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"path"
	"strconv"
	"time"

	"github.com/Vernacular-ai/godub"
)

func main() {
	filePath := path.Join("code-geass.wav")
	dat, _ := ioutil.ReadFile(filePath)

	// get AudioSegment from buffer
	segment, _ := godub.NewLoader().Load(bytes.NewReader(dat))

	tFP := "./out"

	// Actual splitting
	// taking the threshold as a ratio of the avg Volume of the audio. Change the "0.35" to change ratio
	//threshold := godub.NewVolumeFromRatio(segment.DBFS().ToRatio(false)*0.15, 0, false)
	threshold := godub.Volume(-23) //Setting a threshold manually

	// Use the below comment to set a static threshold
	// threshold := godub.Volume(-23)
	chunks, timings, err := godub.SplitOnSilence(segment, 1000, threshold, 1000, 1)

	if err != nil {
		fmt.Printf("Error: %v", err)
	}
	// fmt.Printf("%v", timings)
	start := 0
	end := len(chunks)

	var segs []*godub.AudioSegment

	dur := time.Duration(0) * time.Millisecond

	var chunkTimes [][]float32

	// Combines chunks if below 30 seconds. Will keep combining chunks until it is below 30s. If adding the
	// next chunk makes it over 30, it won't add that. So the audio may be 10s long if the next chunk is
	// 22s
	// Change maxLen to change the maximum chunk lenght. Should be [10, 45]
	maxLen := 30
	for i := 0; i < end; i++ {
		if dur+chunks[i].Duration() >= time.Duration(maxLen*1000)*time.Millisecond {
			seg, _ := godub.NewAudioSegment([]byte{}, godub.Channels(1), godub.SampleWidth(segment.SampleWidth()), godub.FrameRate(segment.FrameRate()), godub.FrameWidth(segment.FrameWidth()))

			seg, _ = seg.Append(chunks[start:i]...)
			segs = append(segs, seg)
			chunkTimes = append(chunkTimes, []float32{timings[start][0], timings[i-1][1]})
			start = i
			dur = chunks[i].Duration()
		} else {
			dur += chunks[i].Duration()

		}

	}

	// Check for last chunk. There's one left
	if start != end {

		seg, _ := godub.NewAudioSegment([]byte{}, godub.Channels(1), godub.SampleWidth(segment.SampleWidth()), godub.FrameRate(segment.FrameRate()), godub.FrameWidth(segment.FrameWidth()))
		seg, _ = seg.Append(chunks[start:end]...)
		chunkTimes = append(chunkTimes, []float32{timings[start][0], timings[end-1][1]})
		segs = append(segs, seg)
	}

	// Save all chunks
	for i := range segs {
		toFilePath := path.Join(tFP, "seg_"+strconv.Itoa(i))
		godub.NewExporter(toFilePath).WithDstFormat("wav").WithBitRate(8000).Export(segs[i])

	}
}
