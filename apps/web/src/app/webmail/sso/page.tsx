'use client';

import React, { Suspense, useEffect, useState } from 'react';
import { useRouter, useSearchParams } from 'next/navigation';
import Link from 'next/link';
import { apiFetch, setStoredWebmailToken } from '@/lib/api';
import { useWebmail } from '@/context/WebmailContext';
import { Mail, ShieldCheck, AlertCircle, ArrowLeft, RefreshCw } from 'lucide-react';

function WebmailSSOContent() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const { verifySession } = useWebmail();

  const [status, setStatus] = useState<'validating' | 'success' | 'error'>('validating');
  const [errorMessage, setErrorMessage] = useState<string>('');

  useEffect(() => {
    const ticket = searchParams.get('ticket');
    if (!ticket) {
      setStatus('error');
      setErrorMessage('Missing SSO security ticket. Please launch Webmail from your Hosting Control Panel.');
      return;
    }

    let isMounted = true;

    async function validateTicket() {
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
        }>('/api/v1/webmail/sso/validate', {
          method: 'POST',
          body: JSON.stringify({ ticket }),
        });

        if (!isMounted) return;

        if (res.data && res.data.token && res.data.mailbox) {
          setStoredWebmailToken(res.data.token);
          await verifySession();
          setStatus('success');
          router.replace('/webmail/inbox');
        } else {
          setStatus('error');
          setErrorMessage(res.error?.message || 'The SSO token is invalid, expired, or has already been used.');
        }
      } catch (err: unknown) {
        if (!isMounted) return;
        setStatus('error');
        setErrorMessage(err instanceof Error ? err.message : 'Failed to validate Webmail SSO session.');
      }
    }

    validateTicket();

    return () => {
      isMounted = false;
    };
  }, [searchParams, router, verifySession]);

  return (
    <div className="min-h-screen h-screen flex flex-col items-center justify-center bg-slate-900 text-white font-sans p-4 select-none">
      <div className="w-full max-w-md bg-slate-800/80 border border-slate-700/60 backdrop-blur-xl rounded-3xl p-8 shadow-2xl text-center space-y-6 animate-in fade-in">
        {/* Brand Header */}
        <div className="inline-flex items-center justify-center w-16 h-16 rounded-2xl bg-gradient-to-tr from-emerald-500 to-teal-500 text-white shadow-xl shadow-emerald-500/20 mx-auto">
          <Mail className="w-8 h-8" />
        </div>

        <div>
          <h1 className="text-2xl font-black tracking-tight text-white flex items-center justify-center gap-1.5">
            Hostvra <span className="text-emerald-400">Webmail SSO</span>
          </h1>
          <p className="text-xs text-slate-400 mt-1">
            Single Sign-On Authentication Gate
          </p>
        </div>

        {status === 'validating' && (
          <div className="space-y-4 py-4">
            <div className="flex items-center justify-center gap-2 text-sm text-slate-300">
              <RefreshCw className="w-4 h-4 text-emerald-400 animate-spin" />
              <span>Validating cryptographically signed SSO ticket...</span>
            </div>
            <div className="w-48 h-1.5 bg-slate-700 rounded-full overflow-hidden mx-auto">
              <div className="h-full bg-emerald-500 rounded-full animate-pulse" style={{ width: '70%' }} />
            </div>
            <p className="text-[11px] text-slate-500">
              Establishing end-to-end encrypted mailbox session
            </p>
          </div>
        )}

        {status === 'success' && (
          <div className="space-y-2 py-4 text-emerald-400">
            <div className="flex items-center justify-center gap-2 text-sm font-semibold">
              <ShieldCheck className="w-5 h-5 text-emerald-400" />
              <span>Session Authenticated! Loading Mailbox...</span>
            </div>
          </div>
        )}

        {status === 'error' && (
          <div className="space-y-5 py-2">
            <div className="p-4 bg-rose-500/10 border border-rose-500/30 rounded-2xl text-rose-300 text-xs flex items-start gap-3 text-left">
              <AlertCircle className="w-5 h-5 text-rose-400 shrink-0 mt-0.5" />
              <div>
                <p className="font-semibold text-rose-200">Authentication Failed</p>
                <p className="mt-1 text-slate-300">{errorMessage}</p>
              </div>
            </div>

            <div className="flex flex-col gap-2 pt-2">
              <Link
                href="/webmail/login"
                className="w-full py-3 px-4 bg-gradient-to-r from-emerald-600 to-teal-600 hover:from-emerald-500 hover:to-teal-500 text-white rounded-xl text-xs font-bold transition shadow-md shadow-emerald-600/20"
              >
                Sign In with Mailbox Credentials
              </Link>

              <Link
                href="/email"
                className="inline-flex items-center justify-center gap-1.5 py-2.5 text-xs text-slate-400 hover:text-white transition"
              >
                <ArrowLeft className="w-3.5 h-3.5" />
                <span>Return to Hostvra Control Panel</span>
              </Link>
            </div>
          </div>
        )}

        <div className="pt-4 border-t border-slate-700/50 flex items-center justify-center gap-1.5 text-[11px] text-slate-500">
          <ShieldCheck className="w-3.5 h-3.5 text-emerald-400" />
          <span>Single-use ticket &bull; 60s TTL &bull; Cryptographically bound</span>
        </div>
      </div>
    </div>
  );
}

export default function WebmailSSOPage() {
  return (
    <Suspense
      fallback={
        <div className="min-h-screen h-screen flex flex-col items-center justify-center bg-slate-900 text-white font-sans">
          <div className="w-8 h-8 border-2 border-emerald-500/20 border-t-emerald-500 rounded-full animate-spin" />
        </div>
      }
    >
      <WebmailSSOContent />
    </Suspense>
  );
}
