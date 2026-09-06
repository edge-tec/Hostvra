package docker

import (
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"
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

func (d *DockerController) IsInstalled() bool {
	_, err := exec.LookPath("docker")
	return err == nil
}

func (d *DockerController) IsDaemonRunning() bool {
	if !d.IsInstalled() {
		return false
	}
	cmd := exec.Command("docker", "info")
	return cmd.Run() == nil
}

func (d *DockerController) ListContainers() ([]ContainerInfo, error) {
	if !d.IsDaemonRunning() {
		// If Docker daemon is not active on this node, return empty list gracefully
		return []ContainerInfo{}, nil
	}

	format := `{"id":"{{.ID}}","names":"{{.Names}}","image":"{{.Image}}","status":"{{.Status}}","state":"{{.State}}","ports":"{{.Ports}}","created":"{{.CreatedAt}}"}`
	cmd := exec.Command("docker", "ps", "-a", "--format", format)
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
	if !d.IsDaemonRunning() {
		return errors.New("docker daemon is not running on this server")
	}
	cmd := exec.Command("docker", "start", idOrName)
	return cmd.Run()
}

func (d *DockerController) StopContainer(idOrName string) error {
	if !d.IsDaemonRunning() {
		return errors.New("docker daemon is not running on this server")
	}
	cmd := exec.Command("docker", "stop", idOrName)
	return cmd.Run()
}

func (d *DockerController) RestartContainer(idOrName string) error {
	if !d.IsDaemonRunning() {
		return errors.New("docker daemon is not running on this server")
	}
	cmd := exec.Command("docker", "restart", idOrName)
	return cmd.Run()
}

func (d *DockerController) GetContainerLogs(idOrName string, tailLines int) (string, error) {
	if !d.IsDaemonRunning() {
		return "", errors.New("docker daemon is not running on this server")
	}
	if tailLines <= 0 {
		tailLines = 100
	}
	cmd := exec.Command("docker", "logs", "--tail", fmt.Sprintf("%d", tailLines), idOrName)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return string(out), err
	}
	return string(out), nil
}

func (d *DockerController) ComposeUp(projectDir string) error {
	cmd := exec.Command("docker", "compose", "up", "-d")
	cmd.Dir = projectDir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("docker compose up failed: %s (%w)", string(out), err)
	}
	return nil
}
