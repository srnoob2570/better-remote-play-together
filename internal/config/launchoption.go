package config

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/srnoob2570/better-rpt/internal/domain"
)

// BuildLaunchOption returns the exact shell fragment Steam stores for the
// donor. The helper path is single-quoted; its own single quotes are escaped.
func BuildLaunchOption(helperPath string, targetAppID uint32, session string) (string, error) {
	if strings.TrimSpace(helperPath) == "" || !filepath.IsAbs(helperPath) {
		return "", fmt.Errorf("%w: helper path must be absolute", domain.ErrHelperMissing)
	}
	for i := 0; i < len(helperPath); i++ {
		if helperPath[i] < 0x20 || helperPath[i] == 0x7f {
			return "", fmt.Errorf("%w: helper path contains control characters", domain.ErrHelperMissing)
		}
	}
	if targetAppID == 0 || targetAppID == domain.SpaceWarAppID {
		return "", fmt.Errorf("%w: invalid target %d", domain.ErrInvalidSession, targetAppID)
	}
	if !domain.ValidSessionToken(session) {
		return "", fmt.Errorf("%w: invalid session token", domain.ErrInvalidSession)
	}
	return fmt.Sprintf("%s --donor --target %d --session %s # %%command%%",
		shellSingleQuote(helperPath), targetAppID, session), nil
}

func shellSingleQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}

// BuildDispatcherLaunchOption returns the persistent SpaceWar wrapper. Steam
// starts the helper inside the donor context; the helper reads Better RPT's
// launch intent or executes the original game command when there is none.
func BuildDispatcherLaunchOption(helperPath string) (string, error) {
	if strings.TrimSpace(helperPath) == "" || !filepath.IsAbs(helperPath) {
		return "", fmt.Errorf("%w: helper path must be absolute", domain.ErrHelperMissing)
	}
	for i := 0; i < len(helperPath); i++ {
		if helperPath[i] < 0x20 || helperPath[i] == 0x7f {
			return "", fmt.Errorf("%w: helper path contains control characters", domain.ErrHelperMissing)
		}
	}
	return fmt.Sprintf("%s --donor -- %%command%%", shellSingleQuote(helperPath)), nil
}

// IsDispatcherMarker reports whether value is the persistent helper wrapper.
func IsDispatcherMarker(value string) bool {
	const suffix = "--donor -- %command%"
	if !strings.HasSuffix(strings.TrimSpace(value), suffix) {
		return false
	}
	return strings.Contains(value, "better-rpt-helper") && !strings.Contains(value, "--session ")
}
