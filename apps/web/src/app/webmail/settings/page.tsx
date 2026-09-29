'use client';

import { useState, useEffect } from 'react';
import { useWebmail } from '@/context/WebmailContext';
import {
  Settings,
  User,
  Mail,
  Filter,
  Share2,
  Shield,
  Bell,
  Edit3,
  Plus,
  Trash2,
  Check,
  Save,
  AlertCircle,
  Lock,
  LogOut,
  Volume2,
  VolumeX,
} from 'lucide-react';

type SettingsTab = 'general' | 'identities' | 'signature' | 'filters' | 'forwarding' | 'security' | 'notifications';

export default function WebmailSettingsPage() {
  const {
    activeAccount,
    accounts,
    identities,
    fetchIdentities,
    saveIdentity,
    filters,
    fetchFilters,
    saveFilter,
    deleteFilter,
    forwardingRule,
    fetchForwarding,
    saveForwarding,
    deleteForwarding,
    soundEnabled,
    setSoundEnabled,
    desktopNotifications,
    requestNotificationPermission,
    logoutCurrentAccount,
    logoutAllAccounts,
    readingPaneLayout,
    setReadingPaneLayout,
    emailsPerPage,
    setEmailsPerPage,
    defaultFolder,
    setDefaultFolder,
    syncInterval,
    setSyncInterval,
    updatePreferences,
  } = useWebmail();

  const [activeTab, setActiveTab] = useState<SettingsTab>('general');
  const [successMsg, setSuccessMsg] = useState('');
  const [errorMsg, setErrorMsg] = useState('');

  // General tab local state
  const [selectedDefaultFolder, setSelectedDefaultFolder] = useState(defaultFolder || 'inbox');
  const [selectedPageSize, setSelectedPageSize] = useState(emailsPerPage || 50);
  const [selectedLayout, setSelectedLayout] = useState<'split' | 'full'>(readingPaneLayout || 'split');
  const [selectedSyncInterval, setSelectedSyncInterval] = useState(syncInterval || 'realtime');

  // Identity state
  const [displayName, setDisplayName] = useState('');
  const [replyTo, setReplyTo] = useState('');
  const [signatureText, setSignatureText] = useState('');

  // Filter creation modal/state
  const [newFilterName, setNewFilterName] = useState('');
  const [filterCriteriaField, setFilterCriteriaField] = useState('from');
  const [filterCriteriaVal, setFilterCriteriaVal] = useState('');
  const [filterAction, setFilterAction] = useState('move');
  const [filterActionTarget, setFilterActionTarget] = useState('Archive');

  // Forwarding state
  const [forwardDestination, setForwardDestination] = useState('');
  const [forwardKeepCopy, setForwardKeepCopy] = useState(true);

  // Security
  const [currentPassword, setCurrentPassword] = useState('');
  const [newPassword, setNewPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');

  useEffect(() => {
    setSelectedDefaultFolder(defaultFolder);
    setSelectedPageSize(emailsPerPage);
    setSelectedLayout(readingPaneLayout);
    setSelectedSyncInterval(syncInterval);
  }, [defaultFolder, emailsPerPage, readingPaneLayout, syncInterval]);

  useEffect(() => {
    if (activeAccount) {
      fetchIdentities();
      fetchFilters();
      fetchForwarding();
    }
  }, [activeAccount?.id]);

  useEffect(() => {
    if (identities && identities.length > 0) {
      const primary = identities.find(i => i.is_default) || identities[0];
      setDisplayName(primary.display_name || '');
      setReplyTo(primary.reply_to || '');
      setSignatureText(primary.signature_html || primary.signature_text || '');
    } else if (activeAccount) {
      setDisplayName(activeAccount.name || '');
    }

    if (forwardingRule) {
      setForwardDestination(forwardingRule.forward_to);
      setForwardKeepCopy(forwardingRule.keep_copy);
    }
  }, [identities, forwardingRule, activeAccount]);

  const showNotification = (msg: string, isError = false) => {
    if (isError) {
      setErrorMsg(msg);
      setSuccessMsg('');
    } else {
      setSuccessMsg(msg);
      setErrorMsg('');
    }
    setTimeout(() => {
      setSuccessMsg('');
      setErrorMsg('');
    }, 4000);
  };

  const handleSaveGeneral = async () => {
    try {
      await updatePreferences({
        readingPaneLayout: selectedLayout,
        emailsPerPage: Number(selectedPageSize),
        defaultFolder: selectedDefaultFolder,
        syncInterval: selectedSyncInterval,
      });
      showNotification('General layout and mailbox preferences saved successfully!');
    } catch {
      showNotification('Failed to save general preferences', true);
    }
  };

  const handleSaveIdentity = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!activeAccount) {
      showNotification('No active account selected', true);
      return;
    }
    const success = await saveIdentity({
      mailbox_id: activeAccount.id,
      email: activeAccount.email,
      display_name: displayName,
      reply_to: replyTo,
      signature_html: signatureText,
      signature_text: signatureText.replace(/<[^>]*>/g, ''),
      is_default: true,
    });
    if (success) {
      showNotification('Email identity and signature saved successfully!');
    } else {
      showNotification('Failed to save identity changes', true);
    }
  };

  const handleCreateFilter = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!activeAccount) {
      showNotification('No active mailbox selected', true);
      return;
    }
    if (!newFilterName || !filterCriteriaVal) {
      showNotification('Filter name and keyword pattern are required', true);
      return;
    }
    const success = await saveFilter({
      mailbox_id: activeAccount.id,
      name: newFilterName,
      criteria_field: filterCriteriaField,
      criteria_pattern: filterCriteriaVal,
      action: filterAction,
      action_target: filterAction === 'move' ? filterActionTarget : '',
      is_active: true,
      priority: filters.length + 1,
    });
    if (success) {
      setNewFilterName('');
      setFilterCriteriaVal('');
      showNotification('Mail filter rule created and active!');
    } else {
      showNotification('Failed to create mail filter rule', true);
    }
  };

  const handleDeleteFilter = async (id: string) => {
    if (confirm('Are you sure you want to delete this filter rule?')) {
      const ok = await deleteFilter(id);
      if (ok) showNotification('Filter rule deleted');
      else showNotification('Failed to delete filter', true);
    }
  };

  const handleSaveForwarding = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!forwardDestination) {
      showNotification('Forwarding email destination required', true);
      return;
    }
    const success = await saveForwarding(forwardDestination, forwardKeepCopy);
    if (success) {
      showNotification('Email forwarding rule configured successfully!');
    } else {
      showNotification('Failed to save forwarding rule', true);
    }
  };

  const handleDeleteForwarding = async () => {
    if (confirm('Disable and remove active email forwarding?')) {
      const ok = await deleteForwarding();
      if (ok) {
        setForwardDestination('');
        showNotification('Forwarding disabled');
      } else {
        showNotification('Failed to delete forwarding rule', true);
      }
    }
  };

  const handleUpdatePassword = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!activeAccount) return;
    if (newPassword !== confirmPassword) {
      showNotification('New passwords do not match', true);
      return;
    }
    if (newPassword.length < 8) {
      showNotification('Password must be at least 8 characters long', true);
      return;
    }
    try {
      const res = await fetch(`/api/v1/email/mailboxes/${activeAccount.id}/password`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          current_password: currentPassword,
          new_password: newPassword,
        }),
      });
      if (res.ok) {
        showNotification('Password updated successfully! Please re-login if required.');
        setCurrentPassword('');
        setNewPassword('');
        setConfirmPassword('');
      } else {
        const err = await res.json().catch(() => ({}));
        showNotification(err.error || 'Failed to update password', true);
      }
    } catch {
      showNotification('Network error updating password', true);
    }
  };

  return (
    <div className="flex-1 overflow-y-auto bg-slate-50 p-6 md:p-8">
      <div className="max-w-5xl mx-auto space-y-6">
        {/* Header Title */}
        <div className="flex items-center justify-between pb-4 border-b border-slate-200">
          <div>
            <h1 className="text-2xl font-bold text-slate-900 flex items-center gap-2.5">
              <Settings className="w-6 h-6 text-emerald-600" />
              Webmail Settings
            </h1>
            <p className="text-xs text-slate-500 mt-1">
              Configure identities, filters, signatures, security and preferences for{' '}
              <span className="font-semibold text-emerald-700">{activeAccount?.email || 'Active Account'}</span>
            </p>
          </div>

          <div className="text-xs text-slate-600 font-mono bg-white px-3 py-1.5 rounded-lg border border-slate-200 shadow-xs">
            Hostvra Mail v2.5
          </div>
        </div>

        {/* Global Notifications */}
        {successMsg && (
          <div className="p-3.5 bg-emerald-50 border border-emerald-200 rounded-xl text-emerald-800 text-xs flex items-center gap-2 animate-in fade-in shadow-xs">
            <Check className="w-4 h-4 text-emerald-600 shrink-0" />
            <span>{successMsg}</span>
          </div>
        )}
        {errorMsg && (
          <div className="p-3.5 bg-rose-50 border border-rose-200 rounded-xl text-rose-800 text-xs flex items-center gap-2 animate-in fade-in shadow-xs">
            <AlertCircle className="w-4 h-4 text-rose-600 shrink-0" />
            <span>{errorMsg}</span>
          </div>
        )}

        {/* Settings Navigation Tabs */}
        <div className="flex overflow-x-auto gap-2 p-1.5 bg-white rounded-2xl border border-slate-200 shadow-xs scrollbar-none">
          <button
            onClick={() => setActiveTab('general')}
            className={`flex items-center gap-2 px-4 py-2 rounded-xl text-xs font-semibold transition-all shrink-0 ${
              activeTab === 'general' ? 'bg-emerald-600 text-white shadow-md shadow-emerald-600/20' : 'text-slate-600 hover:text-slate-900 hover:bg-slate-100'
            }`}
          >
            <Settings className="w-3.5 h-3.5" />
            General
          </button>
          <button
            onClick={() => setActiveTab('identities')}
            className={`flex items-center gap-2 px-4 py-2 rounded-xl text-xs font-semibold transition-all shrink-0 ${
              activeTab === 'identities' ? 'bg-emerald-600 text-white shadow-md shadow-emerald-600/20' : 'text-slate-600 hover:text-slate-900 hover:bg-slate-100'
            }`}
          >
            <User className="w-3.5 h-3.5" />
            Identities
          </button>
          <button
            onClick={() => setActiveTab('signature')}
            className={`flex items-center gap-2 px-4 py-2 rounded-xl text-xs font-semibold transition-all shrink-0 ${
              activeTab === 'signature' ? 'bg-emerald-600 text-white shadow-md shadow-emerald-600/20' : 'text-slate-600 hover:text-slate-900 hover:bg-slate-100'
            }`}
          >
            <Edit3 className="w-3.5 h-3.5" />
            Signatures
          </button>
          <button
            onClick={() => setActiveTab('filters')}
            className={`flex items-center gap-2 px-4 py-2 rounded-xl text-xs font-semibold transition-all shrink-0 ${
              activeTab === 'filters' ? 'bg-emerald-600 text-white shadow-md shadow-emerald-600/20' : 'text-slate-600 hover:text-slate-900 hover:bg-slate-100'
            }`}
          >
            <Filter className="w-3.5 h-3.5" />
            Rules & Filters
          </button>
          <button
            onClick={() => setActiveTab('forwarding')}
            className={`flex items-center gap-2 px-4 py-2 rounded-xl text-xs font-semibold transition-all shrink-0 ${
              activeTab === 'forwarding' ? 'bg-emerald-600 text-white shadow-md shadow-emerald-600/20' : 'text-slate-600 hover:text-slate-900 hover:bg-slate-100'
            }`}
          >
            <Share2 className="w-3.5 h-3.5" />
            Forwarding
          </button>
          <button
            onClick={() => setActiveTab('notifications')}
            className={`flex items-center gap-2 px-4 py-2 rounded-xl text-xs font-semibold transition-all shrink-0 ${
              activeTab === 'notifications' ? 'bg-emerald-600 text-white shadow-md shadow-emerald-600/20' : 'text-slate-600 hover:text-slate-900 hover:bg-slate-100'
            }`}
          >
            <Bell className="w-3.5 h-3.5" />
            Notifications
          </button>
          <button
            onClick={() => setActiveTab('security')}
            className={`flex items-center gap-2 px-4 py-2 rounded-xl text-xs font-semibold transition-all shrink-0 ${
              activeTab === 'security' ? 'bg-emerald-600 text-white shadow-md shadow-emerald-600/20' : 'text-slate-600 hover:text-slate-900 hover:bg-slate-100'
            }`}
          >
            <Shield className="w-3.5 h-3.5" />
            Security & Accounts
          </button>
        </div>

        {/* TAB 1: GENERAL & LAYOUT */}
        {activeTab === 'general' && (
          <div className="bg-white border border-slate-200/90 rounded-2xl p-6 space-y-6 shadow-xs">
            <h2 className="text-base font-bold text-slate-900 flex items-center gap-2">
              <Mail className="w-4 h-4 text-emerald-600" />
              General & Layout Preferences
            </h2>

            <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
              <div className="space-y-2">
                <label className="text-xs font-semibold text-slate-700">Reading Pane Layout</label>
                <select
                  value={selectedLayout}
                  onChange={(e) => setSelectedLayout(e.target.value as 'split' | 'full')}
                  className="w-full bg-slate-50 border border-slate-200 rounded-xl px-3.5 py-2.5 text-xs text-slate-900 focus:outline-none focus:border-emerald-500"
                >
                  <option value="split">Split View (List on left, preview on right - Default)</option>
                  <option value="full">Full Width (Click message to open full page)</option>
                </select>
                <p className="text-[11px] text-slate-500">Standard 2-pane Gmail layout or full-screen single message view.</p>
              </div>

              <div className="space-y-2">
                <label className="text-xs font-semibold text-slate-700">Emails Per Page</label>
                <select
                  value={selectedPageSize}
                  onChange={(e) => setSelectedPageSize(Number(e.target.value))}
                  className="w-full bg-slate-50 border border-slate-200 rounded-xl px-3.5 py-2.5 text-xs text-slate-900 focus:outline-none focus:border-emerald-500"
                >
                  <option value={25}>25 conversations</option>
                  <option value={50}>50 conversations (Recommended)</option>
                  <option value={100}>100 conversations</option>
                </select>
                <p className="text-[11px] text-slate-500">Number of messages displayed simultaneously in mailbox list.</p>
              </div>

              <div className="space-y-2">
                <label className="text-xs font-semibold text-slate-700">Default Mailbox Folder</label>
                <select
                  value={selectedDefaultFolder}
                  onChange={(e) => setSelectedDefaultFolder(e.target.value)}
                  className="w-full bg-slate-50 border border-slate-200 rounded-xl px-3.5 py-2.5 text-xs text-slate-900 focus:outline-none focus:border-emerald-500"
                >
                  <option value="inbox">Inbox</option>
                  <option value="starred">Starred</option>
                  <option value="archive">Archive</option>
                </select>
                <p className="text-[11px] text-slate-500">Folder to open immediately after loading webmail.</p>
              </div>

              <div className="space-y-2">
                <label className="text-xs font-semibold text-slate-700">Sync Interval</label>
                <select
                  value={selectedSyncInterval}
                  onChange={(e) => setSelectedSyncInterval(e.target.value)}
                  className="w-full bg-slate-50 border border-slate-200 rounded-xl px-3.5 py-2.5 text-xs text-slate-900 focus:outline-none focus:border-emerald-500"
                >
                  <option value="realtime">Real-time SSE Push (Instant)</option>
                  <option value="60">Every 1 minute</option>
                  <option value="300">Every 5 minutes</option>
                </select>
                <p className="text-[11px] text-slate-500">How frequently the mail server is polled for background sync.</p>
              </div>
            </div>

            <div className="pt-4 border-t border-slate-100 flex justify-end">
              <button
                onClick={handleSaveGeneral}
                className="flex items-center gap-2 px-5 py-2.5 bg-emerald-600 hover:bg-emerald-500 text-white rounded-xl text-xs font-bold shadow-md shadow-emerald-600/20 transition-all hover:scale-[1.02] active:scale-[0.98]"
              >
                <Save className="w-3.5 h-3.5" />
                Save Preferences
              </button>
            </div>
          </div>
        )}

        {/* TAB 2: IDENTITIES */}
        {activeTab === 'identities' && (
          <div className="bg-white border border-slate-200/90 rounded-2xl p-6 space-y-6 shadow-xs">
            <div>
              <h2 className="text-base font-bold text-slate-900 flex items-center gap-2">
                <User className="w-4 h-4 text-emerald-600" />
                Email Identities & Display Names
              </h2>
              <p className="text-xs text-slate-500 mt-1">
                Customize how your name and address appear to recipients when you send an email.
              </p>
            </div>

            <form onSubmit={handleSaveIdentity} className="space-y-5">
              <div className="grid grid-cols-1 md:grid-cols-2 gap-5">
                <div className="space-y-2">
                  <label className="text-xs font-semibold text-slate-700">Sending Email Address</label>
                  <input
                    type="text"
                    disabled
                    value={activeAccount?.email || ''}
                    className="w-full bg-slate-100 border border-slate-200 rounded-xl px-3.5 py-2.5 text-xs text-slate-500 cursor-not-allowed"
                  />
                  <p className="text-[11px] text-slate-400">Managed by Hostvra Mail Server.</p>
                </div>

                <div className="space-y-2">
                  <label className="text-xs font-semibold text-slate-700">
                    Display Name <span className="text-emerald-600">*</span>
                  </label>
                  <input
                    type="text"
                    required
                    value={displayName}
                    onChange={(e) => setDisplayName(e.target.value)}
                    placeholder="e.g. Hostvra Support or Mizanur Rahman"
                    className="w-full bg-slate-50 border border-slate-200 rounded-xl px-3.5 py-2.5 text-xs text-slate-900 focus:outline-none focus:border-emerald-500"
                  />
                  <p className="text-[11px] text-slate-400">Recipients will see: &quot;{displayName || 'Your Name'} &lt;{activeAccount?.email}&gt;&quot;</p>
                </div>

                <div className="space-y-2 md:col-span-2">
                  <label className="text-xs font-semibold text-slate-700">Reply-To Address (Optional)</label>
                  <input
                    type="email"
                    value={replyTo}
                    onChange={(e) => setReplyTo(e.target.value)}
                    placeholder="e.g. replies@yourdomain.com (leave blank to reply directly to sender)"
                    className="w-full bg-slate-50 border border-slate-200 rounded-xl px-3.5 py-2.5 text-xs text-slate-900 focus:outline-none focus:border-emerald-500"
                  />
                </div>
              </div>

              <div className="pt-4 border-t border-slate-100 flex justify-end">
                <button
                  type="submit"
                  className="flex items-center gap-2 px-5 py-2.5 bg-emerald-600 hover:bg-emerald-500 text-white rounded-xl text-xs font-bold shadow-md shadow-emerald-600/20 transition-all"
                >
                  <Save className="w-3.5 h-3.5" />
                  Save Identity
                </button>
              </div>
            </form>
          </div>
        )}

        {/* TAB 3: SIGNATURE */}
        {activeTab === 'signature' && (
          <div className="bg-white border border-slate-200/90 rounded-2xl p-6 space-y-6 shadow-xs">
            <div>
              <h2 className="text-base font-bold text-slate-900 flex items-center gap-2">
                <Edit3 className="w-4 h-4 text-emerald-600" />
                Email Signature
              </h2>
              <p className="text-xs text-slate-500 mt-1">
                This signature will be automatically appended to the bottom of all outgoing emails and replies.
              </p>
            </div>

            <form onSubmit={handleSaveIdentity} className="space-y-5">
              <div className="space-y-2">
                <label className="text-xs font-semibold text-slate-700">Signature HTML / Text</label>
                <textarea
                  rows={6}
                  value={signatureText}
                  onChange={(e) => setSignatureText(e.target.value)}
                  placeholder="--&#10;Best regards,&#10;Hostvra Support Team&#10;https://hostvra.com"
                  className="w-full bg-slate-50 border border-slate-200 rounded-xl p-3.5 text-xs text-slate-900 font-mono focus:outline-none focus:border-emerald-500 leading-relaxed"
                />
              </div>

              {signatureText && (
                <div className="p-4 bg-slate-50 border border-slate-200 rounded-xl space-y-1.5">
                  <span className="text-[11px] font-bold text-slate-500 uppercase tracking-wider">Live Preview:</span>
                  <div
                    className="text-xs text-slate-700 pt-2 border-t border-slate-200"
                    dangerouslySetInnerHTML={{ __html: signatureText.replace(/\n/g, '<br/>') }}
                  />
                </div>
              )}

              <div className="pt-4 border-t border-slate-100 flex justify-end">
                <button
                  type="submit"
                  className="flex items-center gap-2 px-5 py-2.5 bg-emerald-600 hover:bg-emerald-500 text-white rounded-xl text-xs font-bold shadow-md shadow-emerald-600/20 transition-all"
                >
                  <Save className="w-3.5 h-3.5" />
                  Save Signature
                </button>
              </div>
            </form>
          </div>
        )}

        {/* TAB 4: FILTERS */}
        {activeTab === 'filters' && (
          <div className="space-y-6">
            <div className="bg-white border border-slate-200/90 rounded-2xl p-6 space-y-6 shadow-xs">
              <div>
                <h2 className="text-base font-bold text-slate-900 flex items-center gap-2">
                  <Plus className="w-4 h-4 text-emerald-600" />
                  Create Real Mail Filter
                </h2>
                <p className="text-xs text-slate-500 mt-1">
                  Incoming emails are evaluated automatically on the server against active filter rules.
                </p>
              </div>

              <form onSubmit={handleCreateFilter} className="space-y-4">
                <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                  <div className="space-y-1.5">
                    <label className="text-xs font-semibold text-slate-700">Filter Name</label>
                    <input
                      type="text"
                      required
                      placeholder="e.g. Move Invoices to Billing"
                      value={newFilterName}
                      onChange={(e) => setNewFilterName(e.target.value)}
                      className="w-full bg-slate-50 border border-slate-200 rounded-xl px-3 py-2 text-xs text-slate-900 focus:outline-none focus:border-emerald-500"
                    />
                  </div>

                  <div className="space-y-1.5">
                    <label className="text-xs font-semibold text-slate-700">If Message Condition</label>
                    <select
                      value={filterCriteriaField}
                      onChange={(e) => setFilterCriteriaField(e.target.value)}
                      className="w-full bg-slate-50 border border-slate-200 rounded-xl px-3 py-2 text-xs text-slate-900 focus:outline-none focus:border-emerald-500"
                    >
                      <option value="from">From contains</option>
                      <option value="to">To contains</option>
                      <option value="subject">Subject contains</option>
                      <option value="body">Body contains</option>
                    </select>
                  </div>

                  <div className="space-y-1.5">
                    <label className="text-xs font-semibold text-slate-700">Pattern / Keyword</label>
                    <input
                      type="text"
                      required
                      placeholder="e.g. invoice, billing@, alerts"
                      value={filterCriteriaVal}
                      onChange={(e) => setFilterCriteriaVal(e.target.value)}
                      className="w-full bg-slate-50 border border-slate-200 rounded-xl px-3 py-2 text-xs text-slate-900 focus:outline-none focus:border-emerald-500"
                    />
                  </div>
                </div>

                <div className="grid grid-cols-1 md:grid-cols-2 gap-4 pt-2">
                  <div className="space-y-1.5">
                    <label className="text-xs font-semibold text-slate-700">Action to Take</label>
                    <select
                      value={filterAction}
                      onChange={(e) => setFilterAction(e.target.value)}
                      className="w-full bg-slate-50 border border-slate-200 rounded-xl px-3 py-2 text-xs text-slate-900 focus:outline-none focus:border-emerald-500"
                    >
                      <option value="move">Move to Folder</option>
                      <option value="star">Star Message</option>
                      <option value="mark_read">Mark as Read</option>
                      <option value="delete">Delete / Move to Trash</option>
                      <option value="mark_spam">Mark as Spam</option>
                    </select>
                  </div>

                  {filterAction === 'move' && (
                    <div className="space-y-1.5">
                      <label className="text-xs font-semibold text-slate-700">Target Folder</label>
                      <select
                        value={filterActionTarget}
                        onChange={(e) => setFilterActionTarget(e.target.value)}
                        className="w-full bg-slate-50 border border-slate-200 rounded-xl px-3 py-2 text-xs text-slate-900 focus:outline-none focus:border-emerald-500"
                      >
                        <option value="Archive">Archive</option>
                        <option value="Trash">Trash</option>
                        <option value="Spam">Spam</option>
                        <option value="Billing">Billing</option>
                        <option value="Work">Work</option>
                      </select>
                    </div>
                  )}
                </div>

                <div className="pt-3 flex justify-end">
                  <button
                    type="submit"
                    className="flex items-center gap-2 px-4 py-2 bg-emerald-600 hover:bg-emerald-500 text-white rounded-xl text-xs font-bold shadow-md shadow-emerald-600/20 transition-all"
                  >
                    <Plus className="w-3.5 h-3.5" />
                    Save Filter Rule
                  </button>
                </div>
              </form>
            </div>

            {/* Existing Filters List */}
            <div className="bg-white border border-slate-200/90 rounded-2xl p-6 space-y-4 shadow-xs">
              <h3 className="text-xs font-bold text-slate-700 uppercase tracking-wider">
                Active Filter Rules ({filters.length})
              </h3>

              {filters.length === 0 ? (
                <div className="text-center py-8 text-xs text-slate-400">
                  No automated filters configured yet. Incoming emails are placed directly in Inbox.
                </div>
              ) : (
                <div className="divide-y divide-slate-100">
                  {filters.map((f) => (
                    <div key={f.id} className="py-3 flex items-center justify-between text-xs">
                      <div>
                        <p className="font-semibold text-slate-900">{f.name}</p>
                        <p className="text-[11px] text-slate-500 mt-0.5">
                          If {f.criteria_field} contains{' '}
                          <code className="bg-slate-100 px-1.5 py-0.5 rounded text-emerald-700 font-mono">{f.criteria_pattern}</code>{' '}
                          &rarr; {f.action} {f.action_target ? `(${f.action_target})` : ''}
                        </p>
                      </div>
                      <button
                        onClick={() => handleDeleteFilter(f.id)}
                        className="p-1.5 text-slate-400 hover:text-rose-600 hover:bg-rose-50 rounded-lg transition-colors"
                        title="Delete filter"
                      >
                        <Trash2 className="w-4 h-4" />
                      </button>
                    </div>
                  ))}
                </div>
              )}
            </div>
          </div>
        )}

        {/* TAB 5: FORWARDING */}
        {activeTab === 'forwarding' && (
          <div className="bg-white border border-slate-200/90 rounded-2xl p-6 space-y-6 shadow-xs">
            <div>
              <h2 className="text-base font-bold text-slate-900 flex items-center gap-2">
                <Share2 className="w-4 h-4 text-emerald-600" />
                Automated Email Forwarding
              </h2>
              <p className="text-xs text-slate-500 mt-1">
                Automatically forward a copy of all incoming messages to an external email address (e.g. Gmail).
              </p>
            </div>

            <form onSubmit={handleSaveForwarding} className="space-y-4 max-w-xl">
              <div className="space-y-2">
                <label className="text-xs font-semibold text-slate-700">Forward All Incoming Emails To:</label>
                <input
                  type="email"
                  required
                  placeholder="e.g. mypersonalemail@gmail.com"
                  value={forwardDestination}
                  onChange={(e) => setForwardDestination(e.target.value)}
                  className="w-full bg-slate-50 border border-slate-200 rounded-xl px-3.5 py-2.5 text-xs text-slate-900 focus:outline-none focus:border-emerald-500"
                />
              </div>

              <div className="flex items-center gap-2 pt-1">
                <input
                  type="checkbox"
                  id="keep_copy"
                  checked={forwardKeepCopy}
                  onChange={(e) => setForwardKeepCopy(e.target.checked)}
                  className="rounded border-slate-300 text-emerald-600 focus:ring-emerald-500 w-4 h-4 cursor-pointer"
                />
                <label htmlFor="keep_copy" className="text-xs text-slate-700 cursor-pointer">
                  Keep a copy of forwarded messages in this Hostvra Inbox
                </label>
              </div>

              <div className="pt-4 border-t border-slate-100 flex items-center gap-3">
                <button
                  type="submit"
                  className="flex items-center gap-2 px-5 py-2.5 bg-emerald-600 hover:bg-emerald-500 text-white rounded-xl text-xs font-bold shadow-md shadow-emerald-600/20 transition-all"
                >
                  <Save className="w-3.5 h-3.5" />
                  Save Forwarding Rule
                </button>
                {forwardingRule && (
                  <button
                    type="button"
                    onClick={handleDeleteForwarding}
                    className="flex items-center gap-2 px-4 py-2.5 border border-rose-200 text-rose-600 hover:bg-rose-50 rounded-xl text-xs font-semibold transition-colors"
                  >
                    <Trash2 className="w-3.5 h-3.5" />
                    Disable Forwarding
                  </button>
                )}
              </div>
            </form>
          </div>
        )}

        {/* TAB 6: NOTIFICATIONS */}
        {activeTab === 'notifications' && (
          <div className="bg-white border border-slate-200/90 rounded-2xl p-6 space-y-6 shadow-xs">
            <div>
              <h2 className="text-base font-bold text-slate-900 flex items-center gap-2">
                <Bell className="w-4 h-4 text-emerald-600" />
                Notification Preferences
              </h2>
              <p className="text-xs text-slate-500 mt-1">
                Control audio alerts and native desktop notifications for incoming emails.
              </p>
            </div>

            <div className="space-y-4 max-w-2xl">
              <div className="flex items-center justify-between p-4 bg-slate-50 border border-slate-200 rounded-xl">
                <div className="flex items-center gap-3">
                  <div className="w-9 h-9 rounded-xl bg-white border border-slate-200 flex items-center justify-center text-emerald-600 shadow-xs">
                    {soundEnabled ? <Volume2 className="w-4 h-4" /> : <VolumeX className="w-4 h-4 text-slate-400" />}
                  </div>
                  <div>
                    <p className="text-xs font-bold text-slate-900">Audio Chime Alert</p>
                    <p className="text-[11px] text-slate-500">Play a pleasant chime when an email arrives in Inbox.</p>
                  </div>
                </div>
                <button
                  onClick={() => {
                    const next = !soundEnabled;
                    setSoundEnabled(next);
                    localStorage.setItem('hostvra_webmail_sound_enabled', String(next));
                    showNotification(`Sound alerts ${next ? 'enabled' : 'disabled'}`);
                  }}
                  className={`px-4 py-2 rounded-xl text-xs font-semibold transition-all ${
                    soundEnabled ? 'bg-emerald-600 text-white' : 'bg-slate-200 text-slate-700 hover:bg-slate-300'
                  }`}
                >
                  {soundEnabled ? 'Enabled' : 'Muted'}
                </button>
              </div>

              <div className="flex items-center justify-between p-4 bg-slate-50 border border-slate-200 rounded-xl">
                <div className="flex items-center gap-3">
                  <div className="w-9 h-9 rounded-xl bg-white border border-slate-200 flex items-center justify-center text-emerald-600 shadow-xs">
                    <Bell className="w-4 h-4" />
                  </div>
                  <div>
                    <p className="text-xs font-bold text-slate-900">Desktop Push Notifications</p>
                    <p className="text-[11px] text-slate-500">Show system banners even when the webmail tab is in background.</p>
                  </div>
                </div>
                <button
                  onClick={async () => {
                    await requestNotificationPermission();
                    showNotification('Desktop notification permission requested');
                  }}
                  className={`px-4 py-2 rounded-xl text-xs font-semibold transition-all ${
                    desktopNotifications ? 'bg-emerald-600 text-white' : 'bg-slate-200 text-slate-700 hover:bg-slate-300'
                  }`}
                >
                  {desktopNotifications ? 'Active' : 'Request Permission'}
                </button>
              </div>
            </div>
          </div>
        )}

        {/* TAB 7: SECURITY & ACCOUNTS */}
        {activeTab === 'security' && (
          <div className="bg-white border border-slate-200/90 rounded-2xl p-6 space-y-6 shadow-xs">
            <div>
              <h2 className="text-base font-bold text-slate-900 flex items-center gap-2">
                <Shield className="w-4 h-4 text-emerald-600" />
                Mailbox Security & Account Management
              </h2>
              <p className="text-xs text-slate-500 mt-1">
                Manage your mailbox password, active sessions and connected email accounts.
              </p>
            </div>

            {/* Change Password Form */}
            <form onSubmit={handleUpdatePassword} className="space-y-4 max-w-md pt-2">
              <h3 className="text-xs font-bold text-slate-800 uppercase tracking-wider flex items-center gap-2">
                <Lock className="w-3.5 h-3.5 text-emerald-600" /> Change Mailbox Password
              </h3>

              <div className="space-y-1.5">
                <label className="text-xs font-semibold text-slate-700">Current Password</label>
                <input
                  type="password"
                  required
                  placeholder="••••••••••••"
                  value={currentPassword}
                  onChange={(e) => setCurrentPassword(e.target.value)}
                  className="w-full bg-slate-50 border border-slate-200 rounded-xl px-3.5 py-2 text-xs text-slate-900 focus:outline-none focus:border-emerald-500"
                />
              </div>

              <div className="space-y-1.5">
                <label className="text-xs font-semibold text-slate-700">New Password</label>
                <input
                  type="password"
                  required
                  placeholder="•••••••••••• (min 8 characters)"
                  value={newPassword}
                  onChange={(e) => setNewPassword(e.target.value)}
                  className="w-full bg-slate-50 border border-slate-200 rounded-xl px-3.5 py-2 text-xs text-slate-900 focus:outline-none focus:border-emerald-500"
                />
              </div>

              <div className="space-y-1.5">
                <label className="text-xs font-semibold text-slate-700">Confirm New Password</label>
                <input
                  type="password"
                  required
                  placeholder="••••••••••••"
                  value={confirmPassword}
                  onChange={(e) => setConfirmPassword(e.target.value)}
                  className="w-full bg-slate-50 border border-slate-200 rounded-xl px-3.5 py-2 text-xs text-slate-900 focus:outline-none focus:border-emerald-500"
                />
              </div>

              <div className="pt-2">
                <button
                  type="submit"
                  className="flex items-center gap-2 px-5 py-2 bg-emerald-600 hover:bg-emerald-500 text-white rounded-xl text-xs font-bold shadow-md shadow-emerald-600/20 transition-all"
                >
                  <Lock className="w-3.5 h-3.5" />
                  Update Password
                </button>
              </div>
            </form>

            {/* Account Sign Out Actions */}
            <div className="pt-6 border-t border-slate-100 max-w-xl space-y-3">
              <h3 className="text-xs font-bold text-slate-800 uppercase tracking-wider">
                Sign Out & Session Controls
              </h3>
              <div className="flex flex-wrap gap-3">
                <button
                  type="button"
                  onClick={() => {
                    if (confirm('Sign out of the current mailbox?')) {
                      logoutCurrentAccount();
                    }
                  }}
                  className="flex items-center gap-2 px-4 py-2.5 rounded-xl border border-amber-200 bg-amber-50 text-amber-800 hover:bg-amber-100 text-xs font-semibold transition-colors"
                >
                  <LogOut className="w-4 h-4 text-amber-600" />
                  Sign Out of Current Mailbox
                </button>
                <button
                  type="button"
                  onClick={() => {
                    if (confirm('Sign out of all email accounts on this device?')) {
                      logoutAllAccounts();
                    }
                  }}
                  className="flex items-center gap-2 px-4 py-2.5 rounded-xl border border-rose-200 bg-rose-50 text-rose-800 hover:bg-rose-100 text-xs font-semibold transition-colors"
                >
                  <LogOut className="w-4 h-4 text-rose-600" />
                  Sign Out of All Accounts
                </button>
              </div>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
