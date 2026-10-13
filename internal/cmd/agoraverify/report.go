package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"

	agorasdk "github.com/zyy17/agora-server-sdk"
	"github.com/zyy17/agora-server-sdk/messaging"
)

// Reports deliberately have no arbitrary text fields: SDK error details,
// identities, credentials, tokens, native output and log contents cannot escape.
type result struct {
	Operation string `json:"operation"`
	Status    string `json:"status"`
	Code      int    `json:"code"`
}

type report struct {
	Results []result `json:"results"`
}

func (r *report) add(operation string, err error) {
	v := result{Operation: operation, Status: "passed"}
	if err != nil {
		v.Status = "failed"
		var rtcErr *agorasdk.Error
		var rtmErr *messaging.Error
		switch {
		case errors.Is(err, context.DeadlineExceeded):
			v.Status = "timed_out"
		case errors.Is(err, context.Canceled):
			v.Status = "canceled"
		case errors.As(err, &rtcErr):
			v.Code = rtcErr.Code
		case errors.As(err, &rtmErr):
			v.Code = int(rtmErr.Code)
		}
	}
	r.Results = append(r.Results, v)
}

func (r report) save(dir string) error {
	data, err := json.Marshal(r)
	if err != nil {
		return err
	}
	// Atomic replacement preserves the last complete checkpoint if native code
	// is terminated during the next operation or cleanup.
	if err = os.WriteFile(filepath.Join(dir, "result.tmp"), data, 0o600); err != nil {
		return err
	}
	return os.Rename(filepath.Join(dir, "result.tmp"), filepath.Join(dir, "result.json"))
}

func readReport(dir, component string) (report, error) {
	var r report
	f, err := os.Open(filepath.Join(dir, "result.json"))
	if err != nil {
		return r, errors.New("report unavailable")
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, 4097))
	if err != nil || len(data) > 4096 {
		return r, errors.New("report unavailable")
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err = decoder.Decode(&r); err != nil {
		return report{}, errors.New("report invalid")
	}
	if decoder.Decode(new(any)) != io.EOF || !validReport(r, component) {
		return report{}, errors.New("report invalid")
	}
	return r, nil
}

func validReport(r report, component string) bool {
	planned := operations(component)
	if len(planned) == 0 || len(r.Results) == 0 || len(r.Results) > len(planned) {
		return false
	}
	allowed := make(map[string]bool)
	for _, operation := range planned {
		allowed[operation] = true
	}
	seen := make(map[string]bool)
	for _, v := range r.Results {
		if !allowed[v.Operation] || seen[v.Operation] || v.Code < -100000 || v.Code > 100000 {
			return false
		}
		seen[v.Operation] = true
		switch v.Status {
		case "passed", "failed", "timed_out", "canceled", "not_run":
		default:
			return false
		}
		if v.Status == "passed" && v.Code != 0 {
			return false
		}
	}
	return true
}

func (r report) passed(component string) bool {
	required := operations(component)
	if !validReport(r, component) || len(r.Results) != len(required) {
		return false
	}
	seen := make(map[string]bool)
	for _, v := range r.Results {
		if v.Status != "passed" {
			return false
		}
		seen[v.Operation] = true
	}
	for _, operation := range required {
		if !seen[operation] {
			return false
		}
	}
	return true
}

var rtmFeatureOperations = []string{
	"prepare", "initialize", "peer_initialize", "login", "peer_login", "subscribe", "peer_subscribe",
	"publish", "receive", "renew_token", "presence_set", "presence_get", "presence_remove",
	"metadata_read", "locks_read", "history_read", "unsubscribe", "peer_unsubscribe",
	"logout", "peer_logout", "close", "peer_close",
}

func operations(component string) []string {
	switch component {
	case "rtc":
		return []string{"prepare", "initialize", "join", "leave", "close"}
	case "rtm":
		return []string{"prepare", "initialize", "login", "logout", "close"}
	case "rtc_features":
		return rtcFeatureOperations
	case "rtm_features":
		return rtmFeatureOperations
	default:
		return nil
	}
}

func (r *report) fillMissing(component string) {
	seen := make(map[string]bool)
	for _, v := range r.Results {
		seen[v.Operation] = true
	}
	for _, operation := range operations(component) {
		if !seen[operation] {
			r.Results = append(r.Results, result{Operation: operation, Status: "not_run"})
		}
	}
}

func (r report) authenticated(component string) bool {
	needed := []string{"join", "peer_join"}
	if component == "rtm_features" {
		needed = []string{"login", "peer_login"}
	}
	for _, operation := range needed {
		found := false
		for _, v := range r.Results {
			if v.Operation == operation && v.Status == "passed" {
				found = true
			}
		}
		if !found {
			return false
		}
	}
	return true
}
