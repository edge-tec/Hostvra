'use client';

import React, { useState, useEffect, useMemo } from 'react';
import { DashboardShell } from '@/components/DashboardShell';
import { apiFetch } from '@/lib/api';
import {
  Layers,
  Server,
  Play,
  RotateCw,
  Square,
  AlertTriangle,
  CheckCircle2,
  XCircle,
  Shield,
  FileCode2,
  RefreshCw,
  Globe,
  ArrowRightLeft,
  Cpu,
  Lock,
  Activity,
  Sliders,
  Check,
  AlertCircle,
  ExternalLink,
  Code2,
} from 'lucide-react';

interface WebServerInstance {
  id?: string;
  server_id: string;
  server_type: 'nginx' | 'apache' | 'openlitespeed' | 'litespeed';
  name?: string;
  version?: string;
  binary_path?: string;
  config_path?: string;
  service_name: string;
  is_installed: boolean;
  is_running?: boolean;
  is_active_default: boolean;
  http_port: number;
  https_port: number;
  status: 'running' | 'stopped' | 'failed' | 'not_installed';
  license_status?: string;
  installed_modules?: string[];
  active_vhosts?: number;
  updated_at?: string;
}

interface PortConflict {
  port: number;
  process_name: string;
  pid: number;
  server_type: string;
  command_line: string;
}

interface ManagedServer {
  id: string;
  name: string;
  hostname: string;
  ip_address: string;
  status: string;
}

interface WebsiteItem {
  id: string;
  primary_domain: string;
  document_root: string;
  web_server_type: string;
  app_type: string;
  php_version?: string;
  ssl_enabled: boolean;
  status: string;
}

