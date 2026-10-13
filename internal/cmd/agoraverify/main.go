// Command agoraverify performs bounded Agora authentication checks or a selected
// synthetic RTC/RTM integration suite. It does not certify the full SDK surface.
package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"
)

const (
	liveTimeout  = 90 * time.Second
	probeTimeout = 35 * time.Second
	killGrace    = 5 * time.Second
)

func main() { os.Exit(run(os.Args[1:], os.Stdout)) }

func run(args []string, out io.Writer) int {
	if len(args) == 0 || (args[0] != "--check" && args[0] != "--live" && args[0] != "--probe") {
		fmt.Fprintln(out, "usage: agoraverify --check | --live [--features] (requires AGORA_VERIFY_LIVE=1)")
		return 2
	}
	features := len(args) == 2 && args[0] == "--live" && args[1] == "--features"
	if (args[0] == "--check" && len(args) != 1) || (args[0] == "--live" && len(args) != 1 && !features) {
		return 2
	}
	c := credentials{os.Getenv("AGORA_APP_ID"), os.Getenv("AGORA_APP_CERTIFICATE")}
	if err := c.validate(); err != nil {
		fmt.Fprintln(out, err)
		return 2
	}
	if args[0] == "--check" {
		fmt.Fprintln(out, "Credential presence and format: passed (not live authentication proof)")
		return 0
	}
	if os.Getenv("AGORA_VERIFY_LIVE") != "1" {
		fmt.Fprintln(out, "Live checks require explicit AGORA_VERIFY_LIVE=1; no Agora connection attempted")
		return 2
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if args[0] == "--probe" {
		if len(args) != 3 {
			return 2
		}
		return probe(ctx, c, args[1], args[2])
	}
	ctx, cancel := context.WithTimeout(ctx, liveTimeout)
	defer cancel()
	exe, err := os.Executable()
	if err != nil {
		fmt.Fprintln(out, "Verifier setup: failed")
		return 1
	}
	root, err := os.MkdirTemp("", "agora-verify-")
	if err != nil {
		fmt.Fprintln(out, "Verifier temporary directory: failed")
		return 1
	}
	defer os.RemoveAll(root)
	components := []string{"rtc", "rtm"}
	if features {
		components = []string{"rtc_features", "rtm_features"}
		fmt.Fprintln(out, "Bounded synthetic RTC/RTM integration checks; no external AI/model calls")
		fmt.Fprintln(out, "Not exercised: encoded codecs/other CPU formats, encryption/proxy/platform variants, reconnect/expiry/failure injection, APM/VAD/AI, RTM stream topics, durable storage/lock mutations, and load. Example programs are compiled separately, not run here.")
	} else {
		fmt.Fprintln(out, "Auth-only verification; no audio, video, messages, AI, or model calls")
	}
	return superviseComponents(ctx, out, components, func(ctx context.Context, component string) (report, error) {
		dir := filepath.Join(root, component)
		if err := os.Mkdir(dir, 0o700); err != nil {
			return report{}, err
		}
		defer os.RemoveAll(dir)
		pctx, pcancel := context.WithTimeout(ctx, probeTimeout)
		defer pcancel()
		cmd := exec.CommandContext(pctx, exe, "--probe", component, dir)
		cmd.Dir = dir
		cmd.Env = os.Environ()
		return isolatedProbe(pctx, cmd, dir, component, killGrace)
	})
}

// isolatedProbe contains the process boundary exercised by offline subprocess
// tests. The command must have been created with the supplied context.
func isolatedProbe(ctx context.Context, cmd *exec.Cmd, dir, component string, grace time.Duration) (report, error) {
	// Never stream native diagnostics to CI or retain them as artifacts.
	// Tokens stay in child memory; no derived token needs to be printed even
	// as a masking command. Native file paths are inside this private dir.
	cmd.Stdout, cmd.Stderr = io.Discard, io.Discard
	cmd.Cancel = func() error { return cmd.Process.Signal(syscall.SIGTERM) }
	cmd.WaitDelay = grace
	runErr := cmd.Run()
	r, reportErr := readReport(dir, component)
	if ctx.Err() != nil {
		return r, ctx.Err()
	}
	if runErr != nil {
		return r, runErr
	}
	return r, reportErr
}

type probeRunner func(context.Context, string) (report, error)

func supervise(ctx context.Context, out io.Writer, runProbe probeRunner) int {
	return superviseComponents(ctx, out, []string{"rtc", "rtm"}, runProbe)
}

func superviseComponents(ctx context.Context, out io.Writer, components []string, runProbe probeRunner) int {
	status := 0
	for _, component := range components {
		if ctx.Err() != nil {
			fmt.Fprintf(out, "%s overall: not_run (live budget exhausted or canceled)\n", component)
			status = 1
			continue
		}
		r, err := runProbe(ctx, component)
		if len(r.Results) == 0 || validReport(r, component) {
			r.fillMissing(component)
		}
		if validReport(r, component) {
			for _, v := range r.Results {
				fmt.Fprintf(out, "%s %s: %s (code=%d)\n", component, v.Operation, v.Status, v.Code)
			}
		}
		if component == "rtc_features" || component == "rtm_features" {
			if validReport(r, component) && r.authenticated(component) {
				fmt.Fprintf(out, "%s authentication: passed (feature results are independent)\n", component)
			} else {
				fmt.Fprintf(out, "%s authentication: failed_or_incomplete\n", component)
			}
		}
		if err != nil || !r.passed(component) {
			// Never format err: native details, exit errors and paths are untrusted.
			switch {
			case errors.Is(err, context.DeadlineExceeded):
				fmt.Fprintf(out, "%s overall: timed_out\n", component)
			case errors.Is(err, context.Canceled):
				fmt.Fprintf(out, "%s overall: canceled\n", component)
			default:
				fmt.Fprintf(out, "%s overall: failed_or_incomplete\n", component)
			}
			status = 1
		} else {
			fmt.Fprintf(out, "%s overall: passed\n", component)
		}
	}
	fmt.Fprintln(out, "Interpret RTC and RTM independently; RTM may need separate project enablement. Token creation alone is not proof.")
	return status
}
