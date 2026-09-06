'use client';

import React, { useState } from 'react';
import { DashboardShell } from '@/components/DashboardShell';
import {
  HardDriveDownload,
  Plus,
  RefreshCw,
  RotateCcw,
  CheckCircle2,
  AlertTriangle,
  Server,
  Cloud,
  HardDrive,
  FileArchive,
  Database,
  Sliders,
  ShieldCheck,
  Calendar,
} from 'lucide-react';

interface BackupRecord {
  id: string;
  server_id: string;
  type: 'website' | 'database' | 'full_config';
  target_name: string;
  storage: 'local' | 's3';
  size_bytes: number;
  status: 'completed' | 'failed' | 'in_progress';
  created_at: string;
}

const initialBackups: BackupRecord[] = [
  {
    id: 'bk-web-849281',
    server_id: 'prod-edge-01 (192.168.1.100)',
    type: 'website',
    target_name: 'hostvra.com',
    storage: 's3',
    size_bytes: 48312091,
    status: 'completed',
    created_at: '2026-09-06 06:30 UTC',
  },
  {
    id: 'bk-db-312984',
    server_id: 'prod-edge-01 (192.168.1.100)',
    type: 'database',
    target_name: 'hostvra_prod_db',
    storage: 's3',
    size_bytes: 14209118,
    status: 'completed',
    created_at: '2026-09-06 04:00 UTC',
  },
  {
    id: 'bk-full-109283',
    server_id: 'prod-edge-01 (192.168.1.100)',
    type: 'full_config',
    target_name: 'Nginx, Vhosts & SSL Metadata',
    storage: 'local',
    size_bytes: 2840192,
    status: 'completed',
    created_at: '2026-09-05 23:00 UTC',
  },
];

function formatBytes(bytes: number): string {
  if (bytes < 1024) return bytes + ' B';
  if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB';
  if (bytes < 1024 * 1024 * 1024) return (bytes / (1024 * 1024)).toFixed(1) + ' MB';
  return (bytes / (1024 * 1024 * 1024)).toFixed(2) + ' GB';
}

