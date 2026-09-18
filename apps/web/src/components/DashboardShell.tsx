'use client';

import React, { useEffect, useState } from 'react';
import { useRouter, usePathname } from 'next/navigation';
import { Sidebar } from './Sidebar';
import { Header } from './Header';
import { getStoredToken } from '@/lib/api';

export function DashboardShell({ children }: { children: React.ReactNode }) {
  const router = useRouter();
  const pathname = usePathname();
  const [isAuthenticated, setIsAuthenticated] = useState<boolean | null>(null);

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

  // While checking authentication or if unauthenticated, show a clean loading screen
  if (isAuthenticated !== true) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-white dark:bg-[#0B1120]">
        <div className="flex flex-col items-center gap-4 text-center px-4">
          <div className="w-10 h-10 border-3 border-indigo-600 border-t-transparent rounded-full animate-spin" />
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
    <div className="flex min-h-screen bg-white dark:bg-[#0B1120] text-slate-900 dark:text-slate-100 transition-colors duration-150 font-sans">
      <Sidebar />
      <div className="flex-1 flex flex-col min-w-0 overflow-hidden bg-white dark:bg-[#0B1120]">
        <Header />
        <main className="flex-1 p-4 sm:p-6 lg:p-8 overflow-y-auto max-w-[1600px] w-full mx-auto space-y-6 bg-white dark:bg-[#0B1120]">
          {children}
        </main>
      </div>
    </div>
  );
}

