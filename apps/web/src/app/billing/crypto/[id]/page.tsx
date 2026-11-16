'use client';

import React, { useState, useEffect, useCallback } from 'react';
import { useParams, useRouter } from 'next/navigation';
import { apiFetch, getStoredUserRole } from '@/lib/api';
import {
  ShieldCheck,
  Clock,
  Copy,
  Check,
  RefreshCw,
  AlertTriangle,
  ArrowLeft,
  ExternalLink,
  Coins,
  CheckCircle2,
  Sparkles,
  Zap,
} from 'lucide-react';

interface CryptoCurrencyOption {
  id: string;
  network_id: string;
  network_name: string;
  symbol: string;
  name: string;
  decimals: number;
  min_confirmations: number;
  min_amount_usd: number;
  icon: string;
  price_usd: number;
  has_treasury?: boolean;
}

interface CryptoInvoice {
  id: string;
  invoice_code: string;
  payable_type: string;
  payable_id: string;
  fiat_amount_usd: number;
  crypto_amount: number;
  exchange_rate: number;
  currency_id: string;
  network_id: string;
  deposit_address: string;
  tx_hash?: string;
  confirmations: number;
  min_confirmations: number;
  amount_received: number;
  status: string; // pending, detecting, confirming, paid, underpaid, overpaid, expired, failed
  qr_uri: string;
  expires_at: string;
  currency_symbol?: string;
  network_name?: string;
  explorer_tx_url?: string;
  seconds_remaining?: number;
}

