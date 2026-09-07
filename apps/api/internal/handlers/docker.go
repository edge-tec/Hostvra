package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"hostvra/agent/pkg/docker"
	"hostvra/api/internal/audit"
	"hostvra/api/internal/config"
	"hostvra/api/internal/response"
	"hostvra/api/internal/store"
)

type DockerHandler struct {
	cfg       *config.Config
	store     store.Store
	audit     *audit.Logger
	dockerMgr *docker.DockerManager
}

func NewDockerHandler(cfg *config.Config, s store.Store, a *audit.Logger) *DockerHandler {
	return &DockerHandler{
		cfg:       cfg,
		store:     s,
		audit:     a,
		dockerMgr: docker.NewDockerManager(),
	}
}

// GetStatus checks engine version, containers count, and daemon state
func (h *DockerHandler) GetStatus(w http.ResponseWriter, r *http.Request) {
	status, err := h.dockerMgr.GetStatus()
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "DOCKER_STATUS_ERROR", err.Error(), nil, "")
		return
	}
	response.JSON(w, http.StatusOK, status, nil)
}

// ListContainers retrieves containers on host
func (h *DockerHandler) ListContainers(w http.ResponseWriter, r *http.Request) {
	all := r.URL.Query().Get("all") != "false"
	containers, err := h.dockerMgr.ListContainers(all)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "DOCKER_CONTAINERS_ERROR", err.Error(), nil, "")
		return
	}
	response.JSON(w, http.StatusOK, map[string]interface{}{
		"containers": containers,
		"count":      len(containers),
	}, nil)
}

// StartContainer starts a container
func (h *DockerHandler) StartContainer(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		response.Error(w, http.StatusBadRequest, "MISSING_ID", "Container ID is required", nil, "")
		return
	}

	err := h.dockerMgr.StartContainer(id)
	if err != nil {
		if errors.Is(err, docker.ErrDockerDaemonOffline) {
			response.Error(w, http.StatusServiceUnavailable, "DAEMON_OFFLINE", err.Error(), nil, "")
			return
		}
		if errors.Is(err, docker.ErrInvalidContainerIdent) {
			response.Error(w, http.StatusBadRequest, "INVALID_ID", err.Error(), nil, "")
			return
		}
		response.Error(w, http.StatusInternalServerError, "START_FAILED", err.Error(), nil, "")
		return
	}

	h.audit.Log(r.Context(), r, "docker.container_start", "docker", id, "success", "", nil)

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Container started successfully",
	}, nil)
}

// StopContainer stops a container
func (h *DockerHandler) StopContainer(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		response.Error(w, http.StatusBadRequest, "MISSING_ID", "Container ID is required", nil, "")
		return
	}

	err := h.dockerMgr.StopContainer(id)
	if err != nil {
		if errors.Is(err, docker.ErrDockerDaemonOffline) {
			response.Error(w, http.StatusServiceUnavailable, "DAEMON_OFFLINE", err.Error(), nil, "")
			return
		}
		if errors.Is(err, docker.ErrInvalidContainerIdent) {
			response.Error(w, http.StatusBadRequest, "INVALID_ID", err.Error(), nil, "")
			return
		}
		response.Error(w, http.StatusInternalServerError, "STOP_FAILED", err.Error(), nil, "")
		return
	}

	h.audit.Log(r.Context(), r, "docker.container_stop", "docker", id, "success", "", nil)

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Container stopped successfully",
	}, nil)
}

// RestartContainer restarts a container
func (h *DockerHandler) RestartContainer(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		response.Error(w, http.StatusBadRequest, "MISSING_ID", "Container ID is required", nil, "")
		return
	}

	err := h.dockerMgr.RestartContainer(id)
	if err != nil {
		if errors.Is(err, docker.ErrDockerDaemonOffline) {
			response.Error(w, http.StatusServiceUnavailable, "DAEMON_OFFLINE", err.Error(), nil, "")
			return
		}
		response.Error(w, http.StatusInternalServerError, "RESTART_FAILED", err.Error(), nil, "")
		return
	}

	h.audit.Log(r.Context(), r, "docker.container_restart", "docker", id, "success", "", nil)

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Container restarted successfully",
	}, nil)
}

