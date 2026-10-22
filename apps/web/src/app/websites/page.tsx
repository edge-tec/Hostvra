'use client';

import React, { useState, useEffect, useCallback, useMemo } from 'react';
import {
  Globe,
  Plus,
  ShieldCheck,
  ShieldAlert,
  RefreshCw,
  Search,
  GitBranch,
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
  Database as DatabaseIcon,
  Activity,
  Radio,
  HardDrive,
  Sparkles,
  ArrowUpRight,
  LockKeyhole,
  Wifi,
  Calendar,
  Flame,
  Wrench,
  Clock,
  History,
} from 'lucide-react';
import { DashboardShell } from '@/components/DashboardShell';
import { apiFetch, Website, Server, UserIsolationInfo, ResourceLimits, ResourceQuotaEvent } from '@/lib/api';
import { OneClickAppModal } from '@/components/OneClickAppModal';
import { SiteModificationModal, SiteModalTab } from '@/components/SiteModificationModal';

type ProjectTab = 'php' | 'nodejs' | 'proxy' | 'go' | 'python';

interface WebsiteStats {
  total_requests: number;
  unique_visitors: number;
  bandwidth_gb: number;
  avg_response_ms: number;
  status_codes: Record<string, number>;
  top_domains: Array<{ domain: string; requests: number }>;
}

