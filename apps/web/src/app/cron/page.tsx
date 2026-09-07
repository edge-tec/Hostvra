'use client';

import React, { useState, useEffect, useCallback } from 'react';
import {
  Clock,
  Plus,
  Play,
  Trash2,
  Power,
  RefreshCw,
  Search,
  CheckCircle2,
  AlertCircle,
  X,
  Terminal,
  Copy,
  Check,
  Edit2,
  Server,
  Zap,
} from 'lucide-react';
import { DashboardShell } from '@/components/DashboardShell';
import {
  apiFetch,
  CronDaemonStatus,
  CronJob,
  CronExecutionResult,
} from '@/lib/api';

export default function CronPage() {
  const [daemonStatus, setDaemonStatus] = useState<CronDaemonStatus | null>(null);
  const [jobs, setJobs] = useState<CronJob[]>([]);
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const [search, setSearch] = useState('');

  // Alerts
  const [errorMsg, setErrorMsg] = useState<string | null>(null);
  const [successMsg, setSuccessMsg] = useState<string | null>(null);

  // Add / Edit Modal
  const [modalOpen, setModalOpen] = useState(false);
  const [editingJobId, setEditingJobId] = useState<string | null>(null);
  const [schedule, setSchedule] = useState('0 2 * * *');
  const [command, setCommand] = useState('');
  const [description, setDescription] = useState('');
  const [systemUser, setSystemUser] = useState('root');
  const [submittingJob, setSubmittingJob] = useState(false);

  // Execution Result Modal
  const [executionResult, setExecutionResult] = useState<CronExecutionResult | null>(null);
  const [executingJobId, setExecutingJobId] = useState<string | null>(null);
  const [actionLoadingId, setActionLoadingId] = useState<string | null>(null);
  const [copiedCmd, setCopiedCmd] = useState<string | null>(null);

  const showNotification = (msg: string, isError = false) => {
    if (isError) {
      setErrorMsg(msg);
      setTimeout(() => setErrorMsg(null), 7000);
    } else {
      setSuccessMsg(msg);
      setTimeout(() => setSuccessMsg(null), 5000);
    }
  };

  const loadData = useCallback(async (showIndicator = true) => {
    if (showIndicator) setRefreshing(true);
    setErrorMsg(null);

    try {
      // 1. Fetch Cron Daemon Status
      const statusRes = await apiFetch<CronDaemonStatus>('/api/v1/cron/status');
      if (statusRes.success && statusRes.data) {
        setDaemonStatus(statusRes.data);
      }

      // 2. Fetch Jobs List
      const jobsRes = await apiFetch<{ jobs: CronJob[]; count: number }>('/api/v1/cron/jobs');
      if (jobsRes.success && jobsRes.data) {
        setJobs(jobsRes.data.jobs || []);
      }
    } catch (err: any) {
      showNotification(err.message || 'Failed to load scheduled cron jobs', true);
    } finally {
      setLoading(false);
      setRefreshing(false);
    }
  }, []);

  useEffect(() => {
    loadData(false);
  }, [loadData]);

  // Open Add Modal
  const handleOpenAdd = () => {
    setEditingJobId(null);
    setSchedule('0 2 * * *');
    setCommand('');
    setDescription('');
    setSystemUser('root');
    setModalOpen(true);
  };

  // Open Edit Modal
  const handleOpenEdit = (job: CronJob) => {
    setEditingJobId(job.id);
    setSchedule(job.schedule);
    setCommand(job.command);
    setDescription(job.description);
    setSystemUser(job.system_user);
    setModalOpen(true);
  };

  // Submit Add or Edit
  const handleSubmitJob = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!command.trim() || !schedule.trim()) {
      showNotification('Schedule and command are required', true);
      return;
    }

    setSubmittingJob(true);
    let res;

    if (editingJobId) {
      // Update
      res = await apiFetch<any>(`/api/v1/cron/jobs/${editingJobId}`, {
        method: 'PUT',
        body: JSON.stringify({
          schedule: schedule.trim(),
          command: command.trim(),
          system_user: systemUser.trim(),
          description: description.trim(),
          is_enabled: true,
        }),
      });
    } else {
      // Create
      res = await apiFetch<any>('/api/v1/cron/jobs', {
        method: 'POST',
        body: JSON.stringify({
          schedule: schedule.trim(),
          command: command.trim(),
          system_user: systemUser.trim(),
          description: description.trim(),
        }),
      });
    }

    setSubmittingJob(false);

    if (res.success) {
      showNotification(editingJobId ? 'Cron job updated successfully' : 'Cron job scheduled successfully');
      setModalOpen(false);
      loadData(false);
    } else {
      showNotification(res.error?.message || 'Failed to save cron job', true);
    }
  };

  // Toggle Job (Enable/Disable)
  const handleToggleJob = async (id: string) => {
    setActionLoadingId(`toggle-${id}`);
    const res = await apiFetch<CronJob>(`/api/v1/cron/jobs/${id}/toggle`, {
      method: 'POST',
    });
    setActionLoadingId(null);

    if (res.success && res.data) {
      setJobs((prev) =>
        prev.map((j) => (j.id === id ? { ...j, is_enabled: res.data!.is_enabled } : j))
      );
      showNotification(res.data.is_enabled ? 'Cron job enabled' : 'Cron job paused');
    } else {
      showNotification(res.error?.message || 'Failed to toggle job state', true);
    }
  };

  // Delete Job
  const handleDeleteJob = async (job: CronJob) => {
    if (!window.confirm(`Are you sure you want to delete cron task: "${job.command}"?`)) {
      return;
    }

    setActionLoadingId(`delete-${job.id}`);
    const res = await apiFetch<any>(`/api/v1/cron/jobs/${job.id}`, {
      method: 'DELETE',
    });
    setActionLoadingId(null);

    if (res.success) {
      showNotification('Cron task deleted successfully');
      loadData(false);
    } else {
      showNotification(res.error?.message || 'Failed to delete cron task', true);
    }
  };

  // Run Job Immediately
  const handleRunNow = async (id: string) => {
    setExecutingJobId(id);
    const res = await apiFetch<CronExecutionResult>(`/api/v1/cron/jobs/${id}/run`, {
      method: 'POST',
    });
    setExecutingJobId(null);

    if (res.success && res.data) {
      setExecutionResult(res.data);
      loadData(false);
    } else {
      showNotification(res.error?.message || 'Failed to execute job', true);
    }
  };

  const copyToClipboard = (text: string, id: string) => {
    navigator.clipboard.writeText(text);
    setCopiedCmd(id);
    setTimeout(() => setCopiedCmd(null), 2000);
  };

  const filteredJobs = jobs.filter(
    (job) =>
      job.command.toLowerCase().includes(search.toLowerCase()) ||
      job.description.toLowerCase().includes(search.toLowerCase()) ||
      job.schedule.includes(search) ||
      job.system_user.toLowerCase().includes(search.toLowerCase())
  );

  return (
    <DashboardShell>
      <div className="space-y-6 animate-fadeIn max-w-7xl mx-auto pb-16">
        {/* Alerts */}
        {errorMsg && (
          <div className="p-4 rounded-xl bg-rose-50 dark:bg-rose-950/40 border border-rose-200 dark:border-rose-900/50 flex items-start gap-3 text-rose-800 dark:text-rose-200 text-sm shadow-xs">
            <AlertCircle className="w-5 h-5 flex-shrink-0 text-rose-600 dark:text-rose-400 mt-0.5" />
            <div className="flex-1 font-medium">{errorMsg}</div>
            <button onClick={() => setErrorMsg(null)} className="text-rose-600 dark:text-rose-400 hover:opacity-75">
              <X className="w-4 h-4" />
            </button>
          </div>
        )}

        {successMsg && (
          <div className="p-4 rounded-xl bg-emerald-50 dark:bg-emerald-950/40 border border-emerald-200 dark:border-emerald-900/50 flex items-start gap-3 text-emerald-800 dark:text-emerald-200 text-sm shadow-xs">
            <CheckCircle2 className="w-5 h-5 flex-shrink-0 text-emerald-600 dark:text-emerald-400 mt-0.5" />
            <div className="flex-1 font-medium">{successMsg}</div>
            <button onClick={() => setSuccessMsg(null)} className="text-emerald-600 dark:text-emerald-400 hover:opacity-75">
              <X className="w-4 h-4" />
            </button>
          </div>
        )}

        {/* Header */}
        <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
          <div>
            <div className="flex items-center gap-3">
              <h1 className="text-2xl font-bold tracking-tight text-slate-950 dark:text-white">
                Scheduled Cron Jobs
              </h1>
              <span className="px-2.5 py-0.5 rounded-full text-xs font-mono font-semibold uppercase bg-indigo-50 dark:bg-indigo-950/60 text-indigo-700 dark:text-indigo-400 border border-indigo-200 dark:border-indigo-800">
                {daemonStatus?.daemon || 'crond'}
              </span>
            </div>
            <p className="text-sm text-slate-600 dark:text-slate-400 mt-1">
              Production background worker execution, automated maintenance, and command scheduling.
            </p>
          </div>

          <div className="flex items-center gap-2">
            <button
              onClick={() => loadData(true)}
              disabled={refreshing}
              className="p-2.5 rounded-xl border border-slate-300 dark:border-surface-700 bg-white dark:bg-surface-900 text-slate-700 dark:text-slate-300 hover:bg-slate-50 dark:hover:bg-surface-800 transition-colors shadow-xs"
              title="Refresh Cron Tasks"
            >
              <RefreshCw className={`w-4 h-4 ${refreshing ? 'animate-spin text-indigo-500' : ''}`} />
            </button>
            <button
              onClick={handleOpenAdd}
              className="inline-flex items-center gap-2 px-4 py-2.5 rounded-xl bg-indigo-600 hover:bg-indigo-500 text-white text-sm font-semibold shadow-md shadow-indigo-600/20 transition-all"
            >
              <Plus className="w-4 h-4" />
              Add Cron Job
            </button>
          </div>
        </div>

        {/* Status Banners */}
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          <div className="p-5 rounded-2xl bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-800 shadow-xs flex items-center gap-3.5">
            <div
              className={`w-11 h-11 rounded-xl flex items-center justify-center ${
                daemonStatus?.is_active
                  ? 'bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 border border-emerald-500/20'
                  : 'bg-amber-500/10 text-amber-600 dark:text-amber-400 border border-amber-500/20'
              }`}
            >
              <Server className="w-6 h-6" />
            </div>
            <div>
              <div className="flex items-center gap-2">
                <span className="font-bold text-slate-950 dark:text-white text-sm">
                  {daemonStatus?.is_active ? 'Daemon Active' : 'Daemon Inactive'}
                </span>
                <span
                  className={`w-2 h-2 rounded-full ${
                    daemonStatus?.is_active ? 'bg-emerald-500 animate-pulse' : 'bg-amber-500'
                  }`}
                />
              </div>
              <p className="text-xs text-slate-500 dark:text-slate-400 mt-0.5 font-medium">
                System cron engine running via /etc/cron.d
              </p>
            </div>
          </div>

          <div className="p-5 rounded-2xl bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-800 shadow-xs flex items-center gap-3.5">
            <div className="w-11 h-11 rounded-xl bg-indigo-500/10 text-indigo-600 dark:text-indigo-400 border border-indigo-500/20 flex items-center justify-center">
              <Clock className="w-6 h-6" />
            </div>
            <div>
              <div className="flex items-center gap-2">
                <span className="font-bold text-slate-950 dark:text-white text-sm">Configured Tasks</span>
                <span className="px-2 py-0.5 rounded-full text-[10px] font-bold bg-indigo-500/10 text-indigo-600 dark:text-indigo-400 border border-indigo-500/20">
                  {jobs.length} Total
                </span>
              </div>
              <p className="text-xs text-slate-500 dark:text-slate-400 mt-0.5 font-medium">
                {jobs.filter((j) => j.is_enabled).length} Enabled • {jobs.filter((j) => !j.is_enabled).length} Paused
              </p>
            </div>
          </div>

          <div className="p-5 rounded-2xl bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-800 shadow-xs flex items-center gap-3.5">
            <div className="w-11 h-11 rounded-xl bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 border border-emerald-500/20 flex items-center justify-center">
              <Zap className="w-6 h-6" />
            </div>
            <div>
              <div className="flex items-center gap-2">
                <span className="font-bold text-slate-950 dark:text-white text-sm">Safety Sanitizer</span>
                <span className="px-2 py-0.5 rounded-full text-[10px] font-bold bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 border border-emerald-500/20">
                  Active
                </span>
              </div>
              <p className="text-xs text-slate-500 dark:text-slate-400 mt-0.5 font-medium">
                Destructive commands (rm -rf /, fork bombs) blocked
              </p>
            </div>
          </div>
        </div>

        {/* Search Bar */}
        <div className="flex flex-col sm:flex-row items-center justify-between gap-4 bg-white dark:bg-[#10141d] border border-slate-200 dark:border-surface-800 rounded-2xl p-3 shadow-xs">
          <div className="flex items-center gap-3 w-full sm:w-80 bg-slate-50 dark:bg-[#121824] border border-slate-300 dark:border-surface-700 rounded-xl px-3.5 py-2 shadow-xs focus-within:border-indigo-500 focus-within:ring-2 focus-within:ring-indigo-500/20 transition-all">
            <Search className="w-4 h-4 text-slate-400 flex-shrink-0" />
            <input
              type="text"
              placeholder="Search cron tasks by command or schedule..."
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              className="w-full bg-transparent text-xs font-medium text-slate-950 dark:text-white placeholder:text-slate-400 focus:outline-none"
            />
            {search && (
              <button onClick={() => setSearch('')} className="text-slate-400 hover:text-slate-600 dark:hover:text-white p-0.5">
                <X className="w-3.5 h-3.5" />
              </button>
            )}
          </div>
          <div className="text-xs font-semibold text-slate-600 dark:text-slate-400 px-2 self-end sm:self-center">
            Showing {filteredJobs.length} of {jobs.length} tasks
          </div>
        </div>

        {/* Cron Table */}
        <div className="bg-white dark:bg-[#10141d] border border-slate-200 dark:border-surface-800 rounded-2xl overflow-hidden shadow-xs">
          <div className="overflow-x-auto">
            <table className="w-full text-left text-sm border-collapse">
              <thead>
                <tr className="border-b border-slate-200 dark:border-surface-800 bg-slate-50 dark:bg-[#121824] text-slate-700 dark:text-slate-300 text-[11px] font-bold uppercase tracking-wider">
                  <th className="px-6 py-3.5">Schedule</th>
                  <th className="px-6 py-3.5">Command & Description</th>
                  <th className="px-6 py-3.5">System User</th>
                  <th className="px-6 py-3.5">Status</th>
                  <th className="px-6 py-3.5">Last Run</th>
                  <th className="px-6 py-3.5 text-right">Actions</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-200 dark:divide-surface-800/80">
                {loading ? (
                  <tr>
                    <td colSpan={6} className="px-6 py-12 text-center text-slate-500 font-medium text-sm">
                      <RefreshCw className="w-6 h-6 animate-spin mx-auto mb-2 text-indigo-500" />
                      Loading system scheduled jobs...
                    </td>
                  </tr>
                ) : filteredJobs.length === 0 ? (
                  <tr>
                    <td colSpan={6} className="px-6 py-12 text-center text-slate-500 font-medium text-sm">
                      No scheduled cron jobs found. Click "Add Cron Job" to create one.
                    </td>
                  </tr>
                ) : (
                  filteredJobs.map((job) => (
                    <tr key={job.id} className="hover:bg-slate-50/80 dark:hover:bg-[#151d2d] transition-colors">
                      <td className="px-6 py-4">
                        <span className="font-mono text-xs px-2.5 py-1 rounded-lg bg-indigo-50 dark:bg-indigo-950/60 text-indigo-700 dark:text-indigo-300 font-bold border border-indigo-200 dark:border-indigo-800/50">
                          {job.schedule}
                        </span>
                      </td>
                      <td className="px-6 py-4">
                        <div className="flex items-center gap-2 max-w-lg">
                          <span className="font-mono text-xs font-semibold text-slate-950 dark:text-white truncate">
                            {job.command}
                          </span>
                          <button
                            onClick={() => copyToClipboard(job.command, job.id)}
                            className="p-1 rounded text-slate-400 hover:text-slate-600 dark:hover:text-slate-200"
                            title="Copy Command"
                          >
                            {copiedCmd === job.id ? (
                              <Check className="w-3.5 h-3.5 text-emerald-500" />
                            ) : (
                              <Copy className="w-3.5 h-3.5" />
                            )}
                          </button>
                        </div>
                        {job.description && (
                          <div className="text-xs text-slate-500 dark:text-slate-400 mt-0.5 font-medium">
                            {job.description}
                          </div>
                        )}
                      </td>
                      <td className="px-6 py-4 font-mono text-xs font-semibold text-slate-700 dark:text-slate-300">
                        {job.system_user}
                      </td>
                      <td className="px-6 py-4">
                        <span
                          className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-[11px] font-bold uppercase ${
                            job.is_enabled
                              ? 'bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 border border-emerald-500/20'
                              : 'bg-slate-100 dark:bg-surface-800 text-slate-500 dark:text-slate-400 border border-slate-300 dark:border-surface-700'
                          }`}
                        >
                          {job.is_enabled ? 'Active' : 'Paused'}
                        </span>
                      </td>
                      <td className="px-6 py-4 text-xs font-mono text-slate-500">
                        {job.last_run_at ? (
                          <div className="flex items-center gap-1.5">
                            <span
                              className={`w-1.5 h-1.5 rounded-full ${
                                job.last_status === 'success' ? 'bg-emerald-500' : 'bg-rose-500'
                              }`}
                            />
                            <span>{new Date(job.last_run_at).toLocaleTimeString()}</span>
                          </div>
                        ) : (
                          '—'
                        )}
                      </td>
                      <td className="px-6 py-4 text-right">
                        <div className="flex items-center justify-end gap-1.5">
                          <button
                            onClick={() => handleRunNow(job.id)}
                            disabled={executingJobId === job.id}
                            title="Execute Immediately"
                            className="p-1.5 rounded-lg border border-slate-300 dark:border-surface-700 text-slate-600 dark:text-slate-400 hover:text-emerald-600 dark:hover:text-emerald-400 hover:bg-emerald-50 dark:hover:bg-emerald-950/30 transition-colors"
                          >
                            {executingJobId === job.id ? (
                              <RefreshCw className="w-3.5 h-3.5 animate-spin text-emerald-500" />
                            ) : (
                              <Play className="w-3.5 h-3.5" />
                            )}
                          </button>
                          <button
                            onClick={() => handleToggleJob(job.id)}
                            disabled={actionLoadingId === `toggle-${job.id}`}
                            title={job.is_enabled ? 'Pause Task' : 'Enable Task'}
                            className="p-1.5 rounded-lg border border-slate-300 dark:border-surface-700 text-slate-600 dark:text-slate-400 hover:text-amber-600 dark:hover:text-amber-400 hover:bg-amber-50 dark:hover:bg-amber-950/30 transition-colors"
                          >
                            <Power className="w-3.5 h-3.5" />
                          </button>
                          <button
                            onClick={() => handleOpenEdit(job)}
                            title="Edit"
                            className="p-1.5 rounded-lg border border-slate-300 dark:border-surface-700 text-slate-600 dark:text-slate-400 hover:text-indigo-600 dark:hover:text-indigo-400 hover:bg-indigo-50 dark:hover:bg-indigo-950/30 transition-colors"
                          >
                            <Edit2 className="w-3.5 h-3.5" />
                          </button>
                          <button
                            onClick={() => handleDeleteJob(job)}
                            disabled={actionLoadingId === `delete-${job.id}`}
                            title="Delete Task"
                            className="p-1.5 rounded-lg border border-slate-300 dark:border-surface-700 text-slate-600 dark:text-slate-400 hover:text-rose-600 dark:hover:text-rose-400 hover:bg-rose-50 dark:hover:bg-rose-950/30 transition-colors"
                          >
                            <Trash2 className="w-3.5 h-3.5" />
                          </button>
                        </div>
                      </td>
                    </tr>
                  ))
                )}
              </tbody>
            </table>
          </div>
        </div>

        {/* Modal: Execution Output Result */}
        {executionResult && (
          <div className="fixed inset-0 z-50 bg-black/70 backdrop-blur-sm flex items-center justify-center p-4">
            <div className="w-full max-w-2xl bg-white dark:bg-[#10141d] border border-slate-200 dark:border-surface-700 rounded-2xl shadow-2xl p-6 relative">
              <button
                onClick={() => setExecutionResult(null)}
                className="absolute top-5 right-5 p-1 rounded-lg text-slate-400 hover:text-slate-700 dark:hover:text-white transition-colors"
              >
                <X className="w-5 h-5" />
              </button>

              <div className="flex items-center gap-3 mb-4">
                <div
                  className={`w-10 h-10 rounded-xl flex items-center justify-center ${
                    executionResult.success
                      ? 'bg-emerald-500/10 text-emerald-600 border border-emerald-500/20'
                      : 'bg-rose-500/10 text-rose-600 border border-rose-500/20'
                  }`}
                >
                  <Terminal className="w-5 h-5" />
                </div>
                <div>
                  <h2 className="text-lg font-bold text-slate-950 dark:text-white">
                    Cron Execution Result
                  </h2>
                  <p className="text-xs text-slate-500 dark:text-slate-400 font-mono">
                    User: {executionResult.system_user} • Duration: {executionResult.duration_ms}ms • Exit Code: {executionResult.exit_code}
                  </p>
                </div>
              </div>

              <div className="space-y-3">
                <div className="p-3 rounded-xl bg-slate-100 dark:bg-surface-950 border border-slate-200 dark:border-surface-800 font-mono text-xs text-slate-900 dark:text-slate-200 break-all">
                  <span className="text-indigo-500 select-none">$ </span>
                  {executionResult.command}
                </div>

                <div>
                  <label className="block text-[11px] font-bold uppercase tracking-wider text-slate-500 dark:text-slate-400 mb-1">
                    Standard Output / Errors:
                  </label>
                  <pre className="p-3.5 rounded-xl bg-slate-950 text-slate-100 font-mono text-xs overflow-x-auto max-h-72 whitespace-pre-wrap">
                    {executionResult.stdout || '(Execution completed with no console output)'}
                  </pre>
                </div>
              </div>

              <div className="flex justify-end pt-4">
                <button
                  onClick={() => setExecutionResult(null)}
                  className="px-5 py-2 rounded-xl bg-indigo-600 hover:bg-indigo-500 text-white text-xs font-semibold shadow-md transition-all"
                >
                  Close Output
                </button>
              </div>
            </div>
          </div>
        )}

        {/* Modal: Add or Edit Cron Job */}
        {modalOpen && (
          <div className="fixed inset-0 z-50 bg-black/70 backdrop-blur-sm flex items-center justify-center p-4">
            <div className="w-full max-w-lg bg-white dark:bg-[#10141d] border border-slate-200 dark:border-surface-700 rounded-2xl shadow-2xl p-6 relative">
              <button
                onClick={() => setModalOpen(false)}
                className="absolute top-5 right-5 p-1 rounded-lg text-slate-400 hover:text-slate-700 dark:hover:text-white transition-colors"
              >
                <X className="w-5 h-5" />
              </button>

              <div className="flex items-center gap-3 mb-5">
                <div className="w-10 h-10 rounded-xl bg-indigo-500/10 border border-indigo-500/20 flex items-center justify-center text-indigo-600 dark:text-indigo-400">
                  <Clock className="w-5 h-5" />
                </div>
                <div>
                  <h2 className="text-lg font-bold text-slate-950 dark:text-white">
                    {editingJobId ? 'Edit Cron Job' : 'Add Scheduled Cron Job'}
                  </h2>
                  <p className="text-xs text-slate-500 dark:text-slate-400">
                    Automate execution via system crontab
                  </p>
                </div>
              </div>

              {/* Schedule Presets */}
              <div className="mb-4">
                <label className="block text-[11px] font-bold uppercase tracking-wider text-slate-500 dark:text-slate-400 mb-1.5">
                  Schedule Presets
                </label>
                <div className="flex flex-wrap gap-1.5">
                  <button
                    type="button"
                    onClick={() => setSchedule('* * * * *')}
                    className="px-2.5 py-1 rounded-lg bg-slate-50 dark:bg-surface-900 hover:bg-slate-100 dark:hover:bg-surface-800 border border-slate-300 dark:border-surface-700 text-xs text-slate-700 dark:text-slate-300 font-medium"
                  >
                    Every Minute
                  </button>
                  <button
                    type="button"
                    onClick={() => setSchedule('*/5 * * * *')}
                    className="px-2.5 py-1 rounded-lg bg-slate-50 dark:bg-surface-900 hover:bg-slate-100 dark:hover:bg-surface-800 border border-slate-300 dark:border-surface-700 text-xs text-slate-700 dark:text-slate-300 font-medium"
                  >
                    Every 5 Mins
                  </button>
                  <button
                    type="button"
                    onClick={() => setSchedule('0 * * * *')}
                    className="px-2.5 py-1 rounded-lg bg-slate-50 dark:bg-surface-900 hover:bg-slate-100 dark:hover:bg-surface-800 border border-slate-300 dark:border-surface-700 text-xs text-slate-700 dark:text-slate-300 font-medium"
                  >
                    Hourly
                  </button>
                  <button
                    type="button"
                    onClick={() => setSchedule('0 2 * * *')}
                    className="px-2.5 py-1 rounded-lg bg-slate-50 dark:bg-surface-900 hover:bg-slate-100 dark:hover:bg-surface-800 border border-slate-300 dark:border-surface-700 text-xs text-slate-700 dark:text-slate-300 font-medium"
                  >
                    Daily at 2 AM
                  </button>
                  <button
                    type="button"
                    onClick={() => setSchedule('0 0 * * 0')}
                    className="px-2.5 py-1 rounded-lg bg-slate-50 dark:bg-surface-900 hover:bg-slate-100 dark:hover:bg-surface-800 border border-slate-300 dark:border-surface-700 text-xs text-slate-700 dark:text-slate-300 font-medium"
                  >
                    Weekly (Sunday)
                  </button>
                </div>
              </div>

              <form onSubmit={handleSubmitJob} className="space-y-4">
                <div>
                  <label className="block text-xs font-semibold uppercase tracking-wider text-slate-700 dark:text-slate-300 mb-1.5">
                    Cron Schedule Expression (5 fields or @macro)
                  </label>
                  <input
                    type="text"
                    required
                    value={schedule}
                    onChange={(e) => setSchedule(e.target.value)}
                    placeholder="e.g. 0 2 * * *"
                    className="w-full px-3.5 py-2.5 rounded-xl bg-slate-50 dark:bg-surface-950 border border-slate-300 dark:border-surface-700 text-slate-950 dark:text-slate-100 font-mono text-sm focus:outline-none focus:border-indigo-500"
                  />
                </div>

                <div>
                  <label className="block text-xs font-semibold uppercase tracking-wider text-slate-700 dark:text-slate-300 mb-1.5">
                    Command to Execute
                  </label>
                  <textarea
                    required
                    rows={2}
                    value={command}
                    onChange={(e) => setCommand(e.target.value)}
                    placeholder="php /var/www/site/artisan schedule:run"
                    className="w-full px-3.5 py-2.5 rounded-xl bg-slate-50 dark:bg-surface-950 border border-slate-300 dark:border-surface-700 text-slate-950 dark:text-slate-100 font-mono text-xs focus:outline-none focus:border-indigo-500 resize-none"
                  />
                </div>

                <div className="grid grid-cols-2 gap-3">
                  <div>
                    <label className="block text-xs font-semibold uppercase tracking-wider text-slate-700 dark:text-slate-300 mb-1.5">
                      System User
                    </label>
                    <select
                      value={systemUser}
                      onChange={(e) => setSystemUser(e.target.value)}
                      className="w-full px-3.5 py-2.5 rounded-xl bg-slate-50 dark:bg-surface-950 border border-slate-300 dark:border-surface-700 text-slate-950 dark:text-slate-100 font-mono text-xs focus:outline-none focus:border-indigo-500"
                    >
                      <option value="root">root</option>
                      <option value="www-data">www-data</option>
                      <option value="hostvra">hostvra</option>
                      <option value="nginx">nginx</option>
                    </select>
                  </div>

                  <div>
                    <label className="block text-xs font-semibold uppercase tracking-wider text-slate-700 dark:text-slate-300 mb-1.5">
                      Description / Note
                    </label>
                    <input
                      type="text"
                      placeholder="e.g. Daily Laravel scheduled tasks"
                      value={description}
                      onChange={(e) => setDescription(e.target.value)}
                      className="w-full px-3.5 py-2.5 rounded-xl bg-slate-50 dark:bg-surface-950 border border-slate-300 dark:border-surface-700 text-slate-950 dark:text-slate-100 text-xs focus:outline-none focus:border-indigo-500"
                    />
                  </div>
                </div>

                <div className="flex justify-end gap-2 pt-3">
                  <button
                    type="button"
                    onClick={() => setModalOpen(false)}
                    className="px-4 py-2.5 rounded-xl text-xs font-medium text-slate-600 dark:text-slate-400 hover:bg-slate-100 dark:hover:bg-surface-800 transition-colors"
                  >
                    Cancel
                  </button>
                  <button
                    type="submit"
                    disabled={submittingJob}
                    className="px-5 py-2.5 rounded-xl bg-indigo-600 hover:bg-indigo-500 text-white text-xs font-semibold shadow-md shadow-indigo-600/20 transition-all flex items-center gap-1.5"
                  >
                    {submittingJob ? <RefreshCw className="w-3.5 h-3.5 animate-spin" /> : null}
                    {editingJobId ? 'Update Cron Job' : 'Save Cron Job'}
                  </button>
                </div>
              </form>
            </div>
          </div>
        )}
      </div>
    </DashboardShell>
  );
}
