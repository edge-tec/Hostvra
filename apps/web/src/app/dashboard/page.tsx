'use client';

import React, { useState, useEffect, useMemo } from 'react';
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

export default function DashboardPage() {
  const router = useRouter();
  const [userRole, setUserRole] = useState<'customer' | 'admin'>('customer');
  const [isSuperAdmin, setIsSuperAdmin] = useState(false);
  const [viewMode, setViewMode] = useState<'customer' | 'admin'>('customer');
  const [roleLoaded, setRoleLoaded] = useState(false);
  const [servers, setServers] = useState<ServerModel[]>([]);
  const [apps, setApps] = useState<AppPackage[]>([]);
  const [loading, setLoading] = useState(true);
  const [selectedControlApp, setSelectedControlApp] = useState<AppPackage | null>(null);
  const [controlModalOpen, setControlModalOpen] = useState(false);
  const [pinnedAppIds, setPinnedAppIds] = useState<string[]>([]);
  const [chartExpanded, setChartExpanded] = useState(false);

  // Memo Note State
  const [memoText, setMemoText] = useState('');
  const [isEditingMemo, setIsEditingMemo] = useState(false);

  // Monitor Chart State (Traffic vs Disk IO)
  const [chartTab, setChartTab] = useState<'traffic' | 'disk'>('traffic');
  const [networkInterface, setNetworkInterface] = useState('All');
  const [hoverIdx, setHoverIdx] = useState<number | null>(null);
  const [modalHoverIdx, setModalHoverIdx] = useState<number | null>(null);
  const [chartHistory, setChartHistory] = useState<
    Array<{ time: string; up: number; down: number; read: number; write: number }>
  >([]);

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
    load: { text: 'Sampling...', avg: '0.00 / 0.00 / 0.00', percent: 0 },
    cpu: { cores: 1, percent: 0 },
    ram: { used: '0 GB', total: '0 GB', percent: 0 },
    disk: { used: '0 GB', total: '0 GB', percent: 0 },
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

    // Also check server auth session to guarantee accuracy
    apiFetch<{ role?: string; is_superadmin?: boolean }>('/api/v1/auth/me').then((res) => {
      if (res.success && res.data) {
        const liveAdmin = res.data.role === 'admin' || res.data.role === 'superadmin' || Boolean(res.data.is_superadmin);
        setUserRole(liveAdmin ? 'admin' : 'customer');
        setIsSuperAdmin(Boolean(res.data.is_superadmin));
        setViewMode(liveAdmin ? 'admin' : 'customer');
      }
    }).catch(() => {});
  }, []);

  const fetchData = async () => {
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
            load: { text: t.load.text, avg: t.load.avg, percent: Math.round(t.load.percent) },
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
    }
  };

  useEffect(() => {
    fetchData();
    setPinnedAppIds(getPinnedAppIds());

    // Load saved memo
    const savedMemo = localStorage.getItem('hostvra_dashboard_memo');
    if (savedMemo) {
      setMemoText(savedMemo);
    }

    // Initialize rolling chart history buffer (last 8 points)
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

    // Live Chart & Telemetry ticker every 2.5 seconds directly from server Linux metrics
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
              load: { text: t.load.text, avg: t.load.avg, percent: Math.round(t.load.percent) },
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

  const handleOpenApp = (app: AppPackage) => {
    setSelectedControlApp(app);
    setControlModalOpen(true);
  };

  const installedApps = useMemo(() => apps.filter((a) => a.is_installed), [apps]);

  const getGaugeColor = (pct: number) => {
    if (pct >= 85) return '#ef4444';
    if (pct >= 70) return '#f59e0b';
    return '#20a53a';
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

    // Series mapping:
    // Traffic: line1 = Upstream (green), line2 = Downstream (amber)
    // Disk:    line1 = Read (green),     line2 = Write (amber)
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
        // Strictly clamp ratio between 0 and 1 so points never escape chart bounds
        const ratio = Math.max(0, Math.min(1, val / maxY));
        const y = baselineY - ratio * chartHeight;
        return { x, y, val };
      });
    };

    const pts1 = getPoints(chartTab === 'traffic' ? 'up' : 'read');
    const pts2 = getPoints(chartTab === 'traffic' ? 'down' : 'write');

    // Smooth cubic bezier curve with monotonic slope clamping to prevent overshoot/undershoot
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

        // If segment is flat, slopes at ends must be zero (prevents dipping below baseline)
        if (Math.abs(s) < 0.0001) {
          m1 = 0;
          m2 = 0;
        } else {
          // If tangents have opposite sign from segment slope, zero them to avoid loops
          if (m1 * s < 0) m1 = 0;
          if (m2 * s < 0) m2 = 0;
          // Clamp slope to prevent extreme curvature
          if (Math.abs(m1) > 2.5 * Math.abs(s)) m1 = 2.5 * s;
          if (Math.abs(m2) > 2.5 * Math.abs(s)) m2 = 2.5 * s;
        }

        const cp1x = p1.x + dx / 3;
        let cp1y = p1.y + (m1 * dx) / 3;
        const cp2x = p2.x - dx / 3;
        let cp2y = p2.y - (m2 * dx) / 3;

        // Strict boundary clamping within segment range and SVG viewport
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

  if (viewMode === 'customer') {
    return (
      <DashboardShell>
        {userRole === 'admin' && (
          <div className="mb-4 flex items-center justify-between p-3 rounded-xl bg-emerald-50 dark:bg-emerald-950/40 border border-emerald-200 dark:border-emerald-800 text-xs">
            <span className="font-semibold text-emerald-900 dark:text-emerald-300">
              Administrator Preview: Viewing Customer Hosting Dashboard
            </span>
            <div className="flex items-center gap-2">
              <Link
                href="/admin/users"
                className="px-3 py-1 font-semibold rounded-lg bg-emerald-100 dark:bg-emerald-900/50 text-emerald-800 dark:text-emerald-200 hover:bg-emerald-200 transition-colors"
              >
                User Management
              </Link>
              <button
                onClick={() => setViewMode('admin')}
                className="px-3 py-1 font-bold rounded-lg bg-[#16A34A] hover:bg-[#15803D] text-white transition-colors cursor-pointer"
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
      <div className="space-y-4">
        {userRole === 'admin' && (
          <div className="flex items-center justify-between p-3 rounded-xl bg-slate-100 dark:bg-surface-800/80 border border-slate-200 dark:border-surface-700 text-xs">
            <div className="flex items-center gap-2 text-slate-700 dark:text-slate-300">
              <span className="w-2 h-2 rounded-full bg-emerald-500" />
              <span className="font-bold">Root Server Infrastructure Node (Admin Control Plane)</span>
            </div>
            <div className="flex items-center gap-2">
              <Link
                href="/admin/users"
                className="px-3 py-1 font-semibold rounded-lg bg-[#16A34A] hover:bg-[#15803D] text-white transition-colors"
              >
                Manage Tenant Accounts
              </Link>
              <button
                onClick={() => setViewMode('customer')}
                className="px-3 py-1 font-semibold rounded-lg bg-slate-200 dark:bg-surface-700 text-slate-800 dark:text-slate-200 hover:bg-slate-300 dark:hover:bg-surface-600 transition-colors cursor-pointer"
              >
                Customer Portal View
              </button>
            </div>
          </div>
        )}

        {/* Realtime API Disconnected Alert */}
        {apiError && (
          <div className="p-3.5 rounded-xl bg-amber-500/10 border border-amber-500/30 text-amber-300 text-xs flex items-center justify-between shadow-sm">
            <div className="flex items-center gap-2">
              <ShieldAlert className="w-4 h-4 text-amber-400 shrink-0" />
              <span><strong>Live Telemetry Disconnected:</strong> {apiError}</span>
            </div>
            <button
              onClick={fetchData}
              className="px-2.5 py-1 rounded bg-amber-500/20 hover:bg-amber-500/30 text-[11px] font-semibold text-amber-200 transition-colors"
            >
              Retry
            </button>
          </div>
        )}

        {/* ROW 1: 4 Circular System Gauge Cards (Load, CPU, RAM, Disk) */}
        <div className="grid grid-cols-1 sm:grid-cols-2 xl:grid-cols-4 gap-3 sm:gap-4">
          {/* Card 1: Load Status */}
          <div className="p-3.5 sm:p-4 rounded-xl bg-white dark:bg-[#121824] border border-slate-200 dark:border-surface-800 shadow-sm flex items-center justify-between">
            <div className="space-y-1 min-w-0">
              <span className="text-xs font-semibold text-slate-500 dark:text-slate-400 block">
                {telemetry.load.text}
              </span>
              <div className="flex items-center gap-1">
                <span className="text-xs sm:text-[13px] font-mono font-bold text-slate-800 dark:text-slate-200">
                  {telemetry.load.avg}
                </span>
                <span title="1, 5, and 15-minute load averages" className="flex items-center">
                  <HelpCircle className="w-3.5 h-3.5 text-slate-400 flex-shrink-0" />
                </span>
              </div>
            </div>

            {/* Circular Gauge */}
            <div className="relative w-14 h-14 sm:w-16 sm:h-16 flex items-center justify-center flex-shrink-0">
              <svg className="w-14 h-14 sm:w-16 sm:h-16 transform -rotate-90" viewBox="0 0 80 80">
                <circle
                  cx="40"
                  cy="40"
                  r="34"
                  strokeWidth="6"
                  className="gauge-track"
                  fill="transparent"
                />
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
                  className="transition-all duration-500"
                />
              </svg>
              <span className="absolute text-xs sm:text-sm font-extrabold text-slate-900 dark:text-white">
                {telemetry.load.percent}%
              </span>
            </div>
          </div>

          {/* Card 2: CPU */}
          <div className="p-3.5 sm:p-4 rounded-xl bg-white dark:bg-[#121824] border border-slate-200 dark:border-surface-800 shadow-sm flex items-center justify-between">
            <div className="space-y-1 min-w-0">
              <span className="text-xs font-semibold text-slate-500 dark:text-slate-400 block">CPU</span>
              <div className="text-xs sm:text-[13px] font-mono font-bold text-slate-800 dark:text-slate-200">
                {telemetry.cpu.cores} Cores
              </div>
            </div>

            <div className="relative w-14 h-14 sm:w-16 sm:h-16 flex items-center justify-center flex-shrink-0">
              <svg className="w-14 h-14 sm:w-16 sm:h-16 transform -rotate-90" viewBox="0 0 80 80">
                <circle
                  cx="40"
                  cy="40"
                  r="34"
                  strokeWidth="6"
                  className="gauge-track"
                  fill="transparent"
                />
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
                  className="transition-all duration-500"
                />
              </svg>
              <span className="absolute text-xs sm:text-sm font-extrabold text-slate-900 dark:text-white">
                {telemetry.cpu.percent}%
              </span>
            </div>
          </div>

          {/* Card 3: RAM */}
          <div className="p-3.5 sm:p-4 rounded-xl bg-white dark:bg-[#121824] border border-slate-200 dark:border-surface-800 shadow-sm flex items-center justify-between">
            <div className="space-y-1 min-w-0">
              <span className="text-xs font-semibold text-slate-500 dark:text-slate-400 block">RAM</span>
              <div className="text-xs sm:text-[13px] font-mono font-bold text-slate-800 dark:text-slate-200 truncate">
                <span className="text-emerald-600 dark:text-emerald-400 font-bold">{telemetry.ram.used}</span> /{' '}
                {telemetry.ram.total}
              </div>
            </div>

            <div className="relative w-14 h-14 sm:w-16 sm:h-16 flex items-center justify-center flex-shrink-0">
              <svg className="w-14 h-14 sm:w-16 sm:h-16 transform -rotate-90" viewBox="0 0 80 80">
                <circle
                  cx="40"
                  cy="40"
                  r="34"
                  strokeWidth="6"
                  className="gauge-track"
                  fill="transparent"
                />
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
                  className="transition-all duration-500"
                />
              </svg>
              <span className="absolute text-xs sm:text-sm font-extrabold text-slate-900 dark:text-white">
                {telemetry.ram.percent}%
              </span>
            </div>
          </div>

          {/* Card 4: Disk */}
          <div className="p-3.5 sm:p-4 rounded-xl bg-white dark:bg-[#121824] border border-slate-200 dark:border-surface-800 shadow-sm flex items-center justify-between">
            <div className="space-y-1 min-w-0">
              <span className="text-xs font-semibold text-slate-500 dark:text-slate-400 block">/</span>
              <div className="text-xs sm:text-[13px] font-mono font-bold text-slate-800 dark:text-slate-200 truncate">
                <span className="text-emerald-600 dark:text-emerald-400 font-bold">{telemetry.disk.used}</span> /{' '}
                {telemetry.disk.total}
              </div>
            </div>

            <div className="relative w-14 h-14 sm:w-16 sm:h-16 flex items-center justify-center flex-shrink-0">
              <svg className="w-14 h-14 sm:w-16 sm:h-16 transform -rotate-90" viewBox="0 0 80 80">
                <circle
                  cx="40"
                  cy="40"
                  r="34"
                  strokeWidth="6"
                  className="gauge-track"
                  fill="transparent"
                />
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
                  className="transition-all duration-500"
                />
              </svg>
              <span className="absolute text-xs sm:text-sm font-extrabold text-slate-900 dark:text-white">
                {telemetry.disk.percent}%
              </span>
            </div>
          </div>
        </div>

        {/* ROW 2: Overview Card */}
        <div className="p-4 rounded-xl bg-white dark:bg-[#121824] border border-slate-200 dark:border-surface-800 shadow-sm space-y-3">
          <div className="flex items-center justify-between border-b border-slate-100 dark:border-surface-800 pb-2.5">
            <h2 className="text-xs font-bold text-slate-800 dark:text-slate-100">Overview</h2>
            <button className="text-slate-400 hover:text-slate-600 dark:hover:text-slate-200">
              <MoreVertical className="w-3.5 h-3.5" />
            </button>
          </div>

          <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-5 gap-3 text-xs">
            {/* Site - ALL */}
            <Link
              href="/websites"
              className="flex items-center justify-between p-2.5 rounded-lg hover:bg-slate-50 dark:hover:bg-surface-800/60 transition group border border-slate-100 dark:border-surface-800/40"
            >
              <div>
                <span className="font-semibold text-slate-700 dark:text-slate-200 group-hover:text-emerald-600 dark:group-hover:text-emerald-400">
                  Site - ALL
                </span>
                <div className="mt-1 flex flex-wrap items-center gap-2">
                  <span className="text-emerald-600 dark:text-emerald-400 font-bold whitespace-nowrap">
                    Running: {counts.websites_running}
                  </span>
                  <span className="text-slate-400 whitespace-nowrap">Stopped: {counts.websites_stopped}</span>
                </div>
                <span className="text-[11px] text-slate-400 block mt-0.5">ALL: {counts.websites_total}</span>
              </div>
              <Globe className="w-4 h-4 text-emerald-600 dark:text-emerald-400 flex-shrink-0" />
            </Link>

            {/* FTP */}
            <Link
              href="/servers"
              className="flex items-center justify-between p-2.5 rounded-lg hover:bg-slate-50 dark:hover:bg-surface-800/60 transition group border border-slate-100 dark:border-surface-800/40"
            >
              <div>
                <span className="font-semibold text-slate-700 dark:text-slate-200 group-hover:text-emerald-600 dark:group-hover:text-emerald-400">
                  FTP
                </span>
                <div className="mt-1 text-emerald-600 dark:text-emerald-400 font-bold">
                  Accounts: {counts.ftp_accounts_total}
                </div>
              </div>
              <ArrowLeftRight className="w-4 h-4 text-emerald-600 dark:text-emerald-400 flex-shrink-0" />
            </Link>

            {/* Database */}
            <Link
              href="/databases"
              className="flex items-center justify-between p-2.5 rounded-lg hover:bg-slate-50 dark:hover:bg-surface-800/60 transition group border border-slate-100 dark:border-surface-800/40"
            >
              <div>
                <span className="font-semibold text-slate-700 dark:text-slate-200 group-hover:text-emerald-600 dark:group-hover:text-emerald-400">
                  Database
                </span>
                <div className="mt-1 text-emerald-600 dark:text-emerald-400 font-bold">
                  ALL: {counts.databases_total}
                </div>
              </div>
              <Database className="w-4 h-4 text-emerald-600 dark:text-emerald-400 flex-shrink-0" />
            </Link>

            {/* Security */}
            <Link
              href="/firewall"
              className="flex items-center justify-between p-2.5 rounded-lg hover:bg-slate-50 dark:hover:bg-surface-800/60 transition group border border-slate-100 dark:border-surface-800/40"
            >
              <div className="min-w-0 pr-1">
                <span className="font-semibold text-slate-700 dark:text-slate-200 group-hover:text-emerald-600 dark:group-hover:text-emerald-400">
                  Security
                </span>
                <div className="mt-1 text-slate-600 dark:text-slate-300 font-medium">
                  Security Risk: {counts.security_risks}
                </div>
                <span className="text-[10px] text-slate-400 block mt-0.5 truncate">
                  Last Scan: {counts.last_security_scan}
                </span>
              </div>
              <Shield className="w-4 h-4 text-emerald-600 dark:text-emerald-400 flex-shrink-0" />
            </Link>

            {/* Memo (Editable) */}
            <div
              id="dashboard-memo-card"
              onClick={() => setIsEditingMemo(true)}
              className="flex items-start justify-between p-2.5 rounded-lg bg-slate-50 dark:bg-surface-800/40 border border-dashed border-slate-200 dark:border-surface-700 hover:border-emerald-500 cursor-pointer transition sm:col-span-2 lg:col-span-1"
            >
              <div className="flex-1 min-w-0 pr-2">
                <span className="font-semibold text-slate-700 dark:text-slate-200 block">Memo</span>
                {isEditingMemo ? (
                  <textarea
                    value={memoText}
                    onChange={(e) => saveMemo(e.target.value)}
                    onBlur={() => setIsEditingMemo(false)}
                    autoFocus
                    placeholder="Type server notes here..."
                    className="w-full mt-1 p-1 text-xs bg-white dark:bg-surface-900 border border-slate-300 dark:border-surface-700 rounded text-slate-800 dark:text-slate-200 focus:outline-none focus:ring-1 focus:ring-emerald-500 resize-none h-12"
                  />
                ) : (
                  <p className="mt-1 text-[11px] text-slate-400 line-clamp-2">
                    {memoText || 'Current content is empty, click to edit'}
                  </p>
                )}
              </div>
              <FileText className="w-4 h-4 text-emerald-600 dark:text-emerald-400 flex-shrink-0 mt-0.5" />
            </div>
          </div>
        </div>

        {/* ROW 3: Core Management Tools */}
        <div className="p-4 rounded-xl bg-white dark:bg-[#121824] border border-slate-200 dark:border-surface-800 shadow-sm space-y-3">
          <div className="flex items-center justify-between border-b border-slate-100 dark:border-surface-800 pb-2.5">
            <div className="flex items-center gap-2">
              <h2 className="text-xs font-bold text-slate-800 dark:text-slate-100">Core Management Tools</h2>
              <span className="text-[10px] px-1.5 py-0.5 rounded bg-emerald-100 dark:bg-emerald-950/50 text-emerald-700 dark:text-emerald-300 font-semibold">
                100% Functional
              </span>
            </div>
            <Link
              href="/app-store"
              className="text-emerald-600 dark:text-emerald-400 hover:underline text-xs font-semibold flex items-center gap-1"
            >
              + App Store
            </Link>
          </div>

          <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 xl:grid-cols-4 gap-3 pt-1">
            {/* Tool 1: Web Servers */}
            <div
              onClick={() => router.push('/webservers')}
              className="flex flex-col p-3 rounded-xl border border-slate-200 dark:border-surface-800 hover:border-emerald-500 hover:shadow-md cursor-pointer transition bg-slate-50/50 dark:bg-surface-800/30 group"
            >
              <div className="flex items-center justify-between mb-2">
                <div className="w-8 h-8 rounded-lg bg-emerald-100 dark:bg-emerald-950/50 flex items-center justify-center text-emerald-600 dark:text-emerald-400 group-hover:scale-105 transition">
                  <Layers className="w-4 h-4" />
                </div>
                <span className="text-[10px] font-semibold text-emerald-600 dark:text-emerald-400 bg-emerald-50 dark:bg-emerald-950/40 px-1.5 py-0.5 rounded border border-emerald-500/20">
                  Active
                </span>
              </div>
              <span className="text-xs font-bold text-slate-800 dark:text-slate-200 group-hover:text-emerald-600 dark:group-hover:text-emerald-400">
                Web Servers
              </span>
              <span className="text-[11px] text-slate-500 dark:text-slate-400 truncate mt-0.5">
                Nginx, Apache, OpenLiteSpeed
              </span>
            </div>

            {/* Tool 2: PHP Management */}
            <div
              onClick={() => router.push('/php')}
              className="flex flex-col p-3 rounded-xl border border-slate-200 dark:border-surface-800 hover:border-emerald-500 hover:shadow-md cursor-pointer transition bg-slate-50/50 dark:bg-surface-800/30 group"
            >
              <div className="flex items-center justify-between mb-2">
                <div className="w-8 h-8 rounded-lg bg-emerald-100 dark:bg-emerald-950/50 flex items-center justify-center text-emerald-600 dark:text-emerald-400 group-hover:scale-105 transition">
                  <Code2 className="w-4 h-4" />
                </div>
                <span className="text-[10px] font-semibold text-emerald-700 dark:text-emerald-300 bg-emerald-50 dark:bg-emerald-950/40 px-1.5 py-0.5 rounded border border-emerald-500/20">
                  Ready
                </span>
              </div>
              <span className="text-xs font-bold text-slate-800 dark:text-slate-200 group-hover:text-emerald-600 dark:group-hover:text-emerald-400">
                PHP Management
              </span>
              <span className="text-[11px] text-slate-500 dark:text-slate-400 truncate mt-0.5">
                PHP-FPM, Extensions, Pools
              </span>
            </div>

            {/* Tool 3: Databases */}
            <div
              onClick={() => router.push('/databases')}
              className="flex flex-col p-3 rounded-xl border border-slate-200 dark:border-surface-800 hover:border-emerald-500 hover:shadow-md cursor-pointer transition bg-slate-50/50 dark:bg-surface-800/30 group"
            >
              <div className="flex items-center justify-between mb-2">
                <div className="w-8 h-8 rounded-lg bg-amber-100 dark:bg-amber-950/50 flex items-center justify-center text-amber-600 dark:text-amber-400 group-hover:scale-105 transition">
                  <Database className="w-4 h-4" />
                </div>
                <span className="text-[10px] font-semibold text-amber-600 dark:text-amber-400 bg-amber-50 dark:bg-amber-950/40 px-1.5 py-0.5 rounded border border-amber-500/20">
                  Online
                </span>
              </div>
              <span className="text-xs font-bold text-slate-800 dark:text-slate-200 group-hover:text-emerald-600 dark:group-hover:text-emerald-400">
                Databases
              </span>
              <span className="text-[11px] text-slate-500 dark:text-slate-400 truncate mt-0.5">
                MySQL, MariaDB, phpMyAdmin
              </span>
            </div>

            {/* Tool 4: File Manager */}
            <div
              onClick={() => router.push('/files')}
              className="flex flex-col p-3 rounded-xl border border-slate-200 dark:border-surface-800 hover:border-emerald-500 hover:shadow-md cursor-pointer transition bg-slate-50/50 dark:bg-surface-800/30 group"
            >
              <div className="flex items-center justify-between mb-2">
                <div className="w-8 h-8 rounded-lg bg-sky-100 dark:bg-sky-950/50 flex items-center justify-center text-sky-600 dark:text-sky-400 group-hover:scale-105 transition">
                  <FolderOpen className="w-4 h-4" />
                </div>
                <span className="text-[10px] font-semibold text-sky-600 dark:text-sky-400 bg-sky-50 dark:bg-sky-950/40 px-1.5 py-0.5 rounded border border-sky-500/20">
                  Ready
                </span>
              </div>
              <span className="text-xs font-bold text-slate-800 dark:text-slate-200 group-hover:text-emerald-600 dark:group-hover:text-emerald-400">
                File Manager
              </span>
              <span className="text-[11px] text-slate-500 dark:text-slate-400 truncate mt-0.5">
                Directory Browser & Code Editor
              </span>
            </div>

            {/* Tool 5: Terminal */}
            <div
              onClick={() => router.push('/terminal')}
              className="flex flex-col p-3 rounded-xl border border-slate-200 dark:border-surface-800 hover:border-emerald-500 hover:shadow-md cursor-pointer transition bg-slate-50/50 dark:bg-surface-800/30 group"
            >
              <div className="flex items-center justify-between mb-2">
                <div className="w-8 h-8 rounded-lg bg-slate-200 dark:bg-surface-700 flex items-center justify-center text-slate-700 dark:text-slate-300 group-hover:scale-105 transition">
                  <Terminal className="w-4 h-4" />
                </div>
                <span className="text-[10px] font-semibold text-slate-600 dark:text-slate-300 bg-slate-100 dark:bg-surface-700/50 px-1.5 py-0.5 rounded border border-slate-300/30">
                  CLI Shell
                </span>
              </div>
              <span className="text-xs font-bold text-slate-800 dark:text-slate-200 group-hover:text-emerald-600 dark:group-hover:text-emerald-400">
                Terminal
              </span>
              <span className="text-[11px] text-slate-500 dark:text-slate-400 truncate mt-0.5">
                Root Web SSH Console
              </span>
            </div>

            {/* Tool 6: Firewall */}
            <div
              onClick={() => router.push('/firewall')}
              className="flex flex-col p-3 rounded-xl border border-slate-200 dark:border-surface-800 hover:border-emerald-500 hover:shadow-md cursor-pointer transition bg-slate-50/50 dark:bg-surface-800/30 group"
            >
              <div className="flex items-center justify-between mb-2">
                <div className="w-8 h-8 rounded-lg bg-emerald-100 dark:bg-emerald-950/50 flex items-center justify-center text-emerald-600 dark:text-emerald-400 group-hover:scale-105 transition">
                  <Shield className="w-4 h-4" />
                </div>
                <span className="text-[10px] font-semibold text-emerald-600 dark:text-emerald-400 bg-emerald-50 dark:bg-emerald-950/40 px-1.5 py-0.5 rounded border border-emerald-500/20">
                  Protected
                </span>
              </div>
              <span className="text-xs font-bold text-slate-800 dark:text-slate-200 group-hover:text-emerald-600 dark:group-hover:text-emerald-400">
                Firewall
              </span>
              <span className="text-[11px] text-slate-500 dark:text-slate-400 truncate mt-0.5">
                UFW Ports, Rules & IP Defense
              </span>
            </div>

            {/* Tool 7: Cron Jobs */}
            <div
              onClick={() => router.push('/cron')}
              className="flex flex-col p-3 rounded-xl border border-slate-200 dark:border-surface-800 hover:border-emerald-500 hover:shadow-md cursor-pointer transition bg-slate-50/50 dark:bg-surface-800/30 group"
            >
              <div className="flex items-center justify-between mb-2">
                <div className="w-8 h-8 rounded-lg bg-purple-100 dark:bg-purple-950/50 flex items-center justify-center text-purple-600 dark:text-purple-400 group-hover:scale-105 transition">
                  <Clock className="w-4 h-4" />
                </div>
                <span className="text-[10px] font-semibold text-purple-600 dark:text-purple-400 bg-purple-50 dark:bg-purple-950/40 px-1.5 py-0.5 rounded border border-purple-500/20">
                  Schedule
                </span>
              </div>
              <span className="text-xs font-bold text-slate-800 dark:text-slate-200 group-hover:text-emerald-600 dark:group-hover:text-emerald-400">
                Cron Jobs
              </span>
              <span className="text-[11px] text-slate-500 dark:text-slate-400 truncate mt-0.5">
                Scheduled Tasks & Automation
              </span>
            </div>

            {/* Tool 8: SSL Certificates */}
            <div
              onClick={() => router.push('/ssl')}
              className="flex flex-col p-3 rounded-xl border border-slate-200 dark:border-surface-800 hover:border-emerald-500 hover:shadow-md cursor-pointer transition bg-slate-50/50 dark:bg-surface-800/30 group"
            >
              <div className="flex items-center justify-between mb-2">
                <div className="w-8 h-8 rounded-lg bg-rose-100 dark:bg-rose-950/50 flex items-center justify-center text-rose-600 dark:text-rose-400 group-hover:scale-105 transition">
                  <Lock className="w-4 h-4" />
                </div>
                <span className="text-[10px] font-semibold text-rose-600 dark:text-rose-400 bg-rose-50 dark:bg-rose-950/40 px-1.5 py-0.5 rounded border border-rose-500/20">
                  Secured
                </span>
              </div>
              <span className="text-xs font-bold text-slate-800 dark:text-slate-200 group-hover:text-emerald-600 dark:group-hover:text-emerald-400">
                SSL Certificates
              </span>
              <span className="text-[11px] text-slate-500 dark:text-slate-400 truncate mt-0.5">
                Let&apos;s Encrypt & Custom SSL
              </span>
            </div>
          </div>
        </div>

        {/* ROW 4: Installed Services */}
        <div className="space-y-2.5">
          <div className="flex items-center justify-between text-xs font-bold text-slate-800 dark:text-slate-200 px-0.5">
            <span>Installed Services ({installedApps.length})</span>
            <Link
              href="/app-store"
              className="text-emerald-600 dark:text-emerald-400 hover:underline font-semibold flex items-center gap-1"
            >
              + Add Software
            </Link>
          </div>

          {installedApps.length > 0 ? (
            <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 gap-2.5">
              {installedApps.map((app) => (
                <button
                  key={app.id}
                  onClick={() => handleOpenApp(app)}
                  className="p-3 rounded-xl border border-slate-200 dark:border-surface-800 hover:border-emerald-500 bg-white dark:bg-[#121824] flex items-center justify-between text-left transition shadow-xs group min-w-0"
                >
                  <div className="flex items-center gap-2.5 min-w-0">
                    <span
                      className={`w-2.5 h-2.5 rounded-full flex-shrink-0 ${
                        app.status === 'running' ? 'bg-emerald-500' : 'bg-amber-500'
                      }`}
                    />
                    <div className="min-w-0">
                      <span className="text-xs font-bold text-slate-800 dark:text-slate-200 truncate block group-hover:text-emerald-600 dark:group-hover:text-emerald-400">
                        {app.name}
                      </span>
                      <span className="text-[10px] text-slate-400 block truncate">
                        {app.version ? `v${app.version}` : 'Active'} · {app.category || 'Package'}
                      </span>
                    </div>
                  </div>
                  <span className="text-[10px] text-emerald-600 dark:text-emerald-400 font-medium px-2 py-0.5 rounded bg-emerald-50 dark:bg-emerald-950/40 border border-emerald-500/20 opacity-0 group-hover:opacity-100 transition">
                    Manage
                  </span>
                </button>
              ))}
            </div>
          ) : (
            <div className="p-6 rounded-xl border border-dashed border-slate-300 dark:border-surface-700 bg-white dark:bg-[#121824] text-center space-y-2">
              <Boxes className="w-8 h-8 text-slate-400 dark:text-slate-500 mx-auto" />
              <div className="space-y-0.5">
                <p className="text-xs font-bold text-slate-700 dark:text-slate-200">
                  No Software Installed
                </p>
                <p className="text-[11px] text-slate-500 dark:text-slate-400 max-w-sm mx-auto">
                  Install Nginx, Apache, MySQL, PHP, Redis, or Node.js with 1-click from the App Store.
                </p>
              </div>
              <Link
                href="/app-store"
                className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-emerald-600 hover:bg-emerald-500 text-white font-semibold text-xs transition mt-1 shadow-xs"
              >
                + Browse App Store
              </Link>
            </div>
          )}
        </div>

        {/* ROW 5: Traffic & Disk IO Realtime Monitor */}
        <div className="p-4 rounded-xl bg-white dark:bg-[#121824] border border-slate-200 dark:border-surface-800 shadow-sm space-y-3">
          {/* Header with Tabs, Fullscreen & Net Interface dropdown */}
          <div className="flex items-center justify-between border-b border-slate-100 dark:border-surface-800 pb-2.5">
            <div className="flex items-center gap-4">
              <button
                onClick={() => setChartTab('traffic')}
                className={`text-xs font-bold transition ${
                  chartTab === 'traffic'
                    ? 'text-emerald-600 dark:text-emerald-400 border-b-2 border-emerald-600 pb-1 -mb-3'
                    : 'text-slate-500 hover:text-slate-800 dark:hover:text-white'
                }`}
              >
                Traffic
              </button>
              <button
                onClick={() => setChartTab('disk')}
                className={`text-xs font-bold transition ${
                  chartTab === 'disk'
                    ? 'text-emerald-600 dark:text-emerald-400 border-b-2 border-emerald-600 pb-1 -mb-3'
                    : 'text-slate-500 hover:text-slate-800 dark:hover:text-white'
                }`}
              >
                Disk IO
              </button>
              <button
                onClick={() => setChartExpanded(true)}
                className="text-slate-400 hover:text-slate-600 dark:hover:text-slate-200"
                title="Expand fullscreen view"
              >
                <Maximize2 className="w-3.5 h-3.5" />
              </button>
            </div>

            {/* Net Interface Dropdown & Peak Indicator */}
            <div className="flex items-center gap-2">
              <span className="hidden sm:inline-block text-[11px] font-mono text-slate-400 dark:text-slate-500">
                Peak: {formatSpeed(chartSvgPaths.maxPeak)}
              </span>
              <div className="flex items-center gap-1 text-xs text-slate-600 dark:text-slate-300 border border-slate-200 dark:border-surface-700 px-2 py-0.5 rounded-md">
                <span>Net: {networkInterface}</span>
              </div>
            </div>
          </div>

          {/* Realtime Telemetry Stats Strip */}
          <div className="flex flex-wrap items-center justify-between gap-x-4 gap-y-2 text-xs py-1">
            {chartTab === 'traffic' ? (
              <>
                <div className="flex items-center gap-1.5 whitespace-nowrap">
                  <span className="w-2 h-2 rounded-full bg-emerald-500 flex-shrink-0" />
                  <span className="text-slate-500 dark:text-slate-400">Upstream:</span>
                  <span className="font-mono font-bold text-slate-800 dark:text-slate-200">
                    {telemetry.upstreamMb}
                  </span>
                </div>
                <div className="flex items-center gap-1.5 whitespace-nowrap">
                  <span className="w-2 h-2 rounded-full bg-amber-500 flex-shrink-0" />
                  <span className="text-slate-500 dark:text-slate-400">Downstream:</span>
                  <span className="font-mono font-bold text-slate-800 dark:text-slate-200">
                    {telemetry.downstreamMb}
                  </span>
                </div>
                <div className="whitespace-nowrap">
                  <span className="text-slate-500 dark:text-slate-400">Total sent: </span>
                  <span className="font-mono font-bold text-slate-800 dark:text-slate-200">
                    {telemetry.totalSentGb}
                  </span>
                </div>
                <div className="whitespace-nowrap">
                  <span className="text-slate-500 dark:text-slate-400">Total received: </span>
                  <span className="font-mono font-bold text-slate-800 dark:text-slate-200">
                    {telemetry.totalReceivedGb}
                  </span>
                </div>
              </>
            ) : (
              <>
                <div className="flex items-center gap-1.5 whitespace-nowrap">
                  <span className="w-2 h-2 rounded-full bg-emerald-500 flex-shrink-0" />
                  <span className="text-slate-500 dark:text-slate-400">Read:</span>
                  <span className="font-mono font-bold text-slate-800 dark:text-slate-200">
                    {telemetry.readMb}/s
                  </span>
                </div>
                <div className="flex items-center gap-1.5 whitespace-nowrap">
                  <span className="w-2 h-2 rounded-full bg-amber-500 flex-shrink-0" />
                  <span className="text-slate-500 dark:text-slate-400">Write:</span>
                  <span className="font-mono font-bold text-slate-800 dark:text-slate-200">
                    {telemetry.writeMb}/s
                  </span>
                </div>
                <div className="whitespace-nowrap">
                  <span className="text-slate-500 dark:text-slate-400">Total read: </span>
                  <span className="font-mono font-bold text-slate-800 dark:text-slate-200">
                    {telemetry.totalReadGb}
                  </span>
                </div>
                <div className="whitespace-nowrap">
                  <span className="text-slate-500 dark:text-slate-400">Total write: </span>
                  <span className="font-mono font-bold text-slate-800 dark:text-slate-200">
                    {telemetry.totalWriteGb}
                  </span>
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
                  <rect x="0" y="0" width="600" height="180" rx="4" />
                </clipPath>
                <linearGradient id="trafficGreenGrad" x1="0" y1="0" x2="0" y2="1">
                  <stop offset="0%" stopColor="#20a53a" stopOpacity="0.35" />
                  <stop offset="100%" stopColor="#20a53a" stopOpacity="0.02" />
                </linearGradient>
                <linearGradient id="trafficOrangeGrad" x1="0" y1="0" x2="0" y2="1">
                  <stop offset="0%" stopColor="#f59e0b" stopOpacity="0.35" />
                  <stop offset="100%" stopColor="#f59e0b" stopOpacity="0.02" />
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
                    className="text-slate-200 dark:text-surface-800"
                    strokeWidth="1"
                    vectorEffect="non-scaling-stroke"
                  />
                  {(i === 0 || i === 2 || i === 4) && (
                    <text
                      x="6"
                      y={i === 0 ? 12 : i === 4 ? 172 : y - 3}
                      className="text-[9px] font-mono fill-slate-400 dark:fill-slate-500 opacity-60 select-none"
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
                  stroke="#20a53a"
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

            {/* HTML Overlay for Data Points & Tooltip - perfectly circular across all aspect ratios */}
            <div className="absolute inset-x-0 top-2 h-36 sm:h-40 pointer-events-none">
              {chartSvgPaths.pts1?.map((p, idx) => (
                <div
                  key={`pt1-${idx}`}
                  className="absolute -translate-x-1/2 -translate-y-1/2 w-2 h-2 rounded-full bg-[#20a53a] border border-white dark:border-surface-900 shadow-xs transition-transform duration-150"
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
                  <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-emerald-400 opacity-75"></span>
                  <span className="relative inline-flex rounded-full h-3 w-3 bg-emerald-500 border border-white dark:border-surface-900"></span>
                </span>
              )}

              {/* Active Hover Crosshair Line and Tooltip */}
              {hoverIdx !== null && chartHistory[hoverIdx] && (
                <>
                  <div
                    className="absolute top-0 bottom-0 w-px border-l border-dashed border-slate-400 dark:border-slate-500 pointer-events-none z-20"
                    style={{ left: `${(hoverIdx / (chartHistory.length - 1)) * 100}%` }}
                  />
                  <div
                    className="absolute -top-1 -translate-y-full -translate-x-1/2 pointer-events-none z-30 px-2.5 py-1 rounded-lg bg-slate-900/95 dark:bg-surface-900/95 text-white text-[10px] shadow-xl backdrop-blur-md flex items-center gap-2.5 border border-slate-700/50"
                    style={{
                      left: `${Math.max(14, Math.min(86, (hoverIdx / (chartHistory.length - 1)) * 100))}%`,
                    }}
                  >
                    <span className="font-mono text-slate-300 font-medium">
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

            {/* X-axis Timestamps - responsive and aligned to exact points */}
            <div className="relative w-full h-5 text-[10px] font-mono text-slate-400 mt-1 pt-1 border-t border-slate-100 dark:border-surface-800">
              {chartHistory.map((pt, idx) => {
                const isFirst = idx === 0;
                const isLast = idx === chartHistory.length - 1;
                const isMid = idx === Math.floor(chartHistory.length / 2);
                const pct = (idx / (chartHistory.length - 1)) * 100;

                return (
                  <span
                    key={idx}
                    className={`absolute top-1 select-none ${
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

        {/* Clean Footer */}
        <div className="py-3 text-center text-[11px] text-slate-400 dark:text-slate-500 space-y-1">
          <p>
            Hostvra Linux panel ©2024-2026 Hostvra{' '}
            <span className="text-emerald-600 dark:text-emerald-400 font-medium">www.hostvra.com</span> · Forum ·
            Documentation · Support: Telegram · Discord · Email: support@hostvra.com
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
      {chartExpanded && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/70 backdrop-blur-xs animate-fadeIn">
          <div className="bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-700 rounded-2xl w-full max-w-4xl shadow-2xl overflow-hidden p-6 space-y-4">
            <div className="flex items-center justify-between">
              <h3 className="text-base font-bold text-slate-900 dark:text-white flex items-center gap-2">
                <Maximize2 className="w-5 h-5 text-emerald-600" />
                Detailed Real-Time Monitoring ({chartTab === 'traffic' ? 'Network Traffic' : 'Disk I/O Throughput'})
              </h3>
              <button
                onClick={() => setChartExpanded(false)}
                className="w-7 h-7 rounded-full bg-slate-100 dark:bg-surface-800 text-slate-500 hover:text-slate-900 dark:hover:text-white flex items-center justify-center"
              >
                <X className="w-4 h-4" />
              </button>
            </div>

            <div className="flex items-center justify-between text-xs py-2 border-y border-slate-100 dark:border-surface-800">
              <div className="flex gap-3">
                <button
                  onClick={() => setChartTab('traffic')}
                  className={`font-bold pb-1 ${chartTab === 'traffic' ? 'text-emerald-600 border-b-2 border-emerald-600' : 'text-slate-500'}`}
                >
                  Network Traffic
                </button>
                <button
                  onClick={() => setChartTab('disk')}
                  className={`font-bold pb-1 ${chartTab === 'disk' ? 'text-emerald-600 border-b-2 border-emerald-600' : 'text-slate-500'}`}
                >
                  Disk I/O Throughput
                </button>
              </div>
              <span className="font-mono text-slate-500">Live Rate: {chartTab === 'traffic' ? telemetry.upstreamMb : telemetry.writeMb + '/s'}</span>
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
              <svg
                className="w-full h-56 overflow-hidden block"
                viewBox="0 0 600 180"
                preserveAspectRatio="none"
              >
                <defs>
                  <clipPath id="modalChartAreaClip">
                    <rect x="0" y="0" width="600" height="180" rx="4" />
                  </clipPath>
                  <linearGradient id="modalGreenGrad" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="0%" stopColor="#20a53a" stopOpacity="0.35" />
                    <stop offset="100%" stopColor="#20a53a" stopOpacity="0.02" />
                  </linearGradient>
                  <linearGradient id="modalOrangeGrad" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="0%" stopColor="#f59e0b" stopOpacity="0.35" />
                    <stop offset="100%" stopColor="#f59e0b" stopOpacity="0.02" />
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
                      className="text-slate-200 dark:text-surface-800"
                      strokeWidth="1"
                      vectorEffect="non-scaling-stroke"
                    />
                    {(i === 0 || i === 2 || i === 4) && (
                      <text
                        x="6"
                        y={i === 0 ? 12 : i === 4 ? 172 : y - 3}
                        className="text-[9px] font-mono fill-slate-400 dark:fill-slate-500 opacity-60 select-none"
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
                    stroke="#20a53a"
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
                    className="absolute -translate-x-1/2 -translate-y-1/2 w-2.5 h-2.5 rounded-full bg-[#20a53a] border border-white dark:border-surface-900 shadow-xs transition-transform duration-150"
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
                    <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-emerald-400 opacity-75"></span>
                    <span className="relative inline-flex rounded-full h-3.5 w-3.5 bg-emerald-500 border border-white dark:border-surface-900"></span>
                  </span>
                )}

                {modalHoverIdx !== null && chartHistory[modalHoverIdx] && (
                  <>
                    <div
                      className="absolute top-0 bottom-0 w-px border-l border-dashed border-slate-400 dark:border-slate-500 pointer-events-none z-20"
                      style={{ left: `${(modalHoverIdx / (chartHistory.length - 1)) * 100}%` }}
                    />
                    <div
                      className="absolute -top-1 -translate-y-full -translate-x-1/2 pointer-events-none z-30 px-3 py-1.5 rounded-lg bg-slate-900/95 dark:bg-surface-900/95 text-white text-xs shadow-xl backdrop-blur-md flex items-center gap-3 border border-slate-700/50"
                      style={{
                        left: `${Math.max(14, Math.min(86, (modalHoverIdx / (chartHistory.length - 1)) * 100))}%`,
                      }}
                    >
                      <span className="font-mono text-slate-300 font-medium">
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
                      className={`absolute top-1 select-none ${
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
                className="px-4 py-2 rounded-lg bg-slate-100 dark:bg-surface-800 text-slate-700 dark:text-slate-200 text-xs font-semibold hover:bg-slate-200 dark:hover:bg-surface-700 transition"
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
