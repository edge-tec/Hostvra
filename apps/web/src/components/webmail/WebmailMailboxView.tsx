'use client';

import React, { useState, useEffect, useCallback, useMemo } from 'react';
import { useRouter } from 'next/navigation';
import {
  Inbox,
  Star,
  Trash2,
  Archive,
  AlertOctagon,
  Mail,
  MailOpen,
  Reply,
  ReplyAll,
  Forward,
  Printer,
  Download,
  Paperclip,
  CheckSquare,
  Square,
  RefreshCw,
  Search,
  ChevronLeft,
  ChevronRight,
  MoreVertical,
  ShieldCheck,
  Lock,
  ArrowLeft,
  Send,
  FileText,
  Clock,
  Sparkles,
  ExternalLink,
  ChevronDown,
  Layout,
  Columns2,
  Maximize2,
  Plus,
} from 'lucide-react';
import { useWebmail, WebmailAccount } from '@/context/WebmailContext';
import { apiFetch } from '@/lib/api';

export interface WebmailMessage {
  id: string;
  mailbox_id: string;
  account_email: string;
  folder: string;
  message_id?: string;
  from_name: string;
  from_email: string;
  to_name: string;
  to_email: string;
  cc?: string;
  bcc?: string;
  subject: string;
  snippet: string;
  body_text: string;
  body_html: string;
  is_unread: boolean;
  is_starred: boolean;
  is_important: boolean;
  has_attachment: boolean;
  priority: string;
  size_bytes: number;
  attachments?: Array<{
    id: string;
    filename: string;
    content_type: string;
    size_bytes: number;
  }>;
  created_at: string;
}

interface WebmailMailboxViewProps {
  folder: string; // inbox, starred, sent, drafts, archive, spam, trash
}

