package mediaio

import (
	"encoding/binary"
	"fmt"
	"io"
)

// AACFrame is one raw AAC-LC access unit with its ADTS header and optional CRC
// removed. [ReadADTS] returns only mono or stereo frames containing one raw block;
// the compressed payload is not decoded or otherwise validated.
type AACFrame struct {
	// Data contains the encoded AAC access unit without ADTS framing.
	// ReadADTS allocates this slice; the caller owns it and may retain or modify it.
	Data []byte
	// SampleRate is the sampling frequency in hertz, as declared by the header.
	SampleRate int
	// Channels is the number of audio channels: 1 for mono or 2 for stereo.
	Channels int
	// SamplesPerChannel is the number of samples represented per channel.
	// ReadADTS sets this to 1024 for every supported AAC-LC access unit.
	SamplesPerChannel int
}

// ReadADTS reads one AAC-LC frame from an ADTS stream. It accepts MPEG-2 or MPEG-4
// ADTS with an explicit mono or stereo channel configuration and exactly one raw
// data block. Implicit/PCE channel configurations, other profiles, and reserved
// sample-rate indexes are rejected. Supported sample rates are 7350, 8000, 11025,
// 12000, 16000, 22050, 24000, 32000, 44100, 48000, 64000, 88200, and 96000 Hz.
//
// The total frame length includes the seven-byte header and, when present, a
// two-byte CRC. It must include at least one payload byte and cannot exceed 8191
// bytes. The CRC is consumed but not checked. On success, the result owns its
// payload, and the reader is positioned immediately after the frame.
//
// Invalid lengths return [ErrInvalidPacket]; unsupported or malformed header
// fields return descriptive errors. Header, CRC, and payload reads propagate
// errors from [io.ReadFull]. If input ends partway through any of those segments,
// the error is [io.ErrUnexpectedEOF]. EOF before the first byte of a segment is
// [io.EOF], including an entirely missing CRC or payload after a valid header.
// Every error returns a zero [AACFrame].
//
// ReadADTS neither closes r nor scans ahead to recover from malformed input.
// Calls sharing a reader must be serialized by the caller.
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
