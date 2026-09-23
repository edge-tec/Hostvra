'use client';

import React, { useState } from 'react';
import { Trash2, AlertOctagon, X, AlertTriangle } from 'lucide-react';
import { apiFetch } from '@/lib/api';

interface EmptyTrashModalProps {
  totalFiles: number;
  totalFolders: number;
  totalSize: number;
  domain?: string;
  onClose: () => void;
  onSuccess: (freedBytes: number, filesPurged: number) => void;
}

export const EmptyTrashModal: React.FC<EmptyTrashModalProps> = ({
  totalFiles,
  totalFolders,
  totalSize,
  domain,
  onClose,
  onSuccess,
}) => {
  const [confirmText, setConfirmText] = useState('');
  const [purging, setPurging] = useState(false);
  const [errorMsg, setErrorMsg] = useState<string | null>(null);

  const isConfirmed = confirmText.trim() === 'DELETE';

  const formatBytes = (bytes: number) => {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i];
  };

  const handleEmptyTrash = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!isConfirmed) return;

    try {
      setPurging(true);
      setErrorMsg(null);

      const res = await apiFetch<{
        empty: boolean;
        files_purged: number;
        bytes_freed: number;
      }>('/api/v1/filemanager/trash/empty', {
        method: 'DELETE',
        body: JSON.stringify({
          confirmation: 'DELETE',
          domain: domain || '',
        }),
      });

      if (res.success && res.data) {
        onSuccess(res.data.bytes_freed, res.data.files_purged);
        onClose();
      } else {
        setErrorMsg(res.error?.message || 'Failed to empty trash');
      }
    } catch (err: any) {
      setErrorMsg(err.message || 'Error communicating with trash purge service');
    } finally {
      setPurging(false);
    }
  };

  return (
    <div className="fixed inset-0 z-50 bg-black/60 backdrop-blur-xs flex items-center justify-center p-4 animate-fadeIn">
      <div className="w-full max-w-md bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-700 rounded-2xl shadow-2xl p-6 relative">
        <button
          type="button"
          onClick={onClose}
          className="absolute top-5 right-5 p-1 rounded-lg text-slate-400 hover:text-slate-700 dark:hover:text-white"
        >
          <X className="w-5 h-5" />
        </button>

        <div className="flex items-center gap-3 mb-4">
          <div className="w-10 h-10 rounded-xl bg-rose-50 dark:bg-rose-950/40 text-rose-600 flex items-center justify-center">
            <AlertOctagon className="w-5 h-5" />
          </div>
          <div>
            <h2 className="text-base font-bold text-slate-900 dark:text-white">
              Permanently Empty Trash?
            </h2>
            <p className="text-xs text-slate-500">This action cannot be undone.</p>
          </div>
        </div>

        {errorMsg && (
          <div className="mb-4 p-3 rounded-xl bg-rose-50 dark:bg-rose-950/40 border border-rose-200 dark:border-rose-900 text-xs text-rose-700 dark:text-rose-300 flex items-center gap-2">
            <AlertTriangle className="w-4 h-4 shrink-0" />
            <span>{errorMsg}</span>
          </div>
        )}

        <form onSubmit={handleEmptyTrash} className="space-y-4 text-xs font-sans">
          {/* Storage impact breakdown */}
          <div className="p-3.5 rounded-xl bg-slate-50 dark:bg-surface-800 border border-slate-200 dark:border-surface-700 space-y-2">
            <div className="flex items-center justify-between text-slate-600 dark:text-slate-400">
              <span>Total Files:</span>
              <span className="font-mono font-bold text-slate-900 dark:text-white">
                {totalFiles}
              </span>
            </div>
            <div className="flex items-center justify-between text-slate-600 dark:text-slate-400">
              <span>Total Folders:</span>
              <span className="font-mono font-bold text-slate-900 dark:text-white">
                {totalFolders}
              </span>
            </div>
            <div className="flex items-center justify-between text-slate-600 dark:text-slate-400 pt-1.5 border-t border-slate-200 dark:border-surface-700">
              <span className="font-bold">Total Storage to Free:</span>
              <span className="font-mono font-bold text-emerald-600 dark:text-emerald-400 text-sm">
                {formatBytes(totalSize)}
              </span>
            </div>
          </div>

          <p className="text-slate-600 dark:text-slate-400 text-[11px] leading-relaxed">
            All physical files, directories, database entries, thumbnails, and temporary caches
            will be permanently removed from disk. Only the audit log will preserve record of deletion.
          </p>

          {/* Security confirmation input */}
          <div>
            <label className="font-bold text-slate-800 dark:text-white block mb-1.5">
              Type <span className="font-mono text-rose-600 bg-rose-50 px-1 py-0.5 rounded">DELETE</span> to confirm:
            </label>
            <input
              type="text"
              value={confirmText}
              onChange={(e) => setConfirmText(e.target.value)}
              placeholder="DELETE"
              className="w-full px-3.5 py-2 rounded-xl bg-white dark:bg-surface-900 border border-slate-300 dark:border-surface-700 text-slate-900 dark:text-white font-mono font-bold text-sm tracking-wider focus:outline-hidden focus:ring-2 focus:ring-rose-500"
            />
          </div>

          {/* Buttons */}
          <div className="flex items-center justify-end gap-2.5 pt-3 border-t border-slate-200 dark:border-surface-800">
            <button
              type="button"
              onClick={onClose}
              className="px-4 py-2 rounded-xl border border-slate-200 dark:border-surface-700 font-bold text-slate-700 dark:text-slate-300 hover:bg-slate-100 transition"
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={!isConfirmed || purging}
              className="px-5 py-2 rounded-xl bg-rose-600 hover:bg-rose-700 text-white font-bold transition shadow-xs disabled:opacity-40 disabled:cursor-not-allowed flex items-center gap-1.5"
            >
              <Trash2 className="w-3.5 h-3.5" />
              <span>{purging ? 'Purging...' : 'Permanently Delete Everything'}</span>
            </button>
          </div>
        </form>
      </div>
    </div>
  );
};
