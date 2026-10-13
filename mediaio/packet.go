package mediaio

import (
	"encoding/binary"
	"errors"
	"io"
)

// MaxPacketSize is the maximum length-prefix packet payload, in bytes (16 MiB).
// The four-byte length header is not included. It does not set the ADTS limit;
// ADTS frames have their own 13-bit total-length field.
const MaxPacketSize = 16 << 20

// ErrInvalidPacket indicates an empty or oversized length-prefix packet, or an
// ADTS frame length that leaves no payload. Other ADTS format errors are separate
// errors. Callers can identify this sentinel with errors.Is.
var ErrInvalidPacket = errors.New("mediaio: invalid packet size")

// ReadPacket reads one packet: a four-byte unsigned big-endian payload length
// followed by exactly that many payload bytes. Valid lengths are 1 through
// [MaxPacketSize], inclusive. It reads no bytes beyond that packet and returns a
// newly allocated slice that the caller may retain and modify.
//
// A zero or oversized length returns [ErrInvalidPacket] after consuming the
// header, without reading the payload. Reader errors are propagated from
// [io.ReadFull], and every error returns a nil packet. At end of input, a partial
// header or payload returns [io.ErrUnexpectedEOF]. [io.EOF] is returned if no
// header bytes remain, but also if a valid header has no payload bytes at all.
//
// ReadPacket neither closes r nor attempts to resynchronize after an error.
// Calls sharing a reader must be serialized by the caller.
func ReadPacket(r io.Reader) ([]byte, error) {
	var header [4]byte
	if _, err := io.ReadFull(r, header[:]); err != nil {
		return nil, err
	}
	size := binary.BigEndian.Uint32(header[:])
	if size == 0 || size > MaxPacketSize {
		return nil, ErrInvalidPacket
	}
	packet := make([]byte, int(size))
	if _, err := io.ReadFull(r, packet); err != nil {
		return nil, err
	}
	return packet, nil
}

// WritePacket writes packet as a four-byte unsigned big-endian payload length
// followed by the payload. It returns [ErrInvalidPacket] without writing when
// packet is empty or larger than [MaxPacketSize].
//
// WritePacket does not modify or retain packet. The caller must not modify the
// slice until the call returns. Short writes with progress are retried; a writer
// that reports no progress or an impossible byte count causes [io.ErrShortWrite].
// Other writer errors are returned unchanged. An error can leave a partial
// header or payload in w; writes are not transactional.
//
// WritePacket neither flushes nor closes w. Calls sharing a writer must be
// serialized by the caller to preserve packet boundaries.
func WritePacket(w io.Writer, packet []byte) error {
	if len(packet) == 0 || len(packet) > MaxPacketSize {
		return ErrInvalidPacket
	}
	var header [4]byte
	binary.BigEndian.PutUint32(header[:], uint32(len(packet)))
	if err := writeAll(w, header[:]); err != nil {
		return err
	}
	return writeAll(w, packet)
}

func writeAll(w io.Writer, data []byte) error {
	for len(data) > 0 {
		n, err := w.Write(data)
		if n < 0 || n > len(data) {
			return io.ErrShortWrite
		}
		data = data[n:]
		if err != nil {
			return err
		}
		if n == 0 {
			return io.ErrShortWrite
		}
	}
	return nil
}
