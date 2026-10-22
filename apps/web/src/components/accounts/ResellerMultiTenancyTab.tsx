'use client';

import React, { useState, useEffect } from 'react';
import {
  Users,
  HardDrive,
  Network,
  ShieldCheck,
  Globe,
  Mail,
  Palette,
  Save,
  RefreshCw,
  Sparkles,
  Layers,
  CheckCircle2,
  AlertCircle,
  ExternalLink,
} from 'lucide-react';
import {
  ResellerUsageSummary,
  BrandingInfo,
  getResellerUsage,
  getBranding,
  updateResellerBranding,
} from '@/lib/api';

interface ResellerMultiTenancyTabProps {
  showNotify: (type: 'success' | 'error', message: string) => void;
}

export function ResellerMultiTenancyTab({ showNotify }: ResellerMultiTenancyTabProps) {
  const [usage, setUsage] = useState<ResellerUsageSummary | null>(null);
  const [branding, setBranding] = useState<BrandingInfo>({
    brand_name: 'Hostvra',
    brand_logo_url: '',
    custom_nameservers: ['ns1.hostvra.com', 'ns2.hostvra.com'],
    custom_email_sender: 'noreply@hostvra.com',
  });
  const [loading, setLoading] = useState(true);
  const [savingBranding, setSavingBranding] = useState(false);

  // Form inputs
  const [brandName, setBrandName] = useState('');
  const [brandLogoUrl, setBrandLogoUrl] = useState('');
  const [ns1, setNs1] = useState('');
  const [ns2, setNs2] = useState('');
  const [ns3, setNs3] = useState('');
  const [ns4, setNs4] = useState('');
  const [emailSender, setEmailSender] = useState('');

  const loadData = async () => {
    setLoading(true);
    try {
      const [usageRes, brandRes] = await Promise.all([
        getResellerUsage(),
        getBranding(),
      ]);

      if (usageRes.data) {
        setUsage(usageRes.data);
      }
      if (brandRes.data) {
        setBranding(brandRes.data);
        setBrandName(brandRes.data.brand_name || '');
        setBrandLogoUrl(brandRes.data.brand_logo_url || '');
        setEmailSender(brandRes.data.custom_email_sender || '');
        const ns = brandRes.data.custom_nameservers || [];
        setNs1(ns[0] || '');
        setNs2(ns[1] || '');
        setNs3(ns[2] || '');
        setNs4(ns[3] || '');
      }
    } catch (err: any) {
      console.warn('Reseller data notice:', err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadData();
  }, []);

  const handleSaveBranding = async (e: React.FormEvent) => {
    e.preventDefault();
    setSavingBranding(true);
    try {
      const nsList = [ns1, ns2, ns3, ns4].map(s => s.trim()).filter(Boolean);
      const res = await updateResellerBranding({
        brand_name: brandName.trim(),
        brand_logo_url: brandLogoUrl.trim(),
        custom_nameservers: nsList.length > 0 ? nsList : ['ns1.hostvra.com', 'ns2.hostvra.com'],
        custom_email_sender: emailSender.trim(),
      });

      if (res.data) {
        setBranding(res.data);
        showNotify('success', 'White-label branding updated successfully!');
      } else {
        showNotify('error', res.error?.message || 'Failed to update branding');
      }
    } catch (err: any) {
      showNotify('error', err.message || 'Server error updating branding');
    } finally {
      setSavingBranding(false);
    }
  };

  // Safe percentage calculation
  const getPercent = (used: number, total: number) => {
    if (!total || total <= 0) return 0;
    const p = Math.round((used / total) * 100);
    return Math.min(100, Math.max(0, p));
  };

  const accountPercent = usage ? getPercent(usage.used_accounts, usage.max_accounts) : 0;
  const diskPercent = usage ? getPercent(Number(usage.used_disk_mb), Number(usage.allocated_disk_mb)) : 0;
  const bwPercent = usage ? getPercent(Number(usage.used_bandwidth_mb), Number(usage.allocated_bandwidth_mb)) : 0;

  return (
    <div className="space-y-8 animate-in fade-in duration-200">
      {/* Top Summary Banner */}
      <div className="rounded-3xl bg-white p-6 sm:p-8 border border-slate-200 shadow-sm flex flex-col md:flex-row md:items-center justify-between gap-6">
        <div className="space-y-2">
          <div className="inline-flex items-center gap-2 px-3 py-1 rounded-full bg-purple-50 border border-purple-100 text-xs font-semibold text-purple-700">
            <Sparkles className="w-3.5 h-3.5 text-purple-600" />
            <span>Reseller Multi-Tenancy & White-Label Suite</span>
          </div>
          <h2 className="text-2xl font-bold text-slate-900">Pooled Quotas & White-Label Customization</h2>
          <p className="text-sm text-slate-600 max-w-2xl leading-relaxed">
            Manage your aggregated resource pools, overselling privileges, and custom white-label branding across your client base.
          </p>
        </div>
        <button
          onClick={loadData}
          disabled={loading}
          className="p-2.5 rounded-xl border border-slate-200 bg-white text-slate-600 hover:text-slate-900 shadow-sm transition-all self-start md:self-center"
          title="Refresh Quotas"
        >
          <RefreshCw className={`w-4 h-4 ${loading ? 'animate-spin text-purple-600' : ''}`} />
        </button>
      </div>

      {/* Quota Pooling Meters */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
        {/* Accounts Pool */}
        <div className="p-6 rounded-3xl bg-white border border-slate-200 shadow-sm space-y-4">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-3">
              <div className="p-3 rounded-2xl bg-blue-50 text-blue-600">
                <Users className="w-5 h-5" />
              </div>
              <div>
                <div className="text-xs font-bold uppercase tracking-wider text-slate-500">Account Pool</div>
                <div className="text-xl font-black text-slate-900">
                  {usage ? `${usage.used_accounts} / ${usage.max_accounts}` : '—'}
                </div>
              </div>
            </div>
            <span className={`px-2.5 py-1 rounded-full text-xs font-bold ${
              accountPercent >= 90
                ? 'bg-rose-100 text-rose-700'
                : accountPercent >= 75
                ? 'bg-amber-100 text-amber-700'
                : 'bg-blue-100 text-blue-700'
            }`}>
              {accountPercent}% Used
            </span>
          </div>
          <div className="w-full bg-slate-100 rounded-full h-2.5 overflow-hidden">
            <div
              className={`h-2.5 rounded-full transition-all duration-500 ${
                accountPercent >= 90 ? 'bg-rose-500' : accountPercent >= 75 ? 'bg-amber-500' : 'bg-blue-600'
              }`}
              style={{ width: `${accountPercent}%` }}
            />
          </div>
          <div className="text-[11px] text-slate-500 flex justify-between">
            <span>Remaining: {usage ? Math.max(0, usage.max_accounts - usage.used_accounts) : 0} accounts</span>
            <span>Limit: {usage?.max_accounts || 0}</span>
          </div>
        </div>

        {/* Disk Storage Pool */}
        <div className="p-6 rounded-3xl bg-white border border-slate-200 shadow-sm space-y-4">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-3">
              <div className="p-3 rounded-2xl bg-emerald-50 text-[#16A34A]">
                <HardDrive className="w-5 h-5" />
              </div>
              <div>
                <div className="text-xs font-bold uppercase tracking-wider text-slate-500">NVMe Disk Pool</div>
                <div className="text-xl font-black text-slate-900">
                  {usage
                    ? `${(usage.used_disk_mb / 1024).toFixed(1)} GB / ${(usage.allocated_disk_mb / 1024).toFixed(0)} GB`
                    : '—'}
                </div>
              </div>
            </div>
            <span className={`px-2.5 py-1 rounded-full text-xs font-bold ${
              diskPercent >= 90
                ? 'bg-rose-100 text-rose-700'
                : diskPercent >= 75
                ? 'bg-amber-100 text-amber-700'
                : 'bg-emerald-100 text-emerald-700'
            }`}>
              {diskPercent}% Used
            </span>
          </div>
          <div className="w-full bg-slate-100 rounded-full h-2.5 overflow-hidden">
            <div
              className={`h-2.5 rounded-full transition-all duration-500 ${
                diskPercent >= 90 ? 'bg-rose-500' : diskPercent >= 75 ? 'bg-amber-500' : 'bg-[#16A34A]'
              }`}
              style={{ width: `${diskPercent}%` }}
            />
          </div>
          <div className="text-[11px] text-slate-500 flex justify-between">
            <span>
              Free: {usage ? ((usage.allocated_disk_mb - usage.used_disk_mb) / 1024).toFixed(1) : 0} GB
            </span>
            <span>Allocated: {usage ? (usage.allocated_disk_mb / 1024).toFixed(0) : 0} GB</span>
          </div>
        </div>

        {/* Bandwidth Pool */}
        <div className="p-6 rounded-3xl bg-white border border-slate-200 shadow-sm space-y-4">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-3">
              <div className="p-3 rounded-2xl bg-purple-50 text-purple-600">
                <Network className="w-5 h-5" />
              </div>
              <div>
                <div className="text-xs font-bold uppercase tracking-wider text-slate-500">Bandwidth Pool</div>
                <div className="text-xl font-black text-slate-900">
                  {usage
                    ? `${(usage.used_bandwidth_mb / 1024).toFixed(1)} GB / ${(usage.allocated_bandwidth_mb / 1024).toFixed(0)} GB`
                    : '—'}
                </div>
              </div>
            </div>
            <span className={`px-2.5 py-1 rounded-full text-xs font-bold ${
              bwPercent >= 90
                ? 'bg-rose-100 text-rose-700'
                : bwPercent >= 75
                ? 'bg-amber-100 text-amber-700'
                : 'bg-purple-100 text-purple-700'
            }`}>
              {bwPercent}% Used
            </span>
          </div>
          <div className="w-full bg-slate-100 rounded-full h-2.5 overflow-hidden">
            <div
              className={`h-2.5 rounded-full transition-all duration-500 ${
                bwPercent >= 90 ? 'bg-rose-500' : bwPercent >= 75 ? 'bg-amber-500' : 'bg-purple-600'
              }`}
              style={{ width: `${bwPercent}%` }}
            />
          </div>
          <div className="text-[11px] text-slate-500 flex justify-between">
            <span>
              Remaining: {usage ? ((usage.allocated_bandwidth_mb - usage.used_bandwidth_mb) / 1024).toFixed(1) : 0} GB
            </span>
            <span>Policy: {usage?.allow_overselling ? 'Overselling Allowed' : 'Strict Quota Lock'}</span>
          </div>
        </div>
      </div>

      {/* White-Label Branding Settings & Live Preview */}
      <div className="grid grid-cols-1 lg:grid-cols-12 gap-8">
        {/* Settings Form */}
        <div className="lg:col-span-7 rounded-3xl bg-white p-7 border border-slate-200 shadow-sm space-y-6">
          <div className="space-y-1">
            <h3 className="text-lg font-bold text-slate-900 flex items-center gap-2">
              <Palette className="w-5 h-5 text-purple-600" />
              <span>White-Label Brand Customization</span>
            </h3>
            <p className="text-xs text-slate-500">
              Personalize your control panel name, logo, custom nameservers, and system notification sender.
            </p>
          </div>

          <form onSubmit={handleSaveBranding} className="space-y-5">
            <div>
              <label className="block text-xs font-bold text-slate-700 uppercase tracking-wider mb-1.5">
                Brand / Platform Name
              </label>
              <input
                type="text"
                value={brandName}
                onChange={e => setBrandName(e.target.value)}
                placeholder="e.g. Acme Cloud Hosting"
                className="w-full px-4 py-2.5 rounded-xl border border-slate-200 text-xs text-slate-900 focus:outline-none focus:ring-2 focus:ring-purple-500"
              />
              <span className="text-[11px] text-slate-400 mt-1 block">
                Replaces &quot;Hostvra&quot; across dashboard navigation, title tags, and client login screens.
              </span>
            </div>

            <div>
              <label className="block text-xs font-bold text-slate-700 uppercase tracking-wider mb-1.5">
                Brand Logo URL (Optional)
              </label>
              <input
                type="url"
                value={brandLogoUrl}
                onChange={e => setBrandLogoUrl(e.target.value)}
                placeholder="https://example.com/logo.svg"
                className="w-full px-4 py-2.5 rounded-xl border border-slate-200 text-xs text-slate-900 focus:outline-none focus:ring-2 focus:ring-purple-500"
              />
            </div>

            <div className="space-y-3">
              <label className="block text-xs font-bold text-slate-700 uppercase tracking-wider">
                Custom Nameservers
              </label>
              <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
                <div>
                  <span className="text-[11px] font-semibold text-slate-500 mb-1 block">Primary (NS1)</span>
                  <input
                    type="text"
                    value={ns1}
                    onChange={e => setNs1(e.target.value)}
                    placeholder="ns1.yourdomain.com"
                    className="w-full px-3.5 py-2 rounded-xl border border-slate-200 text-xs text-slate-900 focus:outline-none focus:ring-2 focus:ring-purple-500 font-mono"
                  />
                </div>
                <div>
                  <span className="text-[11px] font-semibold text-slate-500 mb-1 block">Secondary (NS2)</span>
                  <input
                    type="text"
                    value={ns2}
                    onChange={e => setNs2(e.target.value)}
                    placeholder="ns2.yourdomain.com"
                    className="w-full px-3.5 py-2 rounded-xl border border-slate-200 text-xs text-slate-900 focus:outline-none focus:ring-2 focus:ring-purple-500 font-mono"
                  />
                </div>
                <div>
                  <span className="text-[11px] font-semibold text-slate-500 mb-1 block">Tertiary (NS3 - Optional)</span>
                  <input
                    type="text"
                    value={ns3}
                    onChange={e => setNs3(e.target.value)}
                    placeholder="ns3.yourdomain.com"
                    className="w-full px-3.5 py-2 rounded-xl border border-slate-200 text-xs text-slate-900 focus:outline-none focus:ring-2 focus:ring-purple-500 font-mono"
                  />
                </div>
                <div>
                  <span className="text-[11px] font-semibold text-slate-500 mb-1 block">Quaternary (NS4 - Optional)</span>
                  <input
                    type="text"
                    value={ns4}
                    onChange={e => setNs4(e.target.value)}
                    placeholder="ns4.yourdomain.com"
                    className="w-full px-3.5 py-2 rounded-xl border border-slate-200 text-xs text-slate-900 focus:outline-none focus:ring-2 focus:ring-purple-500 font-mono"
                  />
                </div>
              </div>
            </div>

            <div>
              <label className="block text-xs font-bold text-slate-700 uppercase tracking-wider mb-1.5">
                Automated System Email Sender
              </label>
              <input
                type="email"
                value={emailSender}
                onChange={e => setEmailSender(e.target.value)}
                placeholder="support@yourdomain.com"
                className="w-full px-4 py-2.5 rounded-xl border border-slate-200 text-xs text-slate-900 focus:outline-none focus:ring-2 focus:ring-purple-500 font-mono"
              />
            </div>

            <div className="pt-3 flex justify-end">
              <button
                type="submit"
                disabled={savingBranding}
                className="px-6 py-2.5 rounded-xl bg-[#16A34A] hover:bg-emerald-700 text-white text-xs font-bold shadow-sm transition-all flex items-center gap-2"
              >
                {savingBranding ? <RefreshCw className="w-4 h-4 animate-spin" /> : <Save className="w-4 h-4" />}
                <span>Save White-Label Branding</span>
              </button>
            </div>
          </form>
        </div>

        {/* Live Client Experience Preview */}
        <div className="lg:col-span-5 rounded-3xl bg-white p-7 border border-slate-200 shadow-sm space-y-5">
          <div className="space-y-1">
            <h3 className="text-base font-bold text-slate-900 flex items-center gap-2">
              <Globe className="w-4 h-4 text-emerald-600" />
              <span>Live Client Panel Preview</span>
            </h3>
            <p className="text-xs text-slate-500">
              Interactive preview of what your customers see when logging in.
            </p>
          </div>

          <div className="rounded-2xl border border-slate-200 p-4 bg-slate-50/50 space-y-4">
            {/* Header simulation */}
            <div className="bg-white p-3.5 rounded-xl border border-slate-200 flex items-center justify-between shadow-xs">
              <div className="flex items-center gap-2.5">
                {brandLogoUrl ? (
                  // eslint-disable-next-line @next/next/no-img-element
                  <img src={brandLogoUrl} alt="Logo" className="w-6 h-6 object-contain rounded" />
                ) : (
                  <div className="w-6 h-6 rounded bg-[#16A34A] flex items-center justify-center text-white text-xs font-black">
                    {brandName ? brandName.charAt(0).toUpperCase() : 'H'}
                  </div>
                )}
                <span className="font-extrabold text-xs text-slate-900">{brandName || 'Hostvra Platform'}</span>
              </div>
              <div className="flex items-center gap-2">
                <span className="w-2 h-2 rounded-full bg-emerald-500" />
                <span className="text-[10px] text-slate-500">Client Panel</span>
              </div>
            </div>

            {/* Email notification card simulation */}
            <div className="bg-white p-3.5 rounded-xl border border-slate-200 space-y-2 text-xs">
              <div className="flex items-center gap-2 text-slate-500 text-[11px] pb-2 border-b border-slate-100">
                <Mail className="w-3.5 h-3.5 text-slate-400" />
                <span>From: {emailSender || 'noreply@hostvra.com'}</span>
              </div>
              <div className="font-bold text-slate-800">Welcome to Your New Hosting Account!</div>
              <p className="text-[11px] text-slate-600 leading-relaxed">
                Your hosting service with <strong>{brandName || 'Hostvra'}</strong> is active. Please point your domain nameservers to:
              </p>
              <div className="bg-slate-50 p-2.5 rounded-lg border border-slate-100 font-mono text-[11px] text-purple-700 space-y-1">
                <div>• {ns1 || 'ns1.hostvra.com'}</div>
                <div>• {ns2 || 'ns2.hostvra.com'}</div>
              </div>
            </div>

            {/* Status indicator */}
            <div className="text-[11px] text-emerald-800 bg-emerald-50 p-3 rounded-xl border border-emerald-200 flex items-center gap-2">
              <CheckCircle2 className="w-4 h-4 text-emerald-600 shrink-0" />
              <span>Full white-label isolation: Hostvra branding is masked for your sub-tenants.</span>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
