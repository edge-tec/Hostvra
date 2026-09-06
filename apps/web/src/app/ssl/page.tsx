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
} from 'lucide-react';
import { DashboardShell } from '@/components/DashboardShell';
import { apiFetch, Website } from '@/lib/api';

export default function SSLPage() {
  const [websites, setWebsites] = useState<Website[]>([]);
  const [loading, setLoading] = useState(true);
  const [search, setSearch] = useState('');
  const [renewing, setRenewing] = useState<string | null>(null);

  const fetchWebsites = async () => {
    setLoading(true);
    const res = await apiFetch<Website[]>('/api/v1/websites');
    if (res.success && res.data) {
      setWebsites(res.data);
    }
    setLoading(false);
  };

  useEffect(() => {
    fetchWebsites();
  }, []);

  const handleIssueOrRenew = async (siteId: string) => {
    setRenewing(siteId);
    const res = await apiFetch(`/api/v1/websites/${siteId}/ssl`, {
      method: 'POST',
    });
    setRenewing(null);
    if (res.success) {
      fetchWebsites();
    }
  };

  const filtered = websites.filter((w) =>
    w.primary_domain.toLowerCase().includes(search.toLowerCase())
  );

  return (
    <DashboardShell>
      <div className="space-y-8">
        <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
          <div>
            <h1 className="text-2xl font-bold tracking-tight text-white">SSL / TLS Certificates</h1>
            <p className="text-sm text-slate-400 mt-1">
              Automated ACME Let&apos;s Encrypt certificate issuance, renewal monitoring, and HTTPS enforcement.
            </p>
          </div>
          <button
            onClick={fetchWebsites}
            className="p-2.5 rounded-xl bg-surface-900 border border-surface-800 text-slate-300 hover:text-white hover:bg-surface-800 transition-colors self-start"
          >
            <RefreshCw className={`w-4 h-4 ${loading ? 'animate-spin' : ''}`} />
          </button>
        </div>

        {/* Search */}
        <div className="flex items-center gap-3 bg-surface-900 border border-surface-800 rounded-xl px-4 py-2.5">
          <Search className="w-4 h-4 text-slate-500" />
          <input
            type="text"
            placeholder="Filter certificates by domain..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="w-full bg-transparent text-sm text-slate-200 placeholder-slate-500 focus:outline-none"
          />
        </div>

        {/* Certificates Inventory */}
        {loading ? (
          <div className="py-20 text-center text-slate-400">
            <div className="w-8 h-8 border-2 border-indigo-500 border-t-transparent rounded-full animate-spin mx-auto mb-3" />
            Loading certificate inventory...
          </div>
        ) : filtered.length === 0 ? (
          <div className="py-16 text-center border border-dashed border-surface-800 rounded-2xl bg-surface-900/50">
            <Lock className="w-12 h-12 mx-auto text-slate-600 mb-3" />
            <h3 className="text-base font-semibold text-white">No websites available for SSL</h3>
            <p className="text-xs text-slate-400 mt-1 max-w-sm mx-auto">
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
          <div className="bg-surface-900 border border-surface-800 rounded-2xl overflow-hidden shadow-xl">
            <table className="w-full text-left text-sm">
              <thead>
                <tr className="border-b border-surface-800 bg-surface-950/40 text-slate-400 text-xs uppercase tracking-wider">
                  <th className="px-6 py-3.5 font-semibold">Protected Domain</th>
                  <th className="px-6 py-3.5 font-semibold">Certificate Authority</th>
                  <th className="px-6 py-3.5 font-semibold">Protection Status</th>
                  <th className="px-6 py-3.5 font-semibold">Auto-Renewal</th>
                  <th className="px-6 py-3.5 font-semibold text-right">Actions</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-surface-800/60">
                {filtered.map((site) => (
                  <tr key={site.id} className="hover:bg-surface-800/30 transition-colors">
                    <td className="px-6 py-4 font-bold text-white flex items-center gap-2">
                      <Lock className="w-4 h-4 text-indigo-400" />
                      <span>{site.primary_domain}</span>
                    </td>
                    <td className="px-6 py-4 text-xs text-slate-300">
                      {site.ssl_enabled ? "Let's Encrypt Authority" : 'None'}
                    </td>
                    <td className="px-6 py-4">
                      {site.ssl_enabled ? (
                        <span className="inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full text-[11px] font-semibold bg-emerald-500/10 text-emerald-400 border border-emerald-500/20">
                          <CheckCircle2 className="w-3 h-3" />
                          <span>Active / Valid</span>
                        </span>
                      ) : (
                        <span className="inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full text-[11px] font-semibold bg-amber-500/10 text-amber-400 border border-amber-500/20">
                          <ShieldAlert className="w-3 h-3" />
                          <span>Not Secured</span>
                        </span>
                      )}
                    </td>
                    <td className="px-6 py-4 text-xs text-slate-400">
                      {site.ssl_enabled ? (
                        <span className="text-emerald-400 font-medium">Enabled (60 days)</span>
                      ) : (
                        <span>Disabled</span>
                      )}
                    </td>
                    <td className="px-6 py-4 text-right">
                      <button
                        onClick={() => handleIssueOrRenew(site.id)}
                        disabled={renewing === site.id}
                        className="px-3 py-1.5 rounded-lg bg-surface-800 hover:bg-surface-700 text-xs font-semibold text-slate-200 transition-colors disabled:opacity-50"
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
        )}
      </div>
    </DashboardShell>
  );
}
