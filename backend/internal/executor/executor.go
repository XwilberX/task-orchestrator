package executor

import (
	"archive/tar"
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/XwilberX/task-orchestrator/internal/runtime"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
)

// Executor ejecuta tareas en contenedores Docker aislados con gVisor.
type Executor struct {
	cli     *client.Client
	runtime string // "runsc" para gVisor, "" para runc (dev)
	cache   imageCache
}

// New crea un Executor conectado al daemon Docker.
// gvisorRuntime debe ser "runsc" en producción y "" en desarrollo/CI.
func New(gvisorRuntime string) (*Executor, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("docker client: %w", err)
	}
	return &Executor{cli: cli, runtime: gvisorRuntime}, nil
}

// Run ejecuta la tarea y devuelve el resultado.
// Siempre elimina el contenedor al terminar, incluso si hay panic.
func (e *Executor) Run(ctx context.Context, cfg RunConfig) (*ExecResult, error) {
	rt, err := runtime.Get(cfg.Runtime)
	if err != nil {
		return nil, err
	}

	// 1. Asegurar que la imagen esté disponible
	if err := e.cache.ensure(ctx, e.cli, rt.Image()); err != nil {
		return nil, fmt.Errorf("pull imagen: %w", err)
	}

	// 2. Construir el comando del contenedor
	cmd := buildContainerCmd(rt, cfg)

	// 3. Configuración de red
	networkMode := container.NetworkMode("none")
	if cfg.NetworkEnabled {
		networkMode = ""
	}

	// 4. Crear contenedor
	resp, err := e.cli.ContainerCreate(ctx,
		&container.Config{
			Image:      rt.Image(),
			Cmd:        cmd,
			WorkingDir: "/task",
		},
		&container.HostConfig{
			Runtime:     e.runtime,
			NetworkMode: networkMode,
			Resources: container.Resources{
				Memory:    int64(cfg.MemoryMB) * 1024 * 1024,
				CPUShares: int64(cfg.CPUShares),
			},
		},
		nil, nil, "",
	)
	if err != nil {
		return nil, fmt.Errorf("crear contenedor: %w", err)
	}

	// defer de limpieza — siempre se ejecuta
	defer func() {
		cleanCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		e.cli.ContainerRemove(cleanCtx, resp.ID, container.RemoveOptions{Force: true})
	}()

	// 5. Inyectar código vía tar (sin bind mounts)
	if err := e.copyCode(ctx, resp.ID, rt.Entrypoint(), cfg.Code); err != nil {
		return nil, fmt.Errorf("copiar código: %w", err)
	}

	// 6. Arrancar contenedor
	if err := e.cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		return nil, fmt.Errorf("iniciar contenedor: %w", err)
	}

	// 7. Esperar con timeout
	timeout := time.Duration(cfg.TimeoutSeconds) * time.Second
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	return e.waitAndCollect(runCtx, resp.ID, cfg.TimeoutSeconds)
}

// waitAndCollect espera a que el contenedor termine y recolecta stdout+stderr.
func (e *Executor) waitAndCollect(ctx context.Context, containerID string, timeoutSecs int) (*ExecResult, error) {
	statusCh, errCh := e.cli.ContainerWait(ctx, containerID, container.WaitConditionNotRunning)

	select {
	case err := <-errCh:
		if err != nil {
			// Timeout del context
			if ctx.Err() != nil {
				killCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				e.cli.ContainerKill(killCtx, containerID, "SIGKILL")
				return &ExecResult{ExitCode: -1, TimedOut: true}, nil
			}
			return nil, fmt.Errorf("esperar contenedor: %w", err)
		}
	case status := <-statusCh:
		output, _ := e.collectLogs(context.Background(), containerID)
		return &ExecResult{
			ExitCode: int(status.StatusCode),
			Output:   output,
		}, nil
	case <-ctx.Done():
		killCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		e.cli.ContainerKill(killCtx, containerID, "SIGKILL")
		return &ExecResult{ExitCode: -1, TimedOut: true}, nil
	}

	output, _ := e.collectLogs(context.Background(), containerID)
	return &ExecResult{Output: output}, nil
}

// collectLogs obtiene stdout+stderr del contenedor.
func (e *Executor) collectLogs(ctx context.Context, containerID string) (string, error) {
	rc, err := e.cli.ContainerLogs(ctx, containerID, container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
	})
	if err != nil {
		return "", err
	}
	defer rc.Close()

	var buf bytes.Buffer
	io.Copy(&buf, rc)
	return buf.String(), nil
}

// copyCode crea un tar en memoria e inyecta el código en el contenedor.
func (e *Executor) copyCode(ctx context.Context, containerID, filename, code string) error {
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	content := []byte(code)
	if err := tw.WriteHeader(&tar.Header{
		Name: filename,
		Mode: 0644,
		Size: int64(len(content)),
	}); err != nil {
		return err
	}
	if _, err := tw.Write(content); err != nil {
		return err
	}
	if err := tw.Close(); err != nil {
		return err
	}
	return e.cli.CopyToContainer(ctx, containerID, "/task/", &buf, types.CopyToContainerOptions{})
}

// buildContainerCmd construye el comando final del contenedor.
func buildContainerCmd(rt runtime.Runtime, cfg RunConfig) []string {
	entrypoint := "/task/" + rt.Entrypoint()
	runParts := rt.RunCommand(entrypoint, cfg.Args)
	runCmd := strings.Join(runParts, " ")

	// Java ya devuelve un comando con sh -c incluido
	if cfg.Runtime == "java" {
		return runParts
	}

	if cfg.Packages != "" {
		installCmd := rt.InstallCommand(cfg.Packages)
		return []string{"sh", "-c", installCmd + " && " + runCmd}
	}
	return []string{"sh", "-c", runCmd}
}
