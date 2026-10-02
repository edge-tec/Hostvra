'use client';

import React, { useEffect, useState } from 'react';
import { useRouter, usePathname } from 'next/navigation';
import { Sidebar } from './Sidebar';
import { Header } from './Header';
import { getStoredToken } from '@/lib/api';

export function DashboardShell({ children }: { children: React.ReactNode }) {
  const router = useRouter();
  const pathname = usePathname();
  const [isAuthenticated, setIsAuthenticated] = useState<boolean | null>(() => {
    if (typeof window !== 'undefined') {
      return getStoredToken() !== null;
    }
    return null;
  });

  useEffect(() => {
    const token = getStoredToken();
    if (!token) {
      setIsAuthenticated(false);
      const redirectUrl = pathname && pathname !== '/' 
        ? `/login?redirect=${encodeURIComponent(pathname)}` 
        : '/login';
      router.replace(redirectUrl);
    } else {
      setIsAuthenticated(true);
      // Synchronize cookie for server-side / middleware requests
      if (typeof document !== 'undefined' && !document.cookie.includes('hostvra_token=')) {
        document.cookie = `hostvra_token=${encodeURIComponent(token)}; path=/; max-age=604800; SameSite=Lax`;
      }
    }
  }, [pathname, router]);

  const [isImpersonating, setIsImpersonating] = useState(false);

  useEffect(() => {
    if (typeof window !== 'undefined') {
      setIsImpersonating(Boolean(localStorage.getItem('hostvra_admin_backup_token')));
    }
  }, [pathname]);

  const returnToAdmin = () => {
    const adminToken = localStorage.getItem('hostvra_admin_backup_token');
    if (adminToken) {
      localStorage.setItem('hostvra_token', adminToken);
      localStorage.removeItem('hostvra_admin_backup_token');
      document.cookie = `hostvra_token=${encodeURIComponent(adminToken)}; path=/; max-age=604800; SameSite=Lax`;
      const userStr = localStorage.getItem('hostvra_user');
      if (userStr) {
        try {
          const u = JSON.parse(userStr);
          u.role = 'admin';
          u.is_superadmin = true;
          delete u.impersonated;
          localStorage.setItem('hostvra_user', JSON.stringify(u));
        } catch {}
      }
      window.location.href = '/admin/users';
    }
  };

  // While checking authentication or if unauthenticated, show a clean loading screen
  if (isAuthenticated !== true) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-white dark:bg-[#0B1120]">
        <div className="flex flex-col items-center gap-4 text-center px-4">
          <div className="w-10 h-10 border-3 border-emerald-600 border-t-transparent rounded-full animate-spin" />
          <div>
            <h2 className="text-base font-bold text-slate-900 dark:text-white">
              Authenticating Session
            </h2>
            <p className="text-xs text-slate-500 dark:text-slate-400 mt-1">
              Protected area. Redirecting to login...
            </p>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="flex h-screen w-full overflow-hidden bg-white dark:bg-[#0B1120] text-slate-900 dark:text-slate-100 transition-colors duration-150 font-sans">
      <Sidebar />
      <div className="flex-1 flex flex-col h-full min-w-0 overflow-hidden bg-white dark:bg-[#0B1120]">
        {isImpersonating && (
          <div className="bg-amber-500 text-slate-950 px-4 py-2 text-xs font-bold flex items-center justify-between shadow-md z-50 border-b border-amber-600">
            <div className="flex items-center gap-2">
              <span className="inline-block w-2 h-2 rounded-full bg-red-600 animate-pulse" />
              <span>AUDIT NOTICE: You are logged into a customer account via Admin Impersonation. Admin actions are restricted.</span>
            </div>
            <button
              onClick={returnToAdmin}
              className="px-3 py-1 bg-slate-950 hover:bg-slate-800 text-white rounded-lg text-xs font-bold transition-colors cursor-pointer"
            >
              Return to Admin Panel
            </button>
          </div>
        )}
        <Header />
        <main className="flex-1 overflow-y-auto overflow-x-hidden p-3.5 sm:p-6 lg:p-8 w-full bg-white dark:bg-[#0B1120]">
          <div className="w-full max-w-[1600px] mx-auto space-y-6">
            {children}
          </div>
        </main>
      </div>
    </div>
  );
}

