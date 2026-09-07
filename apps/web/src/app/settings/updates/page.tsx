'use client';

import React, { useState, useEffect, useRef } from 'react';
import Link from 'next/link';
import {
  RefreshCw,
  CheckCircle2,
  AlertTriangle,
  XCircle,
  ArrowUpCircle,
  Clock,
  ShieldCheck,
  RotateCcw,
  Sliders,
  FileText,
  Layers,
  ChevronRight,
  ExternalLink,
  Info,
  Calendar,
  Zap,
  HardDrive,
  Terminal,
} from 'lucide-react';
import { DashboardShell } from '@/components/DashboardShell';
import {
  apiFetch,
  SystemVersionInfo,
  ReleaseMetadata,
  CompatibilityReport,
  UpdateJob,
  UpdateStep,
} from '@/lib/api';

interface StatusResponse {
  system: SystemVersionInfo;
  latest_release: ReleaseMetadata;
  compatibility: CompatibilityReport;
  active_job?: UpdateJob;
}

export default function UpdatesPage() {
  const [status, setStatus] = useState<StatusResponse | null>(null);
  const [jobs, setJobs] = useState<UpdateJob[]>([]);
  const [loading, setLoading] = useState(true);
  const [actionLoading, setActionLoading] = useState(false);
  const [selectedJob, setSelectedJob] = useState<UpdateJob | null>(null);
  const [activeJob, setActiveJob] = useState<UpdateJob | null>(null);

  // Modals & Panels
  const [confirmUpdateModal, setConfirmUpdateModal] = useState(false);
  const [rollbackModal, setRollbackModal] = useState(false);
  const [scheduleModal, setScheduleModal] = useState(false);

  // Form states
  const [cronExpression, setCronExpression] = useState('0 3 * * 0'); // Sundays at 3:00 AM
  const [autoBackup, setAutoBackup] = useState(true);
  const [autoRollback, setAutoRollback] = useState(true);
  const [channelSelection, setChannelSelection] = useState<'stable' | 'beta' | 'nightly'>('stable');
  const [notification, setNotification] = useState<{ type: 'success' | 'error'; message: string } | null>(null);

  const pollIntervalRef = useRef<NodeJS.Timeout | null>(null);

  const showNotification = (type: 'success' | 'error', message: string) => {
    setNotification({ type, message });
    setTimeout(() => setNotification(null), 5000);
  };

  const fetchStatus = async () => {
    const res = await apiFetch<StatusResponse>('/api/v1/system/updates/status');
    if (res.success && res.data) {
      setStatus(res.data);
      if (res.data.active_job) {
        setActiveJob(res.data.active_job);
      } else {
        setActiveJob(null);
      }
      if (res.data.system?.channel) {
        setChannelSelection(res.data.system.channel);
      }
    }
  };

  const fetchJobs = async () => {
    const res = await apiFetch<UpdateJob[]>('/api/v1/system/updates/jobs');
    if (res.success && res.data) {
      setJobs(res.data);
    }
  };

  const loadData = async () => {
    setLoading(true);
    await Promise.all([fetchStatus(), fetchJobs()]);
    setLoading(false);
  };

  useEffect(() => {
    loadData();
  }, []);

  // Poll when an update job is in progress
  useEffect(() => {
    if (activeJob && !['completed', 'failed', 'rolled_back'].includes(activeJob.status)) {
      pollIntervalRef.current = setInterval(async () => {
        const res = await apiFetch<UpdateJob>(`/api/v1/system/updates/jobs/${activeJob.id}`);
        if (res.success && res.data) {
          setActiveJob(res.data);
          if (['completed', 'failed', 'rolled_back'].includes(res.data.status)) {
            fetchStatus();
            fetchJobs();
          }
        }
      }, 1500);
    } else {
      if (pollIntervalRef.current) {
        clearInterval(pollIntervalRef.current);
        pollIntervalRef.current = null;
      }
    }

    return () => {
      if (pollIntervalRef.current) {
        clearInterval(pollIntervalRef.current);
      }
    };
  }, [activeJob?.status, activeJob?.id]);

  const handleSwitchChannel = async (newChannel: 'stable' | 'beta' | 'nightly') => {
    setActionLoading(true);
    const res = await apiFetch<{ channel: string; message: string }>('/api/v1/system/updates/channel', {
      method: 'POST',
      body: JSON.stringify({ channel: newChannel }),
    });
    setActionLoading(false);
    if (res.success) {
      setChannelSelection(newChannel);
      showNotification('success', res.data?.message || `Switched to ${newChannel} channel`);
      fetchStatus();
    } else {
      showNotification('error', res.error?.message || 'Failed to update channel');
    }
  };

  const handleStartUpdate = async () => {
    if (!status?.latest_release) return;
    setConfirmUpdateModal(false);
    setActionLoading(true);

    const res = await apiFetch<UpdateJob>('/api/v1/system/updates/start', {
      method: 'POST',
      body: JSON.stringify({
        target_version: status.latest_release.version,
        channel: channelSelection,
      }),
    });

    setActionLoading(false);
    if (res.success && res.data) {
      setActiveJob(res.data);
      showNotification('success', `Update to v${status.latest_release.version} initiated!`);
      fetchJobs();
    } else {
      showNotification('error', res.error?.message || 'Failed to initiate update');
    }
  };

  const handleRollback = async () => {
    setRollbackModal(false);
    setActionLoading(true);

    const res = await apiFetch<{ message: string }>('/api/v1/system/updates/rollback', {
      method: 'POST',
    });

    setActionLoading(false);
    if (res.success) {
      showNotification('success', res.data?.message || 'Rollback initiated successfully.');
      fetchStatus();
      fetchJobs();
    } else {
      showNotification('error', res.error?.message || 'Failed to execute rollback');
    }
  };

  const handleSaveSchedule = async (e: React.FormEvent) => {
    e.preventDefault();
    setActionLoading(true);

    const res = await apiFetch<{ message: string }>('/api/v1/system/updates/schedule', {
      method: 'POST',
      body: JSON.stringify({
        cron_expression: cronExpression,
        channel: channelSelection,
        auto_backup: autoBackup,
        auto_rollback: autoRollback,
      }),
    });

    setActionLoading(false);
    setScheduleModal(false);
    if (res.success) {
      showNotification('success', res.data?.message || 'Automated update window saved');
    } else {
      showNotification('error', res.error?.message || 'Failed to save maintenance schedule');
    }
  };

  const viewJobDetails = async (job: UpdateJob) => {
    const res = await apiFetch<UpdateJob>(`/api/v1/system/updates/jobs/${job.id}`);
    if (res.success && res.data) {
      setSelectedJob(res.data);
    } else {
      setSelectedJob(job);
    }
  };

  return (
    <DashboardShell>
      <div className="space-y-8 max-w-6xl">
        {/* Header Bar */}
        <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
          <div>
            <div className="flex items-center gap-2 text-xs text-slate-400 mb-1">
              <Link href="/settings" className="hover:text-slate-200">Settings</Link>
              <span>/</span>
              <span className="text-slate-200 font-medium">System Updates</span>
            </div>
            <h1 className="text-2xl font-bold tracking-tight text-white flex items-center gap-3">
              Live Update & Upgrade Engine
              <span className="px-2.5 py-0.5 rounded-full text-xs font-semibold bg-emerald-500/10 text-emerald-400 border border-emerald-500/20">
                Zero-Downtime
              </span>
            </h1>
            <p className="text-sm text-slate-400 mt-1">
              Automated release verification, atomic symlink deployment, pre-update snapshots, and instant recovery.
            </p>
          </div>

          <div className="flex items-center gap-2.5">
            <button
              onClick={() => setScheduleModal(true)}
              className="px-3 py-2 rounded-xl bg-surface-900 border border-surface-700 text-slate-300 hover:text-white hover:bg-surface-800 text-xs font-semibold flex items-center gap-2 transition"
            >
              <Calendar className="w-4 h-4 text-indigo-400" />
              Maintenance Schedule
            </button>
            <button
              onClick={() => setRollbackModal(true)}
              className="px-3 py-2 rounded-xl bg-rose-500/10 border border-rose-500/20 text-rose-300 hover:bg-rose-500/20 text-xs font-semibold flex items-center gap-2 transition"
            >
              <RotateCcw className="w-4 h-4" />
              Rollback
            </button>
            <button
              onClick={loadData}
              disabled={loading || actionLoading}
              className="px-3.5 py-2 rounded-xl bg-indigo-600 hover:bg-indigo-500 text-white text-xs font-semibold flex items-center gap-2 transition disabled:opacity-50"
            >
              <RefreshCw className={`w-4 h-4 ${loading ? 'animate-spin' : ''}`} />
              Check Updates
            </button>
          </div>
        </div>

        {/* Global Notification Banner */}
        {notification && (
          <div
            className={`p-4 rounded-xl border flex items-center justify-between text-sm ${
              notification.type === 'success'
                ? 'bg-emerald-500/10 border-emerald-500/20 text-emerald-300'
                : 'bg-rose-500/10 border-rose-500/20 text-rose-300'
            }`}
          >
            <div className="flex items-center gap-2.5">
              {notification.type === 'success' ? (
                <CheckCircle2 className="w-5 h-5 flex-shrink-0" />
              ) : (
                <AlertTriangle className="w-5 h-5 flex-shrink-0" />
              )}
              <span>{notification.message}</span>
            </div>
            <button
              onClick={() => setNotification(null)}
              className="text-xs opacity-70 hover:opacity-100 font-bold ml-4"
            >
              ✕
            </button>
          </div>
        )}

        {/* Live Active Job Progress Banner */}
        {activeJob && (
          <div className={`bg-gradient-to-r ${
            activeJob.status === 'completed'
              ? 'from-emerald-950/60 via-surface-900 to-surface-900 border-emerald-500/30'
              : activeJob.status === 'failed' || activeJob.status === 'rolled_back'
              ? 'from-rose-950/60 via-surface-900 to-surface-900 border-rose-500/30'
              : 'from-indigo-950/80 via-surface-900 to-surface-900 border-indigo-500/30'
          } border rounded-2xl p-6 shadow-2xl space-y-4`}>
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-3">
                <div className={`w-10 h-10 rounded-xl flex items-center justify-center border ${
                  activeJob.status === 'completed'
                    ? 'bg-emerald-500/20 border-emerald-500/40 text-emerald-400'
                    : activeJob.status === 'failed' || activeJob.status === 'rolled_back'
                    ? 'bg-rose-500/20 border-rose-500/40 text-rose-400'
                    : 'bg-indigo-500/20 border-indigo-500/40 text-indigo-400'
                }`}>
                  {activeJob.status === 'completed' ? (
                    <CheckCircle2 className="w-5 h-5 text-emerald-400" />
                  ) : activeJob.status === 'failed' || activeJob.status === 'rolled_back' ? (
                    <AlertTriangle className="w-5 h-5 text-rose-400" />
                  ) : (
                    <RefreshCw className="w-5 h-5 animate-spin text-indigo-400" />
                  )}
                </div>
                <div>
                  <div className="flex items-center gap-2">
                    <span className="text-sm font-bold text-white">
                      {activeJob.status === 'completed'
                        ? `System Successfully Upgraded to v${activeJob.target_version}`
                        : `Updating System to v${activeJob.target_version}`}
                    </span>
                    <span className={`px-2 py-0.5 rounded text-[11px] font-mono uppercase border ${
                      activeJob.status === 'completed'
                        ? 'bg-emerald-500/20 text-emerald-300 border-emerald-500/30'
                        : activeJob.status === 'failed' || activeJob.status === 'rolled_back'
                        ? 'bg-rose-500/20 text-rose-300 border-rose-500/30'
                        : 'bg-indigo-500/20 text-indigo-300 border-indigo-500/30'
                    }`}>
                      {activeJob.status.replace('_', ' ')}
                    </span>
                  </div>
                  <p className="text-xs text-slate-400 mt-0.5">
                    {activeJob.status === 'completed'
                      ? 'Live deployment and health verification completed with zero customer downtime.'
                      : activeJob.current_step_description || 'Orchestrating live deployment pipeline...'}
                  </p>
                </div>
              </div>
              <div className="flex items-center gap-3">
                <span className={`text-xl font-bold font-mono ${
                  activeJob.status === 'completed' ? 'text-emerald-400' : 'text-indigo-400'
                }`}>
                  {activeJob.status === 'completed' ? 100 : (activeJob.progress_percent || 100)}%
                </span>
                {['completed', 'failed', 'rolled_back'].includes(activeJob.status) && (
                  <button
                    onClick={() => setActiveJob(null)}
                    className="px-3 py-1.5 bg-surface-800 hover:bg-surface-700 text-slate-200 rounded-lg border border-surface-700 text-xs font-semibold transition"
                  >
                    Dismiss
                  </button>
                )}
              </div>
            </div>

            {/* Progress Bar */}
            <div className="w-full bg-surface-950 rounded-full h-2.5 overflow-hidden border border-surface-700">
              <div
                className={`h-full transition-all duration-500 rounded-full ${
                  activeJob.status === 'completed'
                    ? 'bg-gradient-to-r from-emerald-500 to-teal-400'
                    : 'bg-gradient-to-r from-indigo-500 to-purple-500'
                }`}
                style={{ width: `${activeJob.status === 'completed' ? 100 : Math.max(activeJob.progress_percent || 100, 5)}%` }}
              />
            </div>

            {/* Step Indicators */}
            <div className="grid grid-cols-2 sm:grid-cols-4 md:grid-cols-7 gap-2 pt-2 text-[11px] font-mono">
              {[
                { key: 'prechecking', label: 'Verify' },
                { key: 'backing_up', label: 'Snapshot' },
                { key: 'downloading', label: 'Download' },
                { key: 'migrating', label: 'Database' },
                { key: 'installing', label: 'Deploy' },
                { key: 'activating', label: 'Symlink' },
                { key: 'health_checking', label: 'Health' },
              ].map((step) => {
                const isPassed = activeJob.status === 'completed' || (activeJob.progress_percent || 0) >= 100;
                const isCurrent = activeJob.status.includes(step.key);
                return (
                  <div
                    key={step.key}
                    className={`px-2.5 py-1.5 rounded-lg border text-center transition ${
                      isCurrent
                        ? 'bg-indigo-500/20 border-indigo-500/40 text-indigo-300 font-semibold'
                        : isPassed
                        ? 'bg-emerald-500/10 border-emerald-500/20 text-emerald-400'
                        : 'bg-surface-950/60 border-surface-800 text-slate-400'
                    }`}
                  >
                    {step.label}
                  </div>
                );
              })}
            </div>
          </div>
        )}

        {/* Matrix & Release Cards Grid */}
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
          {/* Card 1: Current System State */}
          <div className="bg-surface-900 border border-surface-800 rounded-2xl p-6 shadow-xl space-y-5">
            <div className="flex items-center justify-between border-b border-surface-800 pb-4">
              <div className="flex items-center gap-3">
                <div className="w-9 h-9 rounded-xl bg-indigo-500/10 border border-indigo-500/20 flex items-center justify-center text-indigo-400">
                  <ShieldCheck className="w-5 h-5" />
                </div>
                <div>
                  <h2 className="text-sm font-bold text-white">Current Installation</h2>
                  <p className="text-[11px] text-slate-400">Hostvra Control Plane</p>
                </div>
              </div>
              <span className="px-2 py-0.5 rounded text-[11px] font-mono bg-emerald-500/10 text-emerald-400 border border-emerald-500/20">
                Healthy
              </span>
            </div>

            <div className="space-y-3 text-xs">
              <div className="flex justify-between items-center py-1 border-b border-surface-800/60">
                <span className="text-slate-400">API Gateway Version</span>
                <span className="font-mono text-slate-200 font-semibold">
                  v{status?.system?.api_version || '1.0.0'}
                </span>
              </div>
              <div className="flex justify-between items-center py-1 border-b border-surface-800/60">
                <span className="text-slate-400">Hostvra Linux Agent</span>
                <span className="font-mono text-slate-200 font-semibold">
                  v{status?.system?.agent_version || '1.0.0'}
                </span>
              </div>
              <div className="flex justify-between items-center py-1 border-b border-surface-800/60">
                <span className="text-slate-400">Database Schema</span>
                <span className="font-mono text-slate-200 font-semibold">
                  Revision #{status?.system?.db_schema_version || 5}
                </span>
              </div>
              <div className="flex justify-between items-center py-1 border-b border-surface-800/60">
                <span className="text-slate-400">OS Architecture</span>
                <span className="font-mono text-slate-300">
                  {status?.system?.os_arch || 'linux/amd64'}
                </span>
              </div>
              <div className="flex justify-between items-center py-1">
                <span className="text-slate-400">Runtime Engine</span>
                <span className="font-mono text-slate-300">
                  {status?.system?.go_version || 'go1.27.1'}
                </span>
              </div>
            </div>

            {/* Release Channel Selector */}
            <div className="pt-2 border-t border-surface-800">
              <label className="block text-[11px] font-semibold text-slate-400 uppercase tracking-wider mb-2">
                Release Channel
              </label>
              <div className="grid grid-cols-3 gap-2">
                {(['stable', 'beta', 'nightly'] as const).map((ch) => (
                  <button
                    key={ch}
                    onClick={() => handleSwitchChannel(ch)}
                    className={`py-1.5 px-2 rounded-xl text-xs font-semibold capitalize border transition ${
                      channelSelection === ch
                        ? 'bg-indigo-600/20 border-indigo-500 text-indigo-300'
                        : 'bg-surface-950 border-surface-800 text-slate-400 hover:text-slate-200'
                    }`}
                  >
                    {ch}
                  </button>
                ))}
              </div>
            </div>
          </div>

          {/* Card 2: Latest Release Available */}
          <div className="bg-surface-900 border border-surface-800 rounded-2xl p-6 shadow-xl space-y-5 lg:col-span-2">
            <div className="flex items-center justify-between border-b border-surface-800 pb-4">
              <div className="flex items-center gap-3">
                <div className="w-9 h-9 rounded-xl bg-purple-500/10 border border-purple-500/20 flex items-center justify-center text-purple-400">
                  <ArrowUpCircle className="w-5 h-5" />
                </div>
                <div>
                  <h2 className="text-sm font-bold text-white">Available Upgrade</h2>
                  <p className="text-[11px] text-slate-400">Cryptographically Signed Release</p>
                </div>
              </div>

              {status?.system?.update_available ? (
                <span className="px-2.5 py-1 rounded-full text-xs font-bold bg-indigo-500/20 text-indigo-300 border border-indigo-500/30 flex items-center gap-1.5">
                  <Zap className="w-3.5 h-3.5 text-indigo-400" />
                  Upgrade Ready
                </span>
              ) : (
                <span className="px-2.5 py-1 rounded-full text-xs font-semibold bg-slate-800 text-slate-400 border border-slate-700">
                  System Up to Date
                </span>
              )}
            </div>

            {status?.latest_release ? (
              <div className="space-y-4">
                <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-2 p-3.5 rounded-xl bg-surface-950 border border-surface-800">
                  <div>
                    <div className="flex items-center gap-2">
                      <span className="text-base font-bold text-white font-mono">
                        v{status.latest_release.version}
                      </span>
                      <span className="text-xs px-2 py-0.5 rounded bg-purple-500/10 text-purple-400 border border-purple-500/20 uppercase font-mono">
                        {status.latest_release.channel}
                      </span>
                    </div>
                    <p className="text-xs text-slate-400 mt-1">
                      Package Size: {(status.latest_release.package_size_bytes / (1024 * 1024)).toFixed(1)} MB
                      {' • '}
                      Minimum Required: v{status.latest_release.min_supported_version}
                    </p>
                  </div>

                  <button
                    onClick={() => setConfirmUpdateModal(true)}
                    disabled={!status.system?.update_available || !!activeJob}
                    className="px-4 py-2.5 rounded-xl bg-gradient-to-r from-indigo-600 to-purple-600 hover:from-indigo-500 hover:to-purple-500 text-white font-semibold text-xs shadow-lg shadow-indigo-600/20 transition disabled:opacity-40 disabled:cursor-not-allowed flex items-center justify-center gap-2"
                  >
                    <ArrowUpCircle className="w-4 h-4" />
                    Install Update Now
                  </button>
                </div>

                {/* Compatibility Check Indicators */}
                <div className="grid grid-cols-1 sm:grid-cols-3 gap-3 text-xs">
                  <div className="p-3 rounded-xl bg-surface-950 border border-surface-800 flex items-center gap-2.5">
                    <CheckCircle2 className="w-4 h-4 text-emerald-400 flex-shrink-0" />
                    <div>
                      <p className="font-semibold text-slate-200">OS Compatibility</p>
                      <p className="text-slate-400 text-[10px]">Ubuntu 20.04+, Debian 11+</p>
                    </div>
                  </div>
                  <div className="p-3 rounded-xl bg-surface-950 border border-surface-800 flex items-center gap-2.5">
                    <CheckCircle2 className="w-4 h-4 text-emerald-400 flex-shrink-0" />
                    <div>
                      <p className="font-semibold text-slate-200">CPU Architecture</p>
                      <p className="text-slate-400 text-[10px]">x86_64 / arm64 Verified</p>
                    </div>
                  </div>
                  <div className="p-3 rounded-xl bg-surface-950 border border-surface-800 flex items-center gap-2.5">
                    <CheckCircle2 className="w-4 h-4 text-emerald-400 flex-shrink-0" />
                    <div>
                      <p className="font-semibold text-slate-200">Ed25519 Signature</p>
                      <p className="text-slate-400 text-[10px]">Hostvra Security Keyring</p>
                    </div>
                  </div>
                </div>

                {/* Release Notes Preview */}
                <div className="p-4 rounded-xl bg-surface-950/80 border border-surface-800 space-y-2">
                  <div className="flex items-center gap-2 text-xs font-bold text-slate-300">
                    <FileText className="w-3.5 h-3.5 text-indigo-400" />
                    Release Notes & Highlights
                  </div>
                  <p className="text-xs text-slate-400 leading-relaxed whitespace-pre-line">
                    {status.latest_release.release_notes}
                  </p>
                </div>
              </div>
            ) : (
              <div className="p-8 text-center text-slate-400 text-xs">
                No update metadata available currently.
              </div>
            )}
          </div>
        </div>

        {/* Update History Table */}
        <div className="bg-surface-900 border border-surface-800 rounded-2xl shadow-xl overflow-hidden">
          <div className="p-6 border-b border-surface-800 flex items-center justify-between">
            <div>
              <h2 className="text-base font-bold text-white">Update & Rollback History</h2>
              <p className="text-xs text-slate-400">
                Audit trail of all previous patch applications, schema migrations, and recovery points
              </p>
            </div>
            <span className="text-xs text-slate-400 font-mono">
              Total Records: {jobs.length}
            </span>
          </div>

          <div className="overflow-x-auto">
            <table className="w-full text-left text-xs">
              <thead className="bg-surface-950/70 border-b border-surface-800 text-slate-400 uppercase tracking-wider font-semibold">
                <tr>
                  <th className="px-6 py-3.5">Version Target</th>
                  <th className="px-6 py-3.5">Channel</th>
                  <th className="px-6 py-3.5">Status</th>
                  <th className="px-6 py-3.5">Progress</th>
                  <th className="px-6 py-3.5">Started At</th>
                  <th className="px-6 py-3.5 text-right">Action</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-surface-800/60">
                {jobs.length === 0 ? (
                  <tr>
                    <td colSpan={6} className="px-6 py-8 text-center text-slate-500">
                      No update executions recorded yet.
                    </td>
                  </tr>
                ) : (
                  jobs.map((job) => (
                    <tr key={job.id} className="hover:bg-surface-800/40 transition">
                      <td className="px-6 py-4 font-mono font-medium text-slate-200">
                        v{job.target_version}
                        <span className="text-[10px] text-slate-500 ml-1.5">
                          (from v{job.previous_version})
                        </span>
                      </td>
                      <td className="px-6 py-4 capitalize text-slate-300">
                        {job.channel}
                      </td>
                      <td className="px-6 py-4">
                        <span
                          className={`inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full text-[11px] font-semibold ${
                            job.status === 'completed'
                              ? 'bg-emerald-500/10 text-emerald-400 border border-emerald-500/20'
                              : job.status === 'rolled_back'
                              ? 'bg-amber-500/10 text-amber-400 border border-amber-500/20'
                              : job.status === 'failed'
                              ? 'bg-rose-500/10 text-rose-400 border border-rose-500/20'
                              : 'bg-indigo-500/10 text-indigo-400 border border-indigo-500/20'
                          }`}
                        >
                          {job.status.replace('_', ' ')}
                        </span>
                      </td>
                      <td className="px-6 py-4 font-mono text-slate-300">
                        {job.progress_percent}%
                      </td>
                      <td className="px-6 py-4 text-slate-400">
                        {new Date(job.started_at).toLocaleString()}
                      </td>
                      <td className="px-6 py-4 text-right">
                        <button
                          onClick={() => viewJobDetails(job)}
                          className="px-2.5 py-1 rounded-lg bg-surface-800 hover:bg-surface-700 text-slate-200 text-xs font-medium transition"
                        >
                          Details
                        </button>
                      </td>
                    </tr>
                  ))
                )}
              </tbody>
            </table>
          </div>
        </div>
      </div>

      {/* MODAL 1: Confirm Live Update */}
      {confirmUpdateModal && status?.latest_release && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/70 backdrop-blur-sm">
          <div className="bg-surface-900 border border-surface-700 rounded-2xl max-w-lg w-full p-6 shadow-2xl space-y-5">
            <div className="flex items-center gap-3 border-b border-surface-800 pb-4">
              <div className="w-10 h-10 rounded-xl bg-indigo-500/10 border border-indigo-500/20 flex items-center justify-center text-indigo-400">
                <ArrowUpCircle className="w-5 h-5" />
              </div>
              <div>
                <h3 className="text-base font-bold text-white">
                  Confirm System Update (v{status.latest_release.version})
                </h3>
                <p className="text-xs text-slate-400">
                  Pre-flight safety inspection & deployment execution
                </p>
              </div>
            </div>

            <div className="space-y-3 text-xs">
              <div className="p-3 rounded-xl bg-surface-950 border border-surface-800 space-y-1.5">
                <div className="flex items-center gap-2 font-semibold text-emerald-400">
                  <CheckCircle2 className="w-4 h-4" />
                  Pre-Update Snapshot Guarantee
                </div>
                <p className="text-slate-400 text-[11px] leading-relaxed">
                  Hostvra will automatically snapshot configuration files, database tables, and binary assets before making changes.
                </p>
              </div>

              <div className="p-3 rounded-xl bg-surface-950 border border-surface-800 space-y-1.5">
                <div className="flex items-center gap-2 font-semibold text-emerald-400">
                  <CheckCircle2 className="w-4 h-4" />
                  Automated Health Check & Safe Rollback
                </div>
                <p className="text-slate-400 text-[11px] leading-relaxed">
                  If post-upgrade HTTP health probes fail, the atomic symlink switches back immediately to the existing release.
                </p>
              </div>

              <div className="p-3 rounded-xl bg-amber-500/10 border border-amber-500/20 text-amber-300 space-y-1">
                <div className="flex items-center gap-2 font-semibold">
                  <AlertTriangle className="w-4 h-4" />
                  Zero / Sub-second Downtime
                </div>
                <p className="text-[11px] text-amber-300/80">
                  Customer websites and DNS are served uninterrupted. Only the control plane service reloads atomically.
                </p>
              </div>
            </div>

            <div className="flex justify-end gap-3 pt-4 border-t border-surface-800">
              <button
                onClick={() => setConfirmUpdateModal(false)}
                className="px-4 py-2 rounded-xl bg-surface-800 hover:bg-surface-700 text-slate-300 text-xs font-semibold transition"
              >
                Cancel
              </button>
              <button
                onClick={handleStartUpdate}
                disabled={actionLoading}
                className="px-4 py-2 rounded-xl bg-indigo-600 hover:bg-indigo-500 text-white text-xs font-semibold shadow-lg shadow-indigo-600/20 transition flex items-center gap-2"
              >
                <Zap className="w-4 h-4" />
                Proceed with Update
              </button>
            </div>
          </div>
        </div>
      )}

      {/* MODAL 2: Emergency Rollback */}
      {rollbackModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/70 backdrop-blur-sm">
          <div className="bg-surface-900 border border-surface-700 rounded-2xl max-w-md w-full p-6 shadow-2xl space-y-5">
            <div className="flex items-center gap-3 border-b border-surface-800 pb-4">
              <div className="w-10 h-10 rounded-xl bg-rose-500/10 border border-rose-500/20 flex items-center justify-center text-rose-400">
                <RotateCcw className="w-5 h-5" />
              </div>
              <div>
                <h3 className="text-base font-bold text-white">Initiate Emergency Rollback</h3>
                <p className="text-xs text-slate-400">Revert to Previous Stable Snapshot</p>
              </div>
            </div>

            <p className="text-xs text-slate-300 leading-relaxed">
              This will switch the atomic symlink back to the prior known-good version and restore configuration files. Running updates will be cancelled immediately.
            </p>

            <div className="flex justify-end gap-3 pt-4 border-t border-surface-800">
              <button
                onClick={() => setRollbackModal(false)}
                className="px-4 py-2 rounded-xl bg-surface-800 hover:bg-surface-700 text-slate-300 text-xs font-semibold transition"
              >
                Cancel
              </button>
              <button
                onClick={handleRollback}
                disabled={actionLoading}
                className="px-4 py-2 rounded-xl bg-rose-600 hover:bg-rose-500 text-white text-xs font-semibold shadow-lg shadow-rose-600/20 transition flex items-center gap-2"
              >
                <RotateCcw className="w-4 h-4" />
                Confirm Rollback
              </button>
            </div>
          </div>
        </div>
      )}

      {/* MODAL 3: Maintenance Schedule */}
      {scheduleModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/70 backdrop-blur-sm">
          <div className="bg-surface-900 border border-surface-700 rounded-2xl max-w-md w-full p-6 shadow-2xl space-y-5">
            <div className="flex items-center gap-3 border-b border-surface-800 pb-4">
              <div className="w-10 h-10 rounded-xl bg-indigo-500/10 border border-indigo-500/20 flex items-center justify-center text-indigo-400">
                <Calendar className="w-5 h-5" />
              </div>
              <div>
                <h3 className="text-base font-bold text-white">Maintenance Window & Schedule</h3>
                <p className="text-xs text-slate-400">Automated unattended patch schedules</p>
              </div>
            </div>

            <form onSubmit={handleSaveSchedule} className="space-y-4 text-xs">
              <div>
                <label className="block font-semibold text-slate-300 mb-1">
                  Cron Expression (UTC)
                </label>
                <input
                  type="text"
                  value={cronExpression}
                  onChange={(e) => setCronExpression(e.target.value)}
                  className="w-full px-3.5 py-2 rounded-xl bg-surface-950 border border-surface-700 text-white font-mono text-xs focus:outline-none focus:border-indigo-500"
                  placeholder="0 3 * * 0"
                  required
                />
                <p className="text-[10px] text-slate-500 mt-1">
                  Default: 0 3 * * 0 (Every Sunday at 03:00 UTC)
                </p>
              </div>

              <div className="space-y-2 pt-2">
                <label className="flex items-center gap-2.5 cursor-pointer">
                  <input
                    type="checkbox"
                    checked={autoBackup}
                    onChange={(e) => setAutoBackup(e.target.checked)}
                    className="rounded border-surface-700 text-indigo-600 focus:ring-0 bg-surface-950"
                  />
                  <span className="text-slate-300 font-medium">
                    Mandatory pre-upgrade recovery backup
                  </span>
                </label>
                <label className="flex items-center gap-2.5 cursor-pointer">
                  <input
                    type="checkbox"
                    checked={autoRollback}
                    onChange={(e) => setAutoRollback(e.target.checked)}
                    className="rounded border-surface-700 text-indigo-600 focus:ring-0 bg-surface-950"
                  />
                  <span className="text-slate-300 font-medium">
                    Auto-rollback if post-update probe fails
                  </span>
                </label>
              </div>

              <div className="flex justify-end gap-3 pt-4 border-t border-surface-800">
                <button
                  type="button"
                  onClick={() => setScheduleModal(false)}
                  className="px-4 py-2 rounded-xl bg-surface-800 hover:bg-surface-700 text-slate-300 font-semibold transition"
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  disabled={actionLoading}
                  className="px-4 py-2 rounded-xl bg-indigo-600 hover:bg-indigo-500 text-white font-semibold transition"
                >
                  Save Schedule
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* DRAWER / MODAL 4: Job Details Log Viewer */}
      {selectedJob && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/70 backdrop-blur-sm">
          <div className="bg-surface-900 border border-surface-700 rounded-2xl max-w-2xl w-full p-6 shadow-2xl space-y-5 max-h-[85vh] flex flex-col">
            <div className="flex items-center justify-between border-b border-surface-800 pb-4">
              <div className="flex items-center gap-3">
                <div className="w-10 h-10 rounded-xl bg-indigo-500/10 border border-indigo-500/20 flex items-center justify-center text-indigo-400">
                  <Terminal className="w-5 h-5" />
                </div>
                <div>
                  <h3 className="text-base font-bold text-white">
                    Update Job v{selectedJob.target_version}
                  </h3>
                  <p className="text-xs font-mono text-slate-400">ID: {selectedJob.id}</p>
                </div>
              </div>
              <button
                onClick={() => setSelectedJob(null)}
                className="text-slate-400 hover:text-white text-base font-bold"
              >
                ✕
              </button>
            </div>

            <div className="flex-1 overflow-y-auto space-y-3 pr-1 text-xs">
              <div className="grid grid-cols-2 gap-3 p-3 rounded-xl bg-surface-950 border border-surface-800">
                <div>
                  <span className="text-slate-500 block text-[10px] uppercase font-semibold">Status</span>
                  <span className="font-semibold text-slate-200 capitalize">
                    {selectedJob.status.replace('_', ' ')}
                  </span>
                </div>
                <div>
                  <span className="text-slate-500 block text-[10px] uppercase font-semibold">Initiated By</span>
                  <span className="font-mono text-slate-200">
                    {selectedJob.initiated_by || 'System Scheduler'}
                  </span>
                </div>
              </div>

              <div className="space-y-2">
                <h4 className="font-semibold text-slate-300">Execution Pipeline Steps</h4>
                {selectedJob.steps && selectedJob.steps.length > 0 ? (
                  <div className="space-y-1.5">
                    {selectedJob.steps.map((s, idx) => (
                      <div
                        key={s.id || idx}
                        className="p-2.5 rounded-lg bg-surface-950 border border-surface-800/80 flex items-center justify-between"
                      >
                        <div className="flex items-center gap-2.5">
                          {s.status === 'completed' ? (
                            <CheckCircle2 className="w-4 h-4 text-emerald-400 flex-shrink-0" />
                          ) : s.status === 'failed' ? (
                            <XCircle className="w-4 h-4 text-rose-400 flex-shrink-0" />
                          ) : (
                            <RefreshCw className="w-4 h-4 text-indigo-400 animate-spin flex-shrink-0" />
                          )}
                          <div>
                            <p className="font-mono font-medium text-slate-200">{s.step_name}</p>
                            {s.error_message && (
                              <p className="text-rose-400 text-[11px] mt-0.5">{s.error_message}</p>
                            )}
                          </div>
                        </div>
                        <span className="font-mono text-[10px] text-slate-500 uppercase">
                          {s.status}
                        </span>
                      </div>
                    ))}
                  </div>
                ) : (
                  <p className="text-slate-500 italic">No granular step logs recorded.</p>
                )}
              </div>
            </div>

            <div className="flex justify-end pt-3 border-t border-surface-800">
              <button
                onClick={() => setSelectedJob(null)}
                className="px-4 py-2 rounded-xl bg-surface-800 hover:bg-surface-700 text-slate-300 text-xs font-semibold transition"
              >
                Close
              </button>
            </div>
          </div>
        </div>
      )}
    </DashboardShell>
  );
}
