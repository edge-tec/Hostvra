'use client';

import React, { useState, useEffect } from 'react';
import { DashboardShell } from '@/components/DashboardShell';
import {
  KeyRound,
  Plus,
  Trash2,
  Copy,
  Check,
  AlertTriangle,
  Clock,
  Terminal,
  RefreshCw,
  X,
  Shield,
  Search,
} from 'lucide-react';
import { apiFetch } from '@/lib/api';

interface APIKeyRecord {
  id: string;
  name: string;
  key_prefix: string;
  full_secret?: string;
  role: string;
  created_at: string;
  last_used_at?: string;
}

export default function APIKeysPage() {
  const [keys, setKeys] = useState<APIKeyRecord[]>([]);
  const [loading, setLoading] = useState(true);
  const [search, setSearch] = useState('');
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [createdSecret, setCreatedSecret] = useState<string | null>(null);
  const [copied, setCopied] = useState(false);
  const [actionLoading, setActionLoading] = useState(false);

  // Form state
  const [keyName, setKeyName] = useState('');
  const [keyRole, setKeyRole] = useState<'admin' | 'manager' | 'viewer'>('manager');

  const filteredKeys = keys.filter(
    (k) =>
      k.name.toLowerCase().includes(search.toLowerCase()) ||
      k.key_prefix.toLowerCase().includes(search.toLowerCase()) ||
      k.role.toLowerCase().includes(search.toLowerCase())
  );

  const loadKeys = async () => {
    setLoading(true);
    try {
      const res = await apiFetch<APIKeyRecord[]>('/api/v1/api-keys');
      if (res.success && Array.isArray(res.data)) {
        setKeys(res.data);
      }
    } catch (err) {
      console.error('Failed to load API keys', err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadKeys();
  }, []);

  const handleCreateKey = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!keyName.trim()) return;

    setActionLoading(true);
    try {
      const res = await apiFetch<APIKeyRecord>('/api/v1/api-keys', {
        method: 'POST',
        body: JSON.stringify({
          name: keyName.trim(),
          role: keyRole,
        }),
      });

      if (res.success && res.data) {
        setCreatedSecret(res.data.full_secret || null);
        setKeys((prev) => [res.data!, ...prev]);
        setKeyName('');
      }
    } finally {
      setActionLoading(false);
    }
  };

  const handleCopy = () => {
    if (createdSecret) {
      navigator.clipboard.writeText(createdSecret);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    }
  };

  const handleRevoke = async (id: string) => {
    if (!confirm('Are you sure you want to revoke this API token? Any active automations using it will stop working immediately.')) {
      return;
    }

    try {
      const res = await apiFetch(`/api/v1/api-keys/${id}`, {
        method: 'DELETE',
      });
      if (res.success) {
        setKeys((prev) => prev.filter((k) => k.id !== id));
      }
    } catch (err) {
      console.error('Failed to revoke API key', err);
    }
  };

  const formatDate = (dateStr?: string) => {
    if (!dateStr) return 'Never';
    try {
      const d = new Date(dateStr);
      return d.toLocaleDateString(undefined, {
        month: 'short',
        day: 'numeric',
        year: 'numeric',
      });
    } catch {
      return dateStr;
    }
  };

  return (
    <DashboardShell>
      <div className="space-y-6">
        {/* Page Header */}
        <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
          <div>
            <h1 className="text-2xl font-bold tracking-tight text-slate-950 dark:text-white flex items-center gap-2.5">
              <span className="p-2 rounded-xl bg-indigo-500/10 text-indigo-600 dark:text-indigo-400">
                <KeyRound className="w-6 h-6" />
              </span>
              API Access Tokens
            </h1>
            <p className="text-sm text-slate-600 dark:text-slate-400 mt-1">
              Bearer tokens for programmatic automation, CLI scripts, Terraform, and CI/CD pipelines.
            </p>
          </div>
          <div className="flex items-center gap-2">
            <button
              onClick={loadKeys}
              disabled={loading}
              className="p-2.5 rounded-xl bg-white dark:bg-surface-800 hover:bg-slate-100 dark:hover:bg-surface-700 text-slate-700 dark:text-slate-300 border border-slate-300 dark:border-surface-700 transition-colors shadow-xs"
              title="Refresh Keys"
            >
              <RefreshCw className={`w-4 h-4 ${loading ? 'animate-spin' : ''}`} />
            </button>
            <button
              onClick={() => {
                setCreatedSecret(null);
                setShowCreateModal(true);
              }}
              className="inline-flex items-center gap-2 px-4 py-2.5 rounded-xl bg-indigo-600 hover:bg-indigo-500 text-white text-sm font-semibold shadow-lg shadow-indigo-600/25 transition-all"
            >
              <Plus className="w-4 h-4" />
              Generate New Token
            </button>
          </div>
        </div>

        {/* Filter Bar */}
        <div className="flex flex-col sm:flex-row items-center justify-between gap-4 bg-white dark:bg-[#10141d] border border-slate-200 dark:border-surface-800 rounded-2xl p-3 shadow-xs">
          <div className="flex items-center gap-3 w-full sm:w-80 bg-slate-50 dark:bg-[#121824] border border-slate-300 dark:border-surface-700 rounded-xl px-3.5 py-2 shadow-xs focus-within:border-[#20a53a] focus-within:ring-2 focus-within:ring-[#20a53a]/20 transition-all">
            <Search className="w-4 h-4 text-slate-400 dark:text-slate-500 flex-shrink-0" />
            <input
              type="text"
              placeholder="Search API keys..."
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
            Showing {filteredKeys.length} of {keys.length} active tokens
          </div>
        </div>

        {/* Keys Table */}
        <div className="bg-white dark:bg-[#10141d] border border-slate-200 dark:border-surface-800 rounded-2xl overflow-hidden shadow-xs dark:shadow-xl">
          {loading ? (
            <div className="py-16 text-center text-slate-500 dark:text-slate-400 text-xs flex flex-col items-center gap-2">
              <RefreshCw className="w-5 h-5 animate-spin text-[#20a53a]" />
              Loading API access tokens...
            </div>
          ) : keys.length === 0 ? (
            <div className="py-16 text-center text-slate-500 dark:text-slate-400 text-xs">
              <KeyRound className="w-8 h-8 mx-auto mb-2 text-slate-400 opacity-50" />
              <p className="font-bold text-slate-900 dark:text-white text-sm">No API keys created yet</p>
              <p className="text-slate-600 dark:text-slate-400 mt-1">Generate a token to integrate with CI/CD or scripts.</p>
            </div>
          ) : (
            <div className="overflow-x-auto">
              <table className="w-full text-left text-sm border-collapse">
                <thead>
                  <tr className="border-b border-slate-200 dark:border-surface-800 bg-slate-50 dark:bg-[#121824] text-slate-700 dark:text-slate-300 text-[11px] font-bold uppercase tracking-wider">
                    <th className="px-6 py-3.5">Name</th>
                    <th className="px-6 py-3.5">Key Prefix</th>
                    <th className="px-6 py-3.5">Role Scope</th>
                    <th className="px-6 py-3.5">Created</th>
                    <th className="px-6 py-3.5">Last Used</th>
                    <th className="px-6 py-3.5 text-right">Actions</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-slate-200/80 dark:divide-surface-800/80">
                  {filteredKeys.map((k) => (
                    <tr key={k.id} className="hover:bg-slate-50 dark:hover:bg-[#151d2d] transition-colors">
                      <td className="px-6 py-4 font-bold text-slate-950 dark:text-white">
                        {k.name}
                      </td>
                      <td className="px-6 py-4 font-mono font-semibold text-indigo-600 dark:text-indigo-400">
                        {k.key_prefix}••••••••
                      </td>
                      <td className="px-6 py-4 uppercase font-sans">
                        <span className="px-2.5 py-0.5 rounded-full bg-slate-100 dark:bg-surface-800 text-slate-800 dark:text-slate-300 border border-slate-200 dark:border-surface-700 text-[10px] font-bold">
                          {k.role}
                        </span>
                      </td>
                      <td className="px-6 py-4 text-xs font-medium text-slate-600 dark:text-slate-400 font-mono">
                        {formatDate(k.created_at)}
                      </td>
                      <td className="px-6 py-4 text-xs font-medium text-slate-600 dark:text-slate-400 font-mono">
                        {formatDate(k.last_used_at)}
                      </td>
                      <td className="px-6 py-4 text-right">
                        <button
                          onClick={() => handleRevoke(k.id)}
                          className="p-1.5 rounded-lg border border-slate-300 dark:border-surface-700 text-slate-600 dark:text-slate-400 hover:text-rose-600 dark:hover:text-rose-400 hover:bg-slate-100 dark:hover:bg-surface-800 transition-colors"
                          title="Revoke Token"
                        >
                          <Trash2 className="w-3.5 h-3.5" />
                        </button>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </div>

        {/* Create API Key Modal */}
        {showCreateModal && (
          <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm p-4">
            <div className="bg-white dark:bg-surface-900 border border-surface-200 dark:border-surface-800 rounded-2xl max-w-md w-full p-6 shadow-2xl space-y-4">
              <div className="flex items-center justify-between">
                <h3 className="text-lg font-bold text-slate-900 dark:text-white flex items-center gap-2">
                  <KeyRound className="w-5 h-5 text-indigo-500" />
                  Generate API Secret Key
                </h3>
                <button
                  onClick={() => setShowCreateModal(false)}
                  className="p-1 rounded-lg text-slate-400 hover:text-slate-600 dark:hover:text-white"
                >
                  <X className="w-4 h-4" />
                </button>
              </div>

              {createdSecret ? (
                <div className="space-y-4">
                  <div className="p-4 rounded-xl bg-emerald-500/10 border border-emerald-500/20 text-xs text-emerald-700 dark:text-emerald-300 space-y-2">
                    <div className="font-bold flex items-center gap-1.5 text-emerald-600 dark:text-emerald-400">
                      <AlertTriangle className="w-4 h-4 text-amber-500" />
                      Save this token now!
                    </div>
                    <p className="leading-relaxed">
                      For security, this secret token will never be displayed again. Store it securely in your secrets vault or environment variables.
                    </p>
                  </div>

                  <div className="p-3 rounded-xl bg-surface-100 dark:bg-surface-950 border border-surface-200 dark:border-surface-800 font-mono text-xs text-indigo-600 dark:text-indigo-400 break-all flex items-center justify-between gap-2">
                    <span className="select-all">{createdSecret}</span>
                    <button
                      onClick={handleCopy}
                      className="p-2 rounded-lg bg-white dark:bg-surface-800 hover:bg-surface-100 dark:hover:bg-surface-700 text-slate-700 dark:text-white shrink-0 transition-colors shadow-sm"
                      title="Copy Token"
                    >
                      {copied ? <Check className="w-4 h-4 text-emerald-500" /> : <Copy className="w-4 h-4" />}
                    </button>
                  </div>

                  <div className="flex justify-end pt-2">
                    <button
                      onClick={() => setShowCreateModal(false)}
                      className="px-4 py-2 rounded-xl bg-indigo-600 hover:bg-indigo-500 text-white text-xs font-semibold transition-colors"
                    >
                      Done
                    </button>
                  </div>
                </div>
              ) : (
                <form onSubmit={handleCreateKey} className="space-y-4 text-sm">
                  <div>
                    <label className="block text-xs font-semibold text-slate-700 dark:text-slate-300 mb-1.5">
                      Token Name / Purpose
                    </label>
                    <input
                      type="text"
                      required
                      placeholder="e.g. GitHub Actions Deployment"
                      value={keyName}
                      onChange={(e) => setKeyName(e.target.value)}
                      className="w-full px-3.5 py-2 rounded-xl bg-surface-50 dark:bg-surface-950 border border-surface-200 dark:border-surface-800 text-slate-900 dark:text-white text-xs focus:outline-none focus:ring-2 focus:ring-indigo-500/30"
                    />
                  </div>

                  <div>
                    <label className="block text-xs font-semibold text-slate-700 dark:text-slate-300 mb-1.5">
                      Assigned Role Scope
                    </label>
                    <select
                      value={keyRole}
                      onChange={(e) => setKeyRole(e.target.value as any)}
                      className="w-full px-3.5 py-2 rounded-xl bg-surface-50 dark:bg-surface-950 border border-surface-200 dark:border-surface-800 text-slate-900 dark:text-white text-xs focus:outline-none focus:ring-2 focus:ring-indigo-500/30 capitalize"
                    >
                      <option value="manager">Manager (Websites, databases, backups)</option>
                      <option value="admin">Admin (Full cluster management)</option>
                      <option value="viewer">Viewer (Read-only metrics & logs)</option>
                    </select>
                  </div>

                  <div className="flex items-center justify-end gap-3 pt-3">
                    <button
                      type="button"
                      onClick={() => setShowCreateModal(false)}
                      className="px-4 py-2 rounded-xl text-xs font-semibold text-slate-500 hover:text-slate-800 dark:hover:text-white"
                    >
                      Cancel
                    </button>
                    <button
                      type="submit"
                      disabled={actionLoading || !keyName.trim()}
                      className="px-4 py-2 rounded-xl bg-indigo-600 hover:bg-indigo-500 disabled:opacity-50 text-white text-xs font-semibold transition-all flex items-center gap-1.5"
                    >
                      {actionLoading && <RefreshCw className="w-3.5 h-3.5 animate-spin" />}
                      Generate Key
                    </button>
                  </div>
                </form>
              )}
            </div>
          </div>
        )}
      </div>
    </DashboardShell>
  );
}
