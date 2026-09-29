'use client';

import React, { useState, useEffect } from 'react';
import Link from 'next/link';
import { X, Mail, Lock, AlertCircle, Loader2, Plus, Sparkles, ExternalLink, Check } from 'lucide-react';
import { useWebmail } from '@/context/WebmailContext';
import { apiFetch } from '@/lib/api';

interface ServerMailbox {
  id: string;
  email: string;
  name: string;
  quota_bytes: number;
  used_bytes: number;
}

export function WebmailAddAccountModal() {
  const { isAddAccountOpen, closeAddAccount, addAccount, accounts } = useWebmail();
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [rememberMe, setRememberMe] = useState(true);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Available server mailboxes from Hostvra Control Panel
  const [serverMailboxes, setServerMailboxes] = useState<ServerMailbox[]>([]);
  const [loadingMailboxes, setLoadingMailboxes] = useState(false);

  useEffect(() => {
    if (!isAddAccountOpen) return;

    let mounted = true;
    async function loadServerMailboxes() {
      setLoadingMailboxes(true);
      try {
        const res = await apiFetch<ServerMailbox[]>('/api/v1/email/mailboxes');
        if (mounted && res.data && Array.isArray(res.data)) {
          setServerMailboxes(res.data);
        }
      } catch (err) {
        // User might not have control panel token or endpoint not accessible
        console.debug('Failed to fetch server mailboxes:', err);
      } finally {
        if (mounted) setLoadingMailboxes(false);
      }
    }

    loadServerMailboxes();
    return () => {
      mounted = false;
    };
  }, [isAddAccountOpen]);

  if (!isAddAccountOpen) return null;

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!email.trim() || !password.trim()) {
      setError('Please provide email address and password');
      return;
    }

    setLoading(true);
    setError(null);

    try {
      const res = await apiFetch<{
        token: string;
        mailbox: {
          id: string;
          email: string;
          name: string;
          quota_bytes: number;
          used_bytes: number;
        };
      }>('/api/v1/webmail/auth', {
        method: 'POST',
        body: JSON.stringify({ email: email.trim().toLowerCase(), password }),
      });

      if (res.data && res.data.mailbox) {
        addAccount({
          id: res.data.mailbox.id,
          email: res.data.mailbox.email,
          name: res.data.mailbox.name || res.data.mailbox.email.split('@')[0],
          token: res.data.token,
          quotaBytes: res.data.mailbox.quota_bytes,
          usedBytes: res.data.mailbox.used_bytes,
        });
        setEmail('');
        setPassword('');
        closeAddAccount();
      } else {
        setError('Authentication failed. Please verify mailbox credentials.');
      }
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Invalid mailbox email or password.');
    } finally {
      setLoading(false);
    }
  };

  const handleSelectServerMailbox = (mb: ServerMailbox) => {
    setEmail(mb.email);
    setError(null);
  };

  const activeEmailSet = new Set(accounts.map((a) => a.email.toLowerCase()));

  return (
    <div className="fixed inset-0 z-50 bg-slate-900/40 backdrop-blur-xs flex items-center justify-center p-4 animate-in fade-in duration-150">
      <div className="w-full max-w-lg bg-white border border-slate-200 rounded-3xl shadow-2xl overflow-hidden p-6">
        <div className="flex items-center justify-between pb-4 border-b border-slate-100 mb-5">
          <div className="flex items-center gap-3">
            <div className="w-10 h-10 rounded-xl bg-emerald-50 text-emerald-600 flex items-center justify-center font-bold">
              <Mail className="w-5 h-5" />
            </div>
            <div>
              <h2 className="text-base font-bold text-slate-900">
                Connect Mailbox
              </h2>
              <p className="text-xs text-slate-500">
                Sign into your Hostvra business email address
              </p>
            </div>
          </div>
          <button
            onClick={closeAddAccount}
            className="p-1.5 text-slate-400 hover:text-slate-600 rounded-xl hover:bg-slate-100 transition-colors"
          >
            <X className="w-5 h-5" />
          </button>
        </div>

        {error && (
          <div className="mb-4 p-3 bg-rose-50 border border-rose-200 rounded-xl flex items-center gap-2.5 text-xs text-rose-700">
            <AlertCircle className="w-4 h-4 flex-shrink-0 text-rose-500" />
            <span>{error}</span>
          </div>
        )}

        {/* Server Mailboxes Quick Selector */}
        {serverMailboxes.length > 0 && (
          <div className="mb-5 p-3.5 bg-slate-50 border border-slate-200/80 rounded-2xl">
            <div className="flex items-center justify-between mb-2">
              <span className="text-[11px] font-bold text-slate-700 uppercase tracking-wider flex items-center gap-1.5">
                <Sparkles className="w-3.5 h-3.5 text-emerald-600" />
                Detected Mailboxes On Your Server
              </span>
              <span className="text-[10px] text-slate-400">Click to select</span>
            </div>
            <div className="space-y-1.5 max-h-36 overflow-y-auto pr-1">
              {serverMailboxes.map((mb) => {
                const isAlreadyConnected = activeEmailSet.has(mb.email.toLowerCase());
                const isSelected = email.toLowerCase() === mb.email.toLowerCase();

                return (
                  <button
                    key={mb.id}
                    type="button"
                    onClick={() => handleSelectServerMailbox(mb)}
                    className={`w-full flex items-center justify-between p-2 rounded-xl text-xs transition-colors text-left border ${
                      isSelected
                        ? 'bg-emerald-50 border-emerald-300 text-emerald-800 font-bold'
                        : isAlreadyConnected
                        ? 'bg-white border-slate-200 text-slate-500 hover:border-slate-300'
                        : 'bg-white border-slate-200 text-slate-800 hover:border-emerald-300'
                    }`}
                  >
                    <div className="flex items-center gap-2 min-w-0">
                      <div className="w-6 h-6 rounded-lg bg-emerald-100 text-emerald-700 flex items-center justify-center font-bold text-[10px]">
                        @
                      </div>
                      <span className="truncate">{mb.email}</span>
                    </div>
                    {isAlreadyConnected ? (
                      <span className="text-[10px] px-2 py-0.5 rounded-full bg-slate-100 text-slate-500 font-semibold flex items-center gap-1">
                        <Check className="w-3 h-3" /> Connected
                      </span>
                    ) : (
                      <span className="text-[10px] text-emerald-600 font-semibold">
                        Select
                      </span>
                    )}
                  </button>
                );
              })}
            </div>
          </div>
        )}

        <form onSubmit={handleSubmit} className="space-y-4">
          <div>
            <label className="block text-xs font-semibold text-slate-700 mb-1.5">
              Email Address
            </label>
            <div className="relative">
              <div className="absolute inset-y-0 left-0 pl-3.5 flex items-center pointer-events-none text-slate-400">
                <Mail className="w-4 h-4" />
              </div>
              <input
                type="email"
                required
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                placeholder="e.g. info@yourdomain.com"
                className="w-full pl-10 pr-3.5 py-2.5 bg-slate-50 border border-slate-200 rounded-xl text-xs text-slate-900 focus:outline-none focus:border-emerald-500 focus:bg-white focus:ring-1 focus:ring-emerald-500 transition-colors"
              />
            </div>
          </div>

          <div>
            <label className="block text-xs font-semibold text-slate-700 mb-1.5">
              Mailbox Password
            </label>
            <div className="relative">
              <div className="absolute inset-y-0 left-0 pl-3.5 flex items-center pointer-events-none text-slate-400">
                <Lock className="w-4 h-4" />
              </div>
              <input
                type="password"
                required
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                placeholder="••••••••••••"
                className="w-full pl-10 pr-3.5 py-2.5 bg-slate-50 border border-slate-200 rounded-xl text-xs text-slate-900 focus:outline-none focus:border-emerald-500 focus:bg-white focus:ring-1 focus:ring-emerald-500 transition-colors"
              />
            </div>
          </div>

          <div className="flex items-center justify-between text-xs pt-1">
            <label className="flex items-center gap-2 cursor-pointer text-slate-600">
              <input
                type="checkbox"
                checked={rememberMe}
                onChange={(e) => setRememberMe(e.target.checked)}
                className="w-4 h-4 rounded text-emerald-600 focus:ring-emerald-500 border-slate-300 cursor-pointer"
              />
              <span>Keep mailbox session saved</span>
            </label>
          </div>

          <div className="pt-3 flex items-center justify-between border-t border-slate-100">
            <Link
              href="/email"
              onClick={closeAddAccount}
              className="text-xs text-emerald-600 hover:text-emerald-700 font-semibold inline-flex items-center gap-1"
            >
              <span>Create New Mailbox</span>
              <ExternalLink className="w-3 h-3" />
            </Link>

            <div className="flex items-center gap-2">
              <button
                type="button"
                onClick={closeAddAccount}
                className="px-4 py-2.5 rounded-xl border border-slate-200 text-xs font-semibold text-slate-600 hover:bg-slate-50 transition-colors"
              >
                Cancel
              </button>
              <button
                type="submit"
                disabled={loading}
                className="flex items-center gap-2 px-5 py-2.5 bg-gradient-to-r from-emerald-600 to-teal-600 hover:from-emerald-500 hover:to-teal-500 text-white font-bold text-xs rounded-xl shadow-md shadow-emerald-600/20 disabled:opacity-50 transition-all hover:scale-[1.02] active:scale-[0.98]"
              >
                {loading ? (
                  <>
                    <Loader2 className="w-4 h-4 animate-spin" />
                    <span>Signing in...</span>
                  </>
                ) : (
                  <span>Connect Mailbox</span>
                )}
              </button>
            </div>
          </div>
        </form>
      </div>
    </div>
  );
}
