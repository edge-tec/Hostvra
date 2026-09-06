'use client';

import React, { useState, useEffect } from 'react';
import {
  Globe,
  Plus,
  ShieldCheck,
  ShieldAlert,
  Server as ServerIcon,
  RefreshCw,
  Search,
  ExternalLink,
  Trash2,
  Power,
  X,
  Check,
  Code2,
  Layers,
} from 'lucide-react';
import { DashboardShell } from '@/components/DashboardShell';
import { apiFetch, Website, Server } from '@/lib/api';

export default function WebsitesPage() {
  const [websites, setWebsites] = useState<Website[]>([]);
  const [servers, setServers] = useState<Server[]>([]);
  const [loading, setLoading] = useState(true);
  const [search, setSearch] = useState('');
  const [modalOpen, setModalOpen] = useState(false);

  // Form State
  const [domain, setDomain] = useState('');
  const [selectedServer, setSelectedServer] = useState('');
  const [appType, setAppType] = useState<'php' | 'static' | 'proxy'>('php');
  const [phpVersion, setPhpVersion] = useState('8.3');
  const [proxyPort, setProxyPort] = useState(3000);
  const [creating, setCreating] = useState(false);
  const [issuingSSL, setIssuingSSL] = useState<string | null>(null);

  const fetchData = async () => {
    setLoading(true);
    const [sitesRes, serversRes] = await Promise.all([
      apiFetch<Website[]>('/api/v1/websites'),
      apiFetch<Server[]>('/api/v1/servers'),
    ]);

    if (sitesRes.success && sitesRes.data) setWebsites(sitesRes.data);
    if (serversRes.success && serversRes.data) {
      setServers(serversRes.data);
      if (serversRes.data.length > 0 && !selectedServer) {
        setSelectedServer(serversRes.data[0].id);
      }
    }
    setLoading(false);
  };

  useEffect(() => {
    fetchData();
  }, []);

  const handleCreateWebsite = async (e: React.FormEvent) => {
    e.preventDefault();
    setCreating(true);

    const res = await apiFetch<Website>('/api/v1/websites', {
      method: 'POST',
      body: JSON.stringify({
        server_id: selectedServer,
        primary_domain: domain,
        app_type: appType,
        php_version: appType === 'php' ? phpVersion : undefined,
        proxy_port: appType === 'proxy' ? Number(proxyPort) : undefined,
      }),
    });

    setCreating(false);
    if (res.success && res.data) {
      setModalOpen(false);
      setDomain('');
      fetchData();
    }
  };

  const handleToggleStatus = async (site: Website) => {
    const nextStatus = site.status === 'active' ? 'suspended' : 'active';
    const res = await apiFetch<Website>(`/api/v1/websites/${site.id}/status`, {
      method: 'POST',
      body: JSON.stringify({ status: nextStatus }),
    });

    if (res.success) {
      fetchData();
    }
  };

  const handleIssueSSL = async (siteId: string) => {
    setIssuingSSL(siteId);
    const res = await apiFetch(`/api/v1/websites/${siteId}/ssl`, {
      method: 'POST',
    });
    setIssuingSSL(null);

    if (res.success) {
      fetchData();
    }
  };

  const handleDeleteWebsite = async (siteId: string, domainName: string) => {
    if (!confirm(`Are you sure you want to delete ${domainName}? This will remove the virtual host configuration.`)) {
      return;
    }

    const res = await apiFetch(`/api/v1/websites/${siteId}`, {
      method: 'DELETE',
    });

    if (res.success) {
      fetchData();
    }
  };

  const filteredWebsites = websites.filter(
    (w) =>
      w.primary_domain.toLowerCase().includes(search.toLowerCase()) ||
      w.document_root.toLowerCase().includes(search.toLowerCase())
  );

  return (
    <DashboardShell>
      <div className="space-y-8">
        <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
          <div>
            <h1 className="text-2xl font-bold tracking-tight text-white">Websites & Virtual Hosts</h1>
            <p className="text-sm text-slate-400 mt-1">
              Manage production Nginx web servers, PHP-FPM pools, and reverse proxies.
            </p>
          </div>
          <div className="flex items-center gap-3">
            <button
              onClick={fetchData}
              className="p-2.5 rounded-xl bg-surface-900 border border-surface-800 text-slate-300 hover:text-white hover:bg-surface-800 transition-colors"
            >
              <RefreshCw className={`w-4 h-4 ${loading ? 'animate-spin' : ''}`} />
            </button>
            <button
              onClick={() => setModalOpen(true)}
              className="inline-flex items-center gap-2 px-4 py-2.5 rounded-xl bg-indigo-600 hover:bg-indigo-500 text-white text-sm font-semibold shadow-lg shadow-indigo-600/25 transition-all"
            >
              <Plus className="w-4 h-4" />
              Create Website
            </button>
          </div>
        </div>

        {/* Search */}
        <div className="flex items-center gap-3 bg-surface-900 border border-surface-800 rounded-xl px-4 py-2.5">
          <Search className="w-4 h-4 text-slate-500" />
          <input
            type="text"
            placeholder="Search domains (e.g. example.com)..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="w-full bg-transparent text-sm text-slate-200 placeholder-slate-500 focus:outline-none"
          />
        </div>

        {/* Websites List */}
        {loading ? (
          <div className="py-20 text-center text-slate-400">
            <div className="w-8 h-8 border-2 border-indigo-500 border-t-transparent rounded-full animate-spin mx-auto mb-3" />
            Loading websites...
          </div>
        ) : filteredWebsites.length === 0 ? (
          <div className="py-16 text-center border border-dashed border-surface-800 rounded-2xl bg-surface-900/50">
            <Globe className="w-12 h-12 mx-auto text-slate-600 mb-3" />
            <h3 className="text-base font-semibold text-white">No websites deployed yet</h3>
            <p className="text-xs text-slate-400 mt-1 max-w-sm mx-auto">
              Create your first virtual host to serve PHP, static HTML, or reverse proxy applications.
            </p>
            <button
              onClick={() => setModalOpen(true)}
              className="mt-4 inline-flex items-center gap-2 px-4 py-2 rounded-xl bg-indigo-600 hover:bg-indigo-500 text-white text-xs font-semibold shadow-md transition-all"
            >
              <Plus className="w-3.5 h-3.5" />
              Create Website
            </button>
          </div>
        ) : (
          <div className="bg-surface-900 border border-surface-800 rounded-2xl overflow-hidden shadow-xl">
            <table className="w-full text-left text-sm">
              <thead>
                <tr className="border-b border-surface-800 bg-surface-950/40 text-slate-400 text-xs uppercase tracking-wider">
                  <th className="px-6 py-3.5 font-semibold">Primary Domain</th>
                  <th className="px-6 py-3.5 font-semibold">Type & Runtime</th>
                  <th className="px-6 py-3.5 font-semibold">Document Root</th>
                  <th className="px-6 py-3.5 font-semibold">Status</th>
                  <th className="px-6 py-3.5 font-semibold">SSL / HTTPS</th>
                  <th className="px-6 py-3.5 font-semibold text-right">Actions</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-surface-800/60">
                {filteredWebsites.map((site) => (
                  <tr key={site.id} className="hover:bg-surface-800/30 transition-colors">
                    <td className="px-6 py-4">
                      <div className="flex items-center gap-2 font-bold text-white">
                        <span>{site.primary_domain}</span>
                        <a
                          href={`http://${site.primary_domain}`}
                          target="_blank"
                          rel="noreferrer"
                          className="text-slate-500 hover:text-indigo-400 transition-colors"
                        >
                          <ExternalLink className="w-3.5 h-3.5" />
                        </a>
                      </div>
                    </td>
                    <td className="px-6 py-4">
                      <div className="flex items-center gap-1.5">
                        <span className="inline-flex items-center gap-1.5 px-2 py-0.5 rounded text-xs font-medium bg-surface-800 text-slate-200 border border-surface-700">
                          <Code2 className="w-3.5 h-3.5 text-indigo-400" />
                          {site.app_type === 'php'
                            ? `PHP ${site.php_version || '8.3'}`
                            : site.app_type === 'proxy'
                            ? `Proxy :${site.proxy_port}`
                            : 'Static'}
                        </span>
                        <span className="inline-flex items-center gap-1 px-1.5 py-0.5 rounded text-[10px] font-mono font-medium bg-indigo-500/10 text-indigo-400 border border-indigo-500/20 uppercase">
                          {site.web_server_type || 'nginx'}
                        </span>
                      </div>
                    </td>
                    <td className="px-6 py-4 font-mono text-xs text-slate-400 max-w-xs truncate">
                      {site.document_root}
                    </td>
                    <td className="px-6 py-4">
                      <span
                        className={`inline-flex items-center px-2 py-0.5 rounded-full text-[11px] font-semibold capitalize ${
                          site.status === 'active'
                            ? 'bg-emerald-500/10 text-emerald-400 border border-emerald-500/20'
                            : 'bg-amber-500/10 text-amber-400 border border-amber-500/20'
                        }`}
                      >
                        {site.status}
                      </span>
                    </td>
                    <td className="px-6 py-4">
                      {site.ssl_enabled ? (
                        <span className="inline-flex items-center gap-1 text-xs text-emerald-400 font-medium">
                          <ShieldCheck className="w-4 h-4" />
                          <span>Protected</span>
                        </span>
                      ) : (
                        <button
                          onClick={() => handleIssueSSL(site.id)}
                          disabled={issuingSSL === site.id}
                          className="inline-flex items-center gap-1.5 text-xs text-indigo-400 hover:text-indigo-300 font-medium"
                        >
                          <ShieldAlert className="w-3.5 h-3.5 text-amber-400" />
                          <span>{issuingSSL === site.id ? 'Issuing...' : 'Issue Free SSL'}</span>
                        </button>
                      )}
                    </td>
                    <td className="px-6 py-4 text-right">
                      <div className="flex items-center justify-end gap-2">
                        <a
                          href="/webservers"
                          title="Manage Web Server Engine & VHost"
                          className="p-1.5 rounded-lg border border-surface-700 text-slate-400 hover:text-indigo-400 hover:bg-surface-800 transition-colors inline-flex items-center"
                        >
                          <Layers className="w-3.5 h-3.5 text-indigo-400" />
                        </a>
                        {site.app_type === 'php' && (
                          <a
                            href="/php"
                            title="Configure PHP Settings & Pool"
                            className="p-1.5 rounded-lg border border-surface-700 text-slate-400 hover:text-sky-400 hover:bg-surface-800 transition-colors inline-flex items-center"
                          >
                            <Code2 className="w-3.5 h-3.5 text-sky-400" />
                          </a>
                        )}
                        <button
                          onClick={() => handleToggleStatus(site)}
                          title={site.status === 'active' ? 'Suspend Website' : 'Activate Website'}
                          className={`p-1.5 rounded-lg border transition-colors ${
                            site.status === 'active'
                              ? 'border-surface-700 text-slate-400 hover:text-amber-400 hover:bg-surface-800'
                              : 'border-emerald-500/30 text-emerald-400 hover:bg-emerald-500/10'
                          }`}
                        >
                          <Power className="w-3.5 h-3.5" />
                        </button>
                        <button
                          onClick={() => handleDeleteWebsite(site.id, site.primary_domain)}
                          title="Delete Website"
                          className="p-1.5 rounded-lg border border-surface-700 text-slate-400 hover:text-rose-400 hover:bg-surface-800 transition-colors"
                        >
                          <Trash2 className="w-3.5 h-3.5" />
                        </button>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}

        {/* Create Website Modal */}
        {modalOpen && (
          <div className="fixed inset-0 z-50 bg-black/70 backdrop-blur-sm flex items-center justify-center p-4">
            <div className="w-full max-w-lg bg-surface-900 border border-surface-700 rounded-2xl shadow-2xl p-6 relative">
              <button
                onClick={() => setModalOpen(false)}
                className="absolute top-5 right-5 p-1 rounded-lg text-slate-400 hover:text-white hover:bg-surface-800 transition-colors"
              >
                <X className="w-5 h-5" />
              </button>

              <div className="flex items-center gap-3 mb-5">
                <div className="w-10 h-10 rounded-xl bg-indigo-500/10 border border-indigo-500/20 flex items-center justify-center text-indigo-400">
                  <Globe className="w-5 h-5" />
                </div>
                <div>
                  <h2 className="text-lg font-bold text-white">Create Virtual Host</h2>
                  <p className="text-xs text-slate-400">Configure Nginx virtual host with safe-rollback validation</p>
                </div>
              </div>

              <form onSubmit={handleCreateWebsite} className="space-y-4">
                <div>
                  <label className="block text-xs font-semibold uppercase tracking-wider text-slate-300 mb-1.5">
                    Target Server
                  </label>
                  <select
                    value={selectedServer}
                    onChange={(e) => setSelectedServer(e.target.value)}
                    required
                    className="w-full px-3.5 py-2.5 rounded-xl bg-surface-950 border border-surface-700 text-slate-100 text-sm focus:outline-none focus:border-indigo-500"
                  >
                    {servers.map((s) => (
                      <option key={s.id} value={s.id}>
                        {s.name} ({s.ip_address}) — {s.os_name}
                      </option>
                    ))}
                  </select>
                </div>

                <div>
                  <label className="block text-xs font-semibold uppercase tracking-wider text-slate-300 mb-1.5">
                    Primary Domain Name
                  </label>
                  <input
                    type="text"
                    required
                    value={domain}
                    onChange={(e) => setDomain(e.target.value)}
                    placeholder="app.yourdomain.com"
                    className="w-full px-3.5 py-2.5 rounded-xl bg-surface-950 border border-surface-700 text-slate-100 placeholder-slate-500 text-sm focus:outline-none focus:border-indigo-500"
                  />
                </div>

                <div className="grid grid-cols-2 gap-4">
                  <div>
                    <label className="block text-xs font-semibold uppercase tracking-wider text-slate-300 mb-1.5">
                      Application Runtime
                    </label>
                    <select
                      value={appType}
                      onChange={(e) => setAppType(e.target.value as any)}
                      className="w-full px-3.5 py-2.5 rounded-xl bg-surface-950 border border-surface-700 text-slate-100 text-sm focus:outline-none focus:border-indigo-500"
                    >
                      <option value="php">PHP Application</option>
                      <option value="proxy">Reverse Proxy (Node/Python)</option>
                      <option value="static">Static HTML / Jamstack</option>
                    </select>
                  </div>

                  {appType === 'php' && (
                    <div>
                      <label className="block text-xs font-semibold uppercase tracking-wider text-slate-300 mb-1.5">
                        PHP-FPM Version
                      </label>
                      <select
                        value={phpVersion}
                        onChange={(e) => setPhpVersion(e.target.value)}
                        className="w-full px-3.5 py-2.5 rounded-xl bg-surface-950 border border-surface-700 text-slate-100 text-sm focus:outline-none focus:border-indigo-500"
                      >
                        <option value="8.4">PHP 8.4 (Latest)</option>
                        <option value="8.3">PHP 8.3 (Stable)</option>
                        <option value="8.2">PHP 8.2</option>
                        <option value="8.1">PHP 8.1</option>
                      </select>
                    </div>
                  )}

                  {appType === 'proxy' && (
                    <div>
                      <label className="block text-xs font-semibold uppercase tracking-wider text-slate-300 mb-1.5">
                        Internal Port
                      </label>
                      <input
                        type="number"
                        value={proxyPort}
                        onChange={(e) => setProxyPort(Number(e.target.value))}
                        className="w-full px-3.5 py-2.5 rounded-xl bg-surface-950 border border-surface-700 text-slate-100 text-sm focus:outline-none focus:border-indigo-500"
                      />
                    </div>
                  )}
                </div>

                <div className="p-3 rounded-xl bg-surface-950/60 border border-surface-800 text-[11px] text-slate-400">
                  <p>• Document Root: <span className="font-mono text-slate-300">/var/www/{domain || 'domain'}/public</span></p>
                  <p>• Automated Nginx atomic swap and <span className="font-mono text-slate-300">nginx -t</span> syntax test before reload.</p>
                </div>

                <div className="flex justify-end gap-3 pt-3">
                  <button
                    type="button"
                    onClick={() => setModalOpen(false)}
                    className="px-4 py-2 rounded-xl text-sm text-slate-400 hover:text-white transition-colors"
                  >
                    Cancel
                  </button>
                  <button
                    type="submit"
                    disabled={creating || !selectedServer}
                    className="px-5 py-2 rounded-xl bg-indigo-600 hover:bg-indigo-500 disabled:bg-indigo-600/50 text-white text-sm font-semibold shadow-md transition-all flex items-center gap-2"
                  >
                    {creating ? (
                      <div className="w-4 h-4 border-2 border-white border-t-transparent rounded-full animate-spin" />
                    ) : (
                      'Provision Website'
                    )}
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
