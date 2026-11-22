package runner

import (
	"bytes"
	"codepulse/internal/domain"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
)

type DockerRunner struct {
	cli *client.Client
}

func NewDockerRunner() (*DockerRunner, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}
	return &DockerRunner{cli: cli}, nil
}

func (r *DockerRunner) Close() error {
	if err := r.cli.Close(); err != nil {
		return err
	}
	return nil
}

func (r *DockerRunner) Ping(ctx context.Context) error {
	_, err := r.cli.Ping(ctx)
	if err != nil {
		return err
	}
	return nil
}

func (r *DockerRunner) Run(ctx context.Context, sourceCode string, tests []domain.Test, timeLimitMs int, memoryLimitMB int) (*domain.ExecutionResult, error) {
	dir, err := os.MkdirTemp("", "submission-*")
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := os.RemoveAll(dir); err != nil {
			slog.Error("failed to remove temp dir", "dir", dir, "error", err)
		}
	}()

	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte(sourceCode), 0644); err != nil {
		return nil, err
	}

	compileErr, err := r.compile(ctx, dir)
	if err != nil {
		return nil, err
	}

	if compileErr != "" {
		return &domain.ExecutionResult{
			Verdict:      domain.StatusCompilationError,
			ErrorMessage: compileErr,
		}, nil
	}

	maxDuration := 0
	for _, test := range tests {
		status, time, runErr, err := r.runTest(ctx, dir, test, timeLimitMs, memoryLimitMB)
		if err != nil {
			return nil, err
		}

		if time > maxDuration {
			maxDuration = time
		}

		if status != domain.StatusAccepted {
			return &domain.ExecutionResult{
				Verdict:       status,
				ExecutionTime: time,
				ErrorMessage:  runErr,
			}, nil
		}
	}

	return &domain.ExecutionResult{
		Verdict:       domain.StatusAccepted,
		ExecutionTime: maxDuration,
		ErrorMessage:  "",
	}, nil
}

func (r *DockerRunner) compile(ctx context.Context, dir string) (string, error) {
	cfg := container.Config{
		Image: "golang:1.27-alpine",
		Cmd:   []string{"go", "build", "-o", "/sandbox/solution", "/sandbox/main.go"},
	}

	hostCfg := container.HostConfig{
		Binds: []string{dir + ":/sandbox"},
	}

	resp, err := r.cli.ContainerCreate(ctx, &cfg, &hostCfg, nil, nil, "")
	if err != nil {
		return "", err
	}
	defer func() {
		if err := r.cli.ContainerRemove(context.Background(), resp.ID, container.RemoveOptions{Force: true}); err != nil {
			slog.Error("failed to remove container", "container", resp.ID, "error", err)
		}
	}()

	err = r.cli.ContainerStart(ctx, resp.ID, container.StartOptions{})
	if err != nil {
		return "", err
	}

	statusCh, errCh := r.cli.ContainerWait(ctx, resp.ID, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil {
			return "", err
		}
	case status := <-statusCh:
		if status.StatusCode == 0 {
			return "", nil
		}
	}

	out, err := r.cli.ContainerLogs(ctx, resp.ID, container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
	})
	if err != nil {
		return "", err
	}
	defer func() {
		if err := out.Close(); err != nil {
			slog.Error("failed to close ContainerLogs", "logs", out, "error", err)
		}
	}()

	output, err := io.ReadAll(out)
	if err != nil {
		return "", err
	}

	return string(output), nil
}

func (r *DockerRunner) runTest(ctx context.Context, dir string, test domain.Test, timeLimitMs int, memoryLimitMB int) (domain.SubmissionStatus, int, string, error) {
	if err := os.WriteFile(filepath.Join(dir, "input.txt"), []byte(test.Input), 0644); err != nil {
		return "", 0, "", fmt.Errorf("failed to write input file: %w", err)
	}

	cfg := container.Config{
		Image: "golang:1.27-alpine",
		Cmd:   []string{"sh", "-c", "/sandbox/solution < /sandbox/input.txt"},
	}

	hostCfg := container.HostConfig{
		Binds:       []string{dir + ":/sandbox:ro"},
		NetworkMode: "none",
		Resources: container.Resources{
			Memory:     int64(memoryLimitMB) * 1024 * 1024,
			MemorySwap: int64(memoryLimitMB) * 1024 * 1024,
			NanoCPUs:   1_000_000_000,
		},
	}

	resp, err := r.cli.ContainerCreate(ctx, &cfg, &hostCfg, nil, nil, "")
	if err != nil {
		return "", 0, "", fmt.Errorf("failed to create container: %w", err)
	}
	defer func() {
		if err := r.cli.ContainerRemove(context.Background(), resp.ID, container.RemoveOptions{Force: true}); err != nil {
			slog.Error("failed to remove container", "container", resp.ID, "error", err)
		}
	}()

	testCtx, cancel := context.WithTimeout(ctx, time.Duration(timeLimitMs)*time.Millisecond)
	defer cancel()

	err = r.cli.ContainerStart(testCtx, resp.ID, container.StartOptions{})
	if err != nil {
		return "", 0, "", fmt.Errorf("failed to start container: %w", err)
	}
	start := time.Now()

	statusCh, errCh := r.cli.ContainerWait(testCtx, resp.ID, container.WaitConditionNotRunning)

	select {
	case err := <-errCh:
		duration := int(time.Since(start).Milliseconds())
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(testCtx.Err(), context.DeadlineExceeded) {
			return domain.StatusTimeLimitExceeded, duration, "Time Limit Exceeded", nil
		}
		return "", duration, "", fmt.Errorf("container wait error: %w", err)

	case status := <-statusCh:
		duration := int(time.Since(start).Milliseconds())

		out, err := r.cli.ContainerLogs(ctx, resp.ID, container.LogsOptions{
			ShowStdout: true,
			ShowStderr: true,
		})
		if err != nil {
			return "", duration, "", fmt.Errorf("failed to get container logs: %w", err)
		}

		defer func() {
			if err := out.Close(); err != nil {
				slog.Error("failed to close ContainerLogs", "logs", out, "error", err)
			}
		}()

		var stdout, stderr bytes.Buffer
		if _, err := stdcopy.StdCopy(&stdout, &stderr, out); err != nil {
			return "", duration, "", fmt.Errorf("failed to demux container logs: %w", err)
		}

		if status.StatusCode != 0 {
			inspect, inspectErr := r.cli.ContainerInspect(context.Background(), resp.ID)
			if inspectErr == nil && inspect.State.OOMKilled {
				return domain.StatusMemoryLimitExceeded, duration, "Memory Limit Exceeded", nil
			}

			return domain.StatusRuntimeError, duration, stderr.String(), nil
		}
		if strings.TrimSpace(stdout.String()) == strings.TrimSpace(test.ExpectedOutput) {
			return domain.StatusAccepted, duration, "", nil
		}
		return domain.StatusWrongAnswer, duration, "", nil
	}

}
