'use client';

import React, { useState, useEffect } from 'react';
import Link from 'next/link';
import {
  Server,
  Activity,
  Globe,
  ShieldCheck,
  Plus,
  ArrowUpRight,
  HardDrive,
  Cpu,
  Layers,
  ScrollText,
  Clock,
  CheckCircle2,
  AlertTriangle,
  Boxes,
  Play,
  Square,
  RotateCw,
  ExternalLink,
  Bookmark,
  BookmarkCheck,
  Sparkles,
  Database,
  Code2,
  Shield,
  Wrench,
  Mail,
  Sliders,
} from 'lucide-react';
import { DashboardShell } from '@/components/DashboardShell';
import { apiFetch, Server as ServerModel, AuditLog, AppPackage } from '@/lib/api';
import { AppControlModal } from '@/components/AppControlModal';
import { getAppLaunchTarget, getPinnedAppIds, togglePinApp } from '@/lib/appstore-utils';

export default function DashboardPage() {
  const [servers, setServers] = useState<ServerModel[]>([]);
  const [auditLogs, setAuditLogs] = useState<AuditLog[]>([]);
  const [apps, setApps] = useState<AppPackage[]>([]);
  const [loading, setLoading] = useState(true);
  const [actionLoadingId, setActionLoadingId] = useState<string | null>(null);
  const [selectedControlApp, setSelectedControlApp] = useState<AppPackage | null>(null);
  const [controlModalOpen, setControlModalOpen] = useState(false);
  const [pinnedAppIds, setPinnedAppIds] = useState<string[]>([]);
  const [appFilter, setAppFilter] = useState<'all' | 'pinned'>('all');

  const fetchData = async () => {
    try {
      const [serversRes, auditRes, appsRes] = await Promise.all([
        apiFetch<ServerModel[]>('/api/v1/servers'),
        apiFetch<AuditLog[]>('/api/v1/audit-logs?limit=5'),
        apiFetch<AppPackage[]>('/api/v1/apps'),
      ]);

      if (serversRes.success && serversRes.data) {
        setServers(serversRes.data);
      }
      if (auditRes.success && auditRes.data) {
        setAuditLogs(auditRes.data);
      }
      if (appsRes.success && appsRes.data) {
        setApps(appsRes.data);
      }
    } catch (err) {
      console.error('Failed to load dashboard data', err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchData();
    setPinnedAppIds(getPinnedAppIds());

    const handlePinnedChange = () => setPinnedAppIds(getPinnedAppIds());
    window.addEventListener('hostvra_pinned_apps_changed', handlePinnedChange);
    return () => window.removeEventListener('hostvra_pinned_apps_changed', handlePinnedChange);
  }, []);

  const handleServiceControl = async (app: AppPackage, action: 'start' | 'stop' | 'restart') => {
    setActionLoadingId(app.id);
    try {
      const res = await apiFetch<AppPackage>(`/api/v1/apps/${app.id}/service`, {
        method: 'POST',
        body: JSON.stringify({ action }),
      });
      if (res.success && res.data) {
        // Update local state
        setApps((prev) => prev.map((a) => (a.id === app.id ? res.data! : a)));
        if (selectedControlApp?.id === app.id) {
          setSelectedControlApp(res.data);
        }
      }
    } finally {
      setActionLoadingId(null);
    }
  };

  const handleOpenApp = (app: AppPackage) => {
    setSelectedControlApp(app);
    setControlModalOpen(true);
  };

  const onlineServers = servers.filter((s) => s.status === 'online').length;
  const installedApps = apps.filter((a) => a.is_installed);
  const runningApps = installedApps.filter((a) => a.status === 'running');
  const displayedApps = appFilter === 'pinned'
    ? installedApps.filter((a) => pinnedAppIds.includes(a.id))
    : installedApps;

  const getCategoryIcon = (category: string) => {
    switch (category) {
      case 'process_manager':
        return <Cpu className="w-4 h-4 text-indigo-400" />;
      case 'web_server':
        return <Server className="w-4 h-4 text-emerald-400" />;
      case 'database':
        return <Database className="w-4 h-4 text-amber-400" />;
      case 'runtime':
        return <Code2 className="w-4 h-4 text-blue-400" />;
      case 'security':
        return <Shield className="w-4 h-4 text-purple-400" />;
      case 'monitoring':
        return <Activity className="w-4 h-4 text-rose-400" />;
      case 'mail':
        return <Mail className="w-4 h-4 text-sky-400" />;
      default:
        return <Wrench className="w-4 h-4 text-teal-400" />;
    }
  };

  return (
    <DashboardShell>
      <div className="space-y-8">
        {/* Page Header */}
        <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
          <div>
            <h1 className="text-2xl font-bold tracking-tight text-white">Cluster Overview</h1>
            <p className="text-sm text-slate-400 mt-1">
              Real-time telemetry, server fleet, and software service controls.
            </p>
          </div>
          <div className="flex items-center gap-3">
            <Link
              href="/app-store"
              className="inline-flex items-center gap-2 px-4 py-2.5 rounded-xl bg-surface-800 hover:bg-surface-700 text-slate-200 text-sm font-semibold border border-surface-700 transition-all shadow-sm"
            >
              <Boxes className="w-4 h-4 text-indigo-400" />
              1-Click App Store
            </Link>
            <Link
              href="/servers"
              className="inline-flex items-center gap-2 px-4 py-2.5 rounded-xl bg-indigo-600 hover:bg-indigo-500 text-white text-sm font-semibold shadow-lg shadow-indigo-600/25 transition-all"
            >
              <Plus className="w-4 h-4" />
              Enroll Server
            </Link>
          </div>
        </div>

        {/* Fleet KPI Metric Cards */}
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-5">
          <div className="p-5 rounded-2xl bg-surface-900 border border-surface-800 shadow-xl flex items-center justify-between">
            <div>
              <p className="text-xs font-semibold text-slate-400 uppercase tracking-wider">Managed Nodes</p>
              <div className="flex items-baseline gap-2 mt-2">
                <span className="text-3xl font-extrabold text-white">{servers.length}</span>
                <span className="text-xs text-emerald-400 font-medium">
                  {onlineServers} Online
                </span>
              </div>
            </div>
            <div className="w-12 h-12 rounded-xl bg-indigo-500/10 border border-indigo-500/20 flex items-center justify-center text-indigo-400">
              <Server className="w-6 h-6" />
            </div>
          </div>

          <div className="p-5 rounded-2xl bg-surface-900 border border-surface-800 shadow-xl flex items-center justify-between">
            <div>
              <p className="text-xs font-semibold text-slate-400 uppercase tracking-wider">Websites & Vhosts</p>
              <div className="flex items-baseline gap-2 mt-2">
                <span className="text-3xl font-extrabold text-white">0</span>
                <span className="text-xs text-slate-400">Nginx Managed</span>
              </div>
            </div>
            <div className="w-12 h-12 rounded-xl bg-emerald-500/10 border border-emerald-500/20 flex items-center justify-center text-emerald-400">
              <Globe className="w-6 h-6" />
            </div>
          </div>

          <div className="p-5 rounded-2xl bg-surface-900 border border-surface-800 shadow-xl flex items-center justify-between">
            <div>
              <p className="text-xs font-semibold text-slate-400 uppercase tracking-wider">Installed Software</p>
              <div className="flex items-baseline gap-2 mt-2">
                <span className="text-3xl font-extrabold text-white">{installedApps.length}</span>
                <span className="text-xs text-emerald-400 font-medium">
                  {runningApps.length} Running
                </span>
              </div>
            </div>
            <div className="w-12 h-12 rounded-xl bg-purple-500/10 border border-purple-500/20 flex items-center justify-center text-purple-400">
              <Boxes className="w-6 h-6" />
            </div>
          </div>

          <div className="p-5 rounded-2xl bg-surface-900 border border-surface-800 shadow-xl flex items-center justify-between">
            <div>
              <p className="text-xs font-semibold text-slate-400 uppercase tracking-wider">SSL Certificates</p>
              <div className="flex items-baseline gap-2 mt-2">
                <span className="text-3xl font-extrabold text-white">100%</span>
                <span className="text-xs text-emerald-400">Automated ACME</span>
              </div>
            </div>
            <div className="w-12 h-12 rounded-xl bg-sky-500/10 border border-sky-500/20 flex items-center justify-center text-sky-400">
              <ShieldCheck className="w-6 h-6" />
            </div>
          </div>
        </div>

        {/* Installed Applications & Services Control Widget */}
        <div className="bg-surface-900 border border-surface-800 rounded-2xl p-6 shadow-xl space-y-5">
          <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
            <div className="flex items-center gap-3">
              <div className="w-10 h-10 rounded-xl bg-indigo-600/10 border border-indigo-500/20 flex items-center justify-center text-indigo-400 shadow-inner">
                <Boxes className="w-5 h-5" />
              </div>
              <div>
                <h2 className="text-lg font-bold text-white tracking-tight flex items-center gap-2">
                  Installed Applications & Control
                  <span className="text-xs px-2 py-0.5 rounded-full bg-indigo-500/20 text-indigo-300 font-medium">
                    {installedApps.length} active
                  </span>
                </h2>
                <p className="text-xs text-slate-400">
                  Control, start/stop, restart, and launch your installed server packages directly from the dashboard.
                </p>
              </div>
            </div>

            <div className="flex items-center gap-3">
              {/* Filter Tabs (All Installed vs Pinned) */}
              <div className="inline-flex rounded-xl bg-surface-950 p-1 border border-surface-800 text-xs">
                <button
                  onClick={() => setAppFilter('all')}
                  className={`px-3 py-1.5 rounded-lg font-medium transition-all ${
                    appFilter === 'all'
                      ? 'bg-surface-800 text-white shadow-sm'
                      : 'text-slate-400 hover:text-white'
                  }`}
                >
                  All ({installedApps.length})
                </button>
                <button
                  onClick={() => setAppFilter('pinned')}
                  className={`px-3 py-1.5 rounded-lg font-medium transition-all flex items-center gap-1.5 ${
                    appFilter === 'pinned'
                      ? 'bg-amber-500/20 text-amber-300 shadow-sm'
                      : 'text-slate-400 hover:text-white'
                  }`}
                >
                  <Bookmark className="w-3 h-3 text-amber-400" />
                  Pinned ({pinnedAppIds.length})
                </button>
              </div>

              <Link
                href="/app-store"
                className="text-xs font-medium text-indigo-400 hover:text-indigo-300 flex items-center gap-1 transition-colors whitespace-nowrap"
              >
                <span>App Store</span>
                <ArrowUpRight className="w-3.5 h-3.5" />
              </Link>
            </div>
          </div>

          {/* Apps Cards Grid */}
          {loading ? (
            <div className="py-12 text-center text-slate-400 text-sm">
              <div className="w-6 h-6 border-2 border-indigo-500 border-t-transparent rounded-full animate-spin mx-auto mb-2" />
              Loading installed applications...
            </div>
          ) : displayedApps.length === 0 ? (
            <div className="py-10 px-4 rounded-xl border border-dashed border-surface-800 text-center bg-surface-950/40">
              <div className="w-12 h-12 rounded-2xl bg-surface-800 flex items-center justify-center mx-auto mb-3 text-slate-400">
                <Boxes className="w-6 h-6 text-slate-400" />
              </div>
              <h3 className="text-sm font-semibold text-slate-200">
                {appFilter === 'pinned' ? 'No pinned apps on dashboard' : 'No applications installed yet'}
              </h3>
              <p className="text-xs text-slate-400 mt-1 max-w-sm mx-auto">
                {appFilter === 'pinned'
                  ? 'Pin your favorite applications from the list or App Store to have quick one-click access here.'
                  : 'Install Docker, Nginx, Redis, Supervisor, PHP, or databases in 1-click from the App Store.'}
              </p>
              <Link
                href="/app-store"
                className="mt-4 inline-flex items-center gap-2 px-4 py-2 rounded-xl bg-indigo-600 hover:bg-indigo-500 text-white text-xs font-semibold shadow-md transition-all"
              >
                <Sparkles className="w-3.5 h-3.5" />
                Browse 1-Click App Store
              </Link>
            </div>
          ) : (
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
              {displayedApps.map((app) => {
                const isActing = actionLoadingId === app.id;
                const isPinned = pinnedAppIds.includes(app.id);
                const isRunning = app.status === 'running';

                return (
                  <div
                    key={app.id}
                    className="p-4 rounded-xl bg-surface-950/60 border border-surface-800 hover:border-surface-700 transition-all flex flex-col justify-between group shadow-sm"
                  >
                    <div>
                      {/* Card Top: Icon, Title, Pin & Status */}
                      <div className="flex items-start justify-between gap-3">
                        <div className="flex items-center gap-3">
                          <div className="w-9 h-9 rounded-lg bg-surface-900 border border-surface-750 flex items-center justify-center flex-shrink-0">
                            {getCategoryIcon(app.category)}
                          </div>
                          <div>
                            <h3 className="text-sm font-bold text-white tracking-tight leading-snug">
                              {app.name}
                            </h3>
                            <span className="text-[10px] text-slate-400 font-mono">
                              v{app.version}
                            </span>
                          </div>
                        </div>

                        {/* Pin Button */}
                        <button
                          onClick={() => {
                            togglePinApp(app.id);
                            setPinnedAppIds(getPinnedAppIds());
                          }}
                          className={`p-1.5 rounded-lg border transition-all ${
                            isPinned
                              ? 'bg-amber-500/10 text-amber-400 border-amber-500/30'
                              : 'text-slate-500 border-transparent hover:text-slate-300 hover:bg-surface-800'
                          }`}
                          title={isPinned ? 'Unpin from Dashboard' : 'Pin to Dashboard'}
                        >
                          {isPinned ? (
                            <BookmarkCheck className="w-3.5 h-3.5 text-amber-400" />
                          ) : (
                            <Bookmark className="w-3.5 h-3.5" />
                          )}
                        </button>
                      </div>

                      {/* Status indicator & Description */}
                      <div className="mt-3 flex items-center gap-2">
                        <span
                          className={`inline-flex items-center gap-1.5 px-2 py-0.5 rounded-full text-[11px] font-semibold ${
                            isRunning
                              ? 'bg-emerald-500/10 text-emerald-400 border border-emerald-500/20'
                              : 'bg-amber-500/10 text-amber-400 border border-amber-500/20'
                          }`}
                        >
                          <span
                            className={`w-1.5 h-1.5 rounded-full ${
                              isRunning ? 'bg-emerald-400 animate-pulse' : 'bg-amber-400'
                            }`}
                          />
                          {isRunning ? 'Running' : 'Stopped'}
                        </span>
                        <span className="text-[11px] text-slate-500 capitalize">
                          {app.category.replace('_', ' ')}
                        </span>
                      </div>

                      <p className="text-xs text-slate-400 mt-2 line-clamp-2 leading-relaxed">
                        {app.description}
                      </p>
                    </div>

                    {/* Card Actions Footer: Service Switch & Open Button */}
                    <div className="mt-4 pt-3 border-t border-surface-800/80 flex items-center justify-between gap-2">
                      <div className="flex items-center gap-2">
                        {/* Service Start/Stop Toggle */}
                        {app.service_name && (
                          <button
                            onClick={() =>
                              handleServiceControl(app, isRunning ? 'stop' : 'start')
                            }
                            disabled={isActing}
                            title={isRunning ? 'Click to Stop Service' : 'Click to Start Service'}
                            className={`relative inline-flex h-5 w-9 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none ${
                              isRunning ? 'bg-emerald-500' : 'bg-slate-700'
                            } disabled:opacity-50`}
                          >
                            <span
                              className={`pointer-events-none inline-block h-4 w-4 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out ${
                                isRunning ? 'translate-x-4' : 'translate-x-0'
                              }`}
                            />
                          </button>
                        )}

                        {/* Restart Button */}
                        {app.service_name && (
                          <button
                            onClick={() => handleServiceControl(app, 'restart')}
                            disabled={isActing}
                            title="Restart Service"
                            className="p-1 rounded-md text-slate-400 hover:text-indigo-400 hover:bg-surface-800 transition-colors disabled:opacity-50"
                          >
                            <RotateCw className="w-3.5 h-3.5" />
                          </button>
                        )}
                      </div>

                      {/* Open & Manage Button */}
                      <button
                        onClick={() => handleOpenApp(app)}
                        className="px-3 py-1 rounded-lg bg-indigo-600 hover:bg-indigo-500 text-white font-semibold text-xs shadow-sm flex items-center gap-1.5 transition-all active:scale-95"
                      >
                        <ExternalLink className="w-3 h-3" />
                        Open
                      </button>
                    </div>
                  </div>
                );
              })}
            </div>
          )}
        </div>

        {/* Server Fleet Grid / Table Preview */}
        <div className="bg-surface-900 border border-surface-800 rounded-2xl p-6 shadow-xl">
          <div className="flex items-center justify-between mb-5">
            <div className="flex items-center gap-2">
              <Server className="w-5 h-5 text-indigo-400" />
              <h2 className="text-lg font-bold text-white">Registered Server Fleet</h2>
            </div>
            <Link
              href="/servers"
              className="text-xs font-medium text-indigo-400 hover:text-indigo-300 flex items-center gap-1 transition-colors"
            >
              <span>View all servers</span>
              <ArrowUpRight className="w-3.5 h-3.5" />
            </Link>
          </div>

          {loading ? (
            <div className="py-12 text-center text-slate-400 text-sm">
              <div className="w-6 h-6 border-2 border-indigo-500 border-t-transparent rounded-full animate-spin mx-auto mb-2" />
              Loading server telemetry...
            </div>
          ) : servers.length === 0 ? (
            <div className="py-12 px-4 rounded-xl border border-dashed border-surface-700 text-center bg-surface-950/40">
              <div className="w-12 h-12 rounded-2xl bg-surface-800 flex items-center justify-center mx-auto mb-3 text-slate-400">
                <Server className="w-6 h-6" />
              </div>
              <h3 className="text-base font-semibold text-slate-200">No servers enrolled yet</h3>
              <p className="text-xs text-slate-400 mt-1 max-w-sm mx-auto">
                Connect your first Linux VPS or dedicated server using the Hostvra one-command installer.
              </p>
              <Link
                href="/servers"
                className="mt-4 inline-flex items-center gap-2 px-4 py-2 rounded-xl bg-indigo-600 hover:bg-indigo-500 text-white text-xs font-semibold shadow-md transition-all"
              >
                <Plus className="w-3.5 h-3.5" />
                Add Your First Server
              </Link>
            </div>
          ) : (
            <div className="overflow-x-auto">
              <table className="w-full text-left text-sm">
                <thead>
                  <tr className="border-b border-surface-800 text-slate-400 text-xs uppercase tracking-wider">
                    <th className="pb-3 font-semibold">Server Name</th>
                    <th className="pb-3 font-semibold">IP Address</th>
                    <th className="pb-3 font-semibold">OS / Arch</th>
                    <th className="pb-3 font-semibold">Status</th>
                    <th className="pb-3 font-semibold">Agent</th>
                    <th className="pb-3 font-semibold text-right">Actions</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-surface-800/60">
                  {servers.map((server) => (
                    <tr key={server.id} className="hover:bg-surface-800/40 transition-colors">
                      <td className="py-3.5 font-medium text-white flex items-center gap-2.5">
                        <span className={`w-2 h-2 rounded-full ${server.status === 'online' ? 'bg-emerald-500' : 'bg-slate-500'}`} />
                        <div>
                          <div>{server.name}</div>
                          <div className="text-xs text-slate-400 font-mono font-normal">{server.hostname}</div>
                        </div>
                      </td>
                      <td className="py-3.5 font-mono text-xs text-slate-300">{server.ip_address}</td>
                      <td className="py-3.5 text-xs text-slate-300">
                        {server.os_name} {server.os_version} ({server.architecture})
                      </td>
                      <td className="py-3.5">
                        <span
                          className={`inline-flex items-center px-2 py-0.5 rounded text-[11px] font-semibold capitalize ${
                            server.status === 'online'
                              ? 'bg-emerald-500/10 text-emerald-400 border border-emerald-500/20'
                              : 'bg-slate-500/10 text-slate-400 border border-slate-500/20'
                          }`}
                        >
                          {server.status}
                        </span>
                      </td>
                      <td className="py-3.5 text-xs font-mono text-slate-400">v{server.agent_version}</td>
                      <td className="py-3.5 text-right">
                        <Link
                          href={`/servers?id=${server.id}`}
                          className="text-xs text-indigo-400 hover:text-indigo-300 font-medium"
                        >
                          Manage →
                        </Link>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </div>

        {/* Security & Audit Trail Preview */}
        <div className="bg-surface-900 border border-surface-800 rounded-2xl p-6 shadow-xl">
          <div className="flex items-center justify-between mb-4">
            <div className="flex items-center gap-2">
              <ScrollText className="w-5 h-5 text-indigo-400" />
              <h2 className="text-lg font-bold text-white">Recent Security Audit Trail</h2>
            </div>
            <Link
              href="/audit-logs"
              className="text-xs font-medium text-indigo-400 hover:text-indigo-300 flex items-center gap-1 transition-colors"
            >
              <span>Full Audit Stream</span>
              <ArrowUpRight className="w-3.5 h-3.5" />
            </Link>
          </div>

          {auditLogs.length === 0 ? (
            <p className="text-xs text-slate-400 py-4">No recent security events recorded.</p>
          ) : (
            <div className="divide-y divide-surface-800/60">
              {auditLogs.map((log) => (
                <div key={log.id} className="py-3 flex items-center justify-between text-xs">
                  <div className="flex items-center gap-3">
                    <span
                      className={`w-1.5 h-1.5 rounded-full ${
                        log.status === 'success' ? 'bg-emerald-500' : 'bg-rose-500'
                      }`}
                    />
                    <span className="font-mono font-medium text-slate-200">{log.action}</span>
                    <span className="text-slate-400 font-mono">[{log.resource_type}]</span>
                  </div>
                  <div className="flex items-center gap-4 text-slate-400">
                    {log.ip_address && <span className="font-mono">{log.ip_address}</span>}
                    <span>{new Date(log.created_at).toLocaleTimeString()}</span>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>

        {/* App Control Modal */}
        <AppControlModal
          app={selectedControlApp}
          isOpen={controlModalOpen}
          onClose={() => setControlModalOpen(false)}
          onServiceControl={handleServiceControl}
          isActing={actionLoadingId === selectedControlApp?.id}
        />
      </div>
    </DashboardShell>
  );
}
