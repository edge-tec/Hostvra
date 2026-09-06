'use client';

import React, { useState, useEffect } from 'react';
import { useRouter } from 'next/navigation';
import {
  Bell,
  Building2,
  ChevronDown,
  LogOut,
  Shield,
  User as UserIcon,
  Server,
} from 'lucide-react';
import { clearStoredAuth, apiFetch, User, Organization } from '@/lib/api';

export function Header() {
  const router = useRouter();
  const [user, setUser] = useState<User | null>(null);
  const [org, setOrg] = useState<Organization | null>(null);
  const [menuOpen, setMenuOpen] = useState(false);

  useEffect(() => {
    async function loadMe() {
      const res = await apiFetch<{ user: User; org: Organization }>('/api/v1/auth/me');
      if (res.success && res.data) {
        setUser(res.data.user);
        setOrg(res.data.org);
      }
    }
    loadMe();
  }, []);

  const handleLogout = () => {
    clearStoredAuth();
    router.push('/login');
  };

  return (
    <header className="h-16 border-b border-surface-800 bg-surface-900/60 backdrop-blur-md px-6 flex items-center justify-between sticky top-0 z-20">
      {/* Organization Switcher */}
      <div className="flex items-center gap-3">
        <div className="flex items-center gap-2 px-3 py-1.5 rounded-lg bg-surface-800/80 border border-surface-700 text-sm font-medium text-slate-200">
          <Building2 className="w-4 h-4 text-indigo-400" />
          <span>{org ? org.name : 'Hostvra Cloud'}</span>
          <span className="text-[11px] font-semibold uppercase px-1.5 py-0.5 rounded bg-indigo-500/10 text-indigo-400 border border-indigo-500/20">
            {org ? org.plan_tier : 'Free'}
          </span>
        </div>
      </div>

      {/* Right Controls */}
      <div className="flex items-center gap-4">
        <button
          className="p-2 rounded-lg text-slate-400 hover:text-slate-200 hover:bg-surface-800 transition-colors relative"
          title="Notifications"
        >
          <Bell className="w-4 h-4" />
          <span className="w-2 h-2 rounded-full bg-indigo-500 absolute top-1.5 right-1.5 ring-2 ring-surface-900" />
        </button>

        <div className="h-5 w-px bg-surface-800" />

        {/* User Profile */}
        <div className="relative">
          <button
            onClick={() => setMenuOpen(!menuOpen)}
            className="flex items-center gap-3 px-2 py-1.5 rounded-lg hover:bg-surface-800 transition-colors"
          >
            <div className="w-8 h-8 rounded-lg bg-indigo-600/30 border border-indigo-500/40 flex items-center justify-center text-sm font-bold text-indigo-300">
              {user ? user.full_name.charAt(0).toUpperCase() : 'A'}
            </div>
            <div className="text-left hidden sm:block">
              <div className="text-sm font-medium text-slate-200 leading-none">
                {user ? user.full_name : 'Administrator'}
              </div>
              <div className="text-xs text-slate-400 capitalize mt-1">
                {user ? user.role : 'Owner'}
              </div>
            </div>
            <ChevronDown className="w-4 h-4 text-slate-400" />
          </button>

          {menuOpen && (
            <div className="absolute right-0 mt-2 w-56 rounded-xl bg-surface-850 border border-surface-700 shadow-2xl py-1 text-sm z-50">
              <div className="px-4 py-2 border-b border-surface-700">
                <p className="text-xs text-slate-400">Signed in as</p>
                <p className="font-medium text-slate-200 truncate">{user ? user.email : 'admin@hostvra.com'}</p>
              </div>

              <div className="py-1">
                <button
                  onClick={() => router.push('/settings')}
                  className="w-full flex items-center gap-2 px-4 py-2 text-slate-300 hover:bg-surface-800 text-left"
                >
                  <UserIcon className="w-4 h-4 text-slate-400" />
                  Profile Settings
                </button>
                <button
                  onClick={() => router.push('/servers')}
                  className="w-full flex items-center gap-2 px-4 py-2 text-slate-300 hover:bg-surface-800 text-left"
                >
                  <Server className="w-4 h-4 text-slate-400" />
                  Manage Fleet
                </button>
              </div>

              <div className="border-t border-surface-700 py-1">
                <button
                  onClick={handleLogout}
                  className="w-full flex items-center gap-2 px-4 py-2 text-rose-400 hover:bg-rose-500/10 text-left font-medium"
                >
                  <LogOut className="w-4 h-4" />
                  Sign Out
                </button>
              </div>
            </div>
          )}
        </div>
      </div>
    </header>
  );
}
