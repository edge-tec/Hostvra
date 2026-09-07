'use client';

import React, { Suspense } from 'react';
import { useSearchParams } from 'next/navigation';
import { DashboardShell } from '@/components/DashboardShell';
import { WebmailClient } from '@/components/WebmailClient';

function WebmailContent() {
  const searchParams = useSearchParams();
  const accountParam = searchParams.get('account') || undefined;

  return (
    <div className="space-y-4">
      <WebmailClient initialSelectedEmail={accountParam} />
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
