package session

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/srnoob2570/better-rpt/internal/domain"
	"github.com/srnoob2570/better-rpt/internal/fsutil"
)

const (
	maxSessionFile    = 1 << 20
	maxRetainedStates = 20
)

// Store owns the private session directory. Plans and states are keyed by the
// session token and written atomically with 0600 permissions.
type Store struct {
	Dir string
}

// NewStore returns the store rooted at <dataDir>/sessions.
func NewStore(dataDir string) Store {
	return Store{Dir: filepath.Join(dataDir, "sessions")}
}

// Ensure creates and verifies the private session directory.
func (s Store) Ensure() error {
	if strings.TrimSpace(s.Dir) == "" {
		return fmt.Errorf("%w: empty session directory", ErrStore)
	}
	if err := fsutil.EnsurePrivateDir(s.Dir); err != nil {
		return fmt.Errorf("%w: %v", ErrStore, err)
	}
	return nil
}

// PlanPath returns the plan path for a token.
func (s Store) PlanPath(token string) string {
	return filepath.Join(s.Dir, "plan-"+token+".json")
}

// StatePath returns the state path for a token.
func (s Store) StatePath(token string) string {
	return filepath.Join(s.Dir, "state-"+token+".json")
}

// IntentPath returns the single armed launch-intent path.
func (s Store) IntentPath() string {
	return filepath.Join(s.Dir, "launch-intent.json")
}

// LastLaunchPath returns the durable last-launch path.
func (s Store) LastLaunchPath() string {
	return filepath.Join(s.Dir, "last-launch.json")
}

// LockPath returns the global session lock path.
func (s Store) LockPath() string {
	return filepath.Join(s.Dir, "session.lock")
}

// WritePlan validates and atomically stores a plan.
func (s Store) WritePlan(plan Plan) error {
	if err := plan.Validate(time.Now()); err != nil {
		return err
	}
	if err := s.Ensure(); err != nil {
		return err
	}
	data, err := json.MarshalIndent(plan, "", "  ")
	if err != nil {
		return fmt.Errorf("%w: encode plan: %v", ErrStore, err)
	}
	data = append(data, '\n')
	if err := fsutil.AtomicWriteFile(s.PlanPath(plan.Token), data, 0o600); err != nil {
		return fmt.Errorf("%w: write plan: %v", ErrStore, err)
	}
	return nil
}

// ReadPlan reads and validates a plan for the given token.
func (s Store) ReadPlan(token string, now time.Time) (Plan, error) {
	if !domain.ValidSessionToken(token) {
		return Plan{}, fmt.Errorf("%w: token", ErrPlanInvalid)
	}
	data, err := readPrivateFile(s.PlanPath(token))
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return Plan{}, fmt.Errorf("%w: %s", ErrPlanMissing, token)
		}
		return Plan{}, err
	}
	var plan Plan
	if err := json.Unmarshal(data, &plan); err != nil {
		return Plan{}, fmt.Errorf("%w: decode plan: %v", ErrPlanInvalid, err)
	}
	if err := plan.Validate(now); err != nil {
		return Plan{}, err
	}
	if plan.Token != token {
		return Plan{}, fmt.Errorf("%w: token mismatch", ErrPlanInvalid)
	}
	return plan, nil
}

// WriteState validates and atomically stores a state snapshot.
func (s Store) WriteState(state State) error {
	if err := state.Validate(); err != nil {
		return err
	}
	if err := s.Ensure(); err != nil {
		return err
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("%w: encode state: %v", ErrStore, err)
	}
	data = append(data, '\n')
	if err := fsutil.AtomicWriteFile(s.StatePath(state.Token), data, 0o600); err != nil {
		return fmt.Errorf("%w: write state: %v", ErrStore, err)
	}
	return nil
}

