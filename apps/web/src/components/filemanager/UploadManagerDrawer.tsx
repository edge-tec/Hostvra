'use client';

import React, { useState, useEffect, useRef } from 'react';
import {
  UploadCloud,
  X,
  ChevronDown,
  ChevronUp,
  CheckCircle2,
  AlertCircle,
  Pause,
  Play,
  RotateCcw,
  File,
  Folder,
} from 'lucide-react';
import { apiFetch } from '@/lib/api';

export interface UploadTask {
  id: string;
  file: File;
  targetDir: string;
  progress: number;
  uploadedBytes: number;
  totalBytes: number;
  speed: string;
  status: 'queued' | 'uploading' | 'completed' | 'failed' | 'cancelled';
  error?: string;
}

interface UploadManagerDrawerProps {
  tasks: UploadTask[];
  onCancel: (taskId: string) => void;
  onRetry: (taskId: string) => void;
  onClearCompleted: () => void;
}

export const UploadManagerDrawer: React.FC<UploadManagerDrawerProps> = ({
  tasks,
  onCancel,
  onRetry,
  onClearCompleted,
}) => {
  const [minimized, setMinimized] = useState(false);

  if (tasks.length === 0) return null;

  const completedCount = tasks.filter((t) => t.status === 'completed').length;
  const activeCount = tasks.filter((t) => t.status === 'uploading' || t.status === 'queued').length;
  const failedCount = tasks.filter((t) => t.status === 'failed').length;

  const totalBytes = tasks.reduce((sum, t) => sum + t.totalBytes, 0);
  const totalUploaded = tasks.reduce((sum, t) => sum + t.uploadedBytes, 0);
  const overallPercent = totalBytes > 0 ? Math.round((totalUploaded / totalBytes) * 100) : 0;

  const formatBytes = (bytes: number) => {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i];
  };

  return (
    <div className="fixed bottom-4 right-4 z-50 w-96 max-w-[calc(100vw-2rem)] bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-700 rounded-2xl shadow-2xl overflow-hidden font-sans text-xs animate-slideUp">
      {/* Drawer Header */}
      <div
        onClick={() => setMinimized(!minimized)}
        className="flex items-center justify-between p-3.5 bg-slate-50 dark:bg-surface-800 border-b border-slate-200 dark:border-surface-700 cursor-pointer select-none"
      >
        <div className="flex items-center gap-2">
          <UploadCloud className="w-4 h-4 text-emerald-600 animate-pulse" />
          <span className="font-bold text-slate-900 dark:text-white">
            {activeCount > 0
              ? `Uploading ${activeCount} ${activeCount === 1 ? 'file' : 'files'} (${overallPercent}%)`
              : `Uploads Finished (${completedCount}/${tasks.length})`}
          </span>
        </div>

        <div className="flex items-center gap-1.5">
          {completedCount > 0 && activeCount === 0 && (
            <button
              type="button"
              onClick={(e) => {
                e.stopPropagation();
                onClearCompleted();
              }}
              className="text-[10px] text-emerald-600 font-bold hover:underline px-1"
            >
              Clear
            </button>
          )}

          <button
            type="button"
            className="p-1 rounded-lg text-slate-400 hover:text-slate-700 dark:hover:text-white"
          >
            {minimized ? <ChevronUp className="w-4 h-4" /> : <ChevronDown className="w-4 h-4" />}
          </button>
        </div>
      </div>

      {/* Mini Progress Bar when Minimized */}
      {minimized && activeCount > 0 && (
        <div className="w-full bg-slate-200 dark:bg-surface-700 h-1">
          <div
            className="bg-emerald-500 h-1 transition-all duration-300"
            style={{ width: `${overallPercent}%` }}
          />
        </div>
      )}

      {/* Task List */}
      {!minimized && (
        <div className="max-h-64 overflow-y-auto p-3 space-y-2.5">
          {tasks.map((task) => (
            <div
              key={task.id}
              className="p-2.5 rounded-xl bg-slate-50 dark:bg-surface-800/60 border border-slate-100 dark:border-surface-700 space-y-1.5"
            >
              <div className="flex items-center justify-between gap-2">
                <div className="flex items-center gap-2 min-w-0">
                  <File className="w-3.5 h-3.5 text-slate-400 shrink-0" />
                  <span className="font-semibold text-slate-800 dark:text-slate-200 truncate">
                    {task.file.name}
                  </span>
                </div>

                <div className="flex items-center gap-1 shrink-0">
                  {task.status === 'completed' && (
                    <CheckCircle2 className="w-4 h-4 text-emerald-500" />
                  )}
                  {task.status === 'failed' && (
                    <span title={task.error}>
                      <AlertCircle className="w-4 h-4 text-rose-500" />
                    </span>
                  )}

                  {task.status === 'failed' && (
                    <button
                      type="button"
                      onClick={() => onRetry(task.id)}
                      title="Retry"
                      className="p-1 text-slate-400 hover:text-emerald-600 cursor-pointer"
                    >
                      <RotateCcw className="w-3 h-3" />
                    </button>
                  )}

                  {task.status !== 'completed' && (
                    <button
                      type="button"
                      onClick={() => onCancel(task.id)}
                      title="Cancel"
                      className="p-1 text-slate-400 hover:text-rose-600 cursor-pointer"
                    >
                      <X className="w-3 h-3" />
                    </button>
                  )}
                </div>
              </div>

              {/* Progress Bar */}
              <div className="w-full bg-slate-200 dark:bg-surface-700 h-1.5 rounded-full overflow-hidden">
                <div
                  className={`h-full transition-all duration-300 rounded-full ${
                    task.status === 'completed'
                      ? 'bg-emerald-500'
                      : task.status === 'failed'
                      ? 'bg-rose-500'
                      : 'bg-emerald-600'
                  }`}
                  style={{ width: `${task.progress}%` }}
                />
              </div>

              {/* Stats Footer */}
              <div className="flex items-center justify-between text-[10px] text-slate-400 font-mono">
                <span>
                  {formatBytes(task.uploadedBytes)} / {formatBytes(task.totalBytes)} ({task.progress}%)
                </span>
                {task.status === 'uploading' && <span>{task.speed}</span>}
                {task.status === 'completed' && <span className="text-emerald-600 font-bold">Done</span>}
                {task.status === 'failed' && <span className="text-rose-500 font-bold">Failed</span>}
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
};
