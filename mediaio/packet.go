// Package mediaio provides bounded packet framing for encoded-media pipelines.
package mediaio

import (
	"encoding/binary"
	"errors"
	"io"
)

const MaxPacketSize = 16 << 20

var ErrInvalidPacket = errors.New("mediaio: invalid packet size")

// ReadPacket reads a uint32 big-endian size followed by exactly that many bytes.
// The format retains encoder packet/access-unit boundaries; it is not MP4/Ogg.
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