export default function BackupsPage() {
  const [backups, setBackups] = useState<BackupRecord[]>(initialBackups);
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [selectedBackupForRestore, setSelectedBackupForRestore] = useState<BackupRecord | null>(null);
  const [isRestoring, setIsRestoring] = useState(false);
  const [restoreSuccess, setRestoreSuccess] = useState(false);

  // Form state
  const [targetType, setTargetType] = useState<'website' | 'database' | 'full_config'>('website');
  const [targetName, setTargetName] = useState('');
  const [storageTarget, setStorageTarget] = useState<'local' | 's3'>('local');

  const handleCreateBackup = (e: React.FormEvent) => {
    e.preventDefault();
    const newBackup: BackupRecord = {
      id: `bk-${targetType.substring(0, 3)}-${Math.floor(Math.random() * 900000 + 100000)}`,
      server_id: 'prod-edge-01 (192.168.1.100)',
      type: targetType,
      target_name: targetName || (targetType === 'full_config' ? 'Full Server Stack' : 'default'),
      storage: storageTarget,
      size_bytes: Math.floor(Math.random() * 25000000 + 5000000),
      status: 'completed',
      created_at: 'Just now',
    };
    setBackups([newBackup, ...backups]);
    setShowCreateModal(false);
    setTargetName('');
  };

  const handleConfirmRestore = () => {
    setIsRestoring(true);
    setTimeout(() => {
      setIsRestoring(false);
      setRestoreSuccess(true);
      setTimeout(() => {
        setRestoreSuccess(false);
        setSelectedBackupForRestore(null);
      }, 2500);
    }, 1500);
  };

  return (
    <DashboardShell>
      <div className="space-y-6">
        {/* Page Header */}
        <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
          <div>
            <h1 className="text-2xl font-bold text-white tracking-tight flex items-center gap-2">
              <HardDriveDownload className="w-7 h-7 text-brand-400" />
              Automated & On-Demand Backups
            </h1>
            <p className="text-sm text-slate-400 mt-1">
              Deterministic tar.gz compression, path traversal safe verification, and S3-compatible multi-cloud offsite sync.
            </p>
          </div>
          <button
            onClick={() => setShowCreateModal(true)}
            className="inline-flex items-center justify-center gap-2 px-4 py-2.5 rounded-lg bg-brand-500 hover:bg-brand-600 text-white text-sm font-semibold transition-colors shadow-lg shadow-brand-500/20"
          >
            <Plus className="w-4 h-4" />
            Create Backup
          </button>
        </div>

        {/* Storage Repositories Overview Cards */}
        <div className="grid grid-cols-1 md:grid-cols-3 gap-5">
          <div className="bg-surface-900 border border-surface-800 rounded-xl p-5 flex items-start gap-4">
            <div className="p-3 rounded-lg bg-emerald-500/10 text-emerald-400 border border-emerald-500/20">
              <Cloud className="w-6 h-6" />
            </div>
            <div>
              <div className="text-xs font-semibold text-slate-400 uppercase tracking-wider">Primary Offsite</div>
              <div className="text-lg font-bold text-white mt-1">AWS S3 / Cloudflare R2</div>
              <div className="text-xs text-emerald-400 mt-1 flex items-center gap-1 font-medium">
                <CheckCircle2 className="w-3.5 h-3.5" /> Connected & Synced
              </div>
            </div>
          </div>

          <div className="bg-surface-900 border border-surface-800 rounded-xl p-5 flex items-start gap-4">
            <div className="p-3 rounded-lg bg-blue-500/10 text-blue-400 border border-blue-500/20">
              <HardDrive className="w-6 h-6" />
            </div>
            <div>
              <div className="text-xs font-semibold text-slate-400 uppercase tracking-wider">Local Snapshots</div>
              <div className="text-lg font-bold text-white mt-1">/var/lib/hostvra/backups</div>
              <div className="text-xs text-slate-400 mt-1">Retention: 7 days daily, 4 weeks</div>
            </div>
          </div>

          <div className="bg-surface-900 border border-surface-800 rounded-xl p-5 flex items-start gap-4">
            <div className="p-3 rounded-lg bg-indigo-500/10 text-indigo-400 border border-indigo-500/20">
              <ShieldCheck className="w-6 h-6" />
            </div>
            <div>
              <div className="text-xs font-semibold text-slate-400 uppercase tracking-wider">Restore Safety</div>
              <div className="text-lg font-bold text-white mt-1">Atomic Rollback Guard</div>
              <div className="text-xs text-indigo-400 mt-1">Pre-restore snapshot enabled</div>
            </div>
          </div>
        </div>

        {/* Backups Table */}
        <div className="bg-surface-900 border border-surface-800 rounded-xl overflow-hidden">
          <div className="px-6 py-4 border-b border-surface-800 flex items-center justify-between">
            <h2 className="text-sm font-semibold text-white uppercase tracking-wider">Available Archives</h2>
            <span className="text-xs font-mono text-slate-400">{backups.length} snapshots</span>
          </div>

          <div className="overflow-x-auto">
            <table className="w-full text-left text-sm text-slate-300">
              <thead className="bg-surface-950/60 text-xs uppercase font-semibold text-slate-400 border-b border-surface-800">
                <tr>
                  <th className="px-6 py-3.5">Archive ID</th>
                  <th className="px-6 py-3.5">Scope & Target</th>
                  <th className="px-6 py-3.5">Destination</th>
                  <th className="px-6 py-3.5">Size</th>
                  <th className="px-6 py-3.5">Created</th>
                  <th className="px-6 py-3.5">Integrity</th>
                  <th className="px-6 py-3.5 text-right">Actions</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-surface-800/60 font-mono text-xs">
                {backups.map((b) => (
                  <tr key={b.id} className="hover:bg-surface-800/30 transition-colors">
                    <td className="px-6 py-4 font-bold text-white flex items-center gap-2">
                      <FileArchive className="w-4 h-4 text-brand-400" />
                      <span>{b.id}</span>
                    </td>
                    <td className="px-6 py-4">
                      <div className="flex items-center gap-2">
                        {b.type === 'website' && <span className="px-2 py-0.5 rounded bg-blue-500/10 text-blue-400 border border-blue-500/20 font-sans text-[11px] font-semibold uppercase">Site</span>}
                        {b.type === 'database' && <span className="px-2 py-0.5 rounded bg-amber-500/10 text-amber-400 border border-amber-500/20 font-sans text-[11px] font-semibold uppercase">DB</span>}
                        {b.type === 'full_config' && <span className="px-2 py-0.5 rounded bg-purple-500/10 text-purple-400 border border-purple-500/20 font-sans text-[11px] font-semibold uppercase">Full</span>}
                        <span className="text-white font-sans text-sm">{b.target_name}</span>
                      </div>
                    </td>
                    <td className="px-6 py-4">
                      <span className="inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded text-[11px] font-semibold capitalize bg-surface-800 text-slate-300">
                        {b.storage === 's3' ? <Cloud className="w-3 h-3 text-cyan-400" /> : <HardDrive className="w-3 h-3 text-slate-400" />}
                        {b.storage}
                      </span>
                    </td>
                    <td className="px-6 py-4 font-mono text-slate-300">{formatBytes(b.size_bytes)}</td>
                    <td className="px-6 py-4 text-slate-400">{b.created_at}</td>
                    <td className="px-6 py-4">
                      <span className="inline-flex items-center gap-1 text-emerald-400 text-[11px] font-semibold">
                        <CheckCircle2 className="w-3.5 h-3.5" /> SHA256 OK
                      </span>
                    </td>
                    <td className="px-6 py-4 text-right">
                      <button
                        onClick={() => setSelectedBackupForRestore(b)}
                        className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-surface-800 hover:bg-surface-700 text-white text-xs font-semibold transition-colors border border-surface-700"
                      >
                        <RotateCcw className="w-3.5 h-3.5 text-amber-400" />
                        Restore
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>

        {/* Create Backup Modal */}
        {showCreateModal && (
          <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/70 backdrop-blur-sm p-4">
            <div className="bg-surface-900 border border-surface-800 rounded-xl max-w-md w-full p-6 shadow-2xl space-y-5">
              <div>
                <h3 className="text-lg font-bold text-white flex items-center gap-2">
                  <HardDriveDownload className="w-5 h-5 text-brand-400" />
                  Initiate On-Demand Snapshot
                </h3>
                <p className="text-xs text-slate-400 mt-1">
                  Creates a compressed tar.gz archive with atomic rollback and remote cloud sync.
                </p>
              </div>

              <form onSubmit={handleCreateBackup} className="space-y-4 text-sm">
                <div>
                  <label className="block text-xs font-semibold text-slate-300 mb-1">Backup Scope</label>
                  <div className="grid grid-cols-3 gap-2">
                    {(['website', 'database', 'full_config'] as const).map((type) => (
                      <button
                        type="button"
                        key={type}
                        onClick={() => setTargetType(type)}
                        className={`px-3 py-2 rounded-lg text-xs font-semibold border capitalize transition-all ${
                          targetType === type
                            ? 'bg-brand-500/20 border-brand-500 text-brand-400'
                            : 'bg-surface-800 border-surface-700 text-slate-400 hover:text-white'
                        }`}
                      >
                        {type === 'full_config' ? 'Full Stack' : type}
                      </button>
                    ))}
                  </div>
                </div>

                {targetType !== 'full_config' && (
                  <div>
                    <label className="block text-xs font-semibold text-slate-300 mb-1">
                      {targetType === 'website' ? 'Domain Name' : 'Database Name'}
                    </label>
                    <input
                      type="text"
                      required
                      placeholder={targetType === 'website' ? 'e.g. hostvra.com' : 'e.g. app_production_db'}
                      value={targetName}
                      onChange={(e) => setTargetName(e.target.value)}
                      className="w-full px-3.5 py-2.5 rounded-lg bg-surface-950 border border-surface-800 text-white font-mono text-sm focus:outline-none focus:border-brand-500"
                    />
                  </div>
                )}

                <div>
                  <label className="block text-xs font-semibold text-slate-300 mb-1">Storage Destination</label>
                  <div className="grid grid-cols-2 gap-2">
                    <button
                      type="button"
                      onClick={() => setStorageTarget('local')}
                      className={`px-3 py-2 rounded-lg text-xs font-semibold border flex items-center justify-center gap-1.5 transition-all ${
                        storageTarget === 'local'
                          ? 'bg-brand-500/20 border-brand-500 text-brand-400'
                          : 'bg-surface-800 border-surface-700 text-slate-400'
                      }`}
                    >
                      <HardDrive className="w-3.5 h-3.5" /> Local Server
                    </button>
                    <button
                      type="button"
                      onClick={() => setStorageTarget('s3')}
                      className={`px-3 py-2 rounded-lg text-xs font-semibold border flex items-center justify-center gap-1.5 transition-all ${
                        storageTarget === 's3'
                          ? 'bg-brand-500/20 border-brand-500 text-brand-400'
                          : 'bg-surface-800 border-surface-700 text-slate-400'
                      }`}
                    >
                      <Cloud className="w-3.5 h-3.5" /> S3 / R2 Offsite
                    </button>
                  </div>
                </div>

                <div className="p-3 rounded-lg bg-surface-950 border border-surface-800 text-xs text-slate-400 space-y-1">
                  <div className="font-semibold text-slate-300 flex items-center gap-1">
                    <ShieldCheck className="w-3.5 h-3.5 text-emerald-400" /> Automated Safeguards:
                  </div>
                  <div>• Excludes node_modules, .git, and cache files automatically</div>
                  <div>• Streams directly without exhausting available RAM</div>
                </div>

                <div className="flex items-center justify-end gap-3 pt-2">
                  <button
                    type="button"
                    onClick={() => setShowCreateModal(false)}
                    className="px-4 py-2 rounded-lg text-xs font-semibold text-slate-400 hover:text-white"
                  >
                    Cancel
                  </button>
                  <button
                    type="submit"
                    className="px-4 py-2 rounded-lg bg-brand-500 hover:bg-brand-600 text-white text-xs font-semibold transition-colors"
                  >
                    Start Backup Job
                  </button>
                </div>
              </form>
            </div>
          </div>
        )}

        {/* Restore Confirmation Modal */}
        {selectedBackupForRestore && (
          <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/70 backdrop-blur-sm p-4">
            <div className="bg-surface-900 border border-surface-800 rounded-xl max-w-md w-full p-6 shadow-2xl space-y-5">
              <div>
                <h3 className="text-lg font-bold text-white flex items-center gap-2">
                  <RotateCcw className="w-5 h-5 text-amber-400" />
                  Restore Snapshot
                </h3>
                <p className="text-xs text-slate-400 mt-1">
                  Restore target: <span className="text-white font-mono font-bold">{selectedBackupForRestore.target_name}</span> ({selectedBackupForRestore.id})
                </p>
              </div>

              <div className="p-4 rounded-lg bg-amber-500/10 border border-amber-500/20 text-xs text-amber-300 space-y-2">
                <div className="font-bold flex items-center gap-1.5">
                  <AlertTriangle className="w-4 h-4" />
                  Rollback Safeguard Active
                </div>
                <p>
                  Before extracting, Hostvra automatically creates a staging snapshot of the active state. If the restore fails or an error occurs, your existing files will be restored automatically.
                </p>
              </div>

              {restoreSuccess ? (
                <div className="p-4 rounded-lg bg-emerald-500/10 border border-emerald-500/20 text-emerald-400 text-sm font-semibold flex items-center gap-2">
                  <CheckCircle2 className="w-5 h-5" />
                  Snapshot restored successfully with verified state!
                </div>
              ) : (
                <div className="flex items-center justify-end gap-3 pt-2">
                  <button
                    type="button"
                    disabled={isRestoring}
                    onClick={() => setSelectedBackupForRestore(null)}
                    className="px-4 py-2 rounded-lg text-xs font-semibold text-slate-400 hover:text-white disabled:opacity-50"
                  >
                    Cancel
                  </button>
                  <button
                    type="button"
                    disabled={isRestoring}
                    onClick={handleConfirmRestore}
                    className="inline-flex items-center gap-2 px-4 py-2 rounded-lg bg-amber-500 hover:bg-amber-600 text-slate-950 text-xs font-bold transition-colors disabled:opacity-50"
                  >
                    {isRestoring ? (
                      <>
                        <RefreshCw className="w-3.5 h-3.5 animate-spin" />
                        Extracting & Verifying...
                      </>
                    ) : (
                      'Confirm & Restore'
                    )}
                  </button>
                </div>
              )}
            </div>
          </div>
        )}
      </div>
    </DashboardShell>
  );
}
