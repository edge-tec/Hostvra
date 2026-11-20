'use client';

import Link from 'next/link';
import { DashboardShell } from '@/components/DashboardShell';
import {
  Users,
  Globe,
  Coins,
  Tag,
  Server,
  Shield,
  CreditCard,
  Settings,
  ArrowRight,
  Activity,
} from 'lucide-react';

const ADMIN_MODULES = [
  {
    title: 'Commercial & Affiliates',
    description: 'Manage promo coupons, affiliate partners, commission withdrawals, customer credit wallets, and dunning.',
    href: '/admin/commercial',
    icon: Tag,
    color: 'from-emerald-500/20 to-teal-500/10 border-emerald-500/30 text-emerald-400',
  },
  {
    title: 'Crypto & HD Treasury',
    description: 'Configure multi-chain Trust Wallet treasuries, HD wallet seeds, deposit addresses, and automated sweeps.',
    href: '/admin/crypto',
    icon: Coins,
    color: 'from-amber-500/20 to-orange-500/10 border-amber-500/30 text-amber-400',
  },
  {
    title: 'Users & Tenancy',
    description: 'Manage customer accounts, roles, RBAC permissions, plan overrides, and secure admin impersonation.',
    href: '/admin/users',
    icon: Users,
    color: 'from-sky-500/20 to-blue-500/10 border-sky-500/30 text-sky-400',
  },
  {
    title: 'Domain Reseller & TLDs',
    description: 'Configure TLD pricing, ResellerClub registrar credentials, domain orders, transfers, and renewals.',
    href: '/admin/domains',
    icon: Globe,
    color: 'from-violet-500/20 to-purple-500/10 border-violet-500/30 text-violet-400',
  },
  {
    title: 'Billing & Plans',
    description: 'Manage hosting plans, payment gateways (Stripe, bKash, Nagad, Crypto), invoices, and free trials.',
    href: '/billing',
    icon: CreditCard,
    color: 'from-indigo-500/20 to-blue-500/10 border-indigo-500/30 text-indigo-400',
  },
  {
    title: 'Servers & Cloud Nodes',
    description: 'Monitor connected Linux agent nodes, CPU/RAM/disk telemetry, maintenance mode, and migrations.',
    href: '/servers',
    icon: Server,
    color: 'from-cyan-500/20 to-sky-500/10 border-cyan-500/30 text-cyan-400',
  },
  {
    title: 'Security & WAF',
    description: 'Review security scans, malware quarantine, firewall rules, audit logs, and 2FA enforcement.',
    href: '/security',
    icon: Shield,
    color: 'from-rose-500/20 to-red-500/10 border-rose-500/30 text-rose-400',
  },
  {
    title: 'Platform Settings & Updates',
    description: 'Configure SMTP transactional email, white-label branding, system status, and one-click updates.',
    href: '/settings',
    icon: Settings,
    color: 'from-slate-500/20 to-zinc-500/10 border-slate-500/30 text-slate-300',
  },
];

export default function AdminOverviewPage() {
  return (
    <DashboardShell>
      <div className="space-y-6">
        <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
          <div>
            <h1 className="text-2xl sm:text-3xl font-bold text-white tracking-tight">
              Admin Control Center
            </h1>
            <p className="text-sm text-slate-400 mt-1">
              Centralized management for Hostvra commercial billing, crypto treasury, users, domains, and infrastructure.
            </p>
          </div>
          <Link
            href="/status"
            className="inline-flex items-center justify-center gap-2 px-4 py-2 rounded-xl bg-emerald-500/10 border border-emerald-500/30 text-emerald-400 text-sm font-medium hover:bg-emerald-500/20 transition-colors"
          >
            <Activity className="w-4 h-4" />
            Live System Status
          </Link>
        </div>

        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
          {ADMIN_MODULES.map((mod) => {
            const Icon = mod.icon;
            return (
              <Link
                key={mod.href}
                href={mod.href}
                className="group flex flex-col justify-between rounded-2xl bg-slate-900/70 border border-slate-800 hover:border-slate-700 p-5 transition-all hover:-translate-y-0.5"
              >
                <div>
                  <div
                    className={`w-11 h-11 rounded-xl bg-gradient-to-br border flex items-center justify-center mb-4 ${mod.color}`}
                  >
                    <Icon className="w-5 h-5" />
                  </div>
                  <h2 className="text-base font-semibold text-white group-hover:text-emerald-400 transition-colors">
                    {mod.title}
                  </h2>
                  <p className="text-xs text-slate-400 mt-1.5 leading-relaxed">
                    {mod.description}
                  </p>
                </div>
                <div className="mt-4 pt-3 border-t border-slate-800/80 flex items-center justify-between text-xs font-medium text-slate-300 group-hover:text-emerald-400">
                  <span>Open module</span>
                  <ArrowRight className="w-4 h-4 transition-transform group-hover:translate-x-1" />
                </div>
              </Link>
            );
          })}
        </div>
      </div>
    </DashboardShell>
  );
}