export default function CryptoCheckoutPage() {
  const params = useParams();
  const router = useRouter();
  const invoiceId = params?.id as string;

  const [options, setOptions] = useState<CryptoCurrencyOption[]>([]);
  const [selectedCurrencyId, setSelectedCurrencyId] = useState<string>('USDT_TRC20');
  const [invoice, setInvoice] = useState<CryptoInvoice | null>(null);
  const [loading, setLoading] = useState<boolean>(true);
  const [creating, setCreating] = useState<boolean>(false);
  const [error, setError] = useState<string | null>(null);
  const [copiedAddr, setCopiedAddr] = useState<boolean>(false);
  const [copiedAmount, setCopiedAmount] = useState<boolean>(false);
  const [txHashInput, setTxHashInput] = useState<string>('');
  const [verifyingTx, setVerifyingTx] = useState<boolean>(false);
  const [timeRemainingSec, setTimeRemainingSec] = useState<number>(1800);
  const [isAdmin, setIsAdmin] = useState<boolean>(false);

  useEffect(() => {
    const roleInfo = getStoredUserRole();
    setIsAdmin(roleInfo.role === 'admin' || roleInfo.role === 'superadmin' || roleInfo.isSuperAdmin);
  }, []);

  // 1. Fetch Currency Options
  const fetchCurrencies = useCallback(async () => {
    try {
      const res = await apiFetch<CryptoCurrencyOption[]>('/api/v1/crypto/currencies');
      if (res.data && res.data.length > 0) {
        setOptions(res.data);
        const firstActive = res.data.find((o) => o.has_treasury);
        if (firstActive) {
          setSelectedCurrencyId(firstActive.id);
        }
      }
    } catch (e: any) {
      setError(e.message || 'Failed to load supported crypto networks');
    }
  }, []);

  // 2. Fetch or Create Crypto Invoice
  const initInvoice = useCallback(
    async (curId: string) => {
      if (!invoiceId) return;
      setSelectedCurrencyId(curId);
      setCreating(true);
      setError(null);
      setInvoice(null);
      try {
        const selected = options.find((o) => o.id === curId) || options.find((o) => o.has_treasury) || options[0];
        if (options.length > 0 && !options.some((o) => o.has_treasury)) {
          setError('NO_TREASURY_AVAILABLE');
          setLoading(false);
          setCreating(false);
          return;
        }

        if (selected && !selected.has_treasury) {
          setError(`The merchant has not yet configured on-chain recipient addresses for ${selected.name} (${selected.symbol}). Please select an active cryptocurrency marked with a green dot.`);
          setLoading(false);
          setCreating(false);
          return;
        }

        const res = await apiFetch<CryptoInvoice>('/api/v1/crypto/invoices', {
          method: 'POST',
          body: JSON.stringify({
            payable_type: 'invoice',
            payable_id: invoiceId,
            currency_id: selected ? selected.id : curId,
            network_id: selected ? selected.network_id : 'tron',
          }),
        });

        if (res.data) {
          setInvoice(res.data);
          const expMs = new Date(res.data.expires_at).getTime() - Date.now();
          setTimeRemainingSec(Math.max(0, Math.floor(expMs / 1000)));
        } else {
          setError(res.error?.message || 'Failed to generate crypto invoice');
        }
      } catch (e: any) {
        setError(e.message || 'Failed to initialize payment invoice');
      } finally {
        setCreating(false);
        setLoading(false);
      }
    },
    [invoiceId, options]
  );

  useEffect(() => {
    fetchCurrencies();
  }, [fetchCurrencies]);

  useEffect(() => {
    if (options.length > 0 && !invoice) {
      initInvoice(selectedCurrencyId);
    }
  }, [options, invoice, selectedCurrencyId, initInvoice]);

  // 3. Countdown Timer
  useEffect(() => {
    if (!invoice || invoice.status === 'paid' || invoice.status === 'expired') return;
    const timer = setInterval(() => {
      setTimeRemainingSec((prev) => {
        if (prev <= 1) {
          clearInterval(timer);
          return 0;
        }
        return prev - 1;
      });
    }, 1000);
    return () => clearInterval(timer);
  }, [invoice]);

  // 4. Real-time Status Polling
  useEffect(() => {
    if (!invoice || invoice.status === 'paid' || invoice.status === 'expired') return;
    const pollInterval = setInterval(async () => {
      try {
        const res = await apiFetch<CryptoInvoice>(`/api/v1/crypto/invoices/${invoice.invoice_code}`);
        if (res.data) {
          setInvoice(res.data);
          if (res.data.status === 'paid') {
            clearInterval(pollInterval);
          }
        }
      } catch {
        // Continue polling silently
      }
    }, 4000);
    return () => clearInterval(pollInterval);
  }, [invoice]);

  // Format Timer mm:ss
  const formatTime = (secs: number) => {
    const mins = Math.floor(secs / 60);
    const s = secs % 60;
    return `${mins}:${s < 10 ? '0' : ''}${s}`;
  };

  const copyToClipboard = (text: string, type: 'addr' | 'amount') => {
    navigator.clipboard.writeText(text);
    if (type === 'addr') {
      setCopiedAddr(true);
      setTimeout(() => setCopiedAddr(false), 2000);
    } else {
      setCopiedAmount(true);
      setTimeout(() => setCopiedAmount(false), 2000);
    }
  };

  const handleManualVerify = async () => {
    if (!invoice) return;
    setVerifyingTx(true);
    try {
      const res = await apiFetch<CryptoInvoice>(`/api/v1/crypto/invoices/${invoice.invoice_code}/verify`, {
        method: 'POST',
        body: JSON.stringify({ tx_hash: txHashInput.trim() }),
      });
      if (res.data) {
        setInvoice(res.data);
      }
    } catch (e: any) {
      alert(e.message || 'Verification failed. Please ensure the transaction is broadcast on the blockchain.');
    } finally {
      setVerifyingTx(false);
    }
  };

  const currentOption = options.find((o) => o.id === invoice?.currency_id) || options[0];

  return (
    <div className="min-h-screen bg-slate-900 text-slate-100 flex flex-col items-center justify-center p-4">
      <div className="max-w-xl w-full">
        {/* Back Button */}
        <button
          onClick={() => router.push('/billing')}
          className="inline-flex items-center gap-1.5 text-xs text-slate-400 hover:text-slate-200 mb-4 transition cursor-pointer"
        >
          <ArrowLeft className="w-4 h-4" />
          <span>Back to Billing</span>
        </button>

        {/* Card Container */}
        <div className="bg-slate-800 border border-slate-700/80 rounded-2xl shadow-2xl overflow-hidden backdrop-blur-xl">
          {/* Header */}
          <div className="p-6 border-b border-slate-700/80 bg-slate-800/60 flex items-center justify-between flex-wrap gap-4">
            <div>
              <div className="flex items-center gap-2">
                <div className="p-2 bg-emerald-500/10 rounded-lg text-emerald-400">
                  <Coins className="w-5 h-5" />
                </div>
                <div>
                  <h1 className="text-base font-bold text-white flex items-center gap-2">
                    Private Crypto Gateway
                    <span className="text-[10px] uppercase font-semibold px-2 py-0.5 rounded-full bg-emerald-500/20 text-emerald-400 border border-emerald-500/30">
                      0% Fees
                    </span>
                  </h1>
                  <p className="text-xs text-slate-400">
                    HostVra In-House On-Chain Treasury Verification
                  </p>
                </div>
              </div>
            </div>

            {/* Countdown Badge */}
            {invoice && invoice.status !== 'paid' && invoice.status !== 'expired' && (
              <div className="flex items-center gap-1.5 bg-slate-900/80 border border-slate-700 px-3 py-1.5 rounded-full text-xs font-mono">
                <Clock className="w-3.5 h-3.5 text-amber-400 animate-pulse" />
                <span className="text-slate-300 font-semibold">{formatTime(timeRemainingSec)}</span>
              </div>
            )}
          </div>

          {/* Network Selector Tabs */}
          {options.length > 0 && (!invoice || invoice.status === 'pending') && (
            <div className="p-4 bg-slate-900/40 border-b border-slate-700/50">
              <label className="block text-[11px] font-semibold uppercase text-slate-400 mb-2">
                Select Network / Currency:
              </label>
              <div className="grid grid-cols-2 sm:grid-cols-3 gap-2">
                {options.map((opt) => {
                  const isSelected = selectedCurrencyId === opt.id;
                  const isAvailable = !!opt.has_treasury;
                  return (
                    <button
                      key={opt.id}
                      onClick={() => {
                        setSelectedCurrencyId(opt.id);
                        initInvoice(opt.id);
                      }}
                      className={`p-2.5 rounded-xl border text-left transition flex flex-col justify-between ${
                        !isAvailable
                          ? isSelected
                            ? 'border-amber-500/80 bg-amber-500/10 text-amber-200 cursor-pointer'
                            : 'border-slate-800 bg-slate-900/40 text-slate-400 hover:border-slate-700 cursor-pointer'
                          : isSelected
                          ? 'border-emerald-500 bg-emerald-500/10 text-white ring-1 ring-emerald-500 cursor-pointer'
                          : 'border-slate-700/80 bg-slate-800/50 hover:bg-slate-700/50 text-slate-300 cursor-pointer'
                      }`}
                    >
                      <div className="flex items-center justify-between">
                        <span className="font-bold text-xs">{opt.symbol}</span>
                        <div className="flex items-center gap-1">
                          {isAvailable ? (
                            <span className="w-1.5 h-1.5 rounded-full bg-emerald-400" title="Active Treasury" />
                          ) : (
                            <span className="text-[9px] text-slate-400 uppercase font-medium">Inactive</span>
                          )}
                          <span className="text-[10px] text-slate-400 uppercase">{opt.network_id}</span>
                        </div>
                      </div>
                      <div className="text-[10px] text-slate-400 truncate mt-1">{opt.name}</div>
                    </button>
                  );
                })}
              </div>
            </div>
          )}

          {/* Body Content */}
          <div className="p-6 space-y-6">
            {error && (
              <div className="p-5 bg-rose-950/30 border border-rose-800/60 rounded-2xl text-xs text-rose-300 space-y-3">
                <div className="flex items-start gap-2.5">
                  <AlertTriangle className="w-5 h-5 text-rose-400 shrink-0 mt-0.5" />
                  <div className="space-y-1">
                    <p className="font-bold text-rose-200">
                      {error === 'NO_TREASURY_AVAILABLE' || error.includes('no treasury destination address')
                        ? 'Crypto Gateway Under Configuration'
                        : 'Unable to Generate Crypto Invoice'}
                    </p>
                    <p className="text-rose-300/90 leading-relaxed">
                      {error === 'NO_TREASURY_AVAILABLE' || error.includes('no treasury destination address')
                        ? 'The merchant has not yet configured on-chain recipient addresses for this token. Please select an active cryptocurrency above or complete payment via Stripe / Card.'
                        : error}
                    </p>
                  </div>
                </div>
                <div className="pt-2 border-t border-rose-900/40 flex flex-wrap gap-2">
                  {isAdmin && (
                    <button
                      onClick={() => router.push('/admin/crypto')}
                      className="px-3 py-1.5 rounded-lg bg-emerald-600 hover:bg-emerald-500 text-white text-xs font-bold transition flex items-center gap-1.5 cursor-pointer shadow-sm"
                    >
                      <Zap className="w-3.5 h-3.5" />
                      <span>Setup Wallet Addresses in Admin</span>
                      <ExternalLink className="w-3.5 h-3.5" />
                    </button>
                  )}
                  <button
                    onClick={() => router.push('/billing')}
                    className="px-3 py-1.5 rounded-lg bg-slate-800 hover:bg-slate-700 text-slate-200 text-xs font-semibold transition cursor-pointer"
                  >
                    Pay with Stripe / Card
                  </button>
                </div>
              </div>
            )}

            {!error && (loading || creating ? (
              <div className="py-16 flex flex-col items-center justify-center gap-3 text-slate-400">
                <RefreshCw className="w-8 h-8 animate-spin text-emerald-400" />
                <span className="text-xs">Generating dedicated deposit address on blockchain...</span>
              </div>
            ) : invoice ? (
              <>
                {/* PAID SUCCESS BANNER */}
                {invoice.status === 'paid' ? (
                  <div className="p-6 bg-emerald-950/40 border border-emerald-500/40 rounded-2xl text-center space-y-4">
                    <div className="w-16 h-16 bg-emerald-500/20 text-emerald-400 rounded-full flex items-center justify-center mx-auto border border-emerald-500/40 shadow-lg shadow-emerald-500/10">
                      <CheckCircle2 className="w-10 h-10" />
                    </div>
                    <div>
                      <h2 className="text-lg font-bold text-white flex items-center justify-center gap-1.5">
                        <Sparkles className="w-4 h-4 text-emerald-400" />
                        Payment Verified &amp; Settled!
                      </h2>
                      <p className="text-xs text-slate-300 mt-1">
                        Received on-chain:{' '}
                        <strong className="text-emerald-400">
                          {invoice.amount_received} {invoice.currency_id}
                        </strong>
                      </p>
                    </div>

                    {invoice.tx_hash && (
                      <div className="bg-slate-900/80 p-3 rounded-lg border border-slate-700 text-left font-mono text-[11px] break-all">
                        <div className="text-[10px] text-slate-400 uppercase font-semibold">Transaction Hash</div>
                        <div className="text-emerald-400">{invoice.tx_hash}</div>
                      </div>
                    )}

                    <button
                      onClick={() => router.push('/billing')}
                      className="w-full py-3 bg-emerald-600 hover:bg-emerald-500 text-white rounded-xl font-bold text-xs transition cursor-pointer shadow-lg"
                    >
                      Return to Invoices &amp; Subscriptions
                    </button>
                  </div>
                ) : (
                  <>
                    {/* CONFIRMATION / PROGRESS BADGE */}
                    <div className="flex items-center justify-between p-3.5 rounded-xl bg-slate-900/60 border border-slate-700/80">
                      <div className="flex items-center gap-2">
                        {invoice.status === 'confirming' ? (
                          <RefreshCw className="w-4 h-4 text-sky-400 animate-spin" />
                        ) : invoice.status === 'detecting' ? (
                          <RefreshCw className="w-4 h-4 text-amber-400 animate-spin" />
                        ) : (
                          <div className="w-2.5 h-2.5 rounded-full bg-emerald-400 animate-ping" />
                        )}
                        <span className="text-xs font-semibold capitalize text-slate-200">
                          {invoice.status === 'confirming'
                            ? `Confirming (${invoice.confirmations} / ${invoice.min_confirmations} blocks)`
                            : invoice.status === 'detecting'
                            ? 'Deposit Detected! Verifying...'
                            : 'Awaiting On-Chain Payment'}
                        </span>
                      </div>
                      <span className="text-[11px] font-mono text-slate-400">
                        Code: {invoice.invoice_code}
                      </span>
                    </div>

                    {/* QR Code and Amount Display */}
                    <div className="flex flex-col sm:flex-row items-center gap-6 bg-slate-900/40 p-5 rounded-2xl border border-slate-700/60">
                      {/* QR Box */}
                      <div className="bg-white p-3 rounded-xl shadow-md shrink-0">
                        {/* eslint-disable-next-line @next/next/no-img-element */}
                        <img
                          src={`https://api.qrserver.com/v1/create-qr-code/?size=180x180&data=${encodeURIComponent(
                            invoice.qr_uri || invoice.deposit_address
                          )}`}
                          alt="Crypto Deposit QR"
                          className="w-36 h-36"
                        />
                      </div>

                      {/* Payment Exact Amount */}
                      <div className="space-y-3 w-full">
                        <div>
                          <div className="text-[10px] text-slate-400 uppercase font-semibold">Send Exactly</div>
                          <div className="flex items-baseline gap-2">
                            <span className="text-2xl font-extrabold text-emerald-400 font-mono">
                              {invoice.crypto_amount}
                            </span>
                            <span className="text-sm font-bold text-slate-200">
                              {currentOption?.symbol || 'USDT'}
                            </span>
                          </div>
                          <div className="text-xs text-slate-400 font-mono mt-0.5">
                            ≈ ${invoice.fiat_amount_usd.toFixed(2)} USD
                          </div>
                        </div>

                        <button
                          onClick={() => copyToClipboard(invoice.crypto_amount.toString(), 'amount')}
                          className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-slate-800 hover:bg-slate-700 border border-slate-700 text-xs font-medium text-slate-200 transition cursor-pointer"
                        >
                          {copiedAmount ? <Check className="w-3.5 h-3.5 text-emerald-400" /> : <Copy className="w-3.5 h-3.5" />}
                          <span>{copiedAmount ? 'Amount Copied' : 'Copy Exact Amount'}</span>
                        </button>
                      </div>
                    </div>

                    {/* Deposit Address Box */}
                    <div className="space-y-2">
                      <div className="flex items-center justify-between">
                        <label className="text-[11px] font-semibold uppercase text-slate-400">
                          Recipient Deposit Address ({currentOption?.network_name || invoice.network_id.toUpperCase()}):
                        </label>
                        <span className="text-[10px] text-emerald-400 font-medium flex items-center gap-1">
                          <ShieldCheck className="w-3.5 h-3.5" />
                          Isolated Non-Custodial Address
                        </span>
                      </div>

                      <div className="flex items-center gap-2 p-3 bg-slate-950 rounded-xl border border-slate-700">
                        <span className="font-mono text-xs text-slate-200 break-all select-all flex-1">
                          {invoice.deposit_address}
                        </span>
                        <button
                          onClick={() => copyToClipboard(invoice.deposit_address, 'addr')}
                          className="p-2 rounded-lg bg-slate-800 hover:bg-slate-700 text-slate-300 transition cursor-pointer shrink-0"
                          title="Copy Address"
                        >
                          {copiedAddr ? <Check className="w-4 h-4 text-emerald-400" /> : <Copy className="w-4 h-4" />}
                        </button>
                      </div>

                      <p className="text-[11px] text-amber-400/90 leading-relaxed">
                        ⚠️ Send ONLY <strong>{currentOption?.symbol || 'USDT'}</strong> via the{' '}
                        <strong>{currentOption?.network_name || invoice.network_id.toUpperCase()}</strong> network. Sending via any other network will result in lost funds.
                      </p>
                    </div>

                    {/* Transaction Hash Self-Submission */}
                    <div className="pt-2 border-t border-slate-700/80 space-y-2">
                      <label className="block text-[11px] text-slate-400">
                        Already sent funds from your wallet? Speed up confirmation:
                      </label>
                      <div className="flex gap-2">
                        <input
                          type="text"
                          placeholder="Paste blockchain transaction hash (txid)..."
                          value={txHashInput}
                          onChange={(e) => setTxHashInput(e.target.value)}
                          className="flex-1 bg-slate-900 border border-slate-700 rounded-xl px-3 py-2 text-xs font-mono text-slate-200 focus:outline-none focus:border-emerald-500"
                        />
                        <button
                          onClick={handleManualVerify}
                          disabled={verifyingTx || !txHashInput.trim()}
                          className="px-4 py-2 bg-slate-800 hover:bg-slate-700 disabled:opacity-50 text-xs font-semibold rounded-xl text-slate-200 border border-slate-700 transition cursor-pointer flex items-center gap-1.5"
                        >
                          {verifyingTx && <RefreshCw className="w-3.5 h-3.5 animate-spin" />}
                          <span>Verify</span>
                        </button>
                      </div>
                    </div>
                  </>
                )}
              </>
            ) : null)}
          </div>
        </div>
      </div>
    </div>
  );
}
