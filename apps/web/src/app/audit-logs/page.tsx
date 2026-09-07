'use client';

import React, { useState, useEffect } from 'react';
import {
  ScrollText,
  Search,
  RefreshCw,
  ShieldAlert,
  CheckCircle2,
  XCircle,
  Clock,
  Filter,
  X,
} from 'lucide-react';
import { DashboardShell } from '@/components/DashboardShell';
import { apiFetch, AuditLog } from '@/lib/api';

export default function AuditLogsPage() {
  const [logs, setLogs] = useState<AuditLog[]>([]);
  const [loading, setLoading] = useState(true);
  const [search, setSearch] = useState('');

  const fetchLogs = async () => {
    setLoading(true);
    const res = await apiFetch<AuditLog[]>('/api/v1/audit-logs?limit=100');
    if (res.success && res.data) {
      setLogs(res.data);
    }
    setLoading(false);
  };

  useEffect(() => {
    fetchLogs();
  }, []);

  const filteredLogs = logs.filter((l) =>
    l.action.toLowerCase().includes(search.toLowerCase()) ||
    l.resource_type.toLowerCase().includes(search.toLowerCase()) ||
    (l.ip_address && l.ip_address.includes(search))
  );

  return (
    <DashboardShell>
      <div className="space-y-6 animate-fadeIn max-w-7xl mx-auto pb-12">
        <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
          <div>
            <h1 className="text-2xl font-bold tracking-tight text-slate-950 dark:text-white">Security & Audit Logs</h1>
            <p className="text-sm text-slate-600 dark:text-slate-400 mt-1">
              Immutable historical event log of administrative actions, logins, and system changes.
            </p>
          </div>
          <button
            onClick={fetchLogs}
            className="p-2.5 rounded-xl bg-white dark:bg-surface-900 border border-slate-300 dark:border-surface-800 text-slate-700 dark:text-slate-300 hover:text-slate-950 dark:hover:text-white hover:bg-slate-100 dark:hover:bg-surface-800 transition-colors self-start shadow-xs"
            title="Refresh logs"
          >
            <RefreshCw className={`w-4 h-4 ${loading ? 'animate-spin' : ''}`} />
          </button>
        </div>

        {/* Filter Bar */}
        <div className="flex flex-col sm:flex-row items-center justify-between gap-4 bg-white dark:bg-[#10141d] border border-slate-200 dark:border-surface-800 rounded-2xl p-3 shadow-xs">
          <div className="flex items-center gap-3 w-full sm:w-96 bg-slate-50 dark:bg-[#121824] border border-slate-300 dark:border-surface-700 rounded-xl px-3.5 py-2 shadow-xs focus-within:border-[#20a53a] focus-within:ring-2 focus-within:ring-[#20a53a]/20 transition-all">
            <Search className="w-4 h-4 text-slate-400 dark:text-slate-500 flex-shrink-0" />
            <input
              type="text"
              placeholder="Search audit actions (e.g. auth.login, server.enrolled)..."
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
            Showing {filteredLogs.length} of {logs.length} events
          </div>
        </div>

        {/* Logs Table */}
        <div className="bg-white dark:bg-[#10141d] border border-slate-200 dark:border-surface-800 rounded-2xl overflow-hidden shadow-xs dark:shadow-xl">
          {loading ? (
            <div className="py-20 text-center text-slate-500 dark:text-slate-400">
              <div className="w-8 h-8 border-2 border-[#20a53a] border-t-transparent rounded-full animate-spin mx-auto mb-3" />
              Loading security audit stream...
            </div>
          ) : filteredLogs.length === 0 ? (
            <div className="py-16 text-center text-slate-500 dark:text-slate-400">
              <ScrollText className="w-10 h-10 mx-auto text-slate-400 dark:text-slate-600 mb-2" />
              <p className="text-sm font-semibold text-slate-900 dark:text-slate-300">No audit events match your query</p>
            </div>
          ) : (
            <div className="overflow-x-auto">
              <table className="w-full text-left text-sm border-collapse">
                <thead>
                  <tr className="border-b border-slate-200 dark:border-surface-800 bg-slate-50 dark:bg-[#121824] text-slate-700 dark:text-slate-300 text-[11px] font-bold uppercase tracking-wider">
                    <th className="px-6 py-3.5">Action</th>
                    <th className="px-6 py-3.5">Target Resource</th>
                    <th className="px-6 py-3.5">Status</th>
                    <th className="px-6 py-3.5">Origin IP</th>
                    <th className="px-6 py-3.5 text-right">Timestamp</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-slate-200/80 dark:divide-surface-800/80">
                  {filteredLogs.map((log) => (
                    <tr key={log.id} className="hover:bg-slate-50 dark:hover:bg-[#151d2d] transition-colors">
                      <td className="px-6 py-3.5 font-mono text-xs font-bold text-slate-950 dark:text-white">
                        {log.action}
                      </td>
                      <td className="px-6 py-3.5 text-xs text-slate-700 dark:text-slate-300 font-mono">
                        <span className="px-2 py-0.5 rounded bg-slate-100 dark:bg-surface-800 text-slate-800 dark:text-slate-300 font-medium border border-slate-200 dark:border-surface-700">
                          {log.resource_type}
                        </span>
                        {log.resource_id && (
                          <span className="text-slate-500 dark:text-slate-400 ml-2">ID: {log.resource_id.substring(0, 8)}...</span>
                        )}
                      </td>
                      <td className="px-6 py-3.5">
                        <span
                          className={`inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full text-[11px] font-semibold capitalize ${
                            log.status === 'success'
                              ? 'bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 border border-emerald-500/20'
                              : 'bg-rose-500/10 text-rose-600 dark:text-rose-400 border border-rose-500/20'
                          }`}
                        >
                          {log.status === 'success' ? (
                            <CheckCircle2 className="w-3 h-3" />
                          ) : (
                            <XCircle className="w-3 h-3" />
                          )}
                          {log.status}
                        </span>
                      </td>
                      <td className="px-6 py-3.5 font-mono text-xs font-medium text-slate-700 dark:text-slate-400">
                        {log.ip_address || '127.0.0.1'}
                      </td>
                      <td className="px-6 py-3.5 text-xs text-slate-600 dark:text-slate-400 text-right font-mono font-medium">
                        {new Date(log.created_at).toLocaleString()}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </div>
      </div>
    </DashboardShell>
  );
}
