'use client';

import { use } from 'react';
import { WebmailMailboxView } from '@/components/webmail/WebmailMailboxView';

interface FolderPageProps {
  params: Promise<{
    folder: string;
  }>;
}

export default function FolderPage({ params }: FolderPageProps) {
  const resolvedParams = use(params);
  const folder = resolvedParams.folder || 'inbox';

  return <WebmailMailboxView folder={folder} />;
}
