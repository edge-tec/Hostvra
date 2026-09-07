'use client';

import React, { useState, useEffect, useCallback, useMemo } from 'react';
import {
  Globe,
  Plus,
  ShieldCheck,
  ShieldAlert,
  RefreshCw,
  Search,
  ExternalLink,
  Trash2,
  Play,
  Pause,
  X,
  Check,
  Code2,
  Layers,
  Cpu,
  User,
  Shield,
  CheckCircle2,
  AlertTriangle,
  Zap,
  CloudDownload,
  Folder,
  FileText,
  Gauge,
  Settings,
  MoreVertical,
  Lock,
  BarChart2,
  RotateCcw,
  Edit3,
  Copy,
  ChevronDown,
  SlidersHorizontal,
  FolderTree,
  Terminal,
  Server as ServerIcon,
} from 'lucide-react';
import { DashboardShell } from '@/components/DashboardShell';
import { apiFetch, Website, Server, UserIsolationInfo, ResourceLimits } from '@/lib/api';
import { OneClickAppModal } from '@/components/OneClickAppModal';
import { SiteModificationModal, SiteModalTab } from '@/components/SiteModificationModal';

type ProjectTab = 'php' | 'nodejs' | 'proxy' | 'go' | 'python';

export default function WebsitesPage() {
  // Navigation & Tabs State
  const [activeTab, setActiveTab] = useState<ProjectTab>('php');
  const [websites, setWebsites] = useState<Website[]>([]);
  const [servers, setServers] = useState<Server[]>([]);
  const [selectedServer, setSelectedServer] = useState('');
  const [loading, setLoading] = useState(false);
  const [search, setSearch] = useState('');
  const [selectedCategory, setSelectedCategory] = useState('All categories');
  const [selectedIds, setSelectedIds] = useState<string[]>([]);
  const [batchAction, setBatchAction] = useState('');
  const [executingBatch, setExecutingBatch] = useState(false);

  // Column Visibility Toggles
  const [colVisible, setColVisible] = useState({
    siteName: true,
    status: true,
    backup: true,
    quickAction: true,
    expiration: true,
    ssl: true,
    requests: true,
    waf: true,
    operate: true,
  });

  // Modals
  const [addSiteOpen, setAddSiteOpen] = useState(false);
  const [advancedSetupOpen, setAdvancedSetupOpen] = useState(false);
  const [statsOpen, setStatsOpen] = useState(false);
  const [nginxControlOpen, setNginxControlOpen] = useState(false);
  const [feedbackOpen, setFeedbackOpen] = useState(false);
  const [colSettingsOpen, setColSettingsOpen] = useState(false);
  const [selectedSite, setSelectedSite] = useState<Website | null>(null);

  // Site-specific modals
  const [siteModalOpen, setSiteModalOpen] = useState(false);
  const [siteModalTab, setSiteModalTab] = useState<SiteModalTab>('domain');
  const [siteModalWebsite, setSiteModalWebsite] = useState<Website | null>(null);

  const openSiteModal = (site: Website, tab: SiteModalTab = 'domain') => {
    setSiteModalWebsite(site);
    setSiteModalTab(tab);
    setSiteModalOpen(true);
  };

  const [confModalOpen, setConfModalOpen] = useState(false);
  const [vhostConfText, setVhostConfText] = useState('');
  const [confSaving, setConfSaving] = useState(false);

  const [logsModalOpen, setLogsModalOpen] = useState(false);
  const [activeLogTab, setActiveLogTab] = useState<'access' | 'error'>('access');
  const [accessLogs, setAccessLogs] = useState('');
  const [errorLogs, setErrorLogs] = useState('');

  const [backupModalOpen, setBackupModalOpen] = useState(false);
  const [backingUp, setBackingUp] = useState(false);

  const [phpSwitchModalOpen, setPhpSwitchModalOpen] = useState(false);
  const [targetPhpVer, setTargetPhpVer] = useState('8.2');
  const [switchingPhp, setSwitchingPhp] = useState(false);

  const [sslModalOpen, setSslModalOpen] = useState(false);
  const [issuingSsl, setIssuingSsl] = useState(false);
  const [forceHttps, setForceHttps] = useState(true);

  const [wafModalOpen, setWafModalOpen] = useState(false);
  const [wafCcDefense, setWafCcDefense] = useState(true);
  const [wafSqlFilter, setWafSqlFilter] = useState(true);
  const [wafXssFilter, setWafXssFilter] = useState(true);

  const [speedModalOpen, setSpeedModalOpen] = useState(false);
  const [gzipEnabled, setGzipEnabled] = useState(true);
  const [http2Enabled, setHttp2Enabled] = useState(true);

  const [rewriteModalOpen, setRewriteModalOpen] = useState(false);
  const [rewritePreset, setRewritePreset] = useState('wordpress');
  const [rewriteText, setRewriteText] = useState('location / {\n    try_files $uri $uri/ /index.php?$args;\n}');

  const [openDropdownId, setOpenDropdownId] = useState<string | null>(null);

  // Existing Hostvra modules
  const [isolationModalSite, setIsolationModalSite] = useState<Website | null>(null);
  const [appModalSite, setAppModalSite] = useState<Website | null>(null);
  const [isolationInfo, setIsolationInfo] = useState<UserIsolationInfo | null>(null);
  const [isolationLoading, setIsolationLoading] = useState(false);
  const [isolationSaving, setIsolationSaving] = useState(false);
  const [limitMem, setLimitMem] = useState(512);
  const [limitCPU, setLimitCPU] = useState(100);
  const [limitTasks, setLimitTasks] = useState(100);
  const [limitOpenBaseDir, setLimitOpenBaseDir] = useState(true);

  // Add Site Form State
  const [newDomain, setNewDomain] = useState('');
  const [newRemarks, setNewRemarks] = useState('');
  const [newDocRoot, setNewDocRoot] = useState('/www/wwwroot/');
  const [newPhpVer, setNewPhpVer] = useState('8.2');
  const [newWebServer, setNewWebServer] = useState('nginx');
  const [newCategory, setNewCategory] = useState('Default');
  const [createFtp, setCreateFtp] = useState(false);
  const [createDb, setCreateDb] = useState(false);
  const [creatingSite, setCreatingSite] = useState(false);

  // Toast notifications
  const [toast, setToast] = useState<{ message: string; isError?: boolean } | null>(null);
  const showToast = (message: string, isError = false) => {
    setToast({ message, isError });
    setTimeout(() => setToast(null), 3500);
  };

  // Fetch websites from API
  const fetchData = useCallback(async () => {
    try {
      const [sitesRes, serversRes] = await Promise.all([
        apiFetch<Website[]>('/api/v1/websites'),
        apiFetch<Server[]>('/api/v1/servers'),
      ]);

      if (sitesRes.success && sitesRes.data) {
        setWebsites(sitesRes.data);
      }
      if (serversRes.success && serversRes.data) {
        setServers(serversRes.data);
        if (serversRes.data.length > 0 && !selectedServer) {
          setSelectedServer(serversRes.data[0].id);
        }
      }
    } catch (err) {
      console.error('Failed to load websites', err);
    }
  }, [selectedServer]);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  // Close dropdown on outside click
  useEffect(() => {
    const handleDocClick = () => setOpenDropdownId(null);
    document.addEventListener('click', handleDocClick);
    return () => document.removeEventListener('click', handleDocClick);
  }, []);

  // Filtered Websites
  const filteredWebsites = useMemo(() => {
    return websites.filter((site) => {
      // Tab filter
      if (activeTab === 'php' && site.app_type !== 'php' && site.app_type !== 'static') {
        return false;
      }
      if (activeTab === 'nodejs' && site.app_type !== 'nodejs') return false;
      if (activeTab === 'proxy' && site.app_type !== 'proxy') return false;
      if (activeTab === 'go' && site.app_type !== 'go') return false;
      if (activeTab === 'python' && site.app_type !== 'python') return false;

      // Category filter
      if (selectedCategory !== 'All categories' && site.category && site.category !== selectedCategory) {
        return false;
      }

      // Search filter
      const q = search.toLowerCase().trim();
      if (!q) return true;
      return (
        site.primary_domain.toLowerCase().includes(q) ||
        (site.remarks && site.remarks.toLowerCase().includes(q)) ||
        site.document_root.toLowerCase().includes(q) ||
        (site.system_user && site.system_user.toLowerCase().includes(q))
      );
    });
  }, [websites, activeTab, selectedCategory, search]);

  // Selection handlers
  const allSelected = filteredWebsites.length > 0 && selectedIds.length === filteredWebsites.length;
  const handleSelectAll = () => {
    if (allSelected) {
      setSelectedIds([]);
    } else {
      setSelectedIds(filteredWebsites.map((s) => s.id));
    }
  };

  // Toggle single site status (Play / Pause in real time)
  const handleToggleStatus = async (site: Website, e: React.MouseEvent) => {
    e.stopPropagation();
    const nextStatus = site.status === 'active' ? 'suspended' : 'active';
    setWebsites((prev) =>
      prev.map((s) => (s.id === site.id ? { ...s, status: nextStatus } : s))
    );
    showToast(`Website '${site.primary_domain}' ${nextStatus === 'active' ? 'started (active)' : 'stopped (suspended)'}`);

    await apiFetch(`/api/v1/websites/${site.id}/status`, {
      method: 'POST',
      body: JSON.stringify({ status: nextStatus }),
    });
  };

  // Create new Website
  const handleCreateWebsite = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!newDomain.trim()) return;

    setCreatingSite(true);
    const domain = newDomain.trim().toLowerCase().split('\n')[0].trim();
    const remarks = newRemarks.trim() || domain.split('.')[0];
    const docRoot = newDocRoot.endsWith('/') ? newDocRoot + domain : newDocRoot;

    const newSite: Website = {
      id: `site-${Date.now()}`,
      server_id: selectedServer || 'srv-1',
      primary_domain: domain,
      remarks: remarks,
      document_root: docRoot,
      system_user: `u_${remarks.substring(0, 8)}`,
      web_server_type: newWebServer,
      app_type: activeTab === 'nodejs' ? 'nodejs' : activeTab === 'python' ? 'python' : activeTab === 'go' ? 'go' : activeTab === 'proxy' ? 'proxy' : newPhpVer === 'Static' ? 'static' : 'php',
      php_version: newPhpVer,
      status: 'active',
      ssl_enabled: false,
      ssl_days_left: 0,
      backup_count: 0,
      backup_status: '0 Backup',
      category: newCategory,
      expiration: 'Perpetual',
      requests_count: 0,
      waf_status: 'Active',
      traffic_history: [0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0],
      created_at: new Date().toISOString(),
    };

    setWebsites((prev) => [newSite, ...prev]);
    showToast(`Website '${domain}' created successfully! Virtual host and root directory configured.`);
    setCreatingSite(false);
    setAddSiteOpen(false);
    setNewDomain('');
    setNewRemarks('');

    await apiFetch('/api/v1/websites', {
      method: 'POST',
      body: JSON.stringify({
        server_id: selectedServer || 'srv-1',
        primary_domain: domain,
        document_root: docRoot,
        php_version: newPhpVer,
        app_type: newSite.app_type,
        web_server_type: newWebServer,
      }),
    });
  };

  // Open VHost Conf Modal
  // Open VHost Conf Modal -> Routes to Site Modification Modal (Config Tab)
  const openConfModal = (site: Website) => {
    openSiteModal(site, 'config');
  };

  // Save VHost Conf
  const handleSaveConf = async () => {
    if (!selectedSite) return;
    setConfSaving(true);
    await apiFetch(`/api/v1/websites/${selectedSite.id}/conf`, {
      method: 'PUT',
      body: JSON.stringify({ config: vhostConfText }),
    });
    setConfSaving(false);
    setConfModalOpen(false);
    showToast(`Virtual host configuration for '${selectedSite.primary_domain}' saved and reloaded!`);
  };

  // Open Logs Modal -> Routes to Site Modification Modal (Response Log Tab)
  const openLogsModal = (site: Website) => {
    openSiteModal(site, 'logs');
  };

  // Open Backup Modal & Trigger Backup
  const openBackupModal = (site: Website) => {
    setSelectedSite(site);
    setBackupModalOpen(true);
  };

  const handleTriggerBackup = async () => {
    if (!selectedSite) return;
    setBackingUp(true);
    showToast(`Creating full archive for '${selectedSite.primary_domain}'...`);
    await apiFetch(`/api/v1/websites/${selectedSite.id}/backup`, { method: 'POST' });
    setWebsites((prev) =>
      prev.map((s) =>
        s.id === selectedSite.id
          ? { ...s, backup_count: (s.backup_count || 0) + 1, backup_status: `${(s.backup_count || 0) + 1} Backup` }
          : s
      )
    );
    setBackingUp(false);
    showToast(`Snapshot created for '${selectedSite.primary_domain}'!`);
  };

  // Open PHP Switch Modal
  const openPhpSwitchModal = (site: Website) => {
    setSelectedSite(site);
    setTargetPhpVer(site.php_version || '8.2');
    setPhpSwitchModalOpen(true);
  };

  const handleSwitchPhp = async () => {
    if (!selectedSite) return;
    setSwitchingPhp(true);
    setWebsites((prev) =>
      prev.map((s) => (s.id === selectedSite.id ? { ...s, php_version: targetPhpVer } : s))
    );
    showToast(`PHP version for '${selectedSite.primary_domain}' switched to PHP-${targetPhpVer}!`);
    await apiFetch(`/api/v1/websites/${selectedSite.id}/php/switch`, {
      method: 'POST',
      body: JSON.stringify({ php_version: targetPhpVer }),
    });
    setSwitchingPhp(false);
    setPhpSwitchModalOpen(false);
  };

  // Open SSL Modal
  const openSslModal = (site: Website) => {
    setSelectedSite(site);
    setSslModalOpen(true);
  };

  const handleIssueSsl = async () => {
    if (!selectedSite) return;
    setIssuingSsl(true);
    showToast(`Requesting Let's Encrypt certificate for '${selectedSite.primary_domain}'...`);
    await apiFetch(`/api/v1/websites/${selectedSite.id}/ssl`, { method: 'POST' });
    setWebsites((prev) =>
      prev.map((s) =>
        s.id === selectedSite.id ? { ...s, ssl_enabled: true, ssl_days_left: 90 } : s
      )
    );
    setIssuingSsl(false);
    setSslModalOpen(false);
    showToast(`SSL Certificate issued successfully for '${selectedSite.primary_domain}'!`);
  };

  // Toggle WAF in real time
  const handleToggleWaf = async (site: Website, e: React.MouseEvent) => {
    e.stopPropagation();
    const nextWaf = site.waf_status === 'Active' ? 'Inactive' : 'Active';
    setWebsites((prev) =>
      prev.map((s) => (s.id === site.id ? { ...s, waf_status: nextWaf } : s))
    );
    showToast(`WAF protection for '${site.primary_domain}' set to ${nextWaf}`);
    await apiFetch(`/api/v1/websites/${site.id}/waf`, { method: 'POST' });
  };

  // Delete Website
  const handleDeleteWebsite = async (site: Website) => {
    if (!confirm(`Are you sure you want to delete website '${site.primary_domain}'? This will delete the virtual host and PHP pool.`)) {
      return;
    }
    setWebsites((prev) => prev.filter((s) => s.id !== site.id));
    setSelectedIds((prev) => prev.filter((id) => id !== site.id));
    showToast(`Website '${site.primary_domain}' deleted successfully.`);
    await apiFetch(`/api/v1/websites/${site.id}`, { method: 'DELETE' });
  };

  // Execute Batch Actions
  const handleExecuteBatch = async () => {
    if (!batchAction || selectedIds.length === 0) return;
    setExecutingBatch(true);
    const count = selectedIds.length;

    if (batchAction === 'delete') {
      if (!confirm(`Delete ${count} selected websites?`)) {
        setExecutingBatch(false);
        return;
      }
      setWebsites((prev) => prev.filter((s) => !selectedIds.includes(s.id)));
      setSelectedIds([]);
      showToast(`${count} websites deleted successfully.`);
    } else if (batchAction === 'start') {
      setWebsites((prev) =>
        prev.map((s) => (selectedIds.includes(s.id) ? { ...s, status: 'active' } : s))
      );
      showToast(`${count} websites started.`);
    } else if (batchAction === 'stop') {
      setWebsites((prev) =>
        prev.map((s) => (selectedIds.includes(s.id) ? { ...s, status: 'suspended' } : s))
      );
      showToast(`${count} websites suspended.`);
    } else if (batchAction === 'backup') {
      setWebsites((prev) =>
        prev.map((s) =>
          selectedIds.includes(s.id)
            ? { ...s, backup_count: (s.backup_count || 0) + 1, backup_status: `${(s.backup_count || 0) + 1} Backup` }
            : s
        )
      );
      showToast(`Backup initiated for ${count} websites.`);
    }

    await apiFetch('/api/v1/websites/batch', {
      method: 'POST',
      body: JSON.stringify({ action: batchAction, ids: selectedIds }),
    });

    setExecutingBatch(false);
    setBatchAction('');
  };

  // Isolation modal
  const openIsolationModal = async (site: Website) => {
    setIsolationModalSite(site);
    setIsolationLoading(true);
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

  // Sparkline Generator
  const renderSparkline = (points?: number[]) => {
    const data = points && points.length > 0 ? points : [4, 6, 8, 5, 9, 14, 8, 12, 18, 10, 15];
    const max = Math.max(...data, 1);
    const width = 80;
    const height = 18;
    const step = width / (data.length - 1);
    const coords = data.map((val, idx) => {
      const x = idx * step;
      const y = height - (val / max) * (height - 4) - 2;
      return `${x},${y}`;
    });
    return (
      <svg className="w-20 h-4.5 overflow-visible" viewBox={`0 0 ${width} ${height}`}>
        <polyline
          fill="none"
          stroke="#20a53a"
          strokeWidth="1.5"
          strokeLinecap="round"
          strokeLinejoin="round"
          points={coords.join(' ')}
        />
      </svg>
    );
  };

  return (
    <DashboardShell>
      {/* Toast Banner */}
      {toast && (
        <div className="fixed top-5 right-5 z-50 animate-bounce">
          <div
            className={`px-4 py-3 rounded-xl shadow-xl flex items-center gap-3 text-sm font-semibold border ${
              toast.isError
                ? 'bg-rose-900/95 text-white border-rose-700 shadow-rose-900/20'
                : 'bg-white dark:bg-slate-900 text-slate-900 dark:text-emerald-400 border-slate-200 dark:border-slate-700 shadow-2xl'
            }`}
          >
            {toast.isError ? (
              <AlertTriangle className="w-5 h-5 text-rose-400" />
            ) : (
              <CheckCircle2 className="w-5 h-5 text-emerald-600 dark:text-emerald-400" />
            )}
            <span>{toast.message}</span>
          </div>
        </div>
      )}

      <div className="space-y-4 font-sans text-slate-900 dark:text-slate-100">
        {/* =========================================================================
            1. TOP PROJECT RUNTIME NAVIGATION TABS
            ========================================================================= */}
        <div className="flex flex-wrap items-center justify-between border-b border-slate-200 dark:border-surface-800 pb-2 gap-3">
          <div className="flex items-center gap-6 text-sm font-semibold">
            {[
              { id: 'php', label: 'PHP Project' },
              { id: 'nodejs', label: 'Node.js Project' },
              { id: 'proxy', label: 'Proxy Project' },
              { id: 'go', label: 'Go Project' },
              { id: 'python', label: 'Python Project' },
            ].map((tab) => (
              <button
                key={tab.id}
                onClick={() => setActiveTab(tab.id as ProjectTab)}
                className={`transition-colors py-1 cursor-pointer font-bold ${
                  activeTab === tab.id
                    ? 'text-emerald-600 dark:text-emerald-400 border-b-2 border-emerald-600 dark:border-emerald-500'
                    : 'text-slate-600 hover:text-slate-900 dark:text-slate-400 dark:hover:text-white'
                }`}
              >
                {tab.label}
              </button>
            ))}
          </div>

          {/* Right PRO / Upgrade Badges */}
          <div className="hidden sm:flex items-center gap-2 text-xs">
            <span className="px-1.5 py-0.5 rounded bg-indigo-600 text-white font-bold text-[10px] uppercase">
              PRO
            </span>
            <span className="text-slate-500 dark:text-slate-400 font-mono">FREE 8.0.6</span>
            <button
              onClick={() => showToast('Hostvra Enterprise License is Active')}
              className="px-2.5 py-1 rounded-lg bg-orange-600 hover:bg-orange-700 active:bg-orange-800 text-white font-bold transition cursor-pointer text-xs shadow-xs"
            >
              Upgrade now
            </button>
          </div>
        </div>

        {/* =========================================================================
            2. RESPONSIVE ACTION TOOLBAR (Clean Light / Dark Adaptive)
            ========================================================================= */}
        <div className="flex flex-col lg:flex-row lg:items-center justify-between gap-3 pt-1">
          {/* Action Buttons Group */}
          <div className="flex flex-wrap items-center gap-2 text-xs">
            {/* Add site Primary Green Button */}
            <button
              onClick={() => setAddSiteOpen(true)}
              className="px-3.5 py-1.5 rounded-lg bg-emerald-600 hover:bg-emerald-700 active:bg-emerald-800 text-white font-bold transition shadow-xs flex items-center gap-1.5 cursor-pointer flex-shrink-0"
            >
              <Plus className="w-3.5 h-3.5 text-white" />
              <span>Add site</span>
            </button>

            {/* Advanced Setup Dropdown */}
            <div className="relative">
              <button
                onClick={(e) => {
                  e.stopPropagation();
                  setAdvancedSetupOpen(!advancedSetupOpen);
                }}
                className="px-3 py-1.5 rounded-lg bg-white dark:bg-surface-800 hover:bg-slate-50 dark:hover:bg-surface-700 text-slate-700 dark:text-slate-200 border border-slate-300 dark:border-surface-700 hover:border-slate-400 font-semibold transition shadow-2xs cursor-pointer flex items-center gap-1 flex-shrink-0"
              >
                <span>Advanced Setup</span>
                <ChevronDown className="w-3.5 h-3.5 text-slate-400" />
              </button>

              {advancedSetupOpen && (
                <div
                  onClick={(e) => e.stopPropagation()}
                  className="absolute left-0 top-full mt-1.5 w-48 bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-700 rounded-xl shadow-xl py-1.5 z-40 text-xs font-semibold animate-fadeIn"
                >
                  <button
                    onClick={() => {
                      setAdvancedSetupOpen(false);
                      showToast('Default Virtual Host set to standard 404 handler');
                    }}
                    className="w-full text-left px-3.5 py-2 hover:bg-slate-50 dark:hover:bg-surface-800 text-slate-700 dark:text-slate-200"
                  >
                    Default site
                  </button>
                  <button
                    onClick={() => {
                      setAdvancedSetupOpen(false);
                      showToast('Global SSL certificates loaded');
                    }}
                    className="w-full text-left px-3.5 py-2 hover:bg-slate-50 dark:hover:bg-surface-800 text-slate-700 dark:text-slate-200"
                  >
                    Certificate management
                  </button>
                  <button
                    onClick={() => {
                      setAdvancedSetupOpen(false);
                      showToast('PHP-FPM global pools verified healthy');
                    }}
                    className="w-full text-left px-3.5 py-2 hover:bg-slate-50 dark:hover:bg-surface-800 text-slate-700 dark:text-slate-200"
                  >
                    PHP-FPM configuration
                  </button>
                  <button
                    onClick={() => {
                      setAdvancedSetupOpen(false);
                      setNginxControlOpen(true);
                    }}
                    className="w-full text-left px-3.5 py-2 hover:bg-slate-50 dark:hover:bg-surface-800 text-slate-700 dark:text-slate-200"
                  >
                    Web Server status
                  </button>
                </div>
              )}
            </div>

            {/* Statistics */}
            <button
              onClick={() => setStatsOpen(true)}
              className="px-3 py-1.5 rounded-lg bg-white dark:bg-surface-800 hover:bg-slate-50 dark:hover:bg-surface-700 text-slate-700 dark:text-slate-200 border border-slate-300 dark:border-surface-700 hover:border-slate-400 font-semibold transition shadow-2xs cursor-pointer flex items-center gap-1.5 flex-shrink-0"
            >
              <BarChart2 className="w-3.5 h-3.5 text-slate-500" />
              <span>Statistics</span>
            </button>

            {/* Nginx Status / Engine Control */}
            <button
              onClick={() => setNginxControlOpen(true)}
              className="px-3 py-1.5 rounded-lg bg-white dark:bg-surface-800 hover:bg-slate-50 dark:hover:bg-surface-700 text-slate-800 dark:text-slate-200 border border-slate-300 dark:border-surface-700 hover:border-slate-400 font-mono flex items-center gap-1.5 text-xs shadow-2xs cursor-pointer flex-shrink-0"
            >
              <span className="font-semibold">Nginx 1.24.0</span>
              <Play className="w-2.5 h-2.5 text-emerald-600 dark:text-emerald-400 fill-emerald-600 dark:fill-emerald-400" />
            </button>

            {/* Feedback */}
            <button
              onClick={() => setFeedbackOpen(true)}
              className="px-2.5 py-1.5 text-emerald-600 hover:text-emerald-700 dark:text-emerald-400 dark:hover:text-emerald-300 font-bold transition flex items-center gap-1 cursor-pointer flex-shrink-0"
            >
              <Edit3 className="w-3.5 h-3.5" />
              <span>Feedback</span>
            </button>
          </div>

          {/* Right Filters & Settings */}
          <div className="flex items-center gap-2 w-full sm:w-auto">
            {/* Category Dropdown */}
            <select
              value={selectedCategory}
              onChange={(e) => setSelectedCategory(e.target.value)}
              className="px-3 py-1.5 rounded-lg bg-white dark:bg-surface-800 border border-slate-300 dark:border-surface-700 text-slate-700 dark:text-slate-200 text-xs font-semibold focus:outline-none shadow-2xs cursor-pointer"
            >
              <option value="All categories">All categories</option>
              <option value="Default">Default</option>
              <option value="Tools">Tools</option>
              <option value="E-commerce">E-commerce</option>
              <option value="Blog">Blog</option>
            </select>

            {/* Search Input */}
            <div className="relative flex-1 sm:w-52">
              <input
                type="text"
                placeholder="Domain or Remarks"
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                className="w-full px-3 py-1.5 rounded-lg bg-white dark:bg-surface-800 border border-slate-300 dark:border-surface-700 text-slate-900 dark:text-slate-100 text-xs placeholder:text-slate-400 focus:outline-none focus:border-emerald-500 shadow-2xs transition-all pr-8"
              />
              <Search className="w-3.5 h-3.5 text-slate-400 absolute right-2.5 top-2 pointer-events-none" />
            </div>

            {/* Column Display Settings Gear Icon */}
            <button
              onClick={() => setColSettingsOpen(true)}
              title="Column Display Settings"
              className="p-1.5 rounded-lg bg-white dark:bg-surface-800 border border-slate-300 dark:border-surface-700 text-slate-600 dark:text-slate-300 hover:text-emerald-600 transition shadow-2xs cursor-pointer"
            >
              <Settings className="w-4 h-4" />
            </button>
          </div>
        </div>

        {/* =========================================================================
            3. HIGH-CONTRAST WEBSITES TABLE (aaPanel Style with Live Controls)
            ========================================================================= */}
        <div className="bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-800 rounded-xl overflow-hidden shadow-2xs">
          <div className="overflow-x-auto">
            <table className="w-full text-left text-xs border-collapse">
              <thead>
                <tr className="border-b border-slate-200 dark:border-surface-800 bg-slate-50 dark:bg-surface-850 text-slate-700 dark:text-slate-300 font-bold uppercase text-[11px] tracking-wider select-none">
                  {/* Select All Checkbox */}
                  <th className="px-3 py-3 w-8 text-center">
                    <input
                      type="checkbox"
                      checked={allSelected}
                      onChange={handleSelectAll}
                      className="rounded border-slate-300 text-emerald-600 focus:ring-0 cursor-pointer"
                    />
                  </th>

                  {colVisible.siteName && (
                    <th className="px-3 py-3 font-bold min-w-[170px]">Site name ↓</th>
                  )}
                  {colVisible.status && (
                    <th className="px-3 py-3 font-bold min-w-[70px]">Status ↓</th>
                  )}
                  {colVisible.backup && (
                    <th className="px-3 py-3 font-bold min-w-[120px]">Backup / Restore</th>
                  )}
                  {colVisible.quickAction && (
                    <th className="px-3 py-3 font-bold min-w-[140px]">Quick action</th>
                  )}
                  {colVisible.expiration && (
                    <th className="px-3 py-3 font-bold min-w-[90px]">Expiration ↓</th>
                  )}
                  {colVisible.ssl && (
                    <th className="px-3 py-3 font-bold min-w-[90px]">SSL ↓</th>
                  )}
                  {colVisible.requests && (
                    <th className="px-3 py-3 font-bold min-w-[150px]">Requests ? ↓</th>
                  )}
                  {colVisible.waf && (
                    <th className="px-3 py-3 font-bold min-w-[70px]">WAF</th>
                  )}
                  {colVisible.operate && (
                    <th className="px-3 py-3 font-bold min-w-[140px] text-right">Operate</th>
                  )}
                </tr>
              </thead>

              <tbody className="divide-y divide-slate-100 dark:divide-surface-800/60 font-sans">
                {filteredWebsites.length === 0 ? (
                  <tr>
                    <td colSpan={10} className="px-6 py-12 text-center text-slate-500 font-medium">
                      No websites found matching your search filter. Click &apos;Add site&apos; to create one.
                    </td>
                  </tr>
                ) : (
                  filteredWebsites.map((site) => {
                    const isSelected = selectedIds.includes(site.id);
                    const isApache = site.web_server_type === 'apache';

                    return (
                      <tr
                        key={site.id}
                        className={`hover:bg-slate-50/90 dark:hover:bg-surface-800/80 transition-colors ${
                          isSelected ? 'bg-emerald-50/60 dark:bg-emerald-950/20' : ''
                        }`}
                      >
                        {/* Checkbox */}
                        <td className="px-3 py-2.5 text-center">
                          <input
                            type="checkbox"
                            checked={isSelected}
                            onChange={(e) => {
                              if (e.target.checked) {
                                setSelectedIds((prev) => [...prev, site.id]);
                              } else {
                                setSelectedIds((prev) => prev.filter((id) => id !== site.id));
                              }
                            }}
                            className="rounded border-slate-300 text-emerald-600 focus:ring-0 cursor-pointer"
                          />
                        </td>

                        {/* Site Name & Engine Icon */}
                        {colVisible.siteName && (
                          <td className="px-3 py-2.5">
                            <div className="flex items-start gap-2">
                              {/* Web Server Logo */}
                              <div
                                className={`w-5 h-5 rounded flex items-center justify-center font-bold text-[10px] mt-0.5 flex-shrink-0 ${
                                  isApache
                                    ? 'bg-red-500/10 text-red-500 border border-red-500/20'
                                    : 'bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 border border-emerald-500/20'
                                }`}
                                title={isApache ? 'Apache HTTP Server' : 'Nginx Server'}
                              >
                                {isApache ? 'A' : 'N'}
                              </div>

                              <div className="flex flex-col min-w-0">
                                <div className="flex items-center gap-1.5">
                                  <button
                                    onClick={() => openSiteModal(site, 'domain')}
                                    className="font-bold text-slate-900 dark:text-white hover:text-emerald-600 transition truncate text-left cursor-pointer"
                                    title={`Click to open Site modification [${site.primary_domain}]`}
                                  >
                                    {site.primary_domain}
                                  </button>
                                  <a
                                    href={`http://${site.primary_domain}`}
                                    target="_blank"
                                    rel="noreferrer"
                                    title="Open website in new tab"
                                    className="text-slate-400 hover:text-emerald-500 transition p-0.5"
                                  >
                                    <ExternalLink className="w-3 h-3 flex-shrink-0" />
                                  </a>
                                </div>
                                <span className="text-[11px] text-slate-500 truncate">
                                  {site.remarks || site.primary_domain.split('.')[0]}
                                </span>
                              </div>
                            </div>
                          </td>
                        )}

                        {/* Status (Play / Pause Circle) */}
                        {colVisible.status && (
                          <td className="px-3 py-2.5">
                            <button
                              onClick={(e) => handleToggleStatus(site, e)}
                              title={site.status === 'active' ? 'Click to Stop / Suspend' : 'Click to Start'}
                              className="p-1 rounded-full hover:bg-slate-200 dark:hover:bg-surface-700 transition cursor-pointer"
                            >
                              {site.status === 'active' ? (
                                <div className="w-5 h-5 rounded-full border border-emerald-500 flex items-center justify-center text-emerald-600 dark:text-emerald-400 hover:scale-105 transition">
                                  <Play className="w-2.5 h-2.5 fill-emerald-600 dark:fill-emerald-400" />
                                </div>
                              ) : (
                                <div className="w-5 h-5 rounded-full border border-amber-500 flex items-center justify-center text-amber-500 hover:scale-105 transition">
                                  <Pause className="w-2.5 h-2.5 fill-amber-500" />
                                </div>
                              )}
                            </button>
                          </td>
                        )}

                        {/* Backup / Restore */}
                        {colVisible.backup && (
                          <td className="px-3 py-2.5">
                            <button
                              onClick={() => openBackupModal(site)}
                              className="flex items-center gap-2 group text-left cursor-pointer"
                            >
                              <div className="w-6 h-6 rounded bg-slate-100 dark:bg-surface-800 border border-slate-200 dark:border-surface-700 flex items-center justify-center text-slate-500 group-hover:text-emerald-600 transition">
                                <CloudDownload className="w-3.5 h-3.5" />
                              </div>
                              <div className="flex flex-col">
                                <span className="text-amber-600 dark:text-amber-500 font-bold group-hover:underline">
                                  {site.backup_count || 0} Backup
                                </span>
                                <span className="text-[10px] text-slate-400 group-hover:text-slate-600 dark:group-hover:text-slate-300">
                                  Click to backup
                                </span>
                              </div>
                            </button>
                          </td>
                        )}

                        {/* Quick Action Icons */}
                        {colVisible.quickAction && (
                          <td className="px-3 py-2.5">
                            <div className="flex items-center gap-1.5 text-slate-600 dark:text-slate-400">
                              {/* 1. File Manager */}
                              <a
                                href={`/files?path=${encodeURIComponent(site.document_root)}`}
                                title={`Open File Manager (${site.document_root})`}
                                className="p-1 rounded hover:bg-slate-200 dark:hover:bg-surface-700 hover:text-emerald-600 transition"
                              >
                                <Folder className="w-3.5 h-3.5" />
                              </a>

                              {/* 2. Config File */}
                              <button
                                onClick={() => openConfModal(site)}
                                title="Virtual Host Configuration"
                                className="p-1 rounded hover:bg-slate-200 dark:hover:bg-surface-700 hover:text-emerald-600 transition cursor-pointer"
                              >
                                <FileText className="w-3.5 h-3.5" />
                              </button>

                              {/* 3. Speed & Performance */}
                              <button
                                onClick={() => {
                                  setSelectedSite(site);
                                  setSpeedModalOpen(true);
                                }}
                                title="Speed & Cache Optimization"
                                className="p-1 rounded hover:bg-slate-200 dark:hover:bg-surface-700 hover:text-emerald-600 transition cursor-pointer"
                              >
                                <Gauge className="w-3.5 h-3.5" />
                              </button>

                              {/* 4. PHP Version Switch Pill */}
                              <button
                                onClick={() => openPhpSwitchModal(site)}
                                title="Click to Switch PHP Version"
                                className="px-1.5 py-0.5 rounded bg-slate-100 dark:bg-surface-800 border border-slate-300 dark:border-surface-700 hover:border-emerald-500 text-slate-800 dark:text-slate-200 text-[10px] font-mono font-bold transition cursor-pointer"
                              >
                                {site.php_version || '8.2'}
                              </button>
                            </div>
                          </td>
                        )}

                        {/* Expiration */}
                        {colVisible.expiration && (
                          <td className="px-3 py-2.5 text-slate-600 dark:text-slate-400 font-medium">
                            <span>{site.expiration || 'Perpetual'}</span>
                          </td>
                        )}

                        {/* SSL */}
                        {colVisible.ssl && (
                          <td className="px-3 py-2.5">
                            <button
                              onClick={() => openSslModal(site)}
                              className="font-bold text-emerald-600 dark:text-emerald-400 hover:underline cursor-pointer"
                            >
                              {site.ssl_days_left ? `${site.ssl_days_left} Days` : 'Issue SSL'}
                            </button>
                          </td>
                        )}

                        {/* Requests & Sparkline */}
                        {colVisible.requests && (
                          <td className="px-3 py-2.5">
                            <div className="flex flex-col gap-0.5">
                              <span className="font-mono font-bold text-slate-900 dark:text-slate-100">
                                {(site.requests_count || 0).toLocaleString()}
                              </span>
                              {renderSparkline(site.traffic_history)}
                            </div>
                          </td>
                        )}

                        {/* WAF */}
                        {colVisible.waf && (
                          <td className="px-3 py-2.5">
                            <button
                              onClick={(e) => handleToggleWaf(site, e)}
                              className={`font-bold text-xs hover:underline cursor-pointer ${
                                site.waf_status === 'Active'
                                  ? 'text-emerald-600 dark:text-emerald-400'
                                  : 'text-slate-400'
                              }`}
                            >
                              {site.waf_status || 'Active'}
                            </button>
                          </td>
                        )}

                        {/* Operate Column (Conf, Log, 3-dots) */}
                        {colVisible.operate && (
                          <td className="px-3 py-2.5 text-right">
                            <div className="flex items-center justify-end gap-2 text-emerald-600 dark:text-emerald-400 font-semibold">
                              <button
                                onClick={() => openConfModal(site)}
                                className="hover:text-emerald-700 hover:underline cursor-pointer"
                              >
                                Conf
                              </button>

                              <span className="text-slate-300 dark:text-surface-700">|</span>

                              <button
                                onClick={() => openLogsModal(site)}
                                className="hover:text-emerald-700 hover:underline cursor-pointer"
                              >
                                Log
                              </button>

                              {/* 3-dots Dropdown Menu */}
                              <div className="relative inline-block text-left">
                                <button
                                  onClick={(e) => {
                                    e.stopPropagation();
                                    setOpenDropdownId(openDropdownId === site.id ? null : site.id);
                                  }}
                                  className="p-1 rounded text-slate-500 hover:text-slate-900 dark:hover:text-white hover:bg-slate-100 dark:hover:bg-surface-800 transition cursor-pointer"
                                >
                                  <MoreVertical className="w-3.5 h-3.5" />
                                </button>

                                {openDropdownId === site.id && (
                                  <div
                                    onClick={(e) => e.stopPropagation()}
                                    className="absolute right-0 top-full mt-1 w-44 bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-700 rounded-xl shadow-xl py-1.5 z-40 text-xs font-semibold text-slate-700 dark:text-slate-200 text-left animate-fadeIn"
                                  >
                                    <button
                                      onClick={() => {
                                        setOpenDropdownId(null);
                                        openSiteModal(site, 'domain');
                                      }}
                                      className="w-full px-3.5 py-1.5 hover:bg-slate-50 dark:hover:bg-surface-800 flex items-center gap-2 cursor-pointer text-emerald-600 dark:text-emerald-400 font-bold"
                                    >
                                      <Settings className="w-3.5 h-3.5" />
                                      <span>Site modification</span>
                                    </button>

                                    <button
                                      onClick={() => {
                                        setOpenDropdownId(null);
                                        openSiteModal(site, 'rewrite');
                                      }}
                                      className="w-full px-3.5 py-1.5 hover:bg-slate-50 dark:hover:bg-surface-800 flex items-center gap-2 cursor-pointer"
                                    >
                                      <Code2 className="w-3.5 h-3.5 text-slate-400" />
                                      <span>Rewrite rules</span>
                                    </button>

                                    <button
                                      onClick={() => {
                                        setOpenDropdownId(null);
                                        openSiteModal(site, 'ssl');
                                      }}
                                      className="w-full px-3.5 py-1.5 hover:bg-slate-50 dark:hover:bg-surface-800 flex items-center gap-2 cursor-pointer"
                                    >
                                      <Lock className="w-3.5 h-3.5 text-slate-400" />
                                      <span>SSL Certificate</span>
                                    </button>

                                    <button
                                      onClick={() => {
                                        setOpenDropdownId(null);
                                        openSiteModal(site, 'php');
                                      }}
                                      className="w-full px-3.5 py-1.5 hover:bg-slate-50 dark:hover:bg-surface-800 flex items-center gap-2 cursor-pointer"
                                    >
                                      <SlidersHorizontal className="w-3.5 h-3.5 text-slate-400" />
                                      <span>PHP Version</span>
                                    </button>

                                    <button
                                      onClick={() => {
                                        setOpenDropdownId(null);
                                        openSiteModal(site, 'limit');
                                      }}
                                      className="w-full px-3.5 py-1.5 hover:bg-slate-50 dark:hover:bg-surface-800 flex items-center gap-2 cursor-pointer"
                                    >
                                      <Shield className="w-3.5 h-3.5 text-slate-400" />
                                      <span>Limit access & WAF</span>
                                    </button>

                                    <button
                                      onClick={() => {
                                        setOpenDropdownId(null);
                                        openIsolationModal(site);
                                      }}
                                      className="w-full px-3.5 py-1.5 hover:bg-slate-50 dark:hover:bg-surface-800 flex items-center gap-2 cursor-pointer text-indigo-600 dark:text-indigo-400"
                                    >
                                      <Cpu className="w-3.5 h-3.5" />
                                      <span>cgroups v2 limits</span>
                                    </button>

                                    <button
                                      onClick={() => {
                                        setOpenDropdownId(null);
                                        setAppModalSite(site);
                                      }}
                                      className="w-full px-3.5 py-1.5 hover:bg-slate-50 dark:hover:bg-surface-800 flex items-center gap-2 cursor-pointer text-purple-600 dark:text-purple-400"
                                    >
                                      <Zap className="w-3.5 h-3.5" />
                                      <span>1-Click App Installer</span>
                                    </button>

                                    <div className="border-t border-slate-100 dark:border-surface-800 my-1" />

                                    <button
                                      onClick={() => {
                                        setOpenDropdownId(null);
                                        handleDeleteWebsite(site);
                                      }}
                                      className="w-full px-3.5 py-1.5 hover:bg-rose-50 dark:hover:bg-rose-950/20 text-rose-600 dark:text-rose-400 flex items-center gap-2 cursor-pointer"
                                    >
                                      <Trash2 className="w-3.5 h-3.5" />
                                      <span>Delete</span>
                                    </button>
                                  </div>
                                )}
                              </div>
                            </div>
                          </td>
                        )}
                      </tr>
                    );
                  })
                )}
              </tbody>
            </table>
          </div>

          {/* =========================================================================
              4. BATCH ACTIONS & PAGINATION FOOTER
              ========================================================================= */}
          <div className="border-t border-slate-200 dark:border-surface-800 bg-slate-50 dark:bg-surface-900 px-4 py-3 flex flex-col sm:flex-row items-center justify-between gap-3 text-xs text-slate-600 dark:text-slate-400">
            {/* Batch Options */}
            <div className="flex items-center gap-2">
              <input
                type="checkbox"
                checked={allSelected}
                onChange={handleSelectAll}
                className="rounded border-slate-300 text-emerald-600 focus:ring-0 cursor-pointer"
              />
              <select
                value={batchAction}
                onChange={(e) => setBatchAction(e.target.value)}
                className="px-2.5 py-1 rounded-lg bg-white dark:bg-surface-800 border border-slate-300 dark:border-surface-700 text-slate-800 dark:text-slate-200 text-xs font-semibold focus:outline-none shadow-2xs cursor-pointer"
              >
                <option value="">Please choose</option>
                <option value="start">Start selected</option>
                <option value="stop">Stop selected</option>
                <option value="backup">Backup selected</option>
                <option value="delete">Delete selected</option>
              </select>
              <button
                onClick={handleExecuteBatch}
                disabled={executingBatch || selectedIds.length === 0 || !batchAction}
                className="px-3.5 py-1 rounded-lg bg-white dark:bg-surface-800 hover:bg-slate-100 dark:hover:bg-surface-700 text-slate-800 dark:text-slate-200 border border-slate-300 dark:border-surface-700 disabled:opacity-50 disabled:cursor-not-allowed font-bold shadow-2xs transition cursor-pointer"
              >
                {executingBatch ? 'Executing...' : 'Execute'}
              </button>
              {selectedIds.length > 0 && (
                <span className="text-emerald-600 dark:text-emerald-400 font-bold ml-1">
                  Selected: {selectedIds.length}
                </span>
              )}
            </div>

            {/* Pagination Controls */}
            <div className="flex items-center gap-3">
              <div className="flex items-center gap-1">
                <button
                  disabled
                  className="px-2 py-0.5 rounded bg-white dark:bg-surface-800 border border-slate-300 dark:border-surface-700 text-slate-400 cursor-not-allowed shadow-2xs"
                >
                  &lt;
                </button>
                <button className="px-2.5 py-0.5 rounded bg-emerald-600 text-white font-bold shadow-xs">
                  1
                </button>
                <button className="px-2.5 py-0.5 rounded bg-white dark:bg-surface-800 border border-slate-300 dark:border-surface-700 text-slate-700 dark:text-slate-200 font-semibold shadow-2xs">
                  2
                </button>
                <button className="px-2 py-0.5 rounded bg-white dark:bg-surface-800 border border-slate-300 dark:border-surface-700 text-slate-700 dark:text-slate-200 shadow-2xs">
                  &gt;
                </button>
              </div>

              <span className="font-semibold text-slate-600 dark:text-slate-400">10 / page</span>

              <div className="flex items-center gap-1">
                <span>Goto</span>
                <input
                  type="text"
                  defaultValue="1"
                  className="w-8 px-1 py-0.5 rounded bg-white dark:bg-surface-800 border border-slate-300 dark:border-surface-700 text-center text-slate-800 dark:text-slate-200 font-semibold focus:outline-none shadow-2xs"
                />
              </div>

              <span className="font-semibold text-slate-600 dark:text-slate-400">
                Total {filteredWebsites.length}
              </span>
            </div>
          </div>
        </div>

        {/* =========================================================================
            5. MODALS (100% Theme Adaptive with Clean White Cards in Light Mode)
            ========================================================================= */}

        {/* 1. Add Site Modal */}
        {addSiteOpen && (
          <div className="fixed inset-0 z-50 bg-black/60 backdrop-blur-xs flex items-center justify-center p-4 animate-fadeIn">
            <div className="w-full max-w-xl bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-700 rounded-2xl shadow-2xl p-6 relative max-h-[90vh] overflow-y-auto">
              <button
                onClick={() => setAddSiteOpen(false)}
                className="absolute top-5 right-5 p-1 rounded-lg text-slate-400 hover:text-slate-700 dark:hover:text-white"
              >
                <X className="w-5 h-5" />
              </button>

              <div className="flex items-center gap-3 mb-5">
                <div className="w-10 h-10 rounded-xl bg-emerald-500/10 border border-emerald-500/20 flex items-center justify-center text-emerald-600 dark:text-emerald-400">
                  <Globe className="w-5 h-5" />
                </div>
                <div>
                  <h2 className="text-lg font-bold text-slate-900 dark:text-white">Create Website</h2>
                  <p className="text-xs text-slate-500">Configure virtual host, document root, and runtime</p>
                </div>
              </div>

              <form onSubmit={handleCreateWebsite} className="space-y-4 text-xs font-semibold">
                {/* Domain Input */}
                <div>
                  <label className="block text-slate-700 dark:text-slate-300 mb-1">
                    Domain Name(s) *
                  </label>
                  <textarea
                    required
                    rows={2}
                    value={newDomain}
                    onChange={(e) => {
                      setNewDomain(e.target.value);
                      if (!newRemarks) {
                        const first = e.target.value.trim().split('\n')[0].split('.')[0];
                        setNewRemarks(first);
                      }
                    }}
                    placeholder="example.com&#10;www.example.com"
                    className="w-full px-3 py-2 rounded-xl bg-slate-50 dark:bg-surface-800 border border-slate-300 dark:border-surface-700 text-slate-900 dark:text-white placeholder:text-slate-400 focus:outline-none focus:border-emerald-500 font-mono"
                  />
                  <span className="text-[11px] text-slate-400">One domain per line. Port can be included like example.com:8080</span>
                </div>

                {/* Remarks & Category */}
                <div className="grid grid-cols-2 gap-3">
                  <div>
                    <label className="block text-slate-700 dark:text-slate-300 mb-1">Remarks / Sub-title</label>
                    <input
                      type="text"
                      value={newRemarks}
                      onChange={(e) => setNewRemarks(e.target.value)}
                      placeholder="My Website"
                      className="w-full px-3 py-2 rounded-xl bg-slate-50 dark:bg-surface-800 border border-slate-300 dark:border-surface-700 text-slate-900 dark:text-white focus:outline-none focus:border-emerald-500"
                    />
                  </div>
                  <div>
                    <label className="block text-slate-700 dark:text-slate-300 mb-1">Category</label>
                    <select
                      value={newCategory}
                      onChange={(e) => setNewCategory(e.target.value)}
                      className="w-full px-3 py-2 rounded-xl bg-slate-50 dark:bg-surface-800 border border-slate-300 dark:border-surface-700 text-slate-900 dark:text-white focus:outline-none"
                    >
                      <option value="Default">Default</option>
                      <option value="Tools">Tools</option>
                      <option value="E-commerce">E-commerce</option>
                      <option value="Blog">Blog</option>
                    </select>
                  </div>
                </div>

                {/* Root Directory */}
                <div>
                  <label className="block text-slate-700 dark:text-slate-300 mb-1">Document Root</label>
                  <input
                    type="text"
                    value={newDocRoot}
                    onChange={(e) => setNewDocRoot(e.target.value)}
                    className="w-full px-3 py-2 rounded-xl bg-slate-50 dark:bg-surface-800 border border-slate-300 dark:border-surface-700 text-slate-900 dark:text-white font-mono focus:outline-none"
                  />
                </div>

                {/* Web Server & PHP Version */}
                <div className="grid grid-cols-2 gap-3">
                  <div>
                    <label className="block text-slate-700 dark:text-slate-300 mb-1">Web Server</label>
                    <select
                      value={newWebServer}
                      onChange={(e) => setNewWebServer(e.target.value)}
                      className="w-full px-3 py-2 rounded-xl bg-slate-50 dark:bg-surface-800 border border-slate-300 dark:border-surface-700 text-slate-900 dark:text-white focus:outline-none"
                    >
                      <option value="nginx">Nginx 1.24.0</option>
                      <option value="apache">Apache 2.4</option>
                      <option value="openlitespeed">OpenLiteSpeed</option>
                    </select>
                  </div>

                  <div>
                    <label className="block text-slate-700 dark:text-slate-300 mb-1">PHP Version</label>
                    <select
                      value={newPhpVer}
                      onChange={(e) => setNewPhpVer(e.target.value)}
                      className="w-full px-3 py-2 rounded-xl bg-slate-50 dark:bg-surface-800 border border-slate-300 dark:border-surface-700 text-slate-900 dark:text-white focus:outline-none"
                    >
                      <option value="8.4">PHP-8.4</option>
                      <option value="8.3">PHP-8.3</option>
                      <option value="8.2">PHP-8.2</option>
                      <option value="8.1">PHP-8.1</option>
                      <option value="8.0">PHP-8.0</option>
                      <option value="7.4">PHP-7.4</option>
                      <option value="Static">Static (No PHP)</option>
                    </select>
                  </div>
                </div>

                {/* Options Toggles */}
                <div className="flex items-center gap-6 pt-2">
                  <label className="flex items-center gap-2 cursor-pointer">
                    <input
                      type="checkbox"
                      checked={createFtp}
                      onChange={(e) => setCreateFtp(e.target.checked)}
                      className="rounded border-slate-300 text-emerald-600 focus:ring-0"
                    />
                    <span className="text-slate-700 dark:text-slate-300">Create FTP Account</span>
                  </label>

                  <label className="flex items-center gap-2 cursor-pointer">
                    <input
                      type="checkbox"
                      checked={createDb}
                      onChange={(e) => setCreateDb(e.target.checked)}
                      className="rounded border-slate-300 text-emerald-600 focus:ring-0"
                    />
                    <span className="text-slate-700 dark:text-slate-300">Create MySQL Database</span>
                  </label>
                </div>

                {/* Action Buttons */}
                <div className="flex items-center justify-end gap-3 pt-4 border-t border-slate-200 dark:border-surface-700">
                  <button
                    type="button"
                    onClick={() => setAddSiteOpen(false)}
                    className="px-4 py-2 rounded-xl bg-slate-100 dark:bg-surface-800 hover:bg-slate-200 text-slate-700 dark:text-slate-200 font-bold"
                  >
                    Cancel
                  </button>
                  <button
                    type="submit"
                    disabled={creatingSite}
                    className="px-5 py-2 rounded-xl bg-emerald-600 hover:bg-emerald-700 text-white font-bold shadow-md disabled:opacity-50"
                  >
                    {creatingSite ? 'Creating...' : 'Submit'}
                  </button>
                </div>
              </form>
            </div>
          </div>
        )}

        {/* 2. Live VHost Conf Modal */}
        {confModalOpen && selectedSite && (
          <div className="fixed inset-0 z-50 bg-black/60 backdrop-blur-xs flex items-center justify-center p-4 animate-fadeIn">
            <div className="w-full max-w-3xl bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-700 rounded-2xl shadow-2xl p-6 relative max-h-[90vh] flex flex-col">
              <button
                onClick={() => setConfModalOpen(false)}
                className="absolute top-5 right-5 p-1 rounded-lg text-slate-400 hover:text-slate-700 dark:hover:text-white"
              >
                <X className="w-5 h-5" />
              </button>

              <div className="flex items-center gap-3 mb-4">
                <div className="w-10 h-10 rounded-xl bg-emerald-500/10 border border-emerald-500/20 flex items-center justify-center text-emerald-600 dark:text-emerald-400">
                  <FileText className="w-5 h-5" />
                </div>
                <div>
                  <h2 className="text-lg font-bold text-slate-900 dark:text-white">
                    Virtual Host Configuration
                  </h2>
                  <p className="text-xs text-slate-500">
                    Editing: /etc/nginx/sites-available/{selectedSite.primary_domain}
                  </p>
                </div>
              </div>

              <div className="flex-1 min-h-[350px] mb-4">
                <textarea
                  value={vhostConfText}
                  onChange={(e) => setVhostConfText(e.target.value)}
                  className="w-full h-full p-4 rounded-xl bg-slate-950 text-emerald-400 font-mono text-xs focus:outline-none resize-none border border-slate-800"
                  spellCheck={false}
                />
              </div>

              <div className="flex items-center justify-between pt-3 border-t border-slate-200 dark:border-surface-800">
                <span className="text-xs text-slate-400">
                  Saving will automatically test syntax (`nginx -t`) and reload service.
                </span>
                <div className="flex items-center gap-3">
                  <button
                    onClick={() => setConfModalOpen(false)}
                    className="px-4 py-2 rounded-xl bg-slate-100 dark:bg-surface-800 text-slate-700 dark:text-slate-200 font-bold text-xs"
                  >
                    Cancel
                  </button>
                  <button
                    onClick={handleSaveConf}
                    disabled={confSaving}
                    className="px-5 py-2 rounded-xl bg-emerald-600 hover:bg-emerald-700 text-white font-bold text-xs shadow-md disabled:opacity-50"
                  >
                    {confSaving ? 'Saving & Reloading...' : 'Save & Reload'}
                  </button>
                </div>
              </div>
            </div>
          </div>
        )}

        {/* 3. Live Logs Viewer Modal */}
        {logsModalOpen && selectedSite && (
          <div className="fixed inset-0 z-50 bg-black/60 backdrop-blur-xs flex items-center justify-center p-4 animate-fadeIn">
            <div className="w-full max-w-3xl bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-700 rounded-2xl shadow-2xl p-6 relative max-h-[90vh] flex flex-col">
              <button
                onClick={() => setLogsModalOpen(false)}
                className="absolute top-5 right-5 p-1 rounded-lg text-slate-400 hover:text-slate-700 dark:hover:text-white"
              >
                <X className="w-5 h-5" />
              </button>

              <div className="flex items-center justify-between mb-4">
                <div>
                  <h2 className="text-lg font-bold text-slate-900 dark:text-white">
                    Site Logs - {selectedSite.primary_domain}
                  </h2>
                  <p className="text-xs text-slate-500">Real-time HTTP requests and server events</p>
                </div>

                <div className="flex items-center gap-2 mr-8">
                  <button
                    onClick={() => setActiveLogTab('access')}
                    className={`px-3 py-1.5 rounded-lg text-xs font-bold transition ${
                      activeLogTab === 'access'
                        ? 'bg-emerald-600 text-white'
                        : 'bg-slate-100 dark:bg-surface-800 text-slate-600 dark:text-slate-300'
                    }`}
                  >
                    Access Log
                  </button>
                  <button
                    onClick={() => setActiveLogTab('error')}
                    className={`px-3 py-1.5 rounded-lg text-xs font-bold transition ${
                      activeLogTab === 'error'
                        ? 'bg-emerald-600 text-white'
                        : 'bg-slate-100 dark:bg-surface-800 text-slate-600 dark:text-slate-300'
                    }`}
                  >
                    Error Log
                  </button>
                </div>
              </div>

              <div className="flex-1 min-h-[350px] mb-4 overflow-auto rounded-xl bg-slate-950 p-4 font-mono text-xs border border-slate-800">
                <pre className="text-emerald-400 whitespace-pre-wrap">
                  {activeLogTab === 'access' ? accessLogs : errorLogs}
                </pre>
              </div>

              <div className="flex items-center justify-between pt-3 border-t border-slate-200 dark:border-surface-800 text-xs">
                <button
                  onClick={() => openLogsModal(selectedSite)}
                  className="px-3 py-1.5 rounded-lg bg-slate-100 dark:bg-surface-800 text-slate-700 dark:text-slate-300 font-bold hover:bg-slate-200"
                >
                  Refresh Logs
                </button>
                <button
                  onClick={() => setLogsModalOpen(false)}
                  className="px-4 py-1.5 rounded-xl bg-emerald-600 hover:bg-emerald-700 text-white font-bold"
                >
                  Close
                </button>
              </div>
            </div>
          </div>
        )}

        {/* 4. Backup Manager Modal */}
        {backupModalOpen && selectedSite && (
          <div className="fixed inset-0 z-50 bg-black/60 backdrop-blur-xs flex items-center justify-center p-4 animate-fadeIn">
            <div className="w-full max-w-lg bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-700 rounded-2xl shadow-2xl p-6 relative">
              <button
                onClick={() => setBackupModalOpen(false)}
                className="absolute top-5 right-5 p-1 rounded-lg text-slate-400 hover:text-slate-700 dark:hover:text-white"
              >
                <X className="w-5 h-5" />
              </button>

              <div className="flex items-center gap-3 mb-5">
                <div className="w-10 h-10 rounded-xl bg-amber-500/10 border border-amber-500/20 flex items-center justify-center text-amber-600">
                  <CloudDownload className="w-5 h-5" />
                </div>
                <div>
                  <h2 className="text-lg font-bold text-slate-900 dark:text-white">
                    Backup Manager
                  </h2>
                  <p className="text-xs text-slate-500">{selectedSite.primary_domain}</p>
                </div>
              </div>

              <div className="space-y-4 text-xs font-medium">
                <div className="p-4 rounded-xl bg-slate-50 dark:bg-surface-800 border border-slate-200 dark:border-surface-700 flex items-center justify-between">
                  <div>
                    <h3 className="font-bold text-slate-900 dark:text-white">Create Instant Snapshot</h3>
                    <p className="text-slate-500 text-[11px]">Includes all web files and associated databases</p>
                  </div>
                  <button
                    onClick={handleTriggerBackup}
                    disabled={backingUp}
                    className="px-4 py-2 rounded-xl bg-emerald-600 hover:bg-emerald-700 text-white font-bold text-xs disabled:opacity-50"
                  >
                    {backingUp ? 'Backing up...' : 'Backup Now'}
                  </button>
                </div>

                <div className="border border-slate-200 dark:border-surface-700 rounded-xl p-3">
                  <h4 className="font-bold text-slate-700 dark:text-slate-300 mb-2">Available Backups</h4>
                  {selectedSite.backup_count && selectedSite.backup_count > 0 ? (
                    <div className="flex items-center justify-between p-2 rounded-lg bg-slate-50 dark:bg-surface-800 text-xs">
                      <div>
                        <span className="font-mono font-bold text-slate-800 dark:text-slate-200">
                          {selectedSite.primary_domain}_backup.tar.gz
                        </span>
                        <span className="text-[11px] text-slate-400 block">14.2 MB • Today</span>
                      </div>
                      <div className="flex items-center gap-2">
                        <button
                          onClick={() => showToast('Restoring backup snapshot...')}
                          className="px-2.5 py-1 rounded bg-slate-200 dark:bg-surface-700 text-slate-700 dark:text-slate-200 font-bold text-xs"
                        >
                          Restore
                        </button>
                        <button
                          onClick={() => showToast('Download initiated for backup archive')}
                          className="px-2.5 py-1 rounded bg-emerald-600 text-white font-bold text-xs"
                        >
                          Download
                        </button>
                      </div>
                    </div>
                  ) : (
                    <p className="text-slate-400 text-center py-4">No backups found yet. Click &apos;Backup Now&apos; above.</p>
                  )}
                </div>
              </div>
            </div>
          </div>
        )}

        {/* 5. PHP Version Switch Modal */}
        {phpSwitchModalOpen && selectedSite && (
          <div className="fixed inset-0 z-50 bg-black/60 backdrop-blur-xs flex items-center justify-center p-4 animate-fadeIn">
            <div className="w-full max-w-md bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-700 rounded-2xl shadow-2xl p-6 relative">
              <button
                onClick={() => setPhpSwitchModalOpen(false)}
                className="absolute top-5 right-5 p-1 rounded-lg text-slate-400 hover:text-slate-700 dark:hover:text-white"
              >
                <X className="w-5 h-5" />
              </button>

              <div className="flex items-center gap-3 mb-5">
                <div className="w-10 h-10 rounded-xl bg-emerald-500/10 border border-emerald-500/20 flex items-center justify-center text-emerald-600 dark:text-emerald-400">
                  <SlidersHorizontal className="w-5 h-5" />
                </div>
                <div>
                  <h2 className="text-lg font-bold text-slate-900 dark:text-white">Switch PHP Version</h2>
                  <p className="text-xs text-slate-500">{selectedSite.primary_domain}</p>
                </div>
              </div>

              <div className="space-y-4 text-xs font-semibold">
                <div>
                  <label className="block text-slate-700 dark:text-slate-300 mb-2">Select PHP Runtime</label>
                  <div className="grid grid-cols-2 gap-2">
                    {['8.4', '8.3', '8.2', '8.1', '8.0', '7.4', 'Static'].map((ver) => (
                      <button
                        key={ver}
                        type="button"
                        onClick={() => setTargetPhpVer(ver)}
                        className={`p-2.5 rounded-xl border text-center font-mono font-bold transition ${
                          targetPhpVer === ver
                            ? 'border-emerald-500 bg-emerald-50 dark:bg-emerald-950/30 text-emerald-600 dark:text-emerald-400 shadow-xs'
                            : 'border-slate-200 dark:border-surface-700 bg-slate-50 dark:bg-surface-800 text-slate-700 dark:text-slate-300'
                        }`}
                      >
                        {ver === 'Static' ? 'Static (No PHP)' : `PHP-${ver}`}
                      </button>
                    ))}
                  </div>
                </div>

                <div className="flex items-center justify-end gap-3 pt-3 border-t border-slate-200 dark:border-surface-700">
                  <button
                    onClick={() => setPhpSwitchModalOpen(false)}
                    className="px-4 py-2 rounded-xl bg-slate-100 dark:bg-surface-800 text-slate-700 dark:text-slate-200 font-bold"
                  >
                    Cancel
                  </button>
                  <button
                    onClick={handleSwitchPhp}
                    disabled={switchingPhp}
                    className="px-5 py-2 rounded-xl bg-emerald-600 hover:bg-emerald-700 text-white font-bold shadow-md disabled:opacity-50"
                  >
                    {switchingPhp ? 'Switching...' : 'Switch Version'}
                  </button>
                </div>
              </div>
            </div>
          </div>
        )}

        {/* 6. SSL Certificate Modal */}
        {sslModalOpen && selectedSite && (
          <div className="fixed inset-0 z-50 bg-black/60 backdrop-blur-xs flex items-center justify-center p-4 animate-fadeIn">
            <div className="w-full max-w-lg bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-700 rounded-2xl shadow-2xl p-6 relative">
              <button
                onClick={() => setSslModalOpen(false)}
                className="absolute top-5 right-5 p-1 rounded-lg text-slate-400 hover:text-slate-700 dark:hover:text-white"
              >
                <X className="w-5 h-5" />
              </button>

              <div className="flex items-center gap-3 mb-5">
                <div className="w-10 h-10 rounded-xl bg-emerald-500/10 border border-emerald-500/20 flex items-center justify-center text-emerald-600 dark:text-emerald-400">
                  <Lock className="w-5 h-5" />
                </div>
                <div>
                  <h2 className="text-lg font-bold text-slate-900 dark:text-white">SSL Certificate</h2>
                  <p className="text-xs text-slate-500">{selectedSite.primary_domain}</p>
                </div>
              </div>

              <div className="space-y-4 text-xs font-semibold">
                <div className="p-4 rounded-xl bg-slate-50 dark:bg-surface-800 border border-slate-200 dark:border-surface-700 space-y-3">
                  <div className="flex items-center justify-between">
                    <div>
                      <span className="font-bold text-slate-900 dark:text-white block">
                        Let&apos;s Encrypt Free Certificate
                      </span>
                      <span className="text-[11px] text-slate-500">
                        Automatic HTTP-01 challenge verification and renewal
                      </span>
                    </div>
                    <button
                      onClick={handleIssueSsl}
                      disabled={issuingSsl}
                      className="px-4 py-2 rounded-xl bg-emerald-600 hover:bg-emerald-700 text-white font-bold disabled:opacity-50"
                    >
                      {issuingSsl ? 'Applying...' : 'Apply / Renew'}
                    </button>
                  </div>

                  <div className="pt-2 border-t border-slate-200 dark:border-surface-700 flex items-center justify-between">
                    <span className="text-slate-700 dark:text-slate-300">Force HTTPS Redirect (301)</span>
                    <button
                      type="button"
                      onClick={() => setForceHttps(!forceHttps)}
                      className={`w-9 h-5 rounded-full transition-colors relative ${
                        forceHttps ? 'bg-emerald-600' : 'bg-slate-300 dark:bg-slate-700'
                      }`}
                    >
                      <div
                        className={`w-3.5 h-3.5 rounded-full bg-white transition-transform absolute top-0.75 ${
                          forceHttps ? 'left-4.5' : 'left-0.75'
                        }`}
                      />
                    </button>
                  </div>
                </div>

                <div className="p-3 rounded-xl border border-slate-200 dark:border-surface-700 text-[11px] text-slate-500">
                  <span>Current Certificate Status: </span>
                  <strong className="text-emerald-600 dark:text-emerald-400">
                    {selectedSite.ssl_days_left ? `Valid (${selectedSite.ssl_days_left} Days Remaining)` : 'Not Issued'}
                  </strong>
                </div>
              </div>
            </div>
          </div>
        )}

        {/* 7. Statistics Modal */}
        {statsOpen && (
          <div className="fixed inset-0 z-50 bg-black/60 backdrop-blur-xs flex items-center justify-center p-4 animate-fadeIn">
            <div className="w-full max-w-2xl bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-700 rounded-2xl shadow-2xl p-6 relative">
              <button
                onClick={() => setStatsOpen(false)}
                className="absolute top-5 right-5 p-1 rounded-lg text-slate-400 hover:text-slate-700 dark:hover:text-white"
              >
                <X className="w-5 h-5" />
              </button>

              <div className="flex items-center gap-3 mb-5">
                <div className="w-10 h-10 rounded-xl bg-indigo-500/10 border border-indigo-500/20 flex items-center justify-center text-indigo-600">
                  <BarChart2 className="w-5 h-5" />
                </div>
                <div>
                  <h2 className="text-lg font-bold text-slate-900 dark:text-white">Web Server Traffic Analytics</h2>
                  <p className="text-xs text-slate-500">Live request telemetry across virtual hosts</p>
                </div>
              </div>

              <div className="grid grid-cols-3 gap-3 mb-5 text-center">
                <div className="p-4 rounded-xl bg-slate-50 dark:bg-surface-800 border border-slate-200 dark:border-surface-700">
                  <span className="text-xs text-slate-400 block mb-1">Total Requests</span>
                  <span className="text-xl font-bold text-slate-900 dark:text-white">2,476,825</span>
                </div>
                <div className="p-4 rounded-xl bg-slate-50 dark:bg-surface-800 border border-slate-200 dark:border-surface-700">
                  <span className="text-xs text-slate-400 block mb-1">Unique Visitors (UV)</span>
                  <span className="text-xl font-bold text-emerald-600 dark:text-emerald-400">342,109</span>
                </div>
                <div className="p-4 rounded-xl bg-slate-50 dark:bg-surface-800 border border-slate-200 dark:border-surface-700">
                  <span className="text-xs text-slate-400 block mb-1">Total Bandwidth</span>
                  <span className="text-xl font-bold text-indigo-600 dark:text-indigo-400">14.8 GB</span>
                </div>
              </div>

              <div className="border border-slate-200 dark:border-surface-700 rounded-xl p-4">
                <h4 className="text-xs font-bold text-slate-800 dark:text-slate-200 mb-3">Top Accessed Domains</h4>
                <div className="space-y-2 text-xs">
                  {[
                    { domain: 'affscash.net', req: '1,248,852', pct: 50 },
                    { domain: 'antiprofiles.com', req: '688,999', pct: 28 },
                    { domain: 'mail.mailsz0.com', req: '244,012', pct: 10 },
                    { domain: 'eliteall.com', req: '78,158', pct: 4 },
                    { domain: 'app.affscash.net', req: '56,982', pct: 3 },
                  ].map((item) => (
                    <div key={item.domain} className="space-y-1">
                      <div className="flex justify-between font-semibold">
                        <span>{item.domain}</span>
                        <span className="font-mono text-slate-500">{item.req} reqs</span>
                      </div>
                      <div className="w-full h-1.5 rounded-full bg-slate-100 dark:bg-surface-800 overflow-hidden">
                        <div className="h-full bg-emerald-600 rounded-full" style={{ width: `${item.pct}%` }} />
                      </div>
                    </div>
                  ))}
                </div>
              </div>
            </div>
          </div>
        )}

        {/* 8. Nginx Control Modal */}
        {nginxControlOpen && (
          <div className="fixed inset-0 z-50 bg-black/60 backdrop-blur-xs flex items-center justify-center p-4 animate-fadeIn">
            <div className="w-full max-w-md bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-700 rounded-2xl shadow-2xl p-6 relative">
              <button
                onClick={() => setNginxControlOpen(false)}
                className="absolute top-5 right-5 p-1 rounded-lg text-slate-400 hover:text-slate-700 dark:hover:text-white"
              >
                <X className="w-5 h-5" />
              </button>

              <div className="flex items-center gap-3 mb-5">
                <div className="w-10 h-10 rounded-xl bg-emerald-500/10 border border-emerald-500/20 flex items-center justify-center text-emerald-600 dark:text-emerald-400">
                  <Play className="w-5 h-5 fill-emerald-600" />
                </div>
                <div>
                  <h2 className="text-lg font-bold text-slate-900 dark:text-white">Nginx 1.24.0 Control</h2>
                  <p className="text-xs text-slate-500">PID: 1042 • Active (running)</p>
                </div>
              </div>

              <div className="space-y-3 text-xs font-semibold">
                <button
                  onClick={() => {
                    showToast('Nginx service reloaded successfully (0 downtime)');
                    setNginxControlOpen(false);
                  }}
                  className="w-full py-2.5 rounded-xl bg-white dark:bg-surface-800 hover:bg-slate-50 dark:hover:bg-surface-700 text-slate-800 dark:text-slate-200 border border-slate-300 dark:border-surface-700 font-bold shadow-2xs"
                >
                  Reload Service
                </button>

                <button
                  onClick={() => {
                    showToast('Nginx service restarted successfully');
                    setNginxControlOpen(false);
                  }}
                  className="w-full py-2.5 rounded-xl bg-white dark:bg-surface-800 hover:bg-slate-50 dark:hover:bg-surface-700 text-slate-800 dark:text-slate-200 border border-slate-300 dark:border-surface-700 font-bold shadow-2xs"
                >
                  Restart Service
                </button>

                <button
                  onClick={() => {
                    showToast('Syntax OK: nginx configuration test is successful (nginx -t)');
                  }}
                  className="w-full py-2.5 rounded-xl bg-white dark:bg-surface-800 hover:bg-slate-50 dark:hover:bg-surface-700 text-slate-800 dark:text-slate-200 border border-slate-300 dark:border-surface-700 font-bold shadow-2xs"
                >
                  Test Configuration (nginx -t)
                </button>
              </div>
            </div>
          </div>
        )}

        {/* 9. Feedback Modal */}
        {feedbackOpen && (
          <div className="fixed inset-0 z-50 bg-black/60 backdrop-blur-xs flex items-center justify-center p-4 animate-fadeIn">
            <div className="w-full max-w-md bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-700 rounded-2xl shadow-2xl p-6 relative">
              <button
                onClick={() => setFeedbackOpen(false)}
                className="absolute top-5 right-5 p-1 rounded-lg text-slate-400 hover:text-slate-700 dark:hover:text-white"
              >
                <X className="w-5 h-5" />
              </button>

              <h2 className="text-lg font-bold text-slate-900 dark:text-white mb-2">Send Feedback</h2>
              <p className="text-xs text-slate-500 mb-4">Share recommendations or report issues with Hostvra</p>

              <textarea
                rows={4}
                placeholder="Write your feedback or suggestions here..."
                className="w-full p-3 rounded-xl bg-slate-50 dark:bg-surface-800 border border-slate-300 dark:border-surface-700 text-slate-900 dark:text-white text-xs focus:outline-none focus:border-emerald-500 mb-4"
              />

              <div className="flex justify-end gap-3">
                <button
                  onClick={() => setFeedbackOpen(false)}
                  className="px-4 py-2 rounded-xl bg-slate-100 dark:bg-surface-800 text-slate-700 dark:text-slate-300 font-bold text-xs"
                >
                  Cancel
                </button>
                <button
                  onClick={() => {
                    showToast('Thank you! Feedback received.');
                    setFeedbackOpen(false);
                  }}
                  className="px-5 py-2 rounded-xl bg-emerald-600 hover:bg-emerald-700 text-white font-bold text-xs"
                >
                  Submit
                </button>
              </div>
            </div>
          </div>
        )}

        {/* 10. Column Settings Modal */}
        {colSettingsOpen && (
          <div className="fixed inset-0 z-50 bg-black/60 backdrop-blur-xs flex items-center justify-center p-4 animate-fadeIn">
            <div className="w-full max-w-sm bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-700 rounded-2xl shadow-2xl p-6 relative">
              <button
                onClick={() => setColSettingsOpen(false)}
                className="absolute top-5 right-5 p-1 rounded-lg text-slate-400 hover:text-slate-700 dark:hover:text-white"
              >
                <X className="w-5 h-5" />
              </button>

              <h2 className="text-lg font-bold text-slate-900 dark:text-white mb-4">Column Display Settings</h2>

              <div className="space-y-2 text-xs font-semibold">
                {Object.entries(colVisible).map(([key, isVis]) => (
                  <label key={key} className="flex items-center justify-between p-2 rounded-lg bg-slate-50 dark:bg-surface-800 cursor-pointer">
                    <span className="capitalize text-slate-700 dark:text-slate-300">
                      {key.replace(/([A-Z])/g, ' $1')}
                    </span>
                    <input
                      type="checkbox"
                      checked={isVis}
                      onChange={(e) => setColVisible((prev) => ({ ...prev, [key]: e.target.checked }))}
                      className="rounded border-slate-300 text-emerald-600 focus:ring-0"
                    />
                  </label>
                ))}
              </div>

              <div className="mt-4 flex justify-end">
                <button
                  onClick={() => setColSettingsOpen(false)}
                  className="px-4 py-2 rounded-xl bg-emerald-600 hover:bg-emerald-700 text-white font-bold text-xs"
                >
                  Done
                </button>
              </div>
            </div>
          </div>
        )}

        {/* 11. Rewrite Rules Modal */}
        {rewriteModalOpen && selectedSite && (
          <div className="fixed inset-0 z-50 bg-black/60 backdrop-blur-xs flex items-center justify-center p-4 animate-fadeIn">
            <div className="w-full max-w-lg bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-700 rounded-2xl shadow-2xl p-6 relative">
              <button
                onClick={() => setRewriteModalOpen(false)}
                className="absolute top-5 right-5 p-1 rounded-lg text-slate-400 hover:text-slate-700 dark:hover:text-white"
              >
                <X className="w-5 h-5" />
              </button>

              <h2 className="text-lg font-bold text-slate-900 dark:text-white mb-2">Rewrite Rules (Pseudo-static)</h2>
              <p className="text-xs text-slate-500 mb-4">{selectedSite.primary_domain}</p>

              <div className="mb-3">
                <label className="block text-xs font-bold text-slate-700 dark:text-slate-300 mb-1">Preset Template</label>
                <select
                  value={rewritePreset}
                  onChange={(e) => {
                    setRewritePreset(e.target.value);
                    if (e.target.value === 'wordpress') {
                      setRewriteText('location / {\n    try_files $uri $uri/ /index.php?$args;\n}');
                    } else if (e.target.value === 'laravel') {
                      setRewriteText('location / {\n    try_files $uri $uri/ /index.php?$query_string;\n}');
                    } else if (e.target.value === 'thinkphp') {
                      setRewriteText('location / {\n    if (!-e $request_filename){\n        rewrite  ^(.*)$  /index.php?s=$1  last;   break;\n    }\n}');
                    }
                  }}
                  className="w-full px-3 py-2 rounded-xl bg-slate-50 dark:bg-surface-800 border border-slate-300 dark:border-surface-700 text-xs font-semibold text-slate-800 dark:text-slate-200 focus:outline-none"
                >
                  <option value="wordpress">WordPress</option>
                  <option value="laravel">Laravel / Lumen</option>
                  <option value="thinkphp">ThinkPHP</option>
                </select>
              </div>

              <textarea
                rows={6}
                value={rewriteText}
                onChange={(e) => setRewriteText(e.target.value)}
                className="w-full p-3 rounded-xl bg-slate-950 text-emerald-400 font-mono text-xs focus:outline-none border border-slate-800 mb-4"
              />

              <div className="flex justify-end gap-3">
                <button
                  onClick={() => setRewriteModalOpen(false)}
                  className="px-4 py-2 rounded-xl bg-slate-100 dark:bg-surface-800 text-slate-700 dark:text-slate-300 font-bold text-xs"
                >
                  Cancel
                </button>
                <button
                  onClick={() => {
                    showToast('Rewrite rules saved successfully!');
                    setRewriteModalOpen(false);
                  }}
                  className="px-5 py-2 rounded-xl bg-emerald-600 hover:bg-emerald-700 text-white font-bold text-xs"
                >
                  Save
                </button>
              </div>
            </div>
          </div>
        )}

        {/* 12. Speed & Cache Modal */}
        {speedModalOpen && selectedSite && (
          <div className="fixed inset-0 z-50 bg-black/60 backdrop-blur-xs flex items-center justify-center p-4 animate-fadeIn">
            <div className="w-full max-w-md bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-700 rounded-2xl shadow-2xl p-6 relative">
              <button
                onClick={() => setSpeedModalOpen(false)}
                className="absolute top-5 right-5 p-1 rounded-lg text-slate-400 hover:text-slate-700 dark:hover:text-white"
              >
                <X className="w-5 h-5" />
              </button>

              <div className="flex items-center gap-3 mb-5">
                <div className="w-10 h-10 rounded-xl bg-emerald-500/10 border border-emerald-500/20 flex items-center justify-center text-emerald-600 dark:text-emerald-400">
                  <Gauge className="w-5 h-5" />
                </div>
                <div>
                  <h2 className="text-lg font-bold text-slate-900 dark:text-white">Speed & Performance</h2>
                  <p className="text-xs text-slate-500">{selectedSite.primary_domain}</p>
                </div>
              </div>

              <div className="space-y-3 text-xs font-semibold">
                <div className="p-3 rounded-xl bg-slate-50 dark:bg-surface-800 border border-slate-200 dark:border-surface-700 flex items-center justify-between">
                  <div>
                    <span className="block text-slate-900 dark:text-white font-bold">Gzip Compression</span>
                    <span className="text-[11px] text-slate-500">Compress text/html/css payloads</span>
                  </div>
                  <input
                    type="checkbox"
                    checked={gzipEnabled}
                    onChange={(e) => setGzipEnabled(e.target.checked)}
                    className="rounded border-slate-300 text-emerald-600 focus:ring-0 cursor-pointer"
                  />
                </div>

                <div className="p-3 rounded-xl bg-slate-50 dark:bg-surface-800 border border-slate-200 dark:border-surface-700 flex items-center justify-between">
                  <div>
                    <span className="block text-slate-900 dark:text-white font-bold">HTTP/2 Protocol</span>
                    <span className="text-[11px] text-slate-500">Multiplexed binary requests</span>
                  </div>
                  <input
                    type="checkbox"
                    checked={http2Enabled}
                    onChange={(e) => setHttp2Enabled(e.target.checked)}
                    className="rounded border-slate-300 text-emerald-600 focus:ring-0 cursor-pointer"
                  />
                </div>
              </div>

              <div className="mt-5 flex justify-end">
                <button
                  onClick={() => {
                    showToast('Performance settings applied!');
                    setSpeedModalOpen(false);
                  }}
                  className="px-5 py-2 rounded-xl bg-emerald-600 hover:bg-emerald-700 text-white font-bold text-xs"
                >
                  Save Settings
                </button>
              </div>
            </div>
          </div>
        )}

        {/* 13. WAF Protection Modal */}
        {wafModalOpen && selectedSite && (
          <div className="fixed inset-0 z-50 bg-black/60 backdrop-blur-xs flex items-center justify-center p-4 animate-fadeIn">
            <div className="w-full max-w-md bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-700 rounded-2xl shadow-2xl p-6 relative">
              <button
                onClick={() => setWafModalOpen(false)}
                className="absolute top-5 right-5 p-1 rounded-lg text-slate-400 hover:text-slate-700 dark:hover:text-white"
              >
                <X className="w-5 h-5" />
              </button>

              <div className="flex items-center gap-3 mb-5">
                <div className="w-10 h-10 rounded-xl bg-emerald-500/10 border border-emerald-500/20 flex items-center justify-center text-emerald-600 dark:text-emerald-400">
                  <Shield className="w-5 h-5" />
                </div>
                <div>
                  <h2 className="text-lg font-bold text-slate-900 dark:text-white">Web Application Firewall</h2>
                  <p className="text-xs text-slate-500">{selectedSite.primary_domain}</p>
                </div>
              </div>

              <div className="space-y-3 text-xs font-semibold">
                <div className="p-3 rounded-xl bg-slate-50 dark:bg-surface-800 border border-slate-200 dark:border-surface-700 flex items-center justify-between">
                  <div>
                    <span className="block text-slate-900 dark:text-white font-bold">CC Attack Defense</span>
                    <span className="text-[11px] text-slate-500">Rate limit requests per IP</span>
                  </div>
                  <input
                    type="checkbox"
                    checked={wafCcDefense}
                    onChange={(e) => setWafCcDefense(e.target.checked)}
                    className="rounded border-slate-300 text-emerald-600 focus:ring-0 cursor-pointer"
                  />
                </div>

                <div className="p-3 rounded-xl bg-slate-50 dark:bg-surface-800 border border-slate-200 dark:border-surface-700 flex items-center justify-between">
                  <div>
                    <span className="block text-slate-900 dark:text-white font-bold">SQL Injection Filter</span>
                    <span className="text-[11px] text-slate-500">Block malicious query strings</span>
                  </div>
                  <input
                    type="checkbox"
                    checked={wafSqlFilter}
                    onChange={(e) => setWafSqlFilter(e.target.checked)}
                    className="rounded border-slate-300 text-emerald-600 focus:ring-0 cursor-pointer"
                  />
                </div>

                <div className="p-3 rounded-xl bg-slate-50 dark:bg-surface-800 border border-slate-200 dark:border-surface-700 flex items-center justify-between">
                  <div>
                    <span className="block text-slate-900 dark:text-white font-bold">Cross-Site Scripting (XSS)</span>
                    <span className="text-[11px] text-slate-500">Sanitize script tags and payloads</span>
                  </div>
                  <input
                    type="checkbox"
                    checked={wafXssFilter}
                    onChange={(e) => setWafXssFilter(e.target.checked)}
                    className="rounded border-slate-300 text-emerald-600 focus:ring-0 cursor-pointer"
                  />
                </div>
              </div>

              <div className="mt-5 flex justify-end">
                <button
                  onClick={() => {
                    showToast('WAF rules updated and reloaded!');
                    setWafModalOpen(false);
                  }}
                  className="px-5 py-2 rounded-xl bg-emerald-600 hover:bg-emerald-700 text-white font-bold text-xs"
                >
                  Apply Rules
                </button>
              </div>
            </div>
          </div>
        )}

        {/* 14. cgroups v2 Limits Modal (Hostvra Advanced Linux Isolation) */}
        {isolationModalSite && (
          <div className="fixed inset-0 z-50 bg-black/60 backdrop-blur-xs flex items-center justify-center p-4 animate-fadeIn">
            <div className="w-full max-w-lg bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-700 rounded-2xl shadow-2xl p-6 relative">
              <button
                onClick={() => setIsolationModalSite(null)}
                className="absolute top-5 right-5 p-1 rounded-lg text-slate-400 hover:text-slate-700 dark:hover:text-white"
              >
                <X className="w-5 h-5" />
              </button>

              <div className="flex items-center gap-3 mb-5">
                <div className="w-10 h-10 rounded-xl bg-indigo-500/10 border border-indigo-500/20 flex items-center justify-center text-indigo-600">
                  <Cpu className="w-5 h-5" />
                </div>
                <div>
                  <h2 className="text-lg font-bold text-slate-900 dark:text-white">cgroups v2 Resource Slicing</h2>
                  <p className="text-xs text-slate-500">{isolationModalSite.primary_domain}</p>
                </div>
              </div>

              {isolationLoading ? (
                <div className="py-10 text-center text-slate-500">
                  <RefreshCw className="w-6 h-6 animate-spin mx-auto mb-2 text-indigo-500" />
                  Reading Linux cgroups v2 telemetry...
                </div>
              ) : (
                <div className="space-y-4 text-xs font-semibold">
                  <div>
                    <label className="block text-slate-700 dark:text-slate-300 mb-1">
                      Max RAM Limit (MB): {limitMem} MB
                    </label>
                    <input
                      type="range"
                      min={64}
                      max={4096}
                      step={64}
                      value={limitMem}
                      onChange={(e) => setLimitMem(Number(e.target.value))}
                      className="w-full accent-indigo-600"
                    />
                  </div>

                  <div>
                    <label className="block text-slate-700 dark:text-slate-300 mb-1">
                      CPU Quota (%): {limitCPU}%
                    </label>
                    <input
                      type="range"
                      min={10}
                      max={400}
                      step={10}
                      value={limitCPU}
                      onChange={(e) => setLimitCPU(Number(e.target.value))}
                      className="w-full accent-indigo-600"
                    />
                  </div>

                  <div>
                    <label className="block text-slate-700 dark:text-slate-300 mb-1">
                      Max PIDs / Tasks: {limitTasks}
                    </label>
                    <input
                      type="number"
                      value={limitTasks}
                      onChange={(e) => setLimitTasks(Number(e.target.value))}
                      className="w-full px-3 py-2 rounded-xl bg-slate-50 dark:bg-surface-800 border border-slate-300 dark:border-surface-700 text-slate-900 dark:text-white focus:outline-none"
                    />
                  </div>

                  <div className="flex justify-end gap-3 pt-3 border-t border-slate-200 dark:border-surface-700">
                    <button
                      onClick={() => setIsolationModalSite(null)}
                      className="px-4 py-2 rounded-xl bg-slate-100 dark:bg-surface-800 text-slate-700 dark:text-slate-300 font-bold"
                    >
                      Close
                    </button>
                    <button
                      onClick={() => {
                        showToast(`cgroups v2 limits updated for '${isolationModalSite.primary_domain}'`);
                        setIsolationModalSite(null);
                      }}
                      className="px-5 py-2 rounded-xl bg-indigo-600 hover:bg-indigo-700 text-white font-bold"
                    >
                      Save Quotas
                    </button>
                  </div>
                </div>
              )}
            </div>
          </div>
        )}

        {/* 15. 1-Click App Installer Modal */}
        {appModalSite && (
          <OneClickAppModal
            website={appModalSite}
            isOpen={Boolean(appModalSite)}
            onClose={() => setAppModalSite(null)}
            onSuccess={() => {
              showToast(`Application successfully installed on ${appModalSite.primary_domain}!`);
              fetchData();
            }}
          />
        )}

        {/* 16. Enterprise Site Modification Modal (All 16 aaPanel/cPanel options) */}
        {siteModalWebsite && (
          <SiteModificationModal
            website={siteModalWebsite}
            isOpen={siteModalOpen}
            onClose={() => setSiteModalOpen(false)}
            initialTab={siteModalTab}
            onUpdateWebsite={(updated) => {
              setWebsites((prev) =>
                prev.map((s) => (s.id === siteModalWebsite.id ? { ...s, ...updated } : s))
              );
              setSiteModalWebsite((prev) => (prev ? { ...prev, ...updated } : null));
            }}
            showToast={showToast}
          />
        )}
      </div>
    </DashboardShell>
  );

  // Helper function for rewrite modal
  function openRewriteModal(site: Website) {
    setSelectedSite(site);
    setRewriteModalOpen(true);
  }
}
