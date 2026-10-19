package launch

import "errors"

var (
	// ErrNotGame means appinfo's common.type is not Game; tools/runtimes are
	// excluded from the catalog.
	ErrNotGame = errors.New("app is not a game")
	// ErrInfoState means the appinfo record is not in the normal state.
	ErrInfoState = errors.New("appinfo state is not normal")
	// ErrNoLaunchConfig means appinfo has no config.launch object.
	ErrNoLaunchConfig = errors.New("appinfo has no launch configuration")
	// ErrNoLaunchForOS means no launch entry targets the requested OS.
	ErrNoLaunchForOS = errors.New("appinfo has no launch entry for this OS")
	// ErrLaunchTemplate means the only matching entry uses %command%, so it is a
	// Steam launch-option template rather than a runnable command.
	ErrLaunchTemplate = errors.New("launch entry uses %command%")
	// ErrExecutableMissing means the declared executable is not present.
	ErrExecutableMissing = errors.New("launch executable is missing")
	// ErrExecutableInvalid means the executable exists but is not a regular
	// executable file, or its metadata cannot be read.
	ErrExecutableInvalid = errors.New("launch executable is invalid")
	// ErrAmbiguousExecutable means a case-insensitive lookup found more than one
	// candidate.
	ErrAmbiguousExecutable = errors.New("launch executable is ambiguous")
	// ErrArguments means the launch arguments are not lexically valid.
	ErrArguments = errors.New("launch arguments are malformed")
	// ErrWorkingDir means the resolved working directory is missing or invalid.
	ErrWorkingDir = errors.New("launch working directory is invalid")
	// ErrCommand means the resolved launch spec cannot form a safe command.
	ErrCommand = errors.New("invalid launch command")
	// ErrCompatDataUnwritable means Proton's compatdata prefix cannot be created
	// or written by this user.
	ErrCompatDataUnwritable = errors.New("Steam compat data path is not writable")
)
