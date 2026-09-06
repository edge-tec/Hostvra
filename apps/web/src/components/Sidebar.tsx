'use client';

import React from 'react';
import Link from 'next/link';
import { usePathname } from 'next/navigation';
import {
  LayoutDashboard,
  Server,
  Globe,
  Database,
  Container,
  Terminal,
  FolderTree,
  Clock,
  ShieldCheck,
  ShieldAlert,
  HardDriveDownload,
  Activity,
  ScrollText,
  Users,
  KeyRound,
  BadgePercent,
  Settings,
  Flame,
} from 'lucide-react';

interface NavItem {
  label: string;
  href: string;
  icon: React.ElementType;
  badge?: string;
}

const mainNavItems: NavItem[] = [
  { label: 'Dashboard', href: '/dashboard', icon: LayoutDashboard },
  { label: 'Servers', href: '/servers', icon: Server },
  { label: 'Websites', href: '/websites', icon: Globe },
  { label: 'Databases', href: '/databases', icon: Database },
  { label: 'Docker', href: '/docker', icon: Container },
  { label: 'File Manager', href: '/files', icon: FolderTree },
  { label: 'Cron Jobs', href: '/cron', icon: Clock },
  { label: 'SSL Certificates', href: '/ssl', icon: ShieldCheck },
  { label: 'Firewall', href: '/firewall', icon: ShieldAlert },
  { label: 'Backups', href: '/backups', icon: HardDriveDownload },
  { label: 'Monitoring', href: '/monitoring', icon: Activity },
  { label: 'Audit Logs', href: '/audit-logs', icon: ScrollText },
];

const secondaryNavItems: NavItem[] = [
  { label: 'Team', href: '/team', icon: Users },
  { label: 'API Keys', href: '/api-keys', icon: KeyRound },
  { label: 'Licensing', href: '/license', icon: BadgePercent, badge: 'Free' },
  { label: 'Settings', href: '/settings', icon: Settings },
];

export function Sidebar() {
  const pathname = usePathname();

  return (
    <aside className="w-64 bg-surface-900 border-r border-surface-800 flex flex-col h-screen select-none sticky top-0">
      {/* Brand Header */}
      <div className="h-16 flex items-center px-6 border-b border-surface-800 gap-3">
        <div className="w-9 h-9 rounded-xl bg-gradient-to-br from-indigo-500 to-purple-600 flex items-center justify-center shadow-lg shadow-indigo-500/20">
          <Flame className="w-5 h-5 text-white" />
        </div>
        <div>
          <span className="font-bold text-lg tracking-tight text-white flex items-center gap-1.5">
            Hostvra <span className="text-xs px-1.5 py-0.5 rounded bg-indigo-500/20 text-indigo-400 font-mono font-medium">v1.0</span>
          </span>
          <p className="text-[11px] text-slate-400 font-medium">Server Fleet Control</p>
        </div>
      </div>

      {/* Navigation Scroll */}
      <div className="flex-1 overflow-y-auto px-3 py-4 space-y-6">
        <div>
          <div className="px-3 mb-2 text-[11px] font-semibold text-slate-400 tracking-wider uppercase">
            Management
          </div>
          <nav className="space-y-0.5">
            {mainNavItems.map((item) => {
              const Icon = item.icon;
              const isActive = pathname === item.href || (item.href !== '/dashboard' && pathname?.startsWith(item.href));
              return (
                <Link
                  key={item.href}
                  href={item.href}
                  className={`flex items-center justify-between px-3 py-2 rounded-lg text-sm font-medium transition-all ${
                    isActive
                      ? 'bg-indigo-600/15 text-indigo-400 border border-indigo-500/20'
                      : 'text-slate-400 hover:text-slate-200 hover:bg-surface-800/60'
                  }`}
                >
                  <div className="flex items-center gap-3">
                    <Icon className={`w-4 h-4 ${isActive ? 'text-indigo-400' : 'text-slate-400'}`} />
                    <span>{item.label}</span>
                  </div>
                  {item.badge && (
                    <span className="text-[10px] font-semibold px-1.5 py-0.2 rounded bg-indigo-500/20 text-indigo-300">
                      {item.badge}
                    </span>
                  )}
                </Link>
              );
            })}
          </nav>
        </div>

        <div>
          <div className="px-3 mb-2 text-[11px] font-semibold text-slate-400 tracking-wider uppercase">
            System & Billing
          </div>
          <nav className="space-y-0.5">
            {secondaryNavItems.map((item) => {
              const Icon = item.icon;
              const isActive = pathname === item.href;
              return (
                <Link
                  key={item.href}
                  href={item.href}
                  className={`flex items-center justify-between px-3 py-2 rounded-lg text-sm font-medium transition-all ${
                    isActive
                      ? 'bg-indigo-600/15 text-indigo-400 border border-indigo-500/20'
                      : 'text-slate-400 hover:text-slate-200 hover:bg-surface-800/60'
                  }`}
                >
                  <div className="flex items-center gap-3">
                    <Icon className={`w-4 h-4 ${isActive ? 'text-indigo-400' : 'text-slate-400'}`} />
                    <span>{item.label}</span>
                  </div>
                  {item.badge && (
                    <span className="text-[10px] font-semibold px-2 py-0.5 rounded bg-emerald-500/10 text-emerald-400 border border-emerald-500/20">
                      {item.badge}
                    </span>
                  )}
                </Link>
              );
            })}
          </nav>
        </div>
      </div>

      {/* System Status Footer */}
      <div className="p-3 border-t border-surface-800 bg-surface-950/40">
        <div className="flex items-center justify-between text-xs text-slate-400 px-2 py-1">
          <span className="flex items-center gap-2">
            <span className="w-2 h-2 rounded-full bg-emerald-500 animate-pulse" />
            Hostvra Core API
          </span>
          <span className="text-[11px] font-mono text-emerald-400">ONLINE</span>
        </div>
      </div>
    </aside>
  );
}
