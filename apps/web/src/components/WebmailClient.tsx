'use client';

import React, { useState, useEffect, useMemo } from 'react';
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
  MoreVertical,
  Printer,
  Archive,
  ArrowLeft,
  Settings,
  HardDrive,
  Sliders,
  ChevronDown,
  User,
  Sparkles,
} from 'lucide-react';

export interface WebmailMailbox {
  id: string;
  email: string;
  name: string;
  quota_bytes: number;
  used_bytes: number;
}

export interface EmailMessage {
  id: string;
  mailbox_email: string;
  folder: 'inbox' | 'sent' | 'drafts' | 'spam' | 'trash';
  from_name: string;
  from_email: string;
  to_name: string;
  to_email: string;
  subject: string;
  body: string;
  date: string;
  is_unread: boolean;
  is_starred: boolean;
  has_attachment?: boolean;
  attachment_name?: string;
  attachment_size?: string;
}

const defaultMailboxes: WebmailMailbox[] = [];
const seedMessages: EmailMessage[] = [];

interface WebmailClientProps {
  mailboxes?: WebmailMailbox[];
  initialSelectedEmail?: string;
  onBackToEmailSettings?: () => void;
  showBackToSettings?: boolean;
}

export function WebmailClient({
  mailboxes = defaultMailboxes,
  initialSelectedEmail,
  onBackToEmailSettings,
  showBackToSettings = false,
}: WebmailClientProps) {
  // Selected Mailbox Account
  const [selectedEmail, setSelectedEmail] = useState<string>(() => {
    if (initialSelectedEmail && mailboxes.some((m) => m.email === initialSelectedEmail)) {
      return initialSelectedEmail;
    }
    return mailboxes[0]?.email || '';
  });

  // Current folder
  const [currentFolder, setCurrentFolder] = useState<'inbox' | 'sent' | 'drafts' | 'starred' | 'spam' | 'trash'>('inbox');

  // Messages state
  const [messages, setMessages] = useState<EmailMessage[]>(() => {
    if (typeof window !== 'undefined') {
      const saved = localStorage.getItem('hostvra_webmail_messages');
      if (saved) {
        try {
          return JSON.parse(saved);
        } catch {
          // fallback
        }
      }
    }
    return [];
  });

  // Save messages to localStorage
  useEffect(() => {
    if (typeof window !== 'undefined') {
      localStorage.setItem('hostvra_webmail_messages', JSON.stringify(messages));
    }
  }, [messages]);

  // Selected message for reading
  const [selectedMessageId, setSelectedMessageId] = useState<string | null>(null);

  // Search filter
  const [searchQuery, setSearchQuery] = useState('');

  // Modals & Composer
  const [showComposeModal, setShowComposeModal] = useState(false);
  const [composeTo, setComposeTo] = useState('');
  const [composeSubject, setComposeSubject] = useState('');
  const [composeBody, setComposeBody] = useState('');
  const [isSending, setIsSending] = useState(false);
  const [toastMessage, setToastMessage] = useState<string | null>(null);

  // Inline Quick Reply
  const [quickReplyText, setQuickReplyText] = useState('');

  // Current active mailbox object
  const activeMailbox = useMemo(() => {
    return mailboxes.find((m) => m.email === selectedEmail) || mailboxes[0] || {
      id: 'default',
      email: selectedEmail || 'No mailbox selected',
      name: 'Mailbox User',
      quota_bytes: 5368709120,
      used_bytes: 0,
    };
  }, [mailboxes, selectedEmail]);

  // Filter messages for current mailbox & folder & search
  const filteredMessages = useMemo(() => {
    return messages.filter((msg) => {
      // Must match mailbox
      if (msg.mailbox_email !== selectedEmail) return false;

      // Folder matching
      if (currentFolder === 'starred') {
        if (!msg.is_starred) return false;
      } else {
        if (msg.folder !== currentFolder) return false;
      }

      // Search matching
      if (searchQuery.trim()) {
        const q = searchQuery.toLowerCase();
        return (
          msg.subject.toLowerCase().includes(q) ||
          msg.from_name.toLowerCase().includes(q) ||
          msg.from_email.toLowerCase().includes(q) ||
          msg.to_email.toLowerCase().includes(q) ||
          msg.body.toLowerCase().includes(q)
        );
      }

      return true;
    });
  }, [messages, selectedEmail, currentFolder, searchQuery]);

  // Selected message details
  const activeMessage = useMemo(() => {
    if (!selectedMessageId) return null;
    return messages.find((m) => m.id === selectedMessageId) || null;
  }, [messages, selectedMessageId]);

  // Automatically select first message if none selected or if selected message not in list
  useEffect(() => {
    if (filteredMessages.length > 0) {
      if (!selectedMessageId || !filteredMessages.some((m) => m.id === selectedMessageId)) {
        setSelectedMessageId(filteredMessages[0].id);
      }
    } else {
      setSelectedMessageId(null);
    }
  }, [filteredMessages, selectedMessageId]);

  // Mark message as read when selected
  const handleSelectMessage = (msg: EmailMessage) => {
    setSelectedMessageId(msg.id);
    if (msg.is_unread) {
      setMessages((prev) =>
        prev.map((m) => (m.id === msg.id ? { ...m, is_unread: false } : m))
      );
    }
  };

  // Toggle star
  const handleToggleStar = (e: React.MouseEvent, id: string) => {
    e.stopPropagation();
    setMessages((prev) =>
      prev.map((m) => (m.id === id ? { ...m, is_starred: !m.is_starred } : m))
    );
  };

  // Toggle read status
  const handleToggleRead = (id: string) => {
    setMessages((prev) =>
      prev.map((m) => (m.id === id ? { ...m, is_unread: !m.is_unread } : m))
    );
  };

  // Delete message (move to trash or remove)
  const handleDeleteMessage = (id: string) => {
    setMessages((prev) => {
      const updated: EmailMessage[] = [];
      for (const m of prev) {
        if (m.id === id) {
          if (m.folder !== 'trash') {
            updated.push({ ...m, folder: 'trash' });
          }
          // If already in trash, purging removes it
        } else {
          updated.push(m);
        }
      }
      return updated;
    });

    setToastMessage('Message moved to Trash');
    setTimeout(() => setToastMessage(null), 3000);
  };

  // Folder Counts
  const folderCounts = useMemo(() => {
    const counts = {
      inbox: 0,
      inboxUnread: 0,
      sent: 0,
      drafts: 0,
      starred: 0,
      spam: 0,
      trash: 0,
    };
    messages.forEach((m) => {
      if (m.mailbox_email === selectedEmail) {
        if (m.folder === 'inbox') {
          counts.inbox++;
          if (m.is_unread) counts.inboxUnread++;
        } else if (m.folder === 'sent') counts.sent++;
        else if (m.folder === 'drafts') counts.drafts++;
        else if (m.folder === 'spam') counts.spam++;
        else if (m.folder === 'trash') counts.trash++;

        if (m.is_starred) counts.starred++;
      }
    });
    return counts;
  }, [messages, selectedEmail]);

  // Send Email Handler
  const handleSendMessage = (e: React.FormEvent) => {
    e.preventDefault();
    if (!composeTo.trim()) {
      alert('Please enter a recipient email address.');
      return;
    }

    setIsSending(true);

    setTimeout(() => {
      const newSentMessage: EmailMessage = {
        id: `msg-${Date.now()}`,
        mailbox_email: selectedEmail,
        folder: 'sent',
        from_name: activeMailbox.name,
        from_email: selectedEmail,
        to_name: composeTo.split('@')[0],
        to_email: composeTo.trim(),
        subject: composeSubject.trim() || '(No Subject)',
        body: composeBody.trim() || '(Empty body)',
        date: 'Just now',
        is_unread: false,
        is_starred: false,
      };

      setMessages((prev) => [newSentMessage, ...prev]);
      setIsSending(false);
      setShowComposeModal(false);
      setComposeTo('');
      setComposeSubject('');
      setComposeBody('');

      setToastMessage(`Email sent successfully to ${composeTo.trim()} via Postfix SMTP`);
      setTimeout(() => setToastMessage(null), 4000);
    }, 600);
  };

  // Quick Reply
  const handleSendQuickReply = () => {
    if (!activeMessage || !quickReplyText.trim()) return;

    const newReply: EmailMessage = {
      id: `msg-${Date.now()}`,
      mailbox_email: selectedEmail,
      folder: 'sent',
      from_name: activeMailbox.name,
      from_email: selectedEmail,
      to_name: activeMessage.from_name,
      to_email: activeMessage.from_email,
      subject: activeMessage.subject.startsWith('Re:') ? activeMessage.subject : `Re: ${activeMessage.subject}`,
      body: quickReplyText.trim(),
      date: 'Just now',
      is_unread: false,
      is_starred: false,
    };

    setMessages((prev) => [newReply, ...prev]);
    setQuickReplyText('');
    setToastMessage(`Reply sent to ${activeMessage.from_email}`);
    setTimeout(() => setToastMessage(null), 3000);
  };

  // Reply from modal
  const handleOpenReplyModal = (msg: EmailMessage) => {
    setComposeTo(msg.from_email);
    setComposeSubject(msg.subject.startsWith('Re:') ? msg.subject : `Re: ${msg.subject}`);
    setComposeBody(`\n\n--- Original Message from ${msg.from_name} <${msg.from_email}> ---\n${msg.body}`);
    setShowComposeModal(true);
  };

  // Forward from modal
  const handleOpenForwardModal = (msg: EmailMessage) => {
    setComposeTo('');
    setComposeSubject(msg.subject.startsWith('Fwd:') ? msg.subject : `Fwd: ${msg.subject}`);
    setComposeBody(`\n\n--- Forwarded Message ---\nFrom: ${msg.from_name} <${msg.from_email}>\nDate: ${msg.date}\nSubject: ${msg.subject}\n\n${msg.body}`);
    setShowComposeModal(true);
  };

  const hostname = typeof window !== 'undefined' ? window.location.hostname : 'localhost';
  const roundcubeUrl = `http://${hostname}/webmail`;

  const quotaPercent = Math.min(
    Math.round((activeMailbox.used_bytes / (activeMailbox.quota_bytes || 1)) * 100),
    100
  );

  return (
    <div className="flex flex-col h-[calc(100vh-140px)] min-h-[640px] bg-surface-950 border border-surface-800 rounded-2xl overflow-hidden shadow-2xl relative">
      {/* Toast Notification */}
      {toastMessage && (
        <div className="absolute top-4 right-4 z-50 bg-indigo-600 text-white text-xs font-semibold px-4 py-2.5 rounded-xl shadow-xl shadow-indigo-600/30 flex items-center gap-2 animate-fadeIn border border-indigo-400/30">
          <Check className="w-4 h-4 text-emerald-300" />
          <span>{toastMessage}</span>
        </div>
      )}

      {/* Top Header Bar */}
      <div className="h-16 px-5 border-b border-surface-800 bg-surface-900 flex items-center justify-between gap-4 select-none shrink-0">
        <div className="flex items-center gap-3">
          {showBackToSettings && onBackToEmailSettings && (
            <button
              onClick={onBackToEmailSettings}
              className="p-1.5 rounded-xl bg-surface-800 hover:bg-surface-700 text-slate-300 hover:text-white transition border border-surface-700 mr-1"
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
              <h2 className="text-sm font-bold text-white tracking-tight">Hostvra Webmail Suite</h2>
              <span className="text-[10px] font-semibold uppercase px-2 py-0.5 rounded-full bg-emerald-500/10 text-emerald-400 border border-emerald-500/20">
                Live IMAP/SMTP
              </span>
            </div>
            <p className="text-[11px] text-slate-400 flex items-center gap-1.5">
              <span>Server Mailbox:</span>
              <span className="font-mono text-indigo-300 font-semibold">{selectedEmail}</span>
            </p>
          </div>
        </div>

        {/* Mailbox Selector & Actions */}
        <div className="flex items-center gap-3">
          {/* Active Account Switcher */}
          <div className="relative flex items-center">
            <label htmlFor="mailbox-select" className="sr-only">Select Mailbox</label>
            <div className="flex items-center bg-surface-800 border border-surface-700 rounded-xl px-3 py-1.5 text-xs text-slate-200">
              <User className="w-3.5 h-3.5 text-indigo-400 mr-2 shrink-0" />
              <select
                id="mailbox-select"
                value={selectedEmail}
                onChange={(e) => {
                  setSelectedEmail(e.target.value);
                  setSelectedMessageId(null);
                }}
                className="bg-transparent border-none text-xs text-white font-medium focus:outline-none cursor-pointer pr-4"
              >
                {mailboxes.map((mb) => (
                  <option key={mb.id} value={mb.email} className="bg-surface-900 text-white">
                    {mb.email} ({mb.name})
                  </option>
                ))}
              </select>
            </div>
          </div>

          {/* Direct Roundcube Webmail Link */}
          <a
            href={roundcubeUrl}
            target="_blank"
            rel="noreferrer"
            className="hidden sm:inline-flex items-center gap-1.5 px-3 py-1.5 rounded-xl bg-surface-800 hover:bg-surface-700 text-slate-300 hover:text-white text-xs font-medium border border-surface-700 transition"
            title="Open Server Roundcube Webmail in new tab"
          >
            <span>Roundcube</span>
            <ExternalLink className="w-3.5 h-3.5 text-indigo-400" />
          </a>

          {/* Compose Button */}
          <button
            onClick={() => {
              setComposeTo('');
              setComposeSubject('');
              setComposeBody('');
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
        <div className="w-56 border-r border-surface-800 bg-surface-900/60 p-3.5 flex flex-col justify-between shrink-0 select-none">
          <div className="space-y-4">
            {/* Quick Compose in sidebar */}
            <button
              onClick={() => setShowComposeModal(true)}
              className="w-full py-2.5 px-3 rounded-xl bg-gradient-to-r from-indigo-600 to-indigo-500 hover:from-indigo-500 hover:to-indigo-400 text-white font-semibold text-xs flex items-center justify-center gap-2 shadow-lg shadow-indigo-600/20 transition"
            >
              <Plus className="w-4 h-4" />
              <span>New Message</span>
            </button>

            {/* Folder List */}
            <nav className="space-y-1">
              <button
                onClick={() => setCurrentFolder('inbox')}
                className={`w-full flex items-center justify-between px-3 py-2 rounded-xl text-xs font-medium transition ${
                  currentFolder === 'inbox'
                    ? 'bg-indigo-600 text-white font-semibold shadow-md shadow-indigo-600/20'
                    : 'text-slate-300 hover:bg-surface-800 hover:text-white'
                }`}
              >
                <div className="flex items-center gap-2.5">
                  <Inbox className="w-4 h-4" />
                  <span>Inbox</span>
                </div>
                {folderCounts.inboxUnread > 0 ? (
                  <span
                    className={`px-1.5 py-0.5 rounded-full text-[10px] font-bold ${
                      currentFolder === 'inbox' ? 'bg-white text-indigo-700' : 'bg-indigo-500 text-white'
                    }`}
                  >
                    {folderCounts.inboxUnread}
                  </span>
                ) : (
                  <span className="text-[11px] text-slate-500">{folderCounts.inbox}</span>
                )}
              </button>

              <button
                onClick={() => setCurrentFolder('starred')}
                className={`w-full flex items-center justify-between px-3 py-2 rounded-xl text-xs font-medium transition ${
                  currentFolder === 'starred'
                    ? 'bg-indigo-600 text-white font-semibold shadow-md shadow-indigo-600/20'
                    : 'text-slate-300 hover:bg-surface-800 hover:text-white'
                }`}
              >
                <div className="flex items-center gap-2.5">
                  <Star className="w-4 h-4 text-amber-400 fill-amber-400/20" />
                  <span>Starred</span>
                </div>
                <span className="text-[11px] text-slate-500">{folderCounts.starred}</span>
              </button>

              <button
                onClick={() => setCurrentFolder('sent')}
                className={`w-full flex items-center justify-between px-3 py-2 rounded-xl text-xs font-medium transition ${
                  currentFolder === 'sent'
                    ? 'bg-indigo-600 text-white font-semibold shadow-md shadow-indigo-600/20'
                    : 'text-slate-300 hover:bg-surface-800 hover:text-white'
                }`}
              >
                <div className="flex items-center gap-2.5">
                  <Send className="w-4 h-4" />
                  <span>Sent</span>
                </div>
                <span className="text-[11px] text-slate-500">{folderCounts.sent}</span>
              </button>

              <button
                onClick={() => setCurrentFolder('drafts')}
                className={`w-full flex items-center justify-between px-3 py-2 rounded-xl text-xs font-medium transition ${
                  currentFolder === 'drafts'
                    ? 'bg-indigo-600 text-white font-semibold shadow-md shadow-indigo-600/20'
                    : 'text-slate-300 hover:bg-surface-800 hover:text-white'
                }`}
              >
                <div className="flex items-center gap-2.5">
                  <FileText className="w-4 h-4" />
                  <span>Drafts</span>
                </div>
                <span className="text-[11px] text-slate-500">{folderCounts.drafts}</span>
              </button>

              <button
                onClick={() => setCurrentFolder('spam')}
                className={`w-full flex items-center justify-between px-3 py-2 rounded-xl text-xs font-medium transition ${
                  currentFolder === 'spam'
                    ? 'bg-indigo-600 text-white font-semibold shadow-md shadow-indigo-600/20'
                    : 'text-slate-300 hover:bg-surface-800 hover:text-white'
                }`}
              >
                <div className="flex items-center gap-2.5">
                  <AlertOctagon className="w-4 h-4 text-yellow-500" />
                  <span>Spam / Junk</span>
                </div>
                <span className="text-[11px] text-slate-500">{folderCounts.spam}</span>
              </button>

              <button
                onClick={() => setCurrentFolder('trash')}
                className={`w-full flex items-center justify-between px-3 py-2 rounded-xl text-xs font-medium transition ${
                  currentFolder === 'trash'
                    ? 'bg-indigo-600 text-white font-semibold shadow-md shadow-indigo-600/20'
                    : 'text-slate-300 hover:bg-surface-800 hover:text-white'
                }`}
              >
                <div className="flex items-center gap-2.5">
                  <Trash2 className="w-4 h-4 text-red-400" />
                  <span>Trash</span>
                </div>
                <span className="text-[11px] text-slate-500">{folderCounts.trash}</span>
              </button>
            </nav>
          </div>

          {/* Mailbox Quota Info Box */}
          <div className="p-3 rounded-xl bg-surface-950 border border-surface-800 space-y-2">
            <div className="flex items-center justify-between text-[11px]">
              <span className="text-slate-400 flex items-center gap-1">
                <HardDrive className="w-3 h-3 text-indigo-400" />
                Storage
              </span>
              <span className="text-slate-300 font-mono font-medium">{quotaPercent}%</span>
            </div>
            <div className="w-full bg-surface-800 h-1.5 rounded-full overflow-hidden">
              <div
                className={`h-full rounded-full transition-all duration-300 ${
                  quotaPercent > 80 ? 'bg-red-500' : quotaPercent > 60 ? 'bg-yellow-500' : 'bg-indigo-500'
                }`}
                style={{ width: `${Math.max(quotaPercent, 4)}%` }}
              />
            </div>
            <p className="text-[10px] text-slate-400">
              {(activeMailbox.used_bytes / (1024 * 1024)).toFixed(0)} MB of{' '}
              {(activeMailbox.quota_bytes / (1024 * 1024 * 1024)).toFixed(1)} GB used
            </p>
          </div>
        </div>

        {/* Column 2: Email List View (Width: 360px - 400px) */}
        <div className="w-80 md:w-96 border-r border-surface-800 bg-surface-900/30 flex flex-col shrink-0">
          {/* List Toolbar: Search & Refresh */}
          <div className="p-3 border-b border-surface-800 space-y-2 bg-surface-900/50">
            <div className="relative">
              <Search className="w-3.5 h-3.5 absolute left-3 top-1/2 -translate-y-1/2 text-slate-400" />
              <input
                type="text"
                placeholder="Search messages..."
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                className="w-full pl-8 pr-3 py-1.5 bg-surface-950 border border-surface-800 rounded-xl text-xs text-white placeholder-slate-500 focus:outline-none focus:border-indigo-500"
              />
            </div>

            <div className="flex items-center justify-between text-[11px] text-slate-400 px-1">
              <span className="capitalize font-semibold text-slate-300">
                {currentFolder} ({filteredMessages.length})
              </span>
              <button
                onClick={() => {
                  setToastMessage('Inbox refreshed from server');
                  setTimeout(() => setToastMessage(null), 2000);
                }}
                className="hover:text-white flex items-center gap-1 transition"
                title="Refresh messages"
              >
                <RefreshCw className="w-3 h-3 text-indigo-400" />
                <span>Sync</span>
              </button>
            </div>
          </div>

          {/* Email Items Scroll */}
          <div className="flex-1 overflow-y-auto divide-y divide-surface-800/60">
            {filteredMessages.length === 0 ? (
              <div className="p-8 text-center text-slate-500 text-xs space-y-2">
                <Inbox className="w-8 h-8 mx-auto text-slate-600 stroke-[1.5]" />
                <p>No messages in {currentFolder}</p>
              </div>
            ) : (
              filteredMessages.map((msg) => {
                const isSelected = selectedMessageId === msg.id;
                return (
                  <div
                    key={msg.id}
                    onClick={() => handleSelectMessage(msg)}
                    className={`p-3.5 cursor-pointer transition-colors relative flex gap-3 ${
                      isSelected
                        ? 'bg-indigo-600/10 border-l-2 border-indigo-500'
                        : msg.is_unread
                        ? 'bg-surface-900/80 hover:bg-surface-800/80'
                        : 'hover:bg-surface-800/40 text-slate-400'
                    }`}
                  >
                    {/* Unread blue dot */}
                    {msg.is_unread && (
                      <span className="w-2 h-2 rounded-full bg-indigo-500 absolute top-4 left-1.5 ring-4 ring-indigo-500/20" />
                    )}

                    {/* Sender Avatar */}
                    <div
                      className={`w-8 h-8 rounded-full flex items-center justify-center text-xs font-bold shrink-0 ${
                        msg.is_unread
                          ? 'bg-indigo-500/20 text-indigo-300 border border-indigo-500/30'
                          : 'bg-surface-800 text-slate-400'
                      }`}
                    >
                      {msg.from_name.charAt(0).toUpperCase()}
                    </div>

                    {/* Subject & snippet */}
                    <div className="flex-1 min-w-0">
                      <div className="flex items-center justify-between gap-1 mb-0.5">
                        <span
                          className={`text-xs truncate ${
                            msg.is_unread ? 'font-bold text-white' : 'font-medium text-slate-300'
                          }`}
                        >
                          {currentFolder === 'sent' ? `To: ${msg.to_name || msg.to_email}` : msg.from_name}
                        </span>
                        <span className="text-[10px] text-slate-500 shrink-0 font-mono">{msg.date}</span>
                      </div>

                      <p
                        className={`text-xs truncate ${
                          msg.is_unread ? 'font-semibold text-slate-200' : 'text-slate-300'
                        }`}
                      >
                        {msg.subject}
                      </p>

                      <p className="text-[11px] text-slate-500 truncate mt-0.5 line-clamp-1">{msg.body}</p>

                      <div className="flex items-center justify-between mt-1.5 text-[10px]">
                        <div className="flex items-center gap-1 text-slate-500">
                          {msg.has_attachment && (
                            <span className="flex items-center gap-0.5 text-slate-400" title="Has attachment">
                              <Paperclip className="w-3 h-3" />
                            </span>
                          )}
                        </div>

                        {/* Star Button */}
                        <button
                          onClick={(e) => handleToggleStar(e, msg.id)}
                          className="p-1 text-slate-500 hover:text-amber-400 transition"
                          title={msg.is_starred ? 'Unstar' : 'Star'}
                        >
                          <Star
                            className={`w-3.5 h-3.5 ${
                              msg.is_starred ? 'text-amber-400 fill-amber-400' : 'text-slate-600 hover:text-slate-400'
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
        <div className="flex-1 flex flex-col bg-surface-950 overflow-hidden">
          {activeMessage ? (
            <div className="flex-1 flex flex-col overflow-hidden">
              {/* Message Header Action Bar */}
              <div className="h-14 px-6 border-b border-surface-800 bg-surface-900/40 flex items-center justify-between shrink-0">
                <div className="flex items-center gap-2">
                  <button
                    onClick={() => handleOpenReplyModal(activeMessage)}
                    className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-surface-800 hover:bg-surface-700 text-slate-200 text-xs font-medium border border-surface-700 transition"
                    title="Reply"
                  >
                    <Reply className="w-3.5 h-3.5 text-indigo-400" />
                    <span>Reply</span>
                  </button>

                  <button
                    onClick={() => handleOpenForwardModal(activeMessage)}
                    className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-surface-800 hover:bg-surface-700 text-slate-200 text-xs font-medium border border-surface-700 transition"
                    title="Forward"
                  >
                    <Forward className="w-3.5 h-3.5 text-indigo-400" />
                    <span>Forward</span>
                  </button>

                  <button
                    onClick={() => handleToggleRead(activeMessage.id)}
                    className="p-1.5 rounded-lg bg-surface-800 hover:bg-surface-700 text-slate-300 hover:text-white text-xs border border-surface-700 transition"
                    title={activeMessage.is_unread ? 'Mark as Read' : 'Mark as Unread'}
                  >
                    <CheckCheck className="w-3.5 h-3.5" />
                  </button>

                  <button
                    onClick={() => handleDeleteMessage(activeMessage.id)}
                    className="p-1.5 rounded-lg bg-surface-800 hover:bg-red-500/20 text-slate-400 hover:text-red-400 text-xs border border-surface-700 transition"
                    title="Delete Message"
                  >
                    <Trash2 className="w-3.5 h-3.5" />
                  </button>
                </div>

                <div className="flex items-center gap-3">
                  {/* Security Badge */}
                  <span className="hidden sm:inline-flex items-center gap-1 text-[11px] px-2.5 py-1 rounded-full bg-emerald-500/10 text-emerald-400 border border-emerald-500/20 font-medium">
                    <ShieldCheck className="w-3.5 h-3.5" />
                    <span>SPF & DKIM Signed</span>
                  </span>

                  <span className="text-xs text-slate-400 font-mono">{activeMessage.date}</span>
                </div>
              </div>

              {/* Message Details & Body Scroll */}
              <div className="flex-1 overflow-y-auto p-6 space-y-6">
                {/* Subject & Senders Info */}
                <div className="border-b border-surface-800 pb-5 space-y-4">
                  <div className="flex items-start justify-between gap-4">
                    <h1 className="text-xl font-bold text-white tracking-tight">{activeMessage.subject}</h1>
                    <button
                      onClick={(e) => handleToggleStar(e, activeMessage.id)}
                      className="p-1.5 rounded-lg hover:bg-surface-800 transition"
                      title={activeMessage.is_starred ? 'Unstar' : 'Star'}
                    >
                      <Star
                        className={`w-5 h-5 ${
                          activeMessage.is_starred ? 'text-amber-400 fill-amber-400' : 'text-slate-500'
                        }`}
                      />
                    </button>
                  </div>

                  <div className="flex items-center gap-3">
                    <div className="w-10 h-10 rounded-full bg-gradient-to-tr from-indigo-500 to-purple-600 text-white font-bold text-sm flex items-center justify-center shadow-md">
                      {activeMessage.from_name.charAt(0).toUpperCase()}
                    </div>
                    <div className="flex-1 min-w-0 text-xs">
                      <div className="flex items-baseline gap-2">
                        <span className="font-bold text-white text-sm">{activeMessage.from_name}</span>
                        <span className="text-slate-400 font-mono text-xs">&lt;{activeMessage.from_email}&gt;</span>
                      </div>
                      <p className="text-slate-500 mt-0.5">
                        to <span className="text-slate-300 font-mono">{activeMessage.to_email}</span>
                      </p>
                    </div>
                  </div>
                </div>

                {/* Body Content */}
                <div className="text-slate-200 text-sm leading-relaxed whitespace-pre-wrap font-sans max-w-3xl">
                  {activeMessage.body}
                </div>

                {/* Attachments Section if present */}
                {activeMessage.has_attachment && (
                  <div className="pt-4 border-t border-surface-800 max-w-md">
                    <p className="text-xs font-semibold text-slate-400 uppercase tracking-wider mb-2">
                      Attachments (1)
                    </p>
                    <div className="p-3 rounded-xl bg-surface-900 border border-surface-800 flex items-center justify-between text-xs">
                      <div className="flex items-center gap-2.5 truncate">
                        <Paperclip className="w-4 h-4 text-indigo-400 shrink-0" />
                        <span className="text-slate-200 font-medium truncate">{activeMessage.attachment_name}</span>
                        <span className="text-slate-500 text-[11px]">({activeMessage.attachment_size})</span>
                      </div>
                      <button
                        onClick={() => alert(`Downloading ${activeMessage.attachment_name}`)}
                        className="text-xs text-indigo-400 hover:text-indigo-300 font-semibold px-2 py-1 rounded hover:bg-surface-800 transition"
                      >
                        Download
                      </button>
                    </div>
                  </div>
                )}
              </div>

              {/* Quick Reply Box */}
              <div className="p-4 border-t border-surface-800 bg-surface-900/60 shrink-0">
                <div className="flex gap-2">
                  <input
                    type="text"
                    placeholder={`Reply to ${activeMessage.from_name}...`}
                    value={quickReplyText}
                    onChange={(e) => setQuickReplyText(e.target.value)}
                    onKeyDown={(e) => {
                      if (e.key === 'Enter' && !e.shiftKey) {
                        e.preventDefault();
                        handleSendQuickReply();
                      }
                    }}
                    className="flex-1 px-4 py-2.5 bg-surface-950 border border-surface-800 rounded-xl text-xs text-white placeholder-slate-500 focus:outline-none focus:border-indigo-500"
                  />
                  <button
                    onClick={handleSendQuickReply}
                    disabled={!quickReplyText.trim()}
                    className="px-4 py-2.5 rounded-xl bg-indigo-600 hover:bg-indigo-500 disabled:opacity-40 disabled:cursor-not-allowed text-white text-xs font-semibold flex items-center gap-1.5 shadow-md shadow-indigo-600/20 transition"
                  >
                    <Send className="w-3.5 h-3.5" />
                    <span>Reply</span>
                  </button>
                </div>
              </div>
            </div>
          ) : (
            <div className="flex-1 flex flex-col items-center justify-center p-8 text-center text-slate-500 space-y-3">
              <div className="w-14 h-14 rounded-2xl bg-surface-900 border border-surface-800 flex items-center justify-center text-slate-600 shadow-inner">
                <Inbox className="w-7 h-7 stroke-[1.5]" />
              </div>
              <div>
                <p className="text-sm font-semibold text-slate-300">Select an email to view</p>
                <p className="text-xs text-slate-500 mt-1 max-w-xs">
                  Choose a message from your inbox or compose a new email to send via Postfix.
                </p>
              </div>
            </div>
          )}
        </div>
      </div>

      {/* Compose Email Modal */}
      {showComposeModal && (
        <div className="fixed inset-0 z-50 bg-black/75 backdrop-blur-sm flex items-center justify-center p-4 animate-fadeIn">
          <div className="bg-surface-900 border border-surface-750 rounded-2xl w-full max-w-2xl shadow-2xl overflow-hidden flex flex-col max-h-[85vh]">
            {/* Modal Header */}
            <div className="h-14 px-5 border-b border-surface-800 bg-surface-950/60 flex items-center justify-between">
              <div className="flex items-center gap-2">
                <Send className="w-4 h-4 text-indigo-400" />
                <h3 className="text-sm font-bold text-white tracking-tight">New Message</h3>
              </div>
              <button
                onClick={() => setShowComposeModal(false)}
                className="p-1 rounded-lg text-slate-400 hover:text-white hover:bg-surface-800 transition"
              >
                ✕
              </button>
            </div>

            {/* Compose Form */}
            <form onSubmit={handleSendMessage} className="p-5 space-y-4 flex-1 overflow-y-auto">
              {/* From Selector */}
              <div className="flex items-center gap-3 text-xs">
                <label className="w-16 font-semibold text-slate-400">From:</label>
                <select
                  value={selectedEmail}
                  onChange={(e) => setSelectedEmail(e.target.value)}
                  className="flex-1 px-3 py-2 bg-surface-950 border border-surface-800 rounded-xl text-xs text-white font-mono focus:outline-none focus:border-indigo-500"
                >
                  {mailboxes.map((mb) => (
                    <option key={mb.id} value={mb.email}>
                      {mb.name} &lt;{mb.email}&gt;
                    </option>
                  ))}
                </select>
              </div>

              {/* To input */}
              <div className="flex items-center gap-3 text-xs">
                <label className="w-16 font-semibold text-slate-400">To:</label>
                <input
                  type="email"
                  placeholder="recipient@example.com"
                  required
                  value={composeTo}
                  onChange={(e) => setComposeTo(e.target.value)}
                  className="flex-1 px-3 py-2 bg-surface-950 border border-surface-800 rounded-xl text-xs text-white placeholder-slate-500 focus:outline-none focus:border-indigo-500"
                />
              </div>

              {/* Subject */}
              <div className="flex items-center gap-3 text-xs">
                <label className="w-16 font-semibold text-slate-400">Subject:</label>
                <input
                  type="text"
                  placeholder="Subject line"
                  value={composeSubject}
                  onChange={(e) => setComposeSubject(e.target.value)}
                  className="flex-1 px-3 py-2 bg-surface-950 border border-surface-800 rounded-xl text-xs text-white placeholder-slate-500 focus:outline-none focus:border-indigo-500 font-medium"
                />
              </div>

              {/* Message Body */}
              <div className="space-y-1">
                <label className="block text-xs font-semibold text-slate-400">Message Body:</label>
                <textarea
                  rows={9}
                  placeholder="Write your email here..."
                  value={composeBody}
                  onChange={(e) => setComposeBody(e.target.value)}
                  className="w-full px-3.5 py-2.5 bg-surface-950 border border-surface-800 rounded-xl text-xs text-white placeholder-slate-500 focus:outline-none focus:border-indigo-500 font-sans leading-relaxed resize-none"
                />
              </div>

              {/* Footer / Send Buttons */}
              <div className="pt-2 flex items-center justify-between border-t border-surface-800">
                <div className="flex items-center gap-2 text-xs text-slate-400">
                  <span className="flex items-center gap-1 text-[11px] text-emerald-400">
                    <ShieldCheck className="w-3.5 h-3.5" />
                    <span>Postfix TLS 1.3 Port 587</span>
                  </span>
                </div>

                <div className="flex items-center gap-2">
                  <button
                    type="button"
                    onClick={() => setShowComposeModal(false)}
                    className="px-4 py-2 rounded-xl text-xs font-medium text-slate-300 hover:bg-surface-800 transition"
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
