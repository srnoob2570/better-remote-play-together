package diagnostics

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var glibcPattern = regexp.MustCompile(`(?i)(?:glibc|libc)[^\d]*(\d+\.\d+(?:\.\d+)?)`)

func (e *evaluator) checkEnvironment(ctx context.Context) {
	version := strings.TrimSpace(e.version)
	if version == "" {
		version = "dev"
	}
	e.add(CheckVersion, StatusOK, "diag.system.version.ok", []any{version}, []string{
		fmt.Sprintf("version=%s", version),
	}, "")

	e.add(CheckFormat, StatusOK, "diag.system.format.ok", []any{e.format}, []string{
		fmt.Sprintf("format=%s", e.format),
	}, "")

	osStatus := e.checkOS()
	osKey := "diag.system.os.ok"
	if osStatus == StatusSkip {
		osKey = "diag.system.os.skip"
	} else if osStatus == StatusWarn {
		osKey = "diag.system.os.warn"
	}
	e.add(CheckOS, osStatus, osKey, nil, e.osEvidence, "")

	kernel := e.readText("/proc/sys/kernel/osrelease")
	if kernel == "" {
		e.add(CheckKernel, StatusSkip, "diag.system.kernel.skip", nil, nil, "")
	} else {
		e.add(CheckKernel, StatusOK, "", nil, []string{fmt.Sprintf("kernel=%s", kernel)}, "")
	}

	if e.deps.GOARCH == "amd64" {
		e.add(CheckArch, StatusOK, "", nil, []string{fmt.Sprintf("arch=%s", e.deps.GOARCH)}, "")
	} else {
		e.add(CheckArch, StatusWarn, "diag.system.arch.warn", []any{e.deps.GOARCH}, []string{
			fmt.Sprintf("arch=%s", e.deps.GOARCH),
		}, "diag.system.arch.next")
	}

	glibcStatus := e.checkGlibc(ctx)
	glibcKey := "diag.system.glibc.ok"
	if glibcStatus == StatusSkip {
		glibcKey = "diag.system.glibc.skip"
	} else if glibcStatus == StatusFail {
		glibcKey = "diag.system.glibc.fail"
	}
	e.add(CheckGlibc, glibcStatus, glibcKey, nil, e.glibcEvidence, e.glibcNext, "docs/appimage.md")

	locale := firstNonEmpty(e.env("LC_ALL"), e.env("LC_MESSAGES"), e.env("LANG"))
	if locale == "" {
		e.add(CheckLocale, StatusSkip, "diag.system.locale.skip", nil, nil, "")
	} else if locale == "C" || locale == "POSIX" {
		e.add(CheckLocale, StatusWarn, "diag.system.locale.warn", []any{locale}, []string{"locale=" + locale},
			"diag.system.locale.next")
	} else {
		e.add(CheckLocale, StatusOK, "", nil, []string{"locale=" + locale}, "")
	}

	sessionType := strings.ToLower(e.env("XDG_SESSION_TYPE"))
	wayland := e.env("WAYLAND_DISPLAY")
	display := e.env("DISPLAY")
	evidence := []string{}
	if sessionType != "" {
		evidence = append(evidence, "XDG_SESSION_TYPE="+sessionType)
	}
	if wayland != "" {
		evidence = append(evidence, "WAYLAND_DISPLAY="+wayland)
	}
	if display != "" {
		evidence = append(evidence, "DISPLAY="+display)
	}
	switch {
	case wayland != "" || sessionType == "wayland":
		e.add(CheckSessionEnv, StatusOK, "diag.system.session.wayland", nil, evidence, "")
	case display != "" || sessionType == "x11":
		e.add(CheckSessionEnv, StatusOK, "diag.system.session.x11", nil, evidence, "")
	default:
		e.add(CheckSessionEnv, StatusFail, "diag.system.session.fail", nil, evidence,
			"diag.system.session.next", "docs/appimage.md", "docs/known-issues.md")
	}

	desktop := firstNonEmpty(e.env("XDG_CURRENT_DESKTOP"), e.env("DESKTOP_SESSION"))
	if desktop == "" {
		e.add(CheckDesktop, StatusSkip, "diag.system.desktop.skip", nil, nil, "")
	} else {
		e.add(CheckDesktop, StatusOK, "", nil, []string{"desktop=" + desktop}, "")
	}

	if e.deps.EUID == 0 {
		e.add(CheckRoot, StatusFail, "diag.system.root.fail", nil, []string{"euid=0"},
			"diag.system.root.next", "docs/known-issues.md")
	} else {
		e.add(CheckRoot, StatusOK, "", nil, []string{fmt.Sprintf("euid=%d", e.deps.EUID)}, "")
	}
}

