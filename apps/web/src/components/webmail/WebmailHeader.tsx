'use client';

import React, { useState, useRef, useEffect } from 'react';
import Link from 'next/link';
import { usePathname } from 'next/navigation';
import {
  Mail,
  Search,
  RefreshCw,
  Settings,
  Bell,
  BellOff,
  Volume2,
  VolumeX,
  Sun,
  Moon,
  LogOut,
  UserPlus,
  Check,
  ChevronDown,
  HardDrive,
  SlidersHorizontal,
  X,
  Inbox,
  Filter,
} from 'lucide-react';
import { useWebmail } from '@/context/WebmailContext';
import { useTheme } from '@/components/ThemeProvider';

function formatBytes(bytes?: number): string {
  if (!bytes || bytes === 0) return '0 B';
  const k = 1024;
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i];
}

export function WebmailHeader() {
  const {
    accounts,
    activeAccount,
    switchAccount,
    logoutCurrentAccount,
    logoutAllAccounts,
    isSyncing,
    refreshFolderCounts,
    searchQuery,
    setSearchQuery,
    openAddAccount,
    soundEnabled,
    toggleSound,
    desktopNotificationsEnabled,
    requestDesktopNotifications,
  } = useWebmail();

  const { theme, toggleTheme } = useTheme();
  const [accountMenuOpen, setAccountMenuOpen] = useState(false);
  const [filterHelperOpen, setFilterHelperOpen] = useState(false);
  const accountMenuRef = useRef<HTMLDivElement>(null);
  const searchInputRef = useRef<HTMLInputElement>(null);

  // Close menus on outside click
  useEffect(() => {
    function handleClickOutside(e: MouseEvent) {
      if (accountMenuRef.current && !accountMenuRef.current.contains(e.target as Node)) {
        setAccountMenuOpen(false);
      }
    }
    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, []);

  // Keyboard shortcut '/' to focus search
  useEffect(() => {
    function handleKeyDown(e: KeyboardEvent) {
      if (e.key === '/' && document.activeElement?.tagName !== 'INPUT' && document.activeElement?.tagName !== 'TEXTAREA') {
        e.preventDefault();
        searchInputRef.current?.focus();
      }
    }
    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, []);

  const usedBytes = activeAccount?.usedBytes || 0;
  const quotaBytes = activeAccount?.quotaBytes || (1024 * 1024 * 1024 * 2); // Default 2GB if not set
  const usagePercent = Math.min(100, Math.round((usedBytes / quotaBytes) * 100));

  // Compute initials
  const initials = activeAccount?.name
    ? activeAccount.name.slice(0, 2).toUpperCase()
    : (activeAccount?.email?.slice(0, 2).toUpperCase() || 'WM');

  const searchOperators = [
    { label: 'from:', desc: 'from:user@domain.com', tip: 'from:' },
    { label: 'to:', desc: 'to:support@hostvra.com', tip: 'to:' },
    { label: 'subject:', desc: 'subject:invoice', tip: 'subject:' },
    { label: 'has:attachment', desc: 'Emails with files attached', tip: 'has:attachment ' },
    { label: 'is:unread', desc: 'Only unread messages', tip: 'is:unread ' },
    { label: 'is:starred', desc: 'Only starred messages', tip: 'is:starred ' },
  ];

  return (
    <header className="h-16 px-4 sm:px-6 bg-white dark:bg-[#0B1120] border-b border-slate-200 dark:border-slate-800 flex items-center justify-between gap-4 sticky top-0 z-30 transition-colors">
      {/* Left: Hostvra Webmail Brand */}
      <div className="flex items-center gap-3 min-w-[200px] flex-shrink-0">
        <Link href="/webmail" className="flex items-center gap-2.5 group">
          <div className="w-9 h-9 rounded-xl bg-gradient-to-tr from-emerald-600 to-teal-500 flex items-center justify-center text-white shadow-md shadow-emerald-500/20 group-hover:scale-105 transition-transform">
            <Mail className="w-5 h-5" />
          </div>
          <div>
            <div className="flex items-center gap-1.5">
              <span className="font-extrabold text-base tracking-tight text-slate-900 dark:text-white">
                Hostvra
              </span>
              <span className="text-[11px] font-bold px-1.5 py-0.5 rounded bg-emerald-100 dark:bg-emerald-950/80 text-emerald-700 dark:text-emerald-400 border border-emerald-300 dark:border-emerald-800">
                Mail
              </span>
            </div>
            <p className="text-[10px] text-slate-400 dark:text-slate-500 font-medium">
              Enterprise Standalone
            </p>
          </div>
        </Link>
      </div>

      {/* Middle: Gmail-like Omnibox Search Bar */}
      <div className="flex-1 max-w-2xl relative hidden md:block">
        <div className="relative flex items-center">
          <div className="absolute inset-y-0 left-0 pl-3.5 flex items-center pointer-events-none text-slate-400">
            <Search className="w-4 h-4" />
          </div>
          <input
            ref={searchInputRef}
            type="text"
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            onFocus={() => setFilterHelperOpen(true)}
            placeholder="Search mail (e.g. from:support, has:attachment, subject:invoice)..."
            className="w-full pl-10 pr-20 py-2.5 bg-slate-100 dark:bg-slate-900/90 text-slate-900 dark:text-slate-100 text-xs rounded-xl border border-transparent focus:border-emerald-500 focus:bg-white dark:focus:bg-slate-900 focus:ring-2 focus:ring-emerald-500/20 transition-all placeholder:text-slate-400"
          />
          <div className="absolute inset-y-0 right-0 pr-2.5 flex items-center gap-1">
            {searchQuery && (
              <button
                onClick={() => setSearchQuery('')}
                className="p-1 text-slate-400 hover:text-slate-600 dark:hover:text-slate-200 rounded-md"
                title="Clear search"
              >
                <X className="w-3.5 h-3.5" />
              </button>
            )}
            <kbd className="hidden lg:inline-flex items-center px-1.5 py-0.5 text-[10px] font-mono text-slate-400 bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 rounded shadow-xs">
              /
            </kbd>
          </div>
        </div>

        {/* Search Operator Helpers Dropdown */}
        {filterHelperOpen && (
          <div
            className="absolute left-0 right-0 top-full mt-1.5 p-3 bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 rounded-xl shadow-xl z-50 animate-in fade-in slide-in-from-top-1 duration-150"
            onMouseDown={(e) => e.preventDefault()}
          >
            <div className="flex items-center justify-between pb-2 mb-2 border-b border-slate-100 dark:border-slate-800 text-[11px] font-semibold text-slate-500">
              <span className="flex items-center gap-1.5">
                <Filter className="w-3 h-3 text-emerald-500" /> Search Filters & Operators
              </span>
              <button
                onClick={() => setFilterHelperOpen(false)}
                className="text-slate-400 hover:text-slate-600 text-[10px]"
              >
                Close
              </button>
            </div>
            <div className="grid grid-cols-2 gap-1.5">
              {searchOperators.map((op) => (
                <button
                  key={op.label}
                  onClick={() => {
                    setSearchQuery((prev) => prev ? `${prev} ${op.tip}` : op.tip);
                    searchInputRef.current?.focus();
                  }}
                  className="flex items-center justify-between px-2.5 py-1.5 text-left rounded-lg hover:bg-slate-100 dark:hover:bg-slate-800/80 transition-colors group"
                >
                  <span className="font-mono text-xs font-semibold text-emerald-600 dark:text-emerald-400">
                    {op.label}
                  </span>
                  <span className="text-[10px] text-slate-400 truncate max-w-[120px]">
                    {op.desc}
                  </span>
                </button>
              ))}
            </div>
          </div>
        )}
      </div>

      {/* Right: Quick Tools & Multi-Account Switcher */}
      <div className="flex items-center gap-2 sm:gap-3 flex-shrink-0">
        {/* Refresh / Sync button */}
        <button
          onClick={refreshFolderCounts}
          disabled={isSyncing}
          className={`p-2 rounded-xl text-slate-500 dark:text-slate-400 hover:text-slate-900 dark:hover:text-white hover:bg-slate-100 dark:hover:bg-slate-800 transition-all ${
            isSyncing ? 'animate-spin text-emerald-600' : ''
          }`}
          title="Sync / Refresh Mailbox"
        >
          <RefreshCw className="w-4 h-4" />
        </button>

        {/* Audio chime toggle */}
        <button
          onClick={toggleSound}
          className="p-2 rounded-xl text-slate-500 dark:text-slate-400 hover:text-slate-900 dark:hover:text-white hover:bg-slate-100 dark:hover:bg-slate-800 transition-all"
          title={soundEnabled ? 'Incoming mail chime is ON' : 'Incoming mail chime is MUTED'}
        >
          {soundEnabled ? (
            <Volume2 className="w-4 h-4 text-emerald-600 dark:text-emerald-400" />
          ) : (
            <VolumeX className="w-4 h-4 text-slate-400" />
          )}
        </button>

        {/* Desktop notification button */}
        <button
          onClick={requestDesktopNotifications}
          className="p-2 rounded-xl text-slate-500 dark:text-slate-400 hover:text-slate-900 dark:hover:text-white hover:bg-slate-100 dark:hover:bg-slate-800 transition-all"
          title={desktopNotificationsEnabled ? 'Desktop notifications ACTIVE' : 'Enable Desktop Notifications'}
        >
          {desktopNotificationsEnabled ? (
            <Bell className="w-4 h-4 text-emerald-600 dark:text-emerald-400" />
          ) : (
            <BellOff className="w-4 h-4 text-slate-400" />
          )}
        </button>

        {/* Theme Toggle */}
        <button
          onClick={toggleTheme}
          className="p-2 rounded-xl text-slate-500 dark:text-slate-400 hover:text-slate-900 dark:hover:text-white hover:bg-slate-100 dark:hover:bg-slate-800 transition-all"
          title="Toggle Dark / Light Mode"
        >
          {theme === 'dark' ? <Sun className="w-4 h-4 text-amber-400" /> : <Moon className="w-4 h-4" />}
        </button>

        {/* Webmail Settings Link */}
        <Link
          href="/webmail/settings"
          className="p-2 rounded-xl text-slate-500 dark:text-slate-400 hover:text-slate-900 dark:hover:text-white hover:bg-slate-100 dark:hover:bg-slate-800 transition-all"
          title="Webmail Settings"
        >
          <Settings className="w-4 h-4" />
        </Link>

        {/* Multi-Account Switcher Avatar Menu */}
        <div className="relative" ref={accountMenuRef}>
          <button
            onClick={() => setAccountMenuOpen(!accountMenuOpen)}
            className="flex items-center gap-2 pl-1 pr-2 py-1 rounded-xl bg-slate-100 dark:bg-slate-800 hover:bg-slate-200 dark:hover:bg-slate-700 transition-all border border-slate-200/80 dark:border-slate-700"
          >
            <div className="w-7 h-7 rounded-lg bg-gradient-to-tr from-emerald-600 to-teal-500 text-white font-bold text-xs flex items-center justify-center shadow-xs">
              {initials}
            </div>
            <span className="hidden xl:inline text-xs font-semibold text-slate-800 dark:text-slate-200 max-w-[130px] truncate">
              {activeAccount?.email || 'Sign In'}
            </span>
            <ChevronDown className="w-3.5 h-3.5 text-slate-400" />
          </button>

          {/* Dropdown Menu */}
          {accountMenuOpen && (
            <div className="absolute right-0 top-full mt-2 w-80 bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 rounded-2xl shadow-2xl z-50 p-3 animate-in fade-in slide-in-from-top-2 duration-150">
              {/* Current Account Card */}
              {activeAccount ? (
                <div className="p-3 bg-slate-50 dark:bg-slate-800/60 rounded-xl mb-3 border border-slate-200/60 dark:border-slate-700/60">
                  <div className="flex items-center gap-3 mb-2">
                    <div className="w-10 h-10 rounded-xl bg-gradient-to-tr from-emerald-600 to-teal-500 text-white font-bold text-sm flex items-center justify-center shadow-sm">
                      {initials}
                    </div>
                    <div className="min-w-0 flex-1">
                      <p className="text-xs font-bold text-slate-900 dark:text-white truncate">
                        {activeAccount.name}
                      </p>
                      <p className="text-[11px] text-slate-500 dark:text-slate-400 truncate">
                        {activeAccount.email}
                      </p>
                    </div>
                  </div>

                  {/* Quota Progress */}
                  <div className="pt-2 border-t border-slate-200/60 dark:border-slate-700/60">
                    <div className="flex items-center justify-between text-[10px] text-slate-500 dark:text-slate-400 mb-1">
                      <span className="flex items-center gap-1">
                        <HardDrive className="w-3 h-3" /> Storage
                      </span>
                      <span>
                        {formatBytes(usedBytes)} of {formatBytes(quotaBytes)} ({usagePercent}%)
                      </span>
                    </div>
                    <div className="w-full bg-slate-200 dark:bg-slate-700 h-1.5 rounded-full overflow-hidden">
                      <div
                        className={`h-full transition-all duration-300 ${
                          usagePercent > 85 ? 'bg-rose-500' : usagePercent > 65 ? 'bg-amber-500' : 'bg-emerald-500'
                        }`}
                        style={{ width: `${usagePercent}%` }}
                      />
                    </div>
                  </div>
                </div>
              ) : (
                <div className="p-3 text-center text-xs text-slate-400">
                  No mailbox selected
                </div>
              )}

              {/* Other Accounts List */}
              <div className="space-y-1 mb-3">
                <p className="px-2 py-1 text-[10px] font-bold uppercase tracking-wider text-slate-400">
                  Switch Mailbox ({accounts.length})
                </p>
                {accounts.length > 0 ? (
                  accounts.map((acc) => {
                    const isCurrent = acc.email.toLowerCase() === activeAccount?.email.toLowerCase();
                    return (
                      <button
                        key={acc.email}
                        onClick={() => {
                          switchAccount(acc.email);
                          setAccountMenuOpen(false);
                        }}
                        className={`w-full flex items-center justify-between p-2 rounded-xl text-left transition-all ${
                          isCurrent
                            ? 'bg-emerald-50 dark:bg-emerald-950/40 text-emerald-700 dark:text-emerald-300 font-semibold'
                            : 'hover:bg-slate-100 dark:hover:bg-slate-800 text-slate-700 dark:text-slate-300'
                        }`}
                      >
                        <div className="flex items-center gap-2.5 min-w-0">
                          <div className="w-6 h-6 rounded-lg bg-slate-200 dark:bg-slate-700 flex items-center justify-center text-[10px] font-bold text-slate-700 dark:text-slate-300">
                            {acc.email.slice(0, 2).toUpperCase()}
                          </div>
                          <span className="text-xs truncate">{acc.email}</span>
                        </div>
                        {isCurrent && <Check className="w-3.5 h-3.5 text-emerald-600" />}
                      </button>
                    );
                  })
                ) : (
                  <p className="px-2 py-1 text-xs text-slate-400">No other mailboxes configured</p>
                )}
              </div>

              {/* Actions: Add Account & Logouts */}
              <div className="pt-2 border-t border-slate-100 dark:border-slate-800 space-y-1">
                <button
                  onClick={() => {
                    setAccountMenuOpen(false);
                    openAddAccount();
                  }}
                  className="w-full flex items-center gap-2 px-3 py-2 text-xs font-medium text-slate-700 dark:text-slate-300 hover:bg-slate-100 dark:hover:bg-slate-800 rounded-xl transition-colors"
                >
                  <UserPlus className="w-4 h-4 text-emerald-600 dark:text-emerald-400" />
                  Add another email account
                </button>

                <button
                  onClick={() => {
                    setAccountMenuOpen(false);
                    logoutCurrentAccount();
                  }}
                  className="w-full flex items-center gap-2 px-3 py-2 text-xs font-medium text-slate-700 dark:text-slate-300 hover:bg-slate-100 dark:hover:bg-slate-800 rounded-xl transition-colors"
                >
                  <LogOut className="w-4 h-4 text-amber-500" />
                  Sign out of this mailbox
                </button>

                <button
                  onClick={() => {
                    setAccountMenuOpen(false);
                    logoutAllAccounts();
                  }}
                  className="w-full flex items-center gap-2 px-3 py-2 text-xs font-medium text-rose-600 dark:text-rose-400 hover:bg-rose-50 dark:hover:bg-rose-950/30 rounded-xl transition-colors"
                >
                  <LogOut className="w-4 h-4 text-rose-600" />
                  Sign out of all accounts
                </button>
              </div>
            </div>
          )}
        </div>
      </div>
    </header>
  );
}
