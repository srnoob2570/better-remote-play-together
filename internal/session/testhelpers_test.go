package session

import (
	"os"
	"testing"
	"time"

	"github.com/srnoob2570/better-rpt/internal/domain"
)

const testToken = "0123456789abcdef0123456789abcdef"

func testSpec(target uint32) domain.LaunchSpec {
	return domain.LaunchSpec{
		AppID:      target,
		Kind:       domain.KindNative,
		Executable: "/bin/true",
		WorkingDir: "/tmp",
	}
}

func sessionStore(t *testing.T) Store {
	t.Helper()
	store := NewStore(t.TempDir())
	if err := store.Ensure(); err != nil {
		t.Fatalf("ensure store: %v", err)
	}
	return store
}

func writeTestPlan(t *testing.T, store Store, target uint32, now time.Time) Plan {
	t.Helper()
	plan := NewPlan(testToken, target, testSpec(target), now)
	if err := store.WritePlan(plan); err != nil {
		t.Fatalf("write plan: %v", err)
	}
	return plan
}

func helperStart(t *testing.T) uint64 {
	t.Helper()
	start, err := ProcessStartTime(os.Getpid())
	if err != nil {
		t.Fatalf("self start time: %v", err)
	}
	return start
}
