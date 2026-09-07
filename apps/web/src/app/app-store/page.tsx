'use client';

import React, { useState, useEffect, useRef } from 'react';
import {
  Boxes,
  Search,
  RefreshCw,
  Play,
  Square,
  RotateCw,
  Trash2,
  CheckCircle2,
  AlertCircle,
  Clock,
  ExternalLink,
  Sliders,
  Sparkles,
  Server,
  Database,
  Cpu,
  Shield,
  Layers,
  Code2,
  Terminal,
  X,
  ChevronRight,
  Filter,
  Activity,
  Mail,
  Wrench,
  Bookmark,
  BookmarkCheck,
  Zap,
  Globe,
} from 'lucide-react';
import { DashboardShell } from '@/components/DashboardShell';
import { apiFetch, AppPackage, AppInstallJob, Website } from '@/lib/api';
import { AppControlModal } from '@/components/AppControlModal';
import { OneClickAppModal } from '@/components/OneClickAppModal';
import { getPinnedAppIds, togglePinApp } from '@/lib/appstore-utils';

const CATEGORIES = [
  { id: 'all', label: 'All Software', icon: Boxes },
  { id: 'web_apps', label: '1-Click CMS & Apps', icon: Sparkles },
  { id: 'web_server', label: 'Web Servers', icon: Server },
  { id: 'runtime', label: 'PHP & Runtimes', icon: Code2 },
  { id: 'database', label: 'Databases & Cache', icon: Database },
  { id: 'process_manager', label: 'Process & Containers', icon: Cpu },
  { id: 'security', label: 'Security & Firewall', icon: Shield },
  { id: 'monitoring', label: 'Monitoring & Health', icon: Activity },
  { id: 'mail', label: 'Mail Servers', icon: Mail },
  { id: 'tools', label: 'DevOps & Tools', icon: Wrench },
];

