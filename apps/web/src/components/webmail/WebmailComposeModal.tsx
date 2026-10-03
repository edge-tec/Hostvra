'use client';

import React, { useState, useEffect, useRef, useCallback } from 'react';
import {
  X,
  Minus,
  Maximize2,
  Minimize2,
  Paperclip,
  Send,
  Trash2,
  Bold,
  Italic,
  Underline,
  List,
  ListOrdered,
  AlignLeft,
  AlignCenter,
  AlignRight,
  Link as LinkIcon,
  Image as ImageIcon,
  CheckCircle2,
  AlertCircle,
  Clock,
  Sparkles,
  ChevronDown,
} from 'lucide-react';
import { useWebmail } from '@/context/WebmailContext';
import { apiFetch } from '@/lib/api';

interface UploadedAttachment {
  id: string;
  filename: string;
  content_type: string;
  size_bytes: number;
}

interface ContactSuggestion {
  id: string;
  name: string;
  email: string;
}

function formatBytes(bytes: number): string {
  if (bytes === 0) return '0 B';
  const k = 1024;
  const sizes = ['B', 'KB', 'MB', 'GB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i];
}

export function WebmailComposeModal() {
  const {
    isComposeOpen,
    closeCompose,
    composeInitial,
    activeAccount,
    accounts,
    switchAccount,
    openAddAccount,
    refreshFolderCounts,
  } = useWebmail();

  const [minimized, setMinimized] = useState(false);
  const [fullscreen, setFullscreen] = useState(false);

  const [to, setTo] = useState('');
  const [showCc, setShowCc] = useState(false);
  const [cc, setCc] = useState('');
  const [showBcc, setShowBcc] = useState(false);
  const [bcc, setBcc] = useState('');
  const [subject, setSubject] = useState('');
  const [bodyHTML, setBodyHTML] = useState('');
  const [attachments, setAttachments] = useState<UploadedAttachment[]>([]);
  const [signatureText, setSignatureText] = useState('');

  // UI state
  const [isSending, setIsSending] = useState(false);
  const [isUploading, setIsUploading] = useState(false);
  const [draftStatus, setDraftStatus] = useState<string>('');
  const [draftId, setDraftId] = useState<string | null>(null);
  const [errorMessage, setErrorMessage] = useState<string | null>(null);

  // Recipient autocomplete suggestions
  const [contactSuggestions, setContactSuggestions] = useState<ContactSuggestion[]>([]);
  const [showSuggestions, setShowSuggestions] = useState(false);

  const editorRef = useRef<HTMLDivElement>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);
  const draftTimerRef = useRef<NodeJS.Timeout | null>(null);

  // Load active account signature
  useEffect(() => {
    if (!activeAccount) return;
    async function loadSig() {
      if (!activeAccount) return;
      try {
        const res = await apiFetch<Array<{ html?: string; content?: string }>>(
          `/api/v1/webmail/signatures?mailbox_id=${activeAccount.id}`
        );
        if (res.data && res.data.length > 0) {
          const sig = res.data[0];
          setSignatureText(sig.html || sig.content || '');
        }
      } catch (err) {
        console.debug('No custom signature loaded:', err);
      }
    }
    loadSig();
  }, [activeAccount]);

  // Populate initial compose data (e.g. Reply, Reply All, Forward)
  useEffect(() => {
    if (!isComposeOpen) return;
    setMinimized(false);
    setErrorMessage(null);

    if (composeInitial) {
      setTo(composeInitial.to || '');
      setCc(composeInitial.cc || '');
      setShowCc(!!composeInitial.cc);
      setBcc(composeInitial.bcc || '');
      setShowBcc(!!composeInitial.bcc);
      setSubject(composeInitial.subject || '');
      
      let initialBody = composeInitial.bodyHTML || composeInitial.bodyText || '';
      if (signatureText && !initialBody.includes(signatureText)) {
        initialBody = `${initialBody}<br/><br/>--<br/>${signatureText}`;
      }
      setBodyHTML(initialBody);
      if (editorRef.current) {
        editorRef.current.innerHTML = initialBody;
      }
      if (composeInitial.attachments) {
        setAttachments(composeInitial.attachments);
      }
    } else {
      setTo('');
      setCc('');
      setShowCc(false);
      setBcc('');
      setShowBcc(false);
      setSubject('');
      const defaultBody = signatureText ? `<br/><br/>--<br/>${signatureText}` : '';
      setBodyHTML(defaultBody);
      if (editorRef.current) {
        editorRef.current.innerHTML = defaultBody;
      }
      setAttachments([]);
      setDraftId(null);
    }
  }, [isComposeOpen, composeInitial, signatureText]);

  // Search contacts for autocomplete
  useEffect(() => {
    if (!activeAccount || !to || to.includes(',')) {
      setShowSuggestions(false);
      return;
    }
    const timer = setTimeout(async () => {
      try {
        const res = await apiFetch<ContactSuggestion[]>(
          `/api/v1/webmail/contacts?mailbox_id=${activeAccount.id}&q=${encodeURIComponent(to)}`
        );
        if (res.data && res.data.length > 0) {
          setContactSuggestions(res.data);
          setShowSuggestions(true);
        } else {
          setShowSuggestions(false);
        }
      } catch (e) {
        setShowSuggestions(false);
      }
    }, 250);
    return () => clearTimeout(timer);
  }, [to, activeAccount]);

  // Auto-save draft every 15 seconds
  const saveDraft = useCallback(async () => {
    if (!activeAccount || (!to && !subject && !bodyHTML)) return;
    try {
      setDraftStatus('Saving draft...');
      const payload: Record<string, unknown> = {
        mailbox_id: activeAccount.id,
        account_email: activeAccount.email,
        to_email: to,
        cc: cc,
        bcc: bcc,
        subject: subject || '(No subject)',
        body_html: bodyHTML,
        body_text: bodyHTML.replace(/<[^>]*>/g, ' '),
        attachments: attachments,
      };
      if (draftId) {
        payload.id = draftId;
      }
      const res = await apiFetch<{ id: string }>('/api/v1/webmail/draft', {
        method: 'POST',
        headers: activeAccount?.token ? {
          'Authorization': `Bearer ${activeAccount.token}`,
          'X-Webmail-Token': activeAccount.token,
        } : {},
        body: JSON.stringify(payload),
      });
      if (res.data?.id) {
        setDraftId(res.data.id);
        const timeStr = new Date().toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
        setDraftStatus(`Draft saved at ${timeStr}`);
      }
    } catch (e) {
      console.debug('Failed to save draft:', e);
      setDraftStatus('Error saving draft');
    }
  }, [activeAccount, to, cc, bcc, subject, bodyHTML, attachments, draftId]);

  useEffect(() => {
    if (!isComposeOpen) return;
    if (draftTimerRef.current) clearInterval(draftTimerRef.current);
    draftTimerRef.current = setInterval(() => {
      saveDraft();
    }, 15000);
    return () => {
      if (draftTimerRef.current) clearInterval(draftTimerRef.current);
    };
  }, [isComposeOpen, saveDraft]);

  // Format WYSIWYG commands
  const formatDoc = (cmd: string, val?: string) => {
    document.execCommand(cmd, false, val);
    if (editorRef.current) {
      setBodyHTML(editorRef.current.innerHTML);
    }
  };

  // Upload file attachment
  const handleFileUpload = async (files: FileList | null) => {
    if (!files || files.length === 0) return;
    setIsUploading(true);
    setErrorMessage(null);

    for (let i = 0; i < files.length; i++) {
      const file = files[i];
      if (file.size > 25 * 1024 * 1024) {
        setErrorMessage(`File "${file.name}" exceeds 25MB maximum limit.`);
        continue;
      }
      const formData = new FormData();
      formData.append('file', file);

      try {
        const res = await fetch('/api/v1/webmail/attachments/upload', {
          method: 'POST',
          body: formData,
        });
        const json = await res.json();
        if (json.success && json.data) {
          setAttachments((prev) => [...prev, json.data]);
        } else {
          setErrorMessage(json.error?.message || `Failed to upload ${file.name}`);
        }
      } catch (err: unknown) {
        setErrorMessage(err instanceof Error ? err.message : 'Upload failed due to connection error');
      }
    }
    setIsUploading(false);
  };

  // Remove attachment
  const handleRemoveAttachment = (id: string) => {
    setAttachments((prev) => prev.filter((a) => a.id !== id));
  };

  // Send message
  const handleSend = async () => {
    let currentAcc = activeAccount;
    if (!currentAcc && accounts.length > 0) {
      currentAcc = accounts[0];
      switchAccount(currentAcc.email);
    }

    if (!currentAcc) {
      setErrorMessage('No email mailbox connected. Please connect your email address first.');
      openAddAccount();
      return;
    }

    if (!to.trim()) {
      setErrorMessage('Please specify at least one recipient email address.');
      return;
    }
    setIsSending(true);
    setErrorMessage(null);

    const payload = {
      mailbox_id: currentAcc.id,
      account_email: currentAcc.email,
      from_email: currentAcc.email,
      to_email: to,
      to: to.split(',').map((e) => e.trim()).filter(Boolean),
      cc: cc,
      bcc: bcc,
      subject: subject || '(No subject)',
      body_html: bodyHTML,
      body_text: bodyHTML.replace(/<[^>]*>/g, ' '),
      attachments: attachments,
    };

    try {
      const res = await apiFetch<any>('/api/v1/webmail/send', {
        method: 'POST',
        headers: currentAcc?.token ? {
          'Authorization': `Bearer ${currentAcc.token}`,
          'X-Webmail-Token': currentAcc.token,
        } : {},
        body: JSON.stringify(payload),
      });

      if (res.data) {
        // If there was a draft, clean it up
        if (draftId) {
          apiFetch(`/api/v1/webmail/messages/${draftId}`, { method: 'DELETE' }).catch(() => {});
        }
        await refreshFolderCounts();
        closeCompose();
      } else {
        setErrorMessage(res.error?.message || 'Failed to send email. Please check server mail logs.');
      }
    } catch (err: unknown) {
      setErrorMessage(err instanceof Error ? err.message : 'Failed to send message via mail server');
    } finally {
      setIsSending(false);
    }
  };

  if (!isComposeOpen) return null;

  // Minimized state (bottom-right tray)
  if (minimized) {
    return (
      <div className="fixed bottom-0 right-8 z-50 w-72 bg-white border border-slate-200 rounded-t-2xl shadow-2xl flex items-center justify-between p-3 cursor-pointer hover:bg-slate-50 transition-colors">
        <div className="flex items-center gap-2 truncate" onClick={() => setMinimized(false)}>
          <div className="w-2.5 h-2.5 rounded-full bg-emerald-500" />
          <span className="text-xs font-bold text-slate-800 truncate">
            {subject || 'New Message'}
          </span>
        </div>
        <div className="flex items-center gap-1">
          <button
            onClick={() => setMinimized(false)}
            className="p-1 text-slate-400 hover:text-slate-600"
            title="Restore"
          >
            <Maximize2 className="w-3.5 h-3.5" />
          </button>
          <button
            onClick={closeCompose}
            className="p-1 text-slate-400 hover:text-rose-500"
            title="Close"
          >
            <X className="w-3.5 h-3.5" />
          </button>
        </div>
      </div>
    );
  }

  return (
    <div
      className={`fixed z-50 flex flex-col bg-white border border-slate-200 rounded-2xl shadow-2xl overflow-hidden transition-all duration-200 ${
        fullscreen
          ? 'inset-4 sm:inset-10'
          : 'bottom-0 right-4 sm:right-8 w-full max-w-2xl h-[620px] rounded-b-none'
      }`}
    >
      {/* Header */}
      <div className="h-11 px-4 bg-slate-100 border-b border-slate-200 flex items-center justify-between select-none">
        <div className="flex items-center gap-2 min-w-0">
          <span className="text-xs font-bold text-slate-900 truncate">
            {subject ? subject : 'New Message'}
          </span>
          {activeAccount && (
            <span className="text-[10px] text-slate-400 truncate">
              ({activeAccount.email})
            </span>
          )}
        </div>
        <div className="flex items-center gap-1">
          <button
            onClick={() => setMinimized(true)}
            className="p-1 text-slate-400 hover:text-slate-700 rounded-md"
            title="Minimize"
          >
            <Minus className="w-3.5 h-3.5" />
          </button>
          <button
            onClick={() => setFullscreen(!fullscreen)}
            className="p-1 text-slate-400 hover:text-slate-700 rounded-md"
            title={fullscreen ? 'Exit Fullscreen' : 'Fullscreen'}
          >
            {fullscreen ? <Minimize2 className="w-3.5 h-3.5" /> : <Maximize2 className="w-3.5 h-3.5" />}
          </button>
          <button
            onClick={closeCompose}
            className="p-1 text-slate-400 hover:text-rose-600 rounded-md"
            title="Close"
          >
            <X className="w-4 h-4" />
          </button>
        </div>
      </div>

      {/* Error alert */}
      {errorMessage && (
        <div className="px-4 py-2 bg-rose-50 border-b border-rose-200 flex items-center gap-2 text-xs text-rose-700">
          <AlertCircle className="w-4 h-4 flex-shrink-0" />
          <span className="flex-1 truncate">{errorMessage}</span>
          <button onClick={() => setErrorMessage(null)} className="text-rose-400 hover:text-rose-600">
            <X className="w-3.5 h-3.5" />
          </button>
        </div>
      )}

      {/* Recipient inputs */}
      <div className="px-4 py-1.5 space-y-1 border-b border-slate-200 text-xs">
        {/* TO field */}
        <div className="relative flex items-center gap-2 py-1">
          <span className="text-slate-400 font-medium w-14">To:</span>
          <input
            type="text"
            value={to}
            onChange={(e) => setTo(e.target.value)}
            placeholder="Recipient email address (comma-separated)"
            className="flex-1 bg-transparent border-none text-slate-900 placeholder:text-slate-400 focus:outline-none"
          />
          <div className="flex items-center gap-2 text-[11px] text-slate-400 font-medium">
            {!showCc && (
              <button onClick={() => setShowCc(true)} className="hover:text-slate-700">
                Cc
              </button>
            )}
            {!showBcc && (
              <button onClick={() => setShowBcc(true)} className="hover:text-slate-700">
                Bcc
              </button>
            )}
          </div>

          {/* Autocomplete Dropdown */}
          {showSuggestions && contactSuggestions.length > 0 && (
            <div className="absolute left-14 top-full mt-1 w-72 bg-white border border-slate-200 rounded-xl shadow-xl z-50 max-h-48 overflow-y-auto">
              {contactSuggestions.map((c) => (
                <button
                  key={c.id}
                  onClick={() => {
                    setTo(c.email);
                    setShowSuggestions(false);
                  }}
                  className="w-full text-left px-3 py-2 hover:bg-emerald-50 flex items-center justify-between text-xs"
                >
                  <span className="font-semibold text-slate-900 truncate">{c.name}</span>
                  <span className="text-[11px] text-slate-400 truncate">{c.email}</span>
                </button>
              ))}
            </div>
          )}
        </div>

        {/* CC field */}
        {showCc && (
          <div className="flex items-center gap-2 py-1 border-t border-slate-100">
            <span className="text-slate-400 font-medium w-14">Cc:</span>
            <input
              type="text"
              value={cc}
              onChange={(e) => setCc(e.target.value)}
              placeholder="Cc recipients"
              className="flex-1 bg-transparent border-none text-slate-900 placeholder:text-slate-400 focus:outline-none"
            />
          </div>
        )}

        {/* BCC field */}
        {showBcc && (
          <div className="flex items-center gap-2 py-1 border-t border-slate-100">
            <span className="text-slate-400 font-medium w-14">Bcc:</span>
            <input
              type="text"
              value={bcc}
              onChange={(e) => setBcc(e.target.value)}
              placeholder="Bcc recipients"
              className="flex-1 bg-transparent border-none text-slate-900 placeholder:text-slate-400 focus:outline-none"
            />
          </div>
        )}

        {/* SUBJECT field */}
        <div className="flex items-center gap-2 py-1 border-t border-slate-100">
          <span className="text-slate-400 font-medium w-14">Subject:</span>
          <input
            type="text"
            value={subject}
            onChange={(e) => setSubject(e.target.value)}
            placeholder="Subject line"
            className="flex-1 bg-transparent border-none text-slate-900 font-semibold placeholder:text-slate-400 focus:outline-none"
          />
        </div>
      </div>

      {/* WYSIWYG Editor Toolbar */}
      <div className="px-3 py-1.5 bg-slate-50 border-b border-slate-200 flex items-center gap-1 flex-wrap text-slate-600">
        <button
          onClick={() => formatDoc('bold')}
          className="p-1 hover:bg-slate-200 rounded-md"
          title="Bold (Ctrl+B)"
        >
          <Bold className="w-3.5 h-3.5" />
        </button>
        <button
          onClick={() => formatDoc('italic')}
          className="p-1 hover:bg-slate-200 rounded-md"
          title="Italic (Ctrl+I)"
        >
          <Italic className="w-3.5 h-3.5" />
        </button>
        <button
          onClick={() => formatDoc('underline')}
          className="p-1 hover:bg-slate-200 rounded-md"
          title="Underline (Ctrl+U)"
        >
          <Underline className="w-3.5 h-3.5" />
        </button>
        <div className="w-[1px] h-4 bg-slate-300 mx-1" />
        <button
          onClick={() => formatDoc('insertUnorderedList')}
          className="p-1 hover:bg-slate-200 rounded-md"
          title="Bulleted List"
        >
          <List className="w-3.5 h-3.5" />
        </button>
        <button
          onClick={() => formatDoc('insertOrderedList')}
          className="p-1 hover:bg-slate-200 rounded-md"
          title="Numbered List"
        >
          <ListOrdered className="w-3.5 h-3.5" />
        </button>
        <div className="w-[1px] h-4 bg-slate-300 mx-1" />
        <button
          onClick={() => formatDoc('justifyLeft')}
          className="p-1 hover:bg-slate-200 rounded-md"
          title="Align Left"
        >
          <AlignLeft className="w-3.5 h-3.5" />
        </button>
        <button
          onClick={() => formatDoc('justifyCenter')}
          className="p-1 hover:bg-slate-200 rounded-md"
          title="Align Center"
        >
          <AlignCenter className="w-3.5 h-3.5" />
        </button>
        <button
          onClick={() => formatDoc('justifyRight')}
          className="p-1 hover:bg-slate-200 rounded-md"
          title="Align Right"
        >
          <AlignRight className="w-3.5 h-3.5" />
        </button>
        <div className="w-[1px] h-4 bg-slate-300 mx-1" />
        <button
          onClick={() => {
            const url = prompt('Enter link URL:');
            if (url) formatDoc('createLink', url);
          }}
          className="p-1 hover:bg-slate-200 rounded-md"
          title="Insert Link"
        >
          <LinkIcon className="w-3.5 h-3.5" />
        </button>
        <button
          onClick={() => {
            const url = prompt('Enter image URL:');
            if (url) formatDoc('insertImage', url);
          }}
          className="p-1 hover:bg-slate-200 rounded-md"
          title="Insert Image URL"
        >
          <ImageIcon className="w-3.5 h-3.5" />
        </button>
      </div>

      {/* Editor Body */}
      <div
        ref={editorRef}
        contentEditable
        onInput={(e) => setBodyHTML(e.currentTarget.innerHTML)}
        className="flex-1 p-4 overflow-y-auto focus:outline-none text-xs text-slate-800 font-sans leading-relaxed selection:bg-emerald-200"
        style={{ minHeight: '180px' }}
      />

      {/* Uploaded Attachments List */}
      {attachments.length > 0 && (
        <div className="px-4 py-2 bg-slate-50 border-t border-slate-200 flex items-center gap-2 flex-wrap max-h-24 overflow-y-auto">
          {attachments.map((att) => (
            <div
              key={att.id}
              className="flex items-center gap-2 px-2.5 py-1 bg-white border border-slate-200 rounded-lg text-[11px] text-slate-700"
            >
              <Paperclip className="w-3 h-3 text-emerald-500" />
              <span className="truncate max-w-[140px] font-medium">{att.filename}</span>
              <span className="text-[10px] text-slate-400">({formatBytes(att.size_bytes)})</span>
              <button
                onClick={() => handleRemoveAttachment(att.id)}
                className="text-slate-400 hover:text-rose-500"
                title="Remove"
              >
                <X className="w-3 h-3" />
              </button>
            </div>
          ))}
        </div>
      )}

      {/* Bottom Action Footer */}
      <div className="h-14 px-4 bg-slate-50 border-t border-slate-200 flex items-center justify-between">
        <div className="flex items-center gap-3">
          {/* Send button */}
          <button
            onClick={handleSend}
            disabled={isSending || isUploading}
            className="flex items-center gap-2 px-5 py-2 bg-gradient-to-r from-emerald-600 to-teal-600 hover:from-emerald-500 hover:to-teal-500 text-white font-bold text-xs rounded-xl shadow-md shadow-emerald-600/20 disabled:opacity-50 transition-all hover:scale-[1.02] active:scale-[0.98]"
          >
            {isSending ? (
              <span className="flex items-center gap-1.5">
                <Clock className="w-3.5 h-3.5 animate-spin" /> Sending...
              </span>
            ) : (
              <>
                <Send className="w-3.5 h-3.5" />
                <span>Send</span>
              </>
            )}
          </button>

          {/* Attach file button */}
          <button
            onClick={() => fileInputRef.current?.click()}
            disabled={isUploading}
            className="p-2 text-slate-500 hover:text-slate-900 hover:bg-slate-200 rounded-xl transition-colors"
            title="Attach files (Max 25MB)"
          >
            <Paperclip className={`w-4 h-4 ${isUploading ? 'animate-bounce text-emerald-500' : ''}`} />
          </button>
          <input
            ref={fileInputRef}
            type="file"
            multiple
            className="hidden"
            onChange={(e) => handleFileUpload(e.target.files)}
          />

          {/* Draft status indicator */}
          {draftStatus && (
            <span className="text-[11px] text-slate-400 flex items-center gap-1">
              <Clock className="w-3 h-3 text-slate-400" />
              {draftStatus}
            </span>
          )}
        </div>

        {/* Discard draft trash button */}
        <button
          onClick={() => {
            if (confirm('Discard this draft?')) {
              if (draftId) {
                apiFetch(`/api/v1/webmail/messages/${draftId}`, { method: 'DELETE' }).catch(() => {});
              }
              closeCompose();
            }
          }}
          className="p-2 text-slate-400 hover:text-rose-600 hover:bg-rose-50 rounded-xl transition-colors"
          title="Discard draft"
        >
          <Trash2 className="w-4 h-4" />
        </button>
      </div>
    </div>
  );
}
