package renovate

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/fortnoxab/renovator/mocks"
	"github.com/stretchr/testify/mock"
)

func TestRunRenovate(t *testing.T) {
	base := t.TempDir()
	commanderMock := mocks.NewMockCommander(t)
	r := NewRunner(commanderMock)
	r.BaseDir = base
	r.Platform = "bitbucket-server"

	dir := filepath.Join(base, "repos", "bitbucket-server", "fo", "wzrd")
	commanderMock.On("RunWithEnv", []string{"LOG_LEVEL=debug"}, "renovate", "--persist-repo-data=true", "fo/wzrd").
		Run(func(mock.Arguments) {
			if err := os.MkdirAll(dir, 0o750); err != nil {
				t.Error(err)
			}
		}).
		Return(nil).
		Once()

	run, err := r.RunRenovate("fo/wzrd?loglevel=debug")
	if err != nil {
		t.Fatal(err)
	}
	if run.Slug != "fo/wzrd" {
		t.Errorf("slug = %q, want %q", run.Slug, "fo/wzrd")
	}
	if run.CloneDir != dir {
		t.Errorf("cloneDir = %q, want %q", run.CloneDir, dir)
	}
	if _, err := os.Stat(dir); err != nil {
		t.Fatalf("clone should exist before cleanup: %s", err)
	}

	run.Cleanup()

	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Errorf("clone %s was not removed", dir)
	}
}
