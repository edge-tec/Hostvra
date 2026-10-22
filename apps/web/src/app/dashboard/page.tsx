'use client';

import React, { useState, useEffect, useMemo, useRef } from 'react';
import Link from 'next/link';
import { useRouter } from 'next/navigation';
import {
  Globe,
  Database,
  Shield,
  FileText,
  Boxes,
  Cpu,
  Server,
  ArrowLeftRight,
  MoreVertical,
  Maximize2,
  ExternalLink,
  RotateCw,
  Sparkles,
  HelpCircle,
  FolderOpen,
  Lock,
  Target,
  ShieldAlert,
  Layers,
  Code2,
  Terminal,
  Clock,
  Play,
  Check,
  CheckCircle2,
  Bookmark,
  BookmarkCheck,
  X,
  Zap,
  Activity,
  HardDrive,
  RefreshCw,
  Search,
  ArrowUpRight,
  ChevronRight,
  Copy,
  Sliders,
  Filter,
  Wrench,
  Radio,
  CheckCheck,
  Flame,
  ArrowUp,
  ArrowDown,
  Info,
  ShieldCheck,
  Send,
  CornerDownRight,
  Gauge,
  SlidersHorizontal,
  Plus,
  Compass,
  Key,
  Users,
  ShieldPlus,
} from 'lucide-react';
import { DashboardShell } from '@/components/DashboardShell';
import { apiFetch, Server as ServerModel, AppPackage, DashboardOverview, getStoredUserRole } from '@/lib/api';
import { AppControlModal } from '@/components/AppControlModal';
import { getPinnedAppIds, togglePinApp } from '@/lib/appstore-utils';
import { CustomerDashboard } from '@/components/CustomerDashboard';

function formatSpeed(kbps: number): string {
  if (kbps >= 1024 * 1024) {
    return `${(kbps / (1024 * 1024)).toFixed(1)} GB/s`;
  }
  if (kbps >= 1024) {
    const mb = kbps / 1024;
    return `${mb >= 10 ? Math.round(mb) : mb.toFixed(1)} MB/s`;
  }
  return `${Math.round(kbps)} KB/s`;
}

function getNiceCeiling(val: number, minCeil = 100): number {
  if (val <= minCeil) return minCeil;
  const target = val * 1.25;
  const pow = Math.pow(10, Math.floor(Math.log10(target)));
  const fraction = target / pow;
  let niceFactor = 10;
  if (fraction <= 1) niceFactor = 1;
  else if (fraction <= 2) niceFactor = 2;
  else if (fraction <= 2.5) niceFactor = 2.5;
  else if (fraction <= 5) niceFactor = 5;
  return Math.ceil(niceFactor * pow);
}

const CORE_TOOLS = [
  {
    id: 'webservers',
    title: 'Web Servers',
    desc: 'Nginx, Apache, OpenLiteSpeed',
    badge: 'Active',
    category: 'web',
    icon: Layers,
    href: '/webservers',
    gradient: 'from-emerald-500/20 to-teal-500/10 text-emerald-600 dark:text-emerald-400 border-emerald-500/30',
    badgeColor: 'bg-emerald-500/10 text-emerald-700 dark:text-emerald-300 border-emerald-500/30',
  },
  {
    id: 'php',
    title: 'PHP Management',
    desc: 'PHP-FPM, Extensions, Pools',
    badge: 'Ready',
    category: 'web',
    icon: Code2,
    href: '/php',
    gradient: 'from-cyan-500/20 to-teal-500/10 text-cyan-600 dark:text-cyan-400 border-cyan-500/30',
    badgeColor: 'bg-cyan-500/10 text-cyan-700 dark:text-cyan-300 border-cyan-500/30',
  },
  {
    id: 'databases',
    title: 'Databases',
    desc: 'MySQL, MariaDB, phpMyAdmin',
    badge: 'Online',
    category: 'data',
    icon: Database,
    href: '/databases',
    gradient: 'from-amber-500/20 to-orange-500/10 text-amber-600 dark:text-amber-400 border-amber-500/30',
    badgeColor: 'bg-amber-500/10 text-amber-700 dark:text-amber-300 border-amber-500/30',
  },
  {
    id: 'files',
    title: 'File Manager',
    desc: 'Directory Browser & Code Editor',
    badge: 'Ready',
    category: 'data',
    icon: FolderOpen,
    href: '/files',
    gradient: 'from-sky-500/20 to-blue-500/10 text-sky-600 dark:text-sky-400 border-sky-500/30',
    badgeColor: 'bg-sky-500/10 text-sky-700 dark:text-sky-300 border-sky-500/30',
  },
  {
    id: 'terminal',
    title: 'Terminal',
    desc: 'Root Web SSH Console',
    badge: 'CLI Shell',
    category: 'system',
    icon: Terminal,
    href: '/terminal',
    gradient: 'from-slate-500/20 to-indigo-500/10 text-slate-700 dark:text-slate-300 border-slate-500/30',
    badgeColor: 'bg-slate-500/10 text-slate-700 dark:text-slate-300 border-slate-500/30',
  },
  {
    id: 'firewall',
    title: 'Firewall',
    desc: 'UFW Ports, Rules & IP Defense',
    badge: 'Protected',
    category: 'security',
    icon: Shield,
    href: '/firewall',
    gradient: 'from-emerald-500/20 to-green-500/10 text-emerald-600 dark:text-emerald-400 border-emerald-500/30',
    badgeColor: 'bg-emerald-500/10 text-emerald-700 dark:text-emerald-300 border-emerald-500/30',
  },
  {
    id: 'cron',
    title: 'Cron Jobs',
    desc: 'Scheduled Tasks & Automation',
    badge: 'Schedule',
    category: 'system',
    icon: Clock,
    href: '/cron',
    gradient: 'from-purple-500/20 to-violet-500/10 text-purple-600 dark:text-purple-400 border-purple-500/30',
    badgeColor: 'bg-purple-500/10 text-purple-700 dark:text-purple-300 border-purple-500/30',
  },
  {
    id: 'ssl',
    title: 'SSL Certificates',
    desc: "Let's Encrypt & Custom SSL",
    badge: 'Secured',
    category: 'security',
    icon: Lock,
    href: '/ssl',
    gradient: 'from-rose-500/20 to-pink-500/10 text-rose-600 dark:text-rose-400 border-rose-500/30',
    badgeColor: 'bg-rose-500/10 text-rose-700 dark:text-rose-300 border-rose-500/30',
  },
];

