'use client';

import React, { useState } from 'react';
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
} from 'lucide-react';

interface KeyItem {
  id: string;
  name: string;
  prefix: string;
  role: string;
  created_at: string;
  last_used: string;
}

const initialKeys: KeyItem[] = [
  {
    id: '1',
    name: 'GitHub Actions Deployment Token',
    prefix: 'hv_live_9f8a',
    role: 'manager',
    created_at: '2 weeks ago',
    last_used: '3 hours ago',
  },
  {
    id: '2',
    name: 'Datadog / Prometheus Scraper',
    prefix: 'hv_live_3b1e',
    role: 'viewer',
    created_at: '1 month ago',
    last_used: 'Just now',
  },
];

export default function APIKeysPage() {
  const [keys, setKeys] = useState<KeyItem[]>(initialKeys);
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [generatedSecret, setGeneratedSecret] = useState<string | null>(null);
  const [copied, setCopied] = useState(false);

  // Form state
  const [keyName, setKeyName] = useState('');
  const [keyRole, setKeyRole] = useState<'admin' | 'manager' | 'viewer'>('manager');

  const handleCreateKey = (e: React.FormEvent) => {
    e.preventDefault();
    const rawBytes = Array.from(crypto.getRandomValues(new Uint8Array(24)))
      .map((b) => b.toString(16).padStart(2, '0'))
      .join('');
    const fullSecret = `hv_live_${rawBytes}`;
    const prefix = fullSecret.substring(0, 12);

    const newKey: KeyItem = {
      id: Math.random().toString(36).substring(7),
      name: keyName,
      prefix: prefix,
      role: keyRole,
      created_at: 'Just now',
      last_used: 'Never',
    };

    setKeys([newKey, ...keys]);
    setGeneratedSecret(fullSecret);
    setKeyName('');
  };

  const handleCopy = () => {
    if (generatedSecret) {
      navigator.clipboard.writeText(generatedSecret);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    }
  };

  const handleRevoke = (id: string) => {
    setKeys(keys.filter((k) => k.id !== id));
  };

  return (
    <DashboardShell>
      <div className="space-y-6">
        {/* Page Header */}
        <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
          <div>
            <h1 className="text-2xl font-bold text-white tracking-tight flex items-center gap-2">
              <KeyRound className="w-7 h-7 text-brand-400" />
              API Access Tokens
            </h1>
            <p className="text-sm text-slate-400 mt-1">
              Bearer tokens for programmatic automation, CLI scripts, Terraform, and CI/CD pipelines.
            </p>
          </div>
          <button
            onClick={() => {
              setGeneratedSecret(null);
              setShowCreateModal(true);
            }}
            className="inline-flex items-center gap-2 px-4 py-2 rounded-lg bg-brand-500 hover:bg-brand-600 text-white text-xs font-semibold transition-colors shadow-lg shadow-brand-500/20"
          >
            <Plus className="w-4 h-4" />
            Generate New Token
          </button>
        </div>

        {/* Keys Table */}
        <div className="bg-surface-900 border border-surface-800 rounded-xl overflow-hidden">
          <div className="px-6 py-4 border-b border-surface-800 flex items-center justify-between">
            <h2 className="text-sm font-semibold text-white uppercase tracking-wider">Active API Keys</h2>
            <span className="text-xs font-mono text-slate-400">{keys.length} active tokens</span>
          </div>

          <div className="overflow-x-auto">
            <table className="w-full text-left text-sm text-slate-300">
              <thead className="bg-surface-950/60 text-xs uppercase font-semibold text-slate-400 border-b border-surface-800">
                <tr>
                  <th className="px-6 py-3.5">Name</th>
                  <th className="px-6 py-3.5">Key Prefix</th>
                  <th className="px-6 py-3.5">Role Scope</th>
                  <th className="px-6 py-3.5">Created</th>
                  <th className="px-6 py-3.5">Last Used</th>
                  <th className="px-6 py-3.5 text-right">Actions</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-surface-800/60 font-mono text-xs">
                {keys.map((k) => (
                  <tr key={k.id} className="hover:bg-surface-800/30 transition-colors">
                    <td className="px-6 py-4 font-sans font-bold text-white">{k.name}</td>
                    <td className="px-6 py-4 text-brand-400">{k.prefix}••••••••</td>
                    <td className="px-6 py-4 font-sans uppercase">
                      <span className="px-2 py-0.5 rounded bg-surface-800 text-slate-300 border border-surface-700 text-[11px] font-semibold">
                        {k.role}
                      </span>
                    </td>
                    <td className="px-6 py-4 text-slate-400 font-sans">{k.created_at}</td>
                    <td className="px-6 py-4 text-slate-400 font-sans">{k.last_used}</td>
                    <td className="px-6 py-4 text-right">
                      <button
                        onClick={() => handleRevoke(k.id)}
                        className="p-1.5 rounded-lg hover:bg-rose-500/10 text-slate-400 hover:text-rose-400 transition-colors"
                        title="Revoke Token"
                      >
                        <Trash2 className="w-4 h-4" />
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>

        {/* Create API Key Modal */}
        {showCreateModal && (
          <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/70 backdrop-blur-sm p-4">
            <div className="bg-surface-900 border border-surface-800 rounded-xl max-w-md w-full p-6 shadow-2xl space-y-4">
              <h3 className="text-lg font-bold text-white flex items-center gap-2">
                <KeyRound className="w-5 h-5 text-brand-400" />
                Generate API Secret Key
              </h3>

              {generatedSecret ? (
                <div className="space-y-4">
                  <div className="p-4 rounded-lg bg-emerald-500/10 border border-emerald-500/20 text-xs text-emerald-300 space-y-2">
                    <div className="font-bold flex items-center gap-1.5 text-emerald-400">
                      <AlertTriangle className="w-4 h-4 text-amber-400" />
                      Save this token now!
                    </div>
                    <p>
                      For security, this secret token will never be displayed again. Store it securely in your secrets vault or environment variables.
                    </p>
                  </div>

                  <div className="p-3 rounded-lg bg-surface-950 border border-surface-800 font-mono text-xs text-brand-300 break-all flex items-center justify-between gap-2">
                    <span>{generatedSecret}</span>
                    <button
                      onClick={handleCopy}
                      className="p-2 rounded bg-surface-800 hover:bg-surface-700 text-white shrink-0 transition-colors"
                      title="Copy Token"
                    >
                      {copied ? <Check className="w-4 h-4 text-emerald-400" /> : <Copy className="w-4 h-4" />}
                    </button>
                  </div>

                  <div className="flex justify-end pt-2">
                    <button
                      onClick={() => setShowCreateModal(false)}
                      className="px-4 py-2 rounded-lg bg-brand-500 hover:bg-brand-600 text-white text-xs font-semibold transition-colors"
                    >
                      Done
                    </button>
                  </div>
                </div>
              ) : (
                <form onSubmit={handleCreateKey} className="space-y-3.5 text-sm">
                  <div>
                    <label className="block text-xs font-semibold text-slate-300 mb-1">Token Name / Purpose</label>
                    <input
                      type="text"
                      required
                      placeholder="e.g. Terraform Production Provider"
                      value={keyName}
                      onChange={(e) => setKeyName(e.target.value)}
                      className="w-full px-3.5 py-2 rounded-lg bg-surface-950 border border-surface-800 text-white text-sm focus:outline-none focus:border-brand-500"
                    />
                  </div>

                  <div>
                    <label className="block text-xs font-semibold text-slate-300 mb-1">Assigned Role</label>
                    <select
                      value={keyRole}
                      onChange={(e) => setKeyRole(e.target.value as any)}
                      className="w-full px-3.5 py-2 rounded-lg bg-surface-950 border border-surface-800 text-white text-sm focus:outline-none focus:border-brand-500 capitalize"
                    >
                      <option value="manager">Manager (Sites, databases, backups)</option>
                      <option value="admin">Admin (Full administrative privileges)</option>
                      <option value="viewer">Viewer (Read-only metrics & probes)</option>
                    </select>
                  </div>

                  <div className="flex items-center justify-end gap-3 pt-3">
                    <button
                      type="button"
                      onClick={() => setShowCreateModal(false)}
                      className="px-4 py-2 rounded-lg text-xs font-semibold text-slate-400 hover:text-white"
                    >
                      Cancel
                    </button>
                    <button
                      type="submit"
                      className="px-4 py-2 rounded-lg bg-brand-500 hover:bg-brand-600 text-white text-xs font-semibold transition-colors"
                    >
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
