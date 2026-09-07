'use client';

import React, { useState, useEffect } from 'react';
import Link from 'next/link';
import {
  ShieldCheck,
  ShieldAlert,
  Lock,
  RefreshCw,
  Search,
  CheckCircle2,
  Clock,
  Globe,
  X,
  Plus,
  Key,
  Trash2,
  Copy,
  Check,
  AlertTriangle,
  Info,
  Sparkles,
  ExternalLink,
  ChevronRight,
  Upload,
} from 'lucide-react';
import { DashboardShell } from '@/components/DashboardShell';
import { apiFetch, SSLCertificate, Website, SSLChallengeInfo } from '@/lib/api';

export default function SSLPage() {
  const [certificates, setCertificates] = useState<SSLCertificate[]>([]);
  const [websites, setWebsites] = useState<Website[]>([]);
  const [loading, setLoading] = useState(true);
  const [search, setSearch] = useState('');
  const [filterType, setFilterType] = useState<'all' | 'wildcard' | 'single' | 'expiring'>('all');
  
  // Modals
  const [showIssueModal, setShowIssueModal] = useState(false);
  const [showCustomModal, setShowCustomModal] = useState(false);
  const [selectedCert, setSelectedCert] = useState<SSLCertificate | null>(null);
  const [deleteCert, setDeleteCert] = useState<SSLCertificate | null>(null);
  
  // Issuance State
  const [issueMode, setIssueMode] = useState<'wildcard' | 'standard'>('wildcard');
  const [issueDomain, setIssueDomain] = useState('');
  const [issueProvider, setIssueProvider] = useState<'cloudflare' | 'digitalocean' | 'local' | 'manual'>('cloudflare');
  const [issueToken, setIssueToken] = useState('');
  const [issueZoneID, setIssueZoneID] = useState('');
  const [issueEmail, setIssueEmail] = useState('');
  const [issueSiteID, setIssueSiteID] = useState('');
  const [issuing, setIssuing] = useState(false);
  
  // Manual Challenge State
  const [activeChallenge, setActiveChallenge] = useState<SSLChallengeInfo | null>(null);
  const [verifyingChallenge, setVerifyingChallenge] = useState(false);
  const [copiedField, setCopiedField] = useState<string | null>(null);

  // Custom Cert State
  const [customDomain, setCustomDomain] = useState('');
  const [customCertPEM, setCustomCertPEM] = useState('');
  const [customKeyPEM, setCustomKeyPEM] = useState('');
  const [customChainPEM, setCustomChainPEM] = useState('');
  const [customSiteID, setCustomSiteID] = useState('');
  const [importing, setImporting] = useState(false);

  // Action states
  const [renewingId, setRenewingId] = useState<string | null>(null);
  const [scanningAutoRenew, setScanningAutoRenew] = useState(false);
  const [actionError, setActionError] = useState<string | null>(null);
  const [actionSuccess, setActionSuccess] = useState<string | null>(null);

  const fetchData = async () => {
    try {
      setLoading(true);
      const [certRes, siteRes] = await Promise.all([
        apiFetch<SSLCertificate[]>('/api/v1/ssl/certificates'),
        apiFetch<Website[]>('/api/v1/websites'),
      ]);

      if (certRes.success && certRes.data) {
        setCertificates(certRes.data);
      } else {
        setCertificates([]);
      }

      if (siteRes.success && siteRes.data) {
        setWebsites(siteRes.data);
      } else {
        setWebsites([]);
      }
    } catch {
      setCertificates([]);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchData();
  }, []);

  const handleCopy = (text: string, field: string) => {
    navigator.clipboard.writeText(text);
    setCopiedField(field);
    setTimeout(() => setCopiedField(null), 2000);
  };

  // 1. Issue or Prepare Challenge
  const handleIssueSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!issueDomain.trim()) return;

    setActionError(null);
    setActionSuccess(null);

    // If manual DNS-01, prepare challenge and present TXT record
    if (issueMode === 'wildcard' && issueProvider === 'manual') {
      try {
        setIssuing(true);
        const res = await apiFetch<SSLChallengeInfo>('/api/v1/ssl/challenge', {
          method: 'POST',
          body: JSON.stringify({
            primary_domain: issueDomain.trim(),
            wildcard: true,
            provider: 'manual',
            email: issueEmail.trim(),
            website_id: issueSiteID,
          }),
        });

        if (res.success && res.data) {
          setActiveChallenge(res.data);
        } else {
          setActionError(res.error?.message || 'Failed to prepare manual challenge');
        }
      } catch (err: any) {
        setActionError(err.message || 'Challenge preparation failed');
      } finally {
        setIssuing(false);
      }
      return;
    }

    // Automated issuance (Cloudflare, DigitalOcean, Local, or HTTP-01)
    try {
      setIssuing(true);
      const res = await apiFetch<SSLCertificate>('/api/v1/ssl/issue', {
        method: 'POST',
        body: JSON.stringify({
          primary_domain: issueDomain.trim(),
          wildcard: issueMode === 'wildcard',
          provider: issueMode === 'wildcard' ? issueProvider : 'http01',
          provider_token: issueToken.trim(),
          zone_id: issueZoneID.trim(),
          email: issueEmail.trim(),
          website_id: issueSiteID,
        }),
      });

      if (res.success) {
        setActionSuccess(`Certificate for ${issueDomain} issued and activated successfully!`);
        setShowIssueModal(false);
        resetIssueForm();
        await fetchData();
      } else {
        setActionError(res.error?.message || 'Issuance failed');
      }
    } catch (err: any) {
      setActionError(err.message || 'Issuance failed');
    } finally {
      setIssuing(false);
    }
  };

  // 2. Verify Manual Challenge & Issue
  const handleVerifyManualChallenge = async () => {
    if (!activeChallenge) return;
    try {
      setVerifyingChallenge(true);
      setActionError(null);

      const res = await apiFetch<any>('/api/v1/ssl/verify-challenge', {
        method: 'POST',
        body: JSON.stringify({
          domain: activeChallenge.domain,
          txt_host: activeChallenge.txt_host,
          txt_value: activeChallenge.txt_value,
          complete_issue: true,
          website_id: issueSiteID,
        }),
      });

      if (res.success) {
        if (res.data?.propagated === false) {
          setActionError(res.data?.message || 'DNS record not yet detected. Please wait a moment and try again.');
        } else {
          setActionSuccess(`Wildcard certificate for *.${activeChallenge.domain} successfully verified and installed!`);
          setActiveChallenge(null);
          setShowIssueModal(false);
          resetIssueForm();
          await fetchData();
        }
      } else {
        setActionError(res.error?.message || 'Verification failed');
      }
    } catch (err: any) {
      setActionError(err.message || 'Verification error');
    } finally {
      setVerifyingChallenge(false);
    }
  };

  // 3. Custom Cert Import
  const handleCustomImport = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!customDomain.trim() || !customCertPEM.trim() || !customKeyPEM.trim()) return;

    try {
      setImporting(true);
      setActionError(null);

      const res = await apiFetch<SSLCertificate>('/api/v1/ssl/custom', {
        method: 'POST',
        body: JSON.stringify({
          domain: customDomain.trim(),
          cert_pem: customCertPEM.trim(),
          key_pem: customKeyPEM.trim(),
          chain_pem: customChainPEM.trim(),
          website_id: customSiteID,
        }),
      });

      if (res.success) {
        setActionSuccess(`Custom SSL certificate for ${customDomain} installed successfully!`);
        setShowCustomModal(false);
        resetCustomForm();
        await fetchData();
      } else {
        setActionError(res.error?.message || 'Failed to import certificate');
      }
    } catch (err: any) {
      setActionError(err.message || 'Failed to import certificate');
    } finally {
      setImporting(false);
    }
  };

  // 4. Force Renew
  const handleRenew = async (id: string, domain: string) => {
    try {
      setRenewingId(id);
      setActionError(null);

      const res = await apiFetch<SSLCertificate>(`/api/v1/ssl/renew/${id}`, { method: 'POST' });
      if (res.success) {
        setActionSuccess(`Certificate for ${domain} renewed successfully!`);
        await fetchData();
      } else {
        setActionError(res.error?.message || 'Renewal failed');
      }
    } catch (err: any) {
      setActionError(err.message || 'Renewal failed');
    } finally {
      setRenewingId(null);
    }
  };

  // 5. Auto Renew Scan
  const handleAutoRenewScan = async () => {
    try {
      setScanningAutoRenew(true);
      setActionError(null);

      const res = await apiFetch<{ renewed_count: number }>('/api/v1/ssl/auto-renew', { method: 'POST' });
      if (res.success) {
        setActionSuccess(`Auto-renew scan completed. ${res.data?.renewed_count || 0} certificates processed.`);
        await fetchData();
      } else {
        setActionError(res.error?.message || 'Scan failed');
      }
    } catch (err: any) {
      setActionError(err.message || 'Auto-renew scan failed');
    } finally {
      setScanningAutoRenew(false);
    }
  };

  // 6. Delete Certificate
  const handleDeleteCert = async () => {
    if (!deleteCert) return;
    try {
      const res = await apiFetch(`/api/v1/ssl/${deleteCert.id}`, { method: 'DELETE' });
      if (res.success) {
        setActionSuccess('Certificate removed from system.');
        setDeleteCert(null);
        await fetchData();
      } else {
        setActionError(res.error?.message || 'Delete failed');
      }
    } catch (err: any) {
      setActionError(err.message || 'Delete failed');
    }
  };

  const resetIssueForm = () => {
    setIssueDomain('');
    setIssueToken('');
    setIssueZoneID('');
    setIssueEmail('');
    setIssueSiteID('');
    setActiveChallenge(null);
  };

  const resetCustomForm = () => {
    setCustomDomain('');
    setCustomCertPEM('');
    setCustomKeyPEM('');
    setCustomChainPEM('');
    setCustomSiteID('');
  };

  // Filter logic
  const filtered = certificates.filter((c) => {
    const domainMatch = c.domain_list.some((d) => d.toLowerCase().includes(search.toLowerCase())) ||
      c.issuer.toLowerCase().includes(search.toLowerCase());
    if (!domainMatch) return false;

    if (filterType === 'wildcard') return c.is_wildcard;
    if (filterType === 'single') return !c.is_wildcard;
    if (filterType === 'expiring') return (c.days_remaining !== undefined && c.days_remaining <= 30);
    return true;
  });

  // Metrics
  const totalCerts = certificates.length;
  const wildcardCount = certificates.filter((c) => c.is_wildcard).length;
  const expiringSoonCount = certificates.filter((c) => c.days_remaining !== undefined && c.days_remaining <= 30).length;
  const minExpiry = certificates.reduce((min, c) => {
    if (c.days_remaining !== undefined && (min === null || c.days_remaining < min)) {
      return c.days_remaining;
    }
    return min;
  }, null as number | null);

  return (
    <DashboardShell>
      <div className="space-y-6 animate-fadeIn max-w-7xl mx-auto pb-12">
        {/* Header */}
        <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
          <div>
            <div className="flex items-center gap-2.5">
              <h1 className="text-2xl font-bold tracking-tight text-slate-950 dark:text-white">
                SSL / TLS Certificates
              </h1>
              <span className="px-2.5 py-0.5 rounded-full text-xs font-bold bg-purple-500/10 text-purple-600 dark:text-purple-400 border border-purple-500/20">
                DNS-01 Wildcard
              </span>
            </div>
            <p className="text-sm text-slate-600 dark:text-slate-400 mt-1">
              Automated ACME Let&apos;s Encrypt DNS-01 wildcard certificates, Cloudflare integration, and custom SSL import.
            </p>
          </div>

          <div className="flex items-center gap-2.5 flex-wrap">
            <button
              onClick={handleAutoRenewScan}
              disabled={scanningAutoRenew}
              className="inline-flex items-center gap-1.5 px-3 py-2 rounded-xl bg-white dark:bg-surface-900 border border-slate-300 dark:border-surface-800 text-slate-700 dark:text-slate-300 hover:text-slate-950 dark:hover:text-white hover:bg-slate-100 dark:hover:bg-surface-800 text-xs font-semibold transition-colors shadow-xs disabled:opacity-50"
              title="Scan all certificates expiring within 30 days and renew automatically"
            >
              <RefreshCw className={`w-3.5 h-3.5 ${scanningAutoRenew ? 'animate-spin' : ''}`} />
              <span>Auto-Renew Scan</span>
            </button>

            <button
              onClick={() => setShowCustomModal(true)}
              className="inline-flex items-center gap-1.5 px-3 py-2 rounded-xl bg-white dark:bg-surface-900 border border-slate-300 dark:border-surface-800 text-slate-700 dark:text-slate-300 hover:text-slate-950 dark:hover:text-white hover:bg-slate-100 dark:hover:bg-surface-800 text-xs font-semibold transition-colors shadow-xs"
            >
              <Upload className="w-3.5 h-3.5" />
              <span>Import Custom</span>
            </button>

            <button
              onClick={() => setShowIssueModal(true)}
              className="inline-flex items-center gap-1.5 px-4 py-2 rounded-xl bg-gradient-to-r from-emerald-600 to-teal-600 hover:from-emerald-500 hover:to-teal-500 text-white text-xs font-bold shadow-md shadow-emerald-500/10 transition-all"
            >
              <Plus className="w-4 h-4" />
              <span>Issue Certificate</span>
            </button>
          </div>
        </div>

        {/* Notifications */}
        {actionSuccess && (
          <div className="p-3.5 rounded-xl bg-emerald-500/10 border border-emerald-500/20 text-emerald-700 dark:text-emerald-400 text-xs font-medium flex items-center justify-between">
            <div className="flex items-center gap-2">
              <CheckCircle2 className="w-4 h-4 flex-shrink-0" />
              <span>{actionSuccess}</span>
            </div>
            <button onClick={() => setActionSuccess(null)} className="text-emerald-600 hover:text-emerald-800">
              <X className="w-3.5 h-3.5" />
            </button>
          </div>
        )}

        {actionError && (
          <div className="p-3.5 rounded-xl bg-rose-500/10 border border-rose-500/20 text-rose-700 dark:text-rose-400 text-xs font-medium flex items-center justify-between">
            <div className="flex items-center gap-2">
              <AlertTriangle className="w-4 h-4 flex-shrink-0" />
              <span>{actionError}</span>
            </div>
            <button onClick={() => setActionError(null)} className="text-rose-600 hover:text-rose-800">
              <X className="w-3.5 h-3.5" />
            </button>
          </div>
        )}

        {/* Top Metric Cards */}
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
          <div className="p-4 rounded-2xl bg-white dark:bg-[#10141d] border border-slate-200 dark:border-surface-800 shadow-xs">
            <div className="flex items-center justify-between">
              <span className="text-xs font-semibold text-slate-500 dark:text-slate-400">Total Installed</span>
              <div className="p-2 rounded-xl bg-emerald-500/10 text-emerald-600 dark:text-emerald-400">
                <Lock className="w-4 h-4" />
              </div>
            </div>
            <div className="mt-3">
              <span className="text-2xl font-black text-slate-950 dark:text-white">{totalCerts}</span>
              <p className="text-[11px] text-slate-500 dark:text-slate-400 mt-0.5">Active SSL protection active</p>
            </div>
          </div>

          <div className="p-4 rounded-2xl bg-white dark:bg-[#10141d] border border-slate-200 dark:border-surface-800 shadow-xs">
            <div className="flex items-center justify-between">
              <span className="text-xs font-semibold text-slate-500 dark:text-slate-400">Wildcards (*.domain)</span>
              <div className="p-2 rounded-xl bg-purple-500/10 text-purple-600 dark:text-purple-400">
                <Sparkles className="w-4 h-4" />
              </div>
            </div>
            <div className="mt-3">
              <span className="text-2xl font-black text-purple-600 dark:text-purple-400">{wildcardCount}</span>
              <p className="text-[11px] text-slate-500 dark:text-slate-400 mt-0.5">DNS-01 verified wildcards</p>
            </div>
          </div>

          <div className="p-4 rounded-2xl bg-white dark:bg-[#10141d] border border-slate-200 dark:border-surface-800 shadow-xs">
            <div className="flex items-center justify-between">
              <span className="text-xs font-semibold text-slate-500 dark:text-slate-400">Expiring Soon (&le;30d)</span>
              <div className="p-2 rounded-xl bg-amber-500/10 text-amber-600 dark:text-amber-400">
                <Clock className="w-4 h-4" />
              </div>
            </div>
            <div className="mt-3">
              <span className={`text-2xl font-black ${expiringSoonCount > 0 ? 'text-amber-500' : 'text-slate-950 dark:text-white'}`}>
                {expiringSoonCount}
              </span>
              <p className="text-[11px] text-slate-500 dark:text-slate-400 mt-0.5">Auto-renewal eligible</p>
            </div>
          </div>

          <div className="p-4 rounded-2xl bg-white dark:bg-[#10141d] border border-slate-200 dark:border-surface-800 shadow-xs">
            <div className="flex items-center justify-between">
              <span className="text-xs font-semibold text-slate-500 dark:text-slate-400">Auto-Renewal Engine</span>
              <div className="p-2 rounded-xl bg-teal-500/10 text-teal-600 dark:text-teal-400">
                <CheckCircle2 className="w-4 h-4" />
              </div>
            </div>
            <div className="mt-3">
              <div className="flex items-center gap-1.5 text-emerald-600 dark:text-emerald-400 font-bold text-sm">
                <span className="w-2 h-2 rounded-full bg-emerald-500 animate-pulse" />
                Active Daemon
              </div>
              <p className="text-[11px] text-slate-500 dark:text-slate-400 mt-0.5">Checks expiration daily</p>
            </div>
          </div>
        </div>

        {/* Filters and Search Bar */}
        <div className="flex flex-col sm:flex-row items-center justify-between gap-4 bg-white dark:bg-[#10141d] border border-slate-200 dark:border-surface-800 rounded-2xl p-3 shadow-xs">
          <div className="flex items-center gap-3 w-full sm:w-80 bg-slate-50 dark:bg-[#121824] border border-slate-300 dark:border-surface-700 rounded-xl px-3.5 py-2 shadow-xs focus-within:border-emerald-500 focus-within:ring-2 focus-within:ring-emerald-500/20 transition-all">
            <Search className="w-4 h-4 text-slate-400 dark:text-slate-500 flex-shrink-0" />
            <input
              type="text"
              placeholder="Search domain or issuer..."
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

          <div className="flex items-center gap-1.5 self-start sm:self-center">
            {(['all', 'wildcard', 'single', 'expiring'] as const).map((t) => (
              <button
                key={t}
                onClick={() => setFilterType(t)}
                className={`px-3 py-1.5 rounded-xl text-xs font-semibold transition-colors capitalize ${
                  filterType === t
                    ? 'bg-slate-950 dark:bg-white text-white dark:text-slate-950 shadow-xs'
                    : 'text-slate-600 dark:text-slate-400 hover:bg-slate-100 dark:hover:bg-surface-800'
                }`}
              >
                {t === 'wildcard' ? 'Wildcards (*)' : t}
              </button>
            ))}
          </div>
        </div>

        {/* Certificates Table */}
        {loading ? (
          <div className="py-20 text-center text-slate-500 dark:text-slate-400">
            <div className="w-8 h-8 border-2 border-emerald-500 border-t-transparent rounded-full animate-spin mx-auto mb-3" />
            Loading certificate inventory...
          </div>
        ) : filtered.length === 0 ? (
          <div className="py-16 text-center border border-dashed border-slate-200 dark:border-surface-800 rounded-2xl bg-white dark:bg-surface-900/50 shadow-xs">
            <Lock className="w-12 h-12 mx-auto text-slate-400 dark:text-slate-600 mb-3" />
            <h3 className="text-base font-bold text-slate-950 dark:text-white">No certificates found</h3>
            <p className="text-xs text-slate-600 dark:text-slate-400 mt-1 max-w-sm mx-auto">
              Issue your first automated Let&apos;s Encrypt wildcard certificate or import custom PEM keys.
            </p>
            <button
              onClick={() => setShowIssueModal(true)}
              className="mt-4 inline-flex items-center gap-2 px-4 py-2 rounded-xl bg-emerald-600 hover:bg-emerald-500 text-white text-xs font-semibold shadow-md transition-all"
            >
              <Plus className="w-3.5 h-3.5" />
              Issue Wildcard SSL
            </button>
          </div>
        ) : (
          <div className="bg-white dark:bg-[#10141d] border border-slate-200 dark:border-surface-800 rounded-2xl overflow-hidden shadow-xs dark:shadow-xl">
            <div className="overflow-x-auto">
              <table className="w-full text-left text-sm border-collapse">
                <thead>
                  <tr className="border-b border-slate-200 dark:border-surface-800 bg-slate-50 dark:bg-[#121824] text-slate-700 dark:text-slate-300 text-[11px] font-bold uppercase tracking-wider">
                    <th className="px-6 py-3.5">Protected Domain</th>
                    <th className="px-6 py-3.5">Type & Provider</th>
                    <th className="px-6 py-3.5">Issuer</th>
                    <th className="px-6 py-3.5">Validity & Expiration</th>
                    <th className="px-6 py-3.5">Auto-Renew</th>
                    <th className="px-6 py-3.5 text-right">Actions</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-slate-200/80 dark:divide-surface-800/80">
                  {filtered.map((cert) => {
                    const primary = cert.domain_list[0] || 'Unknown';
                    const days = cert.days_remaining ?? Math.max(0, Math.floor((new Date(cert.expires_at).getTime() - Date.now()) / (1000 * 60 * 60 * 24)));
                    const isExpiring = days <= 30;

                    return (
                      <tr key={cert.id} className="hover:bg-slate-50 dark:hover:bg-[#151d2d] transition-colors">
                        <td className="px-6 py-4">
                          <div className="flex items-center gap-2.5">
                            <div className={`p-2 rounded-xl ${cert.is_wildcard ? 'bg-purple-500/10 text-purple-600 dark:text-purple-400' : 'bg-emerald-500/10 text-emerald-600 dark:text-emerald-400'}`}>
                              <Lock className="w-4 h-4" />
                            </div>
                            <div>
                              <div className="font-bold text-sm text-slate-950 dark:text-white flex items-center gap-2">
                                {primary}
                                {cert.is_wildcard && (
                                  <span className="px-1.5 py-0.5 rounded text-[10px] font-black bg-purple-500/10 text-purple-600 dark:text-purple-400 border border-purple-500/20 uppercase tracking-wider">
                                    *.{primary}
                                  </span>
                                )}
                              </div>
                              {cert.domain_list.length > 1 && (
                                <div className="text-[11px] text-slate-500 dark:text-slate-400 mt-0.5">
                                  +{cert.domain_list.length - 1} Subject Alt Names
                                </div>
                              )}
                            </div>
                          </div>
                        </td>

                        <td className="px-6 py-4">
                          <div className="flex items-center gap-1.5 flex-wrap">
                            {cert.is_wildcard ? (
                              <span className="px-2 py-0.5 rounded-full text-[11px] font-semibold bg-purple-500/10 text-purple-600 dark:text-purple-400 border border-purple-500/20">
                                Wildcard DNS-01
                              </span>
                            ) : (
                              <span className="px-2 py-0.5 rounded-full text-[11px] font-semibold bg-blue-500/10 text-blue-600 dark:text-blue-400 border border-blue-500/20">
                                Single Domain
                              </span>
                            )}
                            {cert.dns_provider && (
                              <span className="px-2 py-0.5 rounded-full text-[11px] font-semibold bg-slate-100 dark:bg-surface-800 text-slate-600 dark:text-slate-300 uppercase text-[10px]">
                                {cert.dns_provider}
                              </span>
                            )}
                          </div>
                        </td>

                        <td className="px-6 py-4 text-xs font-medium text-slate-700 dark:text-slate-300">
                          {cert.issuer || "Let's Encrypt Authority"}
                        </td>

                        <td className="px-6 py-4">
                          <div className="space-y-1">
                            <div className="flex items-center gap-2">
                              <span className={`inline-flex items-center gap-1 text-xs font-bold ${
                                isExpiring ? 'text-amber-600 dark:text-amber-400' : 'text-emerald-600 dark:text-emerald-400'
                              }`}>
                                <CheckCircle2 className="w-3.5 h-3.5" />
                                <span>{days} days remaining</span>
                              </span>
                            </div>
                            <div className="w-32 bg-slate-100 dark:bg-surface-800 rounded-full h-1.5 overflow-hidden">
                              <div
                                className={`h-full rounded-full transition-all ${
                                  days <= 15
                                    ? 'bg-rose-500'
                                    : days <= 30
                                    ? 'bg-amber-500'
                                    : 'bg-emerald-500'
                                }`}
                                style={{ width: `${Math.min(100, Math.max(5, (days / 90) * 100))}%` }}
                              />
                            </div>
                            <div className="text-[10px] text-slate-500 dark:text-slate-400">
                              Expires {new Date(cert.expires_at).toLocaleDateString()}
                            </div>
                          </div>
                        </td>

                        <td className="px-6 py-4">
                          {cert.auto_renew ? (
                            <span className="inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full text-[11px] font-semibold bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 border border-emerald-500/20">
                              <span className="w-1.5 h-1.5 rounded-full bg-emerald-500" />
                              Auto (&le;30d)
                            </span>
                          ) : (
                            <span className="inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full text-[11px] font-semibold bg-slate-200 dark:bg-surface-800 text-slate-600 dark:text-slate-400">
                              Manual
                            </span>
                          )}
                        </td>

                        <td className="px-6 py-4 text-right">
                          <div className="flex items-center justify-end gap-1.5">
                            <button
                              onClick={() => setSelectedCert(cert)}
                              className="p-1.5 rounded-lg border border-slate-300 dark:border-surface-700 bg-white dark:bg-surface-800 hover:bg-slate-100 dark:hover:bg-surface-700 text-slate-700 dark:text-slate-300 text-xs transition-colors shadow-xs"
                              title="View certificate details"
                            >
                              <Info className="w-3.5 h-3.5" />
                            </button>

                            <button
                              onClick={() => handleRenew(cert.id, primary)}
                              disabled={renewingId === cert.id}
                              className="inline-flex items-center gap-1 px-2.5 py-1.5 rounded-lg border border-slate-300 dark:border-surface-700 bg-white dark:bg-surface-800 hover:bg-slate-100 dark:hover:bg-surface-700 text-slate-800 dark:text-slate-200 text-xs font-semibold transition-colors disabled:opacity-50 shadow-xs"
                              title="Force certificate renewal"
                            >
                              <RefreshCw className={`w-3 h-3 ${renewingId === cert.id ? 'animate-spin' : ''}`} />
                              <span>Renew</span>
                            </button>

                            <button
                              onClick={() => setDeleteCert(cert)}
                              className="p-1.5 rounded-lg border border-rose-200 dark:border-rose-900/40 bg-rose-50 dark:bg-rose-950/20 hover:bg-rose-100 dark:hover:bg-rose-900/30 text-rose-600 dark:text-rose-400 text-xs transition-colors"
                              title="Revoke / Delete certificate"
                            >
                              <Trash2 className="w-3.5 h-3.5" />
                            </button>
                          </div>
                        </td>
                      </tr>
                    );
                  })}
                </tbody>
              </table>
            </div>
          </div>
        )}

        {/* ISSUE CERTIFICATE MODAL */}
        {showIssueModal && (
          <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/60 backdrop-blur-xs animate-fadeIn">
            <div className="bg-white dark:bg-[#10141d] border border-slate-200 dark:border-surface-800 rounded-2xl w-full max-w-xl max-h-[90vh] overflow-y-auto shadow-2xl">
              <div className="flex items-center justify-between p-5 border-b border-slate-200 dark:border-surface-800">
                <div className="flex items-center gap-2">
                  <div className="p-2 rounded-xl bg-emerald-500/10 text-emerald-600 dark:text-emerald-400">
                    <ShieldCheck className="w-5 h-5" />
                  </div>
                  <div>
                    <h2 className="text-base font-bold text-slate-950 dark:text-white">Issue SSL / TLS Certificate</h2>
                    <p className="text-xs text-slate-500 dark:text-slate-400">ACME RFC 8555 Automated Certificate Issuance</p>
                  </div>
                </div>
                <button onClick={() => { setShowIssueModal(false); setActiveChallenge(null); }} className="p-1.5 rounded-lg text-slate-400 hover:text-slate-600 dark:hover:text-white">
                  <X className="w-4 h-4" />
                </button>
              </div>

              {/* Mode Switcher */}
              <div className="p-5 pb-0">
                <div className="grid grid-cols-2 gap-2 bg-slate-100 dark:bg-[#121824] p-1 rounded-xl">
                  <button
                    type="button"
                    onClick={() => { setIssueMode('wildcard'); setActiveChallenge(null); }}
                    className={`py-2 px-3 rounded-lg text-xs font-bold transition-all ${
                      issueMode === 'wildcard'
                        ? 'bg-white dark:bg-surface-800 text-purple-600 dark:text-purple-400 shadow-xs'
                        : 'text-slate-600 dark:text-slate-400 hover:text-slate-950'
                    }`}
                  >
                    ✨ Wildcard DNS-01 (*.domain)
                  </button>
                  <button
                    type="button"
                    onClick={() => { setIssueMode('standard'); setActiveChallenge(null); }}
                    className={`py-2 px-3 rounded-lg text-xs font-bold transition-all ${
                      issueMode === 'standard'
                        ? 'bg-white dark:bg-surface-800 text-emerald-600 dark:text-emerald-400 shadow-xs'
                        : 'text-slate-600 dark:text-slate-400 hover:text-slate-950'
                    }`}
                  >
                    Standard HTTP-01
                  </button>
                </div>
              </div>

              {/* Active Manual Challenge View */}
              {activeChallenge ? (
                <div className="p-5 space-y-4">
                  <div className="p-3.5 rounded-xl bg-purple-500/10 border border-purple-500/20 text-purple-700 dark:text-purple-300 text-xs">
                    <div className="font-bold flex items-center gap-1.5 mb-1">
                      <Sparkles className="w-3.5 h-3.5" />
                      Manual DNS-01 TXT Challenge Instructions
                    </div>
                    Add the following TXT record at your DNS provider or domain registrar to verify ownership of <strong>*.{activeChallenge.domain}</strong>:
                  </div>

                  <div className="space-y-3">
                    <div className="bg-slate-50 dark:bg-[#121824] p-3 rounded-xl border border-slate-200 dark:border-surface-700">
                      <div className="flex items-center justify-between text-[11px] font-semibold text-slate-500 dark:text-slate-400 mb-1">
                        <span>Record Type</span>
                        <span>TTL</span>
                      </div>
                      <div className="flex items-center justify-between font-mono font-bold text-xs text-slate-950 dark:text-white">
                        <span>TXT</span>
                        <span>120 seconds (or Auto)</span>
                      </div>
                    </div>

                    <div className="bg-slate-50 dark:bg-[#121824] p-3 rounded-xl border border-slate-200 dark:border-surface-700">
                      <div className="flex items-center justify-between text-[11px] font-semibold text-slate-500 dark:text-slate-400 mb-1">
                        <span>Host / Name</span>
                        <button
                          type="button"
                          onClick={() => handleCopy(activeChallenge.txt_host, 'host')}
                          className="flex items-center gap-1 text-emerald-600 dark:text-emerald-400 hover:underline"
                        >
                          {copiedField === 'host' ? <Check className="w-3 h-3" /> : <Copy className="w-3 h-3" />}
                          <span>{copiedField === 'host' ? 'Copied!' : 'Copy'}</span>
                        </button>
                      </div>
                      <div className="font-mono text-xs font-bold text-slate-950 dark:text-white break-all">
                        {activeChallenge.txt_host}
                      </div>
                    </div>

                    <div className="bg-slate-50 dark:bg-[#121824] p-3 rounded-xl border border-slate-200 dark:border-surface-700">
                      <div className="flex items-center justify-between text-[11px] font-semibold text-slate-500 dark:text-slate-400 mb-1">
                        <span>TXT Value / Digest</span>
                        <button
                          type="button"
                          onClick={() => handleCopy(activeChallenge.txt_value, 'value')}
                          className="flex items-center gap-1 text-emerald-600 dark:text-emerald-400 hover:underline"
                        >
                          {copiedField === 'value' ? <Check className="w-3 h-3" /> : <Copy className="w-3 h-3" />}
                          <span>{copiedField === 'value' ? 'Copied!' : 'Copy'}</span>
                        </button>
                      </div>
                      <div className="font-mono text-xs font-bold text-slate-950 dark:text-white break-all">
                        {activeChallenge.txt_value}
                      </div>
                    </div>
                  </div>

                  <div className="flex items-center justify-end gap-2 pt-3 border-t border-slate-200 dark:border-surface-800">
                    <button
                      type="button"
                      onClick={() => setActiveChallenge(null)}
                      className="px-4 py-2 rounded-xl border border-slate-300 dark:border-surface-700 text-xs font-semibold text-slate-700 dark:text-slate-300"
                    >
                      Back
                    </button>
                    <button
                      type="button"
                      onClick={handleVerifyManualChallenge}
                      disabled={verifyingChallenge}
                      className="inline-flex items-center gap-2 px-4 py-2 rounded-xl bg-purple-600 hover:bg-purple-500 text-white text-xs font-bold shadow-md transition-all disabled:opacity-50"
                    >
                      <RefreshCw className={`w-3.5 h-3.5 ${verifyingChallenge ? 'animate-spin' : ''}`} />
                      <span>{verifyingChallenge ? 'Verifying DNS...' : 'Verify DNS & Complete Issue'}</span>
                    </button>
                  </div>
                </div>
              ) : (
                <form onSubmit={handleIssueSubmit} className="p-5 space-y-4">
                  {/* Domain Name */}
                  <div>
                    <label className="block text-xs font-bold text-slate-800 dark:text-slate-200 mb-1">
                      Primary Domain <span className="text-rose-500">*</span>
                    </label>
                    <input
                      type="text"
                      required
                      placeholder="e.g. example.com"
                      value={issueDomain}
                      onChange={(e) => setIssueDomain(e.target.value)}
                      className="w-full px-3.5 py-2.5 rounded-xl border border-slate-300 dark:border-surface-700 bg-slate-50 dark:bg-[#121824] text-xs font-medium text-slate-950 dark:text-white placeholder:text-slate-400 focus:outline-none focus:border-emerald-500 focus:ring-2 focus:ring-emerald-500/20"
                    />
                    {issueMode === 'wildcard' && (
                      <p className="text-[11px] text-purple-600 dark:text-purple-400 mt-1 font-semibold">
                        Will automatically secure {issueDomain || 'example.com'} and *.{issueDomain || 'example.com'}
                      </p>
                    )}
                  </div>

                  {/* Associated Website */}
                  <div>
                    <label className="block text-xs font-bold text-slate-800 dark:text-slate-200 mb-1">
                      Attach to Website (Optional)
                    </label>
                    <select
                      value={issueSiteID}
                      onChange={(e) => {
                        setIssueSiteID(e.target.value);
                        const s = websites.find((w) => w.id === e.target.value);
                        if (s && !issueDomain) setIssueDomain(s.primary_domain);
                      }}
                      className="w-full px-3.5 py-2.5 rounded-xl border border-slate-300 dark:border-surface-700 bg-slate-50 dark:bg-[#121824] text-xs font-medium text-slate-950 dark:text-white focus:outline-none focus:border-emerald-500"
                    >
                      <option value="">None / Standalone Certificate</option>
                      {websites.map((w) => (
                        <option key={w.id} value={w.id}>
                          {w.primary_domain}
                        </option>
                      ))}
                    </select>
                  </div>

                  {/* DNS Provider for Wildcard */}
                  {issueMode === 'wildcard' && (
                    <div className="space-y-3 p-3.5 rounded-xl bg-slate-50 dark:bg-[#121824] border border-slate-200 dark:border-surface-700">
                      <div>
                        <label className="block text-xs font-bold text-slate-800 dark:text-slate-200 mb-1">
                          DNS-01 Provider <span className="text-rose-500">*</span>
                        </label>
                        <select
                          value={issueProvider}
                          onChange={(e: any) => setIssueProvider(e.target.value)}
                          className="w-full px-3.5 py-2.5 rounded-xl border border-slate-300 dark:border-surface-700 bg-white dark:bg-surface-800 text-xs font-semibold text-slate-950 dark:text-white focus:outline-none"
                        >
                          <option value="cloudflare">Cloudflare API (Recommended & Automated)</option>
                          <option value="digitalocean">DigitalOcean DNS</option>
                          <option value="local">Hostvra Local Bind9 DNS</option>
                          <option value="manual">Manual TXT Record (Any Registrar / Nameserver)</option>
                        </select>
                      </div>

                      {issueProvider === 'cloudflare' && (
                        <div className="space-y-2 pt-2 border-t border-slate-200 dark:border-surface-700">
                          <div>
                            <label className="block text-xs font-semibold text-slate-700 dark:text-slate-300 mb-1">
                              Cloudflare API Token <span className="text-rose-500">*</span>
                            </label>
                            <input
                              type="password"
                              required
                              placeholder="Zone.DNS:Edit scoped API token"
                              value={issueToken}
                              onChange={(e) => setIssueToken(e.target.value)}
                              className="w-full px-3.5 py-2 rounded-xl border border-slate-300 dark:border-surface-700 bg-white dark:bg-surface-800 text-xs text-slate-950 dark:text-white"
                            />
                          </div>
                          <div>
                            <label className="block text-[11px] font-medium text-slate-500 dark:text-slate-400 mb-0.5">
                              Zone ID (Optional, auto-detected if omitted)
                            </label>
                            <input
                              type="text"
                              placeholder="32-character hexadecimal Zone ID"
                              value={issueZoneID}
                              onChange={(e) => setIssueZoneID(e.target.value)}
                              className="w-full px-3.5 py-1.5 rounded-xl border border-slate-300 dark:border-surface-700 bg-white dark:bg-surface-800 text-xs text-slate-950 dark:text-white"
                            />
                          </div>
                        </div>
                      )}

                      {issueProvider === 'digitalocean' && (
                        <div className="pt-2 border-t border-slate-200 dark:border-surface-700">
                          <label className="block text-xs font-semibold text-slate-700 dark:text-slate-300 mb-1">
                            DigitalOcean Personal Access Token <span className="text-rose-500">*</span>
                          </label>
                          <input
                            type="password"
                            required
                            placeholder="dop_v1_..."
                            value={issueToken}
                            onChange={(e) => setIssueToken(e.target.value)}
                            className="w-full px-3.5 py-2 rounded-xl border border-slate-300 dark:border-surface-700 bg-white dark:bg-surface-800 text-xs text-slate-950 dark:text-white"
                          />
                        </div>
                      )}
                    </div>
                  )}

                  {/* Contact Email */}
                  <div>
                    <label className="block text-xs font-bold text-slate-800 dark:text-slate-200 mb-1">
                      ACME Contact Email
                    </label>
                    <input
                      type="email"
                      placeholder="admin@yourdomain.com"
                      value={issueEmail}
                      onChange={(e) => setIssueEmail(e.target.value)}
                      className="w-full px-3.5 py-2.5 rounded-xl border border-slate-300 dark:border-surface-700 bg-slate-50 dark:bg-[#121824] text-xs font-medium text-slate-950 dark:text-white"
                    />
                  </div>

                  <div className="flex items-center justify-end gap-2 pt-3 border-t border-slate-200 dark:border-surface-800">
                    <button
                      type="button"
                      onClick={() => setShowIssueModal(false)}
                      className="px-4 py-2 rounded-xl border border-slate-300 dark:border-surface-700 text-xs font-semibold text-slate-700 dark:text-slate-300"
                    >
                      Cancel
                    </button>
                    <button
                      type="submit"
                      disabled={issuing}
                      className="inline-flex items-center gap-2 px-5 py-2 rounded-xl bg-emerald-600 hover:bg-emerald-500 text-white text-xs font-bold shadow-md transition-all disabled:opacity-50"
                    >
                      <RefreshCw className={`w-3.5 h-3.5 ${issuing ? 'animate-spin' : ''}`} />
                      <span>{issuing ? 'Processing...' : issueMode === 'wildcard' && issueProvider === 'manual' ? 'Next: View DNS Challenge' : 'Issue Certificate'}</span>
                    </button>
                  </div>
                </form>
              )}
            </div>
          </div>
        )}

        {/* IMPORT CUSTOM CERTIFICATE MODAL */}
        {showCustomModal && (
          <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/60 backdrop-blur-xs animate-fadeIn">
            <div className="bg-white dark:bg-[#10141d] border border-slate-200 dark:border-surface-800 rounded-2xl w-full max-w-xl max-h-[90vh] overflow-y-auto shadow-2xl">
              <div className="flex items-center justify-between p-5 border-b border-slate-200 dark:border-surface-800">
                <div className="flex items-center gap-2">
                  <div className="p-2 rounded-xl bg-blue-500/10 text-blue-600 dark:text-blue-400">
                    <Upload className="w-5 h-5" />
                  </div>
                  <div>
                    <h2 className="text-base font-bold text-slate-950 dark:text-white">Import Custom SSL Certificate</h2>
                    <p className="text-xs text-slate-500 dark:text-slate-400">Upload existing X.509 PEM certificate and private key</p>
                  </div>
                </div>
                <button onClick={() => setShowCustomModal(false)} className="p-1.5 rounded-lg text-slate-400 hover:text-slate-600 dark:hover:text-white">
                  <X className="w-4 h-4" />
                </button>
              </div>

              <form onSubmit={handleCustomImport} className="p-5 space-y-4">
                <div>
                  <label className="block text-xs font-bold text-slate-800 dark:text-slate-200 mb-1">
                    Primary Domain <span className="text-rose-500">*</span>
                  </label>
                  <input
                    type="text"
                    required
                    placeholder="e.g. app.mycompany.com"
                    value={customDomain}
                    onChange={(e) => setCustomDomain(e.target.value)}
                    className="w-full px-3.5 py-2.5 rounded-xl border border-slate-300 dark:border-surface-700 bg-slate-50 dark:bg-[#121824] text-xs font-medium text-slate-950 dark:text-white"
                  />
                </div>

                <div>
                  <label className="block text-xs font-bold text-slate-800 dark:text-slate-200 mb-1">
                    Certificate PEM (fullchain.pem) <span className="text-rose-500">*</span>
                  </label>
                  <textarea
                    required
                    rows={4}
                    placeholder="-----BEGIN CERTIFICATE-----&#10;...&#10;-----END CERTIFICATE-----"
                    value={customCertPEM}
                    onChange={(e) => setCustomCertPEM(e.target.value)}
                    className="w-full px-3.5 py-2 rounded-xl border border-slate-300 dark:border-surface-700 bg-slate-50 dark:bg-[#121824] font-mono text-xs text-slate-950 dark:text-white"
                  />
                </div>

                <div>
                  <label className="block text-xs font-bold text-slate-800 dark:text-slate-200 mb-1">
                    Private Key PEM (privkey.pem) <span className="text-rose-500">*</span>
                  </label>
                  <textarea
                    required
                    rows={4}
                    placeholder="-----BEGIN RSA PRIVATE KEY-----&#10;...&#10;-----END RSA PRIVATE KEY-----"
                    value={customKeyPEM}
                    onChange={(e) => setCustomKeyPEM(e.target.value)}
                    className="w-full px-3.5 py-2 rounded-xl border border-slate-300 dark:border-surface-700 bg-slate-50 dark:bg-[#121824] font-mono text-xs text-slate-950 dark:text-white"
                  />
                </div>

                <div>
                  <label className="block text-xs font-bold text-slate-800 dark:text-slate-200 mb-1">
                    Attach to Website (Optional)
                  </label>
                  <select
                    value={customSiteID}
                    onChange={(e) => setCustomSiteID(e.target.value)}
                    className="w-full px-3.5 py-2.5 rounded-xl border border-slate-300 dark:border-surface-700 bg-slate-50 dark:bg-[#121824] text-xs font-medium text-slate-950 dark:text-white"
                  >
                    <option value="">None / Standalone Certificate</option>
                    {websites.map((w) => (
                      <option key={w.id} value={w.id}>
                        {w.primary_domain}
                      </option>
                    ))}
                  </select>
                </div>

                <div className="flex items-center justify-end gap-2 pt-3 border-t border-slate-200 dark:border-surface-800">
                  <button
                    type="button"
                    onClick={() => setShowCustomModal(false)}
                    className="px-4 py-2 rounded-xl border border-slate-300 dark:border-surface-700 text-xs font-semibold text-slate-700 dark:text-slate-300"
                  >
                    Cancel
                  </button>
                  <button
                    type="submit"
                    disabled={importing}
                    className="inline-flex items-center gap-2 px-5 py-2 rounded-xl bg-blue-600 hover:bg-blue-500 text-white text-xs font-bold shadow-md transition-all disabled:opacity-50"
                  >
                    <Upload className="w-3.5 h-3.5" />
                    <span>{importing ? 'Validating & Installing...' : 'Install Custom SSL'}</span>
                  </button>
                </div>
              </form>
            </div>
          </div>
        )}

        {/* VIEW CERTIFICATE DETAILS MODAL */}
        {selectedCert && (
          <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/60 backdrop-blur-xs animate-fadeIn">
            <div className="bg-white dark:bg-[#10141d] border border-slate-200 dark:border-surface-800 rounded-2xl w-full max-w-lg overflow-hidden shadow-2xl">
              <div className="flex items-center justify-between p-5 border-b border-slate-200 dark:border-surface-800">
                <div className="flex items-center gap-2">
                  <div className="p-2 rounded-xl bg-purple-500/10 text-purple-600 dark:text-purple-400">
                    <Key className="w-5 h-5" />
                  </div>
                  <div>
                    <h2 className="text-base font-bold text-slate-950 dark:text-white">
                      {selectedCert.domain_list[0]}
                    </h2>
                    <p className="text-xs text-slate-500 dark:text-slate-400">Certificate Specification & Diagnostics</p>
                  </div>
                </div>
                <button onClick={() => setSelectedCert(null)} className="p-1.5 rounded-lg text-slate-400 hover:text-slate-600 dark:hover:text-white">
                  <X className="w-4 h-4" />
                </button>
              </div>

              <div className="p-5 space-y-3.5 text-xs">
                <div className="flex justify-between py-1.5 border-b border-slate-100 dark:border-surface-800">
                  <span className="text-slate-500 dark:text-slate-400">Certificate Authority</span>
                  <span className="font-bold text-slate-950 dark:text-white">{selectedCert.issuer}</span>
                </div>

                <div className="flex justify-between py-1.5 border-b border-slate-100 dark:border-surface-800">
                  <span className="text-slate-500 dark:text-slate-400">Type</span>
                  <span className="font-bold text-purple-600 dark:text-purple-400">
                    {selectedCert.is_wildcard ? 'Wildcard DNS-01' : 'Single Domain'}
                  </span>
                </div>

                <div className="flex justify-between py-1.5 border-b border-slate-100 dark:border-surface-800">
                  <span className="text-slate-500 dark:text-slate-400">Issued On</span>
                  <span className="font-medium text-slate-950 dark:text-white">
                    {new Date(selectedCert.issued_at).toLocaleString()}
                  </span>
                </div>

                <div className="flex justify-between py-1.5 border-b border-slate-100 dark:border-surface-800">
                  <span className="text-slate-500 dark:text-slate-400">Expires On</span>
                  <span className="font-medium text-slate-950 dark:text-white">
                    {new Date(selectedCert.expires_at).toLocaleString()} ({selectedCert.days_remaining} days remaining)
                  </span>
                </div>

                <div className="py-1.5 border-b border-slate-100 dark:border-surface-800">
                  <span className="text-slate-500 dark:text-slate-400 block mb-1">Subject Alternative Names (SANs)</span>
                  <div className="flex flex-wrap gap-1.5">
                    {selectedCert.domain_list.map((d, i) => (
                      <span key={i} className="px-2 py-0.5 rounded bg-slate-100 dark:bg-surface-800 font-mono text-[11px] text-slate-800 dark:text-slate-200">
                        {d}
                      </span>
                    ))}
                  </div>
                </div>

                <div className="py-1.5">
                  <span className="text-slate-500 dark:text-slate-400 block mb-1">Filesystem Storage</span>
                  <div className="font-mono text-[11px] text-slate-700 dark:text-slate-300 bg-slate-50 dark:bg-[#121824] p-2 rounded-xl border border-slate-200 dark:border-surface-700 space-y-1">
                    <div>Cert: {selectedCert.cert_path}</div>
                    <div>Key:  {selectedCert.key_path}</div>
                  </div>
                </div>
              </div>

              <div className="p-4 bg-slate-50 dark:bg-[#121824] border-t border-slate-200 dark:border-surface-800 flex justify-end">
                <button
                  onClick={() => setSelectedCert(null)}
                  className="px-4 py-2 rounded-xl bg-slate-950 dark:bg-white text-white dark:text-slate-950 text-xs font-bold"
                >
                  Close
                </button>
              </div>
            </div>
          </div>
        )}

        {/* DELETE CONFIRMATION MODAL */}
        {deleteCert && (
          <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/60 backdrop-blur-xs animate-fadeIn">
            <div className="bg-white dark:bg-[#10141d] border border-slate-200 dark:border-surface-800 rounded-2xl w-full max-w-md p-6 shadow-2xl">
              <div className="flex items-center gap-3 text-rose-600 mb-4">
                <div className="p-3 rounded-full bg-rose-500/10">
                  <AlertTriangle className="w-6 h-6" />
                </div>
                <div>
                  <h3 className="text-base font-bold text-slate-950 dark:text-white">Delete Certificate</h3>
                  <p className="text-xs text-slate-500 dark:text-slate-400">Revoke and remove from web server</p>
                </div>
              </div>

              <p className="text-xs text-slate-600 dark:text-slate-400 mb-5 leading-relaxed">
                Are you sure you want to delete the SSL certificate for <strong>{deleteCert.domain_list[0]}</strong>?
                HTTPS connections will stop working until a new certificate is issued.
              </p>

              <div className="flex items-center justify-end gap-2">
                <button
                  onClick={() => setDeleteCert(null)}
                  className="px-4 py-2 rounded-xl border border-slate-300 dark:border-surface-700 text-xs font-semibold text-slate-700 dark:text-slate-300"
                >
                  Cancel
                </button>
                <button
                  onClick={handleDeleteCert}
                  className="px-4 py-2 rounded-xl bg-rose-600 hover:bg-rose-500 text-white text-xs font-bold transition-colors"
                >
                  Delete Certificate
                </button>
              </div>
            </div>
          </div>
        )}
      </div>
    </DashboardShell>
  );
}