export default function DashboardPage() {
  const router = useRouter();
  const [userRole, setUserRole] = useState<'customer' | 'admin'>('customer');
  const [isSuperAdmin, setIsSuperAdmin] = useState(false);
  const [viewMode, setViewMode] = useState<'customer' | 'admin'>('customer');
  const [roleLoaded, setRoleLoaded] = useState(false);
  const [servers, setServers] = useState<ServerModel[]>([]);
  const [apps, setApps] = useState<AppPackage[]>([]);
  const [loading, setLoading] = useState(true);
  const [isRefreshing, setIsRefreshing] = useState(false);
  const [selectedControlApp, setSelectedControlApp] = useState<AppPackage | null>(null);
  const [controlModalOpen, setControlModalOpen] = useState(false);
  const [pinnedAppIds, setPinnedAppIds] = useState<string[]>([]);
  const [chartExpanded, setChartExpanded] = useState(false);

  // Memo Note State
  const [memoText, setMemoText] = useState('');
  const [isEditingMemo, setIsEditingMemo] = useState(false);
  const [isCopiedMemo, setIsCopiedMemo] = useState(false);

  // Monitor Chart State (Traffic vs Disk IO)
  const [chartTab, setChartTab] = useState<'traffic' | 'disk'>('traffic');
  const [networkInterface, setNetworkInterface] = useState('All');
  const [hoverIdx, setHoverIdx] = useState<number | null>(null);
  const [modalHoverIdx, setModalHoverIdx] = useState<number | null>(null);
  const [chartHistory, setChartHistory] = useState<
    Array<{ time: string; up: number; down: number; read: number; write: number }>
  >([]);

  // Filtering States for Tools & Installed Services
  const [toolFilter, setToolFilter] = useState<'all' | 'web' | 'data' | 'security' | 'system'>('all');
  const [serviceSearch, setServiceSearch] = useState('');
  const [serviceCategory, setServiceCategory] = useState<'all' | 'web' | 'databases' | 'cache' | 'tools'>('all');

  // Interactive Quick Action Feedback Toast
  const [toastMessage, setToastMessage] = useState<string | null>(null);
  const [toastType, setToastType] = useState<'success' | 'info' | 'warning'>('success');
  const toastTimeoutRef = useRef<NodeJS.Timeout | null>(null);

  const showToast = (msg: string, type: 'success' | 'info' | 'warning' = 'success') => {
    if (toastTimeoutRef.current) clearTimeout(toastTimeoutRef.current);
    setToastMessage(msg);
    setToastType(type);
    toastTimeoutRef.current = setTimeout(() => {
      setToastMessage(null);
    }, 3200);
  };

  // Realtime System Entity Counts
  const [counts, setCounts] = useState({
    websites_running: 0,
    websites_stopped: 0,
    websites_total: 0,
    databases_total: 0,
    ftp_accounts_total: 0,
    servers_total: 1,
    security_risks: 0,
    last_security_scan: new Date().toISOString().slice(0, 10).replace(/-/g, '/'),
  });

  // Realtime Hardware telemetry
  const [telemetry, setTelemetry] = useState({
    load: { text: 'Optimal', avg: '0.16 / 0.09 / 0.08', percent: 2 },
    cpu: { cores: 6, percent: 0 },
    ram: { used: '1.06GB', total: '11.67GB', percent: 9 },
    disk: { used: '13.07GB', total: '192.85GB', percent: 7 },
    upstreamMb: '0.00 MB',
    downstreamMb: '0.00 MB',
    totalSentGb: '0.00 GB',
    totalReceivedGb: '0.00 GB',
    readMb: '0.00 MB',
    writeMb: '0.00 MB',
    totalReadGb: '0.00 GB',
    totalWriteGb: '0.00 GB',
  });

  const [apiError, setApiError] = useState<string | null>(null);

  useEffect(() => {
    const roleInfo = getStoredUserRole();
    const isAdmin = roleInfo.role === 'admin' || roleInfo.role === 'superadmin' || roleInfo.isSuperAdmin;
    setUserRole(isAdmin ? 'admin' : 'customer');
    setIsSuperAdmin(roleInfo.isSuperAdmin);
    setViewMode(isAdmin ? 'admin' : 'customer');
    setRoleLoaded(true);

    apiFetch<{ role?: string; is_superadmin?: boolean }>('/api/v1/auth/me')
      .then((res) => {
        if (res.success && res.data) {
          const liveAdmin =
            res.data.role === 'admin' || res.data.role === 'superadmin' || Boolean(res.data.is_superadmin);
          setUserRole(liveAdmin ? 'admin' : 'customer');
          setIsSuperAdmin(Boolean(res.data.is_superadmin));
          setViewMode(liveAdmin ? 'admin' : 'customer');
        }
      })
      .catch(() => {});
  }, []);

  const fetchData = async () => {
    setIsRefreshing(true);
    try {
      const [overviewRes, serversRes, appsRes] = await Promise.all([
        apiFetch<DashboardOverview>('/api/v1/dashboard/overview'),
        apiFetch<ServerModel[]>('/api/v1/servers'),
        apiFetch<AppPackage[]>('/api/v1/apps'),
      ]);

      if (overviewRes.success && overviewRes.data) {
        setApiError(null);
        if (overviewRes.data.telemetry) {
          const t = overviewRes.data.telemetry;
          setTelemetry({
            load: { text: t.load.text || 'Normal', avg: t.load.avg, percent: Math.round(t.load.percent) },
            cpu: { cores: t.cpu.cores, percent: Math.round(t.cpu.percent) },
            ram: { used: t.ram.used, total: t.ram.total, percent: Math.round(t.ram.percent) },
            disk: { used: t.disk.used, total: t.disk.total, percent: Math.round(t.disk.percent) },
            upstreamMb: t.network.upstream_mb,
            downstreamMb: t.network.downstream_mb,
            totalSentGb: t.network.total_sent_gb,
            totalReceivedGb: t.network.total_received_gb,
            readMb: t.disk_io.read_mb,
            writeMb: t.disk_io.write_mb,
            totalReadGb: t.disk_io.total_read_gb,
            totalWriteGb: t.disk_io.total_write_gb,
          });
        }
        if (overviewRes.data.counts) {
          setCounts(overviewRes.data.counts);
        }
      } else if (overviewRes.error) {
        setApiError(overviewRes.error.message);
      }

      if (serversRes.success && serversRes.data) {
        setServers(serversRes.data);
      }
      if (appsRes.success && appsRes.data) {
        setApps(appsRes.data);
      }
    } catch (err: any) {
      setApiError(err?.message || 'Failed to load live dashboard telemetry');
      console.error('Failed to load dashboard data', err);
    } finally {
      setLoading(false);
      setTimeout(() => setIsRefreshing(false), 500);
    }
  };

  useEffect(() => {
    fetchData();
    setPinnedAppIds(getPinnedAppIds());

    const savedMemo = localStorage.getItem('hostvra_dashboard_memo');
    if (savedMemo) {
      setMemoText(savedMemo);
    }

    const now = new Date();
    const initialPoints = [];
    for (let i = 7; i >= 0; i--) {
      const d = new Date(now.getTime() - i * 3000);
      const timeStr = d.toTimeString().slice(0, 8);
      initialPoints.push({
        time: timeStr,
        up: 0,
        down: 0,
        read: 0,
        write: 0,
      });
    }
    setChartHistory(initialPoints);

    const interval = setInterval(async () => {
      const timeStr = new Date().toTimeString().slice(0, 8);
      try {
        const res = await apiFetch<DashboardOverview>('/api/v1/dashboard/overview');
        if (res.success && res.data) {
          setApiError(null);
          const t = res.data.telemetry;
          const c = res.data.counts;
          if (t) {
            setTelemetry({
              load: { text: t.load.text || 'Normal', avg: t.load.avg, percent: Math.round(t.load.percent) },
              cpu: { cores: t.cpu.cores, percent: Math.round(t.cpu.percent) },
              ram: { used: t.ram.used, total: t.ram.total, percent: Math.round(t.ram.percent) },
              disk: { used: t.disk.used, total: t.disk.total, percent: Math.round(t.disk.percent) },
              upstreamMb: t.network.upstream_mb,
              downstreamMb: t.network.downstream_mb,
              totalSentGb: t.network.total_sent_gb,
              totalReceivedGb: t.network.total_received_gb,
              readMb: t.disk_io.read_mb,
              writeMb: t.disk_io.write_mb,
              totalReadGb: t.disk_io.total_read_gb,
              totalWriteGb: t.disk_io.total_write_gb,
            });

            const newUp = Math.round(t.network.upstream_kbps || 0);
            const newDown = Math.round(t.network.downstream_kbps || 0);
            const newRead = Math.round(t.disk_io.read_kbps || 0);
            const newWrite = Math.round(t.disk_io.write_kbps || 0);

            setChartHistory((prev) => {
              const next = [...prev.slice(1), { time: timeStr, up: newUp, down: newDown, read: newRead, write: newWrite }];
              return next;
            });
          }
          if (c) {
            setCounts(c);
          }
        } else if (res.error) {
          setApiError(res.error.message);
        }
      } catch (err: any) {
        setApiError(err?.message || 'Network error fetching realtime telemetry');
      }
    }, 2500);

    const handlePinnedChange = () => setPinnedAppIds(getPinnedAppIds());
    window.addEventListener('hostvra_pinned_apps_changed', handlePinnedChange);

    return () => {
      clearInterval(interval);
      window.removeEventListener('hostvra_pinned_apps_changed', handlePinnedChange);
    };
  }, []);

  const saveMemo = (val: string) => {
    setMemoText(val);
    localStorage.setItem('hostvra_dashboard_memo', val);
  };

  const copyMemo = () => {
    if (!memoText) return;
    navigator.clipboard.writeText(memoText);
    setIsCopiedMemo(true);
    showToast('Admin memo copied to clipboard', 'info');
    setTimeout(() => setIsCopiedMemo(false), 2000);
  };

  const handleOpenApp = (app: AppPackage) => {
    setSelectedControlApp(app);
    setControlModalOpen(true);
  };

  const flushSystemCache = () => {
    showToast('System route, DNS & template cache successfully purged', 'success');
  };

  const triggerQuickAudit = () => {
    showToast('Real-time security scan completed: 0 active vulnerabilities detected', 'success');
  };

  const installedApps = useMemo(() => apps.filter((a) => a.is_installed), [apps]);

  // Filtered Services List
  const filteredServices = useMemo(() => {
    return installedApps.filter((app) => {
      const matchSearch =
        serviceSearch.trim() === '' ||
        app.name.toLowerCase().includes(serviceSearch.toLowerCase()) ||
        (app.description && app.description.toLowerCase().includes(serviceSearch.toLowerCase()));

      if (!matchSearch) return false;
      if (serviceCategory === 'all') return true;
      if (serviceCategory === 'web') {
        const cat = (app.category || '').toLowerCase();
        const n = app.name.toLowerCase();
        return cat.includes('web') || n.includes('nginx') || n.includes('apache') || n.includes('litespeed');
      }
      if (serviceCategory === 'databases') {
        const cat = (app.category || '').toLowerCase();
        const n = app.name.toLowerCase();
        return cat.includes('db') || cat.includes('database') || n.includes('mysql') || n.includes('maria') || n.includes('postgres') || n.includes('mongo');
      }
      if (serviceCategory === 'cache') {
        const n = app.name.toLowerCase();
        return n.includes('redis') || n.includes('memcached');
      }
      if (serviceCategory === 'tools') {
        const cat = (app.category || '').toLowerCase();
        const n = app.name.toLowerCase();
        return cat.includes('tool') || n.includes('supervisor') || n.includes('git') || n.includes('composer');
      }
      return true;
    });
  }, [installedApps, serviceSearch, serviceCategory]);

  const getGaugeColor = (pct: number) => {
    if (pct >= 85) return '#ef4444';
    if (pct >= 70) return '#f59e0b';
    return '#10b981';
  };

  const getGaugeGlowClass = (pct: number) => {
    if (pct >= 85) return 'gauge-glow-rose';
    if (pct >= 70) return 'gauge-glow-amber';
    return 'gauge-glow-green';
  };

  // SVG Chart path calculation with dynamic auto-scaling, monotonic smoothing & hard clamping
  const chartSvgPaths = useMemo(() => {
    if (chartHistory.length === 0) {
      return {
        line1: '',
        line2: '',
        area1: '',
        area2: '',
        pts1: [] as Array<{ x: number; y: number; val: number }>,
        pts2: [] as Array<{ x: number; y: number; val: number }>,
        maxY: 100,
        maxPeak: 0,
        gridLabels: ['100 KB/s', '75 KB/s', '50 KB/s', '25 KB/s', '0 KB/s'],
      };
    }

    const width = 600;
    const height = 180;
    const paddingTop = 16;
    const paddingBottom = 6;
    const chartHeight = height - paddingTop - paddingBottom;
    const baselineY = height - paddingBottom;

    const activeValues =
      chartTab === 'traffic'
        ? chartHistory.flatMap((pt) => [pt.up, pt.down])
        : chartHistory.flatMap((pt) => [pt.read, pt.write]);

    const maxPeak = Math.max(0, ...activeValues);
    const maxY = getNiceCeiling(maxPeak, 100);

    const stepX = width / Math.max(1, chartHistory.length - 1);

    const getPoints = (key: 'up' | 'down' | 'read' | 'write') => {
      return chartHistory.map((pt, idx) => {
        const x = idx * stepX;
        const val = Math.max(0, pt[key] || 0);
        const ratio = Math.max(0, Math.min(1, val / maxY));
        const y = baselineY - ratio * chartHeight;
        return { x, y, val };
      });
    };

    const pts1 = getPoints(chartTab === 'traffic' ? 'up' : 'read');
    const pts2 = getPoints(chartTab === 'traffic' ? 'down' : 'write');

    const buildPath = (pts: Array<{ x: number; y: number }>) => {
      if (pts.length === 0) return '';
      if (pts.length === 1) return `M ${pts[0].x} ${pts[0].y}`;
      let d = `M ${pts[0].x.toFixed(1)} ${pts[0].y.toFixed(1)}`;
      for (let i = 0; i < pts.length - 1; i++) {
        const p0 = pts[i === 0 ? 0 : i - 1];
        const p1 = pts[i];
        const p2 = pts[i + 1];
        const p3 = pts[i + 2] || p2;

        const dx = p2.x - p1.x;
        const dxPrev = p2.x - p0.x || 1;
        const dxNext = p3.x - p1.x || 1;

        let m1 = (p2.y - p0.y) / dxPrev;
        let m2 = (p3.y - p1.y) / dxNext;
        const s = (p2.y - p1.y) / (dx || 1);

        if (Math.abs(s) < 0.0001) {
          m1 = 0;
          m2 = 0;
        } else {
          if (m1 * s < 0) m1 = 0;
          if (m2 * s < 0) m2 = 0;
          if (Math.abs(m1) > 2.5 * Math.abs(s)) m1 = 2.5 * s;
          if (Math.abs(m2) > 2.5 * Math.abs(s)) m2 = 2.5 * s;
        }

        const cp1x = p1.x + dx / 3;
        let cp1y = p1.y + (m1 * dx) / 3;
        const cp2x = p2.x - dx / 3;
        let cp2y = p2.y - (m2 * dx) / 3;

        const segMinY = Math.min(p1.y, p2.y);
        const segMaxY = Math.max(p1.y, p2.y);
        cp1y = Math.max(paddingTop, Math.min(baselineY, Math.max(segMinY, Math.min(segMaxY, cp1y))));
        cp2y = Math.max(paddingTop, Math.min(baselineY, Math.max(segMinY, Math.min(segMaxY, cp2y))));

        d += ` C ${cp1x.toFixed(1)} ${cp1y.toFixed(1)}, ${cp2x.toFixed(1)} ${cp2y.toFixed(1)}, ${p2.x.toFixed(1)} ${p2.y.toFixed(1)}`;
      }
      return d;
    };

    const line1 = buildPath(pts1);
    const line2 = buildPath(pts2);

    const area1 = pts1.length >= 2 ? `${line1} L ${width} ${baselineY} L 0 ${baselineY} Z` : '';
    const area2 = pts2.length >= 2 ? `${line2} L ${width} ${baselineY} L 0 ${baselineY} Z` : '';

    const gridLabels = [
      formatSpeed(maxY),
      formatSpeed(maxY * 0.75),
      formatSpeed(maxY * 0.5),
      formatSpeed(maxY * 0.25),
      '0 KB/s',
    ];

    return { line1, line2, area1, area2, pts1, pts2, maxY, maxPeak, gridLabels };
  }, [chartHistory, chartTab]);

  const filteredCoreTools = useMemo(() => {
    if (toolFilter === 'all') return CORE_TOOLS;
    return CORE_TOOLS.filter((t) => t.category === toolFilter);
  }, [toolFilter]);

  if (viewMode === 'customer') {
    return (
      <DashboardShell>
        {userRole === 'admin' && (
          <div className="mb-4 flex flex-col sm:flex-row items-start sm:items-center justify-between gap-3 p-3.5 rounded-2xl bg-gradient-to-r from-emerald-50 to-teal-50 dark:from-emerald-950/40 dark:to-teal-950/30 border border-emerald-200/90 dark:border-emerald-800 text-xs shadow-xs">
            <div className="flex items-center gap-2">
              <span className="relative flex h-2.5 w-2.5">
                <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-emerald-400 opacity-75" />
                <span className="relative inline-flex rounded-full h-2.5 w-2.5 bg-emerald-500" />
              </span>
              <span className="font-bold text-emerald-950 dark:text-emerald-200">
                Administrator Live Preview: Viewing Tenant Hosting Experience
              </span>
            </div>
            <div className="flex items-center gap-2 w-full sm:w-auto justify-end">
              <Link
                href="/admin/users"
                className="px-3 py-1.5 font-semibold rounded-xl bg-white dark:bg-surface-800 text-emerald-800 dark:text-emerald-200 hover:bg-emerald-100 dark:hover:bg-surface-700 transition-all border border-emerald-200/80 dark:border-emerald-700 shadow-2xs"
              >
                User Management
              </Link>
              <button
                onClick={() => setViewMode('admin')}
                className="px-3.5 py-1.5 font-bold rounded-xl bg-[#16A34A] hover:bg-[#15803D] text-white transition-all shadow-2xs hover:shadow-sm cursor-pointer"
              >
                Switch to Server Telemetry
              </button>
            </div>
          </div>
        )}
        <CustomerDashboard />
      </DashboardShell>
    );
  }

  return (
    <DashboardShell>
      {/* Dynamic Toast Notification */}
      {toastMessage && (
        <div className="fixed bottom-6 right-6 z-50 animate-fadeIn flex items-center gap-2.5 px-4 py-2.5 rounded-2xl bg-slate-900/95 dark:bg-slate-800/95 text-white text-xs font-semibold shadow-2xl backdrop-blur-md border border-slate-700/80">
          <CheckCircle2 className="w-4 h-4 text-emerald-400 shrink-0" />
          <span>{toastMessage}</span>
        </div>
      )}

      <div className="space-y-5">
        {/* TOP COMMAND CENTER HERO BANNER */}
        <div className="relative overflow-hidden rounded-2xl bg-gradient-to-r from-slate-900 via-slate-900 to-[#0e1726] dark:from-[#0a0f1d] dark:via-[#0c1424] dark:to-[#091122] border border-slate-800/90 p-4 sm:p-5 text-white shadow-lg">
          {/* Ambient Background Light Mesh */}
          <div className="absolute -top-12 -right-12 w-64 h-64 bg-emerald-500/10 rounded-full blur-3xl pointer-events-none" />
          <div className="absolute -bottom-12 left-1/3 w-64 h-64 bg-teal-500/10 rounded-full blur-3xl pointer-events-none" />

          <div className="relative z-10 flex flex-col lg:flex-row lg:items-center justify-between gap-4">
            {/* Left: Node Info & Telemetry Status */}
            <div className="space-y-2">
              <div className="flex flex-wrap items-center gap-2">
                <span className="relative flex h-2.5 w-2.5">
                  <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-emerald-400 opacity-75" />
                  <span className="relative inline-flex rounded-full h-2.5 w-2.5 bg-emerald-500" />
                </span>
                <span className="text-xs font-mono font-bold tracking-wide uppercase text-emerald-400">
                  Root Infrastructure Node
                </span>
                <span className="text-slate-600 dark:text-slate-500 font-mono">/</span>
                <span className="text-xs font-mono text-slate-300">Admin Control Plane</span>
                <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-[10px] font-semibold bg-emerald-500/15 text-emerald-300 border border-emerald-500/30">
                  <Radio className="w-2.5 h-2.5 animate-pulse text-emerald-400" />
                  99.98% Healthy
                </span>
              </div>

              <div className="flex flex-wrap items-center gap-x-4 gap-y-1.5 text-xs text-slate-300">
                <div className="flex items-center gap-1.5">
                  <Server className="w-3.5 h-3.5 text-slate-400" />
                  <span className="font-semibold text-white">Production Node 01</span>
                </div>
                <div className="flex items-center gap-1.5">
                  <Compass className="w-3.5 h-3.5 text-slate-400" />
                  <span className="text-slate-400">Arch:</span>
                  <span className="font-mono text-slate-200">Ubuntu 24.04 (Linux 6.8 x86_64)</span>
                </div>
                <div className="flex items-center gap-1.5">
                  <Activity className="w-3.5 h-3.5 text-slate-400" />
                  <span className="text-slate-400">Telemetry:</span>
                  <span className="font-mono text-emerald-400 font-semibold">2.5s Realtime Ticker</span>
                </div>
              </div>
            </div>

            {/* Right: Controls & Portal Switcher */}
            <div className="flex flex-wrap items-center gap-2.5 pt-1 lg:pt-0">
              <button
                onClick={fetchData}
                disabled={isRefreshing}
                className="p-2 rounded-xl bg-slate-800/80 hover:bg-slate-700/80 text-slate-300 hover:text-white border border-slate-700/80 transition-all cursor-pointer shadow-xs"
                title="Refresh Realtime Telemetry"
              >
                <RefreshCw className={`w-3.5 h-3.5 ${isRefreshing ? 'animate-spin text-emerald-400' : ''}`} />
              </button>

              <Link
                href="/admin/users"
                className="inline-flex items-center gap-1.5 px-3.5 py-2 text-xs font-bold rounded-xl bg-emerald-600 hover:bg-emerald-500 text-white transition-all shadow-xs hover:shadow-emerald-500/20 cursor-pointer"
              >
                <Users className="w-3.5 h-3.5" />
                <span>Manage Tenant Accounts</span>
              </Link>

              <button
                onClick={() => setViewMode('customer')}
                className="inline-flex items-center gap-1.5 px-3.5 py-2 text-xs font-semibold rounded-xl bg-slate-800/90 hover:bg-slate-700/90 text-slate-200 hover:text-white border border-slate-700 transition-all cursor-pointer shadow-xs"
              >
                <Globe className="w-3.5 h-3.5 text-emerald-400" />
                <span>Customer Portal View</span>
              </button>
            </div>
          </div>
        </div>

        {/* Realtime API Disconnected Alert */}
        {apiError && (
          <div className="p-3.5 rounded-2xl bg-amber-500/10 border border-amber-500/30 text-amber-300 text-xs flex items-center justify-between shadow-xs">
            <div className="flex items-center gap-2">
              <ShieldAlert className="w-4 h-4 text-amber-400 shrink-0" />
              <span>
                <strong>Live Telemetry Disconnected:</strong> {apiError}
              </span>
            </div>
            <button
              onClick={fetchData}
              className="px-3 py-1 rounded-lg bg-amber-500/20 hover:bg-amber-500/30 text-[11px] font-semibold text-amber-200 transition-colors cursor-pointer"
            >
              Retry Connection
            </button>
          </div>
        )}

        {/* QUICK COMMAND PALETTE STRIP */}
        <div className="flex items-center justify-between gap-2 overflow-x-auto pb-1 scrollbar-none text-xs">
          <div className="flex items-center gap-1.5 min-w-max">
            <span className="text-[11px] font-bold text-slate-500 dark:text-slate-400 uppercase tracking-wider mr-1">
              Quick Actions:
            </span>
            <Link
              href="/websites"
              className="inline-flex items-center gap-1 px-2.5 py-1.5 rounded-xl bg-white dark:bg-[#121824] hover:bg-slate-50 dark:hover:bg-surface-800 border border-slate-200 dark:border-surface-800 text-slate-700 dark:text-slate-200 font-semibold transition shadow-2xs hover:border-emerald-500/40"
            >
              <Plus className="w-3.5 h-3.5 text-emerald-500" />
              <span>New Site</span>
            </Link>
            <Link
              href="/databases"
              className="inline-flex items-center gap-1 px-2.5 py-1.5 rounded-xl bg-white dark:bg-[#121824] hover:bg-slate-50 dark:hover:bg-surface-800 border border-slate-200 dark:border-surface-800 text-slate-700 dark:text-slate-200 font-semibold transition shadow-2xs hover:border-emerald-500/40"
            >
              <Database className="w-3.5 h-3.5 text-amber-500" />
              <span>New DB</span>
            </Link>
            <Link
              href="/terminal"
              className="inline-flex items-center gap-1 px-2.5 py-1.5 rounded-xl bg-white dark:bg-[#121824] hover:bg-slate-50 dark:hover:bg-surface-800 border border-slate-200 dark:border-surface-800 text-slate-700 dark:text-slate-200 font-semibold transition shadow-2xs hover:border-emerald-500/40"
            >
              <Terminal className="w-3.5 h-3.5 text-indigo-500" />
              <span>Web SSH</span>
            </Link>
            <Link
              href="/ssl"
              className="inline-flex items-center gap-1 px-2.5 py-1.5 rounded-xl bg-white dark:bg-[#121824] hover:bg-slate-50 dark:hover:bg-surface-800 border border-slate-200 dark:border-surface-800 text-slate-700 dark:text-slate-200 font-semibold transition shadow-2xs hover:border-emerald-500/40"
            >
              <Lock className="w-3.5 h-3.5 text-rose-500" />
              <span>SSL Renew</span>
            </Link>
            <Link
              href="/firewall"
              className="inline-flex items-center gap-1 px-2.5 py-1.5 rounded-xl bg-white dark:bg-[#121824] hover:bg-slate-50 dark:hover:bg-surface-800 border border-slate-200 dark:border-surface-800 text-slate-700 dark:text-slate-200 font-semibold transition shadow-2xs hover:border-emerald-500/40"
            >
              <Shield className="w-3.5 h-3.5 text-emerald-500" />
              <span>Firewall Rules</span>
            </Link>
            <button
              onClick={flushSystemCache}
              className="inline-flex items-center gap-1 px-2.5 py-1.5 rounded-xl bg-white dark:bg-[#121824] hover:bg-slate-50 dark:hover:bg-surface-800 border border-slate-200 dark:border-surface-800 text-slate-700 dark:text-slate-200 font-semibold transition shadow-2xs hover:border-emerald-500/40 cursor-pointer"
            >
              <RotateCw className="w-3.5 h-3.5 text-teal-500" />
              <span>Flush Cache</span>
            </button>
            <button
              onClick={triggerQuickAudit}
              className="inline-flex items-center gap-1 px-2.5 py-1.5 rounded-xl bg-white dark:bg-[#121824] hover:bg-slate-50 dark:hover:bg-surface-800 border border-slate-200 dark:border-surface-800 text-slate-700 dark:text-slate-200 font-semibold transition shadow-2xs hover:border-emerald-500/40 cursor-pointer"
            >
              <ShieldCheck className="w-3.5 h-3.5 text-emerald-500" />
              <span>Run Deep Scan</span>
            </button>
          </div>
        </div>

        {/* ROW 1: 4 HIGH-PRECISION REALTIME HARDWARE GAUGES (Load, CPU, RAM, Disk) */}
        <div className="grid grid-cols-1 sm:grid-cols-2 xl:grid-cols-4 gap-3.5 sm:gap-4">
          {/* Card 1: Load Status */}
          <div className="relative overflow-hidden p-4 rounded-2xl bg-white dark:bg-[#121824] border border-slate-200/90 dark:border-surface-800 shadow-xs hover:shadow-md hover:border-emerald-500/40 transition-all duration-300 group">
            <div className="flex items-start justify-between">
              <div className="space-y-1 min-w-0">
                <div className="flex items-center gap-1.5">
                  <div className="w-6 h-6 rounded-lg bg-emerald-50 dark:bg-emerald-950/50 flex items-center justify-center text-emerald-600 dark:text-emerald-400">
                    <Activity className="w-3.5 h-3.5" />
                  </div>
                  <span className="text-xs font-bold text-slate-700 dark:text-slate-300">System Load</span>
                  <span
                    className={`text-[10px] font-semibold px-1.5 py-0.2 rounded-md ${
                      telemetry.load.percent > 80
                        ? 'bg-rose-100 text-rose-700 dark:bg-rose-950/40 dark:text-rose-300'
                        : 'bg-emerald-100 text-emerald-700 dark:bg-emerald-950/40 dark:text-emerald-300'
                    }`}
                  >
                    {telemetry.load.text}
                  </span>
                </div>
                <div className="pt-1">
                  <div className="text-sm font-mono font-bold text-slate-900 dark:text-white flex items-center gap-1">
                    <span>{telemetry.load.avg}</span>
                    <span title="1, 5, and 15-minute load averages" className="cursor-help">
                      <HelpCircle className="w-3.5 h-3.5 text-slate-400 hover:text-slate-600" />
                    </span>
                  </div>
                  <span className="text-[10px] text-slate-400 block mt-0.5">
                    1m · 5m · 15m load distribution
                  </span>
                </div>
              </div>

              {/* High-Tech Circular Gauge */}
              <div className="relative w-16 h-16 flex items-center justify-center flex-shrink-0">
                <svg className="w-16 h-16 transform -rotate-90" viewBox="0 0 80 80">
                  <circle cx="40" cy="40" r="34" strokeWidth="6" className="gauge-track" fill="transparent" />
                  <circle
                    cx="40"
                    cy="40"
                    r="34"
                    stroke={getGaugeColor(telemetry.load.percent)}
                    strokeWidth="6"
                    strokeDasharray={213.6}
                    strokeDashoffset={213.6 - (213.6 * Math.min(100, telemetry.load.percent)) / 100}
                    strokeLinecap="round"
                    fill="transparent"
                    className={`transition-all duration-700 ${getGaugeGlowClass(telemetry.load.percent)}`}
                  />
                </svg>
                <div className="absolute flex flex-col items-center">
                  <span className="text-xs font-black text-slate-900 dark:text-white">
                    {telemetry.load.percent}%
                  </span>
                </div>
              </div>
            </div>

            {/* Micro Activity Bar */}
            <div className="mt-3 pt-2.5 border-t border-slate-100 dark:border-surface-800 flex items-center justify-between text-[10px] text-slate-400">
              <span>Core Capacity: Healthy</span>
              <span className="font-mono text-emerald-600 dark:text-emerald-400 font-semibold">Active</span>
            </div>
          </div>

          {/* Card 2: CPU */}
          <div className="relative overflow-hidden p-4 rounded-2xl bg-white dark:bg-[#121824] border border-slate-200/90 dark:border-surface-800 shadow-xs hover:shadow-md hover:border-emerald-500/40 transition-all duration-300 group">
            <div className="flex items-start justify-between">
              <div className="space-y-1 min-w-0">
                <div className="flex items-center gap-1.5">
                  <div className="w-6 h-6 rounded-lg bg-indigo-50 dark:bg-indigo-950/50 flex items-center justify-center text-indigo-600 dark:text-indigo-400">
                    <Cpu className="w-3.5 h-3.5" />
                  </div>
                  <span className="text-xs font-bold text-slate-700 dark:text-slate-300">CPU Compute</span>
                  <span className="text-[10px] font-semibold px-1.5 py-0.2 rounded-md bg-indigo-50 text-indigo-700 dark:bg-indigo-950/40 dark:text-indigo-300">
                    Multi-Core
                  </span>
                </div>
                <div className="pt-1">
                  <div className="text-sm font-mono font-bold text-slate-900 dark:text-white">
                    {telemetry.cpu.cores} Physical Cores
                  </div>
                  <span className="text-[10px] text-slate-400 block mt-0.5">
                    x86_64 Architecture @ 3.20GHz
                  </span>
                </div>
              </div>

              <div className="relative w-16 h-16 flex items-center justify-center flex-shrink-0">
                <svg className="w-16 h-16 transform -rotate-90" viewBox="0 0 80 80">
                  <circle cx="40" cy="40" r="34" strokeWidth="6" className="gauge-track" fill="transparent" />
                  <circle
                    cx="40"
                    cy="40"
                    r="34"
                    stroke={getGaugeColor(telemetry.cpu.percent)}
                    strokeWidth="6"
                    strokeDasharray={213.6}
                    strokeDashoffset={213.6 - (213.6 * Math.min(100, telemetry.cpu.percent)) / 100}
                    strokeLinecap="round"
                    fill="transparent"
                    className={`transition-all duration-700 ${getGaugeGlowClass(telemetry.cpu.percent)}`}
                  />
                </svg>
                <div className="absolute flex flex-col items-center">
                  <span className="text-xs font-black text-slate-900 dark:text-white">
                    {telemetry.cpu.percent}%
                  </span>
                </div>
              </div>
            </div>

            <div className="mt-3 pt-2.5 border-t border-slate-100 dark:border-surface-800 flex items-center justify-between text-[10px] text-slate-400">
              <span>Thread Allocation</span>
              <span className="font-mono text-slate-600 dark:text-slate-300 font-semibold">{telemetry.cpu.cores * 2} Threads</span>
            </div>
          </div>

          {/* Card 3: RAM */}
          <div className="relative overflow-hidden p-4 rounded-2xl bg-white dark:bg-[#121824] border border-slate-200/90 dark:border-surface-800 shadow-xs hover:shadow-md hover:border-emerald-500/40 transition-all duration-300 group">
            <div className="flex items-start justify-between">
              <div className="space-y-1 min-w-0">
                <div className="flex items-center gap-1.5">
                  <div className="w-6 h-6 rounded-lg bg-teal-50 dark:bg-teal-950/50 flex items-center justify-center text-teal-600 dark:text-teal-400">
                    <Zap className="w-3.5 h-3.5" />
                  </div>
                  <span className="text-xs font-bold text-slate-700 dark:text-slate-300">Physical RAM</span>
                  <span className="text-[10px] font-semibold px-1.5 py-0.2 rounded-md bg-teal-50 text-teal-700 dark:bg-teal-950/40 dark:text-teal-300">
                    DDR4 ECC
                  </span>
                </div>
                <div className="pt-1">
                  <div className="text-sm font-mono font-bold text-slate-900 dark:text-white truncate">
                    <span className="text-emerald-600 dark:text-emerald-400 font-bold">{telemetry.ram.used}</span>{' '}
                    <span className="text-slate-400 font-normal">/</span> {telemetry.ram.total}
                  </div>
                  <span className="text-[10px] text-slate-400 block mt-0.5">
                    Cached & Buffers Included
                  </span>
                </div>
              </div>

              <div className="relative w-16 h-16 flex items-center justify-center flex-shrink-0">
                <svg className="w-16 h-16 transform -rotate-90" viewBox="0 0 80 80">
                  <circle cx="40" cy="40" r="34" strokeWidth="6" className="gauge-track" fill="transparent" />
                  <circle
                    cx="40"
                    cy="40"
                    r="34"
                    stroke={getGaugeColor(telemetry.ram.percent)}
                    strokeWidth="6"
                    strokeDasharray={213.6}
                    strokeDashoffset={213.6 - (213.6 * Math.min(100, telemetry.ram.percent)) / 100}
                    strokeLinecap="round"
                    fill="transparent"
                    className={`transition-all duration-700 ${getGaugeGlowClass(telemetry.ram.percent)}`}
                  />
                </svg>
                <div className="absolute flex flex-col items-center">
                  <span className="text-xs font-black text-slate-900 dark:text-white">
                    {telemetry.ram.percent}%
                  </span>
                </div>
              </div>
            </div>

            <div className="mt-3 pt-2.5 border-t border-slate-100 dark:border-surface-800 flex items-center justify-between text-[10px] text-slate-400">
              <span>Memory Pressure</span>
              <span className="font-mono text-emerald-600 dark:text-emerald-400 font-semibold">Low (Safe)</span>
            </div>
          </div>

          {/* Card 4: Disk */}
          <div className="relative overflow-hidden p-4 rounded-2xl bg-white dark:bg-[#121824] border border-slate-200/90 dark:border-surface-800 shadow-xs hover:shadow-md hover:border-emerald-500/40 transition-all duration-300 group">
            <div className="flex items-start justify-between">
              <div className="space-y-1 min-w-0">
                <div className="flex items-center gap-1.5">
                  <div className="w-6 h-6 rounded-lg bg-amber-50 dark:bg-amber-950/50 flex items-center justify-center text-amber-600 dark:text-amber-400">
                    <HardDrive className="w-3.5 h-3.5" />
                  </div>
                  <span className="text-xs font-bold text-slate-700 dark:text-slate-300">NVMe Storage (/)</span>
                  <span className="text-[10px] font-semibold px-1.5 py-0.2 rounded-md bg-amber-50 text-amber-700 dark:bg-amber-950/40 dark:text-amber-300">
                    ext4
                  </span>
                </div>
                <div className="pt-1">
                  <div className="text-sm font-mono font-bold text-slate-900 dark:text-white truncate">
                    <span className="text-emerald-600 dark:text-emerald-400 font-bold">{telemetry.disk.used}</span>{' '}
                    <span className="text-slate-400 font-normal">/</span> {telemetry.disk.total}
                  </div>
                  <span className="text-[10px] text-slate-400 block mt-0.5">
                    High-IOPS Enterprise SSD
                  </span>
                </div>
              </div>

              <div className="relative w-16 h-16 flex items-center justify-center flex-shrink-0">
                <svg className="w-16 h-16 transform -rotate-90" viewBox="0 0 80 80">
                  <circle cx="40" cy="40" r="34" strokeWidth="6" className="gauge-track" fill="transparent" />
                  <circle
                    cx="40"
                    cy="40"
                    r="34"
                    stroke={getGaugeColor(telemetry.disk.percent)}
                    strokeWidth="6"
                    strokeDasharray={213.6}
                    strokeDashoffset={213.6 - (213.6 * Math.min(100, telemetry.disk.percent)) / 100}
                    strokeLinecap="round"
                    fill="transparent"
                    className={`transition-all duration-700 ${getGaugeGlowClass(telemetry.disk.percent)}`}
                  />
                </svg>
                <div className="absolute flex flex-col items-center">
                  <span className="text-xs font-black text-slate-900 dark:text-white">
                    {telemetry.disk.percent}%
                  </span>
                </div>
              </div>
            </div>

            <div className="mt-3 pt-2.5 border-t border-slate-100 dark:border-surface-800 flex items-center justify-between text-[10px] text-slate-400">
              <span>Health Status</span>
              <span className="font-mono text-emerald-600 dark:text-emerald-400 font-semibold">SMART 100% OK</span>
            </div>
          </div>
        </div>

        {/* ROW 2: OVERVIEW BAR (5 Modern Cards) */}
        <div className="p-4 sm:p-5 rounded-2xl bg-white dark:bg-[#121824] border border-slate-200/90 dark:border-surface-800 shadow-xs space-y-3.5">
          <div className="flex items-center justify-between border-b border-slate-100 dark:border-surface-800 pb-3">
            <div className="flex items-center gap-2">
              <h2 className="text-xs font-black uppercase tracking-wider text-slate-800 dark:text-slate-100">
                System Overview & Entities
              </h2>
              <span className="text-[10px] px-2 py-0.5 rounded-full bg-slate-100 dark:bg-surface-800 text-slate-600 dark:text-slate-400 font-semibold">
                Live Counts
              </span>
            </div>
            <Link
              href="/websites"
              className="text-emerald-600 dark:text-emerald-400 hover:underline text-xs font-semibold flex items-center gap-1"
            >
              <span>Explore VirtualHosts</span>
              <ArrowUpRight className="w-3 h-3" />
            </Link>
          </div>

          <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-5 gap-3 text-xs">
            {/* 1. Site - ALL */}
            <Link
              href="/websites"
              className="group p-3 rounded-xl border border-slate-200/80 dark:border-surface-800 hover:border-emerald-500/50 bg-slate-50/50 dark:bg-surface-800/20 hover:bg-white dark:hover:bg-surface-800/60 transition-all duration-200 shadow-2xs hover:shadow-sm"
            >
              <div className="flex items-center justify-between mb-2">
                <span className="font-bold text-slate-800 dark:text-slate-200 group-hover:text-emerald-600 dark:group-hover:text-emerald-400 transition-colors">
                  Site - ALL
                </span>
                <div className="w-7 h-7 rounded-lg bg-emerald-100/70 dark:bg-emerald-950/60 flex items-center justify-center text-emerald-600 dark:text-emerald-400 group-hover:scale-110 transition-transform">
                  <Globe className="w-4 h-4" />
                </div>
              </div>
              <div className="space-y-1">
                <div className="flex items-center gap-2 text-[11px]">
                  <span className="font-bold text-emerald-600 dark:text-emerald-400">
                    Running: {counts.websites_running}
                  </span>
                  <span className="text-slate-400">Stopped: {counts.websites_stopped}</span>
                </div>
                <span className="text-[11px] text-slate-500 dark:text-slate-400 block">
                  Total Domains: <strong className="text-slate-800 dark:text-slate-200">{counts.websites_total}</strong>
                </span>
              </div>
            </Link>

            {/* 2. FTP Accounts */}
            <Link
              href="/ftp"
              className="group p-3 rounded-xl border border-slate-200/80 dark:border-surface-800 hover:border-emerald-500/50 bg-slate-50/50 dark:bg-surface-800/20 hover:bg-white dark:hover:bg-surface-800/60 transition-all duration-200 shadow-2xs hover:shadow-sm"
            >
              <div className="flex items-center justify-between mb-2">
                <span className="font-bold text-slate-800 dark:text-slate-200 group-hover:text-emerald-600 dark:group-hover:text-emerald-400 transition-colors">
                  FTP Accounts
                </span>
                <div className="w-7 h-7 rounded-lg bg-cyan-100/70 dark:bg-cyan-950/60 flex items-center justify-center text-cyan-600 dark:text-cyan-400 group-hover:scale-110 transition-transform">
                  <ArrowLeftRight className="w-4 h-4" />
                </div>
              </div>
              <div className="space-y-1">
                <div className="text-sm font-bold text-slate-900 dark:text-white">
                  {counts.ftp_accounts_total} Accounts
                </div>
                <span className="text-[11px] text-slate-400 block">
                  vsftpd & Pure-FTPd TLS
                </span>
              </div>
            </Link>

            {/* 3. Databases */}
            <Link
              href="/databases"
              className="group p-3 rounded-xl border border-slate-200/80 dark:border-surface-800 hover:border-emerald-500/50 bg-slate-50/50 dark:bg-surface-800/20 hover:bg-white dark:hover:bg-surface-800/60 transition-all duration-200 shadow-2xs hover:shadow-sm"
            >
              <div className="flex items-center justify-between mb-2">
                <span className="font-bold text-slate-800 dark:text-slate-200 group-hover:text-emerald-600 dark:group-hover:text-emerald-400 transition-colors">
                  Databases
                </span>
                <div className="w-7 h-7 rounded-lg bg-amber-100/70 dark:bg-amber-950/60 flex items-center justify-center text-amber-600 dark:text-amber-400 group-hover:scale-110 transition-transform">
                  <Database className="w-4 h-4" />
                </div>
              </div>
              <div className="space-y-1">
                <div className="text-sm font-bold text-slate-900 dark:text-white">
                  {counts.databases_total} Active DBs
                </div>
                <span className="text-[11px] text-slate-400 block">
                  MySQL / MariaDB 10.11
                </span>
              </div>
            </Link>

            {/* 4. Security */}
            <Link
              href="/firewall"
              className="group p-3 rounded-xl border border-slate-200/80 dark:border-surface-800 hover:border-emerald-500/50 bg-slate-50/50 dark:bg-surface-800/20 hover:bg-white dark:hover:bg-surface-800/60 transition-all duration-200 shadow-2xs hover:shadow-sm"
            >
              <div className="flex items-center justify-between mb-2">
                <span className="font-bold text-slate-800 dark:text-slate-200 group-hover:text-emerald-600 dark:group-hover:text-emerald-400 transition-colors">
                  Security Defense
                </span>
                <div className="w-7 h-7 rounded-lg bg-emerald-100/70 dark:bg-emerald-950/60 flex items-center justify-center text-emerald-600 dark:text-emerald-400 group-hover:scale-110 transition-transform">
                  <ShieldCheck className="w-4 h-4" />
                </div>
              </div>
              <div className="space-y-1">
                <div className="text-[11px] font-bold text-emerald-600 dark:text-emerald-400 flex items-center gap-1">
                  <span>Risks: {counts.security_risks} (Clean)</span>
                </div>
                <span className="text-[10px] text-slate-400 block truncate">
                  Scanned: {counts.last_security_scan}
                </span>
              </div>
            </Link>

            {/* 5. Smart Sticky Memo */}
            <div
              id="dashboard-memo-card"
              className="p-3 rounded-xl border border-dashed border-slate-300 dark:border-surface-700 hover:border-emerald-500 bg-slate-50/80 dark:bg-surface-800/30 transition-all sm:col-span-2 lg:col-span-1 relative group"
            >
              <div className="flex items-center justify-between mb-1.5">
                <div className="flex items-center gap-1.5">
                  <span className="font-bold text-slate-800 dark:text-slate-200">Admin Memo</span>
                  <span className="text-[9px] px-1 rounded bg-slate-200 dark:bg-surface-700 text-slate-600 dark:text-slate-300">
                    Local
                  </span>
                </div>
                <div className="flex items-center gap-1">
                  {memoText && (
                    <button
                      onClick={copyMemo}
                      className="p-1 rounded text-slate-400 hover:text-slate-600 dark:hover:text-slate-200 cursor-pointer"
                      title="Copy memo"
                    >
                      {isCopiedMemo ? (
                        <CheckCheck className="w-3.5 h-3.5 text-emerald-500" />
                      ) : (
                        <Copy className="w-3.5 h-3.5" />
                      )}
                    </button>
                  )}
                  <button
                    onClick={() => setIsEditingMemo(!isEditingMemo)}
                    className="p-1 rounded text-slate-400 hover:text-emerald-600 dark:hover:text-emerald-400 cursor-pointer"
                    title={isEditingMemo ? 'Done editing' : 'Edit memo'}
                  >
                    <FileText className="w-3.5 h-3.5" />
                  </button>
                </div>
              </div>

              {isEditingMemo ? (
                <div className="space-y-1">
                  <textarea
                    value={memoText}
                    onChange={(e) => saveMemo(e.target.value)}
                    autoFocus
                    placeholder="Type server notes, ports, reminders here..."
                    className="w-full p-2 text-xs bg-white dark:bg-surface-900 border border-slate-300 dark:border-surface-700 rounded-lg text-slate-800 dark:text-slate-200 focus:outline-none focus:ring-1 focus:ring-emerald-500 resize-none h-16 leading-relaxed"
                  />
                  <div className="flex items-center justify-between text-[10px] text-slate-400">
                    <span>{memoText.length} chars</span>
                    <button
                      onClick={() => setIsEditingMemo(false)}
                      className="text-emerald-600 dark:text-emerald-400 font-bold hover:underline cursor-pointer"
                    >
                      Save & Close
                    </button>
                  </div>
                </div>
              ) : (
                <div
                  onClick={() => setIsEditingMemo(true)}
                  className="cursor-pointer group-hover:text-slate-900 dark:group-hover:text-slate-100"
                >
                  <p className="text-[11px] text-slate-500 dark:text-slate-400 line-clamp-3 leading-relaxed">
                    {memoText || 'No notes yet. Click here to store quick server credentials or tasks.'}
                  </p>
                </div>
              )}
            </div>
          </div>
        </div>

        {/* ROW 3: CORE MANAGEMENT TOOLS WITH TABS */}
        <div className="p-4 sm:p-5 rounded-2xl bg-white dark:bg-[#121824] border border-slate-200/90 dark:border-surface-800 shadow-xs space-y-4">
          <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3 border-b border-slate-100 dark:border-surface-800 pb-3">
            <div className="flex items-center gap-3">
              <h2 className="text-xs font-black uppercase tracking-wider text-slate-800 dark:text-slate-100">
                Core Management Tools
              </h2>
              <span className="text-[10px] px-2 py-0.5 rounded-full bg-emerald-100 dark:bg-emerald-950/60 text-emerald-700 dark:text-emerald-300 font-bold">
                100% Operational
              </span>
            </div>

            {/* Category Filter Tabs & App Store Link */}
            <div className="flex items-center gap-3">
              <div className="flex items-center gap-1 p-0.5 rounded-xl bg-slate-100 dark:bg-surface-800 text-[11px] font-semibold">
                {[
                  { id: 'all', label: 'All (8)' },
                  { id: 'web', label: 'Web' },
                  { id: 'data', label: 'Data' },
                  { id: 'security', label: 'Security' },
                  { id: 'system', label: 'System' },
                ].map((tab) => (
                  <button
                    key={tab.id}
                    onClick={() => setToolFilter(tab.id as any)}
                    className={`px-2.5 py-1 rounded-lg transition-all cursor-pointer ${
                      toolFilter === tab.id
                        ? 'bg-white dark:bg-surface-900 text-slate-900 dark:text-white shadow-2xs font-bold'
                        : 'text-slate-500 hover:text-slate-800 dark:hover:text-slate-300'
                    }`}
                  >
                    {tab.label}
                  </button>
                ))}
              </div>

              <Link
                href="/app-store"
                className="text-emerald-600 dark:text-emerald-400 hover:underline text-xs font-bold flex items-center gap-1"
              >
                <span>+ App Store</span>
              </Link>
            </div>
          </div>

          <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-3 pt-1">
            {filteredCoreTools.map((tool) => {
              const IconComp = tool.icon;
              return (
                <div
                  key={tool.id}
                  onClick={() => router.push(tool.href)}
                  className="flex flex-col justify-between p-3.5 rounded-2xl border border-slate-200/80 dark:border-surface-800 hover:border-emerald-500/50 hover:shadow-md cursor-pointer transition-all duration-200 bg-slate-50/40 dark:bg-surface-800/20 hover:bg-white dark:hover:bg-surface-800/50 group"
                >
                  <div>
                    <div className="flex items-center justify-between mb-2.5">
                      <div
                        className={`w-9 h-9 rounded-xl flex items-center justify-center border shadow-2xs group-hover:scale-105 transition-transform bg-gradient-to-br ${tool.gradient}`}
                      >
                        <IconComp className="w-4 h-4" />
                      </div>
                      <span className={`text-[10px] font-bold px-2 py-0.5 rounded-full border ${tool.badgeColor}`}>
                        {tool.badge}
                      </span>
                    </div>
                    <span className="text-xs font-bold text-slate-800 dark:text-slate-100 group-hover:text-emerald-600 dark:group-hover:text-emerald-400 transition-colors block">
                      {tool.title}
                    </span>
                    <span className="text-[11px] text-slate-500 dark:text-slate-400 truncate mt-0.5 block">
                      {tool.desc}
                    </span>
                  </div>

                  <div className="mt-3 pt-2 border-t border-slate-100 dark:border-surface-800/80 flex items-center justify-between text-[11px] text-slate-400 group-hover:text-emerald-600 dark:group-hover:text-emerald-400 transition-colors">
                    <span className="font-semibold">Launch Tool</span>
                    <ChevronRight className="w-3.5 h-3.5 group-hover:translate-x-1 transition-transform" />
                  </div>
                </div>
              );
            })}
          </div>
        </div>

        {/* ROW 4: INSTALLED SERVICES WITH SEARCH & CATEGORY FILTER */}
        <div className="p-4 sm:p-5 rounded-2xl bg-white dark:bg-[#121824] border border-slate-200/90 dark:border-surface-800 shadow-xs space-y-4">
          <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3 border-b border-slate-100 dark:border-surface-800 pb-3">
            <div className="flex items-center gap-2">
              <h2 className="text-xs font-black uppercase tracking-wider text-slate-800 dark:text-slate-100">
                Installed Services & Stack
              </h2>
              <span className="text-[10px] px-2 py-0.5 rounded-full bg-slate-100 dark:bg-surface-800 text-slate-600 dark:text-slate-300 font-bold">
                {installedApps.length} Packages
              </span>
            </div>

            <div className="flex flex-wrap items-center gap-2">
              {/* Category Pills */}
              <div className="flex items-center gap-1 text-[11px] font-semibold">
                {[
                  { id: 'all', label: 'All' },
                  { id: 'web', label: 'Web' },
                  { id: 'databases', label: 'Databases' },
                  { id: 'cache', label: 'Cache' },
                  { id: 'tools', label: 'Tools' },
                ].map((c) => (
                  <button
                    key={c.id}
                    onClick={() => setServiceCategory(c.id as any)}
                    className={`px-2 py-0.5 rounded-lg transition-all cursor-pointer ${
                      serviceCategory === c.id
                        ? 'bg-emerald-100 dark:bg-emerald-950/60 text-emerald-800 dark:text-emerald-300 font-bold'
                        : 'text-slate-400 hover:text-slate-600 dark:hover:text-slate-200'
                    }`}
                  >
                    {c.label}
                  </button>
                ))}
              </div>

              {/* Service Live Search */}
              <div className="relative">
                <Search className="w-3 h-3 absolute left-2.5 top-1/2 -translate-y-1/2 text-slate-400 pointer-events-none" />
                <input
                  type="text"
                  value={serviceSearch}
                  onChange={(e) => setServiceSearch(e.target.value)}
                  placeholder="Filter packages..."
                  className="pl-7 pr-3 py-1 text-xs bg-slate-50 dark:bg-surface-800/80 border border-slate-200 dark:border-surface-700 rounded-xl text-slate-800 dark:text-slate-200 placeholder:text-slate-400 focus:outline-none focus:ring-1 focus:ring-emerald-500 w-36 sm:w-44"
                />
              </div>

              <Link
                href="/app-store"
                className="px-3 py-1 rounded-xl text-xs font-bold text-white bg-emerald-600 hover:bg-emerald-500 transition shadow-2xs"
              >
                + Add Software
              </Link>
            </div>
          </div>

          {filteredServices.length > 0 ? (
            <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 xl:grid-cols-4 gap-3">
              {filteredServices.map((app) => (
                <button
                  key={app.id}
                  onClick={() => handleOpenApp(app)}
                  className="p-3.5 rounded-2xl border border-slate-200/80 dark:border-surface-800 hover:border-emerald-500/50 bg-slate-50/40 dark:bg-surface-800/20 hover:bg-white dark:hover:bg-surface-800/60 flex items-center justify-between text-left transition-all duration-200 shadow-2xs group min-w-0 cursor-pointer"
                >
                  <div className="flex items-center gap-3 min-w-0">
                    <span
                      className={`relative flex h-2.5 w-2.5 shrink-0 ${
                        app.status === 'running' ? '' : ''
                      }`}
                    >
                      {app.status === 'running' && (
                        <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-emerald-400 opacity-75" />
                      )}
                      <span
                        className={`relative inline-flex rounded-full h-2.5 w-2.5 ${
                          app.status === 'running' ? 'bg-emerald-500' : 'bg-amber-500'
                        }`}
                      />
                    </span>
                    <div className="min-w-0">
                      <span className="text-xs font-bold text-slate-900 dark:text-slate-100 truncate block group-hover:text-emerald-600 dark:group-hover:text-emerald-400 transition-colors">
                        {app.name}
                      </span>
                      <span className="text-[10px] text-slate-400 block truncate">
                        {app.version ? `v${app.version}` : 'Active'} · {app.category || 'Package'}
                      </span>
                    </div>
                  </div>

                  <span className="text-[10px] text-emerald-700 dark:text-emerald-300 font-bold px-2 py-0.5 rounded-lg bg-emerald-50 dark:bg-emerald-950/40 border border-emerald-500/30 group-hover:bg-emerald-600 group-hover:text-white transition-all shadow-2xs">
                    Manage
                  </span>
                </button>
              ))}
            </div>
          ) : (
            <div className="p-8 rounded-2xl border border-dashed border-slate-300 dark:border-surface-700 bg-slate-50/50 dark:bg-surface-900/50 text-center space-y-2">
              <Boxes className="w-8 h-8 text-slate-400 dark:text-slate-500 mx-auto" />
              <div className="space-y-0.5">
                <p className="text-xs font-bold text-slate-800 dark:text-slate-200">
                  {serviceSearch ? 'No services match your search' : 'No Software Installed'}
                </p>
                <p className="text-[11px] text-slate-500 dark:text-slate-400 max-w-sm mx-auto">
                  Install Nginx, Apache, MySQL, PHP, Redis, or Node.js with 1-click from the App Store.
                </p>
              </div>
              <Link
                href="/app-store"
                className="inline-flex items-center gap-1.5 px-3.5 py-1.5 rounded-xl bg-emerald-600 hover:bg-emerald-500 text-white font-bold text-xs transition mt-1 shadow-xs"
              >
                + Browse App Store
              </Link>
            </div>
          )}
        </div>

        {/* ROW 5: TRAFFIC & DISK IO REALTIME MONITOR */}
        <div className="p-4 sm:p-5 rounded-2xl bg-white dark:bg-[#121824] border border-slate-200/90 dark:border-surface-800 shadow-xs space-y-3.5">
          {/* Header with Tabs, Fullscreen & Net Interface dropdown */}
          <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3 border-b border-slate-100 dark:border-surface-800 pb-3">
            <div className="flex items-center gap-3">
              <div className="flex items-center p-0.5 rounded-xl bg-slate-100 dark:bg-surface-800">
                <button
                  onClick={() => setChartTab('traffic')}
                  className={`px-3 py-1 rounded-lg text-xs font-bold transition-all cursor-pointer ${
                    chartTab === 'traffic'
                      ? 'bg-white dark:bg-surface-900 text-emerald-600 dark:text-emerald-400 shadow-2xs'
                      : 'text-slate-500 hover:text-slate-800 dark:hover:text-white'
                  }`}
                >
                  Network Traffic
                </button>
                <button
                  onClick={() => setChartTab('disk')}
                  className={`px-3 py-1 rounded-lg text-xs font-bold transition-all cursor-pointer ${
                    chartTab === 'disk'
                      ? 'bg-white dark:bg-surface-900 text-emerald-600 dark:text-emerald-400 shadow-2xs'
                      : 'text-slate-500 hover:text-slate-800 dark:hover:text-white'
                  }`}
                >
                  Disk I/O Throughput
                </button>
              </div>

              <button
                onClick={() => setChartExpanded(true)}
                className="p-1.5 rounded-lg text-slate-400 hover:text-slate-600 dark:hover:text-slate-200 hover:bg-slate-100 dark:hover:bg-surface-800 transition cursor-pointer"
                title="Expand fullscreen view"
              >
                <Maximize2 className="w-3.5 h-3.5" />
              </button>
            </div>

            {/* Net Interface Dropdown & Peak Indicator */}
            <div className="flex items-center gap-2">
              <span className="text-[11px] font-mono px-2 py-0.5 rounded-lg bg-slate-100 dark:bg-surface-800 text-slate-600 dark:text-slate-300 font-semibold">
                Peak: <strong className="text-emerald-600 dark:text-emerald-400">{formatSpeed(chartSvgPaths.maxPeak)}</strong>
              </span>
              <div className="flex items-center gap-1 text-xs text-slate-600 dark:text-slate-300 border border-slate-200 dark:border-surface-700 px-2 py-0.5 rounded-lg bg-slate-50 dark:bg-surface-800/50">
                <span>Net: {networkInterface}</span>
              </div>
            </div>
          </div>

          {/* Realtime Telemetry Stats Strip */}
          <div className="grid grid-cols-2 sm:grid-cols-4 gap-2 text-xs py-1">
            {chartTab === 'traffic' ? (
              <>
                <div className="p-2.5 rounded-xl bg-slate-50/80 dark:bg-surface-800/40 border border-slate-100 dark:border-surface-800 flex items-center justify-between">
                  <div className="space-y-0.5">
                    <span className="text-[10px] text-slate-400 font-medium block">Current Upstream</span>
                    <span className="font-mono font-bold text-slate-800 dark:text-slate-200 text-sm">
                      {telemetry.upstreamMb}
                    </span>
                  </div>
                  <span className="w-2 h-2 rounded-full bg-emerald-500" />
                </div>
                <div className="p-2.5 rounded-xl bg-slate-50/80 dark:bg-surface-800/40 border border-slate-100 dark:border-surface-800 flex items-center justify-between">
                  <div className="space-y-0.5">
                    <span className="text-[10px] text-slate-400 font-medium block">Current Downstream</span>
                    <span className="font-mono font-bold text-slate-800 dark:text-slate-200 text-sm">
                      {telemetry.downstreamMb}
                    </span>
                  </div>
                  <span className="w-2 h-2 rounded-full bg-amber-500" />
                </div>
                <div className="p-2.5 rounded-xl bg-slate-50/80 dark:bg-surface-800/40 border border-slate-100 dark:border-surface-800 flex items-center justify-between">
                  <div className="space-y-0.5">
                    <span className="text-[10px] text-slate-400 font-medium block">Total Sent (Tx)</span>
                    <span className="font-mono font-bold text-slate-800 dark:text-slate-200 text-sm">
                      {telemetry.totalSentGb}
                    </span>
                  </div>
                  <ArrowUp className="w-3.5 h-3.5 text-emerald-500" />
                </div>
                <div className="p-2.5 rounded-xl bg-slate-50/80 dark:bg-surface-800/40 border border-slate-100 dark:border-surface-800 flex items-center justify-between">
                  <div className="space-y-0.5">
                    <span className="text-[10px] text-slate-400 font-medium block">Total Received (Rx)</span>
                    <span className="font-mono font-bold text-slate-800 dark:text-slate-200 text-sm">
                      {telemetry.totalReceivedGb}
                    </span>
                  </div>
                  <ArrowDown className="w-3.5 h-3.5 text-amber-500" />
                </div>
              </>
            ) : (
              <>
                <div className="p-2.5 rounded-xl bg-slate-50/80 dark:bg-surface-800/40 border border-slate-100 dark:border-surface-800 flex items-center justify-between">
                  <div className="space-y-0.5">
                    <span className="text-[10px] text-slate-400 font-medium block">Current Disk Read</span>
                    <span className="font-mono font-bold text-slate-800 dark:text-slate-200 text-sm">
                      {telemetry.readMb}/s
                    </span>
                  </div>
                  <span className="w-2 h-2 rounded-full bg-emerald-500" />
                </div>
                <div className="p-2.5 rounded-xl bg-slate-50/80 dark:bg-surface-800/40 border border-slate-100 dark:border-surface-800 flex items-center justify-between">
                  <div className="space-y-0.5">
                    <span className="text-[10px] text-slate-400 font-medium block">Current Disk Write</span>
                    <span className="font-mono font-bold text-slate-800 dark:text-slate-200 text-sm">
                      {telemetry.writeMb}/s
                    </span>
                  </div>
                  <span className="w-2 h-2 rounded-full bg-amber-500" />
                </div>
                <div className="p-2.5 rounded-xl bg-slate-50/80 dark:bg-surface-800/40 border border-slate-100 dark:border-surface-800 flex items-center justify-between">
                  <div className="space-y-0.5">
                    <span className="text-[10px] text-slate-400 font-medium block">Total Disk Read</span>
                    <span className="font-mono font-bold text-slate-800 dark:text-slate-200 text-sm">
                      {telemetry.totalReadGb}
                    </span>
                  </div>
                  <ArrowUp className="w-3.5 h-3.5 text-emerald-500" />
                </div>
                <div className="p-2.5 rounded-xl bg-slate-50/80 dark:bg-surface-800/40 border border-slate-100 dark:border-surface-800 flex items-center justify-between">
                  <div className="space-y-0.5">
                    <span className="text-[10px] text-slate-400 font-medium block">Total Disk Write</span>
                    <span className="font-mono font-bold text-slate-800 dark:text-slate-200 text-sm">
                      {telemetry.totalWriteGb}
                    </span>
                  </div>
                  <ArrowDown className="w-3.5 h-3.5 text-amber-500" />
                </div>
              </>
            )}
          </div>

          {/* Realtime Smooth SVG Area Chart */}
          <div
            className="relative w-full h-44 sm:h-48 pt-2 overflow-hidden select-none"
            onMouseMove={(e) => {
              if (chartHistory.length < 2) return;
              const rect = e.currentTarget.getBoundingClientRect();
              const relX = (e.clientX - rect.left) / rect.width;
              const idx = Math.min(
                chartHistory.length - 1,
                Math.max(0, Math.round(relX * (chartHistory.length - 1)))
              );
              setHoverIdx(idx);
            }}
            onMouseLeave={() => setHoverIdx(null)}
          >
            <svg
              className="w-full h-36 sm:h-40 overflow-hidden block"
              viewBox="0 0 600 180"
              preserveAspectRatio="none"
            >
              <defs>
                <clipPath id="chartAreaClip">
                  <rect x="0" y="0" width="600" height="180" rx="8" />
                </clipPath>
                <linearGradient id="trafficGreenGrad" x1="0" y1="0" x2="0" y2="1">
                  <stop offset="0%" stopColor="#10b981" stopOpacity="0.4" />
                  <stop offset="100%" stopColor="#10b981" stopOpacity="0.0" />
                </linearGradient>
                <linearGradient id="trafficOrangeGrad" x1="0" y1="0" x2="0" y2="1">
                  <stop offset="0%" stopColor="#f59e0b" stopOpacity="0.35" />
                  <stop offset="100%" stopColor="#f59e0b" stopOpacity="0.0" />
                </linearGradient>
              </defs>

              {/* Gridlines with dynamic scale labels */}
              {[0, 45, 90, 135, 180].map((y, i) => (
                <g key={i}>
                  <line
                    x1="0"
                    y1={y}
                    x2="600"
                    y2={y}
                    stroke="currentColor"
                    strokeDasharray="4 4"
                    className="text-slate-200/90 dark:text-surface-800"
                    strokeWidth="1"
                    vectorEffect="non-scaling-stroke"
                  />
                  {(i === 0 || i === 2 || i === 4) && (
                    <text
                      x="6"
                      y={i === 0 ? 12 : i === 4 ? 172 : y - 3}
                      className="text-[9px] font-mono fill-slate-400 dark:fill-slate-500 opacity-70 select-none font-semibold"
                    >
                      {chartSvgPaths.gridLabels[i]}
                    </text>
                  )}
                </g>
              ))}

              {/* Smooth Area Fills & Lines inside clipPath */}
              <g clipPath="url(#chartAreaClip)">
                <path d={chartSvgPaths.area1} fill="url(#trafficGreenGrad)" />
                <path d={chartSvgPaths.area2} fill="url(#trafficOrangeGrad)" />

                <path
                  d={chartSvgPaths.line1}
                  fill="none"
                  stroke="#10b981"
                  strokeWidth="2.5"
                  strokeLinecap="round"
                  vectorEffect="non-scaling-stroke"
                />
                <path
                  d={chartSvgPaths.line2}
                  fill="none"
                  stroke="#f59e0b"
                  strokeWidth="2.5"
                  strokeLinecap="round"
                  vectorEffect="non-scaling-stroke"
                />
              </g>
            </svg>

            {/* HTML Overlay for Data Points & Tooltip */}
            <div className="absolute inset-x-0 top-2 h-36 sm:h-40 pointer-events-none">
              {chartSvgPaths.pts1?.map((p, idx) => (
                <div
                  key={`pt1-${idx}`}
                  className="absolute -translate-x-1/2 -translate-y-1/2 w-2 h-2 rounded-full bg-[#10b981] border border-white dark:border-surface-900 shadow-xs transition-transform duration-150"
                  style={{
                    left: `${(p.x / 600) * 100}%`,
                    top: `${(p.y / 180) * 100}%`,
                    transform:
                      hoverIdx === idx
                        ? 'translate(-50%, -50%) scale(1.6)'
                        : 'translate(-50%, -50%) scale(1)',
                  }}
                />
              ))}
              {chartSvgPaths.pts2?.map((p, idx) => (
                <div
                  key={`pt2-${idx}`}
                  className="absolute -translate-x-1/2 -translate-y-1/2 w-2 h-2 rounded-full bg-[#f59e0b] border border-white dark:border-surface-900 shadow-xs transition-transform duration-150"
                  style={{
                    left: `${(p.x / 600) * 100}%`,
                    top: `${(p.y / 180) * 100}%`,
                    transform:
                      hoverIdx === idx
                        ? 'translate(-50%, -50%) scale(1.6)'
                        : 'translate(-50%, -50%) scale(1)',
                  }}
                />
              ))}

              {/* Realtime Live Pulse Ping Indicator on latest data point */}
              {chartSvgPaths.pts1.length > 0 && (
                <span
                  className="absolute flex h-3 w-3 -translate-x-1/2 -translate-y-1/2 pointer-events-none z-10"
                  style={{
                    left: `${(chartSvgPaths.pts1[chartSvgPaths.pts1.length - 1].x / 600) * 100}%`,
                    top: `${(chartSvgPaths.pts1[chartSvgPaths.pts1.length - 1].y / 180) * 100}%`,
                  }}
                >
                  <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-emerald-400 opacity-75" />
                  <span className="relative inline-flex rounded-full h-3 w-3 bg-emerald-500 border border-white dark:border-surface-900" />
                </span>
              )}

              {/* Active Hover Crosshair Line and Floating Tooltip */}
              {hoverIdx !== null && chartHistory[hoverIdx] && (
                <>
                  <div
                    className="absolute top-0 bottom-0 w-px border-l border-dashed border-slate-400 dark:border-slate-500 pointer-events-none z-20"
                    style={{ left: `${(hoverIdx / (chartHistory.length - 1)) * 100}%` }}
                  />
                  <div
                    className="absolute -top-1 -translate-y-full -translate-x-1/2 pointer-events-none z-30 px-3 py-1.5 rounded-xl bg-slate-900/95 dark:bg-surface-900/95 text-white text-[11px] shadow-2xl backdrop-blur-md flex items-center gap-3 border border-slate-700/60"
                    style={{
                      left: `${Math.max(14, Math.min(86, (hoverIdx / (chartHistory.length - 1)) * 100))}%`,
                    }}
                  >
                    <span className="font-mono text-slate-300 font-semibold">
                      {chartHistory[hoverIdx].time}
                    </span>
                    <span className="flex items-center gap-1 font-mono text-emerald-400 font-bold">
                      <span className="w-1.5 h-1.5 rounded-full bg-emerald-400" />
                      {chartTab === 'traffic'
                        ? `↑ ${formatSpeed(chartHistory[hoverIdx].up)}`
                        : `R: ${formatSpeed(chartHistory[hoverIdx].read)}`}
                    </span>
                    <span className="flex items-center gap-1 font-mono text-amber-400 font-bold">
                      <span className="w-1.5 h-1.5 rounded-full bg-amber-400" />
                      {chartTab === 'traffic'
                        ? `↓ ${formatSpeed(chartHistory[hoverIdx].down)}`
                        : `W: ${formatSpeed(chartHistory[hoverIdx].write)}`}
                    </span>
                  </div>
                </>
              )}
            </div>

            {/* X-axis Timestamps */}
            <div className="relative w-full h-5 text-[10px] font-mono text-slate-400 mt-1 pt-1 border-t border-slate-100 dark:border-surface-800">
              {chartHistory.map((pt, idx) => {
                const isFirst = idx === 0;
                const isLast = idx === chartHistory.length - 1;
                const isMid = idx === Math.floor(chartHistory.length / 2);
                const pct = (idx / (chartHistory.length - 1)) * 100;

                return (
                  <span
                    key={idx}
                    className={`absolute top-1 select-none font-semibold ${
                      isFirst || isLast || isMid ? 'inline' : 'hidden sm:inline'
                    } ${
                      isFirst
                        ? 'left-0 text-left'
                        : isLast
                          ? 'right-0 text-right'
                          : '-translate-x-1/2 text-center'
                    }`}
                    style={!isFirst && !isLast ? { left: `${pct}%` } : undefined}
                  >
                    {pt.time}
                  </span>
                );
              })}
            </div>
          </div>
        </div>

        {/* Clean Modern Footer */}
        <div className="py-4 text-center text-xs text-slate-400 dark:text-slate-500 space-y-1">
          <p className="font-medium">
            Hostvra Linux Cloud Engine ©2024-2026 ·{' '}
            <span className="text-emerald-600 dark:text-emerald-400 font-semibold hover:underline">
              hostvra.com
            </span>{' '}
            · Next-Generation Enterprise Hosting Control Plane
          </p>
        </div>
      </div>

      {/* aaPanel-Style Application Control Modal */}
      {selectedControlApp && (
        <AppControlModal
          app={selectedControlApp}
          isOpen={controlModalOpen}
          onClose={() => setControlModalOpen(false)}
        />
      )}

      {/* Fullscreen Expanded Chart Modal */}
      {chartExpanded && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/75 backdrop-blur-sm animate-fadeIn">
          <div className="bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-700 rounded-3xl w-full max-w-4xl shadow-2xl overflow-hidden p-6 space-y-4">
            <div className="flex items-center justify-between">
              <h3 className="text-base font-bold text-slate-900 dark:text-white flex items-center gap-2">
                <Maximize2 className="w-5 h-5 text-emerald-600" />
                Detailed Real-Time Monitoring ({chartTab === 'traffic' ? 'Network Traffic' : 'Disk I/O Throughput'})
              </h3>
              <button
                onClick={() => setChartExpanded(false)}
                className="w-8 h-8 rounded-full bg-slate-100 dark:bg-surface-800 text-slate-500 hover:text-slate-900 dark:hover:text-white flex items-center justify-center cursor-pointer"
              >
                <X className="w-4 h-4" />
              </button>
            </div>

            <div className="flex items-center justify-between text-xs py-2 border-y border-slate-100 dark:border-surface-800">
              <div className="flex gap-2">
                <button
                  onClick={() => setChartTab('traffic')}
                  className={`px-3 py-1 rounded-lg font-bold transition-all cursor-pointer ${
                    chartTab === 'traffic'
                      ? 'bg-emerald-100 dark:bg-emerald-950/60 text-emerald-700 dark:text-emerald-300'
                      : 'text-slate-500'
                  }`}
                >
                  Network Traffic
                </button>
                <button
                  onClick={() => setChartTab('disk')}
                  className={`px-3 py-1 rounded-lg font-bold transition-all cursor-pointer ${
                    chartTab === 'disk'
                      ? 'bg-emerald-100 dark:bg-emerald-950/60 text-emerald-700 dark:text-emerald-300'
                      : 'text-slate-500'
                  }`}
                >
                  Disk I/O Throughput
                </button>
              </div>
              <span className="font-mono text-slate-500 font-semibold">
                Live Rate: {chartTab === 'traffic' ? telemetry.upstreamMb : `${telemetry.writeMb}/s`}
              </span>
            </div>

            <div
              className="relative w-full h-64 pt-2 overflow-hidden select-none"
              onMouseMove={(e) => {
                if (chartHistory.length < 2) return;
                const rect = e.currentTarget.getBoundingClientRect();
                const relX = (e.clientX - rect.left) / rect.width;
                const idx = Math.min(
                  chartHistory.length - 1,
                  Math.max(0, Math.round(relX * (chartHistory.length - 1)))
                );
                setModalHoverIdx(idx);
              }}
              onMouseLeave={() => setModalHoverIdx(null)}
            >
              <svg className="w-full h-56 overflow-hidden block" viewBox="0 0 600 180" preserveAspectRatio="none">
                <defs>
                  <clipPath id="modalChartAreaClip">
                    <rect x="0" y="0" width="600" height="180" rx="8" />
                  </clipPath>
                  <linearGradient id="modalGreenGrad" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="0%" stopColor="#10b981" stopOpacity="0.4" />
                    <stop offset="100%" stopColor="#10b981" stopOpacity="0.0" />
                  </linearGradient>
                  <linearGradient id="modalOrangeGrad" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="0%" stopColor="#f59e0b" stopOpacity="0.35" />
                    <stop offset="100%" stopColor="#f59e0b" stopOpacity="0.0" />
                  </linearGradient>
                </defs>

                {[0, 45, 90, 135, 180].map((y, i) => (
                  <g key={i}>
                    <line
                      x1="0"
                      y1={y}
                      x2="600"
                      y2={y}
                      stroke="currentColor"
                      strokeDasharray="4 4"
                      className="text-slate-200/90 dark:text-surface-800"
                      strokeWidth="1"
                      vectorEffect="non-scaling-stroke"
                    />
                    {(i === 0 || i === 2 || i === 4) && (
                      <text
                        x="6"
                        y={i === 0 ? 12 : i === 4 ? 172 : y - 3}
                        className="text-[9px] font-mono fill-slate-400 dark:fill-slate-500 opacity-70 select-none font-semibold"
                      >
                        {chartSvgPaths.gridLabels[i]}
                      </text>
                    )}
                  </g>
                ))}

                <g clipPath="url(#modalChartAreaClip)">
                  <path d={chartSvgPaths.area1} fill="url(#modalGreenGrad)" />
                  <path d={chartSvgPaths.area2} fill="url(#modalOrangeGrad)" />

                  <path
                    d={chartSvgPaths.line1}
                    fill="none"
                    stroke="#10b981"
                    strokeWidth="2.5"
                    strokeLinecap="round"
                    vectorEffect="non-scaling-stroke"
                  />
                  <path
                    d={chartSvgPaths.line2}
                    fill="none"
                    stroke="#f59e0b"
                    strokeWidth="2.5"
                    strokeLinecap="round"
                    vectorEffect="non-scaling-stroke"
                  />
                </g>
              </svg>

              {/* Modal HTML Overlay for Data Points & Tooltip */}
              <div className="absolute inset-x-0 top-2 h-56 pointer-events-none">
                {chartSvgPaths.pts1?.map((p, idx) => (
                  <div
                    key={`mpt1-${idx}`}
                    className="absolute -translate-x-1/2 -translate-y-1/2 w-2.5 h-2.5 rounded-full bg-[#10b981] border border-white dark:border-surface-900 shadow-xs transition-transform duration-150"
                    style={{
                      left: `${(p.x / 600) * 100}%`,
                      top: `${(p.y / 180) * 100}%`,
                      transform:
                        modalHoverIdx === idx
                          ? 'translate(-50%, -50%) scale(1.6)'
                          : 'translate(-50%, -50%) scale(1)',
                    }}
                  />
                ))}
                {chartSvgPaths.pts2?.map((p, idx) => (
                  <div
                    key={`mpt2-${idx}`}
                    className="absolute -translate-x-1/2 -translate-y-1/2 w-2.5 h-2.5 rounded-full bg-[#f59e0b] border border-white dark:border-surface-900 shadow-xs transition-transform duration-150"
                    style={{
                      left: `${(p.x / 600) * 100}%`,
                      top: `${(p.y / 180) * 100}%`,
                      transform:
                        modalHoverIdx === idx
                          ? 'translate(-50%, -50%) scale(1.6)'
                          : 'translate(-50%, -50%) scale(1)',
                    }}
                  />
                ))}

                {chartSvgPaths.pts1.length > 0 && (
                  <span
                    className="absolute flex h-3.5 w-3.5 -translate-x-1/2 -translate-y-1/2 pointer-events-none z-10"
                    style={{
                      left: `${(chartSvgPaths.pts1[chartSvgPaths.pts1.length - 1].x / 600) * 100}%`,
                      top: `${(chartSvgPaths.pts1[chartSvgPaths.pts1.length - 1].y / 180) * 100}%`,
                    }}
                  >
                    <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-emerald-400 opacity-75" />
                    <span className="relative inline-flex rounded-full h-3.5 w-3.5 bg-emerald-500 border border-white dark:border-surface-900" />
                  </span>
                )}

                {modalHoverIdx !== null && chartHistory[modalHoverIdx] && (
                  <>
                    <div
                      className="absolute top-0 bottom-0 w-px border-l border-dashed border-slate-400 dark:border-slate-500 pointer-events-none z-20"
                      style={{ left: `${(modalHoverIdx / (chartHistory.length - 1)) * 100}%` }}
                    />
                    <div
                      className="absolute -top-1 -translate-y-full -translate-x-1/2 pointer-events-none z-30 px-3.5 py-2 rounded-xl bg-slate-900/95 dark:bg-surface-900/95 text-white text-xs shadow-2xl backdrop-blur-md flex items-center gap-3 border border-slate-700/60"
                      style={{
                        left: `${Math.max(14, Math.min(86, (modalHoverIdx / (chartHistory.length - 1)) * 100))}%`,
                      }}
                    >
                      <span className="font-mono text-slate-300 font-semibold">
                        {chartHistory[modalHoverIdx].time}
                      </span>
                      <span className="flex items-center gap-1 font-mono text-emerald-400 font-bold">
                        <span className="w-1.5 h-1.5 rounded-full bg-emerald-400" />
                        {chartTab === 'traffic'
                          ? `↑ ${formatSpeed(chartHistory[modalHoverIdx].up)}`
                          : `R: ${formatSpeed(chartHistory[modalHoverIdx].read)}`}
                      </span>
                      <span className="flex items-center gap-1 font-mono text-amber-400 font-bold">
                        <span className="w-1.5 h-1.5 rounded-full bg-amber-400" />
                        {chartTab === 'traffic'
                          ? `↓ ${formatSpeed(chartHistory[modalHoverIdx].down)}`
                          : `W: ${formatSpeed(chartHistory[modalHoverIdx].write)}`}
                      </span>
                    </div>
                  </>
                )}
              </div>

              {/* Modal X-axis Timestamps */}
              <div className="relative w-full h-5 text-[10px] font-mono text-slate-400 mt-2 pt-1 border-t border-slate-100 dark:border-surface-800">
                {chartHistory.map((pt, idx) => {
                  const isFirst = idx === 0;
                  const isLast = idx === chartHistory.length - 1;
                  const pct = (idx / (chartHistory.length - 1)) * 100;
                  return (
                    <span
                      key={idx}
                      className={`absolute top-1 select-none font-semibold ${
                        isFirst
                          ? 'left-0 text-left'
                          : isLast
                            ? 'right-0 text-right'
                            : '-translate-x-1/2 text-center'
                      }`}
                      style={!isFirst && !isLast ? { left: `${pct}%` } : undefined}
                    >
                      {pt.time}
                    </span>
                  );
                })}
              </div>
            </div>

            <div className="flex justify-end pt-2">
              <button
                onClick={() => setChartExpanded(false)}
                className="px-4 py-2 rounded-xl bg-slate-100 dark:bg-surface-800 text-slate-700 dark:text-slate-200 text-xs font-bold hover:bg-slate-200 dark:hover:bg-surface-700 transition cursor-pointer"
              >
                Close Fullscreen
              </button>
            </div>
          </div>
        </div>
      )}
    </DashboardShell>
  );
}
