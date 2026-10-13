package mediaio

import (
	"encoding/binary"
	"fmt"
	"io"
)

// AACFrame is a raw AAC-LC access unit with its ADTS header removed.
type AACFrame struct {
	Data                                    []byte
	SampleRate, Channels, SamplesPerChannel int
}

// ReadADTS reads one AAC-LC ADTS frame. Multi-block and implicit/PCE channel
// configurations are rejected rather than guessed.
func ReadADTS(r io.Reader) (AACFrame, error) {
	var h [7]byte
	if _, err := io.ReadFull(r, h[:]); err != nil {
		return AACFrame{}, err
	}
	if h[0] != 0xff || h[1]&0xf6 != 0xf0 {
		return AACFrame{}, fmt.Errorf("mediaio: invalid ADTS sync/layer")
	}
	rates := [...]int{96000, 88200, 64000, 48000, 44100, 32000, 24000, 22050, 16000, 12000, 11025, 8000, 7350}
	index := (h[2] >> 2) & 15
	if int(index) >= len(rates) {
		return AACFrame{}, fmt.Errorf("mediaio: invalid ADTS sample rate")
	}
	if h[2]>>6 != 1 {
		return AACFrame{}, fmt.Errorf("mediaio: only AAC-LC ADTS is supported")
	}
	channels := int(h[2]&1)<<2 | int(h[3]>>6)
	if channels < 1 || channels > 2 || h[6]&3 != 0 {
		return AACFrame{}, fmt.Errorf("mediaio: ADTS requires mono/stereo and one raw block")
	}
	length := int(binary.BigEndian.Uint16(h[3:5])&0x03ff)<<3 | int(h[5]>>5)
	headerSize := 7
	if h[1]&1 == 0 {
		headerSize = 9
		var crc [2]byte
		if _, err := io.ReadFull(r, crc[:]); err != nil {
			return AACFrame{}, err
		}
	}
	if length <= headerSize || length > 8191 {
		return AACFrame{}, ErrInvalidPacket
	}
	data := make([]byte, length-headerSize)
	if _, err := io.ReadFull(r, data); err != nil {
		return AACFrame{}, err
	}
	return AACFrame{data, rates[index], channels, 1024}, nil
}