// DeleteContainer removes a container
func (h *DockerHandler) DeleteContainer(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		response.Error(w, http.StatusBadRequest, "MISSING_ID", "Container ID is required", nil, "")
		return
	}

	force := r.URL.Query().Get("force") == "true"
	err := h.dockerMgr.RemoveContainer(id, force)
	if err != nil {
		if errors.Is(err, docker.ErrDockerDaemonOffline) {
			response.Error(w, http.StatusServiceUnavailable, "DAEMON_OFFLINE", err.Error(), nil, "")
			return
		}
		response.Error(w, http.StatusInternalServerError, "DELETE_FAILED", err.Error(), nil, "")
		return
	}

	h.audit.Log(r.Context(), r, "docker.container_delete", "docker", id, "success", "", map[string]interface{}{
		"force": force,
	})

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Container deleted successfully",
	}, nil)
}

// GetLogs returns container console output
func (h *DockerHandler) GetLogs(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		response.Error(w, http.StatusBadRequest, "MISSING_ID", "Container ID is required", nil, "")
		return
	}

	tail := 100
	if tailParam := r.URL.Query().Get("tail"); tailParam != "" {
		if t, err := strconv.Atoi(tailParam); err == nil && t > 0 {
			tail = t
		}
	}

	logs, err := h.dockerMgr.GetContainerLogs(id, tail)
	if err != nil {
		if errors.Is(err, docker.ErrDockerDaemonOffline) {
			response.Error(w, http.StatusServiceUnavailable, "DAEMON_OFFLINE", err.Error(), nil, "")
			return
		}
		response.Error(w, http.StatusInternalServerError, "LOGS_FAILED", err.Error(), nil, "")
		return
	}

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"id":   id,
		"logs": logs,
		"tail": tail,
	}, nil)
}

// GetStats returns current container resource usage stats
func (h *DockerHandler) GetStats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.dockerMgr.GetContainerStats()
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "STATS_FAILED", err.Error(), nil, "")
		return
	}
	response.JSON(w, http.StatusOK, map[string]interface{}{
		"stats": stats,
		"count": len(stats),
	}, nil)
}

// ListImages returns local Docker images
func (h *DockerHandler) ListImages(w http.ResponseWriter, r *http.Request) {
	images, err := h.dockerMgr.ListImages()
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "IMAGES_FAILED", err.Error(), nil, "")
		return
	}
	response.JSON(w, http.StatusOK, map[string]interface{}{
		"images": images,
		"count":  len(images),
	}, nil)
}

// DeleteImage deletes a local image
func (h *DockerHandler) DeleteImage(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		response.Error(w, http.StatusBadRequest, "MISSING_ID", "Image ID is required", nil, "")
		return
	}

	force := r.URL.Query().Get("force") == "true"
	err := h.dockerMgr.RemoveImage(id, force)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "DELETE_IMAGE_FAILED", err.Error(), nil, "")
		return
	}

	h.audit.Log(r.Context(), r, "docker.image_delete", "docker", id, "success", "", map[string]interface{}{
		"force": force,
	})

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Image deleted successfully",
	}, nil)
}

// RunContainer launches a new container instance
func (h *DockerHandler) RunContainer(w http.ResponseWriter, r *http.Request) {
	var req docker.RunContainerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_BODY", "Invalid JSON request body", nil, "")
		return
	}

	if req.Image == "" {
		response.Error(w, http.StatusBadRequest, "MISSING_IMAGE", "Image name is required", nil, "")
		return
	}

	containerID, err := h.dockerMgr.RunContainer(req)
	if err != nil {
		if errors.Is(err, docker.ErrDockerDaemonOffline) {
			response.Error(w, http.StatusServiceUnavailable, "DAEMON_OFFLINE", err.Error(), nil, "")
			return
		}
		response.Error(w, http.StatusInternalServerError, "RUN_FAILED", err.Error(), nil, "")
		return
	}

	h.audit.Log(r.Context(), r, "docker.container_run", "docker", containerID, "success", "", map[string]interface{}{
		"image": req.Image,
		"name":  req.Name,
	})

	response.JSON(w, http.StatusCreated, map[string]interface{}{
		"success":      true,
		"container_id": containerID,
		"message":      "Container successfully created and started",
	}, nil)
}

// PruneSystem cleans unused Docker data
func (h *DockerHandler) PruneSystem(w http.ResponseWriter, r *http.Request) {
	out, err := h.dockerMgr.PruneSystem()
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "PRUNE_FAILED", err.Error(), nil, "")
		return
	}

	h.audit.Log(r.Context(), r, "docker.prune", "docker", "system", "success", "", nil)

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"output":  out,
		"message": "Docker system pruned successfully",
	}, nil)
}
