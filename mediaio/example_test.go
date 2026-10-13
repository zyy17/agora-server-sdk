package mediaio_test

import (
	"bytes"
	"fmt"
	"io"

	"github.com/zyy17/agora-server-sdk/mediaio"
)

func ExampleWritePacket() {
	var stream bytes.Buffer
	if err := mediaio.WritePacket(&stream, []byte{1, 2, 3}); err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("% x\n", stream.Bytes())
	// Output: 00 00 00 03 01 02 03
}

func ExampleReadPacket() {
	// Two complete packets with lengths of two bytes and one byte.
	stream := bytes.NewReader([]byte{
		0, 0, 0, 2, 0x11, 0x22,
		0, 0, 0, 1, 0x33,
	})
	for {
		packet, err := mediaio.ReadPacket(stream)
		if err == io.EOF {
			break
		}
		if err != nil {
			fmt.Println(err)
			return
		}
		fmt.Printf("% x\n", packet)
	}
	// Output:
	// 11 22
	// 33
}

func ExampleReadADTS() {
	// A synthetic ADTS header declaring AAC-LC, 16 kHz, mono, and one block.
	// The three placeholder payload bytes are framed, not decoded.
	stream := bytes.NewReader([]byte{
		0xff, 0xf1, 0x60, 0x40, 0x01, 0x5f, 0xfc,
		1, 2, 3,
	})
	frame, err := mediaio.ReadADTS(stream)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("%d Hz, %d channel, %d samples per channel\n",
		frame.SampleRate, frame.Channels, frame.SamplesPerChannel)
	fmt.Printf("payload: % x\n", frame.Data)
	// Output:
	// 16000 Hz, 1 channel, 1024 samples per channel
	// payload: 01 02 03
}
