package diagnostics

import "testing"

func TestCheckAppImageMountIgnoresNosuid(t *testing.T) {
	const executable = "/tmp/.mount_BetterABC123/usr/bin/better-rpt"
	cases := []struct {
		name    string
		options string
		want    Status
		next    string
	}{
		{
			name:    "runtime mount with nosuid",
			options: "ro,nosuid,nodev,relatime",
			want:    StatusOK,
		},
		{
			name:    "noexec mount",
			options: "ro,noexec,nosuid,nodev,relatime",
			want:    StatusFail,
			next:    "diag.appimage.mount.next",
		},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			e := &evaluator{
				executable: executable,
				deps: &Deps{
					MountInfo: "/proc/self/mountinfo",
					ReadFile: func(string) ([]byte, error) {
						return []byte("36 35 0:145 / " + executable + " " + test.options + " - fuse.test none ro\n"), nil
					},
				},
			}
			if got := e.checkAppImageMount(); got != test.want {
				t.Fatalf("status = %s, want %s", got, test.want)
			}
			if e.mountNext != test.next {
				t.Fatalf("next = %q, want %q", e.mountNext, test.next)
			}
		})
	}
}
