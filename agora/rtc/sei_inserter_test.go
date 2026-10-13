package agoraservice

import (
	"bytes"
	"testing"
)

func TestSEIRoundTrip(t *testing.T) {
	for _, codec := range []VideoCodecType{VideoCodecTypeH264, VideoCodecTypeH265} {
		for _, size := range []int{0, 1, 254, 255, 256, 2048, 65536} {
			payload := bytes.Repeat([]byte{42}, size)
			nal := AddSEI(payload, codec == VideoCodecTypeH264)
			found, n := FindSEI(nal, codec)
			if n != size || !bytes.Equal(found, payload) {
				t.Fatalf("codec %d, size %d: got %d", codec, size, n)
			}
			if len(found) > 0 {
				found[0] ^= 0xff
				if bytes.Equal(found, payload) {
					t.Fatal("returned payload not independent")
				}
			}
		}
	}
}

func TestSEIInsertionPreservesInput(t *testing.T) {
	backing := bytes.Repeat([]byte{0x77}, 4096)
	before := bytes.Clone(backing)
	result := InsertSEIToEncodedData(backing[:6], []byte("metadata"), VideoCodecTypeH264)
	if len(result) <= 6 || !bytes.Equal(backing, before) {
		t.Fatal("insertion changed caller backing array")
	}
}

func TestSEIRejectsTruncatedPayload(t *testing.T) {
	nal := AddSEI(bytes.Repeat([]byte{42}, 2048), true)
	for end := 0; end < len(nal); end++ {
		if _, n := FindSEI(nal[:end], VideoCodecTypeH264); n >= 0 {
			t.Fatalf("accepted truncation at %d", end)
		}
	}
}

func FuzzFindSEI(f *testing.F) {
	f.Add(AddSEI([]byte("metadata"), true), true)
	f.Add([]byte{0, 0, 1, 6, 0x65, 0xff, 0xff}, false)
	f.Fuzz(func(t *testing.T, data []byte, h264 bool) {
		codec := VideoCodecTypeH265
		if h264 {
			codec = VideoCodecTypeH264
		}
		out, size := FindSEI(data, codec)
		if size >= 0 && len(out) != size {
			t.Fatal("invalid returned length")
		}
	})
}
