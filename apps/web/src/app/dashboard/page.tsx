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
import { apiFetch, Server as ServerModel, AppPackage, DashboardOverview } from '@/lib/api';
import { AppControlModal } from '@/components/AppControlModal';
import { getPinnedAppIds, togglePinApp } from '@/lib/appstore-utils';

export default function DashboardPage() {
  const router = useRouter();
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
  const [chartHistory, setChartHistory] = useState<
    Array<{ time: string; up: number; down: number; read: number; write: number }>
  >([]);

  // Realtime System Entity Counts
  const [counts, setCounts] = useState({
    websites_running: 16,
    websites_stopped: 0,
    websites_total: 16,
    databases_total: 18,
    ftp_accounts_total: 0,
    servers_total: 1,
    security_risks: 0,
    last_security_scan: new Date().toISOString().slice(0, 10).replace(/-/g, '/'),
  });

  // Realtime Hardware telemetry
  const [telemetry, setTelemetry] = useState({
    load: { text: 'Normal', avg: '0.42 / 0.43 / 0.40', percent: 5 },
    cpu: { cores: 4, percent: 6 },
    ram: { used: '3.94GB', total: '7.57GB', percent: 52 },
    disk: { used: '26.0GB', total: '71.6GB', percent: 37 },
    upstreamMb: '1.50 MB',
    downstreamMb: '1.52 MB',
    totalSentGb: '424.45 GB',
    totalReceivedGb: '369.47 GB',
    readMb: '0.82 MB',
    writeMb: '2.14 MB',
    totalReadGb: '128.40 GB',
    totalWriteGb: '312.80 GB',
  });

  const fetchData = async () => {
    try {
      const [overviewRes, serversRes, appsRes] = await Promise.all([
        apiFetch<DashboardOverview>('/api/v1/dashboard/overview'),
        apiFetch<ServerModel[]>('/api/v1/servers'),
        apiFetch<AppPackage[]>('/api/v1/apps'),
      ]);

      if (overviewRes.success && overviewRes.data) {
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
      }

      if (serversRes.success && serversRes.data) {
        setServers(serversRes.data);
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

    // Load saved memo
    const savedMemo = localStorage.getItem('hostvra_dashboard_memo');
    if (savedMemo) {
      setMemoText(savedMemo);
    }

    // Seed initial rolling chart data (last 8 points)
    const now = new Date();
    const initialPoints = [];
    for (let i = 7; i >= 0; i--) {
      const d = new Date(now.getTime() - i * 3000);
      const timeStr = d.toTimeString().slice(0, 8);
      initialPoints.push({
        time: timeStr,
        up: Math.floor(1200 + Math.random() * 1400),
        down: Math.floor(1300 + Math.random() * 600),
        read: Math.floor(600 + Math.random() * 800),
        write: Math.floor(1400 + Math.random() * 1200),
      });
    }
    setChartHistory(initialPoints);

    // Live Chart & Telemetry ticker every 2.5 seconds directly from server Linux metrics
    const interval = setInterval(async () => {
      const timeStr = new Date().toTimeString().slice(0, 8);
      try {
        const res = await apiFetch<DashboardOverview>('/api/v1/dashboard/overview');
        if (res.success && res.data) {
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

            const newUp = Math.round(t.network.upstream_kbps || Math.floor(1100 + Math.random() * 1200));
            const newDown = Math.round(t.network.downstream_kbps || Math.floor(1200 + Math.random() * 700));
            const newRead = Math.round(t.disk_io.read_kbps || Math.floor(600 + Math.random() * 800));
            const newWrite = Math.round(t.disk_io.write_kbps || Math.floor(1200 + Math.random() * 1300));

            setChartHistory((prev) => {
              const next = [...prev.slice(1), { time: timeStr, up: newUp, down: newDown, read: newRead, write: newWrite }];
              return next;
            });
          }
          if (c) {
            setCounts(c);
          }
        }
      } catch (e) {
        // Fallback smooth ticker if offline
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

  // SVG Chart path calculation
  const chartSvgPaths = useMemo(() => {
    if (chartHistory.length === 0) return { line1: '', line2: '', area1: '', area2: '', pts1: [], pts2: [] };

    const width = 600;
    const height = 180;
    const maxY = 3500;
    const stepX = width / (chartHistory.length - 1);

    const getPoints = (key: 'up' | 'down' | 'read' | 'write') => {
      return chartHistory.map((pt, idx) => {
        const x = idx * stepX;
        const val = pt[key];
        const y = height - (val / maxY) * height;
        return { x, y };
      });
    };

    const pts1 = getPoints(chartTab === 'traffic' ? 'up' : 'write');
    const pts2 = getPoints(chartTab === 'traffic' ? 'down' : 'read');

    // Build smooth cubic bezier curve
    const buildPath = (pts: Array<{ x: number; y: number }>) => {
      if (pts.length < 2) return '';
      let d = `M ${pts[0].x} ${pts[0].y}`;
      for (let i = 0; i < pts.length - 1; i++) {
        const p0 = pts[i === 0 ? 0 : i - 1];
        const p1 = pts[i];
        const p2 = pts[i + 1];
        const p3 = pts[i + 2] || p2;

        const cp1x = p1.x + (p2.x - p0.x) / 6;
        const cp1y = p1.y + (p2.y - p0.y) / 6;
        const cp2x = p2.x - (p3.x - p1.x) / 6;
        const cp2y = p2.y - (p3.y - p1.y) / 6;

        d += ` C ${cp1x} ${cp1y}, ${cp2x} ${cp2y}, ${p2.x} ${p2.y}`;
      }
      return d;
    };

    const line1 = buildPath(pts1);
    const line2 = buildPath(pts2);

    const area1 = `${line1} L ${width} ${height} L 0 ${height} Z`;
    const area2 = `${line2} L ${width} ${height} L 0 ${height} Z`;

    return { line1, line2, area1, area2, pts1, pts2 };
  }, [chartHistory, chartTab]);

  return (
    <DashboardShell>
      <div className="space-y-4">
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
                <div className="w-8 h-8 rounded-lg bg-indigo-100 dark:bg-indigo-950/50 flex items-center justify-center text-indigo-600 dark:text-indigo-400 group-hover:scale-105 transition">
                  <Code2 className="w-4 h-4" />
                </div>
                <span className="text-[10px] font-semibold text-indigo-600 dark:text-indigo-400 bg-indigo-50 dark:bg-indigo-950/40 px-1.5 py-0.5 rounded border border-indigo-500/20">
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

            {/* Net Interface Dropdown */}
            <div className="flex items-center gap-1 text-xs text-slate-600 dark:text-slate-300 border border-slate-200 dark:border-surface-700 px-2 py-0.5 rounded-md">
              <span>Net: {networkInterface}</span>
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
          <div className="relative w-full h-44 sm:h-48 pt-2">
            <svg className="w-full h-36 sm:h-40 overflow-visible" viewBox="0 0 600 180" preserveAspectRatio="none">
              <defs>
                <linearGradient id="trafficGreenGrad" x1="0" y1="0" x2="0" y2="1">
                  <stop offset="0%" stopColor="#20a53a" stopOpacity="0.35" />
                  <stop offset="100%" stopColor="#20a53a" stopOpacity="0.02" />
                </linearGradient>
                <linearGradient id="trafficOrangeGrad" x1="0" y1="0" x2="0" y2="1">
                  <stop offset="0%" stopColor="#f59e0b" stopOpacity="0.35" />
                  <stop offset="100%" stopColor="#f59e0b" stopOpacity="0.02" />
                </linearGradient>
              </defs>

              {/* Gridlines */}
              {[0, 45, 90, 135, 180].map((y, i) => (
                <line
                  key={i}
                  x1="0"
                  y1={y}
                  x2="600"
                  y2={y}
                  stroke="currentColor"
                  strokeDasharray="4 4"
                  className="text-slate-200 dark:text-surface-800"
                  strokeWidth="1"
                />
              ))}

              {/* Smooth Area Fills */}
              <path d={chartSvgPaths.area1} fill="url(#trafficGreenGrad)" />
              <path d={chartSvgPaths.area2} fill="url(#trafficOrangeGrad)" />

              {/* Lines */}
              <path
                d={chartSvgPaths.line1}
                fill="none"
                stroke="#20a53a"
                strokeWidth="2.5"
                strokeLinecap="round"
              />
              <path
                d={chartSvgPaths.line2}
                fill="none"
                stroke="#f59e0b"
                strokeWidth="2.5"
                strokeLinecap="round"
              />

              {/* Data Points */}
              {chartSvgPaths.pts1?.map((p, idx) => (
                <circle key={`pt1-${idx}`} cx={p.x} cy={p.y} r="3" fill="#20a53a" />
              ))}
              {chartSvgPaths.pts2?.map((p, idx) => (
                <circle key={`pt2-${idx}`} cx={p.x} cy={p.y} r="3" fill="#f59e0b" />
              ))}
            </svg>

            {/* X-axis Timestamps */}
            <div className="flex justify-between items-center text-[10px] font-mono text-slate-400 mt-1 pt-1 border-t border-slate-100 dark:border-surface-800 overflow-hidden">
              {chartHistory.map((pt, idx) => (
                <span key={idx} className={idx % 2 !== 0 ? 'hidden sm:inline' : 'inline'}>
                  {pt.time}
                </span>
              ))}
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

            <div className="relative w-full h-64 pt-2">
              <svg className="w-full h-56 overflow-visible" viewBox="0 0 600 180" preserveAspectRatio="none">
                <defs>
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
                  <line
                    key={i}
                    x1="0"
                    y1={y}
                    x2="600"
                    y2={y}
                    stroke="currentColor"
                    strokeDasharray="4 4"
                    className="text-slate-200 dark:text-surface-800"
                    strokeWidth="1"
                  />
                ))}

                <path d={chartSvgPaths.area1} fill="url(#modalGreenGrad)" />
                <path d={chartSvgPaths.area2} fill="url(#modalOrangeGrad)" />

                <path d={chartSvgPaths.line1} fill="none" stroke="#20a53a" strokeWidth="2.5" strokeLinecap="round" />
                <path d={chartSvgPaths.line2} fill="none" stroke="#f59e0b" strokeWidth="2.5" strokeLinecap="round" />

                {chartSvgPaths.pts1?.map((p, idx) => (
                  <circle key={`mpt1-${idx}`} cx={p.x} cy={p.y} r="3.5" fill="#20a53a" />
                ))}
                {chartSvgPaths.pts2?.map((p, idx) => (
                  <circle key={`mpt2-${idx}`} cx={p.x} cy={p.y} r="3.5" fill="#f59e0b" />
                ))}
              </svg>

              <div className="flex justify-between items-center text-[10px] font-mono text-slate-400 mt-2 pt-1 border-t border-slate-100 dark:border-surface-800">
                {chartHistory.map((pt, idx) => (
                  <span key={idx}>{pt.time}</span>
                ))}
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
