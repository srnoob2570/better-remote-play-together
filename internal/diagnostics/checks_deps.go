package diagnostics

import (
	"context"
	"strings"
)

func (e *evaluator) checkDependencies(ctx context.Context) {
	if e.deps.GOOS != "" && e.deps.GOOS != "linux" {
		e.add(CheckELF, StatusSkip, "diag.deps.elf.skip", nil, []string{"os=" + e.deps.GOOS}, "")
		return
	}
	if e.executable == "" {
		e.add(CheckELF, StatusSkip, "diag.deps.elf.skip", nil, []string{"executable no disponible"}, "")
		return
	}
	evidence := []string{"executable=" + e.executable}
	neededCount := -1
	if e.deps.ELFNeeded != nil {
		needed, err := e.deps.ELFNeeded(e.executable)
		if err != nil {
			evidence = append(evidence, "DT_NEEDED no legible: "+err.Error())
		} else {
			neededCount = len(needed)
			if len(needed) > 0 {
				evidence = append(evidence, "DT_NEEDED="+strings.Join(needed, ", "))
			} else {
				evidence = append(evidence, "DT_NEEDED=ninguna")
			}
		}
	}
	if neededCount == 0 {
		e.add(CheckELF, StatusOK, "diag.deps.elf.static", nil, evidence, "")
		return
	}
	if e.deps.LDD == nil {
		e.add(CheckELF, StatusSkip, "diag.deps.elf.skip", nil, evidence, "")
		return
	}
	output, err := e.deps.LDD(ctx, e.executable)
	text := string(output)
	if strings.Contains(strings.ToLower(text), "not a dynamic executable") {
		e.add(CheckELF, StatusOK, "diag.deps.elf.static", nil, evidence, "")
		return
	}
	missing := missingLibraries(text)
	if len(missing) > 0 {
		evidence = append(evidence, "missing="+strings.Join(missing, ", "))
		e.add(CheckELF, StatusFail, "diag.deps.elf.missing", []any{len(missing)}, evidence,
			"diag.deps.elf.next", "docs/appimage.md")
		return
	}
	if err != nil {
		evidence = append(evidence, "ldd: "+strings.TrimSpace(firstLine(text)))
		e.add(CheckELF, StatusWarn, "diag.deps.elf.unresolved", nil, evidence,
			"diag.deps.elf.next", "docs/appimage.md")
		return
	}
	e.add(CheckELF, StatusOK, "", nil, evidence, "")
}

func missingLibraries(output string) []string {
	missing := make([]string, 0)
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if !strings.Contains(line, "=>") || !strings.Contains(line, "not found") {
			continue
		}
		name, _, _ := strings.Cut(line, "=>")
		name = strings.TrimSpace(name)
		if name != "" {
			missing = append(missing, name)
		}
	}
	return missing
}
