package agent

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/fortnoxab/renovator/mocks"
	localredis "github.com/fortnoxab/renovator/pkg/redis"
	"github.com/fortnoxab/renovator/pkg/renovate"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/mock"
)

func TestRun(t *testing.T) {
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetFormatter(&logrus.TextFormatter{TimestampFormat: time.RFC3339Nano, FullTimestamp: true})

	commanderMock := mocks.NewMockCommander(t)
	redisMock := mocks.NewMockCmdable(t)
	runner := renovate.NewRunner(commanderMock)
	runner.BaseDir = t.TempDir()
	runner.Platform = "bitbucket-server"
	a := &Agent{
		Renovator:       runner,
		RedisClient:     redisMock,
		MaxProcessCount: 2,
	}

	repos := []string{"project1/repo1", "project1/repo2", "project2/repo1"}
	cloneDir := func(repo string) string {
		return filepath.Join(runner.BaseDir, "repos", runner.Platform, filepath.FromSlash(repo))
	}

	redisMockList := redisMockList{list: repos}

	redisMockCall := redisMock.On("BLPop", mock.Anything, time.Duration(time.Second*5), "renovator-joblist")
	redisMockCall.RunFn = func(a mock.Arguments) {
		redisMockCall.ReturnArguments = mock.Arguments{redisMockList.LPop()}
	}

	for _, repo := range repos {
		dir := cloneDir(repo)
		commanderMock.On("RunWithEnv", []string{}, "renovate", "--persist-repo-data=true", repo).
			Run(func(mock.Arguments) {
				if err := os.MkdirAll(dir, 0o750); err != nil { // renovate keeps the clone
					t.Error(err)
				}
				time.Sleep(200 * time.Millisecond)
			}).
			Return(nil).
			Once()
	}

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	a.Run(ctx)

	// The agent must remove the clones, or the disk fills up.
	for _, repo := range repos {
		if _, err := os.Stat(cloneDir(repo)); !os.IsNotExist(err) {
			t.Errorf("clone %s was not removed", cloneDir(repo))
		}
	}
}

type redisMockList struct {
	lock sync.RWMutex
	list []string
}

func (t *redisMockList) LPop() *redis.StringSliceCmd {
	t.lock.Lock()
	defer t.lock.Unlock()

	if len(t.list) == 0 {
		return redis.NewStringSliceResult(nil, redis.Nil)
	}
	// Take first value and shift remaining
	first := t.list[0]
	t.list = t.list[1:]
	return redis.NewStringSliceResult([]string{localredis.RedisRepoListKey, first}, nil)
}
