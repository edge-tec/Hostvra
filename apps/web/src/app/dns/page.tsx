'use client';

import React, { useState } from 'react';
import { DashboardShell } from '@/components/DashboardShell';
import {
  Network,
  Plus,
  Trash2,
  Download,
  CheckCircle2,
  Globe,
  Cloud,
  FileCode2,
  Search,
  Zap,
} from 'lucide-react';

interface DNSRecord {
  id: string;
  type: 'A' | 'AAAA' | 'CNAME' | 'TXT' | 'MX';
  name: string;
  content: string;
  ttl: number;
  priority?: number;
  proxied: boolean;
}

const initialRecords: DNSRecord[] = [
  { id: '1', type: 'A', name: '@', content: '198.51.100.42', ttl: 300, proxied: true },
  { id: '2', type: 'A', name: 'www', content: '198.51.100.42', ttl: 300, proxied: true },
  { id: '3', type: 'CNAME', name: 'api', content: 'hostvra.com', ttl: 300, proxied: false },
  { id: '4', type: 'MX', name: 'mail', content: 'mail.hostvra.com.', ttl: 3600, priority: 10, proxied: false },
  { id: '5', type: 'TXT', name: '@', content: 'v=spf1 mx a include:_spf.hostvra.com ~all', ttl: 3600, proxied: false },
];

