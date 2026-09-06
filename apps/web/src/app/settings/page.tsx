'use client';

import React, { useState, useEffect } from 'react';
import {
  Settings as SettingsIcon,
  Building2,
  Shield,
  BadgePercent,
  Key,
  CheckCircle2,
  Server,
  Globe,
} from 'lucide-react';
import { DashboardShell } from '@/components/DashboardShell';
import { apiFetch, User, Organization } from '@/lib/api';

export default function SettingsPage() {
  const [user, setUser] = useState<User | null>(null);
  const [org, setOrg] = useState<Organization | null>(null);

  useEffect(() => {
    async function load() {
      const res = await apiFetch<{ user: User; org: Organization }>('/api/v1/auth/me');
      if (res.success && res.data) {
        setUser(res.data.user);
        setOrg(res.data.org);
      }
    }
    load();
  }, []);

  return (
    <DashboardShell>
      <div className="space-y-8 max-w-4xl">
        <div>
          <h1 className="text-2xl font-bold tracking-tight text-white">Organization & System Settings</h1>
          <p className="text-sm text-slate-400 mt-1">
            Manage your Hostvra organization, team entitlements, and licensing tier.
          </p>
        </div>

        {/* Organization Card */}
        <div className="bg-surface-900 border border-surface-800 rounded-2xl p-6 shadow-xl space-y-6">
          <div className="flex items-center gap-3 border-b border-surface-800 pb-4">
            <div className="w-10 h-10 rounded-xl bg-indigo-500/10 border border-indigo-500/20 flex items-center justify-center text-indigo-400">
              <Building2 className="w-5 h-5" />
            </div>
            <div>
              <h2 className="text-base font-bold text-white">Organization Profile</h2>
              <p className="text-xs text-slate-400">Primary tenant details and resource quotas</p>
            </div>
          </div>

          <div className="grid grid-cols-1 sm:grid-cols-2 gap-4 text-sm">
            <div>
              <label className="block text-xs font-semibold uppercase tracking-wider text-slate-400 mb-1">
                Organization Name
              </label>
              <input
                type="text"
                disabled
                value={org ? org.name : 'Hostvra Cloud'}
                className="w-full px-3.5 py-2 rounded-xl bg-surface-950 border border-surface-700 text-slate-300 text-sm font-medium"
              />
            </div>
            <div>
              <label className="block text-xs font-semibold uppercase tracking-wider text-slate-400 mb-1">
                Tenant Slug
              </label>
              <input
                type="text"
                disabled
                value={org ? org.slug : 'hostvra-cloud-default'}
                className="w-full px-3.5 py-2 rounded-xl bg-surface-950 border border-surface-700 text-slate-400 font-mono text-xs"
              />
            </div>
          </div>

          <div className="pt-2 border-t border-surface-800 flex items-center justify-between text-xs text-slate-400">
            <span>Fleet Allocation: {org ? org.max_servers : 1} Server Quota</span>
            <span>Virtual Host Allocation: {org ? org.max_websites : 5} Websites</span>
          </div>
        </div>

        {/* Licensing & Feature Flags */}
        <div className="bg-surface-900 border border-surface-800 rounded-2xl p-6 shadow-xl space-y-5">
          <div className="flex items-center justify-between border-b border-surface-800 pb-4">
            <div className="flex items-center gap-3">
              <div className="w-10 h-10 rounded-xl bg-emerald-500/10 border border-emerald-500/20 flex items-center justify-center text-emerald-400">
                <BadgePercent className="w-5 h-5" />
              </div>
              <div>
                <h2 className="text-base font-bold text-white">Licensing & Entitlements</h2>
                <p className="text-xs text-slate-400">Current tier and active capability flags</p>
              </div>
            </div>
            <span className="px-3 py-1 rounded-full text-xs font-bold uppercase tracking-wider bg-indigo-500/10 text-indigo-400 border border-indigo-500/20">
              Community Free Edition
            </span>
          </div>

          <div className="grid grid-cols-1 sm:grid-cols-2 gap-3 text-xs">
            <div className="p-3.5 rounded-xl bg-surface-950 border border-surface-800 flex items-center gap-3">
              <CheckCircle2 className="w-4 h-4 text-emerald-400 flex-shrink-0" />
              <div>
                <p className="font-semibold text-slate-200">Native Go Agent Daemon</p>
                <p className="text-slate-400 text-[11px]">Real-time systemd telemetry and typed operations</p>
              </div>
            </div>

            <div className="p-3.5 rounded-xl bg-surface-950 border border-surface-800 flex items-center gap-3">
              <CheckCircle2 className="w-4 h-4 text-emerald-400 flex-shrink-0" />
              <div>
                <p className="font-semibold text-slate-200">Granular Role-Based Access Control</p>
                <p className="text-slate-400 text-[11px]">Owner, Admin, Manager, Developer, Viewer</p>
              </div>
            </div>

            <div className="p-3.5 rounded-xl bg-surface-950 border border-surface-800 flex items-center gap-3">
              <CheckCircle2 className="w-4 h-4 text-emerald-400 flex-shrink-0" />
              <div>
                <p className="font-semibold text-slate-200">Immutable Audit Logging</p>
                <p className="text-slate-400 text-[11px]">Security events, IP tracing, and action auditing</p>
              </div>
            </div>

            <div className="p-3.5 rounded-xl bg-surface-950 border border-surface-800 flex items-center gap-3">
              <CheckCircle2 className="w-4 h-4 text-emerald-400 flex-shrink-0" />
              <div>
                <p className="font-semibold text-slate-200">One-Command Agent Enrollment</p>
                <p className="text-slate-400 text-[11px]">Single-use signed tokens with automatic rollback</p>
              </div>
            </div>
          </div>
        </div>
      </div>
    </DashboardShell>
  );
}
