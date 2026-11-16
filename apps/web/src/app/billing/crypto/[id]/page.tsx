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
  Info,
  CreditCard,
  ChevronDown,
  ChevronUp,
  HelpCircle,
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

const COIN_THEMES: Record<
  string,
  {
    bg: string;
    text: string;
    border: string;
    ring: string;
    symbol: string;
    tag: string;
    networkTag: string;
    accent: string;
  }
> = {
  USDT: {
    bg: 'bg-emerald-500/15',
    text: 'text-emerald-400',
    border: 'border-emerald-500/30',
    ring: 'ring-emerald-500',
    symbol: '₮',
    tag: 'Stablecoin',
    networkTag: 'Multi-Chain',
    accent: 'text-emerald-400',
  },
  USDC: {
    bg: 'bg-blue-500/15',
    text: 'text-blue-400',
    border: 'border-blue-500/30',
    ring: 'ring-blue-500',
    symbol: '$',
    tag: 'Stablecoin',
    networkTag: 'Multi-Chain',
    accent: 'text-blue-400',
  },
  BTC: {
    bg: 'bg-amber-500/15',
    text: 'text-amber-400',
    border: 'border-amber-500/30',
    ring: 'ring-amber-500',
    symbol: '₿',
    tag: 'Bitcoin Core',
    networkTag: 'Layer 1',
    accent: 'text-amber-400',
  },
  ETH: {
    bg: 'bg-indigo-500/15',
    text: 'text-indigo-400',
    border: 'border-indigo-500/30',
    ring: 'ring-indigo-500',
    symbol: 'Ξ',
    tag: 'Ethereum Mainnet',
    networkTag: 'ERC-20',
    accent: 'text-indigo-400',
  },
  BNB: {
    bg: 'bg-yellow-500/15',
    text: 'text-yellow-400',
    border: 'border-yellow-500/30',
    ring: 'ring-yellow-500',
    symbol: 'BNB',
    tag: 'BNB Smart Chain',
    networkTag: 'BEP-20',
    accent: 'text-yellow-400',
  },
  TRX: {
    bg: 'bg-rose-500/15',
    text: 'text-rose-400',
    border: 'border-rose-500/30',
    ring: 'ring-rose-500',
    symbol: 'TRX',
    tag: 'TRON Protocol',
    networkTag: 'TRC-20',
    accent: 'text-rose-400',
  },
  SOL: {
    bg: 'bg-purple-500/15',
    text: 'text-purple-400',
    border: 'border-purple-500/30',
    ring: 'ring-purple-500',
    symbol: 'SOL',
    tag: 'Solana High-Speed',
    networkTag: 'SPL',
    accent: 'text-purple-400',
  },
  LTC: {
    bg: 'bg-sky-500/15',
    text: 'text-sky-400',
    border: 'border-sky-500/30',
    ring: 'ring-sky-500',
    symbol: 'Ł',
    tag: 'Litecoin Network',
    networkTag: 'Scrypt',
    accent: 'text-sky-400',
  },
};