// ReadState reads and validates a state snapshot.
func (s Store) ReadState(token string) (State, error) {
	if !domain.ValidSessionToken(token) {
		return State{}, fmt.Errorf("%w: token", ErrStateInvalid)
	}
	data, err := readPrivateFile(s.StatePath(token))
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return State{}, fmt.Errorf("%w: %s", ErrStateMissing, token)
		}
		return State{}, err
	}
	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		return State{}, fmt.Errorf("%w: decode state: %v", ErrStateInvalid, err)
	}
	if err := state.Validate(); err != nil {
		return State{}, err
	}
	if state.Token != token {
		return State{}, fmt.Errorf("%w: token mismatch", ErrStateInvalid)
	}
	return state, nil
}

// ListStates returns every readable state, newest first.
func (s Store) ListStates() ([]State, error) {
	entries, err := os.ReadDir(s.Dir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("%w: read session directory: %v", ErrStore, err)
	}
	states := make([]State, 0)
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasPrefix(name, "state-") || !strings.HasSuffix(name, ".json") {
			continue
		}
		token := strings.TrimSuffix(strings.TrimPrefix(name, "state-"), ".json")
		state, err := s.ReadState(token)
		if err != nil {
			if errors.Is(err, ErrStateMissing) || errors.Is(err, ErrStateInvalid) {
				continue
			}
			return nil, err
		}
		states = append(states, state)
	}
	sort.SliceStable(states, func(i, j int) bool {
		if states[i].UpdatedAt.Equal(states[j].UpdatedAt) {
			return states[i].Token > states[j].Token
		}
		return states[i].UpdatedAt.After(states[j].UpdatedAt)
	})
	return states, nil
}

// PruneStates removes older terminal states, keeping the newest keep of them.
// Non-terminal states are never pruned. It is called when no helper owns the
// session lock, so removing history cannot race an active session.
func (s Store) PruneStates(keep int) (int, error) {
	if keep < 0 {
		keep = 0
	}
	states, err := s.ListStates()
	if err != nil {
		return 0, err
	}
	removed := 0
	kept := 0
	for _, state := range states {
		if !state.Terminal() {
			continue
		}
		if kept < keep {
			kept++
			continue
		}
		if err := s.RemoveState(state.Token); err != nil {
			return removed, err
		}
		removed++
	}
	return removed, nil
}

// ActiveState returns the newest state that has not reached a terminal phase.
func (s Store) ActiveState() (*State, error) {
	states, err := s.ListStates()
	if err != nil {
		return nil, err
	}
	for index := range states {
		if !states[index].Terminal() && states[index].Phase != "" {
			return &states[index], nil
		}
	}
	return nil, nil
}

// RemoveExpiredPlans removes plan files whose expiry is at or before now.
// It is used during startup recovery; active sessions are handled before this
// call and keep their plan.
func (s Store) RemoveExpiredPlans(now time.Time) ([]string, error) {
	entries, err := os.ReadDir(s.Dir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("%w: read session directory: %v", ErrStore, err)
	}
	removed := make([]string, 0)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasPrefix(name, "plan-") || !strings.HasSuffix(name, ".json") {
			continue
		}
		token := strings.TrimSuffix(strings.TrimPrefix(name, "plan-"), ".json")
		if !domain.ValidSessionToken(token) {
			continue
		}
		if _, err := s.ReadPlan(token, now); err == nil {
			continue
		} else if errors.Is(err, ErrPlanExpired) || errors.Is(err, ErrPlanInvalid) {
			// Expired and unusable plans are removed; neither can arm a launch.
		} else if errors.Is(err, ErrPlanMissing) {
			continue
		} else {
			return nil, err
		}
		if err := s.RemovePlan(token); err != nil {
			return nil, err
		}
		removed = append(removed, token)
	}
	return removed, nil
}

// RemovePlan removes a consumed plan. A missing plan is not an error.
func (s Store) RemovePlan(token string) error {
	if !domain.ValidSessionToken(token) {
		return fmt.Errorf("%w: token", ErrPlanInvalid)
	}
	if err := os.Remove(s.PlanPath(token)); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("%w: remove plan: %v", ErrStore, err)
	}
	return nil
}