export default function WebServersPage() {
  const [servers, setServers] = useState<ManagedServer[]>([]);
  const [selectedServerId, setSelectedServerId] = useState<string>('');
  const [webServers, setWebServers] = useState<WebServerInstance[]>([]);
  const [portConflicts, setPortConflicts] = useState<PortConflict[]>([]);
  const [websites, setWebsites] = useState<WebsiteItem[]>([]);
  const [loading, setLoading] = useState<boolean>(true);
  const [actionLoading, setActionLoading] = useState<string | null>(null);
  const [activeTab, setActiveTab] = useState<'overview' | 'config' | 'vhosts' | 'migration'>('overview');
  
  // Notification & Feedback
  const [toast, setToast] = useState<{ type: 'success' | 'error' | 'info'; message: string } | null>(null);

  // Config Editor State
  const [configServerType, setConfigServerType] = useState<string>('nginx');
  const [configContent, setConfigContent] = useState<string>('');
  const [configLoading, setConfigLoading] = useState<boolean>(false);
  const [configSaving, setConfigSaving] = useState<boolean>(false);

  // Migration Modal / State
  const [sourceServer, setSourceServer] = useState<string>('nginx');
  const [targetServer, setTargetServer] = useState<string>('apache');
  const [migrating, setMigrating] = useState<boolean>(false);
  const [migrationResult, setMigrationResult] = useState<any | null>(null);

  // VHost View Modal
  const [viewingVHost, setViewingVHost] = useState<{ domain: string; content: string; serverType: string } | null>(null);

  const showToast = (type: 'success' | 'error' | 'info', message: string) => {
    setToast({ type, message });
    setTimeout(() => setToast(null), 5000);
  };

  // Fetch servers list
  useEffect(() => {
    async function loadServers() {
      const res = await apiFetch<ManagedServer[]>('/api/v1/servers');
      if (res.success && res.data && res.data.length > 0) {
        setServers(res.data);
        setSelectedServerId(res.data[0].id);
      } else {
        setLoading(false);
      }
    }
    loadServers();
  }, []);

  // Fetch web servers data whenever selectedServerId changes
  useEffect(() => {
    if (!selectedServerId) return;
    fetchServerData(selectedServerId);
  }, [selectedServerId]);

  async function fetchServerData(serverId: string) {
    setLoading(true);
    try {
      // 1. Fetch web server instances
      const wsRes = await apiFetch<WebServerInstance[]>(`/api/v1/servers/${serverId}/webservers`);
      if (wsRes.success && wsRes.data) {
        setWebServers(wsRes.data);
      }

      // 2. Fetch port conflicts
      const confRes = await apiFetch<PortConflict[]>(`/api/v1/servers/${serverId}/webservers/conflicts`);
      if (confRes.success && confRes.data) {
        setPortConflicts(confRes.data);
      }

      // 3. Fetch websites
      const webRes = await apiFetch<WebsiteItem[]>('/api/v1/websites');
      if (webRes.success && webRes.data) {
        setWebsites(webRes.data);
      }
    } catch (err: any) {
      showToast('error', err.message || 'Failed to fetch web server telemetry');
    } finally {
      setLoading(false);
    }
  }

  // Load master config when config tab is open
  useEffect(() => {
    if (activeTab === 'config' && selectedServerId) {
      loadMasterConfig(configServerType);
    }
  }, [activeTab, configServerType, selectedServerId]);

  async function loadMasterConfig(type: string) {
    setConfigLoading(true);
    const res = await apiFetch<{ server_type: string; content: string }>(`/api/v1/servers/${selectedServerId}/webservers/${type}/config`);
    if (res.success && res.data) {
      setConfigContent(res.data.content);
    } else {
      showToast('error', res.error?.message || `Failed to read ${type} configuration`);
      setConfigContent('');
    }
    setConfigLoading(false);
  }

  async function handleSaveConfig() {
    setConfigSaving(true);
    const res = await apiFetch<{ message: string }>(`/api/v1/servers/${selectedServerId}/webservers/${configServerType}/config`, {
      method: 'PUT',
      body: JSON.stringify({
        content: configContent,
        description: 'Updated via Hostvra Web Server Manager UI',
      }),
    });

    if (res.success) {
      showToast('success', res.data?.message || 'Configuration syntax verified and saved successfully!');
    } else {
      showToast('error', res.error?.message || 'Configuration test failed. Changes were safely rolled back.');
    }
    setConfigSaving(false);
  }

  // Service actions
  async function handleServiceAction(serverType: string, action: 'start' | 'stop' | 'restart' | 'reload') {
    const key = `${serverType}:${action}`;
    setActionLoading(key);
    const res = await apiFetch<{ message: string }>(`/api/v1/servers/${selectedServerId}/webservers/${serverType}/service`, {
      method: 'POST',
      body: JSON.stringify({ action }),
    });

    if (res.success) {
      showToast('success', res.data?.message || `${serverType} ${action} executed successfully`);
      await fetchServerData(selectedServerId);
    } else {
      showToast('error', res.error?.message || `Failed to ${action} ${serverType}`);
    }
    setActionLoading(null);
  }

  // Install / Uninstall
  async function handleInstall(serverType: string) {
    setActionLoading(`install:${serverType}`);
    const res = await apiFetch<{ message: string }>(`/api/v1/servers/${selectedServerId}/webservers/${serverType}/install`, {
      method: 'POST',
    });
    if (res.success) {
      showToast('success', res.data?.message || `${serverType} installed successfully`);
      await fetchServerData(selectedServerId);
    } else {
      showToast('error', res.error?.message || `Failed to install ${serverType}`);
    }
    setActionLoading(null);
  }

  async function handleUninstall(serverType: string) {
    if (!confirm(`Are you sure you want to uninstall ${serverType}? All virtualhosts must be migrated first.`)) {
      return;
    }
    setActionLoading(`uninstall:${serverType}`);
    const res = await apiFetch<{ message: string }>(`/api/v1/servers/${selectedServerId}/webservers/${serverType}/uninstall`, {
      method: 'POST',
    });
    if (res.success) {
      showToast('success', res.data?.message || `${serverType} uninstalled`);
      await fetchServerData(selectedServerId);
    } else {
      showToast('error', res.error?.message || `Failed to uninstall ${serverType}`);
    }
    setActionLoading(null);
  }

  // Full Safe Migration
  async function handleExecuteMigration() {
    if (sourceServer === targetServer) {
      showToast('error', 'Source and Target web servers cannot be identical');
      return;
    }
    setMigrating(true);
    setMigrationResult(null);

    const res = await apiFetch<any>(`/api/v1/servers/${selectedServerId}/webservers/migrate`, {
      method: 'POST',
      body: JSON.stringify({
        source_type: sourceServer,
        target_type: targetServer,
        auto_start: true,
      }),
    });

    if (res.success && res.data) {
      setMigrationResult(res.data);
      showToast('success', res.data.message || 'Migration successfully completed with zero downtime!');
      await fetchServerData(selectedServerId);
    } else {
      setMigrationResult(res.error?.details || { message: res.error?.message, success: false });
      showToast('error', res.error?.message || 'Migration encountered an error and was safely rolled back.');
    }
    setMigrating(false);
  }

  // Single Website Switch
  async function handleSwitchWebsite(websiteId: string, newType: string) {
    setActionLoading(`switch:${websiteId}`);
    const res = await apiFetch<{ message: string }>(`/api/v1/websites/${websiteId}/webserver/switch`, {
      method: 'POST',
      body: JSON.stringify({ target_server_type: newType }),
    });

    if (res.success) {
      showToast('success', res.data?.message || 'Website web server switched successfully');
      await fetchServerData(selectedServerId);
    } else {
      showToast('error', res.error?.message || 'Failed to switch website web server');
    }
    setActionLoading(null);
  }

  // View VHost
  async function handleInspectVHost(website: WebsiteItem) {
    const res = await apiFetch<{ vhost_content: string; web_server_type: string }>(`/api/v1/websites/${website.id}/webserver`);
    if (res.success && res.data) {
      setViewingVHost({
        domain: website.primary_domain,
        content: res.data.vhost_content || '# VirtualHost configuration not yet written',
        serverType: res.data.web_server_type,
      });
    } else {
      showToast('error', 'Failed to inspect VirtualHost configuration');
    }
  }

  // Helper metadata
  const serverMeta: Record<string, { title: string; color: string; desc: string }> = {
    nginx: {
      title: 'Nginx Engine',
      color: 'from-emerald-500 to-teal-600',
      desc: 'High performance event-driven reverse proxy & web server. Ultra-low RAM footprint.',
    },
    apache: {
      title: 'Apache HTTP Server',
      color: 'from-orange-500 to-amber-600',
      desc: 'Industry classic modular web server with native .htaccess rewrite engine & dynamic modules.',
    },
    openlitespeed: {
      title: 'OpenLiteSpeed',
      color: 'from-blue-500 to-indigo-600',
      desc: 'Ultra-fast event-driven web server with native LSPHP engine and built-in LiteSpeed Cache support.',
    },
    litespeed: {
      title: 'LiteSpeed Enterprise',
      color: 'from-purple-500 to-pink-600',
      desc: 'Commercial drop-in Apache replacement with HTTP/3, QUIC, and massive concurrency capability.',
    },
  };

  return (
    <DashboardShell>
      <div className="space-y-6 pb-16">
        {/* Header with Server Selector & Tabs */}
        <div className="flex flex-col md:flex-row md:items-center justify-between gap-4 border-b border-surface-800 pb-5">
          <div>
            <h1 className="text-2xl font-bold text-white tracking-tight flex items-center gap-2.5">
              <Layers className="w-6 h-6 text-indigo-400" />
              Web Server Management
            </h1>
            <p className="text-sm text-surface-400 mt-1">
              Production-grade orchestration for Nginx, Apache HTTP Server, OpenLiteSpeed & LiteSpeed Enterprise
            </p>
          </div>

          {/* Node Selector & Refresh */}
          <div className="flex items-center gap-3">
            <div className="flex items-center gap-2 bg-surface-900 border border-surface-800 rounded-lg px-3 py-1.5">
              <Server className="w-4 h-4 text-surface-400" />
              <select
                aria-label="Select managed server"
                value={selectedServerId}
                onChange={(e) => setSelectedServerId(e.target.value)}
                className="bg-transparent text-sm text-surface-200 outline-none cursor-pointer"
              >
                {servers.map((s) => (
                  <option key={s.id} value={s.id} className="bg-surface-900 text-white">
                    {s.name || s.hostname} ({s.ip_address})
                  </option>
                ))}
              </select>
            </div>

            <button
              onClick={() => selectedServerId && fetchServerData(selectedServerId)}
              disabled={loading}
              className="p-2 bg-surface-800 hover:bg-surface-700 text-surface-200 rounded-lg border border-surface-700 transition"
              title="Refresh telemetry"
            >
              <RefreshCw className={`w-4 h-4 ${loading ? 'animate-spin' : ''}`} />
            </button>
          </div>
        </div>

        {/* Global Toast Notification */}
        {toast && (
          <div
            className={`p-4 rounded-xl flex items-center justify-between border shadow-lg animate-in fade-in slide-in-from-top-2 ${
              toast.type === 'success'
                ? 'bg-emerald-950/80 border-emerald-500/30 text-emerald-300'
                : toast.type === 'error'
                ? 'bg-rose-950/80 border-rose-500/30 text-rose-300'
                : 'bg-indigo-950/80 border-indigo-500/30 text-indigo-300'
            }`}
          >
            <div className="flex items-center gap-2.5 text-sm font-medium">
              {toast.type === 'success' && <CheckCircle2 className="w-5 h-5 text-emerald-400" />}
              {toast.type === 'error' && <AlertTriangle className="w-5 h-5 text-rose-400" />}
              {toast.type === 'info' && <Activity className="w-5 h-5 text-indigo-400" />}
              <span>{toast.message}</span>
            </div>
            <button
              onClick={() => setToast(null)}
              className="text-xs opacity-70 hover:opacity-100 uppercase tracking-wider font-semibold ml-4"
            >
              Dismiss
            </button>
          </div>
        )}

        {/* Port Conflict Protection Alert */}
        {portConflicts.length > 0 && (
          <div className="bg-amber-950/60 border border-amber-500/40 rounded-xl p-4 flex flex-col md:flex-row items-start md:items-center justify-between gap-4">
            <div className="flex items-start gap-3">
              <div className="p-2 bg-amber-500/10 rounded-lg text-amber-400">
                <AlertTriangle className="w-5 h-5" />
              </div>
              <div>
                <h4 className="text-sm font-semibold text-amber-300">Port 80/443 Conflict Protection Active</h4>
                <p className="text-xs text-amber-400/80 mt-0.5">
                  Hostvra detected active daemon listeners on web traffic ports. Safe migration protection will verify port handoffs:
                </p>
                <div className="flex flex-wrap gap-2 mt-2">
                  {portConflicts.map((c, idx) => (
                    <span
                      key={idx}
                      className="text-xs px-2 py-0.5 rounded bg-amber-500/20 text-amber-200 border border-amber-500/30 font-mono"
                    >
                      Port {c.port}: {c.process_name} (PID {c.pid || 'Active'})
                    </span>
                  ))}
                </div>
              </div>
            </div>
            <button
              onClick={() => setActiveTab('migration')}
              className="px-3.5 py-1.5 bg-amber-600 hover:bg-amber-500 text-white text-xs font-medium rounded-lg shadow transition whitespace-nowrap"
            >
              Switch / Migrate Engines
            </button>
          </div>
        )}

        {/* Tab Navigation */}
        <div className="flex items-center gap-2 border-b border-surface-800">
          <button
            onClick={() => setActiveTab('overview')}
            className={`px-4 py-2.5 text-sm font-medium border-b-2 transition flex items-center gap-2 ${
              activeTab === 'overview'
                ? 'border-indigo-500 text-white'
                : 'border-transparent text-surface-400 hover:text-surface-200'
            }`}
          >
            <Cpu className="w-4 h-4" />
            Engines Overview
          </button>
          <button
            onClick={() => setActiveTab('vhosts')}
            className={`px-4 py-2.5 text-sm font-medium border-b-2 transition flex items-center gap-2 ${
              activeTab === 'vhosts'
                ? 'border-indigo-500 text-white'
                : 'border-transparent text-surface-400 hover:text-surface-200'
            }`}
          >
            <Globe className="w-4 h-4" />
            VirtualHosts ({websites.length})
          </button>
          <button
            onClick={() => setActiveTab('migration')}
            className={`px-4 py-2.5 text-sm font-medium border-b-2 transition flex items-center gap-2 ${
              activeTab === 'migration'
                ? 'border-indigo-500 text-white'
                : 'border-transparent text-surface-400 hover:text-surface-200'
            }`}
          >
            <ArrowRightLeft className="w-4 h-4" />
            Safe Engine Migration
          </button>
          <button
            onClick={() => setActiveTab('config')}
            className={`px-4 py-2.5 text-sm font-medium border-b-2 transition flex items-center gap-2 ${
              activeTab === 'config'
                ? 'border-indigo-500 text-white'
                : 'border-transparent text-surface-400 hover:text-surface-200'
            }`}
          >
            <FileCode2 className="w-4 h-4" />
            Master Config Editor
          </button>
        </div>

        {/* ------------------------------------------------------------------ */}
        {/* TAB 1: ENGINES OVERVIEW */}
        {/* ------------------------------------------------------------------ */}
        {activeTab === 'overview' && (
          <div className="space-y-6">
            <div className="grid grid-cols-1 md:grid-cols-2 gap-5">
              {(['nginx', 'apache', 'openlitespeed', 'litespeed'] as const).map((serverType) => {
                const ws = webServers.find((s) => s.server_type === serverType);
                const isInstalled = ws?.is_installed ?? false;
                const isRunning = ws?.is_running || ws?.status === 'running';
                const meta = serverMeta[serverType];

                return (
                  <div
                    key={serverType}
                    className="bg-surface-900 border border-surface-800 rounded-2xl p-5 relative overflow-hidden flex flex-col justify-between hover:border-surface-700 transition"
                  >
                    <div className="space-y-3">
                      {/* Card Header */}
                      <div className="flex items-center justify-between">
                        <div className="flex items-center gap-3">
                          <div className={`w-10 h-10 rounded-xl bg-gradient-to-br ${meta.color} flex items-center justify-center text-white shadow-lg`}>
                            <Layers className="w-5 h-5" />
                          </div>
                          <div>
                            <h3 className="text-base font-bold text-white">{meta.title}</h3>
                            <span className="text-xs text-surface-400 font-mono">
                              {isInstalled ? (ws?.version ? `v${ws.version}` : 'Installed') : 'Not Installed'}
                            </span>
                          </div>
                        </div>

                        {/* Status Badges */}
                        <div className="flex items-center gap-2">
                          {serverType === 'litespeed' && (
                            <span
                              className={`text-[11px] font-semibold px-2 py-0.5 rounded-full border ${
                                ws?.license_status === 'Active'
                                  ? 'bg-emerald-500/10 text-emerald-400 border-emerald-500/30'
                                  : 'bg-rose-500/10 text-rose-400 border-rose-500/30'
                              }`}
                            >
                              {ws?.license_status || 'Unlicensed'}
                            </span>
                          )}

                          <span
                            className={`text-xs font-semibold px-2.5 py-1 rounded-full flex items-center gap-1.5 border ${
                              isRunning
                                ? 'bg-emerald-500/10 text-emerald-400 border-emerald-500/30'
                                : isInstalled
                                ? 'bg-surface-800 text-surface-400 border-surface-700'
                                : 'bg-surface-800/40 text-surface-500 border-surface-800'
                            }`}
                          >
                            <span
                              className={`w-1.5 h-1.5 rounded-full ${
                                isRunning ? 'bg-emerald-400 animate-pulse' : 'bg-surface-500'
                              }`}
                            />
                            {isRunning ? 'Running' : isInstalled ? 'Stopped' : 'Not Installed'}
                          </span>
                        </div>
                      </div>

                      <p className="text-xs text-surface-400 leading-relaxed">{meta.desc}</p>

                      {/* Ports and Service Details */}
                      {isInstalled && (
                        <div className="bg-surface-950/60 rounded-xl p-3 border border-surface-800/80 space-y-1.5 text-xs">
                          <div className="flex justify-between text-surface-400">
                            <span>Service Daemon:</span>
                            <span className="font-mono text-surface-200">{ws?.service_name || serverType}</span>
                          </div>
                          <div className="flex justify-between text-surface-400">
                            <span>Port 80 (HTTP):</span>
                            <span className={ws?.http_port ? 'text-emerald-400 font-semibold' : 'text-surface-500'}>
                              {ws?.http_port ? 'Bound & Serving' : 'Inactive'}
                            </span>
                          </div>
                          <div className="flex justify-between text-surface-400">
                            <span>Port 443 (HTTPS):</span>
                            <span className={ws?.https_port ? 'text-emerald-400 font-semibold' : 'text-surface-500'}>
                              {ws?.https_port ? 'Bound (SSL Active)' : 'Inactive'}
                            </span>
                          </div>
                          {ws?.config_path && (
                            <div className="flex justify-between text-surface-400 truncate">
                              <span>Config:</span>
                              <span className="font-mono text-surface-300 truncate max-w-[200px]" title={ws.config_path}>
                                {ws.config_path}
                              </span>
                            </div>
                          )}
                        </div>
                      )}
                    </div>

                    {/* Action Controls */}
                    <div className="pt-4 mt-4 border-t border-surface-800 flex items-center justify-between gap-2">
                      {isInstalled ? (
                        <>
                          <div className="flex items-center gap-1.5">
                            {isRunning ? (
                              <button
                                onClick={() => handleServiceAction(serverType, 'stop')}
                                disabled={actionLoading !== null}
                                className="px-3 py-1.5 bg-surface-800 hover:bg-surface-700 text-rose-400 text-xs font-medium rounded-lg border border-surface-700 transition flex items-center gap-1"
                              >
                                <Square className="w-3.5 h-3.5" />
                                Stop
                              </button>
                            ) : (
                              <button
                                onClick={() => handleServiceAction(serverType, 'start')}
                                disabled={actionLoading !== null}
                                className="px-3 py-1.5 bg-emerald-600 hover:bg-emerald-500 text-white text-xs font-medium rounded-lg transition flex items-center gap-1 shadow"
                              >
                                <Play className="w-3.5 h-3.5" />
                                Start
                              </button>
                            )}

                            <button
                              onClick={() => handleServiceAction(serverType, 'restart')}
                              disabled={actionLoading !== null || !isRunning}
                              className="px-3 py-1.5 bg-surface-800 hover:bg-surface-700 text-surface-200 text-xs font-medium rounded-lg border border-surface-700 transition flex items-center gap-1 disabled:opacity-50"
                            >
                              <RotateCw className="w-3.5 h-3.5" />
                              Restart
                            </button>

                            <button
                              onClick={() => handleServiceAction(serverType, 'reload')}
                              disabled={actionLoading !== null || !isRunning}
                              className="px-3 py-1.5 bg-surface-800 hover:bg-surface-700 text-surface-200 text-xs font-medium rounded-lg border border-surface-700 transition flex items-center gap-1 disabled:opacity-50"
                            >
                              <RefreshCw className="w-3.5 h-3.5" />
                              Reload
                            </button>
                          </div>

                          <button
                            onClick={() => handleUninstall(serverType)}
                            disabled={actionLoading !== null}
                            className="text-xs text-surface-500 hover:text-rose-400 transition underline underline-offset-2"
                          >
                            Uninstall
                          </button>
                        </>
                      ) : (
                        <button
                          onClick={() => handleInstall(serverType)}
                          disabled={actionLoading !== null}
                          className="w-full py-2 bg-indigo-600 hover:bg-indigo-500 text-white text-xs font-semibold rounded-lg shadow transition flex items-center justify-center gap-1.5"
                        >
                          <Cpu className="w-3.5 h-3.5" />
                          Install {meta.title}
                        </button>
                      )}
                    </div>
                  </div>
                );
              })}
            </div>
          </div>
        )}

        {/* ------------------------------------------------------------------ */}
        {/* TAB 2: VIRTUALHOSTS EXPLORER */}
        {/* ------------------------------------------------------------------ */}
        {activeTab === 'vhosts' && (
          <div className="bg-surface-900 border border-surface-800 rounded-2xl p-5 space-y-4">
            <div className="flex items-center justify-between">
              <div>
                <h3 className="text-base font-bold text-white">Active VirtualHosts</h3>
                <p className="text-xs text-surface-400">
                  Inspect or switch web server engines individually per website
                </p>
              </div>
            </div>

            {websites.length === 0 ? (
              <div className="p-8 text-center text-surface-500 text-sm">
                No websites currently deployed on this server.
              </div>
            ) : (
              <div className="overflow-x-auto">
                <table className="w-full text-left text-xs">
                  <thead className="border-b border-surface-800 text-surface-400 uppercase tracking-wider text-[11px]">
                    <tr>
                      <th className="py-3 px-3">Domain</th>
                      <th className="py-3 px-3">Engine</th>
                      <th className="py-3 px-3">App Type</th>
                      <th className="py-3 px-3">Document Root</th>
                      <th className="py-3 px-3">SSL</th>
                      <th className="py-3 px-3 text-right">Actions</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-surface-800/60 text-surface-200">
                    {websites.map((site) => (
                      <tr key={site.id} className="hover:bg-surface-800/30 transition">
                        <td className="py-3 px-3 font-semibold text-white">
                          {site.primary_domain}
                        </td>
                        <td className="py-3 px-3">
                          <select
                            aria-label={`Select engine for ${site.primary_domain}`}
                            value={site.web_server_type || 'nginx'}
                            onChange={(e) => handleSwitchWebsite(site.id, e.target.value)}
                            disabled={actionLoading === `switch:${site.id}`}
                            className="bg-surface-800 border border-surface-700 rounded-lg px-2.5 py-1 text-xs text-indigo-300 outline-none cursor-pointer"
                          >
                            <option value="nginx">Nginx</option>
                            <option value="apache">Apache</option>
                            <option value="openlitespeed">OpenLiteSpeed</option>
                            <option value="litespeed">LiteSpeed Enterprise</option>
                          </select>
                        </td>
                        <td className="py-3 px-3 uppercase text-[10px] tracking-wider text-surface-400 font-mono">
                          {site.app_type}
                        </td>
                        <td className="py-3 px-3 font-mono text-surface-400 truncate max-w-[220px]">
                          {site.document_root}
                        </td>
                        <td className="py-3 px-3">
                          {site.ssl_enabled ? (
                            <span className="text-emerald-400 flex items-center gap-1">
                              <Shield className="w-3.5 h-3.5" /> HTTPS
                            </span>
                          ) : (
                            <span className="text-surface-500">HTTP</span>
                          )}
                        </td>
                        <td className="py-3 px-3 text-right space-x-2">
                          <button
                            onClick={() => handleInspectVHost(site)}
                            className="px-2.5 py-1 bg-surface-800 hover:bg-surface-700 text-surface-300 rounded border border-surface-700 text-xs transition"
                          >
                            Inspect VHost
                          </button>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </div>
        )}

        {/* ------------------------------------------------------------------ */}
        {/* TAB 3: SAFE ENGINE MIGRATION */}
        {/* ------------------------------------------------------------------ */}
        {activeTab === 'migration' && (
          <div className="bg-surface-900 border border-surface-800 rounded-2xl p-6 space-y-6">
            <div>
              <h3 className="text-lg font-bold text-white flex items-center gap-2">
                <ArrowRightLeft className="w-5 h-5 text-indigo-400" />
                Atomic Zero-Downtime Web Server Migration
              </h3>
              <p className="text-xs text-surface-400 mt-1">
                Safely migrate all websites and reverse proxy configurations between web servers with automated rollback on failure.
              </p>
            </div>

            {/* Migration Engine Chooser */}
            <div className="grid grid-cols-1 md:grid-cols-2 gap-6 bg-surface-950/60 p-5 rounded-xl border border-surface-800">
              <div className="space-y-2">
                <label className="text-xs font-semibold text-surface-300 uppercase tracking-wider">
                  Source Web Server
                </label>
                <select
                  value={sourceServer}
                  onChange={(e) => setSourceServer(e.target.value)}
                  className="w-full bg-surface-800 border border-surface-700 rounded-lg p-2.5 text-sm text-white outline-none"
                >
                  <option value="nginx">Nginx (Current Active)</option>
                  <option value="apache">Apache HTTP Server</option>
                  <option value="openlitespeed">OpenLiteSpeed</option>
                  <option value="litespeed">LiteSpeed Enterprise</option>
                </select>
              </div>

              <div className="space-y-2">
                <label className="text-xs font-semibold text-surface-300 uppercase tracking-wider">
                  Target Web Server
                </label>
                <select
                  value={targetServer}
                  onChange={(e) => setTargetServer(e.target.value)}
                  className="w-full bg-surface-800 border border-surface-700 rounded-lg p-2.5 text-sm text-white outline-none"
                >
                  <option value="apache">Apache HTTP Server</option>
                  <option value="nginx">Nginx</option>
                  <option value="openlitespeed">OpenLiteSpeed</option>
                  <option value="litespeed">LiteSpeed Enterprise</option>
                </select>
              </div>
            </div>

            {/* Migration Pipeline Visualization */}
            <div className="space-y-3">
              <h4 className="text-xs font-semibold text-surface-300 uppercase tracking-wider">
                Safety Guarantee Pipeline
              </h4>
              <div className="grid grid-cols-1 sm:grid-cols-5 gap-3 text-xs">
                <div className="bg-surface-800/60 p-3 rounded-lg border border-surface-700/60 space-y-1">
                  <span className="font-bold text-indigo-400">1. Preflight</span>
                  <p className="text-surface-400 text-[11px]">Verifies target daemon & license authenticity</p>
                </div>
                <div className="bg-surface-800/60 p-3 rounded-lg border border-surface-700/60 space-y-1">
                  <span className="font-bold text-indigo-400">2. Backup</span>
                  <p className="text-surface-400 text-[11px]">Takes snapshot of current configs for rollback</p>
                </div>
                <div className="bg-surface-800/60 p-3 rounded-lg border border-surface-700/60 space-y-1">
                  <span className="font-bold text-indigo-400">3. VHost Generation</span>
                  <p className="text-surface-400 text-[11px]">Generates syntax-checked configs for {websites.length} sites</p>
                </div>
                <div className="bg-surface-800/60 p-3 rounded-lg border border-surface-700/60 space-y-1">
                  <span className="font-bold text-indigo-400">4. Port Handover</span>
                  <p className="text-surface-400 text-[11px]">Stops source & starts target on 80/443 atomically</p>
                </div>
                <div className="bg-surface-800/60 p-3 rounded-lg border border-surface-700/60 space-y-1">
                  <span className="font-bold text-indigo-400">5. Health Probe</span>
                  <p className="text-surface-400 text-[11px]">Probes connectivity or automatically rolls back</p>
                </div>
              </div>
            </div>

            {/* Action Button */}
            <div className="pt-2">
              <button
                onClick={handleExecuteMigration}
                disabled={migrating || sourceServer === targetServer}
                className="px-6 py-2.5 bg-gradient-to-r from-indigo-600 to-purple-600 hover:from-indigo-500 hover:to-purple-500 text-white text-sm font-semibold rounded-xl shadow-lg transition flex items-center gap-2 disabled:opacity-50 cursor-pointer"
              >
                {migrating ? (
                  <>
                    <RefreshCw className="w-4 h-4 animate-spin" />
                    Executing Migration Pipeline...
                  </>
                ) : (
                  <>
                    <ArrowRightLeft className="w-4 h-4" />
                    Start Migration: {sourceServer} → {targetServer}
                  </>
                )}
              </button>
            </div>

            {/* Migration Result Report */}
            {migrationResult && (
              <div
                className={`p-4 rounded-xl border space-y-2 ${
                  migrationResult.success
                    ? 'bg-emerald-950/60 border-emerald-500/40 text-emerald-200'
                    : 'bg-rose-950/60 border-rose-500/40 text-rose-200'
                }`}
              >
                <div className="flex items-center gap-2 font-semibold text-sm">
                  {migrationResult.success ? <CheckCircle2 className="w-5 h-5" /> : <XCircle className="w-5 h-5" />}
                  <span>{migrationResult.message}</span>
                </div>
                {migrationResult.migrated_list && (
                  <p className="text-xs text-surface-300">
                    Migrated Domains: {migrationResult.migrated_list.join(', ')}
                  </p>
                )}
                {migrationResult.rolled_back && (
                  <p className="text-xs font-semibold text-amber-300">
                    System was safely rolled back to {sourceServer}. No websites experienced permanent outage.
                  </p>
                )}
              </div>
            )}
          </div>
        )}

        {/* ------------------------------------------------------------------ */}
        {/* TAB 4: MASTER CONFIG EDITOR */}
        {/* ------------------------------------------------------------------ */}
        {activeTab === 'config' && (
          <div className="bg-surface-900 border border-surface-800 rounded-2xl p-6 space-y-4">
            <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3">
              <div>
                <h3 className="text-base font-bold text-white flex items-center gap-2">
                  <FileCode2 className="w-5 h-5 text-indigo-400" />
                  Master Configuration File
                </h3>
                <p className="text-xs text-surface-400">
                  Direct editor with automated test validation. Syntax errors trigger instant rollback.
                </p>
              </div>

              <div className="flex items-center gap-2">
                <select
                  aria-label="Select configuration engine"
                  value={configServerType}
                  onChange={(e) => setConfigServerType(e.target.value)}
                  className="bg-surface-800 border border-surface-700 text-surface-200 text-xs rounded-lg px-3 py-1.5 outline-none cursor-pointer"
                >
                  <option value="nginx">Nginx (nginx.conf)</option>
                  <option value="apache">Apache (apache2.conf / httpd.conf)</option>
                  <option value="openlitespeed">OpenLiteSpeed (httpd_config.conf)</option>
                  <option value="litespeed">LiteSpeed Enterprise</option>
                </select>

                <button
                  onClick={handleSaveConfig}
                  disabled={configSaving || configLoading}
                  className="px-4 py-1.5 bg-indigo-600 hover:bg-indigo-500 text-white text-xs font-semibold rounded-lg shadow transition flex items-center gap-1.5 disabled:opacity-50"
                >
                  {configSaving ? <RefreshCw className="w-3.5 h-3.5 animate-spin" /> : <Check className="w-3.5 h-3.5" />}
                  Verify & Save
                </button>
              </div>
            </div>

            {configLoading ? (
              <div className="h-64 flex items-center justify-center text-surface-500 text-xs">
                <RefreshCw className="w-5 h-5 animate-spin mr-2" />
                Reading master configuration...
              </div>
            ) : (
              <textarea
                value={configContent}
                onChange={(e) => setConfigContent(e.target.value)}
                rows={20}
                spellCheck={false}
                className="w-full bg-surface-950 font-mono text-xs text-surface-200 p-4 rounded-xl border border-surface-800 outline-none focus:border-indigo-500 transition resize-y"
              />
            )}
          </div>
        )}

        {/* VHost Inspection Modal */}
        {viewingVHost && (
          <div className="fixed inset-0 bg-black/70 backdrop-blur-sm flex items-center justify-center z-50 p-4">
            <div className="bg-surface-900 border border-surface-800 rounded-2xl max-w-3xl w-full max-h-[85vh] flex flex-col shadow-2xl animate-in zoom-in-95">
              <div className="p-5 border-b border-surface-800 flex items-center justify-between">
                <div>
                  <h3 className="text-base font-bold text-white">{viewingVHost.domain}</h3>
                  <span className="text-xs text-indigo-400 font-mono uppercase">
                    VirtualHost for {viewingVHost.serverType}
                  </span>
                </div>
                <button
                  onClick={() => setViewingVHost(null)}
                  className="p-1 text-surface-400 hover:text-white rounded"
                >
                  ✕
                </button>
              </div>

              <div className="p-5 overflow-y-auto flex-1">
                <pre className="bg-surface-950 p-4 rounded-xl text-xs font-mono text-surface-200 overflow-x-auto whitespace-pre">
                  {viewingVHost.content}
                </pre>
              </div>

              <div className="p-4 border-t border-surface-800 flex justify-end">
                <button
                  onClick={() => setViewingVHost(null)}
                  className="px-4 py-1.5 bg-surface-800 hover:bg-surface-700 text-surface-200 text-xs font-medium rounded-lg"
                >
                  Close
                </button>
              </div>
            </div>
          </div>
        )}
      </div>
    </DashboardShell>
  );
}
