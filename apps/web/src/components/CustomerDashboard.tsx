'use client';

import React, { useState, useEffect } from 'react';
import Link from 'next/link';
import {
  Globe,
  Database,
  Mail,
  FolderTree,
  ShieldCheck,
  DownloadCloud,
  Clock,
  Terminal,
  Boxes,
  Zap,
  HardDrive,
  Activity,
  ArrowRight,
  ExternalLink,
  Plus,
  RefreshCw,
  AlertTriangle,
  CheckCircle2,
  Lock,
  Layers,
  CreditCard,
  LifeBuoy,
  FileCode2,
} from 'lucide-react';
import {
  apiFetch,
  EffectiveUserPlan,
  fetchUserEffectivePlan,
  Website as WebsiteModel,
} from '@/lib/api';

export function CustomerDashboard() {
  const [plan, setPlan] = useState<EffectiveUserPlan | null>(null);
  const [websites, setWebsites] = useState<WebsiteModel[]>([]);
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);

  const loadData = async () => {
    try {
      const [planRes, sitesRes] = await Promise.all([
        fetchUserEffectivePlan(),
        apiFetch<WebsiteModel[]>('/api/v1/websites'),
      ]);

      if (planRes.success && planRes.data) {
        setPlan(planRes.data);
      }
      if (sitesRes.success && sitesRes.data) {
        setWebsites(sitesRes.data);
      }
    } catch (err) {
      console.error('Failed to load user hosting dashboard data', err);
    } finally {
      setLoading(false);
      setRefreshing(false);
    }
  };

  useEffect(() => {
    loadData();
  }, []);

  const handleRefresh = () => {
    setRefreshing(true);
    loadData();
  };

  const getPercent = (used: number, max: number) => {
    if (max <= 0) return 0;
    const p = Math.round((used / max) * 100);
    return Math.min(p, 100);
  };

  const getProgressColor = (percent: number) => {
    if (percent >= 90) return 'bg-rose-500';
    if (percent >= 75) return 'bg-amber-500';
    return 'bg-emerald-500';
  };

  const formatMB = (mb: number) => {
    if (mb >= 1024) {
      return `${(mb / 1024).toFixed(1)} GB`;
    }
    return `${mb} MB`;
  };

  if (loading) {
    return (
      <div className="flex flex-col items-center justify-center min-h-[500px] space-y-4">
        <div className="w-10 h-10 border-4 border-emerald-500/20 border-t-emerald-500 rounded-full animate-spin" />
        <p className="text-sm font-medium text-slate-500 dark:text-slate-400">Loading your hosting portal...</p>
      </div>
    );
  }

  const usage = plan?.usage || {
    websites_count: websites.length,
    databases_count: 0,
    mailboxes_count: 0,
    ftp_count: 0,
    cron_count: 0,
    subdomains_count: 0,
    disk_used_mb: 0,
    bandwidth_used_mb: 0,
  };

  const maxSites = plan?.max_websites ?? 1;
  const maxDbs = plan?.max_databases ?? 2;
  const maxMail = plan?.max_mailboxes ?? 5;
  const maxDisk = plan?.disk_space_mb ?? 10240;
  const maxBandwidth = plan?.bandwidth_mb ?? 102400;

  const sitesPercent = getPercent(usage.websites_count, maxSites);
  const dbsPercent = getPercent(usage.databases_count, maxDbs);
  const mailPercent = getPercent(usage.mailboxes_count, maxMail);
  const diskPercent = getPercent(Number(usage.disk_used_mb), Number(maxDisk));
  const bwPercent = getPercent(Number(usage.bandwidth_used_mb), Number(maxBandwidth));

  const hasTerminal = plan?.permissions?.terminal ?? false;

  return (
    <div className="space-y-8 pb-12">
      {/* Top Banner / Welcome Card */}
      <div className="relative overflow-hidden rounded-2xl bg-gradient-to-r from-slate-900 via-indigo-950 to-slate-900 border border-slate-800 p-6 sm:p-8 text-white shadow-xl">
        <div className="absolute top-0 right-0 -mt-12 -mr-12 w-64 h-64 bg-indigo-500/20 rounded-full blur-3xl pointer-events-none" />
        <div className="absolute bottom-0 left-1/3 -mb-8 w-48 h-48 bg-emerald-500/10 rounded-full blur-2xl pointer-events-none" />

        <div className="relative z-10 flex flex-col md:flex-row items-start md:items-center justify-between gap-6">
          <div className="space-y-2">
            <div className="inline-flex items-center gap-2 px-3 py-1 rounded-full text-xs font-semibold bg-emerald-500/20 text-emerald-400 border border-emerald-500/30">
              <span className="w-2 h-2 rounded-full bg-emerald-400 animate-pulse" />
              Hosting Account Active
            </div>
            <h1 className="text-2xl sm:text-3xl font-black tracking-tight">
              Welcome back, {plan?.user_name || 'Customer'}
            </h1>
            <p className="text-sm text-slate-300 max-w-xl">
              Manage your websites, databases, webmail, files and security credentials from your high-performance cloud portal.
            </p>
          </div>

          <div className="flex flex-wrap items-center gap-3">
            <button
              onClick={handleRefresh}
              disabled={refreshing}
              className="inline-flex items-center gap-2 px-3.5 py-2 rounded-xl text-xs font-semibold bg-white/10 hover:bg-white/15 text-white transition-colors border border-white/10"
              title="Refresh Quota"
            >
              <RefreshCw className={`w-3.5 h-3.5 ${refreshing ? 'animate-spin' : ''}`} />
              Sync Usage
            </button>
            <Link
              href="/websites"
              className="inline-flex items-center gap-2 px-4 py-2 rounded-xl text-xs font-bold bg-[#16A34A] hover:bg-[#15803D] text-white shadow-lg shadow-emerald-600/30 transition-all hover:scale-105"
            >
              <Plus className="w-4 h-4" />
              Add Website
            </Link>
          </div>
        </div>

        {/* Plan Header Strip */}
        <div className="mt-6 pt-6 border-t border-slate-800/80 flex flex-wrap items-center justify-between gap-4 text-xs">
          <div className="flex items-center gap-4">
            <span className="text-slate-400">Current Package:</span>
            <span className="font-bold text-white px-2.5 py-1 rounded-lg bg-indigo-500/20 border border-indigo-500/30 text-indigo-300">
              {plan?.plan_name || 'Starter Cloud'}
            </span>
            {plan?.subscription_status === 'trial' && (
              <span className="font-medium text-amber-400 bg-amber-500/10 px-2 py-0.5 rounded border border-amber-500/20">
                14-Day Free Trial
              </span>
            )}
          </div>
          <Link
            href="/billing"
            className="inline-flex items-center gap-1.5 text-indigo-400 hover:text-indigo-300 font-semibold transition-colors"
          >
            Upgrade or Extend Plan <ArrowRight className="w-3.5 h-3.5" />
          </Link>
        </div>
      </div>

      {/* Quota & Capacity Meters Grid */}
      <div>
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-lg font-bold text-slate-900 dark:text-white flex items-center gap-2">
            <Activity className="w-5 h-5 text-indigo-500" />
            Package Resource Quotas
          </h2>
          <span className="text-xs text-slate-500 dark:text-slate-400">
            Real-time backend quota enforcement
          </span>
        </div>

        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
          {/* Websites Meter */}
          <div className="bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-800 rounded-2xl p-5 shadow-sm space-y-3">
            <div className="flex items-center justify-between">
              <span className="text-xs font-semibold text-slate-500 dark:text-slate-400 flex items-center gap-1.5">
                <Globe className="w-4 h-4 text-emerald-500" /> Websites
              </span>
              <span className="text-xs font-bold text-slate-900 dark:text-white">
                {usage.websites_count} / {maxSites}
              </span>
            </div>
            <div className="w-full h-2 rounded-full bg-slate-100 dark:bg-surface-800 overflow-hidden">
              <div
                className={`h-full rounded-full transition-all duration-500 ${getProgressColor(sitesPercent)}`}
                style={{ width: `${sitesPercent}%` }}
              />
            </div>
            <div className="flex items-center justify-between text-[11px] text-slate-500 dark:text-slate-400">
              <span>{sitesPercent}% Used</span>
              <span>{Math.max(0, maxSites - usage.websites_count)} remaining</span>
            </div>
          </div>

          {/* Databases Meter */}
          <div className="bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-800 rounded-2xl p-5 shadow-sm space-y-3">
            <div className="flex items-center justify-between">
              <span className="text-xs font-semibold text-slate-500 dark:text-slate-400 flex items-center gap-1.5">
                <Database className="w-4 h-4 text-blue-500" /> Databases
              </span>
              <span className="text-xs font-bold text-slate-900 dark:text-white">
                {usage.databases_count} / {maxDbs}
              </span>
            </div>
            <div className="w-full h-2 rounded-full bg-slate-100 dark:bg-surface-800 overflow-hidden">
              <div
                className={`h-full rounded-full transition-all duration-500 ${getProgressColor(dbsPercent)}`}
                style={{ width: `${dbsPercent}%` }}
              />
            </div>
            <div className="flex items-center justify-between text-[11px] text-slate-500 dark:text-slate-400">
              <span>{dbsPercent}% Used</span>
              <span>{Math.max(0, maxDbs - usage.databases_count)} remaining</span>
            </div>
          </div>

          {/* Mailboxes Meter */}
          <div className="bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-800 rounded-2xl p-5 shadow-sm space-y-3">
            <div className="flex items-center justify-between">
              <span className="text-xs font-semibold text-slate-500 dark:text-slate-400 flex items-center gap-1.5">
                <Mail className="w-4 h-4 text-purple-500" /> Email Inboxes
              </span>
              <span className="text-xs font-bold text-slate-900 dark:text-white">
                {usage.mailboxes_count} / {maxMail}
              </span>
            </div>
            <div className="w-full h-2 rounded-full bg-slate-100 dark:bg-surface-800 overflow-hidden">
              <div
                className={`h-full rounded-full transition-all duration-500 ${getProgressColor(mailPercent)}`}
                style={{ width: `${mailPercent}%` }}
              />
            </div>
            <div className="flex items-center justify-between text-[11px] text-slate-500 dark:text-slate-400">
              <span>{mailPercent}% Used</span>
              <span>{Math.max(0, maxMail - usage.mailboxes_count)} remaining</span>
            </div>
          </div>

          {/* Storage (Disk) Meter */}
          <div className="bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-800 rounded-2xl p-5 shadow-sm space-y-3">
            <div className="flex items-center justify-between">
              <span className="text-xs font-semibold text-slate-500 dark:text-slate-400 flex items-center gap-1.5">
                <HardDrive className="w-4 h-4 text-amber-500" /> NVMe Storage
              </span>
              <span className="text-xs font-bold text-slate-900 dark:text-white">
                {formatMB(Number(usage.disk_used_mb))} / {formatMB(Number(maxDisk))}
              </span>
            </div>
            <div className="w-full h-2 rounded-full bg-slate-100 dark:bg-surface-800 overflow-hidden">
              <div
                className={`h-full rounded-full transition-all duration-500 ${getProgressColor(diskPercent)}`}
                style={{ width: `${diskPercent}%` }}
              />
            </div>
            <div className="flex items-center justify-between text-[11px] text-slate-500 dark:text-slate-400">
              <span>{diskPercent}% Used</span>
              <span>{formatMB(Math.max(0, Number(maxDisk) - Number(usage.disk_used_mb)))} free</span>
            </div>
          </div>
        </div>
      </div>

      {/* Quick Launch Hosting Tools */}
      <div>
        <h2 className="text-lg font-bold text-slate-900 dark:text-white mb-4 flex items-center gap-2">
          <Zap className="w-5 h-5 text-amber-500" />
          Hosting Control Center
        </h2>

        <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-4 gap-4">
          <Link
            href="/websites"
            className="group p-5 bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-800 rounded-2xl hover:border-emerald-500 dark:hover:border-emerald-500/50 hover:shadow-md transition-all flex flex-col justify-between"
          >
            <div className="w-10 h-10 rounded-xl bg-emerald-50 dark:bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 flex items-center justify-center mb-3 group-hover:scale-110 transition-transform">
              <Globe className="w-5 h-5" />
            </div>
            <div>
              <h3 className="font-bold text-sm text-slate-900 dark:text-white group-hover:text-emerald-600 dark:group-hover:text-emerald-400 transition-colors">
                Websites & Domains
              </h3>
              <p className="text-xs text-slate-500 dark:text-slate-400 mt-1">
                Configure vHosts, domains and PHP settings
              </p>
            </div>
          </Link>

          <Link
            href="/databases"
            className="group p-5 bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-800 rounded-2xl hover:border-blue-500 dark:hover:border-blue-500/50 hover:shadow-md transition-all flex flex-col justify-between"
          >
            <div className="w-10 h-10 rounded-xl bg-blue-50 dark:bg-blue-500/10 text-blue-600 dark:text-blue-400 flex items-center justify-center mb-3 group-hover:scale-110 transition-transform">
              <Database className="w-5 h-5" />
            </div>
            <div>
              <h3 className="font-bold text-sm text-slate-900 dark:text-white group-hover:text-blue-600 dark:group-hover:text-blue-400 transition-colors">
                Databases (MySQL)
              </h3>
              <p className="text-xs text-slate-500 dark:text-slate-400 mt-1">
                Manage SQL schemas, database users and privileges
              </p>
            </div>
          </Link>

          <Link
            href="/phpmyadmin"
            className="group p-5 bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-800 rounded-2xl hover:border-amber-500 dark:hover:border-amber-500/50 hover:shadow-md transition-all flex flex-col justify-between"
          >
            <div className="w-10 h-10 rounded-xl bg-amber-50 dark:bg-amber-500/10 text-amber-600 dark:text-amber-400 flex items-center justify-center mb-3 group-hover:scale-110 transition-transform">
              <Layers className="w-5 h-5" />
            </div>
            <div>
              <h3 className="font-bold text-sm text-slate-900 dark:text-white group-hover:text-amber-600 dark:group-hover:text-amber-400 transition-colors">
                phpMyAdmin
              </h3>
              <p className="text-xs text-slate-500 dark:text-slate-400 mt-1">
                Visual database web GUI and table management
              </p>
            </div>
          </Link>

          <Link
            href="/files"
            className="group p-5 bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-800 rounded-2xl hover:border-indigo-500 dark:hover:border-indigo-500/50 hover:shadow-md transition-all flex flex-col justify-between"
          >
            <div className="w-10 h-10 rounded-xl bg-indigo-50 dark:bg-indigo-500/10 text-indigo-600 dark:text-indigo-400 flex items-center justify-center mb-3 group-hover:scale-110 transition-transform">
              <FolderTree className="w-5 h-5" />
            </div>
            <div>
              <h3 className="font-bold text-sm text-slate-900 dark:text-white group-hover:text-indigo-600 dark:group-hover:text-indigo-400 transition-colors">
                File Manager
              </h3>
              <p className="text-xs text-slate-500 dark:text-slate-400 mt-1">
                Upload files, edit code and manage permissions
              </p>
            </div>
          </Link>

          <Link
            href="/email"
            className="group p-5 bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-800 rounded-2xl hover:border-purple-500 dark:hover:border-purple-500/50 hover:shadow-md transition-all flex flex-col justify-between"
          >
            <div className="w-10 h-10 rounded-xl bg-purple-50 dark:bg-purple-500/10 text-purple-600 dark:text-purple-400 flex items-center justify-center mb-3 group-hover:scale-110 transition-transform">
              <Mail className="w-5 h-5" />
            </div>
            <div>
              <h3 className="font-bold text-sm text-slate-900 dark:text-white group-hover:text-purple-600 dark:group-hover:text-purple-400 transition-colors">
                Mailboxes & Forwarders
              </h3>
              <p className="text-xs text-slate-500 dark:text-slate-400 mt-1">
                Custom domain business email accounts
              </p>
            </div>
          </Link>

          <Link
            href="/webmail"
            className="group p-5 bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-800 rounded-2xl hover:border-emerald-500 dark:hover:border-emerald-500/50 hover:shadow-md transition-all flex flex-col justify-between"
          >
            <div className="w-10 h-10 rounded-xl bg-emerald-50 dark:bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 flex items-center justify-center mb-3 group-hover:scale-110 transition-transform">
              <Mail className="w-5 h-5" />
            </div>
            <div>
              <h3 className="font-bold text-sm text-slate-900 dark:text-white group-hover:text-emerald-600 dark:group-hover:text-emerald-400 transition-colors">
                Webmail Client
              </h3>
              <p className="text-xs text-slate-500 dark:text-slate-400 mt-1">
                Access Roundcube / webmail web client
              </p>
            </div>
          </Link>

          <Link
            href="/ssl"
            className="group p-5 bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-800 rounded-2xl hover:border-cyan-500 dark:hover:border-cyan-500/50 hover:shadow-md transition-all flex flex-col justify-between"
          >
            <div className="w-10 h-10 rounded-xl bg-cyan-50 dark:bg-cyan-500/10 text-cyan-600 dark:text-cyan-400 flex items-center justify-center mb-3 group-hover:scale-110 transition-transform">
              <ShieldCheck className="w-5 h-5" />
            </div>
            <div>
              <h3 className="font-bold text-sm text-slate-900 dark:text-white group-hover:text-cyan-600 dark:group-hover:text-cyan-400 transition-colors">
                SSL Certificates
              </h3>
              <p className="text-xs text-slate-500 dark:text-slate-400 mt-1">
                Auto-renewable Let&apos;s Encrypt & Custom SSL
              </p>
            </div>
          </Link>

          {/* Terminal (Conditionally Permitted) */}
          <Link
            href={hasTerminal ? '/terminal' : '#'}
            onClick={(e) => {
              if (!hasTerminal) {
                e.preventDefault();
                alert('SSH / Web Terminal access is disabled on your current plan. Please upgrade to Pro or request an admin override.');
              }
            }}
            className={`group p-5 bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-800 rounded-2xl transition-all flex flex-col justify-between ${
              hasTerminal
                ? 'hover:border-slate-900 dark:hover:border-slate-400 hover:shadow-md'
                : 'opacity-60 cursor-not-allowed'
            }`}
          >
            <div className="flex items-center justify-between mb-3">
              <div className="w-10 h-10 rounded-xl bg-slate-100 dark:bg-slate-800 text-slate-800 dark:text-slate-200 flex items-center justify-center group-hover:scale-110 transition-transform">
                <Terminal className="w-5 h-5" />
              </div>
              {!hasTerminal && (
                <span className="text-[10px] font-bold px-2 py-0.5 rounded bg-slate-100 dark:bg-surface-800 text-slate-500 flex items-center gap-1">
                  <Lock className="w-3 h-3" /> Plan Upgrade
                </span>
              )}
            </div>
            <div>
              <h3 className="font-bold text-sm text-slate-900 dark:text-white">
                Web Terminal (SSH)
              </h3>
              <p className="text-xs text-slate-500 dark:text-slate-400 mt-1">
                {hasTerminal ? 'Direct shell access to your environment' : 'Disabled on current tier'}
              </p>
            </div>
          </Link>
        </div>
      </div>

      {/* My Websites List */}
      <div className="bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-800 rounded-2xl overflow-hidden shadow-sm">
        <div className="p-5 border-b border-slate-200 dark:border-surface-800 flex items-center justify-between">
          <div>
            <h3 className="font-bold text-base text-slate-900 dark:text-white">
              My Active Websites ({websites.length})
            </h3>
            <p className="text-xs text-slate-500 dark:text-slate-400">
              Websites currently provisioned and routed on your account
            </p>
          </div>
          <Link
            href="/websites"
            className="text-xs font-bold text-emerald-600 hover:text-emerald-700 dark:text-emerald-400 inline-flex items-center gap-1"
          >
            Manage All <ArrowRight className="w-3.5 h-3.5" />
          </Link>
        </div>

        {websites.length === 0 ? (
          <div className="p-12 text-center space-y-4">
            <div className="w-12 h-12 rounded-2xl bg-emerald-50 dark:bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 flex items-center justify-center mx-auto">
              <Globe className="w-6 h-6" />
            </div>
            <div className="space-y-1">
              <h4 className="font-bold text-slate-900 dark:text-white">No websites deployed yet</h4>
              <p className="text-xs text-slate-500 dark:text-slate-400 max-w-sm mx-auto">
                Create your first website or link your domain to start publishing content and web applications.
              </p>
            </div>
            <Link
              href="/websites"
              className="inline-flex items-center gap-2 px-4 py-2 rounded-xl text-xs font-bold bg-[#16A34A] text-white hover:bg-[#15803D] transition-colors"
            >
              <Plus className="w-4 h-4" /> Add First Website
            </Link>
          </div>
        ) : (
          <div className="divide-y divide-slate-100 dark:divide-surface-800">
            {websites.slice(0, 5).map((w) => (
              <div key={w.id} className="p-4 sm:px-6 flex items-center justify-between hover:bg-slate-50/50 dark:hover:bg-surface-800/40 transition-colors">
                <div className="flex items-center gap-3">
                  <div className="w-9 h-9 rounded-xl bg-slate-100 dark:bg-surface-800 flex items-center justify-center text-slate-600 dark:text-slate-300">
                    <Globe className="w-4 h-4" />
                  </div>
                  <div>
                    <div className="flex items-center gap-2">
                      <span className="font-bold text-sm text-slate-900 dark:text-white">
                        {w.primary_domain}
                      </span>
                      <span className={`text-[10px] font-bold px-2 py-0.5 rounded-full ${
                        w.status === 'active'
                          ? 'bg-emerald-100 dark:bg-emerald-500/15 text-emerald-700 dark:text-emerald-300'
                          : 'bg-amber-100 dark:bg-amber-500/15 text-amber-700 dark:text-amber-300'
                      }`}>
                        {w.status.toUpperCase()}
                      </span>
                    </div>
                    <span className="text-xs text-slate-400 font-mono">
                      {w.document_root}
                    </span>
                  </div>
                </div>

                <div className="flex items-center gap-3">
                  {w.php_version && (
                    <span className="hidden sm:inline-flex text-[11px] font-medium px-2 py-0.5 rounded bg-slate-100 dark:bg-surface-800 text-slate-600 dark:text-slate-300">
                      PHP {w.php_version}
                    </span>
                  )}
                  {w.ssl_enabled ? (
                    <span className="text-[11px] text-emerald-600 dark:text-emerald-400 flex items-center gap-1 font-medium">
                      <ShieldCheck className="w-3.5 h-3.5" /> SSL Active
                    </span>
                  ) : (
                    <span className="text-[11px] text-slate-400 flex items-center gap-1">
                      No SSL
                    </span>
                  )}
                  <a
                    href={`http://${w.primary_domain}`}
                    target="_blank"
                    rel="noreferrer"
                    className="p-1.5 rounded-lg text-slate-400 hover:text-slate-900 dark:hover:text-white hover:bg-slate-100 dark:hover:bg-surface-800 transition-colors"
                  >
                    <ExternalLink className="w-4 h-4" />
                  </a>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
