package godub

import (
	"time"

	"github.com/Vernacular-ai/godub/utils"

	"github.com/google/go-cmp/cmp"
)

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func matchTargetAmp(sound *AudioSegment, targetDBFS Volume) *AudioSegment {
	changeInDBFS := targetDBFS - sound.DBFS()
	ret, _ := sound.ApplyGain(changeInDBFS)
	return ret
}

func detectSilence(seg *AudioSegment, minSilenceLen int, silenceThresh Volume, seekStep int) [][]int {
	segLen := utils.Milliseconds(seg.Duration())

	// you can't have a silent portion of a sound that is longer than the sound
	if segLen < minSilenceLen {
		var emp [][]int

		return emp
	}

	// convert silence threshold to a float value (so we can compare it to rms)
	var silThresh = silenceThresh.ToRatio(true) * seg.MaxPossibleAmplitude()

	// find silence and add start and end indicies to the to_cut list
	var silenceStarts []int

	// check successive (1 ms by default) chunk of sound for silence
	// try a chunk at every "seek step" (or every chunk for a seek step == 1)
	lastSliceStart := segLen - minSilenceLen

	var sliceStarts []int
	for i := 0; i < lastSliceStart+1; i += seekStep {
		sliceStarts = append(sliceStarts, i)
	}

	// guarantee lastSliceStart is included in the range
	// to make sure the last portion of the audio is searched
	if (lastSliceStart % seekStep) != 0 {
		sliceStarts = append(sliceStarts, lastSliceStart)
	}

	for _, i := range sliceStarts {
		audioSlice, _ := seg.Slice(time.Duration(i), time.Duration(i+minSilenceLen))
		if audioSlice.RMS() <= silThresh {
			silenceStarts = append(silenceStarts, i)

		}
	}
	// short circuit when there is no silence
	if len(silenceStarts) == 0 {
		var silentRanges [][]int
		return silentRanges
	}

	// combine the silence we detected into ranges (start ms - end ms)
	var silentRanges [][]int

	prevI, silenceStarts := silenceStarts[0], silenceStarts[1:]
	currentRangeStart := prevI

	for _, silenceStartI := range silenceStarts {
		var continuous bool
		var silenceHasGap bool
		if silenceStartI == prevI+seekStep {
			continuous = true
		} else {
			continuous = false
		}

		// sometimes two small blips are enough for one particular slice to be
		// non-silent, despite the silence all running together. Just combine
		// the two overlapping silent ranges.

		if silenceStartI > prevI+minSilenceLen {
			silenceHasGap = true
		} else {
			silenceHasGap = false
		}

		if continuous == false && silenceHasGap == true {

			silentRanges = append(silentRanges, []int{currentRangeStart, prevI + minSilenceLen})
			currentRangeStart = silenceStartI
		}
		prevI = silenceStartI

	}
	silentRanges = append(silentRanges, []int{currentRangeStart, prevI + minSilenceLen})

	return silentRanges
}

func detectNonsilent(seg *AudioSegment, minSilenceLen int, silenceThresh Volume, seekStep int) [][]int {

	silentRanges := detectSilence(seg, minSilenceLen, silenceThresh, seekStep)

	lenSeg := utils.Milliseconds(seg.Duration())
	var nonsilentRanges [][]int
	// if there is no silence, the whole thing is nonsilent
	if len(silentRanges) == 0 {
		return append(nonsilentRanges, []int{0, lenSeg})
	}

	// short circuit when the whole audio segment is silent
	if silentRanges[0][0] == 0 && silentRanges[0][1] == lenSeg {
		return nonsilentRanges
	}

	prevEndI := 0
	endI := 0
	for i := range silentRanges {

		nonsilentRanges = append(nonsilentRanges, []int{prevEndI, silentRanges[i][0]})
		prevEndI = silentRanges[i][1]

		endI = prevEndI
	}

	if endI != lenSeg {
		nonsilentRanges = append(nonsilentRanges, []int{prevEndI, lenSeg})
	}

	if cmp.Equal(nonsilentRanges[0], []time.Duration{0, 0}) {
		nonsilentRanges = nonsilentRanges[1:]
	}

	return nonsilentRanges
}

func SplitOnSilence(seg *AudioSegment, minSilenceLen int, silenceThresh Volume, keep_silence int, seekStep int) []*AudioSegment {
	chunks := []*AudioSegment{}
	normAudio, _ := seg.derive(seg.RawData())
	normAudio = matchTargetAmp(seg, -20.0)

	notSilenceRanges := detectNonsilent(normAudio, minSilenceLen, silenceThresh, seekStep)

	startMin := 0

	if len(notSilenceRanges) == 1 {
		chunks = append(chunks, seg)
		return chunks

	}
	for i := 0; i < len(notSilenceRanges)-1; i++ {
		endMax := notSilenceRanges[i][1] + (notSilenceRanges[i+1][0]-notSilenceRanges[i][1]+1)/2
		startI := max(int(startMin), int(notSilenceRanges[i][0]-keep_silence))
		endI := min(int(endMax), int(notSilenceRanges[i][1]+keep_silence))

		temp1, _ := seg.Slice(time.Duration(startI), time.Duration(endI))
		if temp1 != nil {
			chunks = append(chunks, temp1)
		}

		startMin = notSilenceRanges[i][1]
	}

	temp2, _ := seg.Slice(time.Duration(max(startMin, notSilenceRanges[len(notSilenceRanges)-1][0]-keep_silence)), time.Duration(min(utils.Milliseconds(seg.Duration()), notSilenceRanges[len(notSilenceRanges)-1][1]+keep_silence)))
	if temp2 != nil {
		chunks = append(chunks, temp2)

	}
	return chunks
}

func detectLeadingSilence(sound *AudioSegment, silenceThreshold Volume, chunkSize int) int {
	trimMS := 0
	for trimMS < utils.Milliseconds(sound.Duration()) {
		temp1, _ := sound.Slice(time.Duration(trimMS), time.Duration(trimMS+chunkSize))
		if temp1.DBFS() < silenceThreshold {
			trimMS += chunkSize
		} else {
			break
		}
	}

	return trimMS
}
