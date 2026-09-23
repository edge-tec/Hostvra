'use client';

import React, { useState } from 'react';
import {
  RotateCcw,
  X,
  Folder,
  Globe,
  CheckCircle2,
  AlertTriangle,
  FolderInput,
  ShieldAlert,
} from 'lucide-react';
import { apiFetch } from '@/lib/api';

export interface TrashItemData {
  id: string;
  name: string;
  original_path: string;
  domain: string;
  size: number;
  is_dir: boolean;
  file_type?: string;
  deleted_by?: string;
  deleted_at?: string;
}

interface RestoreWizardModalProps {
  items: TrashItemData[];
  domains: Array<{ domain: string; document_root: string }>;
  onClose: () => void;
  onSuccess: (count: number) => void;
}

export const RestoreWizardModal: React.FC<RestoreWizardModalProps> = ({
  items,
  domains,
  onClose,
  onSuccess,
}) => {
  const [restoreTo, setRestoreTo] = useState<'original' | 'custom_folder' | 'domain'>('original');
  const [customPath, setCustomPath] = useState('');
  const [targetDomain, setTargetDomain] = useState(domains[0]?.domain || '');
  const [conflictStrategy, setConflictStrategy] = useState<'rename' | 'replace' | 'skip'>('rename');
  const [restoring, setRestoring] = useState(false);
  const [errorMsg, setErrorMsg] = useState<string | null>(null);

  const handleRestore = async (e: React.FormEvent) => {
    e.preventDefault();
    if (items.length === 0) return;

    try {
      setRestoring(true);
      setErrorMsg(null);

      const res = await apiFetch<{ restored: boolean; count: number; paths: string[] }>(
        '/api/v1/filemanager/restore',
        {
          method: 'POST',
          body: JSON.stringify({
            trash_ids: items.map((i) => i.id),
            restore_to: restoreTo,
            custom_path: customPath,
            target_domain: targetDomain,
            conflict_strategy: conflictStrategy,
          }),
        }
      );

      if (res.success) {
        onSuccess(items.length);
        onClose();
      } else {
        setErrorMsg(res.error?.message || 'Failed to restore selected items');
      }
    } catch (err: any) {
      setErrorMsg(err.message || 'Error communicating with restore service');
    } finally {
      setRestoring(false);
    }
  };

  return (
    <div className="fixed inset-0 z-50 bg-black/60 backdrop-blur-xs flex items-center justify-center p-4 animate-fadeIn">
      <div className="w-full max-w-lg bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-700 rounded-2xl shadow-2xl p-6 relative">
        <button
          type="button"
          onClick={onClose}
          className="absolute top-5 right-5 p-1 rounded-lg text-slate-400 hover:text-slate-700 dark:hover:text-white"
        >
          <X className="w-5 h-5" />
        </button>

        <div className="flex items-center gap-3 mb-4">
          <div className="w-10 h-10 rounded-xl bg-emerald-50 dark:bg-emerald-950/40 text-emerald-600 flex items-center justify-center">
            <RotateCcw className="w-5 h-5" />
          </div>
          <div>
            <h2 className="text-base font-bold text-slate-900 dark:text-white">
              Restore Wizard
            </h2>
            <p className="text-xs text-slate-500">
              Recover {items.length} {items.length === 1 ? 'item' : 'items'} from the Enterprise Trash Bin
            </p>
          </div>
        </div>

        {errorMsg && (
          <div className="mb-4 p-3 rounded-xl bg-rose-50 dark:bg-rose-950/40 border border-rose-200 dark:border-rose-900 text-xs text-rose-700 dark:text-rose-300 flex items-center gap-2">
            <AlertTriangle className="w-4 h-4 shrink-0" />
            <span>{errorMsg}</span>
          </div>
        )}

        <form onSubmit={handleRestore} className="space-y-4 text-xs font-sans">
          {/* Items Summary Preview */}
          <div className="p-3 bg-slate-50 dark:bg-surface-800 rounded-xl border border-slate-200 dark:border-surface-700 max-h-32 overflow-y-auto space-y-1">
            {items.map((it) => (
              <div key={it.id} className="flex items-center justify-between text-slate-700 dark:text-slate-300">
                <span className="font-semibold truncate max-w-[240px]">{it.name}</span>
                <span className="text-[10px] text-slate-400 font-mono truncate max-w-[180px]">
                  {it.original_path}
                </span>
              </div>
            ))}
          </div>

          {/* Destination Option */}
          <div>
            <label className="font-bold text-slate-800 dark:text-white block mb-2">
              Restore Destination
            </label>
            <div className="space-y-2">
              <label
                className={`flex items-start gap-3 p-3 rounded-xl border cursor-pointer transition ${
                  restoreTo === 'original'
                    ? 'border-emerald-500 bg-emerald-50/50 dark:bg-emerald-950/20 text-slate-900 dark:text-white font-medium'
                    : 'border-slate-200 dark:border-surface-700 text-slate-700 dark:text-slate-300 hover:bg-slate-50'
                }`}
              >
                <input
                  type="radio"
                  name="restoreTo"
                  checked={restoreTo === 'original'}
                  onChange={() => setRestoreTo('original')}
                  className="mt-0.5 text-emerald-600 focus:ring-emerald-500"
                />
                <div>
                  <span className="font-bold">Original Location</span>
                  <p className="text-[11px] text-slate-500">
                    Re-creates missing parent folders automatically if they were deleted.
                  </p>
                </div>
              </label>

              <label
                className={`flex items-start gap-3 p-3 rounded-xl border cursor-pointer transition ${
                  restoreTo === 'custom_folder'
                    ? 'border-emerald-500 bg-emerald-50/50 dark:bg-emerald-950/20 text-slate-900 dark:text-white font-medium'
                    : 'border-slate-200 dark:border-surface-700 text-slate-700 dark:text-slate-300 hover:bg-slate-50'
                }`}
              >
                <input
                  type="radio"
                  name="restoreTo"
                  checked={restoreTo === 'custom_folder'}
                  onChange={() => setRestoreTo('custom_folder')}
                  className="mt-0.5 text-emerald-600 focus:ring-emerald-500"
                />
                <div className="flex-1">
                  <span className="font-bold">Different Folder</span>
                  <p className="text-[11px] text-slate-500 mb-2">
                    Specify an alternative directory to extract the restored files into.
                  </p>
                  {restoreTo === 'custom_folder' && (
                    <input
                      type="text"
                      value={customPath}
                      onChange={(e) => setCustomPath(e.target.value)}
                      placeholder="/var/www/mywebsite/restored"
                      required
                      className="w-full px-3 py-1.5 rounded-lg bg-white dark:bg-surface-900 border border-slate-300 dark:border-surface-700 text-slate-900 dark:text-white font-mono text-xs focus:ring-1 focus:ring-emerald-500"
                    />
                  )}
                </div>
              </label>

              <label
                className={`flex items-start gap-3 p-3 rounded-xl border cursor-pointer transition ${
                  restoreTo === 'domain'
                    ? 'border-emerald-500 bg-emerald-50/50 dark:bg-emerald-950/20 text-slate-900 dark:text-white font-medium'
                    : 'border-slate-200 dark:border-surface-700 text-slate-700 dark:text-slate-300 hover:bg-slate-50'
                }`}
              >
                <input
                  type="radio"
                  name="restoreTo"
                  checked={restoreTo === 'domain'}
                  onChange={() => setRestoreTo('domain')}
                  className="mt-0.5 text-emerald-600 focus:ring-emerald-500"
                />
                <div className="flex-1">
                  <span className="font-bold">Different Website / Domain</span>
                  <p className="text-[11px] text-slate-500 mb-2">
                    Move the restored item directly into another website root.
                  </p>
                  {restoreTo === 'domain' && (
                    <select
                      value={targetDomain}
                      onChange={(e) => setTargetDomain(e.target.value)}
                      className="w-full px-3 py-1.5 rounded-lg bg-white dark:bg-surface-900 border border-slate-300 dark:border-surface-700 text-slate-900 dark:text-white text-xs font-semibold focus:ring-1 focus:ring-emerald-500"
                    >
                      {domains.map((d) => (
                        <option key={d.domain} value={d.domain}>
                          {d.domain} ({d.document_root})
                        </option>
                      ))}
                    </select>
                  )}
                </div>
              </label>
            </div>
          </div>

          {/* Conflict Strategy */}
          <div>
            <label className="font-bold text-slate-800 dark:text-white block mb-1.5">
              If File Already Exists
            </label>
            <div className="grid grid-cols-3 gap-2">
              {[
                { id: 'rename', label: 'Auto-Rename', desc: 'Add (1), (2)' },
                { id: 'replace', label: 'Overwrite', desc: 'Replace existing' },
                { id: 'skip', label: 'Skip File', desc: 'Do not restore' },
              ].map((strat) => (
                <button
                  key={strat.id}
                  type="button"
                  onClick={() => setConflictStrategy(strat.id as any)}
                  className={`p-2.5 rounded-xl border text-left cursor-pointer transition ${
                    conflictStrategy === strat.id
                      ? 'border-emerald-600 bg-emerald-50/50 dark:bg-emerald-950/30 text-emerald-900 dark:text-emerald-200 font-bold'
                      : 'border-slate-200 dark:border-surface-700 text-slate-600 dark:text-slate-400 hover:bg-slate-50'
                  }`}
                >
                  <span className="block text-xs font-bold">{strat.label}</span>
                  <span className="block text-[10px] text-slate-400 font-normal">{strat.desc}</span>
                </button>
              ))}
            </div>
          </div>

          {/* Actions */}
          <div className="flex items-center justify-end gap-2.5 pt-4 border-t border-slate-200 dark:border-surface-800">
            <button
              type="button"
              onClick={onClose}
              className="px-4 py-2 rounded-xl border border-slate-200 dark:border-surface-700 font-bold text-slate-700 dark:text-slate-300 hover:bg-slate-100 transition"
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={restoring}
              className="px-5 py-2 rounded-xl bg-emerald-600 hover:bg-emerald-700 text-white font-bold transition shadow-xs disabled:opacity-50 flex items-center gap-1.5"
            >
              <RotateCcw className={`w-3.5 h-3.5 ${restoring ? 'animate-spin' : ''}`} />
              <span>{restoring ? 'Restoring...' : 'Restore Items'}</span>
            </button>
          </div>
        </form>
      </div>
    </div>
  );
};
