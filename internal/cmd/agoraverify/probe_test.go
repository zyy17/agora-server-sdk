package main

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	agorasdk "github.com/zyy17/agora-server-sdk"
	"github.com/zyy17/agora-server-sdk/messaging"
)

type fakeRTC struct {
	calls             *[]string
	joinErr, closeErr error
	session           *fakeSession
}

func (s *fakeRTC) join(ctx context.Context, value, channel, user string) (rtcSession, error) {
	*s.calls = append(*s.calls, "join")
	if value != "synthetic-token" || channel != "synthetic-channel" || user != "synthetic-user" {
		return nil, errors.New("unexpected identity")
	}
	if s.joinErr != nil {
		return nil, s.joinErr
	}
	return s.session, ctx.Err()
}

func (s *fakeRTC) Close() error { *s.calls = append(*s.calls, "close"); return s.closeErr }

type fakeSession struct {
	calls *[]string
	err   error
}

func (s *fakeSession) Close() error { *s.calls = append(*s.calls, "leave"); return s.err }

type fakeRTM struct {
	calls                         *[]string
	loginErr, logoutErr, closeErr error
	freshLogoutContext            bool
}

func (c *fakeRTM) Login(ctx context.Context, _ string) error {
	*c.calls = append(*c.calls, "login")
	if c.loginErr != nil {
		return c.loginErr
	}
	return ctx.Err()
}

func (c *fakeRTM) Logout(ctx context.Context) error {
	*c.calls = append(*c.calls, "logout")
	_, bounded := ctx.Deadline()
	c.freshLogoutContext = ctx.Err() == nil && bounded
	return c.logoutErr
}

func (c *fakeRTM) Close() error { *c.calls = append(*c.calls, "close"); return c.closeErr }

func TestRTCCleanupOnSuccessAndFailure(t *testing.T) {
	for _, failure := range []string{"", "initialize", "join", "leave", "close"} {
		t.Run(failure, func(t *testing.T) {
			var calls []string
			var r report
			s := &fakeRTC{calls: &calls, session: &fakeSession{calls: &calls}}
			sdkErr := &agorasdk.Error{Op: "do-not-log-this", Code: -101}
			switch failure {
			case "join":
				s.joinErr = sdkErr
			case "leave":
				s.session.err = sdkErr
			case "close":
				s.closeErr = sdkErr
			}
			create := func() (rtcService, error) {
				if failure == "initialize" {
					return nil, sdkErr
				}
				return s, nil
			}
			checkRTC(context.Background(), create, "synthetic-token", "synthetic-channel", "synthetic-user", r.add)
			want := []string{"join", "leave", "close"}
			if failure == "initialize" {
				want = nil
			} else if failure == "join" {
				want = []string{"join", "close"}
			}
			if !reflect.DeepEqual(calls, want) {
				t.Fatalf("calls = %v; want %v", calls, want)
			}
			for _, v := range r.Results {
				if v.Operation == failure && (v.Status != "failed" || v.Code != -101) {
					t.Fatal("missing failure status/code")
				}
			}
		})
	}
}

func TestRTMAlwaysLogsOutAndCloses(t *testing.T) {
	for _, failure := range []string{"", "login", "logout", "close", "canceled"} {
		t.Run(failure, func(t *testing.T) {
			var calls []string
			var r report
			c := &fakeRTM{calls: &calls}
			err := &messaging.Error{Op: "secret-operation", Code: -10005, Details: "synthetic-token"}
			switch failure {
			case "login":
				c.loginErr = err
			case "logout":
				c.logoutErr = err
			case "close":
				c.closeErr = err
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if failure == "canceled" {
				cancel()
			}
			checkRTM(ctx, func() (rtmClient, error) { return c, nil }, "synthetic-token", r.add)
			if !reflect.DeepEqual(calls, []string{"login", "logout", "close"}) || !c.freshLogoutContext {
				t.Fatal("cleanup not attempted with a fresh bounded context")
			}
			for _, v := range r.Results {
				if v.Operation == failure && (v.Status != "failed" || v.Code != -10005) {
					t.Fatal("missing failure status/code")
				}
			}
		})
	}
}

func successfulReport(component string) report {
	r := report{}
	r.add("prepare", nil)
	r.add("initialize", nil)
	if component == "rtc" {
		r.add("join", nil)
		r.add("leave", nil)
	} else {
		r.add("login", nil)
		r.add("logout", nil)
	}
	r.add("close", nil)
	return r
}

func TestSupervisorKeepsResultsIndependentAndSanitized(t *testing.T) {
	var out bytes.Buffer
	var calls []string
	code := supervise(context.Background(), &out, func(_ context.Context, component string) (report, error) {
		calls = append(calls, component)
		if component == "rtc" {
			return report{}, errors.New("synthetic-sensitive-error")
		}
		return successfulReport(component), nil
	})
	if code != 1 || !reflect.DeepEqual(calls, []string{"rtc", "rtm"}) || !strings.Contains(out.String(), "rtm overall: passed") || strings.Contains(out.String(), "synthetic-sensitive-error") {
		t.Fatal("independent/sanitized reporting failed")
	}
	if (report{Results: []result{{Operation: "prepare", Status: "passed"}}}).passed("rtc") {
		t.Fatal("token generation alone counted as authentication proof")
	}
}

func TestReportsRejectUntrustedTextAndRequireFullLifecycle(t *testing.T) {
	dir := t.TempDir()
	r := successfulReport("rtc")
	if err := r.save(dir); err != nil {
		t.Fatal(err)
	}
	if parsed, err := readReport(dir, "rtc"); err != nil || !parsed.passed("rtc") {
		t.Fatal("valid report rejected")
	}
	for _, raw := range []string{
		`{"results":[{"operation":"synthetic-secret","status":"passed","code":0}]}`,
		`{"results":[{"operation":"join","status":"synthetic-secret","code":0}]}`,
		`{"results":[],"details":"synthetic-secret"}`,
		`{"results":[{"operation":"join","status":"passed","code":1}]}`,
		`{"results":[{"operation":"join","status":"passed","code":0},{"operation":"join","status":"passed","code":0}]}`,
	} {
		if err := os.WriteFile(filepath.Join(dir, "result.json"), []byte(raw), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := readReport(dir, "rtc"); err == nil {
			t.Fatal("unsafe report accepted")
		}
	}
}

func TestLiveOptInIsRequired(t *testing.T) {
	c := syntheticCredentials()
	t.Setenv("AGORA_APP_ID", c.appID)
	t.Setenv("AGORA_APP_CERTIFICATE", c.certificate)
	t.Setenv("AGORA_VERIFY_LIVE", "")
	var out bytes.Buffer
	if run([]string{"--live"}, &out) != 2 {
		t.Fatal("live request without opt-in accepted")
	}
	if strings.Contains(out.String(), c.appID) || strings.Contains(out.String(), c.certificate) {
		t.Fatal("credential exposure")
	}
	out.Reset()
	if run([]string{"--check"}, &out) != 0 || !strings.Contains(out.String(), "not live authentication proof") {
		t.Fatal("format check incorrectly reported")
	}
}
