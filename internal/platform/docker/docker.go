package docker

import (
	"archive/tar"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/docker/go-units"
)

const (
	buildSourceDir      = "/app/src"
	buildNodeModulesDir = "/app/src/node_modules"
	buildCacheDir       = "/app/.build-cache"
	buildOutputDir      = "/app/output"
	buildPathEnv        = buildCacheDir + "/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"
)

// Client wraps the Docker Engine SDK for build operations.
type Client struct {
	cli *client.Client
}

// NewClient creates a Docker client from the default environment.
func NewClient() (*Client, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("docker client init: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := cli.Ping(ctx); err != nil {
		return nil, fmt.Errorf("docker ping failed: %w", err)
	}
	return &Client{cli: cli}, nil
}

// Close releases the Docker client resources.
func (c *Client) Close() error {
	return c.cli.Close()
}

// BuildContainerOpts configures a build container.
type BuildContainerOpts struct {
	DeploymentID string
	NodeVersion  string
	SourceDir    string
	OutputDir    string
	CacheVolume  string
	BuildCache   string
	EnvVars      []string // KEY=VALUE format
	MemoryBytes  int64
	NanoCPUs     int64
	PIDLimit     int64
	WorkDir      string
}

// CreateBuildContainer creates and starts a container configured for builds.
func (c *Client) CreateBuildContainer(ctx context.Context, opts BuildContainerOpts) (string, error) {
	img := "node:" + opts.NodeVersion + "-slim"

	if err := c.ensureImage(ctx, img); err != nil {
		return "", fmt.Errorf("ensure image %s: %w", img, err)
	}

	workDir := opts.WorkDir
	if workDir == "" {
		workDir = buildSourceDir
	}

	pidLimit := opts.PIDLimit
	config := &container.Config{
		Image:      img,
		Env:        opts.EnvVars,
		WorkingDir: workDir,
		Labels: map[string]string{
			"hostbox.deployment": opts.DeploymentID,
			"hostbox.managed":    "true",
		},
		Cmd:       []string{"sleep", "infinity"},
		Tty:       false,
		OpenStdin: false,
	}

	hostConfig := &container.HostConfig{
		Resources: container.Resources{
			Memory:    opts.MemoryBytes,
			NanoCPUs:  opts.NanoCPUs,
			PidsLimit: &pidLimit,
			Ulimits: []*units.Ulimit{
				{Name: "nofile", Soft: 1024, Hard: 1024},
			},
		},
		SecurityOpt:    []string{"no-new-privileges"},
		CapDrop:        []string{"ALL"},
		CapAdd:         []string{"DAC_OVERRIDE", "FOWNER"},
		ReadonlyRootfs: true,
		Tmpfs: map[string]string{
			"/tmp": "rw,noexec,nosuid,size=512m",
		},
		Mounts: buildContainerMounts(opts),
	}

	resp, err := c.cli.ContainerCreate(ctx, config, hostConfig, nil, nil, "build-"+opts.DeploymentID)
	if err != nil {
		return "", fmt.Errorf("container create: %w", err)
	}

	if err := c.cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		_ = c.cli.ContainerRemove(ctx, resp.ID, container.RemoveOptions{Force: true})
		return "", fmt.Errorf("container start: %w", err)
	}

	return resp.ID, nil
}

// ExecCommand runs a shell command inside a running container.
func (c *Client) ExecCommand(ctx context.Context, containerID string, cmd string, stdout, stderr io.Writer) error {
	execConfig := container.ExecOptions{
		Cmd:          []string{"sh", "-c", wrapBuildCommand(cmd)},
		AttachStdout: true,
		AttachStderr: true,
	}

	execResp, err := c.cli.ContainerExecCreate(ctx, containerID, execConfig)
	if err != nil {
		return fmt.Errorf("exec create: %w", err)
	}

	attachResp, err := c.cli.ContainerExecAttach(ctx, execResp.ID, container.ExecAttachOptions{})
	if err != nil {
		return fmt.Errorf("exec attach: %w", err)
	}
	defer attachResp.Close()

	if _, err := stdcopy.StdCopy(stdout, stderr, attachResp.Reader); err != nil {
		return fmt.Errorf("exec stream: %w", err)
	}

	inspectResp, err := c.cli.ContainerExecInspect(ctx, execResp.ID)
	if err != nil {
		return fmt.Errorf("exec inspect: %w", err)
	}
	if inspectResp.ExitCode != 0 {
		return fmt.Errorf("command exited with code %d", inspectResp.ExitCode)
	}

	return nil
}

func buildContainerMounts(opts BuildContainerOpts) []mount.Mount {
	return []mount.Mount{
		{Type: mount.TypeBind, Source: opts.SourceDir, Target: buildSourceDir, ReadOnly: false},
		{Type: mount.TypeVolume, Source: opts.CacheVolume, Target: buildNodeModulesDir},
		{Type: mount.TypeVolume, Source: opts.BuildCache, Target: buildCacheDir},
		{Type: mount.TypeBind, Source: opts.OutputDir, Target: buildOutputDir, ReadOnly: false},
	}
}

