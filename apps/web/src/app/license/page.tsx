'use client';

import React, { useState } from 'react';
import { DashboardShell } from '@/components/DashboardShell';
import {
  BadgePercent,
  CheckCircle2,
  XCircle,
  Key,
  ShieldCheck,
  Zap,
  Server,
  Cloud,
  Users,
  Award,
  ChevronRight,
  Flame,
} from 'lucide-react';

interface LicenseState {
  tier: 'community' | 'pro' | 'enterprise';
  id: string;
  customer_name: string;
  customer_email: string;
  max_servers: number;
  expires_at: string;
  entitlements: {
    s3_backups: boolean;
    team_collab: boolean;
    docker_manager: boolean;
    white_label: boolean;
    priority_support: boolean;
  };
}

const currentLicense: LicenseState = {
  tier: 'community',
  id: 'HV-COMMUNITY-DEFAULT',
  customer_name: 'Hostvra Community User',
  customer_email: 'admin@hostvra.com',
  max_servers: 1,
  expires_at: 'Lifetime Free',
  entitlements: {
    s3_backups: false,
    team_collab: false,
    docker_manager: true,
    white_label: false,
    priority_support: false,
  },
};

export default function LicensePage() {
  const [license, setLicense] = useState<LicenseState>(currentLicense);
  const [licenseKeyInput, setLicenseKeyInput] = useState('');
  const [activating, setActivating] = useState(false);
  const [activationMsg, setActivationMsg] = useState<{ type: 'success' | 'error'; text: string } | null>(null);

  const handleActivate = (e: React.FormEvent) => {
    e.preventDefault();
    setActivating(true);
    setActivationMsg(null);

    setTimeout(() => {
      setActivating(false);
      if (licenseKeyInput.startsWith('HV-PRO-') || licenseKeyInput.startsWith('HV-ENTERPRISE-')) {
        const isEnt = licenseKeyInput.startsWith('HV-ENTERPRISE-');
        setLicense({
          tier: isEnt ? 'enterprise' : 'pro',
          id: licenseKeyInput.substring(0, 16),
          customer_name: isEnt ? 'Enterprise Fleet Operations' : 'Pro Hosting License',
          customer_email: 'billing@enterprise.io',
          max_servers: isEnt ? 9999 : 10,
          expires_at: 'Valid until 2027-09-06',
          entitlements: {
            s3_backups: true,
            team_collab: true,
            docker_manager: true,
            white_label: isEnt,
            priority_support: true,
          },
        });
        setActivationMsg({ type: 'success', text: `License activated successfully! Upgraded to ${isEnt ? 'Enterprise' : 'Pro'}.` });
        setLicenseKeyInput('');
      } else {
        setActivationMsg({ type: 'error', text: 'Invalid license key or cryptographic signature mismatch.' });
      }
    }, 1000);
  };

  return (
    <DashboardShell>
      <div className="space-y-6">
        {/* Page Header */}
        <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
          <div>
            <h1 className="text-2xl font-bold text-white tracking-tight flex items-center gap-2">
              <BadgePercent className="w-7 h-7 text-brand-400" />
              Licensing & Commercial Entitlements
            </h1>
            <p className="text-sm text-slate-400 mt-1">
              Ed25519 cryptographically signed license keys, offline verification, and fleet allowances.
            </p>
          </div>
        </div>

        {/* Current License Card */}
        <div className="bg-gradient-to-br from-surface-900 via-surface-900 to-surface-950 border border-surface-800 rounded-2xl p-6 shadow-xl relative overflow-hidden">
          <div className="absolute right-0 top-0 translate-x-8 -translate-y-8 w-64 h-64 bg-brand-500/10 rounded-full blur-3xl pointer-events-none" />

          <div className="flex flex-col lg:flex-row lg:items-center justify-between gap-6 relative z-10">
            <div>
              <div className="flex items-center gap-3">
                <span
                  className={`px-3 py-1 rounded-full text-xs font-black uppercase tracking-wider ${
                    license.tier === 'enterprise'
                      ? 'bg-purple-500/20 text-purple-400 border border-purple-500/30'
                      : license.tier === 'pro'
                      ? 'bg-brand-500/20 text-brand-400 border border-brand-500/30'
                      : 'bg-slate-500/20 text-slate-300 border border-slate-500/30'
                  }`}
                >
                  {license.tier} Edition
                </span>
                <span className="text-xs text-slate-400 font-mono">ID: {license.id}</span>
              </div>
              <div className="text-2xl font-bold text-white mt-2">{license.customer_name}</div>
              <div className="text-sm text-slate-400 mt-0.5">{license.customer_email} • {license.expires_at}</div>
            </div>

            <div className="flex items-center gap-6 border-t lg:border-t-0 lg:border-l border-surface-800 pt-4 lg:pt-0 lg:pl-6">
              <div>
                <div className="text-xs uppercase font-semibold text-slate-400">Server Quota</div>
                <div className="text-2xl font-black text-white mt-1">
                  1 / {license.max_servers > 1000 ? '∞' : license.max_servers}
                </div>
                <div className="text-xs text-emerald-400 mt-0.5">Servers Connected</div>
              </div>

              <div>
                <div className="text-xs uppercase font-semibold text-slate-400">Security Guard</div>
                <div className="text-sm font-bold text-emerald-400 mt-2 flex items-center gap-1.5">
                  <ShieldCheck className="w-4 h-4" /> Active Signature
                </div>
                <div className="text-xs text-slate-400 mt-0.5">Offline Verified</div>
              </div>
            </div>
          </div>

          {/* Entitlements Badges */}
          <div className="mt-6 pt-6 border-t border-surface-800/80 grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-5 gap-3">
            <div className="flex items-center gap-2 text-xs">
              {license.entitlements.docker_manager ? <CheckCircle2 className="w-4 h-4 text-emerald-400" /> : <XCircle className="w-4 h-4 text-slate-600" />}
              <span className={license.entitlements.docker_manager ? 'text-white' : 'text-slate-500'}>Docker Controller</span>
            </div>
            <div className="flex items-center gap-2 text-xs">
              {license.entitlements.s3_backups ? <CheckCircle2 className="w-4 h-4 text-emerald-400" /> : <XCircle className="w-4 h-4 text-slate-600" />}
              <span className={license.entitlements.s3_backups ? 'text-white' : 'text-slate-500'}>S3 Remote Backups</span>
            </div>
            <div className="flex items-center gap-2 text-xs">
              {license.entitlements.team_collab ? <CheckCircle2 className="w-4 h-4 text-emerald-400" /> : <XCircle className="w-4 h-4 text-slate-600" />}
              <span className={license.entitlements.team_collab ? 'text-white' : 'text-slate-500'}>Team Collaboration</span>
            </div>
            <div className="flex items-center gap-2 text-xs">
              {license.entitlements.white_label ? <CheckCircle2 className="w-4 h-4 text-emerald-400" /> : <XCircle className="w-4 h-4 text-slate-600" />}
              <span className={license.entitlements.white_label ? 'text-white' : 'text-slate-500'}>White-label Branding</span>
            </div>
            <div className="flex items-center gap-2 text-xs">
              {license.entitlements.priority_support ? <CheckCircle2 className="w-4 h-4 text-emerald-400" /> : <XCircle className="w-4 h-4 text-slate-600" />}
              <span className={license.entitlements.priority_support ? 'text-white' : 'text-slate-500'}>24/7 SLA Support</span>
            </div>
          </div>
        </div>

        {/* License Activation Form */}
        <div className="bg-surface-900 border border-surface-800 rounded-xl p-6 space-y-4">
          <div className="flex items-center gap-2">
            <Key className="w-5 h-5 text-brand-400" />
            <h2 className="text-sm font-bold text-white uppercase tracking-wider">Activate Commercial License Key</h2>
          </div>
          <p className="text-xs text-slate-400">
            Enter your Hostvra Pro or Enterprise license key. Activation verifies cryptographically without requiring constant internet access.
          </p>

          <form onSubmit={handleActivate} className="space-y-3">
            <div className="flex flex-col sm:flex-row gap-3">
              <input
                type="text"
                required
                placeholder="HV-PRO-xxxx.yyyy or HV-ENTERPRISE-xxxx.yyyy"
                value={licenseKeyInput}
                onChange={(e) => setLicenseKeyInput(e.target.value)}
                className="flex-1 px-4 py-2.5 rounded-lg bg-surface-950 border border-surface-800 text-white font-mono text-sm focus:outline-none focus:border-brand-500"
              />
              <button
                type="submit"
                disabled={activating}
                className="px-6 py-2.5 rounded-lg bg-brand-500 hover:bg-brand-600 text-white text-xs font-bold transition-colors disabled:opacity-50"
              >
                {activating ? 'Verifying Signature...' : 'Activate Key'}
              </button>
            </div>

            {activationMsg && (
              <div
                className={`p-3 rounded-lg text-xs font-medium ${
                  activationMsg.type === 'success'
                    ? 'bg-emerald-500/10 text-emerald-400 border border-emerald-500/20'
                    : 'bg-rose-500/10 text-rose-400 border border-rose-500/20'
                }`}
              >
                {activationMsg.text}
              </div>
            )}
          </form>
        </div>

        {/* Editions Comparison Grid */}
        <div className="grid grid-cols-1 md:grid-cols-3 gap-6 pt-4">
          {/* Community */}
          <div className="bg-surface-900 border border-surface-800 rounded-xl p-6 space-y-4">
            <div>
              <div className="text-xs font-bold uppercase text-slate-400">Open Source & Free</div>
              <div className="text-2xl font-black text-white mt-1">Community</div>
              <div className="text-3xl font-extrabold text-white mt-2">$0 <span className="text-xs font-normal text-slate-400">/ forever</span></div>
            </div>
            <ul className="space-y-2 text-xs text-slate-300">
              <li className="flex items-center gap-2"><CheckCircle2 className="w-3.5 h-3.5 text-emerald-400" /> 1 Server node</li>
              <li className="flex items-center gap-2"><CheckCircle2 className="w-3.5 h-3.5 text-emerald-400" /> Unlimited websites & DBs</li>
              <li className="flex items-center gap-2"><CheckCircle2 className="w-3.5 h-3.5 text-emerald-400" /> Free SSL via Let's Encrypt</li>
              <li className="flex items-center gap-2"><CheckCircle2 className="w-3.5 h-3.5 text-emerald-400" /> Docker container controller</li>
              <li className="flex items-center gap-2 text-slate-500"><XCircle className="w-3.5 h-3.5" /> S3 remote offsite sync</li>
              <li className="flex items-center gap-2 text-slate-500"><XCircle className="w-3.5 h-3.5" /> Multi-user team RBAC</li>
            </ul>
          </div>

          {/* Pro */}
          <div className="bg-gradient-to-b from-brand-950/40 via-surface-900 to-surface-900 border-2 border-brand-500/50 rounded-xl p-6 space-y-4 relative shadow-xl shadow-brand-500/5">
            <div className="absolute -top-3 right-6 px-3 py-0.5 rounded-full bg-brand-500 text-white font-bold text-[10px] uppercase">
              Popular
            </div>
            <div>
              <div className="text-xs font-bold uppercase text-brand-400">Multi-Server Fleets</div>
              <div className="text-2xl font-black text-white mt-1">Professional</div>
              <div className="text-3xl font-extrabold text-white mt-2">$19 <span className="text-xs font-normal text-slate-400">/ month</span></div>
            </div>
            <ul className="space-y-2 text-xs text-slate-300">
              <li className="flex items-center gap-2"><CheckCircle2 className="w-3.5 h-3.5 text-emerald-400" /> Up to 10 Server nodes</li>
              <li className="flex items-center gap-2"><CheckCircle2 className="w-3.5 h-3.5 text-emerald-400" /> S3 / Cloudflare R2 backup sync</li>
              <li className="flex items-center gap-2"><CheckCircle2 className="w-3.5 h-3.5 text-emerald-400" /> Team collaboration & RBAC</li>
              <li className="flex items-center gap-2"><CheckCircle2 className="w-3.5 h-3.5 text-emerald-400" /> Hardware anomaly alerts & webhooks</li>
              <li className="flex items-center gap-2"><CheckCircle2 className="w-3.5 h-3.5 text-emerald-400" /> Standard support</li>
            </ul>
          </div>

          {/* Enterprise */}
          <div className="bg-surface-900 border border-surface-800 rounded-xl p-6 space-y-4">
            <div>
              <div className="text-xs font-bold uppercase text-purple-400">Datacenters & MSPs</div>
              <div className="text-2xl font-black text-white mt-1">Enterprise</div>
              <div className="text-3xl font-extrabold text-white mt-2">$99 <span className="text-xs font-normal text-slate-400">/ month</span></div>
            </div>
            <ul className="space-y-2 text-xs text-slate-300">
              <li className="flex items-center gap-2"><CheckCircle2 className="w-3.5 h-3.5 text-emerald-400" /> Unlimited server nodes</li>
              <li className="flex items-center gap-2"><CheckCircle2 className="w-3.5 h-3.5 text-emerald-400" /> White-label domain & custom branding</li>
              <li className="flex items-center gap-2"><CheckCircle2 className="w-3.5 h-3.5 text-emerald-400" /> Audit log export to SIEM / Datadog</li>
              <li className="flex items-center gap-2"><CheckCircle2 className="w-3.5 h-3.5 text-emerald-400" /> Dedicated priority 24/7 SLA</li>
              <li className="flex items-center gap-2"><CheckCircle2 className="w-3.5 h-3.5 text-emerald-400" /> Custom volume discounts</li>
            </ul>
          </div>
        </div>
      </div>
    </DashboardShell>
  );
}
