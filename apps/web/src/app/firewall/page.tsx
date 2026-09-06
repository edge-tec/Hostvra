'use client';

import React, { useState } from 'react';
import {
  ShieldAlert,
  ShieldCheck,
  Plus,
  Trash2,
  Lock,
  RefreshCw,
  AlertTriangle,
  X,
  Check,
} from 'lucide-react';
import { DashboardShell } from '@/components/DashboardShell';

interface FirewallRuleItem {
  id: string;
  port: string;
  protocol: 'tcp' | 'udp' | 'both';
  action: 'allow' | 'deny';
  comment: string;
  is_protected?: boolean;
}

export default function FirewallPage() {
  const [rules, setRules] = useState<FirewallRuleItem[]>([
    { id: '1', port: '22', protocol: 'tcp', action: 'allow', comment: 'SSH Administrative Remote Access', is_protected: true },
    { id: '2', port: '80', protocol: 'tcp', action: 'allow', comment: 'HTTP Web Traffic' },
    { id: '3', port: '443', protocol: 'tcp', action: 'allow', comment: 'HTTPS Secure Web Traffic' },
    { id: '4', port: '3306', protocol: 'tcp', action: 'deny', comment: 'Block External MySQL Port' },
  ]);
  const [modalOpen, setModalOpen] = useState(false);
  const [port, setPort] = useState('');
  const [protocol, setProtocol] = useState<'tcp' | 'udp' | 'both'>('tcp');
  const [action, setAction] = useState<'allow' | 'deny'>('allow');
  const [comment, setComment] = useState('');

  const handleAddRule = (e: React.FormEvent) => {
    e.preventDefault();
    if (!port) return;

    const newRule: FirewallRuleItem = {
      id: 'rule-' + Date.now(),
      port,
      protocol,
      action,
      comment: comment || 'Custom port rule',
      is_protected: port === '22',
    };

    setRules([...rules, newRule]);
    setModalOpen(false);
    setPort('');
    setComment('');
  };

  const handleDeleteRule = (rule: FirewallRuleItem) => {
    if (rule.port === '22' || rule.is_protected) {
      alert('SSH Lockout Protection: Port 22 cannot be removed or denied to prevent administrative lockout.');
      return;
    }

    if (confirm(`Remove firewall rule for port ${rule.port}?`)) {
      setRules(rules.filter((r) => r.id !== rule.id));
    }
  };

  const handlePresetSelect = (presetPort: string, presetComment: string) => {
    setPort(presetPort);
    setComment(presetComment);
  };

  return (
    <DashboardShell>
      <div className="space-y-8">
        <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
          <div>
            <h1 className="text-2xl font-bold tracking-tight text-white">Firewall & Security Rules</h1>
            <p className="text-sm text-slate-400 mt-1">
              UFW / nftables network traffic filtering with automated SSH lockout protection.
            </p>
          </div>
          <button
            onClick={() => setModalOpen(true)}
            className="inline-flex items-center gap-2 px-4 py-2.5 rounded-xl bg-indigo-600 hover:bg-indigo-500 text-white text-sm font-semibold shadow-lg shadow-indigo-600/25 transition-all"
          >
            <Plus className="w-4 h-4" />
            Add Firewall Rule
          </button>
        </div>

        {/* Firewall Status Banner */}
        <div className="p-5 rounded-2xl bg-surface-900 border border-surface-800 shadow-xl flex flex-col sm:flex-row items-start sm:items-center justify-between gap-4">
          <div className="flex items-center gap-3">
            <div className="w-10 h-10 rounded-xl bg-emerald-500/10 border border-emerald-500/20 flex items-center justify-center text-emerald-400">
              <ShieldCheck className="w-5 h-5" />
            </div>
            <div>
              <div className="flex items-center gap-2">
                <h2 className="font-bold text-white text-sm">System Firewall (UFW) Active</h2>
                <span className="w-2 h-2 rounded-full bg-emerald-500 animate-pulse" />
              </div>
              <p className="text-xs text-slate-400 mt-0.5">
                Default: Incoming traffic DENIED • Outgoing traffic ALLOWED
              </p>
            </div>
          </div>
          <div className="text-xs font-mono px-3 py-1 rounded-lg bg-surface-950 border border-surface-700 text-indigo-400 flex items-center gap-1.5">
            <Lock className="w-3.5 h-3.5 text-emerald-400" />
            <span>SSH Lockout Guard Enabled</span>
          </div>
        </div>

        {/* Rules Table */}
        <div className="bg-surface-900 border border-surface-800 rounded-2xl overflow-hidden shadow-xl">
          <table className="w-full text-left text-sm">
            <thead>
              <tr className="border-b border-surface-800 bg-surface-950/40 text-slate-400 text-xs uppercase tracking-wider">
                <th className="px-6 py-3.5 font-semibold">Port</th>
                <th className="px-6 py-3.5 font-semibold">Protocol</th>
                <th className="px-6 py-3.5 font-semibold">Action</th>
                <th className="px-6 py-3.5 font-semibold">Rule Purpose / Comment</th>
                <th className="px-6 py-3.5 font-semibold text-right">Actions</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-surface-800/60">
              {rules.map((r) => (
                <tr key={r.id} className="hover:bg-surface-800/30 transition-colors">
                  <td className="px-6 py-4 font-mono font-bold text-white flex items-center gap-2">
                    {r.is_protected && (
                      <span title="Protected System Port">
                        <Lock className="w-3.5 h-3.5 text-emerald-400" />
                      </span>
                    )}
                    <span>{r.port}</span>
                  </td>
                  <td className="px-6 py-4 uppercase font-mono text-xs text-slate-300">{r.protocol}</td>
                  <td className="px-6 py-4">
                    <span
                      className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-[11px] font-semibold uppercase ${
                        r.action === 'allow'
                          ? 'bg-emerald-500/10 text-emerald-400 border border-emerald-500/20'
                          : 'bg-rose-500/10 text-rose-400 border border-rose-500/20'
                      }`}
                    >
                      {r.action}
                    </span>
                  </td>
                  <td className="px-6 py-4 text-xs text-slate-300">{r.comment}</td>
                  <td className="px-6 py-4 text-right">
                    {r.is_protected ? (
                      <span className="text-[11px] text-slate-500 font-mono">Protected</span>
                    ) : (
                      <button
                        onClick={() => handleDeleteRule(r)}
                        className="p-1.5 rounded-lg border border-surface-700 text-slate-400 hover:text-rose-400 hover:bg-surface-800 transition-colors"
                      >
                        <Trash2 className="w-3.5 h-3.5" />
                      </button>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>

        {/* Add Rule Modal */}
        {modalOpen && (
          <div className="fixed inset-0 z-50 bg-black/70 backdrop-blur-sm flex items-center justify-center p-4">
            <div className="w-full max-w-md bg-surface-900 border border-surface-700 rounded-2xl shadow-2xl p-6 relative">
              <button
                onClick={() => setModalOpen(false)}
                className="absolute top-5 right-5 p-1 rounded-lg text-slate-400 hover:text-white hover:bg-surface-800 transition-colors"
              >
                <X className="w-5 h-5" />
              </button>

              <div className="flex items-center gap-3 mb-5">
                <div className="w-10 h-10 rounded-xl bg-indigo-500/10 border border-indigo-500/20 flex items-center justify-center text-indigo-400">
                  <ShieldAlert className="w-5 h-5" />
                </div>
                <div>
                  <h2 className="text-lg font-bold text-white">Add Port Rule</h2>
                  <p className="text-xs text-slate-400">Configure firewall port ingress rules</p>
                </div>
              </div>

              {/* Common Presets */}
              <div className="mb-4">
                <label className="block text-[11px] font-semibold uppercase tracking-wider text-slate-400 mb-1.5">
                  Common Presets
                </label>
                <div className="flex flex-wrap gap-1.5">
                  <button
                    type="button"
                    onClick={() => handlePresetSelect('80', 'HTTP Web Server')}
                    className="px-2.5 py-1 rounded-lg bg-surface-950 hover:bg-surface-800 border border-surface-700 text-xs text-slate-300"
                  >
                    HTTP (80)
                  </button>
                  <button
                    type="button"
                    onClick={() => handlePresetSelect('443', 'HTTPS Secure Web')}
                    className="px-2.5 py-1 rounded-lg bg-surface-950 hover:bg-surface-800 border border-surface-700 text-xs text-slate-300"
                  >
                    HTTPS (443)
                  </button>
                  <button
                    type="button"
                    onClick={() => handlePresetSelect('3306', 'MySQL Database')}
                    className="px-2.5 py-1 rounded-lg bg-surface-950 hover:bg-surface-800 border border-surface-700 text-xs text-slate-300"
                  >
                    MySQL (3306)
                  </button>
                  <button
                    type="button"
                    onClick={() => handlePresetSelect('5432', 'PostgreSQL Database')}
                    className="px-2.5 py-1 rounded-lg bg-surface-950 hover:bg-surface-800 border border-surface-700 text-xs text-slate-300"
                  >
                    Postgres (5432)
                  </button>
                </div>
              </div>

              <form onSubmit={handleAddRule} className="space-y-4">
                <div className="grid grid-cols-2 gap-3">
                  <div>
                    <label className="block text-xs font-semibold uppercase tracking-wider text-slate-300 mb-1.5">
                      Port or Range
                    </label>
                    <input
                      type="text"
                      required
                      placeholder="e.g. 8080"
                      value={port}
                      onChange={(e) => setPort(e.target.value)}
                      className="w-full px-3.5 py-2.5 rounded-xl bg-surface-950 border border-surface-700 text-slate-100 font-mono text-sm focus:outline-none focus:border-indigo-500"
                    />
                  </div>
                  <div>
                    <label className="block text-xs font-semibold uppercase tracking-wider text-slate-300 mb-1.5">
                      Protocol
                    </label>
                    <select
                      value={protocol}
                      onChange={(e) => setProtocol(e.target.value as any)}
                      className="w-full px-3.5 py-2.5 rounded-xl bg-surface-950 border border-surface-700 text-slate-100 text-sm focus:outline-none focus:border-indigo-500 uppercase"
                    >
                      <option value="tcp">TCP</option>
                      <option value="udp">UDP</option>
                      <option value="both">BOTH</option>
                    </select>
                  </div>
                </div>

                <div>
                  <label className="block text-xs font-semibold uppercase tracking-wider text-slate-300 mb-1.5">
                    Traffic Action
                  </label>
                  <select
                    value={action}
                    onChange={(e) => setAction(e.target.value as any)}
                    className="w-full px-3.5 py-2.5 rounded-xl bg-surface-950 border border-surface-700 text-slate-100 text-sm focus:outline-none focus:border-indigo-500 uppercase"
                  >
                    <option value="allow">ALLOW</option>
                    <option value="deny">DENY</option>
                  </select>
                </div>

                <div>
                  <label className="block text-xs font-semibold uppercase tracking-wider text-slate-300 mb-1.5">
                    Comment / Description
                  </label>
                  <input
                    type="text"
                    placeholder="e.g. Node.js backend port"
                    value={comment}
                    onChange={(e) => setComment(e.target.value)}
                    className="w-full px-3.5 py-2.5 rounded-xl bg-surface-950 border border-surface-700 text-slate-100 text-sm focus:outline-none focus:border-indigo-500"
                  />
                </div>

                <div className="flex justify-end gap-2 pt-2">
                  <button
                    type="button"
                    onClick={() => setModalOpen(false)}
                    className="px-4 py-2 rounded-xl text-xs text-slate-400 hover:text-white transition-colors"
                  >
                    Cancel
                  </button>
                  <button
                    type="submit"
                    className="px-5 py-2 rounded-xl bg-indigo-600 hover:bg-indigo-500 text-white text-xs font-semibold shadow-md transition-all"
                  >
                    Apply Rule
                  </button>
                </div>
              </form>
            </div>
          </div>
        )}
      </div>
    </DashboardShell>
  );
}
