'use client';

import React, { useState, useEffect } from 'react';
import {
  Server as ServerIcon,
  Plus,
  Copy,
  Check,
  Terminal,
  Activity,
  Cpu,
  HardDrive,
  Clock,
  Shield,
  RefreshCw,
  Search,
  X,
} from 'lucide-react';
import { DashboardShell } from '@/components/DashboardShell';
import { apiFetch, Server as ServerModel, EnrollmentTokenResponse } from '@/lib/api';

export default function ServersPage() {
  const [servers, setServers] = useState<ServerModel[]>([]);
  const [loading, setLoading] = useState(true);
  const [searchQuery, setSearchQuery] = useState('');
  const [enrollModalOpen, setEnrollModalOpen] = useState(false);
  const [enrollLabel, setEnrollLabel] = useState('');
  const [enrollTokenData, setEnrollTokenData] = useState<EnrollmentTokenResponse | null>(null);
  const [generatingToken, setGeneratingToken] = useState(false);
  const [copied, setCopied] = useState(false);

  const fetchServers = async () => {
    setLoading(true);
    const res = await apiFetch<ServerModel[]>('/api/v1/servers');
    if (res.success && res.data) {
      setServers(res.data);
    }
    setLoading(false);
  };

  useEffect(() => {
    fetchServers();
  }, []);

  const handleGenerateEnrollment = async (e: React.FormEvent) => {
    e.preventDefault();
    setGeneratingToken(true);

    const res = await apiFetch<EnrollmentTokenResponse>('/api/v1/servers/enrollment-tokens', {
      method: 'POST',
      body: JSON.stringify({
        label: enrollLabel || 'Production Node',
        expiration_minutes: 120,
      }),
    });

    setGeneratingToken(false);
    if (res.success && res.data) {
      setEnrollTokenData(res.data);
    }
  };

  const handleCopyCommand = () => {
    if (enrollTokenData) {
      navigator.clipboard.writeText(enrollTokenData.install_command);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    }
  };

  const filteredServers = servers.filter(
    (s) =>
      s.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
      s.hostname.toLowerCase().includes(searchQuery.toLowerCase()) ||
      s.ip_address.includes(searchQuery)
  );

  return (
    <DashboardShell>
      <div className="space-y-8">
        {/* Header */}
        <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
          <div>
            <h1 className="text-2xl font-bold tracking-tight text-white">Server Fleet Management</h1>
            <p className="text-sm text-slate-400 mt-1">
              Connect, monitor, and configure your Linux VPS and dedicated servers.
            </p>
          </div>
          <div className="flex items-center gap-3">
            <button
              onClick={fetchServers}
              className="p-2.5 rounded-xl bg-surface-900 border border-surface-800 text-slate-300 hover:text-white hover:bg-surface-800 transition-colors"
              title="Refresh Fleet"
            >
              <RefreshCw className={`w-4 h-4 ${loading ? 'animate-spin' : ''}`} />
            </button>
            <button
              onClick={() => {
                setEnrollModalOpen(true);
                setEnrollTokenData(null);
                setEnrollLabel('');
              }}
              className="inline-flex items-center gap-2 px-4 py-2.5 rounded-xl bg-indigo-600 hover:bg-indigo-500 text-white text-sm font-semibold shadow-lg shadow-indigo-600/25 transition-all"
            >
              <Plus className="w-4 h-4" />
              Enroll Server
            </button>
          </div>
        </div>

        {/* Filter Bar */}
        <div className="flex flex-col sm:flex-row items-center justify-between gap-4 bg-white dark:bg-[#10141d] border border-slate-200 dark:border-surface-800 rounded-2xl p-3 shadow-xs">
          <div className="flex items-center gap-3 w-full sm:w-96 bg-slate-50 dark:bg-[#121824] border border-slate-300 dark:border-surface-700 rounded-xl px-3.5 py-2 shadow-xs focus-within:border-[#20a53a] focus-within:ring-2 focus-within:ring-[#20a53a]/20 transition-all">
            <Search className="w-4 h-4 text-slate-400 dark:text-slate-500 flex-shrink-0" />
            <input
              type="text"
              placeholder="Search by server name, hostname, or IP address..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              className="w-full bg-transparent text-xs font-medium text-slate-950 dark:text-white placeholder:text-slate-400 focus:outline-none"
            />
            {searchQuery && (
              <button onClick={() => setSearchQuery('')} className="text-slate-400 hover:text-slate-600 dark:hover:text-white p-0.5">
                <X className="w-3.5 h-3.5" />
              </button>
            )}
          </div>
          <div className="text-xs font-semibold text-slate-600 dark:text-slate-400 px-2 self-end sm:self-center">
            Showing {filteredServers.length} of {servers.length} servers
          </div>
        </div>

        {/* Fleet Grid */}
        {loading ? (
          <div className="py-20 text-center text-slate-400">
            <div className="w-8 h-8 border-2 border-indigo-500 border-t-transparent rounded-full animate-spin mx-auto mb-3" />
            Loading managed server fleet...
          </div>
        ) : filteredServers.length === 0 ? (
          <div className="py-16 px-4 rounded-2xl border border-dashed border-surface-800 text-center bg-surface-900/50">
            <div className="w-14 h-14 rounded-2xl bg-surface-800 flex items-center justify-center mx-auto mb-4 text-indigo-400">
              <ServerIcon className="w-7 h-7" />
            </div>
            <h3 className="text-lg font-bold text-white">No Servers Connected</h3>
            <p className="text-sm text-slate-400 mt-1 max-w-md mx-auto">
              You haven&apos;t enrolled any Linux servers yet. Click Enroll Server to install the lightweight Hostvra Agent daemon on any Ubuntu, Debian, or RHEL-based server.
            </p>
            <button
              onClick={() => setEnrollModalOpen(true)}
              className="mt-5 inline-flex items-center gap-2 px-5 py-2.5 rounded-xl bg-indigo-600 hover:bg-indigo-500 text-white text-sm font-semibold shadow-lg shadow-indigo-600/25 transition-all"
            >
              <Plus className="w-4 h-4" />
              Enroll First Server
            </button>
          </div>
        ) : (
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
            {filteredServers.map((server) => (
              <div
                key={server.id}
                className="bg-surface-900 border border-surface-800 rounded-2xl p-6 shadow-xl hover:border-surface-700 transition-all flex flex-col justify-between"
              >
                <div>
                  <div className="flex items-start justify-between gap-3 mb-4">
                    <div className="flex items-center gap-3">
                      <div className="w-10 h-10 rounded-xl bg-indigo-500/10 border border-indigo-500/20 flex items-center justify-center text-indigo-400">
                        <ServerIcon className="w-5 h-5" />
                      </div>
                      <div>
                        <h3 className="font-bold text-base text-white">{server.name}</h3>
                        <p className="text-xs text-slate-400 font-mono">{server.hostname}</p>
                      </div>
                    </div>
                    <span
                      className={`inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-[11px] font-semibold capitalize ${
                        server.status === 'online'
                          ? 'bg-emerald-500/10 text-emerald-400 border border-emerald-500/20'
                          : 'bg-slate-500/10 text-slate-400 border border-slate-500/20'
                      }`}
                    >
                      <span className={`w-1.5 h-1.5 rounded-full ${server.status === 'online' ? 'bg-emerald-500' : 'bg-slate-500'}`} />
                      {server.status}
                    </span>
                  </div>

                  <div className="space-y-2 text-xs py-3 border-y border-surface-800 text-slate-300">
                    <div className="flex justify-between">
                      <span className="text-slate-400">IP Address:</span>
                      <span className="font-mono text-slate-100">{server.ip_address}</span>
                    </div>
                    <div className="flex justify-between">
                      <span className="text-slate-400">Operating System:</span>
                      <span>{server.os_name} {server.os_version}</span>
                    </div>
                    <div className="flex justify-between">
                      <span className="text-slate-400">Hardware Spec:</span>
                      <span>{server.cpu_cores} vCPU / {(server.ram_total_mb / 1024).toFixed(1)} GB RAM</span>
                    </div>
                    <div className="flex justify-between">
                      <span className="text-slate-400">Agent Version:</span>
                      <span className="font-mono text-indigo-400">v{server.agent_version}</span>
                    </div>
                  </div>
                </div>

                <div className="mt-5 flex items-center justify-between pt-2">
                  <span className="text-[11px] text-slate-400 flex items-center gap-1">
                    <Clock className="w-3.5 h-3.5" />
                    Uptime: {Math.floor((server.uptime_seconds || 0) / 3600)}h
                  </span>
                  <button className="text-xs font-semibold px-3 py-1.5 rounded-lg bg-surface-800 hover:bg-surface-700 text-slate-200 transition-colors">
                    Configure Node
                  </button>
                </div>
              </div>
            ))}
          </div>
        )}

        {/* Enrollment Modal */}
        {enrollModalOpen && (
          <div className="fixed inset-0 z-50 bg-black/70 backdrop-blur-sm flex items-center justify-center p-4">
            <div className="w-full max-w-xl bg-surface-900 border border-surface-700 rounded-2xl shadow-2xl p-6 relative">
              <button
                onClick={() => setEnrollModalOpen(false)}
                className="absolute top-5 right-5 p-1 rounded-lg text-slate-400 hover:text-white hover:bg-surface-800 transition-colors"
              >
                <X className="w-5 h-5" />
              </button>

              <div className="flex items-center gap-3 mb-4">
                <div className="w-10 h-10 rounded-xl bg-indigo-500/10 border border-indigo-500/20 flex items-center justify-center text-indigo-400">
                  <Terminal className="w-5 h-5" />
                </div>
                <div>
                  <h2 className="text-lg font-bold text-white">Enroll New Server</h2>
                  <p className="text-xs text-slate-400">Deploy Hostvra Agent via one-command curl installer</p>
                </div>
              </div>

              {!enrollTokenData ? (
                <form onSubmit={handleGenerateEnrollment} className="space-y-4 pt-2">
                  <div>
                    <label className="block text-xs font-semibold text-slate-300 uppercase tracking-wider mb-1.5">
                      Server Label / Environment
                    </label>
                    <input
                      type="text"
                      required
                      value={enrollLabel}
                      onChange={(e) => setEnrollLabel(e.target.value)}
                      placeholder="e.g. Production Web US-East-1"
                      className="w-full px-3.5 py-2.5 rounded-xl bg-surface-950 border border-surface-700 text-slate-100 placeholder-slate-500 text-sm focus:outline-none focus:border-indigo-500"
                    />
                  </div>

                  <p className="text-xs text-slate-400 leading-relaxed">
                    This generates a cryptographically signed, single-use enrollment token valid for 2 hours. Once the agent executes on the target server, it exchanges this token for a permanent mutual authentication key.
                  </p>

                  <div className="flex justify-end gap-3 pt-3">
                    <button
                      type="button"
                      onClick={() => setEnrollModalOpen(false)}
                      className="px-4 py-2 rounded-xl text-sm text-slate-400 hover:text-white transition-colors"
                    >
                      Cancel
                    </button>
                    <button
                      type="submit"
                      disabled={generatingToken}
                      className="px-5 py-2 rounded-xl bg-indigo-600 hover:bg-indigo-500 disabled:bg-indigo-600/50 text-white text-sm font-semibold shadow-md transition-all flex items-center gap-2"
                    >
                      {generatingToken ? (
                        <div className="w-4 h-4 border-2 border-white border-t-transparent rounded-full animate-spin" />
                      ) : (
                        'Generate Enrollment Command'
                      )}
                    </button>
                  </div>
                </form>
              ) : (
                <div className="space-y-4 pt-2">
                  <div className="p-4 rounded-xl bg-emerald-500/10 border border-emerald-500/20 text-emerald-400 text-xs flex items-center gap-2">
                    <Check className="w-4 h-4 flex-shrink-0" />
                    <span>Enrollment token generated successfully. Valid until {new Date(enrollTokenData.expires_at).toLocaleTimeString()}.</span>
                  </div>

                  <div>
                    <label className="block text-xs font-semibold text-slate-300 uppercase tracking-wider mb-2">
                      Run this on your clean Linux server as root:
                    </label>
                    <div className="relative group">
                      <pre className="p-4 rounded-xl bg-surface-950 border border-surface-700 font-mono text-xs text-indigo-300 overflow-x-auto whitespace-pre-wrap select-all">
                        {enrollTokenData.install_command}
                      </pre>
                      <button
                        onClick={handleCopyCommand}
                        className="absolute top-2.5 right-2.5 px-3 py-1.5 rounded-lg bg-surface-800 hover:bg-surface-700 text-white text-xs font-medium flex items-center gap-1.5 shadow-md transition-all"
                      >
                        {copied ? <Check className="w-3.5 h-3.5 text-emerald-400" /> : <Copy className="w-3.5 h-3.5" />}
                        <span>{copied ? 'Copied!' : 'Copy'}</span>
                      </button>
                    </div>
                  </div>

                  <div className="p-3.5 rounded-xl bg-surface-950/60 border border-surface-800 text-[11px] text-slate-400 space-y-1">
                    <p className="font-semibold text-slate-300">Target Requirements:</p>
                    <p>• Supported OS: Ubuntu 22.04/24.04 LTS, Debian 12, AlmaLinux 9, Rocky Linux 9</p>
                    <p>• Architecture: x86_64, arm64</p>
                    <p>• Minimum Hardware: 1 vCPU, 1 GB RAM, 10 GB Disk</p>
                  </div>

                  <div className="flex justify-end pt-2">
                    <button
                      type="button"
                      onClick={() => {
                        setEnrollModalOpen(false);
                        fetchServers();
                      }}
                      className="px-5 py-2 rounded-xl bg-surface-800 hover:bg-surface-700 text-white text-sm font-semibold transition-all"
                    >
                      Done & Close
                    </button>
                  </div>
                </div>
              )}
            </div>
          </div>
        )}
      </div>
    </DashboardShell>
  );
}
