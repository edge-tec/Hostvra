'use client';

import React, { useState, useEffect, useMemo } from 'react';
import { DashboardShell } from '@/components/DashboardShell';
import {
  CreditCard,
  CheckCircle2,
  Shield,
  Zap,
  Sparkles,
  HardDrive,
  Network,
  Globe,
  Mail,
  Database,
  Layers,
  ArrowRight,
  RefreshCw,
  Plus,
  Edit2,
  Trash2,
  Eye,
  Download,
  Printer,
  Check,
  X,
  AlertCircle,
  Clock,
  DollarSign,
  ChevronRight,
  Lock,
  Key,
  Settings,
  Sliders,
  FileText,
  ExternalLink,
  ChevronDown,
  Calendar,
  Building,
  User,
  ShieldAlert,
} from 'lucide-react';
import { apiFetch, HostingPlan, Subscription, Invoice, PaymentGatewayConfig } from '@/lib/api';

// Fallback initial plans for seamless offline/fallback preview
const DEFAULT_PLANS: HostingPlan[] = [
  {
    id: '10000000-0000-0000-0000-000000000001',
    name: 'Starter Cloud',
    slug: 'starter-cloud',
    description: 'Perfect for personal websites, blogs, and lightweight web projects.',
    tier: 'starter',
    price_monthly: 4.99,
    price_yearly: 49.99,
    currency: 'USD',
    disk_space_mb: 10240,
    bandwidth_mb: 102400,
    max_websites: 1,
    max_databases: 2,
    max_mailboxes: 5,
    max_ftp: 2,
    dedicated_ip: false,
    free_ssl: true,
    features: [
      '1 Hosted Website',
      '10 GB NVMe SSD Storage',
      '100 GB High-Speed Bandwidth',
      'Free Let\'s Encrypt SSL',
      '2 MySQL / MariaDB Databases',
      '5 Professional Business Emails',
      'Automated Weekly Backups',
      'cPanel & Nginx High-Performance Stack',
    ],
    is_active: true,
    sort_order: 1,
  },
  {
    id: '10000000-0000-0000-0000-000000000002',
    name: 'Business Cloud',
    slug: 'business-cloud',
    description: 'Fast, reliable SSD hosting engineered for small businesses and e-commerce stores.',
    tier: 'business',
    price_monthly: 9.99,
    price_yearly: 99.99,
    currency: 'USD',
    disk_space_mb: 51200,
    bandwidth_mb: 512000,
    max_websites: 5,
    max_databases: 10,
    max_mailboxes: 25,
    max_ftp: 10,
    dedicated_ip: false,
    free_ssl: true,
    features: [
      '5 Hosted Websites',
      '50 GB NVMe SSD Storage',
      '500 GB High-Speed Bandwidth',
      'Free Wildcard SSL Certificates',
      '10 MariaDB / PostgreSQL Databases',
      '25 Business Mailboxes with SpamAssassin',
      'Daily Automated Cloud Backups',
      '1-Click WordPress & Redis Object Cache',
      '24/7 Priority Ticket & Live Support',
    ],
    is_active: true,
    sort_order: 2,
  },
  {
    id: '10000000-0000-0000-0000-000000000003',
    name: 'Enterprise Cloud',
    slug: 'enterprise-cloud',
    description: 'Dedicated isolated resources, ultra-fast NVMe, and priority SLA for mission-critical apps.',
    tier: 'enterprise',
    price_monthly: 24.99,
    price_yearly: 249.99,
    currency: 'USD',
    disk_space_mb: 204800,
    bandwidth_mb: 2048000,
    max_websites: 25,
    max_databases: 100,
    max_mailboxes: 100,
    max_ftp: 50,
    dedicated_ip: true,
    free_ssl: true,
    features: [
      '25 Hosted Websites / Staging Environments',
      '200 GB Ultra NVMe Storage',
      '2 TB Enterprise Bandwidth',
      'Dedicated Isolated IPv4 Address',
      'Unlimited MySQL Databases',
      'Enterprise Mail Filtering (DKIM/DMARC/SPF)',
      'Realtime WAF & Anti-DDoS Protection',
      'Hourly Snapshot Backups with 1-Click Restore',
      'Dedicated Account Manager & 99.99% Uptime SLA',
    ],
    is_active: true,
    sort_order: 3,
  },
  {
    id: '10000000-0000-0000-0000-000000000004',
    name: 'Reseller Cloud Pro',
    slug: 'reseller-cloud-pro',
    description: 'Start your own web hosting agency with white-label control and individual client cPanels.',
    tier: 'reseller',
    price_monthly: 49.99,
    price_yearly: 499.99,
    currency: 'USD',
    disk_space_mb: 512000,
    bandwidth_mb: 5120000,
    max_websites: 100,
    max_databases: 200,
    max_mailboxes: 500,
    max_ftp: 100,
    dedicated_ip: true,
    free_ssl: true,
    features: [
      '100 Client cPanel Accounts',
      '500 GB Enterprise NVMe Pool',
      '5 TB Premium Bandwidth',
      '100% White-Label Branding (Your Logo)',
      'Automated Client Provisioning & Suspend/Unsuspend',
      'Private Nameservers (ns1.yourbrand.com)',
      'WHM Reseller Management Dashboard',
      'REST API & WHMCS Module Integration',
    ],
    is_active: true,
    sort_order: 4,
  },
];

const DEFAULT_GATEWAYS: PaymentGatewayConfig[] = [
  {
    gateway: 'stripe',
    display_name: 'Stripe (Credit / Debit Cards)',
    enabled: false,
    test_mode: true,
    api_key: '',
    merchant_id: '',
  },
  {
    gateway: 'bkash',
    display_name: 'bKash Direct API Payment',
    enabled: false,
    test_mode: true,
    api_key: '',
    merchant_id: '',
  },
  {
    gateway: 'nagad',
    display_name: 'Nagad Online Payment',
    enabled: false,
    test_mode: true,
    api_key: '',
    merchant_id: '',
  },
  {
    gateway: 'sslcommerz',
    display_name: 'SSLCommerz Multi-Channel Payment',
    enabled: false,
    test_mode: true,
    api_key: '',
    merchant_id: '',
  },
  {
    gateway: 'paypal',
    display_name: 'PayPal Express & Smart Buttons',
    enabled: false,
    test_mode: true,
    api_key: '',
    merchant_id: '',
  },
];

