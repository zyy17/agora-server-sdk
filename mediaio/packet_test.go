package mediaio

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"testing"
)

func TestPacketRoundTrip(t *testing.T) {
	var b bytes.Buffer
	for _, p := range [][]byte{{1}, {2, 3, 4}} {
		if err := WritePacket(&b, p); err != nil {
			t.Fatal(err)
		}
	}
	for _, want := range [][]byte{{1}, {2, 3, 4}} {
		p, err := ReadPacket(&b)
		if err != nil || !bytes.Equal(p, want) {
			t.Fatal(p, err)
		}
	}
	if _, err := ReadPacket(&b); !errors.Is(err, io.EOF) {
		t.Fatal(err)
	}
}

func TestPacketRejectsSizeAndTruncation(t *testing.T) {
	for _, n := range []uint32{0, MaxPacketSize + 1} {
		var b [4]byte
		binary.BigEndian.PutUint32(b[:], n)
		if _, err := ReadPacket(bytes.NewReader(b[:])); !errors.Is(err, ErrInvalidPacket) {
			t.Fatal(err)
		}
	}
	if _, err := ReadPacket(bytes.NewReader([]byte{0, 0, 0, 3, 1})); !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatal(err)
	}
	if err := WritePacket(io.Discard, nil); !errors.Is(err, ErrInvalidPacket) {
		t.Fatal(err)
	}
}

func TestADTS(t *testing.T) {
	frame := []byte{0xff, 0xf1, 0x60, 0x40, 0x01, 0x5f, 0xfc, 1, 2, 3}
	got, err := ReadADTS(bytes.NewReader(frame))
	if err != nil {
		t.Fatal(err)
	}
	if got.SampleRate != 16000 || got.Channels != 1 || !bytes.Equal(got.Data, []byte{1, 2, 3}) {
		t.Fatal(got)
	}
	for _, v := range [][]byte{{}, {0}, {0xff, 0xf1, 0xfc, 0, 0, 0, 0}} {
		if _, err := ReadADTS(bytes.NewReader(v)); err == nil {
			t.Fatal("invalid ADTS accepted")
		}
	}
}

func FuzzPacket(f *testing.F) {
	f.Add([]byte{0, 0, 0, 1, 1})
	f.Fuzz(func(t *testing.T, b []byte) {
		if len(b) > MaxPacketSize+4 {
			return
		}
		_, _ = ReadPacket(bytes.NewReader(b))
	})
}

func FuzzADTS(f *testing.F) {
	f.Add([]byte{0xff, 0xf1, 0x60, 0x40, 0x01, 0x5f, 0xfc, 1, 2, 3})
	f.Fuzz(func(t *testing.T, b []byte) { _, _ = ReadADTS(bytes.NewReader(b)) })
}
