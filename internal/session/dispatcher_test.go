package session

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"reflect"
	"testing"
	"time"

	"github.com/srnoob2570/better-rpt/internal/domain"
)

func dispatcherOptions(t *testing.T, store *Store) DispatcherOptions {
	t.Helper()
	return DispatcherOptions{
		HelperOptions: helperOptions(store, 0),
	}
}

func TestDispatcherWithoutIntentRunsOriginalGame(t *testing.T) {
	store := sessionStore(t)
	options := dispatcherOptions(t, &store)
	var gotArgs, gotEnv []string
	options.Fallback = []string{"/game/spacewar", "--windowed"}
	options.RunFallback = func(_ context.Context, args, env []string) error {
		gotArgs = append([]string(nil), args...)
		gotEnv = append([]string(nil), env...)
		return nil
	}
	dispatcher, err := NewDispatcher(options)
	if err != nil {
		t.Fatalf("new dispatcher: %v", err)
	}
	if _, err := dispatcher.Run(context.Background()); err != nil {
		t.Fatalf("run: %v", err)
	}
	if !reflect.DeepEqual(gotArgs, options.Fallback) {
		t.Fatalf("fallback args = %v, want %v", gotArgs, options.Fallback)
	}
	if len(gotEnv) == 0 {
		t.Fatal("fallback did not receive the donor environment")
	}
	if _, err := store.ReadState(testToken); !errors.Is(err, ErrStateMissing) {
		t.Fatalf("fallback wrote session state: %v", err)
	}
}

func TestDispatcherConsumesArmedIntentAndLaunchesTarget(t *testing.T) {
	store := sessionStore(t)
	now := time.Now().UTC()
	writeTestPlan(t, store, 900001, now)
	if err := store.WriteIntent(NewIntent(testToken, 900001, now)); err != nil {
		t.Fatalf("write intent: %v", err)
	}
	options := dispatcherOptions(t, &store)
	options.Fallback = []string{"/game/spacewar"}
	options.RunFallback = func(context.Context, []string, []string) error {
		t.Fatal("armed intent must not run the fallback game")
		return nil
	}
	options.BuildCommand = commandBuilder(exec.Command("sleep", "0.08"))
	dispatcher, err := NewDispatcher(options)
	if err != nil {
		t.Fatalf("new dispatcher: %v", err)
	}
	state, err := dispatcher.Run(context.Background())
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if state.Phase != domain.PhaseEnded || state.TargetAppID != 900001 {
		t.Fatalf("state = %+v", state)
	}
	intent, err := store.ReadIntent(time.Now())
	if err != nil {
		t.Fatalf("read intent: %v", err)
	}
	if intent.State != IntentConsumed {
		t.Fatalf("intent = %+v, want consumed", intent)
	}
}

func TestDispatcherExpiredIntentRunsOriginalGame(t *testing.T) {
	store := sessionStore(t)
	now := time.Now().UTC()
	writeTestPlan(t, store, 900001, now)
	intent := NewIntent(testToken, 900001, now.Add(-2*PlanTTL))
	data, err := json.Marshal(intent)
	if err != nil {
		t.Fatalf("marshal intent: %v", err)
	}
	if err := os.WriteFile(store.IntentPath(), data, 0o600); err != nil {
		t.Fatalf("write expired intent: %v", err)
	}
	options := dispatcherOptions(t, &store)
	options.Fallback = []string{"/game/spacewar"}
	fallback := false
	options.RunFallback = func(context.Context, []string, []string) error {
		fallback = true
		return nil
	}
	dispatcher, err := NewDispatcher(options)
	if err != nil {
		t.Fatalf("new dispatcher: %v", err)
	}
	if _, err := dispatcher.Run(context.Background()); err != nil {
		t.Fatalf("run: %v", err)
	}
	if !fallback {
		t.Fatal("expired intent did not run the fallback game")
	}
	if _, err := store.ReadIntent(time.Now()); !errors.Is(err, ErrIntentMissing) {
		t.Fatalf("expired intent was not cleared: %v", err)
	}
}

func TestDispatcherCorruptIntentRunsOriginalGame(t *testing.T) {
	store := sessionStore(t)
	if err := os.WriteFile(store.IntentPath(), []byte("{"), 0o600); err != nil {
		t.Fatalf("write corrupt intent: %v", err)
	}
	options := dispatcherOptions(t, &store)
	options.Fallback = []string{"/game/spacewar"}
	fallback := false
	options.RunFallback = func(context.Context, []string, []string) error {
		fallback = true
		return nil
	}
	dispatcher, err := NewDispatcher(options)
	if err != nil {
		t.Fatalf("new dispatcher: %v", err)
	}
	if _, err := dispatcher.Run(context.Background()); err != nil {
		t.Fatalf("run: %v", err)
	}
	if !fallback {
		t.Fatal("corrupt intent did not run the fallback game")
	}
	if _, err := store.ReadIntent(time.Now()); !errors.Is(err, ErrIntentMissing) {
		t.Fatalf("corrupt intent was not cleared: %v", err)
	}
}

func TestDispatcherWithoutIntentOrCommandFails(t *testing.T) {
	store := sessionStore(t)
	dispatcher, err := NewDispatcher(dispatcherOptions(t, &store))
	if err != nil {
		t.Fatalf("new dispatcher: %v", err)
	}
	if _, err := dispatcher.Run(context.Background()); !errors.Is(err, ErrIntentMissing) {
		t.Fatalf("error = %v, want ErrIntentMissing", err)
	}
}

func TestDispatcherRejectsIntentPlanMismatch(t *testing.T) {
	store := sessionStore(t)
	now := time.Now().UTC()
	writeTestPlan(t, store, 900001, now)
	if err := store.WriteIntent(NewIntent(testToken, 900002, now)); err != nil {
		t.Fatalf("write intent: %v", err)
	}
	dispatcher, err := NewDispatcher(dispatcherOptions(t, &store))
	if err != nil {
		t.Fatalf("new dispatcher: %v", err)
	}
	if _, err := dispatcher.Run(context.Background()); !errors.Is(err, ErrPlanInvalid) {
		t.Fatalf("error = %v, want ErrPlanInvalid", err)
	}
	if _, err := store.ReadIntent(time.Now()); !errors.Is(err, ErrIntentMissing) {
		t.Fatalf("mismatched intent was not cleared: %v", err)
	}
}
