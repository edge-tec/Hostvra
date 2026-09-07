'use client';

import React, { useState, useEffect } from 'react';
import Link from 'next/link';
import { usePathname, useRouter } from 'next/navigation';
import {
  LayoutDashboard,
  Server,
  Globe,
  Boxes,
  Layers,
  Code2,
  Mail,
  Inbox,
  Database,
  Container,
  FolderTree,
  FolderSync,
  Terminal,
  Clock,
  ShieldCheck,
  Network,
  Shield,
  ShieldAlert,
  DownloadCloud,
  Bell,
  ScrollText,
  RefreshCw,
  Users,
  Key,
  Award,
  Settings,
  LogOut,
  ChevronLeft,
  ChevronRight,
  X,
  Flame,
  CreditCard,
} from 'lucide-react';
import { clearStoredAuth } from '@/lib/api';

interface NavItem {
  label: string;
  href: string;
  icon: React.ElementType;
  badge?: string;
  badgeColor?: 'emerald' | 'blue' | 'purple' | 'amber';
}

interface NavGroup {
  title: string;
  items: NavItem[];
}

const navGroups: NavGroup[] = [
  {
    title: 'MANAGEMENT',
    items: [
      { label: 'Dashboard', href: '/dashboard', icon: LayoutDashboard },
      { label: 'Servers', href: '/servers', icon: Server },
      { label: 'Websites', href: '/websites', icon: Globe },
      { label: 'Client Accounts', href: '/accounts', icon: Users, badge: 'WHM', badgeColor: 'purple' },
      { label: 'App Store', href: '/app-store', icon: Boxes, badge: '1-Click', badgeColor: 'blue' },
      { label: 'Web Servers', href: '/webservers', icon: Layers, badge: 'Multi', badgeColor: 'purple' },
      { label: 'PHP Management', href: '/php', icon: Code2 },
      { label: 'Email Hosting', href: '/email', icon: Mail },
      { label: 'Webmail', href: '/webmail', icon: Inbox, badge: 'Web', badgeColor: 'emerald' },
      { label: 'Databases', href: '/databases', icon: Database },
      { label: 'Docker', href: '/docker', icon: Container },
      { label: 'File Manager', href: '/files', icon: FolderTree },
      { label: 'FTP Accounts', href: '/ftp', icon: FolderSync },
      { label: 'Terminal', href: '/terminal', icon: Terminal, badge: 'CLI', badgeColor: 'blue' },
      { label: 'Cron Jobs', href: '/cron', icon: Clock },
      { label: 'SSL Certificates', href: '/ssl', icon: ShieldCheck },
      { label: 'Domains & DNS', href: '/domains', icon: Network },
      { label: 'Firewall', href: '/firewall', icon: Shield },
      { label: 'WAF / ModSecurity', href: '/waf', icon: ShieldAlert, badge: 'OWASP', badgeColor: 'purple' },
      { label: 'Backups', href: '/backups', icon: DownloadCloud },
      { label: 'Alerts & Incidents', href: '/alerts', icon: Bell },
      { label: 'Audit Logs', href: '/audit-logs', icon: ScrollText },
    ],
  },
  {
    title: 'SYSTEM & BILLING',
    items: [
      { label: 'Billing & Plans', href: '/billing', icon: CreditCard, badge: 'Hosting', badgeColor: 'amber' },
      { label: 'System Updates', href: '/settings/updates', icon: RefreshCw, badge: 'Live', badgeColor: 'emerald' },
      { label: 'Team', href: '/team', icon: Users },
      { label: 'API Keys', href: '/api-keys', icon: Key },
      { label: 'Licensing', href: '/license', icon: Award, badge: 'Free', badgeColor: 'emerald' },
      { label: 'Settings', href: '/settings', icon: Settings },
    ],
  },
];

