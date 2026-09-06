'use client';

import React, { useState, useEffect } from 'react';
import Link from 'next/link';
import {
  Server,
  Activity,
  Globe,
  ShieldCheck,
  Plus,
  ArrowUpRight,
  HardDrive,
  Cpu,
  Layers,
  ScrollText,
  Clock,
  CheckCircle2,
  AlertTriangle,
} from 'lucide-react';
import { DashboardShell } from '@/components/DashboardShell';
import { apiFetch, Server as ServerModel, AuditLog } from '@/lib/api';

export default function DashboardPage() {
  const [servers, setServers] = useState<ServerModel[]>([]);
  const [auditLogs, setAuditLogs] = useState<AuditLog[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    async function fetchData() {
      try {
        const [serversRes, auditRes] = await Promise.all([
          apiFetch<ServerModel[]>('/api/v1/servers'),
          apiFetch<AuditLog[]>('/api/v1/audit-logs?limit=5'),
        ]);

        if (serversRes.success && serversRes.data) {
          setServers(serversRes.data);
        }
        if (auditRes.success && auditRes.data) {
          setAuditLogs(auditRes.data);
        }
      } catch (err) {
        console.error('Failed to load dashboard data', err);
      } finally {
        setLoading(false);
      }
    }
    fetchData();
  }, []);

  const onlineServers = servers.filter((s) => s.status === 'online').length;

  return (
    <DashboardShell>
      <div className="space-y-8">
        {/* Page Header */}
        <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
          <div>
            <h1 className="text-2xl font-bold tracking-tight text-white">Cluster Overview</h1>
            <p className="text-sm text-slate-400 mt-1">
              Real-time telemetry and management across your Hostvra server fleet.
            </p>
          </div>
          <div className="flex items-center gap-3">
            <Link
              href="/servers"
              className="inline-flex items-center gap-2 px-4 py-2.5 rounded-xl bg-indigo-600 hover:bg-indigo-500 text-white text-sm font-semibold shadow-lg shadow-indigo-600/25 transition-all"
            >
              <Plus className="w-4 h-4" />
              Enroll Server
            </Link>
          </div>
        </div>

        {/* Fleet KPI Metric Cards */}
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-5">
          <div className="p-5 rounded-2xl bg-surface-900 border border-surface-800 shadow-xl flex items-center justify-between">
            <div>
              <p className="text-xs font-semibold text-slate-400 uppercase tracking-wider">Managed Nodes</p>
              <div className="flex items-baseline gap-2 mt-2">
                <span className="text-3xl font-extrabold text-white">{servers.length}</span>
                <span className="text-xs text-emerald-400 font-medium">
                  {onlineServers} Online
                </span>
              </div>
            </div>
            <div className="w-12 h-12 rounded-xl bg-indigo-500/10 border border-indigo-500/20 flex items-center justify-center text-indigo-400">
              <Server className="w-6 h-6" />
            </div>
          </div>

          <div className="p-5 rounded-2xl bg-surface-900 border border-surface-800 shadow-xl flex items-center justify-between">
            <div>
              <p className="text-xs font-semibold text-slate-400 uppercase tracking-wider">Websites & Vhosts</p>
              <div className="flex items-baseline gap-2 mt-2">
                <span className="text-3xl font-extrabold text-white">0</span>
                <span className="text-xs text-slate-400">Nginx Managed</span>
              </div>
            </div>
            <div className="w-12 h-12 rounded-xl bg-emerald-500/10 border border-emerald-500/20 flex items-center justify-center text-emerald-400">
              <Globe className="w-6 h-6" />
            </div>
          </div>

          <div className="p-5 rounded-2xl bg-surface-900 border border-surface-800 shadow-xl flex items-center justify-between">
            <div>
              <p className="text-xs font-semibold text-slate-400 uppercase tracking-wider">SSL Certificates</p>
              <div className="flex items-baseline gap-2 mt-2">
                <span className="text-3xl font-extrabold text-white">100%</span>
                <span className="text-xs text-emerald-400">Automated ACME</span>
              </div>
            </div>
            <div className="w-12 h-12 rounded-xl bg-purple-500/10 border border-purple-500/20 flex items-center justify-center text-purple-400">
              <ShieldCheck className="w-6 h-6" />
            </div>
          </div>

          <div className="p-5 rounded-2xl bg-surface-900 border border-surface-800 shadow-xl flex items-center justify-between">
            <div>
              <p className="text-xs font-semibold text-slate-400 uppercase tracking-wider">Platform Edition</p>
              <div className="flex items-baseline gap-2 mt-2">
                <span className="text-xl font-extrabold text-white capitalize">Community</span>
                <span className="text-[11px] font-semibold px-1.5 py-0.5 rounded bg-indigo-500/20 text-indigo-300">
                  v1.0.0
                </span>
              </div>
            </div>
            <div className="w-12 h-12 rounded-xl bg-sky-500/10 border border-sky-500/20 flex items-center justify-center text-sky-400">
              <Activity className="w-6 h-6" />
            </div>
          </div>
        </div>

        {/* Server Fleet Grid / Table Preview */}
        <div className="bg-surface-900 border border-surface-800 rounded-2xl p-6 shadow-xl">
          <div className="flex items-center justify-between mb-5">
            <div className="flex items-center gap-2">
              <Server className="w-5 h-5 text-indigo-400" />
              <h2 className="text-lg font-bold text-white">Registered Server Fleet</h2>
            </div>
            <Link
              href="/servers"
              className="text-xs font-medium text-indigo-400 hover:text-indigo-300 flex items-center gap-1 transition-colors"
            >
              <span>View all servers</span>
              <ArrowUpRight className="w-3.5 h-3.5" />
            </Link>
          </div>

          {loading ? (
            <div className="py-12 text-center text-slate-400 text-sm">
              <div className="w-6 h-6 border-2 border-indigo-500 border-t-transparent rounded-full animate-spin mx-auto mb-2" />
              Loading server telemetry...
            </div>
          ) : servers.length === 0 ? (
            <div className="py-12 px-4 rounded-xl border border-dashed border-surface-700 text-center bg-surface-950/40">
              <div className="w-12 h-12 rounded-2xl bg-surface-800 flex items-center justify-center mx-auto mb-3 text-slate-400">
                <Server className="w-6 h-6" />
              </div>
              <h3 className="text-base font-semibold text-slate-200">No servers enrolled yet</h3>
              <p className="text-xs text-slate-400 mt-1 max-w-sm mx-auto">
                Connect your first Linux VPS or dedicated server using the Hostvra one-command installer.
              </p>
              <Link
                href="/servers"
                className="mt-4 inline-flex items-center gap-2 px-4 py-2 rounded-xl bg-indigo-600 hover:bg-indigo-500 text-white text-xs font-semibold shadow-md transition-all"
              >
                <Plus className="w-3.5 h-3.5" />
                Add Your First Server
              </Link>
            </div>
          ) : (
            <div className="overflow-x-auto">
              <table className="w-full text-left text-sm">
                <thead>
                  <tr className="border-b border-surface-800 text-slate-400 text-xs uppercase tracking-wider">
                    <th className="pb-3 font-semibold">Server Name</th>
                    <th className="pb-3 font-semibold">IP Address</th>
                    <th className="pb-3 font-semibold">OS / Arch</th>
                    <th className="pb-3 font-semibold">Status</th>
                    <th className="pb-3 font-semibold">Agent</th>
                    <th className="pb-3 font-semibold text-right">Actions</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-surface-800/60">
                  {servers.map((server) => (
                    <tr key={server.id} className="hover:bg-surface-800/40 transition-colors">
                      <td className="py-3.5 font-medium text-white flex items-center gap-2.5">
                        <span className={`w-2 h-2 rounded-full ${server.status === 'online' ? 'bg-emerald-500' : 'bg-slate-500'}`} />
                        <div>
                          <div>{server.name}</div>
                          <div className="text-xs text-slate-400 font-mono font-normal">{server.hostname}</div>
                        </div>
                      </td>
                      <td className="py-3.5 font-mono text-xs text-slate-300">{server.ip_address}</td>
                      <td className="py-3.5 text-xs text-slate-300">
                        {server.os_name} {server.os_version} ({server.architecture})
                      </td>
                      <td className="py-3.5">
                        <span
                          className={`inline-flex items-center px-2 py-0.5 rounded text-[11px] font-semibold capitalize ${
                            server.status === 'online'
                              ? 'bg-emerald-500/10 text-emerald-400 border border-emerald-500/20'
                              : 'bg-slate-500/10 text-slate-400 border border-slate-500/20'
                          }`}
                        >
                          {server.status}
                        </span>
                      </td>
                      <td className="py-3.5 text-xs font-mono text-slate-400">v{server.agent_version}</td>
                      <td className="py-3.5 text-right">
                        <Link
                          href={`/servers?id=${server.id}`}
                          className="text-xs text-indigo-400 hover:text-indigo-300 font-medium"
                        >
                          Manage →
                        </Link>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </div>

        {/* Security & Audit Trail Preview */}
        <div className="bg-surface-900 border border-surface-800 rounded-2xl p-6 shadow-xl">
          <div className="flex items-center justify-between mb-4">
            <div className="flex items-center gap-2">
              <ScrollText className="w-5 h-5 text-indigo-400" />
              <h2 className="text-lg font-bold text-white">Recent Security Audit Trail</h2>
            </div>
            <Link
              href="/audit-logs"
              className="text-xs font-medium text-indigo-400 hover:text-indigo-300 flex items-center gap-1 transition-colors"
            >
              <span>Full Audit Stream</span>
              <ArrowUpRight className="w-3.5 h-3.5" />
            </Link>
          </div>

          {auditLogs.length === 0 ? (
            <p className="text-xs text-slate-400 py-4">No recent security events recorded.</p>
          ) : (
            <div className="divide-y divide-surface-800/60">
              {auditLogs.map((log) => (
                <div key={log.id} className="py-3 flex items-center justify-between text-xs">
                  <div className="flex items-center gap-3">
                    <span
                      className={`w-1.5 h-1.5 rounded-full ${
                        log.status === 'success' ? 'bg-emerald-500' : 'bg-rose-500'
                      }`}
                    />
                    <span className="font-mono font-medium text-slate-200">{log.action}</span>
                    <span className="text-slate-400 font-mono">[{log.resource_type}]</span>
                  </div>
                  <div className="flex items-center gap-4 text-slate-400">
                    {log.ip_address && <span className="font-mono">{log.ip_address}</span>}
                    <span>{new Date(log.created_at).toLocaleTimeString()}</span>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      </div>
    </DashboardShell>
  );
}
