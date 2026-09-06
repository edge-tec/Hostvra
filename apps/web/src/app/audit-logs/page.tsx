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

  const filteredLogs = logs.filter(
    (l) =>
      l.action.toLowerCase().includes(search.toLowerCase()) ||
      l.resource_type.toLowerCase().includes(search.toLowerCase()) ||
      (l.ip_address && l.ip_address.includes(search))
  );

  return (
    <DashboardShell>
      <div className="space-y-8">
        <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
          <div>
            <h1 className="text-2xl font-bold tracking-tight text-white">Security & Audit Trails</h1>
            <p className="text-sm text-slate-400 mt-1">
              Immutable, tamper-resistant record of administrative actions and access logs across the cluster.
            </p>
          </div>
          <button
            onClick={fetchLogs}
            className="p-2.5 rounded-xl bg-surface-900 border border-surface-800 text-slate-300 hover:text-white hover:bg-surface-800 transition-colors self-start"
            title="Refresh logs"
          >
            <RefreshCw className={`w-4 h-4 ${loading ? 'animate-spin' : ''}`} />
          </button>
        </div>

        {/* Filter Bar */}
        <div className="flex items-center gap-3 bg-surface-900 border border-surface-800 rounded-xl px-4 py-2.5">
          <Search className="w-4 h-4 text-slate-500" />
          <input
            type="text"
            placeholder="Search audit actions (e.g. auth.login, server.enrolled)..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="w-full bg-transparent text-sm text-slate-200 placeholder-slate-500 focus:outline-none"
          />
        </div>

        {/* Logs Table */}
        <div className="bg-surface-900 border border-surface-800 rounded-2xl overflow-hidden shadow-xl">
          {loading ? (
            <div className="py-20 text-center text-slate-400">
              <div className="w-8 h-8 border-2 border-indigo-500 border-t-transparent rounded-full animate-spin mx-auto mb-3" />
              Loading security audit stream...
            </div>
          ) : filteredLogs.length === 0 ? (
            <div className="py-16 text-center text-slate-400">
              <ScrollText className="w-10 h-10 mx-auto text-slate-600 mb-2" />
              <p className="text-sm font-medium text-slate-300">No audit events match your query</p>
            </div>
          ) : (
            <div className="overflow-x-auto">
              <table className="w-full text-left text-sm">
                <thead>
                  <tr className="border-b border-surface-800 bg-surface-950/40 text-slate-400 text-xs uppercase tracking-wider">
                    <th className="px-6 py-3.5 font-semibold">Action</th>
                    <th className="px-6 py-3.5 font-semibold">Target Resource</th>
                    <th className="px-6 py-3.5 font-semibold">Status</th>
                    <th className="px-6 py-3.5 font-semibold">Origin IP</th>
                    <th className="px-6 py-3.5 font-semibold text-right">Timestamp</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-surface-800/60">
                  {filteredLogs.map((log) => (
                    <tr key={log.id} className="hover:bg-surface-800/30 transition-colors">
                      <td className="px-6 py-3.5 font-mono text-xs font-semibold text-white">
                        {log.action}
                      </td>
                      <td className="px-6 py-3.5 text-xs text-slate-300 font-mono">
                        <span className="px-2 py-0.5 rounded bg-surface-800 text-slate-300">
                          {log.resource_type}
                        </span>
                        {log.resource_id && (
                          <span className="text-slate-400 ml-2">ID: {log.resource_id.substring(0, 8)}...</span>
                        )}
                      </td>
                      <td className="px-6 py-3.5">
                        <span
                          className={`inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full text-[11px] font-semibold capitalize ${
                            log.status === 'success'
                              ? 'bg-emerald-500/10 text-emerald-400 border border-emerald-500/20'
                              : 'bg-rose-500/10 text-rose-400 border border-rose-500/20'
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
                      <td className="px-6 py-3.5 font-mono text-xs text-slate-400">
                        {log.ip_address || '127.0.0.1'}
                      </td>
                      <td className="px-6 py-3.5 text-xs text-slate-400 text-right font-mono">
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
