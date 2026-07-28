package repoowner

import (
	"os"
	"path/filepath"
	"testing"
)

func writeConfig(t *testing.T, dir, body string) string {
	t.Helper()
	if err := os.MkdirAll(dir, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.json"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestResolve(t *testing.T) {
	base := t.TempDir()

	cases := []struct {
		name       string
		cloneDir   string
		slug       string
		wantTeam   string
		wantSvcNam string
	}{
		{
			"full config",
			writeConfig(t, filepath.Join(base, "currencies"), `{"serviceName":"currencies","team":"gonix"}`),
			"fnx/currencies", "gonix", "currencies",
		},
		{
			"missing team falls back, keeps serviceName",
			writeConfig(t, filepath.Join(base, "no-team"), `{"serviceName":"cache"}`),
			"gl/no-team", Unknown, "cache",
		},
		{
			"broken json falls back to slug",
			writeConfig(t, filepath.Join(base, "broken"), `{not json`),
			"gl/broken", Unknown, "gl/broken",
		},
		{
			"missing clone falls back to slug",
			filepath.Join(base, "does-not-exist"),
			"gl/does-not-exist", Unknown, "gl/does-not-exist",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			team, name := Resolve(tc.cloneDir, tc.slug)
			if team != tc.wantTeam {
				t.Errorf("team = %q, want %q", team, tc.wantTeam)
			}
			if name != tc.wantSvcNam {
				t.Errorf("name = %q, want %q", name, tc.wantSvcNam)
			}
		})
	}
}
