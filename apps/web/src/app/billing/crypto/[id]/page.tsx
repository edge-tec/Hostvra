'use client';

import React, { useState, useEffect, useCallback, useMemo } from 'react';
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
  CheckCircle2,
  Sparkles,
  Zap,
  Wallet,
  ArrowUpRight,
  CreditCard,
  ChevronDown,
  ChevronUp,
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

const COIN_METADATA: Record<
  string,
  {
    bgLight: string;
    textLight: string;
    borderLight: string;
    symbol: string;
    tag: string;
    color: string;
  }
> = {
  USDT: {
    bgLight: 'bg-emerald-50 text-emerald-600 border-emerald-200 dark:bg-emerald-500/10 dark:text-emerald-400 dark:border-emerald-500/20',
    textLight: 'text-emerald-600 dark:text-emerald-400',
    borderLight: 'border-emerald-500',
    symbol: '₮',
    tag: 'USDT',
    color: '#10b981',
  },
  BNB: {
    bgLight: 'bg-amber-50 text-amber-600 border-amber-200 dark:bg-amber-500/10 dark:text-amber-400 dark:border-amber-500/20',
    textLight: 'text-amber-600 dark:text-amber-400',
    borderLight: 'border-amber-500',
    symbol: 'BNB',
    tag: 'BNB',
    color: '#f59e0b',
  },
  TRX: {
    bgLight: 'bg-red-50 text-red-600 border-red-200 dark:bg-red-500/10 dark:text-red-400 dark:border-red-500/20',
    textLight: 'text-red-600 dark:text-red-400',
    borderLight: 'border-red-500',
    symbol: 'TRX',
    tag: 'TRON',
    color: '#ef4444',
  },
  BTC: {
    bgLight: 'bg-orange-50 text-orange-600 border-orange-200 dark:bg-orange-500/10 dark:text-orange-400 dark:border-orange-500/20',
    textLight: 'text-orange-600 dark:text-orange-400',
    borderLight: 'border-orange-500',
    symbol: '₿',
    tag: 'BTC',
    color: '#f97316',
  },
  ETH: {
    bgLight: 'bg-indigo-50 text-indigo-600 border-indigo-200 dark:bg-indigo-500/10 dark:text-indigo-400 dark:border-indigo-500/20',
    textLight: 'text-indigo-600 dark:text-indigo-400',
    borderLight: 'border-indigo-500',
    symbol: 'Ξ',
    tag: 'ETH',
    color: '#6366f1',
  },
  LTC: {
    bgLight: 'bg-sky-50 text-sky-600 border-sky-200 dark:bg-sky-500/10 dark:text-sky-400 dark:border-sky-500/20',
    textLight: 'text-sky-600 dark:text-sky-400',
    borderLight: 'border-sky-500',
    symbol: 'Ł',
    tag: 'LTC',
    color: '#0ea5e9',
  },
  DOGE: {
    bgLight: 'bg-amber-50 text-yellow-600 border-yellow-200 dark:bg-yellow-500/10 dark:text-yellow-400 dark:border-yellow-500/20',
    textLight: 'text-yellow-600 dark:text-yellow-400',
    borderLight: 'border-yellow-500',
    symbol: 'Ð',
    tag: 'DOGE',
    color: '#eab308',
  },
  USDC: {
    bgLight: 'bg-blue-50 text-blue-600 border-blue-200 dark:bg-blue-500/10 dark:text-blue-400 dark:border-blue-500/20',
    textLight: 'text-blue-600 dark:text-blue-400',
    borderLight: 'border-blue-500',
    symbol: '$',
    tag: 'USDC',
    color: '#3b82f6',
  },
};