export default function DNSPage() {
  const [activeZone, setActiveZone] = useState('hostvra.com');
  const [records, setRecords] = useState<DNSRecord[]>(initialRecords);
  const [searchTerm, setSearchTerm] = useState('');
  const [showAddModal, setShowAddModal] = useState(false);
  const [showExportModal, setShowExportModal] = useState(false);

  // New record form state
  const [recType, setRecType] = useState<'A' | 'AAAA' | 'CNAME' | 'TXT' | 'MX'>('A');
  const [recName, setRecName] = useState('');
  const [recContent, setRecContent] = useState('');
  const [recTTL, setRecTTL] = useState(300);
  const [recPriority, setRecPriority] = useState(10);
  const [recProxied, setRecProxied] = useState(false);

  const handleAddRecord = (e: React.FormEvent) => {
    e.preventDefault();
    const newRec: DNSRecord = {
      id: Math.random().toString(36).substring(7),
      type: recType,
      name: recName || '@',
      content: recContent,
      ttl: recTTL,
      priority: recType === 'MX' ? recPriority : undefined,
      proxied: recProxied,
    };
    setRecords([...records, newRec]);
    setShowAddModal(false);
    setRecName('');
    setRecContent('');
  };

  const handleDeleteRecord = (id: string) => {
    setRecords(records.filter((r) => r.id !== id));
  };

  const filteredRecords = records.filter(
    (r) =>
      r.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
      r.content.toLowerCase().includes(searchTerm.toLowerCase()) ||
      r.type.toLowerCase().includes(searchTerm.toLowerCase())
  );

  const generateBindFile = () => {
    let out = `; Zone file for ${activeZone} (RFC 1035)\n$TTL 300\n@ IN SOA ns1.hostvra.com. admin.${activeZone}. ( 2026090601 3600 1800 604800 86400 )\n\n`;
    records.forEach((r) => {
      const fqdn = r.name === '@' ? `${activeZone}.` : `${r.name}.${activeZone}.`;
      if (r.type === 'MX') {
        out += `${fqdn.padEnd(25)} ${r.ttl.toString().padEnd(6)} IN MX   ${r.priority} ${r.content}\n`;
      } else {
        out += `${fqdn.padEnd(25)} ${r.ttl.toString().padEnd(6)} IN ${r.type.padEnd(5)} ${r.content}\n`;
      }
    });
    return out;
  };

  return (
    <DashboardShell>
      <div className="space-y-6">
        {/* Page Header */}
        <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
          <div>
            <h1 className="text-2xl font-bold text-white tracking-tight flex items-center gap-2">
              <Network className="w-7 h-7 text-brand-400" />
              DNS Zone Management
            </h1>
            <p className="text-sm text-slate-400 mt-1">
              Authoritative DNS records management with RFC validation, Cloudflare proxy synchronization, and BIND zone export.
            </p>
          </div>
          <div className="flex items-center gap-3">
            <button
              onClick={() => setShowExportModal(true)}
              className="inline-flex items-center gap-2 px-3.5 py-2 rounded-lg bg-surface-800 hover:bg-surface-700 text-white text-xs font-semibold transition-colors border border-surface-700"
            >
              <Download className="w-4 h-4" />
              Export Zone File
            </button>
            <button
              onClick={() => setShowAddModal(true)}
              className="inline-flex items-center gap-2 px-4 py-2 rounded-lg bg-brand-500 hover:bg-brand-600 text-white text-xs font-semibold transition-colors shadow-lg shadow-brand-500/20"
            >
              <Plus className="w-4 h-4" />
              Add Record
            </button>
          </div>
        </div>

        {/* Zone Selector & Search Bar */}
        <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4 bg-surface-900 border border-surface-800 rounded-xl p-4">
          <div className="flex items-center gap-3">
            <Globe className="w-5 h-5 text-brand-400" />
            <span className="text-xs font-semibold text-slate-400 uppercase">Active Zone:</span>
            <select
              value={activeZone}
              onChange={(e) => setActiveZone(e.target.value)}
              className="px-3 py-1.5 rounded-lg bg-surface-950 border border-surface-700 text-white font-mono text-sm font-bold focus:outline-none"
            >
              <option value="hostvra.com">hostvra.com (Primary)</option>
              <option value="cloud-edge.net">cloud-edge.net</option>
            </select>
            <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded text-[11px] font-semibold bg-emerald-500/10 text-emerald-400 border border-emerald-500/20">
              <CheckCircle2 className="w-3 h-3" /> Authoritative Active
            </span>
          </div>

          <div className="relative">
            <Search className="w-4 h-4 absolute left-3 top-1/2 -translate-y-1/2 text-slate-400" />
            <input
              type="text"
              placeholder="Search records..."
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
              className="pl-9 pr-4 py-1.5 rounded-lg bg-surface-950 border border-surface-800 text-white text-xs placeholder:text-slate-500 focus:outline-none focus:border-brand-500 w-full sm:w-64"
            />
          </div>
        </div>

        {/* DNS Records Table */}
        <div className="bg-surface-900 border border-surface-800 rounded-xl overflow-hidden">
          <div className="overflow-x-auto">
            <table className="w-full text-left text-sm text-slate-300">
              <thead className="bg-surface-950/60 text-xs uppercase font-semibold text-slate-400 border-b border-surface-800">
                <tr>
                  <th className="px-6 py-3.5">Type</th>
                  <th className="px-6 py-3.5">Name</th>
                  <th className="px-6 py-3.5">Content / Target</th>
                  <th className="px-6 py-3.5">TTL</th>
                  <th className="px-6 py-3.5">Proxy Status</th>
                  <th className="px-6 py-3.5 text-right">Actions</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-surface-800/60 font-mono text-xs">
                {filteredRecords.map((r) => (
                  <tr key={r.id} className="hover:bg-surface-800/30 transition-colors">
                    <td className="px-6 py-4">
                      <span className="px-2.5 py-1 rounded bg-brand-500/10 text-brand-400 border border-brand-500/20 font-bold">
                        {r.type}
                      </span>
                    </td>
                    <td className="px-6 py-4 font-bold text-white">{r.name}</td>
                    <td className="px-6 py-4 text-slate-200 truncate max-w-xs">
                      {r.type === 'MX' && <span className="text-amber-400 mr-2">[Priority {r.priority}]</span>}
                      {r.content}
                    </td>
                    <td className="px-6 py-4 text-slate-400">{r.ttl}s</td>
                    <td className="px-6 py-4">
                      {r.proxied ? (
                        <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded text-[11px] font-semibold bg-amber-500/10 text-amber-400 border border-amber-500/20">
                          <Cloud className="w-3 h-3 fill-amber-400" /> Proxied
                        </span>
                      ) : (
                        <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded text-[11px] font-semibold bg-surface-800 text-slate-400 border border-surface-700">
                          DNS Only
                        </span>
                      )}
                    </td>
                    <td className="px-6 py-4 text-right">
                      <button
                        onClick={() => handleDeleteRecord(r.id)}
                        className="p-1.5 rounded-lg hover:bg-rose-500/10 text-slate-400 hover:text-rose-400 transition-colors"
                        title="Delete Record"
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

        {/* Add Record Modal */}
        {showAddModal && (
          <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/70 backdrop-blur-sm p-4">
            <div className="bg-surface-900 border border-surface-800 rounded-xl max-w-md w-full p-6 shadow-2xl space-y-4">
              <h3 className="text-lg font-bold text-white flex items-center gap-2">
                <Plus className="w-5 h-5 text-brand-400" />
                Add New DNS Record
              </h3>

              <form onSubmit={handleAddRecord} className="space-y-3.5 text-sm">
                <div>
                  <label className="block text-xs font-semibold text-slate-300 mb-1">Record Type</label>
                  <div className="grid grid-cols-5 gap-1.5">
                    {(['A', 'AAAA', 'CNAME', 'TXT', 'MX'] as const).map((t) => (
                      <button
                        type="button"
                        key={t}
                        onClick={() => setRecType(t)}
                        className={`py-1.5 rounded text-xs font-bold border transition-all ${
                          recType === t
                            ? 'bg-brand-500/20 border-brand-500 text-brand-400'
                            : 'bg-surface-800 border-surface-700 text-slate-400 hover:text-white'
                        }`}
                      >
                        {t}
                      </button>
                    ))}
                  </div>
                </div>

                <div>
                  <label className="block text-xs font-semibold text-slate-300 mb-1">Name / Subdomain</label>
                  <input
                    type="text"
                    required
                    placeholder="@ or www, mail"
                    value={recName}
                    onChange={(e) => setRecName(e.target.value)}
                    className="w-full px-3.5 py-2 rounded-lg bg-surface-950 border border-surface-800 text-white font-mono text-sm focus:outline-none focus:border-brand-500"
                  />
                </div>

                <div>
                  <label className="block text-xs font-semibold text-slate-300 mb-1">
                    {recType === 'A' ? 'IPv4 Address' : recType === 'AAAA' ? 'IPv6 Address' : 'Content / Value'}
                  </label>
                  <input
                    type="text"
                    required
                    placeholder={recType === 'A' ? '198.51.100.42' : 'target destination'}
                    value={recContent}
                    onChange={(e) => setRecContent(e.target.value)}
                    className="w-full px-3.5 py-2 rounded-lg bg-surface-950 border border-surface-800 text-white font-mono text-sm focus:outline-none focus:border-brand-500"
                  />
                </div>

                {recType === 'MX' && (
                  <div>
                    <label className="block text-xs font-semibold text-slate-300 mb-1">Priority</label>
                    <input
                      type="number"
                      min={0}
                      max={65535}
                      value={recPriority}
                      onChange={(e) => setRecPriority(Number(e.target.value))}
                      className="w-full px-3.5 py-2 rounded-lg bg-surface-950 border border-surface-800 text-white font-mono text-sm focus:outline-none focus:border-brand-500"
                    />
                  </div>
                )}

                <div className="flex items-center justify-between pt-1">
                  <label className="text-xs font-semibold text-slate-300 flex items-center gap-2 cursor-pointer">
                    <input
                      type="checkbox"
                      checked={recProxied}
                      onChange={(e) => setRecProxied(e.target.checked)}
                      className="rounded bg-surface-950 border-surface-700 text-brand-500 focus:ring-0"
                    />
                    Proxy traffic (Cloudflare CDN / WAF)
                  </label>
                </div>

                <div className="flex items-center justify-end gap-3 pt-3">
                  <button
                    type="button"
                    onClick={() => setShowAddModal(false)}
                    className="px-4 py-2 rounded-lg text-xs font-semibold text-slate-400 hover:text-white"
                  >
                    Cancel
                  </button>
                  <button
                    type="submit"
                    className="px-4 py-2 rounded-lg bg-brand-500 hover:bg-brand-600 text-white text-xs font-semibold transition-colors"
                  >
                    Save Record
                  </button>
                </div>
              </form>
            </div>
          </div>
        )}

        {/* Export BIND Modal */}
        {showExportModal && (
          <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/70 backdrop-blur-sm p-4">
            <div className="bg-surface-900 border border-surface-800 rounded-xl max-w-xl w-full p-6 shadow-2xl space-y-4">
              <div className="flex items-center justify-between">
                <h3 className="text-lg font-bold text-white flex items-center gap-2">
                  <FileCode2 className="w-5 h-5 text-brand-400" />
                  RFC 1035 Zone File Export
                </h3>
                <span className="text-xs font-mono text-slate-400">{activeZone}</span>
              </div>

              <pre className="p-4 rounded-lg bg-surface-950 border border-surface-800 text-emerald-400 font-mono text-xs overflow-x-auto select-all max-h-72">
                {generateBindFile()}
              </pre>

              <div className="flex items-center justify-end gap-3 pt-2">
                <button
                  type="button"
                  onClick={() => setShowExportModal(false)}
                  className="px-4 py-2 rounded-lg bg-brand-500 hover:bg-brand-600 text-white text-xs font-semibold transition-colors"
                >
                  Close
                </button>
              </div>
            </div>
          </div>
        )}
      </div>
    </DashboardShell>
  );
}
