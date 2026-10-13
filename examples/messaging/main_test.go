package main

import (
	"context"
	"testing"
	"time"
)

func TestScenarioValidation(t *testing.T) {
	for _, name := range []string{"pubsub", "presence", "metadata", "locks", "history", "stream", "direct"} {
		if err := (scenarioOptions{name: name, peer: "peer"}).validate(); err != nil {
			t.Errorf("%s: %v", name, err)
		}
	}
	for _, options := range []scenarioOptions{
		{name: "unknown"},
		{name: "direct"},
		{name: "pubsub", listen: -time.Second},
	} {
		if err := options.validate(); err == nil {
			t.Errorf("accepted %+v", options)
		}
	}
}

func TestListenForEventsStopsOnCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	listenForEvents(ctx, time.Hour)
	listenForEvents(context.Background(), 0)
}
