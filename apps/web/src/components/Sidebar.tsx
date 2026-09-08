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
  LifeBuoy,
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
    title: 'HOSTING & DOMAINS',
    items: [
      { label: 'Dashboard', href: '/dashboard', icon: LayoutDashboard },
      { label: 'Domain Manager', href: '/domains', icon: Globe },
      { label: 'DNS Zones', href: '/dns', icon: Globe },
      { label: 'Websites', href: '/websites', icon: Server },
      { label: 'Directory (Files)', href: '/files', icon: FolderTree },
      { label: 'FTP Accounts', href: '/ftp', icon: FolderSync },
      { label: 'SSL Certificates', href: '/ssl', icon: ShieldCheck },
      { label: 'PHP Versions', href: '/php', icon: Code2 },
      { label: 'Web Servers', href: '/webservers', icon: Layers, badge: 'Multi', badgeColor: 'purple' },
      { label: 'Databases', href: '/databases', icon: Database },
      { label: 'Email Hosting', href: '/email', icon: Mail },
      { label: 'Webmail', href: '/webmail', icon: Inbox, badge: 'Web', badgeColor: 'emerald' },
    ],
  },
  {
    title: 'SECURITY & ACCESS',
    items: [
      { label: 'Firewall & Access', href: '/firewall', icon: Shield },
      { label: 'WAF / ModSecurity', href: '/waf', icon: ShieldAlert, badge: 'OWASP', badgeColor: 'blue' },
      { label: 'Audit & Response Logs', href: '/audit-logs', icon: ScrollText },
      { label: 'Alerts & Incidents', href: '/alerts', icon: Bell },
    ],
  },
  {
    title: 'TOOLS & RUNTIMES',
    items: [
      { label: 'Docker Containers', href: '/docker', icon: Container },
      { label: 'Terminal', href: '/terminal', icon: Terminal, badge: 'CLI', badgeColor: 'blue' },
      { label: 'Cron Jobs', href: '/cron', icon: Clock },
      { label: 'Backups', href: '/backups', icon: DownloadCloud },
      { label: 'App Store', href: '/app-store', icon: Boxes, badge: '1-Click', badgeColor: 'blue' },
      { label: 'Client Accounts', href: '/accounts', icon: Users, badge: 'WHM', badgeColor: 'purple' },
    ],
  },
  {
    title: 'SYSTEM & BILLING',
    items: [
      { label: 'Billing & Plans', href: '/billing', icon: CreditCard, badge: 'Cloud', badgeColor: 'amber' },
      { label: 'Domain Reseller', href: '/admin/domains', icon: Globe, badge: 'Admin', badgeColor: 'purple' },
      { label: 'Support & Helpdesk', href: '/support', icon: LifeBuoy, badge: '24/7', badgeColor: 'emerald' },
      { label: 'System Updates', href: '/settings/updates', icon: RefreshCw, badge: 'Live', badgeColor: 'emerald' },
      { label: 'Team', href: '/team', icon: Users },
      { label: 'API Keys', href: '/api-keys', icon: Key },
      { label: 'Licensing', href: '/license', icon: Award, badge: 'Enterprise', badgeColor: 'purple' },
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
    if (isActive) return 'bg-white/30 text-white font-bold';
    switch (color) {
      case 'amber':
        return 'bg-amber-100 dark:bg-amber-500/15 text-amber-800 dark:text-amber-300 border border-amber-200 dark:border-amber-400/20';
      case 'purple':
        return 'bg-purple-100 dark:bg-purple-500/15 text-purple-800 dark:text-purple-300 border border-purple-200 dark:border-purple-400/20';
      case 'blue':
        return 'bg-blue-100 dark:bg-blue-500/15 text-blue-800 dark:text-blue-300 border border-blue-200 dark:border-blue-400/20';
      case 'emerald':
      default:
        return 'bg-emerald-100 dark:bg-emerald-500/15 text-emerald-800 dark:text-emerald-300 border border-emerald-200 dark:border-emerald-400/20';
    }
  };

  const renderNavList = (isMobileView = false) => (
    <div className="space-y-4">
      {navGroups.map((group) => (
        <div key={group.title} className="space-y-0.5">
          {/* Group Header */}
          {(isMobileView || !collapsed) && (
            <div className="px-3 pt-3 pb-1 text-[10px] font-bold tracking-wider text-slate-500 dark:text-slate-400 uppercase">
              {group.title}
            </div>
          )}
          {(!isMobileView && collapsed) && (
            <div className="h-px bg-slate-200 dark:bg-slate-800 my-2 mx-2" />
          )}

          {/* Group Items */}
          <nav className="space-y-0.5">
            {group.items.map((item) => {
              const Icon = item.icon;
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
                  } rounded-lg text-xs font-semibold transition-all group ${
                    isActive
                      ? 'bg-[#16A34A] text-white font-bold shadow-sm shadow-emerald-600/25'
                      : 'text-slate-700 dark:text-slate-300 hover:text-slate-950 dark:hover:text-white hover:bg-slate-100 dark:hover:bg-white/[0.08]'
                  }`}
                >
                  <div className={`flex items-center ${!isMobileView && collapsed ? '' : 'gap-2.5'} truncate min-w-0`}>
                    <Icon
                      className={`w-4 h-4 flex-shrink-0 transition-colors ${
                        isActive ? 'text-white' : 'text-slate-500 dark:text-slate-400 group-hover:text-slate-900 dark:group-hover:text-white'
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
      <div className="pt-2 border-t border-slate-200 dark:border-slate-800">
        <button
          onClick={handleLogout}
          title={!isMobileView && collapsed ? 'Log out' : undefined}
          className={`w-full flex items-center ${
            !isMobileView && collapsed ? 'justify-center px-0 py-2.5' : 'px-3 py-2 gap-2.5'
          } rounded-lg text-xs font-semibold text-slate-600 dark:text-slate-400 hover:text-rose-600 dark:hover:text-rose-400 hover:bg-rose-50 dark:hover:bg-rose-500/10 transition-all text-left`}
        >
          <LogOut className="w-4 h-4 flex-shrink-0 text-slate-500 dark:text-slate-400 group-hover:text-rose-600" />
          {(isMobileView || !collapsed) && <span>Log out</span>}
        </button>
      </div>
    </div>
  );

  return (
    <>
      {/* 1. DESKTOP SIDEBAR (Visible on lg: screens and up) */}
      <aside
        data-component="sidebar"
        className={`hidden lg:flex ${
          collapsed ? 'w-16' : 'w-60'
        } bg-white dark:bg-[#0F172A] border-r border-slate-200 dark:border-slate-800 flex-col h-screen select-none sticky top-0 transition-all duration-200 ease-in-out z-30 flex-shrink-0 shadow-xs`}
      >
        {/* Brand Header */}
        <div className="h-16 flex items-center px-3.5 border-b border-slate-200 dark:border-slate-800 justify-between flex-shrink-0 overflow-hidden bg-white dark:bg-[#0F172A]">
          <div className="flex items-center gap-2.5 min-w-0">
            <div className="w-9 h-9 rounded-xl bg-gradient-to-tr from-emerald-600 to-teal-600 flex items-center justify-center text-white flex-shrink-0 shadow-sm shadow-emerald-600/20">
              <Flame className="w-5 h-5 text-amber-300 fill-amber-300" />
            </div>
            {!collapsed && (
              <div className="min-w-0 truncate">
                <div className="flex items-center gap-1.5 leading-tight">
                  <span className="text-sm font-bold text-slate-900 dark:text-white tracking-tight">Hostvra</span>
                  <span className="text-[10px] font-bold px-1.5 py-0.5 rounded bg-emerald-100 dark:bg-emerald-500/20 text-emerald-800 dark:text-emerald-300 border border-emerald-200 dark:border-emerald-500/30">
                    Control
                  </span>
                </div>
                <div className="text-[10px] text-slate-500 dark:text-slate-400 font-medium truncate mt-0.5">
                  Hosting Control Panel
                </div>
              </div>
            )}
          </div>
        </div>

        {/* Navigation Scroll */}
        <div className="flex-1 overflow-y-auto px-2.5 py-3 custom-scrollbar bg-white dark:bg-[#0F172A]">
          {renderNavList(false)}
        </div>

        {/* Bottom Core API Status & Collapse Footer */}
        <div className="p-2.5 border-t border-slate-200 dark:border-slate-800 bg-slate-50 dark:bg-[#0B1120] flex-shrink-0 space-y-2">
          {!collapsed && (
            <div className="flex items-center justify-between px-2.5 py-1.5 rounded-lg bg-white dark:bg-[#162033] border border-slate-200 dark:border-slate-700/50 text-[11px]">
              <div className="flex items-center gap-1.5 min-w-0">
                <span className="w-2 h-2 rounded-full bg-emerald-500 animate-pulse flex-shrink-0" />
                <span className="text-slate-700 dark:text-slate-300 font-semibold truncate">Control Plane</span>
              </div>
              <span className="text-[9px] font-bold text-emerald-700 dark:text-emerald-300 bg-emerald-50 dark:bg-emerald-500/15 border border-emerald-200 dark:border-emerald-500/20 px-1.5 py-0.5 rounded uppercase tracking-wider">
                ONLINE
              </span>
            </div>
          )}

          <button
            onClick={toggleCollapse}
            className="w-full py-1.5 rounded-lg flex items-center justify-center text-slate-600 dark:text-slate-400 hover:text-slate-900 dark:hover:text-white hover:bg-slate-100 dark:hover:bg-white/[0.08] transition"
            title={collapsed ? 'Expand sidebar' : 'Collapse sidebar'}
          >
            {collapsed ? (
              <ChevronRight className="w-4 h-4" />
            ) : (
              <div className="flex items-center gap-1.5 text-[11px] font-semibold text-slate-600 dark:text-slate-400">
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
            className="fixed inset-0 bg-black/60 backdrop-blur-xs transition-opacity animate-fadeIn"
          />

          {/* Drawer Content */}
          <div className="relative w-64 bg-white dark:bg-[#0F172A] border-r border-slate-200 dark:border-slate-800 flex flex-col h-full shadow-2xl z-10 animate-fadeIn">
            <div className="h-16 flex items-center justify-between px-4 border-b border-slate-200 dark:border-slate-800 flex-shrink-0">
              <div className="flex items-center gap-2.5">
                <div className="w-9 h-9 rounded-xl bg-gradient-to-tr from-emerald-600 to-teal-600 flex items-center justify-center text-white flex-shrink-0 shadow-sm shadow-emerald-600/20">
                  <Flame className="w-5 h-5 text-amber-300 fill-amber-300" />
                </div>
                <div>
                  <div className="flex items-center gap-1.5 leading-tight">
                    <span className="text-sm font-bold text-slate-900 dark:text-white tracking-tight">Hostvra</span>
                    <span className="text-[10px] font-bold px-1.5 py-0.5 rounded bg-emerald-100 dark:bg-emerald-500/20 text-emerald-800 dark:text-emerald-300">
                      Control
                    </span>
                  </div>
                  <div className="text-[10px] text-slate-500 dark:text-slate-400 font-medium">
                    Hosting Control Panel
                  </div>
                </div>
              </div>

              <button
                onClick={() => setMobileOpen(false)}
                className="w-7 h-7 rounded-lg bg-slate-100 dark:bg-slate-800 text-slate-600 dark:text-slate-400 hover:text-slate-900 dark:hover:text-white flex items-center justify-center"
              >
                <X className="w-4 h-4" />
              </button>
            </div>

            <div className="flex-1 overflow-y-auto px-3 py-3 custom-scrollbar bg-white dark:bg-[#0F172A]">
              {renderNavList(true)}
            </div>

            <div className="p-3 border-t border-slate-200 dark:border-slate-800 bg-slate-50 dark:bg-[#0B1120] flex-shrink-0">
              <div className="flex items-center justify-between px-2.5 py-1.5 rounded-lg bg-white dark:bg-[#162033] border border-slate-200 dark:border-slate-700/50 text-[11px]">
                <div className="flex items-center gap-1.5 min-w-0">
                  <span className="w-2 h-2 rounded-full bg-emerald-500 animate-pulse flex-shrink-0" />
                  <span className="text-slate-700 dark:text-slate-300 font-semibold truncate">Control Plane</span>
                </div>
                <span className="text-[9px] font-bold text-emerald-700 dark:text-emerald-300 bg-emerald-50 dark:bg-emerald-500/15 border border-emerald-200 dark:border-emerald-500/20 px-1.5 py-0.5 rounded uppercase tracking-wider">
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