// RemoveState removes a state file. It is used only for explicit cleanup.
func (s Store) RemoveState(token string) error {
	if !domain.ValidSessionToken(token) {
		return fmt.Errorf("%w: token", ErrStateInvalid)
	}
	if err := os.Remove(s.StatePath(token)); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("%w: remove state: %v", ErrStore, err)
	}
	return nil
}

// WriteIntent validates and atomically stores the app-owned launch intent.
func (s Store) WriteIntent(intent Intent) error {
	if err := intent.Validate(time.Now()); err != nil {
		return err
	}
	if err := s.Ensure(); err != nil {
		return err
	}
	data, err := json.MarshalIndent(intent, "", "  ")
	if err != nil {
		return fmt.Errorf("%w: encode intent: %v", ErrStore, err)
	}
	data = append(data, '\n')
	if err := fsutil.AtomicWriteFile(s.IntentPath(), data, 0o600); err != nil {
		return fmt.Errorf("%w: write intent: %v", ErrStore, err)
	}
	return nil
}

// ReadIntent reads and validates the launch intent, including its expiry.
func (s Store) ReadIntent(now time.Time) (*Intent, error) {
	data, err := readPrivateFile(s.IntentPath())
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, ErrIntentMissing
		}
		return nil, err
	}
	var intent Intent
	if err := json.Unmarshal(data, &intent); err != nil {
		return nil, fmt.Errorf("%w: decode intent: %v", ErrIntentInvalid, err)
	}
	if err := intent.Validate(now); err != nil {
		return nil, err
	}
	return &intent, nil
}

// ClearIntent removes the launch intent. A missing intent is not an error.
func (s Store) ClearIntent() error {
	if err := os.Remove(s.IntentPath()); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("%w: remove intent: %v", ErrStore, err)
	}
	return nil
}

// WriteLastLaunch validates and atomically stores the last launch record.
func (s Store) WriteLastLaunch(last LastLaunch) error {
	if err := last.Validate(); err != nil {
		return err
	}
	if err := s.Ensure(); err != nil {
		return err
	}
	data, err := json.MarshalIndent(last, "", "  ")
	if err != nil {
		return fmt.Errorf("%w: encode last launch: %v", ErrStore, err)
	}
	data = append(data, '\n')
	if err := fsutil.AtomicWriteFile(s.LastLaunchPath(), data, 0o600); err != nil {
		return fmt.Errorf("%w: write last launch: %v", ErrStore, err)
	}
	return nil
}

// ReadLastLaunch reads the durable last launch. A missing record is reported as
// ErrLastLaunchMissing so callers can show an empty state.
func (s Store) ReadLastLaunch() (*LastLaunch, error) {
	data, err := readPrivateFile(s.LastLaunchPath())
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, ErrLastLaunchMissing
		}
		return nil, err
	}
	var last LastLaunch
	if err := json.Unmarshal(data, &last); err != nil {
		return nil, fmt.Errorf("%w: decode last launch: %v", ErrLastLaunchInvalid, err)
	}
	if err := last.Validate(); err != nil {
		return nil, err
	}
	return &last, nil
}

func readPrivateFile(path string) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("%w: %s is a symlink", ErrStore, path)
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("%w: %s is not a regular file", ErrStore, path)
	}
	if info.Mode().Perm()&0o077 != 0 {
		return nil, fmt.Errorf("%w: %s is not private", ErrStore, path)
	}
	if stat, ok := info.Sys().(*syscall.Stat_t); ok && int(stat.Uid) != os.Geteuid() {
		return nil, fmt.Errorf("%w: %s is owned by another user", ErrStore, path)
	}
	if info.Size() > maxSessionFile {
		return nil, fmt.Errorf("%w: %s is too large", ErrStore, path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("%w: read %s: %v", ErrStore, path, err)
	}
	return data, nil
}
