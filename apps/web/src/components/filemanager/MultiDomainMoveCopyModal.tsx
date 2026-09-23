'use client';

import React, { useState } from 'react';
import {
  X,
  Globe,
  Folder,
  ArrowRight,
  Copy,
  Scissors,
  CheckCircle2,
  AlertTriangle,
} from 'lucide-react';
import { FileItem } from '@/app/files/page';
import { apiFetch } from '@/lib/api';

interface MultiDomainMoveCopyModalProps {
  mode: 'move' | 'copy';
  items: FileItem[];
  currentDomain?: string;
  domains: Array<{ domain: string; document_root: string }>;
  onClose: () => void;
  onSuccess: (mode: 'move' | 'copy', count: number) => void;
}

export const MultiDomainMoveCopyModal: React.FC<MultiDomainMoveCopyModalProps> = ({
  mode,
  items,
  currentDomain,
  domains,
  onClose,
  onSuccess,
}) => {
  const [destDomain, setDestDomain] = useState(
    domains.find((d) => d.domain !== currentDomain)?.domain || domains[0]?.domain || ''
  );
  const selectedDomainObj = domains.find((d) => d.domain === destDomain);
  const defaultSubDir = selectedDomainObj?.document_root || '/var/www';

  const [destPath, setDestPath] = useState(defaultSubDir);
  const [conflictStrategy, setConflictStrategy] = useState<'rename' | 'replace' | 'skip'>('rename');
  const [processing, setProcessing] = useState(false);
  const [errorMsg, setErrorMsg] = useState<string | null>(null);

  const handleDomainChange = (dom: string) => {
    setDestDomain(dom);
    const dObj = domains.find((d) => d.domain === dom);
    if (dObj) {
      setDestPath(dObj.document_root);
    }
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (items.length === 0 || !destPath.trim()) return;

    try {
      setProcessing(true);
      setErrorMsg(null);

      const endpoint = mode === 'move' ? '/api/v1/filemanager/move' : '/api/v1/filemanager/copy';
      const res = await apiFetch<{ count: number }>(endpoint, {
        method: 'POST',
        body: JSON.stringify({
          src_paths: items.map((i) => i.path),
          dest_path: destPath,
          dest_domain: destDomain,
          conflict_strategy: conflictStrategy,
        }),
      });

      if (res.success) {
        onSuccess(mode, items.length);
        onClose();
      } else {
        setErrorMsg(res.error?.message || `Failed to ${mode} items`);
      }
    } catch (err: any) {
      setErrorMsg(err.message || `Error executing ${mode} operation`);
    } finally {
      setProcessing(false);
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
          <div
            className={`w-10 h-10 rounded-xl flex items-center justify-center ${
              mode === 'move'
                ? 'bg-amber-50 dark:bg-amber-950/40 text-amber-600'
                : 'bg-emerald-50 dark:bg-emerald-950/40 text-emerald-600'
            }`}
          >
            {mode === 'move' ? <Scissors className="w-5 h-5" /> : <Copy className="w-5 h-5" />}
          </div>
          <div>
            <h2 className="text-base font-bold text-slate-900 dark:text-white capitalize">
              {mode} to Another Domain
            </h2>
            <p className="text-xs text-slate-500">
              Transfer {items.length} {items.length === 1 ? 'item' : 'items'} across websites with conflict resolution
            </p>
          </div>
        </div>

        {errorMsg && (
          <div className="mb-4 p-3 rounded-xl bg-rose-50 dark:bg-rose-950/40 border border-rose-200 dark:border-rose-900 text-xs text-rose-700 dark:text-rose-300 flex items-center gap-2">
            <AlertTriangle className="w-4 h-4 shrink-0" />
            <span>{errorMsg}</span>
          </div>
        )}

        <form onSubmit={handleSubmit} className="space-y-4 text-xs font-sans">
          {/* Domain Transfer Visual Indicator */}
          <div className="flex items-center justify-between p-3.5 bg-slate-50 dark:bg-surface-800 rounded-xl border border-slate-200 dark:border-surface-700">
            <div className="min-w-0">
              <span className="text-[10px] text-slate-400 block uppercase font-bold">Source Domain</span>
              <span className="font-bold text-slate-800 dark:text-white truncate block">
                {currentDomain || 'Server Root'}
              </span>
            </div>

            <div className="px-3">
              <ArrowRight className="w-4 h-4 text-emerald-600" />
            </div>

            <div className="min-w-0 text-right">
              <span className="text-[10px] text-slate-400 block uppercase font-bold">Target Domain</span>
              <span className="font-bold text-emerald-600 dark:text-emerald-400 truncate block">
                {destDomain}
              </span>
            </div>
          </div>

          {/* Selected Items List */}
          <div>
            <span className="font-bold text-slate-800 dark:text-white block mb-1.5">
              Selected Items ({items.length})
            </span>
            <div className="max-h-28 overflow-y-auto p-2 bg-slate-50 dark:bg-surface-800 rounded-xl border border-slate-200 dark:border-surface-700 space-y-1">
              {items.map((it) => (
                <div key={it.path} className="flex items-center gap-2 text-slate-700 dark:text-slate-300">
                  <Folder className="w-3.5 h-3.5 text-amber-500 shrink-0" />
                  <span className="truncate">{it.name}</span>
                </div>
              ))}
            </div>
          </div>

          {/* Target Domain Selector */}
          <div>
            <label className="font-bold text-slate-800 dark:text-white block mb-1.5">
              Destination Website / Domain
            </label>
            <div className="relative">
              <Globe className="w-4 h-4 absolute left-3 top-1/2 -translate-y-1/2 text-slate-400" />
              <select
                value={destDomain}
                onChange={(e) => handleDomainChange(e.target.value)}
                className="w-full pl-9 pr-3 py-2 rounded-xl bg-white dark:bg-surface-900 border border-slate-300 dark:border-surface-700 text-slate-900 dark:text-white font-semibold text-xs focus:ring-2 focus:ring-emerald-500"
              >
                {domains.map((d) => (
                  <option key={d.domain} value={d.domain}>
                    {d.domain} ({d.document_root})
                  </option>
                ))}
              </select>
            </div>
          </div>

          {/* Target Directory Path */}
          <div>
            <label className="font-bold text-slate-800 dark:text-white block mb-1.5">
              Target Directory Path
            </label>
            <input
              type="text"
              value={destPath}
              onChange={(e) => setDestPath(e.target.value)}
              placeholder="/var/www/domain/public"
              required
              className="w-full px-3.5 py-2 rounded-xl bg-white dark:bg-surface-900 border border-slate-300 dark:border-surface-700 text-slate-900 dark:text-white font-mono text-xs focus:ring-2 focus:ring-emerald-500"
            />
            <p className="text-[11px] text-slate-400 mt-1">
              If the folder does not exist, it will be automatically created.
            </p>
          </div>

          {/* Conflict Strategy */}
          <div>
            <label className="font-bold text-slate-800 dark:text-white block mb-1.5">
              If Destination File Already Exists
            </label>
            <div className="grid grid-cols-3 gap-2">
              {[
                { id: 'rename', label: 'Auto-Rename', desc: 'Add (1), (2)' },
                { id: 'replace', label: 'Replace', desc: 'Overwrite file' },
                { id: 'skip', label: 'Skip', desc: 'Do not modify' },
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

          {/* Action Buttons */}
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
              disabled={processing}
              className="px-5 py-2 rounded-xl bg-emerald-600 hover:bg-emerald-700 text-white font-bold transition shadow-xs disabled:opacity-50 flex items-center gap-1.5 capitalize"
            >
              <span>{processing ? 'Processing...' : `${mode} Items`}</span>
            </button>
          </div>
        </form>
      </div>
    </div>
  );
};