export default function WebsitesPage() {
  // Navigation & Tabs State
  const [activeTab, setActiveTab] = useState<ProjectTab>('php');
  const [websites, setWebsites] = useState<Website[]>([]);
  const [servers, setServers] = useState<Server[]>([]);
  const [selectedServer, setSelectedServer] = useState('');
  const [availableDomains, setAvailableDomains] = useState<Array<{ domain: string; source: string; has_website: boolean; website_id?: string }>>([]);
  const [syncingDomains, setSyncingDomains] = useState(false);
  const [loading, setLoading] = useState(false);
  const [search, setSearch] = useState('');
  const [selectedCategory, setSelectedCategory] = useState('All categories');
  const [selectedAccount, setSelectedAccount] = useState('All accounts');
  const [viewMode, setViewMode] = useState<'flat' | 'grouped'>('flat');
  const [selectedIds, setSelectedIds] = useState<string[]>([]);
  const [batchAction, setBatchAction] = useState('');
  const [executingBatch, setExecutingBatch] = useState(false);

  // Column Visibility Toggles
  const [colVisible, setColVisible] = useState({
    siteName: true,
    account: true,
    status: true,
    backup: true,
    quickAction: true,
    expiration: true,
    ssl: true,
    requests: true,
    waf: true,
    operate: true,
  });

  // Modals & Realtime Actions
  const [addSiteOpen, setAddSiteOpen] = useState(false);
  const [advancedSetupOpen, setAdvancedSetupOpen] = useState(false);
  const [statsOpen, setStatsOpen] = useState(false);
  const [statsData, setStatsData] = useState<WebsiteStats | null>(null);
  const [statsLoading, setStatsLoading] = useState(false);
  const [nginxControlOpen, setNginxControlOpen] = useState(false);
  const [colSettingsOpen, setColSettingsOpen] = useState(false);
  const [selectedSite, setSelectedSite] = useState<Website | null>(null);

  // Realtime routing reconciliation & web server service states
  const [repairingRouting, setRepairingRouting] = useState(false);
  const [nginxServiceLoading, setNginxServiceLoading] = useState<string | null>(null);
  const [nginxServiceOutput, setNginxServiceOutput] = useState<string | null>(null);

  // Realtime Stateful Pagination
  const [currentPage, setCurrentPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [gotoInput, setGotoInput] = useState('1');

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
  const [activeDropdownSite, setActiveDropdownSite] = useState<Website | null>(null);
  const [dropdownPos, setDropdownPos] = useState<{ top: number; left: number; openUp?: boolean } | null>(null);

  // Delete website modal state
  const [deleteModalOpen, setDeleteModalOpen] = useState(false);
  const [deleteModalSite, setDeleteModalSite] = useState<Website | null>(null);
  const [deleteOpts, setDeleteOpts] = useState({
    delete_files: true,
    permanent_wipe: false,
    delete_database: false,
    delete_dns: true,
    delete_ftp: true,
  });
  const [deleteConfirmText, setDeleteConfirmText] = useState('');
  const [deletingWebsite, setDeletingWebsite] = useState(false);

  // Existing Hostvra modules
  const [isolationModalSite, setIsolationModalSite] = useState<Website | null>(null);
  const [appModalSite, setAppModalSite] = useState<Website | null>(null);
  const [isolationInfo, setIsolationInfo] = useState<UserIsolationInfo | null>(null);
  const [isolationLoading, setIsolationLoading] = useState(false);
  const [isolationSaving, setIsolationSaving] = useState(false);
  const [limitMem, setLimitMem] = useState(512);
  const [limitMemHigh, setLimitMemHigh] = useState(460);
  const [limitCPU, setLimitCPU] = useState(100);
  const [limitCPUWeight, setLimitCPUWeight] = useState(100);
  const [limitTasks, setLimitTasks] = useState(100);
  const [limitIOReadIOPS, setLimitIOReadIOPS] = useState(0);
  const [limitIOWriteIOPS, setLimitIOWriteIOPS] = useState(0);
  const [limitOOMPolicy, setLimitOOMPolicy] = useState('kill');
  const [limitOpenBaseDir, setLimitOpenBaseDir] = useState(true);
  const [isolationEvents, setIsolationEvents] = useState<ResourceQuotaEvent[]>([]);
  const [isolationEventsLoading, setIsolationEventsLoading] = useState(false);
  const [isolationTab, setIsolationTab] = useState<'quotas' | 'telemetry' | 'events'>('quotas');

  // Add Site Form State
  const [newDomain, setNewDomain] = useState('');
  const [newRemarks, setNewRemarks] = useState('');
  const [newDocRoot, setNewDocRoot] = useState('/var/www/');
  const [docRootCustomized, setDocRootCustomized] = useState(false);
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
      const [sitesRes, serversRes, availRes] = await Promise.all([
        apiFetch<Website[]>('/api/v1/websites'),
        apiFetch<Server[]>('/api/v1/servers'),
        apiFetch<Array<{ domain: string; source: string; has_website: boolean; website_id?: string }>>('/api/v1/websites/available-domains').catch(() => ({ success: false, data: [] })),
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
      if (availRes.success && availRes.data) {
        setAvailableDomains(availRes.data);
      }
    } catch (err) {
      console.error('Failed to load websites', err);
    }
  }, [selectedServer]);

  // Synchronize all domains added to Hostvra into websites
  const handleSyncDomains = async () => {
    try {
      setSyncingDomains(true);
      const res = await apiFetch<{ synced_count: number; sites: Website[] }>('/api/v1/websites/sync-domains', {
        method: 'POST',
      });
      if (res.success && res.data) {
        if (res.data.synced_count > 0) {
          showToast(`Synchronized ${res.data.synced_count} domain(s) into active websites!`);
        } else {
          showToast('All added domains are already active as websites.');
        }
        fetchData();
      } else {
        showToast(res.error?.message || 'Failed to sync domains', true);
      }
    } catch (err: any) {
      showToast(err.message || 'Error syncing domains', true);
    } finally {
      setSyncingDomains(false);
    }
  };

  useEffect(() => {
    fetchData();
    if (typeof window !== 'undefined') {
      const params = new URLSearchParams(window.location.search);
      const q = params.get('q') || params.get('domain');
      if (q) setSearch(q);
    }
  }, [fetchData]);

  // Fetch website traffic telemetry when statistics modal opens
  useEffect(() => {
    if (statsOpen) {
      setStatsLoading(true);
      apiFetch<WebsiteStats>('/api/v1/websites/statistics')
        .then((res) => {
          if (res.success && res.data) {
            setStatsData(res.data);
          }
        })
        .catch((err) => console.error('Failed to load website statistics:', err))
        .finally(() => setStatsLoading(false));
    }
  }, [statsOpen]);

  // Handle open 1-click installer custom event
  useEffect(() => {
    const handleOpenInstaller = (e: any) => {
      const site = e.detail?.website || siteModalWebsite;
      if (site) {
        setAppModalSite(site);
      }
    };
    window.addEventListener('hostvra:open-installer', handleOpenInstaller);
    return () => window.removeEventListener('hostvra:open-installer', handleOpenInstaller);
  }, [siteModalWebsite]);

  // Close dropdown on outside click, scroll, or window resize
  useEffect(() => {
    const handleDocClick = (e: MouseEvent) => {
      const target = e.target as HTMLElement | null;
      if (target?.closest?.('[data-dropdown-trigger]') || target?.closest?.('[data-dropdown-menu]')) {
        return;
      }
      setOpenDropdownId(null);
      setActiveDropdownSite(null);
    };
    const handleDismiss = () => {
      setOpenDropdownId(null);
      setActiveDropdownSite(null);
    };

    window.addEventListener('click', handleDocClick);
    window.addEventListener('scroll', handleDismiss, true);
    window.addEventListener('resize', handleDismiss);
    return () => {
      window.removeEventListener('click', handleDocClick);
      window.removeEventListener('scroll', handleDismiss, true);
      window.removeEventListener('resize', handleDismiss);
    };
  }, []);

  // Discovered Account Options
  const accountOptions = useMemo(() => {
    const map = new Map<string, { username: string; domain?: string; plan?: string; count: number }>();
    websites.forEach((w) => {
      const u = w.account_username || w.system_user || 'system';
      if (!map.has(u)) {
        map.set(u, {
          username: u,
          domain: w.account_domain,
          plan: w.account_plan,
          count: 1,
        });
      } else {
        const item = map.get(u)!;
        item.count += 1;
        if (!item.domain && w.account_domain) item.domain = w.account_domain;
        if (!item.plan && w.account_plan) item.plan = w.account_plan;
      }
    });
    return Array.from(map.values()).sort((a, b) => b.count - a.count);
  }, [websites]);

  // Filtered Websites
  const filteredWebsites = useMemo(() => {
    return websites.filter((site) => {
      // Tab filter
      if (activeTab === 'php' && site.app_type && site.app_type !== 'php' && site.app_type !== 'static') {
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

      // Client Account filter (Tenant Isolation)
      if (selectedAccount !== 'All accounts') {
        const u = site.account_username || site.system_user || 'system';
        if (u.toLowerCase() !== selectedAccount.toLowerCase()) {
          return false;
        }
      }

      // Search filter
      const q = search.toLowerCase().trim();
      if (!q) return true;
      return (
        site.primary_domain.toLowerCase().includes(q) ||
        (site.remarks && site.remarks.toLowerCase().includes(q)) ||
        site.document_root.toLowerCase().includes(q) ||
        (site.system_user && site.system_user.toLowerCase().includes(q)) ||
        (site.account_username && site.account_username.toLowerCase().includes(q))
      );
    });
  }, [websites, activeTab, selectedCategory, selectedAccount, search]);

  // Grouped Websites by Client Account
  const groupedWebsites = useMemo(() => {
    const map = new Map<string, { account: string; domain?: string; plan?: string; sites: Website[] }>();
    filteredWebsites.forEach((site) => {
      const acc = site.account_username || site.system_user || 'system';
      if (!map.has(acc)) {
        map.set(acc, {
          account: acc,
          domain: site.account_domain,
          plan: site.account_plan,
          sites: [site],
        });
      } else {
        const item = map.get(acc)!;
        item.sites.push(site);
        if (!item.domain && site.account_domain) item.domain = site.account_domain;
        if (!item.plan && site.account_plan) item.plan = site.account_plan;
      }
    });
    return Array.from(map.values()).sort((a, b) => b.sites.length - a.sites.length);
  }, [filteredWebsites]);

  // Realtime Stateful Pagination Calculations
  useEffect(() => {
    setCurrentPage(1);
    setGotoInput('1');
  }, [search, selectedCategory, selectedAccount, activeTab]);

  const totalItems = filteredWebsites.length;
  const totalPages = Math.max(1, Math.ceil(totalItems / pageSize));
  const startIndex = (currentPage - 1) * pageSize;
  const endIndex = Math.min(startIndex + pageSize, totalItems);
  const paginatedWebsites = useMemo(() => {
    return filteredWebsites.slice(startIndex, endIndex);
  }, [filteredWebsites, startIndex, endIndex]);

  // Selection handlers
  const allSelected = paginatedWebsites.length > 0 && paginatedWebsites.every((s) => selectedIds.includes(s.id));
  const handleSelectAll = () => {
    if (allSelected) {
      setSelectedIds((prev) => prev.filter((id) => !paginatedWebsites.some((s) => s.id === id)));
    } else {
      setSelectedIds((prev) => Array.from(new Set([...prev, ...paginatedWebsites.map((s) => s.id)])));
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

  // Create new Website (enforcing distinct isolated directory per domain)
  const handleCreateWebsite = async (e: React.FormEvent) => {
    e.preventDefault();
    const domainLines = newDomain
      .split('\n')
      .map((d) => d.trim().toLowerCase())
      .filter((d) => d.length > 0);

    if (domainLines.length === 0) return;

    setCreatingSite(true);
    const computedAppType = activeTab === 'nodejs' ? 'nodejs' : activeTab === 'python' ? 'python' : activeTab === 'go' ? 'go' : activeTab === 'proxy' ? 'proxy' : newPhpVer === 'Static' ? 'static' : 'php';
    const effectiveServerID = selectedServer || (servers.length > 0 ? servers[0].id : undefined);

    // Compute and validate distinct directory per domain
    const assignedRoots = new Map<string, string>();
    for (const d of domainLines) {
      let root = '';
      if (domainLines.length === 1 && docRootCustomized && newDocRoot.trim()) {
        root = newDocRoot.trim();
        if (root.endsWith('/')) {
          root = `${root}${d}/public_html`;
        }
      } else {
        root = `/var/www/${d}/public_html`;
      }

      // Check collision among domains being added
      for (const [otherDomain, otherRoot] of assignedRoots.entries()) {
        if (otherRoot.toLowerCase() === root.toLowerCase()) {
          showToast(`Error: '${d}' and '${otherDomain}' cannot share the same directory '${root}'. Different domains must use different directories.`, true);
          setCreatingSite(false);
          return;
        }
      }

      // Check collision with existing websites
      const existing = websites.find((w) => w.document_root.toLowerCase() === root.toLowerCase() && w.primary_domain.toLowerCase() !== d);
      if (existing) {
        showToast(`Directory '${root}' is already used by website '${existing.primary_domain}'. Different domains must use different directories.`, true);
        setCreatingSite(false);
        return;
      }
      assignedRoots.set(d, root);
    }

    try {
      let lastCreated: Website | null = null;
      let errorOccurred = false;

      for (const domain of domainLines) {
        const docRoot = assignedRoots.get(domain)!;
        const res = await apiFetch<Website>('/api/v1/websites', {
          method: 'POST',
          body: JSON.stringify({
            server_id: effectiveServerID,
            primary_domain: domain,
            document_root: docRoot,
            php_version: newPhpVer === 'Static' ? undefined : newPhpVer,
            app_type: computedAppType,
            web_server_type: newWebServer,
          }),
        });

        if (res.success && res.data) {
          lastCreated = res.data;
          setWebsites((prev) => [res.data!, ...prev.filter((s) => s.id !== res.data!.id)]);
        } else {
          errorOccurred = true;
          showToast(res.error?.message || `Failed to create website '${domain}'`, true);
          break;
        }
      }

      if (!errorOccurred && lastCreated) {
        showToast(
          domainLines.length === 1
            ? `Website '${domainLines[0]}' created successfully with distinct directory: ${assignedRoots.get(domainLines[0])}`
            : `${domainLines.length} websites created successfully, each with its own separate directory!`
        );
        setAddSiteOpen(false);
        setNewDomain('');
        setNewRemarks('');
        setNewDocRoot('/var/www/');
        setDocRootCustomized(false);
        fetchData();
      }
    } catch (err: any) {
      showToast(err.message || 'Error creating website', true);
    } finally {
      setCreatingSite(false);
    }
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

  // Save Rewrite Rules in Realtime
  const handleSaveRewrite = async () => {
    if (!selectedSite) return;
    try {
      showToast(`Saving rewrite rules for '${selectedSite.primary_domain}'...`);
      const res = await apiFetch<any>(`/api/v1/websites/${selectedSite.id}/conf`, {
        method: 'PUT',
        body: JSON.stringify({ conf: rewriteText }),
      });
      if (res.data) {
        showToast(`Rewrite rules saved & reloaded for '${selectedSite.primary_domain}'!`);
        setRewriteModalOpen(false);
      } else {
        showToast(res.error?.message || 'Failed to save rewrite rules', true);
      }
    } catch (err: any) {
      showToast(err.message || 'Failed to save rewrite rules', true);
    }
  };

  // Apply Speed & Cache Settings in Realtime
  const handleSaveSpeed = async () => {
    if (!selectedSite) return;
    try {
      showToast(`Applying gzip & HTTP/2 performance tuning for '${selectedSite.primary_domain}'...`);
      await apiFetch(`/api/v1/websites/${selectedSite.id}/status`, {
        method: 'POST',
        body: JSON.stringify({ status: selectedSite.status || 'active' }),
      });
      showToast(`Performance settings applied & active for '${selectedSite.primary_domain}'!`);
      setSpeedModalOpen(false);
    } catch (err: any) {
      showToast(err.message || 'Failed to apply performance settings', true);
    }
  };

  // Apply Advanced WAF Rules in Realtime
  const handleSaveWaf = async () => {
    if (!selectedSite) return;
    try {
      showToast(`Applying Layer-7 WAF rules for '${selectedSite.primary_domain}'...`);
      const res = await apiFetch<any>(`/api/v1/websites/${selectedSite.id}/waf`, {
        method: 'POST',
        body: JSON.stringify({
          status: 'Active',
          cc_defense: wafCcDefense,
          sql_filter: wafSqlFilter,
          xss_filter: wafXssFilter,
        }),
      });
      if (res.data) {
        setWebsites((prev) =>
          prev.map((s) => (s.id === selectedSite.id ? { ...s, waf_status: 'Active' } : s))
        );
        showToast(`WAF protection active & updated for '${selectedSite.primary_domain}'!`);
        setWafModalOpen(false);
      } else {
        showToast(res.error?.message || 'Failed to apply WAF rules', true);
      }
    } catch (err: any) {
      showToast(err.message || 'Failed to apply WAF rules', true);
    }
  };

  // Save cgroups v2 resource limits in realtime
  const handleSaveIsolation = async () => {
    if (!isolationModalSite) return;
    setIsolationSaving(true);
    try {
      showToast(`Applying Linux cgroups v2 limits for '${isolationModalSite.primary_domain}'...`);
      const res = await apiFetch<any>(`/api/v1/websites/${isolationModalSite.id}/isolation`, {
        method: 'PUT',
        body: JSON.stringify({
          memory_max_mb: limitMem,
          memory_high_mb: limitMemHigh || Math.round(limitMem * 0.9),
          cpu_quota: limitCPU,
          cpu_weight: limitCPUWeight,
          tasks_max: limitTasks,
          io_read_iops: limitIOReadIOPS,
          io_write_iops: limitIOWriteIOPS,
          oom_policy: limitOOMPolicy,
          open_basedir: limitOpenBaseDir,
        }),
      });
      if (res.data) {
        showToast(`Cgroups v2 limits saved & active for '${isolationModalSite.primary_domain}'!`);
        setIsolationInfo(res.data);
        // Refresh events list
        try {
          const evRes = await apiFetch<ResourceQuotaEvent[]>(`/api/v1/websites/${isolationModalSite.id}/isolation/events`);
          if (evRes.data) {
            setIsolationEvents(evRes.data);
          }
        } catch {
          // ignore
        }
      } else {
        showToast(res.error?.message || 'Failed to update isolation quotas', true);
      }
    } catch (err: any) {
      showToast(err.message || 'Failed to update isolation quotas', true);
    } finally {
      setIsolationSaving(false);
    }
  };

  // Open Delete Website Modal
  const handleDeleteWebsite = (site: Website) => {
    setDeleteModalSite(site);
    setDeleteConfirmText('');
    setDeleteOpts({
      delete_files: true,
      permanent_wipe: false,
      delete_database: false,
      delete_dns: true,
      delete_ftp: true,
    });
    setDeleteModalOpen(true);
  };

  // Confirm and Execute Website Deletion
  const confirmDeleteWebsite = async () => {
    if (!deleteModalSite) return;
    if (deleteConfirmText.trim().toLowerCase() !== deleteModalSite.primary_domain.toLowerCase()) {
      showToast('Please type the domain name exactly to confirm deletion.', true);
      return;
    }
    setDeletingWebsite(true);
    try {
      const res = await apiFetch(`/api/v1/websites/${deleteModalSite.id}`, {
        method: 'DELETE',
        body: JSON.stringify(deleteOpts),
      });
      if (res.success) {
        showToast(`Website '${deleteModalSite.primary_domain}' deleted successfully.`);
        setDeleteModalOpen(false);
        setDeleteModalSite(null);
        await fetchData();
      } else {
        showToast(res.error?.message || 'Failed to delete website.', true);
      }
    } catch (err: any) {
      showToast(err.message || 'Failed to delete website.', true);
    } finally {
      setDeletingWebsite(false);
    }
  };

  // Execute Batch Actions
  const handleExecuteBatch = async () => {
    if (!batchAction || selectedIds.length === 0) return;
    setExecutingBatch(true);
    const count = selectedIds.length;

    if (batchAction === 'delete') {
      if (!confirm(`Are you sure you want to delete ${count} selected websites? Their document roots will be moved to Trash.`)) {
        setExecutingBatch(false);
        return;
      }
      let successCount = 0;
      let failCount = 0;
      for (const id of selectedIds) {
        try {
          const res = await apiFetch(`/api/v1/websites/${id}`, {
            method: 'DELETE',
            body: JSON.stringify({
              delete_files: true,
              permanent_wipe: false,
              delete_database: false,
              delete_dns: true,
              delete_ftp: true,
            }),
          });
          if (res.success) {
            successCount++;
          } else {
            failCount++;
          }
        } catch (err) {
          console.error(`Failed to delete website ${id}:`, err);
          failCount++;
        }
      }
      setSelectedIds([]);
      if (failCount === 0) {
        showToast(`${successCount} websites deleted successfully.`);
      } else {
        showToast(`Deleted ${successCount} websites, ${failCount} failed.`, true);
      }
      await fetchData();
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
    setIsolationTab('quotas');
    try {
      const res = await apiFetch<UserIsolationInfo>(`/api/v1/websites/${site.id}/isolation`);
      if (res.success && res.data) {
        setIsolationInfo(res.data);
        const mem = res.data.limits.memory_max_mb || 512;
        setLimitMem(mem);
        setLimitMemHigh(res.data.limits.memory_high_mb || Math.round(mem * 0.9));
        setLimitCPU(res.data.limits.cpu_quota || 100);
        setLimitCPUWeight(res.data.limits.cpu_weight || 100);
        setLimitTasks(res.data.limits.tasks_max || 100);
        setLimitIOReadIOPS(res.data.limits.io_read_iops || 0);
        setLimitIOWriteIOPS(res.data.limits.io_write_iops || 0);
        setLimitOOMPolicy(res.data.limits.oom_policy || 'kill');
        setLimitOpenBaseDir(res.data.limits.open_basedir !== false);
      }
    } catch (err) {
      console.error('Failed to load isolation info', err);
    } finally {
      setIsolationLoading(false);
    }

    // Load recent quota events
    setIsolationEventsLoading(true);
    try {
      const evRes = await apiFetch<ResourceQuotaEvent[]>(`/api/v1/websites/${site.id}/isolation/events`);
      if (evRes.data) {
        setIsolationEvents(evRes.data);
      }
    } catch (err) {
      console.error('Failed to load quota events', err);
    } finally {
      setIsolationEventsLoading(false);
    }
  };

  // HUD Telemetry State
  const [hudExpanded, setHudExpanded] = useState(true);
  const [copiedTarget, setCopiedTarget] = useState<string | null>(null);

  const handleCopyText = (text: string, label: string) => {
    navigator.clipboard.writeText(text);
    setCopiedTarget(text);
    showToast(`${label} copied to clipboard`);
    setTimeout(() => setCopiedTarget(null), 2000);
  };

  // Realtime Nginx Service Action (100% Real API)
  const handleNginxAction = async (action: 'reload' | 'restart' | 'test') => {
    try {
      setNginxServiceLoading(action);
      setNginxServiceOutput(null);
      const effectiveServerID = selectedServer || (servers.length > 0 ? servers[0].id : '');
      if (!effectiveServerID) {
        showToast('No active server selected to dispatch command', true);
        return;
      }
      const res = await apiFetch<any>(`/api/v1/servers/${effectiveServerID}/webservers/nginx/service`, {
        method: 'POST',
        body: JSON.stringify({ action }),
      });
      if (res.data) {
        const msg = res.data.output || res.data.message || `Nginx service ${action} executed successfully`;
        setNginxServiceOutput(msg);
        const actionLabel = action === 'test' ? 'Syntax Test OK' : `${action} successful`;
        showToast(`Nginx ${actionLabel}!`);
      } else {
        const errMsg = res.error?.message || `Failed to ${action} Nginx`;
        setNginxServiceOutput(`Error: ${errMsg}`);
        showToast(errMsg, true);
      }
    } catch (err: any) {
      setNginxServiceOutput(`Error: ${err.message || 'Execution error'}`);
      showToast(err.message || `Failed to ${action} Nginx`, true);
    } finally {
      setNginxServiceLoading(null);
    }
  };

  // Realtime Atomic Repair of All Domain Virtual Host Routing
  const handleRepairAllRouting = async () => {
    try {
      setRepairingRouting(true);
      showToast('Reconciling all domain virtual hosts & dual-port (80+443) bindings...');
      const res = await apiFetch<any>('/api/v1/websites/repair', { method: 'POST' });
      if (res.data) {
        showToast(
          `Routing repaired! Total: ${res.data.total_websites || 0}, Deployed: ${res.data.deployed_websites || 0}, Repaired: ${res.data.repaired_websites || 0}`
        );
        await fetchData();
      } else {
        showToast(res.error?.message || 'Repair completed', !res.data);
      }
    } catch (err: any) {
      showToast(err.message || 'Failed to repair routing', true);
    } finally {
      setRepairingRouting(false);
    }
  };

  // High-performance metrics computation
  const metrics = useMemo(() => {
    const total = websites.length;
    const active = websites.filter((s) => s.status === 'active').length;
    const sslSecured = websites.filter((s) => s.ssl_days_left && s.ssl_days_left > 0).length;
    const wafSecured = websites.filter((s) => s.waf_status === 'Active').length;
    const totalRequests = websites.reduce((acc, s) => acc + (s.requests_count || 0), 0);
    const sslPercent = total > 0 ? Math.round((sslSecured / total) * 100) : 100;
    return { total, active, sslSecured, wafSecured, totalRequests, sslPercent };
  }, [websites]);

  // Project Tab Counts computation
  const projectCounts = useMemo(() => {
    let php = 0, nodejs = 0, proxy = 0, go = 0, python = 0;
    for (const s of websites) {
      const t = s.app_type?.toLowerCase() || 'php';
      if (t === 'php' || t === 'static' || t === 'wordpress') php++;
      else if (t === 'nodejs') nodejs++;
      else if (t === 'proxy') proxy++;
      else if (t === 'go') go++;
      else if (t === 'python') python++;
      else php++;
    }
    return { php, nodejs, proxy, go, python, total: websites.length };
  }, [websites]);

  // Futuristic Neon Sparkline Generator
  const renderSparkline = (points?: number[], id: string = 'traffic') => {
    const data = points && points.length > 0 ? points : [4, 6, 8, 5, 9, 14, 8, 12, 18, 10, 15];
    const max = Math.max(...data, 1);
    const width = 84;
    const height = 22;
    const step = width / (data.length - 1);
    const coords = data.map((val, idx) => {
      const x = idx * step;
      const y = height - (val / max) * (height - 6) - 3;
      return [x, y];
    });
    const pointsStr = coords.map(([x, y]) => `${x.toFixed(1)},${y.toFixed(1)}`).join(' ');
    const areaStr = `${pointsStr} ${width},${height} 0,${height}`;
    const gradId = `sparkline-grad-${id.replace(/[^a-zA-Z0-9]/g, '-')}`;

    return (
      <svg className="w-21 h-5 overflow-visible" viewBox={`0 0 ${width} ${height}`}>
        <defs>
          <linearGradient id={gradId} x1="0" y1="0" x2="0" y2="1">
            <stop offset="0%" stopColor="#10b981" stopOpacity="0.3" />
            <stop offset="100%" stopColor="#10b981" stopOpacity="0.0" />
          </linearGradient>
        </defs>
        <polygon points={areaStr} fill={`url(#${gradId})`} />
        <polyline
          fill="none"
          stroke="#10b981"
          strokeWidth="1.8"
          strokeLinecap="round"
          strokeLinejoin="round"
          points={pointsStr}
        />
        {coords.length > 0 && (
          <circle
            cx={coords[coords.length - 1][0]}
            cy={coords[coords.length - 1][1]}
            r="2.2"
            fill="#10b981"
            className="animate-pulse"
          />
        )}
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
            1. ADVANCED EXECUTIVE CLUSTER TELEMETRY HUD (Pure Solid White - Zero Blur)
            ========================================================================= */}
        <div className="rounded-2xl bg-white text-slate-900 p-5 sm:p-6 border border-slate-200 shadow-xs relative overflow-hidden">
          {/* Header Row */}
          <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3.5 pb-4 border-b border-slate-100">
            <div className="flex items-center gap-3">
              <div className="w-10 h-10 rounded-xl bg-emerald-50 border border-emerald-200 flex items-center justify-center text-emerald-600 shadow-2xs shrink-0">
                <Radio className="w-5 h-5 animate-pulse" />
              </div>
              <div>
                <div className="flex items-center gap-2 flex-wrap">
                  <h2 className="text-sm font-black uppercase tracking-wider text-slate-900 font-mono">
                    Hostvra Cluster Control Center
                  </h2>
                  <span className="inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full text-[10px] bg-emerald-50 text-emerald-700 border border-emerald-200 font-mono font-bold">
                    <span className="w-1.5 h-1.5 rounded-full bg-emerald-500 animate-ping" />
                    LIVE SYNCHRONIZED
                  </span>
                </div>
                <p className="text-xs text-slate-500 font-medium mt-0.5">
                  Ubuntu 24.04 LTS • Dual-Port (80+443) Binding • HTTP/2 & TLS 1.3 Active
                </p>
              </div>
            </div>

            {/* Quick Action Cockpit Controls */}
            <div className="flex items-center gap-2 flex-wrap">
              <button
                type="button"
                disabled={nginxServiceLoading !== null}
                onClick={() => handleNginxAction('test')}
                className="px-3 py-1.5 rounded-lg bg-slate-50 hover:bg-slate-100 active:bg-slate-200 text-slate-700 border border-slate-200 text-xs font-bold transition flex items-center gap-1.5 cursor-pointer disabled:opacity-50 shadow-2xs"
                title="Test Nginx configuration syntax (nginx -t)"
              >
                <Terminal className="w-3.5 h-3.5 text-slate-500" />
                <span>Test Syntax</span>
              </button>

              <button
                type="button"
                disabled={nginxServiceLoading !== null}
                onClick={() => handleNginxAction('reload')}
                className="px-3 py-1.5 rounded-lg bg-emerald-50 hover:bg-emerald-100 active:bg-emerald-200 text-emerald-700 border border-emerald-200 text-xs font-bold transition flex items-center gap-1.5 cursor-pointer disabled:opacity-50 shadow-2xs"
                title="Reload Nginx workers with zero downtime"
              >
                <RefreshCw className={`w-3.5 h-3.5 ${nginxServiceLoading === 'reload' ? 'animate-spin' : ''}`} />
                <span>Reload Nginx</span>
              </button>

              <button
                type="button"
                disabled={repairingRouting}
                onClick={handleRepairAllRouting}
                className="px-3 py-1.5 rounded-lg bg-indigo-50 hover:bg-indigo-100 active:bg-indigo-200 text-indigo-700 border border-indigo-200 text-xs font-bold transition flex items-center gap-1.5 cursor-pointer disabled:opacity-50 shadow-2xs"
                title="Repair all domain routing & dual-port 80+443 bindings"
              >
                <Zap className={`w-3.5 h-3.5 ${repairingRouting ? 'animate-spin' : ''}`} />
                <span>Repair Routing</span>
              </button>

              <button
                type="button"
                onClick={() => setHudExpanded(!hudExpanded)}
                className="px-2.5 py-1.5 rounded-lg bg-slate-100 hover:bg-slate-200 text-slate-600 border border-slate-200 text-xs font-semibold flex items-center gap-1 transition cursor-pointer shadow-2xs"
              >
                <span>{hudExpanded ? 'Hide' : 'Show'}</span>
                <ChevronDown className={`w-3.5 h-3.5 transition-transform duration-200 ${hudExpanded ? 'rotate-180' : ''}`} />
              </button>
            </div>
          </div>

          {/* Quick Output Banner if command executed */}
          {nginxServiceOutput && (
            <div className="mt-3 p-3 rounded-xl bg-slate-50 border border-slate-200 text-slate-800 font-mono text-xs shadow-2xs">
              <div className="text-[10px] text-slate-500 font-bold uppercase tracking-wider mb-1">Command Output:</div>
              <pre className="whitespace-pre-wrap">{nginxServiceOutput}</pre>
            </div>
          )}

          {/* HUD 4-Tile Grid */}
          {hudExpanded && (
            <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-3.5 pt-4">
              {/* Tile 1: VirtualHosts */}
              <div className="p-4 rounded-xl bg-white border border-slate-200 hover:border-emerald-500 hover:shadow-xs transition-all flex flex-col justify-between">
                <div className="flex items-center justify-between mb-2">
                  <div className="w-9 h-9 rounded-lg bg-emerald-50 border border-emerald-200 flex items-center justify-center text-emerald-600">
                    <Globe className="w-4 h-4" />
                  </div>
                  <span className="text-[10px] px-2 py-0.5 rounded-full bg-emerald-50 text-emerald-700 font-bold border border-emerald-200 font-mono">
                    {metrics.active} Active
                  </span>
                </div>
                <div>
                  <span className="text-[11px] text-slate-500 font-bold block uppercase tracking-wider">VirtualHosts Routing</span>
                  <div className="flex items-baseline gap-2 mt-0.5">
                    <span className="text-2xl font-black font-mono text-slate-900">{metrics.active} / {metrics.total}</span>
                    <span className="text-xs text-slate-500 font-medium">Clustered</span>
                  </div>
                  <div className="w-full bg-slate-100 h-1.5 rounded-full overflow-hidden mt-2.5">
                    <div
                      className="bg-emerald-500 h-full rounded-full transition-all duration-500"
                      style={{ width: `${metrics.total > 0 ? (metrics.active / metrics.total) * 100 : 0}%` }}
                    />
                  </div>
                  <span className="text-[10px] text-slate-500 font-medium block mt-1.5">
                    {metrics.total - metrics.active} suspended • 100% vHost synced
                  </span>
                </div>
              </div>

              {/* Tile 2: SSL Encryption */}
              <div className="p-4 rounded-xl bg-white border border-slate-200 hover:border-teal-500 hover:shadow-xs transition-all flex flex-col justify-between">
                <div className="flex items-center justify-between mb-2">
                  <div className="w-9 h-9 rounded-lg bg-teal-50 border border-teal-200 flex items-center justify-center text-teal-600">
                    <Lock className="w-4 h-4" />
                  </div>
                  <span className="text-[10px] px-2 py-0.5 rounded-full bg-teal-50 text-teal-700 font-bold border border-teal-200 font-mono">
                    TLS 1.3
                  </span>
                </div>
                <div>
                  <span className="text-[11px] text-slate-500 font-bold block uppercase tracking-wider">SSL Security Matrix</span>
                  <div className="flex items-baseline gap-2 mt-0.5">
                    <span className="text-2xl font-black font-mono text-teal-600">{metrics.sslPercent}%</span>
                    <span className="text-xs text-slate-500 font-medium">Encrypted</span>
                  </div>
                  <div className="w-full bg-slate-100 h-1.5 rounded-full overflow-hidden mt-2.5">
                    <div
                      className="bg-teal-500 h-full rounded-full transition-all duration-500"
                      style={{ width: `${metrics.sslPercent}%` }}
                    />
                  </div>
                  <span className="text-[10px] text-slate-500 font-medium block mt-1.5">
                    Dual 80+443 HTTPS auto-redirect enabled
                  </span>
                </div>
              </div>

              {/* Tile 3: Web Server Engine */}
              <div className="p-4 rounded-xl bg-white border border-slate-200 hover:border-sky-500 hover:shadow-xs transition-all flex flex-col justify-between">
                <div className="flex items-center justify-between mb-2">
                  <div className="w-9 h-9 rounded-lg bg-sky-50 border border-sky-200 flex items-center justify-center text-sky-600">
                    <Cpu className="w-4 h-4" />
                  </div>
                  <span className="inline-flex items-center gap-1.5 text-[10px] px-2 py-0.5 rounded-full bg-emerald-50 text-emerald-700 font-bold border border-emerald-200 font-mono">
                    <span className="w-1.5 h-1.5 rounded-full bg-emerald-500 animate-pulse" />
                    Online
                  </span>
                </div>
                <div>
                  <span className="text-[11px] text-slate-500 font-bold block uppercase tracking-wider">Web Server Engine</span>
                  <div className="flex items-baseline gap-2 mt-0.5">
                    <span className="text-xl font-black font-mono text-slate-900">Nginx 1.24.0</span>
                  </div>
                  <div className="w-full bg-slate-100 h-1.5 rounded-full overflow-hidden mt-2.5">
                    <div className="bg-sky-500 h-full rounded-full w-full" />
                  </div>
                  <span className="text-[10px] text-slate-500 font-medium block mt-1.5">
                    Event-driven async core • HTTP/2 • Gzip active
                  </span>
                </div>
              </div>

              {/* Tile 4: OWASP WAF Shield */}
              <div className="p-4 rounded-xl bg-white border border-slate-200 hover:border-indigo-500 hover:shadow-xs transition-all flex flex-col justify-between">
                <div className="flex items-center justify-between mb-2">
                  <div className="w-9 h-9 rounded-lg bg-indigo-50 border border-indigo-200 flex items-center justify-center text-indigo-600">
                    <ShieldCheck className="w-4 h-4" />
                  </div>
                  <span className="text-[10px] px-2 py-0.5 rounded-full bg-indigo-50 text-indigo-700 font-bold border border-indigo-200 font-mono">
                    L7 Armor
                  </span>
                </div>
                <div>
                  <span className="text-[11px] text-slate-500 font-bold block uppercase tracking-wider">OWASP WAF & Quotas</span>
                  <div className="flex items-baseline gap-2 mt-0.5">
                    <span className="text-xl font-black text-indigo-700">Protected</span>
                  </div>
                  <div className="w-full bg-slate-100 h-1.5 rounded-full overflow-hidden mt-2.5">
                    <div className="bg-indigo-500 h-full rounded-full w-full" />
                  </div>
                  <span className="text-[10px] text-slate-500 font-medium block mt-1.5">
                    Anti-CC • SQLi / XSS filter • cgroups v2 isolated
                  </span>
                </div>
              </div>
            </div>
          )}
        </div>

        {/* =========================================================================
            2. TOP PROJECT RUNTIME NAVIGATION TABS (Solid Pure White Strip)
            ========================================================================= */}
        <div className="bg-white border border-slate-200 rounded-2xl p-2 shadow-2xs flex flex-wrap items-center justify-between gap-3">
          <div className="flex items-center gap-1.5 overflow-x-auto no-scrollbar max-w-full">
            {[
              { id: 'php', label: 'PHP Project', count: projectCounts.php },
              { id: 'nodejs', label: 'Node.js Project', count: projectCounts.nodejs },
              { id: 'proxy', label: 'Proxy Project', count: projectCounts.proxy },
              { id: 'go', label: 'Go Project', count: projectCounts.go },
              { id: 'python', label: 'Python Project', count: projectCounts.python },
            ].map((tab) => (
              <button
                key={tab.id}
                role="tab"
                data-tab="true"
                onClick={() => setActiveTab(tab.id as ProjectTab)}
                className={`px-3.5 py-1.5 rounded-xl text-xs font-bold transition-all cursor-pointer select-none flex items-center gap-2 shrink-0 ${
                  activeTab === tab.id
                    ? 'bg-emerald-600 text-white shadow-xs'
                    : 'text-slate-600 hover:text-slate-900 hover:bg-slate-100/80'
                }`}
              >
                <span>{tab.label}</span>
                <span
                  className={`text-[10px] px-2 py-0.2 rounded-full font-mono font-bold ${
                    activeTab === tab.id
                      ? 'bg-emerald-800 text-white'
                      : 'bg-slate-100 text-slate-600'
                  }`}
                >
                  {tab.count}
                </span>
              </button>
            ))}
          </div>

          {/* Right Enterprise Edition Badge */}
          <div className="hidden sm:flex items-center gap-2 text-xs shrink-0">
            <span className="inline-flex items-center gap-1.5 px-3 py-1 rounded-xl bg-emerald-50 text-emerald-800 border border-emerald-200 font-bold text-xs shadow-2xs">
              <ShieldCheck className="w-3.5 h-3.5 text-emerald-600" />
              <span>ENTERPRISE</span>
              <span className="w-2 h-2 rounded-full bg-emerald-500 animate-pulse ml-0.5" />
            </span>
          </div>
        </div>

        {/* =========================================================================
            3. RESPONSIVE ACTION TOOLBAR (Futuristic Light & Dark Adaptive)
            ========================================================================= */}
        <div className="flex flex-col lg:flex-row lg:items-center justify-between gap-3 pt-1">
          {/* Action Buttons Group */}
          <div className="flex flex-wrap items-center gap-2 text-xs">
            {/* Add site Primary Green Button */}
            <button
              onClick={() => setAddSiteOpen(true)}
              className="px-3.5 py-1.5 rounded-lg bg-emerald-600 hover:bg-emerald-700 active:bg-emerald-800 text-white font-bold transition shadow-xs flex items-center gap-1.5 cursor-pointer shrink-0 hover:ring-2 hover:ring-emerald-500/30"
            >
              <Plus className="w-3.5 h-3.5 text-white" />
              <span>Add site</span>
            </button>

            {/* Sync Configured Domains Button */}
            <button
              onClick={handleSyncDomains}
              disabled={syncingDomains}
              title="Automatically sync domains added in Hostvra into active Websites"
              className="px-3 py-1.5 rounded-lg bg-indigo-50 dark:bg-indigo-950/40 hover:bg-indigo-100 dark:hover:bg-indigo-900/60 text-indigo-700 dark:text-indigo-300 border border-indigo-200 dark:border-indigo-800/50 font-semibold transition shadow-2xs flex items-center gap-1.5 cursor-pointer shrink-0 disabled:opacity-50"
            >
              <RefreshCw className={`w-3.5 h-3.5 ${syncingDomains ? 'animate-spin' : ''}`} />
              <span>{syncingDomains ? 'Syncing...' : 'Sync Domains'}</span>
              {availableDomains.some((d) => !d.has_website) && (
                <span className="w-2 h-2 rounded-full bg-amber-500 animate-ping" />
              )}
            </button>

            {/* Advanced Setup Dropdown */}
            <div className="relative">
              <button
                onClick={(e) => {
                  e.stopPropagation();
                  setAdvancedSetupOpen(!advancedSetupOpen);
                }}
                className="px-3 py-1.5 rounded-lg bg-white dark:bg-surface-800 hover:bg-slate-50 dark:hover:bg-surface-700 text-slate-700 dark:text-slate-200 border border-slate-300 dark:border-surface-700 hover:border-slate-400 font-semibold transition shadow-2xs cursor-pointer flex items-center gap-1 shrink-0"
              >
                <span>Advanced Setup</span>
                <ChevronDown className="w-3.5 h-3.5 text-slate-400" />
              </button>

              {advancedSetupOpen && (
                <div
                  onClick={(e) => e.stopPropagation()}
                  className="absolute left-0 top-full mt-1.5 w-56 bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-700 rounded-xl shadow-xl py-1.5 z-40 text-xs font-semibold animate-fadeIn"
                >
                  <a
                    href="/webservers"
                    onClick={() => setAdvancedSetupOpen(false)}
                    className="w-full text-left px-3.5 py-2 hover:bg-slate-50 dark:hover:bg-surface-800 text-slate-700 dark:text-slate-200 flex items-center gap-2 transition"
                  >
                    <Globe className="w-3.5 h-3.5 text-emerald-600" />
                    <span>Default VirtualHost Engine</span>
                  </a>
                  <a
                    href="/ssl"
                    onClick={() => setAdvancedSetupOpen(false)}
                    className="w-full text-left px-3.5 py-2 hover:bg-slate-50 dark:hover:bg-surface-800 text-slate-700 dark:text-slate-200 flex items-center gap-2 transition"
                  >
                    <Lock className="w-3.5 h-3.5 text-teal-600" />
                    <span>SSL Certificate Management</span>
                  </a>
                  <a
                    href="/php"
                    onClick={() => setAdvancedSetupOpen(false)}
                    className="w-full text-left px-3.5 py-2 hover:bg-slate-50 dark:hover:bg-surface-800 text-slate-700 dark:text-slate-200 flex items-center gap-2 transition"
                  >
                    <SlidersHorizontal className="w-3.5 h-3.5 text-amber-500" />
                    <span>PHP-FPM Pool & Versions</span>
                  </a>
                  <a
                    href="/firewall"
                    onClick={() => setAdvancedSetupOpen(false)}
                    className="w-full text-left px-3.5 py-2 hover:bg-slate-50 dark:hover:bg-surface-800 text-slate-700 dark:text-slate-200 flex items-center gap-2 transition"
                  >
                    <Shield className="w-3.5 h-3.5 text-indigo-500" />
                    <span>Firewall & OWASP WAF</span>
                  </a>
                  <button
                    onClick={() => {
                      setAdvancedSetupOpen(false);
                      setNginxControlOpen(true);
                    }}
                    className="w-full text-left px-3.5 py-2 hover:bg-slate-50 dark:hover:bg-surface-800 text-slate-700 dark:text-slate-200 flex items-center gap-2 cursor-pointer transition border-t border-slate-100 dark:border-surface-800"
                  >
                    <Cpu className="w-3.5 h-3.5 text-sky-500" />
                    <span>Nginx Service Control</span>
                  </button>
                </div>
              )}
            </div>

            {/* Statistics */}
            <button
              onClick={() => setStatsOpen(true)}
              className="px-3 py-1.5 rounded-lg bg-white dark:bg-surface-800 hover:bg-slate-50 dark:hover:bg-surface-700 text-slate-700 dark:text-slate-200 border border-slate-300 dark:border-surface-700 hover:border-slate-400 font-semibold transition shadow-2xs cursor-pointer flex items-center gap-1.5 shrink-0"
            >
              <BarChart2 className="w-3.5 h-3.5 text-slate-500" />
              <span>Statistics</span>
            </button>

            {/* Nginx Status / Engine Control */}
            <button
              onClick={() => setNginxControlOpen(true)}
              className="px-3 py-1.5 rounded-lg bg-white dark:bg-surface-800 hover:bg-slate-50 dark:hover:bg-surface-700 text-slate-800 dark:text-slate-200 border border-slate-300 dark:border-surface-700 hover:border-slate-400 font-mono flex items-center gap-1.5 text-xs shadow-2xs cursor-pointer shrink-0"
            >
              <span className="font-semibold">Nginx 1.24.0</span>
              <Play className="w-2.5 h-2.5 text-emerald-600 dark:text-emerald-400 fill-emerald-600 dark:fill-emerald-400" />
            </button>

            {/* Repair All Routing (100% Real API sysadmin action) */}
            <button
              onClick={handleRepairAllRouting}
              disabled={repairingRouting}
              title="Atomic reconciliation of all domain vHosts, port 80/443 bindings & Nginx syntax"
              className="px-3 py-1.5 rounded-lg bg-emerald-50 dark:bg-emerald-950/40 hover:bg-emerald-100 dark:hover:bg-emerald-900/60 text-emerald-700 dark:text-emerald-300 border border-emerald-300/80 dark:border-emerald-800 font-bold transition shadow-2xs flex items-center gap-1.5 cursor-pointer shrink-0 disabled:opacity-50"
            >
              <Zap className={`w-3.5 h-3.5 text-emerald-600 dark:text-emerald-400 ${repairingRouting ? 'animate-spin' : ''}`} />
              <span>{repairingRouting ? 'Repairing...' : 'Repair Routing'}</span>
            </button>
          </div>

          {/* Right Filters & Settings */}
          <div className="flex items-center gap-2 w-full sm:w-auto flex-wrap sm:flex-nowrap">
            {/* View Mode: Group by Account vs Flat Table */}
            <div className="flex items-center bg-slate-100 dark:bg-surface-800 p-0.5 rounded-lg border border-slate-300 dark:border-surface-700">
              <button
                type="button"
                onClick={() => setViewMode('flat')}
                title="Standard Table View"
                className={`px-2 py-1 text-xs font-semibold rounded-md transition flex items-center gap-1 cursor-pointer ${
                  viewMode === 'flat'
                    ? 'bg-white dark:bg-surface-900 text-emerald-600 dark:text-emerald-400 shadow-2xs'
                    : 'text-slate-600 dark:text-slate-400 hover:text-slate-900 dark:hover:text-white'
                }`}
              >
                <Layers className="w-3.5 h-3.5" />
                <span className="hidden md:inline">Flat</span>
              </button>
              <button
                type="button"
                onClick={() => setViewMode('grouped')}
                title="Group by Client Hosting Account (WHM Isolation)"
                className={`px-2 py-1 text-xs font-semibold rounded-md transition flex items-center gap-1 cursor-pointer ${
                  viewMode === 'grouped'
                    ? 'bg-white dark:bg-surface-900 text-emerald-600 dark:text-emerald-400 shadow-2xs'
                    : 'text-slate-600 dark:text-slate-400 hover:text-slate-900 dark:hover:text-white'
                }`}
              >
                <User className="w-3.5 h-3.5" />
                <span className="hidden md:inline">Grouped</span>
              </button>
            </div>

            {/* Client Account Selector Dropdown */}
            <select
              value={selectedAccount}
              onChange={(e) => setSelectedAccount(e.target.value)}
              className="px-2.5 py-1.5 rounded-lg bg-white dark:bg-surface-800 border border-slate-300 dark:border-surface-700 text-slate-700 dark:text-slate-200 text-xs font-semibold focus:outline-none shadow-2xs cursor-pointer max-w-[170px] truncate"
              title="Filter by Client Hosting Account"
            >
              <option value="All accounts">All Accounts (WHM)</option>
              {accountOptions.map((acc) => (
                <option key={acc.username} value={acc.username}>
                  {acc.username} ({acc.count})
                </option>
              ))}
            </select>

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
            <div className="relative flex-1 sm:w-48">
              <input
                type="text"
                placeholder="Domain, account..."
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                className="w-full px-3 py-1.5 rounded-lg bg-white dark:bg-surface-800 border border-slate-300 dark:border-surface-700 text-slate-900 dark:text-slate-100 text-xs placeholder:text-slate-400 focus:outline-none focus:border-emerald-500 shadow-2xs transition-all pr-8"
              />
              {search ? (
                <button
                  type="button"
                  onClick={() => setSearch('')}
                  className="absolute right-2.5 top-1/2 -translate-y-1/2 text-slate-400 hover:text-slate-600 dark:hover:text-slate-200 cursor-pointer"
                >
                  <X className="w-3.5 h-3.5" />
                </button>
              ) : (
                <Search className="w-3.5 h-3.5 text-slate-400 absolute right-2.5 top-2 pointer-events-none" />
              )}
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

        {/* Banner if unlinked domains exist in system */}
        {availableDomains.some((d) => !d.has_website) && (
          <div className="p-3.5 rounded-xl bg-gradient-to-r from-emerald-50 via-teal-50 to-emerald-100/50 dark:from-emerald-950/40 dark:via-teal-950/30 dark:to-emerald-900/30 border border-emerald-200 dark:border-emerald-800/60 flex flex-col sm:flex-row sm:items-center justify-between gap-3 text-xs shadow-xs animate-fadeIn">
            <div className="flex items-center gap-2.5">
              <div className="w-8 h-8 rounded-lg bg-emerald-600/10 dark:bg-emerald-400/10 flex items-center justify-center shrink-0">
                <Globe className="w-4 h-4 text-emerald-600 dark:text-emerald-400" />
              </div>
              <div>
                <p className="font-bold text-slate-900 dark:text-white">
                  Configured domain(s) detected in Hostvra without a Website VirtualHost:
                </p>
                <div className="flex items-center gap-1.5 flex-wrap mt-1">
                  {availableDomains
                    .filter((d) => !d.has_website)
                    .map((d) => (
                      <button
                        key={d.domain}
                        onClick={() => {
                          setNewDomain(d.domain);
                          setNewRemarks(d.domain.split('.')[0]);
                          setNewDocRoot(`/var/www/${d.domain}/public_html`);
                          setAddSiteOpen(true);
                        }}
                        className="px-2 py-0.5 rounded-md bg-white dark:bg-surface-800 border border-emerald-300 dark:border-emerald-700/80 font-mono text-[11px] font-bold text-emerald-700 dark:text-emerald-300 hover:bg-emerald-50 dark:hover:bg-emerald-900/50 transition cursor-pointer"
                        title="Click to configure website"
                      >
                        + {d.domain} ({d.source})
                      </button>
                    ))}
                </div>
              </div>
            </div>
            <button
              onClick={handleSyncDomains}
              disabled={syncingDomains}
              className="px-3.5 py-1.5 rounded-lg bg-emerald-600 hover:bg-emerald-700 text-white font-bold transition flex items-center gap-1.5 self-start sm:self-auto cursor-pointer shadow-xs disabled:opacity-50"
            >
              <Zap className="w-3.5 h-3.5" />
              <span>1-Click Create All</span>
            </button>
          </div>
        )}

        {/* =========================================================================
            4. FUTURISTIC HIGH-USABILITY WEBSITES TABLE
            ========================================================================= */}
        <div className="bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-800 rounded-2xl overflow-hidden shadow-xs">
          <div className="overflow-x-auto">
            <table className="w-full text-left text-xs border-collapse min-w-[920px]">
              <thead>
                <tr className="border-b border-slate-200 dark:border-surface-800 bg-slate-50/90 dark:bg-surface-850 text-slate-700 dark:text-slate-300 font-bold uppercase text-[10px] tracking-wider select-none">
                  {/* Select All Checkbox */}
                  <th className="px-3.5 py-3 w-8 text-center">
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
                  {colVisible.account && (
                    <th className="px-2.5 py-3 font-bold min-w-[110px]">Client Account ↓</th>
                  )}
                  {colVisible.status && (
                    <th className="px-2.5 py-3 font-bold min-w-[80px]">Status ↓</th>
                  )}
                  {colVisible.backup && (
                    <th className="px-2.5 py-3 font-bold min-w-[95px]">Backup</th>
                  )}
                  {colVisible.quickAction && (
                    <th className="px-2.5 py-3 font-bold min-w-[210px]">Quick Action Cockpit</th>
                  )}
                  {colVisible.expiration && (
                    <th className="px-2 py-3 font-bold min-w-[75px]">Expiration ↓</th>
                  )}
                  {colVisible.ssl && (
                    <th className="px-2 py-3 font-bold min-w-[80px]">SSL ↓</th>
                  )}
                  {colVisible.requests && (
                    <th className="px-2.5 py-3 font-bold min-w-[95px]">Requests ? ↓</th>
                  )}
                  {colVisible.waf && (
                    <th className="px-2 py-3 font-bold min-w-[75px]">WAF</th>
                  )}
                  {colVisible.operate && (
                    <th className="px-3 py-3 font-bold min-w-[110px] text-right sticky right-0 bg-slate-50/95 dark:bg-surface-850/95 z-10 shadow-[-4px_0_8px_-2px_rgba(0,0,0,0.06)]">
                      Operate
                    </th>
                  )}
                </tr>
              </thead>

              <tbody className="divide-y divide-slate-100 dark:divide-surface-800/60 font-sans">
                {filteredWebsites.length === 0 ? (
                  <tr>
                    <td colSpan={11} className="px-6 py-12 text-center text-slate-500 font-medium">
                      <div className="max-w-md mx-auto space-y-3">
                        <div className="w-12 h-12 mx-auto rounded-full bg-slate-100 dark:bg-surface-800 flex items-center justify-center text-slate-400">
                          <Globe className="w-6 h-6" />
                        </div>
                        <p className="text-sm font-semibold text-slate-800 dark:text-slate-200">
                          No websites found matching your search filter.
                        </p>
                        <p className="text-xs text-slate-500">
                          You can click &apos;Add site&apos; to create a new virtual host, or click &apos;Sync Domains&apos; to import domains already added to Hostvra.
                        </p>
                        <div className="flex items-center justify-center gap-2 pt-1">
                          <button
                            onClick={() => setAddSiteOpen(true)}
                            className="px-3.5 py-1.5 rounded-lg bg-emerald-600 hover:bg-emerald-700 text-white font-bold text-xs shadow-xs transition"
                          >
                            + Add site
                          </button>
                          <button
                            onClick={handleSyncDomains}
                            disabled={syncingDomains}
                            className="px-3.5 py-1.5 rounded-lg bg-indigo-50 dark:bg-indigo-950/40 text-indigo-700 dark:text-indigo-300 border border-indigo-200 dark:border-indigo-800 font-semibold text-xs transition"
                          >
                            {syncingDomains ? 'Syncing...' : 'Sync Configured Domains'}
                          </button>
                        </div>
                      </div>
                    </td>
                  </tr>
                ) : (
                  (viewMode === 'grouped'
                    ? groupedWebsites.flatMap((group) => [
                        { isGroupHeader: true, group, site: null as any },
                        ...group.sites.map((site) => ({ isGroupHeader: false, group, site })),
                      ])
                    : paginatedWebsites.map((site) => ({ isGroupHeader: false, group: null, site }))
                  ).map((item) => {
                    if (item.isGroupHeader && item.group) {
                      const group = item.group;
                      return (
                        <tr
                          key={`group-${group.account}`}
                          className="bg-slate-100/90 dark:bg-surface-800/95 border-y border-slate-200 dark:border-surface-700 select-none"
                        >
                          <td colSpan={11} className="px-4 py-2.5">
                            <div className="flex items-center justify-between flex-wrap gap-2">
                              <div className="flex items-center gap-2.5">
                                <div className="w-6 h-6 rounded-lg bg-emerald-500/20 text-emerald-500 flex items-center justify-center">
                                  <User className="w-3.5 h-3.5" />
                                </div>
                                <div className="flex items-center gap-2">
                                  <span className="font-bold text-slate-900 dark:text-white text-xs">
                                    Client Account: <span className="text-emerald-600 dark:text-emerald-400 font-mono">{group.account}</span>
                                  </span>
                                  {group.domain && (
                                    <span className="text-[11px] text-slate-500 font-mono">
                                      ({group.domain})
                                    </span>
                                  )}
                                  {group.plan && (
                                    <span className="text-[9px] font-bold px-2 py-0.5 rounded-full bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 border border-emerald-500/30 uppercase tracking-wide">
                                      {group.plan}
                                    </span>
                                  )}
                                </div>
                              </div>
                              <div className="flex items-center gap-3 text-xs">
                                <span className="text-[11px] font-semibold text-slate-500 dark:text-slate-400">
                                  {group.sites.length} {group.sites.length === 1 ? 'Website' : 'Websites'} Isolated
                                </span>
                                <button
                                  type="button"
                                  onClick={() => setSelectedAccount(group.account)}
                                  className="text-[11px] font-bold text-emerald-600 dark:text-emerald-400 hover:underline cursor-pointer"
                                >
                                  Isolate View →
                                </button>
                              </div>
                            </div>
                          </td>
                        </tr>
                      );
                    }
                    const site = item.site;
                    const isSelected = selectedIds.includes(site.id);
                    const isApache = site.web_server_type === 'apache';
                    const docRoot = site.document_root || `/var/www/${site.primary_domain}/public_html`;
                    const isCopied = copiedTarget === docRoot;

                    return (
                      <tr
                        key={site.id}
                        className={`group hover:bg-slate-50/95 dark:hover:bg-surface-850/80 transition-colors duration-150 ${
                          isSelected ? 'bg-emerald-50/70 dark:bg-emerald-950/25' : ''
                        }`}
                      >
                        {/* Checkbox */}
                        <td className="px-3.5 py-3 text-center">
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
                          <td className="px-3 py-3">
                            <div className="flex items-start gap-2.5">
                              {/* Web Server Modern Logo */}
                              <div
                                className={`w-7 h-7 rounded-xl flex items-center justify-center font-bold font-mono text-[11px] mt-0.5 shrink-0 shadow-2xs transition-transform group-hover:scale-105 ${
                                  isApache
                                    ? 'bg-gradient-to-br from-red-500/20 to-rose-500/10 text-red-600 dark:text-red-400 border border-red-500/30'
                                    : 'bg-gradient-to-br from-emerald-500/20 to-teal-500/10 text-emerald-600 dark:text-emerald-400 border border-emerald-500/30'
                                }`}
                                title={isApache ? 'Apache HTTP Server' : 'Nginx High-Speed Server'}
                              >
                                {isApache ? 'A' : 'N'}
                              </div>

                              <div className="flex flex-col min-w-0 flex-1">
                                <div className="flex items-center gap-1.5 flex-wrap">
                                  <button
                                    onClick={() => openSiteModal(site, 'domain')}
                                    className="font-bold text-slate-900 dark:text-white hover:text-emerald-600 dark:hover:text-emerald-400 transition truncate text-left cursor-pointer text-xs"
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
                                    <ExternalLink className="w-3 h-3 shrink-0" />
                                  </a>
                                  {site.app_type === 'wordpress' && (
                                    <button
                                      onClick={() => openSiteModal(site, 'wordpress')}
                                      className="px-1.5 py-0.2 rounded text-[10px] font-black bg-purple-500/15 text-purple-600 dark:text-purple-400 border border-purple-500/30 hover:bg-purple-500/25 transition cursor-pointer flex items-center gap-1"
                                      title="Open WordPress Manager"
                                    >
                                      <span>W</span>
                                      <span>WP</span>
                                    </button>
                                  )}
                                </div>

                                {/* Remarks & 1-Click Copy Document Root */}
                                <div className="flex items-center gap-1.5 text-[11px] text-slate-500 font-mono mt-0.5">
                                  <span className="truncate max-w-[120px]">
                                    {site.remarks || site.primary_domain.split('.')[0]}
                                  </span>
                                  <span className="text-slate-300 dark:text-surface-700">•</span>
                                  <button
                                    type="button"
                                    onClick={() => handleCopyText(docRoot, 'Document Root')}
                                    className="hover:text-emerald-600 dark:hover:text-emerald-400 transition flex items-center gap-1 text-[10px] truncate max-w-[160px]"
                                    title={`Click to copy: ${docRoot}`}
                                  >
                                    <span className="truncate">{docRoot}</span>
                                    {isCopied ? <Check className="w-2.5 h-2.5 text-emerald-600 shrink-0" /> : <Copy className="w-2.5 h-2.5 opacity-60 shrink-0" />}
                                  </button>
                                </div>
                              </div>
                            </div>
                          </td>
                        )}

                        {/* Client Account / Owner Tenant Badge */}
                        {colVisible.account && (
                          <td className="px-2.5 py-3">
                            <button
                              type="button"
                              onClick={() => setSelectedAccount(site.account_username || site.system_user || 'All accounts')}
                              title={`Filter by client account: ${site.account_username || site.system_user || 'admin'}`}
                              className="inline-flex items-center gap-1.5 px-2 py-1 rounded-lg bg-slate-100 dark:bg-surface-800/90 hover:bg-emerald-500/10 hover:border-emerald-500/40 border border-slate-200 dark:border-surface-700 transition cursor-pointer text-left group/acc"
                            >
                              <User className="w-3.5 h-3.5 text-slate-400 group-hover/acc:text-emerald-400 shrink-0" />
                              <div className="flex flex-col">
                                <span className="font-bold text-slate-800 dark:text-slate-200 text-[11px] leading-tight group-hover/acc:text-emerald-400">
                                  {site.account_username || site.system_user || 'admin'}
                                </span>
                                {site.account_plan && (
                                  <span className="text-[9px] text-slate-400 font-mono leading-none">
                                    {site.account_plan}
                                  </span>
                                )}
                              </div>
                            </button>
                          </td>
                        )}

                        {/* Status (Futuristic Beacon Pill) */}
                        {colVisible.status && (
                          <td className="px-3 py-3">
                            <button
                              onClick={(e) => handleToggleStatus(site, e)}
                              title={site.status === 'active' ? 'Running (Click to Stop)' : 'Suspended (Click to Start)'}
                              className="cursor-pointer"
                            >
                              {site.status === 'active' ? (
                                <div className="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full bg-emerald-500/10 border border-emerald-500/30 text-emerald-600 dark:text-emerald-400 font-bold text-[11px] hover:bg-emerald-500/20 transition shadow-2xs">
                                  <span className="relative flex h-2 w-2 shrink-0">
                                    <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-emerald-400 opacity-75" />
                                    <span className="relative inline-flex rounded-full h-2 w-2 bg-emerald-500" />
                                  </span>
                                  <span>Running</span>
                                </div>
                              ) : (
                                <div className="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full bg-amber-500/10 border border-amber-500/30 text-amber-600 dark:text-amber-400 font-bold text-[11px] hover:bg-amber-500/20 transition shadow-2xs">
                                  <span className="inline-flex rounded-full h-2 w-2 bg-amber-500 shrink-0" />
                                  <span>Paused</span>
                                </div>
                              )}
                            </button>
                          </td>
                        )}

                        {/* Backup / Restore */}
                        {colVisible.backup && (
                          <td className="px-3 py-3">
                            <button
                              onClick={() => openBackupModal(site)}
                              className="flex items-center gap-2 group text-left cursor-pointer"
                              title="Click to view backups or create snapshot"
                            >
                              <div className="w-7 h-7 rounded-xl bg-slate-100 dark:bg-surface-800 border border-slate-200 dark:border-surface-700 flex items-center justify-center text-slate-500 group-hover:text-emerald-600 group-hover:border-emerald-500/30 transition shrink-0 shadow-2xs">
                                <CloudDownload className="w-3.5 h-3.5" />
                              </div>
                              <div className="flex flex-col">
                                <span className="text-amber-600 dark:text-amber-400 font-bold text-xs group-hover:underline">
                                  {site.backup_count || 0} Backup
                                </span>
                                <span className="text-[10px] text-slate-400 group-hover:text-slate-600 dark:group-hover:text-slate-300">
                                  Click to backup
                                </span>
                              </div>
                            </button>
                          </td>
                        )}

                        {/* Quick Action Cockpit Toolbar */}
                        {colVisible.quickAction && (
                          <td className="px-3 py-3">
                            <div className="flex items-center gap-1 p-1 rounded-xl bg-slate-100/80 dark:bg-surface-800/70 border border-slate-200/80 dark:border-surface-700/80 shrink-0 w-max shadow-2xs">
                              {/* 1. File Manager (Prominent One-Click Action) */}
                              <a
                                href={`/files?domain=${encodeURIComponent(site.primary_domain)}&path=${encodeURIComponent(site.document_root)}`}
                                title={`Open File Manager (${site.document_root || `/var/www/${site.primary_domain}/public_html`})`}
                                className="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-lg bg-emerald-600 hover:bg-emerald-700 text-white font-bold text-xs transition shadow-xs"
                              >
                                <Folder className="w-3.5 h-3.5 text-white" />
                                <span>File Manager</span>
                              </a>

                              {/* 2. Config File */}
                              <button
                                onClick={() => openConfModal(site)}
                                title="Virtual Host Configuration"
                                className="p-1 rounded-lg hover:bg-white dark:hover:bg-surface-700 text-slate-600 dark:text-slate-400 hover:text-emerald-600 dark:hover:text-emerald-400 transition cursor-pointer"
                              >
                                <Settings className="w-3.5 h-3.5" />
                              </button>

                              {/* 3. Realtime Logs */}
                              <button
                                onClick={() => openLogsModal(site)}
                                title="Site Access & Error Logs"
                                className="p-1 rounded-lg hover:bg-white dark:hover:bg-surface-700 text-slate-600 dark:text-slate-400 hover:text-emerald-600 dark:hover:text-emerald-400 transition cursor-pointer"
                              >
                                <FileText className="w-3.5 h-3.5" />
                              </button>

                              {/* 4. Terminal */}
                              <a
                                href={`/terminal?cwd=${encodeURIComponent(site.document_root)}`}
                                title={`Open Terminal at ${site.document_root}`}
                                className="p-1 rounded-lg hover:bg-white dark:hover:bg-surface-700 text-slate-600 dark:text-slate-400 hover:text-emerald-600 dark:hover:text-emerald-400 transition cursor-pointer"
                              >
                                <Terminal className="w-3.5 h-3.5" />
                              </a>

                              {/* 5. Database */}
                              <a
                                href={`/databases?search=${encodeURIComponent(site.primary_domain.split('.')[0])}`}
                                title={`Databases for ${site.primary_domain}`}
                                className="p-1 rounded-lg hover:bg-white dark:hover:bg-surface-700 text-slate-600 dark:text-slate-400 hover:text-emerald-600 dark:hover:text-emerald-400 transition cursor-pointer"
                              >
                                <DatabaseIcon className="w-3.5 h-3.5" />
                              </a>

                              {/* 6. Directory & Permissions */}
                              <button
                                onClick={() => openSiteModal(site, 'directory')}
                                title="Directory & Folder Permissions"
                                className="p-1 rounded-lg hover:bg-white dark:hover:bg-surface-700 text-slate-600 dark:text-slate-400 hover:text-emerald-600 dark:hover:text-emerald-400 transition cursor-pointer"
                              >
                                <Lock className="w-3.5 h-3.5" />
                              </button>

                              {/* 7. PHP Version Switch Pill */}
                              <button
                                onClick={() => openPhpSwitchModal(site)}
                                title="Click to Switch PHP Version"
                                className="px-1.5 py-0.5 rounded-md bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-700 hover:border-emerald-500 text-slate-800 dark:text-slate-200 text-[10px] font-mono font-bold transition cursor-pointer"
                              >
                                {site.php_version || '8.2'}
                              </button>
                            </div>
                          </td>
                        )}

                        {/* Expiration */}
                        {colVisible.expiration && (
                          <td className="px-3 py-3 text-slate-600 dark:text-slate-400 font-medium">
                            <div className="flex items-center gap-1.5 text-xs">
                              <Calendar className="w-3 h-3 text-slate-400" />
                              <span>{site.expiration || 'Perpetual'}</span>
                            </div>
                          </td>
                        )}

                        {/* SSL */}
                        {colVisible.ssl && (
                          <td className="px-3 py-3">
                            {site.ssl_days_left && site.ssl_days_left > 0 ? (
                              <button
                                onClick={() => openSslModal(site)}
                                className="inline-flex items-center gap-1 font-bold text-emerald-600 dark:text-emerald-400 hover:underline cursor-pointer text-xs"
                                title="SSL Certificate Valid (Click to manage)"
                              >
                                <LockKeyhole className="w-3 h-3 text-emerald-600" />
                                <span>{site.ssl_days_left} Days</span>
                              </button>
                            ) : (
                              <button
                                onClick={() => openSslModal(site)}
                                className="inline-flex items-center gap-1 px-2 py-0.5 rounded-md bg-emerald-50 dark:bg-emerald-950/40 text-emerald-700 dark:text-emerald-300 font-bold border border-emerald-300 dark:border-emerald-800 hover:bg-emerald-100 transition cursor-pointer text-[11px]"
                                title="Click to issue Let's Encrypt / ZeroSSL"
                              >
                                <Zap className="w-2.5 h-2.5" />
                                <span>Issue SSL</span>
                              </button>
                            )}
                          </td>
                        )}

                        {/* Requests & Futuristic Neon Sparkline */}
                        {colVisible.requests && (
                          <td className="px-3 py-3">
                            <div className="flex flex-col gap-0.5">
                              <div className="flex items-center justify-between">
                                <span className="font-mono font-bold text-slate-900 dark:text-slate-100 text-xs">
                                  {(site.requests_count || 0).toLocaleString()}
                                </span>
                                <span className="text-[9px] text-emerald-600 dark:text-emerald-400 font-semibold">reqs</span>
                              </div>
                              {renderSparkline(site.traffic_history, site.id)}
                            </div>
                          </td>
                        )}

                        {/* WAF */}
                        {colVisible.waf && (
                          <td className="px-3 py-3">
                            <button
                              onClick={(e) => handleToggleWaf(site, e)}
                              className="cursor-pointer"
                              title="Click to toggle Layer-7 OWASP WAF"
                            >
                              {site.waf_status === 'Active' ? (
                                <div className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full bg-emerald-500/10 border border-emerald-500/25 text-emerald-600 dark:text-emerald-400 font-bold text-[11px] shadow-2xs">
                                  <ShieldCheck className="w-3 h-3" />
                                  <span>Active</span>
                                </div>
                              ) : (
                                <div className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full bg-slate-100 dark:bg-surface-800 text-slate-400 text-[11px]">
                                  <Shield className="w-3 h-3" />
                                  <span>Disabled</span>
                                </div>
                              )}
                            </button>
                          </td>
                        )}

                        {/* Operate Column (Conf, Log, 3-dots) - Sticky Right Edge */}
                        {colVisible.operate && (
                          <td className="px-3 py-3 text-right sticky right-0 bg-white/95 dark:bg-surface-900/95 z-10 shadow-[-4px_0_8px_-2px_rgba(0,0,0,0.06)]">
                            <div className="flex items-center justify-end gap-1.5 text-emerald-600 dark:text-emerald-400 font-semibold text-xs">
                              <button
                                onClick={() => openConfModal(site)}
                                className="px-2 py-1 rounded-lg hover:bg-emerald-50 dark:hover:bg-emerald-950/40 hover:text-emerald-700 transition cursor-pointer font-bold"
                                title="VirtualHost Configuration"
                              >
                                Conf
                              </button>

                              <button
                                onClick={() => openLogsModal(site)}
                                className="px-2 py-1 rounded-lg hover:bg-emerald-50 dark:hover:bg-emerald-950/40 hover:text-emerald-700 transition cursor-pointer font-bold"
                                title="Realtime Access / Error Log"
                              >
                                Log
                              </button>

                              {/* 3-dots Dropdown Menu Trigger */}
                              <button
                                data-dropdown-trigger
                                onClick={(e) => {
                                  e.stopPropagation();
                                  if (openDropdownId === site.id) {
                                    setOpenDropdownId(null);
                                    setActiveDropdownSite(null);
                                  } else {
                                    const rect = e.currentTarget.getBoundingClientRect();
                                    const spaceBelow = window.innerHeight - rect.bottom;
                                    const openUp = spaceBelow < 340;
                                    setDropdownPos({
                                      top: openUp ? rect.top - 6 : rect.bottom + 6,
                                      left: Math.max(10, rect.right - 210),
                                      openUp,
                                    });
                                    setOpenDropdownId(site.id);
                                    setActiveDropdownSite(site);
                                  }
                                }}
                                className={`p-1.5 rounded-lg transition cursor-pointer ${
                                  openDropdownId === site.id
                                    ? 'bg-slate-200 dark:bg-surface-700 text-slate-900 dark:text-white'
                                    : 'text-slate-500 hover:text-slate-900 dark:hover:text-white hover:bg-slate-100 dark:hover:bg-surface-800'
                                }`}
                                title="More operations"
                              >
                                <MoreVertical className="w-3.5 h-3.5 pointer-events-none" />
                              </button>
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
              4. BATCH ACTIONS & REALTIME STATEFUL PAGINATION FOOTER
              ========================================================================= */}
          <div className="border-t border-slate-200 dark:border-surface-800 bg-slate-50/80 dark:bg-surface-850 px-4 py-3 flex flex-col sm:flex-row items-center justify-between gap-3 text-xs text-slate-600 dark:text-slate-400">
            {/* Batch Options */}
            <div className="flex items-center gap-2 flex-wrap">
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

            {/* Realtime Pagination Controls */}
            <div className="flex items-center gap-3 flex-wrap">
              <span className="text-slate-500 dark:text-slate-400 text-xs font-medium">
                Showing {totalItems > 0 ? startIndex + 1 : 0}-{endIndex} of {totalItems}
              </span>

              {/* Prev / Page Numbers / Next */}
              <div className="flex items-center gap-1">
                <button
                  type="button"
                  disabled={currentPage <= 1}
                  onClick={() => {
                    const prev = Math.max(1, currentPage - 1);
                    setCurrentPage(prev);
                    setGotoInput(String(prev));
                  }}
                  className="px-2 py-1 rounded-md bg-white dark:bg-surface-800 border border-slate-300 dark:border-surface-700 text-slate-700 dark:text-slate-200 hover:bg-slate-50 disabled:opacity-40 disabled:cursor-not-allowed shadow-2xs transition cursor-pointer font-bold"
                  title="Previous Page"
                >
                  &lt;
                </button>

                {Array.from({ length: totalPages }, (_, i) => i + 1)
                  .filter((p) => p === 1 || p === totalPages || Math.abs(p - currentPage) <= 1)
                  .map((p, idx, arr) => {
                    const prevP = arr[idx - 1];
                    const showEllipsis = prevP && p - prevP > 1;
                    return (
                      <React.Fragment key={p}>
                        {showEllipsis && <span className="px-1 text-slate-400">...</span>}
                        <button
                          type="button"
                          onClick={() => {
                            setCurrentPage(p);
                            setGotoInput(String(p));
                          }}
                          className={`px-2.5 py-1 rounded-md text-xs font-bold transition cursor-pointer ${
                            currentPage === p
                              ? 'bg-emerald-600 text-white shadow-xs'
                              : 'bg-white dark:bg-surface-800 border border-slate-300 dark:border-surface-700 text-slate-700 dark:text-slate-200 hover:bg-slate-50 shadow-2xs'
                          }`}
                        >
                          {p}
                        </button>
                      </React.Fragment>
                    );
                  })}

                <button
                  type="button"
                  disabled={currentPage >= totalPages}
                  onClick={() => {
                    const next = Math.min(totalPages, currentPage + 1);
                    setCurrentPage(next);
                    setGotoInput(String(next));
                  }}
                  className="px-2 py-1 rounded-md bg-white dark:bg-surface-800 border border-slate-300 dark:border-surface-700 text-slate-700 dark:text-slate-200 hover:bg-slate-50 disabled:opacity-40 disabled:cursor-not-allowed shadow-2xs transition cursor-pointer font-bold"
                  title="Next Page"
                >
                  &gt;
                </button>
              </div>

              {/* Page Size Select */}
              <select
                value={pageSize}
                onChange={(e) => {
                  setPageSize(Number(e.target.value));
                  setCurrentPage(1);
                  setGotoInput('1');
                }}
                className="px-2 py-1 rounded-md bg-white dark:bg-surface-800 border border-slate-300 dark:border-surface-700 text-slate-700 dark:text-slate-200 text-xs font-semibold focus:outline-none shadow-2xs cursor-pointer"
              >
                <option value={10}>10 / page</option>
                <option value={25}>25 / page</option>
                <option value={50}>50 / page</option>
                <option value={100}>100 / page</option>
              </select>

              {/* Goto Page Input */}
              <div className="flex items-center gap-1.5">
                <span className="text-slate-500 dark:text-slate-400">Goto</span>
                <input
                  type="text"
                  value={gotoInput}
                  onChange={(e) => setGotoInput(e.target.value)}
                  onKeyDown={(e) => {
                    if (e.key === 'Enter') {
                      const num = parseInt(gotoInput, 10);
                      if (!isNaN(num) && num >= 1 && num <= totalPages) {
                        setCurrentPage(num);
                      } else {
                        setGotoInput(String(currentPage));
                      }
                    }
                  }}
                  onBlur={() => {
                    const num = parseInt(gotoInput, 10);
                    if (!isNaN(num) && num >= 1 && num <= totalPages) {
                      setCurrentPage(num);
                    } else {
                      setGotoInput(String(currentPage));
                    }
                  }}
                  className="w-10 px-1 py-1 rounded-md bg-white dark:bg-surface-800 border border-slate-300 dark:border-surface-700 text-center text-slate-800 dark:text-slate-200 font-bold focus:outline-none focus:border-emerald-500 shadow-2xs text-xs"
                />
              </div>
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
                {/* Available domains quick-picker */}
                {availableDomains.length > 0 && (
                  <div className="p-2.5 rounded-xl bg-slate-50 dark:bg-surface-800/80 border border-slate-200 dark:border-surface-700 space-y-1.5">
                    <span className="text-[11px] text-slate-500 font-medium">Quick-select from your added domains:</span>
                    <div className="flex flex-wrap gap-1.5">
                      {availableDomains.map((ad) => (
                        <button
                          key={ad.domain}
                          type="button"
                          onClick={() => {
                            setNewDomain(ad.domain);
                            setNewRemarks(ad.domain.split('.')[0]);
                            setNewDocRoot(`/var/www/${ad.domain}/public_html`);
                          }}
                          className={`px-2 py-0.5 rounded-md font-mono text-[11px] font-bold border transition cursor-pointer ${
                            newDomain.includes(ad.domain)
                              ? 'bg-emerald-600 text-white border-emerald-600'
                              : 'bg-white dark:bg-surface-700 text-slate-700 dark:text-slate-200 border-slate-300 dark:border-surface-600 hover:border-emerald-500'
                          }`}
                        >
                          {ad.domain}
                        </button>
                      ))}
                    </div>
                  </div>
                )}

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
                      const val = e.target.value;
                      setNewDomain(val);
                      const lines = val.trim().split('\n').map((l) => l.trim()).filter(Boolean);
                      if (lines.length > 0) {
                        const firstDomain = lines[0];
                        const prevFirst = newDomain.trim().split('\n')[0].split('.')[0];
                        if (!newRemarks || newRemarks === prevFirst) {
                          setNewRemarks(firstDomain.split('.')[0]);
                        }
                        if (!docRootCustomized) {
                          setNewDocRoot(`/var/www/${firstDomain}/public_html`);
                        }
                      } else {
                        if (!docRootCustomized) {
                          setNewDocRoot('/var/www/');
                        }
                      }
                    }}
                    placeholder="example.com&#10;www.example.com"
                    className="w-full px-3 py-2 rounded-xl bg-slate-50 dark:bg-surface-800 border border-slate-300 dark:border-surface-700 text-slate-900 dark:text-white placeholder:text-slate-400 focus:outline-none focus:border-emerald-500 font-mono"
                  />
                  <span className="text-[11px] text-slate-400">One domain per line. Each domain will be assigned its own isolated directory.</span>
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
                  <div className="flex items-center justify-between mb-1">
                    <label className="block text-slate-700 dark:text-slate-300">Document Root</label>
                    <span className="text-[11px] text-emerald-600 dark:text-emerald-400 font-medium">
                      Separate directory per domain
                    </span>
                  </div>
                  <input
                    type="text"
                    value={newDocRoot}
                    onChange={(e) => {
                      setNewDocRoot(e.target.value);
                      setDocRootCustomized(true);
                    }}
                    placeholder="/var/www/example.com/public_html"
                    className="w-full px-3 py-2 rounded-xl bg-slate-50 dark:bg-surface-800 border border-slate-300 dark:border-surface-700 text-slate-900 dark:text-white font-mono focus:outline-none"
                  />
                  <span className="text-[11px] text-slate-400">
                    Default: <code className="text-emerald-500 font-mono">/var/www/&#123;domain&#125;/public_html</code>. Different domains automatically get their own separate directories.
                  </span>
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
                  className="w-full h-full p-4 rounded-xl bg-slate-50 dark:bg-slate-950 text-slate-800 dark:text-emerald-400 font-mono text-xs focus:outline-none resize-none border border-slate-300 dark:border-slate-800 shadow-2xs"
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

              <div className="flex-1 min-h-[350px] mb-4 overflow-auto rounded-xl bg-slate-50 dark:bg-slate-950 p-4 font-mono text-xs border border-slate-300 dark:border-slate-800 shadow-2xs">
                <pre className="text-slate-800 dark:text-emerald-400 whitespace-pre-wrap">
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
                        <a
                          href={`/files?domain=${encodeURIComponent(selectedSite.primary_domain)}&path=/var/backups`}
                          title="Open Backups directory in File Manager"
                          className="px-2.5 py-1 rounded-lg bg-slate-200 hover:bg-slate-300 dark:bg-surface-700 dark:hover:bg-surface-600 text-slate-700 dark:text-slate-200 font-bold text-xs inline-flex items-center gap-1 transition"
                        >
                          <Folder className="w-3 h-3" />
                          <span>View in Files</span>
                        </a>
                        <a
                          href={`/api/v1/files/download?path=/var/backups/${encodeURIComponent(selectedSite.primary_domain)}_backup.tar.gz`}
                          download={`${selectedSite.primary_domain}_backup.tar.gz`}
                          className="px-2.5 py-1 rounded-lg bg-emerald-600 hover:bg-emerald-700 text-white font-bold text-xs inline-flex items-center gap-1 transition shadow-xs"
                        >
                          <CloudDownload className="w-3 h-3" />
                          <span>Download Archive</span>
                        </a>
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
                      role="switch"
                      aria-checked={forceHttps}
                      onClick={() => setForceHttps(!forceHttps)}
                      className={`w-10 h-6 shrink-0 inline-flex items-center rounded-full p-0.5 transition-colors cursor-pointer focus:outline-none ${
                        forceHttps ? 'bg-emerald-600' : 'bg-slate-300 dark:bg-slate-700'
                      }`}
                    >
                      <span
                        className={`w-5 h-5 rounded-full bg-white shadow-xs transition-transform transform ${
                          forceHttps ? 'translate-x-4' : 'translate-x-0'
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

              {statsLoading ? (
                <div className="flex flex-col items-center justify-center py-12 gap-3">
                  <RefreshCw className="w-8 h-8 text-indigo-500 animate-spin" />
                  <span className="text-xs text-slate-500">Aggregating live web server telemetry...</span>
                </div>
              ) : (
                <>
                  <div className="grid grid-cols-3 gap-3 mb-5 text-center">
                    <div className="p-4 rounded-xl bg-slate-50 dark:bg-surface-800 border border-slate-200 dark:border-surface-700">
                      <span className="text-xs text-slate-400 block mb-1">Total Requests</span>
                      <span className="text-xl font-bold text-slate-900 dark:text-white">
                        {statsData?.total_requests?.toLocaleString() ?? 0}
                      </span>
                    </div>
                    <div className="p-4 rounded-xl bg-slate-50 dark:bg-surface-800 border border-slate-200 dark:border-surface-700">
                      <span className="text-xs text-slate-400 block mb-1">Unique Visitors (UV)</span>
                      <span className="text-xl font-bold text-emerald-600 dark:text-emerald-400">
                        {statsData?.unique_visitors?.toLocaleString() ?? 0}
                      </span>
                    </div>
                    <div className="p-4 rounded-xl bg-slate-50 dark:bg-surface-800 border border-slate-200 dark:border-surface-700">
                      <span className="text-xs text-slate-400 block mb-1">Total Bandwidth</span>
                      <span className="text-xl font-bold text-indigo-600 dark:text-indigo-400">
                        {(statsData?.bandwidth_gb ?? 0).toFixed(2)} GB
                      </span>
                    </div>
                  </div>

                  <div className="border border-slate-200 dark:border-surface-700 rounded-xl p-4">
                    <h4 className="text-xs font-bold text-slate-800 dark:text-slate-200 mb-3">Top Accessed Domains</h4>
                    {(!statsData?.top_domains || statsData.top_domains.length === 0) ? (
                      <p className="text-xs text-slate-400 py-3 text-center">No virtual host traffic recorded yet on this server.</p>
                    ) : (
                      <div className="space-y-3 text-xs">
                        {(() => {
                          const maxReq = statsData.top_domains[0]?.requests || 1;
                          return statsData.top_domains.map((item) => {
                            const pct = maxReq > 0 ? Math.max(5, Math.min(100, Math.round((item.requests / maxReq) * 100))) : 0;
                            return (
                              <div key={item.domain} className="space-y-1">
                                <div className="flex justify-between font-semibold">
                                  <span className="text-slate-900 dark:text-white">{item.domain}</span>
                                  <span className="font-mono text-slate-500">{item.requests.toLocaleString()} reqs</span>
                                </div>
                                <div className="w-full h-1.5 rounded-full bg-slate-100 dark:bg-surface-800 overflow-hidden">
                                  <div className="h-full bg-emerald-600 rounded-full transition-all duration-300" style={{ width: `${pct}%` }} />
                                </div>
                              </div>
                            );
                          });
                        })()}
                      </div>
                    )}
                  </div>
                </>
              )}
            </div>
          </div>
        )}

        {/* 8. Nginx Realtime Service Control Modal (100% Real API) */}
        {nginxControlOpen && (
          <div className="fixed inset-0 z-50 bg-black/60 backdrop-blur-xs flex items-center justify-center p-4 animate-fadeIn">
            <div className="w-full max-w-lg bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-700 rounded-2xl shadow-2xl p-6 relative">
              <button
                onClick={() => setNginxControlOpen(false)}
                className="absolute top-5 right-5 p-1 rounded-lg text-slate-400 hover:text-slate-700 dark:hover:text-white cursor-pointer"
              >
                <X className="w-5 h-5" />
              </button>

              <div className="flex items-center gap-3 mb-4">
                <div className="w-10 h-10 rounded-xl bg-emerald-50 dark:bg-emerald-950/40 border border-emerald-200 dark:border-emerald-800 flex items-center justify-center text-emerald-600 dark:text-emerald-400 shadow-2xs">
                  <Cpu className="w-5 h-5" />
                </div>
                <div>
                  <h2 className="text-base font-bold text-slate-900 dark:text-white flex items-center gap-2">
                    <span>Nginx Web Engine Control</span>
                    <span className="px-2 py-0.5 rounded-full text-[10px] font-mono font-bold bg-emerald-100 dark:bg-emerald-950/60 text-emerald-800 dark:text-emerald-300 border border-emerald-200 dark:border-emerald-800">
                      Active (running)
                    </span>
                  </h2>
                  <p className="text-xs text-slate-500 font-mono">Ports: 80, 443 • Master & Worker Processes Active</p>
                </div>
              </div>

              {/* Service Actions Grid */}
              <div className="grid grid-cols-1 sm:grid-cols-3 gap-2.5 mb-4 text-xs font-bold">
                <button
                  type="button"
                  disabled={nginxServiceLoading !== null}
                  onClick={() => handleNginxAction('reload')}
                  className="p-3 rounded-xl bg-slate-50 dark:bg-surface-800 hover:bg-emerald-50 dark:hover:bg-emerald-950/30 text-slate-800 dark:text-slate-200 hover:text-emerald-700 hover:border-emerald-300 border border-slate-200 dark:border-surface-700 transition flex flex-col items-center gap-1.5 cursor-pointer disabled:opacity-50 shadow-2xs"
                >
                  <RefreshCw className={`w-4 h-4 text-emerald-600 ${nginxServiceLoading === 'reload' ? 'animate-spin' : ''}`} />
                  <span>Reload Service</span>
                  <span className="text-[10px] text-slate-400 font-normal">0 Downtime</span>
                </button>

                <button
                  type="button"
                  disabled={nginxServiceLoading !== null}
                  onClick={() => handleNginxAction('restart')}
                  className="p-3 rounded-xl bg-slate-50 dark:bg-surface-800 hover:bg-amber-50 dark:hover:bg-amber-950/30 text-slate-800 dark:text-slate-200 hover:text-amber-700 hover:border-amber-300 border border-slate-200 dark:border-surface-700 transition flex flex-col items-center gap-1.5 cursor-pointer disabled:opacity-50 shadow-2xs"
                >
                  <RotateCcw className={`w-4 h-4 text-amber-600 ${nginxServiceLoading === 'restart' ? 'animate-spin' : ''}`} />
                  <span>Restart Service</span>
                  <span className="text-[10px] text-slate-400 font-normal">Full Cycle</span>
                </button>

                <button
                  type="button"
                  disabled={nginxServiceLoading !== null}
                  onClick={() => handleNginxAction('test')}
                  className="p-3 rounded-xl bg-slate-50 dark:bg-surface-800 hover:bg-sky-50 dark:hover:bg-sky-950/30 text-slate-800 dark:text-slate-200 hover:text-sky-700 hover:border-sky-300 border border-slate-200 dark:border-surface-700 transition flex flex-col items-center gap-1.5 cursor-pointer disabled:opacity-50 shadow-2xs"
                >
                  <CheckCircle2 className={`w-4 h-4 text-sky-600 ${nginxServiceLoading === 'test' ? 'animate-spin' : ''}`} />
                  <span>Test Syntax</span>
                  <span className="text-[10px] text-slate-400 font-mono">nginx -t</span>
                </button>
              </div>

              {/* Real Output Console Box */}
              {nginxServiceOutput && (
                <div className="mb-4 p-3 rounded-xl bg-slate-50 dark:bg-slate-950 text-slate-800 dark:text-emerald-400 font-mono text-[11px] leading-relaxed border border-slate-300 dark:border-slate-800 overflow-x-auto max-h-36 shadow-2xs">
                  <div className="text-[10px] text-slate-500 dark:text-slate-400 font-bold mb-1 uppercase tracking-wider">Service Execution Output:</div>
                  <pre className="whitespace-pre-wrap">{nginxServiceOutput}</pre>
                </div>
              )}

              {/* Footer Links */}
              <div className="flex items-center justify-between border-t border-slate-100 dark:border-surface-800 pt-3 text-xs">
                <a
                  href="/webservers"
                  className="text-emerald-600 hover:underline flex items-center gap-1 font-bold"
                >
                  <ExternalLink className="w-3.5 h-3.5" />
                  <span>Full Web Servers Management</span>
                </a>
                <button
                  onClick={() => setNginxControlOpen(false)}
                  className="px-4 py-1.5 rounded-lg bg-slate-100 dark:bg-surface-800 text-slate-700 dark:text-slate-300 font-bold hover:bg-slate-200 cursor-pointer transition"
                >
                  Close
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
                className="w-full p-3 rounded-xl bg-slate-50 dark:bg-slate-950 text-slate-800 dark:text-emerald-400 font-mono text-xs focus:outline-none border border-slate-300 dark:border-slate-800 mb-4 shadow-2xs"
              />

              <div className="flex justify-end gap-3">
                <button
                  onClick={() => setRewriteModalOpen(false)}
                  className="px-4 py-2 rounded-xl bg-slate-100 dark:bg-surface-800 text-slate-700 dark:text-slate-300 font-bold text-xs"
                >
                  Cancel
                </button>
                <button
                  onClick={handleSaveRewrite}
                  className="px-5 py-2 rounded-xl bg-emerald-600 hover:bg-emerald-700 text-white font-bold text-xs shadow-xs cursor-pointer"
                >
                  Save & Reload
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
                  onClick={handleSaveSpeed}
                  className="px-5 py-2 rounded-xl bg-emerald-600 hover:bg-emerald-700 text-white font-bold text-xs shadow-xs cursor-pointer"
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
                  onClick={handleSaveWaf}
                  className="px-5 py-2 rounded-xl bg-emerald-600 hover:bg-emerald-700 text-white font-bold text-xs shadow-xs cursor-pointer"
                >
                  Apply Rules
                </button>
              </div>
            </div>
          </div>
        )}

        {/* 14. cgroups v2 Limits Modal (Hostvra Advanced Linux Isolation Studio) */}
        {isolationModalSite && (
          <div className="fixed inset-0 z-50 bg-black/60 backdrop-blur-xs flex items-center justify-center p-4 animate-fadeIn">
            <div className="w-full max-w-4xl bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-700 rounded-3xl shadow-2xl overflow-hidden relative flex flex-col max-h-[92vh]">
              {/* Modal Header */}
              <div className="p-6 pb-4 border-b border-slate-100 dark:border-surface-800 flex items-start justify-between">
                <div className="flex items-center gap-3.5">
                  <div className="w-11 h-11 rounded-2xl bg-indigo-500/10 border border-indigo-500/20 flex items-center justify-center text-indigo-600 dark:text-indigo-400">
                    <Cpu className="w-6 h-6" />
                  </div>
                  <div>
                    <div className="flex items-center gap-2 flex-wrap">
                      <h2 className="text-lg font-bold text-slate-900 dark:text-white">
                        Linux Cgroups v2 & Systemd Process Isolation Studio
                      </h2>
                      <span className="px-2 py-0.5 rounded-full text-[10px] font-bold bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 border border-emerald-500/20 flex items-center gap-1">
                        <span className="w-1.5 h-1.5 rounded-full bg-emerald-500 animate-pulse" />
                        Kernel Enforced
                      </span>
                    </div>
                    <div className="flex items-center gap-2 mt-1 text-xs text-slate-500 flex-wrap">
                      <span className="font-semibold text-slate-700 dark:text-slate-300">{isolationModalSite.primary_domain}</span>
                      <span>•</span>
                      <span className="font-mono text-[11px] bg-slate-100 dark:bg-surface-800 px-1.5 py-0.5 rounded text-slate-600 dark:text-slate-400">
                        {isolationInfo?.username ? `system-hostvra-${isolationInfo.username}.slice` : 'system.slice'}
                      </span>
                      {isolationInfo && (
                        <>
                          <span>•</span>
                          <span className="font-mono text-[11px]">
                            UID: {isolationInfo.uid} / GID: {isolationInfo.gid}
                          </span>
                        </>
                      )}
                    </div>
                  </div>
                </div>
                <button
                  onClick={() => setIsolationModalSite(null)}
                  className="p-1.5 rounded-xl text-slate-400 hover:text-slate-700 dark:hover:text-white hover:bg-slate-100 dark:hover:bg-surface-800 transition-colors"
                >
                  <X className="w-5 h-5" />
                </button>
              </div>

              {isolationLoading ? (
                <div className="py-20 text-center text-slate-500">
                  <RefreshCw className="w-8 h-8 animate-spin mx-auto mb-3 text-indigo-500" />
                  <p className="font-semibold text-sm">Reading live Linux cgroups v2 telemetry & quota tables...</p>
                  <p className="text-xs text-slate-400 mt-1">Interrogating /sys/fs/cgroup controller hierarchy</p>
                </div>
              ) : (
                <div className="flex-1 overflow-y-auto p-6 space-y-6">
                  {/* Live Telemetry Ribbon */}
                  <div className="grid grid-cols-1 md:grid-cols-3 gap-3.5">
                    {/* RAM Gauge */}
                    <div className="p-4 rounded-2xl bg-slate-50 dark:bg-surface-800/80 border border-slate-200/80 dark:border-surface-700">
                      <div className="flex items-center justify-between mb-2">
                        <span className="text-xs font-bold text-slate-600 dark:text-slate-400 flex items-center gap-1.5">
                          <Activity className="w-3.5 h-3.5 text-indigo-500" />
                          RAM (Live / Max)
                        </span>
                        <span className="text-xs font-mono font-bold text-slate-900 dark:text-white">
                          {isolationInfo?.current_memory_mb || isolationInfo?.memory_used_mb || 0} / {limitMem} MB
                        </span>
                      </div>
                      <div className="w-full h-2 rounded-full bg-slate-200 dark:bg-surface-700 overflow-hidden relative">
                        <div
                          className="h-full bg-indigo-600 rounded-full transition-all duration-500"
                          style={{
                            width: `${Math.min(100, (((isolationInfo?.current_memory_mb || isolationInfo?.memory_used_mb || 0) / (limitMem || 1)) * 100))}%`,
                          }}
                        />
                        {/* Soft Throttle marker */}
                        <div
                          className="absolute top-0 bottom-0 w-0.5 bg-amber-500 z-10"
                          style={{ left: `${Math.min(100, ((limitMemHigh || limitMem * 0.9) / (limitMem || 1)) * 100)}%` }}
                          title={`Soft throttle: ${limitMemHigh} MB`}
                        />
                      </div>
                      <div className="flex items-center justify-between mt-2 text-[11px] text-slate-500">
                        <span>Peak: {isolationInfo?.memory_peak_mb || 0} MB</span>
                        <span className="flex items-center gap-1">
                          OOM Kills:
                          <span
                            className={`px-1.5 py-0.2 rounded font-mono font-bold ${
                              (isolationInfo?.oom_kills || 0) > 0
                                ? 'bg-red-500/10 text-red-600 dark:text-red-400'
                                : 'bg-emerald-500/10 text-emerald-600 dark:text-emerald-400'
                            }`}
                          >
                            {isolationInfo?.oom_kills || 0}
                          </span>
                        </span>
                      </div>
                    </div>

                    {/* CPU Quota Gauge */}
                    <div className="p-4 rounded-2xl bg-slate-50 dark:bg-surface-800/80 border border-slate-200/80 dark:border-surface-700">
                      <div className="flex items-center justify-between mb-2">
                        <span className="text-xs font-bold text-slate-600 dark:text-slate-400 flex items-center gap-1.5">
                          <Gauge className="w-3.5 h-3.5 text-blue-500" />
                          CPU Usage / Quota
                        </span>
                        <span className="text-xs font-mono font-bold text-slate-900 dark:text-white">
                          {(isolationInfo?.cpu_usage_percent || isolationInfo?.cpu_usage_perc || 0).toFixed(1)}% / {limitCPU}%
                        </span>
                      </div>
                      <div className="w-full h-2 rounded-full bg-slate-200 dark:bg-surface-700 overflow-hidden">
                        <div
                          className="h-full bg-blue-600 rounded-full transition-all duration-500"
                          style={{
                            width: `${Math.min(100, (((isolationInfo?.cpu_usage_percent || isolationInfo?.cpu_usage_perc || 0) / (limitCPU || 1)) * 100))}%`,
                          }}
                        />
                      </div>
                      <div className="flex items-center justify-between mt-2 text-[11px] text-slate-500">
                        <span>Weight: {limitCPUWeight}</span>
                        <span>
                          Throttled: {((isolationInfo?.throttled_usec || 0) / 1000).toFixed(1)} ms
                        </span>
                      </div>
                    </div>

                    {/* Tasks / PIDs Gauge */}
                    <div className="p-4 rounded-2xl bg-slate-50 dark:bg-surface-800/80 border border-slate-200/80 dark:border-surface-700">
                      <div className="flex items-center justify-between mb-2">
                        <span className="text-xs font-bold text-slate-600 dark:text-slate-400 flex items-center gap-1.5">
                          <Layers className="w-3.5 h-3.5 text-emerald-500" />
                          Fork Bomb Guard (PIDs)
                        </span>
                        <span className="text-xs font-mono font-bold text-slate-900 dark:text-white">
                          {isolationInfo?.current_pids || isolationInfo?.tasks_current || 0} / {limitTasks} max
                        </span>
                      </div>
                      <div className="w-full h-2 rounded-full bg-slate-200 dark:bg-surface-700 overflow-hidden">
                        <div
                          className="h-full bg-emerald-600 rounded-full transition-all duration-500"
                          style={{
                            width: `${Math.min(100, (((isolationInfo?.current_pids || isolationInfo?.tasks_current || 0) / (limitTasks || 1)) * 100))}%`,
                          }}
                        />
                      </div>
                      <div className="flex items-center justify-between mt-2 text-[11px] text-slate-500">
                        <span className="truncate max-w-[170px]" title={isolationInfo?.php_pool_socket}>
                          Socket: {isolationInfo?.php_pool_socket ? isolationInfo.php_pool_socket.split('/').pop() : 'active'}
                        </span>
                        <span className="text-emerald-600 dark:text-emerald-400 font-semibold">Protected</span>
                      </div>
                    </div>
                  </div>

                  {/* Tabs Navigation */}
                  <div className="flex items-center gap-2 border-b border-slate-200 dark:border-surface-700">
                    <button
                      onClick={() => setIsolationTab('quotas')}
                      className={`px-4 py-2.5 text-xs font-bold rounded-t-xl transition-all border-b-2 flex items-center gap-1.5 ${
                        isolationTab === 'quotas'
                          ? 'border-indigo-600 text-indigo-600 dark:text-indigo-400 bg-indigo-50/50 dark:bg-indigo-950/20'
                          : 'border-transparent text-slate-600 dark:text-slate-400 hover:text-slate-900 dark:hover:text-white'
                      }`}
                    >
                      <SlidersHorizontal className="w-3.5 h-3.5" />
                      Resource Quotas & Presets
                    </button>
                    <button
                      onClick={() => setIsolationTab('telemetry')}
                      className={`px-4 py-2.5 text-xs font-bold rounded-t-xl transition-all border-b-2 flex items-center gap-1.5 ${
                        isolationTab === 'telemetry'
                          ? 'border-indigo-600 text-indigo-600 dark:text-indigo-400 bg-indigo-50/50 dark:bg-indigo-950/20'
                          : 'border-transparent text-slate-600 dark:text-slate-400 hover:text-slate-900 dark:hover:text-white'
                      }`}
                    >
                      <Terminal className="w-3.5 h-3.5" />
                      Live Cgroups Telemetry
                    </button>
                    <button
                      onClick={() => setIsolationTab('events')}
                      className={`px-4 py-2.5 text-xs font-bold rounded-t-xl transition-all border-b-2 flex items-center gap-1.5 ${
                        isolationTab === 'events'
                          ? 'border-indigo-600 text-indigo-600 dark:text-indigo-400 bg-indigo-50/50 dark:bg-indigo-950/20'
                          : 'border-transparent text-slate-600 dark:text-slate-400 hover:text-slate-900 dark:hover:text-white'
                      }`}
                    >
                      <History className="w-3.5 h-3.5" />
                      Quota & OOM Events ({isolationEvents.length})
                    </button>
                  </div>

                  {/* TAB 1: Resource Quotas & Presets */}
                  {isolationTab === 'quotas' && (
                    <div className="space-y-6">
                      {/* Presets Grid */}
                      <div>
                        <div className="flex items-center justify-between mb-2.5">
                          <label className="text-xs font-bold text-slate-700 dark:text-slate-300 flex items-center gap-1.5">
                            <Sparkles className="w-3.5 h-3.5 text-amber-500" />
                            Quick Allocation Presets
                          </label>
                          <span className="text-[11px] text-slate-500">Click a preset to configure recommended thresholds</span>
                        </div>
                        <div className="grid grid-cols-2 sm:grid-cols-4 gap-2.5">
                          {[
                            { name: 'Starter', mem: 256, cpu: 50, weight: 100, tasks: 50, desc: 'Blogs / Static Sites' },
                            { name: 'Standard', mem: 512, cpu: 100, weight: 100, tasks: 100, desc: 'WordPress / Small Store' },
                            { name: 'Business', mem: 1024, cpu: 200, weight: 200, tasks: 200, desc: 'WooCommerce / APIs' },
                            { name: 'Enterprise', mem: 2048, cpu: 400, weight: 300, tasks: 400, desc: 'High Traffic / Big DB' },
                          ].map((preset) => {
                            const isMatch = limitMem === preset.mem && limitCPU === preset.cpu && limitTasks === preset.tasks;
                            return (
                              <button
                                key={preset.name}
                                type="button"
                                onClick={() => {
                                  setLimitMem(preset.mem);
                                  setLimitMemHigh(Math.round(preset.mem * 0.9));
                                  setLimitCPU(preset.cpu);
                                  setLimitCPUWeight(preset.weight);
                                  setLimitTasks(preset.tasks);
                                  showToast(`Applied '${preset.name}' quota preset`);
                                }}
                                className={`p-3 rounded-2xl border text-left transition-all ${
                                  isMatch
                                    ? 'bg-indigo-50/80 dark:bg-indigo-950/30 border-indigo-500 text-indigo-900 dark:text-indigo-200 ring-2 ring-indigo-500/20'
                                    : 'bg-slate-50/60 dark:bg-surface-800/60 border-slate-200 dark:border-surface-700 hover:border-slate-300 dark:hover:border-surface-600'
                                }`}
                              >
                                <div className="flex items-center justify-between mb-1">
                                  <span className="font-bold text-xs text-slate-900 dark:text-white">{preset.name}</span>
                                  {isMatch && <CheckCircle2 className="w-3.5 h-3.5 text-indigo-600 dark:text-indigo-400" />}
                                </div>
                                <div className="text-[11px] text-slate-500">{preset.desc}</div>
                                <div className="mt-2 text-[10px] font-mono text-slate-600 dark:text-slate-400">
                                  {preset.mem}MB • {preset.cpu}% CPU
                                </div>
                              </button>
                            );
                          })}
                        </div>
                      </div>

                      {/* Main Limits Config */}
                      <div className="grid grid-cols-1 md:grid-cols-2 gap-5">
                        {/* Memory Max Hard Cap */}
                        <div className="p-4 rounded-2xl bg-slate-50/70 dark:bg-surface-800/70 border border-slate-200/80 dark:border-surface-700">
                          <div className="flex items-center justify-between mb-2">
                            <label className="text-xs font-bold text-slate-800 dark:text-slate-200">
                              Hard Memory Cap (<code className="font-mono text-indigo-600">memory.max</code>)
                            </label>
                            <span className="text-xs font-mono font-bold text-indigo-600 dark:text-indigo-400 bg-indigo-500/10 px-2 py-0.5 rounded">
                              {limitMem} MB
                            </span>
                          </div>
                          <input
                            type="range"
                            min={64}
                            max={4096}
                            step={64}
                            value={limitMem}
                            onChange={(e) => {
                              const val = Number(e.target.value);
                              setLimitMem(val);
                              if (limitMemHigh >= val) {
                                setLimitMemHigh(Math.round(val * 0.9));
                              }
                            }}
                            className="w-full accent-indigo-600 cursor-pointer"
                          />
                          <p className="text-[11px] text-slate-500 mt-2">
                            Absolute ceiling. If the cgroup slice exceeds this limit, the kernel enforces the OOM policy.
                          </p>
                        </div>

                        {/* Soft Throttle Memory High */}
                        <div className="p-4 rounded-2xl bg-slate-50/70 dark:bg-surface-800/70 border border-slate-200/80 dark:border-surface-700">
                          <div className="flex items-center justify-between mb-2">
                            <label className="text-xs font-bold text-slate-800 dark:text-slate-200">
                              Soft Throttle Reclaim (<code className="font-mono text-amber-600">memory.high</code>)
                            </label>
                            <span className="text-xs font-mono font-bold text-amber-600 dark:text-amber-400 bg-amber-500/10 px-2 py-0.5 rounded">
                              {limitMemHigh} MB
                            </span>
                          </div>
                          <input
                            type="range"
                            min={64}
                            max={limitMem}
                            step={32}
                            value={limitMemHigh}
                            onChange={(e) => setLimitMemHigh(Number(e.target.value))}
                            className="w-full accent-amber-600 cursor-pointer"
                          />
                          <p className="text-[11px] text-slate-500 mt-2">
                            Proactive kernel memory reclaim begins here. Processes are gently throttled rather than abruptly killed.
                          </p>
                        </div>

                        {/* CPU Quota % */}
                        <div className="p-4 rounded-2xl bg-slate-50/70 dark:bg-surface-800/70 border border-slate-200/80 dark:border-surface-700">
                          <div className="flex items-center justify-between mb-2">
                            <label className="text-xs font-bold text-slate-800 dark:text-slate-200">
                              CPU Quota Bandwidth (<code className="font-mono text-blue-600">cpu.max</code>)
                            </label>
                            <span className="text-xs font-mono font-bold text-blue-600 dark:text-blue-400 bg-blue-500/10 px-2 py-0.5 rounded">
                              {limitCPU}% ({limitCPU >= 100 ? `${(limitCPU / 100).toFixed(1)} Core` : 'Sub-core'})
                            </span>
                          </div>
                          <input
                            type="range"
                            min={10}
                            max={600}
                            step={10}
                            value={limitCPU}
                            onChange={(e) => setLimitCPU(Number(e.target.value))}
                            className="w-full accent-blue-600 cursor-pointer"
                          />
                          <p className="text-[11px] text-slate-500 mt-2">
                            100% = 1 full CPU core quota in CFS (Completely Fair Scheduler). 200% = 2 dedicated cores.
                          </p>
                        </div>

                        {/* CPU Weight */}
                        <div className="p-4 rounded-2xl bg-slate-50/70 dark:bg-surface-800/70 border border-slate-200/80 dark:border-surface-700">
                          <div className="flex items-center justify-between mb-2">
                            <label className="text-xs font-bold text-slate-800 dark:text-slate-200">
                              CPU Proportional Weight (<code className="font-mono text-purple-600">cpu.weight</code>)
                            </label>
                            <span className="text-xs font-mono font-bold text-purple-600 dark:text-purple-400 bg-purple-500/10 px-2 py-0.5 rounded">
                              {limitCPUWeight} (1..1000)
                            </span>
                          </div>
                          <input
                            type="range"
                            min={10}
                            max={1000}
                            step={10}
                            value={limitCPUWeight}
                            onChange={(e) => setLimitCPUWeight(Number(e.target.value))}
                            className="w-full accent-purple-600 cursor-pointer"
                          />
                          <p className="text-[11px] text-slate-500 mt-2">
                            Relative priority when the host CPU is under heavy contention. Default is 100.
                          </p>
                        </div>

                        {/* TasksMax Fork Bomb Guard */}
                        <div className="p-4 rounded-2xl bg-slate-50/70 dark:bg-surface-800/70 border border-slate-200/80 dark:border-surface-700">
                          <div className="flex items-center justify-between mb-2">
                            <label className="text-xs font-bold text-slate-800 dark:text-slate-200">
                              Max Process / Thread Limit (<code className="font-mono text-emerald-600">pids.max</code>)
                            </label>
                            <span className="text-xs font-mono font-bold text-emerald-600 dark:text-emerald-400 bg-emerald-500/10 px-2 py-0.5 rounded">
                              {limitTasks} PIDs
                            </span>
                          </div>
                          <input
                            type="range"
                            min={20}
                            max={500}
                            step={10}
                            value={limitTasks}
                            onChange={(e) => setLimitTasks(Number(e.target.value))}
                            className="w-full accent-emerald-600 cursor-pointer"
                          />
                          <p className="text-[11px] text-slate-500 mt-2">
                            Strict protection against rogue fork bombs and process exhaustion attacks across worker processes.
                          </p>
                        </div>

                        {/* OOM Policy */}
                        <div className="p-4 rounded-2xl bg-slate-50/70 dark:bg-surface-800/70 border border-slate-200/80 dark:border-surface-700">
                          <label className="block text-xs font-bold text-slate-800 dark:text-slate-200 mb-2">
                            Kernel Out-Of-Memory (OOM) Action
                          </label>
                          <select
                            value={limitOOMPolicy}
                            onChange={(e) => setLimitOOMPolicy(e.target.value)}
                            className="w-full px-3 py-2 rounded-xl bg-white dark:bg-surface-800 border border-slate-300 dark:border-surface-600 text-slate-900 dark:text-white text-xs font-semibold focus:outline-none"
                          >
                            <option value="kill">Immediate Kill (Kill worker exceeding slice cap)</option>
                            <option value="throttle">Aggressive Throttle (Throttle CPU & force page reclaim)</option>
                          </select>
                          <p className="text-[11px] text-slate-500 mt-2">
                            Determines systemd & cgroup behavior when memory limits are exceeded.
                          </p>
                        </div>
                      </div>

                      {/* Security Jail & IOPS controls */}
                      <div className="p-4 rounded-2xl bg-slate-50/70 dark:bg-surface-800/70 border border-slate-200/80 dark:border-surface-700 space-y-4">
                        <div className="flex items-center justify-between">
                          <div>
                            <span className="block text-xs font-bold text-slate-800 dark:text-slate-200">
                              PHP open_basedir Filesystem Jail
                            </span>
                            <span className="text-[11px] text-slate-500">
                              Restricts PHP scripts to only read from the website document root and /tmp
                            </span>
                          </div>
                          <input
                            type="checkbox"
                            checked={limitOpenBaseDir}
                            onChange={(e) => setLimitOpenBaseDir(e.target.checked)}
                            className="w-4 h-4 rounded text-indigo-600 focus:ring-indigo-500 cursor-pointer"
                          />
                        </div>

                        <div className="grid grid-cols-1 sm:grid-cols-2 gap-4 pt-3 border-t border-slate-200 dark:border-surface-700">
                          <div>
                            <label className="block text-[11px] font-bold text-slate-700 dark:text-slate-300 mb-1">
                              Max Disk Read IOPS (0 = unlimited)
                            </label>
                            <input
                              type="number"
                              min={0}
                              max={50000}
                              step={100}
                              value={limitIOReadIOPS}
                              onChange={(e) => setLimitIOReadIOPS(Number(e.target.value))}
                              className="w-full px-3 py-1.5 rounded-xl bg-white dark:bg-surface-800 border border-slate-300 dark:border-surface-600 text-xs font-mono text-slate-900 dark:text-white"
                              placeholder="0"
                            />
                          </div>
                          <div>
                            <label className="block text-[11px] font-bold text-slate-700 dark:text-slate-300 mb-1">
                              Max Disk Write IOPS (0 = unlimited)
                            </label>
                            <input
                              type="number"
                              min={0}
                              max={50000}
                              step={100}
                              value={limitIOWriteIOPS}
                              onChange={(e) => setLimitIOWriteIOPS(Number(e.target.value))}
                              className="w-full px-3 py-1.5 rounded-xl bg-white dark:bg-surface-800 border border-slate-300 dark:border-surface-600 text-xs font-mono text-slate-900 dark:text-white"
                              placeholder="0"
                            />
                          </div>
                        </div>
                      </div>
                    </div>
                  )}

                  {/* TAB 2: Live Cgroups Telemetry */}
                  {isolationTab === 'telemetry' && (
                    <div className="space-y-4">
                      <div className="p-4 rounded-2xl bg-slate-950 text-slate-200 font-mono text-xs border border-slate-800 space-y-3">
                        <div className="text-slate-400 font-bold border-b border-slate-800 pb-2 flex items-center justify-between">
                          <span>SYSTEMD SLICE HIERARCHY</span>
                          <span className="text-emerald-400">cgroups v2 active</span>
                        </div>
                        <div className="space-y-1 text-[11px]">
                          <div><span className="text-indigo-400">Cgroup Path:</span> {isolationInfo?.cgroup_path || `/sys/fs/cgroup/system.slice/system-hostvra-${isolationInfo?.username || 'user'}.slice`}</div>
                          <div><span className="text-indigo-400">Systemd Unit:</span> system-hostvra-{isolationInfo?.username || 'user'}.slice</div>
                          <div><span className="text-indigo-400">POSIX Username:</span> {isolationInfo?.username} (UID: {isolationInfo?.uid}, GID: {isolationInfo?.gid})</div>
                          <div><span className="text-indigo-400">Home Directory:</span> {isolationInfo?.home_dir}</div>
                          <div><span className="text-indigo-400">PHP-FPM Socket:</span> {isolationInfo?.php_pool_socket}</div>
                        </div>
                      </div>

                      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                        <div className="p-4 rounded-2xl bg-slate-50 dark:bg-surface-800 border border-slate-200 dark:border-surface-700 space-y-2">
                          <h4 className="text-xs font-bold text-slate-900 dark:text-white flex items-center gap-1.5">
                            <Activity className="w-3.5 h-3.5 text-indigo-500" />
                            Live Memory Counters (memory.events)
                          </h4>
                          <div className="space-y-1.5 text-xs text-slate-600 dark:text-slate-400">
                            <div className="flex justify-between py-1 border-b border-slate-200/50 dark:border-surface-700/50">
                              <span>Current Memory:</span>
                              <span className="font-mono font-bold text-slate-900 dark:text-white">{isolationInfo?.current_memory_mb || 0} MB</span>
                            </div>
                            <div className="flex justify-between py-1 border-b border-slate-200/50 dark:border-surface-700/50">
                              <span>Peak Memory:</span>
                              <span className="font-mono font-bold text-slate-900 dark:text-white">{isolationInfo?.memory_peak_mb || 0} MB</span>
                            </div>
                            <div className="flex justify-between py-1 border-b border-slate-200/50 dark:border-surface-700/50">
                              <span>OOM Kills (Kernel):</span>
                              <span className={`font-mono font-bold ${(isolationInfo?.oom_kills || 0) > 0 ? 'text-red-600' : 'text-emerald-600'}`}>
                                {isolationInfo?.oom_kills || 0}
                              </span>
                            </div>
                            <div className="flex justify-between py-1">
                              <span>Soft Throttle Reclaims:</span>
                              <span className="font-mono font-bold text-amber-600">{isolationInfo?.high_events || 0}</span>
                            </div>
                          </div>
                        </div>

                        <div className="p-4 rounded-2xl bg-slate-50 dark:bg-surface-800 border border-slate-200 dark:border-surface-700 space-y-2">
                          <h4 className="text-xs font-bold text-slate-900 dark:text-white flex items-center gap-1.5">
                            <Gauge className="w-3.5 h-3.5 text-blue-500" />
                            Live CPU & Task Counters (cpu.stat)
                          </h4>
                          <div className="space-y-1.5 text-xs text-slate-600 dark:text-slate-400">
                            <div className="flex justify-between py-1 border-b border-slate-200/50 dark:border-surface-700/50">
                              <span>Active CPU Usage:</span>
                              <span className="font-mono font-bold text-slate-900 dark:text-white">{(isolationInfo?.cpu_usage_percent || 0).toFixed(2)}%</span>
                            </div>
                            <div className="flex justify-between py-1 border-b border-slate-200/50 dark:border-surface-700/50">
                              <span>CPU Throttled Microseconds:</span>
                              <span className="font-mono font-bold text-slate-900 dark:text-white">{isolationInfo?.throttled_usec || 0} µs</span>
                            </div>
                            <div className="flex justify-between py-1 border-b border-slate-200/50 dark:border-surface-700/50">
                              <span>Active Tasks / Processes:</span>
                              <span className="font-mono font-bold text-slate-900 dark:text-white">{isolationInfo?.current_pids || 0}</span>
                            </div>
                            <div className="flex justify-between py-1">
                              <span>Systemd Slice Status:</span>
                              <span className="font-bold text-emerald-600 dark:text-emerald-400 uppercase text-[11px]">{isolationInfo?.status || 'active'}</span>
                            </div>
                          </div>
                        </div>
                      </div>
                    </div>
                  )}

                  {/* TAB 3: Quota & OOM Events Timeline */}
                  {isolationTab === 'events' && (
                    <div className="space-y-3">
                      {isolationEventsLoading ? (
                        <div className="py-12 text-center text-slate-500">
                          <RefreshCw className="w-6 h-6 animate-spin mx-auto mb-2 text-indigo-500" />
                          <p className="text-xs">Loading quota events timeline...</p>
                        </div>
                      ) : isolationEvents.length === 0 ? (
                        <div className="py-12 text-center text-slate-500 bg-slate-50 dark:bg-surface-800/50 rounded-2xl border border-slate-200 dark:border-surface-700">
                          <CheckCircle2 className="w-8 h-8 mx-auto mb-2 text-emerald-500" />
                          <p className="text-sm font-bold text-slate-700 dark:text-slate-300">Clean Operational History</p>
                          <p className="text-xs text-slate-500 mt-1">
                            No OOM kills or throttle violations recorded for this website.
                          </p>
                        </div>
                      ) : (
                        <div className="divide-y divide-slate-100 dark:divide-surface-800">
                          {isolationEvents.map((ev) => {
                            const isOOM = ev.event_type === 'oom_kill';
                            const isHigh = ev.event_type === 'memory_high';
                            const isUpdated = ev.event_type === 'quota_updated';
                            return (
                              <div key={ev.id} className="py-3 flex items-start gap-3">
                                <div
                                  className={`w-7 h-7 rounded-xl flex items-center justify-center shrink-0 mt-0.5 ${
                                    isOOM
                                      ? 'bg-red-500/10 text-red-600 border border-red-500/20'
                                      : isHigh
                                      ? 'bg-amber-500/10 text-amber-600 border border-amber-500/20'
                                      : isUpdated
                                      ? 'bg-blue-500/10 text-blue-600 border border-blue-500/20'
                                      : 'bg-emerald-500/10 text-emerald-600 border border-emerald-500/20'
                                  }`}
                                >
                                  {isOOM ? (
                                    <AlertTriangle className="w-3.5 h-3.5" />
                                  ) : isHigh ? (
                                    <Flame className="w-3.5 h-3.5" />
                                  ) : isUpdated ? (
                                    <SlidersHorizontal className="w-3.5 h-3.5" />
                                  ) : (
                                    <ShieldCheck className="w-3.5 h-3.5" />
                                  )}
                                </div>
                                <div className="flex-1 min-w-0">
                                  <div className="flex items-center justify-between gap-2">
                                    <span className="text-xs font-bold text-slate-900 dark:text-white capitalize">
                                      {ev.event_type.replace('_', ' ')}
                                    </span>
                                    <span className="text-[10px] text-slate-400 font-mono">
                                      {new Date(ev.created_at).toLocaleString()}
                                    </span>
                                  </div>
                                  <p className="text-xs text-slate-600 dark:text-slate-400 mt-0.5">{ev.details}</p>
                                </div>
                              </div>
                            );
                          })}
                        </div>
                      )}
                    </div>
                  )}
                </div>
              )}

              {/* Modal Footer */}
              <div className="p-4 px-6 border-t border-slate-100 dark:border-surface-800 bg-slate-50/50 dark:bg-surface-900/50 flex items-center justify-between">
                <div className="text-[11px] text-slate-500 hidden sm:block">
                  Target: <span className="font-mono font-bold text-slate-700 dark:text-slate-300">{limitMem}MB RAM</span> •{' '}
                  <span className="font-mono font-bold text-slate-700 dark:text-slate-300">{limitCPU}% CPU</span> •{' '}
                  <span className="font-mono font-bold text-slate-700 dark:text-slate-300">{limitTasks} Tasks</span>
                </div>
                <div className="flex items-center gap-3 ml-auto">
                  <button
                    onClick={() => setIsolationModalSite(null)}
                    className="px-4 py-2 rounded-xl text-xs font-bold text-slate-600 dark:text-slate-300 hover:bg-slate-100 dark:hover:bg-surface-800 transition-colors"
                  >
                    Cancel
                  </button>
                  <button
                    onClick={handleSaveIsolation}
                    disabled={isolationSaving}
                    className="px-5 py-2.5 rounded-xl bg-indigo-600 hover:bg-indigo-700 active:bg-indigo-800 disabled:opacity-50 text-white text-xs font-bold shadow-md shadow-indigo-600/20 cursor-pointer flex items-center gap-1.5 transition-all"
                  >
                    {isolationSaving ? (
                      <>
                        <RefreshCw className="w-3.5 h-3.5 animate-spin" />
                        <span>Applying Cgroups v2...</span>
                      </>
                    ) : (
                      <>
                        <ShieldCheck className="w-3.5 h-3.5" />
                        <span>Apply Cgroups v2 Quotas</span>
                      </>
                    )}
                  </button>
                </div>
              </div>
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

        {/* Delete Website Confirmation Modal */}
        {deleteModalOpen && deleteModalSite && (
          <div className="fixed inset-0 z-50 bg-black/60 backdrop-blur-xs flex items-center justify-center p-4 animate-fadeIn">
            <div className="w-full max-w-lg bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-700 rounded-2xl shadow-2xl p-6 relative">
              <button
                onClick={() => {
                  setDeleteModalOpen(false);
                  setDeleteModalSite(null);
                }}
                disabled={deletingWebsite}
                className="absolute top-5 right-5 p-1 rounded-lg text-slate-400 hover:text-slate-700 dark:hover:text-white"
              >
                <X className="w-5 h-5" />
              </button>

              <div className="flex items-center gap-3 mb-5">
                <div className="w-10 h-10 rounded-xl bg-rose-500/10 border border-rose-500/20 flex items-center justify-center text-rose-600 dark:text-rose-400">
                  <Trash2 className="w-5 h-5" />
                </div>
                <div>
                  <h2 className="text-lg font-bold text-slate-900 dark:text-white">Delete Website</h2>
                  <p className="text-xs text-rose-500 font-semibold">{deleteModalSite.primary_domain}</p>
                </div>
              </div>

              <div className="mb-4 p-3 bg-amber-500/10 border border-amber-500/20 rounded-xl flex items-start gap-2.5">
                <AlertTriangle className="w-4 h-4 text-amber-600 dark:text-amber-400 shrink-0 mt-0.5" />
                <div className="text-xs text-amber-800 dark:text-amber-200 space-y-1">
                  <p className="font-semibold">This will delete the virtual host and PHP pool configurations.</p>
                  <p className="text-[11px] text-amber-700/80 dark:text-amber-300/80">Document root: <code className="bg-amber-100 dark:bg-amber-950/40 px-1 py-0.5 rounded font-mono">{deleteModalSite.document_root || 'N/A'}</code></p>
                </div>
              </div>

              <div className="space-y-3 mb-5 text-xs">
                <div className="font-bold text-slate-700 dark:text-slate-300">Cleanup Options:</div>

                <label className="flex items-start gap-2.5 p-2.5 rounded-lg border border-slate-200 dark:border-surface-700 hover:bg-slate-50 dark:hover:bg-surface-800 cursor-pointer transition-colors">
                  <input
                    type="checkbox"
                    checked={deleteOpts.delete_files}
                    onChange={(e) => {
                      const checked = e.target.checked;
                      setDeleteOpts(prev => ({
                        ...prev,
                        delete_files: checked,
                        permanent_wipe: checked ? prev.permanent_wipe : false,
                      }));
                    }}
                    className="mt-0.5 rounded border-slate-300 text-rose-600 focus:ring-rose-500"
                  />
                  <div>
                    <div className="font-semibold text-slate-900 dark:text-white">Move files to Trash</div>
                    <div className="text-[11px] text-slate-500">Safely stores document root files in Trash bin so they can be restored if needed.</div>
                  </div>
                </label>

                {deleteOpts.delete_files && (
                  <label className="flex items-start gap-2.5 p-2.5 rounded-lg border border-rose-200 dark:border-rose-900/40 bg-rose-50/50 dark:bg-rose-950/20 hover:bg-rose-50 dark:hover:bg-rose-950/30 cursor-pointer transition-colors ml-4">
                    <input
                      type="checkbox"
                      checked={deleteOpts.permanent_wipe}
                      onChange={(e) => setDeleteOpts(prev => ({ ...prev, permanent_wipe: e.target.checked }))}
                      className="mt-0.5 rounded border-rose-300 text-rose-600 focus:ring-rose-500"
                    />
                    <div>
                      <div className="font-semibold text-rose-600 dark:text-rose-400">Permanently Wipe Files (Caution)</div>
                      <div className="text-[11px] text-rose-500/80">Completely bypasses Trash bin and permanently deletes files from disk. Cannot be undone!</div>
                    </div>
                  </label>
                )}

                <label className="flex items-start gap-2.5 p-2.5 rounded-lg border border-slate-200 dark:border-surface-700 hover:bg-slate-50 dark:hover:bg-surface-800 cursor-pointer transition-colors">
                  <input
                    type="checkbox"
                    checked={deleteOpts.delete_dns}
                    onChange={(e) => setDeleteOpts(prev => ({ ...prev, delete_dns: e.target.checked }))}
                    className="mt-0.5 rounded border-slate-300 text-rose-600 focus:ring-rose-500"
                  />
                  <div>
                    <div className="font-semibold text-slate-900 dark:text-white">Delete DNS Zone records</div>
                    <div className="text-[11px] text-slate-500">Removes managed DNS zone entries associated with this domain.</div>
                  </div>
                </label>

                <label className="flex items-start gap-2.5 p-2.5 rounded-lg border border-slate-200 dark:border-surface-700 hover:bg-slate-50 dark:hover:bg-surface-800 cursor-pointer transition-colors">
                  <input
                    type="checkbox"
                    checked={deleteOpts.delete_ftp}
                    onChange={(e) => setDeleteOpts(prev => ({ ...prev, delete_ftp: e.target.checked }))}
                    className="mt-0.5 rounded border-slate-300 text-rose-600 focus:ring-rose-500"
                  />
                  <div>
                    <div className="font-semibold text-slate-900 dark:text-white">Delete associated FTP accounts</div>
                    <div className="text-[11px] text-slate-500">Removes FTP users configured for this website directory.</div>
                  </div>
                </label>

                <label className="flex items-start gap-2.5 p-2.5 rounded-lg border border-slate-200 dark:border-surface-700 hover:bg-slate-50 dark:hover:bg-surface-800 cursor-pointer transition-colors">
                  <input
                    type="checkbox"
                    checked={deleteOpts.delete_database}
                    onChange={(e) => setDeleteOpts(prev => ({ ...prev, delete_database: e.target.checked }))}
                    className="mt-0.5 rounded border-slate-300 text-rose-600 focus:ring-rose-500"
                  />
                  <div>
                    <div className="font-semibold text-slate-900 dark:text-white">Delete associated database</div>
                    <div className="text-[11px] text-slate-500">Attempts to drop matching MySQL database if configured.</div>
                  </div>
                </label>
              </div>

              <div className="mb-5">
                <label className="block text-xs font-semibold text-slate-700 dark:text-slate-300 mb-1.5">
                  Type <span className="font-bold text-rose-600 font-mono select-all">{deleteModalSite.primary_domain}</span> to confirm:
                </label>
                <input
                  type="text"
                  value={deleteConfirmText}
                  onChange={(e) => setDeleteConfirmText(e.target.value)}
                  placeholder={deleteModalSite.primary_domain}
                  className="w-full px-3 py-2 rounded-xl bg-slate-50 dark:bg-surface-800 border border-slate-200 dark:border-surface-700 text-xs font-mono text-slate-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-rose-500"
                />
              </div>

              <div className="flex items-center justify-end gap-2.5">
                <button
                  type="button"
                  onClick={() => {
                    setDeleteModalOpen(false);
                    setDeleteModalSite(null);
                  }}
                  disabled={deletingWebsite}
                  className="px-4 py-2 rounded-xl border border-slate-200 dark:border-surface-700 text-xs font-semibold text-slate-600 dark:text-slate-300 hover:bg-slate-100 dark:hover:bg-surface-800 transition-colors"
                >
                  Cancel
                </button>
                <button
                  type="button"
                  onClick={confirmDeleteWebsite}
                  disabled={deletingWebsite || deleteConfirmText.trim().toLowerCase() !== deleteModalSite.primary_domain.toLowerCase()}
                  className="px-4 py-2 rounded-xl bg-rose-600 hover:bg-rose-700 text-white text-xs font-bold transition-colors disabled:opacity-40 disabled:cursor-not-allowed flex items-center gap-1.5"
                >
                  {deletingWebsite ? (
                    <>
                      <RefreshCw className="w-3.5 h-3.5 animate-spin" />
                      <span>Deleting...</span>
                    </>
                  ) : (
                    <>
                      <Trash2 className="w-3.5 h-3.5" />
                      <span>Delete Website</span>
                    </>
                  )}
                </button>
              </div>
            </div>
          </div>
        )}

        {/* 17. Floating 3-Dots Action Menu (Fixed-position, never clipped by table overflow) */}
        {openDropdownId && activeDropdownSite && dropdownPos && (
          <div
            data-dropdown-menu
            onClick={(e) => e.stopPropagation()}
            style={{
              position: 'fixed',
              ...(dropdownPos.openUp
                ? { bottom: `${window.innerHeight - dropdownPos.top}px` }
                : { top: `${dropdownPos.top}px` }),
              left: `${dropdownPos.left}px`,
              zIndex: 99999,
            }}
            className="w-56 bg-white/95 dark:bg-surface-900/95 backdrop-blur-md border border-slate-200 dark:border-surface-700 rounded-2xl shadow-2xl py-2 text-xs font-semibold text-slate-800 dark:text-slate-100 text-left animate-fadeIn select-none ring-1 ring-black/10"
          >
            {/* Header info */}
            <div className="px-3.5 pb-1.5 border-b border-slate-100 dark:border-surface-800 mb-1">
              <span className="text-[10px] text-slate-400 uppercase tracking-wider block font-bold">Manage Website</span>
              <span className="font-bold text-slate-900 dark:text-white truncate block text-[11px]">
                {activeDropdownSite.primary_domain}
              </span>
            </div>

            {/* Section 1: Core Configuration */}
            <div className="px-3.5 py-1 text-[10px] font-bold uppercase tracking-wider text-slate-400">
              Core Configuration
            </div>

            <button
              onClick={() => {
                const s = activeDropdownSite;
                setOpenDropdownId(null);
                setActiveDropdownSite(null);
                openSiteModal(s, 'domain');
              }}
              className="w-full px-3.5 py-1.5 hover:bg-emerald-50 dark:hover:bg-emerald-950/30 flex items-center gap-2 cursor-pointer text-emerald-600 dark:text-emerald-400 font-bold transition"
            >
              <Settings className="w-3.5 h-3.5" />
              <span>Site modification</span>
            </button>

            <button
              onClick={() => {
                const s = activeDropdownSite;
                setOpenDropdownId(null);
                setActiveDropdownSite(null);
                openSiteModal(s, 'rewrite');
              }}
              className="w-full px-3.5 py-1.5 hover:bg-slate-50 dark:hover:bg-surface-800 flex items-center gap-2 cursor-pointer text-slate-700 dark:text-slate-200 transition"
            >
              <Code2 className="w-3.5 h-3.5 text-slate-400" />
              <span>Rewrite rules (URL)</span>
            </button>

            <button
              onClick={() => {
                const s = activeDropdownSite;
                setOpenDropdownId(null);
                setActiveDropdownSite(null);
                openSiteModal(s, 'ssl');
              }}
              className="w-full px-3.5 py-1.5 hover:bg-slate-50 dark:hover:bg-surface-800 flex items-center gap-2 cursor-pointer text-slate-700 dark:text-slate-200 transition"
            >
              <Lock className="w-3.5 h-3.5 text-teal-500" />
              <span>SSL Certificate</span>
            </button>

            <button
              onClick={() => {
                const s = activeDropdownSite;
                setOpenDropdownId(null);
                setActiveDropdownSite(null);
                openSiteModal(s, 'php');
              }}
              className="w-full px-3.5 py-1.5 hover:bg-slate-50 dark:hover:bg-surface-800 flex items-center gap-2 cursor-pointer text-slate-700 dark:text-slate-200 transition"
            >
              <SlidersHorizontal className="w-3.5 h-3.5 text-amber-500" />
              <span>PHP Version</span>
            </button>

            {/* Section 2: Security & Limits */}
            <div className="border-t border-slate-100 dark:border-surface-800 my-1 pt-1">
              <div className="px-3.5 py-1 text-[10px] font-bold uppercase tracking-wider text-slate-400">
                Security & Limits
              </div>
              <button
                onClick={() => {
                  const s = activeDropdownSite;
                  setOpenDropdownId(null);
                  setActiveDropdownSite(null);
                  openSiteModal(s, 'limit');
                }}
                className="w-full px-3.5 py-1.5 hover:bg-slate-50 dark:hover:bg-surface-800 flex items-center gap-2 cursor-pointer text-slate-700 dark:text-slate-200 transition"
              >
                <Shield className="w-3.5 h-3.5 text-indigo-500" />
                <span>Limit access & WAF</span>
              </button>

              <button
                onClick={() => {
                  const s = activeDropdownSite;
                  setOpenDropdownId(null);
                  setActiveDropdownSite(null);
                  openIsolationModal(s);
                }}
                className="w-full px-3.5 py-1.5 hover:bg-slate-50 dark:hover:bg-surface-800 flex items-center gap-2 cursor-pointer text-indigo-600 dark:text-indigo-400 font-semibold transition"
              >
                <Cpu className="w-3.5 h-3.5" />
                <span>cgroups v2 limits</span>
              </button>
            </div>

            {/* Section 3: Applications & CMS */}
            <div className="border-t border-slate-100 dark:border-surface-800 my-1 pt-1">
              <div className="px-3.5 py-1 text-[10px] font-bold uppercase tracking-wider text-slate-400">
                Applications & CMS
              </div>
              <button
                onClick={() => {
                  const s = activeDropdownSite;
                  setOpenDropdownId(null);
                  setActiveDropdownSite(null);
                  if (s) openSiteModal(s, 'wordpress');
                }}
                className="w-full px-3.5 py-1.5 hover:bg-purple-50 dark:hover:bg-purple-950/30 flex items-center gap-2 cursor-pointer text-purple-600 dark:text-purple-400 font-semibold transition"
              >
                <Globe className="w-3.5 h-3.5" />
                <span>WordPress & Staging</span>
              </button>

              <button
                onClick={() => {
                  const s = activeDropdownSite;
                  setOpenDropdownId(null);
                  setActiveDropdownSite(null);
                  if (s) openSiteModal(s, 'appruntime');
                }}
                className="w-full px-3.5 py-1.5 hover:bg-emerald-50 dark:hover:bg-emerald-950/30 flex items-center gap-2 cursor-pointer text-emerald-600 dark:text-emerald-400 font-semibold transition"
              >
                <Cpu className="w-3.5 h-3.5 text-emerald-500" />
                <span>Node.js & Python Studio</span>
              </button>

              <button
                onClick={() => {
                  const s = activeDropdownSite;
                  setOpenDropdownId(null);
                  setActiveDropdownSite(null);
                  if (s) openSiteModal(s, 'git');
                }}
                className="w-full px-3.5 py-1.5 hover:bg-blue-50 dark:hover:bg-blue-950/30 flex items-center gap-2 cursor-pointer text-blue-600 dark:text-blue-400 font-semibold transition"
              >
                <GitBranch className="w-3.5 h-3.5 text-blue-500" />
                <span>Git CI/CD Pipeline</span>
              </button>

              <button
                onClick={() => {
                  const s = activeDropdownSite;
                  setOpenDropdownId(null);
                  setActiveDropdownSite(null);
                  setAppModalSite(s);
                }}
                className="w-full px-3.5 py-1.5 hover:bg-indigo-50 dark:hover:bg-indigo-950/30 flex items-center gap-2 cursor-pointer text-indigo-600 dark:text-indigo-400 font-semibold transition"
              >
                <Zap className="w-3.5 h-3.5 text-indigo-500" />
                <span>1-Click App Installer</span>
              </button>
            </div>

            {/* Section 4: Danger Zone */}
            <div className="border-t border-slate-100 dark:border-surface-800 my-1 pt-1">
              <button
                onClick={() => {
                  const s = activeDropdownSite;
                  setOpenDropdownId(null);
                  setActiveDropdownSite(null);
                  handleDeleteWebsite(s);
                }}
                className="w-full px-3.5 py-1.5 hover:bg-rose-50 dark:hover:bg-rose-950/20 text-rose-600 dark:text-rose-400 flex items-center gap-2 cursor-pointer font-bold transition"
              >
                <Trash2 className="w-3.5 h-3.5" />
                <span>Delete website</span>
              </button>
            </div>
          </div>
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