function formatBytes(bytes?: number): string {
  if (!bytes || bytes === 0) return '0 B';
  const k = 1024;
  const sizes = ['B', 'KB', 'MB', 'GB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i];
}

function formatDate(dateStr: string): string {
  if (!dateStr) return '';
  const d = new Date(dateStr);
  const now = new Date();
  const isToday =
    d.getDate() === now.getDate() &&
    d.getMonth() === now.getMonth() &&
    d.getFullYear() === now.getFullYear();

  if (isToday) {
    return d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
  }
  const isThisYear = d.getFullYear() === now.getFullYear();
  if (isThisYear) {
    return d.toLocaleDateString([], { month: 'short', day: 'numeric' });
  }
  return d.toLocaleDateString([], { year: 'numeric', month: 'numeric', day: 'numeric' });
}

function getAvatarColor(name: string): string {
  const colors = [
    'from-emerald-500 to-teal-600',
    'from-blue-500 to-indigo-600',
    'from-purple-500 to-pink-600',
    'from-amber-500 to-orange-600',
    'from-cyan-500 to-blue-600',
    'from-rose-500 to-red-600',
  ];
  let hash = 0;
  for (let i = 0; i < name.length; i++) {
    hash = name.charCodeAt(i) + ((hash << 5) - hash);
  }
  return colors[Math.abs(hash) % colors.length];
}

export function WebmailMailboxView({ folder }: WebmailMailboxViewProps) {
  const router = useRouter();
  const {
    activeAccount,
    accounts,
    searchQuery,
    refreshFolderCounts,
    openCompose,
    isSyncing,
    readingPaneLayout,
    setReadingPaneLayout,
    emailsPerPage,
    openAddAccount,
  } = useWebmail();

  const [messages, setMessages] = useState<WebmailMessage[]>([]);
  const [total, setTotal] = useState<number>(0);
  const [loading, setLoading] = useState<boolean>(true);
  const [page, setPage] = useState<number>(1);
  const pageSize = emailsPerPage || 50;

  // Selected message for reading pane
  const [selectedMessage, setSelectedMessage] = useState<WebmailMessage | null>(null);
  const [selectedIds, setSelectedIds] = useState<Set<string>>(new Set());

  // Quick reply box
  const [quickReplyText, setQuickReplyText] = useState('');
  const [sendingQuickReply, setSendingQuickReply] = useState(false);

  // Fetch messages from backend
  const fetchMessages = useCallback(async () => {
    const currentAccount = activeAccount || (accounts.length > 0 ? accounts[0] : null);
    if (!currentAccount) {
      setLoading(false);
      setMessages([]);
      setTotal(0);
      return;
    }
    setLoading(true);
    try {
      const offset = (page - 1) * pageSize;
      const params = new URLSearchParams({
        mailbox_id: currentAccount.id,
        account_email: currentAccount.email,
        folder: folder,
        limit: String(pageSize),
        offset: String(offset),
      });
      if (searchQuery.trim()) {
        params.append('search', searchQuery.trim());
      }

      const res = await apiFetch<{
        messages: WebmailMessage[];
        total: number;
      }>(`/api/v1/webmail/messages?${params.toString()}`);

      if (res.data) {
        const msgs = res.data.messages || [];
        setMessages(msgs);
        setTotal(res.data.total || msgs.length);
        setSelectedIds(new Set());
      }
    } catch (err) {
      console.error('Failed fetching messages:', err);
    } finally {
      setLoading(false);
    }
  }, [activeAccount, accounts, folder, page, pageSize, searchQuery]);

  useEffect(() => {
    fetchMessages();
  }, [fetchMessages]);

  // Open message and fetch full details
  const handleSelectMessage = async (msg: WebmailMessage) => {
    try {
      const res = await apiFetch<WebmailMessage>(`/api/v1/webmail/messages/${msg.id}`);
      if (res.data) {
        setSelectedMessage(res.data);
        setMessages((prev) =>
          prev.map((m) => (m.id === msg.id ? { ...m, is_unread: false } : m))
        );
        refreshFolderCounts();
      }
    } catch (e) {
      setSelectedMessage(msg);
    }
  };

  // Toggle selection checkbox
  const handleToggleSelect = (id: string, e: React.MouseEvent) => {
    e.stopPropagation();
    setSelectedIds((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  };

  // Select all on page
  const handleSelectAll = () => {
    if (selectedIds.size === messages.length) {
      setSelectedIds(new Set());
    } else {
      setSelectedIds(new Set(messages.map((m) => m.id)));
    }
  };

  // Star toggle
  const handleToggleStar = async (msg: WebmailMessage, e: React.MouseEvent) => {
    e.stopPropagation();
    const newStar = !msg.is_starred;
    setMessages((prev) =>
      prev.map((m) => (m.id === msg.id ? { ...m, is_starred: newStar } : m))
    );
    if (selectedMessage?.id === msg.id) {
      setSelectedMessage({ ...selectedMessage, is_starred: newStar });
    }
    try {
      await apiFetch(`/api/v1/webmail/messages/${msg.id}/flag`, {
        method: 'PUT',
        body: JSON.stringify({ is_starred: newStar }),
      });
      refreshFolderCounts();
    } catch (err) {
      console.error('Failed toggling star:', err);
    }
  };

  // Batch action dispatcher
  const handleBatchAction = async (action: string, targetFolder?: string) => {
    const ids = Array.from(selectedIds);
    if (ids.length === 0) return;

    try {
      await apiFetch('/api/v1/webmail/messages/batch', {
        method: 'POST',
        body: JSON.stringify({ ids, action, target_folder: targetFolder }),
      });
      await fetchMessages();
      refreshFolderCounts();
      if (selectedMessage && ids.includes(selectedMessage.id)) {
        setSelectedMessage(null);
      }
    } catch (err) {
      console.error('Batch action failed:', err);
    }
  };

  // Single message action
  const handleMessageAction = async (msgId: string, action: string, targetFolder?: string) => {
    try {
      if (action === 'delete') {
        await apiFetch(`/api/v1/webmail/messages/${msgId}`, { method: 'DELETE' });
      } else if (action === 'move' && targetFolder) {
        await apiFetch(`/api/v1/webmail/messages/${msgId}/folder`, {
          method: 'PUT',
          body: JSON.stringify({ target_folder: targetFolder }),
        });
      } else if (action === 'unread') {
        await apiFetch(`/api/v1/webmail/messages/${msgId}/flag`, {
          method: 'PUT',
          body: JSON.stringify({ is_unread: true }),
        });
      }
      setSelectedMessage(null);
      await fetchMessages();
      refreshFolderCounts();
    } catch (err) {
      console.error('Message action failed:', err);
    }
  };

  // Reply / Forward handlers
  const handleReply = () => {
    if (!selectedMessage) return;
    openCompose({
      to: selectedMessage.from_email,
      subject: selectedMessage.subject.startsWith('Re:') ? selectedMessage.subject : `Re: ${selectedMessage.subject}`,
      bodyHTML: `<br/><br/><blockquote style="border-left: 2px solid #ccc; padding-left: 8px; margin-left: 8px; color: #666;">On ${formatDate(selectedMessage.created_at)}, ${selectedMessage.from_name || selectedMessage.from_email} wrote:<br/>${selectedMessage.body_html || selectedMessage.body_text}</blockquote>`,
      replyToMessageId: selectedMessage.message_id,
    });
  };

  const handleReplyAll = () => {
    if (!selectedMessage) return;
    const allRecips = [selectedMessage.to_email, selectedMessage.cc].filter(Boolean).join(', ');
    openCompose({
      to: selectedMessage.from_email,
      cc: allRecips,
      subject: selectedMessage.subject.startsWith('Re:') ? selectedMessage.subject : `Re: ${selectedMessage.subject}`,
      bodyHTML: `<br/><br/><blockquote style="border-left: 2px solid #ccc; padding-left: 8px; margin-left: 8px; color: #666;">On ${formatDate(selectedMessage.created_at)}, ${selectedMessage.from_name || selectedMessage.from_email} wrote:<br/>${selectedMessage.body_html || selectedMessage.body_text}</blockquote>`,
      replyToMessageId: selectedMessage.message_id,
    });
  };

  const handleForward = () => {
    if (!selectedMessage) return;
    openCompose({
      subject: selectedMessage.subject.startsWith('Fwd:') ? selectedMessage.subject : `Fwd: ${selectedMessage.subject}`,
      bodyHTML: `<br/><br/>---------- Forwarded message ---------<br/>From: ${selectedMessage.from_name} &lt;${selectedMessage.from_email}&gt;<br/>Date: ${formatDate(selectedMessage.created_at)}<br/>Subject: ${selectedMessage.subject}<br/>To: ${selectedMessage.to_email}<br/><br/>${selectedMessage.body_html || selectedMessage.body_text}`,
      attachments: selectedMessage.attachments,
    });
  };

  // Quick reply submit
  const handleSendQuickReply = async () => {
    const sender = activeAccount || (accounts.length > 0 ? accounts[0] : null);
    if (!sender) {
      alert('Please connect an email mailbox first before sending a reply.');
      openAddAccount();
      return;
    }
    if (!selectedMessage || !quickReplyText.trim()) return;

    setSendingQuickReply(true);
    try {
      const payload = {
        mailbox_id: sender.id,
        account_email: sender.email,
        from_email: sender.email,
        to_email: selectedMessage.from_email,
        to: [selectedMessage.from_email],
        subject: selectedMessage.subject.startsWith('Re:') ? selectedMessage.subject : `Re: ${selectedMessage.subject}`,
        body_text: quickReplyText,
        body_html: `<p>${quickReplyText.replace(/\n/g, '<br/>')}</p>`,
      };
      await apiFetch('/api/v1/webmail/send', {
        method: 'POST',
        body: JSON.stringify(payload),
      });
      setQuickReplyText('');
      refreshFolderCounts();
      alert('Quick reply sent successfully.');
    } catch (e: unknown) {
      alert(e instanceof Error ? e.message : 'Failed to send quick reply');
    } finally {
      setSendingQuickReply(false);
    }
  };

  const folderTitle = useMemo(() => {
    switch (folder) {
      case 'inbox': return 'Inbox';
      case 'starred': return 'Starred';
      case 'sent': return 'Sent Mail';
      case 'drafts': return 'Drafts';
      case 'archive': return 'Archive';
      case 'spam': return 'Spam / Junk';
      case 'trash': return 'Trash';
      default: return folder.toUpperCase();
    }
  }, [folder]);

  const totalPages = Math.ceil(total / pageSize) || 1;
  const isFullLayout = readingPaneLayout === 'full';

  // If no account is active or present
  const hasAccounts = (accounts && accounts.length > 0) || activeAccount;

  return (
    <div className="flex-1 flex flex-col h-full overflow-hidden bg-slate-50">
      {/* Top Action Toolbar */}
      <div className="h-12 px-4 border-b border-slate-200 flex items-center justify-between gap-3 bg-white flex-shrink-0 shadow-2xs">
        <div className="flex items-center gap-2">
          {/* Back button when viewing email in Full layout or mobile */}
          {selectedMessage && (
            <button
              onClick={() => setSelectedMessage(null)}
              className="p-1.5 rounded-lg text-slate-600 hover:text-slate-900 hover:bg-slate-100 flex items-center gap-1.5 text-xs font-semibold mr-1 transition-colors"
              title="Back to email list"
            >
              <ArrowLeft className="w-4 h-4 text-emerald-600" />
              <span>Back to list</span>
            </button>
          )}

          {/* Select all checkbox */}
          {(!selectedMessage || !isFullLayout) && (
            <button
              onClick={handleSelectAll}
              className="p-1.5 rounded-lg text-slate-500 hover:text-slate-900 hover:bg-slate-100 transition-colors"
              title={selectedIds.size === messages.length && messages.length > 0 ? 'Deselect All' : 'Select All'}
            >
              {selectedIds.size > 0 && selectedIds.size === messages.length ? (
                <CheckSquare className="w-4 h-4 text-emerald-600" />
              ) : (
                <Square className="w-4 h-4 text-slate-400" />
              )}
            </button>
          )}

          {/* Batch Actions Toolbar when selected */}
          {selectedIds.size > 0 ? (
            <div className="flex items-center gap-1 animate-in fade-in duration-100">
              <span className="text-xs font-bold text-slate-700 mr-2">
                {selectedIds.size} selected
              </span>
              <button
                onClick={() => handleBatchAction('read')}
                className="p-1.5 rounded-lg text-slate-500 hover:text-slate-900 hover:bg-slate-100"
                title="Mark as read"
              >
                <MailOpen className="w-4 h-4" />
              </button>
              <button
                onClick={() => handleBatchAction('unread')}
                className="p-1.5 rounded-lg text-slate-500 hover:text-slate-900 hover:bg-slate-100"
                title="Mark as unread"
              >
                <Mail className="w-4 h-4" />
              </button>
              <button
                onClick={() => handleBatchAction('move', 'archive')}
                className="p-1.5 rounded-lg text-slate-500 hover:text-slate-900 hover:bg-slate-100"
                title="Move to Archive"
              >
                <Archive className="w-4 h-4" />
              </button>
              <button
                onClick={() => handleBatchAction('move', 'spam')}
                className="p-1.5 rounded-lg text-slate-500 hover:text-orange-500 hover:bg-slate-100"
                title="Report Spam"
              >
                <AlertOctagon className="w-4 h-4" />
              </button>
              <button
                onClick={() => handleBatchAction('delete')}
                className="p-1.5 rounded-lg text-slate-500 hover:text-rose-600 hover:bg-rose-50"
                title="Move to Trash"
              >
                <Trash2 className="w-4 h-4" />
              </button>
            </div>
          ) : (
            <div className="flex items-center gap-2">
              <button
                onClick={fetchMessages}
                disabled={isSyncing}
                className="p-1.5 rounded-lg text-slate-500 hover:text-slate-900 hover:bg-slate-100"
                title="Refresh folder"
              >
                <RefreshCw className={`w-4 h-4 ${isSyncing ? 'animate-spin text-emerald-600' : ''}`} />
              </button>
              <span className="text-xs font-bold text-slate-800">
                {folderTitle}
              </span>
            </div>
          )}
        </div>

        {/* Right: Layout Switcher & Pagination */}
        <div className="flex items-center gap-3 text-xs text-slate-500">
          {/* Quick Layout Mode Switcher */}
          <div className="hidden sm:flex items-center bg-slate-100 p-0.5 rounded-lg border border-slate-200">
            <button
              onClick={() => setReadingPaneLayout('split')}
              className={`flex items-center gap-1 px-2 py-1 rounded-md text-[11px] font-semibold transition-all ${
                readingPaneLayout === 'split'
                  ? 'bg-white text-emerald-700 shadow-xs'
                  : 'text-slate-500 hover:text-slate-800'
              }`}
              title="Split View: List on left, reading pane on right"
            >
              <Columns2 className="w-3.5 h-3.5" />
              <span>Split</span>
            </button>
            <button
              onClick={() => setReadingPaneLayout('full')}
              className={`flex items-center gap-1 px-2 py-1 rounded-md text-[11px] font-semibold transition-all ${
                readingPaneLayout === 'full'
                  ? 'bg-white text-emerald-700 shadow-xs'
                  : 'text-slate-500 hover:text-slate-800'
              }`}
              title="Full View: Full width message reading"
            >
              <Maximize2 className="w-3.5 h-3.5" />
              <span>Full</span>
            </button>
          </div>

          <span>
            {total > 0 ? (
              <>
                {(page - 1) * pageSize + 1}–{Math.min(page * pageSize, total)} of {total}
              </>
            ) : (
              '0 messages'
            )}
          </span>

          <div className="flex items-center gap-1">
            <button
              onClick={() => setPage((p) => Math.max(1, p - 1))}
              disabled={page <= 1}
              className="p-1 rounded-md hover:bg-slate-100 disabled:opacity-30"
              title="Previous Page"
            >
              <ChevronLeft className="w-4 h-4" />
            </button>
            <button
              onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
              disabled={page >= totalPages}
              className="p-1 rounded-md hover:bg-slate-100 disabled:opacity-30"
              title="Next Page"
            >
              <ChevronRight className="w-4 h-4" />
            </button>
          </div>
        </div>
      </div>

      {/* Main Mail View (Split or Full layout) */}
      <div className="flex-1 flex overflow-hidden">
        {/* Left: Message List */}
        <div
          className={`flex flex-col overflow-y-auto border-r border-slate-200 bg-white ${
            isFullLayout
              ? selectedMessage
                ? 'hidden'
                : 'w-full'
              : selectedMessage
              ? 'hidden lg:flex lg:w-5/12 lg:max-w-md'
              : 'w-full'
          }`}
        >
          {loading ? (
            <div className="p-8 text-center text-xs text-slate-400 flex flex-col items-center justify-center gap-2">
              <RefreshCw className="w-5 h-5 animate-spin text-emerald-500" />
              <span>Fetching {folderTitle.toLowerCase()}...</span>
            </div>
          ) : !hasAccounts ? (
            <div className="flex-1 flex flex-col items-center justify-center p-8 text-center bg-white">
              <div className="w-14 h-14 rounded-2xl bg-emerald-50 flex items-center justify-center mb-3 text-emerald-600 shadow-xs">
                <Mail className="w-7 h-7" />
              </div>
              <h3 className="text-sm font-bold text-slate-800">
                No Email Account Connected
              </h3>
              <p className="text-xs text-slate-500 mt-1.5 max-w-sm leading-relaxed">
                Connect your business mailbox or log in to view emails, send new messages, and manage folders.
              </p>
              <button
                onClick={openAddAccount}
                className="mt-4 flex items-center gap-2 px-4 py-2.5 bg-gradient-to-r from-emerald-600 to-teal-600 hover:from-emerald-500 hover:to-teal-500 text-white font-bold text-xs rounded-xl shadow-md shadow-emerald-600/20 transition-all hover:scale-[1.02] active:scale-[0.98]"
              >
                <Plus className="w-4 h-4" />
                <span>Connect Email Account</span>
              </button>
            </div>
          ) : messages.length === 0 ? (
            <div className="flex-1 flex flex-col items-center justify-center p-8 text-center text-slate-400 bg-white">
              <div className="w-12 h-12 rounded-2xl bg-slate-100 flex items-center justify-center mb-3">
                <Inbox className="w-6 h-6 text-slate-400" />
              </div>
              <p className="text-sm font-bold text-slate-700">
                No messages in {folderTitle}
              </p>
              <p className="text-xs text-slate-400 mt-1 max-w-xs">
                Your mailbox is up to date. Incoming emails synchronize automatically in real-time.
              </p>
            </div>
          ) : (
            <div className="divide-y divide-slate-100">
              {messages.map((msg) => {
                const isSelected = selectedMessage?.id === msg.id;
                const isChecked = selectedIds.has(msg.id);
                const senderInitial = (msg.from_name || msg.from_email || 'U').slice(0, 1).toUpperCase();
                const avatarGradient = getAvatarColor(msg.from_email);

                return (
                  <div
                    key={msg.id}
                    onClick={() => handleSelectMessage(msg)}
                    className={`flex items-center gap-3 px-4 py-3 cursor-pointer select-none transition-colors group relative ${
                      isSelected
                        ? 'bg-emerald-50/90 text-slate-900 font-medium'
                        : msg.is_unread
                        ? 'bg-slate-50/90 hover:bg-slate-100/90'
                        : 'bg-white hover:bg-slate-50'
                    }`}
                  >
                    {/* Left Accent Bar for Unread */}
                    {msg.is_unread && (
                      <div className="absolute left-0 top-0 bottom-0 w-1 bg-emerald-500 rounded-r" />
                    )}

                    {/* Checkbox */}
                    <button
                      onClick={(e) => handleToggleSelect(msg.id, e)}
                      className="p-1 text-slate-400 hover:text-slate-700"
                    >
                      {isChecked ? (
                        <CheckSquare className="w-3.5 h-3.5 text-emerald-600" />
                      ) : (
                        <Square className="w-3.5 h-3.5" />
                      )}
                    </button>

                    {/* Star */}
                    <button
                      onClick={(e) => handleToggleStar(msg, e)}
                      className={`p-1 transition-colors ${
                        msg.is_starred
                          ? 'text-amber-400 fill-amber-400'
                          : 'text-slate-300 hover:text-amber-400'
                      }`}
                      title={msg.is_starred ? 'Unstar' : 'Star'}
                    >
                      <Star className={`w-3.5 h-3.5 ${msg.is_starred ? 'fill-current' : ''}`} />
                    </button>

                    {/* Sender Avatar */}
                    <div
                      className={`w-7 h-7 rounded-lg bg-gradient-to-tr ${avatarGradient} text-white font-bold text-xs flex items-center justify-center flex-shrink-0 shadow-xs`}
                    >
                      {senderInitial}
                    </div>

                    {/* Sender & Subject & Snippet */}
                    <div className="min-w-0 flex-1">
                      <div className="flex items-center justify-between gap-1 mb-0.5">
                        <span
                          className={`text-xs truncate ${
                            msg.is_unread
                              ? 'font-bold text-slate-900'
                              : 'font-medium text-slate-700'
                          }`}
                        >
                          {folder === 'sent' ? `To: ${msg.to_name || msg.to_email}` : (msg.from_name || msg.from_email)}
                        </span>
                        <span className="text-[10px] text-slate-400 flex-shrink-0 font-medium">
                          {formatDate(msg.created_at)}
                        </span>
                      </div>

                      <div className="flex items-center gap-1.5">
                        <p
                          className={`text-xs truncate ${
                            msg.is_unread
                              ? 'font-semibold text-slate-900'
                              : 'text-slate-600'
                          }`}
                        >
                          {msg.subject || '(No subject)'}
                        </p>
                      </div>

                      <p className="text-[11px] text-slate-400 truncate mt-0.5 font-normal">
                        {msg.snippet || msg.body_text?.slice(0, 100) || '...'}
                      </p>
                    </div>

                    {/* Paperclip if has attachments */}
                    {msg.has_attachment && (
                      <Paperclip className="w-3.5 h-3.5 text-slate-400 flex-shrink-0" />
                    )}
                  </div>
                );
              })}
            </div>
          )}
        </div>

        {/* Right: Detailed Reading Pane */}
        {selectedMessage ? (
          <div className="flex-1 flex flex-col overflow-y-auto bg-white">
            {/* Message Action Header Bar */}
            <div className="p-4 border-b border-slate-200 flex items-center justify-between gap-2 flex-wrap bg-slate-50">
              <div className="flex items-center gap-1.5 flex-wrap">
                <button
                  onClick={handleReply}
                  className="flex items-center gap-1.5 px-3 py-1.5 bg-white border border-slate-200 rounded-xl text-xs font-semibold text-slate-700 hover:bg-slate-50 transition-colors shadow-2xs"
                >
                  <Reply className="w-3.5 h-3.5 text-emerald-600" />
                  <span>Reply</span>
                </button>
                <button
                  onClick={handleReplyAll}
                  className="flex items-center gap-1.5 px-3 py-1.5 bg-white border border-slate-200 rounded-xl text-xs font-semibold text-slate-700 hover:bg-slate-50 transition-colors shadow-2xs"
                >
                  <ReplyAll className="w-3.5 h-3.5" />
                  <span>Reply All</span>
                </button>
                <button
                  onClick={handleForward}
                  className="flex items-center gap-1.5 px-3 py-1.5 bg-white border border-slate-200 rounded-xl text-xs font-semibold text-slate-700 hover:bg-slate-50 transition-colors shadow-2xs"
                >
                  <Forward className="w-3.5 h-3.5" />
                  <span>Forward</span>
                </button>
              </div>

              <div className="flex items-center gap-1">
                {/* Print button */}
                <button
                  onClick={() => window.print()}
                  className="p-1.5 rounded-lg text-slate-400 hover:text-slate-700 hover:bg-slate-100"
                  title="Print email"
                >
                  <Printer className="w-4 h-4" />
                </button>

                {/* Download .eml */}
                <a
                  href={`/api/v1/webmail/messages/${selectedMessage.id}/eml`}
                  download={`${selectedMessage.subject || 'message'}.eml`}
                  className="p-1.5 rounded-lg text-slate-400 hover:text-slate-700 hover:bg-slate-100"
                  title="Download .eml format"
                >
                  <Download className="w-4 h-4" />
                </a>

                {/* Mark as unread */}
                <button
                  onClick={() => handleMessageAction(selectedMessage.id, 'unread')}
                  className="p-1.5 rounded-lg text-slate-400 hover:text-slate-700 hover:bg-slate-100"
                  title="Mark as unread"
                >
                  <Mail className="w-4 h-4" />
                </button>

                {/* Move to Archive */}
                <button
                  onClick={() => handleMessageAction(selectedMessage.id, 'move', 'archive')}
                  className="p-1.5 rounded-lg text-slate-400 hover:text-slate-700 hover:bg-slate-100"
                  title="Archive message"
                >
                  <Archive className="w-4 h-4" />
                </button>

                {/* Delete / Trash */}
                <button
                  onClick={() => handleMessageAction(selectedMessage.id, 'delete')}
                  className="p-1.5 rounded-lg text-slate-400 hover:text-rose-600 hover:bg-rose-50"
                  title="Move to Trash"
                >
                  <Trash2 className="w-4 h-4" />
                </button>
              </div>
            </div>

            {/* Email Meta Details */}
            <div className="p-6 border-b border-slate-100 bg-white">
              <h1 className="text-lg font-bold text-slate-900 mb-4">
                {selectedMessage.subject || '(No subject)'}
              </h1>

              <div className="flex items-start justify-between gap-4">
                <div className="flex items-start gap-3 min-w-0">
                  <div
                    className={`w-10 h-10 rounded-xl bg-gradient-to-tr ${getAvatarColor(
                      selectedMessage.from_email
                    )} text-white font-bold text-sm flex items-center justify-center flex-shrink-0 shadow-xs`}
                  >
                    {(selectedMessage.from_name || selectedMessage.from_email || 'U').slice(0, 1).toUpperCase()}
                  </div>
                  <div className="min-w-0">
                    <div className="flex items-center gap-2 flex-wrap">
                      <span className="font-bold text-xs text-slate-900">
                        {selectedMessage.from_name || selectedMessage.from_email}
                      </span>
                      <span className="text-[11px] text-slate-400 font-mono">
                        &lt;{selectedMessage.from_email}&gt;
                      </span>
                      <span className="inline-flex items-center gap-1 px-1.5 py-0.5 rounded text-[10px] font-bold bg-emerald-50 text-emerald-700 border border-emerald-200">
                        <ShieldCheck className="w-3 h-3" /> TLS / DKIM Verified
                      </span>
                    </div>

                    <div className="text-[11px] text-slate-400 mt-0.5">
                      <span>to: </span>
                      <span className="text-slate-700 font-medium">
                        {selectedMessage.to_name || selectedMessage.to_email}
                      </span>
                      {selectedMessage.cc && (
                        <span className="ml-2">
                          cc: <span className="text-slate-700">{selectedMessage.cc}</span>
                        </span>
                      )}
                    </div>
                  </div>
                </div>

                <div className="text-right flex-shrink-0">
                  <span className="text-xs text-slate-400 font-medium block">
                    {new Date(selectedMessage.created_at).toLocaleString([], {
                      dateStyle: 'medium',
                      timeStyle: 'short',
                    })}
                  </span>
                </div>
              </div>
            </div>

            {/* Email Body Content */}
            <div className="p-6 flex-1 text-slate-800 text-xs leading-relaxed overflow-x-auto bg-white">
              {selectedMessage.body_html ? (
                <div
                  className="prose max-w-none text-xs text-slate-800"
                  dangerouslySetInnerHTML={{ __html: selectedMessage.body_html }}
                />
              ) : (
                <pre className="font-sans whitespace-pre-wrap leading-relaxed text-xs text-slate-800">
                  {selectedMessage.body_text || '(No body content)'}
                </pre>
              )}
            </div>

            {/* Attachments Section if present */}
            {selectedMessage.attachments && selectedMessage.attachments.length > 0 && (
              <div className="p-6 border-t border-slate-100 bg-slate-50">
                <div className="flex items-center gap-2 mb-3 text-xs font-bold text-slate-700">
                  <Paperclip className="w-4 h-4 text-emerald-500" />
                  <span>Attachments ({selectedMessage.attachments.length})</span>
                </div>
                <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 gap-3">
                  {selectedMessage.attachments.map((att) => (
                    <a
                      key={att.id}
                      href={`/api/v1/webmail/attachments/${att.id}`}
                      download={att.filename}
                      className="p-3 bg-white border border-slate-200 rounded-xl hover:border-emerald-500 transition-colors flex items-center justify-between group shadow-2xs"
                    >
                      <div className="min-w-0 pr-2">
                        <p className="text-xs font-semibold text-slate-800 truncate group-hover:text-emerald-600">
                          {att.filename}
                        </p>
                        <p className="text-[10px] text-slate-400 mt-0.5">
                          {formatBytes(att.size_bytes)}
                        </p>
                      </div>
                      <Download className="w-4 h-4 text-slate-400 group-hover:text-emerald-500 flex-shrink-0" />
                    </a>
                  ))}
                </div>
              </div>
            )}

            {/* Quick Inline Reply Box */}
            <div className="p-6 border-t border-slate-200 bg-slate-50">
              <div className="flex items-center gap-2 mb-2 text-xs font-bold text-slate-700">
                <Reply className="w-3.5 h-3.5 text-emerald-500" />
                <span>Quick Reply to {selectedMessage.from_name || selectedMessage.from_email}</span>
              </div>
              <textarea
                value={quickReplyText}
                onChange={(e) => setQuickReplyText(e.target.value)}
                placeholder="Type your response here..."
                rows={3}
                className="w-full p-3 bg-white border border-slate-200 rounded-xl text-xs text-slate-900 focus:outline-none focus:border-emerald-500 placeholder:text-slate-400 resize-y"
              />
              <div className="flex items-center justify-between mt-2.5">
                <button
                  onClick={handleReply}
                  className="text-xs font-semibold text-emerald-600 hover:underline flex items-center gap-1"
                >
                  <ExternalLink className="w-3 h-3" /> Open Full Composer
                </button>
                <button
                  onClick={handleSendQuickReply}
                  disabled={sendingQuickReply || !quickReplyText.trim()}
                  className="flex items-center gap-1.5 px-4 py-2 bg-gradient-to-r from-emerald-600 to-teal-600 hover:from-emerald-500 hover:to-teal-500 text-white font-bold text-xs rounded-xl shadow-xs disabled:opacity-50 transition-all hover:scale-[1.02] active:scale-[0.98]"
                >
                  <Send className="w-3.5 h-3.5" />
                  <span>{sendingQuickReply ? 'Sending...' : 'Send Reply'}</span>
                </button>
              </div>
            </div>
          </div>
        ) : !isFullLayout ? (
          <div className="hidden lg:flex flex-1 flex-col items-center justify-center p-8 text-center text-slate-400 bg-slate-50/60">
            <div className="w-16 h-16 rounded-2xl bg-white border border-slate-200 flex items-center justify-center mb-3 text-slate-300 shadow-2xs">
              <Mail className="w-8 h-8 text-slate-400" />
            </div>
            <h3 className="text-sm font-bold text-slate-700">
              Select an email to read
            </h3>
            <p className="text-xs text-slate-400 mt-1 max-w-xs">
              Click any conversation on the left to view contents, download attachments, and reply.
            </p>
          </div>
        ) : null}
      </div>
    </div>
  );
}
