package diagnostics

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func (e *evaluator) checkAppImage(ctx context.Context) {
	if e.format != "appimage" {
		e.add(CheckFUSE, StatusSkip, "diag.appimage.fuse.skip", nil, []string{"format=" + e.format}, "")
		e.add(CheckMount, StatusSkip, "diag.appimage.mount.notapp", nil, []string{"format=" + e.format}, "")
		e.add(CheckMAC, StatusSkip, "diag.appimage.mac.notapp", nil, []string{"format=" + e.format}, "")
		return
	}

	if e.env("APPIMAGE_EXTRACT_AND_RUN") != "" {
		e.add(CheckFUSE, StatusSkip, "diag.appimage.fuse.extract", nil,
			[]string{"APPIMAGE_EXTRACT_AND_RUN=" + e.env("APPIMAGE_EXTRACT_AND_RUN")}, "")
	} else {
		fusermount := ""
		if e.deps.LookPath != nil {
			if path, err := e.deps.LookPath("fusermount3"); err == nil {
				fusermount = path
			}
		}
		fuseDevice := false
		if e.deps.Stat != nil {
			if info, err := e.deps.Stat("/dev/fuse"); err == nil {
				fuseDevice = info.Mode()&os.ModeCharDevice != 0
			}
		}
		evidence := []string{
			fmt.Sprintf("fusermount3=%s", firstNonEmpty(fusermount, "no encontrado")),
			fmt.Sprintf("/dev/fuse=%t", fuseDevice),
		}
		switch {
		case fusermount != "" && fuseDevice:
			e.add(CheckFUSE, StatusOK, "", nil, evidence, "")
		case fusermount == "" && !fuseDevice:
			e.add(CheckFUSE, StatusFail, "diag.appimage.fuse.missing", nil, evidence,
				"diag.appimage.fuse.next", "docs/appimage.md")
		case fusermount == "":
			e.add(CheckFUSE, StatusFail, "diag.appimage.fuse.nofusermount", nil, evidence,
				"diag.appimage.fuse.next", "docs/appimage.md")
		default:
			e.add(CheckFUSE, StatusFail, "diag.appimage.fuse.nodevice", nil, evidence,
				"diag.appimage.fuse.next", "docs/appimage.md")
		}
	}

	mountStatus := e.checkAppImageMount()
	mountKey := "diag.appimage.mount.ok"
	if mountStatus == StatusFail {
		mountKey = "diag.appimage.mount.fail"
	} else if mountStatus == StatusSkip {
		mountKey = "diag.appimage.mount.skip"
	}
	e.add(CheckMount, mountStatus, mountKey, nil, e.mountEvidence, e.mountNext,
		"docs/appimage.md")

	macStatus := e.checkMAC(ctx)
	macKey := "diag.appimage.mac.ok"
	switch macStatus {
	case StatusWarn:
		macKey = "diag.appimage.mac.warn"
	case StatusFail:
		macKey = "diag.appimage.mac.fail"
	case StatusSkip:
		macKey = "diag.appimage.mac.skip"
	}
	e.add(CheckMAC, macStatus, macKey, nil, e.macEvidence, e.macNext,
		"docs/appimage.md", "docs/known-issues.md")
}

func (e *evaluator) checkAppImageMount() Status {
	e.mountEvidence = nil
	e.mountNext = ""
	if e.executable == "" {
		return StatusSkip
	}
	data, err := e.readFile(e.mountInfoPath())
	if err != nil {
		return StatusSkip
	}
	mount, options, ok := mountFor(string(data), e.executable)
	if !ok {
		e.mountEvidence = []string{"executable=" + e.executable}
		return StatusSkip
	}
	e.mountEvidence = []string{"mount=" + mount, "options=" + options}
	// The AppImage runtime mounts the payload read-only with nosuid,nodev.
	// Those flags do not prevent execution; only noexec does.
	noexec := false
	for _, part := range strings.Split(options, ",") {
		if part == "noexec" {
			noexec = true
			break
		}
	}
	if noexec {
		e.mountNext = "diag.appimage.mount.next"
		return StatusFail
	}
	return StatusOK
}

func (e *evaluator) checkMAC(ctx context.Context) Status {
	e.macEvidence = nil
	e.macNext = ""
	selinux, selinuxErr := e.readFile("/sys/fs/selinux/enforce")
	if selinuxErr == nil {
		enforcing := strings.TrimSpace(string(selinux)) == "1"
		contextText := e.readText("/proc/self/attr/current")
		e.macEvidence = append(e.macEvidence, "selinux_enforcing="+fmt.Sprint(enforcing))
		if contextText != "" {
			e.macEvidence = append(e.macEvidence, "selinux_context="+contextText)
		}
		if !enforcing {
			return StatusOK
		}
		if e.deps.Run == nil {
			e.macNext = "diag.appimage.mac.next"
			return StatusSkip
		}
		output, err := e.deps.Run(ctx, "ausearch", "-m", "avc", "-ts", "recent")
		lower := strings.ToLower(string(output))
		if strings.Contains(lower, "no matches") {
			return StatusOK
		}
		if err != nil || len(output) == 0 {
			e.macNext = "diag.appimage.mac.next"
			return StatusSkip
		}
		if strings.Contains(lower, "better-rpt") || strings.Contains(lower, "appimage") {
			e.macNext = "diag.appimage.mac.next"
			return StatusFail
		}
		return StatusOK
	}

	current := e.readText("/proc/self/attr/current")
	if current == "" {
		return StatusSkip
	}
	e.macEvidence = []string{"apparmor_profile=" + current}
	if strings.Contains(strings.ToLower(current), "unconfined") {
		return StatusOK
	}
	if enabled := e.readText("/sys/module/apparmor/parameters/enabled"); strings.HasPrefix(strings.ToUpper(enabled), "Y") {
		e.macEvidence = append(e.macEvidence, "apparmor_enabled=true")
		if e.deps.Run == nil {
			e.macNext = "diag.appimage.mac.next"
			return StatusSkip
		}
		output, err := e.deps.Run(ctx, "dmesg")
		if err != nil || len(output) == 0 {
			e.macNext = "diag.appimage.mac.next"
			return StatusSkip
		}
		lower := strings.ToLower(string(output))
		if strings.Contains(lower, "apparmor=\"denied\"") &&
			(strings.Contains(lower, "better-rpt") || strings.Contains(lower, "appimage")) {
			e.macNext = "diag.appimage.mac.next"
			return StatusFail
		}
		return StatusOK
	}
	return StatusSkip
}

func (e *evaluator) mountInfoPath() string {
	if e.deps.MountInfo != "" {
		return e.deps.MountInfo
	}
	return "/proc/self/mountinfo"
}

func mountFor(data, executable string) (string, string, bool) {
	executable = filepath.Clean(executable)
	best := ""
	options := ""
	for _, line := range strings.Split(data, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 6 {
			continue
		}
		mount := decodeMount(fields[4])
		if mount == "" {
			continue
		}
		if executable == mount || strings.HasPrefix(executable, strings.TrimSuffix(mount, "/")+"/") {
			if len(mount) > len(best) {
				best = mount
				options = fields[5]
			}
		}
	}
	if best == "" {
		return "", "", false
	}
	return best, options, true
}

func decodeMount(value string) string {
	replacer := strings.NewReplacer(
		`\040`, " ",
		`\011`, "\t",
		`\012`, "\n",
		`\134`, `\`,
	)
	return replacer.Replace(value)
}
