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
  Play,
  Check,
  CheckCircle2,
  Bookmark,
  BookmarkCheck,
} from 'lucide-react';
import { DashboardShell } from '@/components/DashboardShell';
import { apiFetch, Server as ServerModel, AppPackage } from '@/lib/api';
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

  // Memo Note State
  const [memoText, setMemoText] = useState('');
  const [isEditingMemo, setIsEditingMemo] = useState(false);

  // Monitor Chart State (Traffic vs Disk IO)
  const [chartTab, setChartTab] = useState<'traffic' | 'disk'>('traffic');
  const [networkInterface, setNetworkInterface] = useState('All');
  const [chartHistory, setChartHistory] = useState<
    Array<{ time: string; up: number; down: number; read: number; write: number }>
  >([]);

  // Hardware telemetry simulator/live
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
      const [serversRes, appsRes] = await Promise.all([
        apiFetch<ServerModel[]>('/api/v1/servers'),
        apiFetch<AppPackage[]>('/api/v1/apps'),
      ]);

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

    // Live Chart & Telemetry ticker every 3 seconds
    const interval = setInterval(() => {
      const timeStr = new Date().toTimeString().slice(0, 8);
      const newUp = Math.floor(1000 + Math.random() * 1800);
      const newDown = Math.floor(1200 + Math.random() * 700);
      const newRead = Math.floor(500 + Math.random() * 900);
      const newWrite = Math.floor(1100 + Math.random() * 1500);

      setChartHistory((prev) => {
        const next = [...prev.slice(1), { time: timeStr, up: newUp, down: newDown, read: newRead, write: newWrite }];
        return next;
      });

      setTelemetry((prev) => ({
        ...prev,
        upstreamMb: (newUp / 1024).toFixed(2) + ' MB',
        downstreamMb: (newDown / 1024).toFixed(2) + ' MB',
        readMb: (newRead / 1024).toFixed(2) + ' MB',
        writeMb: (newWrite / 1024).toFixed(2) + ' MB',
      }));
    }, 3000);

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

  // SVG Chart path calculation
  const chartSvgPaths = useMemo(() => {
    if (chartHistory.length === 0) return { upPath: '', downPath: '', upArea: '', downArea: '' };

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
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
          {/* Card 1: Load Status */}
          <div className="p-4 rounded-xl bg-white dark:bg-[#121824] border border-slate-200 dark:border-surface-800 shadow-sm flex items-center justify-between">
            <div className="space-y-1">
              <span className="text-xs font-semibold text-slate-500 dark:text-slate-400">
                {telemetry.load.text}
              </span>
              <div className="flex items-center gap-1">
                <span className="text-xs font-mono font-bold text-slate-800 dark:text-slate-200">
                  {telemetry.load.avg}
                </span>
                <HelpCircle className="w-3.5 h-3.5 text-slate-400" />
              </div>
            </div>

            {/* Circular Gauge */}
            <div className="relative w-16 h-16 flex items-center justify-center">
              <svg className="w-16 h-16 transform -rotate-90" viewBox="0 0 80 80">
                <circle
                  cx="40"
                  cy="40"
                  r="34"
                  stroke="currentColor"
                  strokeWidth="6"
                  className="text-slate-100 dark:text-surface-800"
                  fill="transparent"
                />
                <circle
                  cx="40"
                  cy="40"
                  r="34"
                  stroke="#20a53a"
                  strokeWidth="6"
                  strokeDasharray={213.6}
                  strokeDashoffset={213.6 - (213.6 * telemetry.load.percent) / 100}
                  strokeLinecap="round"
                  fill="transparent"
                  className="transition-all duration-500"
                />
              </svg>
              <span className="absolute text-sm font-extrabold text-slate-900 dark:text-white">
                {telemetry.load.percent}%
              </span>
            </div>
          </div>

          {/* Card 2: CPU */}
          <div className="p-4 rounded-xl bg-white dark:bg-[#121824] border border-slate-200 dark:border-surface-800 shadow-sm flex items-center justify-between">
            <div className="space-y-1">
              <span className="text-xs font-semibold text-slate-500 dark:text-slate-400">CPU</span>
              <div className="text-xs font-mono font-bold text-slate-800 dark:text-slate-200">
                {telemetry.cpu.cores} Cores
              </div>
            </div>

            <div className="relative w-16 h-16 flex items-center justify-center">
              <svg className="w-16 h-16 transform -rotate-90" viewBox="0 0 80 80">
                <circle
                  cx="40"
                  cy="40"
                  r="34"
                  stroke="currentColor"
                  strokeWidth="6"
                  className="text-slate-100 dark:text-surface-800"
                  fill="transparent"
                />
                <circle
                  cx="40"
                  cy="40"
                  r="34"
                  stroke="#20a53a"
                  strokeWidth="6"
                  strokeDasharray={213.6}
                  strokeDashoffset={213.6 - (213.6 * telemetry.cpu.percent) / 100}
                  strokeLinecap="round"
                  fill="transparent"
                  className="transition-all duration-500"
                />
              </svg>
              <span className="absolute text-sm font-extrabold text-slate-900 dark:text-white">
                {telemetry.cpu.percent}%
              </span>
            </div>
          </div>

          {/* Card 3: RAM */}
          <div className="p-4 rounded-xl bg-white dark:bg-[#121824] border border-slate-200 dark:border-surface-800 shadow-sm flex items-center justify-between">
            <div className="space-y-1">
              <span className="text-xs font-semibold text-slate-500 dark:text-slate-400">RAM</span>
              <div className="text-xs font-mono font-bold text-slate-800 dark:text-slate-200">
                <span className="text-emerald-600 dark:text-emerald-400 font-bold">{telemetry.ram.used}</span> /{' '}
                {telemetry.ram.total}
              </div>
            </div>

            <div className="relative w-16 h-16 flex items-center justify-center">
              <svg className="w-16 h-16 transform -rotate-90" viewBox="0 0 80 80">
                <circle
                  cx="40"
                  cy="40"
                  r="34"
                  stroke="currentColor"
                  strokeWidth="6"
                  className="text-slate-100 dark:text-surface-800"
                  fill="transparent"
                />
                <circle
                  cx="40"
                  cy="40"
                  r="34"
                  stroke="#20a53a"
                  strokeWidth="6"
                  strokeDasharray={213.6}
                  strokeDashoffset={213.6 - (213.6 * telemetry.ram.percent) / 100}
                  strokeLinecap="round"
                  fill="transparent"
                  className="transition-all duration-500"
                />
              </svg>
              <span className="absolute text-sm font-extrabold text-slate-900 dark:text-white">
                {telemetry.ram.percent}%
              </span>
            </div>
          </div>

          {/* Card 4: Disk */}
          <div className="p-4 rounded-xl bg-white dark:bg-[#121824] border border-slate-200 dark:border-surface-800 shadow-sm flex items-center justify-between">
            <div className="space-y-1">
              <span className="text-xs font-semibold text-slate-500 dark:text-slate-400">/</span>
              <div className="text-xs font-mono font-bold text-slate-800 dark:text-slate-200">
                <span className="text-emerald-600 dark:text-emerald-400 font-bold">{telemetry.disk.used}</span> /{' '}
                {telemetry.disk.total}
              </div>
            </div>

            <div className="relative w-16 h-16 flex items-center justify-center">
              <svg className="w-16 h-16 transform -rotate-90" viewBox="0 0 80 80">
                <circle
                  cx="40"
                  cy="40"
                  r="34"
                  stroke="currentColor"
                  strokeWidth="6"
                  className="text-slate-100 dark:text-surface-800"
                  fill="transparent"
                />
                <circle
                  cx="40"
                  cy="40"
                  r="34"
                  stroke="#20a53a"
                  strokeWidth="6"
                  strokeDasharray={213.6}
                  strokeDashoffset={213.6 - (213.6 * telemetry.disk.percent) / 100}
                  strokeLinecap="round"
                  fill="transparent"
                  className="transition-all duration-500"
                />
              </svg>
              <span className="absolute text-sm font-extrabold text-slate-900 dark:text-white">
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

          <div className="grid grid-cols-2 md:grid-cols-5 gap-4 text-xs">
            {/* Site - ALL */}
            <Link
              href="/websites"
              className="flex items-center justify-between p-2.5 rounded-lg hover:bg-slate-50 dark:hover:bg-surface-800/60 transition group"
            >
              <div>
                <span className="font-semibold text-slate-700 dark:text-slate-200 group-hover:text-emerald-600 dark:group-hover:text-emerald-400">
                  Site - ALL
                </span>
                <div className="mt-1 flex items-center gap-2">
                  <span className="text-emerald-600 dark:text-emerald-400 font-bold">Running: 16</span>
                  <span className="text-slate-400">Stopped: 0</span>
                </div>
                <span className="text-[11px] text-slate-400">ALL: 16</span>
              </div>
              <Globe className="w-4 h-4 text-emerald-600 dark:text-emerald-400" />
            </Link>

            {/* FTP */}
            <Link
              href="/servers"
              className="flex items-center justify-between p-2.5 rounded-lg hover:bg-slate-50 dark:hover:bg-surface-800/60 transition group"
            >
              <div>
                <span className="font-semibold text-slate-700 dark:text-slate-200 group-hover:text-emerald-600 dark:group-hover:text-emerald-400">
                  FTP
                </span>
                <div className="mt-1 text-emerald-600 dark:text-emerald-400 font-bold">Accounts: 0</div>
              </div>
              <ArrowLeftRight className="w-4 h-4 text-emerald-600 dark:text-emerald-400" />
            </Link>

            {/* Database */}
            <Link
              href="/databases"
              className="flex items-center justify-between p-2.5 rounded-lg hover:bg-slate-50 dark:hover:bg-surface-800/60 transition group"
            >
              <div>
                <span className="font-semibold text-slate-700 dark:text-slate-200 group-hover:text-emerald-600 dark:group-hover:text-emerald-400">
                  Database
                </span>
                <div className="mt-1 text-emerald-600 dark:text-emerald-400 font-bold">ALL: 18</div>
              </div>
              <Database className="w-4 h-4 text-emerald-600 dark:text-emerald-400" />
            </Link>

            {/* Security */}
            <Link
              href="/firewall"
              className="flex items-center justify-between p-2.5 rounded-lg hover:bg-slate-50 dark:hover:bg-surface-800/60 transition group"
            >
              <div>
                <span className="font-semibold text-slate-700 dark:text-slate-200 group-hover:text-emerald-600 dark:group-hover:text-emerald-400">
                  Security
                </span>
                <div className="mt-1 text-slate-600 dark:text-slate-300">Security Risk: 0</div>
                <span className="text-[10px] text-slate-400 block mt-0.5">Last Scan: 2026/09/07 05:01:35</span>
              </div>
              <Shield className="w-4 h-4 text-emerald-600 dark:text-emerald-400" />
            </Link>

            {/* Memo (Editable) */}
            <div
              id="dashboard-memo-card"
              onClick={() => setIsEditingMemo(true)}
              className="flex items-start justify-between p-2.5 rounded-lg bg-slate-50 dark:bg-surface-800/40 border border-dashed border-slate-200 dark:border-surface-700 hover:border-emerald-500 cursor-pointer transition"
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
                    className="w-full mt-1 p-1 text-xs bg-white dark:bg-surface-900 border border-slate-300 dark:border-surface-700 rounded text-slate-800 dark:text-slate-200 focus:outline-none focus:ring-1 focus:ring-emerald-500 resize-none h-14"
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

        {/* ROW 3: Two Column Layout: Left = Software, Right = Traffic/Disk IO Monitor */}
        <div className="grid grid-cols-1 lg:grid-cols-12 gap-4">
          {/* Left Column: Software (Quick Launch / App Manager) */}
          <div className="lg:col-span-5 p-4 rounded-xl bg-white dark:bg-[#121824] border border-slate-200 dark:border-surface-800 shadow-sm space-y-4 flex flex-col justify-between">
            <div>
              <div className="flex items-center justify-between border-b border-slate-100 dark:border-surface-800 pb-2.5">
                <h2 className="text-xs font-bold text-slate-800 dark:text-slate-100">Software</h2>
                <button className="text-slate-400 hover:text-slate-600 dark:hover:text-slate-200">
                  <MoreVertical className="w-3.5 h-3.5" />
                </button>
              </div>

              {/* Grid of App Tiles (Matches Screenshot 1 & 2) */}
              <div className="grid grid-cols-3 sm:grid-cols-5 gap-3 pt-3">
                {/* Supervisor 3.1 Tile (Exact Match to Reference Screenshot) */}
                <div
                  onClick={() => {
                    const supervisorApp = installedApps.find((a) => a.id.toLowerCase().includes('supervisor')) || {
                      id: 'supervisor',
                      name: 'Supervisor',
                      display_name: 'Supervisor Manager',
                      version: '3.1',
                      category: 'process_manager',
                      is_installed: true,
                      status: 'running',
                      description: 'Daemon process manager for background workers.',
                      config_path: '/etc/supervisor/supervisord.conf',
                    };
                    handleOpenApp(supervisorApp as AppPackage);
                  }}
                  className="flex flex-col items-center justify-center p-2 rounded-xl border border-slate-200 dark:border-surface-800 hover:border-emerald-500 hover:shadow-md cursor-pointer transition bg-slate-50/50 dark:bg-surface-800/30 group relative"
                >
                  <div className="w-10 h-10 rounded-xl bg-sky-100 dark:bg-sky-950/40 flex items-center justify-center text-sky-600 dark:text-sky-400 mb-1.5 shadow-sm group-hover:scale-105 transition">
                    <span className="text-lg">🐧</span>
                  </div>
                  <span className="text-[11px] font-bold text-slate-800 dark:text-slate-200 text-center flex items-center gap-1">
                    Supervisor 3.1
                    <span className="text-emerald-500 text-[9px]">▶</span>
                  </span>
                </div>

                {/* AD Tile 1: Website Firewall (WAF) */}
                <div
                  onClick={() => router.push('/firewall')}
                  className="flex flex-col items-center justify-between p-2 rounded-xl border border-slate-200 dark:border-surface-800 hover:border-emerald-500 cursor-pointer transition bg-slate-50/50 dark:bg-surface-800/30 relative"
                >
                  <span className="absolute top-1 left-1 text-[8px] font-bold px-1 rounded bg-slate-200 dark:bg-surface-700 text-slate-500">
                    AD
                  </span>
                  <div className="w-9 h-9 rounded-xl bg-emerald-100 dark:bg-emerald-950/40 flex items-center justify-center text-emerald-600 dark:text-emerald-400 mt-1">
                    <Shield className="w-4 h-4" />
                  </div>
                  <span className="text-[10px] font-medium text-slate-700 dark:text-slate-300 text-center truncate w-full mt-1">
                    Website Firewall
                  </span>
                  <div className="flex items-center gap-1 mt-1">
                    <span className="text-[9px] px-1.5 py-0.2 rounded bg-slate-200 dark:bg-surface-700 text-slate-600 dark:text-slate-300">
                      Preview
                    </span>
                    <span className="text-[9px] px-1.5 py-0.2 rounded bg-emerald-600 text-white font-bold">
                      Buy now
                    </span>
                  </div>
                </div>

                {/* AD Tile 2: Website statistics-v2 */}
                <div
                  onClick={() => router.push('/audit-logs')}
                  className="flex flex-col items-center justify-between p-2 rounded-xl border border-slate-200 dark:border-surface-800 hover:border-emerald-500 cursor-pointer transition bg-slate-50/50 dark:bg-surface-800/30 relative"
                >
                  <span className="absolute top-1 left-1 text-[8px] font-bold px-1 rounded bg-slate-200 dark:bg-surface-700 text-slate-500">
                    AD
                  </span>
                  <div className="w-9 h-9 rounded-xl bg-blue-100 dark:bg-blue-950/40 flex items-center justify-center text-blue-600 dark:text-blue-400 mt-1">
                    <Target className="w-4 h-4" />
                  </div>
                  <span className="text-[10px] font-medium text-slate-700 dark:text-slate-300 text-center truncate w-full mt-1">
                    Website statistics-v2
                  </span>
                  <div className="flex items-center gap-1 mt-1">
                    <span className="text-[9px] px-1.5 py-0.2 rounded bg-slate-200 dark:bg-surface-700 text-slate-600 dark:text-slate-300">
                      Preview
                    </span>
                    <span className="text-[9px] px-1.5 py-0.2 rounded bg-emerald-600 text-white font-bold">
                      Buy now
                    </span>
                  </div>
                </div>

                {/* AD Tile 3: Website Tamper-proof for Enterprise */}
                <div
                  onClick={() => router.push('/ssl')}
                  className="flex flex-col items-center justify-between p-2 rounded-xl border border-emerald-500/30 dark:border-emerald-500/20 bg-emerald-50/30 dark:bg-emerald-950/20 hover:border-emerald-500 cursor-pointer transition relative"
                >
                  <span className="absolute top-1 left-1 text-[8px] font-bold px-1 rounded bg-emerald-100 dark:bg-emerald-900 text-emerald-700 dark:text-emerald-300">
                    AD
                  </span>
                  <div className="w-9 h-9 rounded-xl bg-emerald-100 dark:bg-emerald-950/40 flex items-center justify-center text-emerald-600 dark:text-emerald-400 mt-1">
                    <Lock className="w-4 h-4" />
                  </div>
                  <span className="text-[10px] font-medium text-slate-700 dark:text-slate-300 text-center truncate w-full mt-1">
                    Website Tamper-proof
                  </span>
                  <div className="flex items-center gap-1 mt-1">
                    <span className="text-[9px] px-1.5 py-0.2 rounded bg-slate-200 dark:bg-surface-700 text-slate-600 dark:text-slate-300">
                      Preview
                    </span>
                    <span className="text-[9px] px-1.5 py-0.2 rounded bg-emerald-600 text-white font-bold">
                      Buy now
                    </span>
                  </div>
                </div>

                {/* AD Tile 4: Anti-intrusion */}
                <div
                  onClick={() => router.push('/firewall')}
                  className="flex flex-col items-center justify-between p-2 rounded-xl border border-slate-200 dark:border-surface-800 hover:border-emerald-500 cursor-pointer transition bg-slate-50/50 dark:bg-surface-800/30 relative"
                >
                  <span className="absolute top-1 left-1 text-[8px] font-bold px-1 rounded bg-slate-200 dark:bg-surface-700 text-slate-500">
                    AD
                  </span>
                  <div className="w-9 h-9 rounded-xl bg-purple-100 dark:bg-purple-950/40 flex items-center justify-center text-purple-600 dark:text-purple-400 mt-1">
                    <ShieldAlert className="w-4 h-4" />
                  </div>
                  <span className="text-[10px] font-medium text-slate-700 dark:text-slate-300 text-center truncate w-full mt-1">
                    Anti-intrusion
                  </span>
                  <div className="flex items-center gap-1 mt-1">
                    <span className="text-[9px] px-1.5 py-0.2 rounded bg-slate-200 dark:bg-surface-700 text-slate-600 dark:text-slate-300">
                      Preview
                    </span>
                    <span className="text-[9px] px-1.5 py-0.2 rounded bg-emerald-600 text-white font-bold">
                      Buy now
                    </span>
                  </div>
                </div>
              </div>

              {/* Installed User Packages List */}
              {installedApps.length > 0 && (
                <div className="pt-3 border-t border-slate-100 dark:border-surface-800 mt-3">
                  <div className="flex items-center justify-between text-[11px] font-bold text-slate-500 dark:text-slate-400 mb-2">
                    <span>Installed Services ({installedApps.length})</span>
                    <Link href="/app-store" className="text-emerald-600 dark:text-emerald-400 hover:underline">
                      + Add Software
                    </Link>
                  </div>
                  <div className="grid grid-cols-2 sm:grid-cols-3 gap-2">
                    {installedApps.slice(0, 6).map((app) => (
                      <button
                        key={app.id}
                        onClick={() => handleOpenApp(app)}
                        className="p-2 rounded-lg border border-slate-200 dark:border-surface-800 hover:border-emerald-500 bg-white dark:bg-surface-900 flex items-center gap-2 text-left transition shadow-xs group"
                      >
                        <span className="w-2 h-2 rounded-full bg-emerald-500 animate-pulse flex-shrink-0" />
                        <span className="text-xs font-semibold text-slate-800 dark:text-slate-200 truncate group-hover:text-emerald-600 dark:group-hover:text-emerald-400">
                          {app.name}
                        </span>
                      </button>
                    ))}
                  </div>
                </div>
              )}
            </div>

            <div className="pt-2">
              <Link
                href="/app-store"
                className="w-full py-2 rounded-lg bg-slate-100 dark:bg-surface-800 hover:bg-slate-200 dark:hover:bg-surface-700 text-slate-700 dark:text-slate-200 text-xs font-bold text-center block transition"
              >
                Open App Store
              </Link>
            </div>
          </div>

          {/* Right Column: Traffic & Disk IO Realtime Monitor */}
          <div className="lg:col-span-7 p-4 rounded-xl bg-white dark:bg-[#121824] border border-slate-200 dark:border-surface-800 shadow-sm space-y-3">
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
                <button className="text-slate-400 hover:text-slate-600 dark:hover:text-slate-200">
                  <Maximize2 className="w-3.5 h-3.5" />
                </button>
              </div>

              {/* Net Interface Dropdown */}
              <div className="flex items-center gap-1 text-xs text-slate-600 dark:text-slate-300 border border-slate-200 dark:border-surface-700 px-2 py-0.5 rounded-md">
                <span>Net: {networkInterface}</span>
              </div>
            </div>

            {/* Realtime Telemetry Stats Strip */}
            <div className="grid grid-cols-2 sm:grid-cols-4 gap-2 text-xs py-1">
              {chartTab === 'traffic' ? (
                <>
                  <div className="flex items-center gap-2">
                    <span className="w-2 h-2 rounded-full bg-emerald-500" />
                    <span className="text-slate-500 dark:text-slate-400">Upstream:</span>
                    <span className="font-mono font-bold text-slate-800 dark:text-slate-200">
                      {telemetry.upstreamMb}
                    </span>
                  </div>
                  <div className="flex items-center gap-2">
                    <span className="w-2 h-2 rounded-full bg-amber-500" />
                    <span className="text-slate-500 dark:text-slate-400">Downstream:</span>
                    <span className="font-mono font-bold text-slate-800 dark:text-slate-200">
                      {telemetry.downstreamMb}
                    </span>
                  </div>
                  <div>
                    <span className="text-slate-500 dark:text-slate-400">Total sent: </span>
                    <span className="font-mono font-bold text-slate-800 dark:text-slate-200">
                      {telemetry.totalSentGb}
                    </span>
                  </div>
                  <div>
                    <span className="text-slate-500 dark:text-slate-400">Total received: </span>
                    <span className="font-mono font-bold text-slate-800 dark:text-slate-200">
                      {telemetry.totalReceivedGb}
                    </span>
                  </div>
                </>
              ) : (
                <>
                  <div className="flex items-center gap-2">
                    <span className="w-2 h-2 rounded-full bg-emerald-500" />
                    <span className="text-slate-500 dark:text-slate-400">Read:</span>
                    <span className="font-mono font-bold text-slate-800 dark:text-slate-200">
                      {telemetry.readMb}/s
                    </span>
                  </div>
                  <div className="flex items-center gap-2">
                    <span className="w-2 h-2 rounded-full bg-amber-500" />
                    <span className="text-slate-500 dark:text-slate-400">Write:</span>
                    <span className="font-mono font-bold text-slate-800 dark:text-slate-200">
                      {telemetry.writeMb}/s
                    </span>
                  </div>
                  <div>
                    <span className="text-slate-500 dark:text-slate-400">Total read: </span>
                    <span className="font-mono font-bold text-slate-800 dark:text-slate-200">
                      {telemetry.totalReadGb}
                    </span>
                  </div>
                  <div>
                    <span className="text-slate-500 dark:text-slate-400">Total write: </span>
                    <span className="font-mono font-bold text-slate-800 dark:text-slate-200">
                      {telemetry.totalWriteGb}
                    </span>
                  </div>
                </>
              )}
            </div>

            {/* Realtime Smooth SVG Area Chart */}
            <div className="relative w-full h-48 pt-2">
              <svg className="w-full h-40 overflow-visible" viewBox="0 0 600 180" preserveAspectRatio="none">
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
              <div className="flex justify-between items-center text-[10px] font-mono text-slate-400 mt-1 pt-1 border-t border-slate-100 dark:border-surface-800">
                {chartHistory.map((pt, idx) => (
                  <span key={idx}>{pt.time}</span>
                ))}
              </div>
            </div>
          </div>
        </div>

        {/* ROW 4: Bottom PRO Banner (Matches Screenshot 1 & 2) */}
        <div className="p-3 rounded-xl bg-white dark:bg-[#121824] border border-slate-200 dark:border-surface-800 shadow-sm flex flex-wrap items-center justify-between gap-3 text-xs">
          <div className="flex items-center gap-2">
            <button className="px-3 py-1 rounded bg-amber-500 hover:bg-amber-400 text-white font-bold text-xs shadow-xs transition">
              Buy now
            </button>
            <span className="px-2 py-0.5 rounded bg-slate-900 text-amber-400 font-bold text-[10px] border border-amber-400/20">
              PRO
            </span>
          </div>

          <div className="flex flex-wrap items-center gap-4 text-slate-600 dark:text-slate-300 text-[11px]">
            <span className="flex items-center gap-1">
              <Check className="w-3.5 h-3.5 text-amber-500" />
              All paid plugins
            </span>
            <span className="flex items-center gap-1">
              <Check className="w-3.5 h-3.5 text-amber-500" />
              15 days no reason to refund
            </span>
            <span className="flex items-center gap-1">
              <Check className="w-3.5 h-3.5 text-amber-500" />
              Replaceable authorized IP
            </span>
            <span className="flex items-center gap-1">
              <Check className="w-3.5 h-3.5 text-amber-500" />
              2 free 1-year DV SSL (for year)
            </span>
            <span className="flex items-center gap-1">
              <Check className="w-3.5 h-3.5 text-amber-500" />
              Priority response service
            </span>
            <span className="flex items-center gap-1">
              <Check className="w-3.5 h-3.5 text-amber-500" />
              Paid service group (for year)
            </span>
          </div>
        </div>

        {/* ROW 5: Footer */}
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
    </DashboardShell>
  );
}
