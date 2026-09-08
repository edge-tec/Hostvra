'use client';

import React, { useState, useEffect, useCallback } from 'react';
import {
  HardDriveDownload,
  Plus,
  RefreshCw,
  RotateCcw,
  CheckCircle2,
  AlertTriangle,
  Cloud,
  HardDrive,
  FileArchive,
  Database,
  ShieldCheck,
  Calendar,
  Trash2,
  Download,
  X,
  Search,
  Check,
  Copy,
  Server,
  Key,
  ExternalLink,
  Layers,
  Settings2,
  Clock,
} from 'lucide-react';
import { DashboardShell } from '@/components/DashboardShell';
import {
  apiFetch,
  getApiBaseUrl,
  BackupRecord,
  BackupDestination,
  BackupSchedule,
} from '@/lib/api';

function formatBytes(bytes: number): string {
  if (bytes === 0) return '0 B';
  if (bytes < 1024) return bytes + ' B';
  if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB';
  if (bytes < 1024 * 1024 * 1024) return (bytes / (1024 * 1024)).toFixed(1) + ' MB';
  return (bytes / (1024 * 1024 * 1024)).toFixed(2) + ' GB';
}

export default function BackupsPage() {
  const [activeTab, setActiveTab] = useState<'snapshots' | 'destinations' | 'schedules'>('snapshots');

  // Core Data States
  const [backups, setBackups] = useState<BackupRecord[]>([]);
  const [destinations, setDestinations] = useState<BackupDestination[]>([]);
  const [schedules, setSchedules] = useState<BackupSchedule[]>([]);
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);

  // Filters & Search
  const [scopeFilter, setScopeFilter] = useState<'all' | 'website' | 'database' | 'full_config'>('all');
  const [searchQuery, setSearchQuery] = useState('');

  // Alerts & Notifications
  const [errorMsg, setErrorMsg] = useState<string | null>(null);
  const [successMsg, setSuccessMsg] = useState<string | null>(null);
  const [copiedHash, setCopiedHash] = useState<string | null>(null);

  // Modal States
  const [createModalOpen, setCreateModalOpen] = useState(false);
  const [submittingCreate, setSubmittingCreate] = useState(false);
  const [restoreModalRecord, setRestoreModalRecord] = useState<BackupRecord | null>(null);
  const [submittingRestore, setSubmittingRestore] = useState(false);
  const [restoreSuccess, setRestoreSuccess] = useState(false);

  // Destination Modal States
  const [destModalOpen, setDestModalOpen] = useState(false);
  const [submittingDest, setSubmittingDest] = useState(false);
  const [testingDest, setTestingDest] = useState(false);
  const [testSuccess, setTestSuccess] = useState<string | null>(null);
  const [testError, setTestError] = useState<string | null>(null);

  // Schedule Modal States
  const [schedModalOpen, setSchedModalOpen] = useState(false);
  const [submittingSched, setSubmittingSched] = useState(false);

  // Create Form State
  const [createType, setCreateType] = useState<'website' | 'database' | 'full_config'>('website');
  const [createTargetName, setCreateTargetName] = useState('');
  const [createStorage, setCreateStorage] = useState<'local' | 's3'>('local');
  const [createDestID, setCreateDestID] = useState('');
  const [createRetention, setCreateRetention] = useState('7');

  // Destination Form State
  const [destProvider, setDestProvider] = useState<'s3' | 'r2' | 'b2' | 'sftp'>('s3');
  const [destName, setDestName] = useState('');
  const [destEndpoint, setDestEndpoint] = useState('');
  const [destRegion, setDestRegion] = useState('us-east-1');
  const [destBucket, setDestBucket] = useState('');
  const [destAccessKey, setDestAccessKey] = useState('');
  const [destSecretKey, setDestSecretKey] = useState('');
  const [destPrefix, setDestPrefix] = useState('hostvra-backups');
  const [destIsDefault, setDestIsDefault] = useState(true);

  // Schedule Form State
  const [schedName, setSchedName] = useState('');
  const [schedScope, setSchedScope] = useState<'website' | 'database' | 'full_config'>('website');
  const [schedTargetName, setSchedTargetName] = useState('');
  const [schedDestID, setSchedDestID] = useState('');
  const [schedFreq, setSchedFreq] = useState<'daily' | 'weekly' | 'monthly' | 'cron'>('daily');
  const [schedRetention, setSchedRetention] = useState('7');

  // Auto-dismiss alert messages
  useEffect(() => {
    if (successMsg) {
      const t = setTimeout(() => setSuccessMsg(null), 4000);
      return () => clearTimeout(t);
    }
  }, [successMsg]);

  useEffect(() => {
    if (errorMsg) {
      const t = setTimeout(() => setErrorMsg(null), 6000);
      return () => clearTimeout(t);
    }
  }, [errorMsg]);

  // Load all initial data from real backend
  const loadData = useCallback(async (isRefresh = false) => {
    if (isRefresh) setRefreshing(true);
    else setLoading(true);

    try {
      const [bkRes, destRes, schedRes] = await Promise.all([
        apiFetch<BackupRecord[]>('/api/v1/backups'),
        apiFetch<BackupDestination[]>('/api/v1/backups/destinations'),
        apiFetch<BackupSchedule[]>('/api/v1/backups/schedules'),
      ]);

      if (bkRes.success && bkRes.data) {
        setBackups(bkRes.data);
      }
      if (destRes.success && destRes.data) {
        setDestinations(destRes.data);
        if (destRes.data.length > 0 && !createDestID) {
          const defaultDest = destRes.data.find((d) => d.is_default) || destRes.data[0];
          setCreateDestID(defaultDest.id);
          setSchedDestID(defaultDest.id);
        }
      }
      if (schedRes.success && schedRes.data) {
        setSchedules(schedRes.data);
      }
    } catch (err: any) {
      setErrorMsg(err.message || 'Failed to load backup configuration from server');
    } finally {
      setLoading(false);
      setRefreshing(false);
    }
  }, [createDestID]);

  useEffect(() => {
    loadData();
  }, [loadData]);

  // Handle Create Backup
  const handleCreateBackup = async (e: React.FormEvent) => {
    e.preventDefault();
    setSubmittingCreate(true);
    setErrorMsg(null);

    try {
      const payload = {
        server_id: 'srv-main-01',
        type: createType,
        target_name: createType === 'full_config' ? 'Full Server Stack' : createTargetName.trim(),
        storage: createStorage,
        destination_id: createStorage === 's3' ? createDestID : undefined,
        retention: parseInt(createRetention, 10) || 7,
      };

      const res = await apiFetch<BackupRecord>('/api/v1/backups/create', {
        method: 'POST',
        body: JSON.stringify(payload),
      });

      if (res.success && res.data) {
        setSuccessMsg(`Snapshot ${res.data.id} created and verified successfully.`);
        setCreateModalOpen(false);
        setCreateTargetName('');
        loadData(true);
      } else {
        setErrorMsg(res.error?.message || 'Failed to trigger backup');
      }
    } catch (err: any) {
      setErrorMsg(err.message || 'Network error while creating backup');
    } finally {
      setSubmittingCreate(false);
    }
  };

  // Handle Restore
  const handleConfirmRestore = async () => {
    if (!restoreModalRecord) return;
    setSubmittingRestore(true);
    setErrorMsg(null);

    try {
      const res = await apiFetch<{ status: string; backup_id: string }>('/api/v1/backups/restore', {
        method: 'POST',
        body: JSON.stringify({
          backup_id: restoreModalRecord.id,
          server_id: restoreModalRecord.server_id,
        }),
      });

      if (res.success) {
        setRestoreSuccess(true);
        setTimeout(() => {
          setRestoreSuccess(false);
          setRestoreModalRecord(null);
          setSuccessMsg(`Snapshot ${restoreModalRecord.id} restored successfully with verified rollback.`);
        }, 1800);
      } else {
        setErrorMsg(res.error?.message || 'Restore failed');
      }
    } catch (err: any) {
      setErrorMsg(err.message || 'Failed to execute restore');
    } finally {
      setSubmittingRestore(false);
    }
  };

  // Handle Delete Backup
  const handleDeleteBackup = async (id: string) => {
    if (!confirm(`Are you sure you want to permanently delete archive ${id}?`)) return;

    try {
      const res = await apiFetch(`/api/v1/backups/${id}`, { method: 'DELETE' });
      if (res.success) {
        setSuccessMsg(`Backup ${id} deleted successfully.`);
        setBackups((prev) => prev.filter((b) => b.id !== id));
      } else {
        setErrorMsg(res.error?.message || 'Failed to delete backup');
      }
    } catch (err: any) {
      setErrorMsg(err.message || 'Failed to delete backup');
    }
  };

  // Handle Download Backup
  const handleDownloadBackup = (id: string, fileName: string) => {
    const token = typeof window !== 'undefined' ? localStorage.getItem('hostvra_token') : '';
    const url = `${getApiBaseUrl()}/api/v1/backups/download/${id}`;
    const a = document.createElement('a');
    a.href = url;
    a.download = fileName;
    if (token) {
      // In web browser download, open directly or fetch as blob
      window.open(url, '_blank');
    } else {
      window.open(url, '_blank');
    }
  };

  // Handle Test Destination
  const handleTestDestination = async () => {
    setTestingDest(true);
    setTestSuccess(null);
    setTestError(null);

    try {
      const payload = {
        name: destName || 'Test Provider',
        type: destProvider,
        endpoint: destEndpoint.trim(),
        region: destRegion.trim(),
        bucket: destBucket.trim(),
        access_key: destAccessKey.trim(),
        secret_key: destSecretKey.trim(),
        prefix: destPrefix.trim(),
      };

      const res = await apiFetch<{ status: string; message: string }>('/api/v1/backups/destinations/test', {
        method: 'POST',
        body: JSON.stringify(payload),
      });

      if (res.success) {
        setTestSuccess('Successfully reached bucket and verified S3 SigV4 credentials!');
      } else {
        setTestError(res.error?.message || 'Connection test failed. Check endpoint, bucket, or keys.');
      }
    } catch (err: any) {
      setTestError(err.message || 'Network error testing cloud storage');
    } finally {
      setTestingDest(false);
    }
  };

  // Handle Save Destination
  const handleSaveDestination = async (e: React.FormEvent) => {
    e.preventDefault();
    setSubmittingDest(true);
    setErrorMsg(null);

    try {
      const payload = {
        name: destName.trim(),
        type: destProvider,
        endpoint: destEndpoint.trim(),
        region: destRegion.trim(),
        bucket: destBucket.trim(),
        access_key: destAccessKey.trim(),
        secret_key: destSecretKey.trim(),
        prefix: destPrefix.trim(),
        is_default: destIsDefault,
      };

      const res = await apiFetch<BackupDestination>('/api/v1/backups/destinations', {
        method: 'POST',
        body: JSON.stringify(payload),
      });

      if (res.success && res.data) {
        setSuccessMsg(`Cloud destination "${res.data.name}" saved successfully.`);
        setDestModalOpen(false);
        setDestName('');
        setDestEndpoint('');
        setDestBucket('');
        setDestAccessKey('');
        setDestSecretKey('');
        setTestSuccess(null);
        setTestError(null);
        loadData(true);
      } else {
        setErrorMsg(res.error?.message || 'Failed to save cloud destination');
      }
    } catch (err: any) {
      setErrorMsg(err.message || 'Network error saving cloud destination');
    } finally {
      setSubmittingDest(false);
    }
  };

  // Handle Delete Destination
  const handleDeleteDestination = async (id: string, name: string) => {
    if (!confirm(`Are you sure you want to remove remote cloud target "${name}"?`)) return;

    try {
      const res = await apiFetch(`/api/v1/backups/destinations/${id}`, { method: 'DELETE' });
      if (res.success) {
        setSuccessMsg(`Destination "${name}" removed.`);
        setDestinations((prev) => prev.filter((d) => d.id !== id));
      } else {
        setErrorMsg(res.error?.message || 'Failed to remove destination');
      }
    } catch (err: any) {
      setErrorMsg(err.message || 'Error removing destination');
    }
  };

  // Handle Save Schedule
  const handleSaveSchedule = async (e: React.FormEvent) => {
    e.preventDefault();
    setSubmittingSched(true);
    setErrorMsg(null);

    try {
      const payload = {
        name: schedName.trim(),
        scope: schedScope,
        target_name: schedScope === 'full_config' ? 'Full Server Stack' : schedTargetName.trim(),
        destination_id: schedDestID,
        frequency: schedFreq,
        retention: parseInt(schedRetention, 10) || 7,
        enabled: true,
      };

      const res = await apiFetch<BackupSchedule>('/api/v1/backups/schedules', {
        method: 'POST',
        body: JSON.stringify(payload),
      });

      if (res.success && res.data) {
        setSuccessMsg(`Schedule "${res.data.name}" activated.`);
        setSchedModalOpen(false);
        setSchedName('');
        setSchedTargetName('');
        loadData(true);
      } else {
        setErrorMsg(res.error?.message || 'Failed to create schedule');
      }
    } catch (err: any) {
      setErrorMsg(err.message || 'Network error creating schedule');
    } finally {
      setSubmittingSched(false);
    }
  };

  // Handle Delete Schedule
  const handleDeleteSchedule = async (id: string, name: string) => {
    if (!confirm(`Delete recurring schedule "${name}"?`)) return;

    try {
      const res = await apiFetch(`/api/v1/backups/schedules/${id}`, { method: 'DELETE' });
      if (res.success) {
        setSuccessMsg(`Schedule "${name}" deleted.`);
        setSchedules((prev) => prev.filter((s) => s.id !== id));
      } else {
        setErrorMsg(res.error?.message || 'Failed to delete schedule');
      }
    } catch (err: any) {
      setErrorMsg(err.message || 'Error deleting schedule');
    }
  };

  // Copy SHA256 Helper
  const copySHA = (hash: string) => {
    navigator.clipboard.writeText(hash);
    setCopiedHash(hash);
    setTimeout(() => setCopiedHash(null), 2000);
  };

  // Filtered backups list
  const filteredBackups = backups.filter((b) => {
    if (scopeFilter !== 'all' && b.type !== scopeFilter) return false;
    if (searchQuery) {
      const q = searchQuery.toLowerCase();
      return (
        b.id.toLowerCase().includes(q) ||
        b.target_name.toLowerCase().includes(q) ||
        b.storage.toLowerCase().includes(q)
      );
    }
    return true;
  });

  // Calculate totals
  const totalSizeBytes = backups.reduce((acc, b) => acc + (b.size_bytes || 0), 0);
  const defaultDest = destinations.find((d) => d.is_default);

  return (
    <DashboardShell>
      <div className="space-y-6">
        {/* Banner Alerts */}
        {errorMsg && (
          <div className="p-4 rounded-xl bg-red-500/10 border border-red-500/20 text-red-400 text-sm flex items-center justify-between animate-fadeIn">
            <div className="flex items-center gap-2">
              <AlertTriangle className="w-5 h-5 flex-shrink-0" />
              <span>{errorMsg}</span>
            </div>
            <button onClick={() => setErrorMsg(null)} className="hover:text-white">
              <X className="w-4 h-4" />
            </button>
          </div>
        )}

        {successMsg && (
          <div className="p-4 rounded-xl bg-emerald-500/10 border border-emerald-500/20 text-emerald-400 text-sm flex items-center justify-between animate-fadeIn">
            <div className="flex items-center gap-2">
              <CheckCircle2 className="w-5 h-5 flex-shrink-0" />
              <span>{successMsg}</span>
            </div>
            <button onClick={() => setSuccessMsg(null)} className="hover:text-white">
              <X className="w-4 h-4" />
            </button>
          </div>
        )}

        {/* Header */}
        <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
          <div>
            <h1 className="text-2xl font-bold text-white tracking-tight flex items-center gap-2">
              <HardDriveDownload className="w-7 h-7 text-brand-400" />
              Automated & Cloud Backups
            </h1>
            <p className="text-sm text-slate-400 mt-1">
              Production snapshot engine with deterministic tar.gz streaming, SHA256 integrity, atomic rollback guard, and AWS S3 / Cloudflare R2 multi-cloud offsite sync.
            </p>
          </div>

          <div className="flex items-center gap-2.5">
            <button
              onClick={() => loadData(true)}
              disabled={refreshing}
              className="p-2.5 rounded-lg bg-surface-900 hover:bg-surface-800 text-slate-400 hover:text-white border border-surface-800 transition-colors disabled:opacity-50"
              title="Refresh from Server"
            >
              <RefreshCw className={`w-4 h-4 ${refreshing ? 'animate-spin text-brand-400' : ''}`} />
            </button>

            {activeTab === 'destinations' ? (
              <button
                onClick={() => setDestModalOpen(true)}
                className="inline-flex items-center justify-center gap-2 px-4 py-2.5 rounded-lg bg-brand-500 hover:bg-brand-600 text-white text-sm font-semibold transition-colors shadow-lg shadow-brand-500/20"
              >
                <Cloud className="w-4 h-4" />
                Add Cloud Target
              </button>
            ) : activeTab === 'schedules' ? (
              <button
                onClick={() => setSchedModalOpen(true)}
                className="inline-flex items-center justify-center gap-2 px-4 py-2.5 rounded-lg bg-brand-500 hover:bg-brand-600 text-white text-sm font-semibold transition-colors shadow-lg shadow-brand-500/20"
              >
                <Calendar className="w-4 h-4" />
                New Schedule
              </button>
            ) : (
              <button
                onClick={() => setCreateModalOpen(true)}
                className="inline-flex items-center justify-center gap-2 px-4 py-2.5 rounded-lg bg-brand-500 hover:bg-brand-600 text-white text-sm font-semibold transition-colors shadow-lg shadow-brand-500/20"
              >
                <Plus className="w-4 h-4" />
                Create Backup
              </button>
            )}
          </div>
        </div>

        {/* Overview Stats Cards */}
        <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
          <div className="bg-surface-900 border border-surface-800 rounded-xl p-5 flex items-start gap-4">
            <div className="p-3 rounded-lg bg-brand-500/10 text-brand-400 border border-brand-500/20">
              <FileArchive className="w-5 h-5" />
            </div>
            <div>
              <div className="text-xs font-semibold text-slate-400 uppercase tracking-wider">Total Snapshots</div>
              <div className="text-xl font-bold text-white mt-1">{backups.length}</div>
              <div className="text-xs text-slate-400 mt-1 font-mono">{formatBytes(totalSizeBytes)} stored</div>
            </div>
          </div>

          <div className="bg-surface-900 border border-surface-800 rounded-xl p-5 flex items-start gap-4">
            <div className="p-3 rounded-lg bg-blue-500/10 text-blue-400 border border-blue-500/20">
              <Cloud className="w-5 h-5" />
            </div>
            <div>
              <div className="text-xs font-semibold text-slate-400 uppercase tracking-wider">Cloud Offsite</div>
              <div className="text-xl font-bold text-white mt-1">
                {destinations.length > 0 ? (
                  <span className="text-emerald-400 text-base flex items-center gap-1">
                    <CheckCircle2 className="w-4 h-4" />
                    {defaultDest?.name || `${destinations.length} Configured`}
                  </span>
                ) : (
                  <span className="text-slate-400 text-sm">Not Configured</span>
                )}
              </div>
              <div className="text-xs text-slate-400 mt-1">S3 / R2 / Backblaze</div>
            </div>
          </div>

          <div className="bg-surface-900 border border-surface-800 rounded-xl p-5 flex items-start gap-4">
            <div className="p-3 rounded-lg bg-amber-500/10 text-amber-400 border border-amber-500/20">
              <Clock className="w-5 h-5" />
            </div>
            <div>
              <div className="text-xs font-semibold text-slate-400 uppercase tracking-wider">Automated Schedules</div>
              <div className="text-xl font-bold text-white mt-1">{schedules.filter((s) => s.enabled).length} Active</div>
              <div className="text-xs text-slate-400 mt-1">{schedules.length} total tasks</div>
            </div>
          </div>

          <div className="bg-surface-900 border border-surface-800 rounded-xl p-5 flex items-start gap-4">
            <div className="p-3 rounded-lg bg-emerald-500/10 text-emerald-400 border border-emerald-500/20">
              <ShieldCheck className="w-5 h-5" />
            </div>
            <div>
              <div className="text-xs font-semibold text-slate-400 uppercase tracking-wider">Rollback Guard</div>
              <div className="text-xl font-bold text-emerald-400 mt-1">Armed & Ready</div>
              <div className="text-xs text-slate-400 mt-1">Pre-restore staging active</div>
            </div>
          </div>
        </div>

        {/* Navigation Tabs */}
        <div className="flex items-center gap-1.5 p-1 bg-slate-100/80 dark:bg-slate-800/60 rounded-xl overflow-x-auto text-xs font-semibold">
          <button
            role="tab"
            data-tab="true"
            onClick={() => setActiveTab('snapshots')}
            className={`flex items-center gap-2 px-3.5 py-2 rounded-lg transition-all cursor-pointer font-medium whitespace-nowrap ${
              activeTab === 'snapshots'
                ? 'bg-white dark:bg-slate-900 text-brand-500 dark:text-brand-400 shadow-xs'
                : 'text-slate-600 dark:text-slate-400 hover:text-slate-900 dark:hover:text-white hover:bg-white/40 dark:hover:bg-slate-700/40'
            }`}
          >
            <FileArchive className="w-4 h-4" />
            <span>Snapshots & Archives ({backups.length})</span>
          </button>

          <button
            role="tab"
            data-tab="true"
            onClick={() => setActiveTab('destinations')}
            className={`flex items-center gap-2 px-3.5 py-2 rounded-lg transition-all cursor-pointer font-medium whitespace-nowrap ${
              activeTab === 'destinations'
                ? 'bg-white dark:bg-slate-900 text-brand-500 dark:text-brand-400 shadow-xs'
                : 'text-slate-600 dark:text-slate-400 hover:text-slate-900 dark:hover:text-white hover:bg-white/40 dark:hover:bg-slate-700/40'
            }`}
          >
            <Cloud className="w-4 h-4" />
            <span>Remote Cloud Storage ({destinations.length})</span>
          </button>

          <button
            role="tab"
            data-tab="true"
            onClick={() => setActiveTab('schedules')}
            className={`flex items-center gap-2 px-3.5 py-2 rounded-lg transition-all cursor-pointer font-medium whitespace-nowrap ${
              activeTab === 'schedules'
                ? 'bg-white dark:bg-slate-900 text-brand-500 dark:text-brand-400 shadow-xs'
                : 'text-slate-600 dark:text-slate-400 hover:text-slate-900 dark:hover:text-white hover:bg-white/40 dark:hover:bg-slate-700/40'
            }`}
          >
            <Calendar className="w-4 h-4" />
            <span>Automated Schedules ({schedules.length})</span>
          </button>
        </div>

        {/* TAB 1: SNAPSHOTS & ARCHIVES */}
        {activeTab === 'snapshots' && (
          <div className="space-y-4">
            {/* Filters and Search Bar */}
            <div className="flex flex-col sm:flex-row items-center justify-between gap-3">
              <div className="flex items-center gap-1.5 p-1 bg-surface-900 border border-surface-800 rounded-lg w-full sm:w-auto">
                {(['all', 'website', 'database', 'full_config'] as const).map((scope) => (
                  <button
                    key={scope}
                    onClick={() => setScopeFilter(scope)}
                    className={`px-3 py-1.5 rounded-md text-xs font-semibold capitalize transition-all ${
                      scopeFilter === scope
                        ? 'bg-brand-500 text-white shadow-sm'
                        : 'text-slate-400 hover:text-white'
                    }`}
                  >
                    {scope === 'all' ? 'All Scopes' : scope === 'full_config' ? 'Full Stack' : scope}
                  </button>
                ))}
              </div>

              <div className="relative w-full sm:w-72">
                <Search className="w-4 h-4 text-slate-500 absolute left-3 top-1/2 -translate-y-1/2" />
                <input
                  type="text"
                  placeholder="Search by target or ID..."
                  value={searchQuery}
                  onChange={(e) => setSearchQuery(e.target.value)}
                  className="w-full pl-9 pr-3 py-2 rounded-lg bg-surface-900 border border-surface-800 text-sm text-white placeholder-slate-500 focus:outline-none focus:border-brand-500"
                />
              </div>
            </div>

            {/* Backups Table */}
            <div className="bg-surface-900 border border-surface-800 rounded-xl overflow-hidden shadow-sm">
              <div className="overflow-x-auto">
                <table className="w-full text-left text-sm text-slate-300">
                  <thead className="bg-surface-950/60 text-xs uppercase font-semibold text-slate-400 border-b border-surface-800">
                    <tr>
                      <th className="px-6 py-3.5">Archive ID</th>
                      <th className="px-6 py-3.5">Scope & Target</th>
                      <th className="px-6 py-3.5">Destination</th>
                      <th className="px-6 py-3.5">Size</th>
                      <th className="px-6 py-3.5">Created</th>
                      <th className="px-6 py-3.5">SHA256 Integrity</th>
                      <th className="px-6 py-3.5 text-right">Actions</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-surface-800/60 font-mono text-xs">
                    {loading ? (
                      <tr>
                        <td colSpan={7} className="px-6 py-12 text-center text-slate-400">
                          <RefreshCw className="w-6 h-6 animate-spin mx-auto text-brand-400 mb-2" />
                          Loading backup records from server...
                        </td>
                      </tr>
                    ) : filteredBackups.length === 0 ? (
                      <tr>
                        <td colSpan={7} className="px-6 py-12 text-center text-slate-400">
                          <FileArchive className="w-8 h-8 mx-auto text-slate-600 mb-2" />
                          No backup archives found matching the current criteria.
                          <div className="mt-3">
                            <button
                              onClick={() => setCreateModalOpen(true)}
                              className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-brand-500/20 text-brand-400 border border-brand-500/30 text-xs font-semibold hover:bg-brand-500/30 transition-all font-sans"
                            >
                              <Plus className="w-3.5 h-3.5" />
                              Create your first snapshot
                            </button>
                          </div>
                        </td>
                      </tr>
                    ) : (
                      filteredBackups.map((b) => (
                        <tr key={b.id} className="hover:bg-surface-800/30 transition-colors">
                          <td className="px-6 py-4 font-bold text-white flex items-center gap-2">
                            <FileArchive className="w-4 h-4 text-brand-400 flex-shrink-0" />
                            <span>{b.id}</span>
                          </td>
                          <td className="px-6 py-4">
                            <div className="flex items-center gap-2">
                              {b.type === 'website' && (
                                <span className="px-2 py-0.5 rounded bg-blue-500/10 text-blue-400 border border-blue-500/20 font-sans text-[11px] font-semibold uppercase">
                                  Site
                                </span>
                              )}
                              {b.type === 'database' && (
                                <span className="px-2 py-0.5 rounded bg-amber-500/10 text-amber-400 border border-amber-500/20 font-sans text-[11px] font-semibold uppercase">
                                  DB
                                </span>
                              )}
                              {b.type === 'full_config' && (
                                <span className="px-2 py-0.5 rounded bg-purple-500/10 text-purple-400 border border-purple-500/20 font-sans text-[11px] font-semibold uppercase">
                                  Full
                                </span>
                              )}
                              <span className="text-white font-sans text-sm font-medium">{b.target_name}</span>
                            </div>
                          </td>
                          <td className="px-6 py-4">
                            <span className="inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded text-[11px] font-semibold capitalize bg-surface-800 text-slate-300 border border-surface-700">
                              {b.storage === 'local' ? (
                                <HardDrive className="w-3 h-3 text-slate-400" />
                              ) : (
                                <Cloud className="w-3 h-3 text-cyan-400" />
                              )}
                              {b.storage}
                            </span>
                          </td>
                          <td className="px-6 py-4 text-slate-300 font-semibold">{formatBytes(b.size_bytes)}</td>
                          <td className="px-6 py-4 text-slate-400 font-sans text-xs">
                            {new Date(b.created_at).toLocaleString()}
                          </td>
                          <td className="px-6 py-4">
                            {b.sha256 ? (
                              <button
                                onClick={() => copySHA(b.sha256!)}
                                title="Click to copy full SHA256 checksum"
                                className="inline-flex items-center gap-1.5 px-2 py-1 rounded bg-surface-950 hover:bg-surface-800 border border-surface-800 text-emerald-400 text-[11px] font-mono transition-colors"
                              >
                                {copiedHash === b.sha256 ? (
                                  <>
                                    <Check className="w-3 h-3 text-emerald-400" />
                                    Copied!
                                  </>
                                ) : (
                                  <>
                                    <CheckCircle2 className="w-3 h-3 text-emerald-400" />
                                    {b.sha256.substring(0, 10)}...
                                    <Copy className="w-3 h-3 text-slate-500" />
                                  </>
                                )}
                              </button>
                            ) : (
                              <span className="text-slate-500 font-sans text-xs">Unverified</span>
                            )}
                          </td>
                          <td className="px-6 py-4 text-right">
                            <div className="flex items-center justify-end gap-1.5">
                              <button
                                onClick={() => handleDownloadBackup(b.id, b.file_name)}
                                title="Download .tar.gz"
                                className="p-1.5 rounded-lg bg-surface-800 hover:bg-surface-700 text-slate-300 hover:text-white border border-surface-700 transition-colors"
                              >
                                <Download className="w-3.5 h-3.5" />
                              </button>
                              <button
                                onClick={() => setRestoreModalRecord(b)}
                                title="Restore Snapshot"
                                className="inline-flex items-center gap-1 px-2.5 py-1.5 rounded-lg bg-amber-500/10 hover:bg-amber-500/20 text-amber-400 border border-amber-500/20 text-xs font-semibold font-sans transition-colors"
                              >
                                <RotateCcw className="w-3 h-3" />
                                Restore
                              </button>
                              <button
                                onClick={() => handleDeleteBackup(b.id)}
                                title="Delete Archive"
                                className="p-1.5 rounded-lg bg-surface-800 hover:bg-red-500/20 text-slate-400 hover:text-red-400 border border-surface-700 hover:border-red-500/30 transition-colors"
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
          </div>
        )}

        {/* TAB 2: REMOTE CLOUD STORAGE */}
        {activeTab === 'destinations' && (
          <div className="space-y-4">
            <div className="flex items-center justify-between">
              <div>
                <h3 className="text-base font-bold text-white">Configured Cloud Storage Providers</h3>
                <p className="text-xs text-slate-400 mt-0.5">
                  Synchronize your encrypted backups to AWS S3, Cloudflare R2, Backblaze B2, Wasabi, or MinIO.
                </p>
              </div>
              <button
                onClick={() => setDestModalOpen(true)}
                className="inline-flex items-center gap-1.5 px-3.5 py-2 rounded-lg bg-brand-500 hover:bg-brand-600 text-white text-xs font-semibold transition-colors"
              >
                <Plus className="w-3.5 h-3.5" />
                Add Destination
              </button>
            </div>

            {destinations.length === 0 ? (
              <div className="bg-surface-900 border border-surface-800 rounded-xl p-12 text-center text-slate-400">
                <Cloud className="w-10 h-10 mx-auto text-slate-600 mb-3" />
                <h4 className="text-white font-semibold text-base">No remote storage targets configured</h4>
                <p className="text-xs text-slate-400 max-w-md mx-auto mt-1">
                  Connect your AWS S3, Cloudflare R2, or Backblaze B2 bucket to ensure complete disaster recovery safety outside the local server disk.
                </p>
                <button
                  onClick={() => setDestModalOpen(true)}
                  className="mt-4 inline-flex items-center gap-2 px-4 py-2 rounded-lg bg-brand-500 hover:bg-brand-600 text-white text-xs font-semibold transition-all"
                >
                  <Plus className="w-4 h-4" />
                  Configure First Cloud Target
                </button>
              </div>
            ) : (
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                {destinations.map((d) => (
                  <div key={d.id} className="bg-surface-900 border border-surface-800 rounded-xl p-5 space-y-4 shadow-sm">
                    <div className="flex items-start justify-between">
                      <div className="flex items-center gap-3">
                        <div className="p-2.5 rounded-lg bg-cyan-500/10 text-cyan-400 border border-cyan-500/20">
                          <Cloud className="w-5 h-5" />
                        </div>
                        <div>
                          <div className="flex items-center gap-2">
                            <h4 className="font-bold text-white text-sm">{d.name}</h4>
                            {d.is_default && (
                              <span className="px-2 py-0.5 rounded bg-emerald-500/10 text-emerald-400 border border-emerald-500/20 text-[10px] font-semibold uppercase">
                                Default
                              </span>
                            )}
                          </div>
                          <div className="text-xs text-slate-400 font-mono mt-0.5">Bucket: {d.bucket}</div>
                        </div>
                      </div>

                      <button
                        onClick={() => handleDeleteDestination(d.id, d.name)}
                        className="p-1.5 rounded-lg hover:bg-red-500/20 text-slate-400 hover:text-red-400 transition-colors"
                        title="Remove Target"
                      >
                        <Trash2 className="w-4 h-4" />
                      </button>
                    </div>

                    <div className="grid grid-cols-2 gap-2 text-xs font-mono bg-surface-950 p-3 rounded-lg border border-surface-800">
                      <div>
                        <span className="text-slate-500 block text-[10px] uppercase font-sans">Provider Type</span>
                        <span className="text-slate-300 uppercase">{d.type}</span>
                      </div>
                      <div>
                        <span className="text-slate-500 block text-[10px] uppercase font-sans">Region</span>
                        <span className="text-slate-300">{d.region || 'auto'}</span>
                      </div>
                      <div className="col-span-2 truncate">
                        <span className="text-slate-500 block text-[10px] uppercase font-sans">Endpoint</span>
                        <span className="text-slate-300">{d.endpoint || 'AWS Global S3'}</span>
                      </div>
                    </div>

                    <div className="flex items-center justify-between pt-1 border-t border-surface-800 text-xs">
                      <span className="text-slate-400 font-sans">
                        Updated {new Date(d.updated_at || d.created_at).toLocaleDateString()}
                      </span>
                      <button
                        onClick={async () => {
                          setErrorMsg(null);
                          try {
                            const res = await apiFetch('/api/v1/backups/destinations/test', {
                              method: 'POST',
                              body: JSON.stringify({ id: d.id }),
                            });
                            if (res.success) {
                              setSuccessMsg(`Connection to ${d.name} verified successfully!`);
                            } else {
                              setErrorMsg(`Connection probe failed: ${res.error?.message}`);
                            }
                          } catch (err: any) {
                            setErrorMsg(err.message);
                          }
                        }}
                        className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-surface-800 hover:bg-surface-700 text-white font-semibold transition-colors border border-surface-700"
                      >
                        <CheckCircle2 className="w-3.5 h-3.5 text-emerald-400" />
                        Probe Connection
                      </button>
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>
        )}

        {/* TAB 3: AUTOMATED SCHEDULES */}
        {activeTab === 'schedules' && (
          <div className="space-y-4">
            <div className="flex items-center justify-between">
              <div>
                <h3 className="text-base font-bold text-white">Automated Snapshot Policies</h3>
                <p className="text-xs text-slate-400 mt-0.5">
                  Scheduled cron-driven backup tasks with automatic pruning of older snapshots based on retention limits.
                </p>
              </div>
              <button
                onClick={() => setSchedModalOpen(true)}
                className="inline-flex items-center gap-1.5 px-3.5 py-2 rounded-lg bg-brand-500 hover:bg-brand-600 text-white text-xs font-semibold transition-colors"
              >
                <Plus className="w-3.5 h-3.5" />
                Add Schedule
              </button>
            </div>

            {schedules.length === 0 ? (
              <div className="bg-surface-900 border border-surface-800 rounded-xl p-12 text-center text-slate-400">
                <Calendar className="w-10 h-10 mx-auto text-slate-600 mb-3" />
                <h4 className="text-white font-semibold text-base">No automated schedules active</h4>
                <p className="text-xs text-slate-400 max-w-md mx-auto mt-1">
                  Schedule recurring daily or weekly backups for websites, databases, or full system configuration stacks with retention rules.
                </p>
                <button
                  onClick={() => setSchedModalOpen(true)}
                  className="mt-4 inline-flex items-center gap-2 px-4 py-2 rounded-lg bg-brand-500 hover:bg-brand-600 text-white text-xs font-semibold transition-all"
                >
                  <Plus className="w-4 h-4" />
                  Create First Schedule
                </button>
              </div>
            ) : (
              <div className="bg-surface-900 border border-surface-800 rounded-xl overflow-hidden shadow-sm">
                <table className="w-full text-left text-sm text-slate-300">
                  <thead className="bg-surface-950/60 text-xs uppercase font-semibold text-slate-400 border-b border-surface-800">
                    <tr>
                      <th className="px-6 py-3.5">Schedule Name</th>
                      <th className="px-6 py-3.5">Target & Scope</th>
                      <th className="px-6 py-3.5">Cadence</th>
                      <th className="px-6 py-3.5">Retention Policy</th>
                      <th className="px-6 py-3.5">Next Scheduled Run</th>
                      <th className="px-6 py-3.5">Status</th>
                      <th className="px-6 py-3.5 text-right">Actions</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-surface-800/60 text-xs">
                    {schedules.map((s) => (
                      <tr key={s.id} className="hover:bg-surface-800/30 transition-colors">
                        <td className="px-6 py-4 font-bold text-white">{s.name}</td>
                        <td className="px-6 py-4">
                          <div className="flex items-center gap-2">
                            <span className="px-2 py-0.5 rounded bg-surface-800 text-slate-300 font-mono uppercase text-[10px] border border-surface-700">
                              {s.scope}
                            </span>
                            <span className="text-slate-200 font-medium">{s.target_name}</span>
                          </div>
                        </td>
                        <td className="px-6 py-4 font-semibold text-brand-400 capitalize">{s.frequency}</td>
                        <td className="px-6 py-4 text-slate-300">Keep last {s.retention} snapshots</td>
                        <td className="px-6 py-4 text-slate-400 font-mono">
                          {s.next_run_at ? new Date(s.next_run_at).toLocaleString() : 'Pending trigger'}
                        </td>
                        <td className="px-6 py-4">
                          <span className="inline-flex items-center gap-1 text-emerald-400 font-semibold">
                            <CheckCircle2 className="w-3.5 h-3.5" /> Active
                          </span>
                        </td>
                        <td className="px-6 py-4 text-right">
                          <button
                            onClick={() => handleDeleteSchedule(s.id, s.name)}
                            className="p-1.5 rounded-lg hover:bg-red-500/20 text-slate-400 hover:text-red-400 transition-colors"
                          >
                            <Trash2 className="w-3.5 h-3.5" />
                          </button>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </div>
        )}

        {/* MODAL: CREATE SNAPSHOT */}
        {createModalOpen && (
          <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/75 backdrop-blur-sm p-4 animate-fadeIn">
            <div className="bg-surface-900 border border-surface-800 rounded-xl max-w-md w-full p-6 shadow-2xl space-y-5">
              <div className="flex items-start justify-between">
                <div>
                  <h3 className="text-lg font-bold text-white flex items-center gap-2">
                    <HardDriveDownload className="w-5 h-5 text-brand-400" />
                    Initiate Snapshot
                  </h3>
                  <p className="text-xs text-slate-400 mt-1">
                    Packages data into an atomic, streaming .tar.gz archive with SHA256 integrity.
                  </p>
                </div>
                <button onClick={() => setCreateModalOpen(false)} className="text-slate-400 hover:text-white">
                  <X className="w-4 h-4" />
                </button>
              </div>

              <form onSubmit={handleCreateBackup} className="space-y-4 text-sm">
                <div>
                  <label className="block text-xs font-semibold text-slate-300 mb-1">Backup Scope</label>
                  <div className="grid grid-cols-3 gap-2">
                    {(['website', 'database', 'full_config'] as const).map((type) => (
                      <button
                        type="button"
                        key={type}
                        onClick={() => setCreateType(type)}
                        className={`px-3 py-2 rounded-lg text-xs font-semibold border capitalize transition-all ${
                          createType === type
                            ? 'bg-brand-500/20 border-brand-500 text-brand-400'
                            : 'bg-surface-800 border-surface-700 text-slate-400 hover:text-white'
                        }`}
                      >
                        {type === 'full_config' ? 'Full Stack' : type}
                      </button>
                    ))}
                  </div>
                </div>

                {createType !== 'full_config' && (
                  <div>
                    <label className="block text-xs font-semibold text-slate-300 mb-1">
                      {createType === 'website' ? 'Website Domain' : 'Database Name'}
                    </label>
                    <input
                      type="text"
                      required
                      placeholder={createType === 'website' ? 'e.g. example.com' : 'e.g. app_production_db'}
                      value={createTargetName}
                      onChange={(e) => setCreateTargetName(e.target.value)}
                      className="w-full px-3.5 py-2.5 rounded-lg bg-surface-950 border border-surface-800 text-white font-mono text-sm focus:outline-none focus:border-brand-500"
                    />
                  </div>
                )}

                <div>
                  <label className="block text-xs font-semibold text-slate-300 mb-1">Storage Destination</label>
                  <div className="grid grid-cols-2 gap-2">
                    <button
                      type="button"
                      onClick={() => setCreateStorage('local')}
                      className={`px-3 py-2 rounded-lg text-xs font-semibold border flex items-center justify-center gap-1.5 transition-all ${
                        createStorage === 'local'
                          ? 'bg-brand-500/20 border-brand-500 text-brand-400'
                          : 'bg-surface-800 border-surface-700 text-slate-400 hover:text-white'
                      }`}
                    >
                      <HardDrive className="w-3.5 h-3.5" /> Local Disk
                    </button>
                    <button
                      type="button"
                      onClick={() => setCreateStorage('s3')}
                      className={`px-3 py-2 rounded-lg text-xs font-semibold border flex items-center justify-center gap-1.5 transition-all ${
                        createStorage === 's3'
                          ? 'bg-brand-500/20 border-brand-500 text-brand-400'
                          : 'bg-surface-800 border-surface-700 text-slate-400 hover:text-white'
                      }`}
                    >
                      <Cloud className="w-3.5 h-3.5" /> S3 / R2 Offsite
                    </button>
                  </div>
                </div>

                {createStorage === 's3' && destinations.length > 0 && (
                  <div>
                    <label className="block text-xs font-semibold text-slate-300 mb-1">Target Cloud Bucket</label>
                    <select
                      value={createDestID}
                      onChange={(e) => setCreateDestID(e.target.value)}
                      className="w-full px-3.5 py-2 rounded-lg bg-surface-950 border border-surface-800 text-white text-xs focus:outline-none focus:border-brand-500"
                    >
                      {destinations.map((d) => (
                        <option key={d.id} value={d.id}>
                          {d.name} ({d.bucket})
                        </option>
                      ))}
                    </select>
                  </div>
                )}

                <div>
                  <label className="block text-xs font-semibold text-slate-300 mb-1">Auto-Prune Retention Limit</label>
                  <input
                    type="number"
                    min="1"
                    max="90"
                    value={createRetention}
                    onChange={(e) => setCreateRetention(e.target.value)}
                    className="w-full px-3.5 py-2 rounded-lg bg-surface-950 border border-surface-800 text-white text-xs focus:outline-none focus:border-brand-500"
                  />
                  <span className="text-[11px] text-slate-500 mt-1 block">Keep the latest N snapshots for this target.</span>
                </div>

                <div className="flex items-center justify-end gap-3 pt-3 border-t border-surface-800">
                  <button
                    type="button"
                    onClick={() => setCreateModalOpen(false)}
                    className="px-4 py-2 rounded-lg text-xs font-semibold text-slate-400 hover:text-white"
                  >
                    Cancel
                  </button>
                  <button
                    type="submit"
                    disabled={submittingCreate}
                    className="inline-flex items-center gap-2 px-4 py-2 rounded-lg bg-brand-500 hover:bg-brand-600 text-white text-xs font-semibold transition-colors disabled:opacity-50"
                  >
                    {submittingCreate ? (
                      <>
                        <RefreshCw className="w-3.5 h-3.5 animate-spin" />
                        Packaging & Archiving...
                      </>
                    ) : (
                      'Start Backup Now'
                    )}
                  </button>
                </div>
              </form>
            </div>
          </div>
        )}

        {/* MODAL: RESTORE SNAPSHOT */}
        {restoreModalRecord && (
          <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/75 backdrop-blur-sm p-4 animate-fadeIn">
            <div className="bg-surface-900 border border-surface-800 rounded-xl max-w-md w-full p-6 shadow-2xl space-y-5">
              <div className="flex items-start justify-between">
                <div>
                  <h3 className="text-lg font-bold text-white flex items-center gap-2">
                    <RotateCcw className="w-5 h-5 text-amber-400" />
                    Restore Snapshot
                  </h3>
                  <p className="text-xs text-slate-400 mt-1">
                    Target: <span className="text-white font-mono font-bold">{restoreModalRecord.target_name}</span> ({restoreModalRecord.id})
                  </p>
                </div>
                <button
                  onClick={() => !submittingRestore && setRestoreModalRecord(null)}
                  className="text-slate-400 hover:text-white"
                >
                  <X className="w-4 h-4" />
                </button>
              </div>

              <div className="p-4 rounded-lg bg-amber-500/10 border border-amber-500/20 text-xs text-amber-300 space-y-2">
                <div className="font-bold flex items-center gap-1.5 text-amber-400">
                  <AlertTriangle className="w-4 h-4" />
                  Pre-Restore Rollback Guard
                </div>
                <p className="leading-relaxed">
                  Hostvra will automatically snapshot the current active state before extraction. If an extraction error or power disruption occurs, your original files will be restored automatically.
                </p>
              </div>

              {restoreSuccess ? (
                <div className="p-4 rounded-lg bg-emerald-500/10 border border-emerald-500/20 text-emerald-400 text-sm font-semibold flex items-center gap-2">
                  <CheckCircle2 className="w-5 h-5 flex-shrink-0" />
                  Snapshot restored successfully with verified state!
                </div>
              ) : (
                <div className="flex items-center justify-end gap-3 pt-2">
                  <button
                    type="button"
                    disabled={submittingRestore}
                    onClick={() => setRestoreModalRecord(null)}
                    className="px-4 py-2 rounded-lg text-xs font-semibold text-slate-400 hover:text-white disabled:opacity-50"
                  >
                    Cancel
                  </button>
                  <button
                    type="button"
                    disabled={submittingRestore}
                    onClick={handleConfirmRestore}
                    className="inline-flex items-center gap-2 px-4 py-2 rounded-lg bg-amber-500 hover:bg-amber-600 text-slate-950 text-xs font-bold transition-colors disabled:opacity-50"
                  >
                    {submittingRestore ? (
                      <>
                        <RefreshCw className="w-3.5 h-3.5 animate-spin" />
                        Extracting & Verifying...
                      </>
                    ) : (
                      'Confirm & Execute Restore'
                    )}
                  </button>
                </div>
              )}
            </div>
          </div>
        )}

        {/* MODAL: ADD CLOUD DESTINATION */}
        {destModalOpen && (
          <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/75 backdrop-blur-sm p-4 animate-fadeIn">
            <div className="bg-surface-900 border border-surface-800 rounded-xl max-w-lg w-full p-6 shadow-2xl space-y-5">
              <div className="flex items-start justify-between">
                <div>
                  <h3 className="text-lg font-bold text-white flex items-center gap-2">
                    <Cloud className="w-5 h-5 text-brand-400" />
                    Configure Remote Cloud Storage
                  </h3>
                  <p className="text-xs text-slate-400 mt-1">
                    Connect AWS S3, Cloudflare R2, Backblaze B2, or custom S3-compatible endpoints.
                  </p>
                </div>
                <button onClick={() => setDestModalOpen(false)} className="text-slate-400 hover:text-white">
                  <X className="w-4 h-4" />
                </button>
              </div>

              {testSuccess && (
                <div className="p-3 rounded-lg bg-emerald-500/10 border border-emerald-500/20 text-emerald-400 text-xs flex items-center gap-2">
                  <CheckCircle2 className="w-4 h-4 flex-shrink-0" />
                  <span>{testSuccess}</span>
                </div>
              )}

              {testError && (
                <div className="p-3 rounded-lg bg-red-500/10 border border-red-500/20 text-red-400 text-xs flex items-center gap-2">
                  <AlertTriangle className="w-4 h-4 flex-shrink-0" />
                  <span>{testError}</span>
                </div>
              )}

              <form onSubmit={handleSaveDestination} className="space-y-4 text-xs">
                <div>
                  <label className="block font-semibold text-slate-300 mb-1">Storage Provider</label>
                  <div className="grid grid-cols-4 gap-2">
                    {[
                      { id: 's3', label: 'AWS S3' },
                      { id: 'r2', label: 'Cloudflare R2' },
                      { id: 'b2', label: 'Backblaze B2' },
                      { id: 'sftp', label: 'SFTP' },
                    ].map((p) => (
                      <button
                        type="button"
                        key={p.id}
                        onClick={() => {
                          setDestProvider(p.id as any);
                          if (p.id === 'r2') setDestRegion('auto');
                          if (p.id === 's3') setDestRegion('us-east-1');
                        }}
                        className={`px-3 py-2 rounded-lg text-xs font-semibold border transition-all ${
                          destProvider === p.id
                            ? 'bg-brand-500/20 border-brand-500 text-brand-400'
                            : 'bg-surface-800 border-surface-700 text-slate-400 hover:text-white'
                        }`}
                      >
                        {p.label}
                      </button>
                    ))}
                  </div>
                </div>

                <div className="grid grid-cols-2 gap-3">
                  <div>
                    <label className="block font-semibold text-slate-300 mb-1">Friendly Label</label>
                    <input
                      type="text"
                      required
                      placeholder="e.g. AWS Primary Offsite"
                      value={destName}
                      onChange={(e) => setDestName(e.target.value)}
                      className="w-full px-3 py-2 rounded-lg bg-surface-950 border border-surface-800 text-white font-sans focus:outline-none focus:border-brand-500"
                    />
                  </div>
                  <div>
                    <label className="block font-semibold text-slate-300 mb-1">Bucket Name</label>
                    <input
                      type="text"
                      required
                      placeholder="e.g. my-backup-bucket"
                      value={destBucket}
                      onChange={(e) => setDestBucket(e.target.value)}
                      className="w-full px-3 py-2 rounded-lg bg-surface-950 border border-surface-800 text-white font-mono focus:outline-none focus:border-brand-500"
                    />
                  </div>
                </div>

                <div className="grid grid-cols-2 gap-3">
                  <div>
                    <label className="block font-semibold text-slate-300 mb-1">Custom Endpoint URL</label>
                    <input
                      type="text"
                      placeholder={
                        destProvider === 'r2'
                          ? 'https://<account_id>.r2.cloudflarestorage.com'
                          : 'Leave blank for default AWS'
                      }
                      value={destEndpoint}
                      onChange={(e) => setDestEndpoint(e.target.value)}
                      className="w-full px-3 py-2 rounded-lg bg-surface-950 border border-surface-800 text-white font-mono focus:outline-none focus:border-brand-500"
                    />
                  </div>
                  <div>
                    <label className="block font-semibold text-slate-300 mb-1">Region</label>
                    <input
                      type="text"
                      placeholder="us-east-1 or auto"
                      value={destRegion}
                      onChange={(e) => setDestRegion(e.target.value)}
                      className="w-full px-3 py-2 rounded-lg bg-surface-950 border border-surface-800 text-white font-mono focus:outline-none focus:border-brand-500"
                    />
                  </div>
                </div>

                <div className="grid grid-cols-2 gap-3">
                  <div>
                    <label className="block font-semibold text-slate-300 mb-1">Access Key ID</label>
                    <input
                      type="text"
                      required
                      placeholder="AKIA..."
                      value={destAccessKey}
                      onChange={(e) => setDestAccessKey(e.target.value)}
                      className="w-full px-3 py-2 rounded-lg bg-surface-950 border border-surface-800 text-white font-mono focus:outline-none focus:border-brand-500"
                    />
                  </div>
                  <div>
                    <label className="block font-semibold text-slate-300 mb-1">Secret Access Key</label>
                    <input
                      type="password"
                      required
                      placeholder="••••••••••••••••"
                      value={destSecretKey}
                      onChange={(e) => setDestSecretKey(e.target.value)}
                      className="w-full px-3 py-2 rounded-lg bg-surface-950 border border-surface-800 text-white font-mono focus:outline-none focus:border-brand-500"
                    />
                  </div>
                </div>

                <div className="grid grid-cols-2 gap-3 items-center">
                  <div>
                    <label className="block font-semibold text-slate-300 mb-1">Folder Prefix</label>
                    <input
                      type="text"
                      placeholder="hostvra-backups"
                      value={destPrefix}
                      onChange={(e) => setDestPrefix(e.target.value)}
                      className="w-full px-3 py-2 rounded-lg bg-surface-950 border border-surface-800 text-white font-mono focus:outline-none focus:border-brand-500"
                    />
                  </div>

                  <div className="pt-4">
                    <label className="flex items-center gap-2 cursor-pointer">
                      <input
                        type="checkbox"
                        checked={destIsDefault}
                        onChange={(e) => setDestIsDefault(e.target.checked)}
                        className="rounded bg-surface-950 border-surface-800 text-brand-500 focus:ring-0"
                      />
                      <span className="font-semibold text-slate-300">Set as Default Target</span>
                    </label>
                  </div>
                </div>

                <div className="flex items-center justify-between pt-3 border-t border-surface-800">
                  <button
                    type="button"
                    disabled={testingDest || !destBucket || !destAccessKey}
                    onClick={handleTestDestination}
                    className="inline-flex items-center gap-1.5 px-3 py-2 rounded-lg bg-surface-800 hover:bg-surface-700 text-slate-300 hover:text-white font-semibold transition-colors disabled:opacity-50"
                  >
                    <CheckCircle2 className="w-3.5 h-3.5 text-emerald-400" />
                    {testingDest ? 'Probing...' : 'Test Connection'}
                  </button>

                  <div className="flex items-center gap-2">
                    <button
                      type="button"
                      onClick={() => setDestModalOpen(false)}
                      className="px-4 py-2 rounded-lg font-semibold text-slate-400 hover:text-white"
                    >
                      Cancel
                    </button>
                    <button
                      type="submit"
                      disabled={submittingDest}
                      className="inline-flex items-center gap-2 px-4 py-2 rounded-lg bg-brand-500 hover:bg-brand-600 text-white font-semibold transition-colors disabled:opacity-50"
                    >
                      {submittingDest ? 'Saving...' : 'Save Target'}
                    </button>
                  </div>
                </div>
              </form>
            </div>
          </div>
        )}

        {/* MODAL: NEW SCHEDULE */}
        {schedModalOpen && (
          <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/75 backdrop-blur-sm p-4 animate-fadeIn">
            <div className="bg-surface-900 border border-surface-800 rounded-xl max-w-md w-full p-6 shadow-2xl space-y-5">
              <div className="flex items-start justify-between">
                <div>
                  <h3 className="text-lg font-bold text-white flex items-center gap-2">
                    <Calendar className="w-5 h-5 text-brand-400" />
                    Create Backup Schedule
                  </h3>
                  <p className="text-xs text-slate-400 mt-1">Automate periodic snapshots with retention limits.</p>
                </div>
                <button onClick={() => setSchedModalOpen(false)} className="text-slate-400 hover:text-white">
                  <X className="w-4 h-4" />
                </button>
              </div>

              <form onSubmit={handleSaveSchedule} className="space-y-4 text-xs">
                <div>
                  <label className="block font-semibold text-slate-300 mb-1">Schedule Name</label>
                  <input
                    type="text"
                    required
                    placeholder="e.g. Nightly Production Website Backup"
                    value={schedName}
                    onChange={(e) => setSchedName(e.target.value)}
                    className="w-full px-3.5 py-2 rounded-lg bg-surface-950 border border-surface-800 text-white text-xs focus:outline-none focus:border-brand-500"
                  />
                </div>

                <div>
                  <label className="block font-semibold text-slate-300 mb-1">Scope</label>
                  <div className="grid grid-cols-3 gap-2">
                    {(['website', 'database', 'full_config'] as const).map((type) => (
                      <button
                        type="button"
                        key={type}
                        onClick={() => setSchedScope(type)}
                        className={`px-3 py-2 rounded-lg text-xs font-semibold border capitalize transition-all ${
                          schedScope === type
                            ? 'bg-brand-500/20 border-brand-500 text-brand-400'
                            : 'bg-surface-800 border-surface-700 text-slate-400 hover:text-white'
                        }`}
                      >
                        {type === 'full_config' ? 'Full Stack' : type}
                      </button>
                    ))}
                  </div>
                </div>

                {schedScope !== 'full_config' && (
                  <div>
                    <label className="block font-semibold text-slate-300 mb-1">
                      {schedScope === 'website' ? 'Domain Name' : 'Database Name'}
                    </label>
                    <input
                      type="text"
                      required
                      placeholder="e.g. example.com"
                      value={schedTargetName}
                      onChange={(e) => setSchedTargetName(e.target.value)}
                      className="w-full px-3.5 py-2 rounded-lg bg-surface-950 border border-surface-800 text-white text-xs focus:outline-none focus:border-brand-500"
                    />
                  </div>
                )}

                <div className="grid grid-cols-2 gap-3">
                  <div>
                    <label className="block font-semibold text-slate-300 mb-1">Cadence / Frequency</label>
                    <select
                      value={schedFreq}
                      onChange={(e) => setSchedFreq(e.target.value as any)}
                      className="w-full px-3.5 py-2 rounded-lg bg-surface-950 border border-surface-800 text-white text-xs focus:outline-none focus:border-brand-500"
                    >
                      <option value="daily">Daily at 02:00 UTC</option>
                      <option value="weekly">Weekly on Sunday</option>
                      <option value="monthly">Monthly on 1st</option>
                    </select>
                  </div>

                  <div>
                    <label className="block font-semibold text-slate-300 mb-1">Retention Count</label>
                    <input
                      type="number"
                      min="1"
                      max="60"
                      value={schedRetention}
                      onChange={(e) => setSchedRetention(e.target.value)}
                      className="w-full px-3.5 py-2 rounded-lg bg-surface-950 border border-surface-800 text-white text-xs focus:outline-none focus:border-brand-500"
                    />
                  </div>
                </div>

                {destinations.length > 0 && (
                  <div>
                    <label className="block font-semibold text-slate-300 mb-1">Remote Target</label>
                    <select
                      value={schedDestID}
                      onChange={(e) => setSchedDestID(e.target.value)}
                      className="w-full px-3.5 py-2 rounded-lg bg-surface-950 border border-surface-800 text-white text-xs focus:outline-none focus:border-brand-500"
                    >
                      <option value="">Local Disk Only</option>
                      {destinations.map((d) => (
                        <option key={d.id} value={d.id}>
                          {d.name} ({d.bucket})
                        </option>
                      ))}
                    </select>
                  </div>
                )}

                <div className="flex items-center justify-end gap-3 pt-3 border-t border-surface-800">
                  <button
                    type="button"
                    onClick={() => setSchedModalOpen(false)}
                    className="px-4 py-2 rounded-lg font-semibold text-slate-400 hover:text-white"
                  >
                    Cancel
                  </button>
                  <button
                    type="submit"
                    disabled={submittingSched}
                    className="inline-flex items-center gap-2 px-4 py-2 rounded-lg bg-brand-500 hover:bg-brand-600 text-white font-semibold transition-colors disabled:opacity-50"
                  >
                    {submittingSched ? 'Activating...' : 'Activate Schedule'}
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
