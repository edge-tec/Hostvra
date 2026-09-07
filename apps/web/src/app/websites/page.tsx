'use client';

import React, { useState, useEffect, useCallback } from 'react';
import {
  Globe,
  Plus,
  ShieldCheck,
  ShieldAlert,
  Server as ServerIcon,
  RefreshCw,
  Search,
  ExternalLink,
  Trash2,
  Power,
  X,
  Check,
  Code2,
  Layers,
  Sliders,
  Cpu,
  HardDrive,
  Activity,
  User,
  Shield,
  CheckCircle2,
  AlertTriangle,
  Zap,
} from 'lucide-react';
import { DashboardShell } from '@/components/DashboardShell';
import { apiFetch, Website, Server, UserIsolationInfo, ResourceLimits } from '@/lib/api';
import { OneClickAppModal } from '@/components/OneClickAppModal';

export default function WebsitesPage() {
  const [websites, setWebsites] = useState<Website[]>([]);
  const [servers, setServers] = useState<Server[]>([]);
  const [loading, setLoading] = useState(true);
  const [search, setSearch] = useState('');
  const [modalOpen, setModalOpen] = useState(false);

  // Form State
  const [domain, setDomain] = useState('');
  const [selectedServer, setSelectedServer] = useState('');
  const [appType, setAppType] = useState<'php' | 'static' | 'proxy'>('php');
  const [phpVersion, setPhpVersion] = useState('8.3');
  const [proxyPort, setProxyPort] = useState(3000);
  const [creating, setCreating] = useState(false);
  const [issuingSSL, setIssuingSSL] = useState<string | null>(null);

  // User Isolation & cgroups Modal State
  const [isolationModalSite, setIsolationModalSite] = useState<Website | null>(null);
  const [appModalSite, setAppModalSite] = useState<Website | null>(null);
  const [isolationInfo, setIsolationInfo] = useState<UserIsolationInfo | null>(null);
  const [isolationLoading, setIsolationLoading] = useState(false);
  const [isolationSaving, setIsolationSaving] = useState(false);
  const [isolationSuccess, setIsolationSuccess] = useState(false);

  // Form limits
  const [limitMem, setLimitMem] = useState(512);
  const [limitCPU, setLimitCPU] = useState(100);
  const [limitTasks, setLimitTasks] = useState(100);
  const [limitOpenBaseDir, setLimitOpenBaseDir] = useState(true);

  const fetchData = useCallback(async () => {
    setLoading(true);
    try {
      const [sitesRes, serversRes] = await Promise.all([
        apiFetch<Website[]>('/api/v1/websites'),
        apiFetch<Server[]>('/api/v1/servers'),
      ]);

      if (sitesRes.success && sitesRes.data) setWebsites(sitesRes.data);
      if (serversRes.success && serversRes.data) {
        setServers(serversRes.data);
        if (serversRes.data.length > 0 && !selectedServer) {
          setSelectedServer(serversRes.data[0].id);
        }
      }
    } catch (err) {
      console.error('Failed to load websites', err);
    } finally {
      setLoading(false);
    }
  }, [selectedServer]);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  const handleCreateWebsite = async (e: React.FormEvent) => {
    e.preventDefault();
    setCreating(true);

    const res = await apiFetch<Website>('/api/v1/websites', {
      method: 'POST',
      body: JSON.stringify({
        server_id: selectedServer,
        primary_domain: domain,
        app_type: appType,
        php_version: appType === 'php' ? phpVersion : undefined,
        proxy_port: appType === 'proxy' ? Number(proxyPort) : undefined,
      }),
    });

    setCreating(false);
    if (res.success && res.data) {
      setModalOpen(false);
      setDomain('');
      fetchData();
    }
  };

  const handleToggleStatus = async (site: Website) => {
    const nextStatus = site.status === 'active' ? 'suspended' : 'active';
    const res = await apiFetch<Website>(`/api/v1/websites/${site.id}/status`, {
      method: 'POST',
      body: JSON.stringify({ status: nextStatus }),
    });

    if (res.success) {
      fetchData();
    }
  };

  const handleIssueSSL = async (siteId: string) => {
    setIssuingSSL(siteId);
    const res = await apiFetch(`/api/v1/websites/${siteId}/ssl`, {
      method: 'POST',
    });
    setIssuingSSL(null);

    if (res.success) {
      fetchData();
    }
  };

  const handleDeleteWebsite = async (siteId: string, domainName: string) => {
    if (!confirm(`Are you sure you want to delete ${domainName}? This will remove the virtual host configuration, user isolation, and PHP-FPM pool.`)) {
      return;
    }

    const res = await apiFetch(`/api/v1/websites/${siteId}`, {
      method: 'DELETE',
    });

    if (res.success) {
      fetchData();
    }
  };

  // Open Isolation & cgroups Modal
  const openIsolationModal = async (site: Website) => {
    setIsolationModalSite(site);
    setIsolationLoading(true);
    setIsolationSuccess(false);

    try {
      const res = await apiFetch<UserIsolationInfo>(`/api/v1/websites/${site.id}/isolation`);
      if (res.success && res.data) {
        setIsolationInfo(res.data);
        setLimitMem(res.data.limits.memory_max_mb || 512);
        setLimitCPU(res.data.limits.cpu_quota || 100);
        setLimitTasks(res.data.limits.tasks_max || 100);
        setLimitOpenBaseDir(res.data.limits.open_basedir !== false);
      }
    } catch (err) {
      console.error('Failed to load isolation info', err);
    } finally {
      setIsolationLoading(false);
    }
  };

  // Refresh Live Isolation Metrics
  const refreshIsolationMetrics = async () => {
    if (!isolationModalSite) return;
    try {
      const res = await apiFetch<UserIsolationInfo>(`/api/v1/websites/${isolationModalSite.id}/isolation`);
      if (res.success && res.data) {
        setIsolationInfo(res.data);
      }
    } catch (err) {
      console.error('Failed to refresh metrics', err);
    }
  };

  // Save Updated Limits
  const handleSaveIsolationLimits = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!isolationModalSite) return;

    setIsolationSaving(true);
    setIsolationSuccess(false);

    try {
      const payload: ResourceLimits = {
        memory_max_mb: limitMem,
        cpu_quota: limitCPU,
        tasks_max: limitTasks,
        open_basedir: limitOpenBaseDir,
      };

      const res = await apiFetch<UserIsolationInfo>(`/api/v1/websites/${isolationModalSite.id}/isolation`, {
        method: 'PUT',
        body: JSON.stringify(payload),
      });

      if (res.success && res.data) {
        setIsolationInfo(res.data);
        setIsolationSuccess(true);
        setTimeout(() => setIsolationSuccess(false), 3000);
      }
    } catch (err) {
      console.error('Failed to save isolation limits', err);
    } finally {
      setIsolationSaving(false);
    }
  };

  const filteredWebsites = websites.filter(
    (w) =>
      w.primary_domain.toLowerCase().includes(search.toLowerCase()) ||
      w.document_root.toLowerCase().includes(search.toLowerCase()) ||
      (w.system_user && w.system_user.toLowerCase().includes(search.toLowerCase()))
  );

  return (
    <DashboardShell>
      <div className="space-y-8">
        <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
          <div>
            <h1 className="text-2xl font-bold tracking-tight text-white flex items-center gap-2.5">
              <Globe className="w-7 h-7 text-indigo-400" />
              Websites & Virtual Hosts
            </h1>
            <p className="text-sm text-slate-400 mt-1">
              Linux POSIX user isolation, per-site PHP-FPM pools, cgroups v2 resource slicing, and Nginx vhosts.
            </p>
          </div>
          <div className="flex items-center gap-3">
            <button
              onClick={fetchData}
              className="p-2.5 rounded-xl bg-surface-900 border border-surface-800 text-slate-300 hover:text-white hover:bg-surface-800 transition-colors"
            >
              <RefreshCw className={`w-4 h-4 ${loading ? 'animate-spin' : ''}`} />
            </button>
            <button
              onClick={() => setModalOpen(true)}
              className="inline-flex items-center gap-2 px-4 py-2.5 rounded-xl bg-indigo-600 hover:bg-indigo-500 text-white text-sm font-semibold shadow-lg shadow-indigo-600/25 transition-all"
            >
              <Plus className="w-4 h-4" />
              Create Website
            </button>
          </div>
        </div>

        {/* Search & Stats Bar */}
        <div className="flex flex-col sm:flex-row items-stretch sm:items-center justify-between gap-3">
          <div className="relative flex-1 max-w-md">
            <div className="flex items-center gap-3 bg-white dark:bg-[#121824] border border-slate-300 dark:border-surface-700 rounded-xl px-4 py-2.5 shadow-xs focus-within:border-[#20a53a] focus-within:ring-2 focus-within:ring-[#20a53a]/20 transition-all">
              <Search className="w-4 h-4 text-slate-400 dark:text-slate-500 flex-shrink-0" />
              <input
                type="text"
                placeholder="Search domains or isolated users..."
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                className="w-full bg-transparent text-sm font-medium text-slate-950 dark:text-white placeholder:text-slate-400 focus:outline-none"
              />
              {search && (
                <button
                  type="button"
                  onClick={() => setSearch('')}
                  className="text-xs font-semibold px-1.5 py-0.5 rounded bg-slate-100 dark:bg-surface-800 text-slate-500 hover:text-black dark:hover:text-white"
                >
                  Clear
                </button>
              )}
            </div>
          </div>
          <div className="flex items-center gap-2 text-xs font-semibold text-slate-600 dark:text-slate-400">
            <span className="px-2.5 py-1 rounded-lg bg-slate-100 dark:bg-surface-800 border border-slate-200 dark:border-surface-700">
              Active Virtual Hosts: <strong className="text-slate-900 dark:text-white">{filteredWebsites.length}</strong>
            </span>
            <span className="px-2.5 py-1 rounded-lg bg-emerald-500/10 text-emerald-400 border border-emerald-500/20 font-sans">
              <ShieldCheck className="w-3.5 h-3.5 inline mr-1" />
              cgroups v2 Enforced
            </span>
          </div>
        </div>

        {/* Websites List */}
        {loading ? (
          <div className="py-20 text-center text-slate-500">
            <div className="w-8 h-8 border-2 border-emerald-500 border-t-transparent rounded-full animate-spin mx-auto mb-3" />
            Loading websites and isolation status...
          </div>
        ) : filteredWebsites.length === 0 ? (
          <div className="py-16 text-center border border-dashed border-slate-300 dark:border-surface-800 rounded-2xl bg-white dark:bg-surface-900/50 p-6 shadow-xs">
            <Globe className="w-12 h-12 mx-auto text-slate-400 dark:text-slate-600 mb-3" />
            <h3 className="text-base font-bold text-slate-900 dark:text-white">No websites deployed yet</h3>
            <p className="text-xs text-slate-500 dark:text-slate-400 mt-1 max-w-sm mx-auto font-medium">
              Create your first virtual host with automated Linux user isolation and cgroups quotas.
            </p>
            <button
              onClick={() => setModalOpen(true)}
              className="mt-4 inline-flex items-center gap-2 px-4 py-2.5 rounded-xl bg-[#20a53a] hover:bg-[#1b8c31] text-white text-xs font-bold shadow-md transition-all cursor-pointer"
            >
              <Plus className="w-3.5 h-3.5" />
              Create Website
            </button>
          </div>
        ) : (
          <div className="bg-white dark:bg-[#10141d] border border-slate-200 dark:border-surface-800 rounded-2xl overflow-hidden shadow-xs dark:shadow-xl">
            <div className="overflow-x-auto">
              <table className="w-full text-left text-sm border-collapse">
                <thead>
                  <tr className="border-b border-slate-200 dark:border-surface-800 bg-slate-50 dark:bg-[#151b28]">
                    <th className="px-6 py-3.5 text-[11px] font-bold uppercase tracking-wider text-slate-700 dark:text-slate-300">Primary Domain</th>
                    <th className="px-6 py-3.5 text-[11px] font-bold uppercase tracking-wider text-slate-700 dark:text-slate-300">Security Isolation</th>
                    <th className="px-6 py-3.5 text-[11px] font-bold uppercase tracking-wider text-slate-700 dark:text-slate-300">Runtime</th>
                    <th className="px-6 py-3.5 text-[11px] font-bold uppercase tracking-wider text-slate-700 dark:text-slate-300">Document Root</th>
                    <th className="px-6 py-3.5 text-[11px] font-bold uppercase tracking-wider text-slate-700 dark:text-slate-300">Status</th>
                    <th className="px-6 py-3.5 text-[11px] font-bold uppercase tracking-wider text-slate-700 dark:text-slate-300">SSL</th>
                    <th className="px-6 py-3.5 text-[11px] font-bold uppercase tracking-wider text-slate-700 dark:text-slate-300 text-right">Actions</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-slate-200/80 dark:divide-surface-800/80 font-sans text-xs">
                  {filteredWebsites.map((site) => (
                    <tr key={site.id} className="hover:bg-surface-800/30 transition-colors">
                      <td className="px-6 py-4">
                        <div className="flex items-center gap-2 font-bold text-white text-sm">
                          <span>{site.primary_domain}</span>
                          <a
                            href={`http://${site.primary_domain}`}
                            target="_blank"
                            rel="noreferrer"
                            className="text-slate-500 hover:text-indigo-400 transition-colors"
                          >
                            <ExternalLink className="w-3.5 h-3.5" />
                          </a>
                        </div>
                      </td>

                      {/* Security Isolation User Badge */}
                      <td className="px-6 py-4">
                        <div className="flex flex-col gap-1">
                          <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded text-[11px] font-mono font-medium bg-emerald-500/10 text-emerald-400 border border-emerald-500/20 w-fit">
                            <User className="w-3 h-3 text-emerald-400" />
                            {site.system_user || 'u_isolated'}
                          </span>
                          <span className="text-[10px] text-slate-500 font-mono">0750 cross-tenant guard</span>
                        </div>
                      </td>

                      <td className="px-6 py-4">
                        <div className="flex items-center gap-1.5">
                          <span className="inline-flex items-center gap-1.5 px-2 py-0.5 rounded text-xs font-medium bg-surface-800 text-slate-200 border border-surface-700">
                            <Code2 className="w-3.5 h-3.5 text-indigo-400" />
                            {site.app_type === 'php'
                              ? `PHP ${site.php_version || '8.3'}`
                              : site.app_type === 'proxy'
                              ? `Proxy :${site.proxy_port}`
                              : 'Static'}
                          </span>
                          <span className="inline-flex items-center gap-1 px-1.5 py-0.5 rounded text-[10px] font-mono font-medium bg-indigo-500/10 text-indigo-400 border border-indigo-500/20 uppercase">
                            {site.web_server_type || 'nginx'}
                          </span>
                        </div>
                      </td>

                      <td className="px-6 py-4 font-mono text-xs text-slate-400 max-w-xs truncate">
                        {site.document_root}
                      </td>

                      <td className="px-6 py-4">
                        <span
                          className={`inline-flex items-center px-2 py-0.5 rounded-full text-[11px] font-semibold capitalize ${
                            site.status === 'active'
                              ? 'bg-emerald-500/10 text-emerald-400 border border-emerald-500/20'
                              : 'bg-amber-500/10 text-amber-400 border border-amber-500/20'
                          }`}
                        >
                          {site.status}
                        </span>
                      </td>

                      <td className="px-6 py-4">
                        {site.ssl_enabled ? (
                          <span className="inline-flex items-center gap-1 text-xs text-emerald-400 font-medium">
                            <ShieldCheck className="w-4 h-4" />
                            <span>Protected</span>
                          </span>
                        ) : (
                          <button
                            onClick={() => handleIssueSSL(site.id)}
                            disabled={issuingSSL === site.id}
                            className="inline-flex items-center gap-1.5 text-xs text-indigo-400 hover:text-indigo-300 font-medium"
                          >
                            <ShieldAlert className="w-3.5 h-3.5 text-amber-400" />
                            <span>{issuingSSL === site.id ? 'Issuing...' : 'Issue SSL'}</span>
                          </button>
                        )}
                      </td>

                      <td className="px-6 py-4 text-right">
                        <div className="flex items-center justify-end gap-1.5">
                          {/* 1-Click App Installer Button */}
                          <button
                            onClick={() => setAppModalSite(site)}
                            title="1-Click App Installer (WordPress, Laravel, Next.js)"
                            className="inline-flex items-center gap-1 px-2.5 py-1.5 rounded-lg bg-purple-500/10 hover:bg-purple-500/20 text-purple-600 dark:text-purple-400 border border-purple-500/20 text-xs font-semibold transition-colors"
                          >
                            <Zap className="w-3.5 h-3.5" />
                            <span className="capitalize">{site.app_type && site.app_type !== 'static' && site.app_type !== 'php' ? site.app_type : 'Deploy App'}</span>
                          </button>

                          {/* Resource Limits & Isolation (cgroups) Button */}
                          <button
                            onClick={() => openIsolationModal(site)}
                            title="Resource Limits & cgroups Isolation"
                            className="inline-flex items-center gap-1 px-2.5 py-1.5 rounded-lg bg-indigo-500/10 hover:bg-indigo-500/20 text-indigo-400 border border-indigo-500/20 text-xs font-semibold transition-colors"
                          >
                            <Cpu className="w-3.5 h-3.5" />
                            <span>cgroups</span>
                          </button>

                          <a
                            href="/webservers"
                            title="Manage Web Server Engine & VHost"
                            className="p-1.5 rounded-lg border border-surface-700 text-slate-400 hover:text-indigo-400 hover:bg-surface-800 transition-colors inline-flex items-center"
                          >
                            <Layers className="w-3.5 h-3.5 text-indigo-400" />
                          </a>

                          {site.app_type === 'php' && (
                            <a
                              href="/php"
                              title="Configure PHP Settings & Pool"
                              className="p-1.5 rounded-lg border border-surface-700 text-slate-400 hover:text-sky-400 hover:bg-surface-800 transition-colors inline-flex items-center"
                            >
                              <Code2 className="w-3.5 h-3.5 text-sky-400" />
                            </a>
                          )}

                          <button
                            onClick={() => handleToggleStatus(site)}
                            title={site.status === 'active' ? 'Suspend Website' : 'Activate Website'}
                            className={`p-1.5 rounded-lg border transition-colors ${
                              site.status === 'active'
                                ? 'border-surface-700 text-slate-400 hover:text-amber-400 hover:bg-surface-800'
                                : 'border-emerald-500/30 text-emerald-400 hover:bg-emerald-500/10'
                            }`}
                          >
                            <Power className="w-3.5 h-3.5" />
                          </button>

                          <button
                            onClick={() => handleDeleteWebsite(site.id, site.primary_domain)}
                            title="Delete Website"
                            className="p-1.5 rounded-lg border border-surface-700 text-slate-400 hover:text-rose-400 hover:bg-surface-800 transition-colors"
                          >
                            <Trash2 className="w-3.5 h-3.5" />
                          </button>
                        </div>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </div>
        )}

        {/* MODAL: USER ISOLATION & CGROUPS V2 */}
        {isolationModalSite && (
          <div className="fixed inset-0 z-50 bg-black/75 backdrop-blur-sm flex items-center justify-center p-4 animate-fadeIn">
            <div className="w-full max-w-xl bg-surface-900 border border-surface-700 rounded-2xl shadow-2xl p-6 relative space-y-5">
              <div className="flex items-start justify-between">
                <div>
                  <h2 className="text-lg font-bold text-white flex items-center gap-2">
                    <Shield className="w-5 h-5 text-indigo-400" />
                    Resource Quotas & Isolation (cgroups v2)
                  </h2>
                  <p className="text-xs text-slate-400 mt-1">
                    Enforcing CPU, RAM, and process security boundaries for <span className="text-white font-mono font-bold">{isolationModalSite.primary_domain}</span>
                  </p>
                </div>
                <button
                  onClick={() => setIsolationModalSite(null)}
                  className="text-slate-400 hover:text-white transition-colors"
                >
                  <X className="w-5 h-5" />
                </button>
              </div>

              {isolationLoading ? (
                <div className="py-12 text-center text-slate-400">
                  <RefreshCw className="w-6 h-6 animate-spin mx-auto text-indigo-400 mb-2" />
                  Loading cgroups telemetry and pool socket...
                </div>
              ) : (
                <div className="space-y-5">
                  {/* Live Telemetry Gauges */}
                  <div className="bg-surface-950 border border-surface-800 rounded-xl p-4 space-y-3">
                    <div className="flex items-center justify-between text-xs font-semibold">
                      <span className="text-slate-400 flex items-center gap-1.5 uppercase tracking-wider text-[10px]">
                        <Activity className="w-3.5 h-3.5 text-emerald-400" /> Live Resource Telemetry
                      </span>
                      <button
                        type="button"
                        onClick={refreshIsolationMetrics}
                        className="text-indigo-400 hover:text-indigo-300 inline-flex items-center gap-1"
                      >
                        <RefreshCw className="w-3 h-3" /> Refresh
                      </button>
                    </div>

                    <div className="grid grid-cols-3 gap-3 text-center">
                      <div className="p-2.5 rounded-lg bg-surface-900 border border-surface-800">
                        <div className="text-[10px] text-slate-400 uppercase">Memory Usage</div>
                        <div className="text-base font-bold text-white font-mono mt-0.5">
                          {isolationInfo?.memory_used_mb || 0} <span className="text-xs text-slate-400 font-normal">/ {limitMem} MB</span>
                        </div>
                        <div className="w-full bg-surface-800 rounded-full h-1.5 mt-2 overflow-hidden">
                          <div
                            className="bg-indigo-500 h-1.5 rounded-full"
                            style={{
                              width: `${Math.min(
                                ((isolationInfo?.memory_used_mb || 0) / (limitMem || 512)) * 100,
                                100
                              )}%`,
                            }}
                          />
                        </div>
                      </div>

                      <div className="p-2.5 rounded-lg bg-surface-900 border border-surface-800">
                        <div className="text-[10px] text-slate-400 uppercase">CPU Usage</div>
                        <div className="text-base font-bold text-white font-mono mt-0.5">
                          {isolationInfo?.cpu_usage_perc || 0}% <span className="text-xs text-slate-400 font-normal">/ {limitCPU}%</span>
                        </div>
                        <div className="w-full bg-surface-800 rounded-full h-1.5 mt-2 overflow-hidden">
                          <div
                            className="bg-emerald-500 h-1.5 rounded-full"
                            style={{
                              width: `${Math.min(
                                ((isolationInfo?.cpu_usage_perc || 0) / (limitCPU || 100)) * 100,
                                100
                              )}%`,
                            }}
                          />
                        </div>
                      </div>

                      <div className="p-2.5 rounded-lg bg-surface-900 border border-surface-800">
                        <div className="text-[10px] text-slate-400 uppercase">Active Tasks / PIDs</div>
                        <div className="text-base font-bold text-white font-mono mt-0.5">
                          {isolationInfo?.tasks_current || 0} <span className="text-xs text-slate-400 font-normal">/ {limitTasks} max</span>
                        </div>
                        <div className="w-full bg-surface-800 rounded-full h-1.5 mt-2 overflow-hidden">
                          <div
                            className="bg-amber-500 h-1.5 rounded-full"
                            style={{
                              width: `${Math.min(
                                ((isolationInfo?.tasks_current || 0) / (limitTasks || 100)) * 100,
                                100
                              )}%`,
                            }}
                          />
                        </div>
                      </div>
                    </div>

                    <div className="text-[11px] font-mono text-slate-400 pt-1 space-y-1">
                      <div>• Isolated User: <span className="text-emerald-400 font-bold">{isolationInfo?.username}</span></div>
                      <div className="truncate">• UNIX Socket: <span className="text-slate-300">{isolationInfo?.php_pool_socket}</span></div>
                      <div>• Systemd Slice: <span className="text-slate-300">{isolationInfo?.slice_name}</span></div>
                    </div>
                  </div>

                  {isolationSuccess && (
                    <div className="p-3 rounded-lg bg-emerald-500/10 border border-emerald-500/20 text-emerald-400 text-xs flex items-center gap-2">
                      <CheckCircle2 className="w-4 h-4 flex-shrink-0" />
                      <span>cgroups v2 quotas and PHP-FPM pool reconfigured successfully!</span>
                    </div>
                  )}

                  {/* Resource Limit Controls Form */}
                  <form onSubmit={handleSaveIsolationLimits} className="space-y-4 text-xs">
                    {/* RAM Limit */}
                    <div>
                      <div className="flex justify-between mb-1">
                        <label className="font-semibold text-slate-300">Memory Limit (RAM)</label>
                        <span className="font-mono text-indigo-400 font-bold">{limitMem} MB</span>
                      </div>
                      <div className="flex gap-2 mb-2">
                        {[256, 512, 1024, 2048, 4096].map((mb) => (
                          <button
                            type="button"
                            key={mb}
                            onClick={() => setLimitMem(mb)}
                            className={`flex-1 py-1 rounded border text-[11px] font-semibold transition-all ${
                              limitMem === mb
                                ? 'bg-indigo-600 border-indigo-500 text-white'
                                : 'bg-surface-800 border-surface-700 text-slate-400 hover:text-white'
                            }`}
                          >
                            {mb < 1024 ? `${mb}M` : `${mb / 1024}G`}
                          </button>
                        ))}
                      </div>
                      <input
                        type="range"
                        min="128"
                        max="8192"
                        step="128"
                        value={limitMem}
                        onChange={(e) => setLimitMem(Number(e.target.value))}
                        className="w-full accent-indigo-500"
                      />
                    </div>

                    {/* CPU Quota */}
                    <div>
                      <div className="flex justify-between mb-1">
                        <label className="font-semibold text-slate-300">CPU Core Quota</label>
                        <span className="font-mono text-emerald-400 font-bold">{limitCPU}% ({limitCPU / 100} Cores)</span>
                      </div>
                      <div className="flex gap-2 mb-2">
                        {[50, 100, 200, 400].map((cpu) => (
                          <button
                            type="button"
                            key={cpu}
                            onClick={() => setLimitCPU(cpu)}
                            className={`flex-1 py-1 rounded border text-[11px] font-semibold transition-all ${
                              limitCPU === cpu
                                ? 'bg-emerald-600 border-emerald-500 text-white'
                                : 'bg-surface-800 border-surface-700 text-slate-400 hover:text-white'
                            }`}
                          >
                            {cpu}% ({cpu / 100} Core)
                          </button>
                        ))}
                      </div>
                      <input
                        type="range"
                        min="25"
                        max="800"
                        step="25"
                        value={limitCPU}
                        onChange={(e) => setLimitCPU(Number(e.target.value))}
                        className="w-full accent-emerald-500"
                      />
                    </div>

                    {/* Tasks / PIDs max */}
                    <div>
                      <div className="flex justify-between mb-1">
                        <label className="font-semibold text-slate-300">Max Tasks / Processes (Fork Bomb Protection)</label>
                        <span className="font-mono text-amber-400 font-bold">{limitTasks} Tasks</span>
                      </div>
                      <div className="flex gap-2 mb-2">
                        {[50, 100, 200, 500].map((t) => (
                          <button
                            type="button"
                            key={t}
                            onClick={() => setLimitTasks(t)}
                            className={`flex-1 py-1 rounded border text-[11px] font-semibold transition-all ${
                              limitTasks === t
                                ? 'bg-amber-600 border-amber-500 text-white'
                                : 'bg-surface-800 border-surface-700 text-slate-400 hover:text-white'
                            }`}
                          >
                            {t}
                          </button>
                        ))}
                      </div>
                    </div>

                    {/* open_basedir toggle */}
                    <div className="p-3 rounded-lg bg-surface-950 border border-surface-800 flex items-center justify-between">
                      <div>
                        <div className="font-semibold text-slate-200">Enforce PHP open_basedir Confinement</div>
                        <div className="text-[11px] text-slate-400 mt-0.5">
                          Restricts PHP filesystem access to <span className="font-mono text-slate-300">/var/www/{isolationModalSite.primary_domain}</span> and <span className="font-mono text-slate-300">/tmp</span>.
                        </div>
                      </div>
                      <label className="relative inline-flex items-center cursor-pointer">
                        <input
                          type="checkbox"
                          checked={limitOpenBaseDir}
                          onChange={(e) => setLimitOpenBaseDir(e.target.checked)}
                          className="sr-only peer"
                        />
                        <div className="w-9 h-5 bg-surface-800 peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-slate-300 after:border after:rounded-full after:h-4 after:w-4 after:transition-all peer-checked:bg-indigo-600"></div>
                      </label>
                    </div>

                    <div className="flex justify-end gap-3 pt-3 border-t border-surface-800">
                      <button
                        type="button"
                        onClick={() => setIsolationModalSite(null)}
                        className="px-4 py-2 rounded-xl text-xs font-semibold text-slate-400 hover:text-white transition-colors"
                      >
                        Close
                      </button>
                      <button
                        type="submit"
                        disabled={isolationSaving}
                        className="px-5 py-2 rounded-xl bg-indigo-600 hover:bg-indigo-500 disabled:bg-indigo-600/50 text-white text-xs font-semibold shadow-md transition-all flex items-center gap-2"
                      >
                        {isolationSaving ? (
                          <>
                            <RefreshCw className="w-3.5 h-3.5 animate-spin" />
                            Applying cgroups quotas...
                          </>
                        ) : (
                          'Save & Apply Limits'
                        )}
                      </button>
                    </div>
                  </form>
                </div>
              )}
            </div>
          </div>
        )}

        {/* Create Website Modal */}
        {modalOpen && (
          <div className="fixed inset-0 z-50 bg-black/70 backdrop-blur-sm flex items-center justify-center p-4">
            <div className="w-full max-w-lg bg-surface-900 border border-surface-700 rounded-2xl shadow-2xl p-6 relative">
              <button
                onClick={() => setModalOpen(false)}
                className="absolute top-5 right-5 p-1 rounded-lg text-slate-400 hover:text-white hover:bg-surface-800 transition-colors"
              >
                <X className="w-5 h-5" />
              </button>

              <div className="flex items-center gap-3 mb-5">
                <div className="w-10 h-10 rounded-xl bg-indigo-500/10 border border-indigo-500/20 flex items-center justify-center text-indigo-400">
                  <Globe className="w-5 h-5" />
                </div>
                <div>
                  <h2 className="text-lg font-bold text-white">Create Isolated Website</h2>
                  <p className="text-xs text-slate-400">Provisions Linux POSIX user, dedicated PHP-FPM socket, and cgroups slice</p>
                </div>
              </div>

              <form onSubmit={handleCreateWebsite} className="space-y-4">
                <div>
                  <label className="block text-xs font-semibold uppercase tracking-wider text-slate-300 mb-1.5">
                    Target Server
                  </label>
                  <select
                    value={selectedServer}
                    onChange={(e) => setSelectedServer(e.target.value)}
                    required
                    className="w-full px-3.5 py-2.5 rounded-xl bg-surface-950 border border-surface-700 text-slate-100 text-sm focus:outline-none focus:border-indigo-500"
                  >
                    {servers.map((s) => (
                      <option key={s.id} value={s.id}>
                        {s.name} ({s.ip_address}) — {s.os_name}
                      </option>
                    ))}
                  </select>
                </div>

                <div>
                  <label className="block text-xs font-semibold uppercase tracking-wider text-slate-300 mb-1.5">
                    Primary Domain Name
                  </label>
                  <input
                    type="text"
                    required
                    value={domain}
                    onChange={(e) => setDomain(e.target.value)}
                    placeholder="app.yourdomain.com"
                    className="w-full px-3.5 py-2.5 rounded-xl bg-surface-950 border border-surface-700 text-slate-100 placeholder-slate-500 text-sm focus:outline-none focus:border-indigo-500"
                  />
                </div>

                <div className="grid grid-cols-2 gap-4">
                  <div>
                    <label className="block text-xs font-semibold uppercase tracking-wider text-slate-300 mb-1.5">
                      Application Runtime
                    </label>
                    <select
                      value={appType}
                      onChange={(e) => setAppType(e.target.value as any)}
                      className="w-full px-3.5 py-2.5 rounded-xl bg-surface-950 border border-surface-700 text-slate-100 text-sm focus:outline-none focus:border-indigo-500"
                    >
                      <option value="php">PHP Application</option>
                      <option value="proxy">Reverse Proxy (Node/Python)</option>
                      <option value="static">Static HTML / Jamstack</option>
                    </select>
                  </div>

                  {appType === 'php' && (
                    <div>
                      <label className="block text-xs font-semibold uppercase tracking-wider text-slate-300 mb-1.5">
                        PHP-FPM Version
                      </label>
                      <select
                        value={phpVersion}
                        onChange={(e) => setPhpVersion(e.target.value)}
                        className="w-full px-3.5 py-2.5 rounded-xl bg-surface-950 border border-surface-700 text-slate-100 text-sm focus:outline-none focus:border-indigo-500"
                      >
                        <option value="8.4">PHP 8.4 (Latest)</option>
                        <option value="8.3">PHP 8.3 (Stable)</option>
                        <option value="8.2">PHP 8.2</option>
                        <option value="8.1">PHP 8.1</option>
                      </select>
                    </div>
                  )}

                  {appType === 'proxy' && (
                    <div>
                      <label className="block text-xs font-semibold uppercase tracking-wider text-slate-300 mb-1.5">
                        Internal Port
                      </label>
                      <input
                        type="number"
                        value={proxyPort}
                        onChange={(e) => setProxyPort(Number(e.target.value))}
                        className="w-full px-3.5 py-2.5 rounded-xl bg-surface-950 border border-surface-700 text-slate-100 text-sm focus:outline-none focus:border-indigo-500"
                      />
                    </div>
                  )}
                </div>

                <div className="p-3.5 rounded-xl bg-surface-950/60 border border-surface-800 text-[11px] text-slate-400 space-y-1">
                  <div className="font-semibold text-slate-300 flex items-center gap-1">
                    <ShieldCheck className="w-3.5 h-3.5 text-emerald-400" /> Automated Enterprise Security:
                  </div>
                  <div>• Isolated Linux User: <span className="font-mono text-emerald-400">u_{domain ? domain.toLowerCase().replace(/[^a-z0-9_]/g, '_') : 'domain'}</span></div>
                  <div>• Dedicated UNIX Socket: <span className="font-mono text-slate-300">/run/php/php{phpVersion}-fpm-u_...sock</span></div>
                  <div>• Document root restricted with 0750 permissions and open_basedir</div>
                </div>

                <div className="flex justify-end gap-3 pt-3">
                  <button
                    type="button"
                    onClick={() => setModalOpen(false)}
                    className="px-4 py-2 rounded-xl text-sm text-slate-400 hover:text-white transition-colors"
                  >
                    Cancel
                  </button>
                  <button
                    type="submit"
                    disabled={creating || !selectedServer}
                    className="px-5 py-2 rounded-xl bg-indigo-600 hover:bg-indigo-500 disabled:bg-indigo-600/50 text-white text-sm font-semibold shadow-md transition-all flex items-center gap-2"
                  >
                    {creating ? (
                      <div className="w-4 h-4 border-2 border-white border-t-transparent rounded-full animate-spin" />
                    ) : (
                      'Provision Website'
                    )}
                  </button>
                </div>
              </form>
            </div>
          </div>
        )}

        {/* 1-Click App Installer Modal */}
        {appModalSite && (
          <OneClickAppModal
            website={appModalSite}
            isOpen={!!appModalSite}
            onClose={() => setAppModalSite(null)}
            onSuccess={() => {
              fetchData();
            }}
          />
        )}
      </div>
    </DashboardShell>
  );
}
