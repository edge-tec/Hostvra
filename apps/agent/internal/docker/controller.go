package docker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

var (
	validContainerIDRegex = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.-]{0,127}$`)
	ErrInvalidContainerID = errors.New("invalid container identifier: must be 1-128 characters, alphanumeric with '.', '_' or '-'")
)

type ContainerInfo struct {
	ID      string `json:"id"`
	Names   string `json:"names"`
	Image   string `json:"image"`
	Status  string `json:"status"`
	State   string `json:"state"` // running, exited, paused
	Ports   string `json:"ports"`
	Created string `json:"created"`
}

type DockerController struct{}

func NewDockerController() *DockerController {
	return &DockerController{}
}

func validateContainerIdent(idOrName string) error {
	idOrName = strings.TrimSpace(idOrName)
	if !validContainerIDRegex.MatchString(idOrName) {
		return fmt.Errorf("%w: '%s'", ErrInvalidContainerID, idOrName)
	}
	return nil
}

func (d *DockerController) IsInstalled() bool {
	_, err := exec.LookPath("docker")
	return err == nil
}

func (d *DockerController) IsDaemonRunning() bool {
	if !d.IsInstalled() {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "docker", "info")
	return cmd.Run() == nil
}

func (d *DockerController) ListContainers() ([]ContainerInfo, error) {
	if !d.IsDaemonRunning() {
		return []ContainerInfo{}, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	format := `{"id":"{{.ID}}","names":"{{.Names}}","image":"{{.Image}}","status":"{{.Status}}","state":"{{.State}}","ports":"{{.Ports}}","created":"{{.CreatedAt}}"}`
	cmd := exec.CommandContext(ctx, "docker", "ps", "-a", "--format", format)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list docker containers: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	containers := make([]ContainerInfo, 0, len(lines))

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var c ContainerInfo
		if err := json.Unmarshal([]byte(line), &c); err == nil {
			containers = append(containers, c)
		}
	}

	return containers, nil
}

func (d *DockerController) StartContainer(idOrName string) error {
	if err := validateContainerIdent(idOrName); err != nil {
		return err
	}
	if !d.IsDaemonRunning() {
		return errors.New("docker daemon is not running on this server")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "docker", "start", idOrName)
	return cmd.Run()
}

func (d *DockerController) StopContainer(idOrName string) error {
	if err := validateContainerIdent(idOrName); err != nil {
		return err
	}
	if !d.IsDaemonRunning() {
		return errors.New("docker daemon is not running on this server")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "docker", "stop", idOrName)
	return cmd.Run()
}

func (d *DockerController) RestartContainer(idOrName string) error {
	if err := validateContainerIdent(idOrName); err != nil {
		return err
	}
	if !d.IsDaemonRunning() {
		return errors.New("docker daemon is not running on this server")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "docker", "restart", idOrName)
	return cmd.Run()
}

func (d *DockerController) GetContainerLogs(idOrName string, tailLines int) (string, error) {
	if err := validateContainerIdent(idOrName); err != nil {
		return "", err
	}
	if !d.IsDaemonRunning() {
		return "", errors.New("docker daemon is not running on this server")
	}
	if tailLines <= 0 {
		tailLines = 100
	}
	if tailLines > 5000 {
		tailLines = 5000
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "docker", "logs", "--tail", fmt.Sprintf("%d", tailLines), idOrName)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return string(out), err
	}
	return string(out), nil
}

func (d *DockerController) ComposeUp(projectDir string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx, "docker", "compose", "up", "-d")
	cmd.Dir = projectDir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("docker compose up failed: %s (%w)", string(out), err)
	}
	return nil
}
