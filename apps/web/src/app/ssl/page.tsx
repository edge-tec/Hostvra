'use client';

import React, { useState, useEffect } from 'react';
import Link from 'next/link';
import {
  ShieldCheck,
  ShieldAlert,
  Lock,
  RefreshCw,
  Search,
  ExternalLink,
  CheckCircle2,
  Clock,
  Globe,
  X,
} from 'lucide-react';
import { DashboardShell } from '@/components/DashboardShell';
import { apiFetch, Website } from '@/lib/api';

export default function SSLPage() {
  const [websites, setWebsites] = useState<Website[]>([]);
  const [loading, setLoading] = useState(true);
  const [search, setSearch] = useState('');
  const [renewing, setRenewing] = useState<string | null>(null);

  const fetchWebsites = async () => {
    try {
      setLoading(true);
      const res = await apiFetch<Website[]>('/api/v1/websites');
      if (res.success && res.data) {
        setWebsites(res.data);
      } else {
        setWebsites([]);
      }
    } catch {
      setWebsites([]);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchWebsites();
  }, []);

  const handleIssueOrRenew = async (id: string) => {
    try {
      setRenewing(id);
      await apiFetch(`/api/v1/websites/${id}/ssl`, { method: 'POST' });
      await fetchWebsites();
    } catch (err: any) {
      alert(err.message || 'Failed to issue SSL certificate');
    } finally {
      setRenewing(null);
    }
  };

  const filtered = websites.filter((w) =>
    w.primary_domain.toLowerCase().includes(search.toLowerCase())
  );

  return (
    <DashboardShell>
      <div className="space-y-6 animate-fadeIn max-w-7xl mx-auto pb-12">
        <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
          <div>
            <h1 className="text-2xl font-bold tracking-tight text-slate-950 dark:text-white">SSL / TLS Certificates</h1>
            <p className="text-sm text-slate-600 dark:text-slate-400 mt-1">
              Automated ACME Let&apos;s Encrypt certificate issuance, renewal monitoring, and HTTPS enforcement.
            </p>
          </div>
          <button
            onClick={fetchWebsites}
            className="p-2.5 rounded-xl bg-white dark:bg-surface-900 border border-slate-300 dark:border-surface-800 text-slate-700 dark:text-slate-300 hover:text-slate-950 dark:hover:text-white hover:bg-slate-100 dark:hover:bg-surface-800 transition-colors self-start shadow-xs"
          >
            <RefreshCw className={`w-4 h-4 ${loading ? 'animate-spin' : ''}`} />
          </button>
        </div>

        {/* Search / Filter Bar */}
        <div className="flex flex-col sm:flex-row items-center justify-between gap-4 bg-white dark:bg-[#10141d] border border-slate-200 dark:border-surface-800 rounded-2xl p-3 shadow-xs">
          <div className="flex items-center gap-3 w-full sm:w-80 bg-slate-50 dark:bg-[#121824] border border-slate-300 dark:border-surface-700 rounded-xl px-3.5 py-2 shadow-xs focus-within:border-[#20a53a] focus-within:ring-2 focus-within:ring-[#20a53a]/20 transition-all">
            <Search className="w-4 h-4 text-slate-400 dark:text-slate-500 flex-shrink-0" />
            <input
              type="text"
              placeholder="Filter certificates by domain..."
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
            Showing {filtered.length} of {websites.length} domains
          </div>
        </div>

        {/* Certificates Inventory */}
        {loading ? (
          <div className="py-20 text-center text-slate-500 dark:text-slate-400">
            <div className="w-8 h-8 border-2 border-[#20a53a] border-t-transparent rounded-full animate-spin mx-auto mb-3" />
            Loading certificate inventory...
          </div>
        ) : filtered.length === 0 ? (
          <div className="py-16 text-center border border-dashed border-slate-200 dark:border-surface-800 rounded-2xl bg-white dark:bg-surface-900/50 shadow-xs">
            <Lock className="w-12 h-12 mx-auto text-slate-400 dark:text-slate-600 mb-3" />
            <h3 className="text-base font-bold text-slate-950 dark:text-white">No websites available for SSL</h3>
            <p className="text-xs text-slate-600 dark:text-slate-400 mt-1 max-w-sm mx-auto">
              Deploy a virtual host under the Websites tab to issue automated certificates.
            </p>
            <Link
              href="/websites"
              className="mt-4 inline-flex items-center gap-2 px-4 py-2 rounded-xl bg-indigo-600 hover:bg-indigo-500 text-white text-xs font-semibold shadow-md transition-all"
            >
              <Globe className="w-3.5 h-3.5" />
              Go to Websites
            </Link>
          </div>
        ) : (
          <div className="bg-white dark:bg-[#10141d] border border-slate-200 dark:border-surface-800 rounded-2xl overflow-hidden shadow-xs dark:shadow-xl">
            <div className="overflow-x-auto">
              <table className="w-full text-left text-sm border-collapse">
                <thead>
                  <tr className="border-b border-slate-200 dark:border-surface-800 bg-slate-50 dark:bg-[#121824] text-slate-700 dark:text-slate-300 text-[11px] font-bold uppercase tracking-wider">
                    <th className="px-6 py-3.5">Protected Domain</th>
                    <th className="px-6 py-3.5">Certificate Authority</th>
                    <th className="px-6 py-3.5">Protection Status</th>
                    <th className="px-6 py-3.5">Auto-Renewal</th>
                    <th className="px-6 py-3.5 text-right">Actions</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-slate-200/80 dark:divide-surface-800/80">
                  {filtered.map((site) => (
                    <tr key={site.id} className="hover:bg-slate-50 dark:hover:bg-[#151d2d] transition-colors">
                      <td className="px-6 py-4 font-bold text-slate-950 dark:text-white flex items-center gap-2.5">
                        <Lock className="w-4 h-4 text-indigo-600 dark:text-indigo-400 flex-shrink-0" />
                        <span className="font-semibold text-sm">{site.primary_domain}</span>
                      </td>
                      <td className="px-6 py-4 text-xs font-medium text-slate-700 dark:text-slate-300">
                        {site.ssl_enabled ? "Let's Encrypt Authority" : 'None'}
                      </td>
                      <td className="px-6 py-4">
                        {site.ssl_enabled ? (
                          <span className="inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full text-[11px] font-semibold bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 border border-emerald-500/20">
                            <CheckCircle2 className="w-3 h-3" />
                            <span>Active / Valid</span>
                          </span>
                        ) : (
                          <span className="inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full text-[11px] font-semibold bg-amber-500/10 text-amber-600 dark:text-amber-400 border border-amber-500/20">
                            <ShieldAlert className="w-3 h-3" />
                            <span>Not Secured</span>
                          </span>
                        )}
                      </td>
                      <td className="px-6 py-4 text-xs font-medium text-slate-600 dark:text-slate-400">
                        {site.ssl_enabled ? (
                          <span className="text-emerald-600 dark:text-emerald-400 font-semibold">Enabled (60 days)</span>
                        ) : (
                          <span>Disabled</span>
                        )}
                      </td>
                      <td className="px-6 py-4 text-right">
                        <button
                          onClick={() => handleIssueOrRenew(site.id)}
                          disabled={renewing === site.id}
                          className="px-3.5 py-1.5 rounded-xl border border-slate-300 dark:border-surface-700 bg-white dark:bg-surface-800 hover:bg-slate-100 dark:hover:bg-surface-700 text-xs font-bold text-slate-950 dark:text-slate-100 transition-colors disabled:opacity-50 shadow-xs"
                        >
                          {renewing === site.id
                            ? 'Processing...'
                            : site.ssl_enabled
                            ? 'Renew Certificate'
                            : 'Issue Certificate'}
                        </button>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </div>
        )}
      </div>
    </DashboardShell>
  );
}