export default function CryptoCheckoutPage() {
  const params = useParams();
  const router = useRouter();
  const invoiceId = params?.id as string;

  const [options, setOptions] = useState<CryptoCurrencyOption[]>([]);
  const [selectedCurrencyId, setSelectedCurrencyId] = useState<string>('');
  const [invoice, setInvoice] = useState<CryptoInvoice | null>(null);
  const [loading, setLoading] = useState<boolean>(true);
  const [creating, setCreating] = useState<boolean>(false);
  const [error, setError] = useState<string | null>(null);
  const [copiedAddr, setCopiedAddr] = useState<boolean>(false);
  const [copiedAmount, setCopiedAmount] = useState<boolean>(false);
  const [copiedCode, setCopiedCode] = useState<boolean>(false);
  const [txHashInput, setTxHashInput] = useState<string>('');
  const [verifyingTx, setVerifyingTx] = useState<boolean>(false);
  const [timeRemainingSec, setTimeRemainingSec] = useState<number>(1800);
  const [isAdmin, setIsAdmin] = useState<boolean>(false);
  const [showTxDrawer, setShowTxDrawer] = useState<boolean>(false);

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
        } else {
          setSelectedCurrencyId(res.data[0].id);
        }
      }
    } catch (e: any) {
      setError(e.message || 'Failed to load supported crypto currencies');
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
          setError(
            `The merchant has not yet configured on-chain recipient addresses for ${selected.name} (${selected.symbol}). Please select an active cryptocurrency marked as Ready.`
          );
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
    if (options.length > 0 && !invoice && selectedCurrencyId) {
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
    }, 3500);
    return () => clearInterval(pollInterval);
  }, [invoice]);

  // Format Timer mm:ss
  const formatTime = (secs: number) => {
    const mins = Math.floor(secs / 60);
    const s = secs % 60;
    return `${mins}:${s < 10 ? '0' : ''}${s}`;
  };

  const copyToClipboard = (text: string, type: 'addr' | 'amount' | 'code') => {
    navigator.clipboard.writeText(text);
    if (type === 'addr') {
      setCopiedAddr(true);
      setTimeout(() => setCopiedAddr(false), 2000);
    } else if (type === 'amount') {
      setCopiedAmount(true);
      setTimeout(() => setCopiedAmount(false), 2000);
    } else {
      setCopiedCode(true);
      setTimeout(() => setCopiedCode(false), 2000);
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

  const currentOption = useMemo(() => {
    return options.find((o) => o.id === invoice?.currency_id) || options.find((o) => o.id === selectedCurrencyId) || options[0];
  }, [options, invoice, selectedCurrencyId]);

  const coinMeta = useMemo(() => {
    const sym = currentOption?.symbol?.toUpperCase() || 'USDT';
    return COIN_METADATA[sym] || COIN_METADATA.USDT;
  }, [currentOption]);

  // Show active options first; filter out unprogrammed tokens
  const activeCurrencies = useMemo(() => {
    return options.filter((o) => o.has_treasury);
  }, [options]);

  const displayOptions = useMemo(() => {
    // If active currencies exist, show them directly without any broken/inactive clutter
    if (activeCurrencies.length > 0) {
      return activeCurrencies;
    }
    return options;
  }, [activeCurrencies, options]);

  // Deep Link for Trust Wallet or native Web3
  const walletDeepLink = useMemo(() => {
    if (!invoice) return '';
    if (invoice.network_id === 'tron') {
      return `tron:${invoice.deposit_address}?amount=${invoice.crypto_amount}`;
    }
    if (invoice.network_id === 'ethereum' || invoice.network_id === 'bsc' || invoice.network_id === 'polygon') {
      return invoice.qr_uri || `ethereum:${invoice.deposit_address}`;
    }
    if (invoice.network_id === 'bitcoin') {
      return `bitcoin:${invoice.deposit_address}?amount=${invoice.crypto_amount}`;
    }
    if (invoice.network_id === 'litecoin') {
      return `litecoin:${invoice.deposit_address}?amount=${invoice.crypto_amount}`;
    }
    if (invoice.network_id === 'dogecoin') {
      return `dogecoin:${invoice.deposit_address}?amount=${invoice.crypto_amount}`;
    }
    return `https://link.trustwallet.com/send?asset=c714&address=${invoice.deposit_address}&amount=${invoice.crypto_amount}`;
  }, [invoice]);

  return (
    <div className="min-h-screen bg-slate-50 dark:bg-slate-950 text-slate-900 dark:text-slate-100 flex flex-col items-center justify-center p-4 sm:p-8 antialiased">
      <div className="max-w-2xl w-full mx-auto space-y-4">
        {/* Navigation Bar */}
        <div className="flex items-center justify-between text-xs">
          <button
            onClick={() => router.push('/billing')}
            className="inline-flex items-center gap-2 text-slate-600 dark:text-slate-400 hover:text-slate-900 dark:hover:text-white transition-colors cursor-pointer py-1.5 px-3 rounded-xl bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 shadow-sm font-semibold"
          >
            <ArrowLeft className="w-4 h-4" />
            <span>Back to Invoices</span>
          </button>

          <div className="flex items-center gap-2 text-slate-600 dark:text-slate-400 font-medium text-xs bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 py-1.5 px-3 rounded-xl shadow-sm">
            <ShieldCheck className="w-4 h-4 text-emerald-500" />
            <span>Direct On-Chain Settlement • 0% Fees</span>
          </div>
        </div>

        {/* Clean Main Container */}
        <div className="bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 rounded-3xl shadow-xl overflow-hidden">
          {/* Header */}
          <div className="p-6 border-b border-slate-100 dark:border-slate-800 bg-slate-50/70 dark:bg-slate-900/50 flex items-center justify-between flex-wrap gap-4">
            <div className="flex items-center gap-3">
              <div className="w-12 h-12 rounded-2xl bg-emerald-500/10 border border-emerald-500/20 flex items-center justify-center text-emerald-600 dark:text-emerald-400">
                <Wallet className="w-6 h-6" />
              </div>
              <div>
                <div className="flex items-center gap-2">
                  <h1 className="text-lg font-bold text-slate-900 dark:text-white tracking-tight">
                    HostVra Crypto Checkout
                  </h1>
                  <span className="text-[10px] font-bold uppercase px-2 py-0.5 rounded-full bg-emerald-100 dark:bg-emerald-500/20 text-emerald-700 dark:text-emerald-400 border border-emerald-200 dark:border-emerald-500/30">
                    Zero Fee
                  </span>
                </div>
                <p className="text-xs text-slate-500 dark:text-slate-400 mt-0.5">
                  Direct transfer into merchant Trust Wallet cold treasury
                </p>
              </div>
            </div>

            {/* Countdown Badge */}
            {invoice && invoice.status !== 'paid' && invoice.status !== 'expired' && (
              <div className="flex items-center gap-2 px-3.5 py-1.5 rounded-xl border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 text-xs font-mono font-semibold shadow-sm text-slate-700 dark:text-slate-200">
                <Clock className="w-4 h-4 text-amber-500 animate-pulse" />
                <span>{formatTime(timeRemainingSec)}</span>
                <span className="text-[10px] text-slate-400 font-sans hidden sm:inline">left</span>
              </div>
            )}
          </div>

          {/* Currency Selection Grid - Fully visible, NO CUT-OFF SCROLLBARS */}
          {displayOptions.length > 0 && (!invoice || invoice.status === 'pending') && (
            <div className="p-6 border-b border-slate-100 dark:border-slate-800 bg-slate-50/30 dark:bg-slate-950/20 space-y-3">
              <label className="text-xs font-bold uppercase tracking-wider text-slate-500 dark:text-slate-400 block">
                Choose Payment Coin &amp; Network:
              </label>

              <div className="grid grid-cols-2 sm:grid-cols-4 gap-3 select-none">
                {displayOptions.map((opt) => {
                  const isSelected = selectedCurrencyId === opt.id;
                  const isAvailable = !!opt.has_treasury;
                  const meta = COIN_METADATA[opt.symbol.toUpperCase()] || COIN_METADATA.USDT;

                  return (
                    <button
                      key={opt.id}
                      onClick={() => {
                        setSelectedCurrencyId(opt.id);
                        initInvoice(opt.id);
                      }}
                      className={`p-3.5 rounded-2xl border text-left transition-all duration-150 flex flex-col justify-between gap-2 cursor-pointer shadow-sm ${isSelected
                        ? 'border-emerald-500 ring-2 ring-emerald-500/40 bg-emerald-50/40 dark:bg-emerald-500/10'
                        : 'border-slate-200 dark:border-slate-800 bg-white dark:bg-slate-800 hover:border-slate-300 dark:hover:border-slate-700'
                        }`}
                    >
                      <div className="flex items-center justify-between w-full">
                        <div className="flex items-center gap-2">
                          <div
                            className={`w-7 h-7 rounded-xl flex items-center justify-center font-bold text-xs border ${meta.bgLight}`}
                          >
                            {meta.symbol}
                          </div>
                          <div>
                            <span className="font-extrabold text-sm block text-slate-900 dark:text-white leading-tight">
                              {opt.symbol}
                            </span>
                            <span className="text-[10px] font-bold uppercase font-mono text-slate-400 block">
                              {opt.network_id}
                            </span>
                          </div>
                        </div>

                        {isSelected && (
                          <div className="w-5 h-5 rounded-full bg-emerald-500 text-white flex items-center justify-center shadow-sm">
                            <Check className="w-3 h-3 stroke-[3]" />
                          </div>
                        )}
                      </div>

                      <div className="text-[10px] text-slate-500 dark:text-slate-400 truncate border-t border-slate-100 dark:border-slate-700/60 pt-1.5 flex items-center justify-between">
                        <span>{opt.name}</span>
                        {isAvailable && (
                          <span className="text-emerald-600 dark:text-emerald-400 font-bold uppercase text-[9px]">
                            Ready
                          </span>
                        )}
                      </div>
                    </button>
                  );
                })}
              </div>
            </div>
          )}

          {/* Body Content */}
          <div className="p-6 sm:p-8 space-y-6">
            {/* Error Message Box */}
            {error && (
              <div className="p-6 bg-amber-50 dark:bg-amber-950/20 border border-amber-200 dark:border-amber-800/60 rounded-2xl text-xs text-amber-900 dark:text-amber-200 space-y-4">
                <div className="flex items-start gap-3">
                  <div className="p-2.5 rounded-xl bg-amber-500/10 text-amber-600 dark:text-amber-400 shrink-0">
                    <AlertTriangle className="w-5 h-5" />
                  </div>
                  <div className="space-y-1">
                    <p className="font-bold text-sm text-slate-900 dark:text-amber-100">
                      {error === 'NO_TREASURY_AVAILABLE'
                        ? 'Crypto Gateway Activation Required'
                        : 'Crypto Network Setup Required'}
                    </p>
                    <p className="text-slate-600 dark:text-amber-200/90 leading-relaxed text-xs">
                      {error === 'NO_TREASURY_AVAILABLE'
                        ? 'Super Admin has not yet configured on-chain recipient addresses. Please add your Trust Wallet addresses in Admin Dashboard to enable 1-click customer payments.'
                        : error}
                    </p>
                  </div>
                </div>

                <div className="pt-2 border-t border-amber-200 dark:border-amber-900/40 flex flex-wrap items-center gap-3">
                  {isAdmin && (
                    <button
                      onClick={() => router.push('/admin/crypto')}
                      className="px-4 py-2 rounded-xl bg-emerald-600 hover:bg-emerald-500 text-white text-xs font-bold transition flex items-center gap-1.5 cursor-pointer shadow-md"
                    >
                      <Zap className="w-4 h-4" />
                      <span>Setup Wallet Addresses in Admin</span>
                      <ExternalLink className="w-3.5 h-3.5" />
                    </button>
                  )}
                  <button
                    onClick={() => router.push('/billing')}
                    className="px-4 py-2 rounded-xl bg-white dark:bg-slate-800 hover:bg-slate-100 dark:hover:bg-slate-700 text-slate-700 dark:text-slate-200 text-xs font-semibold transition cursor-pointer border border-slate-200 dark:border-slate-700 shadow-sm"
                  >
                    Pay with Card / Stripe
                  </button>
                </div>
              </div>
            )}

            {/* Loading Indicator */}
            {!error && (loading || creating) && (
              <div className="py-20 flex flex-col items-center justify-center gap-3 text-slate-500 dark:text-slate-400">
                <div className="w-12 h-12 rounded-2xl bg-emerald-50 dark:bg-emerald-500/10 border border-emerald-200 dark:border-emerald-500/20 flex items-center justify-center text-emerald-600 dark:text-emerald-400">
                  <RefreshCw className="w-6 h-6 animate-spin" />
                </div>
                <div className="text-center space-y-0.5">
                  <p className="text-sm font-bold text-slate-900 dark:text-white">Loading On-Chain Invoice</p>
                  <p className="text-xs text-slate-400">Generating recipient address and locking live exchange rate...</p>
                </div>
              </div>
            )}

            {/* Active Invoice Content */}
            {!error && !loading && !creating && invoice && (
              <>
                {/* 1. PAID SUCCESS BANNER */}
                {invoice.status === 'paid' ? (
                  <div className="p-8 bg-emerald-50 dark:bg-emerald-950/30 border border-emerald-200 dark:border-emerald-500/30 rounded-3xl text-center space-y-5">
                    <div className="w-16 h-16 bg-emerald-100 dark:bg-emerald-500/20 text-emerald-600 dark:text-emerald-400 rounded-3xl flex items-center justify-center mx-auto border border-emerald-200 dark:border-emerald-500/30 shadow-lg">
                      <CheckCircle2 className="w-10 h-10" />
                    </div>
                    <div className="space-y-1">
                      <h2 className="text-xl font-bold text-slate-900 dark:text-white flex items-center justify-center gap-2">
                        <Sparkles className="w-5 h-5 text-emerald-500" />
                        Payment Verified &amp; Settled!
                      </h2>
                      <p className="text-sm text-slate-600 dark:text-slate-300">
                        Received on-chain:{' '}
                        <strong className="text-emerald-600 dark:text-emerald-400 font-mono text-base">
                          {invoice.amount_received} {invoice.currency_id}
                        </strong>
                      </p>
                      <p className="text-xs text-slate-500">
                        Your hosting services and subscription have been activated automatically.
                      </p>
                    </div>

                    {invoice.tx_hash && (
                      <div className="bg-white dark:bg-slate-950 p-4 rounded-2xl border border-slate-200 dark:border-slate-800 text-left font-mono text-xs break-all space-y-1 shadow-sm">
                        <div className="text-[10px] text-slate-400 uppercase font-bold">Transaction ID (TxHash)</div>
                        <div className="text-emerald-600 dark:text-emerald-400 font-medium">{invoice.tx_hash}</div>
                      </div>
                    )}

                    <button
                      onClick={() => router.push('/billing')}
                      className="w-full py-3.5 bg-emerald-600 hover:bg-emerald-500 text-white rounded-2xl font-bold text-sm transition cursor-pointer shadow-lg shadow-emerald-600/20 flex items-center justify-center gap-2"
                    >
                      <span>Return to Invoices &amp; Subscriptions</span>
                      <ArrowUpRight className="w-4 h-4" />
                    </button>
                  </div>
                ) : (
                  <>
                    {/* 2. REAL-TIME PROGRESS TRACKER */}
                    <div className="p-4 rounded-2xl bg-slate-50 dark:bg-slate-950/60 border border-slate-200 dark:border-slate-800 space-y-3">
                      <div className="flex items-center justify-between text-xs">
                        <div className="flex items-center gap-2.5">
                          {invoice.status === 'confirming' ? (
                            <RefreshCw className="w-4 h-4 text-sky-500 animate-spin" />
                          ) : invoice.status === 'detecting' ? (
                            <RefreshCw className="w-4 h-4 text-amber-500 animate-spin" />
                          ) : (
                            <span className="relative flex h-2.5 w-2.5">
                              <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-emerald-400 opacity-75" />
                              <span className="relative inline-flex rounded-full h-2.5 w-2.5 bg-emerald-500" />
                            </span>
                          )}
                          <span className="font-bold text-sm text-slate-800 dark:text-white">
                            {invoice.status === 'confirming'
                              ? `Confirming On-Chain (${invoice.confirmations} / ${invoice.min_confirmations} blocks)`
                              : invoice.status === 'detecting'
                                ? 'Deposit Detected! Verifying amount...'
                                : 'Awaiting On-Chain Deposit'}
                          </span>
                        </div>

                        <button
                          onClick={() => copyToClipboard(invoice.invoice_code, 'code')}
                          className="text-[11px] font-mono text-slate-500 hover:text-slate-800 dark:hover:text-white flex items-center gap-1 cursor-pointer bg-white dark:bg-slate-900 px-2 py-1 rounded-lg border border-slate-200 dark:border-slate-800 shadow-sm"
                          title="Copy Invoice Code"
                        >
                          <span>{invoice.invoice_code}</span>
                          {copiedCode ? <Check className="w-3 h-3 text-emerald-500" /> : <Copy className="w-3 h-3" />}
                        </button>
                      </div>

                      {/* Visual Stepper */}
                      <div className="grid grid-cols-3 gap-2 text-[10px] font-bold text-center">
                        <div className="p-2 rounded-xl bg-emerald-100 dark:bg-emerald-500/15 border border-emerald-200 dark:border-emerald-500/30 text-emerald-800 dark:text-emerald-300 flex items-center justify-center gap-1">
                          <Check className="w-3 h-3 stroke-[3]" />
                          <span>1. Created</span>
                        </div>
                        <div
                          className={`p-2 rounded-xl border flex items-center justify-center gap-1 ${invoice.status === 'confirming' || invoice.status === 'detecting'
                            ? 'bg-amber-100 dark:bg-amber-500/15 border-amber-200 dark:border-amber-500/30 text-amber-800 dark:text-amber-300'
                            : 'bg-white dark:bg-slate-900 border-slate-200 dark:border-slate-800 text-slate-400'
                            }`}
                        >
                          {invoice.status === 'confirming' || invoice.status === 'detecting' ? (
                            <RefreshCw className="w-3 h-3 animate-spin" />
                          ) : (
                            <span>2.</span>
                          )}
                          <span>Detected</span>
                        </div>
                        <div
                          className={`p-2 rounded-xl border flex items-center justify-center gap-1 ${invoice.status === 'paid'
                            ? 'bg-emerald-100 dark:bg-emerald-500/15 border-emerald-200 dark:border-emerald-500/30 text-emerald-800 dark:text-emerald-300'
                            : 'bg-white dark:bg-slate-900 border-slate-200 dark:border-slate-800 text-slate-400'
                            }`}
                        >
                          <span>3.</span>
                          <span>Settled</span>
                        </div>
                      </div>
                    </div>

                    {/* 3. QR CODE & AMOUNT CARD */}
                    <div className="flex flex-col sm:flex-row items-center gap-6 bg-slate-50 dark:bg-slate-950 p-6 rounded-3xl border border-slate-200 dark:border-slate-800 shadow-sm">
                      {/* White QR Container */}
                      <div className="bg-white p-3.5 rounded-2xl shadow-md border border-slate-200 dark:border-slate-700 shrink-0 flex flex-col items-center gap-2">
                        {/* eslint-disable-next-line @next/next/no-img-element */}
                        <img
                          src={`https://api.qrserver.com/v1/create-qr-code/?size=180x180&data=${encodeURIComponent(
                            invoice.qr_uri || invoice.deposit_address
                          )}`}
                          alt="Crypto Deposit QR"
                          className="w-36 h-36 sm:w-40 sm:h-40"
                        />
                        <span className="text-[10px] font-bold text-slate-600 uppercase tracking-wider">
                          Scan to Pay
                        </span>
                      </div>

                      {/* Payment Exact Amount */}
                      <div className="space-y-4 w-full text-center sm:text-left">
                        <div>
                          <span className="text-xs text-slate-500 uppercase font-bold tracking-wider block">
                            Send Exact Amount
                          </span>
                          <div className="flex items-baseline justify-center sm:justify-start gap-2 mt-1">
                            <span className="text-3xl sm:text-4xl font-extrabold text-slate-900 dark:text-white font-mono tracking-tight">
                              {invoice.crypto_amount}
                            </span>
                            <span className={`text-base font-bold ${coinMeta.textLight}`}>
                              {currentOption?.symbol || 'USDT'}
                            </span>
                          </div>
                          <div className="text-xs text-slate-500 font-mono mt-1">
                            ≈ ${invoice.fiat_amount_usd.toFixed(2)} USD
                            <span className="text-slate-400 ml-2">
                              (1 {currentOption?.symbol} ≈ ${(1 / invoice.exchange_rate).toFixed(2)})
                            </span>
                          </div>
                        </div>

                        {/* Action Buttons */}
                        <div className="flex flex-wrap items-center justify-center sm:justify-start gap-2 pt-1">
                          <button
                            onClick={() => copyToClipboard(invoice.crypto_amount.toString(), 'amount')}
                            className="inline-flex items-center gap-1.5 px-3.5 py-2 rounded-xl bg-white dark:bg-slate-800 hover:bg-slate-100 dark:hover:bg-slate-700 border border-slate-200 dark:border-slate-700 text-xs font-bold text-slate-800 dark:text-slate-200 transition cursor-pointer shadow-sm active:scale-95"
                          >
                            {copiedAmount ? <Check className="w-3.5 h-3.5 text-emerald-500" /> : <Copy className="w-3.5 h-3.5" />}
                            <span>{copiedAmount ? 'Amount Copied!' : 'Copy Exact Amount'}</span>
                          </button>

                          {walletDeepLink && (
                            <a
                              href={walletDeepLink}
                              target="_blank"
                              rel="noreferrer"
                              className="inline-flex items-center gap-1.5 px-3.5 py-2 rounded-xl bg-emerald-600 hover:bg-emerald-500 text-white text-xs font-bold transition cursor-pointer shadow-sm"
                            >
                              <ExternalLink className="w-3.5 h-3.5" />
                              <span>Open in Trust Wallet</span>
                            </a>
                          )}
                        </div>
                      </div>
                    </div>

                    {/* 4. RECIPIENT DEPOSIT ADDRESS BOX */}
                    <div className="space-y-2">
                      <div className="flex items-center justify-between text-xs">
                        <label className="text-xs font-bold uppercase tracking-wider text-slate-500 dark:text-slate-400">
                          {currentOption?.network_name || invoice.network_id.toUpperCase()} Deposit Address:
                        </label>
                        <span className="text-xs text-emerald-600 dark:text-emerald-400 font-semibold flex items-center gap-1">
                          <ShieldCheck className="w-3.5 h-3.5" />
                          Zero Middleman
                        </span>
                      </div>

                      <div className="flex items-center gap-2 p-3.5 bg-slate-50 dark:bg-slate-950 rounded-2xl border border-slate-200 dark:border-slate-800 focus-within:border-emerald-500 transition shadow-inner">
                        <span className="font-mono text-xs sm:text-sm text-slate-900 dark:text-slate-100 break-all select-all flex-1 tracking-tight font-medium">
                          {invoice.deposit_address}
                        </span>
                        <button
                          onClick={() => copyToClipboard(invoice.deposit_address, 'addr')}
                          className="px-3.5 py-2 rounded-xl bg-emerald-600 hover:bg-emerald-500 text-white font-bold text-xs transition cursor-pointer shrink-0 shadow-md flex items-center gap-1.5 active:scale-95"
                          title="Copy Address"
                        >
                          {copiedAddr ? <Check className="w-4 h-4" /> : <Copy className="w-4 h-4" />}
                          <span>{copiedAddr ? 'Copied' : 'Copy'}</span>
                        </button>
                      </div>

                      {/* Safety Banner */}
                      <div className="p-3 bg-amber-50 dark:bg-amber-950/20 border border-amber-200 dark:border-amber-800/40 rounded-xl text-xs text-amber-900 dark:text-amber-300 leading-relaxed">
                        Send ONLY <strong>{currentOption?.symbol || 'USDT'}</strong> via the{' '}
                        <strong className="underline">
                          {currentOption?.network_name || invoice.network_id.toUpperCase()}
                        </strong>{' '}
                        network. Transferring any other token or using an incorrect blockchain will result in lost funds.
                      </div>
                    </div>

                    {/* 5. TRANSACTION HASH FAST-TRACK ACCORDION */}
                    <div className="rounded-2xl border border-slate-200 dark:border-slate-800 bg-white dark:bg-slate-950/40 overflow-hidden shadow-sm">
                      <button
                        onClick={() => setShowTxDrawer(!showTxDrawer)}
                        className="w-full p-3.5 flex items-center justify-between text-xs text-slate-600 dark:text-slate-400 hover:text-slate-900 dark:hover:text-slate-200 transition cursor-pointer"
                      >
                        <div className="flex items-center gap-2">
                          <Zap className="w-4 h-4 text-amber-500" />
                          <span className="font-semibold">
                            Already transferred? Fast-track confirmation with TxHash
                          </span>
                        </div>
                        {showTxDrawer ? <ChevronUp className="w-4 h-4" /> : <ChevronDown className="w-4 h-4" />}
                      </button>

                      {showTxDrawer && (
                        <div className="p-4 pt-2 border-t border-slate-100 dark:border-slate-800 space-y-2.5">
                          <p className="text-xs text-slate-500">
                            Paste the blockchain transaction hash (txid) from your Trust Wallet or exchange:
                          </p>
                          <div className="flex gap-2">
                            <input
                              type="text"
                              placeholder="0x... or transaction hash..."
                              value={txHashInput}
                              onChange={(e) => setTxHashInput(e.target.value)}
                              className="flex-1 bg-slate-50 dark:bg-slate-900 border border-slate-200 dark:border-slate-700 rounded-xl px-3.5 py-2 text-xs font-mono text-slate-900 dark:text-slate-200 focus:outline-none focus:border-emerald-500 shadow-inner"
                            />
                            <button
                              onClick={handleManualVerify}
                              disabled={verifyingTx || !txHashInput.trim()}
                              className="px-4 py-2 bg-emerald-600 hover:bg-emerald-500 disabled:opacity-50 text-xs font-bold rounded-xl text-white transition cursor-pointer flex items-center gap-1.5 shadow-md"
                            >
                              {verifyingTx && <RefreshCw className="w-3.5 h-3.5 animate-spin" />}
                              <span>Verify Tx</span>
                            </button>
                          </div>
                        </div>
                      )}
                    </div>

                    {/* 6. ALTERNATIVE PAYMENT & SUPPORT FOOTER */}
                    <div className="pt-2 border-t border-slate-100 dark:border-slate-800 flex items-center justify-between flex-wrap gap-2 text-xs text-slate-500">
                      <button
                        onClick={() => router.push('/billing')}
                        className="hover:text-slate-900 dark:hover:text-slate-200 transition flex items-center gap-1.5 cursor-pointer font-medium"
                      >
                        <CreditCard className="w-3.5 h-3.5" />
                        <span>Pay with Card (Stripe) instead</span>
                      </button>

                      <button
                        onClick={() => router.push('/support')}
                        className="hover:text-slate-900 dark:hover:text-slate-200 transition cursor-pointer text-slate-400"
                      >
                        Need Help? Contact Support
                      </button>
                    </div>
                  </>
                )}
              </>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}
