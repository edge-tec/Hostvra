'use client';

import React, { useState, useEffect, useMemo } from 'react';
import { DashboardShell } from '@/components/DashboardShell';
import {
  Code2,
  CheckCircle2,
  XCircle,
  AlertTriangle,
  RefreshCw,
  Play,
  RotateCw,
  Square,
  Search,
  Sliders,
  FileCode2,
  Cpu,
  Layers,
  Activity,
  Download,
  Upload,
  Check,
  Zap,
  Server,
  Terminal,
  Shield,
  Trash2,
  Plus,
  Power,
  Globe,
} from 'lucide-react';
import { apiFetch } from '@/lib/api';

interface PHPVersionStatus {
  version: string;
  is_installed: boolean;
  cli_binary: string;
  fpm_binary: string;
  fpm_service_name: string;
  fpm_socket_path: string;
  ini_path: string;
  pool_dir: string;
  is_default_cli: boolean;
  is_default_fpm: boolean;
  fpm_running: boolean;
  active_pools: number;
}

interface ExtensionInfo {
  name: string;
  package_name: string;
  version?: string;
  description?: string;
  is_installed: boolean;
  is_enabled: boolean;
}

interface SimpleDirective {
  directive: string;
  category: string;
  type: string;
  default: string;
  recommended: string;
  description: string;
  current_value: string;
}

interface FPMStatus {
  version: string;
  service_name: string;
  is_running: boolean;
  pid?: number;
  active_workers: number;
  idle_workers: number;
  total_workers: number;
  memory_usage_mb: number;
  socket_exists: boolean;
  socket_path: string;
  pool_count: number;
  master_conf_path: string;
}

interface FPMPool {
  id: string;
  name: string;
  php_version: string;
  listen_socket: string;
  pool_user: string;
  pool_group: string;
  pm_type: string;
  pm_max_children: number;
  pm_start_servers: number;
  status: string;
}

interface HealthReport {
  version: string;
  cli_available: boolean;
  cli_version?: string;
  ini_valid: boolean;
  ini_error?: string;
  fpm_running: boolean;
  fpm_pid?: number;
  socket_exists: boolean;
  socket_path: string;
  active_workers: number;
  total_workers: number;
  memory_usage_mb: number;
  loaded_modules: string[];
  pool_statuses: Record<string, string>;
  overall_healthy: boolean;
  checked_at: string;
}