export function Sidebar() {
  const pathname = usePathname();
  const router = useRouter();
  const [collapsed, setCollapsed] = useState(false);
  const [mobileOpen, setMobileOpen] = useState(false);

  useEffect(() => {
    const saved = localStorage.getItem('hostvra_sidebar_collapsed');
    if (saved !== null) {
      setCollapsed(saved === 'true');
    }

    const handleToggleMobile = () => setMobileOpen((prev) => !prev);
    const handleCloseMobile = () => setMobileOpen(false);

    window.addEventListener('hostvra_toggle_mobile_sidebar', handleToggleMobile);
    window.addEventListener('hostvra_close_mobile_sidebar', handleCloseMobile);

    return () => {
      window.removeEventListener('hostvra_toggle_mobile_sidebar', handleToggleMobile);
      window.removeEventListener('hostvra_close_mobile_sidebar', handleCloseMobile);
    };
  }, []);

  // Close mobile drawer on route navigation
  useEffect(() => {
    setMobileOpen(false);
  }, [pathname]);

  const toggleCollapse = () => {
    const next = !collapsed;
    setCollapsed(next);
    localStorage.setItem('hostvra_sidebar_collapsed', String(next));
    window.dispatchEvent(new Event('resize'));
  };

  const handleLogout = () => {
    clearStoredAuth();
    router.push('/login');
  };

  const getBadgeClasses = (color?: string, isActive?: boolean) => {
    if (isActive) return 'bg-white/20 text-white';
    switch (color) {
      case 'blue':
        return 'bg-blue-100 dark:bg-blue-950/60 text-blue-600 dark:text-blue-400 border border-blue-200/50 dark:border-blue-900/50';
      case 'purple':
        return 'bg-purple-100 dark:bg-purple-950/60 text-purple-600 dark:text-purple-400 border border-purple-200/50 dark:border-purple-900/50';
      case 'amber':
        return 'bg-amber-100 dark:bg-amber-950/60 text-amber-600 dark:text-amber-400 border border-amber-200/50 dark:border-amber-900/50';
      case 'emerald':
      default:
        return 'bg-emerald-100 dark:bg-emerald-950/60 text-emerald-600 dark:text-emerald-400 border border-emerald-200/50 dark:border-emerald-900/50';
    }
  };

  const renderNavList = (isMobileView = false) => (
    <div className="space-y-4">
      {navGroups.map((group) => (
        <div key={group.title} className="space-y-0.5">
          {/* Group Header */}
          {(isMobileView || !collapsed) && (
            <div className="px-3 pt-2 pb-1 text-[10px] font-bold tracking-wider text-slate-400 dark:text-slate-500 uppercase">
              {group.title}
            </div>
          )}
          {(!isMobileView && collapsed) && (
            <div className="h-px bg-slate-200 dark:bg-surface-800 my-2 mx-2" />
          )}

          {/* Group Items */}
          <nav className="space-y-0.5">
            {group.items.map((item) => {
              const Icon = item.icon;
              // Strict active checking: Dashboard is ONLY active on exact /dashboard
              const isActive =
                item.href === '/dashboard'
                  ? pathname === '/dashboard'
                  : pathname === item.href || (item.href !== '/' && pathname?.startsWith(item.href));

              return (
                <Link
                  key={item.label}
                  href={item.href}
                  onClick={() => isMobileView && setMobileOpen(false)}
                  title={!isMobileView && collapsed ? item.label : undefined}
                  className={`flex items-center ${
                    !isMobileView && collapsed ? 'justify-center px-0 py-2.5' : 'justify-between px-3 py-2'
                  } rounded-lg text-xs font-medium transition-all group ${
                    isActive
                      ? 'bg-[#20a53a] text-white font-semibold shadow-sm'
                      : 'text-slate-600 dark:text-slate-300 hover:text-slate-900 dark:hover:text-white hover:bg-slate-100 dark:hover:bg-surface-800'
                  }`}
                >
                  <div className={`flex items-center ${!isMobileView && collapsed ? '' : 'gap-2.5'} truncate min-w-0`}>
                    <Icon
                      className={`w-4 h-4 flex-shrink-0 transition-colors ${
                        isActive ? 'text-white' : 'text-slate-500 dark:text-slate-400 group-hover:text-slate-800 dark:group-hover:text-white'
                      }`}
                    />
                    {(isMobileView || !collapsed) && <span className="truncate">{item.label}</span>}
                  </div>

                  {(isMobileView || !collapsed) && item.badge && (
                    <span
                      className={`text-[9px] font-bold px-1.5 py-0.5 rounded leading-none ${getBadgeClasses(
                        item.badgeColor,
                        isActive
                      )}`}
                    >
                      {item.badge}
                    </span>
                  )}
                </Link>
              );
            })}
          </nav>
        </div>
      ))}

      {/* Log Out */}
      <div className="pt-2 border-t border-slate-200 dark:border-surface-800">
        <button
          onClick={handleLogout}
          title={!isMobileView && collapsed ? 'Log out' : undefined}
          className={`w-full flex items-center ${
            !isMobileView && collapsed ? 'justify-center px-0 py-2.5' : 'px-3 py-2 gap-2.5'
          } rounded-lg text-xs font-medium text-slate-600 dark:text-slate-300 hover:text-rose-600 dark:hover:text-rose-400 hover:bg-rose-50 dark:hover:bg-rose-950/20 transition-all text-left`}
        >
          <LogOut className="w-4 h-4 flex-shrink-0 text-slate-500 dark:text-slate-400" />
          {(isMobileView || !collapsed) && <span>Log out</span>}
        </button>
      </div>
    </div>
  );

  return (
    <>
      {/* 1. DESKTOP SIDEBAR (Visible on lg: screens and up) */}
      <aside
        className={`hidden lg:flex ${
          collapsed ? 'w-16' : 'w-60'
        } bg-white dark:bg-[#121824] border-r border-slate-200 dark:border-surface-800 flex-col h-screen select-none sticky top-0 transition-all duration-200 ease-in-out z-30 flex-shrink-0 shadow-sm`}
      >
        {/* Brand Header */}
        <div className="h-16 flex items-center px-3.5 border-b border-slate-200 dark:border-surface-800 justify-between flex-shrink-0 overflow-hidden">
          <div className="flex items-center gap-2.5 min-w-0">
            <div className="w-9 h-9 rounded-xl bg-gradient-to-tr from-indigo-600 to-violet-500 flex items-center justify-center text-white flex-shrink-0 shadow-md shadow-indigo-500/20">
              <Flame className="w-5 h-5 text-amber-300 fill-amber-300" />
            </div>
            {!collapsed && (
              <div className="min-w-0 truncate">
                <div className="flex items-center gap-1.5 leading-tight">
                  <span className="text-sm font-bold text-slate-900 dark:text-white tracking-tight">Hostvra</span>
                  <span className="text-[10px] font-bold px-1.5 py-0.5 rounded bg-indigo-100 dark:bg-indigo-950/80 text-indigo-600 dark:text-indigo-400 border border-indigo-200/50 dark:border-indigo-800/50">
                    v1.0
                  </span>
                </div>
                <div className="text-[10px] text-slate-500 dark:text-slate-400 font-medium truncate mt-0.5">
                  Server Fleet Control
                </div>
              </div>
            )}
          </div>
        </div>

        {/* Navigation Scroll */}
        <div className="flex-1 overflow-y-auto px-2.5 py-3 custom-scrollbar">
          {renderNavList(false)}
        </div>

        {/* Bottom Core API Status & Collapse Footer */}
        <div className="p-2.5 border-t border-slate-200 dark:border-surface-800 bg-slate-50 dark:bg-[#0e1420] flex-shrink-0 space-y-2">
          {!collapsed && (
            <div className="flex items-center justify-between px-2 py-1.5 rounded-md bg-white dark:bg-[#121824] border border-slate-200/80 dark:border-surface-800 text-[11px]">
              <div className="flex items-center gap-1.5 min-w-0">
                <span className="w-2 h-2 rounded-full bg-emerald-500 animate-pulse flex-shrink-0" />
                <span className="text-slate-600 dark:text-slate-400 font-medium truncate">Hostvra Core API</span>
              </div>
              <span className="text-[9px] font-bold text-emerald-600 dark:text-emerald-400 bg-emerald-50 dark:bg-emerald-950/40 px-1.5 py-0.5 rounded uppercase tracking-wider">
                ONLINE
              </span>
            </div>
          )}

          <button
            onClick={toggleCollapse}
            className="w-full py-1.5 rounded-lg flex items-center justify-center text-slate-500 hover:text-slate-900 dark:hover:text-white hover:bg-slate-200/60 dark:hover:bg-surface-800 transition"
            title={collapsed ? 'Expand sidebar' : 'Collapse sidebar'}
          >
            {collapsed ? (
              <ChevronRight className="w-4 h-4" />
            ) : (
              <div className="flex items-center gap-1.5 text-[11px] font-medium text-slate-500">
                <ChevronLeft className="w-3.5 h-3.5" />
                <span>Collapse</span>
              </div>
            )}
          </button>
        </div>
      </aside>

      {/* 2. MOBILE OFF-CANVAS DRAWER (Visible on < lg screens when toggled) */}
      {mobileOpen && (
        <div className="fixed inset-0 z-50 lg:hidden flex">
          {/* Backdrop */}
          <div
            onClick={() => setMobileOpen(false)}
            className="fixed inset-0 bg-black/50 backdrop-blur-xs transition-opacity animate-fadeIn"
          />

          {/* Drawer Content */}
          <div className="relative w-64 bg-white dark:bg-[#121824] border-r border-slate-200 dark:border-surface-800 flex flex-col h-full shadow-2xl z-10 animate-fadeIn">
            <div className="h-16 flex items-center justify-between px-4 border-b border-slate-200 dark:border-surface-800 flex-shrink-0">
              <div className="flex items-center gap-2.5">
                <div className="w-9 h-9 rounded-xl bg-gradient-to-tr from-indigo-600 to-violet-500 flex items-center justify-center text-white flex-shrink-0 shadow-md shadow-indigo-500/20">
                  <Flame className="w-5 h-5 text-amber-300 fill-amber-300" />
                </div>
                <div>
                  <div className="flex items-center gap-1.5 leading-tight">
                    <span className="text-sm font-bold text-slate-900 dark:text-white tracking-tight">Hostvra</span>
                    <span className="text-[10px] font-bold px-1.5 py-0.5 rounded bg-indigo-100 dark:bg-indigo-950/80 text-indigo-600 dark:text-indigo-400">
                      v1.0
                    </span>
                  </div>
                  <div className="text-[10px] text-slate-500 dark:text-slate-400 font-medium">
                    Server Fleet Control
                  </div>
                </div>
              </div>

              <button
                onClick={() => setMobileOpen(false)}
                className="w-7 h-7 rounded-lg bg-slate-100 dark:bg-surface-800 text-slate-500 hover:text-slate-800 dark:hover:text-white flex items-center justify-center"
              >
                <X className="w-4 h-4" />
              </button>
            </div>

            <div className="flex-1 overflow-y-auto px-3 py-3 custom-scrollbar">
              {renderNavList(true)}
            </div>

            <div className="p-3 border-t border-slate-200 dark:border-surface-800 bg-slate-50 dark:bg-[#0e1420] flex-shrink-0">
              <div className="flex items-center justify-between px-2.5 py-1.5 rounded-md bg-white dark:bg-[#121824] border border-slate-200/80 dark:border-surface-800 text-[11px]">
                <div className="flex items-center gap-1.5 min-w-0">
                  <span className="w-2 h-2 rounded-full bg-emerald-500 animate-pulse flex-shrink-0" />
                  <span className="text-slate-600 dark:text-slate-400 font-medium truncate">Hostvra Core API</span>
                </div>
                <span className="text-[9px] font-bold text-emerald-600 dark:text-emerald-400 bg-emerald-50 dark:bg-emerald-950/40 px-1.5 py-0.5 rounded uppercase tracking-wider">
                  ONLINE
                </span>
              </div>
            </div>
          </div>
        </div>
      )}
    </>
  );
}
