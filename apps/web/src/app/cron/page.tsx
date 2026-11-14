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
  FileText,
  Globe,
  Database,
  Archive,
  Cpu,
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

  // aaPanel Parity Task Type & Cycle Builder State
  const [taskType, setTaskType] = useState<'shell' | 'url_access' | 'free_ram' | 'backup_site' | 'backup_database' | 'cut_logs'>('shell');
  const [accessUrl, setAccessUrl] = useState('https://');
  const [targetSite, setTargetSite] = useState('');
  const [targetDb, setTargetDb] = useState('');
  const [backupCopies, setBackupCopies] = useState(3);

  const [cycleType, setCycleType] = useState<'day' | 'day_n' | 'hour' | 'hour_n' | 'minute_n' | 'week' | 'month' | 'custom'>('day');
  const [cycleHour, setCycleHour] = useState(2);
  const [cycleMinute, setCycleMinute] = useState(0);
  const [cycleNDays, setCycleNDays] = useState(3);
  const [cycleNHours, setCycleNHours] = useState(2);
  const [cycleNMinutes, setCycleNMinutes] = useState(15);
  const [cycleWeekday, setCycleWeekday] = useState(0); // 0=Sun, 1=Mon...
  const [cycleDayOfMonth, setCycleDayOfMonth] = useState(1);

  // Execution Result & Log Modal
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

  // Synchronize Cycle with Cron Schedule expression
  useEffect(() => {
    if (cycleType === 'custom') return;
    let s = '0 2 * * *';
    switch (cycleType) {
      case 'day':
        s = `${cycleMinute} ${cycleHour} * * *`;
        break;
      case 'day_n':
        s = `${cycleMinute} ${cycleHour} */${Math.max(1, cycleNDays)} * *`;
        break;
      case 'hour':
        s = `${cycleMinute} * * * *`;
        break;
      case 'hour_n':
        s = `${cycleMinute} */${Math.max(1, cycleNHours)} * * *`;
        break;
      case 'minute_n':
        s = `*/${Math.max(1, cycleNMinutes)} * * * *`;
        break;
      case 'week':
        s = `${cycleMinute} ${cycleHour} * * ${cycleWeekday}`;
        break;
      case 'month':
        s = `${cycleMinute} ${cycleHour} ${cycleDayOfMonth} * *`;
        break;
    }
    setSchedule(s);
  }, [cycleType, cycleHour, cycleMinute, cycleNDays, cycleNHours, cycleNMinutes, cycleWeekday, cycleDayOfMonth]);

  // Synchronize Task Type with command
  useEffect(() => {
    if (editingJobId) return;
    switch (taskType) {
      case 'url_access':
        setCommand(`curl -s -S --connect-timeout 30 -m 60 "${accessUrl || 'https://'}"`);
        break;
      case 'free_ram':
        setCommand('sync && echo 3 > /proc/sys/vm/drop_caches');
        break;
      case 'cut_logs':
        setCommand('/usr/sbin/logrotate /etc/logrotate.d/nginx -f 2>/dev/null || true');
        break;
      case 'backup_site':
        const site = targetSite.trim() || 'site';
        setCommand(`mkdir -p /var/backups/hostvra/sites && tar -czf /var/backups/hostvra/sites/${site}_$(date +%Y%m%d_%H%M%S).tar.gz -C /var/www/${site} . && ls -t /var/backups/hostvra/sites/${site}_*.tar.gz | tail -n +${backupCopies + 1} | xargs -r rm -f`);
        break;
      case 'backup_database':
        const db = targetDb.trim() || 'database';
        setCommand(`mkdir -p /var/backups/hostvra/databases && mysqldump --single-transaction --quick ${db} | gzip > /var/backups/hostvra/databases/${db}_$(date +%Y%m%d_%H%M%S).sql.gz && ls -t /var/backups/hostvra/databases/${db}_*.sql.gz | tail -n +${backupCopies + 1} | xargs -r rm -f`);
        break;
    }
  }, [taskType, accessUrl, targetSite, targetDb, backupCopies, editingJobId]);

  // Open Add Modal
  const handleOpenAdd = () => {
    setEditingJobId(null);
    setTaskType('shell');
    setCycleType('day');
    setCycleHour(2);
    setCycleMinute(0);
    setSchedule('0 2 * * *');
    setCommand('');
    setDescription('');
    setSystemUser('root');
    setModalOpen(true);
  };

  // Open Edit Modal
  const handleOpenEdit = (job: CronJob) => {
    setEditingJobId(job.id);
    setTaskType('shell');
    setCycleType('custom');
    setSchedule(job.schedule);
    setCommand(job.command);
    setDescription(job.description);
    setSystemUser(job.system_user);
    setModalOpen(true);
  };

  // View Job Log
  const handleViewLogs = async (id: string) => {
    setActionLoadingId(`logs-${id}`);
    const res = await apiFetch<CronExecutionResult>(`/api/v1/cron/jobs/${id}/logs`);
    setActionLoadingId(null);

    if (res.success && res.data) {
      setExecutionResult(res.data);
    } else {
      showNotification(res.error?.message || 'Failed to fetch task execution log', true);
    }
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
              className="inline-flex items-center gap-2 px-4 py-2.5 rounded-xl bg-[#16A34A] hover:bg-[#15803D] active:bg-[#166534] text-white text-sm font-semibold shadow-sm transition-all"
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
            <table className="w-full text-left text-sm border-collapse min-w-[700px]">
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
                            className="p-1.5 rounded-lg border border-slate-300 dark:border-surface-700 text-slate-600 dark:text-slate-400 hover:text-emerald-600 dark:hover:text-emerald-400 hover:bg-emerald-50 dark:hover:bg-emerald-950/30 transition-colors cursor-pointer"
                          >
                            {executingJobId === job.id ? (
                              <RefreshCw className="w-3.5 h-3.5 animate-spin text-emerald-500" />
                            ) : (
                              <Play className="w-3.5 h-3.5" />
                            )}
                          </button>
                          <button
                            onClick={() => handleViewLogs(job.id)}
                            disabled={actionLoadingId === `logs-${job.id}`}
                            title="View Execution Log"
                            className="p-1.5 rounded-lg border border-slate-300 dark:border-surface-700 text-slate-600 dark:text-slate-400 hover:text-indigo-600 dark:hover:text-indigo-400 hover:bg-indigo-50 dark:hover:bg-indigo-950/30 transition-colors cursor-pointer"
                          >
                            <FileText className="w-3.5 h-3.5" />
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
                  className="px-5 py-2 rounded-xl bg-[#16A34A] hover:bg-[#15803D] active:bg-[#166534] text-white text-xs font-semibold shadow-sm transition-all"
                >
                  Close Output
                </button>
              </div>
            </div>
          </div>
        )}

        {/* Modal: Add or Edit Cron Job (aaPanel Parity) */}
        {modalOpen && (
          <div className="fixed inset-0 z-50 bg-black/70 backdrop-blur-sm flex items-center justify-center p-4">
            <div className="w-full max-w-2xl bg-white dark:bg-[#10141d] border border-slate-200 dark:border-surface-700 rounded-2xl shadow-2xl p-6 relative max-h-[90vh] overflow-y-auto">
              <button
                onClick={() => setModalOpen(false)}
                className="absolute top-5 right-5 p-1 rounded-lg text-slate-400 hover:text-slate-700 dark:hover:text-white transition-colors cursor-pointer"
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
                    aaPanel-compatible task scheduler & automated cloud backups
                  </p>
                </div>
              </div>

              <form onSubmit={handleSubmitJob} className="space-y-4">
                {/* 1. Task Type Selection (aaPanel Parity) */}
                <div>
                  <label className="block text-xs font-semibold uppercase tracking-wider text-slate-700 dark:text-slate-300 mb-2">
                    Task Type (aaPanel Parity)
                  </label>
                  <div className="grid grid-cols-2 sm:grid-cols-3 gap-2">
                    <button
                      type="button"
                      onClick={() => setTaskType('shell')}
                      className={`p-2.5 rounded-xl border text-left transition-all cursor-pointer flex items-center gap-2 text-xs font-semibold ${
                        taskType === 'shell'
                          ? 'border-indigo-500 bg-indigo-500/10 text-indigo-600 dark:text-indigo-400'
                          : 'border-slate-200 dark:border-surface-800 text-slate-700 dark:text-slate-300 hover:bg-slate-50 dark:hover:bg-surface-800'
                      }`}
                    >
                      <Terminal className="w-4 h-4 shrink-0" />
                      <span>Shell Script</span>
                    </button>
                    <button
                      type="button"
                      onClick={() => setTaskType('backup_site')}
                      className={`p-2.5 rounded-xl border text-left transition-all cursor-pointer flex items-center gap-2 text-xs font-semibold ${
                        taskType === 'backup_site'
                          ? 'border-indigo-500 bg-indigo-500/10 text-indigo-600 dark:text-indigo-400'
                          : 'border-slate-200 dark:border-surface-800 text-slate-700 dark:text-slate-300 hover:bg-slate-50 dark:hover:bg-surface-800'
                      }`}
                    >
                      <Archive className="w-4 h-4 shrink-0" />
                      <span>Backup Site</span>
                    </button>
                    <button
                      type="button"
                      onClick={() => setTaskType('backup_database')}
                      className={`p-2.5 rounded-xl border text-left transition-all cursor-pointer flex items-center gap-2 text-xs font-semibold ${
                        taskType === 'backup_database'
                          ? 'border-indigo-500 bg-indigo-500/10 text-indigo-600 dark:text-indigo-400'
                          : 'border-slate-200 dark:border-surface-800 text-slate-700 dark:text-slate-300 hover:bg-slate-50 dark:hover:bg-surface-800'
                      }`}
                    >
                      <Database className="w-4 h-4 shrink-0" />
                      <span>Backup Database</span>
                    </button>
                    <button
                      type="button"
                      onClick={() => setTaskType('url_access')}
                      className={`p-2.5 rounded-xl border text-left transition-all cursor-pointer flex items-center gap-2 text-xs font-semibold ${
                        taskType === 'url_access'
                          ? 'border-indigo-500 bg-indigo-500/10 text-indigo-600 dark:text-indigo-400'
                          : 'border-slate-200 dark:border-surface-800 text-slate-700 dark:text-slate-300 hover:bg-slate-50 dark:hover:bg-surface-800'
                      }`}
                    >
                      <Globe className="w-4 h-4 shrink-0" />
                      <span>Access URL</span>
                    </button>
                    <button
                      type="button"
                      onClick={() => setTaskType('free_ram')}
                      className={`p-2.5 rounded-xl border text-left transition-all cursor-pointer flex items-center gap-2 text-xs font-semibold ${
                        taskType === 'free_ram'
                          ? 'border-indigo-500 bg-indigo-500/10 text-indigo-600 dark:text-indigo-400'
                          : 'border-slate-200 dark:border-surface-800 text-slate-700 dark:text-slate-300 hover:bg-slate-50 dark:hover:bg-surface-800'
                      }`}
                    >
                      <Cpu className="w-4 h-4 shrink-0" />
                      <span>Free RAM</span>
                    </button>
                    <button
                      type="button"
                      onClick={() => setTaskType('cut_logs')}
                      className={`p-2.5 rounded-xl border text-left transition-all cursor-pointer flex items-center gap-2 text-xs font-semibold ${
                        taskType === 'cut_logs'
                          ? 'border-indigo-500 bg-indigo-500/10 text-indigo-600 dark:text-indigo-400'
                          : 'border-slate-200 dark:border-surface-800 text-slate-700 dark:text-slate-300 hover:bg-slate-50 dark:hover:bg-surface-800'
                      }`}
                    >
                      <FileText className="w-4 h-4 shrink-0" />
                      <span>Cut Web Logs</span>
                    </button>
                  </div>
                </div>

                {/* 2. Contextual Inputs based on Task Type */}
                {taskType === 'url_access' && (
                  <div className="p-3.5 bg-slate-50 dark:bg-surface-950 rounded-xl border border-slate-200 dark:border-surface-800 space-y-2">
                    <label className="block text-xs font-semibold text-slate-700 dark:text-slate-300">
                      Target URL (HTTP/HTTPS GET)
                    </label>
                    <input
                      type="url"
                      required
                      value={accessUrl}
                      onChange={(e) => setAccessUrl(e.target.value)}
                      placeholder="https://example.com/wp-cron.php"
                      className="w-full px-3 py-2 rounded-lg bg-white dark:bg-surface-900 border border-slate-300 dark:border-surface-700 text-xs font-mono focus:outline-none focus:border-indigo-500"
                    />
                  </div>
                )}

                {taskType === 'backup_site' && (
                  <div className="p-3.5 bg-slate-50 dark:bg-surface-950 rounded-xl border border-slate-200 dark:border-surface-800 grid grid-cols-1 sm:grid-cols-2 gap-3">
                    <div>
                      <label className="block text-xs font-semibold text-slate-700 dark:text-slate-300 mb-1">
                        Website Domain / Name
                      </label>
                      <input
                        type="text"
                        required
                        value={targetSite}
                        onChange={(e) => setTargetSite(e.target.value)}
                        placeholder="e.g. mysite.com"
                        className="w-full px-3 py-2 rounded-lg bg-white dark:bg-surface-900 border border-slate-300 dark:border-surface-700 text-xs font-mono focus:outline-none focus:border-indigo-500"
                      />
                    </div>
                    <div>
                      <label className="block text-xs font-semibold text-slate-700 dark:text-slate-300 mb-1">
                        Retain Latest Copies
                      </label>
                      <input
                        type="number"
                        min={1}
                        max={30}
                        value={backupCopies}
                        onChange={(e) => setBackupCopies(parseInt(e.target.value) || 3)}
                        className="w-full px-3 py-2 rounded-lg bg-white dark:bg-surface-900 border border-slate-300 dark:border-surface-700 text-xs font-mono focus:outline-none focus:border-indigo-500"
                      />
                    </div>
                  </div>
                )}

                {taskType === 'backup_database' && (
                  <div className="p-3.5 bg-slate-50 dark:bg-surface-950 rounded-xl border border-slate-200 dark:border-surface-800 grid grid-cols-1 sm:grid-cols-2 gap-3">
                    <div>
                      <label className="block text-xs font-semibold text-slate-700 dark:text-slate-300 mb-1">
                        MySQL Database Name
                      </label>
                      <input
                        type="text"
                        required
                        value={targetDb}
                        onChange={(e) => setTargetDb(e.target.value)}
                        placeholder="e.g. app_production"
                        className="w-full px-3 py-2 rounded-lg bg-white dark:bg-surface-900 border border-slate-300 dark:border-surface-700 text-xs font-mono focus:outline-none focus:border-indigo-500"
                      />
                    </div>
                    <div>
                      <label className="block text-xs font-semibold text-slate-700 dark:text-slate-300 mb-1">
                        Retain Latest Copies
                      </label>
                      <input
                        type="number"
                        min={1}
                        max={30}
                        value={backupCopies}
                        onChange={(e) => setBackupCopies(parseInt(e.target.value) || 3)}
                        className="w-full px-3 py-2 rounded-lg bg-white dark:bg-surface-900 border border-slate-300 dark:border-surface-700 text-xs font-mono focus:outline-none focus:border-indigo-500"
                      />
                    </div>
                  </div>
                )}

                {/* 3. Execution Cycle Builder (aaPanel Parity) */}
                <div className="p-4 bg-slate-50 dark:bg-surface-950 rounded-xl border border-slate-200 dark:border-surface-800 space-y-3">
                  <div className="flex items-center justify-between">
                    <label className="block text-xs font-semibold uppercase tracking-wider text-slate-700 dark:text-slate-300">
                      Execution Cycle
                    </label>
                    <span className="font-mono text-xs px-2 py-0.5 rounded bg-indigo-500/10 text-indigo-600 dark:text-indigo-400 border border-indigo-500/20 font-bold">
                      {schedule}
                    </span>
                  </div>

                  <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
                    <div>
                      <label className="block text-[11px] text-slate-500 mb-1">Cycle Type</label>
                      <select
                        value={cycleType}
                        onChange={(e: any) => setCycleType(e.target.value)}
                        className="w-full px-3 py-2 rounded-lg bg-white dark:bg-surface-900 border border-slate-300 dark:border-surface-700 text-xs font-semibold text-slate-900 dark:text-white focus:outline-none focus:border-indigo-500"
                      >
                        <option value="day">Every Day</option>
                        <option value="day_n">Every N Days</option>
                        <option value="hour">Every Hour</option>
                        <option value="hour_n">Every N Hours</option>
                        <option value="minute_n">Every N Minutes</option>
                        <option value="week">Every Week</option>
                        <option value="month">Every Month</option>
                        <option value="custom">Custom Cron Expression</option>
                      </select>
                    </div>

                    {/* Cycle Parameters */}
                    {cycleType === 'day' && (
                      <div className="grid grid-cols-2 gap-2">
                        <div>
                          <label className="block text-[11px] text-slate-500 mb-1">Hour (0-23)</label>
                          <input
                            type="number"
                            min={0}
                            max={23}
                            value={cycleHour}
                            onChange={(e) => setCycleHour(parseInt(e.target.value) || 0)}
                            className="w-full px-2.5 py-2 rounded-lg bg-white dark:bg-surface-900 border border-slate-300 dark:border-surface-700 text-xs font-mono"
                          />
                        </div>
                        <div>
                          <label className="block text-[11px] text-slate-500 mb-1">Minute (0-59)</label>
                          <input
                            type="number"
                            min={0}
                            max={59}
                            value={cycleMinute}
                            onChange={(e) => setCycleMinute(parseInt(e.target.value) || 0)}
                            className="w-full px-2.5 py-2 rounded-lg bg-white dark:bg-surface-900 border border-slate-300 dark:border-surface-700 text-xs font-mono"
                          />
                        </div>
                      </div>
                    )}

                    {cycleType === 'day_n' && (
                      <div className="grid grid-cols-3 gap-2">
                        <div>
                          <label className="block text-[11px] text-slate-500 mb-1">N Days</label>
                          <input
                            type="number"
                            min={1}
                            max={31}
                            value={cycleNDays}
                            onChange={(e) => setCycleNDays(parseInt(e.target.value) || 1)}
                            className="w-full px-2 py-2 rounded-lg bg-white dark:bg-surface-900 border border-slate-300 dark:border-surface-700 text-xs font-mono"
                          />
                        </div>
                        <div>
                          <label className="block text-[11px] text-slate-500 mb-1">Hour</label>
                          <input
                            type="number"
                            min={0}
                            max={23}
                            value={cycleHour}
                            onChange={(e) => setCycleHour(parseInt(e.target.value) || 0)}
                            className="w-full px-2 py-2 rounded-lg bg-white dark:bg-surface-900 border border-slate-300 dark:border-surface-700 text-xs font-mono"
                          />
                        </div>
                        <div>
                          <label className="block text-[11px] text-slate-500 mb-1">Minute</label>
                          <input
                            type="number"
                            min={0}
                            max={59}
                            value={cycleMinute}
                            onChange={(e) => setCycleMinute(parseInt(e.target.value) || 0)}
                            className="w-full px-2 py-2 rounded-lg bg-white dark:bg-surface-900 border border-slate-300 dark:border-surface-700 text-xs font-mono"
                          />
                        </div>
                      </div>
                    )}

                    {cycleType === 'hour' && (
                      <div>
                        <label className="block text-[11px] text-slate-500 mb-1">Minute (0-59)</label>
                        <input
                          type="number"
                          min={0}
                          max={59}
                          value={cycleMinute}
                          onChange={(e) => setCycleMinute(parseInt(e.target.value) || 0)}
                          className="w-full px-3 py-2 rounded-lg bg-white dark:bg-surface-900 border border-slate-300 dark:border-surface-700 text-xs font-mono"
                        />
                      </div>
                    )}

                    {cycleType === 'hour_n' && (
                      <div className="grid grid-cols-2 gap-2">
                        <div>
                          <label className="block text-[11px] text-slate-500 mb-1">Every N Hours</label>
                          <input
                            type="number"
                            min={1}
                            max={23}
                            value={cycleNHours}
                            onChange={(e) => setCycleNHours(parseInt(e.target.value) || 1)}
                            className="w-full px-3 py-2 rounded-lg bg-white dark:bg-surface-900 border border-slate-300 dark:border-surface-700 text-xs font-mono"
                          />
                        </div>
                        <div>
                          <label className="block text-[11px] text-slate-500 mb-1">Minute (0-59)</label>
                          <input
                            type="number"
                            min={0}
                            max={59}
                            value={cycleMinute}
                            onChange={(e) => setCycleMinute(parseInt(e.target.value) || 0)}
                            className="w-full px-3 py-2 rounded-lg bg-white dark:bg-surface-900 border border-slate-300 dark:border-surface-700 text-xs font-mono"
                          />
                        </div>
                      </div>
                    )}

                    {cycleType === 'minute_n' && (
                      <div>
                        <label className="block text-[11px] text-slate-500 mb-1">Every N Minutes (1-59)</label>
                        <input
                          type="number"
                          min={1}
                          max={59}
                          value={cycleNMinutes}
                          onChange={(e) => setCycleNMinutes(parseInt(e.target.value) || 1)}
                          className="w-full px-3 py-2 rounded-lg bg-white dark:bg-surface-900 border border-slate-300 dark:border-surface-700 text-xs font-mono"
                        />
                      </div>
                    )}

                    {cycleType === 'week' && (
                      <div className="grid grid-cols-3 gap-2">
                        <div>
                          <label className="block text-[11px] text-slate-500 mb-1">Day of Week</label>
                          <select
                            value={cycleWeekday}
                            onChange={(e) => setCycleWeekday(parseInt(e.target.value) || 0)}
                            className="w-full px-2 py-2 rounded-lg bg-white dark:bg-surface-900 border border-slate-300 dark:border-surface-700 text-xs font-semibold"
                          >
                            <option value={0}>Sunday</option>
                            <option value={1}>Monday</option>
                            <option value={2}>Tuesday</option>
                            <option value={3}>Wednesday</option>
                            <option value={4}>Thursday</option>
                            <option value={5}>Friday</option>
                            <option value={6}>Saturday</option>
                          </select>
                        </div>
                        <div>
                          <label className="block text-[11px] text-slate-500 mb-1">Hour</label>
                          <input
                            type="number"
                            min={0}
                            max={23}
                            value={cycleHour}
                            onChange={(e) => setCycleHour(parseInt(e.target.value) || 0)}
                            className="w-full px-2 py-2 rounded-lg bg-white dark:bg-surface-900 border border-slate-300 dark:border-surface-700 text-xs font-mono"
                          />
                        </div>
                        <div>
                          <label className="block text-[11px] text-slate-500 mb-1">Minute</label>
                          <input
                            type="number"
                            min={0}
                            max={59}
                            value={cycleMinute}
                            onChange={(e) => setCycleMinute(parseInt(e.target.value) || 0)}
                            className="w-full px-2 py-2 rounded-lg bg-white dark:bg-surface-900 border border-slate-300 dark:border-surface-700 text-xs font-mono"
                          />
                        </div>
                      </div>
                    )}

                    {cycleType === 'month' && (
                      <div className="grid grid-cols-3 gap-2">
                        <div>
                          <label className="block text-[11px] text-slate-500 mb-1">Day of Month</label>
                          <input
                            type="number"
                            min={1}
                            max={31}
                            value={cycleDayOfMonth}
                            onChange={(e) => setCycleDayOfMonth(parseInt(e.target.value) || 1)}
                            className="w-full px-2 py-2 rounded-lg bg-white dark:bg-surface-900 border border-slate-300 dark:border-surface-700 text-xs font-mono"
                          />
                        </div>
                        <div>
                          <label className="block text-[11px] text-slate-500 mb-1">Hour</label>
                          <input
                            type="number"
                            min={0}
                            max={23}
                            value={cycleHour}
                            onChange={(e) => setCycleHour(parseInt(e.target.value) || 0)}
                            className="w-full px-2 py-2 rounded-lg bg-white dark:bg-surface-900 border border-slate-300 dark:border-surface-700 text-xs font-mono"
                          />
                        </div>
                        <div>
                          <label className="block text-[11px] text-slate-500 mb-1">Minute</label>
                          <input
                            type="number"
                            min={0}
                            max={59}
                            value={cycleMinute}
                            onChange={(e) => setCycleMinute(parseInt(e.target.value) || 0)}
                            className="w-full px-2 py-2 rounded-lg bg-white dark:bg-surface-900 border border-slate-300 dark:border-surface-700 text-xs font-mono"
                          />
                        </div>
                      </div>
                    )}

                    {cycleType === 'custom' && (
                      <div>
                        <label className="block text-[11px] text-slate-500 mb-1">Expression (min hour day month weekday)</label>
                        <input
                          type="text"
                          required
                          value={schedule}
                          onChange={(e) => setSchedule(e.target.value)}
                          placeholder="e.g. 0 2 * * *"
                          className="w-full px-3 py-2 rounded-lg bg-white dark:bg-surface-900 border border-slate-300 dark:border-surface-700 text-xs font-mono focus:outline-none focus:border-indigo-500"
                        />
                      </div>
                    )}
                  </div>
                </div>

                {/* 4. Command to Execute */}
                <div>
                  <div className="flex items-center justify-between mb-1.5">
                    <label className="block text-xs font-semibold uppercase tracking-wider text-slate-700 dark:text-slate-300">
                      Script Command to Execute
                    </label>
                    <span className="text-[10px] text-slate-400">Bash syntax</span>
                  </div>
                  <textarea
                    required
                    rows={3}
                    value={command}
                    onChange={(e) => setCommand(e.target.value)}
                    placeholder="php /var/www/site/artisan schedule:run"
                    className="w-full px-3.5 py-2.5 rounded-xl bg-slate-50 dark:bg-surface-950 border border-slate-300 dark:border-surface-700 text-slate-950 dark:text-slate-100 font-mono text-xs focus:outline-none focus:border-indigo-500 resize-y"
                  />
                </div>

                {/* 5. System User & Description */}
                <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
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
                      Description / Task Name
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

                <div className="flex justify-end gap-2 pt-3 border-t border-slate-200 dark:border-surface-800">
                  <button
                    type="button"
                    onClick={() => setModalOpen(false)}
                    className="px-4 py-2.5 rounded-xl text-xs font-medium text-slate-600 dark:text-slate-400 hover:bg-slate-100 dark:hover:bg-surface-800 transition-colors cursor-pointer"
                  >
                    Cancel
                  </button>
                  <button
                    type="submit"
                    disabled={submittingJob}
                    className="px-5 py-2.5 rounded-xl bg-[#16A34A] hover:bg-[#15803D] active:bg-[#166534] text-white text-xs font-semibold shadow-sm transition-all flex items-center gap-1.5 cursor-pointer"
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
