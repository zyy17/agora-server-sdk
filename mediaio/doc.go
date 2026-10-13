// Package mediaio reads AAC-LC ADTS frames and reads or writes bounded,
// length-prefixed packets for encoded-media pipelines. It is pure Go and does
// not require an Agora connection or the native SDK.
//
// # Length-prefixed packets
//
// [WritePacket] and [ReadPacket] use a four-byte unsigned big-endian payload
// length followed by the payload. Each call preserves one encoder packet or
// access-unit boundary. Payloads must contain 1 through [MaxPacketSize] bytes.
// The framing carries no codec, timing, sample-rate, or channel metadata; callers
// must supply that information separately. This is a small application framing
// format, not an MP4, Ogg, or other standard media container.
//
// # ADTS input
//
// [ReadADTS] consumes one ADTS frame and returns the raw AAC-LC access unit plus
// its declared sample rate, channel count, and samples per channel. It supports
// one raw block and explicit mono or stereo channel configurations. It strips
// the header and optional CRC without decoding the audio or checking the CRC.
// Its 8191-byte total-frame limit is independent of [MaxPacketSize].
//
// # Ownership and I/O
//
// Returned payloads are newly allocated and owned by the caller. Functions do
// not retain, close, or flush supplied readers and writers. Callers must
// serialize operations that share a stream. Calls block according to the supplied
// I/O; the package adds no deadlines or cancellation. Errors may leave input
// consumed or output partially written; the package does not roll back or resynchronize.
// Read functions propagate io.ReadFull errors, including io.EOF for an entirely
// missing payload and io.ErrUnexpectedEOF for a partially present segment.
// Consult the individual functions before treating EOF as a clean boundary.
package mediaio
