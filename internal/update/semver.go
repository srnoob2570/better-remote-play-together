package update

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var (
	versionPattern = regexp.MustCompile(
		`^[vV]?(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)` +
			`(?:-([0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*))?` +
			`(?:\+[0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*)?$`,
	)
	numericPattern = regexp.MustCompile(`^[0-9]+$`)
)

// Version is a parsed semantic version. Build metadata is ignored.
type Version struct {
	Major      int
	Minor      int
	Patch      int
	Prerelease []string
}

// ParseVersion parses a semver string, accepting an optional leading v.
func ParseVersion(value string) (Version, error) {
	match := versionPattern.FindStringSubmatch(strings.TrimSpace(value))
	if match == nil {
		return Version{}, fmt.Errorf("invalid semantic version %q", value)
	}
	major, err := strconv.Atoi(match[1])
	if err != nil {
		return Version{}, fmt.Errorf("invalid major version %q: %w", value, err)
	}
	minor, err := strconv.Atoi(match[2])
	if err != nil {
		return Version{}, fmt.Errorf("invalid minor version %q: %w", value, err)
	}
	patch, err := strconv.Atoi(match[3])
	if err != nil {
		return Version{}, fmt.Errorf("invalid patch version %q: %w", value, err)
	}
	version := Version{Major: major, Minor: minor, Patch: patch}
	if match[4] != "" {
		version.Prerelease = strings.Split(match[4], ".")
		for _, identifier := range version.Prerelease {
			if numericPattern.MatchString(identifier) && len(identifier) > 1 && identifier[0] == '0' {
				return Version{}, fmt.Errorf("invalid prerelease identifier %q", identifier)
			}
		}
	}
	return version, nil
}

// String renders the version without a leading v or build metadata.
func (v Version) String() string {
	value := fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
	if len(v.Prerelease) > 0 {
		value += "-" + strings.Join(v.Prerelease, ".")
	}
	return value
}

// Compare returns -1 when v is older than other, 0 when both are equal and 1
// when v is newer. Precedence follows semver 2.0.0.
func (v Version) Compare(other Version) int {
	if v.Major != other.Major {
		return compareInt(v.Major, other.Major)
	}
	if v.Minor != other.Minor {
		return compareInt(v.Minor, other.Minor)
	}
	if v.Patch != other.Patch {
		return compareInt(v.Patch, other.Patch)
	}
	return comparePrerelease(v.Prerelease, other.Prerelease)
}

func compareInt(left, right int) int {
	switch {
	case left < right:
		return -1
	case left > right:
		return 1
	default:
		return 0
	}
}

func comparePrerelease(left, right []string) int {
	switch {
	case len(left) == 0 && len(right) == 0:
		return 0
	case len(left) == 0:
		return 1
	case len(right) == 0:
		return -1
	}
	for index := 0; index < len(left) && index < len(right); index++ {
		if result := compareIdentifier(left[index], right[index]); result != 0 {
			return result
		}
	}
	return compareInt(len(left), len(right))
}

func compareIdentifier(left, right string) int {
	leftNumeric := numericPattern.MatchString(left)
	rightNumeric := numericPattern.MatchString(right)
	switch {
	case leftNumeric && rightNumeric:
		if len(left) != len(right) {
			return compareInt(len(left), len(right))
		}
		return strings.Compare(left, right)
	case leftNumeric:
		return -1
	case rightNumeric:
		return 1
	default:
		return strings.Compare(left, right)
	}
}
