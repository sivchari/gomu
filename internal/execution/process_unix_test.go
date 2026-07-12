//go:build unix

package execution

import (
	"context"
	"errors"
	"os"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestRunBoundedCommandKillsDescendantsOnTimeout(t *testing.T) {
	pidFile := t.TempDir() + "/child.pid"
	t.Setenv("CHILD_PID_FILE", pidFile)

	ctx, cancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
	defer cancel()

	_, err := runBoundedCommand(
		ctx,
		"",
		"sh",
		"-c",
		`sh -c 'while :; do sleep 1; done' & echo $! > "$CHILD_PID_FILE"; wait`,
	)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("runBoundedCommand error = %v, want deadline exceeded", err)
	}

	pidBytes, err := os.ReadFile(pidFile)
	if err != nil {
		t.Fatalf("read child pid: %v", err)
	}

	pid, err := strconv.Atoi(strings.TrimSpace(string(pidBytes)))
	if err != nil {
		t.Fatalf("parse child pid: %v", err)
	}

	deadline := time.Now().Add(2 * time.Second)
	for processExists(pid) && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}

	if processExists(pid) {
		t.Fatalf("descendant process %d survived command timeout", pid)
	}
}

func TestRunBoundedCommandCapsCombinedOutput(t *testing.T) {
	output, err := runBoundedCommand(
		context.Background(),
		"",
		"sh",
		"-c",
		"yes x | head -c 2097152",
	)
	if err != nil {
		t.Fatalf("runBoundedCommand: %v", err)
	}

	if len(output) > maxCommandOutputBytes+len(outputTruncatedMarker) {
		t.Fatalf("output length = %d, exceeds cap", len(output))
	}

	if !strings.HasSuffix(output, string(outputTruncatedMarker)) {
		t.Fatalf("truncated output missing marker")
	}
}

func TestRunBoundedCommandSetsChildMemoryLimit(t *testing.T) {
	t.Setenv("GOMU_CHILD_GOMEMLIMIT", "768MiB")

	output, err := runBoundedCommand(context.Background(), "", "sh", "-c", `printf %s "$GOMEMLIMIT"`)
	if err != nil {
		t.Fatalf("runBoundedCommand: %v", err)
	}

	if output != "768MiB" {
		t.Fatalf("GOMEMLIMIT = %q, want %q", output, "768MiB")
	}
}

func TestResourceLimitsAreConfigurable(t *testing.T) {
	t.Setenv("GOMU_MAX_WORKERS", "")

	if got := configuredMaxWorkers(); got != defaultMaxWorkers {
		t.Fatalf("default max workers = %d, want %d", got, defaultMaxWorkers)
	}

	t.Setenv("GOMU_MAX_WORKERS", "12")

	if got := configuredMaxWorkers(); got != 12 {
		t.Fatalf("configured max workers = %d, want 12", got)
	}

	t.Setenv("GOMU_CHILD_MAX_RSS_MIB", "768")

	if got := childMaxRSSBytes(); got != 768*1024*1024 {
		t.Fatalf("configured RSS limit = %d, want %d", got, 768*1024*1024)
	}
}

func TestProcessGroupRSS(t *testing.T) {
	group, err := syscall.Getpgid(os.Getpid())
	if err != nil {
		t.Fatalf("get process group: %v", err)
	}

	rss, err := processGroupRSS(group)
	if err != nil {
		t.Fatalf("process group RSS: %v", err)
	}

	if rss <= 0 {
		t.Fatalf("process group RSS = %d, want positive", rss)
	}
}

func processExists(pid int) bool {
	err := syscall.Kill(pid, 0)

	return err == nil || errors.Is(err, syscall.EPERM)
}
