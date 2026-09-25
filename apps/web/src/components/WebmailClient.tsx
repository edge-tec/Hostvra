'use client';

import React, { useState, useEffect, useMemo, useRef } from 'react';
import {
  Inbox,
  Send,
  Star,
  FileText,
  AlertOctagon,
  Trash2,
  Search,
  Plus,
  RefreshCw,
  ExternalLink,
  Reply,
  Forward,
  Paperclip,
  Check,
  CheckCheck,
  ShieldCheck,
  Lock,
  Archive,
  ArrowLeft,
  Settings,
  HardDrive,
  User,
  X,
  AlertTriangle,
  Mail,
  Clock,
  Sparkles,
  ChevronDown,
} from 'lucide-react';
import { apiFetch } from '@/lib/api';

export interface WebmailMailbox {
  id: string;
  email: string;
  name: string;
  quota_bytes: number;
  used_bytes: number;
}

export interface WebmailAttachment {
  id: string;
  message_id: string;
  filename: string;
  content_type: string;
  size_bytes: number;
  content?: string;
}

export interface WebmailMessage {
  id: string;
  mailbox_id: string;
  folder: string;
  from_name: string;
  from_email: string;
  to_addresses: string[];
  cc_addresses?: string[];
  bcc_addresses?: string[];
  subject: string;
  body_text: string;
  body_html?: string;
  size_bytes: number;
  is_read: boolean;
  is_starred: boolean;
  is_draft: boolean;
  message_id?: string;
  has_attachments: boolean;
  attachments?: WebmailAttachment[];
  created_at: string;
}

interface WebmailClientProps {
  mailboxes?: WebmailMailbox[];
  initialSelectedEmail?: string;
  onBackToEmailSettings?: () => void;
  showBackToSettings?: boolean;
}

