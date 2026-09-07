package docker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	validContainerIdentRegex = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.-]{0,127}$`)
	ErrInvalidContainerIdent = errors.New("invalid container identifier: must be 1-128 characters, alphanumeric with '.', '_' or '-'")
	ErrDockerNotInstalled    = errors.New("docker binary is not installed on this host")
	ErrDockerDaemonOffline   = errors.New("docker daemon is offline or not responding")
)

type DockerStatus struct {
	IsInstalled       bool   `json:"is_installed"`
	IsDaemonRunning   bool   `json:"is_daemon_running"`
	ServerVersion     string `json:"server_version"`
	ContainersTotal   int    `json:"containers_total"`
	ContainersRunning int    `json:"containers_running"`
	ContainersPaused  int    `json:"containers_paused"`
	ContainersStopped int    `json:"containers_stopped"`
	ImagesTotal       int    `json:"images_total"`
	StorageDriver     string `json:"storage_driver"`
}

type ContainerItem struct {
	ID         string `json:"id"`
	Names      string `json:"names"`
	Image      string `json:"image"`
	Command    string `json:"command"`
	CreatedAt  string `json:"created_at"`
	Status     string `json:"status"`
	State      string `json:"state"` // running, exited, paused, restarting
	Ports      string `json:"ports"`
	Size       string `json:"size,omitempty"`
}

type ImageItem struct {
	ID         string `json:"id"`
	Repository string `json:"repository"`
	Tag        string `json:"tag"`
	Size       string `json:"size"`
	CreatedAt  string `json:"created_at"`
}

type ContainerStats struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	CPUPerc  string `json:"cpu_perc"`
	MemUsage string `json:"mem_usage"`
	MemPerc  string `json:"mem_perc"`
	NetIO    string `json:"net_io"`
	BlockIO  string `json:"block_io"`
	PIDs     string `json:"pids"`
}

type RunContainerRequest struct {
	Image         string            `json:"image"`
	Name          string            `json:"name,omitempty"`
	PortMappings  []string          `json:"port_mappings,omitempty"` // e.g. ["8080:80", "443:443"]
	EnvVars       map[string]string `json:"env_vars,omitempty"`
	VolumeMounts  []string          `json:"volume_mounts,omitempty"` // e.g. ["/data:/app/data"]
	RestartPolicy string            `json:"restart_policy,omitempty"` // always, unless-stopped, on-failure
}

type DockerManager struct{}

func NewDockerManager() *DockerManager {
	return &DockerManager{}
}

func (dm *DockerManager) IsInstalled() bool {
	_, err := exec.LookPath("docker")
	return err == nil
}

func (dm *DockerManager) IsDaemonRunning() bool {
	if !dm.IsInstalled() {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "docker", "info")
	return cmd.Run() == nil
}

// GetStatus checks version, daemon state, container counts, and storage driver
func (dm *DockerManager) GetStatus() (*DockerStatus, error) {
	if !dm.IsInstalled() {
		return &DockerStatus{
			IsInstalled:     false,
			IsDaemonRunning: false,
		}, nil
	}

	if !dm.IsDaemonRunning() {
		return &DockerStatus{
			IsInstalled:     true,
			IsDaemonRunning: false,
		}, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "docker", "info", "--format", `{{json .}}`)
	out, err := cmd.Output()
	if err != nil {
		return &DockerStatus{
			IsInstalled:     true,
			IsDaemonRunning: false,
		}, nil
	}

	var raw map[string]interface{}
	_ = json.Unmarshal(out, &raw)

	status := &DockerStatus{
		IsInstalled:     true,
		IsDaemonRunning: true,
	}

	if v, ok := raw["ServerVersion"].(string); ok {
		status.ServerVersion = v
	}
	if c, ok := raw["Containers"].(float64); ok {
		status.ContainersTotal = int(c)
	}
	if c, ok := raw["ContainersRunning"].(float64); ok {
		status.ContainersRunning = int(c)
	}
	if c, ok := raw["ContainersPaused"].(float64); ok {
		status.ContainersPaused = int(c)
	}
	if c, ok := raw["ContainersStopped"].(float64); ok {
		status.ContainersStopped = int(c)
	}
	if i, ok := raw["Images"].(float64); ok {
		status.ImagesTotal = int(i)
	}
	if d, ok := raw["Driver"].(string); ok {
		status.StorageDriver = d
	}

	return status, nil
}

// ListContainers lists all or running containers
func (dm *DockerManager) ListContainers(all bool) ([]ContainerItem, error) {
	if !dm.IsDaemonRunning() {
		return []ContainerItem{}, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	format := `{"id":"{{.ID}}","names":"{{.Names}}","image":"{{.Image}}","command":{{json .Command}},"status":"{{.Status}}","state":"{{.State}}","ports":"{{.Ports}}","created_at":"{{.CreatedAt}}"}`
	args := []string{"ps", "--format", format}
	if all {
		args = append(args, "-a")
	}

	cmd := exec.CommandContext(ctx, "docker", args...)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list docker containers: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	containers := make([]ContainerItem, 0, len(lines))

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var c ContainerItem
		if err := json.Unmarshal([]byte(line), &c); err == nil {
			containers = append(containers, c)
		}
	}

	return containers, nil
}

// StartContainer starts a stopped container
func (dm *DockerManager) StartContainer(idOrName string) error {
	if err := validateContainerIdent(idOrName); err != nil {
		return err
	}
	if !dm.IsDaemonRunning() {
		return ErrDockerDaemonOffline
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "docker", "start", idOrName)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to start container: %s (%w)", string(out), err)
	}
	return nil
}

// StopContainer stops a running container
func (dm *DockerManager) StopContainer(idOrName string) error {
	if err := validateContainerIdent(idOrName); err != nil {
		return err
	}
	if !dm.IsDaemonRunning() {
		return ErrDockerDaemonOffline
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "docker", "stop", idOrName)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to stop container: %s (%w)", string(out), err)
	}
	return nil
}

// RestartContainer restarts a container
func (dm *DockerManager) RestartContainer(idOrName string) error {
	if err := validateContainerIdent(idOrName); err != nil {
		return err
	}
	if !dm.IsDaemonRunning() {
		return ErrDockerDaemonOffline
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "docker", "restart", idOrName)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to restart container: %s (%w)", string(out), err)
	}
	return nil
}

// RemoveContainer deletes a container
func (dm *DockerManager) RemoveContainer(idOrName string, force bool) error {
	if err := validateContainerIdent(idOrName); err != nil {
		return err
	}
	if !dm.IsDaemonRunning() {
		return ErrDockerDaemonOffline
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	args := []string{"rm"}
	if force {
		args = append(args, "-f")
	}
	args = append(args, idOrName)

	cmd := exec.CommandContext(ctx, "docker", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to remove container: %s (%w)", string(out), err)
	}
	return nil
}

// GetContainerLogs returns real console output
func (dm *DockerManager) GetContainerLogs(idOrName string, tailLines int) (string, error) {
	if err := validateContainerIdent(idOrName); err != nil {
		return "", err
	}
	if !dm.IsDaemonRunning() {
		return "", ErrDockerDaemonOffline
	}
	if tailLines <= 0 {
		tailLines = 100
	}
	if tailLines > 5000 {
		tailLines = 5000
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "docker", "logs", "--tail", strconv.Itoa(tailLines), idOrName)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return string(out), err
	}
	return string(out), nil
}

// GetContainerStats parses instantaneous resource consumption across running containers
func (dm *DockerManager) GetContainerStats() ([]ContainerStats, error) {
	if !dm.IsDaemonRunning() {
		return []ContainerStats{}, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	format := `{"id":"{{.ID}}","name":"{{.Name}}","cpu_perc":"{{.CPUPerc}}","mem_usage":"{{.MemUsage}}","mem_perc":"{{.MemPerc}}","net_io":"{{.NetIO}}","block_io":"{{.BlockIO}}","pids":"{{.PIDs}}"}`
	cmd := exec.CommandContext(ctx, "docker", "stats", "--no-stream", "--format", format)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("docker stats failed: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	stats := make([]ContainerStats, 0, len(lines))

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var s ContainerStats
		if err := json.Unmarshal([]byte(line), &s); err == nil {
			stats = append(stats, s)
		}
	}

	return stats, nil
}

// ListImages returns local Docker images
func (dm *DockerManager) ListImages() ([]ImageItem, error) {
	if !dm.IsDaemonRunning() {
		return []ImageItem{}, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	format := `{"id":"{{.ID}}","repository":"{{.Repository}}","tag":"{{.Tag}}","size":"{{.Size}}","created_at":"{{.CreatedAt}}"}`
	cmd := exec.CommandContext(ctx, "docker", "images", "--format", format)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("docker images failed: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	images := make([]ImageItem, 0, len(lines))

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var img ImageItem
		if err := json.Unmarshal([]byte(line), &img); err == nil {
			images = append(images, img)
		}
	}

	return images, nil
}

// RemoveImage deletes an image
func (dm *DockerManager) RemoveImage(idOrName string, force bool) error {
	if !dm.IsDaemonRunning() {
		return ErrDockerDaemonOffline
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	args := []string{"rmi"}
	if force {
		args = append(args, "-f")
	}
	args = append(args, idOrName)

	cmd := exec.CommandContext(ctx, "docker", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to remove image: %s (%w)", string(out), err)
	}
	return nil
}

// RunContainer launches a new container instance
func (dm *DockerManager) RunContainer(req RunContainerRequest) (string, error) {
	if !dm.IsDaemonRunning() {
		return "", ErrDockerDaemonOffline
	}
	if req.Image == "" {
		return "", errors.New("docker image is required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	args := []string{"run", "-d"}
	if req.Name != "" {
		if err := validateContainerIdent(req.Name); err != nil {
			return "", err
		}
		args = append(args, "--name", req.Name)
	}

	if req.RestartPolicy != "" {
		args = append(args, "--restart", req.RestartPolicy)
	} else {
		args = append(args, "--restart", "unless-stopped")
	}

	for _, p := range req.PortMappings {
		p = strings.TrimSpace(p)
		if p != "" {
			args = append(args, "-p", p)
		}
	}

	for k, v := range req.EnvVars {
		args = append(args, "-e", fmt.Sprintf("%s=%s", k, v))
	}

	for _, v := range req.VolumeMounts {
		v = strings.TrimSpace(v)
		if v != "" {
			args = append(args, "-v", v)
		}
	}

	args = append(args, req.Image)

	cmd := exec.CommandContext(ctx, "docker", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("docker run failed: %s (%w)", string(out), err)
	}

	return strings.TrimSpace(string(out)), nil
}

// PruneSystem cleans unused images, containers, networks
func (dm *DockerManager) PruneSystem() (string, error) {
	if !dm.IsDaemonRunning() {
		return "", ErrDockerDaemonOffline
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx, "docker", "system", "prune", "-f")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return string(out), fmt.Errorf("docker prune failed: %w", err)
	}
	return string(out), nil
}

func validateContainerIdent(idOrName string) error {
	idOrName = strings.TrimSpace(idOrName)
	if !validContainerIdentRegex.MatchString(idOrName) {
		return fmt.Errorf("%w: '%s'", ErrInvalidContainerIdent, idOrName)
	}
	return nil
}