export default function CryptoCheckoutPage() {
  const params = useParams();
  const router = useRouter();
  const invoiceId = params?.id as string;

  const [options, setOptions] = useState<CryptoCurrencyOption[]>([]);
  const [selectedCurrencyId, setSelectedCurrencyId] = useState<string>('USDT_TRC20');
  const [activeFilter, setActiveFilter] = useState<'all' | 'stable' | 'fast' | 'major'>('all');
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
          setError(
            `The merchant has not yet configured on-chain recipient addresses for ${selected.name} (${selected.symbol}). Please select an active cryptocurrency marked with a green dot.`
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

  const coinTheme = useMemo(() => {
    const sym = currentOption?.symbol?.toUpperCase() || 'USDT';
    return COIN_THEMES[sym] || COIN_THEMES.USDT;
  }, [currentOption]);

  const filteredOptions = useMemo(() => {
    if (activeFilter === 'stable') {
      return options.filter((o) => o.symbol === 'USDT' || o.symbol === 'USDC');
    }
    if (activeFilter === 'fast') {
      return options.filter((o) => o.network_id === 'tron' || o.network_id === 'bsc' || o.network_id === 'polygon');
    }
    if (activeFilter === 'major') {
      return options.filter((o) => o.symbol === 'BTC' || o.symbol === 'ETH' || o.symbol === 'BNB');
    }
    return options;
  }, [options, activeFilter]);

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
    return `https://link.trustwallet.com/send?asset=c714&address=${invoice.deposit_address}&amount=${invoice.crypto_amount}`;
  }, [invoice]);

  return (
    <div className="min-h-screen bg-slate-950 text-slate-100 flex flex-col items-center justify-center p-3 sm:p-6 antialiased selection:bg-emerald-500/30 selection:text-emerald-200">
      {/* Background Ambient Glow */}
      <div className="fixed inset-0 overflow-hidden pointer-events-none -z-10">
        <div className="absolute top-1/4 left-1/2 -translate-x-1/2 -translate-y-1/2 w-[600px] h-[350px] bg-gradient-to-tr from-emerald-600/10 via-cyan-500/10 to-transparent blur-3xl opacity-60" />
        <div className="absolute bottom-10 right-10 w-[300px] h-[300px] bg-indigo-600/10 blur-3xl rounded-full" />
      </div>

      <div className="max-w-2xl w-full mx-auto space-y-4">
        {/* Top Header / Back Bar */}
        <div className="flex items-center justify-between text-xs">
          <button
            onClick={() => router.push('/billing')}
            className="inline-flex items-center gap-2 text-slate-400 hover:text-white transition-colors cursor-pointer py-1 px-2.5 rounded-lg hover:bg-slate-900 border border-transparent hover:border-slate-800"
          >
            <ArrowLeft className="w-4 h-4" />
            <span className="font-medium">Back to Invoices</span>
          </button>

          <div className="flex items-center gap-2 text-slate-400">
            <ShieldCheck className="w-4 h-4 text-emerald-400" />
            <span className="hidden sm:inline text-[11px] font-medium text-slate-400">256-bit Direct Blockchain Settlement</span>
          </div>
        </div>

        {/* Main Card Container */}
        <div className="bg-slate-900/90 border border-slate-800/80 rounded-3xl shadow-2xl overflow-hidden backdrop-blur-2xl ring-1 ring-white/5">
          {/* Header Bar */}
          <div className="p-5 sm:p-6 border-b border-slate-800 bg-slate-900/50 flex items-center justify-between flex-wrap gap-4">
            <div className="flex items-center gap-3">
              <div className="w-11 h-11 rounded-2xl bg-gradient-to-tr from-emerald-500/20 to-teal-500/10 border border-emerald-500/30 flex items-center justify-center text-emerald-400 shadow-inner">
                <Wallet className="w-5 h-5" />
              </div>
              <div>
                <div className="flex items-center gap-2">
                  <h1 className="text-base sm:text-lg font-bold text-white tracking-tight">HostVra Crypto Pay</h1>
                  <span className="text-[10px] uppercase font-bold px-2 py-0.5 rounded-full bg-emerald-500/15 text-emerald-400 border border-emerald-500/30 tracking-wider">
                    0% Fee
                  </span>
                </div>
                <p className="text-xs text-slate-400 mt-0.5">Trust Wallet Direct On-Chain Treasury</p>
              </div>
            </div>

            {/* Countdown Badge */}
            {invoice && invoice.status !== 'paid' && invoice.status !== 'expired' && (
              <div
                className={`flex items-center gap-2 px-3 py-1.5 rounded-xl border text-xs font-mono transition-all ${
                  timeRemainingSec < 300
                    ? 'bg-rose-950/40 border-rose-500/40 text-rose-300 animate-pulse'
                    : 'bg-slate-800/90 border-slate-700 text-slate-200'
                }`}
              >
                <Clock className="w-3.5 h-3.5 text-amber-400" />
                <span className="font-semibold">{formatTime(timeRemainingSec)}</span>
                <span className="text-[10px] text-slate-400 font-sans hidden sm:inline">remaining</span>
              </div>
            )}
          </div>

          {/* Network / Currency Selector */}
          {options.length > 0 && (!invoice || invoice.status === 'pending') && (
            <div className="p-4 sm:p-5 bg-slate-950/40 border-b border-slate-800/80 space-y-3">
              <div className="flex items-center justify-between flex-wrap gap-2">
                <label className="text-[11px] font-bold uppercase tracking-wider text-slate-400 flex items-center gap-1.5">
                  <span>Select Payment Currency &amp; Network</span>
                </label>

                {/* Filter Pills */}
                <div className="flex items-center gap-1 bg-slate-900 p-1 rounded-xl border border-slate-800 text-[11px]">
                  <button
                    onClick={() => setActiveFilter('all')}
                    className={`px-2.5 py-1 rounded-lg transition font-medium cursor-pointer ${
                      activeFilter === 'all' ? 'bg-emerald-600 text-white shadow-sm' : 'text-slate-400 hover:text-slate-200'
                    }`}
                  >
                    All ({options.length})
                  </button>
                  <button
                    onClick={() => setActiveFilter('stable')}
                    className={`px-2.5 py-1 rounded-lg transition font-medium cursor-pointer ${
                      activeFilter === 'stable' ? 'bg-emerald-600 text-white shadow-sm' : 'text-slate-400 hover:text-slate-200'
                    }`}
                  >
                    Stablecoins
                  </button>
                  <button
                    onClick={() => setActiveFilter('fast')}
                    className={`px-2.5 py-1 rounded-lg transition font-medium cursor-pointer ${
                      activeFilter === 'fast' ? 'bg-emerald-600 text-white shadow-sm' : 'text-slate-400 hover:text-slate-200'
                    }`}
                  >
                    Low Fee
                  </button>
                  <button
                    onClick={() => setActiveFilter('major')}
                    className={`px-2.5 py-1 rounded-lg transition font-medium cursor-pointer ${
                      activeFilter === 'major' ? 'bg-emerald-600 text-white shadow-sm' : 'text-slate-400 hover:text-slate-200'
                    }`}
                  >
                    Major
                  </button>
                </div>
              </div>

              {/* Currency Buttons Grid */}
              <div className="grid grid-cols-2 sm:grid-cols-3 gap-2.5 max-h-56 overflow-y-auto pr-1 select-none">
                {filteredOptions.map((opt) => {
                  const isSelected = selectedCurrencyId === opt.id;
                  const isAvailable = !!opt.has_treasury;
                  const theme = COIN_THEMES[opt.symbol.toUpperCase()] || COIN_THEMES.USDT;

                  return (
                    <button
                      key={opt.id}
                      onClick={() => {
                        setSelectedCurrencyId(opt.id);
                        initInvoice(opt.id);
                      }}
                      className={`p-3 rounded-2xl border text-left transition-all duration-150 flex flex-col justify-between gap-1.5 relative group ${
                        !isAvailable
                          ? isSelected
                            ? 'border-amber-500/60 bg-amber-500/10 text-amber-200 cursor-pointer'
                            : 'border-slate-800/80 bg-slate-900/30 text-slate-500 hover:border-slate-700 cursor-pointer'
                          : isSelected
                          ? 'border-emerald-500 bg-emerald-500/15 text-white ring-2 ring-emerald-500 shadow-lg shadow-emerald-500/10 cursor-pointer'
                          : 'border-slate-800 bg-slate-850 hover:bg-slate-800/80 hover:border-slate-700 text-slate-300 cursor-pointer'
                      }`}
                    >
                      <div className="flex items-center justify-between w-full">
                        <div className="flex items-center gap-2">
                          <div
                            className={`w-7 h-7 rounded-xl flex items-center justify-center font-bold text-xs ${theme.bg} ${theme.text} border ${theme.border}`}
                          >
                            {theme.symbol}
                          </div>
                          <div>
                            <span className="font-extrabold text-xs block text-white">{opt.symbol}</span>
                            <span className="text-[10px] text-slate-400 uppercase font-mono font-medium block">
                              {opt.network_id}
                            </span>
                          </div>
                        </div>

                        {/* Status Badge */}
                        <div className="flex items-center">
                          {isAvailable ? (
                            <span className="flex items-center gap-1 text-[10px] font-semibold text-emerald-400 bg-emerald-500/10 border border-emerald-500/20 px-1.5 py-0.5 rounded-md">
                              <span className="w-1.5 h-1.5 rounded-full bg-emerald-400 animate-pulse" />
                              Ready
                            </span>
                          ) : (
                            <span className="text-[9px] uppercase font-semibold text-slate-500 bg-slate-800/80 px-1.5 py-0.5 rounded">
                              Inactive
                            </span>
                          )}
                        </div>
                      </div>

                      <div className="flex items-center justify-between text-[10px] text-slate-400 border-t border-white/5 pt-1 mt-0.5">
                        <span className="truncate">{opt.name}</span>
                        {isAvailable && (
                          <span className="text-[9px] font-mono text-emerald-400/90 font-semibold uppercase">
                            Instant
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
          <div className="p-5 sm:p-7 space-y-6">
            {/* Error Message Box */}
            {error && (
              <div className="p-5 bg-rose-950/30 border border-rose-800/60 rounded-2xl text-xs text-rose-300 space-y-3.5">
                <div className="flex items-start gap-3">
                  <div className="p-2 rounded-xl bg-rose-500/20 text-rose-400 shrink-0">
                    <AlertTriangle className="w-5 h-5" />
                  </div>
                  <div className="space-y-1">
                    <p className="font-bold text-sm text-rose-200">
                      {error === 'NO_TREASURY_AVAILABLE' || error.includes('no treasury destination address')
                        ? 'Crypto Network Setup Required'
                        : 'Unable to Generate Crypto Invoice'}
                    </p>
                    <p className="text-rose-300/90 leading-relaxed text-xs">
                      {error === 'NO_TREASURY_AVAILABLE' || error.includes('no treasury destination address')
                        ? 'The merchant has not yet configured on-chain recipient addresses for this token. Please select an active cryptocurrency marked with a green dot above, or complete payment via Card.'
                        : error}
                    </p>
                  </div>
                </div>

                <div className="pt-2 border-t border-rose-900/40 flex flex-wrap items-center gap-2">
                  {isAdmin && (
                    <button
                      onClick={() => router.push('/admin/crypto')}
                      className="px-3.5 py-2 rounded-xl bg-emerald-600 hover:bg-emerald-500 text-white text-xs font-bold transition flex items-center gap-1.5 cursor-pointer shadow-md"
                    >
                      <Zap className="w-3.5 h-3.5" />
                      <span>Setup Wallet Addresses in Admin</span>
                      <ExternalLink className="w-3.5 h-3.5" />
                    </button>
                  )}
                  <button
                    onClick={() => router.push('/billing')}
                    className="px-3.5 py-2 rounded-xl bg-slate-800 hover:bg-slate-700 text-slate-200 text-xs font-semibold transition cursor-pointer border border-slate-700"
                  >
                    Pay with Card / Stripe
                  </button>
                </div>
              </div>
            )}

            {/* Loading Spinner */}
            {!error && (loading || creating) && (
              <div className="py-20 flex flex-col items-center justify-center gap-4 text-slate-400">
                <div className="w-14 h-14 rounded-2xl bg-emerald-500/10 border border-emerald-500/20 flex items-center justify-center text-emerald-400">
                  <RefreshCw className="w-7 h-7 animate-spin" />
                </div>
                <div className="text-center space-y-1">
                  <p className="text-sm font-bold text-white">Generating Dedicated Deposit Address</p>
                  <p className="text-xs text-slate-400">Locking live blockchain exchange rate and securing session...</p>
                </div>
              </div>
            )}

            {/* Active Invoice Content */}
            {!error && !loading && !creating && invoice && (
              <>
                {/* 1. PAID SUCCESS BANNER */}
                {invoice.status === 'paid' ? (
                  <div className="p-8 bg-emerald-950/40 border border-emerald-500/40 rounded-3xl text-center space-y-5">
                    <div className="w-20 h-20 bg-gradient-to-tr from-emerald-500/30 to-teal-500/20 text-emerald-400 rounded-3xl flex items-center justify-center mx-auto border border-emerald-500/50 shadow-2xl shadow-emerald-500/20">
                      <CheckCircle2 className="w-12 h-12" />
                    </div>
                    <div className="space-y-1.5">
                      <h2 className="text-xl font-bold text-white flex items-center justify-center gap-2">
                        <Sparkles className="w-5 h-5 text-emerald-400" />
                        Payment Verified &amp; Settled!
                      </h2>
                      <p className="text-sm text-slate-300">
                        Received on-chain:{' '}
                        <strong className="text-emerald-400 font-mono text-base">
                          {invoice.amount_received} {invoice.currency_id}
                        </strong>
                      </p>
                      <p className="text-xs text-slate-400">
                        Your hosting services and subscription have been activated automatically.
                      </p>
                    </div>

                    {invoice.tx_hash && (
                      <div className="bg-slate-950 p-4 rounded-2xl border border-slate-800 text-left font-mono text-xs break-all space-y-1">
                        <div className="text-[10px] text-slate-400 uppercase font-semibold">Transaction ID (TxHash)</div>
                        <div className="text-emerald-400 font-medium">{invoice.tx_hash}</div>
                      </div>
                    )}

                    <button
                      onClick={() => router.push('/billing')}
                      className="w-full py-3.5 bg-emerald-600 hover:bg-emerald-500 text-white rounded-2xl font-bold text-sm transition cursor-pointer shadow-xl shadow-emerald-600/20 flex items-center justify-center gap-2"
                    >
                      <span>Return to Invoices &amp; Subscriptions</span>
                      <ArrowUpRight className="w-4 h-4" />
                    </button>
                  </div>
                ) : (
                  <>
                    {/* 2. REAL-TIME PROGRESS TRACKER */}
                    <div className="p-4 rounded-2xl bg-slate-950/70 border border-slate-800 space-y-3">
                      <div className="flex items-center justify-between text-xs">
                        <div className="flex items-center gap-2.5">
                          {invoice.status === 'confirming' ? (
                            <RefreshCw className="w-4 h-4 text-sky-400 animate-spin" />
                          ) : invoice.status === 'detecting' ? (
                            <RefreshCw className="w-4 h-4 text-amber-400 animate-spin" />
                          ) : (
                            <span className="relative flex h-3 w-3">
                              <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-emerald-400 opacity-75" />
                              <span className="relative inline-flex rounded-full h-3 w-3 bg-emerald-500" />
                            </span>
                          )}
                          <span className="font-bold text-sm text-white">
                            {invoice.status === 'confirming'
                              ? `Confirming On-Chain (${invoice.confirmations} / ${invoice.min_confirmations} blocks)`
                              : invoice.status === 'detecting'
                              ? 'Deposit Detected! Verifying amount...'
                              : 'Awaiting On-Chain Deposit'}
                          </span>
                        </div>

                        <button
                          onClick={() => copyToClipboard(invoice.invoice_code, 'code')}
                          className="text-[11px] font-mono text-slate-400 hover:text-white flex items-center gap-1 cursor-pointer bg-slate-900 px-2 py-1 rounded-lg border border-slate-800"
                          title="Copy Invoice Code"
                        >
                          <span>{invoice.invoice_code}</span>
                          {copiedCode ? <Check className="w-3 h-3 text-emerald-400" /> : <Copy className="w-3 h-3" />}
                        </button>
                      </div>

                      {/* Visual Stepper */}
                      <div className="grid grid-cols-3 gap-1 pt-1 text-[10px] font-medium text-center">
                        <div className="p-2 rounded-xl bg-emerald-500/15 border border-emerald-500/30 text-emerald-300 flex items-center justify-center gap-1">
                          <Check className="w-3 h-3" />
                          <span>1. Created</span>
                        </div>
                        <div
                          className={`p-2 rounded-xl border flex items-center justify-center gap-1 ${
                            invoice.status === 'confirming' || invoice.status === 'detecting'
                              ? 'bg-amber-500/15 border-amber-500/30 text-amber-300'
                              : 'bg-slate-900 border-slate-800 text-slate-400'
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
                          className={`p-2 rounded-xl border flex items-center justify-center gap-1 ${
                            invoice.status === 'paid'
                              ? 'bg-emerald-500/15 border-emerald-500/30 text-emerald-300'
                              : 'bg-slate-900 border-slate-800 text-slate-500'
                          }`}
                        >
                          <span>3.</span>
                          <span>Settled</span>
                        </div>
                      </div>
                    </div>

                    {/* 3. QR CODE & AMOUNT CARD */}
                    <div className="flex flex-col sm:flex-row items-center gap-6 bg-gradient-to-br from-slate-950 via-slate-900 to-slate-950 p-6 rounded-3xl border border-slate-800 shadow-xl">
                      {/* White QR Container */}
                      <div className="bg-white p-3.5 rounded-2xl shadow-xl shrink-0 flex flex-col items-center gap-2 group">
                        {/* eslint-disable-next-line @next/next/no-img-element */}
                        <img
                          src={`https://api.qrserver.com/v1/create-qr-code/?size=180x180&data=${encodeURIComponent(
                            invoice.qr_uri || invoice.deposit_address
                          )}`}
                          alt="Crypto Deposit QR"
                          className="w-36 h-36 sm:w-40 sm:h-40"
                        />
                        <span className="text-[10px] font-bold text-slate-700 uppercase tracking-wider">
                          Scan to Pay
                        </span>
                      </div>

                      {/* Payment Exact Amount */}
                      <div className="space-y-4 w-full text-center sm:text-left">
                        <div>
                          <span className="text-[11px] text-slate-400 uppercase font-bold tracking-wider block">
                            Send Exactly
                          </span>
                          <div className="flex items-baseline justify-center sm:justify-start gap-2 mt-1">
                            <span className="text-3xl sm:text-4xl font-extrabold text-white font-mono tracking-tight">
                              {invoice.crypto_amount}
                            </span>
                            <span className={`text-base font-bold ${coinTheme.accent}`}>
                              {currentOption?.symbol || 'USDT'}
                            </span>
                          </div>
                          <div className="text-xs text-slate-400 font-mono mt-1">
                            ≈ ${invoice.fiat_amount_usd.toFixed(2)} USD
                            <span className="text-[11px] text-slate-500 ml-2">
                              (1 {currentOption?.symbol} ≈ ${(1 / invoice.exchange_rate).toFixed(2)})
                            </span>
                          </div>
                        </div>

                        {/* Action Buttons */}
                        <div className="flex flex-wrap items-center justify-center sm:justify-start gap-2 pt-1">
                          <button
                            onClick={() => copyToClipboard(invoice.crypto_amount.toString(), 'amount')}
                            className="inline-flex items-center gap-1.5 px-3.5 py-2 rounded-xl bg-slate-800 hover:bg-slate-750 border border-slate-700 text-xs font-bold text-slate-200 transition cursor-pointer shadow-sm active:scale-95"
                          >
                            {copiedAmount ? <Check className="w-3.5 h-3.5 text-emerald-400" /> : <Copy className="w-3.5 h-3.5" />}
                            <span>{copiedAmount ? 'Amount Copied!' : 'Copy Exact Amount'}</span>
                          </button>

                          {walletDeepLink && (
                            <a
                              href={walletDeepLink}
                              target="_blank"
                              rel="noreferrer"
                              className="inline-flex items-center gap-1.5 px-3.5 py-2 rounded-xl bg-emerald-600/20 hover:bg-emerald-600/30 border border-emerald-500/40 text-xs font-bold text-emerald-300 transition cursor-pointer"
                            >
                              <ExternalLink className="w-3.5 h-3.5" />
                              <span>Open in Trust Wallet</span>
                            </a>
                          )}
                        </div>
                      </div>
                    </div>

                    {/* 4. RECIPIENT DEPOSIT ADDRESS BOX */}
                    <div className="space-y-2.5">
                      <div className="flex items-center justify-between text-xs">
                        <label className="text-[11px] font-bold uppercase tracking-wider text-slate-400">
                          {currentOption?.network_name || invoice.network_id.toUpperCase()} Deposit Address:
                        </label>
                        <span className="text-[11px] text-emerald-400 font-medium flex items-center gap-1">
                          <ShieldCheck className="w-3.5 h-3.5" />
                          Zero Intermediary
                        </span>
                      </div>

                      <div className="flex items-center gap-2 p-3.5 bg-slate-950 rounded-2xl border border-slate-800 focus-within:border-emerald-500 transition shadow-inner">
                        <span className="font-mono text-xs sm:text-sm text-slate-200 break-all select-all flex-1 tracking-tight">
                          {invoice.deposit_address}
                        </span>
                        <button
                          onClick={() => copyToClipboard(invoice.deposit_address, 'addr')}
                          className="px-3 py-2 rounded-xl bg-emerald-600 hover:bg-emerald-500 text-white font-bold text-xs transition cursor-pointer shrink-0 shadow-md flex items-center gap-1.5 active:scale-95"
                          title="Copy Address"
                        >
                          {copiedAddr ? <Check className="w-4 h-4" /> : <Copy className="w-4 h-4" />}
                          <span className="hidden sm:inline">{copiedAddr ? 'Copied' : 'Copy'}</span>
                        </button>
                      </div>

                      {/* Safety Banner */}
                      <div className="p-3 bg-amber-500/10 border border-amber-500/20 rounded-xl flex items-start gap-2.5 text-[11px] text-amber-300/90 leading-relaxed">
                        <Info className="w-4 h-4 text-amber-400 shrink-0 mt-0.5" />
                        <div>
                          Send ONLY <strong>{currentOption?.symbol || 'USDT'}</strong> via the{' '}
                          <strong className="underline decoration-amber-400/50">
                            {currentOption?.network_name || invoice.network_id.toUpperCase()}
                          </strong>{' '}
                          network. Transferring any other cryptocurrency or using an incorrect blockchain will result in irreversible fund loss.
                        </div>
                      </div>
                    </div>

                    {/* 5. TRANSACTION HASH FAST-TRACK ACCORDION */}
                    <div className="rounded-2xl border border-slate-800 bg-slate-950/40 overflow-hidden">
                      <button
                        onClick={() => setShowTxDrawer(!showTxDrawer)}
                        className="w-full p-3.5 flex items-center justify-between text-xs text-slate-400 hover:text-slate-200 transition cursor-pointer"
                      >
                        <div className="flex items-center gap-2">
                          <Zap className="w-3.5 h-3.5 text-amber-400" />
                          <span className="font-semibold text-slate-300">
                            Already transferred? Fast-track verification with TxHash
                          </span>
                        </div>
                        {showTxDrawer ? <ChevronUp className="w-4 h-4" /> : <ChevronDown className="w-4 h-4" />}
                      </button>

                      {showTxDrawer && (
                        <div className="p-4 pt-1 border-t border-slate-800/80 space-y-2">
                          <p className="text-[11px] text-slate-400">
                            Paste the blockchain transaction hash / ID from your wallet (e.g. Trust Wallet, Binance, MetaMask):
                          </p>
                          <div className="flex gap-2">
                            <input
                              type="text"
                              placeholder="0x... or transaction id..."
                              value={txHashInput}
                              onChange={(e) => setTxHashInput(e.target.value)}
                              className="flex-1 bg-slate-900 border border-slate-700 rounded-xl px-3.5 py-2 text-xs font-mono text-slate-200 focus:outline-none focus:border-emerald-500 shadow-inner"
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
                    <div className="pt-2 border-t border-slate-800/80 flex items-center justify-between flex-wrap gap-2 text-xs text-slate-400">
                      <button
                        onClick={() => router.push('/billing')}
                        className="hover:text-slate-200 transition flex items-center gap-1.5 cursor-pointer"
                      >
                        <CreditCard className="w-3.5 h-3.5 text-slate-400" />
                        <span>Prefer Credit Card / Stripe? Pay with Card</span>
                      </button>

                      <button
                        onClick={() => router.push('/support')}
                        className="hover:text-slate-200 transition flex items-center gap-1.5 cursor-pointer text-slate-500"
                      >
                        <HelpCircle className="w-3.5 h-3.5" />
                        <span>Need Support?</span>
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
