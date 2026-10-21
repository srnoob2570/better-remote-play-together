package update

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type installInfo struct {
	Kind Kind
	Path string
}

type environment struct {
	getenv       func(string) string
	executable   func() (string, error)
	stat         func(string) (os.FileInfo, error)
	evalSymlinks func(string) (string, error)
	lookPath     func(string) (string, error)
	output       func(context.Context, string, ...string) ([]byte, error)
}

func defaultEnvironment() environment {
	return environment{
		getenv:       os.Getenv,
		executable:   os.Executable,
		stat:         os.Stat,
		evalSymlinks: filepath.EvalSymlinks,
		lookPath:     exec.LookPath,
		output: func(ctx context.Context, name string, args ...string) ([]byte, error) {
			ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
			defer cancel()
			return exec.CommandContext(ctx, name, args...).Output()
		},
	}
}

func detectInstall(env environment) installInfo {
	executable, _ := env.executable()
	if appImage := strings.TrimSpace(env.getenv("APPIMAGE")); appImage != "" {
		if target, ok := appImageTarget(env, executable, appImage); ok {
			return installInfo{Kind: KindAppImage, Path: target}
		}
	}
	if filepath.Clean(executable) == "/usr/bin/better-rpt" {
		if ownedByPackage(env, executable) {
			return installInfo{Kind: KindDeb, Path: executable}
		}
	}
	return installInfo{Kind: KindDevelopment, Path: executable}
}

func appImageTarget(env environment, executable, appImage string) (string, bool) {
	target, err := filepath.Abs(appImage)
	if err != nil {
		return "", false
	}
	if resolved, err := env.evalSymlinks(target); err == nil {
		target = resolved
	}
	info, err := env.stat(target)
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0o111 == 0 {
		return "", false
	}
	if mount := strings.TrimSpace(env.getenv("APPDIR")); mount != "" {
		mountPath, err := filepath.Abs(mount)
		if err != nil {
			return "", false
		}
		if resolved, err := env.evalSymlinks(mountPath); err == nil {
			mountPath = resolved
		}
		if executable == "" {
			return "", false
		}
		exePath, err := filepath.Abs(executable)
		if err != nil {
			return "", false
		}
		if resolved, err := env.evalSymlinks(exePath); err == nil {
			exePath = resolved
		}
		if !strings.HasPrefix(exePath, filepath.Clean(mountPath)+string(filepath.Separator)) {
			return "", false
		}
	}
	return target, true
}

func ownedByPackage(env environment, executable string) bool {
	dpkg, err := env.lookPath("dpkg")
	if err != nil {
		return false
	}
	output, err := env.output(context.Background(), dpkg, "-S", filepath.Clean(executable))
	if err != nil {
		return false
	}
	for _, line := range strings.Split(string(output), "\n") {
		packageName, _, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		packageName = strings.TrimSpace(packageName)
		if packageName == "better-rpt" || strings.HasPrefix(packageName, "better-rpt:") {
			return true
		}
	}
	return false
}