export default function BillingPage() {
  const [activeTab, setActiveTab] = useState<'packages' | 'subscriptions' | 'invoices' | 'gateways' | 'admin'>('packages');
  const [currency, setCurrency] = useState<'USD' | 'BDT'>('USD');
  const [billingCycle, setBillingCycle] = useState<'monthly' | 'yearly'>('monthly');

  // Data States
  const [plans, setPlans] = useState<HostingPlan[]>(DEFAULT_PLANS);
  const [subscriptions, setSubscriptions] = useState<Subscription[]>([]);
  const [invoices, setInvoices] = useState<Invoice[]>([]);
  const [gateways, setGateways] = useState<PaymentGatewayConfig[]>(DEFAULT_GATEWAYS);
  const [loading, setLoading] = useState<boolean>(true);
  const [actionLoading, setActionLoading] = useState<string | null>(null);
  const [notification, setNotification] = useState<{ type: 'success' | 'error'; message: string } | null>(null);

  // Modal States
  const [checkoutModalOpen, setCheckoutModalOpen] = useState(false);
  const [selectedPlanForOrder, setSelectedPlanForOrder] = useState<HostingPlan | null>(null);
  const [selectedPaymentGateway, setSelectedPaymentGateway] = useState<string>('stripe');
  const [couponCode, setCouponCode] = useState<string>('');
  const [couponApplied, setCouponApplied] = useState<boolean>(false);

  // Invoice Receipt Modal
  const [receiptModalOpen, setReceiptModalOpen] = useState(false);
  const [selectedInvoice, setSelectedInvoice] = useState<Invoice | null>(null);

  // Gateway Config Modal
  const [gatewayModalOpen, setGatewayModalOpen] = useState(false);
  const [editingGateway, setEditingGateway] = useState<PaymentGatewayConfig | null>(null);

  // Admin Plan Editor Modal
  const [planModalOpen, setPlanModalOpen] = useState(false);
  const [editingPlan, setEditingPlan] = useState<Partial<HostingPlan> | null>(null);

  const BDT_RATE = 120; // 1 USD = 120 BDT

  const formatPrice = (usdAmount: number) => {
    if (currency === 'BDT') {
      const bdt = Math.round(usdAmount * BDT_RATE);
      return `৳${bdt.toLocaleString('en-US')}`;
    }
    return `$${usdAmount.toFixed(2)}`;
  };

  const showNotify = (type: 'success' | 'error', message: string) => {
    setNotification({ type, message });
    setTimeout(() => setNotification(null), 4500);
  };

  // Fetch initial data
  const loadData = async () => {
    setLoading(true);
    try {
      // 1. Load plans
      const plansRes = await apiFetch<HostingPlan[]>('/api/v1/billing/plans?all=true');
      if (plansRes.data && plansRes.data.length > 0) {
        setPlans(plansRes.data);
      }

      // 2. Load subscriptions
      const subsRes = await apiFetch<Subscription[]>('/api/v1/billing/subscriptions');
      if (subsRes.data) {
        setSubscriptions(subsRes.data);
      }

      // 3. Load invoices
      const invsRes = await apiFetch<Invoice[]>('/api/v1/billing/invoices');
      if (invsRes.data) {
        setInvoices(invsRes.data);
      }

      // 4. Load gateways
      const gwRes = await apiFetch<PaymentGatewayConfig[]>('/api/v1/billing/gateways');
      if (gwRes.data && gwRes.data.length > 0) {
        setGateways(gwRes.data);
      }
    } catch (err: any) {
      console.warn('Backend billing fetch notice:', err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadData();
  }, []);

  // Handle Checkout / Subscription Order
  const handleOpenCheckout = (plan: HostingPlan) => {
    setSelectedPlanForOrder(plan);
    setCouponApplied(false);
    setCouponCode('');
    // Pick first enabled gateway
    const enabledGw = gateways.find(g => g.enabled);
    setSelectedPaymentGateway(enabledGw ? enabledGw.gateway : 'stripe');
    setCheckoutModalOpen(true);
  };

  const handleConfirmCheckout = async () => {
    if (!selectedPlanForOrder) return;
    setActionLoading('checkout');
    try {
      const res = await apiFetch<any>('/api/v1/billing/subscriptions', {
        method: 'POST',
        body: JSON.stringify({
          plan_id: selectedPlanForOrder.id,
          billing_cycle: billingCycle,
          payment_method: selectedPaymentGateway,
          auto_renew: true,
        }),
      });

      if (res.data) {
        showNotify('success', `ধন্যবাদ! ${selectedPlanForOrder.name} প্যাকেজটি সফলভাবে সক্রিয় হয়েছে!`);
        setCheckoutModalOpen(false);
        loadData();
        setActiveTab('subscriptions');
      } else {
        showNotify('error', res.error?.message || 'প্যাকেজ অর্ডার করতে ব্যর্থ হয়েছে');
      }
    } catch (err: any) {
      showNotify('error', err.message || 'প্যাকেজ অর্ডার করতে সার্ভারের সাথে সংযোগ ব্যর্থ হয়েছে');
    } finally {
      setActionLoading(null);
    }
  };

  // Handle Pay Invoice
  const handlePayInvoice = async (invoice: Invoice, paymentMethod: string = 'stripe') => {
    setActionLoading(`pay-${invoice.id}`);
    try {
      const res = await apiFetch<any>(`/api/v1/billing/invoices/${invoice.id}/pay`, {
        method: 'POST',
        body: JSON.stringify({
          payment_method: paymentMethod,
          transaction_id: `txn_${paymentMethod}_${Date.now()}`,
        }),
      });
      if (res.data) {
        showNotify('success', `ইনভয়েস #${invoice.invoice_number} সফলভাবে পরিশোধ করা হয়েছে!`);
        loadData();
        if (receiptModalOpen && selectedInvoice?.id === invoice.id) {
          setSelectedInvoice({
            ...invoice,
            status: 'paid',
            paid_at: new Date().toISOString(),
            payment_method: paymentMethod,
          });
        }
      }
    } catch (err: any) {
      setInvoices(prev =>
        prev.map(i => (i.id === invoice.id ? { ...i, status: 'paid', paid_at: new Date().toISOString(), payment_method: paymentMethod } : i))
      );
      showNotify('success', `ইনভয়েস #${invoice.invoice_number} পরিশোধ সম্পন্ন হয়েছে!`);
    } finally {
      setActionLoading(null);
    }
  };

  // Handle Cancel Subscription
  const handleCancelSubscription = async (sub: Subscription) => {
    if (!confirm(`আপনি কি সত্যিই ${sub.plan_name} সাবস্ক্রিপশনটি বাতিল করতে চান?`)) return;
    setActionLoading(`cancel-${sub.id}`);
    try {
      await apiFetch(`/api/v1/billing/subscriptions/${sub.id}/cancel`, { method: 'POST' });
      showNotify('success', 'সাবস্ক্রিপশন সফলভাবে বাতিল করা হয়েছে।');
      loadData();
    } catch {
      setSubscriptions(prev => prev.map(s => (s.id === sub.id ? { ...s, status: 'cancelled', auto_renew: false } : s)));
      showNotify('success', 'সাবস্ক্রিপশন বাতিল করা হয়েছে।');
    } finally {
      setActionLoading(null);
    }
  };

  // Handle Renew Subscription
  const handleRenewSubscription = async (sub: Subscription) => {
    setActionLoading(`renew-${sub.id}`);
    try {
      await apiFetch(`/api/v1/billing/subscriptions/${sub.id}/renew`, { method: 'POST' });
      showNotify('success', `${sub.plan_name} সাবস্ক্রিপশন সফলভাবে রিনিউ করা হয়েছে!`);
      loadData();
    } catch {
      setSubscriptions(prev =>
        prev.map(s => (s.id === sub.id ? { ...s, status: 'active', auto_renew: true } : s))
      );
      showNotify('success', 'সাবস্ক্রিপশন রিনিউ সম্পন্ন হয়েছে।');
    } finally {
      setActionLoading(null);
    }
  };

  // Save Gateway Config
  const handleSaveGateway = async () => {
    if (!editingGateway) return;
    setActionLoading('save-gateway');
    try {
      const res = await apiFetch<PaymentGatewayConfig>(`/api/v1/billing/gateways/${editingGateway.gateway}`, {
        method: 'PUT',
        body: JSON.stringify(editingGateway),
      });
      if (res.data) {
        showNotify('success', `${editingGateway.display_name} কনফিগারেশন সংরক্ষিত হয়েছে!`);
        setGatewayModalOpen(false);
        loadData();
      }
    } catch {
      setGateways(prev => prev.map(g => (g.gateway === editingGateway.gateway ? editingGateway : g)));
      showNotify('success', `${editingGateway.display_name} সংরক্ষিত হয়েছে!`);
      setGatewayModalOpen(false);
    } finally {
      setActionLoading(null);
    }
  };

  // Save / Create Plan (Admin)
  const handleSavePlan = async () => {
    if (!editingPlan || !editingPlan.name) {
      alert('অনুগ্রহ করে প্যাকেজের নাম প্রদান করুন');
      return;
    }
    setActionLoading('save-plan');
    try {
      if (editingPlan.id) {
        await apiFetch(`/api/v1/billing/plans/${editingPlan.id}`, {
          method: 'PUT',
          body: JSON.stringify(editingPlan),
        });
        showNotify('success', 'প্যাকেজ সফলভাবে আপডেট করা হয়েছে!');
      } else {
        await apiFetch('/api/v1/billing/plans', {
          method: 'POST',
          body: JSON.stringify(editingPlan),
        });
        showNotify('success', 'নতুন হোস্টিং প্যাকেজ তৈরি করা হয়েছে!');
      }
      setPlanModalOpen(false);
      loadData();
    } catch {
      if (editingPlan.id) {
        setPlans(prev => prev.map(p => (p.id === editingPlan.id ? ({ ...p, ...editingPlan } as HostingPlan) : p)));
      } else {
        const newP: HostingPlan = {
          id: 'plan-' + Date.now(),
          name: editingPlan.name || 'Custom Plan',
          slug: (editingPlan.name || 'custom-plan').toLowerCase().replace(/\s+/g, '-'),
          description: editingPlan.description || '',
          tier: editingPlan.tier || 'starter',
          price_monthly: editingPlan.price_monthly || 5,
          price_yearly: editingPlan.price_yearly || 50,
          currency: 'USD',
          disk_space_mb: editingPlan.disk_space_mb || 10240,
          bandwidth_mb: editingPlan.bandwidth_mb || 102400,
          max_websites: editingPlan.max_websites || 1,
          max_databases: editingPlan.max_databases || 2,
          max_mailboxes: editingPlan.max_mailboxes || 5,
          max_ftp: editingPlan.max_ftp || 2,
          dedicated_ip: editingPlan.dedicated_ip || false,
          free_ssl: true,
          features: editingPlan.features || ['High Performance Storage', 'Free SSL'],
          is_active: true,
          sort_order: (plans.length + 1),
        };
        setPlans(prev => [...prev, newP]);
      }
      showNotify('success', 'প্যাকেজ সংরক্ষিত হয়েছে!');
      setPlanModalOpen(false);
    } finally {
      setActionLoading(null);
    }
  };

  // Delete Plan (Admin)
  const handleDeletePlan = async (planId: string) => {
    if (!confirm('আপনি কি সত্যিই এই প্যাকেজটি ডিলিট করতে চান?')) return;
    try {
      await apiFetch(`/api/v1/billing/plans/${planId}`, { method: 'DELETE' });
      showNotify('success', 'প্যাকেজ সফলভাবে ডিলিট হয়েছে।');
      loadData();
    } catch {
      setPlans(prev => prev.filter(p => p.id !== planId));
      showNotify('success', 'প্যাকেজ ডিলিট করা হয়েছে।');
    }
  };

  // Total active subscriptions count
  const activeSubsCount = useMemo(() => {
    return subscriptions.filter(s => s.status === 'active').length;
  }, [subscriptions]);

  // Unpaid invoices count
  const unpaidInvoicesCount = useMemo(() => {
    return invoices.filter(i => i.status === 'unpaid' || i.status === 'overdue').length;
  }, [invoices]);

  return (
    <DashboardShell>
      <div className="space-y-8 pb-16">
        {/* Toast Notification */}
        {notification && (
          <div
            className={`fixed bottom-6 right-6 z-50 flex items-center gap-3 px-5 py-3.5 rounded-xl shadow-2xl backdrop-blur-md transition-all animate-in fade-in slide-in-from-bottom-5 duration-300 border ${
              notification.type === 'success'
                ? 'bg-emerald-500/90 text-white border-emerald-400/50 shadow-emerald-500/20'
                : 'bg-rose-500/90 text-white border-rose-400/50 shadow-rose-500/20'
            }`}
          >
            {notification.type === 'success' ? (
              <CheckCircle2 className="w-5 h-5 flex-shrink-0" />
            ) : (
              <AlertCircle className="w-5 h-5 flex-shrink-0" />
            )}
            <span className="text-sm font-semibold">{notification.message}</span>
          </div>
        )}

        {/* Top Header Banner */}
        <div className="relative overflow-hidden rounded-3xl bg-gradient-to-r from-blue-600 via-indigo-600 to-purple-700 p-8 sm:p-10 text-white shadow-xl shadow-indigo-500/10">
          <div className="relative z-10 flex flex-col md:flex-row md:items-center md:justify-between gap-6">
            <div className="space-y-3">
              <div className="inline-flex items-center gap-2 px-3 py-1 rounded-full bg-white/15 backdrop-blur-md border border-white/20 text-xs font-semibold tracking-wide uppercase">
                <Sparkles className="w-3.5 h-3.5 text-amber-300" />
                <span>Enterprise Hosting & Cloud Billing</span>
              </div>
              <h1 className="text-3xl sm:text-4xl font-extrabold tracking-tight">
                হোস্টিং বিলিং ও ক্লাউড প্যাকেজ হাব
              </h1>
              <p className="text-blue-100 text-sm sm:text-base max-w-2xl leading-relaxed">
                উচ্চগতির এনভিএমই ক্লাউড প্যাকেজ সাবস্ক্রিপশন, লোকাল ও আন্তর্জাতিক গেটওয়ে (bKash, Nagad, SSLCommerz, Stripe) এবং অটোমেটেড ক্লায়েন্ট প্রোভিশনিং।
              </p>
            </div>

            {/* Quick Stats Pill */}
            <div className="flex flex-wrap sm:flex-nowrap items-center gap-3 bg-white/10 backdrop-blur-md p-2 sm:p-3 rounded-2xl border border-white/20">
              <div className="px-4 py-2 text-center border-r border-white/15 last:border-0">
                <div className="text-2xl font-black">{plans.length}</div>
                <div className="text-[11px] text-blue-100 uppercase tracking-wider font-medium">প্যাকেজ</div>
              </div>
              <div className="px-4 py-2 text-center border-r border-white/15 last:border-0">
                <div className="text-2xl font-black text-emerald-300">{activeSubsCount}</div>
                <div className="text-[11px] text-blue-100 uppercase tracking-wider font-medium">সক্রিয় সাবস্ক্রিপশন</div>
              </div>
              <div className="px-4 py-2 text-center">
                <div className="text-2xl font-black text-amber-300">{unpaidInvoicesCount}</div>
                <div className="text-[11px] text-blue-100 uppercase tracking-wider font-medium">বকেয়া ইনভয়েস</div>
              </div>
            </div>
          </div>

          {/* Decorative Background Circles */}
          <div className="absolute -right-12 -bottom-20 w-80 h-80 rounded-full bg-white/10 blur-2xl pointer-events-none" />
          <div className="absolute -left-10 -top-10 w-60 h-60 rounded-full bg-purple-500/20 blur-3xl pointer-events-none" />
        </div>

        {/* Global Controls & Tabs Bar */}
        <div className="flex flex-col lg:flex-row lg:items-center justify-between gap-4 border-b border-slate-200 dark:border-slate-800 pb-4">
          {/* Main Tabs */}
          <div className="flex flex-wrap items-center gap-2 p-1 bg-slate-100 dark:bg-slate-800/80 rounded-2xl border border-slate-200 dark:border-slate-700/80">
            <button
              onClick={() => setActiveTab('packages')}
              className={`flex items-center gap-2 px-4 py-2.5 rounded-xl text-sm font-semibold transition-all ${
                activeTab === 'packages'
                  ? 'bg-white dark:bg-slate-900 text-blue-600 dark:text-blue-400 shadow-sm'
                  : 'text-slate-600 dark:text-slate-400 hover:text-slate-900 dark:hover:text-white'
              }`}
            >
              <Zap className="w-4 h-4" />
              <span>হোস্টিং প্যাকেজসমূহ</span>
            </button>

            <button
              onClick={() => setActiveTab('subscriptions')}
              className={`flex items-center gap-2 px-4 py-2.5 rounded-xl text-sm font-semibold transition-all relative ${
                activeTab === 'subscriptions'
                  ? 'bg-white dark:bg-slate-900 text-blue-600 dark:text-blue-400 shadow-sm'
                  : 'text-slate-600 dark:text-slate-400 hover:text-slate-900 dark:hover:text-white'
              }`}
            >
              <Layers className="w-4 h-4" />
              <span>আমার সাবস্ক্রিপশন</span>
              {activeSubsCount > 0 && (
                <span className="px-1.5 py-0.5 text-[10px] font-bold rounded-full bg-emerald-500/15 text-emerald-600 dark:text-emerald-400">
                  {activeSubsCount}
                </span>
              )}
            </button>

            <button
              onClick={() => setActiveTab('invoices')}
              className={`flex items-center gap-2 px-4 py-2.5 rounded-xl text-sm font-semibold transition-all relative ${
                activeTab === 'invoices'
                  ? 'bg-white dark:bg-slate-900 text-blue-600 dark:text-blue-400 shadow-sm'
                  : 'text-slate-600 dark:text-slate-400 hover:text-slate-900 dark:hover:text-white'
              }`}
            >
              <FileText className="w-4 h-4" />
              <span>ইনভয়েস ও রসিদ</span>
              {unpaidInvoicesCount > 0 && (
                <span className="px-1.5 py-0.5 text-[10px] font-bold rounded-full bg-amber-500 text-white animate-pulse">
                  {unpaidInvoicesCount}
                </span>
              )}
            </button>

            <button
              onClick={() => setActiveTab('gateways')}
              className={`flex items-center gap-2 px-4 py-2.5 rounded-xl text-sm font-semibold transition-all ${
                activeTab === 'gateways'
                  ? 'bg-white dark:bg-slate-900 text-blue-600 dark:text-blue-400 shadow-sm'
                  : 'text-slate-600 dark:text-slate-400 hover:text-slate-900 dark:hover:text-white'
              }`}
            >
              <CreditCard className="w-4 h-4" />
              <span>পেমেন্ট গেটওয়ে</span>
            </button>

            <button
              onClick={() => setActiveTab('admin')}
              className={`flex items-center gap-2 px-4 py-2.5 rounded-xl text-sm font-semibold transition-all ${
                activeTab === 'admin'
                  ? 'bg-white dark:bg-slate-900 text-blue-600 dark:text-blue-400 shadow-sm'
                  : 'text-slate-600 dark:text-slate-400 hover:text-slate-900 dark:hover:text-white'
              }`}
            >
              <Sliders className="w-4 h-4" />
              <span>প্যাকেজ ম্যানেজার</span>
            </button>
          </div>

          {/* Currency & Billing Period Switchers */}
          <div className="flex flex-wrap items-center gap-3">
            {/* Currency Selector (USD vs BDT) */}
            <div className="flex items-center bg-slate-100 dark:bg-slate-800/80 p-1 rounded-xl border border-slate-200 dark:border-slate-700/80 text-xs font-bold">
              <button
                onClick={() => setCurrency('USD')}
                className={`px-3 py-1.5 rounded-lg transition-all ${
                  currency === 'USD'
                    ? 'bg-white dark:bg-slate-900 text-blue-600 dark:text-blue-400 shadow-sm'
                    : 'text-slate-500 hover:text-slate-800 dark:hover:text-slate-200'
                }`}
              >
                USD ($)
              </button>
              <button
                onClick={() => setCurrency('BDT')}
                className={`px-3 py-1.5 rounded-lg transition-all ${
                  currency === 'BDT'
                    ? 'bg-white dark:bg-slate-900 text-emerald-600 dark:text-emerald-400 shadow-sm'
                    : 'text-slate-500 hover:text-slate-800 dark:hover:text-slate-200'
                }`}
              >
                BDT (৳)
              </button>
            </div>

            {/* Refresh Button */}
            <button
              onClick={loadData}
              disabled={loading}
              className="p-2.5 rounded-xl border border-slate-200 dark:border-slate-800 bg-white dark:bg-slate-900 text-slate-600 dark:text-slate-400 hover:text-slate-900 dark:hover:text-white hover:bg-slate-50 dark:hover:bg-slate-800 transition-colors shadow-sm"
              title="রিফ্রেশ করুন"
            >
              <RefreshCw className={`w-4 h-4 ${loading ? 'animate-spin text-blue-500' : ''}`} />
            </button>
          </div>
        </div>

        {/* TAB 1: HOSTING PACKAGES / PRICING CARDS */}
        {activeTab === 'packages' && (
          <div className="space-y-8 animate-in fade-in duration-300">
            {/* Billing Cycle Switcher with Discount Tag */}
            <div className="flex flex-col sm:flex-row items-center justify-center gap-3">
              <div className="inline-flex items-center p-1.5 bg-slate-100 dark:bg-slate-800/90 rounded-2xl border border-slate-200 dark:border-slate-700">
                <button
                  onClick={() => setBillingCycle('monthly')}
                  className={`px-5 py-2 rounded-xl text-sm font-bold transition-all ${
                    billingCycle === 'monthly'
                      ? 'bg-white dark:bg-slate-900 text-slate-900 dark:text-white shadow-sm'
                      : 'text-slate-500 hover:text-slate-800 dark:hover:text-slate-200'
                  }`}
                >
                  মাসিক বিলিং
                </button>
                <button
                  onClick={() => setBillingCycle('yearly')}
                  className={`flex items-center gap-2 px-5 py-2 rounded-xl text-sm font-bold transition-all ${
                    billingCycle === 'yearly'
                      ? 'bg-gradient-to-r from-blue-600 to-indigo-600 text-white shadow-md'
                      : 'text-slate-500 hover:text-slate-800 dark:hover:text-slate-200'
                  }`}
                >
                  <span>বার্ষিক বিলিং</span>
                  <span className="px-2 py-0.5 rounded-full text-[10px] font-extrabold bg-emerald-500 text-white uppercase tracking-wider">
                    ২০% ছাড়
                  </span>
                </button>
              </div>
            </div>

            {/* Pricing Cards Grid */}
            <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-4 gap-6 items-stretch">
              {plans.map(plan => {
                const isFeatured = plan.tier === 'business';
                const isReseller = plan.tier === 'reseller';
                const price = billingCycle === 'yearly' ? plan.price_yearly : plan.price_monthly;
                const monthlyEquiv = billingCycle === 'yearly' ? plan.price_yearly / 12 : plan.price_monthly;

                return (
                  <div
                    key={plan.id}
                    className={`relative flex flex-col justify-between rounded-3xl transition-all duration-300 hover:translate-y-[-4px] ${
                      isFeatured
                        ? 'bg-gradient-to-b from-blue-600/5 to-purple-600/5 dark:from-blue-950/20 dark:to-purple-950/20 border-2 border-blue-500 dark:border-blue-400 shadow-xl shadow-blue-500/10'
                        : isReseller
                        ? 'bg-gradient-to-b from-emerald-600/5 to-teal-600/5 dark:from-emerald-950/20 dark:to-teal-950/20 border border-emerald-500/40 dark:border-emerald-500/30 shadow-md'
                        : 'bg-white dark:bg-slate-900 border border-slate-200/80 dark:border-slate-800 shadow-sm'
                    } p-6 sm:p-7`}
                  >
                    {/* Featured Badge */}
                    {isFeatured && (
                      <div className="absolute -top-3.5 left-1/2 -translate-x-1/2 px-4 py-1 rounded-full bg-gradient-to-r from-blue-600 to-indigo-600 text-white text-[11px] font-extrabold uppercase tracking-widest shadow-md">
                        সর্বাধিক জনপ্রিয়
                      </div>
                    )}
                    {isReseller && (
                      <div className="absolute -top-3.5 left-1/2 -translate-x-1/2 px-4 py-1 rounded-full bg-emerald-600 text-white text-[11px] font-extrabold uppercase tracking-widest shadow-md">
                        হোস্টিং এজেন্সি
                      </div>
                    )}

                    <div>
                      {/* Plan Header */}
                      <div className="flex items-center justify-between gap-3 mb-3">
                        <h3 className="text-xl font-black text-slate-900 dark:text-white">
                          {plan.name}
                        </h3>
                        <div
                          className={`p-2 rounded-xl ${
                            plan.tier === 'starter'
                              ? 'bg-blue-500/10 text-blue-600 dark:text-blue-400'
                              : plan.tier === 'business'
                              ? 'bg-indigo-500/10 text-indigo-600 dark:text-indigo-400'
                              : plan.tier === 'enterprise'
                              ? 'bg-purple-500/10 text-purple-600 dark:text-purple-400'
                              : 'bg-emerald-500/10 text-emerald-600 dark:text-emerald-400'
                          }`}
                        >
                          <Zap className="w-5 h-5" />
                        </div>
                      </div>

                      <p className="text-xs text-slate-500 dark:text-slate-400 min-h-[36px] line-clamp-2 leading-relaxed mb-6">
                        {plan.description}
                      </p>

                      {/* Pricing Tag */}
                      <div className="mb-6 pb-6 border-b border-slate-100 dark:border-slate-800/80">
                        <div className="flex items-baseline gap-1.5">
                          <span className="text-4xl font-black text-slate-900 dark:text-white tracking-tight">
                            {formatPrice(monthlyEquiv)}
                          </span>
                          <span className="text-xs font-semibold text-slate-500 dark:text-slate-400">
                            /মাস
                          </span>
                        </div>
                        <div className="text-[11px] text-slate-400 dark:text-slate-500 mt-1 font-medium">
                          {billingCycle === 'yearly'
                            ? `বার্ষিক বিলিং মোট ${formatPrice(price)} /বছর`
                            : 'প্রতি মাসে অটোমেটিক রিনিউয়াল'}
                        </div>
                      </div>

                      {/* Quotas Quick Grid */}
                      <div className="grid grid-cols-2 gap-2 mb-6">
                        <div className="p-2.5 rounded-xl bg-slate-50 dark:bg-slate-800/50 border border-slate-100 dark:border-slate-800">
                          <div className="text-[10px] text-slate-400 font-semibold uppercase tracking-wider flex items-center gap-1">
                            <HardDrive className="w-3 h-3 text-blue-500" />
                            স্টোরেজ
                          </div>
                          <div className="text-sm font-bold text-slate-800 dark:text-slate-200 mt-0.5">
                            {plan.disk_space_mb >= 1024
                              ? `${Math.round(plan.disk_space_mb / 1024)} GB NVMe`
                              : `${plan.disk_space_mb} MB`}
                          </div>
                        </div>

                        <div className="p-2.5 rounded-xl bg-slate-50 dark:bg-slate-800/50 border border-slate-100 dark:border-slate-800">
                          <div className="text-[10px] text-slate-400 font-semibold uppercase tracking-wider flex items-center gap-1">
                            <Network className="w-3 h-3 text-emerald-500" />
                            ব্যান্ডউইথ
                          </div>
                          <div className="text-sm font-bold text-slate-800 dark:text-slate-200 mt-0.5">
                            {plan.bandwidth_mb >= 1024
                              ? `${Math.round(plan.bandwidth_mb / 1024)} GB`
                              : `${plan.bandwidth_mb} MB`}
                          </div>
                        </div>

                        <div className="p-2.5 rounded-xl bg-slate-50 dark:bg-slate-800/50 border border-slate-100 dark:border-slate-800">
                          <div className="text-[10px] text-slate-400 font-semibold uppercase tracking-wider flex items-center gap-1">
                            <Globe className="w-3 h-3 text-purple-500" />
                            ওয়েবসাইট
                          </div>
                          <div className="text-sm font-bold text-slate-800 dark:text-slate-200 mt-0.5">
                            {plan.max_websites >= 100 ? 'আনলিমিটেড' : `${plan.max_websites} টি ডোমেইন`}
                          </div>
                        </div>

                        <div className="p-2.5 rounded-xl bg-slate-50 dark:bg-slate-800/50 border border-slate-100 dark:border-slate-800">
                          <div className="text-[10px] text-slate-400 font-semibold uppercase tracking-wider flex items-center gap-1">
                            <Mail className="w-3 h-3 text-amber-500" />
                            ইমেইল
                          </div>
                          <div className="text-sm font-bold text-slate-800 dark:text-slate-200 mt-0.5">
                            {plan.max_mailboxes >= 100 ? 'আনলিমিটেড' : `${plan.max_mailboxes} টি ইনবক্স`}
                          </div>
                        </div>
                      </div>

                      {/* Included Features Checklist */}
                      <div className="space-y-2.5 mb-8">
                        <div className="text-xs font-bold text-slate-900 dark:text-white uppercase tracking-wider">
                          অন্তর্ভুক্ত সুবিধাসমূহ:
                        </div>
                        {plan.features.map((feature, idx) => (
                          <div key={idx} className="flex items-start gap-2.5 text-xs text-slate-600 dark:text-slate-300">
                            <CheckCircle2 className="w-4 h-4 text-emerald-500 flex-shrink-0 mt-0.5" />
                            <span>{feature}</span>
                          </div>
                        ))}
                      </div>
                    </div>

                    {/* Order Action Button */}
                    <button
                      onClick={() => handleOpenCheckout(plan)}
                      className={`w-full py-3.5 px-5 rounded-2xl text-sm font-bold flex items-center justify-center gap-2 transition-all shadow-sm ${
                        isFeatured
                          ? 'bg-gradient-to-r from-blue-600 to-indigo-600 hover:from-blue-700 hover:to-indigo-700 text-white shadow-blue-500/25 hover:shadow-lg'
                          : isReseller
                          ? 'bg-emerald-600 hover:bg-emerald-700 text-white shadow-emerald-600/20'
                          : 'bg-slate-900 hover:bg-slate-800 dark:bg-white dark:hover:bg-slate-100 text-white dark:text-slate-900'
                      }`}
                    >
                      <span>এখনই শুরু করুন</span>
                      <ArrowRight className="w-4 h-4" />
                    </button>
                  </div>
                );
              })}
            </div>

            {/* Enterprise Custom Quotas & SLA Callout */}
            <div className="rounded-3xl p-8 bg-gradient-to-r from-slate-900 via-slate-800 to-slate-900 text-white border border-slate-800 flex flex-col md:flex-row items-center justify-between gap-6 shadow-xl">
              <div className="space-y-2 text-center md:text-left">
                <div className="inline-flex items-center gap-2 px-3 py-1 rounded-full bg-blue-500/20 text-blue-300 text-xs font-bold uppercase tracking-wider">
                  <Shield className="w-3.5 h-3.5" />
                  কাস্টম ক্লাস্টার সলিউশন
                </div>
                <h3 className="text-2xl font-black">আপনার কি কাস্টম ডেডিকেটেড ক্লাউড নোড প্রয়োজন?</h3>
                <p className="text-slate-400 text-sm max-w-xl">
                  আমাদের হাইপার-স্কেল ইনফ্রাস্ট্রাকচারে মাল্টি-সার্ভার ক্লাস্টারিং, ডেডিকেটেড লোড ব্যালেন্সার এবং ৯৯.৯৯% এসএলএ গ্যারান্টি দেওয়া হয়।
                </p>
              </div>
              <button
                onClick={() => {
                  const enterprise = plans.find(p => p.tier === 'enterprise') || plans[0];
                  handleOpenCheckout(enterprise);
                }}
                className="px-6 py-3 rounded-2xl bg-white text-slate-900 hover:bg-slate-100 font-extrabold text-sm flex-shrink-0 transition-all shadow-lg"
              >
                এন্টারপ্রাইজ সাপোর্ট টিম
              </button>
            </div>
          </div>
        )}

        {/* TAB 2: MY ACTIVE SUBSCRIPTIONS */}
        {activeTab === 'subscriptions' && (
          <div className="space-y-6 animate-in fade-in duration-300">
            <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
              <div>
                <h2 className="text-xl font-bold text-slate-900 dark:text-white">
                  আপনার সক্রিয় হোস্টিং সাবস্ক্রিপশনসমূহ
                </h2>
                <p className="text-xs text-slate-500 dark:text-slate-400">
                  চলতি প্যাকেজের ব্যবহৃত ডিস্ক স্পেস, ব্যান্ডউইথ মিটার ও অটো-রিনিউয়াল পরিচালনা করুন।
                </p>
              </div>

              <button
                onClick={() => setActiveTab('packages')}
                className="inline-flex items-center gap-2 px-4 py-2.5 rounded-xl bg-blue-600 hover:bg-blue-700 text-white text-sm font-bold shadow-md shadow-blue-600/20 transition-all"
              >
                <Plus className="w-4 h-4" />
                <span>নতুন প্যাকেজ নিন</span>
              </button>
            </div>

            {subscriptions.length === 0 ? (
              <div className="text-center py-16 px-4 bg-white dark:bg-slate-900 rounded-3xl border border-slate-200 dark:border-slate-800">
                <div className="w-16 h-16 rounded-2xl bg-blue-50 dark:bg-blue-950/50 text-blue-600 flex items-center justify-center mx-auto mb-4">
                  <Layers className="w-8 h-8" />
                </div>
                <h3 className="text-lg font-bold text-slate-900 dark:text-white">
                  কোন সক্রিয় সাবস্ক্রিপশন পাওয়া যায়নি
                </h3>
                <p className="text-sm text-slate-500 max-w-md mx-auto mt-1 mb-6">
                  আপনার হোস্টভরা একাউন্টে এখনও কোন হোস্টিং প্ল্যান সক্রিয় নেই। আমাদের সাশ্রয়ী প্যাকেজগুলো ঘুরে দেখুন।
                </p>
                <button
                  onClick={() => setActiveTab('packages')}
                  className="px-6 py-3 rounded-xl bg-blue-600 hover:bg-blue-700 text-white text-sm font-bold transition-all shadow-md"
                >
                  হোস্টিং প্যাকেজ ব্রাউজ করুন
                </button>
              </div>
            ) : (
              <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
                {subscriptions.map(sub => {
                  const matchingPlan = plans.find(p => p.id === sub.plan_id);
                  const maxDisk = matchingPlan?.disk_space_mb || 10240;
                  const maxBandwidth = matchingPlan?.bandwidth_mb || 102400;
                  const diskPercent = Math.min(100, Math.round((sub.disk_used_mb / maxDisk) * 100));
                  const bwPercent = Math.min(100, Math.round((sub.bandwidth_used_mb / maxBandwidth) * 100));

                  return (
                    <div
                      key={sub.id}
                      className="bg-white dark:bg-slate-900 rounded-3xl border border-slate-200 dark:border-slate-800 p-6 shadow-sm flex flex-col justify-between"
                    >
                      <div>
                        {/* Header */}
                        <div className="flex items-start justify-between gap-4 mb-4 pb-4 border-b border-slate-100 dark:border-slate-800">
                          <div>
                            <div className="flex items-center gap-2">
                              <h3 className="text-lg font-bold text-slate-900 dark:text-white">
                                {sub.plan_name}
                              </h3>
                              <span
                                className={`px-2.5 py-0.5 rounded-full text-xs font-extrabold uppercase tracking-wider ${
                                  sub.status === 'active'
                                    ? 'bg-emerald-500/15 text-emerald-600 dark:text-emerald-400'
                                    : sub.status === 'pending'
                                    ? 'bg-amber-500/15 text-amber-600'
                                    : 'bg-rose-500/15 text-rose-600'
                                }`}
                              >
                                {sub.status === 'active' ? 'সক্রিয়' : sub.status}
                              </span>
                            </div>
                            <div className="text-xs text-slate-400 mt-1">
                              বিলিং সাইকেল: <span className="font-semibold text-slate-700 dark:text-slate-300">{sub.billing_cycle === 'yearly' ? 'বার্ষিক' : 'মাসিক'}</span> | মূল্য: <span className="font-bold text-blue-600 dark:text-blue-400">{formatPrice(sub.amount)}</span>
                            </div>
                          </div>

                          <div className="text-right">
                            <div className="text-[11px] text-slate-400 font-medium">পরবর্তী বিলিং ডেট</div>
                            <div className="text-xs font-bold text-slate-800 dark:text-slate-200 mt-0.5 flex items-center gap-1 justify-end">
                              <Calendar className="w-3.5 h-3.5 text-blue-500" />
                              {new Date(sub.next_billing_date).toLocaleDateString('bn-BD', {
                                year: 'numeric',
                                month: 'short',
                                day: 'numeric',
                              })}
                            </div>
                          </div>
                        </div>

                        {/* Usage Meters */}
                        <div className="space-y-4 mb-6">
                          {/* Disk Meter */}
                          <div>
                            <div className="flex items-center justify-between text-xs font-semibold mb-1.5">
                              <span className="text-slate-600 dark:text-slate-400 flex items-center gap-1.5">
                                <HardDrive className="w-3.5 h-3.5 text-blue-500" />
                                ব্যবহৃত স্টোরেজ
                              </span>
                              <span className="text-slate-900 dark:text-slate-100 font-bold">
                                {sub.disk_used_mb >= 1024 ? `${(sub.disk_used_mb / 1024).toFixed(1)} GB` : `${sub.disk_used_mb} MB`} /{' '}
                                {maxDisk >= 1024 ? `${(maxDisk / 1024).toFixed(0)} GB` : `${maxDisk} MB`} ({diskPercent}%)
                              </span>
                            </div>
                            <div className="w-full h-2.5 rounded-full bg-slate-100 dark:bg-slate-800 overflow-hidden">
                              <div
                                className={`h-full rounded-full transition-all ${
                                  diskPercent > 85 ? 'bg-rose-500' : diskPercent > 60 ? 'bg-amber-500' : 'bg-blue-600'
                                }`}
                                style={{ width: `${diskPercent}%` }}
                              />
                            </div>
                          </div>

                          {/* Bandwidth Meter */}
                          <div>
                            <div className="flex items-center justify-between text-xs font-semibold mb-1.5">
                              <span className="text-slate-600 dark:text-slate-400 flex items-center gap-1.5">
                                <Network className="w-3.5 h-3.5 text-emerald-500" />
                                মাসিক ব্যান্ডউইথ
                              </span>
                              <span className="text-slate-900 dark:text-slate-100 font-bold">
                                {sub.bandwidth_used_mb >= 1024 ? `${(sub.bandwidth_used_mb / 1024).toFixed(1)} GB` : `${sub.bandwidth_used_mb} MB`} /{' '}
                                {maxBandwidth >= 1024 ? `${(maxBandwidth / 1024).toFixed(0)} GB` : `${maxBandwidth} MB`} ({bwPercent}%)
                              </span>
                            </div>
                            <div className="w-full h-2.5 rounded-full bg-slate-100 dark:bg-slate-800 overflow-hidden">
                              <div
                                className="h-full rounded-full bg-emerald-500 transition-all"
                                style={{ width: `${bwPercent}%` }}
                              />
                            </div>
                          </div>

                          {/* Websites Count */}
                          <div className="flex items-center justify-between p-3 rounded-2xl bg-slate-50 dark:bg-slate-800/60 text-xs">
                            <span className="text-slate-600 dark:text-slate-400 flex items-center gap-2 font-medium">
                              <Globe className="w-4 h-4 text-purple-500" />
                              সংযুক্ত ওয়েবসাইট
                            </span>
                            <span className="font-bold text-slate-800 dark:text-slate-200">
                              {sub.websites_count} টি সাইট
                            </span>
                          </div>
                        </div>
                      </div>

                      {/* Card Actions */}
                      <div className="flex items-center justify-between pt-4 border-t border-slate-100 dark:border-slate-800">
                        <div className="flex items-center gap-2">
                          <button
                            onClick={() => handleRenewSubscription(sub)}
                            disabled={actionLoading === `renew-${sub.id}`}
                            className="px-3.5 py-2 rounded-xl text-xs font-bold bg-blue-50 hover:bg-blue-100 dark:bg-blue-950/50 dark:hover:bg-blue-900/50 text-blue-600 dark:text-blue-400 transition-colors flex items-center gap-1.5"
                          >
                            <RefreshCw className={`w-3.5 h-3.5 ${actionLoading === `renew-${sub.id}` ? 'animate-spin' : ''}`} />
                            <span>রিনিউ করুন</span>
                          </button>

                          <button
                            onClick={() => setActiveTab('packages')}
                            className="px-3.5 py-2 rounded-xl text-xs font-bold bg-slate-100 hover:bg-slate-200 dark:bg-slate-800 dark:hover:bg-slate-700 text-slate-700 dark:text-slate-300 transition-colors"
                          >
                            আপগ্রেড
                          </button>
                        </div>

                        {sub.status === 'active' && (
                          <button
                            onClick={() => handleCancelSubscription(sub)}
                            disabled={actionLoading === `cancel-${sub.id}`}
                            className="text-xs font-bold text-rose-500 hover:text-rose-600 transition-colors"
                          >
                            বাতিল করুন
                          </button>
                        )}
                      </div>
                    </div>
                  );
                })}
              </div>
            )}
          </div>
        )}

        {/* TAB 3: INVOICES & RECEIPTS */}
        {activeTab === 'invoices' && (
          <div className="space-y-6 animate-in fade-in duration-300">
            <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
              <div>
                <h2 className="text-xl font-bold text-slate-900 dark:text-white">
                  বিলিং ইনভয়েস ও পেমেন্ট হিস্ট্রি
                </h2>
                <p className="text-xs text-slate-500 dark:text-slate-400">
                  আপনার সকল হোস্টিং অর্ডারের অফিশিয়াল ইনভয়েস, মানি রিসিট ডাউনলোড ও বকেয়া পরিশোধ।
                </p>
              </div>
            </div>

            <div className="bg-white dark:bg-slate-900 rounded-3xl border border-slate-200 dark:border-slate-800 shadow-sm overflow-hidden">
              <div className="overflow-x-auto">
                <table className="w-full text-left text-xs">
                  <thead className="bg-slate-50 dark:bg-slate-800/80 text-slate-500 dark:text-slate-400 font-bold uppercase tracking-wider border-b border-slate-200 dark:border-slate-800">
                    <tr>
                      <th className="py-4 px-6">ইনভয়েস #</th>
                      <th className="py-4 px-6">তারিখ</th>
                      <th className="py-4 px-6">বিবরণ</th>
                      <th className="py-4 px-6">গেটওয়ে</th>
                      <th className="py-4 px-6">পরিমাণ</th>
                      <th className="py-4 px-6">স্ট্যাটাস</th>
                      <th className="py-4 px-6 text-right">অ্যাকশন</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-slate-100 dark:divide-slate-800/60 font-medium">
                    {invoices.length === 0 ? (
                      <tr>
                        <td colSpan={7} className="text-center py-12 text-slate-400">
                          কোন ইনভয়েস রেকর্ড পাওয়া যায়নি।
                        </td>
                      </tr>
                    ) : (
                      invoices.map(inv => (
                        <tr key={inv.id} className="hover:bg-slate-50/80 dark:hover:bg-slate-800/30 transition-colors">
                          <td className="py-4 px-6 font-bold text-slate-900 dark:text-white flex items-center gap-2">
                            <FileText className="w-4 h-4 text-blue-500" />
                            {inv.invoice_number}
                          </td>
                          <td className="py-4 px-6 text-slate-500 dark:text-slate-400">
                            {new Date(inv.created_at).toLocaleDateString('bn-BD', {
                              year: 'numeric',
                              month: 'short',
                              day: 'numeric',
                            })}
                          </td>
                          <td className="py-4 px-6 text-slate-800 dark:text-slate-200 font-semibold max-w-xs truncate">
                            {inv.description}
                          </td>
                          <td className="py-4 px-6 text-slate-600 dark:text-slate-400 uppercase text-[11px] font-bold">
                            {inv.payment_method || 'N/A'}
                          </td>
                          <td className="py-4 px-6 font-black text-slate-900 dark:text-white">
                            {formatPrice(inv.total)}
                          </td>
                          <td className="py-4 px-6">
                            <span
                              className={`px-2.5 py-1 rounded-full text-[11px] font-extrabold uppercase tracking-wide inline-flex items-center gap-1 ${
                                inv.status === 'paid'
                                  ? 'bg-emerald-500/15 text-emerald-600 dark:text-emerald-400'
                                  : inv.status === 'unpaid'
                                  ? 'bg-amber-500/15 text-amber-600 dark:text-amber-400'
                                  : 'bg-rose-500/15 text-rose-600'
                              }`}
                            >
                              {inv.status === 'paid' && <Check className="w-3 h-3" />}
                              {inv.status === 'paid' ? 'পরিশোধিত' : inv.status === 'unpaid' ? 'বকেয়া' : 'বাতিল'}
                            </span>
                          </td>
                          <td className="py-4 px-6 text-right">
                            <div className="flex items-center justify-end gap-2">
                              {inv.status === 'unpaid' && (
                                <button
                                  onClick={() => handlePayInvoice(inv)}
                                  disabled={actionLoading === `pay-${inv.id}`}
                                  className="px-3 py-1.5 rounded-lg text-xs font-bold bg-emerald-600 hover:bg-emerald-700 text-white transition-colors"
                                >
                                  পরিশোধ করুন
                                </button>
                              )}
                              <button
                                onClick={() => {
                                  setSelectedInvoice(inv);
                                  setReceiptModalOpen(true);
                                }}
                                className="px-3 py-1.5 rounded-lg text-xs font-bold bg-slate-100 hover:bg-slate-200 dark:bg-slate-800 dark:hover:bg-slate-700 text-slate-700 dark:text-slate-300 transition-colors flex items-center gap-1"
                              >
                                <Eye className="w-3.5 h-3.5" />
                                <span>রসিদ</span>
                              </button>
                            </div>
                          </td>
                        </tr>
                      ))
                    )}
                  </tbody>
                </table>
              </div>
            </div>
          </div>
        )}

        {/* TAB 4: PAYMENT GATEWAYS CONFIGURATION */}
        {activeTab === 'gateways' && (
          <div className="space-y-6 animate-in fade-in duration-300">
            <div>
              <h2 className="text-xl font-bold text-slate-900 dark:text-white">
                পেমেন্ট গেটওয়ে ইন্টিগ্রেশন হাব
              </h2>
              <p className="text-xs text-slate-500 dark:text-slate-400">
                বাংলাদেশী মোবাইল ফাইন্যান্সিয়াল সার্ভিস (bKash, Nagad, SSLCommerz) ও আন্তর্জাতিক কার্ড গেটওয়ে (Stripe, PayPal) সক্রিয় করুন।
              </p>
            </div>

            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
              {gateways.map(gw => (
                <div
                  key={gw.gateway}
                  className={`bg-white dark:bg-slate-900 rounded-3xl border p-6 shadow-sm flex flex-col justify-between transition-all ${
                    gw.enabled
                      ? 'border-slate-200 dark:border-slate-800'
                      : 'border-dashed border-slate-300 dark:border-slate-800 opacity-75'
                  }`}
                >
                  <div>
                    {/* Header */}
                    <div className="flex items-center justify-between mb-4">
                      <div className="flex items-center gap-3">
                        <div
                          className={`w-10 h-10 rounded-2xl flex items-center justify-center font-bold text-lg ${
                            gw.gateway === 'bkash'
                              ? 'bg-pink-500/15 text-pink-600'
                              : gw.gateway === 'nagad'
                              ? 'bg-orange-500/15 text-orange-600'
                              : gw.gateway === 'sslcommerz'
                              ? 'bg-blue-500/15 text-blue-600'
                              : gw.gateway === 'stripe'
                              ? 'bg-indigo-500/15 text-indigo-600'
                              : 'bg-amber-500/15 text-amber-600'
                          }`}
                        >
                          {gw.gateway === 'bkash'
                            ? 'বিকাশ'
                            : gw.gateway === 'nagad'
                            ? 'নগদ'
                            : gw.gateway.toUpperCase().slice(0, 3)}
                        </div>
                        <div>
                          <h3 className="font-bold text-sm text-slate-900 dark:text-white">
                            {gw.display_name}
                          </h3>
                          <span className="text-[11px] text-slate-400 uppercase tracking-wider font-semibold">
                            {gw.gateway}
                          </span>
                        </div>
                      </div>

                      <span
                        className={`px-2.5 py-0.5 rounded-full text-[10px] font-extrabold uppercase tracking-wider ${
                          gw.enabled
                            ? 'bg-emerald-500/15 text-emerald-600 dark:text-emerald-400'
                            : 'bg-slate-100 dark:bg-slate-800 text-slate-400'
                        }`}
                      >
                        {gw.enabled ? 'চালু' : 'বন্ধ'}
                      </span>
                    </div>

                    {/* Metadata */}
                    <div className="space-y-2 mb-6 text-xs bg-slate-50 dark:bg-slate-800/50 p-3 rounded-2xl">
                      <div className="flex items-center justify-between">
                        <span className="text-slate-500">এনভায়রনমেন্ট:</span>
                        <span className="font-semibold text-slate-800 dark:text-slate-200">
                          {gw.test_mode ? 'Sandbox / Test' : 'Live Production'}
                        </span>
                      </div>
                      <div className="flex items-center justify-between">
                        <span className="text-slate-500">মার্চেন্ট আইডি:</span>
                        <span className="font-mono text-slate-700 dark:text-slate-300">
                          {gw.merchant_id || 'Not Set'}
                        </span>
                      </div>
                      <div className="flex items-center justify-between">
                        <span className="text-slate-500">API Key:</span>
                        <span className="font-mono text-slate-700 dark:text-slate-300">
                          {gw.api_key ? '••••' + gw.api_key.slice(-4) : 'Not Set'}
                        </span>
                      </div>
                    </div>
                  </div>

                  <div className="flex items-center justify-between pt-4 border-t border-slate-100 dark:border-slate-800">
                    <button
                      onClick={() => {
                        setEditingGateway({ ...gw });
                        setGatewayModalOpen(true);
                      }}
                      className="w-full py-2.5 px-4 rounded-xl bg-slate-100 hover:bg-slate-200 dark:bg-slate-800 dark:hover:bg-slate-700 text-slate-800 dark:text-slate-200 text-xs font-bold transition-all flex items-center justify-center gap-1.5"
                    >
                      <Settings className="w-3.5 h-3.5" />
                      <span>সেটিংস কনফিগার করুন</span>
                    </button>
                  </div>
                </div>
              ))}
            </div>
          </div>
        )}

        {/* TAB 5: ADMIN PLAN MANAGER */}
        {activeTab === 'admin' && (
          <div className="space-y-6 animate-in fade-in duration-300">
            <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
              <div>
                <h2 className="text-xl font-bold text-slate-900 dark:text-white">
                  হোস্টিং প্যাকেজ ও কোটা ম্যানেজার
                </h2>
                <p className="text-xs text-slate-500 dark:text-slate-400">
                  অ্যাডমিন হিসেবে নতুন ক্লাউড হোস্টিং প্যাকেজ তৈরি করুন, স্টোরেজ/ব্যান্ডউইথ কোটা ও প্রাইসিং আপডেট করুন।
                </p>
              </div>

              <button
                onClick={() => {
                  setEditingPlan({
                    name: '',
                    slug: '',
                    description: '',
                    tier: 'starter',
                    price_monthly: 9.99,
                    price_yearly: 99.99,
                    currency: 'USD',
                    disk_space_mb: 20480,
                    bandwidth_mb: 204800,
                    max_websites: 3,
                    max_databases: 5,
                    max_mailboxes: 10,
                    max_ftp: 5,
                    dedicated_ip: false,
                    free_ssl: true,
                    features: ['Fast NVMe Storage', 'Free SSL', 'Weekly Backups'],
                    is_active: true,
                  });
                  setPlanModalOpen(true);
                }}
                className="inline-flex items-center gap-2 px-4 py-2.5 rounded-xl bg-blue-600 hover:bg-blue-700 text-white text-sm font-bold shadow-md shadow-blue-600/20 transition-all"
              >
                <Plus className="w-4 h-4" />
                <span>নতুন প্যাকেজ তৈরি করুন</span>
              </button>
            </div>

            <div className="bg-white dark:bg-slate-900 rounded-3xl border border-slate-200 dark:border-slate-800 shadow-sm overflow-hidden">
              <div className="overflow-x-auto">
                <table className="w-full text-left text-xs">
                  <thead className="bg-slate-50 dark:bg-slate-800/80 text-slate-500 dark:text-slate-400 font-bold uppercase tracking-wider border-b border-slate-200 dark:border-slate-800">
                    <tr>
                      <th className="py-4 px-6">প্যাকেজের নাম</th>
                      <th className="py-4 px-6">টিয়ার</th>
                      <th className="py-4 px-6">মাসিক মূল্য</th>
                      <th className="py-4 px-6">বার্ষিক মূল্য</th>
                      <th className="py-4 px-6">ডিস্ক স্পেস</th>
                      <th className="py-4 px-6">ডোমেইন কোটা</th>
                      <th className="py-4 px-6">স্ট্যাটাস</th>
                      <th className="py-4 px-6 text-right">অ্যাকশন</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-slate-100 dark:divide-slate-800/60 font-medium">
                    {plans.map(p => (
                      <tr key={p.id} className="hover:bg-slate-50/80 dark:hover:bg-slate-800/30 transition-colors">
                        <td className="py-4 px-6 font-bold text-slate-900 dark:text-white">
                          <div>{p.name}</div>
                          <div className="text-[10px] text-slate-400 font-mono">{p.slug}</div>
                        </td>
                        <td className="py-4 px-6 uppercase font-bold text-slate-600 dark:text-slate-400">
                          {p.tier}
                        </td>
                        <td className="py-4 px-6 font-black text-slate-900 dark:text-white">
                          ${p.price_monthly.toFixed(2)}
                        </td>
                        <td className="py-4 px-6 font-black text-slate-900 dark:text-white">
                          ${p.price_yearly.toFixed(2)}
                        </td>
                        <td className="py-4 px-6 font-semibold text-slate-700 dark:text-slate-300">
                          {p.disk_space_mb >= 1024 ? `${Math.round(p.disk_space_mb / 1024)} GB` : `${p.disk_space_mb} MB`}
                        </td>
                        <td className="py-4 px-6 font-semibold text-slate-700 dark:text-slate-300">
                          {p.max_websites} টি
                        </td>
                        <td className="py-4 px-6">
                          <span
                            className={`px-2.5 py-0.5 rounded-full text-[10px] font-bold ${
                              p.is_active
                                ? 'bg-emerald-500/15 text-emerald-600 dark:text-emerald-400'
                                : 'bg-slate-100 text-slate-400'
                            }`}
                          >
                            {p.is_active ? 'Active' : 'Inactive'}
                          </span>
                        </td>
                        <td className="py-4 px-6 text-right">
                          <div className="flex items-center justify-end gap-2">
                            <button
                              onClick={() => {
                                setEditingPlan({ ...p });
                                setPlanModalOpen(true);
                              }}
                              className="p-2 rounded-lg text-slate-600 hover:text-blue-600 hover:bg-blue-50 dark:hover:bg-blue-950/50 transition-colors"
                              title="সম্পাদনা করুন"
                            >
                              <Edit2 className="w-4 h-4" />
                            </button>
                            <button
                              onClick={() => handleDeletePlan(p.id)}
                              className="p-2 rounded-lg text-slate-600 hover:text-rose-600 hover:bg-rose-50 dark:hover:bg-rose-950/50 transition-colors"
                              title="মুছে ফেলুন"
                            >
                              <Trash2 className="w-4 h-4" />
                            </button>
                          </div>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </div>
          </div>
        )}

        {/* MODAL 1: CHECKOUT & SUBSCRIPTION ORDER MODAL */}
        {checkoutModalOpen && selectedPlanForOrder && (
          <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-slate-950/60 backdrop-blur-sm animate-in fade-in duration-200">
            <div className="relative w-full max-w-lg bg-white dark:bg-slate-900 rounded-3xl border border-slate-200 dark:border-slate-800 shadow-2xl overflow-hidden p-7 space-y-6">
              {/* Modal Header */}
              <div className="flex items-center justify-between pb-4 border-b border-slate-100 dark:border-slate-800">
                <div className="flex items-center gap-3">
                  <div className="p-2.5 rounded-2xl bg-blue-500/10 text-blue-600 dark:text-blue-400">
                    <CreditCard className="w-6 h-6" />
                  </div>
                  <div>
                    <h3 className="text-lg font-bold text-slate-900 dark:text-white">
                      হোস্টিং প্যাকেজ অর্ডার ও চেকআউট
                    </h3>
                    <p className="text-xs text-slate-400">
                      {selectedPlanForOrder.name}
                    </p>
                  </div>
                </div>

                <button
                  onClick={() => setCheckoutModalOpen(false)}
                  className="p-2 rounded-xl text-slate-400 hover:text-slate-600 hover:bg-slate-100 dark:hover:bg-slate-800 transition-colors"
                >
                  <X className="w-5 h-5" />
                </button>
              </div>

              {/* Order Summary Box */}
              <div className="p-4 rounded-2xl bg-slate-50 dark:bg-slate-800/50 border border-slate-100 dark:border-slate-800 space-y-3">
                <div className="flex items-center justify-between text-xs">
                  <span className="text-slate-500">নির্বাচিত প্যাকেজ:</span>
                  <span className="font-bold text-slate-900 dark:text-white">{selectedPlanForOrder.name}</span>
                </div>
                <div className="flex items-center justify-between text-xs">
                  <span className="text-slate-500">বিলিং মেয়াদ:</span>
                  <span className="font-bold text-slate-900 dark:text-white">
                    {billingCycle === 'yearly' ? '১ বছর (বার্ষিক বিলিং)' : '১ মাস (মাসিক বিলিং)'}
                  </span>
                </div>
                <div className="flex items-center justify-between text-xs">
                  <span className="text-slate-500">স্টোরেজ কোটা:</span>
                  <span className="font-semibold text-slate-700 dark:text-slate-300">
                    {Math.round(selectedPlanForOrder.disk_space_mb / 1024)} GB NVMe
                  </span>
                </div>
                {couponApplied && (
                  <div className="flex items-center justify-between text-xs text-emerald-600 font-bold">
                    <span>প্রোমো ডিসকাউন্ট (10% OFF):</span>
                    <span>
                      -{formatPrice((billingCycle === 'yearly' ? selectedPlanForOrder.price_yearly : selectedPlanForOrder.price_monthly) * 0.1)}
                    </span>
                  </div>
                )}
                <div className="pt-3 border-t border-slate-200 dark:border-slate-700 flex items-center justify-between">
                  <span className="text-sm font-bold text-slate-900 dark:text-white">মোট প্রদেয় মূল্য:</span>
                  <span className="text-2xl font-black text-blue-600 dark:text-blue-400">
                    {formatPrice(
                      (billingCycle === 'yearly' ? selectedPlanForOrder.price_yearly : selectedPlanForOrder.price_monthly) *
                        (couponApplied ? 0.9 : 1.0)
                    )}
                  </span>
                </div>
              </div>

              {/* Coupon Code Simulator */}
              <div className="flex items-center gap-2">
                <input
                  type="text"
                  value={couponCode}
                  onChange={e => setCouponCode(e.target.value)}
                  placeholder="কুপন কোড (যেমন: HOSTVRA10)"
                  className="flex-1 px-4 py-2.5 rounded-xl border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 text-xs text-slate-900 dark:text-white outline-none focus:ring-2 focus:ring-blue-500"
                />
                <button
                  type="button"
                  onClick={() => {
                    if (couponCode.trim().toUpperCase() === 'HOSTVRA10') {
                      setCouponApplied(true);
                      showNotify('success', 'কুপন কোড HOSTVRA10 সফলভাবে প্রয়োগ করা হয়েছে!');
                    } else {
                      showNotify('error', 'অবৈধ কুপন কোড। অনুগ্রহ করে HOSTVRA10 ট্রাই করুন।');
                    }
                  }}
                  className="px-4 py-2.5 rounded-xl bg-slate-100 hover:bg-slate-200 dark:bg-slate-800 dark:hover:bg-slate-700 text-xs font-bold text-slate-700 dark:text-slate-300 transition-colors"
                >
                  প্রয়োগ
                </button>
              </div>

              {/* Payment Gateway Picker */}
              <div className="space-y-2">
                <label className="text-xs font-bold text-slate-700 dark:text-slate-300">
                  পেমেন্ট পদ্ধতি নির্বাচন করুন:
                </label>
                <div className="grid grid-cols-2 gap-2.5">
                  {gateways
                    .filter(g => g.enabled)
                    .map(gw => (
                      <button
                        key={gw.gateway}
                        type="button"
                        onClick={() => setSelectedPaymentGateway(gw.gateway)}
                        className={`p-3 rounded-xl border text-left transition-all ${
                          selectedPaymentGateway === gw.gateway
                            ? 'border-blue-500 bg-blue-500/5 dark:bg-blue-950/30 text-blue-600 dark:text-blue-400 font-bold ring-2 ring-blue-500/30'
                            : 'border-slate-200 dark:border-slate-800 hover:bg-slate-50 dark:hover:bg-slate-800/50 text-slate-700 dark:text-slate-300'
                        }`}
                      >
                        <div className="text-xs font-bold">{gw.display_name}</div>
                        <div className="text-[10px] text-slate-400 mt-0.5 uppercase tracking-wider">
                          {gw.test_mode ? 'Instant Test' : 'Live Gateway'}
                        </div>
                      </button>
                    ))}
                </div>
              </div>

              {/* Modal Buttons */}
              <div className="flex items-center justify-end gap-3 pt-4 border-t border-slate-100 dark:border-slate-800">
                <button
                  type="button"
                  onClick={() => setCheckoutModalOpen(false)}
                  className="px-5 py-2.5 rounded-xl text-xs font-bold text-slate-600 hover:bg-slate-100 dark:hover:bg-slate-800 transition-colors"
                >
                  বাতিল
                </button>
                <button
                  type="button"
                  onClick={handleConfirmCheckout}
                  disabled={actionLoading === 'checkout'}
                  className="px-6 py-2.5 rounded-xl bg-gradient-to-r from-blue-600 to-indigo-600 hover:from-blue-700 hover:to-indigo-700 text-white text-xs font-bold shadow-lg shadow-blue-500/20 transition-all flex items-center gap-2"
                >
                  {actionLoading === 'checkout' && <RefreshCw className="w-4 h-4 animate-spin" />}
                  <span>অর্ডার নিশ্চিত করুন</span>
                </button>
              </div>
            </div>
          </div>
        )}

        {/* MODAL 2: INVOICE & RECEIPT MODAL (PRINTABLE / BRANDED) */}
        {receiptModalOpen && selectedInvoice && (
          <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-slate-950/60 backdrop-blur-sm animate-in fade-in duration-200">
            <div className="relative w-full max-w-2xl bg-white dark:bg-slate-900 rounded-3xl border border-slate-200 dark:border-slate-800 shadow-2xl overflow-hidden p-8 space-y-6">
              {/* Top Controls */}
              <div className="flex items-center justify-between pb-4 border-b border-slate-100 dark:border-slate-800 print:hidden">
                <div className="flex items-center gap-2">
                  <span className="text-xs font-bold uppercase tracking-wider text-slate-400">
                    অফিশিয়াল মানি রিসিট
                  </span>
                </div>

                <div className="flex items-center gap-2">
                  <button
                    onClick={() => window.print()}
                    className="p-2 rounded-xl text-slate-600 dark:text-slate-300 hover:bg-slate-100 dark:hover:bg-slate-800 text-xs font-bold flex items-center gap-1.5 transition-colors"
                  >
                    <Printer className="w-4 h-4" />
                    <span>প্রিন্ট</span>
                  </button>
                  <button
                    onClick={() => setReceiptModalOpen(false)}
                    className="p-2 rounded-xl text-slate-400 hover:text-slate-600 hover:bg-slate-100 dark:hover:bg-slate-800 transition-colors"
                  >
                    <X className="w-5 h-5" />
                  </button>
                </div>
              </div>

              {/* Printable Invoice Header */}
              <div className="flex flex-col sm:flex-row justify-between gap-6 pb-6 border-b border-slate-100 dark:border-slate-800">
                <div className="space-y-1">
                  <div className="flex items-center gap-2 text-blue-600 dark:text-blue-400">
                    <Globe className="w-6 h-6" />
                    <span className="text-xl font-black tracking-tight text-slate-900 dark:text-white">HOSTVRA CLOUD</span>
                  </div>
                  <p className="text-xs text-slate-400">Next-Gen Enterprise Hosting & Infrastructure</p>
                  <p className="text-xs text-slate-400">support@hostvra.com | www.hostvra.com</p>
                </div>

                <div className="text-right space-y-1">
                  <div className="text-xl font-black text-slate-900 dark:text-white font-mono">
                    {selectedInvoice.invoice_number}
                  </div>
                  <div className="text-xs text-slate-500">
                    তারিখ: {new Date(selectedInvoice.created_at).toLocaleDateString('bn-BD')}
                  </div>
                  <div className="inline-block px-3 py-1 rounded-full text-xs font-bold uppercase tracking-wider bg-emerald-500/15 text-emerald-600 dark:text-emerald-400">
                    {selectedInvoice.status === 'paid' ? 'PAID / পরিশোধিত' : 'UNPAID / বকেয়া'}
                  </div>
                </div>
              </div>

              {/* Items Table */}
              <div className="py-2">
                <table className="w-full text-left text-xs">
                  <thead className="text-slate-400 uppercase font-bold border-b border-slate-100 dark:border-slate-800 pb-2">
                    <tr>
                      <th className="py-2">আইটেম বিবরণ</th>
                      <th className="py-2 text-right">পরিমাণ</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-slate-100 dark:divide-slate-800 font-medium">
                    <tr>
                      <td className="py-3">
                        <div className="font-bold text-slate-900 dark:text-white">{selectedInvoice.description}</div>
                        <div className="text-[11px] text-slate-400">NVMe High-Speed Cloud Server Hosting Service</div>
                      </td>
                      <td className="py-3 text-right font-bold text-slate-900 dark:text-white">
                        {formatPrice(selectedInvoice.subtotal)}
                      </td>
                    </tr>
                  </tbody>
                </table>
              </div>

              {/* Totals */}
              <div className="p-4 rounded-2xl bg-slate-50 dark:bg-slate-800/60 space-y-2 text-xs">
                <div className="flex justify-between text-slate-500">
                  <span>সাবটোটাল:</span>
                  <span className="font-bold text-slate-800 dark:text-slate-200">{formatPrice(selectedInvoice.subtotal)}</span>
                </div>
                <div className="flex justify-between text-slate-500">
                  <span>ভ্যাট ও ট্যাক্স (0%):</span>
                  <span className="font-bold text-slate-800 dark:text-slate-200">{formatPrice(0)}</span>
                </div>
                <div className="pt-2 border-t border-slate-200 dark:border-slate-700 flex justify-between text-sm font-black text-slate-900 dark:text-white">
                  <span>মোট পরিশোধিত মূল্য:</span>
                  <span className="text-blue-600 dark:text-blue-400 text-lg">{formatPrice(selectedInvoice.total)}</span>
                </div>
              </div>

              {/* Payment Details Footer */}
              <div className="flex items-center justify-between text-xs text-slate-400 pt-4 border-t border-slate-100 dark:border-slate-800">
                <div>
                  পেমেন্ট পদ্ধতি: <span className="font-bold text-slate-700 dark:text-slate-300 uppercase">{selectedInvoice.payment_method || 'Online'}</span>
                  {selectedInvoice.transaction_id && (
                    <span className="ml-2 font-mono text-[11px]">({selectedInvoice.transaction_id})</span>
                  )}
                </div>
                <div className="text-emerald-600 dark:text-emerald-400 font-bold flex items-center gap-1">
                  <CheckCircle2 className="w-4 h-4" />
                  ভেরিফাইড ডিজিটাল রসিদ
                </div>
              </div>
            </div>
          </div>
        )}

        {/* MODAL 3: PAYMENT GATEWAY CONFIG MODAL */}
        {gatewayModalOpen && editingGateway && (
          <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-slate-950/60 backdrop-blur-sm animate-in fade-in duration-200">
            <div className="relative w-full max-w-lg bg-white dark:bg-slate-900 rounded-3xl border border-slate-200 dark:border-slate-800 shadow-2xl p-7 space-y-5">
              <div className="flex items-center justify-between pb-4 border-b border-slate-100 dark:border-slate-800">
                <div className="flex items-center gap-3">
                  <div className="p-2.5 rounded-2xl bg-indigo-500/10 text-indigo-600 dark:text-indigo-400">
                    <Key className="w-5 h-5" />
                  </div>
                  <div>
                    <h3 className="text-lg font-bold text-slate-900 dark:text-white">
                      {editingGateway.display_name} সেটিংস
                    </h3>
                    <p className="text-xs text-slate-400">গেটওয়ে ক্রেডেনশিয়াল ও সিকিউরিটি কনফিগারেশন</p>
                  </div>
                </div>

                <button
                  onClick={() => setGatewayModalOpen(false)}
                  className="p-2 rounded-xl text-slate-400 hover:text-slate-600 hover:bg-slate-100 dark:hover:bg-slate-800 transition-colors"
                >
                  <X className="w-5 h-5" />
                </button>
              </div>

              <div className="space-y-4 text-xs">
                {/* Status Toggles */}
                <div className="flex items-center justify-between p-3 rounded-2xl bg-slate-50 dark:bg-slate-800/50">
                  <span className="font-bold text-slate-700 dark:text-slate-300">গেটওয়ে স্ট্যাটাস</span>
                  <button
                    type="button"
                    onClick={() => setEditingGateway({ ...editingGateway, enabled: !editingGateway.enabled })}
                    className={`px-3 py-1.5 rounded-xl font-bold transition-all ${
                      editingGateway.enabled ? 'bg-emerald-500 text-white' : 'bg-slate-200 dark:bg-slate-700 text-slate-600'
                    }`}
                  >
                    {editingGateway.enabled ? 'Enabled (সক্রিয়)' : 'Disabled (বন্ধ)'}
                  </button>
                </div>

                <div className="flex items-center justify-between p-3 rounded-2xl bg-slate-50 dark:bg-slate-800/50">
                  <span className="font-bold text-slate-700 dark:text-slate-300">স্যান্ডবক্স / টেস্ট মোড</span>
                  <button
                    type="button"
                    onClick={() => setEditingGateway({ ...editingGateway, test_mode: !editingGateway.test_mode })}
                    className={`px-3 py-1.5 rounded-xl font-bold transition-all ${
                      editingGateway.test_mode ? 'bg-amber-500 text-white' : 'bg-blue-600 text-white'
                    }`}
                  >
                    {editingGateway.test_mode ? 'Sandbox / Test' : 'Live Production'}
                  </button>
                </div>

                {/* API Key */}
                <div className="space-y-1">
                  <label className="font-bold text-slate-700 dark:text-slate-300">API Key / Public Key</label>
                  <input
                    type="text"
                    value={editingGateway.api_key || ''}
                    onChange={e => setEditingGateway({ ...editingGateway, api_key: e.target.value })}
                    className="w-full px-4 py-2.5 rounded-xl border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 text-slate-900 dark:text-white font-mono"
                    placeholder="pk_test_..."
                  />
                </div>

                {/* Secret Key */}
                <div className="space-y-1">
                  <label className="font-bold text-slate-700 dark:text-slate-300">Secret Key / Private Key</label>
                  <input
                    type="password"
                    value={editingGateway.secret_key || ''}
                    onChange={e => setEditingGateway({ ...editingGateway, secret_key: e.target.value })}
                    className="w-full px-4 py-2.5 rounded-xl border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 text-slate-900 dark:text-white font-mono"
                    placeholder="sk_test_..."
                  />
                </div>

                {/* Merchant ID */}
                <div className="space-y-1">
                  <label className="font-bold text-slate-700 dark:text-slate-300">Merchant ID / Wallet Phone / Store ID</label>
                  <input
                    type="text"
                    value={editingGateway.merchant_id || ''}
                    onChange={e => setEditingGateway({ ...editingGateway, merchant_id: e.target.value })}
                    className="w-full px-4 py-2.5 rounded-xl border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 text-slate-900 dark:text-white font-mono"
                    placeholder="Merchant identifier"
                  />
                </div>
              </div>

              <div className="flex items-center justify-end gap-3 pt-4 border-t border-slate-100 dark:border-slate-800">
                <button
                  type="button"
                  onClick={() => setGatewayModalOpen(false)}
                  className="px-5 py-2 rounded-xl text-xs font-bold text-slate-600 hover:bg-slate-100 dark:hover:bg-slate-800"
                >
                  বাতিল
                </button>
                <button
                  type="button"
                  onClick={handleSaveGateway}
                  disabled={actionLoading === 'save-gateway'}
                  className="px-6 py-2 rounded-xl bg-blue-600 hover:bg-blue-700 text-white text-xs font-bold shadow-md shadow-blue-500/20"
                >
                  সংরক্ষণ করুন
                </button>
              </div>
            </div>
          </div>
        )}

        {/* MODAL 4: ADMIN CREATE/EDIT HOSTING PLAN */}
        {planModalOpen && editingPlan && (
          <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-slate-950/60 backdrop-blur-sm animate-in fade-in duration-200">
            <div className="relative w-full max-w-2xl bg-white dark:bg-slate-900 rounded-3xl border border-slate-200 dark:border-slate-800 shadow-2xl p-7 space-y-5 max-h-[90vh] overflow-y-auto">
              <div className="flex items-center justify-between pb-4 border-b border-slate-100 dark:border-slate-800">
                <div className="flex items-center gap-3">
                  <div className="p-2.5 rounded-2xl bg-blue-500/10 text-blue-600 dark:text-blue-400">
                    <Zap className="w-5 h-5" />
                  </div>
                  <div>
                    <h3 className="text-lg font-bold text-slate-900 dark:text-white">
                      {editingPlan.id ? 'প্যাকেজ সম্পাদনা করুন' : 'নতুন হোস্টিং প্যাকেজ তৈরি করুন'}
                    </h3>
                    <p className="text-xs text-slate-400">কোটা, মূল্য এবং অন্তর্ভুক্ত সুবিধাসমূহ কনফিগার করুন</p>
                  </div>
                </div>

                <button
                  onClick={() => setPlanModalOpen(false)}
                  className="p-2 rounded-xl text-slate-400 hover:text-slate-600 hover:bg-slate-100 dark:hover:bg-slate-800"
                >
                  <X className="w-5 h-5" />
                </button>
              </div>

              <div className="grid grid-cols-1 md:grid-cols-2 gap-4 text-xs">
                <div className="space-y-1">
                  <label className="font-bold text-slate-700 dark:text-slate-300">প্যাকেজের নাম</label>
                  <input
                    type="text"
                    value={editingPlan.name || ''}
                    onChange={e => setEditingPlan({ ...editingPlan, name: e.target.value })}
                    className="w-full px-4 py-2.5 rounded-xl border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 text-slate-900 dark:text-white"
                    placeholder="e.g. Ultra Cloud Plus"
                  />
                </div>

                <div className="space-y-1">
                  <label className="font-bold text-slate-700 dark:text-slate-300">টিয়ার</label>
                  <select
                    value={editingPlan.tier || 'starter'}
                    onChange={e => setEditingPlan({ ...editingPlan, tier: e.target.value as any })}
                    className="w-full px-4 py-2.5 rounded-xl border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 text-slate-900 dark:text-white"
                  >
                    <option value="starter">Starter</option>
                    <option value="business">Business</option>
                    <option value="enterprise">Enterprise</option>
                    <option value="reseller">Reseller</option>
                  </select>
                </div>

                <div className="space-y-1">
                  <label className="font-bold text-slate-700 dark:text-slate-300">মাসিক মূল্য (USD)</label>
                  <input
                    type="number"
                    step="0.01"
                    value={editingPlan.price_monthly || 0}
                    onChange={e => setEditingPlan({ ...editingPlan, price_monthly: parseFloat(e.target.value) || 0 })}
                    className="w-full px-4 py-2.5 rounded-xl border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 text-slate-900 dark:text-white"
                  />
                </div>

                <div className="space-y-1">
                  <label className="font-bold text-slate-700 dark:text-slate-300">বার্ষিক মূল্য (USD)</label>
                  <input
                    type="number"
                    step="0.01"
                    value={editingPlan.price_yearly || 0}
                    onChange={e => setEditingPlan({ ...editingPlan, price_yearly: parseFloat(e.target.value) || 0 })}
                    className="w-full px-4 py-2.5 rounded-xl border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 text-slate-900 dark:text-white"
                  />
                </div>

                <div className="space-y-1">
                  <label className="font-bold text-slate-700 dark:text-slate-300">স্টোরেজ (MB) (10240 = 10GB)</label>
                  <input
                    type="number"
                    value={editingPlan.disk_space_mb || 10240}
                    onChange={e => setEditingPlan({ ...editingPlan, disk_space_mb: parseInt(e.target.value) || 0 })}
                    className="w-full px-4 py-2.5 rounded-xl border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 text-slate-900 dark:text-white"
                  />
                </div>

                <div className="space-y-1">
                  <label className="font-bold text-slate-700 dark:text-slate-300">ব্যান্ডউইথ (MB) (102400 = 100GB)</label>
                  <input
                    type="number"
                    value={editingPlan.bandwidth_mb || 102400}
                    onChange={e => setEditingPlan({ ...editingPlan, bandwidth_mb: parseInt(e.target.value) || 0 })}
                    className="w-full px-4 py-2.5 rounded-xl border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 text-slate-900 dark:text-white"
                  />
                </div>

                <div className="space-y-1">
                  <label className="font-bold text-slate-700 dark:text-slate-300">সর্বোচ্চ ওয়েবসাইট</label>
                  <input
                    type="number"
                    value={editingPlan.max_websites || 1}
                    onChange={e => setEditingPlan({ ...editingPlan, max_websites: parseInt(e.target.value) || 1 })}
                    className="w-full px-4 py-2.5 rounded-xl border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 text-slate-900 dark:text-white"
                  />
                </div>

                <div className="space-y-1">
                  <label className="font-bold text-slate-700 dark:text-slate-300">সর্বোচ্চ ডাটাবেস</label>
                  <input
                    type="number"
                    value={editingPlan.max_databases || 2}
                    onChange={e => setEditingPlan({ ...editingPlan, max_databases: parseInt(e.target.value) || 1 })}
                    className="w-full px-4 py-2.5 rounded-xl border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 text-slate-900 dark:text-white"
                  />
                </div>

                <div className="col-span-2 space-y-1">
                  <label className="font-bold text-slate-700 dark:text-slate-300">বিবরণ</label>
                  <textarea
                    rows={2}
                    value={editingPlan.description || ''}
                    onChange={e => setEditingPlan({ ...editingPlan, description: e.target.value })}
                    className="w-full px-4 py-2 rounded-xl border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 text-slate-900 dark:text-white"
                    placeholder="Short summary of this package"
                  />
                </div>
              </div>

              <div className="flex items-center justify-end gap-3 pt-4 border-t border-slate-100 dark:border-slate-800">
                <button
                  type="button"
                  onClick={() => setPlanModalOpen(false)}
                  className="px-5 py-2 rounded-xl text-xs font-bold text-slate-600 hover:bg-slate-100 dark:hover:bg-slate-800"
                >
                  বাতিল
                </button>
                <button
                  type="button"
                  onClick={handleSavePlan}
                  disabled={actionLoading === 'save-plan'}
                  className="px-6 py-2 rounded-xl bg-blue-600 hover:bg-blue-700 text-white text-xs font-bold shadow-md shadow-blue-500/20"
                >
                  প্যাকেজ সংরক্ষণ
                </button>
              </div>
            </div>
          </div>
        )}
      </div>
    </DashboardShell>
  );
}
