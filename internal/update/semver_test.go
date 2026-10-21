package update

import "testing"

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		name  string
		left  string
		right string
		want  int
	}{
		{name: "equal", left: "0.2.1", right: "0.2.1", want: 0},
		{name: "v prefix", left: "v0.2.1", right: "0.2.1", want: 0},
		{name: "build metadata ignored", left: "0.2.1+build.7", right: "0.2.1", want: 0},
		{name: "minor is numeric", left: "0.10.0", right: "0.9.0", want: 1},
		{name: "patch is numeric", left: "0.2.10", right: "0.2.9", want: 1},
		{name: "prerelease is older", left: "0.2.1-rc.1", right: "0.2.1", want: -1},
		{name: "numeric prerelease is older", left: "0.2.1-2", right: "0.2.1-10", want: -1},
		{name: "numeric before text", left: "0.2.1-2", right: "0.2.1-rc", want: -1},
		{name: "prerelease lexicographic", left: "0.2.1-beta", right: "0.2.1-rc", want: -1},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			left, err := ParseVersion(test.left)
			if err != nil {
				t.Fatal(err)
			}
			right, err := ParseVersion(test.right)
			if err != nil {
				t.Fatal(err)
			}
			if got := left.Compare(right); got != test.want {
				t.Fatalf("%s.Compare(%s) = %d, want %d", test.left, test.right, got, test.want)
			}
		})
	}
}

func TestParseVersionRejectsInvalid(t *testing.T) {
	for _, value := range []string{"", "dev", "1", "1.2", "01.2.3", "1.02.3", "1.2.03", "1.2.3-01", "1.2.3-"} {
		if _, err := ParseVersion(value); err == nil {
			t.Errorf("ParseVersion(%q) succeeded, want error", value)
		}
	}
}

func TestVersionString(t *testing.T) {
	version, err := ParseVersion("v1.2.3-rc.1+build.9")
	if err != nil {
		t.Fatal(err)
	}
	if got := version.String(); got != "1.2.3-rc.1" {
		t.Fatalf("String() = %q", got)
	}
}
