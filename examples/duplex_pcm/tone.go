package main

import (
	"context"
	"encoding/binary"
	"errors"
	"math"
	"time"
)

func validateDuration(duration time.Duration) error {
	if duration <= 0 || duration > time.Minute || duration%time.Millisecond != 0 {
		return errors.New("-duration must be a positive integral number of milliseconds, up to 1 minute")
	}
	return nil
}

// streamTone demonstrates that SendPCMAt does not pace or reinterpret the PTS.
// Every chunk contains complete interleaved PCM16LE stereo sample pairs.
func streamTone(ctx context.Context, duration time.Duration, pts int64, send func([]byte, int64) error, wait func(context.Context, time.Duration) error) error {
	if err := validateDuration(duration); err != nil {
		return err
	}
	pattern := [...]int{7, 13, 20}
	remaining := int(duration / time.Millisecond)
	sampleOffset := 0
	for i := 0; remaining > 0; i++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		millis := min(pattern[i%len(pattern)], remaining)
		samples := millis * 16 // 16 kHz: 16 samples per channel per millisecond.
		data := stereoTone(sampleOffset, samples)
		if err := send(data, pts); err != nil {
			return err
		}
		sampleOffset += samples
		remaining -= millis
		// Wait after each send, including the final chunk. Slow sends do not
		// trigger a catch-up burst; this simple pacing is not a hardware clock.
		if err := wait(ctx, time.Duration(millis)*time.Millisecond); err != nil {
			return err
		}
	}
	return nil
}

func stereoTone(offset, samples int) []byte {
	data := make([]byte, samples*2*2)
	for i := 0; i < samples; i++ {
		for channel, frequency := range [...]float64{440, 660} {
			value := int16(2000 * math.Sin(2*math.Pi*frequency*float64(offset+i)/16000))
			binary.LittleEndian.PutUint16(data[(i*2+channel)*2:], uint16(value))
		}
	}
	return data
}

func waitContext(ctx context.Context, delay time.Duration) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return ctx.Err()
	}
}
