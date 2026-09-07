'use client';

import React, { Suspense, useEffect, useState } from 'react';
import { useSearchParams } from 'next/navigation';
import { DashboardShell } from '@/components/DashboardShell';
import { WebmailClient, WebmailMailbox } from '@/components/WebmailClient';
import { apiFetch } from '@/lib/api';

function WebmailContent() {
  const searchParams = useSearchParams();
  const accountParam = searchParams.get('account') || undefined;
  const [mailboxes, setMailboxes] = useState<WebmailMailbox[]>([]);

  useEffect(() => {
    async function loadMailboxes() {
      try {
        const res = await apiFetch<WebmailMailbox[]>('/api/v1/email/mailboxes');
        if (res.data && res.data.length > 0) {
          setMailboxes(res.data);
        }
      } catch (err) {
        console.error('Failed to load mailboxes in webmail:', err);
      }
    }
    loadMailboxes();
  }, []);

  return (
    <div className="space-y-4">
      <WebmailClient 
        mailboxes={mailboxes.length > 0 ? mailboxes : undefined}
        initialSelectedEmail={accountParam} 
      />
    </div>
  );
}

export default function WebmailPage() {
  return (
    <DashboardShell>
      <Suspense fallback={<div className="p-8 text-center text-slate-400">Loading Webmail Suite...</div>}>
        <WebmailContent />
      </Suspense>
    </DashboardShell>
  );
}
