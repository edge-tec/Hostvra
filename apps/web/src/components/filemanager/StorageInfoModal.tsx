'use client';

import React, { useState, useEffect } from 'react';
import { HardDrive, X, Folder, Trash2, Archive, PieChart, RefreshCw } from 'lucide-react';
import { apiFetch } from '@/lib/api';

interface StorageInfoModalProps {
  currentPath: string;
  domain?: string;
  onClose: () => void;
}

interface StorageData {
  folder_size: number;
  file_count: number;
  dir_count: number;
  trash_size: number;
  total_disk_bytes: number;
  used_disk_bytes: number;
  free_disk_bytes: number;
}

export const StorageInfoModal: React.FC<StorageInfoModalProps> = ({
  currentPath,
  domain,
  onClose,
}) => {
  const [data, setData] = useState<StorageData | null>(null);
  const [loading, setLoading] = useState(true);

  const fetchStorage = async () => {
    try {
      setLoading(true);
      const res = await apiFetch<StorageData>(
        `/api/v1/filemanager/storage?path=${encodeURIComponent(currentPath)}&domain=${encodeURIComponent(domain || '')}`
      );
      if (res.success && res.data) {
        setData(res.data);
      }
    } catch (err) {
      console.error('Failed to load storage telemetry', err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchStorage();
  }, [currentPath, domain]);

  const formatBytes = (bytes: number) => {
    if (!bytes || bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i];
  };

  const usedPercent =
    data && data.total_disk_bytes > 0
      ? Math.round((data.used_disk_bytes / data.total_disk_bytes) * 100)
      : 0;

  const getBarColor = (pct: number) => {
    if (pct > 90) return 'bg-rose-500';
    if (pct > 75) return 'bg-amber-500';
    return 'bg-emerald-500';
  };

  return (
    <div className="fixed inset-0 z-50 bg-black/60 backdrop-blur-xs flex items-center justify-center p-4 animate-fadeIn">
      <div className="w-full max-w-lg bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-700 rounded-2xl shadow-2xl p-6 relative font-sans">
        <button
          type="button"
          onClick={onClose}
          className="absolute top-5 right-5 p-1 rounded-lg text-slate-400 hover:text-slate-700 dark:hover:text-white"
        >
          <X className="w-5 h-5" />
        </button>

        <div className="flex items-center gap-3 mb-5">
          <div className="w-10 h-10 rounded-xl bg-emerald-50 dark:bg-emerald-950/40 text-emerald-600 flex items-center justify-center">
            <PieChart className="w-5 h-5" />
          </div>
          <div>
            <h2 className="text-base font-bold text-slate-900 dark:text-white">
              Storage Usage & Disk Telemetry
            </h2>
            <p className="text-xs text-slate-500">Live storage analysis and breakdown</p>
          </div>
        </div>

        {loading && !data ? (
          <div className="py-12 text-center text-slate-400 text-xs flex items-center justify-center gap-2">
            <RefreshCw className="w-4 h-4 animate-spin text-emerald-600" />
            <span>Calculating disk and folder storage...</span>
          </div>
        ) : (
          <div className="space-y-4 text-xs">
            {/* Overall Server Storage Bar */}
            <div className="p-4 rounded-xl bg-slate-50 dark:bg-surface-800 border border-slate-200 dark:border-surface-700 space-y-2">
              <div className="flex items-center justify-between">
                <span className="font-bold text-slate-800 dark:text-white flex items-center gap-1.5">
                  <HardDrive className="w-4 h-4 text-emerald-600" />
                  <span>Server Disk Usage</span>
                </span>
                <span className="font-mono font-bold text-slate-900 dark:text-white">
                  {usedPercent}% used
                </span>
              </div>

              {/* Progress bar */}
              <div className="w-full h-3 bg-slate-200 dark:bg-surface-700 rounded-full overflow-hidden">
                <div
                  className={`h-full transition-all duration-500 rounded-full ${getBarColor(usedPercent)}`}
                  style={{ width: `${Math.min(usedPercent, 100)}%` }}
                />
              </div>

              <div className="flex items-center justify-between text-[11px] text-slate-500 font-mono">
                <span>Used: {formatBytes(data?.used_disk_bytes || 0)}</span>
                <span>Free: {formatBytes(data?.free_disk_bytes || 0)}</span>
                <span>Total: {formatBytes(data?.total_disk_bytes || 0)}</span>
              </div>
            </div>

            {/* Folder & Category Cards */}
            <div className="grid grid-cols-2 gap-3">
              {/* Current Folder */}
              <div className="p-3.5 rounded-xl border border-slate-200 dark:border-surface-700 bg-white dark:bg-surface-900 shadow-xs">
                <div className="flex items-center gap-2 mb-2 text-slate-500">
                  <Folder className="w-4 h-4 text-amber-500" />
                  <span className="font-bold">Current Folder</span>
                </div>
                <div className="font-mono font-bold text-base text-slate-900 dark:text-white">
                  {formatBytes(data?.folder_size || 0)}
                </div>
                <div className="text-[11px] text-slate-400 mt-1">
                  {data?.file_count || 0} files, {data?.dir_count || 0} folders
                </div>
              </div>

              {/* Trash Size */}
              <div className="p-3.5 rounded-xl border border-slate-200 dark:border-surface-700 bg-white dark:bg-surface-900 shadow-xs">
                <div className="flex items-center gap-2 mb-2 text-slate-500">
                  <Trash2 className="w-4 h-4 text-rose-500" />
                  <span className="font-bold">Trash Bin</span>
                </div>
                <div className="font-mono font-bold text-base text-rose-600 dark:text-rose-400">
                  {formatBytes(data?.trash_size || 0)}
                </div>
                <div className="text-[11px] text-slate-400 mt-1">Ready to be emptied</div>
              </div>
            </div>

            {/* Target Path Info */}
            <div className="p-3 rounded-xl bg-slate-50 dark:bg-surface-800 border border-slate-200 dark:border-surface-700 text-[11px] text-slate-600 dark:text-slate-400 font-mono">
              <span className="text-slate-400 block mb-0.5">Scanned Directory:</span>
              <span className="break-all">{currentPath}</span>
            </div>

            {/* Action buttons */}
            <div className="flex items-center justify-end gap-2.5 pt-2">
              <button
                type="button"
                onClick={onClose}
                className="px-4 py-2 rounded-xl bg-slate-100 hover:bg-slate-200 dark:bg-surface-800 text-slate-700 dark:text-slate-200 font-bold transition"
              >
                Close
              </button>
            </div>
          </div>
        )}
      </div>
    </div>
  );
};
