'use client';

import React, { useState, useEffect } from 'react';
import {
  Clock,
  X,
  Search,
  Filter,
  RefreshCw,
  Folder,
  FileText,
  Upload,
  Download,
  Trash2,
  RotateCcw,
  Scissors,
  Copy,
  Lock,
  Archive,
} from 'lucide-react';
import { apiFetch } from '@/lib/api';

export interface FileManagerLogItem {
  id: string;
  user_email: string;
  ip_address: string;
  browser: string;
  domain: string;
  action: string;
  source_path: string;
  destination_path: string;
  created_at: string;
  details?: Record<string, any>;
}

interface ActivityLogsModalProps {
  domain?: string;
  onClose: () => void;
}

export const ActivityLogsModal: React.FC<ActivityLogsModalProps> = ({ domain, onClose }) => {
  const [logs, setLogs] = useState<FileManagerLogItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [search, setSearch] = useState('');
  const [selectedAction, setSelectedAction] = useState('all');

  const fetchLogs = async () => {
    try {
      setLoading(true);
      const q = domain ? `?domain=${encodeURIComponent(domain)}&limit=100` : '?limit=100';
      const res = await apiFetch<{ logs: FileManagerLogItem[] }>(`/api/v1/filemanager/activity-logs${q}`);
      if (res.success && res.data) {
        setLogs(res.data.logs || []);
      }
    } catch (err) {
      console.error('Failed to load activity logs', err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchLogs();
  }, [domain]);

  const filteredLogs = logs.filter((l) => {
    if (selectedAction !== 'all' && l.action !== selectedAction) return false;
    if (!search) return true;
    const q = search.toLowerCase();
    return (
      l.action.toLowerCase().includes(q) ||
      l.source_path.toLowerCase().includes(q) ||
      l.destination_path.toLowerCase().includes(q) ||
      l.user_email.toLowerCase().includes(q) ||
      l.ip_address.includes(q)
    );
  });

  const getActionIcon = (action: string) => {
    switch (action) {
      case 'upload':
        return <Upload className="w-3.5 h-3.5 text-sky-500" />;
      case 'download':
        return <Download className="w-3.5 h-3.5 text-emerald-500" />;
      case 'move':
        return <Scissors className="w-3.5 h-3.5 text-amber-500" />;
      case 'copy':
        return <Copy className="w-3.5 h-3.5 text-blue-500" />;
      case 'trash':
      case 'delete':
      case 'empty_trash':
        return <Trash2 className="w-3.5 h-3.5 text-rose-500" />;
      case 'restore':
        return <RotateCcw className="w-3.5 h-3.5 text-emerald-600" />;
      case 'permissions':
        return <Lock className="w-3.5 h-3.5 text-purple-500" />;
      case 'archive':
      case 'extract':
        return <Archive className="w-3.5 h-3.5 text-indigo-500" />;
      default:
        return <FileText className="w-3.5 h-3.5 text-slate-400" />;
    }
  };

  return (
    <div className="fixed inset-0 z-50 bg-black/60 backdrop-blur-xs flex items-center justify-center p-4 animate-fadeIn">
      <div className="w-full max-w-4xl bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-700 rounded-2xl shadow-2xl p-6 relative max-h-[85vh] flex flex-col font-sans">
        <button
          type="button"
          onClick={onClose}
          className="absolute top-5 right-5 p-1 rounded-lg text-slate-400 hover:text-slate-700 dark:hover:text-white"
        >
          <X className="w-5 h-5" />
        </button>

        {/* Header */}
        <div className="flex items-center justify-between mb-4">
          <div className="flex items-center gap-3">
            <div className="w-10 h-10 rounded-xl bg-slate-100 dark:bg-surface-800 text-slate-700 dark:text-slate-300 flex items-center justify-center">
              <Clock className="w-5 h-5" />
            </div>
            <div>
              <h2 className="text-base font-bold text-slate-900 dark:text-white">
                File Manager Activity & Audit Logs
              </h2>
              <p className="text-xs text-slate-500">
                Tracking all file operations, restores, moves, and deletions
              </p>
            </div>
          </div>

          <button
            type="button"
            onClick={fetchLogs}
            className="mr-8 p-1.5 rounded-lg border border-slate-200 dark:border-surface-700 hover:bg-slate-100 text-slate-600 dark:text-slate-400"
            title="Refresh Logs"
          >
            <RefreshCw className={`w-4 h-4 ${loading ? 'animate-spin' : ''}`} />
          </button>
        </div>

        {/* Filter bar */}
        <div className="flex flex-wrap items-center gap-2.5 mb-4">
          <div className="relative flex-1 min-w-[200px]">
            <Search className="w-3.5 h-3.5 absolute left-3 top-1/2 -translate-y-1/2 text-slate-400" />
            <input
              type="text"
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              placeholder="Search by path, user, IP, or action..."
              className="w-full pl-9 pr-3 py-1.5 text-xs rounded-xl bg-slate-50 dark:bg-surface-800 border border-slate-200 dark:border-surface-700 text-slate-900 dark:text-white focus:ring-1 focus:ring-emerald-500"
            />
          </div>

          <select
            value={selectedAction}
            onChange={(e) => setSelectedAction(e.target.value)}
            className="px-3 py-1.5 text-xs rounded-xl bg-slate-50 dark:bg-surface-800 border border-slate-200 dark:border-surface-700 text-slate-800 dark:text-slate-200 font-semibold focus:ring-1 focus:ring-emerald-500"
          >
            <option value="all">All Actions</option>
            <option value="upload">Upload</option>
            <option value="download">Download</option>
            <option value="move">Move</option>
            <option value="copy">Copy</option>
            <option value="rename">Rename</option>
            <option value="trash">Move to Trash</option>
            <option value="restore">Restore</option>
            <option value="empty_trash">Empty Trash</option>
            <option value="permissions">Permissions</option>
          </select>
        </div>

        {/* Logs Table */}
        <div className="flex-1 overflow-auto rounded-xl border border-slate-200 dark:border-surface-700">
          <table className="w-full text-left text-xs">
            <thead className="bg-slate-50 dark:bg-surface-800 text-slate-500 font-bold uppercase text-[10px] tracking-wider border-b border-slate-200 dark:border-surface-700 sticky top-0">
              <tr>
                <th className="py-2.5 px-3">Action</th>
                <th className="py-2.5 px-3">Source / Target Path</th>
                <th className="py-2.5 px-3">User & IP</th>
                <th className="py-2.5 px-3 text-right">Timestamp</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-100 dark:divide-surface-800 font-sans">
              {loading && logs.length === 0 ? (
                <tr>
                  <td colSpan={4} className="py-8 text-center text-slate-400">
                    Loading activity timeline...
                  </td>
                </tr>
              ) : filteredLogs.length === 0 ? (
                <tr>
                  <td colSpan={4} className="py-8 text-center text-slate-400">
                    No activity logs found
                  </td>
                </tr>
              ) : (
                filteredLogs.map((log) => (
                  <tr
                    key={log.id}
                    className="hover:bg-slate-50/60 dark:hover:bg-surface-800/50 transition"
                  >
                    <td className="py-2.5 px-3">
                      <div className="flex items-center gap-1.5 font-bold capitalize">
                        {getActionIcon(log.action)}
                        <span className="text-slate-800 dark:text-slate-200">{log.action}</span>
                      </div>
                    </td>
                    <td className="py-2.5 px-3">
                      <div className="font-mono text-[11px] text-slate-700 dark:text-slate-300 max-w-md truncate">
                        {log.source_path}
                        {log.destination_path && (
                          <span className="text-slate-400"> → {log.destination_path}</span>
                        )}
                      </div>
                    </td>
                    <td className="py-2.5 px-3">
                      <div className="text-slate-700 dark:text-slate-300 font-medium">
                        {log.user_email || 'System'}
                      </div>
                      <div className="text-[10px] text-slate-400 font-mono">{log.ip_address}</div>
                    </td>
                    <td className="py-2.5 px-3 text-right text-slate-500 font-mono text-[11px]">
                      {new Date(log.created_at).toLocaleString()}
                    </td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>

        {/* Footer */}
        <div className="flex items-center justify-between pt-3 border-t border-slate-200 dark:border-surface-800 text-xs text-slate-500 mt-3">
          <span>Showing {filteredLogs.length} events</span>
          <button
            type="button"
            onClick={onClose}
            className="px-4 py-1.5 rounded-xl bg-slate-100 hover:bg-slate-200 dark:bg-surface-800 dark:hover:bg-surface-700 text-slate-700 dark:text-slate-300 font-bold transition"
          >
            Close
          </button>
        </div>
      </div>
    </div>
  );
};
