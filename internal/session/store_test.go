package session

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/srnoob2570/better-rpt/internal/domain"
)

func testState(token string, phase domain.Phase, now time.Time) State {
	state := State{
		Version:     StateVersion,
		Token:       token,
		DonorAppID:  domain.SpaceWarAppID,
		TargetAppID: 900001,
		Phase:       phase,
		CreatedAt:   now.UTC(),
		UpdatedAt:   now.UTC(),
	}
	if phase == domain.PhaseActive {
		state.TargetPID = 12345
	}
	return state
}

func TestStorePlanStateRoundTrip(t *testing.T) {
	store := sessionStore(t)
	now := time.Now().UTC()
	plan := writeTestPlan(t, store, 900001, now)

	readPlan, err := store.ReadPlan(testToken, now)
	if err != nil {
		t.Fatalf("read plan: %v", err)
	}
	if readPlan.Token != plan.Token || readPlan.TargetAppID != 900001 || readPlan.DonorAppID != 480 {
		t.Fatalf("plan = %+v", readPlan)
	}
	info, err := os.Stat(store.PlanPath(testToken))
	if err != nil {
		t.Fatalf("stat plan: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("plan mode = %o, want 600", info.Mode().Perm())
	}

	state := NewState(plan, domain.PhasePreparing, now)
	state.HelperPID = os.Getpid()
	state.HelperStartTime = helperStart(t)
	if err := store.WriteState(state); err != nil {
		t.Fatalf("write state: %v", err)
	}
	readState, err := store.ReadState(testToken)
	if err != nil {
		t.Fatalf("read state: %v", err)
	}
	if readState.Phase != domain.PhasePreparing || readState.HelperPID != os.Getpid() {
		t.Fatalf("state = %+v", readState)
	}
	info, err = os.Stat(store.StatePath(testToken))
	if err != nil {
		t.Fatalf("stat state: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("state mode = %o, want 600", info.Mode().Perm())
	}

	states, err := store.ListStates()
	if err != nil || len(states) != 1 {
		t.Fatalf("list states = %v, %v", states, err)
	}
	active, err := store.ActiveState()
	if err != nil || active == nil || active.Token != testToken {
		t.Fatalf("active state = %+v, %v", active, err)
	}
	if err := store.RemovePlan(testToken); err != nil {
		t.Fatalf("remove plan: %v", err)
	}
	if err := store.RemoveState(testToken); err != nil {
		t.Fatalf("remove state: %v", err)
	}
	if _, err := store.ReadPlan(testToken, now); !errors.Is(err, ErrPlanMissing) {
		t.Fatalf("missing plan error = %v", err)
	}
	if _, err := store.ReadState(testToken); !errors.Is(err, ErrStateMissing) {
		t.Fatalf("missing state error = %v", err)
	}
	if err := store.RemovePlan(testToken); err != nil {
		t.Fatalf("remove missing plan: %v", err)
	}
	if err := store.RemoveState(testToken); err != nil {
		t.Fatalf("remove missing state: %v", err)
	}
}

func TestStoreRejectsExpiredPlan(t *testing.T) {
	store := sessionStore(t)
	plan := writeTestPlan(t, store, 900001, time.Now().UTC())
	if _, err := store.ReadPlan(testToken, plan.ExpiresAt); !errors.Is(err, ErrPlanExpired) {
		t.Fatalf("error = %v, want ErrPlanExpired", err)
	}
}

func TestStoreRejectsCorruptAndInsecureFiles(t *testing.T) {
	store := sessionStore(t)
	if err := os.WriteFile(store.StatePath(testToken), []byte("{"), 0o600); err != nil {
		t.Fatalf("write corrupt state: %v", err)
	}
	if _, err := store.ReadState(testToken); !errors.Is(err, ErrStateInvalid) {
		t.Fatalf("corrupt state error = %v, want ErrStateInvalid", err)
	}

	plan := writeTestPlan(t, store, 900001, time.Now().UTC())
	if err := os.Chmod(store.PlanPath(testToken), 0o644); err != nil {
		t.Fatalf("chmod plan: %v", err)
	}
	if _, err := store.ReadPlan(testToken, plan.CreatedAt); !errors.Is(err, ErrStore) {
		t.Fatalf("insecure plan error = %v, want ErrStore", err)
	}
}

func TestStoreRejectsSymlinkPlan(t *testing.T) {
	store := sessionStore(t)
	target := store.PlanPath(testToken)
	if err := os.Remove(target); err != nil && !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("remove plan: %v", err)
	}
	real := filepath.Join(t.TempDir(), "real.json")
	if err := os.WriteFile(real, []byte("{}"), 0o600); err != nil {
		t.Fatalf("write real: %v", err)
	}
	if err := os.Symlink(real, target); err != nil {
		t.Fatalf("symlink: %v", err)
	}
	if _, err := store.ReadPlan(testToken, time.Now()); !errors.Is(err, ErrStore) {
		t.Fatalf("symlink error = %v, want ErrStore", err)
	}
}

func TestStoreListAndActiveState(t *testing.T) {
	store := sessionStore(t)
	now := time.Now().UTC()
	first := testState(fmt.Sprintf("%032x", 1), domain.PhaseEnded, now)
	second := testState(fmt.Sprintf("%032x", 2), domain.PhaseActive, now.Add(time.Minute))
	third := testState(fmt.Sprintf("%032x", 3), domain.PhasePreparing, now.Add(2*time.Minute))
	for _, state := range []State{first, second, third} {
		if err := store.WriteState(state); err != nil {
			t.Fatalf("write %s: %v", state.Token, err)
		}
	}
	states, err := store.ListStates()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(states) != 3 || states[0].Token != third.Token || states[2].Token != first.Token {
		t.Fatalf("states order = %+v", states)
	}
	active, err := store.ActiveState()
	if err != nil || active == nil || active.Token != third.Token {
		t.Fatalf("active = %+v, %v", active, err)
	}
}
