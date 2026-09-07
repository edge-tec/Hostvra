'use client';

import React from 'react';
import { Sidebar } from './Sidebar';
import { Header } from './Header';

export function DashboardShell({ children }: { children: React.ReactNode }) {
  return (
    <div className="flex min-h-screen bg-white dark:bg-[#0B1120] text-slate-900 dark:text-slate-100 transition-colors duration-150 font-sans">
      <Sidebar />
      <div className="flex-1 flex flex-col min-w-0 overflow-hidden bg-white dark:bg-[#0B1120]">
        <Header />
        <main className="flex-1 p-4 sm:p-6 lg:p-8 overflow-y-auto max-w-[1600px] w-full mx-auto space-y-6 bg-slate-50/50 dark:bg-[#0B1120]">
          {children}
        </main>
      </div>
    </div>
  );
}
