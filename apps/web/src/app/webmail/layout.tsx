'use client';

import React from 'react';
import { WebmailProvider } from '@/context/WebmailContext';
import { WebmailHeader } from '@/components/webmail/WebmailHeader';
import { WebmailSidebar } from '@/components/webmail/WebmailSidebar';
import { WebmailComposeModal } from '@/components/webmail/WebmailComposeModal';
import { WebmailAddAccountModal } from '@/components/webmail/WebmailAddAccountModal';

export default function WebmailStandaloneLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <WebmailProvider>
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
    </WebmailProvider>
  );
}