export default function PHPManagementPage() {
  const [activeTab, setActiveTab] = useState<'overview' | 'versions' | 'extensions' | 'ini' | 'fpm' | 'pools' | 'health'>('overview');
  const [selectedVersion, setSelectedVersion] = useState<string>('8.3');
  const [versions, setVersions] = useState<PHPVersionStatus[]>([]);
  const [extensions, setExtensions] = useState<ExtensionInfo[]>([]);
  const [simpleDirectives, setSimpleDirectives] = useState<SimpleDirective[]>([]);
  const [rawIni, setRawIni] = useState<string>('');
  const [iniMode, setIniMode] = useState<'simple' | 'advanced'>('simple');
  const [fpmStatus, setFpmStatus] = useState<FPMStatus | null>(null);
  const [pools, setPools] = useState<FPMPool[]>([]);
  const [healthReport, setHealthReport] = useState<HealthReport | null>(null);
  const [loading, setLoading] = useState<boolean>(false);
  const [actionLoading, setActionLoading] = useState<string | null>(null);
  const [message, setMessage] = useState<{ type: 'success' | 'error'; text: string } | null>(null);
  const [extSearch, setExtSearch] = useState<string>('');

  // Default server ID placeholder for single-node view
  const serverId = '00000000-0000-0000-0000-000000000001';

  // Load versions
  const loadVersions = async () => {
    setLoading(true);
    try {
      const res = await apiFetch<PHPVersionStatus[]>(`/api/v1/servers/${serverId}/php/versions`);
      if (res && res.data) {
        setVersions(res.data);
        // Set selected version to first installed if present
        const installed = res.data.find((v: PHPVersionStatus) => v.is_installed);
        if (installed) {
          setSelectedVersion(installed.version);
        }
      }
    } catch (e) {
      console.error('Failed to load PHP versions:', e);
    } finally {
      setLoading(false);
    }
  };

  // Load extensions for selected version
  const loadExtensions = async (v: string) => {
    try {
      const res = await apiFetch<ExtensionInfo[]>(`/api/v1/servers/${serverId}/php/${v}/extensions`);
      if (res && res.data) {
        setExtensions(res.data);
      }
    } catch (e) {
      console.error('Failed to load PHP extensions:', e);
    }
  };

  // Load INI settings
  const loadIni = async (v: string) => {
    try {
      const res = await apiFetch<{ simple_settings?: SimpleDirective[]; raw_content?: string }>(`/api/v1/servers/${serverId}/php/${v}/ini`);
      if (res && res.data) {
        setSimpleDirectives(res.data.simple_settings || []);
        setRawIni(res.data.raw_content || '');
      }
    } catch (e) {
      console.error('Failed to load PHP INI:', e);
    }
  };

  // Load FPM Status
  const loadFpmStatus = async (v: string) => {
    try {
      const res = await apiFetch<FPMStatus>(`/api/v1/servers/${serverId}/php/${v}/fpm`);
      if (res && res.data) {
        setFpmStatus(res.data);
      }
    } catch (e) {
      console.error('Failed to load PHP FPM status:', e);
    }
  };

  // Load Health
  const loadHealth = async (v: string) => {
    try {
      const res = await apiFetch<HealthReport>(`/api/v1/servers/${serverId}/php/${v}/health`);
      if (res && res.data) {
        setHealthReport(res.data);
      }
    } catch (e) {
      console.error('Failed to load PHP health report:', e);
    }
  };

  // Load Pools
  const loadPools = async (v: string) => {
    try {
      const res = await apiFetch<FPMPool[]>(`/api/v1/servers/${serverId}/php/${v}/pools`);
      if (res && res.data) {
        setPools(res.data);
      }
    } catch (e) {
      console.error('Failed to load PHP pools:', e);
    }
  };

  useEffect(() => {
    loadVersions();
  }, []);

  useEffect(() => {
    if (selectedVersion) {
      loadExtensions(selectedVersion);
      loadIni(selectedVersion);
      loadFpmStatus(selectedVersion);
      loadHealth(selectedVersion);
      loadPools(selectedVersion);
    }
  }, [selectedVersion]);

  // Install PHP Version
  const handleInstallVersion = async (v: string) => {
    setActionLoading(`install-${v}`);
    setMessage(null);
    try {
      const res = await apiFetch<{ message?: string }>(`/api/v1/servers/${serverId}/php/versions/install`, {
        method: 'POST',
        body: JSON.stringify({ version: v }),
      });
      if (res && res.success) {
        setMessage({ type: 'success', text: res.data?.message || `PHP ${v} installed successfully!` });
        await loadVersions();
      } else {
        setMessage({ type: 'error', text: res?.error?.message || `Failed to install PHP ${v}` });
      }
    } catch (e: any) {
      setMessage({ type: 'error', text: e?.message || 'Network request error' });
    } finally {
      setActionLoading(null);
    }
  };

  // Remove PHP Version
  const handleRemoveVersion = async (v: string) => {
    if (!confirm(`Are you sure you want to remove PHP ${v}? This will stop its FPM service.`)) return;
    setActionLoading(`remove-${v}`);
    setMessage(null);
    try {
      const res = await apiFetch<{ message?: string }>(`/api/v1/servers/${serverId}/php/versions/${v}`, {
        method: 'DELETE',
      });
      if (res && res.success) {
        setMessage({ type: 'success', text: res.data?.message || `PHP ${v} removed` });
        await loadVersions();
      } else {
        setMessage({ type: 'error', text: res?.error?.message || `Failed to remove PHP ${v}` });
      }
    } catch (e: any) {
      setMessage({ type: 'error', text: e?.message || 'Network request error' });
    } finally {
      setActionLoading(null);
    }
  };

  // Set Default CLI
  const handleSetDefaultCLI = async (v: string) => {
    setActionLoading(`cli-${v}`);
    try {
      const res = await apiFetch(`/api/v1/servers/${serverId}/php/versions/${v}/default-cli`, { method: 'POST' });
      if (res && res.success) {
        setMessage({ type: 'success', text: `Default PHP CLI switched to PHP ${v}` });
        await loadVersions();
      } else {
        setMessage({ type: 'error', text: res?.error?.message || 'Failed to switch default CLI' });
      }
    } catch (e: any) {
      setMessage({ type: 'error', text: e?.message || 'Failed to switch default CLI' });
    } finally {
      setActionLoading(null);
    }
  };

  // Toggle Extension
  const handleToggleExtension = async (extName: string, currentlyEnabled: boolean) => {
    setActionLoading(`ext-${extName}`);
    try {
      const res = await apiFetch(`/api/v1/servers/${serverId}/php/${selectedVersion}/extensions/${extName}/toggle`, {
        method: 'POST',
        body: JSON.stringify({ enabled: !currentlyEnabled }),
      });
      if (res && res.success) {
        await loadExtensions(selectedVersion);
        setMessage({ type: 'success', text: `Extension ${extName} ${!currentlyEnabled ? 'enabled' : 'disabled'}` });
      } else {
        setMessage({ type: 'error', text: res?.error?.message || 'Toggle failed' });
      }
    } catch (e: any) {
      setMessage({ type: 'error', text: e?.message || 'Failed to toggle extension' });
    } finally {
      setActionLoading(null);
    }
  };

  // Install Extension
  const handleInstallExtension = async (extName: string) => {
    setActionLoading(`ext-install-${extName}`);
    try {
      const res = await apiFetch(`/api/v1/servers/${serverId}/php/${selectedVersion}/extensions/install`, {
        method: 'POST',
        body: JSON.stringify({ extension: extName }),
      });
      if (res && res.success) {
        await loadExtensions(selectedVersion);
        setMessage({ type: 'success', text: `Extension ${extName} installed successfully` });
      } else {
        setMessage({ type: 'error', text: res?.error?.message || 'Install failed' });
      }
    } catch (e: any) {
      setMessage({ type: 'error', text: e?.message || 'Failed to install extension' });
    } finally {
      setActionLoading(null);
    }
  };

  // FPM Service Control
  const handleFpmAction = async (action: 'start' | 'stop' | 'restart' | 'reload') => {
    setActionLoading(`fpm-${action}`);
    try {
      const res = await apiFetch(`/api/v1/servers/${serverId}/php/${selectedVersion}/fpm/service`, {
        method: 'POST',
        body: JSON.stringify({ action }),
      });
      if (res && res.success) {
        setMessage({ type: 'success', text: `PHP ${selectedVersion}-FPM service ${action}ed successfully` });
        await loadFpmStatus(selectedVersion);
      } else {
        setMessage({ type: 'error', text: res?.error?.message || `Failed to ${action} PHP-FPM` });
      }
    } catch (e: any) {
      setMessage({ type: 'error', text: e?.message || 'Service action failed' });
    } finally {
      setActionLoading(null);
    }
  };

  // Filtered extensions
  const filteredExtensions = useMemo(() => {
    if (!extSearch.trim()) return extensions;
    const q = extSearch.toLowerCase();
    return extensions.filter(
      (e) => e.name.toLowerCase().includes(q) || e.package_name.toLowerCase().includes(q) || (e.description && e.description.toLowerCase().includes(q))
    );
  }, [extensions, extSearch]);

  return (
    <DashboardShell>
      <div className="space-y-6">
        {/* Header */}
        <div className="flex flex-col md:flex-row md:items-center md:justify-between gap-4 border-b border-slate-200 dark:border-slate-800 pb-5">
          <div>
            <div className="flex items-center gap-3">
              <div className="p-2.5 bg-sky-500/10 border border-sky-500/20 rounded-xl text-sky-600 dark:text-sky-400">
                <Code2 className="w-6 h-6" />
              </div>
              <div>
                <h1 className="text-2xl font-bold tracking-tight text-slate-950 dark:text-white flex items-center gap-2">
                  PHP Management
                  <span className="text-xs px-2.5 py-0.5 rounded-full bg-sky-500/10 text-sky-600 dark:text-sky-400 border border-sky-500/20 font-medium">
                    Native Linux Runtime
                  </span>
                </h1>
                <p className="text-sm text-slate-600 dark:text-slate-400">
                  Manage multi-version PHP runtimes, extensions, PHP.ini with atomic rollback, and isolated PHP-FPM pools.
                </p>
              </div>
            </div>
          </div>

          <div className="flex items-center gap-3">
            {/* Version Selector Pill */}
            <div className="flex items-center bg-white dark:bg-slate-800 border border-slate-300 dark:border-slate-700 rounded-lg p-1 shadow-xs">
              <span className="text-xs text-slate-600 dark:text-slate-300 px-2 font-semibold">PHP Version:</span>
              <select
                aria-label="Select PHP Version"
                value={selectedVersion}
                onChange={(e) => setSelectedVersion(e.target.value)}
                className="bg-slate-100 dark:bg-slate-900 border border-slate-200 dark:border-slate-700 text-slate-900 dark:text-white text-xs rounded font-bold px-2 py-1 focus:ring-1 focus:ring-sky-500 outline-none cursor-pointer"
              >
                {versions.map((v) => (
                  <option key={v.version} value={v.version}>
                    PHP {v.version} {v.is_installed ? '(Installed)' : '(Available)'}
                  </option>
                ))}
              </select>
            </div>

            <button
              onClick={() => {
                loadVersions();
                loadExtensions(selectedVersion);
                loadFpmStatus(selectedVersion);
                loadHealth(selectedVersion);
              }}
              className="p-2 bg-white hover:bg-slate-50 dark:bg-slate-800 dark:hover:bg-slate-700 text-slate-700 dark:text-slate-200 border border-slate-300 dark:border-slate-700 rounded-lg transition-colors flex items-center gap-1.5 text-xs font-semibold shadow-xs"
            >
              <RefreshCw className="w-3.5 h-3.5" />
              Refresh
            </button>
          </div>
        </div>

        {/* Global Notifications */}
        {message && (
          <div
            className={`p-4 rounded-xl border flex items-center justify-between text-sm ${
              message.type === 'success'
                ? 'bg-emerald-500/10 border-emerald-500/20 text-emerald-300'
                : 'bg-rose-500/10 border-rose-500/20 text-rose-300'
            }`}
          >
            <div className="flex items-center gap-2.5">
              {message.type === 'success' ? <CheckCircle2 className="w-5 h-5 text-emerald-400" /> : <XCircle className="w-5 h-5 text-rose-400" />}
              <span>{message.text}</span>
            </div>
            <button onClick={() => setMessage(null)} className="text-xs underline hover:opacity-80">
              Dismiss
            </button>
          </div>
        )}

        {/* Navigation Tabs */}
        <div className="flex items-center gap-1.5 p-1 bg-slate-100/80 dark:bg-slate-800/60 rounded-xl overflow-x-auto text-xs font-semibold">
          {[
            { id: 'overview', label: 'Overview', icon: Activity },
            { id: 'versions', label: 'Versions', icon: Server },
            { id: 'extensions', label: 'Extensions', icon: Zap },
            { id: 'ini', label: 'PHP.ini', icon: Sliders },
            { id: 'fpm', label: 'PHP-FPM Service', icon: Cpu },
            { id: 'pools', label: 'FPM Pools', icon: Layers },
            { id: 'health', label: 'Health Check', icon: Shield },
          ].map((tab) => {
            const Icon = tab.icon;
            const isActive = activeTab === tab.id;
            return (
              <button
                key={tab.id}
                role="tab"
                data-tab="true"
                onClick={() => setActiveTab(tab.id as any)}
                className={`flex items-center gap-2 px-3.5 py-2 rounded-lg transition-all whitespace-nowrap cursor-pointer font-medium ${
                  isActive
                    ? 'bg-white dark:bg-slate-900 text-sky-600 dark:text-sky-400 shadow-xs'
                    : 'text-slate-600 dark:text-slate-400 hover:text-slate-900 dark:hover:text-white hover:bg-white/40 dark:hover:bg-slate-700/40'
                }`}
              >
                <Icon className="w-4 h-4" />
                {tab.label}
              </button>
            );
          })}
        </div>

        {/* 1. OVERVIEW TAB */}
        {activeTab === 'overview' && (
          <div className="space-y-6">
            <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
              <div className="bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 rounded-xl p-5 shadow-xs">
                <div className="text-xs text-slate-500 dark:text-slate-400 font-medium mb-1">Installed Versions</div>
                <div className="text-2xl font-bold text-slate-950 dark:text-white">
                  {versions.filter((v) => v.is_installed).length} / {versions.length}
                </div>
                <div className="text-xs text-slate-500 mt-2">Available across OS repos</div>
              </div>

              <div className="bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 rounded-xl p-5 shadow-xs">
                <div className="text-xs text-slate-500 dark:text-slate-400 font-medium mb-1">FPM Service (PHP {selectedVersion})</div>
                <div className="flex items-center gap-2 mt-1">
                  <span
                    className={`w-2.5 h-2.5 rounded-full ${
                      fpmStatus?.is_running ? 'bg-emerald-500 shadow-[0_0_8px_rgba(16,185,129,0.5)]' : 'bg-rose-500'
                    }`}
                  />
                  <span className="text-lg font-bold text-slate-950 dark:text-white">{fpmStatus?.is_running ? 'Active (Running)' : 'Stopped'}</span>
                </div>
                <div className="text-xs text-slate-500 mt-2">PID: {fpmStatus?.pid || 'N/A'}</div>
              </div>

              <div className="bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 rounded-xl p-5 shadow-xs">
                <div className="text-xs text-slate-500 dark:text-slate-400 font-medium mb-1">Active Workers</div>
                <div className="text-2xl font-bold text-sky-600 dark:text-sky-400">{fpmStatus?.total_workers || 0}</div>
                <div className="text-xs text-slate-500 mt-2">RAM: ~{Math.round(fpmStatus?.memory_usage_mb || 0)} MB</div>
              </div>

              <div className="bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 rounded-xl p-5 shadow-xs">
                <div className="text-xs text-slate-500 dark:text-slate-400 font-medium mb-1">Isolated Pools</div>
                <div className="text-2xl font-bold text-slate-950 dark:text-white">{fpmStatus?.pool_count || 0}</div>
                <div className="text-xs text-slate-500 mt-2">Per-website FPM isolation</div>
              </div>
            </div>

            {/* Quick Status Cards */}
            <div className="bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 rounded-xl p-6 shadow-xs">
              <h2 className="text-base font-bold text-slate-950 dark:text-white mb-4 flex items-center gap-2">
                <Server className="w-4 h-4 text-sky-600 dark:text-sky-400" />
                Operating System Runtime Status (PHP {selectedVersion})
              </h2>

              <div className="grid grid-cols-1 md:grid-cols-2 gap-4 text-xs">
                <div className="p-4 bg-slate-50 dark:bg-slate-950/60 border border-slate-200 dark:border-slate-800 rounded-lg space-y-2.5">
                  <div className="flex justify-between items-center">
                    <span className="text-slate-600 dark:text-slate-400 font-medium">CLI Binary:</span>
                    <span className="font-mono text-slate-900 dark:text-slate-100 font-semibold">/usr/bin/php{selectedVersion}</span>
                  </div>
                  <div className="flex justify-between items-center">
                    <span className="text-slate-600 dark:text-slate-400 font-medium">FPM Unix Socket:</span>
                    <span className="font-mono text-slate-900 dark:text-slate-100 font-semibold">/run/php/php{selectedVersion}-fpm.sock</span>
                  </div>
                  <div className="flex justify-between items-center">
                    <span className="text-slate-600 dark:text-slate-400 font-medium">Configuration (php.ini):</span>
                    <span className="font-mono text-slate-900 dark:text-slate-100 font-semibold">/etc/php/{selectedVersion}/fpm/php.ini</span>
                  </div>
                </div>

                <div className="p-4 bg-slate-50 dark:bg-slate-950/60 border border-slate-200 dark:border-slate-800 rounded-lg space-y-2.5">
                  <div className="flex justify-between items-center">
                    <span className="text-slate-600 dark:text-slate-400 font-medium">Pool Directory:</span>
                    <span className="font-mono text-slate-900 dark:text-slate-100 font-semibold">/etc/php/{selectedVersion}/fpm/pool.d/</span>
                  </div>
                  <div className="flex justify-between items-center">
                    <span className="text-slate-600 dark:text-slate-400 font-medium">Service Name:</span>
                    <span className="font-mono text-slate-900 dark:text-slate-100 font-semibold">php{selectedVersion}-fpm.service</span>
                  </div>
                  <div className="flex justify-between items-center">
                    <span className="text-slate-600 dark:text-slate-400 font-medium">Loaded Extensions:</span>
                    <span className="font-bold text-emerald-600 dark:text-emerald-400">{extensions.filter((e) => e.is_enabled).length} Enabled</span>
                  </div>
                </div>
              </div>
            </div>
          </div>
        )}

        {/* 2. VERSIONS TAB */}
        {activeTab === 'versions' && (
          <div className="bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 rounded-xl overflow-hidden shadow-xs">
            <div className="p-5 border-b border-slate-200 dark:border-slate-800 flex justify-between items-center">
              <div>
                <h2 className="text-base font-bold text-slate-950 dark:text-white">PHP Version Management</h2>
                <p className="text-xs text-slate-600 dark:text-slate-400">Install, remove, and configure multiple concurrent PHP runtimes.</p>
              </div>
            </div>

            <div className="overflow-x-auto">
              <table className="w-full text-left text-sm text-slate-700 dark:text-slate-300">
                <thead className="bg-slate-50 dark:bg-slate-950 text-slate-600 dark:text-slate-400 text-xs uppercase tracking-wider border-b border-slate-200 dark:border-slate-800">
                  <tr>
                    <th className="px-6 py-3">Version</th>
                    <th className="px-6 py-3">Installation Status</th>
                    <th className="px-6 py-3">FPM Service</th>
                    <th className="px-6 py-3">Default CLI</th>
                    <th className="px-6 py-3">Active Pools</th>
                    <th className="px-6 py-3 text-right">Actions</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-slate-200 dark:divide-slate-800">
                  {versions.map((v) => (
                    <tr key={v.version} className="hover:bg-slate-50 dark:hover:bg-slate-800/40 transition-colors">
                      <td className="px-6 py-4 font-semibold text-slate-950 dark:text-white flex items-center gap-2">
                        <Code2 className="w-4 h-4 text-sky-600 dark:text-sky-400" />
                        PHP {v.version}
                      </td>
                      <td className="px-6 py-4">
                        {v.is_installed ? (
                          <span className="inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full text-xs font-medium bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 border border-emerald-500/20">
                            <CheckCircle2 className="w-3 h-3" /> Installed
                          </span>
                        ) : (
                          <span className="inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full text-xs font-medium bg-slate-100 dark:bg-slate-800 text-slate-600 dark:text-slate-400 border border-slate-200 dark:border-slate-700">
                            Available in Repos
                          </span>
                        )}
                      </td>
                      <td className="px-6 py-4">
                        {v.is_installed ? (
                          v.fpm_running ? (
                            <span className="inline-flex items-center gap-1.5 text-xs text-emerald-600 dark:text-emerald-400 font-semibold">
                              <span className="w-2 h-2 rounded-full bg-emerald-500 animate-pulse" /> Running
                            </span>
                          ) : (
                            <span className="inline-flex items-center gap-1.5 text-xs text-rose-600 dark:text-rose-400 font-semibold">
                              <span className="w-2 h-2 rounded-full bg-rose-500" /> Inactive
                            </span>
                          )
                        ) : (
                          <span className="text-slate-400 text-xs">—</span>
                        )}
                      </td>
                      <td className="px-6 py-4">
                        {v.is_default_cli ? (
                          <span className="text-xs px-2.5 py-1 rounded-md bg-sky-50 dark:bg-sky-500/10 text-sky-700 dark:text-sky-400 border border-sky-200 dark:border-sky-500/20 font-bold">
                            Default CLI
                          </span>
                        ) : v.is_installed ? (
                          <button
                            onClick={() => handleSetDefaultCLI(v.version)}
                            disabled={actionLoading === `cli-${v.version}`}
                            className="text-xs text-slate-600 dark:text-slate-400 hover:text-sky-600 dark:hover:text-white underline font-medium"
                          >
                            Set Default
                          </button>
                        ) : (
                          <span className="text-slate-400 text-xs">—</span>
                        )}
                      </td>
                      <td className="px-6 py-4 text-slate-700 dark:text-slate-300 font-mono text-xs">{v.is_installed ? v.active_pools : '—'}</td>
                      <td className="px-6 py-4 text-right space-x-2">
                        {v.is_installed ? (
                          <button
                            onClick={() => handleRemoveVersion(v.version)}
                            disabled={actionLoading === `remove-${v.version}`}
                            className="px-3 py-1 bg-rose-50 hover:bg-rose-100 dark:bg-rose-500/10 dark:hover:bg-rose-500/20 text-rose-700 dark:text-rose-400 border border-rose-200 dark:border-rose-500/20 rounded-md text-xs font-semibold transition-colors"
                          >
                            Remove
                          </button>
                        ) : (
                          <button
                            onClick={() => handleInstallVersion(v.version)}
                            disabled={actionLoading === `install-${v.version}`}
                            className="px-3 py-1 bg-sky-600 hover:bg-sky-500 text-white rounded-md text-xs font-semibold transition-colors inline-flex items-center gap-1 shadow-xs"
                          >
                            <Download className="w-3 h-3" /> Install PHP {v.version}
                          </button>
                        )}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </div>
        )}

        {/* 3. EXTENSIONS TAB */}
        {activeTab === 'extensions' && (
          <div className="bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 rounded-xl p-5 space-y-4 shadow-xs">
            <div className="flex flex-col md:flex-row md:items-center md:justify-between gap-3 border-b border-slate-200 dark:border-slate-800 pb-4">
              <div>
                <h2 className="text-base font-bold text-slate-950 dark:text-white flex items-center gap-2">
                  <Zap className="w-4 h-4 text-sky-600 dark:text-sky-400" />
                  PHP {selectedVersion} Extensions
                </h2>
                <p className="text-xs text-slate-600 dark:text-slate-400">Search, install, enable, or disable PHP modules dynamically.</p>
              </div>

              <div className="relative">
                <Search className="w-4 h-4 text-slate-400 absolute left-3 top-1/2 -translate-y-1/2" />
                <input
                  type="text"
                  placeholder="Search extensions (e.g. redis, curl, imagick)..."
                  value={extSearch}
                  onChange={(e) => setExtSearch(e.target.value)}
                  className="bg-white dark:bg-slate-950 border border-slate-300 dark:border-slate-800 rounded-lg pl-9 pr-4 py-1.5 text-xs text-slate-900 dark:text-white placeholder-slate-400 focus:ring-1 focus:ring-sky-500 outline-none w-72 shadow-xs"
                />
              </div>
            </div>

            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-3">
              {filteredExtensions.map((ext) => (
                <div
                  key={ext.name}
                  className="p-4 bg-slate-50 dark:bg-slate-950/50 border border-slate-200 dark:border-slate-800 rounded-lg flex flex-col justify-between hover:border-slate-300 dark:hover:border-slate-700 transition-colors shadow-xs"
                >
                  <div>
                    <div className="flex items-center justify-between">
                      <span className="font-bold text-slate-950 dark:text-white text-sm font-mono">{ext.name}</span>
                      {ext.is_installed ? (
                        ext.is_enabled ? (
                          <span className="text-[10px] uppercase font-bold px-2 py-0.5 rounded bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 border border-emerald-500/20">
                            Enabled
                          </span>
                        ) : (
                          <span className="text-[10px] uppercase font-bold px-2 py-0.5 rounded bg-amber-500/10 text-amber-600 dark:text-amber-400 border border-amber-500/20">
                            Disabled
                          </span>
                        )
                      ) : (
                        <span className="text-[10px] uppercase font-bold px-2 py-0.5 rounded bg-slate-200 dark:bg-slate-800 text-slate-600 dark:text-slate-400">
                          Available
                        </span>
                      )}
                    </div>
                    <p className="text-xs text-slate-600 dark:text-slate-400 mt-1 line-clamp-2">{ext.description || 'Standard PHP extension module.'}</p>
                    <div className="text-[11px] font-mono text-slate-500 dark:text-slate-400 mt-2 truncate">pkg: {ext.package_name}</div>
                  </div>

                  <div className="mt-4 pt-3 border-t border-slate-200 dark:border-slate-800/80 flex items-center justify-end gap-2">
                    {ext.is_installed ? (
                      <button
                        onClick={() => handleToggleExtension(ext.name, ext.is_enabled)}
                        disabled={actionLoading === `ext-${ext.name}`}
                        className={`px-3 py-1 rounded text-xs font-semibold transition-colors ${
                          ext.is_enabled
                            ? 'bg-white hover:bg-slate-100 dark:bg-slate-800 dark:hover:bg-slate-700 text-slate-700 dark:text-slate-300 border border-slate-300 dark:border-slate-700 shadow-xs'
                            : 'bg-emerald-600 hover:bg-emerald-500 text-white shadow-xs'
                        }`}
                      >
                        {ext.is_enabled ? 'Disable' : 'Enable'}
                      </button>
                    ) : (
                      <button
                        onClick={() => handleInstallExtension(ext.name)}
                        disabled={actionLoading === `ext-install-${ext.name}`}
                        className="px-3 py-1 bg-sky-600 hover:bg-sky-500 text-white rounded text-xs font-semibold transition-colors flex items-center gap-1 shadow-xs"
                      >
                        <Plus className="w-3 h-3" /> Install
                      </button>
                    )}
                  </div>
                </div>
              ))}
            </div>
          </div>
        )}

        {/* 4. PHP.INI TAB */}
        {activeTab === 'ini' && (
          <div className="bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 rounded-xl p-5 space-y-5 shadow-xs">
            <div className="flex flex-col md:flex-row md:items-center md:justify-between gap-3 border-b border-slate-200 dark:border-slate-800 pb-4">
              <div>
                <h2 className="text-base font-bold text-slate-950 dark:text-white flex items-center gap-2">
                  <Sliders className="w-4 h-4 text-sky-600 dark:text-sky-400" />
                  PHP {selectedVersion} Configuration Editor (php.ini)
                </h2>
                <p className="text-xs text-slate-600 dark:text-slate-400">
                  Atomic transactional editing with automatic syntax validation and instant rollback on syntax error.
                </p>
              </div>

              {/* Mode Toggle */}
              <div className="flex items-center bg-slate-100 dark:bg-slate-950 border border-slate-200 dark:border-slate-800 p-1 rounded-lg">
                <button
                  onClick={() => setIniMode('simple')}
                  className={`px-3 py-1 rounded text-xs font-semibold transition-colors ${
                    iniMode === 'simple' ? 'bg-sky-600 text-white shadow-xs' : 'text-slate-600 dark:text-slate-400 hover:text-slate-900 dark:hover:text-white'
                  }`}
                >
                  Simple Mode
                </button>
                <button
                  onClick={() => setIniMode('advanced')}
                  className={`px-3 py-1 rounded text-xs font-semibold transition-colors ${
                    iniMode === 'advanced' ? 'bg-sky-600 text-white shadow-xs' : 'text-slate-600 dark:text-slate-400 hover:text-slate-900 dark:hover:text-white'
                  }`}
                >
                  Advanced Raw Editor
                </button>
              </div>
            </div>

            {iniMode === 'simple' ? (
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                {simpleDirectives.map((d) => (
                  <div key={d.directive} className="p-4 bg-slate-50 dark:bg-slate-950/60 border border-slate-200 dark:border-slate-800 rounded-lg space-y-2 shadow-xs">
                    <div className="flex justify-between items-center">
                      <span className="font-mono text-sm font-bold text-slate-950 dark:text-white">{d.directive}</span>
                      <span className="text-[10px] px-2 py-0.5 rounded bg-slate-200 dark:bg-slate-800 text-slate-700 dark:text-slate-400 font-semibold">{d.category}</span>
                    </div>
                    <p className="text-xs text-slate-600 dark:text-slate-400">{d.description}</p>

                    <div className="pt-2 flex items-center justify-between gap-3">
                      <input
                        type="text"
                        defaultValue={d.current_value}
                        onBlur={async (e) => {
                          const val = e.target.value;
                          if (val !== d.current_value) {
                            try {
                              const res = await apiFetch(`/api/v1/servers/${serverId}/php/${selectedVersion}/ini`, {
                                method: 'PUT',
                                body: JSON.stringify({ directives: { [d.directive]: val } }),
                              });
                              if (res && res.success) {
                                setMessage({ type: 'success', text: `Updated ${d.directive} = ${val}` });
                              } else {
                                setMessage({ type: 'error', text: res?.error?.message || 'Update failed' });
                              }
                            } catch (err: any) {
                              setMessage({ type: 'error', text: err?.message || 'Update failed' });
                            }
                          }
                        }}
                        className="bg-white dark:bg-slate-900 border border-slate-300 dark:border-slate-700 rounded px-3 py-1.5 text-xs text-slate-900 dark:text-white font-mono w-44 focus:ring-1 focus:ring-sky-500 outline-none shadow-xs"
                      />
                      <span className="text-[11px] text-slate-500 dark:text-slate-400 font-medium">Rec: {d.recommended}</span>
                    </div>
                  </div>
                ))}
              </div>
            ) : (
              <div className="space-y-4">
                <textarea
                  value={rawIni}
                  onChange={(e) => setRawIni(e.target.value)}
                  rows={20}
                  className="w-full bg-slate-50 dark:bg-slate-950 border border-slate-300 dark:border-slate-800 rounded-lg p-4 font-mono text-xs text-slate-900 dark:text-slate-200 focus:ring-1 focus:ring-sky-500 outline-none leading-relaxed shadow-xs"
                />
                <div className="flex justify-end gap-3">
                  <button
                    onClick={() => loadIni(selectedVersion)}
                    className="px-4 py-2 bg-white hover:bg-slate-100 dark:bg-slate-800 dark:hover:bg-slate-700 text-slate-700 dark:text-slate-200 border border-slate-300 dark:border-slate-700 rounded-lg text-xs font-semibold shadow-xs"
                  >
                    Reset Changes
                  </button>
                  <button
                    onClick={async () => {
                      setActionLoading('save-ini');
                      try {
                        const res = await apiFetch(`/api/v1/servers/${serverId}/php/${selectedVersion}/ini`, {
                          method: 'PUT',
                          body: JSON.stringify({ raw_content: rawIni }),
                        });
                        if (res && res.success) {
                          setMessage({ type: 'success', text: 'Configuration saved and PHP-FPM reloaded safely!' });
                        } else {
                          setMessage({ type: 'error', text: res?.error?.message || 'Save failed and configuration was rolled back.' });
                        }
                      } catch (e: any) {
                        setMessage({ type: 'error', text: e?.message || 'Error saving configuration' });
                      } finally {
                        setActionLoading(null);
                      }
                    }}
                    disabled={actionLoading === 'save-ini'}
                    className="px-5 py-2 bg-sky-600 hover:bg-sky-500 text-white rounded-lg text-xs font-semibold flex items-center gap-1.5 shadow-xs"
                  >
                    <Check className="w-4 h-4" /> Save & Safely Apply
                  </button>
                </div>
              </div>
            )}
          </div>
        )}

        {/* 5. PHP-FPM TAB */}
        {activeTab === 'fpm' && (
          <div className="bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 rounded-xl p-6 space-y-6 shadow-xs">
            <div className="flex flex-col md:flex-row md:items-center md:justify-between gap-4 border-b border-slate-200 dark:border-slate-800 pb-4">
              <div>
                <h2 className="text-base font-bold text-slate-950 dark:text-white flex items-center gap-2">
                  <Cpu className="w-4 h-4 text-sky-600 dark:text-sky-400" />
                  PHP {selectedVersion}-FPM Master Service
                </h2>
                <p className="text-xs text-slate-600 dark:text-slate-400">Control daemon lifecycle and inspect real Linux worker process counts.</p>
              </div>

              <div className="flex items-center gap-2">
                <button
                  onClick={() => handleFpmAction('reload')}
                  disabled={actionLoading === 'fpm-reload'}
                  className="px-3 py-1.5 bg-white hover:bg-slate-50 dark:bg-slate-800 dark:hover:bg-slate-700 text-slate-700 dark:text-slate-200 border border-slate-300 dark:border-slate-700 rounded-lg text-xs font-semibold flex items-center gap-1.5 transition-colors shadow-xs"
                >
                  <RotateCw className="w-3.5 h-3.5 text-sky-600 dark:text-sky-400" /> Graceful Reload
                </button>
                <button
                  onClick={() => handleFpmAction('restart')}
                  disabled={actionLoading === 'fpm-restart'}
                  className="px-3 py-1.5 bg-white hover:bg-slate-50 dark:bg-slate-800 dark:hover:bg-slate-700 text-slate-700 dark:text-slate-200 border border-slate-300 dark:border-slate-700 rounded-lg text-xs font-semibold flex items-center gap-1.5 transition-colors shadow-xs"
                >
                  <RefreshCw className="w-3.5 h-3.5 text-amber-600 dark:text-amber-400" /> Restart
                </button>
                {fpmStatus?.is_running ? (
                  <button
                    onClick={() => handleFpmAction('stop')}
                    disabled={actionLoading === 'fpm-stop'}
                    className="px-3 py-1.5 bg-rose-50 hover:bg-rose-100 dark:bg-rose-500/10 dark:hover:bg-rose-500/20 text-rose-700 dark:text-rose-400 border border-rose-200 dark:border-rose-500/20 rounded-lg text-xs font-semibold flex items-center gap-1.5 transition-colors shadow-xs"
                  >
                    <Square className="w-3.5 h-3.5" /> Stop
                  </button>
                ) : (
                  <button
                    onClick={() => handleFpmAction('start')}
                    disabled={actionLoading === 'fpm-start'}
                    className="px-3 py-1.5 bg-emerald-600 hover:bg-emerald-500 text-white rounded-lg text-xs font-semibold flex items-center gap-1.5 transition-colors shadow-xs"
                  >
                    <Play className="w-3.5 h-3.5" /> Start Service
                  </button>
                )}
              </div>
            </div>

            <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
              <div className="p-4 bg-slate-50 dark:bg-slate-950/60 border border-slate-200 dark:border-slate-800 rounded-lg space-y-1 shadow-xs">
                <div className="text-xs text-slate-500 dark:text-slate-400 font-medium">Master Service Status</div>
                <div className="text-base font-bold text-slate-950 dark:text-white flex items-center gap-2">
                  <span className={`w-2.5 h-2.5 rounded-full ${fpmStatus?.is_running ? 'bg-emerald-500' : 'bg-rose-500'}`} />
                  {fpmStatus?.is_running ? 'Active (Running)' : 'Inactive'}
                </div>
                <div className="text-xs text-slate-500 font-mono">Service: {fpmStatus?.service_name}</div>
              </div>

              <div className="p-4 bg-slate-50 dark:bg-slate-950/60 border border-slate-200 dark:border-slate-800 rounded-lg space-y-1 shadow-xs">
                <div className="text-xs text-slate-500 dark:text-slate-400 font-medium">Total Worker Processes</div>
                <div className="text-base font-bold text-sky-600 dark:text-sky-400">{fpmStatus?.total_workers || 0} active workers</div>
                <div className="text-xs text-slate-500">Idle: {fpmStatus?.idle_workers || 0}</div>
              </div>

              <div className="p-4 bg-slate-50 dark:bg-slate-950/60 border border-slate-200 dark:border-slate-800 rounded-lg space-y-1 shadow-xs">
                <div className="text-xs text-slate-500 dark:text-slate-400 font-medium">Master Unix Socket</div>
                <div className="text-xs font-mono text-slate-900 dark:text-slate-200 truncate">{fpmStatus?.socket_path}</div>
                <div className="text-xs text-emerald-600 dark:text-emerald-400 font-semibold">{fpmStatus?.socket_exists ? '✓ Socket Available' : '⚠ Socket Missing'}</div>
              </div>
            </div>
          </div>
        )}

        {/* 6. FPM POOLS TAB */}
        {activeTab === 'pools' && (
          <div className="bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 rounded-xl p-5 space-y-4 shadow-xs">
            <div className="flex justify-between items-center border-b border-slate-200 dark:border-slate-800 pb-4">
              <div>
                <h2 className="text-base font-bold text-slate-950 dark:text-white flex items-center gap-2">
                  <Layers className="w-4 h-4 text-sky-600 dark:text-sky-400" />
                  Isolated PHP-FPM Pools (PHP {selectedVersion})
                </h2>
                <p className="text-xs text-slate-600 dark:text-slate-400">Each website runs in its own isolated pool and distinct user process tree.</p>
              </div>
            </div>

            <div className="overflow-x-auto">
              <table className="w-full text-left text-sm text-slate-700 dark:text-slate-300">
                <thead className="bg-slate-50 dark:bg-slate-950 text-slate-600 dark:text-slate-400 text-xs uppercase tracking-wider border-b border-slate-200 dark:border-slate-800">
                  <tr>
                    <th className="px-6 py-3">Pool Name</th>
                    <th className="px-6 py-3">Socket</th>
                    <th className="px-6 py-3">Process User</th>
                    <th className="px-6 py-3">PM Type</th>
                    <th className="px-6 py-3">Max Children</th>
                    <th className="px-6 py-3">Status</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-slate-200 dark:divide-slate-800">
                  {pools.length === 0 ? (
                    <tr>
                      <td colSpan={6} className="px-6 py-8 text-center text-slate-500 text-xs">
                        No custom website pools configured for PHP {selectedVersion}. Websites using this version currently connect to the master pool.
                      </td>
                    </tr>
                  ) : (
                    pools.map((p) => (
                      <tr key={p.id} className="hover:bg-slate-50 dark:hover:bg-slate-800/40">
                        <td className="px-6 py-4 font-mono text-xs font-bold text-slate-950 dark:text-white">{p.name}</td>
                        <td className="px-6 py-4 font-mono text-xs text-slate-600 dark:text-slate-400">{p.listen_socket}</td>
                        <td className="px-6 py-4 text-xs text-slate-700 dark:text-slate-300">{p.pool_user}:{p.pool_group}</td>
                        <td className="px-6 py-4 text-xs font-mono text-sky-600 dark:text-sky-400 uppercase font-semibold">{p.pm_type}</td>
                        <td className="px-6 py-4 text-xs text-slate-700 dark:text-slate-300">{p.pm_max_children}</td>
                        <td className="px-6 py-4">
                          <span className="text-[10px] uppercase font-bold px-2 py-0.5 rounded bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 border border-emerald-500/20">
                            {p.status}
                          </span>
                        </td>
                      </tr>
                    ))
                  )}
                </tbody>
              </table>
            </div>
          </div>
        )}

        {/* 7. HEALTH CHECK TAB */}
        {activeTab === 'health' && (
          <div className="bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 rounded-xl p-6 space-y-6 shadow-xs">
            <div className="flex justify-between items-center border-b border-slate-200 dark:border-slate-800 pb-4">
              <div>
                <h2 className="text-base font-bold text-slate-950 dark:text-white flex items-center gap-2">
                  <Shield className="w-4 h-4 text-sky-600 dark:text-sky-400" />
                  Subsystem Health Diagnostic (PHP {selectedVersion})
                </h2>
                <p className="text-xs text-slate-600 dark:text-slate-400">Deep runtime audit testing binary execution, syntax, sockets, and memory.</p>
              </div>

              <button
                onClick={() => loadHealth(selectedVersion)}
                className="px-3.5 py-2 bg-sky-600 hover:bg-sky-500 text-white rounded-lg text-xs font-semibold flex items-center gap-1.5 shadow-xs"
              >
                <RefreshCw className="w-3.5 h-3.5" /> Run Diagnostic Audit
              </button>
            </div>

            {healthReport && (
              <div className="space-y-4">
                <div
                  className={`p-4 rounded-xl border flex items-center gap-3 ${
                    healthReport.overall_healthy
                      ? 'bg-emerald-50 dark:bg-emerald-500/10 border-emerald-200 dark:border-emerald-500/20 text-emerald-800 dark:text-emerald-300'
                      : 'bg-rose-50 dark:bg-rose-500/10 border-rose-200 dark:border-rose-500/20 text-rose-800 dark:text-rose-300'
                  }`}
                >
                  {healthReport.overall_healthy ? <CheckCircle2 className="w-6 h-6 text-emerald-600 dark:text-emerald-400" /> : <AlertTriangle className="w-6 h-6 text-rose-600 dark:text-rose-400" />}
                  <div>
                    <div className="font-bold">
                      {healthReport.overall_healthy ? `PHP ${selectedVersion} Subsystem Healthy` : `PHP ${selectedVersion} Issues Detected`}
                    </div>
                    <div className="text-xs opacity-80">
                      Audit run at {new Date(healthReport.checked_at).toLocaleTimeString()}
                    </div>
                  </div>
                </div>

                <div className="grid grid-cols-1 md:grid-cols-2 gap-4 text-xs">
                  <div className="p-4 bg-slate-50 dark:bg-slate-950/60 border border-slate-200 dark:border-slate-800 rounded-lg space-y-2 shadow-xs">
                    <div className="flex justify-between items-center">
                      <span className="text-slate-600 dark:text-slate-400 font-medium">CLI Binary Execution:</span>
                      <span className="font-bold text-emerald-600 dark:text-emerald-400">{healthReport.cli_available ? '✓ Verified' : '✗ Missing'}</span>
                    </div>
                    <div className="flex justify-between items-center">
                      <span className="text-slate-600 dark:text-slate-400 font-medium">PHP.ini Syntax Validity:</span>
                      <span className="font-bold text-emerald-600 dark:text-emerald-400">{healthReport.ini_valid ? '✓ Valid' : '✗ Syntax Error'}</span>
                    </div>
                    <div className="flex justify-between items-center">
                      <span className="text-slate-600 dark:text-slate-400 font-medium">PHP-FPM Daemon:</span>
                      <span className="font-bold text-emerald-600 dark:text-emerald-400">{healthReport.fpm_running ? '✓ Active' : '✗ Down'}</span>
                    </div>
                  </div>

                  <div className="p-4 bg-slate-50 dark:bg-slate-950/60 border border-slate-200 dark:border-slate-800 rounded-lg space-y-2 shadow-xs">
                    <div className="flex justify-between items-center">
                      <span className="text-slate-600 dark:text-slate-400 font-medium">FastCGI Socket Existence:</span>
                      <span className="font-bold text-emerald-600 dark:text-emerald-400">{healthReport.socket_exists ? '✓ Available' : '✗ Not Found'}</span>
                    </div>
                    <div className="flex justify-between items-center">
                      <span className="text-slate-600 dark:text-slate-400 font-medium">Loaded Modules Count:</span>
                      <span className="font-bold text-sky-600 dark:text-sky-400">{healthReport.loaded_modules.length} Modules</span>
                    </div>
                    <div className="flex justify-between items-center">
                      <span className="text-slate-600 dark:text-slate-400 font-medium">Active Workers:</span>
                      <span className="font-bold text-slate-950 dark:text-white">{healthReport.total_workers} processes</span>
                    </div>
                  </div>
                </div>
              </div>
            )}
          </div>
        )}
      </div>
    </DashboardShell>
  );
}
