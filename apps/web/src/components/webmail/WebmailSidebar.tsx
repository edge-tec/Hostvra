'use client';

import React from 'react';
import Link from 'next/link';
import { usePathname, useRouter } from 'next/navigation';
import {
  Inbox,
  Star,
  Send,
  FileText,
  Archive,
  AlertOctagon,
  Trash2,
  Users,
  Settings,
  Plus,
  PenSquare,
  HardDrive,
} from 'lucide-react';
import { useWebmail } from '@/context/WebmailContext';

function formatBytes(bytes?: number): string {
  if (!bytes || bytes === 0) return '0 B';
  const k = 1024;
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i];
}

export function WebmailSidebar() {
  const pathname = usePathname();
  const router = useRouter();
  const { folderCounts, openCompose, activeAccount } = useWebmail();

  const usedBytes = activeAccount?.usedBytes || 0;
  const quotaBytes = activeAccount?.quotaBytes || (1024 * 1024 * 1024 * 2);
  const usagePercent = Math.min(100, Math.round((usedBytes / quotaBytes) * 100));

  const navItems = [
    {
      id: 'inbox',
      label: 'Inbox',
      href: '/webmail/inbox',
      icon: Inbox,
      count: folderCounts['inbox'] || 0,
      unread: folderCounts['inboxUnread'] || 0,
      activeColor: 'text-emerald-600 dark:text-emerald-400',
    },
    {
      id: 'starred',
      label: 'Starred',
      href: '/webmail/starred',
      icon: Star,
      count: folderCounts['starred'] || 0,
      unread: 0,
      activeColor: 'text-amber-500',
    },
    {
      id: 'sent',
      label: 'Sent',
      href: '/webmail/sent',
      icon: Send,
      count: folderCounts['sent'] || 0,
      unread: 0,
      activeColor: 'text-blue-500',
    },
    {
      id: 'drafts',
      label: 'Drafts',
      href: '/webmail/drafts',
      icon: FileText,
      count: folderCounts['drafts'] || 0,
      unread: 0,
      activeColor: 'text-purple-500',
    },
    {
      id: 'archive',
      label: 'Archive',
      href: '/webmail/archive',
      icon: Archive,
      count: folderCounts['archive'] || 0,
      unread: 0,
      activeColor: 'text-slate-500',
    },
    {
      id: 'spam',
      label: 'Spam',
      href: '/webmail/spam',
      icon: AlertOctagon,
      count: folderCounts['spam'] || 0,
      unread: folderCounts['spamUnread'] || 0,
      activeColor: 'text-orange-500',
    },
    {
      id: 'trash',
      label: 'Trash',
      href: '/webmail/trash',
      icon: Trash2,
      count: folderCounts['trash'] || 0,
      unread: 0,
      activeColor: 'text-rose-500',
    },
  ];

  const isFolderActive = (href: string) => {
    if (pathname === '/webmail' && href === '/webmail/inbox') return true;
    return pathname === href || pathname?.startsWith(href + '/');
  };

  return (
    <aside className="w-64 bg-slate-50/70 dark:bg-[#080E1A] border-r border-slate-200 dark:border-slate-800 flex flex-col justify-between p-3.5 select-none transition-colors">
      <div className="space-y-4">
        {/* Large Gmail-like Compose Button */}
        <button
          onClick={() => openCompose()}
          className="w-full h-12 flex items-center justify-center gap-3 px-5 bg-gradient-to-r from-emerald-600 to-teal-600 hover:from-emerald-500 hover:to-teal-500 text-white font-bold text-sm rounded-2xl shadow-lg shadow-emerald-600/20 hover:shadow-emerald-600/30 transition-all hover:scale-[1.02] active:scale-[0.98]"
        >
          <PenSquare className="w-5 h-5" />
          <span>Compose</span>
        </button>

        {/* Folder Navigation */}
        <nav className="space-y-1">
          {navItems.map((item) => {
            const Icon = item.icon;
            const active = isFolderActive(item.href);

            return (
              <Link
                key={item.id}
                href={item.href}
                className={`flex items-center justify-between px-3.5 py-2.5 rounded-xl text-xs font-semibold transition-all group ${
                  active
                    ? 'bg-emerald-100/70 dark:bg-emerald-950/60 text-emerald-800 dark:text-emerald-300 font-bold'
                    : 'text-slate-600 dark:text-slate-400 hover:text-slate-900 dark:hover:text-white hover:bg-slate-100 dark:hover:bg-slate-800/60'
                }`}
              >
                <div className="flex items-center gap-3">
                  <Icon
                    className={`w-4 h-4 transition-colors ${
                      active ? item.activeColor : 'text-slate-400 group-hover:text-slate-600 dark:group-hover:text-slate-300'
                    }`}
                  />
                  <span>{item.label}</span>
                </div>

                {/* Badge count */}
                {item.unread > 0 ? (
                  <span className="px-2 py-0.5 rounded-full text-[10px] font-bold bg-emerald-600 text-white shadow-xs">
                    {item.unread}
                  </span>
                ) : item.count > 0 ? (
                  <span className="text-[11px] font-medium text-slate-400 dark:text-slate-500">
                    {item.count}
                  </span>
                ) : null}
              </Link>
            );
          })}
        </nav>

        {/* Auxiliary Navigation: Contacts & Settings */}
        <div className="pt-3 border-t border-slate-200/80 dark:border-slate-800 space-y-1">
          <p className="px-3 text-[10px] font-bold uppercase tracking-wider text-slate-400">
            Address & Rules
          </p>
          <Link
            href="/webmail/contacts"
            className={`flex items-center gap-3 px-3.5 py-2 rounded-xl text-xs font-semibold transition-all ${
              pathname === '/webmail/contacts'
                ? 'bg-emerald-100/70 dark:bg-emerald-950/60 text-emerald-800 dark:text-emerald-300 font-bold'
                : 'text-slate-600 dark:text-slate-400 hover:text-slate-900 dark:hover:text-white hover:bg-slate-100 dark:hover:bg-slate-800/60'
            }`}
          >
            <Users className="w-4 h-4 text-indigo-500" />
            <span>Contacts Book</span>
          </Link>

          <Link
            href="/webmail/settings"
            className={`flex items-center gap-3 px-3.5 py-2 rounded-xl text-xs font-semibold transition-all ${
              pathname === '/webmail/settings'
                ? 'bg-emerald-100/70 dark:bg-emerald-950/60 text-emerald-800 dark:text-emerald-300 font-bold'
                : 'text-slate-600 dark:text-slate-400 hover:text-slate-900 dark:hover:text-white hover:bg-slate-100 dark:hover:bg-slate-800/60'
            }`}
          >
            <Settings className="w-4 h-4 text-emerald-500" />
            <span>Mailbox Settings</span>
          </Link>
        </div>
      </div>

      {/* Storage Widget Card */}
      <div className="p-3 bg-white dark:bg-slate-900/80 rounded-2xl border border-slate-200/80 dark:border-slate-800 shadow-xs">
        <div className="flex items-center justify-between text-xs font-bold text-slate-700 dark:text-slate-300 mb-1.5">
          <span className="flex items-center gap-1.5">
            <HardDrive className="w-3.5 h-3.5 text-emerald-500" /> Storage
          </span>
          <span className="text-[11px] font-mono font-semibold text-slate-500 dark:text-slate-400">
            {usagePercent}%
          </span>
        </div>
        <div className="w-full bg-slate-100 dark:bg-slate-800 h-1.5 rounded-full overflow-hidden mb-1.5">
          <div
            className={`h-full rounded-full transition-all duration-300 ${
              usagePercent > 85 ? 'bg-rose-500' : usagePercent > 65 ? 'bg-amber-500' : 'bg-emerald-500'
            }`}
            style={{ width: `${usagePercent}%` }}
          />
        </div>
        <p className="text-[10px] text-slate-400 truncate">
          {formatBytes(usedBytes)} of {formatBytes(quotaBytes)} used
        </p>
      </div>
    </aside>
  );
}