export default function AppStorePage() {
  const [apps, setApps] = useState<AppPackage[]>([]);
  const [loading, setLoading] = useState(true);
  const [searchQuery, setSearchQuery] = useState('');
  const [activeCategory, setActiveCategory] = useState('all');
  const [activeJob, setActiveJob] = useState<AppInstallJob | null>(null);
  const [jobModalOpen, setJobModalOpen] = useState(false);
  const [actionLoadingId, setActionLoadingId] = useState<string | null>(null);
  const [viewMode, setViewMode] = useState<'table' | 'grid'>('table');
  const [selectedControlApp, setSelectedControlApp] = useState<AppPackage | null>(null);
  const [controlModalOpen, setControlModalOpen] = useState(false);
  const [pinnedAppIds, setPinnedAppIds] = useState<string[]>([]);

  const [websites, setWebsites] = useState<Website[]>([]);
  const [deploySite, setDeploySite] = useState<Website | null>(null);
  const [sitePickerApp, setSitePickerApp] = useState<string | null>(null);
  const logsEndRef = useRef<HTMLDivElement>(null);

  // Load applications and websites
  const loadApps = async () => {
    setLoading(true);
    try {
      const [appRes, siteRes] = await Promise.all([
        apiFetch<AppPackage[]>('/api/v1/apps'),
        apiFetch<Website[]>('/api/v1/websites'),
      ]);
      if (appRes.success && appRes.data) {
        setApps(appRes.data);
      }
      if (siteRes.success && siteRes.data) {
        setWebsites(siteRes.data);
      }
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadApps();
    setPinnedAppIds(getPinnedAppIds());
    const handlePinnedChange = () => setPinnedAppIds(getPinnedAppIds());
    window.addEventListener('hostvra_pinned_apps_changed', handlePinnedChange);
    return () => window.removeEventListener('hostvra_pinned_apps_changed', handlePinnedChange);
  }, []);

  const handleOpenApp = (app: AppPackage) => {
    setSelectedControlApp(app);
    setControlModalOpen(true);
  };

  // Poll active install/uninstall job
  useEffect(() => {
    if (!activeJob || activeJob.status === 'completed' || activeJob.status === 'failed') {
      return;
    }

    const interval = setInterval(async () => {
      const res = await apiFetch<AppInstallJob>(`/api/v1/apps/jobs/${activeJob.id}`);
      if (res.success && res.data) {
        setActiveJob(res.data);
        if (res.data.status === 'completed' || res.data.status === 'failed') {
          loadApps();
        }
      }
    }, 1500);

    return () => clearInterval(interval);
  }, [activeJob]);

  // Auto-scroll modal logs
  useEffect(() => {
    if (jobModalOpen && logsEndRef.current) {
      logsEndRef.current.scrollIntoView({ behavior: 'smooth' });
    }
  }, [activeJob?.logs, jobModalOpen]);

  // 1-Click Install trigger
  const handleInstall = async (app: AppPackage) => {
    setActionLoadingId(app.id);
    try {
      const res = await apiFetch<AppInstallJob>(`/api/v1/apps/${app.id}/install`, {
        method: 'POST',
      });
      if (res.success && res.data) {
        setActiveJob(res.data);
        setJobModalOpen(true);
      }
    } finally {
      setActionLoadingId(null);
    }
  };

  // 1-Click Uninstall trigger
  const handleUninstall = async (app: AppPackage) => {
    if (!confirm(`Are you sure you want to uninstall ${app.display_name}? This will stop and remove the package.`)) {
      return;
    }
    setActionLoadingId(app.id);
    try {
      const res = await apiFetch<AppInstallJob>(`/api/v1/apps/${app.id}/uninstall`, {
        method: 'POST',
      });
      if (res.success && res.data) {
        setActiveJob(res.data);
        setJobModalOpen(true);
      }
    } finally {
      setActionLoadingId(null);
    }
  };

  // Service toggle (Start / Stop / Restart)
  const handleServiceControl = async (app: AppPackage, action: 'start' | 'stop' | 'restart') => {
    setActionLoadingId(app.id);
    try {
      const res = await apiFetch<AppPackage>(`/api/v1/apps/${app.id}/service`, {
        method: 'POST',
        body: JSON.stringify({ action }),
      });
      if (res.success && res.data) {
        setApps((prev) => prev.map((p) => (p.id === app.id ? res.data! : p)));
      }
    } finally {
      setActionLoadingId(null);
    }
  };

  // Filter apps by category & search query
  const filteredApps = apps.filter((app) => {
    const matchesCategory =
      activeCategory === 'all' ||
      app.category === activeCategory ||
      (activeCategory === 'security' && (app.category === 'security' || app.category === 'tools'));

    const matchesSearch =
      searchQuery.trim() === '' ||
      app.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
      app.display_name.toLowerCase().includes(searchQuery.toLowerCase()) ||
      app.description.toLowerCase().includes(searchQuery.toLowerCase());

    return matchesCategory && matchesSearch;
  });

  const getCategoryIcon = (category: string) => {
    switch (category) {
      case 'process_manager':
        return <Cpu className="w-4 h-4 text-indigo-500" />;
      case 'web_server':
        return <Server className="w-4 h-4 text-emerald-500" />;
      case 'database':
        return <Database className="w-4 h-4 text-amber-500" />;
      case 'runtime':
        return <Code2 className="w-4 h-4 text-blue-500" />;
      case 'security':
        return <Shield className="w-4 h-4 text-purple-500" />;
      case 'monitoring':
        return <Activity className="w-4 h-4 text-rose-500" />;
      case 'mail':
        return <Mail className="w-4 h-4 text-sky-500" />;
      case 'tools':
      default:
        return <Wrench className="w-4 h-4 text-teal-500" />;
    }
  };

  return (
    <DashboardShell>
      <div className="space-y-6">
        {/* Top Header */}
        <div className="flex flex-col md:flex-row md:items-center justify-between gap-4">
          <div>
            <h1 className="text-2xl font-bold tracking-tight text-slate-900 dark:text-white flex items-center gap-2.5">
              <span className="p-2 rounded-xl bg-indigo-500/10 text-indigo-500 dark:text-indigo-400">
                <Boxes className="w-6 h-6" />
              </span>
              App Store
              <span className="text-xs px-2.5 py-0.5 rounded-full bg-indigo-500/10 text-indigo-600 dark:text-indigo-400 border border-indigo-500/20 font-medium">
                1-Click Software Manager
              </span>
            </h1>
            <p className="text-sm text-slate-500 dark:text-slate-400 mt-1">
              Easily install, manage, and configure server software extensions, process managers, runtimes, and databases without running terminal commands.
            </p>
          </div>

          {/* Header Action Tools */}
          <div className="flex items-center gap-2">
            <button
              onClick={loadApps}
              disabled={loading}
              className="p-2 rounded-xl bg-surface-100 dark:bg-surface-800 hover:bg-surface-200 dark:hover:bg-surface-700 text-slate-600 dark:text-slate-300 border border-surface-200 dark:border-surface-700 transition-colors"
              title="Refresh Catalog"
            >
              <RefreshCw className={`w-4 h-4 ${loading ? 'animate-spin' : ''}`} />
            </button>
            <div className="flex items-center rounded-xl bg-surface-100 dark:bg-surface-800 p-1 border border-surface-200 dark:border-surface-700 text-xs font-medium">
              <button
                onClick={() => setViewMode('table')}
                className={`px-3 py-1 rounded-lg transition-colors ${
                  viewMode === 'table'
                    ? 'bg-white dark:bg-surface-700 text-slate-900 dark:text-white shadow-sm'
                    : 'text-slate-500 hover:text-slate-800 dark:hover:text-slate-200'
                }`}
              >
                Table List
              </button>
              <button
                onClick={() => setViewMode('grid')}
                className={`px-3 py-1 rounded-lg transition-colors ${
                  viewMode === 'grid'
                    ? 'bg-white dark:bg-surface-700 text-slate-900 dark:text-white shadow-sm'
                    : 'text-slate-500 hover:text-slate-800 dark:hover:text-slate-200'
                }`}
              >
                Grid Cards
              </button>
            </div>
          </div>
        </div>

        {/* Search & Category Filter Toolbar */}
        <div className="flex flex-col sm:flex-row items-stretch sm:items-center justify-between gap-3">
          {/* Categories Tabs */}
          <div className="flex items-center gap-1 overflow-x-auto pb-1 scrollbar-none">
            {CATEGORIES.map((cat) => {
              const Icon = cat.icon;
              const isActive = activeCategory === cat.id;
              return (
                <button
                  key={cat.id}
                  onClick={() => setActiveCategory(cat.id)}
                  className={`px-3.5 py-2 rounded-xl text-xs font-medium whitespace-nowrap flex items-center gap-2 transition-all ${
                    isActive
                      ? 'bg-indigo-600 text-white shadow-sm shadow-indigo-500/20 font-semibold'
                      : 'bg-surface-100 dark:bg-surface-800/80 hover:bg-surface-200 dark:hover:bg-surface-700 text-slate-600 dark:text-slate-300 border border-surface-200 dark:border-surface-750'
                  }`}
                >
                  <Icon className={`w-3.5 h-3.5 ${isActive ? 'text-white' : 'text-slate-400'}`} />
                  {cat.label}
                </button>
              );
            })}
          </div>

          {/* Search Input Box */}
          <div className="relative min-w-[240px]">
            <Search className="w-4 h-4 absolute left-3 top-1/2 -translate-y-1/2 text-slate-400" />
            <input
              type="text"
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              placeholder="Search software (Supervisor, Redis)..."
              className="w-full pl-9 pr-4 py-2 rounded-xl text-xs bg-surface-100 dark:bg-surface-800 border border-surface-200 dark:border-surface-700 text-slate-900 dark:text-white placeholder:text-slate-400 focus:outline-none focus:ring-2 focus:ring-indigo-500/30"
            />
          </div>
        </div>

        {/* Software Views */}
        {activeCategory === 'web_apps' ? (
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
            {[
              {
                id: 'wordpress',
                name: 'WordPress',
                version: '6.7.x',
                category: 'Content Management System',
                desc: 'World\'s #1 open-source CMS for blogs, e-commerce (WooCommerce), and enterprise portals. Auto-configures MySQL, wp-config.php and security salts.',
                icon: Globe,
                badge: 'v6.7 LTS',
                specs: 'PHP 8.1+ • MySQL 5.7+ • 512MB RAM',
              },
              {
                id: 'laravel',
                name: 'Laravel',
                version: '11.x',
                category: 'PHP MVC Framework',
                desc: 'Full-stack enterprise framework with queue workers, migration scaffolding, artisan CLI, and environment key generation.',
                icon: Code2,
                badge: 'v11.x',
                specs: 'PHP 8.2+ • MySQL / SQLite • 1GB RAM',
              },
              {
                id: 'nextjs',
                name: 'Next.js Starter',
                version: '15.x',
                category: 'Full-Stack React Framework',
                desc: 'High-performance React application framework with hybrid static & server rendering and PM2 ecosystem process configuration.',
                icon: Cpu,
                badge: 'v15.1',
                specs: 'Node.js 18+ • PM2 • 1GB RAM',
              },
              {
                id: 'drupal',
                name: 'Drupal',
                version: '10.x',
                category: 'Enterprise CMS',
                desc: 'High-security modular digital experience platform with robust content modeling, multilingual support, and taxonomy workflows.',
                icon: Layers,
                badge: 'v10.3',
                specs: 'PHP 8.2+ • MySQL 8.0+ • 1GB RAM',
              },
              {
                id: 'phpmyadmin',
                name: 'phpMyAdmin',
                version: '5.2.x',
                category: 'Database Administration',
                desc: 'Web-based graphical user interface for managing MySQL and MariaDB databases, tables, columns, indexes, and queries.',
                icon: Database,
                badge: 'v5.2.1',
                specs: 'PHP 8.0+ • MySQL / MariaDB',
              },
            ].map((app) => {
              const Icon = app.icon;
              return (
                <div
                  key={app.id}
                  className="p-5 rounded-2xl bg-white dark:bg-[#10141d] border border-slate-200 dark:border-surface-800 shadow-xs hover:border-purple-500/40 dark:hover:border-purple-500/40 transition-all flex flex-col justify-between"
                >
                  <div>
                    <div className="flex items-start justify-between gap-3 mb-3">
                      <div className="p-2.5 rounded-xl bg-purple-500/10 text-purple-600 dark:text-purple-400">
                        <Icon className="w-5 h-5" />
                      </div>
                      <span className="px-2 py-0.5 rounded-full text-[10px] font-black bg-purple-500/10 text-purple-600 dark:text-purple-400 border border-purple-500/20">
                        {app.badge}
                      </span>
                    </div>

                    <h3 className="text-base font-bold text-slate-950 dark:text-white">{app.name}</h3>
                    <div className="text-[11px] font-semibold text-purple-600 dark:text-purple-400 mt-0.5 mb-2">{app.category}</div>
                    <p className="text-xs text-slate-600 dark:text-slate-400 leading-relaxed">{app.desc}</p>
                  </div>

                  <div className="mt-5 pt-3 border-t border-slate-100 dark:border-surface-800 flex items-center justify-between gap-2">
                    <div className="text-[10px] text-slate-500 dark:text-slate-400 font-mono truncate">
                      {app.specs}
                    </div>
                    <button
                      type="button"
                      onClick={() => {
                        if (websites.length === 0) {
                          alert('Please add a website under Websites first to deploy applications.');
                          return;
                        }
                        if (websites.length === 1) {
                          setDeploySite(websites[0]);
                          setSitePickerApp(app.id);
                        } else {
                          setSitePickerApp(app.id);
                        }
                      }}
                      className="inline-flex items-center gap-1.5 px-3.5 py-1.5 rounded-xl bg-purple-600 hover:bg-purple-500 text-white text-xs font-bold shadow-xs transition-all flex-shrink-0"
                    >
                      <Zap className="w-3 h-3" />
                      <span>Deploy App</span>
                    </button>
                  </div>
                </div>
              );
            })}
          </div>
        ) : viewMode === 'table' ? (
          <div className="rounded-2xl border border-surface-200 dark:border-surface-800 bg-white dark:bg-surface-900 overflow-hidden shadow-sm">
            <div className="overflow-x-auto">
              <table className="w-full text-left text-xs">
                <thead className="bg-surface-50 dark:bg-surface-800/60 border-b border-surface-200 dark:border-surface-800 text-slate-500 dark:text-slate-400 font-semibold uppercase tracking-wider">
                  <tr>
                    <th className="py-3.5 px-4">Software name</th>
                    <th className="py-3.5 px-3">Developer</th>
                    <th className="py-3.5 px-4 min-w-[280px]">Instructions</th>
                    <th className="py-3.5 px-3">Price</th>
                    <th className="py-3.5 px-3 text-center">Status</th>
                    <th className="py-3.5 px-3 text-center">Service</th>
                    <th className="py-3.5 px-4 text-right">Operate</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-surface-200 dark:divide-surface-800 text-slate-700 dark:text-slate-300">
                  {filteredApps.map((app) => {
                    const isActing = actionLoadingId === app.id;
                    return (
                      <tr
                        key={app.id}
                        className="hover:bg-surface-50 dark:hover:bg-surface-800/40 transition-colors"
                      >
                        {/* Name & Icon */}
                        <td className="py-3.5 px-4">
                          <div className="flex items-center gap-3">
                            <div className="w-8 h-8 rounded-lg bg-surface-100 dark:bg-surface-800 flex items-center justify-center border border-surface-200 dark:border-surface-700 flex-shrink-0">
                              {getCategoryIcon(app.category)}
                            </div>
                            <div>
                              <div className="font-semibold text-slate-900 dark:text-white flex items-center gap-1.5">
                                {app.name}
                                {app.id === 'supervisor' && (
                                  <span className="text-[10px] px-1.5 py-0.2 rounded bg-indigo-500/10 text-indigo-500 font-semibold border border-indigo-500/20">
                                    Process Mgr
                                  </span>
                                )}
                              </div>
                              <div className="text-[11px] text-slate-400 font-mono">
                                {app.display_name}
                              </div>
                            </div>
                          </div>
                        </td>

                        {/* Developer */}
                        <td className="py-3.5 px-3 text-slate-400 font-medium">
                          {app.developer}
                        </td>

                        {/* Instructions / Description */}
                        <td className="py-3.5 px-4 text-slate-500 dark:text-slate-400 line-clamp-2 pr-6">
                          {app.description}
                        </td>

                        {/* Price */}
                        <td className="py-3.5 px-3 font-medium text-emerald-600 dark:text-emerald-400">
                          {app.price}
                        </td>

                        {/* Status (Running / Stopped / Not Installed) */}
                        <td className="py-3.5 px-3 text-center">
                          {app.status === 'running' ? (
                            <span className="inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full text-[11px] font-medium bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 border border-emerald-500/20">
                              <span className="w-1.5 h-1.5 rounded-full bg-emerald-500 animate-pulse" />
                              Running
                            </span>
                          ) : app.status === 'stopped' ? (
                            <span className="inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full text-[11px] font-medium bg-amber-500/10 text-amber-600 dark:text-amber-400 border border-amber-500/20">
                              <span className="w-1.5 h-1.5 rounded-full bg-amber-500" />
                              Stopped
                            </span>
                          ) : (
                            <span className="text-slate-400 text-xs">--</span>
                          )}
                        </td>

                        {/* Service Control Toggle Switch */}
                        <td className="py-3.5 px-3 text-center">
                          {app.is_installed && app.service_name ? (
                            <button
                              onClick={() =>
                                handleServiceControl(
                                  app,
                                  app.status === 'running' ? 'stop' : 'start'
                                )
                              }
                              disabled={isActing}
                              title={app.status === 'running' ? 'Click to Stop' : 'Click to Start'}
                              className={`relative inline-flex h-5 w-9 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none ${
                                app.status === 'running'
                                  ? 'bg-emerald-500'
                                  : 'bg-slate-300 dark:bg-slate-700'
                              } disabled:opacity-50`}
                            >
                              <span
                                className={`pointer-events-none inline-block h-4 w-4 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out ${
                                  app.status === 'running' ? 'translate-x-4' : 'translate-x-0'
                                }`}
                              />
                            </button>
                          ) : (
                            <span className="text-slate-400 text-xs">--</span>
                          )}
                        </td>

                        {/* Actions (Install / Setting / Restart / Uninstall) */}
                        <td className="py-3.5 px-4 text-right whitespace-nowrap">
                          {isActing ? (
                            <span className="inline-flex items-center gap-1 text-xs text-indigo-500 font-medium">
                              <RefreshCw className="w-3.5 h-3.5 animate-spin" /> Processing...
                            </span>
                          ) : !app.is_installed ? (
                            <button
                              onClick={() => handleInstall(app)}
                              className="px-3 py-1.5 rounded-lg bg-emerald-600 hover:bg-emerald-500 text-white font-medium text-xs shadow-sm transition-all active:scale-95 flex items-center gap-1.5 ml-auto"
                            >
                              <Sparkles className="w-3 h-3" />
                              Install
                            </button>
                          ) : (
                            <div className="flex items-center justify-end gap-1.5">
                              {/* Open / Launch Button */}
                              <button
                                onClick={() => handleOpenApp(app)}
                                className="px-2.5 py-1 rounded-lg bg-indigo-600 hover:bg-indigo-500 text-white font-semibold text-xs shadow-sm flex items-center gap-1 transition-all active:scale-95"
                                title="Open & Manage Application"
                              >
                                <ExternalLink className="w-3 h-3" />
                                Open
                              </button>

                              {/* Pin to Dashboard */}
                              <button
                                onClick={() => {
                                  togglePinApp(app.id);
                                  setPinnedAppIds(getPinnedAppIds());
                                }}
                                className={`p-1.5 rounded-lg border transition-all ${
                                  pinnedAppIds.includes(app.id)
                                    ? 'bg-amber-500/10 text-amber-400 border-amber-500/30'
                                    : 'hover:bg-surface-200 dark:hover:bg-surface-700 text-slate-400 border-surface-200 dark:border-surface-700'
                                }`}
                                title={pinnedAppIds.includes(app.id) ? 'Pinned to Dashboard' : 'Pin to Dashboard'}
                              >
                                {pinnedAppIds.includes(app.id) ? (
                                  <BookmarkCheck className="w-3.5 h-3.5 text-amber-400" />
                                ) : (
                                  <Bookmark className="w-3.5 h-3.5" />
                                )}
                              </button>

                              {app.service_name && (
                                <button
                                  onClick={() => handleServiceControl(app, 'restart')}
                                  title="Restart Service"
                                  className="p-1.5 rounded-md hover:bg-surface-200 dark:hover:bg-surface-700 text-slate-500 dark:text-slate-400 hover:text-indigo-500 transition-colors"
                                >
                                  <RotateCw className="w-3.5 h-3.5" />
                                </button>
                              )}

                              <button
                                onClick={() => handleUninstall(app)}
                                className="px-2 py-1 rounded-md text-rose-500 hover:bg-rose-50 dark:hover:bg-rose-950/30 text-xs font-medium transition-colors"
                              >
                                Uninstall
                              </button>
                            </div>
                          )}
                        </td>
                      </tr>
                    );
                  })}
                </tbody>
              </table>
            </div>
          </div>
        ) : (
          /* Grid Cards View */
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
            {filteredApps.map((app) => {
              const isActing = actionLoadingId === app.id;
              return (
                <div
                  key={app.id}
                  className="rounded-2xl border border-surface-200 dark:border-surface-800 bg-white dark:bg-surface-900 p-5 shadow-sm hover:border-indigo-500/30 transition-all flex flex-col justify-between"
                >
                  <div>
                    <div className="flex items-start justify-between gap-3">
                      <div className="w-10 h-10 rounded-xl bg-surface-100 dark:bg-surface-800 flex items-center justify-center border border-surface-200 dark:border-surface-700 flex-shrink-0">
                        {getCategoryIcon(app.category)}
                      </div>
                      <div>
                        {app.status === 'running' ? (
                          <span className="inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full text-[11px] font-medium bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 border border-emerald-500/20">
                            <span className="w-1.5 h-1.5 rounded-full bg-emerald-500 animate-pulse" />
                            Running
                          </span>
                        ) : app.status === 'stopped' ? (
                          <span className="inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full text-[11px] font-medium bg-amber-500/10 text-amber-600 dark:text-amber-400 border border-amber-500/20">
                            Stopped
                          </span>
                        ) : (
                          <span className="inline-flex items-center px-2 py-0.5 rounded text-[11px] font-medium bg-surface-100 dark:bg-surface-800 text-slate-500 border border-surface-200 dark:border-surface-700">
                            Not Installed
                          </span>
                        )}
                      </div>
                    </div>

                    <div className="mt-3">
                      <h3 className="font-semibold text-slate-900 dark:text-white text-base flex items-center gap-2">
                        {app.name}
                        {app.id === 'supervisor' && (
                          <span className="text-[10px] px-1.5 py-0.2 rounded bg-indigo-500/10 text-indigo-500 font-semibold border border-indigo-500/20">
                            Process Mgr
                          </span>
                        )}
                      </h3>
                      <p className="text-xs text-slate-400 font-mono mt-0.5">{app.display_name}</p>
                      <p className="text-xs text-slate-500 dark:text-slate-400 mt-2 line-clamp-3 leading-relaxed">
                        {app.description}
                      </p>
                    </div>
                  </div>

                  <div className="mt-5 pt-4 border-t border-surface-100 dark:border-surface-800 flex items-center justify-between">
                    <span className="text-xs font-semibold text-emerald-600 dark:text-emerald-400">
                      {app.price}
                    </span>

                    <div>
                      {isActing ? (
                        <span className="inline-flex items-center gap-1 text-xs text-indigo-500 font-medium">
                          <RefreshCw className="w-3.5 h-3.5 animate-spin" /> Processing...
                        </span>
                      ) : !app.is_installed ? (
                        <button
                          onClick={() => handleInstall(app)}
                          className="px-3.5 py-1.5 rounded-xl bg-emerald-600 hover:bg-emerald-500 text-white font-medium text-xs shadow-sm transition-all active:scale-95 flex items-center gap-1.5"
                        >
                          <Sparkles className="w-3 h-3" />
                          1-Click Install
                        </button>
                      ) : (
                        <div className="flex items-center gap-1.5">
                          {/* Open / Manage Button */}
                          <button
                            onClick={() => handleOpenApp(app)}
                            className="px-2.5 py-1 rounded-lg bg-indigo-600 hover:bg-indigo-500 text-white font-semibold text-xs shadow-sm flex items-center gap-1 transition-all active:scale-95"
                          >
                            <ExternalLink className="w-3 h-3" />
                            Open
                          </button>

                          {/* Pin to Dashboard */}
                          <button
                            onClick={() => {
                              togglePinApp(app.id);
                              setPinnedAppIds(getPinnedAppIds());
                            }}
                            className={`p-1.5 rounded-lg border transition-all ${
                              pinnedAppIds.includes(app.id)
                                ? 'bg-amber-500/10 text-amber-400 border-amber-500/30'
                                : 'hover:bg-surface-200 dark:hover:bg-surface-700 text-slate-400 border-surface-200 dark:border-surface-700'
                            }`}
                            title={pinnedAppIds.includes(app.id) ? 'Pinned to Dashboard' : 'Pin to Dashboard'}
                          >
                            {pinnedAppIds.includes(app.id) ? (
                              <BookmarkCheck className="w-3.5 h-3.5 text-amber-400" />
                            ) : (
                              <Bookmark className="w-3.5 h-3.5" />
                            )}
                          </button>

                          {app.service_name && (
                            <button
                              onClick={() =>
                                handleServiceControl(
                                  app,
                                  app.status === 'running' ? 'stop' : 'start'
                                )
                              }
                              className="px-2 py-1 rounded-lg bg-surface-100 dark:bg-surface-800 hover:bg-surface-200 dark:hover:bg-surface-700 text-xs font-medium text-slate-700 dark:text-slate-300 transition-colors"
                            >
                              {app.status === 'running' ? 'Stop' : 'Start'}
                            </button>
                          )}
                          <button
                            onClick={() => handleUninstall(app)}
                            className="px-2 py-1 rounded-lg text-rose-500 hover:bg-rose-50 dark:hover:bg-rose-950/30 text-xs font-medium transition-colors"
                          >
                            Uninstall
                          </button>
                        </div>
                      )}
                    </div>
                  </div>
                </div>
              );
            })}
          </div>
        )}

        {/* Live Installation Progress Modal */}
        {jobModalOpen && activeJob && (
          <div className="fixed inset-0 z-50 bg-black/60 backdrop-blur-sm flex items-center justify-center p-4">
            <div className="bg-white dark:bg-slate-900 rounded-2xl border border-surface-200 dark:border-slate-800 shadow-2xl max-w-2xl w-full overflow-hidden flex flex-col max-h-[85vh]">
              {/* Modal Header */}
              <div className="px-6 py-4 border-b border-surface-200 dark:border-slate-800 flex items-center justify-between">
                <div>
                  <h3 className="font-bold text-slate-900 dark:text-white flex items-center gap-2">
                    {activeJob.status === 'running' ? (
                      <RefreshCw className="w-4 h-4 text-indigo-500 animate-spin" />
                    ) : activeJob.status === 'completed' ? (
                      <CheckCircle2 className="w-4 h-4 text-emerald-500" />
                    ) : (
                      <AlertCircle className="w-4 h-4 text-rose-500" />
                    )}
                    {activeJob.action === 'install' ? 'Installing' : 'Uninstalling'} {activeJob.app_id}
                  </h3>
                  <p className="text-xs text-slate-500 mt-0.5">
                    {activeJob.status === 'running'
                      ? 'Package script executing in server background...'
                      : activeJob.status === 'completed'
                      ? 'Execution completed successfully!'
                      : 'Execution encountered an issue.'}
                  </p>
                </div>
                <button
                  onClick={() => setJobModalOpen(false)}
                  className="p-1 rounded-lg text-slate-400 hover:text-slate-600 dark:hover:text-white hover:bg-surface-100 dark:hover:bg-slate-800"
                >
                  <X className="w-4 h-4" />
                </button>
              </div>

              {/* Progress Bar */}
              <div className="w-full bg-surface-100 dark:bg-slate-800 h-1.5">
                <div
                  className={`h-full transition-all duration-300 ${
                    activeJob.status === 'failed'
                      ? 'bg-rose-500'
                      : activeJob.status === 'completed'
                      ? 'bg-emerald-500'
                      : 'bg-indigo-500'
                  }`}
                  style={{ width: `${activeJob.progress}%` }}
                />
              </div>

              {/* Terminal Logs Window */}
              <div className="flex-1 overflow-y-auto p-4 bg-[#0a0e14] font-mono text-xs text-slate-300 space-y-1 select-text min-h-[250px] max-h-[380px]">
                {activeJob.logs?.map((line, idx) => (
                  <div
                    key={idx}
                    className={`leading-relaxed ${
                      line.includes('ERROR') || line.includes('failed')
                        ? 'text-rose-400 font-semibold'
                        : line.includes('completed successfully')
                        ? 'text-emerald-400 font-semibold'
                        : ''
                    }`}
                  >
                    {line}
                  </div>
                ))}
                <div ref={logsEndRef} />
              </div>

              {/* Modal Footer */}
              <div className="px-6 py-3 border-t border-surface-200 dark:border-slate-800 flex items-center justify-between bg-surface-50 dark:bg-slate-900/50">
                <span className="text-xs text-slate-500">
                  Status: <strong className="uppercase">{activeJob.status}</strong>
                </span>
                <button
                  onClick={() => setJobModalOpen(false)}
                  className="px-4 py-1.5 rounded-lg bg-indigo-600 hover:bg-indigo-500 text-white text-xs font-semibold"
                >
                  {activeJob.status === 'running' ? 'Hide Window' : 'Close'}
                </button>
              </div>
            </div>
          </div>
        )}

        {/* Interactive App Open & Control Modal */}
        <AppControlModal
          app={selectedControlApp}
          isOpen={controlModalOpen}
          onClose={() => setControlModalOpen(false)}
          onServiceControl={handleServiceControl}
          isActing={actionLoadingId === selectedControlApp?.id}
        />

        {/* Site Picker Modal if multiple websites exist */}
        {sitePickerApp && !deploySite && (
          <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-slate-950/60 backdrop-blur-sm animate-in fade-in duration-150">
            <div className="w-full max-w-md rounded-2xl bg-white dark:bg-[#12161f] border border-slate-200 dark:border-slate-800 shadow-2xl p-6">
              <div className="flex items-center justify-between pb-3 border-b border-slate-100 dark:border-slate-800">
                <div className="flex items-center gap-2">
                  <Globe className="w-5 h-5 text-purple-600 dark:text-purple-400" />
                  <h3 className="text-base font-bold text-slate-950 dark:text-white">
                    Select Target Website
                  </h3>
                </div>
                <button
                  onClick={() => setSitePickerApp(null)}
                  className="p-1 rounded-lg text-slate-400 hover:text-slate-600 dark:hover:text-white hover:bg-slate-100 dark:hover:bg-slate-800"
                >
                  <X className="w-4 h-4" />
                </button>
              </div>
              <p className="text-xs text-slate-600 dark:text-slate-400 mt-2 mb-4">
                Choose the domain or website where you want to deploy this application:
              </p>
              <div className="space-y-2 max-h-60 overflow-y-auto">
                {websites.map((site) => (
                  <button
                    key={site.id}
                    onClick={() => setDeploySite(site)}
                    className="w-full text-left p-3 rounded-xl border border-slate-200 dark:border-slate-800 hover:border-purple-500 hover:bg-purple-500/5 transition-all flex items-center justify-between group"
                  >
                    <div>
                      <div className="text-xs font-bold text-slate-950 dark:text-white group-hover:text-purple-600 dark:group-hover:text-purple-400">
                        {site.primary_domain}
                      </div>
                      <div className="text-[10px] text-slate-500 font-mono mt-0.5">
                        {site.document_root} • {site.php_version || 'Node.js'}
                      </div>
                    </div>
                    <ChevronRight className="w-4 h-4 text-slate-400 group-hover:text-purple-500 transition-transform group-hover:translate-x-0.5" />
                  </button>
                ))}
              </div>
            </div>
          </div>
        )}

        {/* 1-Click Application Installer Modal */}
        {deploySite && (
          <OneClickAppModal
            isOpen={!!deploySite}
            onClose={() => {
              setDeploySite(null);
              setSitePickerApp(null);
            }}
            website={deploySite}
            initialAppId={sitePickerApp || undefined}
            onSuccess={() => {
              loadApps();
            }}
          />
        )}
      </div>
    </DashboardShell>
  );
}
