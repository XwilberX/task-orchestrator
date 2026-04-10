package executor

import (
	"archive/tar"
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/XwilberX/task-orchestrator/internal/events"
	"github.com/XwilberX/task-orchestrator/internal/logger"
	"github.com/XwilberX/task-orchestrator/internal/runtime"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
)

// Executor ejecuta tareas en contenedores Docker aislados con gVisor.
type Executor struct {
	cli       *client.Client
	runtime   string // "runsc" para gVisor, "" para runc (dev)
	cache     imageCache
	vlogs     *logger.Client
	logBroker *events.LogBroker
}

// New crea un Executor conectado al daemon Docker.
func New(gvisorRuntime string, vlogs *logger.Client, logBroker *events.LogBroker) (*Executor, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("docker client: %w", err)
	}
	return &Executor{cli: cli, runtime: gvisorRuntime, vlogs: vlogs, logBroker: logBroker}, nil
}

// Run ejecuta la tarea y devuelve el resultado.
// Siempre elimina el contenedor al terminar, incluso si hay panic.
func (e *Executor) Run(ctx context.Context, cfg RunConfig) (*ExecResult, error) {
	rt, err := runtime.Get(cfg.Runtime)
	if err != nil {
		return nil, err
	}

	// 1. Asegurar imagen disponible
	if err := e.cache.ensure(ctx, e.cli, rt.Image()); err != nil {
		return nil, fmt.Errorf("pull imagen: %w", err)
	}

	// 2. Comando del contenedor
	cmd := buildContainerCmd(rt, cfg)

	// 3. Red
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

	// Limpieza garantizada
	defer func() {
		cleanCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		e.cli.ContainerRemove(cleanCtx, resp.ID, container.RemoveOptions{Force: true})
	}()

	// 5. Inyectar código vía tar
	if err := e.copyCode(ctx, resp.ID, rt.Entrypoint(), cfg.Code); err != nil {
		return nil, fmt.Errorf("copiar código: %w", err)
	}

	// 6. Arrancar contenedor
	if err := e.cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		return nil, fmt.Errorf("iniciar contenedor: %w", err)
	}

	// 7. Esperar y recolectar logs en paralelo
	return e.waitAndStream(ctx, resp.ID, cfg)
}

// waitAndStream espera al contenedor y streamea stdout/stderr a Victoria Logs en tiempo real.
func (e *Executor) waitAndStream(ctx context.Context, containerID string, cfg RunConfig) (*ExecResult, error) {
	// Stream de logs en paralelo (Follow=true)
	logCtx, logCancel := context.WithCancel(ctx)
	defer logCancel()

	var outputBuf bytes.Buffer
	logsDone := make(chan struct{})

	go func() {
		defer close(logsDone)
		e.streamLogs(logCtx, containerID, cfg, &outputBuf)
	}()

	statusCh, errCh := e.cli.ContainerWait(ctx, containerID, container.WaitConditionNotRunning)

	var exitCode int
	var timedOut bool

	select {
	case err := <-errCh:
		if err != nil {
			if ctx.Err() != nil {
				timedOut = true
				killCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				e.cli.ContainerKill(killCtx, containerID, "SIGKILL")
			} else {
				return nil, fmt.Errorf("esperar contenedor: %w", err)
			}
		}
	case status := <-statusCh:
		exitCode = int(status.StatusCode)
	case <-ctx.Done():
		timedOut = true
		killCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		e.cli.ContainerKill(killCtx, containerID, "SIGKILL")
	}

	logCancel()
	<-logsDone

	return &ExecResult{
		ExitCode: exitCode,
		Output:   outputBuf.String(),
		TimedOut: timedOut,
	}, nil
}

// streamLogs lee stdout/stderr del contenedor línea a línea y los envía a Victoria Logs.
func (e *Executor) streamLogs(ctx context.Context, containerID string, cfg RunConfig, out io.Writer) {
	rc, err := e.cli.ContainerLogs(ctx, containerID, container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     true,
		Timestamps: false,
	})
	if err != nil {
		return
	}
	defer rc.Close()

	var stdoutBuf, stderrBuf bytes.Buffer
	stdcopy.StdCopy(&stdoutBuf, &stderrBuf, rc)

	e.scanAndSend(&stdoutBuf, "stdout", cfg, out)
	e.scanAndSend(&stderrBuf, "stderr", cfg, out)
}

// scanAndSend envía cada línea del buffer a Victoria Logs y al LogBroker.
func (e *Executor) scanAndSend(r io.Reader, stream string, cfg RunConfig, out io.Writer) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		if out != nil {
			fmt.Fprintln(out, line)
		}
		if e.vlogs != nil {
			e.vlogs.Write(logger.LogEntry{
				Msg:            line,
				Time:           time.Now().UTC(),
				TaskID:         cfg.TaskID,
				DefinitionName: cfg.DefinitionName,
				Runtime:        cfg.Runtime,
				Attempt:        cfg.Attempt,
				Stream:         stream,
			})
		}
		if e.logBroker != nil {
			e.logBroker.Publish(cfg.TaskID, line)
		}
	}
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

	if cfg.Runtime == "java" {
		return runParts
	}

	if cfg.Packages != "" {
		installCmd := rt.InstallCommand(cfg.Packages)
		return []string{"sh", "-c", installCmd + " && " + runCmd}
	}
	return []string{"sh", "-c", runCmd}
}