export function WebmailClient({
  mailboxes = [],
  initialSelectedEmail,
  onBackToEmailSettings,
  showBackToSettings = false,
}: WebmailClientProps) {
  // Selected Mailbox Account
  const [selectedMailboxId, setSelectedMailboxId] = useState<string>(() => {
    if (initialSelectedEmail) {
      const match = mailboxes.find((m) => m.email === initialSelectedEmail);
      if (match) return match.id;
    }
    return mailboxes[0]?.id || '';
  });

  const activeMailbox = useMemo(() => {
    return (
      mailboxes.find((m) => m.id === selectedMailboxId) ||
      mailboxes[0] || {
        id: '',
        email: 'No mailbox configured',
        name: 'Mailbox User',
        quota_bytes: 5368709120,
        used_bytes: 0,
      }
    );
  }, [mailboxes, selectedMailboxId]);

  // Sync selected mailbox if mailboxes prop updates
  useEffect(() => {
    if (!selectedMailboxId && mailboxes.length > 0) {
      setSelectedMailboxId(mailboxes[0].id);
    }
  }, [mailboxes, selectedMailboxId]);

  // Current folder
  const [currentFolder, setCurrentFolder] = useState<string>('inbox');
  const [searchQuery, setSearchQuery] = useState('');
  const [messages, setMessages] = useState<WebmailMessage[]>([]);
  const [loading, setLoading] = useState(false);
  const [refreshing, setRefreshing] = useState(false);

  // Selected message for reading
  const [selectedMessageId, setSelectedMessageId] = useState<string | null>(null);
  const [activeMessage, setActiveMessage] = useState<WebmailMessage | null>(null);
  const [loadingDetail, setLoadingDetail] = useState(false);

  // Folder Counts
  const [folderCounts, setFolderCounts] = useState<{ [key: string]: number }>({
    inbox: 0,
    inboxUnread: 0,
    sent: 0,
    drafts: 0,
    starred: 0,
    spam: 0,
    trash: 0,
    archive: 0,
  });

  // Modals & Composer
  const [showComposeModal, setShowComposeModal] = useState(false);
  const [composeTo, setComposeTo] = useState('');
  const [composeCc, setComposeCc] = useState('');
  const [composeBcc, setComposeBcc] = useState('');
  const [showCcBcc, setShowCcBcc] = useState(false);
  const [composeSubject, setComposeSubject] = useState('');
  const [composeBody, setComposeBody] = useState('');
  const [isHtmlMode, setIsHtmlMode] = useState(false);
  const [isSending, setIsSending] = useState(false);
  const [isSavingDraft, setIsSavingDraft] = useState(false);
  const [activeDraftId, setActiveDraftId] = useState<string | undefined>(undefined);
  const [toastMessage, setToastMessage] = useState<string | null>(null);

  // Quick Reply
  const [quickReplyText, setQuickReplyText] = useState('');
  const [isSendingReply, setIsSendingReply] = useState(false);

  // Signatures
  const [signature, setSignature] = useState<string>('');

  // Fetch messages from backend API
  const fetchMessages = async (folder = currentFolder, query = searchQuery) => {
    if (!activeMailbox.id) return;
    try {
      setLoading(true);
      const params = new URLSearchParams({
        mailbox_id: activeMailbox.id,
        folder: folder === 'starred' ? '' : folder,
      });
      if (query.trim()) {
        params.set('q', query.trim());
      }

      const res = await apiFetch<WebmailMessage[]>(`/api/v1/webmail/messages?${params.toString()}`);
      if (res.success && res.data) {
        let list = res.data;
        if (folder === 'starred') {
          list = list.filter((m) => m.is_starred);
        }
        setMessages(list);

        // Update active message if currently selected
        if (selectedMessageId) {
          const stillThere = list.find((m) => m.id === selectedMessageId);
          if (!stillThere && list.length > 0) {
            handleSelectMessage(list[0].id);
          } else if (stillThere) {
            handleSelectMessage(stillThere.id);
          } else {
            setActiveMessage(null);
            setSelectedMessageId(null);
          }
        } else if (list.length > 0) {
          handleSelectMessage(list[0].id);
        } else {
          setActiveMessage(null);
          setSelectedMessageId(null);
        }
      }
    } catch (err) {
      console.error('Failed to fetch webmail messages:', err);
    } finally {
      setLoading(false);
      setRefreshing(false);
    }
  };

  // Fetch folder statistics
  const fetchCounts = async () => {
    if (!activeMailbox.id) return;
    try {
      // Fetch inbox & all folders
      const res = await apiFetch<WebmailMessage[]>(`/api/v1/webmail/messages?mailbox_id=${activeMailbox.id}`);
      if (res.success && res.data) {
        const counts: { [key: string]: number } = {
          inbox: 0,
          inboxUnread: 0,
          sent: 0,
          drafts: 0,
          starred: 0,
          spam: 0,
          trash: 0,
          archive: 0,
        };
        res.data.forEach((m) => {
          if (m.is_starred) counts.starred++;
          if (m.folder === 'inbox') {
            counts.inbox++;
            if (!m.is_read) counts.inboxUnread++;
          } else if (m.folder === 'sent') counts.sent++;
          else if (m.folder === 'drafts') counts.drafts++;
          else if (m.folder === 'spam') counts.spam++;
          else if (m.folder === 'trash') counts.trash++;
          else if (m.folder === 'archive') counts.archive++;
        });
        setFolderCounts(counts);
      }
    } catch (err) {
      console.error('Failed to fetch folder counts:', err);
    }
  };

  // Fetch signature
  const fetchSignature = async () => {
    if (!activeMailbox.id) return;
    try {
      const res = await apiFetch<any[]>(`/api/v1/webmail/signatures?mailbox_id=${activeMailbox.id}`);
      if (res.success && res.data && res.data.length > 0) {
        const sig = res.data[0];
        setSignature(sig.content || '');
      }
    } catch {
      // Signature is optional
    }
  };

  useEffect(() => {
    if (activeMailbox.id) {
      fetchMessages(currentFolder, searchQuery);
      fetchCounts();
      fetchSignature();
    }
  }, [activeMailbox.id, currentFolder]);

  // Load message detail
  const handleSelectMessage = async (msgId: string) => {
    setSelectedMessageId(msgId);
    try {
      setLoadingDetail(true);
      const res = await apiFetch<WebmailMessage>(`/api/v1/webmail/messages/${msgId}`);
      if (res.success && res.data) {
        setActiveMessage(res.data);
        // Mark locally as read
        setMessages((prev) =>
          prev.map((m) => (m.id === msgId ? { ...m, is_read: true } : m))
        );
        fetchCounts();
      }
    } catch (err) {
      console.error('Failed to load message detail:', err);
    } finally {
      setLoadingDetail(false);
    }
  };

  // Toggle star
  const handleToggleStar = async (e: React.MouseEvent, msgId: string, currentStarred: boolean) => {
    e.stopPropagation();
    try {
      const res = await apiFetch<WebmailMessage>(`/api/v1/webmail/messages/${msgId}/flag`, {
        method: 'PUT',
        body: JSON.stringify({ is_starred: !currentStarred }),
      });
      if (res.success && res.data) {
        setMessages((prev) =>
          prev.map((m) => (m.id === msgId ? { ...m, is_starred: !currentStarred } : m))
        );
        if (activeMessage?.id === msgId) {
          setActiveMessage((prev) => (prev ? { ...prev, is_starred: !currentStarred } : null));
        }
        fetchCounts();
      }
    } catch (err) {
      console.error('Failed to toggle star:', err);
    }
  };

  // Toggle read status
  const handleToggleRead = async (msgId: string, currentRead: boolean) => {
    try {
      const res = await apiFetch<WebmailMessage>(`/api/v1/webmail/messages/${msgId}/flag`, {
        method: 'PUT',
        body: JSON.stringify({ is_read: !currentRead }),
      });
      if (res.success && res.data) {
        setMessages((prev) =>
          prev.map((m) => (m.id === msgId ? { ...m, is_read: !currentRead } : m))
        );
        if (activeMessage?.id === msgId) {
          setActiveMessage((prev) => (prev ? { ...prev, is_read: !currentRead } : null));
        }
        fetchCounts();
      }
    } catch (err) {
      console.error('Failed to toggle read status:', err);
    }
  };

  // Move message folder (trash, archive, spam, inbox)
  const handleMoveFolder = async (msgId: string, targetFolder: string) => {
    try {
      const res = await apiFetch<WebmailMessage>(`/api/v1/webmail/messages/${msgId}/folder`, {
        method: 'PUT',
        body: JSON.stringify({ folder: targetFolder }),
      });
      if (res.success) {
        setToastMessage(`Message moved to ${targetFolder}`);
        setTimeout(() => setToastMessage(null), 3000);
        // Refresh list and counts
        fetchMessages();
        fetchCounts();
      }
    } catch (err) {
      console.error('Failed to move message:', err);
    }
  };

  // Permanently delete message if already in trash
  const handlePermanentDelete = async (msgId: string) => {
    if (!confirm('Permanently delete this message? This action cannot be undone.')) return;
    try {
      const res = await apiFetch(`/api/v1/webmail/messages/${msgId}`, { method: 'DELETE' });
      if (res.success) {
        setToastMessage('Message permanently deleted');
        setTimeout(() => setToastMessage(null), 3000);
        fetchMessages();
        fetchCounts();
      }
    } catch (err) {
      console.error('Failed to delete message:', err);
    }
  };

  // Send message via real Postfix MTA
  const handleSendMessage = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!composeTo.trim()) {
      alert('Please enter at least one recipient email address.');
      return;
    }

    const recipients = composeTo
      .split(/[,;]/)
      .map((s) => s.trim())
      .filter(Boolean);

    const ccList = composeCc
      .split(/[,;]/)
      .map((s) => s.trim())
      .filter(Boolean);

    const bccList = composeBcc
      .split(/[,;]/)
      .map((s) => s.trim())
      .filter(Boolean);

    try {
      setIsSending(true);
      const res = await apiFetch<WebmailMessage>('/api/v1/webmail/messages/send', {
        method: 'POST',
        body: JSON.stringify({
          mailbox_id: activeMailbox.id,
          to: recipients,
          cc: ccList,
          bcc: bccList,
          subject: composeSubject.trim() || '(No Subject)',
          body_text: isHtmlMode ? '' : composeBody,
          body_html: isHtmlMode ? composeBody : '',
        }),
      });

      if (res.success && res.data) {
        setToastMessage(`Email delivered via Postfix MTA to ${recipients.join(', ')}`);
        setTimeout(() => setToastMessage(null), 4000);
        setShowComposeModal(false);
        setComposeTo('');
        setComposeCc('');
        setComposeBcc('');
        setComposeSubject('');
        setComposeBody('');
        setActiveDraftId(undefined);
        fetchCounts();
        if (currentFolder === 'sent') {
          fetchMessages('sent');
        }
      }
    } catch (err: any) {
      console.error('Failed to send email:', err);
      alert(`Error sending email: ${err.message || 'SMTP delivery rejected by Postfix'}`);
    } finally {
      setIsSending(false);
    }
  };

  // Save draft
  const handleSaveDraft = async () => {
    if (!composeSubject && !composeBody && !composeTo) return;
    try {
      setIsSavingDraft(true);
      const recipients = composeTo
        .split(/[,;]/)
        .map((s) => s.trim())
        .filter(Boolean);

      const res = await apiFetch<WebmailMessage>('/api/v1/webmail/messages/draft', {
        method: 'POST',
        body: JSON.stringify({
          id: activeDraftId,
          mailbox_id: activeMailbox.id,
          to: recipients,
          subject: composeSubject || '(Draft)',
          body_text: composeBody,
        }),
      });

      if (res.success && res.data) {
        setActiveDraftId(res.data.id);
        setToastMessage('Draft autosaved to Maildir');
        setTimeout(() => setToastMessage(null), 2500);
        fetchCounts();
      }
    } catch (err) {
      console.error('Failed to save draft:', err);
    } finally {
      setIsSavingDraft(false);
    }
  };

  // Quick reply
  const handleSendQuickReply = async () => {
    if (!activeMessage || !quickReplyText.trim()) return;
    try {
      setIsSendingReply(true);
      const res = await apiFetch<WebmailMessage>('/api/v1/webmail/messages/send', {
        method: 'POST',
        body: JSON.stringify({
          mailbox_id: activeMailbox.id,
          to: [activeMessage.from_email],
          subject: activeMessage.subject.startsWith('Re:') ? activeMessage.subject : `Re: ${activeMessage.subject}`,
          body_text: quickReplyText.trim(),
        }),
      });

      if (res.success) {
        setQuickReplyText('');
        setToastMessage(`Reply delivered via Postfix to ${activeMessage.from_email}`);
        setTimeout(() => setToastMessage(null), 3000);
        fetchCounts();
      }
    } catch (err: any) {
      alert(`Failed to send reply: ${err.message || 'SMTP Error'}`);
    } finally {
      setIsSendingReply(false);
    }
  };

  // Open Reply Modal
  const handleOpenReplyModal = (msg: WebmailMessage) => {
    setComposeTo(msg.from_email);
    setComposeCc('');
    setComposeBcc('');
    setComposeSubject(msg.subject.startsWith('Re:') ? msg.subject : `Re: ${msg.subject}`);
    const quote = `\n\n--- On ${new Date(msg.created_at).toLocaleString()}, ${msg.from_name} <${msg.from_email}> wrote: ---\n> ${msg.body_text.replace(/\n/g, '\n> ')}`;
    setComposeBody(signature ? `${quote}\n\n-- \n${signature}` : quote);
    setActiveDraftId(undefined);
    setShowComposeModal(true);
  };

  // Open Forward Modal
  const handleOpenForwardModal = (msg: WebmailMessage) => {
    setComposeTo('');
    setComposeCc('');
    setComposeBcc('');
    setComposeSubject(msg.subject.startsWith('Fwd:') ? msg.subject : `Fwd: ${msg.subject}`);
    const forwardHeader = `\n\n---------- Forwarded message ---------\nFrom: ${msg.from_name} <${msg.from_email}>\nDate: ${new Date(msg.created_at).toLocaleString()}\nSubject: ${msg.subject}\nTo: ${msg.to_addresses.join(', ')}\n\n${msg.body_text}`;
    setComposeBody(forwardHeader);
    setActiveDraftId(undefined);
    setShowComposeModal(true);
  };

  const quotaPercent = Math.min(
    Math.round(((activeMailbox.used_bytes || 0) / (activeMailbox.quota_bytes || 1)) * 100),
    100
  );

  return (
    <div className="flex flex-col h-[calc(100vh-140px)] min-h-[640px] bg-white dark:bg-surface-950 border border-slate-200 dark:border-surface-800 rounded-2xl overflow-hidden shadow-xl relative">
      {/* Toast Notification */}
      {toastMessage && (
        <div className="absolute top-4 right-4 z-50 bg-indigo-600 text-white text-xs font-semibold px-4 py-2.5 rounded-xl shadow-xl shadow-indigo-600/30 flex items-center gap-2 animate-fadeIn border border-indigo-400/30">
          <Check className="w-4 h-4 text-emerald-300" />
          <span>{toastMessage}</span>
        </div>
      )}

      {/* Top Header Bar */}
      <div className="h-16 px-5 border-b border-slate-200 dark:border-surface-800 bg-slate-50 dark:bg-surface-900 flex items-center justify-between gap-4 select-none shrink-0">
        <div className="flex items-center gap-3">
          {showBackToSettings && onBackToEmailSettings && (
            <button
              onClick={onBackToEmailSettings}
              className="p-1.5 rounded-xl bg-white dark:bg-surface-800 hover:bg-slate-100 dark:hover:bg-surface-700 text-slate-700 dark:text-slate-300 border border-slate-200 dark:border-surface-700 mr-1 transition shadow-xs"
              title="Back to Email Hosting Admin"
            >
              <ArrowLeft className="w-4 h-4" />
            </button>
          )}
          <div className="w-9 h-9 rounded-xl bg-gradient-to-tr from-indigo-600 to-purple-600 flex items-center justify-center text-white shadow-md shadow-indigo-500/20">
            <Inbox className="w-5 h-5" />
          </div>
          <div>
            <div className="flex items-center gap-2">
              <h2 className="text-sm font-bold text-slate-900 dark:text-white tracking-tight">Hostvra Webmail Suite</h2>
              <span className="text-[10px] font-semibold uppercase px-2 py-0.5 rounded-full bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 border border-emerald-500/20">
                Live Postfix / Dovecot
              </span>
            </div>
            <p className="text-[11px] text-slate-500 dark:text-slate-400 flex items-center gap-1.5">
              <span>Mailbox:</span>
              <span className="font-mono text-indigo-600 dark:text-indigo-400 font-semibold">{activeMailbox.email}</span>
            </p>
          </div>
        </div>

        {/* Mailbox Selector & Actions */}
        <div className="flex items-center gap-3">
          {/* Active Account Switcher */}
          <div className="relative flex items-center">
            <label htmlFor="mailbox-select" className="sr-only">
              Select Mailbox
            </label>
            <div className="flex items-center bg-white dark:bg-surface-800 border border-slate-200 dark:border-surface-700 rounded-xl px-3 py-1.5 text-xs text-slate-700 dark:text-slate-200 shadow-xs">
              <User className="w-3.5 h-3.5 text-indigo-500 dark:text-indigo-400 mr-2 shrink-0" />
              <select
                id="mailbox-select"
                value={activeMailbox.id}
                onChange={(e) => {
                  setSelectedMailboxId(e.target.value);
                  setSelectedMessageId(null);
                  setActiveMessage(null);
                }}
                className="bg-transparent border-none text-xs text-slate-900 dark:text-white font-medium focus:outline-none cursor-pointer pr-4"
              >
                {mailboxes.map((mb) => (
                  <option key={mb.id} value={mb.id} className="bg-white dark:bg-surface-900 text-slate-900 dark:text-white">
                    {mb.email} ({mb.name})
                  </option>
                ))}
              </select>
            </div>
          </div>

          {/* Refresh Button */}
          <button
            onClick={() => {
              setRefreshing(true);
              fetchMessages();
              fetchCounts();
            }}
            disabled={refreshing}
            className="p-2 rounded-xl bg-white dark:bg-surface-800 hover:bg-slate-100 dark:hover:bg-surface-700 text-slate-600 dark:text-slate-300 border border-slate-200 dark:border-surface-700 transition shadow-xs"
            title="Refresh Mailbox"
          >
            <RefreshCw className={`w-3.5 h-3.5 ${refreshing ? 'animate-spin text-indigo-500' : ''}`} />
          </button>

          {/* Compose Button */}
          <button
            onClick={() => {
              setComposeTo('');
              setComposeCc('');
              setComposeBcc('');
              setComposeSubject('');
              setComposeBody(signature ? `\n\n-- \n${signature}` : '');
              setActiveDraftId(undefined);
              setShowComposeModal(true);
            }}
            className="inline-flex items-center gap-2 px-3.5 py-1.5 rounded-xl bg-indigo-600 hover:bg-indigo-500 text-white text-xs font-semibold shadow-md shadow-indigo-600/25 transition active:scale-95"
          >
            <Plus className="w-4 h-4" />
            <span>Compose</span>
          </button>
        </div>
      </div>

      {/* Main 3-Column Layout */}
      <div className="flex-1 flex overflow-hidden">
        {/* Column 1: Folder Navigation & Quota (Width: 220px) */}
        <div className="w-56 border-r border-slate-200 dark:border-surface-800 bg-slate-50/50 dark:bg-surface-900/60 p-3.5 flex flex-col justify-between shrink-0 select-none">
          <div className="space-y-4">
            {/* Quick Compose in sidebar */}
            <button
              onClick={() => {
                setComposeTo('');
                setComposeCc('');
                setComposeBcc('');
                setComposeSubject('');
                setComposeBody(signature ? `\n\n-- \n${signature}` : '');
                setActiveDraftId(undefined);
                setShowComposeModal(true);
              }}
              className="w-full py-2.5 px-3 rounded-xl bg-gradient-to-r from-indigo-600 to-indigo-500 hover:from-indigo-500 hover:to-indigo-400 text-white font-semibold text-xs flex items-center justify-center gap-2 shadow-lg shadow-indigo-600/20 transition"
            >
              <Plus className="w-4 h-4" />
              <span>New Message</span>
            </button>

            {/* Folder List */}
            <nav className="space-y-1">
              {/* Inbox */}
              <button
                onClick={() => setCurrentFolder('inbox')}
                className={`w-full flex items-center justify-between px-3 py-2 rounded-xl text-xs font-medium transition ${
                  currentFolder === 'inbox'
                    ? 'bg-indigo-50 dark:bg-indigo-950/40 text-indigo-700 dark:text-indigo-300 font-semibold'
                    : 'text-slate-600 dark:text-slate-400 hover:bg-slate-100 dark:hover:bg-surface-800'
                }`}
              >
                <div className="flex items-center gap-2.5">
                  <Inbox className="w-4 h-4 text-indigo-500" />
                  <span>Inbox</span>
                </div>
                {folderCounts.inboxUnread > 0 && (
                  <span className="px-1.5 py-0.5 rounded-full text-[10px] font-bold bg-indigo-600 text-white">
                    {folderCounts.inboxUnread}
                  </span>
                )}
              </button>

              {/* Starred */}
              <button
                onClick={() => setCurrentFolder('starred')}
                className={`w-full flex items-center justify-between px-3 py-2 rounded-xl text-xs font-medium transition ${
                  currentFolder === 'starred'
                    ? 'bg-amber-50 dark:bg-amber-950/30 text-amber-700 dark:text-amber-300 font-semibold'
                    : 'text-slate-600 dark:text-slate-400 hover:bg-slate-100 dark:hover:bg-surface-800'
                }`}
              >
                <div className="flex items-center gap-2.5">
                  <Star className="w-4 h-4 text-amber-500" />
                  <span>Starred</span>
                </div>
                {folderCounts.starred > 0 && (
                  <span className="text-[10px] font-medium text-amber-600 dark:text-amber-400">
                    {folderCounts.starred}
                  </span>
                )}
              </button>

              {/* Sent */}
              <button
                onClick={() => setCurrentFolder('sent')}
                className={`w-full flex items-center justify-between px-3 py-2 rounded-xl text-xs font-medium transition ${
                  currentFolder === 'sent'
                    ? 'bg-indigo-50 dark:bg-indigo-950/40 text-indigo-700 dark:text-indigo-300 font-semibold'
                    : 'text-slate-600 dark:text-slate-400 hover:bg-slate-100 dark:hover:bg-surface-800'
                }`}
              >
                <div className="flex items-center gap-2.5">
                  <Send className="w-4 h-4 text-blue-500" />
                  <span>Sent</span>
                </div>
                {folderCounts.sent > 0 && (
                  <span className="text-[10px] font-medium text-slate-500 dark:text-slate-400">
                    {folderCounts.sent}
                  </span>
                )}
              </button>

              {/* Drafts */}
              <button
                onClick={() => setCurrentFolder('drafts')}
                className={`w-full flex items-center justify-between px-3 py-2 rounded-xl text-xs font-medium transition ${
                  currentFolder === 'drafts'
                    ? 'bg-indigo-50 dark:bg-indigo-950/40 text-indigo-700 dark:text-indigo-300 font-semibold'
                    : 'text-slate-600 dark:text-slate-400 hover:bg-slate-100 dark:hover:bg-surface-800'
                }`}
              >
                <div className="flex items-center gap-2.5">
                  <FileText className="w-4 h-4 text-slate-400" />
                  <span>Drafts</span>
                </div>
                {folderCounts.drafts > 0 && (
                  <span className="text-[10px] font-medium text-slate-500 dark:text-slate-400">
                    {folderCounts.drafts}
                  </span>
                )}
              </button>

              {/* Archive */}
              <button
                onClick={() => setCurrentFolder('archive')}
                className={`w-full flex items-center justify-between px-3 py-2 rounded-xl text-xs font-medium transition ${
                  currentFolder === 'archive'
                    ? 'bg-indigo-50 dark:bg-indigo-950/40 text-indigo-700 dark:text-indigo-300 font-semibold'
                    : 'text-slate-600 dark:text-slate-400 hover:bg-slate-100 dark:hover:bg-surface-800'
                }`}
              >
                <div className="flex items-center gap-2.5">
                  <Archive className="w-4 h-4 text-slate-400" />
                  <span>Archive</span>
                </div>
                {folderCounts.archive > 0 && (
                  <span className="text-[10px] font-medium text-slate-500 dark:text-slate-400">
                    {folderCounts.archive}
                  </span>
                )}
              </button>

              {/* Spam */}
              <button
                onClick={() => setCurrentFolder('spam')}
                className={`w-full flex items-center justify-between px-3 py-2 rounded-xl text-xs font-medium transition ${
                  currentFolder === 'spam'
                    ? 'bg-rose-50 dark:bg-rose-950/30 text-rose-700 dark:text-rose-300 font-semibold'
                    : 'text-slate-600 dark:text-slate-400 hover:bg-slate-100 dark:hover:bg-surface-800'
                }`}
              >
                <div className="flex items-center gap-2.5">
                  <AlertOctagon className="w-4 h-4 text-rose-500" />
                  <span>Spam / Junk</span>
                </div>
                {folderCounts.spam > 0 && (
                  <span className="text-[10px] font-medium text-rose-600 dark:text-rose-400">
                    {folderCounts.spam}
                  </span>
                )}
              </button>

              {/* Trash */}
              <button
                onClick={() => setCurrentFolder('trash')}
                className={`w-full flex items-center justify-between px-3 py-2 rounded-xl text-xs font-medium transition ${
                  currentFolder === 'trash'
                    ? 'bg-slate-200 dark:bg-surface-800 text-slate-900 dark:text-white font-semibold'
                    : 'text-slate-600 dark:text-slate-400 hover:bg-slate-100 dark:hover:bg-surface-800'
                }`}
              >
                <div className="flex items-center gap-2.5">
                  <Trash2 className="w-4 h-4 text-slate-500" />
                  <span>Trash</span>
                </div>
                {folderCounts.trash > 0 && (
                  <span className="text-[10px] font-medium text-slate-500 dark:text-slate-400">
                    {folderCounts.trash}
                  </span>
                )}
              </button>
            </nav>
          </div>

          {/* Quota & Server Status Footer */}
          <div className="p-3 bg-white dark:bg-surface-950 border border-slate-200 dark:border-surface-800 rounded-xl space-y-2">
            <div className="flex items-center justify-between text-[11px]">
              <span className="text-slate-600 dark:text-slate-400 font-medium">Maildir Quota</span>
              <span className="text-slate-900 dark:text-white font-semibold font-mono">{quotaPercent}%</span>
            </div>
            <div className="w-full h-1.5 bg-slate-200 dark:bg-surface-800 rounded-full overflow-hidden">
              <div
                className={`h-full rounded-full transition-all ${
                  quotaPercent > 85 ? 'bg-rose-500' : quotaPercent > 65 ? 'bg-amber-500' : 'bg-indigo-600'
                }`}
                style={{ width: `${Math.max(quotaPercent, 2)}%` }}
              />
            </div>
            <div className="flex items-center justify-between text-[10px] text-slate-500">
              <span>{((activeMailbox.used_bytes || 0) / (1024 * 1024)).toFixed(1)} MB</span>
              <span>{((activeMailbox.quota_bytes || 5368709120) / (1024 * 1024 * 1024)).toFixed(0)} GB</span>
            </div>
          </div>
        </div>

        {/* Column 2: Message List (Width: 340px) */}
        <div className="w-80 md:w-96 border-r border-slate-200 dark:border-surface-800 bg-white dark:bg-surface-950 flex flex-col shrink-0">
          {/* Search Header */}
          <div className="p-3 border-b border-slate-200 dark:border-surface-800 bg-slate-50/50 dark:bg-surface-900/40">
            <div className="flex items-center gap-2 bg-white dark:bg-surface-950 border border-slate-200 dark:border-surface-800 rounded-xl px-3 py-2 text-xs">
              <Search className="w-3.5 h-3.5 text-slate-400 shrink-0" />
              <input
                type="text"
                placeholder="Search subject, sender, body..."
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                onKeyDown={(e) => {
                  if (e.key === 'Enter') {
                    fetchMessages(currentFolder, searchQuery);
                  }
                }}
                className="w-full bg-transparent text-slate-900 dark:text-white placeholder-slate-400 focus:outline-none text-xs"
              />
              {searchQuery && (
                <button
                  onClick={() => {
                    setSearchQuery('');
                    fetchMessages(currentFolder, '');
                  }}
                  className="text-slate-400 hover:text-slate-600"
                >
                  <X className="w-3 h-3" />
                </button>
              )}
            </div>
          </div>

          {/* Messages Scroll Area */}
          <div className="flex-1 overflow-y-auto divide-y divide-slate-100 dark:divide-surface-800/60">
            {loading ? (
              <div className="p-8 text-center text-slate-400 text-xs">
                <RefreshCw className="w-5 h-5 animate-spin mx-auto text-indigo-500 mb-2" />
                <span>Reading Maildir messages...</span>
              </div>
            ) : messages.length === 0 ? (
              <div className="p-8 text-center text-slate-400 text-xs space-y-2">
                <Inbox className="w-8 h-8 mx-auto text-slate-300 dark:text-slate-600 stroke-[1.5]" />
                <p className="font-semibold text-slate-600 dark:text-slate-300">No messages in {currentFolder}</p>
                <p className="text-[11px] text-slate-400">Incoming messages will appear here in real-time.</p>
              </div>
            ) : (
              messages.map((msg) => {
                const isSelected = selectedMessageId === msg.id;
                return (
                  <div
                    key={msg.id}
                    onClick={() => handleSelectMessage(msg.id)}
                    className={`p-3.5 cursor-pointer transition flex items-start gap-2.5 relative select-none ${
                      isSelected
                        ? 'bg-indigo-50/80 dark:bg-indigo-950/40 border-l-4 border-indigo-600'
                        : 'hover:bg-slate-50 dark:hover:bg-surface-900/60'
                    }`}
                  >
                    {/* Unread dot */}
                    {!msg.is_read && (
                      <span className="w-2 h-2 rounded-full bg-indigo-600 absolute top-4 left-1.5" />
                    )}

                    <div className="flex-1 min-w-0 pl-1">
                      <div className="flex items-center justify-between gap-1 mb-0.5">
                        <span
                          className={`text-xs truncate ${
                            !msg.is_read
                              ? 'font-bold text-slate-900 dark:text-white'
                              : 'font-medium text-slate-700 dark:text-slate-300'
                          }`}
                        >
                          {currentFolder === 'sent'
                            ? `To: ${msg.to_addresses.join(', ')}`
                            : msg.from_name || msg.from_email}
                        </span>
                        <span className="text-[10px] text-slate-400 shrink-0 font-mono">
                          {new Date(msg.created_at).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}
                        </span>
                      </div>

                      <p
                        className={`text-xs truncate ${
                          !msg.is_read
                            ? 'font-semibold text-slate-900 dark:text-slate-100'
                            : 'text-slate-600 dark:text-slate-400'
                        }`}
                      >
                        {msg.subject || '(No Subject)'}
                      </p>

                      <p className="text-[11px] text-slate-400 truncate mt-0.5 line-clamp-1">
                        {msg.body_text || '(Empty text)'}
                      </p>

                      <div className="flex items-center justify-between mt-1.5 text-[10px]">
                        <div className="flex items-center gap-1.5 text-slate-400">
                          {msg.has_attachments && (
                            <span className="flex items-center gap-0.5 text-slate-500" title="Has attachments">
                              <Paperclip className="w-3 h-3" />
                            </span>
                          )}
                          <span className="font-mono text-[9px]">
                            {msg.size_bytes > 1024
                              ? `${(msg.size_bytes / 1024).toFixed(0)} KB`
                              : `${msg.size_bytes} B`}
                          </span>
                        </div>

                        {/* Star Button */}
                        <button
                          onClick={(e) => handleToggleStar(e, msg.id, msg.is_starred)}
                          className="p-1 text-slate-400 hover:text-amber-500 transition"
                          title={msg.is_starred ? 'Unstar' : 'Star'}
                        >
                          <Star
                            className={`w-3.5 h-3.5 ${
                              msg.is_starred ? 'text-amber-500 fill-amber-500' : 'text-slate-300 dark:text-slate-600'
                            }`}
                          />
                        </button>
                      </div>
                    </div>
                  </div>
                );
              })
            )}
          </div>
        </div>

        {/* Column 3: Full Message Reader Pane */}
        <div className="flex-1 flex flex-col bg-white dark:bg-surface-950 overflow-hidden">
          {loadingDetail ? (
            <div className="flex-1 flex items-center justify-center text-slate-400 text-xs">
              <RefreshCw className="w-5 h-5 animate-spin mx-auto text-indigo-500 mb-2" />
              <span>Fetching email payload...</span>
            </div>
          ) : activeMessage ? (
            <div className="flex-1 flex flex-col overflow-hidden">
              {/* Message Header Action Bar */}
              <div className="h-14 px-6 border-b border-slate-200 dark:border-surface-800 bg-slate-50/50 dark:bg-surface-900/40 flex items-center justify-between shrink-0">
                <div className="flex items-center gap-2">
                  <button
                    onClick={() => handleOpenReplyModal(activeMessage)}
                    className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-white dark:bg-surface-800 hover:bg-slate-100 dark:hover:bg-surface-700 text-slate-700 dark:text-slate-200 text-xs font-medium border border-slate-200 dark:border-surface-700 transition shadow-xs"
                    title="Reply"
                  >
                    <Reply className="w-3.5 h-3.5 text-indigo-500" />
                    <span>Reply</span>
                  </button>

                  <button
                    onClick={() => handleOpenForwardModal(activeMessage)}
                    className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-white dark:bg-surface-800 hover:bg-slate-100 dark:hover:bg-surface-700 text-slate-700 dark:text-slate-200 text-xs font-medium border border-slate-200 dark:border-surface-700 transition shadow-xs"
                    title="Forward"
                  >
                    <Forward className="w-3.5 h-3.5 text-indigo-500" />
                    <span>Forward</span>
                  </button>

                  <button
                    onClick={() => handleToggleRead(activeMessage.id, activeMessage.is_read)}
                    className="p-1.5 rounded-lg bg-white dark:bg-surface-800 hover:bg-slate-100 dark:hover:bg-surface-700 text-slate-600 dark:text-slate-300 text-xs border border-slate-200 dark:border-surface-700 transition shadow-xs"
                    title={activeMessage.is_read ? 'Mark as Unread' : 'Mark as Read'}
                  >
                    <CheckCheck className="w-3.5 h-3.5" />
                  </button>

                  {/* Move to Archive */}
                  <button
                    onClick={() => handleMoveFolder(activeMessage.id, 'archive')}
                    className="p-1.5 rounded-lg bg-white dark:bg-surface-800 hover:bg-slate-100 dark:hover:bg-surface-700 text-slate-600 dark:text-slate-300 text-xs border border-slate-200 dark:border-surface-700 transition shadow-xs"
                    title="Move to Archive"
                  >
                    <Archive className="w-3.5 h-3.5" />
                  </button>

                  {/* Move to Spam */}
                  <button
                    onClick={() => handleMoveFolder(activeMessage.id, 'spam')}
                    className="p-1.5 rounded-lg bg-white dark:bg-surface-800 hover:bg-rose-50 dark:hover:bg-rose-950/30 text-slate-600 dark:text-slate-300 hover:text-rose-600 text-xs border border-slate-200 dark:border-surface-700 transition shadow-xs"
                    title="Mark as Spam / Junk"
                  >
                    <AlertOctagon className="w-3.5 h-3.5" />
                  </button>

                  {/* Trash / Delete */}
                  {activeMessage.folder === 'trash' ? (
                    <button
                      onClick={() => handlePermanentDelete(activeMessage.id)}
                      className="p-1.5 rounded-lg bg-white dark:bg-surface-800 hover:bg-rose-500/20 text-rose-600 text-xs border border-rose-300 dark:border-rose-900 transition shadow-xs"
                      title="Delete Permanently"
                    >
                      <Trash2 className="w-3.5 h-3.5" />
                    </button>
                  ) : (
                    <button
                      onClick={() => handleMoveFolder(activeMessage.id, 'trash')}
                      className="p-1.5 rounded-lg bg-white dark:bg-surface-800 hover:bg-rose-50 dark:hover:bg-rose-950/30 text-slate-600 dark:text-slate-300 hover:text-rose-600 text-xs border border-slate-200 dark:border-surface-700 transition shadow-xs"
                      title="Move to Trash"
                    >
                      <Trash2 className="w-3.5 h-3.5" />
                    </button>
                  )}
                </div>

                <div className="flex items-center gap-3">
                  <span className="hidden sm:inline-flex items-center gap-1 text-[11px] px-2.5 py-1 rounded-full bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 border border-emerald-500/20 font-medium">
                    <ShieldCheck className="w-3.5 h-3.5" />
                    <span>DKIM & Postfix Verified</span>
                  </span>

                  <span className="text-xs text-slate-400 font-mono">
                    {new Date(activeMessage.created_at).toLocaleString()}
                  </span>
                </div>
              </div>

              {/* Message Details & Body Scroll */}
              <div className="flex-1 overflow-y-auto p-6 space-y-6">
                {/* Subject & Senders Info */}
                <div className="border-b border-slate-200 dark:border-surface-800 pb-5 space-y-4">
                  <div className="flex items-start justify-between gap-4">
                    <h1 className="text-xl font-bold text-slate-900 dark:text-white tracking-tight">
                      {activeMessage.subject || '(No Subject)'}
                    </h1>
                    <button
                      onClick={(e) => handleToggleStar(e, activeMessage.id, activeMessage.is_starred)}
                      className="p-1.5 rounded-lg hover:bg-slate-100 dark:hover:bg-surface-800 transition"
                      title={activeMessage.is_starred ? 'Unstar' : 'Star'}
                    >
                      <Star
                        className={`w-5 h-5 ${
                          activeMessage.is_starred
                            ? 'text-amber-500 fill-amber-500'
                            : 'text-slate-300 dark:text-slate-600'
                        }`}
                      />
                    </button>
                  </div>

                  <div className="flex items-center gap-3">
                    <div className="w-10 h-10 rounded-full bg-gradient-to-tr from-indigo-500 to-purple-600 text-white font-bold text-sm flex items-center justify-center shadow-md">
                      {(activeMessage.from_name || activeMessage.from_email).charAt(0).toUpperCase()}
                    </div>
                    <div className="flex-1 min-w-0 text-xs">
                      <div className="flex items-baseline gap-2">
                        <span className="font-bold text-slate-900 dark:text-white text-sm">
                          {activeMessage.from_name || activeMessage.from_email}
                        </span>
                        <span className="text-slate-500 font-mono text-xs">
                          &lt;{activeMessage.from_email}&gt;
                        </span>
                      </div>
                      <p className="text-slate-500 mt-0.5">
                        to{' '}
                        <span className="text-slate-700 dark:text-slate-300 font-mono">
                          {activeMessage.to_addresses.join(', ')}
                        </span>
                        {activeMessage.cc_addresses && activeMessage.cc_addresses.length > 0 && (
                          <span className="ml-2">
                            cc:{' '}
                            <span className="text-slate-600 dark:text-slate-400 font-mono">
                              {activeMessage.cc_addresses.join(', ')}
                            </span>
                          </span>
                        )}
                      </p>
                    </div>
                  </div>
                </div>

                {/* Body Content: Sandboxed HTML or Formatted Plain Text */}
                {activeMessage.body_html ? (
                  <div className="border border-slate-200 dark:border-surface-800 rounded-xl overflow-hidden bg-white min-h-[300px]">
                    <iframe
                      title="Email Content"
                      sandbox="allow-popups allow-popups-to-escape-sandbox"
                      srcDoc={activeMessage.body_html}
                      className="w-full min-h-[380px] border-none"
                    />
                  </div>
                ) : (
                  <div className="text-slate-800 dark:text-slate-200 text-sm leading-relaxed whitespace-pre-wrap font-sans max-w-3xl">
                    {activeMessage.body_text}
                  </div>
                )}

                {/* Attachments Section if present */}
                {activeMessage.has_attachments && activeMessage.attachments && activeMessage.attachments.length > 0 && (
                  <div className="pt-4 border-t border-slate-200 dark:border-surface-800 max-w-md">
                    <p className="text-xs font-semibold text-slate-500 uppercase tracking-wider mb-2">
                      Attachments ({activeMessage.attachments.length})
                    </p>
                    <div className="space-y-2">
                      {activeMessage.attachments.map((att) => (
                        <div
                          key={att.id}
                          className="p-3 rounded-xl bg-slate-50 dark:bg-surface-900 border border-slate-200 dark:border-surface-800 flex items-center justify-between text-xs"
                        >
                          <div className="flex items-center gap-2.5 truncate">
                            <Paperclip className="w-4 h-4 text-indigo-500 shrink-0" />
                            <span className="text-slate-800 dark:text-slate-200 font-medium truncate">
                              {att.filename}
                            </span>
                            <span className="text-slate-400 text-[11px]">
                              ({(att.size_bytes / 1024).toFixed(1)} KB)
                            </span>
                          </div>
                          <a
                            href={`/api/v1/webmail/attachments/${att.id}`}
                            download={att.filename}
                            className="text-xs text-indigo-600 dark:text-indigo-400 hover:underline font-semibold px-2 py-1"
                          >
                            Download
                          </a>
                        </div>
                      ))}
                    </div>
                  </div>
                )}
              </div>

              {/* Quick Reply Box */}
              <div className="p-4 border-t border-slate-200 dark:border-surface-800 bg-slate-50/50 dark:bg-surface-900/60 shrink-0">
                <div className="flex gap-2">
                  <input
                    type="text"
                    placeholder={`Reply to ${activeMessage.from_name || activeMessage.from_email}...`}
                    value={quickReplyText}
                    onChange={(e) => setQuickReplyText(e.target.value)}
                    onKeyDown={(e) => {
                      if (e.key === 'Enter' && !e.shiftKey) {
                        e.preventDefault();
                        handleSendQuickReply();
                      }
                    }}
                    className="flex-1 px-4 py-2.5 bg-white dark:bg-surface-950 border border-slate-200 dark:border-surface-800 rounded-xl text-xs text-slate-900 dark:text-white placeholder-slate-400 focus:outline-none focus:border-indigo-500"
                  />
                  <button
                    onClick={handleSendQuickReply}
                    disabled={!quickReplyText.trim() || isSendingReply}
                    className="px-4 py-2.5 rounded-xl bg-indigo-600 hover:bg-indigo-500 disabled:opacity-40 disabled:cursor-not-allowed text-white text-xs font-semibold flex items-center gap-1.5 shadow-md shadow-indigo-600/20 transition"
                  >
                    <Send className="w-3.5 h-3.5" />
                    <span>{isSendingReply ? 'Sending...' : 'Reply'}</span>
                  </button>
                </div>
              </div>
            </div>
          ) : (
            <div className="flex-1 flex flex-col items-center justify-center p-8 text-center text-slate-400 space-y-3">
              <div className="w-14 h-14 rounded-2xl bg-slate-100 dark:bg-surface-900 border border-slate-200 dark:border-surface-800 flex items-center justify-center text-slate-400 shadow-inner">
                <Inbox className="w-7 h-7 stroke-[1.5]" />
              </div>
              <div>
                <p className="text-sm font-semibold text-slate-700 dark:text-slate-300">Select an email to view</p>
                <p className="text-xs text-slate-400 mt-1 max-w-xs">
                  Choose a message from your mailbox or compose a new email to send via Postfix.
                </p>
              </div>
            </div>
          )}
        </div>
      </div>

      {/* Compose Email Modal */}
      {showComposeModal && (
        <div className="fixed inset-0 z-50 bg-black/75 backdrop-blur-sm flex items-center justify-center p-4 animate-fadeIn">
          <div className="bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-750 rounded-2xl w-full max-w-2xl shadow-2xl overflow-hidden flex flex-col max-h-[85vh]">
            {/* Modal Header */}
            <div className="h-14 px-5 border-b border-slate-200 dark:border-surface-800 bg-slate-50 dark:bg-surface-950/60 flex items-center justify-between">
              <div className="flex items-center gap-2">
                <Send className="w-4 h-4 text-indigo-500" />
                <h3 className="text-sm font-bold text-slate-900 dark:text-white tracking-tight">New Message</h3>
              </div>
              <div className="flex items-center gap-2">
                <button
                  type="button"
                  onClick={handleSaveDraft}
                  disabled={isSavingDraft}
                  className="px-2.5 py-1 text-xs font-medium text-slate-500 hover:text-slate-800 dark:hover:text-white"
                >
                  {isSavingDraft ? 'Saving...' : 'Save Draft'}
                </button>
                <button
                  onClick={() => setShowComposeModal(false)}
                  className="p-1 rounded-lg text-slate-400 hover:text-slate-700 dark:hover:text-white transition"
                >
                  ✕
                </button>
              </div>
            </div>

            {/* Compose Form */}
            <form onSubmit={handleSendMessage} className="p-5 space-y-4 flex-1 overflow-y-auto">
              {/* From Selector */}
              <div className="flex items-center gap-3 text-xs">
                <label className="w-16 font-semibold text-slate-500 dark:text-slate-400">From:</label>
                <div className="flex-1 px-3 py-2 bg-slate-50 dark:bg-surface-950 border border-slate-200 dark:border-surface-800 rounded-xl text-xs text-slate-900 dark:text-white font-mono">
                  {activeMailbox.name} &lt;{activeMailbox.email}&gt;
                </div>
              </div>

              {/* To input */}
              <div className="flex items-center gap-3 text-xs">
                <label className="w-16 font-semibold text-slate-500 dark:text-slate-400">To:</label>
                <input
                  type="text"
                  placeholder="recipient@example.com (comma separated)"
                  required
                  value={composeTo}
                  onChange={(e) => setComposeTo(e.target.value)}
                  className="flex-1 px-3 py-2 bg-white dark:bg-surface-950 border border-slate-200 dark:border-surface-800 rounded-xl text-xs text-slate-900 dark:text-white placeholder-slate-400 focus:outline-none focus:border-indigo-500"
                />
                <button
                  type="button"
                  onClick={() => setShowCcBcc(!showCcBcc)}
                  className="text-xs text-indigo-600 dark:text-indigo-400 hover:underline"
                >
                  {showCcBcc ? 'Hide Cc/Bcc' : 'Cc / Bcc'}
                </button>
              </div>

              {/* Cc / Bcc */}
              {showCcBcc && (
                <>
                  <div className="flex items-center gap-3 text-xs">
                    <label className="w-16 font-semibold text-slate-500 dark:text-slate-400">Cc:</label>
                    <input
                      type="text"
                      placeholder="cc@example.com"
                      value={composeCc}
                      onChange={(e) => setComposeCc(e.target.value)}
                      className="flex-1 px-3 py-2 bg-white dark:bg-surface-950 border border-slate-200 dark:border-surface-800 rounded-xl text-xs text-slate-900 dark:text-white placeholder-slate-400 focus:outline-none focus:border-indigo-500"
                    />
                  </div>
                  <div className="flex items-center gap-3 text-xs">
                    <label className="w-16 font-semibold text-slate-500 dark:text-slate-400">Bcc:</label>
                    <input
                      type="text"
                      placeholder="bcc@example.com"
                      value={composeBcc}
                      onChange={(e) => setComposeBcc(e.target.value)}
                      className="flex-1 px-3 py-2 bg-white dark:bg-surface-950 border border-slate-200 dark:border-surface-800 rounded-xl text-xs text-slate-900 dark:text-white placeholder-slate-400 focus:outline-none focus:border-indigo-500"
                    />
                  </div>
                </>
              )}

              {/* Subject */}
              <div className="flex items-center gap-3 text-xs">
                <label className="w-16 font-semibold text-slate-500 dark:text-slate-400">Subject:</label>
                <input
                  type="text"
                  placeholder="Subject line"
                  value={composeSubject}
                  onChange={(e) => setComposeSubject(e.target.value)}
                  className="flex-1 px-3 py-2 bg-white dark:bg-surface-950 border border-slate-200 dark:border-surface-800 rounded-xl text-xs text-slate-900 dark:text-white placeholder-slate-400 focus:outline-none focus:border-indigo-500 font-medium"
                />
              </div>

              {/* Message Body */}
              <div className="space-y-1">
                <div className="flex items-center justify-between text-xs text-slate-500 dark:text-slate-400 mb-1">
                  <span className="font-semibold">Message Body:</span>
                  <button
                    type="button"
                    onClick={() => setIsHtmlMode(!isHtmlMode)}
                    className="text-xs text-indigo-600 dark:text-indigo-400 hover:underline"
                  >
                    {isHtmlMode ? 'Switch to Plain Text' : 'Switch to Rich HTML'}
                  </button>
                </div>
                <textarea
                  rows={9}
                  placeholder={isHtmlMode ? '<p>Write HTML message here...</p>' : 'Write your email here...'}
                  value={composeBody}
                  onChange={(e) => setComposeBody(e.target.value)}
                  className="w-full px-3.5 py-2.5 bg-white dark:bg-surface-950 border border-slate-200 dark:border-surface-800 rounded-xl text-xs text-slate-900 dark:text-white placeholder-slate-400 focus:outline-none focus:border-indigo-500 font-sans leading-relaxed resize-none"
                />
              </div>

              {/* Footer / Send Buttons */}
              <div className="pt-2 flex items-center justify-between border-t border-slate-200 dark:border-surface-800">
                <div className="flex items-center gap-2 text-xs text-slate-500">
                  <span className="flex items-center gap-1 text-[11px] text-emerald-600 dark:text-emerald-400">
                    <ShieldCheck className="w-3.5 h-3.5" />
                    <span>Postfix TLS 1.3 Port 587</span>
                  </span>
                </div>

                <div className="flex items-center gap-2">
                  <button
                    type="button"
                    onClick={() => setShowComposeModal(false)}
                    className="px-4 py-2 rounded-xl text-xs font-medium text-slate-600 dark:text-slate-300 hover:bg-slate-100 dark:hover:bg-surface-800 transition"
                  >
                    Discard
                  </button>
                  <button
                    type="submit"
                    disabled={isSending}
                    className="px-5 py-2 rounded-xl bg-indigo-600 hover:bg-indigo-500 text-white text-xs font-semibold shadow-lg shadow-indigo-600/25 flex items-center gap-1.5 transition disabled:opacity-50"
                  >
                    <Send className="w-3.5 h-3.5" />
                    <span>{isSending ? 'Sending...' : 'Send Message'}</span>
                  </button>
                </div>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
}
