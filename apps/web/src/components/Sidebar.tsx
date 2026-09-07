'use client';

import React, { useState, useEffect } from 'react';
import Link from 'next/link';
import { usePathname, useRouter } from 'next/navigation';
import {
  Home,
  Globe,
  Layers,
  ArrowLeftRight,
  Database,
  Container,
  Activity,
  Shield,
  ShieldCheck,
  Mail,
  Inbox,
  FolderTree,
  ScrollText,
  Code2,
  Network,
  Terminal,
  Clock,
  Boxes,
  Settings,
  LogOut,
  ChevronLeft,
  ChevronRight,
  X,
} from 'lucide-react';
import { clearStoredAuth } from '@/lib/api';

interface NavItem {
  label: string;
  href: string;
  icon: React.ElementType;
  badge?: string;
}

const navItems: NavItem[] = [
  { label: 'Home', href: '/dashboard', icon: Home },
  { label: 'Website', href: '/websites', icon: Globe },
  { label: 'WP Toolkit', href: '/websites', icon: Layers },
  { label: 'FTP', href: '/servers', icon: ArrowLeftRight },
  { label: 'Databases', href: '/databases', icon: Database },
  { label: 'Docker', href: '/docker', icon: Container },
  { label: 'Monitor', href: '/servers', icon: Activity },
  { label: 'Security', href: '/firewall', icon: Shield },
  { label: 'WAF', href: '/ssl', icon: ShieldCheck },
  { label: 'Mail Server', href: '/email', icon: Mail },
  { label: 'Webmail', href: '/webmail', icon: Inbox },
  { label: 'Files', href: '/files', icon: FolderTree },
  { label: 'Logs', href: '/audit-logs', icon: ScrollText },
  { label: 'Node', href: '/app-store', icon: Code2 },
  { label: 'Domains', href: '/dns', icon: Network },
  { label: 'Terminal', href: '/terminal', icon: Terminal },
  { label: 'Cron', href: '/cron', icon: Clock },
  { label: 'App Store', href: '/app-store', icon: Boxes },
  { label: 'Settings', href: '/settings', icon: Settings },
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

  const renderNavList = (isMobileView = false) => (
    <nav className="space-y-0.5">
      {navItems.map((item) => {
        const Icon = item.icon;
        // Strictly avoid double-highlighting: Home is only active when exact /dashboard
        const isActive =
          item.label === 'Home'
            ? pathname === '/dashboard'
            : item.label !== 'Home' &&
              item.href !== '/dashboard' &&
              (pathname === item.href || (item.href !== '/' && pathname?.startsWith(item.href)));

        return (
          <Link
            key={item.label}
            href={item.href}
            onClick={() => isMobileView && setMobileOpen(false)}
            title={!isMobileView && collapsed ? item.label : undefined}
            className={`flex items-center ${
              !isMobileView && collapsed ? 'justify-center px-0 py-2.5' : 'justify-between px-3 py-2'
            } rounded-lg text-xs font-medium transition-all ${
              isActive
                ? 'bg-[#20a53a] text-white font-semibold shadow-sm'
                : 'text-slate-600 dark:text-slate-300 hover:text-slate-900 dark:hover:text-white hover:bg-slate-100 dark:hover:bg-surface-800'
            }`}
          >
            <div className={`flex items-center ${!isMobileView && collapsed ? '' : 'gap-2.5'} truncate min-w-0`}>
              <Icon
                className={`w-4 h-4 flex-shrink-0 ${
                  isActive ? 'text-white' : 'text-slate-500 dark:text-slate-400'
                }`}
              />
              {(isMobileView || !collapsed) && <span className="truncate">{item.label}</span>}
            </div>

            {(isMobileView || !collapsed) && item.badge && (
              <span
                className={`text-[9px] font-bold px-1.5 py-0.5 rounded ${
                  isActive
                    ? 'bg-white/20 text-white'
                    : 'bg-emerald-100 dark:bg-emerald-950 text-emerald-600 dark:text-emerald-400'
                }`}
              >
                {item.badge}
              </span>
            )}
          </Link>
        );
      })}

      {/* Log Out */}
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
    </nav>
  );

  return (
    <>
      {/* 1. DESKTOP SIDEBAR (Visible on lg: screens and up) */}
      <aside
        className={`hidden lg:flex ${
          collapsed ? 'w-16' : 'w-52'
        } bg-white dark:bg-[#121824] border-r border-slate-200 dark:border-surface-800 flex-col h-screen select-none sticky top-0 transition-all duration-200 ease-in-out z-30 flex-shrink-0 shadow-sm`}
      >
        {/* Brand & Server IP Header */}
        <div className="h-14 flex items-center px-3 border-b border-slate-200 dark:border-surface-800 justify-between flex-shrink-0 overflow-hidden">
          <div className="flex items-center gap-2.5 min-w-0">
            <div className="w-8 h-8 rounded-lg bg-emerald-600 flex items-center justify-center text-white flex-shrink-0 shadow-xs">
              <span className="text-sm font-black tracking-tighter">H</span>
            </div>
            {!collapsed && (
              <div className="min-w-0 truncate">
                <div className="text-[11px] font-mono font-bold text-slate-800 dark:text-slate-200 truncate leading-tight">
                  13.140.157.238
                </div>
                <div className="flex items-center gap-1 mt-0.5">
                  <span className="w-1.5 h-1.5 rounded-full bg-emerald-500 animate-pulse flex-shrink-0" />
                  <span className="text-[10px] text-emerald-600 dark:text-emerald-400 font-medium truncate">
                    Hostvra Panel
                  </span>
                </div>
              </div>
            )}
          </div>

          {!collapsed && (
            <span className="w-4 h-4 rounded-full bg-emerald-100 dark:bg-emerald-950 text-emerald-600 dark:text-emerald-400 text-[10px] font-bold flex items-center justify-center flex-shrink-0">
              0
            </span>
          )}
        </div>

        {/* Navigation Scroll */}
        <div className="flex-1 overflow-y-auto px-2 py-3 space-y-1 custom-scrollbar">
          {renderNavList(false)}
        </div>

        {/* Collapse / Expand Toggle Button Footer */}
        <div className="p-2 border-t border-slate-200 dark:border-surface-800 bg-slate-50 dark:bg-[#0e1420] flex items-center justify-center flex-shrink-0">
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
            <div className="h-14 flex items-center justify-between px-4 border-b border-slate-200 dark:border-surface-800">
              <div className="flex items-center gap-2.5">
                <div className="w-8 h-8 rounded-lg bg-emerald-600 flex items-center justify-center text-white flex-shrink-0 shadow-xs">
                  <span className="text-sm font-black tracking-tighter">H</span>
                </div>
                <div>
                  <div className="text-xs font-mono font-bold text-slate-800 dark:text-slate-200 leading-tight">
                    13.140.157.238
                  </div>
                  <span className="text-[10px] text-emerald-600 dark:text-emerald-400 font-medium">
                    Hostvra Panel
                  </span>
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
          </div>
        </div>
      )}
    </>
  );
}
