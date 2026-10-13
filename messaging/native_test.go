package messaging

import (
	"context"
	"os"
	"testing"
)

// Construction exercises native ownership only. No Login or other server request.
func TestNativeCreateCloseOffline(t *testing.T) {
	if os.Getenv("AGORA_NATIVE_SMOKE") != "1" {
		t.Skip("set AGORA_NATIVE_SMOKE=1 for native construction smoke")
	}
	t.Chdir(t.TempDir())
	client, err := New(Config{AppID: "0123456789abcdef0123456789abcdef", UserID: "go-sdk-offline-test"})
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	if Version() == "" {
		t.Fatal("empty RTM version")
	}
	stream, err := client.NewStream(context.Background(), "offline-example")
	if err != nil {
		t.Fatal(err)
	}
	if err := stream.Close(); err != nil {
		t.Fatal(err)
	}
	if err := stream.Close(); err != nil {
		t.Fatal(err)
	}
	if err := client.Close(); err != nil {
		t.Fatal(err)
	}
	if err := client.Close(); err != nil {
		t.Fatal(err)
	}
}
