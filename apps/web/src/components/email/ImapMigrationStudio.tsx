'use client';

import React, { useState, useEffect, useCallback } from 'react';
import {
  Inbox,
  RefreshCw,
  Plus,
  Play,
  XCircle,
  CheckCircle2,
  AlertTriangle,
  Server,
  Lock,
  Layers,
  ArrowRight,
  ShieldCheck,
  Activity,
  Trash2,
  Eye,
  EyeOff,
  Clock,
  Sparkles,
  Info,
  Check,
} from 'lucide-react';
import {
  EmailMailbox,
  IMAPMigration,
  SourceConnectionOptions,
  IMAPTestResult,
  listIMAPMigrations,
  startIMAPMigration,
  cancelIMAPMigration,
  deleteIMAPMigration,
  testSourceIMAP,
} from '@/lib/api';

interface ImapMigrationStudioProps {
  mailboxes: EmailMailbox[];
  prefilledMailboxId?: string;
  showToast?: (msg: string) => void;
}

interface ProviderPreset {
  id: string;
  name: string;
  host: string;
  port: number;
  useSSL: boolean;
  notes: string;
}

const SOURCE_PRESETS: ProviderPreset[] = [
  {
    id: 'cpanel',
    name: 'cPanel / DirectAdmin',
    host: 'mail.your-old-domain.com',
    port: 993,
    useSSL: true,
    notes: 'Standard IMAP SSL on port 993 with full email username.',
  },
  {
    id: 'gmail',
    name: 'Google Workspace / Gmail',
    host: 'imap.gmail.com',
    port: 993,
    useSSL: true,
    notes: 'Requires Google App Password if 2-Factor Authentication is enabled.',
  },
  {
    id: 'outlook365',
    name: 'Microsoft 365 / Outlook',
    host: 'outlook.office365.com',
    port: 993,
    useSSL: true,
    notes: 'IMAP enabled in Microsoft 365 Admin Center with app password.',
  },
  {
    id: 'generic',
    name: 'Generic IMAP Server',
    host: '',
    port: 993,
    useSSL: true,
    notes: 'Any external IMAP server supporting SSL/TLS or STARTTLS.',
  },
];

function formatBytes(bytes?: number): string {
  if (!bytes || bytes <= 0) return '0 B';
  const k = 1024;
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i];
}

