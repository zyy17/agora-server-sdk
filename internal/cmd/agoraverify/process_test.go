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
	"testing"
	"time"
)

// A fake worker executable emits deliberately credential-like output. It never
// reads credentials or calls native APIs. Its output must be discarded, while
// only the checkpoint file and exit status cross the supervisor boundary.
func TestProbeProcessHelper(t *testing.T) {
	mode := os.Getenv("AGORA_TEST_HELPER_MODE")
	if mode == "" {
		return
	}
	dir := os.Getenv("AGORA_TEST_HELPER_DIR")
	fmt.Fprintln(os.Stdout, "SYNTHETIC-NATIVE-STDOUT-DO-NOT-FORWARD")
	fmt.Fprintln(os.Stderr, "SYNTHETIC-NATIVE-STDERR-DO-NOT-FORWARD")
	if mode == "hang" {
		signal.Ignore(syscall.SIGTERM)
		r := report{}
		r.add("prepare", nil)
		if err := r.save(dir); err != nil {
			os.Exit(3)
		}
		for {
			time.Sleep(time.Hour)
		}
	}
	if mode == "fail" {
		os.Exit(1)
	}
	if err := successfulReport("rtc").save(dir); err != nil {
		os.Exit(3)
	}
	os.Exit(0)
}

func TestIsolatedProcessHasBoundedWaitAndSuppressesNativeOutput(t *testing.T) {
	for _, mode := range []string{"success", "fail", "hang"} {
		t.Run(mode, func(t *testing.T) {
			dir := t.TempDir()
			duration := 3 * time.Second
			if mode == "hang" {
				duration = time.Second
			}
			ctx, cancel := context.WithTimeout(context.Background(), duration)
			defer cancel()
			cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestProbeProcessHelper$")
			cmd.Env = append(os.Environ(), "AGORA_TEST_HELPER_MODE="+mode, "AGORA_TEST_HELPER_DIR="+dir)
			cmd.Dir = dir
			start := time.Now()
			r, err := isolatedProbe(ctx, cmd, dir, "rtc", 100*time.Millisecond)
			if cmd.Stdout != io.Discard || cmd.Stderr != io.Discard {
				t.Fatal("native output was not suppressed")
			}
			switch mode {
			case "success":
				if err != nil || !r.passed("rtc") {
					t.Fatal("fake probe should succeed")
				}
			case "fail":
				if err == nil || r.passed("rtc") {
					t.Fatal("failed probe should fail")
				}
			case "hang":
				if !errors.Is(err, context.DeadlineExceeded) || r.passed("rtc") {
					t.Fatal("hung probe should time out")
				}
				if len(r.Results) != 1 || r.Results[0].Operation != "prepare" {
					t.Fatal("last complete checkpoint lost")
				}
				if time.Since(start) > 2500*time.Millisecond {
					t.Fatal("native hang exceeded bounded termination grace")
				}
			}
			if cmd.ProcessState == nil || !cmd.ProcessState.Exited() && cmd.ProcessState.Sys().(syscall.WaitStatus).Signal() != syscall.SIGKILL {
				t.Fatal("fake probe was not reaped")
			}
			entries, err := os.ReadDir(dir)
			if err != nil {
				t.Fatal(err)
			}
			for _, entry := range entries {
				if filepath.Ext(entry.Name()) != ".json" {
					t.Fatal("unexpected retained diagnostics")
				}
			}
		})
	}
}
