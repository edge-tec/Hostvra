'use client';

import React, { useState, useEffect, useCallback } from 'react';
import {
  Container,
  Play,
  Square,
  RotateCw,
  FileText,
  RefreshCw,
  X,
  CheckCircle2,
  AlertCircle,
  Plus,
  Trash2,
  Search,
  Cpu,
  Layers,
  HardDrive,
  Activity,
  Terminal,
  ExternalLink,
} from 'lucide-react';
import { DashboardShell } from '@/components/DashboardShell';
import {
  apiFetch,
  DockerStatus,
  DockerContainer,
  DockerImage,
  DockerStats,
} from '@/lib/api';

export default function DockerPage() {
  const [activeTab, setActiveTab] = useState<'containers' | 'stats' | 'images'>('containers');

  // Server Data
  const [status, setStatus] = useState<DockerStatus | null>(null);
  const [containers, setContainers] = useState<DockerContainer[]>([]);
  const [images, setImages] = useState<DockerImage[]>([]);
  const [stats, setStats] = useState<DockerStats[]>([]);
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const [search, setSearch] = useState('');

  // Alerts
  const [errorMsg, setErrorMsg] = useState<string | null>(null);
  const [successMsg, setSuccessMsg] = useState<string | null>(null);

  // Action Loading
  const [actionLoadingId, setActionLoadingId] = useState<string | null>(null);

  // Logs Modal
  const [logsModalOpen, setLogsModalOpen] = useState(false);
  const [activeContainer, setActiveContainer] = useState<DockerContainer | null>(null);
  const [logsContent, setLogsContent] = useState('');
  const [logsLoading, setLogsLoading] = useState(false);

  // Run Container Modal
  const [runModalOpen, setRunModalOpen] = useState(false);
  const [newImage, setNewImage] = useState('');
  const [newName, setNewName] = useState('');
  const [newPorts, setNewPorts] = useState('');
  const [newRestartPolicy, setNewRestartPolicy] = useState('unless-stopped');
  const [submittingRun, setSubmittingRun] = useState(false);

  const showNotification = (msg: string, isError = false) => {
    if (isError) {
      setErrorMsg(msg);
      setTimeout(() => setErrorMsg(null), 7000);
    } else {
      setSuccessMsg(msg);
      setTimeout(() => setSuccessMsg(null), 5000);
    }
  };

  const loadData = useCallback(async (showIndicator = true) => {
    if (showIndicator) setRefreshing(true);
    setErrorMsg(null);

    try {
      // 1. Status
      const statusRes = await apiFetch<DockerStatus>('/api/v1/docker/status');
      if (statusRes.success && statusRes.data) {
        setStatus(statusRes.data);
      }

      // 2. Containers
      const containersRes = await apiFetch<{ containers: DockerContainer[]; count: number }>(
        '/api/v1/docker/containers?all=true'
      );
      if (containersRes.success && containersRes.data) {
        setContainers(containersRes.data.containers || []);
      }

      // 3. Images
      const imagesRes = await apiFetch<{ images: DockerImage[]; count: number }>('/api/v1/docker/images');
      if (imagesRes.success && imagesRes.data) {
        setImages(imagesRes.data.images || []);
      }

      // 4. Stats
      const statsRes = await apiFetch<{ stats: DockerStats[]; count: number }>('/api/v1/docker/stats');
      if (statsRes.success && statsRes.data) {
        setStats(statsRes.data.stats || []);
      }
    } catch (err: any) {
      showNotification(err.message || 'Failed to fetch Docker engine data', true);
    } finally {
      setLoading(false);
      setRefreshing(false);
    }
  }, []);

  useEffect(() => {
    loadData(false);
  }, [loadData]);

  // Container Lifecycle Actions (start, stop, restart)
  const handleContainerAction = async (id: string, action: 'start' | 'stop' | 'restart') => {
    setActionLoadingId(`${action}-${id}`);
    const res = await apiFetch<any>(`/api/v1/docker/containers/${id}/${action}`, {
      method: 'POST',
    });
    setActionLoadingId(null);

    if (res.success) {
      showNotification(`Container ${action}ed successfully`);
      loadData(false);
    } else {
      showNotification(res.error?.message || `Failed to ${action} container`, true);
    }
  };

  // Delete Container
  const handleDeleteContainer = async (c: DockerContainer) => {
    if (!window.confirm(`Are you sure you want to remove container "${c.names || c.id}"?`)) {
      return;
    }

    setActionLoadingId(`delete-${c.id}`);
    const res = await apiFetch<any>(`/api/v1/docker/containers/${c.id}?force=true`, {
      method: 'DELETE',
    });
    setActionLoadingId(null);

    if (res.success) {
      showNotification('Container removed successfully');
      loadData(false);
    } else {
      showNotification(res.error?.message || 'Failed to remove container', true);
    }
  };

  // Open Logs
  const handleOpenLogs = async (c: DockerContainer) => {
    setActiveContainer(c);
    setLogsModalOpen(true);
    setLogsLoading(true);
    setLogsContent('');

    const res = await apiFetch<{ id: string; logs: string }>(`/api/v1/docker/containers/${c.id}/logs?tail=200`);
    setLogsLoading(false);

    if (res.success && res.data) {
      setLogsContent(res.data.logs || '(No console logs output produced by container)');
    } else {
      setLogsContent(`Failed to retrieve logs: ${res.error?.message || 'Unknown error'}`);
    }
  };

  // Delete Image
  const handleDeleteImage = async (img: DockerImage) => {
    const label = `${img.repository}:${img.tag}`;
    if (!window.confirm(`Delete image "${label}"?`)) return;

    setActionLoadingId(`del-img-${img.id}`);
    const res = await apiFetch<any>(`/api/v1/docker/images/${img.id}?force=true`, {
      method: 'DELETE',
    });
    setActionLoadingId(null);

    if (res.success) {
      showNotification(`Image "${label}" deleted successfully`);
      loadData(false);
    } else {
      showNotification(res.error?.message || 'Failed to delete image', true);
    }
  };

  // Run Container
  const handleRunContainer = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!newImage.trim()) {
      showNotification('Docker image name is required', true);
      return;
    }

    setSubmittingRun(true);
    const portsList = newPorts
      .split(',')
      .map((p) => p.trim())
      .filter((p) => p.length > 0);

    const res = await apiFetch<any>('/api/v1/docker/containers/run', {
      method: 'POST',
      body: JSON.stringify({
        image: newImage.trim(),
        name: newName.trim() || undefined,
        port_mappings: portsList.length > 0 ? portsList : undefined,
        restart_policy: newRestartPolicy,
      }),
    });
    setSubmittingRun(false);

    if (res.success) {
      showNotification('Container launched successfully');
      setRunModalOpen(false);
      setNewImage('');
      setNewName('');
      setNewPorts('');
      loadData(false);
    } else {
      showNotification(res.error?.message || 'Failed to run container', true);
    }
  };

  // Prune System
  const handlePruneSystem = async () => {
    if (!window.confirm('Prune unused Docker containers, networks, and dangling images?')) return;

    setActionLoadingId('prune');
    const res = await apiFetch<any>('/api/v1/docker/prune', { method: 'POST' });
    setActionLoadingId(null);

    if (res.success) {
      showNotification('Docker system pruned successfully');
      loadData(false);
    } else {
      showNotification(res.error?.message || 'Prune failed', true);
    }
  };

  const filteredContainers = containers.filter(
    (c) =>
      c.names.toLowerCase().includes(search.toLowerCase()) ||
      c.image.toLowerCase().includes(search.toLowerCase()) ||
      c.ports.toLowerCase().includes(search.toLowerCase()) ||
      c.state.toLowerCase().includes(search.toLowerCase())
  );

  const filteredImages = images.filter(
    (img) =>
      img.repository.toLowerCase().includes(search.toLowerCase()) ||
      img.tag.toLowerCase().includes(search.toLowerCase()) ||
      img.id.toLowerCase().includes(search.toLowerCase())
  );

  return (
    <DashboardShell>
      <div className="space-y-6 animate-fadeIn max-w-7xl mx-auto pb-16">
        {/* Alerts */}
        {errorMsg && (
          <div className="p-4 rounded-xl bg-rose-50 dark:bg-rose-950/40 border border-rose-200 dark:border-rose-900/50 flex items-start gap-3 text-rose-800 dark:text-rose-200 text-sm shadow-xs">
            <AlertCircle className="w-5 h-5 flex-shrink-0 text-rose-600 dark:text-rose-400 mt-0.5" />
            <div className="flex-1 font-medium">{errorMsg}</div>
            <button onClick={() => setErrorMsg(null)} className="text-rose-600 dark:text-rose-400 hover:opacity-75">
              <X className="w-4 h-4" />
            </button>
          </div>
        )}

        {successMsg && (
          <div className="p-4 rounded-xl bg-emerald-50 dark:bg-emerald-950/40 border border-emerald-200 dark:border-emerald-900/50 flex items-start gap-3 text-emerald-800 dark:text-emerald-200 text-sm shadow-xs">
            <CheckCircle2 className="w-5 h-5 flex-shrink-0 text-emerald-600 dark:text-emerald-400 mt-0.5" />
            <div className="flex-1 font-medium">{successMsg}</div>
            <button onClick={() => setSuccessMsg(null)} className="text-emerald-600 dark:text-emerald-400 hover:opacity-75">
              <X className="w-4 h-4" />
            </button>
          </div>
        )}

        {/* Header */}
        <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
          <div>
            <div className="flex items-center gap-3">
              <h1 className="text-2xl font-bold tracking-tight text-slate-950 dark:text-white">
                Docker Engine & Containers
              </h1>
              <span className="px-2.5 py-0.5 rounded-full text-xs font-mono font-semibold uppercase bg-indigo-50 dark:bg-indigo-950/60 text-indigo-700 dark:text-indigo-400 border border-indigo-200 dark:border-indigo-800">
                {status?.server_version ? `v${status.server_version}` : 'Docker Engine'}
              </span>
            </div>
            <p className="text-sm text-slate-600 dark:text-slate-400 mt-1">
              Production container virtualization, container lifecycle, and resource utilization.
            </p>
          </div>

          <div className="flex items-center gap-2">
            <button
              onClick={() => loadData(true)}
              disabled={refreshing}
              className="p-2.5 rounded-xl border border-slate-300 dark:border-surface-700 bg-white dark:bg-surface-900 text-slate-700 dark:text-slate-300 hover:bg-slate-50 dark:hover:bg-surface-800 transition-colors shadow-xs"
              title="Refresh Docker Data"
            >
              <RefreshCw className={`w-4 h-4 ${refreshing ? 'animate-spin text-indigo-500' : ''}`} />
            </button>

            <button
              onClick={handlePruneSystem}
              disabled={actionLoadingId === 'prune'}
              className="px-3.5 py-2.5 rounded-xl border border-slate-300 dark:border-surface-700 bg-white dark:bg-surface-900 text-slate-700 dark:text-slate-300 hover:bg-slate-50 dark:hover:bg-surface-800 text-xs font-semibold shadow-xs transition-all"
            >
              {actionLoadingId === 'prune' ? 'Pruning...' : 'Prune Unused'}
            </button>

            <button
              onClick={() => setRunModalOpen(true)}
              className="inline-flex items-center gap-2 px-4 py-2.5 rounded-xl bg-indigo-600 hover:bg-indigo-500 text-white text-sm font-semibold shadow-md shadow-indigo-600/20 transition-all"
            >
              <Plus className="w-4 h-4" />
              Run Container
            </button>
          </div>
        </div>

        {/* Daemon Offline Notice */}
        {status && !status.is_daemon_running && (
          <div className="p-5 rounded-2xl bg-amber-50 dark:bg-amber-950/30 border border-amber-200 dark:border-amber-900/50 flex flex-col sm:flex-row items-start sm:items-center justify-between gap-4">
            <div className="flex items-start gap-3">
              <AlertCircle className="w-5 h-5 text-amber-600 dark:text-amber-400 flex-shrink-0 mt-0.5" />
              <div>
                <h2 className="text-sm font-bold text-amber-950 dark:text-amber-200">
                  {status.is_installed ? 'Docker Daemon Is Not Running' : 'Docker Is Not Installed On This Host'}
                </h2>
                <p className="text-xs text-amber-800 dark:text-amber-300 mt-0.5">
                  {status.is_installed
                    ? 'Start the engine with `systemctl start docker` or click Run Fix in Dashboard Telemetry.'
                    : 'Install Docker via the 1-Click App Store or run `curl -fsSL https://get.docker.com | sh`.'}
                </p>
              </div>
            </div>
            <a
              href="/app-store"
              className="inline-flex items-center gap-1.5 px-4 py-2 rounded-xl bg-amber-600 hover:bg-amber-500 text-white text-xs font-semibold shadow-xs transition-all flex-shrink-0"
            >
              <ExternalLink className="w-3.5 h-3.5" />
              Open App Store
            </a>
          </div>
        )}

        {/* Engine Status Cards */}
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          <div className="p-5 rounded-2xl bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-800 shadow-xs flex items-center gap-3.5">
            <div
              className={`w-11 h-11 rounded-xl flex items-center justify-center ${
                status?.is_daemon_running
                  ? 'bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 border border-emerald-500/20'
                  : 'bg-amber-500/10 text-amber-600 dark:text-amber-400 border border-amber-500/20'
              }`}
            >
              <Container className="w-6 h-6" />
            </div>
            <div>
              <div className="flex items-center gap-2">
                <span className="font-bold text-slate-950 dark:text-white text-sm">
                  {status?.is_daemon_running ? 'Engine Online' : 'Engine Offline'}
                </span>
                <span
                  className={`w-2 h-2 rounded-full ${
                    status?.is_daemon_running ? 'bg-emerald-500 animate-pulse' : 'bg-amber-500'
                  }`}
                />
              </div>
              <p className="text-xs text-slate-500 dark:text-slate-400 mt-0.5 font-medium">
                Driver: {status?.storage_driver || 'overlay2'}
              </p>
            </div>
          </div>

          <div className="p-5 rounded-2xl bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-800 shadow-xs flex items-center gap-3.5">
            <div className="w-11 h-11 rounded-xl bg-indigo-500/10 text-indigo-600 dark:text-indigo-400 border border-indigo-500/20 flex items-center justify-center">
              <Activity className="w-6 h-6" />
            </div>
            <div>
              <div className="flex items-center gap-2">
                <span className="font-bold text-slate-950 dark:text-white text-sm">Containers</span>
                <span className="px-2 py-0.5 rounded-full text-[10px] font-bold bg-indigo-500/10 text-indigo-600 dark:text-indigo-400 border border-indigo-500/20">
                  {containers.length} Total
                </span>
              </div>
              <p className="text-xs text-slate-500 dark:text-slate-400 mt-0.5 font-medium">
                {containers.filter((c) => c.state === 'running').length} Running •{' '}
                {containers.filter((c) => c.state !== 'running').length} Stopped
              </p>
            </div>
          </div>

          <div className="p-5 rounded-2xl bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-800 shadow-xs flex items-center gap-3.5">
            <div className="w-11 h-11 rounded-xl bg-purple-500/10 text-purple-600 dark:text-purple-400 border border-purple-500/20 flex items-center justify-center">
              <Layers className="w-6 h-6" />
            </div>
            <div>
              <div className="flex items-center gap-2">
                <span className="font-bold text-slate-950 dark:text-white text-sm">Local Images</span>
                <span className="px-2 py-0.5 rounded-full text-[10px] font-bold bg-purple-500/10 text-purple-600 dark:text-purple-400 border border-purple-500/20">
                  {images.length} Cached
                </span>
              </div>
              <p className="text-xs text-slate-500 dark:text-slate-400 mt-0.5 font-medium">
                Stored in local host Docker registry
              </p>
            </div>
          </div>
        </div>

        {/* Tab Navigation */}
        <div className="flex border-b border-slate-200 dark:border-surface-800">
          <button
            onClick={() => setActiveTab('containers')}
            className={`px-5 py-3 text-sm font-semibold border-b-2 transition-all flex items-center gap-2 ${
              activeTab === 'containers'
                ? 'border-indigo-600 text-indigo-600 dark:text-indigo-400'
                : 'border-transparent text-slate-600 dark:text-slate-400 hover:text-slate-900 dark:hover:text-white'
            }`}
          >
            <Container className="w-4 h-4" />
            Containers ({containers.length})
          </button>
          <button
            onClick={() => setActiveTab('stats')}
            className={`px-5 py-3 text-sm font-semibold border-b-2 transition-all flex items-center gap-2 ${
              activeTab === 'stats'
                ? 'border-indigo-600 text-indigo-600 dark:text-indigo-400'
                : 'border-transparent text-slate-600 dark:text-slate-400 hover:text-slate-900 dark:hover:text-white'
            }`}
          >
            <Cpu className="w-4 h-4" />
            Live Resource Stats ({stats.length})
          </button>
          <button
            onClick={() => setActiveTab('images')}
            className={`px-5 py-3 text-sm font-semibold border-b-2 transition-all flex items-center gap-2 ${
              activeTab === 'images'
                ? 'border-indigo-600 text-indigo-600 dark:text-indigo-400'
                : 'border-transparent text-slate-600 dark:text-slate-400 hover:text-slate-900 dark:hover:text-white'
            }`}
          >
            <Layers className="w-4 h-4" />
            Images ({images.length})
          </button>
        </div>

        {/* Search Bar */}
        <div className="flex flex-col sm:flex-row items-center justify-between gap-4 bg-white dark:bg-[#10141d] border border-slate-200 dark:border-surface-800 rounded-2xl p-3 shadow-xs">
          <div className="flex items-center gap-3 w-full sm:w-80 bg-slate-50 dark:bg-[#121824] border border-slate-300 dark:border-surface-700 rounded-xl px-3.5 py-2 shadow-xs focus-within:border-indigo-500 focus-within:ring-2 focus-within:ring-indigo-500/20 transition-all">
            <Search className="w-4 h-4 text-slate-400 flex-shrink-0" />
            <input
              type="text"
              placeholder={activeTab === 'images' ? 'Search images or tags...' : 'Search container name, image, or ports...'}
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              className="w-full bg-transparent text-xs font-medium text-slate-950 dark:text-white placeholder:text-slate-400 focus:outline-none"
            />
            {search && (
              <button onClick={() => setSearch('')} className="text-slate-400 hover:text-slate-600 dark:hover:text-white p-0.5">
                <X className="w-3.5 h-3.5" />
              </button>
            )}
          </div>
          <div className="text-xs font-semibold text-slate-600 dark:text-slate-400 px-2 self-end sm:self-center">
            {activeTab === 'containers'
              ? `Showing ${filteredContainers.length} of ${containers.length} containers`
              : activeTab === 'images'
              ? `Showing ${filteredImages.length} of ${images.length} images`
              : `Tracking ${stats.length} active containers`}
          </div>
        </div>

        {/* TAB 1: CONTAINERS TABLE */}
        {activeTab === 'containers' && (
          <div className="bg-white dark:bg-[#10141d] border border-slate-200 dark:border-surface-800 rounded-2xl overflow-hidden shadow-xs">
            <div className="overflow-x-auto">
              <table className="w-full text-left text-sm border-collapse">
                <thead>
                  <tr className="border-b border-slate-200 dark:border-surface-800 bg-slate-50 dark:bg-[#121824] text-slate-700 dark:text-slate-300 text-[11px] font-bold uppercase tracking-wider">
                    <th className="px-6 py-3.5">Container Name</th>
                    <th className="px-6 py-3.5">Image</th>
                    <th className="px-6 py-3.5">State</th>
                    <th className="px-6 py-3.5">Port Mappings</th>
                    <th className="px-6 py-3.5">Status</th>
                    <th className="px-6 py-3.5 text-right">Actions</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-slate-200 dark:divide-surface-800/80">
                  {loading ? (
                    <tr>
                      <td colSpan={6} className="px-6 py-12 text-center text-slate-500 font-medium text-sm">
                        <RefreshCw className="w-6 h-6 animate-spin mx-auto mb-2 text-indigo-500" />
                        Querying Docker engine containers...
                      </td>
                    </tr>
                  ) : filteredContainers.length === 0 ? (
                    <tr>
                      <td colSpan={6} className="px-6 py-12 text-center text-slate-500 font-medium text-sm">
                        No Docker containers found on this server. Click "Run Container" to deploy one.
                      </td>
                    </tr>
                  ) : (
                    filteredContainers.map((c) => (
                      <tr key={c.id} className="hover:bg-slate-50/80 dark:hover:bg-[#151d2d] transition-colors">
                        <td className="px-6 py-4 font-bold text-slate-950 dark:text-white flex items-center gap-2.5">
                          <Container className="w-4 h-4 text-indigo-500 flex-shrink-0" />
                          <span className="truncate max-w-xs">{c.names || c.id.slice(0, 12)}</span>
                        </td>
                        <td className="px-6 py-4 font-mono text-xs text-slate-600 dark:text-slate-300">
                          {c.image}
                        </td>
                        <td className="px-6 py-4">
                          <span
                            className={`inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full text-[11px] font-bold uppercase ${
                              c.state === 'running'
                                ? 'bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 border border-emerald-500/20'
                                : c.state === 'paused'
                                ? 'bg-amber-500/10 text-amber-600 dark:text-amber-400 border border-amber-500/20'
                                : 'bg-slate-100 dark:bg-surface-800 text-slate-500 border border-slate-300 dark:border-surface-700'
                            }`}
                          >
                            <span
                              className={`w-1.5 h-1.5 rounded-full ${
                                c.state === 'running' ? 'bg-emerald-500 animate-pulse' : 'bg-slate-400'
                              }`}
                            />
                            {c.state}
                          </span>
                        </td>
                        <td className="px-6 py-4 font-mono text-xs text-slate-700 dark:text-slate-300">
                          {c.ports || '—'}
                        </td>
                        <td className="px-6 py-4 text-xs text-slate-500 dark:text-slate-400 font-medium">
                          {c.status}
                        </td>
                        <td className="px-6 py-4 text-right">
                          <div className="flex items-center justify-end gap-1.5">
                            {c.state === 'running' ? (
                              <button
                                onClick={() => handleContainerAction(c.id, 'stop')}
                                disabled={actionLoadingId === `stop-${c.id}`}
                                title="Stop Container"
                                className="p-1.5 rounded-lg border border-slate-300 dark:border-surface-700 text-slate-600 dark:text-slate-400 hover:text-amber-600 dark:hover:text-amber-400 hover:bg-amber-50 dark:hover:bg-amber-950/30 transition-colors"
                              >
                                {actionLoadingId === `stop-${c.id}` ? (
                                  <RefreshCw className="w-3.5 h-3.5 animate-spin" />
                                ) : (
                                  <Square className="w-3.5 h-3.5" />
                                )}
                              </button>
                            ) : (
                              <button
                                onClick={() => handleContainerAction(c.id, 'start')}
                                disabled={actionLoadingId === `start-${c.id}`}
                                title="Start Container"
                                className="p-1.5 rounded-lg border border-slate-300 dark:border-surface-700 text-slate-600 dark:text-slate-400 hover:text-emerald-600 dark:hover:text-emerald-400 hover:bg-emerald-50 dark:hover:bg-emerald-950/30 transition-colors"
                              >
                                {actionLoadingId === `start-${c.id}` ? (
                                  <RefreshCw className="w-3.5 h-3.5 animate-spin text-emerald-500" />
                                ) : (
                                  <Play className="w-3.5 h-3.5" />
                                )}
                              </button>
                            )}

                            <button
                              onClick={() => handleContainerAction(c.id, 'restart')}
                              disabled={actionLoadingId === `restart-${c.id}`}
                              title="Restart Container"
                              className="p-1.5 rounded-lg border border-slate-300 dark:border-surface-700 text-slate-600 dark:text-slate-400 hover:text-indigo-600 dark:hover:text-indigo-400 hover:bg-indigo-50 dark:hover:bg-indigo-950/30 transition-colors"
                            >
                              {actionLoadingId === `restart-${c.id}` ? (
                                <RefreshCw className="w-3.5 h-3.5 animate-spin text-indigo-500" />
                              ) : (
                                <RotateCw className="w-3.5 h-3.5" />
                              )}
                            </button>

                            <button
                              onClick={() => handleOpenLogs(c)}
                              title="View Logs"
                              className="p-1.5 rounded-lg border border-slate-300 dark:border-surface-700 text-slate-600 dark:text-slate-400 hover:text-indigo-600 dark:hover:text-indigo-400 hover:bg-indigo-50 dark:hover:bg-indigo-950/30 transition-colors"
                            >
                              <FileText className="w-3.5 h-3.5" />
                            </button>

                            <button
                              onClick={() => handleDeleteContainer(c)}
                              disabled={actionLoadingId === `delete-${c.id}`}
                              title="Delete Container"
                              className="p-1.5 rounded-lg border border-slate-300 dark:border-surface-700 text-slate-600 dark:text-slate-400 hover:text-rose-600 dark:hover:text-rose-400 hover:bg-rose-50 dark:hover:bg-rose-950/30 transition-colors"
                            >
                              <Trash2 className="w-3.5 h-3.5" />
                            </button>
                          </div>
                        </td>
                      </tr>
                    ))
                  )}
                </tbody>
              </table>
            </div>
          </div>
        )}

        {/* TAB 2: LIVE RESOURCE STATS */}
        {activeTab === 'stats' && (
          <div className="bg-white dark:bg-[#10141d] border border-slate-200 dark:border-surface-800 rounded-2xl overflow-hidden shadow-xs">
            <div className="overflow-x-auto">
              <table className="w-full text-left text-sm border-collapse">
                <thead>
                  <tr className="border-b border-slate-200 dark:border-surface-800 bg-slate-50 dark:bg-[#121824] text-slate-700 dark:text-slate-300 text-[11px] font-bold uppercase tracking-wider">
                    <th className="px-6 py-3.5">Container</th>
                    <th className="px-6 py-3.5">CPU Usage</th>
                    <th className="px-6 py-3.5">Memory Usage / Limit</th>
                    <th className="px-6 py-3.5">Memory %</th>
                    <th className="px-6 py-3.5">Network I/O</th>
                    <th className="px-6 py-3.5">Block I/O</th>
                    <th className="px-6 py-3.5">PIDs</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-slate-200 dark:divide-surface-800/80">
                  {stats.length === 0 ? (
                    <tr>
                      <td colSpan={7} className="px-6 py-12 text-center text-slate-500 font-medium text-sm">
                        No running containers currently streaming resource stats.
                      </td>
                    </tr>
                  ) : (
                    stats.map((s) => (
                      <tr key={s.id} className="hover:bg-slate-50/80 dark:hover:bg-[#151d2d] transition-colors">
                        <td className="px-6 py-4 font-mono font-bold text-slate-950 dark:text-white">
                          {s.name}
                        </td>
                        <td className="px-6 py-4 font-mono text-xs font-semibold text-indigo-600 dark:text-indigo-400">
                          {s.cpu_perc}
                        </td>
                        <td className="px-6 py-4 font-mono text-xs text-slate-700 dark:text-slate-300">
                          {s.mem_usage}
                        </td>
                        <td className="px-6 py-4 font-mono text-xs font-semibold text-purple-600 dark:text-purple-400">
                          {s.mem_perc}
                        </td>
                        <td className="px-6 py-4 font-mono text-xs text-slate-600 dark:text-slate-400">
                          {s.net_io}
                        </td>
                        <td className="px-6 py-4 font-mono text-xs text-slate-600 dark:text-slate-400">
                          {s.block_io}
                        </td>
                        <td className="px-6 py-4 font-mono text-xs text-slate-600 dark:text-slate-400">
                          {s.pids}
                        </td>
                      </tr>
                    ))
                  )}
                </tbody>
              </table>
            </div>
          </div>
        )}

        {/* TAB 3: DOCKER IMAGES */}
        {activeTab === 'images' && (
          <div className="bg-white dark:bg-[#10141d] border border-slate-200 dark:border-surface-800 rounded-2xl overflow-hidden shadow-xs">
            <div className="overflow-x-auto">
              <table className="w-full text-left text-sm border-collapse">
                <thead>
                  <tr className="border-b border-slate-200 dark:border-surface-800 bg-slate-50 dark:bg-[#121824] text-slate-700 dark:text-slate-300 text-[11px] font-bold uppercase tracking-wider">
                    <th className="px-6 py-3.5">Repository</th>
                    <th className="px-6 py-3.5">Tag</th>
                    <th className="px-6 py-3.5">Image ID</th>
                    <th className="px-6 py-3.5">Virtual Size</th>
                    <th className="px-6 py-3.5">Created</th>
                    <th className="px-6 py-3.5 text-right">Actions</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-slate-200 dark:divide-surface-800/80">
                  {filteredImages.length === 0 ? (
                    <tr>
                      <td colSpan={6} className="px-6 py-12 text-center text-slate-500 font-medium text-sm">
                        No Docker images cached locally. Run a container to pull images automatically.
                      </td>
                    </tr>
                  ) : (
                    filteredImages.map((img) => (
                      <tr key={img.id} className="hover:bg-slate-50/80 dark:hover:bg-[#151d2d] transition-colors">
                        <td className="px-6 py-4 font-mono font-bold text-slate-950 dark:text-white flex items-center gap-2">
                          <Layers className="w-4 h-4 text-purple-500" />
                          <span>{img.repository}</span>
                        </td>
                        <td className="px-6 py-4">
                          <span className="px-2 py-0.5 rounded-md font-mono text-xs font-semibold bg-slate-100 dark:bg-surface-800 text-slate-700 dark:text-slate-300 border border-slate-200 dark:border-surface-700">
                            {img.tag}
                          </span>
                        </td>
                        <td className="px-6 py-4 font-mono text-xs text-slate-500">
                          {img.id.slice(0, 12)}
                        </td>
                        <td className="px-6 py-4 font-mono text-xs font-semibold text-slate-800 dark:text-slate-300">
                          {img.size}
                        </td>
                        <td className="px-6 py-4 text-xs text-slate-500 font-medium">
                          {img.created_at}
                        </td>
                        <td className="px-6 py-4 text-right">
                          <button
                            onClick={() => handleDeleteImage(img)}
                            disabled={actionLoadingId === `del-img-${img.id}`}
                            className="p-1.5 rounded-lg border border-slate-300 dark:border-surface-700 text-slate-600 dark:text-slate-400 hover:text-rose-600 dark:hover:text-rose-400 hover:bg-rose-50 dark:hover:bg-rose-950/30 transition-colors"
                            title="Delete Image"
                          >
                            <Trash2 className="w-3.5 h-3.5" />
                          </button>
                        </td>
                      </tr>
                    ))
                  )}
                </tbody>
              </table>
            </div>
          </div>
        )}

        {/* Modal: View Container Logs */}
        {logsModalOpen && activeContainer && (
          <div className="fixed inset-0 z-50 bg-black/70 backdrop-blur-sm flex items-center justify-center p-4">
            <div className="w-full max-w-3xl bg-white dark:bg-[#10141d] border border-slate-200 dark:border-surface-700 rounded-2xl shadow-2xl p-6 relative">
              <button
                onClick={() => setLogsModalOpen(false)}
                className="absolute top-5 right-5 p-1 rounded-lg text-slate-400 hover:text-slate-700 dark:hover:text-white transition-colors"
              >
                <X className="w-5 h-5" />
              </button>

              <div className="flex items-center justify-between gap-4 mb-4 pr-10">
                <div className="flex items-center gap-3">
                  <div className="w-10 h-10 rounded-xl bg-indigo-500/10 border border-indigo-500/20 flex items-center justify-center text-indigo-600 dark:text-indigo-400">
                    <Terminal className="w-5 h-5" />
                  </div>
                  <div>
                    <h2 className="text-lg font-bold text-slate-950 dark:text-white">
                      {activeContainer.names || activeContainer.id}
                    </h2>
                    <p className="text-xs font-mono text-slate-500 dark:text-slate-400">
                      Image: {activeContainer.image} • Status: {activeContainer.status}
                    </p>
                  </div>
                </div>

                <button
                  onClick={() => handleOpenLogs(activeContainer)}
                  disabled={logsLoading}
                  className="p-2 rounded-lg border border-slate-300 dark:border-surface-700 hover:bg-slate-100 dark:hover:bg-surface-800 text-slate-600 dark:text-slate-300 transition-colors"
                  title="Refresh Logs"
                >
                  <RefreshCw className={`w-3.5 h-3.5 ${logsLoading ? 'animate-spin' : ''}`} />
                </button>
              </div>

              <div>
                <pre className="p-4 rounded-xl bg-slate-950 text-slate-100 font-mono text-xs overflow-x-auto max-h-96 whitespace-pre-wrap">
                  {logsLoading ? 'Fetching latest container stdout/stderr...' : logsContent}
                </pre>
              </div>

              <div className="flex justify-end pt-4">
                <button
                  onClick={() => setLogsModalOpen(false)}
                  className="px-5 py-2 rounded-xl bg-indigo-600 hover:bg-indigo-500 text-white text-xs font-semibold shadow-md transition-all"
                >
                  Close Console
                </button>
              </div>
            </div>
          </div>
        )}

        {/* Modal: Run Container */}
        {runModalOpen && (
          <div className="fixed inset-0 z-50 bg-black/70 backdrop-blur-sm flex items-center justify-center p-4">
            <div className="w-full max-w-lg bg-white dark:bg-[#10141d] border border-slate-200 dark:border-surface-700 rounded-2xl shadow-2xl p-6 relative">
              <button
                onClick={() => setRunModalOpen(false)}
                className="absolute top-5 right-5 p-1 rounded-lg text-slate-400 hover:text-slate-700 dark:hover:text-white transition-colors"
              >
                <X className="w-5 h-5" />
              </button>

              <div className="flex items-center gap-3 mb-5">
                <div className="w-10 h-10 rounded-xl bg-indigo-500/10 border border-indigo-500/20 flex items-center justify-center text-indigo-600 dark:text-indigo-400">
                  <Container className="w-5 h-5" />
                </div>
                <div>
                  <h2 className="text-lg font-bold text-slate-950 dark:text-white">Run Docker Container</h2>
                  <p className="text-xs text-slate-500 dark:text-slate-400">
                    Deploy container from Docker Hub or local registry
                  </p>
                </div>
              </div>

              <form onSubmit={handleRunContainer} className="space-y-4">
                <div>
                  <label className="block text-xs font-semibold uppercase tracking-wider text-slate-700 dark:text-slate-300 mb-1.5">
                    Docker Image Name *
                  </label>
                  <input
                    type="text"
                    required
                    placeholder="e.g. redis:7-alpine, postgres:16, nginx:alpine"
                    value={newImage}
                    onChange={(e) => setNewImage(e.target.value)}
                    className="w-full px-3.5 py-2.5 rounded-xl bg-slate-50 dark:bg-surface-950 border border-slate-300 dark:border-surface-700 text-slate-950 dark:text-slate-100 font-mono text-sm focus:outline-none focus:border-indigo-500"
                  />
                </div>

                <div className="grid grid-cols-2 gap-3">
                  <div>
                    <label className="block text-xs font-semibold uppercase tracking-wider text-slate-700 dark:text-slate-300 mb-1.5">
                      Container Name (Optional)
                    </label>
                    <input
                      type="text"
                      placeholder="e.g. my-redis"
                      value={newName}
                      onChange={(e) => setNewName(e.target.value)}
                      className="w-full px-3.5 py-2.5 rounded-xl bg-slate-50 dark:bg-surface-950 border border-slate-300 dark:border-surface-700 text-slate-950 dark:text-slate-100 font-mono text-sm focus:outline-none focus:border-indigo-500"
                    />
                  </div>

                  <div>
                    <label className="block text-xs font-semibold uppercase tracking-wider text-slate-700 dark:text-slate-300 mb-1.5">
                      Restart Policy
                    </label>
                    <select
                      value={newRestartPolicy}
                      onChange={(e) => setNewRestartPolicy(e.target.value)}
                      className="w-full px-3.5 py-2.5 rounded-xl bg-slate-50 dark:bg-surface-950 border border-slate-300 dark:border-surface-700 text-slate-950 dark:text-slate-100 text-sm focus:outline-none focus:border-indigo-500 font-mono"
                    >
                      <option value="unless-stopped">unless-stopped</option>
                      <option value="always">always</option>
                      <option value="on-failure">on-failure</option>
                      <option value="no">no</option>
                    </select>
                  </div>
                </div>

                <div>
                  <label className="block text-xs font-semibold uppercase tracking-wider text-slate-700 dark:text-slate-300 mb-1.5">
                    Port Mappings (Comma separated, e.g. 8080:80, 6379:6379)
                  </label>
                  <input
                    type="text"
                    placeholder="e.g. 8080:80, 443:443"
                    value={newPorts}
                    onChange={(e) => setNewPorts(e.target.value)}
                    className="w-full px-3.5 py-2.5 rounded-xl bg-slate-50 dark:bg-surface-950 border border-slate-300 dark:border-surface-700 text-slate-950 dark:text-slate-100 font-mono text-sm focus:outline-none focus:border-indigo-500"
                  />
                </div>

                <div className="flex justify-end gap-2 pt-3">
                  <button
                    type="button"
                    onClick={() => setRunModalOpen(false)}
                    className="px-4 py-2.5 rounded-xl text-xs font-medium text-slate-600 dark:text-slate-400 hover:bg-slate-100 dark:hover:bg-surface-800 transition-colors"
                  >
                    Cancel
                  </button>
                  <button
                    type="submit"
                    disabled={submittingRun}
                    className="px-5 py-2.5 rounded-xl bg-indigo-600 hover:bg-indigo-500 text-white text-xs font-semibold shadow-md shadow-indigo-600/20 transition-all flex items-center gap-1.5"
                  >
                    {submittingRun ? <RefreshCw className="w-3.5 h-3.5 animate-spin" /> : null}
                    Launch Container
                  </button>
                </div>
              </form>
            </div>
          </div>
        )}
      </div>
    </DashboardShell>
  );
}
