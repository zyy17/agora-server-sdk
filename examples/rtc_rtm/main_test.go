package main

import (
	"context"
	"io"
	"reflect"
	"testing"
)

func TestDrainConsumesBufferedValuesBeforeEOF(t *testing.T) {
	queue := make(chan []byte, 2)
	queue <- []byte{0x00, 0xff}
	queue <- []byte{0x01, 0x80}
	close(queue)
	var got [][]byte
	drain(func(ctx context.Context) ([]byte, error) {
		if ctx.Err() != nil {
			t.Fatal("drain used a cancelled context")
		}
		value, ok := <-queue
		if !ok {
			return nil, io.EOF
		}
		return value, nil
	}, func(data []byte) { got = append(got, data) })
	if !reflect.DeepEqual(got, [][]byte{{0x00, 0xff}, {0x01, 0x80}}) {
		t.Fatalf("got %x", got)
	}
}