// glibcVersionPattern is used when ldd prints only a bare version.
var glibcVersionPattern = regexp.MustCompile(`\b(\d+\.\d+(?:\.\d+)?)\b`)

func (e *evaluator) checkOS() Status {
	e.osEvidence = nil
	data, err := e.readFile("/etc/os-release")
	if err != nil {
		return StatusSkip
	}
	values := map[string]string{}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		values[key] = strings.Trim(strings.TrimSpace(value), `"'`)
	}
	pretty := values["PRETTY_NAME"]
	if pretty == "" {
		pretty = strings.TrimSpace(values["NAME"] + " " + values["VERSION_ID"])
	}
	if pretty == "" {
		return StatusSkip
	}
	e.osEvidence = []string{
		"os=" + pretty,
		"id=" + values["ID"],
		"version_id=" + values["VERSION_ID"],
	}
	if values["ID"] == "" {
		return StatusWarn
	}
	return StatusOK
}

func (e *evaluator) checkGlibc(ctx context.Context) Status {
	e.glibcEvidence = nil
	e.glibcNext = ""
	var output []byte
	var err error
	if e.deps.Run != nil {
		output, err = e.runCommand(ctx, "ldd", "--version")
	}
	if err != nil || len(output) == 0 {
		if e.deps.Run != nil {
			output, err = e.runCommand(ctx, "getconf", "GNU_LIBC_VERSION")
		}
	}
	if err != nil || len(output) == 0 {
		return StatusSkip
	}
	version := glibcPattern.FindStringSubmatch(string(output))
	if len(version) < 2 {
		version = glibcVersionPattern.FindStringSubmatch(string(output))
	}
	if len(version) < 2 {
		e.glibcEvidence = []string{"salida no reconocida: " + firstLine(string(output))}
		return StatusSkip
	}
	e.glibcEvidence = []string{"glibc=" + version[1]}
	if compareVersion(version[1], "2.35") < 0 {
		e.glibcNext = "diag.system.glibc.next"
		return StatusFail
	}
	return StatusOK
}

func compareVersion(left, right string) int {
	parse := func(value string) []int {
		parts := strings.Split(value, ".")
		out := make([]int, 0, len(parts))
		for _, part := range parts {
			number, _ := strconv.Atoi(part)
			out = append(out, number)
		}
		return out
	}
	a, b := parse(left), parse(right)
	for len(a) < len(b) {
		a = append(a, 0)
	}
	for len(b) < len(a) {
		b = append(b, 0)
	}
	for index := range a {
		if a[index] < b[index] {
			return -1
		}
		if a[index] > b[index] {
			return 1
		}
	}
	return 0
}

func (e *evaluator) readFile(path string) ([]byte, error) {
	if e.deps.ReadFile == nil {
		return nil, fmt.Errorf("file reader unavailable")
	}
	return e.deps.ReadFile(path)
}

func (e *evaluator) readText(path string) string {
	data, err := e.readFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func firstLine(value string) string {
	if index := strings.IndexByte(value, '\n'); index >= 0 {
		value = value[:index]
	}
	return strings.TrimSpace(value)
}