export function ImapMigrationStudio({ mailboxes, prefilledMailboxId, showToast }: ImapMigrationStudioProps) {
  const [migrations, setMigrations] = useState<IMAPMigration[]>([]);
  const [loading, setLoading] = useState(true);
  const [showModal, setShowModal] = useState(false);
  const [starting, setStarting] = useState(false);
  const [testing, setTesting] = useState(false);
  const [testResult, setTestResult] = useState<IMAPTestResult | null>(null);
  const [showPassword, setShowPassword] = useState(false);

  // Form Fields
  const [selectedPreset, setSelectedPreset] = useState<string>('cpanel');
  const [destinationMailboxId, setDestinationMailboxId] = useState<string>(
    prefilledMailboxId || mailboxes[0]?.id || ''
  );
  const [sourceHost, setSourceHost] = useState<string>('mail.your-old-domain.com');
  const [sourcePort, setSourcePort] = useState<number>(993);
  const [sourceUseSSL, setSourceUseSSL] = useState<boolean>(true);
  const [sourceSkipSSLVerify, setSourceSkipSSLVerify] = useState<boolean>(false);
  const [sourceUsername, setSourceUsername] = useState<string>('');
  const [sourcePassword, setSourcePassword] = useState<string>('');
  const [excludeFolders, setExcludeFolders] = useState<string>('Trash, Spam, Junk');
  const [showAdvanced, setShowAdvanced] = useState(false);

  const toast = useCallback((msg: string) => {
    if (showToast) {
      showToast(msg);
    } else {
      alert(msg);
    }
  }, [showToast]);

  const fetchMigrations = useCallback(async () => {
    try {
      const res = await listIMAPMigrations();
      if (res.data) {
        setMigrations(res.data);
      }
    } catch (err: any) {
      console.error('Failed to list migrations', err);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchMigrations();
  }, [fetchMigrations]);

  // Polling for active running migrations
  useEffect(() => {
    const hasActive = migrations.some((m) => m.status === 'running' || m.status === 'pending');
    if (!hasActive) return;

    const interval = setInterval(() => {
      fetchMigrations();
    }, 3000);

    return () => clearInterval(interval);
  }, [migrations, fetchMigrations]);

  // Update destination if prefill changes
  useEffect(() => {
    if (prefilledMailboxId) {
      setDestinationMailboxId(prefilledMailboxId);
      setShowModal(true);
    }
  }, [prefilledMailboxId]);

  const applyPreset = (presetId: string) => {
    setSelectedPreset(presetId);
    const p = SOURCE_PRESETS.find((x) => x.id === presetId);
    if (!p) return;
    if (presetId !== 'generic') {
      setSourceHost(p.host);
      setSourcePort(p.port);
      setSourceUseSSL(p.useSSL);
    }
  };

  const handleTestSource = async () => {
    if (!sourceHost.trim() || !sourceUsername.trim() || !sourcePassword) {
      toast('Please enter source host, username, and password before testing');
      return;
    }

    setTesting(true);
    setTestResult(null);
    try {
      const opts: SourceConnectionOptions = {
        host: sourceHost.trim(),
        port: Number(sourcePort),
        username: sourceUsername.trim(),
        password: sourcePassword,
        use_ssl: sourceUseSSL,
        skip_ssl_verify: sourceSkipSSLVerify,
      };
      const res = await testSourceIMAP(opts);
      if (res.data) {
        setTestResult(res.data);
        if (res.data.success) {
          toast('Source IMAP connection successful');
        } else {
          toast(res.data.error || 'Connection failed');
        }
      }
    } catch (err: any) {
      toast(err.message || 'IMAP test probe error');
    } finally {
      setTesting(false);
    }
  };

  const handleStartMigration = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!destinationMailboxId) {
      toast('Please select a destination mailbox');
      return;
    }
    if (!sourceHost.trim() || !sourceUsername.trim() || !sourcePassword) {
      toast('Please fill in source host, username, and password');
      return;
    }

    setStarting(true);
    try {
      const excludes = excludeFolders
        .split(',')
        .map((s) => s.trim())
        .filter(Boolean);

      const payload = {
        destination_mailbox_id: destinationMailboxId,
        source: {
          host: sourceHost.trim(),
          port: Number(sourcePort),
          username: sourceUsername.trim(),
          password: sourcePassword,
          use_ssl: sourceUseSSL,
          skip_ssl_verify: sourceSkipSSLVerify,
          exclude_folders: excludes,
          batch_size: 50,
          max_concurrency: 2,
        },
      };

      const res = await startIMAPMigration(payload);
      if (res.data) {
        toast(`Migration started for ${res.data.destination_email}`);
        setShowModal(false);
        fetchMigrations();
      }
    } catch (err: any) {
      toast(err.message || 'Failed to start IMAP migration');
    } finally {
      setStarting(false);
    }
  };

  const handleCancel = async (mig: IMAPMigration) => {
    if (!confirm(`Cancel migration into ${mig.destination_email}? Existing synced messages will be preserved.`)) return;
    try {
      await cancelIMAPMigration(mig.id);
      toast('Migration cancelled');
      fetchMigrations();
    } catch (err: any) {
      toast(err.message || 'Failed to cancel migration');
    }
  };

  const handleDelete = async (mig: IMAPMigration) => {
    if (!confirm('Remove migration record from history?')) return;
    try {
      await deleteIMAPMigration(mig.id);
      toast('Migration record deleted');
      setMigrations((prev) => prev.filter((m) => m.id !== mig.id));
    } catch (err: any) {
      toast(err.message || 'Failed to delete migration');
    }
  };

  return (
    <div className="space-y-6">
      {/* Header Banner */}
      <div className="p-6 rounded-2xl bg-gradient-to-r from-blue-900/40 via-indigo-900/30 to-purple-900/30 border border-indigo-500/20 backdrop-blur-md shadow-xs">
        <div className="flex flex-col md:flex-row md:items-center justify-between gap-4">
          <div className="space-y-1.5">
            <div className="inline-flex items-center gap-2 px-2.5 py-0.5 rounded-full text-xs font-semibold bg-indigo-500/20 text-indigo-300 border border-indigo-500/30">
              <ShieldCheck className="w-3.5 h-3.5 text-indigo-400" />
              <span>Non-Destructive Live Transfer Engine</span>
            </div>
            <h2 className="text-xl font-bold text-white tracking-tight">Live IMAP Mailbox Migration Studio</h2>
            <p className="text-xs text-slate-300 max-w-2xl">
              Seamlessly migrate all emails, subfolders, flags, and attachments from cPanel, Plesk, Gmail, Google Workspace, or any IMAP server. Non-destructive with zero data deletion from your source mailbox.
            </p>
          </div>

          <div className="flex items-center gap-3">
            <button
              onClick={fetchMigrations}
              disabled={loading}
              className="p-2.5 rounded-xl bg-white/10 hover:bg-white/15 text-white border border-white/10 transition shadow-xs disabled:opacity-50"
              title="Refresh migrations"
            >
              <RefreshCw className={`w-4 h-4 ${loading ? 'animate-spin' : ''}`} />
            </button>
            <button
              onClick={() => {
                setTestResult(null);
                setShowModal(true);
              }}
              className="inline-flex items-center gap-2 px-4 py-2.5 rounded-xl bg-indigo-600 hover:bg-indigo-500 text-white font-semibold text-xs transition shadow-sm"
            >
              <Plus className="w-4 h-4" />
              <span>Start Mailbox Migration</span>
            </button>
          </div>
        </div>

        {/* Zero Deletion Guarantee Pill */}
        <div className="mt-5 pt-3 border-t border-white/10 flex items-center gap-2 text-xs text-slate-300">
          <ShieldCheck className="w-4 h-4 text-emerald-400 shrink-0" />
          <span>
            <strong>Zero Deletion Guarantee:</strong> The Hostvra migration engine only executes read-only IMAP operations (<code className="font-mono text-emerald-300">FETCH</code>). It will never execute <code className="font-mono text-emerald-300">EXPUNGE</code> or delete source messages under any circumstances.
          </span>
        </div>
      </div>

      {/* Migration Tasks List */}
      {loading ? (
        <div className="p-12 text-center rounded-2xl bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-800">
          <RefreshCw className="w-8 h-8 text-indigo-500 animate-spin mx-auto mb-3" />
          <p className="text-sm font-medium text-slate-600 dark:text-slate-400">Loading migration tasks...</p>
        </div>
      ) : migrations.length === 0 ? (
        <div className="p-12 text-center rounded-2xl bg-white dark:bg-surface-900 border border-dashed border-slate-300 dark:border-surface-700 space-y-4">
          <div className="w-14 h-14 mx-auto rounded-2xl bg-indigo-500/10 flex items-center justify-center text-indigo-500">
            <Inbox className="w-7 h-7" />
          </div>
          <div className="space-y-1">
            <h3 className="text-base font-bold text-slate-900 dark:text-white">No Mailbox Migrations Yet</h3>
            <p className="text-xs text-slate-500 max-w-md mx-auto">
              Ready to move mailboxes from an existing host or email provider? Launch a background migration to copy all historical emails into your Hostvra mailboxes with duplicate suppression.
            </p>
          </div>
          <button
            onClick={() => setShowModal(true)}
            className="inline-flex items-center gap-2 px-4 py-2 rounded-xl bg-indigo-600 hover:bg-indigo-500 text-white font-semibold text-xs shadow-xs transition"
          >
            <Plus className="w-4 h-4" />
            <span>Migrate First Mailbox</span>
          </button>
        </div>
      ) : (
        <div className="space-y-4">
          {migrations.map((mig) => {
            const isRunning = mig.status === 'running' || mig.status === 'pending';
            const isCompleted = mig.status === 'completed';
            const isFailed = mig.status === 'failed';
            const isCancelled = mig.status === 'cancelled';

            const progressPercent =
              mig.total_messages > 0
                ? Math.min(100, Math.round((mig.synced_messages / mig.total_messages) * 100))
                : isCompleted
                ? 100
                : 0;

            return (
              <div
                key={mig.id}
                className="p-5 rounded-2xl bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-800 space-y-4 shadow-xs"
              >
                {/* Header Row */}
                <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3">
                  <div className="flex items-center gap-3">
                    <div
                      className={`w-10 h-10 rounded-xl flex items-center justify-center ${
                        isRunning
                          ? 'bg-blue-500/10 text-blue-500'
                          : isCompleted
                          ? 'bg-emerald-500/10 text-emerald-500'
                          : isFailed
                          ? 'bg-rose-500/10 text-rose-500'
                          : 'bg-slate-500/10 text-slate-500'
                      }`}
                    >
                      {isRunning ? (
                        <RefreshCw className="w-5 h-5 animate-spin" />
                      ) : isCompleted ? (
                        <CheckCircle2 className="w-5 h-5" />
                      ) : isFailed ? (
                        <XCircle className="w-5 h-5" />
                      ) : (
                        <Clock className="w-5 h-5" />
                      )}
                    </div>

                    <div>
                      <div className="flex items-center gap-2">
                        <h4 className="text-sm font-bold text-slate-900 dark:text-white">
                          {mig.source_username} &rarr; {mig.destination_email}
                        </h4>
                        <span
                          className={`px-2 py-0.5 rounded-full text-[10px] font-bold uppercase tracking-wider ${
                            isRunning
                              ? 'bg-blue-500/10 text-blue-500 border border-blue-500/20'
                              : isCompleted
                              ? 'bg-emerald-500/10 text-emerald-500 border border-emerald-500/20'
                              : isFailed
                              ? 'bg-rose-500/10 text-rose-500 border border-rose-500/20'
                              : 'bg-slate-500/10 text-slate-500 border border-slate-500/20'
                          }`}
                        >
                          {mig.status}
                        </span>
                      </div>
                      <p className="text-xs text-slate-500 font-mono mt-0.5">
                        Source: {mig.source_host}:{mig.source_port} ({mig.source_use_ssl ? 'SSL/TLS' : 'Plain'})
                      </p>
                    </div>
                  </div>

                  <div className="flex items-center gap-2">
                    {isRunning && (
                      <button
                        onClick={() => handleCancel(mig)}
                        className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-xl bg-rose-50 dark:bg-rose-950/40 hover:bg-rose-100 dark:hover:bg-rose-900/40 text-rose-600 dark:text-rose-400 font-semibold text-xs transition border border-rose-200 dark:border-rose-800"
                      >
                        <XCircle className="w-3.5 h-3.5" />
                        <span>Cancel Migration</span>
                      </button>
                    )}

                    {!isRunning && (
                      <button
                        onClick={() => handleDelete(mig)}
                        className="p-2 rounded-xl text-slate-400 hover:text-rose-500 hover:bg-rose-50 dark:hover:bg-rose-950/30 transition"
                        title="Delete Record"
                      >
                        <Trash2 className="w-4 h-4" />
                      </button>
                    )}
                  </div>
                </div>

                {/* Progress Bar & Current Status */}
                <div className="space-y-2">
                  <div className="flex justify-between items-center text-xs">
                    <span className="font-medium text-slate-600 dark:text-slate-400 flex items-center gap-1.5">
                      {isRunning && <Activity className="w-3.5 h-3.5 text-blue-500 animate-pulse" />}
                      <span>
                        {isRunning
                          ? `Syncing folder: ${mig.current_folder || 'Discovering...'}`
                          : isCompleted
                          ? 'Migration Completed Successfully'
                          : isFailed
                          ? `Failed: ${mig.error_message || 'Unknown error'}`
                          : 'Migration Cancelled'}
                      </span>
                    </span>
                    <span className="font-mono font-bold text-slate-900 dark:text-white">
                      {mig.synced_messages} / {mig.total_messages > 0 ? mig.total_messages : '...'} msgs ({progressPercent}%)
                    </span>
                  </div>

                  <div className="w-full h-2.5 rounded-full bg-slate-100 dark:bg-surface-950 overflow-hidden border border-slate-200/50 dark:border-surface-800">
                    <div
                      className={`h-full transition-all duration-500 ${
                        isCompleted
                          ? 'bg-emerald-500'
                          : isFailed
                          ? 'bg-rose-500'
                          : isCancelled
                          ? 'bg-slate-400'
                          : 'bg-indigo-600 animate-pulse'
                      }`}
                      style={{ width: `${progressPercent}%` }}
                    />
                  </div>
                </div>

                {/* Metrics Breakdown */}
                <div className="grid grid-cols-2 sm:grid-cols-4 gap-2 text-xs font-mono">
                  <div className="p-2 rounded-lg bg-slate-50 dark:bg-surface-950 border border-slate-200/80 dark:border-surface-800">
                    <span className="text-[10px] font-sans font-semibold text-slate-400 uppercase block">Data Copied</span>
                    <span className="text-slate-800 dark:text-slate-200 font-bold">{formatBytes(mig.bytes_transferred)}</span>
                  </div>
                  <div className="p-2 rounded-lg bg-slate-50 dark:bg-surface-950 border border-slate-200/80 dark:border-surface-800">
                    <span className="text-[10px] font-sans font-semibold text-slate-400 uppercase block">Duplicates Skipped</span>
                    <span className="text-slate-800 dark:text-slate-200 font-bold">{mig.skipped_duplicates}</span>
                  </div>
                  <div className="p-2 rounded-lg bg-slate-50 dark:bg-surface-950 border border-slate-200/80 dark:border-surface-800">
                    <span className="text-[10px] font-sans font-semibold text-slate-400 uppercase block">Total Folders</span>
                    <span className="text-slate-800 dark:text-slate-200 font-bold">{mig.total_folders} folders</span>
                  </div>
                  <div className="p-2 rounded-lg bg-slate-50 dark:bg-surface-950 border border-slate-200/80 dark:border-surface-800">
                    <span className="text-[10px] font-sans font-semibold text-slate-400 uppercase block">Started At</span>
                    <span className="text-slate-800 dark:text-slate-200 font-bold">
                      {mig.started_at ? new Date(mig.started_at).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }) : 'Pending'}
                    </span>
                  </div>
                </div>
              </div>
            );
          })}
        </div>
      )}

      {/* ==================================================================== */}
      {/* START MIGRATION MODAL                                                */}
      {/* ==================================================================== */}
      {showModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/60 backdrop-blur-xs">
          <div className="w-full max-w-2xl bg-white dark:bg-surface-900 rounded-2xl border border-slate-200 dark:border-surface-800 shadow-2xl overflow-hidden flex flex-col max-h-[90vh]">
            <div className="px-6 py-4 border-b border-slate-200 dark:border-surface-800 flex items-center justify-between">
              <div>
                <h3 className="text-base font-bold text-slate-900 dark:text-white">
                  Live IMAP Mailbox Migration Wizard
                </h3>
                <p className="text-xs text-slate-500 mt-0.5">
                  Transfer emails from an external server with automatic folder mapping and duplicate deduplication.
                </p>
              </div>
              <button
                onClick={() => setShowModal(false)}
                className="p-1 rounded-lg text-slate-400 hover:text-slate-600 dark:hover:text-white"
              >
                &times;
              </button>
            </div>

            <form onSubmit={handleStartMigration} className="p-6 space-y-5 overflow-y-auto">
              {/* Preset Selector */}
              <div className="space-y-2">
                <label className="text-xs font-bold text-slate-700 dark:text-slate-300 uppercase tracking-wider block">
                  Select Source Server Type
                </label>
                <div className="grid grid-cols-2 sm:grid-cols-4 gap-2">
                  {SOURCE_PRESETS.map((p) => {
                    const isSelected = selectedPreset === p.id;
                    return (
                      <button
                        key={p.id}
                        type="button"
                        onClick={() => applyPreset(p.id)}
                        className={`p-3 rounded-xl border text-left transition flex flex-col justify-between ${
                          isSelected
                            ? 'bg-indigo-50/50 dark:bg-indigo-950/30 border-indigo-500 ring-2 ring-indigo-500/20'
                            : 'bg-slate-50 dark:bg-surface-950 border-slate-200 dark:border-surface-800 hover:border-slate-300 dark:hover:border-surface-700'
                        }`}
                      >
                        <div className="flex items-center justify-between w-full">
                          <span className="text-xs font-bold text-slate-900 dark:text-white">{p.name}</span>
                          {isSelected && <Check className="w-3.5 h-3.5 text-indigo-500" />}
                        </div>
                        <span className="text-[10px] text-slate-500 mt-1 line-clamp-1">{p.notes}</span>
                      </button>
                    );
                  })}
                </div>
              </div>

              {/* Destination Mailbox Picker */}
              <div className="space-y-1.5">
                <label className="text-xs font-semibold text-slate-700 dark:text-slate-300">
                  Target Destination Mailbox (on this server)
                </label>
                <select
                  required
                  value={destinationMailboxId}
                  onChange={(e) => setDestinationMailboxId(e.target.value)}
                  className="w-full px-3 py-2 bg-slate-50 dark:bg-surface-950 border border-slate-200 dark:border-surface-800 rounded-xl text-xs text-slate-900 dark:text-white focus:outline-none focus:border-indigo-500"
                >
                  <option value="">Select Target Mailbox...</option>
                  {mailboxes.map((mb) => (
                    <option key={mb.id} value={mb.id}>
                      {mb.email} ({mb.name || 'Mailbox'})
                    </option>
                  ))}
                </select>
              </div>

              {/* Source IMAP Details */}
              <div className="p-4 rounded-xl bg-slate-50 dark:bg-surface-950 border border-slate-200 dark:border-surface-800 space-y-4">
                <h4 className="text-xs font-bold uppercase tracking-wider text-slate-700 dark:text-slate-300 flex items-center gap-1.5">
                  <Server className="w-3.5 h-3.5 text-indigo-500" />
                  <span>Source IMAP Server Details</span>
                </h4>

                <div className="grid grid-cols-1 sm:grid-cols-3 gap-3">
                  <div className="sm:col-span-2 space-y-1">
                    <label className="text-xs font-medium text-slate-600 dark:text-slate-400">IMAP Server Host</label>
                    <input
                      type="text"
                      required
                      value={sourceHost}
                      onChange={(e) => setSourceHost(e.target.value)}
                      placeholder="e.g. mail.previoushost.com"
                      className="w-full px-3 py-2 bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-800 rounded-xl text-xs text-slate-900 dark:text-white font-mono focus:outline-none focus:border-indigo-500"
                    />
                  </div>

                  <div className="space-y-1">
                    <label className="text-xs font-medium text-slate-600 dark:text-slate-400">Port</label>
                    <input
                      type="number"
                      required
                      value={sourcePort}
                      onChange={(e) => setSourcePort(Number(e.target.value))}
                      className="w-full px-3 py-2 bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-800 rounded-xl text-xs text-slate-900 dark:text-white font-mono focus:outline-none focus:border-indigo-500"
                    />
                  </div>
                </div>

                <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
                  <div className="space-y-1">
                    <label className="text-xs font-medium text-slate-600 dark:text-slate-400">Source Email / Username</label>
                    <input
                      type="text"
                      required
                      value={sourceUsername}
                      onChange={(e) => setSourceUsername(e.target.value)}
                      placeholder="user@oldhost.com"
                      className="w-full px-3 py-2 bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-800 rounded-xl text-xs text-slate-900 dark:text-white font-mono focus:outline-none focus:border-indigo-500"
                    />
                  </div>

                  <div className="space-y-1">
                    <div className="flex items-center justify-between">
                      <label className="text-xs font-medium text-slate-600 dark:text-slate-400">Source Password</label>
                      <button
                        type="button"
                        onClick={() => setShowPassword(!showPassword)}
                        className="text-[10px] text-slate-400 hover:text-slate-600 dark:hover:text-slate-200 inline-flex items-center gap-1"
                      >
                        {showPassword ? <EyeOff className="w-3 h-3" /> : <Eye className="w-3 h-3" />}
                        <span>{showPassword ? 'Hide' : 'Reveal'}</span>
                      </button>
                    </div>
                    <input
                      type={showPassword ? 'text' : 'password'}
                      required
                      value={sourcePassword}
                      onChange={(e) => setSourcePassword(e.target.value)}
                      placeholder="Enter mailbox password or app password"
                      className="w-full px-3 py-2 bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-800 rounded-xl text-xs text-slate-900 dark:text-white font-mono focus:outline-none focus:border-indigo-500"
                    />
                  </div>
                </div>

                <div className="flex flex-wrap items-center justify-between gap-3 pt-2">
                  <div className="flex items-center gap-4">
                    <label className="inline-flex items-center gap-2 cursor-pointer text-xs text-slate-700 dark:text-slate-300">
                      <input
                        type="checkbox"
                        checked={sourceUseSSL}
                        onChange={(e) => {
                          setSourceUseSSL(e.target.checked);
                          if (e.target.checked && sourcePort === 143) setSourcePort(993);
                          if (!e.target.checked && sourcePort === 993) setSourcePort(143);
                        }}
                        className="rounded text-indigo-600 focus:ring-0"
                      />
                      <span>Use SSL/TLS (Port 993)</span>
                    </label>

                    <label className="inline-flex items-center gap-2 cursor-pointer text-xs text-slate-700 dark:text-slate-300">
                      <input
                        type="checkbox"
                        checked={sourceSkipSSLVerify}
                        onChange={(e) => setSourceSkipSSLVerify(e.target.checked)}
                        className="rounded text-indigo-600 focus:ring-0"
                      />
                      <span>Allow Self-Signed Certs</span>
                    </label>
                  </div>

                  <button
                    type="button"
                    onClick={handleTestSource}
                    disabled={testing}
                    className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-indigo-50 dark:bg-indigo-950/40 hover:bg-indigo-100 dark:hover:bg-indigo-900/40 text-indigo-600 dark:text-indigo-400 font-semibold text-xs transition border border-indigo-200 dark:border-indigo-800 disabled:opacity-50"
                  >
                    <Activity className={`w-3.5 h-3.5 ${testing ? 'animate-spin' : ''}`} />
                    <span>{testing ? 'Testing Source...' : 'Test Connection & Discover Folders'}</span>
                  </button>
                </div>

                {/* Test Probe Result Box */}
                {testResult && (
                  <div
                    className={`p-3 rounded-xl border text-xs font-mono ${
                      testResult.success
                        ? 'bg-emerald-500/10 border-emerald-500/20 text-emerald-600 dark:text-emerald-400'
                        : 'bg-rose-500/10 border-rose-500/20 text-rose-600 dark:text-rose-400'
                    }`}
                  >
                    <div className="flex items-center gap-2 font-sans font-bold">
                      {testResult.success ? <CheckCircle2 className="w-4 h-4" /> : <XCircle className="w-4 h-4" />}
                      <span>{testResult.success ? 'IMAP Authentication Successful' : 'Connection Failed'}</span>
                    </div>
                    {testResult.error && <p className="mt-1 text-rose-500 font-sans">{testResult.error}</p>}
                    {testResult.folders_found && testResult.folders_found.length > 0 && (
                      <div className="mt-2 pt-2 border-t border-emerald-500/20">
                        <span className="font-sans font-semibold text-[10px] uppercase block mb-1">
                          Discovered Source Folders ({testResult.folders_found.length}):
                        </span>
                        <div className="flex flex-wrap gap-1">
                          {testResult.folders_found.map((f, i) => (
                            <span key={i} className="px-1.5 py-0.5 rounded bg-white dark:bg-surface-900 border border-emerald-500/30 text-[10px]">
                              {f}
                            </span>
                          ))}
                        </div>
                      </div>
                    )}
                  </div>
                )}
              </div>

              {/* Advanced Folder Exclusion Toggle */}
              <div className="space-y-2">
                <button
                  type="button"
                  onClick={() => setShowAdvanced(!showAdvanced)}
                  className="text-xs font-semibold text-indigo-600 dark:text-indigo-400 hover:underline inline-flex items-center gap-1"
                >
                  <span>{showAdvanced ? 'Hide Advanced Options' : 'Show Advanced Filter Options &bull;'}</span>
                </button>

                {showAdvanced && (
                  <div className="p-4 rounded-xl bg-slate-50 dark:bg-surface-950 border border-slate-200 dark:border-surface-800 space-y-2">
                    <label className="text-xs font-semibold text-slate-700 dark:text-slate-300 block">
                      Exclude Folders (comma-separated)
                    </label>
                    <input
                      type="text"
                      value={excludeFolders}
                      onChange={(e) => setExcludeFolders(e.target.value)}
                      placeholder="Trash, Spam, Junk"
                      className="w-full px-3 py-2 bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-800 rounded-xl text-xs text-slate-900 dark:text-white font-mono focus:outline-none focus:border-indigo-500"
                    />
                    <p className="text-[10px] text-slate-400">
                      Excluded folders will be ignored during synchronization to save storage and bandwidth.
                    </p>
                  </div>
                )}
              </div>

              {/* Modal Actions */}
              <div className="flex items-center justify-end gap-3 pt-3 border-t border-slate-200 dark:border-surface-800">
                <button
                  type="button"
                  onClick={() => setShowModal(false)}
                  className="px-4 py-2 rounded-xl border border-slate-200 dark:border-surface-800 text-xs font-semibold text-slate-600 dark:text-slate-400 hover:bg-slate-100 dark:hover:bg-surface-800 transition"
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  disabled={starting}
                  className="inline-flex items-center gap-2 px-5 py-2 rounded-xl bg-indigo-600 hover:bg-indigo-500 text-white font-semibold text-xs transition shadow-sm disabled:opacity-50"
                >
                  {starting && <RefreshCw className="w-3.5 h-3.5 animate-spin" />}
                  <span>{starting ? 'Initializing Transfer...' : 'Start Background Migration'}</span>
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
}
