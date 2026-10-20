package diagnostics

import (
	"context"
	"debug/elf"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"syscall"
	"time"

	"github.com/srnoob2570/better-rpt/internal/config"
	"github.com/srnoob2570/better-rpt/internal/helper"
	"github.com/srnoob2570/better-rpt/internal/logging"
	"github.com/srnoob2570/better-rpt/internal/session"
	"github.com/srnoob2570/better-rpt/internal/steam"
)

// Deps are the injectable environment, filesystem and process operations used
// by the checks. The zero value is not usable; call DefaultDeps or fill the
// fields in tests.
type Deps struct {
	Now             func() time.Time
	Getenv          func(string) string
	ReadFile        func(string) ([]byte, error)
	ReadDir         func(string) ([]os.DirEntry, error)
	Stat            func(string) (os.FileInfo, error)
	Lstat           func(string) (os.FileInfo, error)
	Access          func(string, uint32) error
	CanWrite        func(string, bool) error
	LookPath        func(string) (string, error)
	Run             func(context.Context, string, ...string) ([]byte, error)
	Executable      func() (string, error)
	EUID            int
	Home            string
	Username        string
	GOOS            string
	GOARCH          string
	ProcRoot        string
	MountInfo       string
	Paths           config.Paths
	PathsErr        error
	LoadConfig      func(config.Paths) (config.Config, error)
	DetectSteam     func(context.Context) (steam.Installation, error)
	LoadCatalog     func(context.Context, steam.Installation) (steam.Catalog, error)
	ScanSteam       func(string) ([]steam.Process, error)
	SelectAccount   func(string) (config.SteamAccount, error)
	DetectSetup     func(config.Paths, string) (config.SetupSnapshot, error)
	ListBackups     func(config.Paths) ([]config.Backup, error)
	InspectOverlay  func(string) (config.OverlayInspection, error)
	InspectControls func(string) (config.ControllerLayoutInspection, error)
	InspectHelper   func(config.Paths) (helper.Info, error)
	ReadLastLaunch  func(config.Paths) (*session.LastLaunch, error)
	ActiveSession   func(config.Paths) (*session.State, error)
	ListStates      func(config.Paths) ([]session.State, error)
	ProcessAlive    func(int, uint64) bool
	PIDAlive        func(int) bool
	ReadLogTail     func(config.Paths, int) ([]string, error)
	LogPath         func(config.Paths) string
	ELFNeeded       func(string) ([]string, error)
	LDD             func(context.Context, string) ([]byte, error)
}

// DefaultDeps returns production dependencies. pathsErr is retained so a path
// resolution failure becomes a report check instead of aborting diagnostics.
func DefaultDeps(paths config.Paths, pathsErr error) *Deps {
	home, _ := os.UserHomeDir()
	return &Deps{
		Now:             time.Now,
		Getenv:          os.Getenv,
		ReadFile:        os.ReadFile,
		ReadDir:         os.ReadDir,
		Stat:            os.Stat,
		Lstat:           os.Lstat,
		Access:          func(path string, mode uint32) error { return syscall.Access(path, mode) },
		CanWrite:        canWrite,
		LookPath:        exec.LookPath,
		Run:             runCommand,
		Executable:      os.Executable,
		EUID:            os.Geteuid(),
		Home:            home,
		Username:        os.Getenv("USER"),
		GOOS:            runtime.GOOS,
		GOARCH:          runtime.GOARCH,
		ProcRoot:        "/proc",
		MountInfo:       "/proc/self/mountinfo",
		Paths:           paths,
		PathsErr:        pathsErr,
		LoadConfig:      config.Load,
		DetectSteam:     steam.Detect,
		LoadCatalog:     steam.Load,
		ScanSteam:       steam.ScanProcesses,
		SelectAccount:   config.SelectAccount,
		DetectSetup:     config.DetectSetup,
		ListBackups:     config.ListBackups,
		InspectOverlay:  config.InspectOverlay,
		InspectControls: config.InspectControllerLayouts,
		InspectHelper:   helper.Inspect,
		ReadLastLaunch: func(paths config.Paths) (*session.LastLaunch, error) {
			return session.NewStore(paths.DataDir).ReadLastLaunch()
		},
		ActiveSession: func(paths config.Paths) (*session.State, error) {
			return session.NewStore(paths.DataDir).ActiveState()
		},
		ListStates: func(paths config.Paths) ([]session.State, error) {
			return session.NewStore(paths.DataDir).ListStates()
		},
		ProcessAlive: session.ProcessAlive,
		PIDAlive:     pidAlive,
		ReadLogTail:  logging.Tail,
		LogPath:      logging.Path,
		ELFNeeded:    elfNeeded,
		LDD:          ldd,
	}
}

func runCommand(ctx context.Context, name string, args ...string) ([]byte, error) {
	command := exec.CommandContext(ctx, name, args...)
	command.Stdin = nil
	return command.CombinedOutput()
}

func canWrite(path string, directory bool) error {
	const wOK uint32 = 2
	const xOK uint32 = 1
	mode := wOK
	if directory {
		mode |= xOK
	}
	return syscall.Access(path, mode)
}

func pidAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	_, err := os.Stat(filepath.Join("/proc", strconv.Itoa(pid)))
	return err == nil
}

func elfNeeded(path string) ([]string, error) {
	file, err := elf.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	return file.ImportedLibraries()
}

func ldd(ctx context.Context, path string) ([]byte, error) {
	command := exec.CommandContext(ctx, "ldd", path)
	command.Stdin = nil
	return command.CombinedOutput()
}
