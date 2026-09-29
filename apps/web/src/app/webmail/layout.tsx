'use client';

import React from 'react';
import { usePathname } from 'next/navigation';
import { WebmailProvider, useWebmail } from '@/context/WebmailContext';
import { WebmailHeader } from '@/components/webmail/WebmailHeader';
import { WebmailSidebar } from '@/components/webmail/WebmailSidebar';
import { WebmailComposeModal } from '@/components/webmail/WebmailComposeModal';
import { WebmailAddAccountModal } from '@/components/webmail/WebmailAddAccountModal';
import WebmailLoginPage from './login/page';
import { Mail, ShieldCheck } from 'lucide-react';

function WebmailAuthGate({ children }: { children: React.ReactNode }) {
  const pathname = usePathname();
  const { isAuthenticated, isAuthLoading } = useWebmail();

  // Standalone unauthenticated pages (Login, SSO Ticket validation)
  const isPublicRoute = pathname === '/webmail/login' || pathname?.startsWith('/webmail/sso');

  if (isPublicRoute) {
    return <>{children}</>;
  }

  // 1. Prevent UI Flash: show minimal, elegant verifying loader ONLY.
  // Never show header, sidebar, dashboard, or mailbox content during verification.
  if (isAuthLoading) {
    return (
      <div className="min-h-screen h-screen flex flex-col items-center justify-center bg-slate-900 text-white font-sans select-none">
        <div className="flex flex-col items-center space-y-4 p-8 text-center animate-in fade-in duration-300">
          <div className="w-16 h-16 rounded-2xl bg-gradient-to-tr from-emerald-500 to-teal-500 flex items-center justify-center text-white shadow-xl shadow-emerald-500/20">
            <Mail className="w-8 h-8 animate-pulse" />
          </div>
          <div className="space-y-1">
            <h2 className="text-xl font-bold tracking-tight text-white flex items-center justify-center gap-1.5">
              Hostvra <span className="text-emerald-400">Webmail</span>
            </h2>
            <p className="text-xs text-slate-400 flex items-center justify-center gap-1.5">
              <ShieldCheck className="w-3.5 h-3.5 text-emerald-400" />
              Verifying secure session...
            </p>
          </div>
          <div className="w-48 h-1 bg-slate-800 rounded-full overflow-hidden mt-4">
            <div className="h-full bg-emerald-500 rounded-full animate-pulse" style={{ width: '60%' }} />
          </div>
        </div>
      </div>
    );
  }

  // 2. Strict Authentication Gate:
  // If not authenticated, render ONLY the dedicated Webmail Login screen.
  // Zero dashboard, zero sidebar, zero mailbox controls.
  if (!isAuthenticated) {
    return <WebmailLoginPage />;
  }

  // 3. Fully Authenticated:
  // Render the complete Webmail Dashboard
  return (
    <div className="min-h-screen h-screen flex flex-col bg-slate-50 text-slate-900 overflow-hidden font-sans">
      {/* Dedicated Standalone Webmail Header */}
      <WebmailHeader />

      {/* Main Body: Sidebar + Dynamic Mailbox / Settings Pane */}
      <div className="flex-1 flex overflow-hidden">
        <WebmailSidebar />
        <main className="flex-1 flex flex-col overflow-hidden bg-slate-50">
          {children}
        </main>
      </div>

      {/* Global Floating Compose Modal */}
      <WebmailComposeModal />

      {/* Global Multi-Account Login Modal */}
      <WebmailAddAccountModal />
    </div>
  );
}

export default function WebmailStandaloneLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <WebmailProvider>
      <WebmailAuthGate>{children}</WebmailAuthGate>
    </WebmailProvider>
  );
}
