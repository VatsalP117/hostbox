package docker

import (
	"context"
	"io"
	"time"

	"github.com/docker/docker/api/types"
)

// DockerClient is the interface used by the build executor.
// The real Client implements it; tests use a mock.
type DockerClient interface {
	CreateBuildContainer(ctx context.Context, opts BuildContainerOpts) (string, error)
	ExecCommand(ctx context.Context, containerID string, cmd string, stdout, stderr io.Writer) error
	StopContainer(ctx context.Context, containerID string, gracePeriod time.Duration) error
	RemoveContainer(ctx context.Context, nameOrID string) error
	RemoveContainerByName(ctx context.Context, name string)
	CopyFromContainer(ctx context.Context, containerID, srcPath, destPath string) (int64, error)
	RemoveVolume(ctx context.Context, name string) error
	ListManagedContainers(ctx context.Context) ([]types.Container, error)
	Close() error
}
