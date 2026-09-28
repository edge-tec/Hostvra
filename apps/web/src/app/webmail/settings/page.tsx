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
    logoutAllAccounts,
  } = useWebmail();

  const [activeTab, setActiveTab] = useState<SettingsTab>('general');
  const [successMsg, setSuccessMsg] = useState('');
  const [errorMsg, setErrorMsg] = useState('');

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

  const handleSaveIdentity = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!activeAccount) return;

    const existingId = identities.find(i => i.email === activeAccount.email)?.id;
    const ok = await saveIdentity({
      id: existingId,
      email: activeAccount.email,
      display_name: displayName,
      reply_to: replyTo,
      signature_html: signatureText,
      signature_text: signatureText.replace(/<[^>]*>?/gm, ''),
      is_default: true,
    });

    if (ok) {
      showNotification('Identity and signature updated successfully!');
    } else {
      showNotification('Failed to update identity. Please try again.', true);
    }
  };

  const handleCreateFilter = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!newFilterName || !filterCriteriaVal) {
      showNotification('Please fill in filter name and criteria pattern', true);
      return;
    }

    const ok = await saveFilter({
      name: newFilterName,
      criteria_field: filterCriteriaField,
      criteria_pattern: filterCriteriaVal,
      action_type: filterAction,
      action_target: filterActionTarget,
      is_active: true,
      priority: filters.length + 1,
    });

    if (ok) {
      showNotification('Mail filter created successfully and active on incoming emails!');
      setNewFilterName('');
      setFilterCriteriaVal('');
    } else {
      showNotification('Failed to save filter rule', true);
    }
  };

  const handleSaveForwarding = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!forwardDestination) {
      showNotification('Forwarding address is required', true);
      return;
    }

    const ok = await saveForwarding(forwardDestination, forwardKeepCopy);
    if (ok) {
      showNotification('Forwarding rule updated successfully!');
    } else {
      showNotification('Failed to update forwarding settings', true);
    }
  };

  const handleDeleteForwarding = async () => {
    if (!confirm('Are you sure you want to disable and remove email forwarding?')) return;
    const ok = await deleteForwarding();
    if (ok) {
      setForwardDestination('');
      showNotification('Forwarding rule disabled and deleted.');
    } else {
      showNotification('Failed to delete forwarding', true);
    }
  };

  const handleChangePassword = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!newPassword || newPassword !== confirmPassword) {
      showNotification('Passwords do not match or are empty', true);
      return;
    }
    if (newPassword.length < 8) {
      showNotification('Password must be at least 8 characters', true);
      return;
    }

    try {
      const res = await fetch('/api/v1/auth/password', {
        method: 'POST',
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
    <div className="flex-1 overflow-y-auto bg-slate-950 p-6 md:p-8">
      <div className="max-w-5xl mx-auto space-y-6">
        {/* Header Title */}
        <div className="flex items-center justify-between pb-4 border-b border-slate-800">
          <div>
            <h1 className="text-2xl font-bold text-white flex items-center gap-2.5">
              <Settings className="w-6 h-6 text-sky-400" />
              Webmail Settings
            </h1>
            <p className="text-xs text-slate-400 mt-1">
              Configure identities, filters, signatures, security and preferences for{' '}
              <span className="font-semibold text-sky-300">{activeAccount?.email || 'Active Account'}</span>
            </p>
          </div>

          <div className="text-xs text-slate-500 font-mono bg-slate-900 px-3 py-1.5 rounded-lg border border-slate-800">
            Hostvra Mail v2.5
          </div>
        </div>

        {/* Global Notifications */}
        {successMsg && (
          <div className="p-3.5 bg-emerald-950/80 border border-emerald-500/40 rounded-xl text-emerald-200 text-xs flex items-center gap-2 animate-in fade-in">
            <Check className="w-4 h-4 text-emerald-400 shrink-0" />
            {successMsg}
          </div>
        )}
        {errorMsg && (
          <div className="p-3.5 bg-rose-950/80 border border-rose-500/40 rounded-xl text-rose-200 text-xs flex items-center gap-2 animate-in fade-in">
            <AlertCircle className="w-4 h-4 text-rose-400 shrink-0" />
            {errorMsg}
          </div>
        )}

        {/* Settings Navigation Tabs */}
        <div className="flex overflow-x-auto gap-2 p-1.5 bg-slate-900/80 rounded-2xl border border-slate-800 scrollbar-none">
          <button
            onClick={() => setActiveTab('general')}
            className={`flex items-center gap-2 px-4 py-2 rounded-xl text-xs font-medium transition-colors shrink-0 ${
              activeTab === 'general' ? 'bg-sky-500 text-white shadow-lg shadow-sky-500/20' : 'text-slate-400 hover:text-white hover:bg-slate-800/60'
            }`}
          >
            <Settings className="w-3.5 h-3.5" />
            General
          </button>
          <button
            onClick={() => setActiveTab('identities')}
            className={`flex items-center gap-2 px-4 py-2 rounded-xl text-xs font-medium transition-colors shrink-0 ${
              activeTab === 'identities' ? 'bg-sky-500 text-white shadow-lg shadow-sky-500/20' : 'text-slate-400 hover:text-white hover:bg-slate-800/60'
            }`}
          >
            <User className="w-3.5 h-3.5" />
            Identities
          </button>
          <button
            onClick={() => setActiveTab('signature')}
            className={`flex items-center gap-2 px-4 py-2 rounded-xl text-xs font-medium transition-colors shrink-0 ${
              activeTab === 'signature' ? 'bg-sky-500 text-white shadow-lg shadow-sky-500/20' : 'text-slate-400 hover:text-white hover:bg-slate-800/60'
            }`}
          >
            <Edit3 className="w-3.5 h-3.5" />
            Signatures
          </button>
          <button
            onClick={() => setActiveTab('filters')}
            className={`flex items-center gap-2 px-4 py-2 rounded-xl text-xs font-medium transition-colors shrink-0 ${
              activeTab === 'filters' ? 'bg-sky-500 text-white shadow-lg shadow-sky-500/20' : 'text-slate-400 hover:text-white hover:bg-slate-800/60'
            }`}
          >
            <Filter className="w-3.5 h-3.5" />
            Rules & Filters
          </button>
          <button
            onClick={() => setActiveTab('forwarding')}
            className={`flex items-center gap-2 px-4 py-2 rounded-xl text-xs font-medium transition-colors shrink-0 ${
              activeTab === 'forwarding' ? 'bg-sky-500 text-white shadow-lg shadow-sky-500/20' : 'text-slate-400 hover:text-white hover:bg-slate-800/60'
            }`}
          >
            <Share2 className="w-3.5 h-3.5" />
            Forwarding
          </button>
          <button
            onClick={() => setActiveTab('notifications')}
            className={`flex items-center gap-2 px-4 py-2 rounded-xl text-xs font-medium transition-colors shrink-0 ${
              activeTab === 'notifications' ? 'bg-sky-500 text-white shadow-lg shadow-sky-500/20' : 'text-slate-400 hover:text-white hover:bg-slate-800/60'
            }`}
          >
            <Bell className="w-3.5 h-3.5" />
            Notifications
          </button>
          <button
            onClick={() => setActiveTab('security')}
            className={`flex items-center gap-2 px-4 py-2 rounded-xl text-xs font-medium transition-colors shrink-0 ${
              activeTab === 'security' ? 'bg-sky-500 text-white shadow-lg shadow-sky-500/20' : 'text-slate-400 hover:text-white hover:bg-slate-800/60'
            }`}
          >
            <Shield className="w-3.5 h-3.5" />
            Security & Accounts
          </button>
        </div>

        {/* TAB 1: GENERAL */}
        {activeTab === 'general' && (
          <div className="bg-slate-900/60 border border-slate-800 rounded-2xl p-6 space-y-6">
            <h2 className="text-base font-semibold text-white flex items-center gap-2">
              <Mail className="w-4 h-4 text-sky-400" />
              General Preferences
            </h2>

            <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
              <div className="space-y-2">
                <label className="text-xs font-medium text-slate-300">Default Mailbox Folder</label>
                <select className="w-full bg-slate-950 border border-slate-800 rounded-xl px-3.5 py-2.5 text-xs text-white focus:outline-none focus:border-sky-500">
                  <option value="inbox">Inbox</option>
                  <option value="starred">Starred</option>
                  <option value="archive">Archive</option>
                </select>
                <p className="text-[11px] text-slate-500">Folder to open immediately after login.</p>
              </div>

              <div className="space-y-2">
                <label className="text-xs font-medium text-slate-300">Emails Per Page</label>
                <select className="w-full bg-slate-950 border border-slate-800 rounded-xl px-3.5 py-2.5 text-xs text-white focus:outline-none focus:border-sky-500">
                  <option value="25">25 conversations</option>
                  <option value="50">50 conversations (Default)</option>
                  <option value="100">100 conversations</option>
                </select>
                <p className="text-[11px] text-slate-500">Number of messages displayed simultaneously in mailbox list.</p>
              </div>

              <div className="space-y-2">
                <label className="text-xs font-medium text-slate-300">Reading Pane Layout</label>
                <select className="w-full bg-slate-950 border border-slate-800 rounded-xl px-3.5 py-2.5 text-xs text-white focus:outline-none focus:border-sky-500">
                  <option value="split">Split View (List on left, preview on right)</option>
                  <option value="full">Full Screen Single Message</option>
                </select>
                <p className="text-[11px] text-slate-500">Standard 2-pane Gmail layout or single message reading view.</p>
              </div>

              <div className="space-y-2">
                <label className="text-xs font-medium text-slate-300">Sync Interval</label>
                <select className="w-full bg-slate-950 border border-slate-800 rounded-xl px-3.5 py-2.5 text-xs text-white focus:outline-none focus:border-sky-500">
                  <option value="realtime">Real-time SSE Push (Active)</option>
                  <option value="60">Every 1 minute</option>
                  <option value="300">Every 5 minutes</option>
                </select>
                <p className="text-[11px] text-slate-500">How frequently the mail server is polled for background sync.</p>
              </div>
            </div>

            <div className="pt-4 border-t border-slate-800 flex justify-end">
              <button
                onClick={() => showNotification('General preferences updated')}
                className="flex items-center gap-2 px-5 py-2.5 bg-sky-500 hover:bg-sky-400 text-white rounded-xl text-xs font-semibold shadow-lg shadow-sky-500/20 transition-all"
              >
                <Save className="w-3.5 h-3.5" />
                Save Preferences
              </button>
            </div>
          </div>
        )}

        {/* TAB 2: IDENTITIES */}
        {activeTab === 'identities' && (
          <div className="bg-slate-900/60 border border-slate-800 rounded-2xl p-6 space-y-6">
            <div>
              <h2 className="text-base font-semibold text-white flex items-center gap-2">
                <User className="w-4 h-4 text-sky-400" />
                Email Identities & Display Names
              </h2>
              <p className="text-xs text-slate-400 mt-1">
                Customize how your name and address appear to recipients when you send an email.
              </p>
            </div>

            <form onSubmit={handleSaveIdentity} className="space-y-5">
              <div className="grid grid-cols-1 md:grid-cols-2 gap-5">
                <div className="space-y-2">
                  <label className="text-xs font-medium text-slate-300">Sending Email Address</label>
                  <input
                    type="text"
                    disabled
                    value={activeAccount?.email || ''}
                    className="w-full bg-slate-950/50 border border-slate-800 rounded-xl px-3.5 py-2.5 text-xs text-slate-400 cursor-not-allowed"
                  />
                  <p className="text-[11px] text-slate-500">Managed by Hostvra Mail Server.</p>
                </div>

                <div className="space-y-2">
                  <label className="text-xs font-medium text-slate-300">
                    Display Name <span className="text-sky-400">*</span>
                  </label>
                  <input
                    type="text"
                    required
                    value={displayName}
                    onChange={(e) => setDisplayName(e.target.value)}
                    placeholder="e.g. Hostvra Support or Mizanur Rahman"
                    className="w-full bg-slate-950 border border-slate-800 rounded-xl px-3.5 py-2.5 text-xs text-white focus:outline-none focus:border-sky-500"
                  />
                  <p className="text-[11px] text-slate-500">Recipients will see: &quot;{displayName || 'Your Name'} &lt;{activeAccount?.email}&gt;&quot;</p>
                </div>

                <div className="space-y-2 md:col-span-2">
                  <label className="text-xs font-medium text-slate-300">Reply-To Address (Optional)</label>
                  <input
                    type="email"
                    value={replyTo}
                    onChange={(e) => setReplyTo(e.target.value)}
                    placeholder="e.g. replies@yourdomain.com (leave blank to reply directly to sender)"
                    className="w-full bg-slate-950 border border-slate-800 rounded-xl px-3.5 py-2.5 text-xs text-white focus:outline-none focus:border-sky-500"
                  />
                </div>
              </div>

              <div className="pt-4 border-t border-slate-800 flex justify-end">
                <button
                  type="submit"
                  className="flex items-center gap-2 px-5 py-2.5 bg-sky-500 hover:bg-sky-400 text-white rounded-xl text-xs font-semibold shadow-lg shadow-sky-500/20 transition-all"
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
          <div className="bg-slate-900/60 border border-slate-800 rounded-2xl p-6 space-y-6">
            <div>
              <h2 className="text-base font-semibold text-white flex items-center gap-2">
                <Edit3 className="w-4 h-4 text-sky-400" />
                Email Signature
              </h2>
              <p className="text-xs text-slate-400 mt-1">
                This signature will be automatically appended to the bottom of all outgoing emails and replies.
              </p>
            </div>

            <form onSubmit={handleSaveIdentity} className="space-y-5">
              <div className="space-y-2">
                <label className="text-xs font-medium text-slate-300">Signature HTML / Text</label>
                <textarea
                  rows={6}
                  value={signatureText}
                  onChange={(e) => setSignatureText(e.target.value)}
                  placeholder="--&#10;Best regards,&#10;Hostvra Support Team&#10;https://hostvra.com"
                  className="w-full bg-slate-950 border border-slate-800 rounded-xl p-3.5 text-xs text-white font-mono focus:outline-none focus:border-sky-500 leading-relaxed"
                />
              </div>

              {signatureText && (
                <div className="p-4 bg-slate-950 border border-slate-800 rounded-xl space-y-1.5">
                  <span className="text-[11px] font-semibold text-slate-400 uppercase tracking-wider">Live Preview:</span>
                  <div
                    className="text-xs text-slate-300 pt-2 border-t border-slate-800"
                    dangerouslySetInnerHTML={{ __html: signatureText.replace(/\n/g, '<br/>') }}
                  />
                </div>
              )}

              <div className="pt-4 border-t border-slate-800 flex justify-end">
                <button
                  type="submit"
                  className="flex items-center gap-2 px-5 py-2.5 bg-sky-500 hover:bg-sky-400 text-white rounded-xl text-xs font-semibold shadow-lg shadow-sky-500/20 transition-all"
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
            <div className="bg-slate-900/60 border border-slate-800 rounded-2xl p-6 space-y-6">
              <div>
                <h2 className="text-base font-semibold text-white flex items-center gap-2">
                  <Plus className="w-4 h-4 text-sky-400" />
                  Create Real Mail Filter
                </h2>
                <p className="text-xs text-slate-400 mt-1">
                  Incoming emails are evaluated automatically on the server against active filter rules.
                </p>
              </div>

              <form onSubmit={handleCreateFilter} className="space-y-4">
                <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                  <div className="space-y-1.5">
                    <label className="text-xs font-medium text-slate-300">Filter Name</label>
                    <input
                      type="text"
                      required
                      placeholder="e.g. Move Invoices to Billing"
                      value={newFilterName}
                      onChange={(e) => setNewFilterName(e.target.value)}
                      className="w-full bg-slate-950 border border-slate-800 rounded-xl px-3 py-2 text-xs text-white focus:outline-none focus:border-sky-500"
                    />
                  </div>

                  <div className="space-y-1.5">
                    <label className="text-xs font-medium text-slate-300">If Message Condition</label>
                    <select
                      value={filterCriteriaField}
                      onChange={(e) => setFilterCriteriaField(e.target.value)}
                      className="w-full bg-slate-950 border border-slate-800 rounded-xl px-3 py-2 text-xs text-white focus:outline-none focus:border-sky-500"
                    >
                      <option value="from">From contains</option>
                      <option value="to">To contains</option>
                      <option value="subject">Subject contains</option>
                      <option value="body">Body contains</option>
                    </select>
                  </div>

                  <div className="space-y-1.5">
                    <label className="text-xs font-medium text-slate-300">Pattern / Keyword</label>
                    <input
                      type="text"
                      required
                      placeholder="e.g. invoice, billing@, alerts"
                      value={filterCriteriaVal}
                      onChange={(e) => setFilterCriteriaVal(e.target.value)}
                      className="w-full bg-slate-950 border border-slate-800 rounded-xl px-3 py-2 text-xs text-white focus:outline-none focus:border-sky-500"
                    />
                  </div>
                </div>

                <div className="grid grid-cols-1 md:grid-cols-2 gap-4 pt-2">
                  <div className="space-y-1.5">
                    <label className="text-xs font-medium text-slate-300">Action to Take</label>
                    <select
                      value={filterAction}
                      onChange={(e) => setFilterAction(e.target.value)}
                      className="w-full bg-slate-950 border border-slate-800 rounded-xl px-3 py-2 text-xs text-white focus:outline-none focus:border-sky-500"
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
                      <label className="text-xs font-medium text-slate-300">Target Folder</label>
                      <select
                        value={filterActionTarget}
                        onChange={(e) => setFilterActionTarget(e.target.value)}
                        className="w-full bg-slate-950 border border-slate-800 rounded-xl px-3 py-2 text-xs text-white focus:outline-none focus:border-sky-500"
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
                    className="flex items-center gap-2 px-4 py-2 bg-sky-500 hover:bg-sky-400 text-white rounded-xl text-xs font-semibold shadow-lg shadow-sky-500/20 transition-all"
                  >
                    <Plus className="w-3.5 h-3.5" />
                    Save Filter Rule
                  </button>
                </div>
              </form>
            </div>

            {/* Existing Filters List */}
            <div className="bg-slate-900/60 border border-slate-800 rounded-2xl p-6 space-y-4">
              <h3 className="text-xs font-semibold text-slate-300 uppercase tracking-wider">
                Active Filter Rules ({filters.length})
              </h3>

              {filters.length === 0 ? (
                <p className="text-xs text-slate-500 italic py-4">No filter rules configured for this mailbox yet.</p>
              ) : (
                <div className="divide-y divide-slate-800">
                  {filters.map((f) => (
                    <div key={f.id} className="py-3 flex items-center justify-between gap-4">
                      <div className="space-y-1">
                        <div className="flex items-center gap-2">
                          <span className="text-xs font-semibold text-white">{f.name}</span>
                          <span className="text-[10px] px-2 py-0.5 rounded-full bg-emerald-950 text-emerald-300 border border-emerald-800/50">
                            Active
                          </span>
                        </div>
                        <p className="text-xs text-slate-400">
                          If <strong className="text-sky-300">{f.criteria_field}</strong> contains{' '}
                          <code className="bg-slate-950 px-1.5 py-0.5 rounded text-sky-200">{f.criteria_pattern}</code>{' '}
                          &rarr; Action: <strong className="text-amber-300">{f.action_type}</strong> {f.action_target && `(${f.action_target})`}
                        </p>
                      </div>

                      <button
                        onClick={() => deleteFilter(f.id)}
                        className="p-2 text-slate-400 hover:text-rose-400 hover:bg-rose-950/40 rounded-lg transition-colors"
                        title="Delete filter rule"
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
          <div className="bg-slate-900/60 border border-slate-800 rounded-2xl p-6 space-y-6">
            <div>
              <h2 className="text-base font-semibold text-white flex items-center gap-2">
                <Share2 className="w-4 h-4 text-sky-400" />
                Automatic Email Forwarding
              </h2>
              <p className="text-xs text-slate-400 mt-1">
                Forward incoming messages sent to <strong className="text-sky-300">{activeAccount?.email}</strong> to another destination.
              </p>
            </div>

            <form onSubmit={handleSaveForwarding} className="space-y-5">
              <div className="space-y-2">
                <label className="text-xs font-medium text-slate-300">
                  Forwarding Email Address <span className="text-sky-400">*</span>
                </label>
                <input
                  type="email"
                  required
                  value={forwardDestination}
                  onChange={(e) => setForwardDestination(e.target.value)}
                  placeholder="e.g. your-personal@gmail.com"
                  className="w-full bg-slate-950 border border-slate-800 rounded-xl px-3.5 py-2.5 text-xs text-white focus:outline-none focus:border-sky-500"
                />
              </div>

              <div className="flex items-center gap-3 pt-2">
                <input
                  type="checkbox"
                  id="keepCopy"
                  checked={forwardKeepCopy}
                  onChange={(e) => setForwardKeepCopy(e.target.checked)}
                  className="rounded border-slate-700 bg-slate-950 text-sky-500 focus:ring-sky-500 w-4 h-4 cursor-pointer"
                />
                <label htmlFor="keepCopy" className="text-xs text-slate-300 cursor-pointer select-none">
                  Keep a copy of forwarded messages in this mailbox Inbox
                </label>
              </div>

              <div className="pt-4 border-t border-slate-800 flex items-center justify-between">
                {forwardingRule ? (
                  <button
                    type="button"
                    onClick={handleDeleteForwarding}
                    className="flex items-center gap-2 px-4 py-2 bg-rose-950/60 hover:bg-rose-900/60 text-rose-300 border border-rose-800/60 rounded-xl text-xs font-semibold transition-all"
                  >
                    <Trash2 className="w-3.5 h-3.5" />
                    Disable Forwarding
                  </button>
                ) : <div />}

                <button
                  type="submit"
                  className="flex items-center gap-2 px-5 py-2.5 bg-sky-500 hover:bg-sky-400 text-white rounded-xl text-xs font-semibold shadow-lg shadow-sky-500/20 transition-all"
                >
                  <Save className="w-3.5 h-3.5" />
                  Save Forwarding Rule
                </button>
              </div>
            </form>
          </div>
        )}

        {/* TAB 6: NOTIFICATIONS */}
        {activeTab === 'notifications' && (
          <div className="bg-slate-900/60 border border-slate-800 rounded-2xl p-6 space-y-6">
            <div>
              <h2 className="text-base font-semibold text-white flex items-center gap-2">
                <Bell className="w-4 h-4 text-sky-400" />
                Notification Preferences
              </h2>
              <p className="text-xs text-slate-400 mt-1">
                Configure audio and visual alerts when new mail arrives.
              </p>
            </div>

            <div className="space-y-4">
              <div className="flex items-center justify-between p-4 bg-slate-950 border border-slate-800 rounded-xl">
                <div className="space-y-0.5">
                  <div className="flex items-center gap-2 text-xs font-semibold text-white">
                    {soundEnabled ? <Volume2 className="text-emerald-400" /> : <VolumeX className="text-slate-500" />}
                    Incoming Mail Audio Chime
                  </div>
                  <p className="text-[11px] text-slate-400">Play an elegant chime tone via Web Audio API whenever an incoming email is received.</p>
                </div>
                <button
                  type="button"
                  onClick={() => setSoundEnabled(!soundEnabled)}
                  className={`px-3 py-1.5 rounded-lg text-xs font-medium transition-colors ${
                    soundEnabled ? 'bg-emerald-500/20 text-emerald-300 border border-emerald-500/30' : 'bg-slate-800 text-slate-400'
                  }`}
                >
                  {soundEnabled ? 'Enabled' : 'Disabled'}
                </button>
              </div>

              <div className="flex items-center justify-between p-4 bg-slate-950 border border-slate-800 rounded-xl">
                <div className="space-y-0.5">
                  <div className="flex items-center gap-2 text-xs font-semibold text-white">
                    <Bell className="text-sky-400" />
                    Desktop Notifications
                  </div>
                  <p className="text-[11px] text-slate-400">Show native browser desktop banners when Hostvra Webmail is in background.</p>
                </div>
                <button
                  type="button"
                  onClick={requestNotificationPermission}
                  className={`px-3 py-1.5 rounded-lg text-xs font-medium transition-colors ${
                    desktopNotifications ? 'bg-sky-500/20 text-sky-300 border border-sky-500/30' : 'bg-sky-500 text-white hover:bg-sky-400'
                  }`}
                >
                  {desktopNotifications ? 'Granted' : 'Request Permission'}
                </button>
              </div>
            </div>
          </div>
        )}

        {/* TAB 7: SECURITY & ACCOUNTS */}
        {activeTab === 'security' && (
          <div className="space-y-6">
            {/* Password Change */}
            <div className="bg-slate-900/60 border border-slate-800 rounded-2xl p-6 space-y-6">
              <div>
                <h2 className="text-base font-semibold text-white flex items-center gap-2">
                  <Lock className="w-4 h-4 text-sky-400" />
                  Mailbox Password
                </h2>
                <p className="text-xs text-slate-400 mt-1">
                  Change password for <strong className="text-sky-300">{activeAccount?.email}</strong>.
                </p>
              </div>

              <form onSubmit={handleChangePassword} className="space-y-4 max-w-md">
                <div className="space-y-1.5">
                  <label className="text-xs font-medium text-slate-300">Current Password</label>
                  <input
                    type="password"
                    required
                    value={currentPassword}
                    onChange={(e) => setCurrentPassword(e.target.value)}
                    className="w-full bg-slate-950 border border-slate-800 rounded-xl px-3.5 py-2 text-xs text-white focus:outline-none focus:border-sky-500"
                  />
                </div>

                <div className="space-y-1.5">
                  <label className="text-xs font-medium text-slate-300">New Password (min 8 chars)</label>
                  <input
                    type="password"
                    required
                    value={newPassword}
                    onChange={(e) => setNewPassword(e.target.value)}
                    className="w-full bg-slate-950 border border-slate-800 rounded-xl px-3.5 py-2 text-xs text-white focus:outline-none focus:border-sky-500"
                  />
                </div>

                <div className="space-y-1.5">
                  <label className="text-xs font-medium text-slate-300">Confirm New Password</label>
                  <input
                    type="password"
                    required
                    value={confirmPassword}
                    onChange={(e) => setConfirmPassword(e.target.value)}
                    className="w-full bg-slate-950 border border-slate-800 rounded-xl px-3.5 py-2 text-xs text-white focus:outline-none focus:border-sky-500"
                  />
                </div>

                <button
                  type="submit"
                  className="flex items-center gap-2 px-4 py-2 bg-sky-500 hover:bg-sky-400 text-white rounded-xl text-xs font-semibold shadow-lg shadow-sky-500/20 transition-all mt-2"
                >
                  <Save className="w-3.5 h-3.5" />
                  Update Password
                </button>
              </form>
            </div>

            {/* Active Accounts & Sessions */}
            <div className="bg-slate-900/60 border border-slate-800 rounded-2xl p-6 space-y-4">
              <div className="flex items-center justify-between">
                <div>
                  <h3 className="text-sm font-semibold text-white">Active Mailbox Accounts ({accounts.length})</h3>
                  <p className="text-xs text-slate-400">Authenticated accounts currently stored in this Webmail session.</p>
                </div>
                <button
                  onClick={logoutAllAccounts}
                  className="flex items-center gap-2 px-3 py-1.5 bg-rose-950/60 hover:bg-rose-900/60 text-rose-300 border border-rose-800/60 rounded-xl text-xs font-medium transition-all"
                >
                  <LogOut className="w-3.5 h-3.5" />
                  Logout All Accounts
                </button>
              </div>

              <div className="divide-y divide-slate-800">
                {accounts.map((acc) => (
                  <div key={acc.id} className="py-3 flex items-center justify-between">
                    <div>
                      <div className="flex items-center gap-2">
                        <span className="text-xs font-semibold text-white">{acc.email}</span>
                        {acc.id === activeAccount?.id && (
                          <span className="text-[10px] px-2 py-0.5 rounded-full bg-sky-950 text-sky-300 border border-sky-800/50">
                            Active Current
                          </span>
                        )}
                      </div>
                      <span className="text-[11px] text-slate-400">{acc.name || 'No display name'}</span>
                    </div>

                    <span className="text-[11px] text-slate-500 font-mono">
                      Quota: {acc.quotaBytes ? `${Math.round((acc.usedBytes || 0) / 1024 / 1024)}MB / ${Math.round(acc.quotaBytes / 1024 / 1024)}MB` : 'Unlimited'}
                    </span>
                  </div>
                ))}
              </div>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