func wrapBuildCommand(cmd string) string {
	prelude := []string{
		fmt.Sprintf("export HOME=%s/home", buildCacheDir),
		fmt.Sprintf("export XDG_CACHE_HOME=%s/xdg-cache", buildCacheDir),
		fmt.Sprintf("export COREPACK_HOME=%s/corepack", buildCacheDir),
		fmt.Sprintf("export NPM_CONFIG_CACHE=%s/npm", buildCacheDir),
		fmt.Sprintf("export YARN_CACHE_FOLDER=%s/yarn", buildCacheDir),
		fmt.Sprintf("export PATH=%q", buildPathEnv),
		fmt.Sprintf("mkdir -p %s/home %s/xdg-cache %s/corepack %s/npm %s/yarn %s/bin", buildCacheDir, buildCacheDir, buildCacheDir, buildCacheDir, buildCacheDir, buildCacheDir),
		fmt.Sprintf("printf '%%s\\n' '#!/bin/sh' 'exec corepack pnpm \"$@\"' > %s/bin/pnpm", buildCacheDir),
		fmt.Sprintf("printf '%%s\\n' '#!/bin/sh' 'exec corepack yarn \"$@\"' > %s/bin/yarn", buildCacheDir),
		fmt.Sprintf("printf '%%s\\n' '#!/bin/sh' 'exec npx --yes bun@1 \"$@\"' > %s/bin/bun", buildCacheDir),
		fmt.Sprintf("chmod +x %s/bin/pnpm %s/bin/yarn %s/bin/bun", buildCacheDir, buildCacheDir, buildCacheDir),
	}

	return strings.Join(append(prelude, cmd), "\n")
}

// StopContainer stops a container with the given grace period.
func (c *Client) StopContainer(ctx context.Context, containerID string, gracePeriod time.Duration) error {
	timeout := int(gracePeriod.Seconds())
	return c.cli.ContainerStop(ctx, containerID, container.StopOptions{Timeout: &timeout})
}

// RemoveContainer force-removes a container.
func (c *Client) RemoveContainer(ctx context.Context, nameOrID string) error {
	return c.cli.ContainerRemove(ctx, nameOrID, container.RemoveOptions{
		Force:         true,
		RemoveVolumes: false,
	})
}

// RemoveContainerByName removes a container by name, ignoring errors.
func (c *Client) RemoveContainerByName(ctx context.Context, name string) {
	_ = c.cli.ContainerRemove(ctx, name, container.RemoveOptions{Force: true})
}

// CopyFromContainer copies a directory from the container to the host.
func (c *Client) CopyFromContainer(ctx context.Context, containerID, srcPath, destPath string) (int64, error) {
	reader, stat, err := c.cli.CopyFromContainer(ctx, containerID, srcPath)
	if err != nil {
		return 0, fmt.Errorf("copy from container %s:%s: %w", containerID, srcPath, err)
	}
	defer reader.Close()

	size, err := extractTar(reader, destPath, stat.Name)
	if err != nil {
		return 0, fmt.Errorf("extract tar to %s: %w", destPath, err)
	}

	return size, nil
}

// RemoveVolume removes a named Docker volume.
func (c *Client) RemoveVolume(ctx context.Context, name string) error {
	return c.cli.VolumeRemove(ctx, name, true)
}

// ListManagedContainers returns all containers with the "hostbox.managed=true" label.
func (c *Client) ListManagedContainers(ctx context.Context) ([]types.Container, error) {
	return c.cli.ContainerList(ctx, container.ListOptions{
		All: true,
		Filters: filters.NewArgs(
			filters.Arg("label", "hostbox.managed=true"),
		),
	})
}

func (c *Client) ensureImage(ctx context.Context, img string) error {
	_, _, err := c.cli.ImageInspectWithRaw(ctx, img)
	if err == nil {
		return nil
	}

	reader, err := c.cli.ImagePull(ctx, "docker.io/library/"+img, image.PullOptions{})
	if err != nil {
		return err
	}
	defer reader.Close()
	_, _ = io.Copy(io.Discard, reader)
	return nil
}

func extractTar(reader io.Reader, destDir, stripRoot string) (int64, error) {
	tr := tar.NewReader(reader)
	baseDir := filepath.Clean(destDir)
	stripRoot = strings.Trim(strings.TrimPrefix(filepath.ToSlash(stripRoot), "./"), "/")
	var totalSize int64

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return 0, err
		}

		name := strings.Trim(strings.TrimPrefix(filepath.ToSlash(header.Name), "./"), "/")
		if stripRoot != "" {
			switch {
			case name == stripRoot:
				continue
			case strings.HasPrefix(name, stripRoot+"/"):
				name = strings.TrimPrefix(name, stripRoot+"/")
			}
		}
		if name == "" {
			continue
		}

		target := filepath.Join(baseDir, filepath.FromSlash(name))

		// Prevent path traversal
		if !isWithinBaseDir(baseDir, target) {
			return 0, fmt.Errorf("invalid tar entry: %s", header.Name)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0755); err != nil {
				return 0, err
			}
		case tar.TypeReg, tar.TypeRegA:
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return 0, err
			}
			f, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(header.Mode))
			if err != nil {
				return 0, err
			}
			written, err := io.Copy(f, tr)
			if err != nil {
				f.Close()
				return 0, err
			}
			f.Close()
			totalSize += written
		}
	}

	return totalSize, nil
}

func isWithinBaseDir(baseDir, target string) bool {
	rel, err := filepath.Rel(baseDir, target)
	if err != nil {
		return false
	}

	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator)))
}
