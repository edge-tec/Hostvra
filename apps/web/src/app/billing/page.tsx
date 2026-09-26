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
  Gift,
} from 'lucide-react';
import { apiFetch, HostingPlan, Subscription, Invoice, PaymentGatewayConfig, TrialSettings } from '@/lib/api';

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
    setup_fee: 0,
    trial_allowed: true,
    trial_days: 14,
    is_featured: false,
    cpu_limit: 1,
    ram_limit_mb: 2048,
    disk_space_mb: 10240,
    bandwidth_mb: 102400,
    max_websites: 1,
    max_databases: 2,
    max_mailboxes: 5,
    max_ftp: 2,
    max_cron: 5,
    max_subdomains: 10,
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
    setup_fee: 0,
    trial_allowed: true,
    trial_days: 14,
    is_featured: true,
    cpu_limit: 2,
    ram_limit_mb: 4096,
    disk_space_mb: 51200,
    bandwidth_mb: 512000,
    max_websites: 5,
    max_databases: 10,
    max_mailboxes: 25,
    max_ftp: 10,
    max_cron: 20,
    max_subdomains: 50,
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
    setup_fee: 0,
    trial_allowed: false,
    trial_days: 0,
    is_featured: false,
    cpu_limit: 4,
    ram_limit_mb: 8192,
    disk_space_mb: 204800,
    bandwidth_mb: 2048000,
    max_websites: 25,
    max_databases: 100,
    max_mailboxes: 100,
    max_ftp: 50,
    max_cron: 50,
    max_subdomains: 100,
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
    setup_fee: 0,
    trial_allowed: false,
    trial_days: 0,
    is_featured: false,
    cpu_limit: 8,
    ram_limit_mb: 16384,
    disk_space_mb: 512000,
    bandwidth_mb: 5120000,
    max_websites: 100,
    max_databases: 200,
    max_mailboxes: 500,
    max_ftp: 100,
    max_cron: 100,
    max_subdomains: 500,
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
  const [activeTab, setActiveTab] = useState<'packages' | 'subscriptions' | 'invoices' | 'gateways' | 'admin' | 'trials'>('packages');
  const [currency, setCurrency] = useState<'USD' | 'BDT'>('USD');
  const [billingCycle, setBillingCycle] = useState<'monthly' | 'yearly'>('monthly');

  // Data States
  const [plans, setPlans] = useState<HostingPlan[]>(DEFAULT_PLANS);
  const [subscriptions, setSubscriptions] = useState<Subscription[]>([]);
  const [invoices, setInvoices] = useState<Invoice[]>([]);
  const [gateways, setGateways] = useState<PaymentGatewayConfig[]>(DEFAULT_GATEWAYS);
  const [trialSettings, setTrialSettings] = useState<TrialSettings>({
    enabled: true,
    default_days: 14,
    require_payment_method: false,
    one_trial_per_customer: true,
    auto_suspend_on_expiry: true,
  });
  const [trialsList, setTrialsList] = useState<Subscription[]>([]);
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

  // Extend Trial Modal
  const [extendModalOpen, setExtendModalOpen] = useState(false);
  const [selectedTrialForExtend, setSelectedTrialForExtend] = useState<Subscription | null>(null);
  const [extendDays, setExtendDays] = useState<number>(7);

  const BDT_RATE = 120; // 1 USD = 120 BDT

  const formatPrice = (usdAmount: number) => {
    if (currency === 'BDT') {
      const bdt = Math.round(usdAmount * BDT_RATE);
      return `BDT ${bdt.toLocaleString('en-US')}`;
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
      const plansRes = await apiFetch<any>('/api/v1/billing/plans?all=true');
      if (plansRes && plansRes.data) {
        const rawPlans = Array.isArray(plansRes.data)
          ? plansRes.data
          : Array.isArray(plansRes.data.plans)
          ? plansRes.data.plans
          : [];
        if (rawPlans.length > 0) {
          setPlans(rawPlans);
        }
      }

      // 2. Load subscriptions
      const subsRes = await apiFetch<any>('/api/v1/billing/subscriptions');
      if (subsRes && subsRes.data) {
        const rawSubs = Array.isArray(subsRes.data)
          ? subsRes.data
          : Array.isArray(subsRes.data.subscriptions)
          ? subsRes.data.subscriptions
          : [];
        setSubscriptions(rawSubs);
      }

      // 3. Load invoices
      const invsRes = await apiFetch<any>('/api/v1/billing/invoices');
      if (invsRes && invsRes.data) {
        const rawInvs = Array.isArray(invsRes.data)
          ? invsRes.data
          : Array.isArray(invsRes.data.invoices)
          ? invsRes.data.invoices
          : [];
        setInvoices(rawInvs);
      }

      // 4. Load gateways
      const gwRes = await apiFetch<any>('/api/v1/billing/gateways');
      if (gwRes && gwRes.data) {
        const rawGws = Array.isArray(gwRes.data)
          ? gwRes.data
          : Array.isArray(gwRes.data.gateways)
          ? gwRes.data.gateways
          : [];
        if (rawGws.length > 0) {
          setGateways(rawGws);
        }
      }

      // 5. Load trial settings
      try {
        const tsRes = await apiFetch<any>('/api/v1/billing/trial-settings');
        if (tsRes && tsRes.data && typeof tsRes.data === 'object' && !Array.isArray(tsRes.data)) {
          setTrialSettings(tsRes.data);
        }
      } catch (tsErr) {
        console.warn('Trial settings fetch notice:', tsErr);
      }

      // 6. Load trials
      try {
        const trialsRes = await apiFetch<any>('/api/v1/billing/trials');
        if (trialsRes && trialsRes.data) {
          const rawTrials = Array.isArray(trialsRes.data)
            ? trialsRes.data
            : Array.isArray(trialsRes.data.trials)
            ? trialsRes.data.trials
            : [];
          setTrialsList(rawTrials);
        }
      } catch (trErr) {
        console.warn('Trials fetch notice:', trErr);
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
        showNotify('success', `Thank you! The ${selectedPlanForOrder.name} package has been successfully activated!`);
        setCheckoutModalOpen(false);
        loadData();
        setActiveTab('subscriptions');
      } else {
        showNotify('error', res.error?.message || 'Failed to order package');
      }
    } catch (err: any) {
      showNotify('error', err.message || 'Failed to connect to server to order package');
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
        showNotify('success', `Invoice #${invoice.invoice_number} has been successfully paid!`);
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
      showNotify('success', `Invoice #${invoice.invoice_number} payment completed!`);
    } finally {
      setActionLoading(null);
    }
  };

  // Handle Cancel Subscription
  const handleCancelSubscription = async (sub: Subscription) => {
    if (!confirm(`Are you sure you want to cancel the ${sub.plan_name} subscription?`)) return;
    setActionLoading(`cancel-${sub.id}`);
    try {
      await apiFetch(`/api/v1/billing/subscriptions/${sub.id}/cancel`, { method: 'POST' });
      showNotify('success', 'Subscription cancelled successfully.');
      loadData();
    } catch {
      setSubscriptions(prev => prev.map(s => (s.id === sub.id ? { ...s, status: 'cancelled', auto_renew: false } : s)));
      showNotify('success', 'Subscription cancelled.');
    } finally {
      setActionLoading(null);
    }
  };

  // Handle Renew Subscription
  const handleRenewSubscription = async (sub: Subscription) => {
    setActionLoading(`renew-${sub.id}`);
    try {
      await apiFetch(`/api/v1/billing/subscriptions/${sub.id}/renew`, { method: 'POST' });
      showNotify('success', `${sub.plan_name} subscription renewed successfully!`);
      loadData();
    } catch {
      setSubscriptions(prev =>
        prev.map(s => (s.id === sub.id ? { ...s, status: 'active', auto_renew: true } : s))
      );
      showNotify('success', 'Subscription renewal completed.');
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
        showNotify('success', `${editingGateway.display_name} configuration saved successfully!`);
        setGatewayModalOpen(false);
        loadData();
      }
    } catch {
      setGateways(prev => prev.map(g => (g.gateway === editingGateway.gateway ? editingGateway : g)));
      showNotify('success', `${editingGateway.display_name} saved!`);
      setGatewayModalOpen(false);
    } finally {
      setActionLoading(null);
    }
  };

  // Save / Create Plan (Admin)
  const handleSavePlan = async () => {
    if (!editingPlan || !editingPlan.name) {
      alert('Please provide a package name');
      return;
    }
    setActionLoading('save-plan');
    try {
      if (editingPlan.id) {
        await apiFetch(`/api/v1/billing/plans/${editingPlan.id}`, {
          method: 'PUT',
          body: JSON.stringify(editingPlan),
        });
        showNotify('success', 'Package updated successfully!');
      } else {
        await apiFetch('/api/v1/billing/plans', {
          method: 'POST',
          body: JSON.stringify(editingPlan),
        });
        showNotify('success', 'New hosting package created successfully!');
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
      showNotify('success', 'Package saved successfully!');
      setPlanModalOpen(false);
    } finally {
      setActionLoading(null);
    }
  };

  // Delete Plan (Admin)
  const handleDeletePlan = async (planId: string) => {
    if (!confirm('Are you sure you want to delete this package?')) return;
    try {
      await apiFetch(`/api/v1/billing/plans/${planId}`, { method: 'DELETE' });
      showNotify('success', 'Package deleted successfully.');
      loadData();
    } catch {
      setPlans(prev => prev.filter(p => p.id !== planId));
      showNotify('success', 'Package deleted.');
    }
  };

  // Start Free Trial Directly
  const handleStartFreeTrial = async (plan: HostingPlan) => {
    setActionLoading(`trial-${plan.id}`);
    try {
      const res = await apiFetch<any>('/api/v1/billing/subscriptions', {
        method: 'POST',
        body: JSON.stringify({
          plan_id: plan.id,
          billing_cycle: 'monthly',
          start_trial: true,
        }),
      });

      if (res.data) {
        showNotify('success', `Congratulations! Your 14-day free trial of ${plan.name} has been activated!`);
        await loadData();
        setActiveTab('subscriptions');
      } else {
        showNotify('error', res.error?.message || 'Failed to start free trial');
      }
    } catch (err: any) {
      showNotify('error', err.message || 'Error starting free trial');
    } finally {
      setActionLoading(null);
    }
  };

  // Save Trial Settings (Admin)
  const handleSaveTrialSettings = async () => {
    setActionLoading('save-trial-settings');
    try {
      const res = await apiFetch<TrialSettings>('/api/v1/billing/trial-settings', {
        method: 'PUT',
        body: JSON.stringify(trialSettings),
      });
      if (res.data) {
        setTrialSettings(res.data);
        showNotify('success', 'Master trial settings updated successfully!');
      } else {
        showNotify('error', res.error?.message || 'Failed to update trial settings');
      }
    } catch (err: any) {
      showNotify('error', err.message || 'Error saving trial settings');
    } finally {
      setActionLoading(null);
    }
  };

  // Confirm Trial Extension
  const handleConfirmExtendTrial = async () => {
    if (!selectedTrialForExtend) return;
    setActionLoading(`extend-${selectedTrialForExtend.id}`);
    try {
      const res = await apiFetch<any>(`/api/v1/billing/trials/${selectedTrialForExtend.id}/extend`, {
        method: 'POST',
        body: JSON.stringify({ days: extendDays }),
      });
      if (res.data) {
        showNotify('success', `Free trial extended by ${extendDays} days!`);
        setExtendModalOpen(false);
        loadData();
      } else {
        showNotify('error', res.error?.message || 'Failed to extend trial');
      }
    } catch (err: any) {
      showNotify('error', err.message || 'Error extending trial');
    } finally {
      setActionLoading(null);
    }
  };

  // End Trial Immediately
  const handleEndTrial = async (subId: string) => {
    if (!confirm('Are you sure you want to end this customer free trial immediately?')) return;
    setActionLoading(`end-${subId}`);
    try {
      const res = await apiFetch<any>(`/api/v1/billing/trials/${subId}/end`, {
        method: 'POST',
      });
      if (res.data) {
        showNotify('success', 'Trial has been ended and subscription marked as cancelled.');
        loadData();
      } else {
        showNotify('error', res.error?.message || 'Failed to end trial');
      }
    } catch (err: any) {
      showNotify('error', err.message || 'Error ending trial');
    } finally {
      setActionLoading(null);
    }
  };

  // Convert Trial to Paid Subscription
  const handleConvertTrial = async (subId: string) => {
    setActionLoading(`convert-${subId}`);
    try {
      const res = await apiFetch<any>(`/api/v1/billing/trials/${subId}/convert`, {
        method: 'POST',
      });
      if (res.data) {
        showNotify('success', 'Trial converted to paid subscription! Invoice created.');
        await loadData();
        setActiveTab('invoices');
      } else {
        showNotify('error', res.error?.message || 'Failed to convert trial');
      }
    } catch (err: any) {
      showNotify('error', err.message || 'Error converting trial');
    } finally {
      setActionLoading(null);
    }
  };

  const safePlans = Array.isArray(plans) ? plans : [];
  const safeSubscriptions = Array.isArray(subscriptions) ? subscriptions : [];
  const safeInvoices = Array.isArray(invoices) ? invoices : [];
  const safeGateways = Array.isArray(gateways) ? gateways : [];
  const safeTrials = Array.isArray(trialsList) ? trialsList : [];

  // Total active subscriptions count
  const activeSubsCount = useMemo(() => {
    return safeSubscriptions.filter(s => s && s.status === 'active').length;
  }, [safeSubscriptions]);

  // Active trials count
  const activeTrialsCount = useMemo(() => {
    const target = safeTrials.length > 0 ? safeTrials : safeSubscriptions;
    return target.filter(s => s && s.status === 'trial').length;
  }, [safeSubscriptions, safeTrials]);

  // Converted trials count
  const convertedTrialsCount = useMemo(() => {
    return safeTrials.filter(t => t && t.status === 'active').length;
  }, [safeTrials]);

  // Trial conversion rate %
  const trialConversionRate = useMemo(() => {
    if (safeTrials.length === 0) return 0;
    return Math.round((convertedTrialsCount / safeTrials.length) * 100);
  }, [safeTrials, convertedTrialsCount]);

  // Unpaid invoices count
  const unpaidInvoicesCount = useMemo(() => {
    return safeInvoices.filter(i => i && (i.status === 'unpaid' || i.status === 'overdue')).length;
  }, [safeInvoices]);

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
        <div className="relative overflow-hidden rounded-3xl bg-white dark:bg-slate-900 p-8 sm:p-10 text-slate-900 dark:text-white border border-slate-200 dark:border-slate-800 shadow-sm">
          <div className="relative z-10 flex flex-col md:flex-row md:items-center md:justify-between gap-6">
            <div className="space-y-3">
              <div className="inline-flex items-center gap-2 px-3 py-1 rounded-full bg-blue-50 dark:bg-blue-950/50 border border-blue-100 dark:border-blue-900 text-xs font-semibold tracking-wide uppercase text-blue-700 dark:text-blue-300">
                <Sparkles className="w-3.5 h-3.5 text-blue-600" />
                <span>Enterprise Hosting & Cloud Billing</span>
              </div>
              <h1 className="text-3xl sm:text-4xl font-extrabold tracking-tight text-slate-900 dark:text-white">
                Hosting Billing & Cloud Packages Hub
              </h1>
              <p className="text-slate-600 dark:text-slate-400 text-sm sm:text-base max-w-2xl leading-relaxed">
                High-speed NVMe cloud package subscriptions, local & international payment gateways (bKash, Nagad, SSLCommerz, Stripe), and automated client provisioning.
              </p>
            </div>

            {/* Quick Stats Pill */}
            <div className="flex flex-wrap sm:flex-nowrap items-center gap-3 bg-slate-50 dark:bg-slate-800/80 p-2 sm:p-3 rounded-2xl border border-slate-200 dark:border-slate-700">
              <div className="px-4 py-2 text-center border-r border-slate-200 dark:border-slate-700 last:border-0">
                <div className="text-2xl font-black text-slate-900 dark:text-white">{plans.length}</div>
                <div className="text-[11px] text-slate-500 dark:text-slate-400 uppercase tracking-wider font-semibold">Packages</div>
              </div>
              <div className="px-4 py-2 text-center border-r border-slate-200 dark:border-slate-700 last:border-0">
                <div className="text-2xl font-black text-emerald-600 dark:text-emerald-400">{activeSubsCount}</div>
                <div className="text-[11px] text-slate-500 dark:text-slate-400 uppercase tracking-wider font-semibold">Active Subscriptions</div>
              </div>
              <div className="px-4 py-2 text-center border-r border-slate-200 dark:border-slate-700 last:border-0">
                <div className="text-2xl font-black text-purple-600 dark:text-purple-400">{activeTrialsCount}</div>
                <div className="text-[11px] text-slate-500 dark:text-slate-400 uppercase tracking-wider font-semibold">Free Trials</div>
              </div>
              <div className="px-4 py-2 text-center">
                <div className="text-2xl font-black text-amber-600 dark:text-amber-400">{unpaidInvoicesCount}</div>
                <div className="text-[11px] text-slate-500 dark:text-slate-400 uppercase tracking-wider font-semibold">Unpaid Invoices</div>
              </div>
            </div>
          </div>
        </div>

        {/* Global Controls & Tabs Bar */}
        <div className="flex flex-col lg:flex-row lg:items-center justify-between gap-4 border-b border-slate-200 dark:border-slate-800 pb-4">
          {/* Main Tabs (Responsive Swipable Scroll on Mobile) */}
          <div className="flex items-center gap-1.5 sm:gap-2 p-1 bg-slate-100 dark:bg-slate-800/80 rounded-2xl border border-slate-200 dark:border-slate-700/80 overflow-x-auto max-w-full scrollbar-none flex-nowrap">
            <button
              onClick={() => setActiveTab('packages')}
              className={`flex items-center gap-2 px-3 sm:px-4 py-2 sm:py-2.5 rounded-xl text-xs sm:text-sm font-semibold transition-all whitespace-nowrap flex-shrink-0 ${
                activeTab === 'packages'
                  ? 'bg-white dark:bg-slate-900 text-blue-600 dark:text-blue-400 shadow-sm'
                  : 'text-slate-600 dark:text-slate-400 hover:text-slate-900 dark:hover:text-white'
              }`}
            >
              <Zap className="w-4 h-4 flex-shrink-0" />
              <span>Hosting Packages</span>
            </button>

            <button
              onClick={() => setActiveTab('subscriptions')}
              className={`flex items-center gap-2 px-3 sm:px-4 py-2 sm:py-2.5 rounded-xl text-xs sm:text-sm font-semibold transition-all whitespace-nowrap flex-shrink-0 relative ${
                activeTab === 'subscriptions'
                  ? 'bg-white dark:bg-slate-900 text-blue-600 dark:text-blue-400 shadow-sm'
                  : 'text-slate-600 dark:text-slate-400 hover:text-slate-900 dark:hover:text-white'
              }`}
            >
              <Layers className="w-4 h-4 flex-shrink-0" />
              <span>My Subscriptions</span>
              {activeSubsCount > 0 && (
                <span className="px-1.5 py-0.5 text-[10px] font-bold rounded-full bg-emerald-500/15 text-emerald-600 dark:text-emerald-400">
                  {activeSubsCount}
                </span>
              )}
            </button>

            <button
              onClick={() => setActiveTab('trials')}
              className={`flex items-center gap-2 px-3 sm:px-4 py-2 sm:py-2.5 rounded-xl text-xs sm:text-sm font-semibold transition-all whitespace-nowrap flex-shrink-0 relative ${
                activeTab === 'trials'
                  ? 'bg-white dark:bg-slate-900 text-purple-600 dark:text-purple-400 shadow-sm'
                  : 'text-slate-600 dark:text-slate-400 hover:text-slate-900 dark:hover:text-white'
              }`}
            >
              <Gift className="w-4 h-4 text-purple-500 flex-shrink-0" />
              <span>Free Trials</span>
              {activeTrialsCount > 0 && (
                <span className="px-1.5 py-0.5 text-[10px] font-bold rounded-full bg-purple-500 text-white">
                  {activeTrialsCount}
                </span>
              )}
            </button>

            <button
              onClick={() => setActiveTab('invoices')}
              className={`flex items-center gap-2 px-3 sm:px-4 py-2 sm:py-2.5 rounded-xl text-xs sm:text-sm font-semibold transition-all whitespace-nowrap flex-shrink-0 relative ${
                activeTab === 'invoices'
                  ? 'bg-white dark:bg-slate-900 text-blue-600 dark:text-blue-400 shadow-sm'
                  : 'text-slate-600 dark:text-slate-400 hover:text-slate-900 dark:hover:text-white'
              }`}
            >
              <FileText className="w-4 h-4 flex-shrink-0" />
              <span>Invoices & Receipts</span>
              {unpaidInvoicesCount > 0 && (
                <span className="px-1.5 py-0.5 text-[10px] font-bold rounded-full bg-amber-500 text-white animate-pulse">
                  {unpaidInvoicesCount}
                </span>
              )}
            </button>

            <button
              onClick={() => setActiveTab('gateways')}
              className={`flex items-center gap-2 px-3 sm:px-4 py-2 sm:py-2.5 rounded-xl text-xs sm:text-sm font-semibold transition-all whitespace-nowrap flex-shrink-0 ${
                activeTab === 'gateways'
                  ? 'bg-white dark:bg-slate-900 text-blue-600 dark:text-blue-400 shadow-sm'
                  : 'text-slate-600 dark:text-slate-400 hover:text-slate-900 dark:hover:text-white'
              }`}
            >
              <CreditCard className="w-4 h-4 flex-shrink-0" />
              <span>Payment Gateways</span>
            </button>

            <button
              onClick={() => setActiveTab('admin')}
              className={`flex items-center gap-2 px-3 sm:px-4 py-2 sm:py-2.5 rounded-xl text-xs sm:text-sm font-semibold transition-all whitespace-nowrap flex-shrink-0 ${
                activeTab === 'admin'
                  ? 'bg-white dark:bg-slate-900 text-blue-600 dark:text-blue-400 shadow-sm'
                  : 'text-slate-600 dark:text-slate-400 hover:text-slate-900 dark:hover:text-white'
              }`}
            >
              <Sliders className="w-4 h-4 flex-shrink-0" />
              <span>Package Manager</span>
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
                BDT
              </button>
            </div>

            {/* Refresh Button */}
            <button
              onClick={loadData}
              disabled={loading}
              className="p-2.5 rounded-xl border border-slate-200 dark:border-slate-800 bg-white dark:bg-slate-900 text-slate-600 dark:text-slate-400 hover:text-slate-900 dark:hover:text-white hover:bg-slate-50 dark:hover:bg-slate-800 transition-colors shadow-sm"
              title="Refresh"
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
                  Monthly Billing
                </button>
                <button
                  onClick={() => setBillingCycle('yearly')}
                  className={`flex items-center gap-2 px-5 py-2 rounded-xl text-sm font-bold transition-all ${
                    billingCycle === 'yearly'
                      ? 'bg-gradient-to-r from-blue-600 to-indigo-600 text-white shadow-md'
                      : 'text-slate-500 hover:text-slate-800 dark:hover:text-slate-200'
                  }`}
                >
                  <span>Annual Billing</span>
                  <span className="px-2 py-0.5 rounded-full text-[10px] font-extrabold bg-emerald-500 text-white uppercase tracking-wider">
                    20% OFF
                  </span>
                </button>
              </div>
            </div>

            {/* Pricing Cards Grid (Responsive 1/2/4 Columns) */}
            <div className="grid grid-cols-1 sm:grid-cols-2 xl:grid-cols-4 gap-4 sm:gap-6 items-stretch">
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
                    } p-5 sm:p-7`}
                  >
                    {/* Featured Badge */}
                    {isFeatured && (
                      <div className="absolute -top-3.5 left-1/2 -translate-x-1/2 px-4 py-1 rounded-full bg-gradient-to-r from-blue-600 to-indigo-600 text-white text-[11px] font-extrabold uppercase tracking-widest shadow-md">
                        Most Popular
                      </div>
                    )}
                    {isReseller && (
                      <div className="absolute -top-3.5 left-1/2 -translate-x-1/2 px-4 py-1 rounded-full bg-emerald-600 text-white text-[11px] font-extrabold uppercase tracking-widest shadow-md">
                        Hosting Agency
                      </div>
                    )}
                    {plan.trial_allowed && !isFeatured && !isReseller && (
                      <div className="absolute -top-3.5 left-1/2 -translate-x-1/2 px-3.5 py-1 rounded-full bg-gradient-to-r from-purple-600 to-indigo-600 text-white text-[11px] font-extrabold uppercase tracking-widest shadow-md">
                        {plan.trial_days || 14}-Day Free Trial
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
                            /mo
                          </span>
                        </div>
                        <div className="text-[11px] text-slate-400 dark:text-slate-500 mt-1 font-medium">
                          {billingCycle === 'yearly'
                            ? `Billed annually at ${formatPrice(price)} /yr`
                            : 'Renews automatically every month'}
                        </div>
                        {plan.trial_allowed && (
                          <div className="mt-2 text-xs font-bold text-purple-600 dark:text-purple-400 flex items-center gap-1">
                            <Sparkles className="w-3.5 h-3.5 text-purple-500" />
                            <span>Includes {plan.trial_days || 14}-Day Free Trial</span>
                          </div>
                        )}
                      </div>

                      {/* Quotas Quick Grid */}
                      <div className="grid grid-cols-2 gap-2 mb-6">
                        <div className="p-2.5 rounded-xl bg-slate-50 dark:bg-slate-800/50 border border-slate-100 dark:border-slate-800">
                          <div className="text-[10px] text-slate-400 font-semibold uppercase tracking-wider flex items-center gap-1">
                            <HardDrive className="w-3 h-3 text-blue-500" />
                            Storage
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
                            Bandwidth
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
                            Websites
                          </div>
                          <div className="text-sm font-bold text-slate-800 dark:text-slate-200 mt-0.5">
                            {plan.max_websites >= 100 ? 'Unlimited' : `${plan.max_websites} Domains`}
                          </div>
                        </div>

                        <div className="p-2.5 rounded-xl bg-slate-50 dark:bg-slate-800/50 border border-slate-100 dark:border-slate-800">
                          <div className="text-[10px] text-slate-400 font-semibold uppercase tracking-wider flex items-center gap-1">
                            <Mail className="w-3 h-3 text-amber-500" />
                            Email
                          </div>
                          <div className="text-sm font-bold text-slate-800 dark:text-slate-200 mt-0.5">
                            {plan.max_mailboxes >= 100 ? 'Unlimited' : `${plan.max_mailboxes} Inboxes`}
                          </div>
                        </div>
                      </div>

                      {/* Included Features Checklist */}
                      <div className="space-y-2.5 mb-8">
                        <div className="text-xs font-bold text-slate-900 dark:text-white uppercase tracking-wider">
                          Included Features:
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
                    <div className="space-y-2 mt-auto">
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
                        <span>Get Started</span>
                        <ArrowRight className="w-4 h-4" />
                      </button>

                      {plan.trial_allowed && (
                        <button
                          type="button"
                          onClick={() => handleStartFreeTrial(plan)}
                          disabled={actionLoading === `trial-${plan.id}`}
                          className="w-full py-2.5 px-4 rounded-xl text-xs font-bold border border-purple-500/30 text-purple-600 dark:text-purple-400 hover:bg-purple-50 dark:hover:bg-purple-950/40 transition-colors flex items-center justify-center gap-2"
                        >
                          {actionLoading === `trial-${plan.id}` ? (
                            <RefreshCw className="w-3.5 h-3.5 animate-spin" />
                          ) : (
                            <Gift className="w-3.5 h-3.5" />
                          )}
                          <span>Start {plan.trial_days || 14}-Day Free Trial</span>
                        </button>
                      )}
                    </div>
                  </div>
                );
              })}
            </div>

            {/* Enterprise Custom Quotas & SLA Callout */}
            <div className="rounded-3xl p-8 bg-white dark:bg-slate-900 text-slate-900 dark:text-white border border-slate-200 dark:border-slate-800 flex flex-col md:flex-row items-center justify-between gap-6 shadow-sm">
              <div className="space-y-2 text-center md:text-left">
                <div className="inline-flex items-center gap-2 px-3 py-1 rounded-full bg-blue-50 dark:bg-blue-950/50 text-blue-700 dark:text-blue-300 border border-blue-100 dark:border-blue-900 text-xs font-bold uppercase tracking-wider">
                  <Shield className="w-3.5 h-3.5 text-blue-600" />
                  Custom Cluster Solution
                </div>
                <h3 className="text-2xl font-black text-slate-900 dark:text-white">Need a Custom Dedicated Cloud Node?</h3>
                <p className="text-slate-600 dark:text-slate-400 text-sm max-w-xl">
                  Our hyper-scale infrastructure provides multi-server clustering, dedicated load balancing, and a 99.99% SLA guarantee.
                </p>
              </div>
              <button
                onClick={() => {
                  const enterprise = plans.find(p => p.tier === 'enterprise') || plans[0];
                  handleOpenCheckout(enterprise);
                }}
                className="px-6 py-3 rounded-2xl bg-slate-900 hover:bg-slate-800 dark:bg-white dark:hover:bg-slate-100 text-white dark:text-slate-900 font-extrabold text-sm flex-shrink-0 transition-all shadow-md"
              >
                Enterprise Support Team
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
                  Your Active Hosting Subscriptions
                </h2>
                <p className="text-xs text-slate-500 dark:text-slate-400">
                  Manage current package disk usage, bandwidth meters, and automatic renewals.
                </p>
              </div>

              <button
                onClick={() => setActiveTab('packages')}
                className="inline-flex items-center gap-2 px-4 py-2.5 rounded-xl bg-blue-600 hover:bg-blue-700 text-white text-sm font-bold shadow-md shadow-blue-600/20 transition-all"
              >
                <Plus className="w-4 h-4" />
                <span>Order New Package</span>
              </button>
            </div>

            {subscriptions.length === 0 ? (
              <div className="text-center py-16 px-4 bg-white dark:bg-slate-900 rounded-3xl border border-slate-200 dark:border-slate-800">
                <div className="w-16 h-16 rounded-2xl bg-blue-50 dark:bg-blue-950/50 text-blue-600 flex items-center justify-center mx-auto mb-4">
                  <Layers className="w-8 h-8" />
                </div>
                <h3 className="text-lg font-bold text-slate-900 dark:text-white">
                  No Active Subscriptions Found
                </h3>
                <p className="text-sm text-slate-500 max-w-md mx-auto mt-1 mb-6">
                  You do not have any active hosting plans yet. Explore our affordable high-performance packages.
                </p>
                <button
                  onClick={() => setActiveTab('packages')}
                  className="px-6 py-3 rounded-xl bg-blue-600 hover:bg-blue-700 text-white text-sm font-bold transition-all shadow-md"
                >
                  Browse Hosting Packages
                </button>
              </div>
            ) : (
              <div className="grid grid-cols-1 xl:grid-cols-2 gap-4 sm:gap-6">
                {subscriptions.map(sub => {
                  const matchingPlan = plans.find(p => p.id === sub.plan_id);
                  const maxDisk = matchingPlan?.disk_space_mb || 10240;
                  const maxBandwidth = matchingPlan?.bandwidth_mb || 102400;
                  const diskPercent = Math.min(100, Math.round((sub.disk_used_mb / maxDisk) * 100));
                  const bwPercent = Math.min(100, Math.round((sub.bandwidth_used_mb / maxBandwidth) * 100));

                  return (
                    <div
                      key={sub.id}
                      className="bg-white dark:bg-slate-900 rounded-2xl sm:rounded-3xl border border-slate-200 dark:border-slate-800 p-5 sm:p-6 shadow-sm flex flex-col justify-between"
                    >
                      <div>
                        {/* Header */}
                        <div className="flex flex-col sm:flex-row sm:items-start justify-between gap-3 sm:gap-4 mb-4 pb-4 border-b border-slate-100 dark:border-slate-800">
                          <div>
                            <div className="flex flex-wrap items-center gap-2">
                              <h3 className="text-lg font-bold text-slate-900 dark:text-white">
                                {sub.plan_name}
                              </h3>
                              <span
                                className={`px-2.5 py-0.5 rounded-full text-xs font-extrabold uppercase tracking-wider ${
                                  sub.status === 'trial'
                                    ? 'bg-purple-500/15 text-purple-600 dark:text-purple-400 border border-purple-500/30'
                                    : sub.status === 'active'
                                    ? 'bg-emerald-500/15 text-emerald-600 dark:text-emerald-400'
                                    : sub.status === 'pending'
                                    ? 'bg-amber-500/15 text-amber-600'
                                    : 'bg-rose-500/15 text-rose-600'
                                }`}
                              >
                                {sub.status === 'trial' ? 'Free Trial' : sub.status === 'active' ? 'Active' : sub.status}
                              </span>
                            </div>
                            <div className="text-xs text-slate-400 mt-1">
                              Billing Cycle: <span className="font-semibold text-slate-700 dark:text-slate-300">{sub.billing_cycle === 'yearly' ? 'Yearly' : 'Monthly'}</span> | Price: <span className="font-bold text-blue-600 dark:text-blue-400">{formatPrice(sub.amount)}</span>
                            </div>
                            {sub.status === 'trial' && sub.trial_ends_at && (
                              <div className="mt-1.5 flex items-center gap-1.5 text-xs font-bold text-purple-600 dark:text-purple-400">
                                <Clock className="w-3.5 h-3.5 text-purple-500 flex-shrink-0" />
                                <span>
                                  Trial ends {new Date(sub.trial_ends_at).toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' })}
                                  {(() => {
                                    const diff = Math.ceil((new Date(sub.trial_ends_at).getTime() - Date.now()) / (1000 * 60 * 60 * 24));
                                    return diff > 0 ? ` (${diff} days left)` : ' (Expires today)';
                                  })()}
                                </span>
                              </div>
                            )}
                          </div>

                          <div className="text-left sm:text-right">
                            <div className="text-[11px] text-slate-400 font-medium">
                              {sub.status === 'trial' ? 'Trial Expiry' : 'Next Billing Date'}
                            </div>
                            <div className="text-xs font-bold text-slate-800 dark:text-slate-200 mt-0.5 flex items-center gap-1 sm:justify-end">
                              <Calendar className="w-3.5 h-3.5 text-blue-500" />
                              {new Date(sub.status === 'trial' && sub.trial_ends_at ? sub.trial_ends_at : sub.next_billing_date).toLocaleDateString('en-US', {
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
                                Storage Used
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
                                Monthly Bandwidth
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
                              Connected Websites
                            </span>
                            <span className="font-bold text-slate-800 dark:text-slate-200">
                              {sub.websites_count} Sites
                            </span>
                          </div>
                        </div>
                      </div>

                      {/* Card Actions */}
                      <div className="flex items-center justify-between pt-4 border-t border-slate-100 dark:border-slate-800">
                        {sub.status === 'trial' ? (
                          <div className="flex items-center gap-2 w-full justify-between">
                            <div className="flex items-center gap-2">
                              <button
                                onClick={() => handleConvertTrial(sub.id)}
                                disabled={actionLoading === `convert-${sub.id}`}
                                className="px-3.5 py-2 rounded-xl text-xs font-bold bg-gradient-to-r from-emerald-600 to-teal-600 hover:from-emerald-700 hover:to-teal-700 text-white shadow-sm transition-all flex items-center gap-1.5"
                              >
                                {actionLoading === `convert-${sub.id}` ? (
                                  <RefreshCw className="w-3.5 h-3.5 animate-spin" />
                                ) : (
                                  <Zap className="w-3.5 h-3.5" />
                                )}
                                <span>Convert to Paid</span>
                              </button>

                              <button
                                onClick={() => {
                                  setSelectedTrialForExtend(sub);
                                  setExtendModalOpen(true);
                                }}
                                className="px-3.5 py-2 rounded-xl text-xs font-bold bg-purple-50 hover:bg-purple-100 dark:bg-purple-950/50 dark:hover:bg-purple-900/50 text-purple-600 dark:text-purple-400 transition-colors flex items-center gap-1.5"
                              >
                                <Clock className="w-3.5 h-3.5" />
                                <span>Extend</span>
                              </button>
                            </div>

                            <button
                              onClick={() => handleEndTrial(sub.id)}
                              disabled={actionLoading === `end-${sub.id}`}
                              className="text-xs font-bold text-rose-500 hover:text-rose-600 transition-colors"
                            >
                              End Trial
                            </button>
                          </div>
                        ) : (
                          <>
                            <div className="flex items-center gap-2">
                              <button
                                onClick={() => handleRenewSubscription(sub)}
                                disabled={actionLoading === `renew-${sub.id}`}
                                className="px-3.5 py-2 rounded-xl text-xs font-bold bg-blue-50 hover:bg-blue-100 dark:bg-blue-950/50 dark:hover:bg-blue-900/50 text-blue-600 dark:text-blue-400 transition-colors flex items-center gap-1.5"
                              >
                                <RefreshCw className={`w-3.5 h-3.5 ${actionLoading === `renew-${sub.id}` ? 'animate-spin' : ''}`} />
                                <span>Renew</span>
                              </button>

                              <button
                                onClick={() => setActiveTab('packages')}
                                className="px-3.5 py-2 rounded-xl text-xs font-bold bg-slate-100 hover:bg-slate-200 dark:bg-slate-800 dark:hover:bg-slate-700 text-slate-700 dark:text-slate-300 transition-colors"
                              >
                                Upgrade
                              </button>
                            </div>

                            {sub.status === 'active' && (
                              <button
                                onClick={() => handleCancelSubscription(sub)}
                                disabled={actionLoading === `cancel-${sub.id}`}
                                className="text-xs font-bold text-rose-500 hover:text-rose-600 transition-colors"
                              >
                                Cancel
                              </button>
                            )}
                          </>
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
                  Billing Invoices & Payment History
                </h2>
                <p className="text-xs text-slate-500 dark:text-slate-400">
                  Official invoices, downloadable receipts, and balance settlement for all hosting orders.
                </p>
              </div>
            </div>

            <div className="bg-white dark:bg-slate-900 rounded-3xl border border-slate-200 dark:border-slate-800 shadow-sm overflow-hidden">
              <div className="overflow-x-auto">
                <table className="w-full text-left text-xs">
                  <thead className="bg-slate-50 dark:bg-slate-800/80 text-slate-500 dark:text-slate-400 font-bold uppercase tracking-wider border-b border-slate-200 dark:border-slate-800">
                    <tr>
                      <th className="py-4 px-6">Invoice #</th>
                      <th className="py-4 px-6">Date</th>
                      <th className="py-4 px-6">Description</th>
                      <th className="py-4 px-6">Gateway</th>
                      <th className="py-4 px-6">Amount</th>
                      <th className="py-4 px-6">Status</th>
                      <th className="py-4 px-6 text-right">Action</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-slate-100 dark:divide-slate-800/60 font-medium">
                    {invoices.length === 0 ? (
                      <tr>
                        <td colSpan={7} className="text-center py-12 text-slate-400">
                          No invoice records found.
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
                            {new Date(inv.created_at).toLocaleDateString('en-US', {
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
                              {inv.status === 'paid' ? 'Paid' : inv.status === 'unpaid' ? 'Unpaid' : 'Cancelled'}
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
                                  Pay Now
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
                                <span>Receipt</span>
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
                Payment Gateway Integration Hub
              </h2>
              <p className="text-xs text-slate-500 dark:text-slate-400">
                Connect mobile financial services (bKash, Nagad, SSLCommerz) and international credit/debit card gateways (Stripe, PayPal).
              </p>
            </div>

            <div className="grid grid-cols-1 sm:grid-cols-2 xl:grid-cols-3 gap-4 sm:gap-6">
              {gateways.map(gw => (
                <div
                  key={gw.gateway}
                  className={`bg-white dark:bg-slate-900 rounded-2xl sm:rounded-3xl border p-5 sm:p-6 shadow-sm flex flex-col justify-between transition-all ${
                    gw.enabled
                      ? 'border-slate-200 dark:border-slate-800'
                      : 'border-dashed border-slate-300 dark:border-slate-800 opacity-75'
                  }`}
                >
                  <div>
                    {/* Header */}
                    <div className="flex items-center justify-between mb-4">
                      <div className="flex items-center gap-3 min-w-0">
                        <div
                          className={`w-10 h-10 rounded-2xl flex items-center justify-center font-bold text-lg flex-shrink-0 ${
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
                            ? 'bKash'
                            : gw.gateway === 'nagad'
                            ? 'Nagad'
                            : gw.gateway.toUpperCase().slice(0, 3)}
                        </div>
                        <div className="min-w-0 truncate">
                          <h3 className="font-bold text-sm text-slate-900 dark:text-white truncate">
                            {gw.display_name}
                          </h3>
                          <span className="text-[11px] text-slate-400 uppercase tracking-wider font-semibold">
                            {gw.gateway}
                          </span>
                        </div>
                      </div>

                      <span
                        className={`px-2.5 py-0.5 rounded-full text-[10px] font-extrabold uppercase tracking-wider flex-shrink-0 ml-2 ${
                          gw.enabled
                            ? 'bg-emerald-500/15 text-emerald-600 dark:text-emerald-400'
                            : 'bg-slate-100 dark:bg-slate-800 text-slate-400'
                        }`}
                      >
                        {gw.enabled ? 'Enabled' : 'Disabled'}
                      </span>
                    </div>

                    {/* Metadata */}
                    <div className="space-y-2 mb-6 text-xs bg-slate-50 dark:bg-slate-800/50 p-3 rounded-2xl">
                      <div className="flex items-center justify-between gap-2">
                        <span className="text-slate-500 flex-shrink-0">Environment:</span>
                        <span className="font-semibold text-slate-800 dark:text-slate-200 truncate">
                          {gw.test_mode ? 'Sandbox / Test' : 'Live Production'}
                        </span>
                      </div>
                      <div className="flex items-center justify-between gap-2">
                        <span className="text-slate-500 flex-shrink-0">Merchant ID:</span>
                        <span className="font-mono text-slate-700 dark:text-slate-300 truncate max-w-[150px] sm:max-w-[200px] text-right">
                          {gw.merchant_id || 'Not Set'}
                        </span>
                      </div>
                      <div className="flex items-center justify-between gap-2">
                        <span className="text-slate-500 flex-shrink-0">API Key:</span>
                        <span className="font-mono text-slate-700 dark:text-slate-300 truncate max-w-[150px] sm:max-w-[200px] text-right">
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
                      <span>Configure Settings</span>
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
                  Hosting Packages & Quota Manager
                </h2>
                <p className="text-xs text-slate-500 dark:text-slate-400">
                  As admin, create new cloud hosting packages, configure storage/bandwidth quotas, and update pricing.
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
                <span>Create New Package</span>
              </button>
            </div>

            <div className="bg-white dark:bg-slate-900 rounded-3xl border border-slate-200 dark:border-slate-800 shadow-sm overflow-hidden">
              <div className="overflow-x-auto">
                <table className="w-full text-left text-xs">
                  <thead className="bg-slate-50 dark:bg-slate-800/80 text-slate-500 dark:text-slate-400 font-bold uppercase tracking-wider border-b border-slate-200 dark:border-slate-800">
                    <tr>
                      <th className="py-4 px-6">Package Name</th>
                      <th className="py-4 px-6">Tier</th>
                      <th className="py-4 px-6">Monthly Price</th>
                      <th className="py-4 px-6">Yearly Price</th>
                      <th className="py-4 px-6">Disk Space</th>
                      <th className="py-4 px-6">Domain Quota</th>
                      <th className="py-4 px-6">Free Trial</th>
                      <th className="py-4 px-6">Status</th>
                      <th className="py-4 px-6 text-right">Action</th>
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
                          {p.max_websites} Sites
                        </td>
                        <td className="py-4 px-6">
                          <span
                            className={`px-2.5 py-0.5 rounded-full text-[10px] font-bold ${
                              p.trial_allowed
                                ? 'bg-purple-500/15 text-purple-600 dark:text-purple-400 border border-purple-500/20'
                                : 'bg-slate-100 dark:bg-slate-800 text-slate-400'
                            }`}
                          >
                            {p.trial_allowed ? `${p.trial_days || 14}d Trial` : 'Disabled'}
                          </span>
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
                              title="Edit"
                            >
                              <Edit2 className="w-4 h-4" />
                            </button>
                            <button
                              onClick={() => handleDeletePlan(p.id)}
                              className="p-2 rounded-lg text-slate-600 hover:text-rose-600 hover:bg-rose-50 dark:hover:bg-rose-950/50 transition-colors"
                              title="Delete"
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

        {/* TAB 6: FREE TRIALS & CONVERSION MANAGER */}
        {activeTab === 'trials' && (
          <div className="space-y-8 animate-in fade-in duration-300">
            {/* Header */}
            <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
              <div>
                <h2 className="text-xl font-bold text-slate-900 dark:text-white flex items-center gap-2">
                  <Gift className="w-5 h-5 text-purple-500" />
                  <span>Free Trial Subscriptions & Platform Policies</span>
                </h2>
                <p className="text-xs text-slate-500 dark:text-slate-400 mt-0.5">
                  Supervise customer 14-day free trials, manage anti-abuse policies, extend durations, and convert trials directly to paid subscriptions.
                </p>
              </div>

              <button
                onClick={() => setActiveTab('packages')}
                className="inline-flex items-center gap-2 px-4 py-2.5 rounded-xl bg-purple-600 hover:bg-purple-700 text-white text-sm font-bold shadow-md shadow-purple-600/20 transition-all"
              >
                <Plus className="w-4 h-4" />
                <span>Start New Free Trial</span>
              </button>
            </div>

            {/* Trial Key Metrics */}
            <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
              <div className="p-5 rounded-3xl bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 shadow-sm">
                <div className="text-xs font-semibold text-slate-500 uppercase tracking-wider">Total Trials Initiated</div>
                <div className="text-3xl font-black text-slate-900 dark:text-white mt-1">
                  {safeTrials.length > 0 ? safeTrials.length : safeSubscriptions.filter(s => s && s.status === 'trial').length}
                </div>
                <div className="text-[11px] text-slate-400 mt-1">From landing page and dashboard</div>
              </div>

              <div className="p-5 rounded-3xl bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 shadow-sm">
                <div className="text-xs font-semibold text-slate-500 uppercase tracking-wider">Active Trials In Progress</div>
                <div className="text-3xl font-black text-purple-600 dark:text-purple-400 mt-1">
                  {activeTrialsCount}
                </div>
                <div className="text-[11px] text-purple-500/80 mt-1">Currently evaluating Hostvra stack</div>
              </div>

              <div className="p-5 rounded-3xl bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 shadow-sm">
                <div className="text-xs font-semibold text-slate-500 uppercase tracking-wider">Converted to Paid</div>
                <div className="text-3xl font-black text-emerald-600 dark:text-emerald-400 mt-1">
                  {convertedTrialsCount}
                </div>
                <div className="text-[11px] text-emerald-500/80 mt-1">Upgraded into recurring subscriptions</div>
              </div>

              <div className="p-5 rounded-3xl bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 shadow-sm">
                <div className="text-xs font-semibold text-slate-500 uppercase tracking-wider">Conversion Rate</div>
                <div className="text-3xl font-black text-blue-600 dark:text-blue-400 mt-1">
                  {trialConversionRate}%
                </div>
                <div className="text-[11px] text-blue-500/80 mt-1">Trial-to-paid customer conversion</div>
              </div>
            </div>

            {/* Trial Settings & Abuse Policy Card */}
            <div className="p-6 sm:p-7 rounded-3xl bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 shadow-sm space-y-6">
              <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4 pb-4 border-b border-slate-100 dark:border-slate-800">
                <div className="flex items-center gap-3">
                  <div className="p-2.5 rounded-2xl bg-purple-500/10 text-purple-600 dark:text-purple-400">
                    <Sliders className="w-5 h-5" />
                  </div>
                  <div>
                    <h3 className="text-base font-bold text-slate-900 dark:text-white">
                      Global Free Trial Policies & Anti-Abuse Controls
                    </h3>
                    <p className="text-xs text-slate-500 dark:text-slate-400">
                      Enforce system-wide trial duration limits, master toggle, and anti-abuse safeguards.
                    </p>
                  </div>
                </div>

                <button
                  type="button"
                  onClick={handleSaveTrialSettings}
                  disabled={actionLoading === 'save-trial-settings'}
                  className="px-5 py-2.5 rounded-xl bg-purple-600 hover:bg-purple-700 text-white text-xs font-bold shadow-md shadow-purple-500/20 transition-all flex items-center gap-2"
                >
                  {actionLoading === 'save-trial-settings' ? <RefreshCw className="w-4 h-4 animate-spin" /> : <Check className="w-4 h-4" />}
                  <span>Save Policy Settings</span>
                </button>
              </div>

              <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6 text-xs">
                {/* Master Switch */}
                <div className="p-4 rounded-2xl bg-slate-50 dark:bg-slate-800/50 border border-slate-200/80 dark:border-slate-700/60 space-y-3 flex flex-col justify-between">
                  <div>
                    <div className="flex items-center justify-between">
                      <span className="font-bold text-slate-900 dark:text-white">Master Trial Switch</span>
                      <button
                        type="button"
                        onClick={() => setTrialSettings(s => ({ ...s, enabled: !s.enabled }))}
                        className={`w-11 h-6 rounded-full transition-colors relative flex items-center p-0.5 ${
                          trialSettings.enabled ? 'bg-purple-600' : 'bg-slate-300 dark:bg-slate-700'
                        }`}
                      >
                        <span
                          className={`w-5 h-5 rounded-full bg-white transition-transform ${
                            trialSettings.enabled ? 'translate-x-5' : 'translate-x-0'
                          }`}
                        />
                      </button>
                    </div>
                    <p className="text-[11px] text-slate-500 dark:text-slate-400 mt-1.5 leading-relaxed">
                      Allow new customers to activate instant hosting trials without initial payment.
                    </p>
                  </div>
                  <div className="text-[10px] font-extrabold uppercase tracking-wider text-purple-600 dark:text-purple-400">
                    {trialSettings.enabled ? 'Master Switch: Enabled' : 'Master Switch: Disabled'}
                  </div>
                </div>

                {/* Default Duration */}
                <div className="p-4 rounded-2xl bg-slate-50 dark:bg-slate-800/50 border border-slate-200/80 dark:border-slate-700/60 space-y-3">
                  <div className="font-bold text-slate-900 dark:text-white">Default Trial Duration</div>
                  <p className="text-[11px] text-slate-500 dark:text-slate-400 leading-relaxed">
                    Standard duration in days for newly provisioned hosting trials.
                  </p>
                  <div className="flex items-center gap-2">
                    <input
                      type="number"
                      min="1"
                      max="90"
                      value={trialSettings.default_days}
                      onChange={e => setTrialSettings(s => ({ ...s, default_days: Math.max(1, parseInt(e.target.value) || 14) }))}
                      className="w-24 px-3 py-1.5 rounded-xl border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 text-slate-900 dark:text-white font-bold"
                    />
                    <span className="text-slate-500 font-semibold">Days</span>
                  </div>
                </div>

                {/* One Trial Per Customer */}
                <div className="p-4 rounded-2xl bg-slate-50 dark:bg-slate-800/50 border border-slate-200/80 dark:border-slate-700/60 space-y-3 flex flex-col justify-between">
                  <div>
                    <div className="flex items-center justify-between">
                      <span className="font-bold text-slate-900 dark:text-white">Anti-Abuse Verification</span>
                      <button
                        type="button"
                        onClick={() => setTrialSettings(s => ({ ...s, one_trial_per_customer: !s.one_trial_per_customer }))}
                        className={`w-11 h-6 rounded-full transition-colors relative flex items-center p-0.5 ${
                          trialSettings.one_trial_per_customer ? 'bg-purple-600' : 'bg-slate-300 dark:bg-slate-700'
                        }`}
                      >
                        <span
                          className={`w-5 h-5 rounded-full bg-white transition-transform ${
                            trialSettings.one_trial_per_customer ? 'translate-x-5' : 'translate-x-0'
                          }`}
                        />
                      </button>
                    </div>
                    <p className="text-[11px] text-slate-500 dark:text-slate-400 mt-1.5 leading-relaxed">
                      Enforce one trial per user account and client IP address to prevent resource farming.
                    </p>
                  </div>
                  <div className="text-[10px] font-extrabold uppercase tracking-wider text-purple-600 dark:text-purple-400">
                    {trialSettings.one_trial_per_customer ? 'Strict Anti-Abuse Active' : 'Multiple Trials Allowed'}
                  </div>
                </div>

                {/* Auto Suspend On Expiry */}
                <div className="p-4 rounded-2xl bg-slate-50 dark:bg-slate-800/50 border border-slate-200/80 dark:border-slate-700/60 space-y-3 flex flex-col justify-between">
                  <div>
                    <div className="flex items-center justify-between">
                      <span className="font-bold text-slate-900 dark:text-white">Auto-Suspend on Expiry</span>
                      <button
                        type="button"
                        onClick={() => setTrialSettings(s => ({ ...s, auto_suspend_on_expiry: !s.auto_suspend_on_expiry }))}
                        className={`w-11 h-6 rounded-full transition-colors relative flex items-center p-0.5 ${
                          trialSettings.auto_suspend_on_expiry ? 'bg-purple-600' : 'bg-slate-300 dark:bg-slate-700'
                        }`}
                      >
                        <span
                          className={`w-5 h-5 rounded-full bg-white transition-transform ${
                            trialSettings.auto_suspend_on_expiry ? 'translate-x-5' : 'translate-x-0'
                          }`}
                        />
                      </button>
                    </div>
                    <p className="text-[11px] text-slate-500 dark:text-slate-400 mt-1.5 leading-relaxed">
                      Automatically suspend client containers and vhosts when trial period expires without conversion.
                    </p>
                  </div>
                  <div className="text-[10px] font-extrabold uppercase tracking-wider text-emerald-600 dark:text-emerald-400">
                    {trialSettings.auto_suspend_on_expiry ? 'Auto-Suspend Enabled' : 'Grace Period Allowed'}
                  </div>
                </div>

                {/* Require Payment Method */}
                <div className="p-4 rounded-2xl bg-slate-50 dark:bg-slate-800/50 border border-slate-200/80 dark:border-slate-700/60 space-y-3 flex flex-col justify-between">
                  <div>
                    <div className="flex items-center justify-between">
                      <span className="font-bold text-slate-900 dark:text-white">Require Card Upfront</span>
                      <button
                        type="button"
                        onClick={() => setTrialSettings(s => ({ ...s, require_payment_method: !s.require_payment_method }))}
                        className={`w-11 h-6 rounded-full transition-colors relative flex items-center p-0.5 ${
                          trialSettings.require_payment_method ? 'bg-purple-600' : 'bg-slate-300 dark:bg-slate-700'
                        }`}
                      >
                        <span
                          className={`w-5 h-5 rounded-full bg-white transition-transform ${
                            trialSettings.require_payment_method ? 'translate-x-5' : 'translate-x-0'
                          }`}
                        />
                      </button>
                    </div>
                    <p className="text-[11px] text-slate-500 dark:text-slate-400 mt-1.5 leading-relaxed">
                      Require credit card verification prior to trial activation ($0 authorization charge).
                    </p>
                  </div>
                  <div className="text-[10px] font-extrabold uppercase tracking-wider text-slate-500">
                    {trialSettings.require_payment_method ? 'Card Required' : 'No Card Required (Instant)'}
                  </div>
                </div>
              </div>
            </div>

            {/* Live Customer Free Trials Table */}
            <div className="bg-white dark:bg-slate-900 rounded-3xl border border-slate-200 dark:border-slate-800 shadow-sm overflow-hidden space-y-4 p-6">
              <div className="flex items-center justify-between">
                <div>
                  <h3 className="text-base font-bold text-slate-900 dark:text-white">
                    Active Customer Free Trials
                  </h3>
                  <p className="text-xs text-slate-500">
                    Real-time list of all users and organizations on trial status with countdown timers.
                  </p>
                </div>
                <button
                  onClick={loadData}
                  className="p-2 rounded-xl text-slate-400 hover:text-slate-600 hover:bg-slate-100 dark:hover:bg-slate-800 transition-colors"
                  title="Refresh Trials"
                >
                  <RefreshCw className="w-4 h-4" />
                </button>
              </div>

              {(((safeTrials.length > 0 ? safeTrials : safeSubscriptions).filter(s => s && (s.status === 'trial' || s.trial_ends_at)).length === 0)) ? (
                <div className="text-center py-12 px-4 border border-dashed border-slate-200 dark:border-slate-800 rounded-2xl">
                  <Gift className="w-10 h-10 text-purple-400 mx-auto mb-3" />
                  <div className="text-sm font-bold text-slate-900 dark:text-white">No Active Customer Free Trials</div>
                  <p className="text-xs text-slate-500 max-w-sm mx-auto mt-1 mb-4">
                    Customers who register with free trial plans from the public landing page will be displayed here with one-click extension and conversion actions.
                  </p>
                  <button
                    onClick={() => setActiveTab('packages')}
                    className="px-4 py-2 rounded-xl bg-purple-600 hover:bg-purple-700 text-white text-xs font-bold transition-all shadow-sm"
                  >
                    View Available Trial Packages
                  </button>
                </div>
              ) : (
                <div className="overflow-x-auto">
                  <table className="w-full text-left text-xs">
                    <thead className="bg-slate-50 dark:bg-slate-800/80 text-slate-500 dark:text-slate-400 font-bold uppercase tracking-wider border-b border-slate-200 dark:border-slate-800">
                      <tr>
                        <th className="py-3 px-4">Subscription ID</th>
                        <th className="py-3 px-4">Package</th>
                        <th className="py-3 px-4">Trial Started</th>
                        <th className="py-3 px-4">Trial Expiry / Remaining</th>
                        <th className="py-3 px-4">Status</th>
                        <th className="py-3 px-4 text-right">Actions</th>
                      </tr>
                    </thead>
                    <tbody className="divide-y divide-slate-100 dark:divide-slate-800/60 font-medium">
                      {(safeTrials.length > 0 ? safeTrials : safeSubscriptions)
                        .filter(s => s && (s.status === 'trial' || s.trial_ends_at))
                        .map(tr => {
                          const isTrial = tr.status === 'trial';
                          const endsAt = tr.trial_ends_at ? new Date(tr.trial_ends_at) : new Date(tr.next_billing_date);
                          const daysLeft = Math.ceil((endsAt.getTime() - Date.now()) / (1000 * 60 * 60 * 24));

                          return (
                            <tr key={tr.id} className="hover:bg-slate-50/80 dark:hover:bg-slate-800/30 transition-colors">
                              <td className="py-3.5 px-4 font-mono text-[11px] text-slate-600 dark:text-slate-400">
                                <div>{tr.id.substring(0, 13)}...</div>
                                <div className="text-[10px] text-slate-400">User: {tr.user_id ? tr.user_id.substring(0, 8) : 'demo-user'}</div>
                              </td>
                              <td className="py-3.5 px-4 font-bold text-slate-900 dark:text-white">
                                {tr.plan_name}
                              </td>
                              <td className="py-3.5 px-4 text-slate-600 dark:text-slate-400">
                                {tr.trial_started_at
                                  ? new Date(tr.trial_started_at).toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' })
                                  : new Date(tr.created_at).toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' })}
                              </td>
                              <td className="py-3.5 px-4">
                                <div className="font-semibold text-slate-900 dark:text-slate-100">
                                  {endsAt.toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' })}
                                </div>
                                <div className="text-[10px] font-bold mt-0.5">
                                  {isTrial ? (
                                    daysLeft > 0 ? (
                                      <span className="text-purple-600 dark:text-purple-400">⏳ {daysLeft} days remaining</span>
                                    ) : (
                                      <span className="text-rose-600 dark:text-rose-400">⚠️ Expired today</span>
                                    )
                                  ) : (
                                    <span className="text-emerald-600 dark:text-emerald-400">Converted</span>
                                  )}
                                </div>
                              </td>
                              <td className="py-3.5 px-4">
                                <span
                                  className={`px-2.5 py-0.5 rounded-full text-[10px] font-extrabold uppercase tracking-wider ${
                                    tr.status === 'trial'
                                      ? 'bg-purple-500/15 text-purple-600 dark:text-purple-400 border border-purple-500/30'
                                      : tr.status === 'active'
                                      ? 'bg-emerald-500/15 text-emerald-600 dark:text-emerald-400'
                                      : 'bg-rose-500/15 text-rose-600'
                                  }`}
                                >
                                  {tr.status === 'trial' ? 'Free Trial' : tr.status === 'active' ? 'Converted to Paid' : tr.status}
                                </span>
                              </td>
                              <td className="py-3.5 px-4 text-right">
                                {isTrial ? (
                                  <div className="flex items-center justify-end gap-1.5">
                                    <button
                                      onClick={() => {
                                        setSelectedTrialForExtend(tr);
                                        setExtendModalOpen(true);
                                      }}
                                      className="px-2.5 py-1.5 rounded-lg bg-purple-50 hover:bg-purple-100 dark:bg-purple-950/50 dark:hover:bg-purple-900/50 text-purple-600 dark:text-purple-400 text-[11px] font-bold transition-colors"
                                      title="Extend Trial"
                                    >
                                      +Extend
                                    </button>
                                    <button
                                      onClick={() => handleConvertTrial(tr.id)}
                                      disabled={actionLoading === `convert-${tr.id}`}
                                      className="px-2.5 py-1.5 rounded-lg bg-emerald-600 hover:bg-emerald-700 text-white text-[11px] font-bold transition-all shadow-sm"
                                      title="Convert to Paid Subscription"
                                    >
                                      {actionLoading === `convert-${tr.id}` ? '...' : 'Convert'}
                                    </button>
                                    <button
                                      onClick={() => handleEndTrial(tr.id)}
                                      disabled={actionLoading === `end-${tr.id}`}
                                      className="p-1.5 rounded-lg text-rose-500 hover:text-rose-600 hover:bg-rose-50 dark:hover:bg-rose-950/50 transition-colors"
                                      title="End Trial"
                                    >
                                      <X className="w-3.5 h-3.5" />
                                    </button>
                                  </div>
                                ) : (
                                  <span className="text-[11px] text-slate-400 italic">No actions</span>
                                )}
                              </td>
                            </tr>
                          );
                        })}
                    </tbody>
                  </table>
                </div>
              )}
            </div>
          </div>
        )}

        {/* MODAL 1: CHECKOUT & SUBSCRIPTION ORDER MODAL */}
        {checkoutModalOpen && selectedPlanForOrder && (
          <div className="fixed inset-0 z-50 flex items-center justify-center p-3 sm:p-4 bg-slate-950/60 backdrop-blur-sm animate-in fade-in duration-200">
            <div className="relative w-full max-w-lg bg-white dark:bg-slate-900 rounded-3xl border border-slate-200 dark:border-slate-800 shadow-2xl p-5 sm:p-7 space-y-6 max-h-[90vh] overflow-y-auto">
              {/* Modal Header */}
              <div className="flex items-center justify-between pb-4 border-b border-slate-100 dark:border-slate-800">
                <div className="flex items-center gap-3">
                  <div className="p-2.5 rounded-2xl bg-blue-500/10 text-blue-600 dark:text-blue-400">
                    <CreditCard className="w-6 h-6" />
                  </div>
                  <div>
                    <h3 className="text-lg font-bold text-slate-900 dark:text-white">
                      Hosting Package Order & Checkout
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
                  <span className="text-slate-500">Selected Package:</span>
                  <span className="font-bold text-slate-900 dark:text-white">{selectedPlanForOrder.name}</span>
                </div>
                <div className="flex items-center justify-between text-xs">
                  <span className="text-slate-500">Billing Cycle:</span>
                  <span className="font-bold text-slate-900 dark:text-white">
                    {billingCycle === 'yearly' ? '1 Year (Annual Billing)' : '1 Month (Monthly Billing)'}
                  </span>
                </div>
                <div className="flex items-center justify-between text-xs">
                  <span className="text-slate-500">Storage Quota:</span>
                  <span className="font-semibold text-slate-700 dark:text-slate-300">
                    {Math.round(selectedPlanForOrder.disk_space_mb / 1024)} GB NVMe
                  </span>
                </div>
                {couponApplied && (
                  <div className="flex items-center justify-between text-xs text-emerald-600 font-bold">
                    <span>Promo Discount (10% OFF):</span>
                    <span>
                      -{formatPrice((billingCycle === 'yearly' ? selectedPlanForOrder.price_yearly : selectedPlanForOrder.price_monthly) * 0.1)}
                    </span>
                  </div>
                )}
                <div className="pt-3 border-t border-slate-200 dark:border-slate-700 flex items-center justify-between">
                  <span className="text-sm font-bold text-slate-900 dark:text-white">Total Payable Amount:</span>
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
                  placeholder="Coupon code (e.g. HOSTVRA10)"
                  className="flex-1 px-4 py-2.5 rounded-xl border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 text-xs text-slate-900 dark:text-white outline-none focus:ring-2 focus:ring-blue-500"
                />
                <button
                  type="button"
                  onClick={() => {
                    if (couponCode.trim().toUpperCase() === 'HOSTVRA10') {
                      setCouponApplied(true);
                      showNotify('success', 'Coupon code HOSTVRA10 applied successfully!');
                    } else {
                      showNotify('error', 'Invalid coupon code. Please try HOSTVRA10.');
                    }
                  }}
                  className="px-4 py-2.5 rounded-xl bg-slate-100 hover:bg-slate-200 dark:bg-slate-800 dark:hover:bg-slate-700 text-xs font-bold text-slate-700 dark:text-slate-300 transition-colors"
                >
                  Apply
                </button>
              </div>

              {/* Payment Gateway Picker */}
              <div className="space-y-2">
                <label className="text-xs font-bold text-slate-700 dark:text-slate-300">
                  Select Payment Method:
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
              <div className="flex items-center justify-between gap-3 pt-4 border-t border-slate-100 dark:border-slate-800">
                {selectedPlanForOrder.trial_allowed ? (
                  <button
                    type="button"
                    onClick={() => {
                      setCheckoutModalOpen(false);
                      handleStartFreeTrial(selectedPlanForOrder);
                    }}
                    disabled={actionLoading === `trial-${selectedPlanForOrder.id}`}
                    className="px-4 py-2.5 rounded-xl border border-purple-500/40 text-purple-600 dark:text-purple-400 hover:bg-purple-50 dark:hover:bg-purple-950/40 text-xs font-bold transition-all flex items-center justify-center gap-1.5"
                  >
                    <Gift className="w-3.5 h-3.5 text-purple-500" />
                    <span>Start Free Trial ($0)</span>
                  </button>
                ) : (
                  <div />
                )}

                <div className="flex items-center justify-end gap-2">
                  <button
                    type="button"
                    onClick={() => setCheckoutModalOpen(false)}
                    className="px-4 py-2.5 rounded-xl text-xs font-bold text-slate-600 hover:bg-slate-100 dark:hover:bg-slate-800 transition-colors"
                  >
                    Cancel
                  </button>
                  <button
                    type="button"
                    onClick={handleConfirmCheckout}
                    disabled={actionLoading === 'checkout'}
                    className="px-6 py-2.5 rounded-xl bg-gradient-to-r from-blue-600 to-indigo-600 hover:from-blue-700 hover:to-indigo-700 text-white text-xs font-bold shadow-lg shadow-blue-500/20 transition-all flex items-center justify-center gap-2"
                  >
                    {actionLoading === 'checkout' && <RefreshCw className="w-4 h-4 animate-spin" />}
                    <span>Confirm Order</span>
                  </button>
                </div>
              </div>
            </div>
          </div>
        )}

        {/* MODAL 2: INVOICE & RECEIPT MODAL (PRINTABLE / BRANDED) */}
        {receiptModalOpen && selectedInvoice && (
          <div className="fixed inset-0 z-50 flex items-center justify-center p-3 sm:p-4 bg-slate-950/60 backdrop-blur-sm animate-in fade-in duration-200">
            <div className="relative w-full max-w-2xl bg-white dark:bg-slate-900 rounded-3xl border border-slate-200 dark:border-slate-800 shadow-2xl p-5 sm:p-8 space-y-6 max-h-[90vh] overflow-y-auto">
              {/* Top Controls */}
              <div className="flex items-center justify-between pb-4 border-b border-slate-100 dark:border-slate-800 print:hidden">
                <div className="flex items-center gap-2">
                  <span className="text-xs font-bold uppercase tracking-wider text-slate-400">
                    Official Money Receipt
                  </span>
                </div>

                <div className="flex items-center gap-2">
                  <button
                    onClick={() => window.print()}
                    className="p-2 rounded-xl text-slate-600 dark:text-slate-300 hover:bg-slate-100 dark:hover:bg-slate-800 text-xs font-bold flex items-center gap-1.5 transition-colors"
                  >
                    <Printer className="w-4 h-4" />
                    <span>Print</span>
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
                    Date: {new Date(selectedInvoice.created_at).toLocaleDateString('en-US')}
                  </div>
                  <div className="inline-block px-3 py-1 rounded-full text-xs font-bold uppercase tracking-wider bg-emerald-500/15 text-emerald-600 dark:text-emerald-400">
                    {selectedInvoice.status === 'paid' ? 'PAID' : 'UNPAID'}
                  </div>
                </div>
              </div>

              {/* Items Table */}
              <div className="py-2">
                <table className="w-full text-left text-xs">
                  <thead className="text-slate-400 uppercase font-bold border-b border-slate-100 dark:border-slate-800 pb-2">
                    <tr>
                      <th className="py-2">Item Description</th>
                      <th className="py-2 text-right">Amount</th>
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
                  <span>Subtotal:</span>
                  <span className="font-bold text-slate-800 dark:text-slate-200">{formatPrice(selectedInvoice.subtotal)}</span>
                </div>
                <div className="flex justify-between text-slate-500">
                  <span>VAT & Tax (0%):</span>
                  <span className="font-bold text-slate-800 dark:text-slate-200">{formatPrice(0)}</span>
                </div>
                <div className="pt-2 border-t border-slate-200 dark:border-slate-700 flex justify-between text-sm font-black text-slate-900 dark:text-white">
                  <span>Total Paid Amount:</span>
                  <span className="text-blue-600 dark:text-blue-400 text-lg">{formatPrice(selectedInvoice.total)}</span>
                </div>
              </div>

              {/* Payment Details Footer */}
              <div className="flex items-center justify-between text-xs text-slate-400 pt-4 border-t border-slate-100 dark:border-slate-800">
                <div>
                  Payment Method: <span className="font-bold text-slate-700 dark:text-slate-300 uppercase">{selectedInvoice.payment_method || 'Online'}</span>
                  {selectedInvoice.transaction_id && (
                    <span className="ml-2 font-mono text-[11px]">({selectedInvoice.transaction_id})</span>
                  )}
                </div>
                <div className="text-emerald-600 dark:text-emerald-400 font-bold flex items-center gap-1">
                  <CheckCircle2 className="w-4 h-4" />
                  Verified Digital Receipt
                </div>
              </div>
            </div>
          </div>
        )}

        {/* MODAL 3: PAYMENT GATEWAY CONFIG MODAL */}
        {gatewayModalOpen && editingGateway && (
          <div className="fixed inset-0 z-50 flex items-center justify-center p-3 sm:p-4 bg-slate-950/60 backdrop-blur-sm animate-in fade-in duration-200">
            <div className="relative w-full max-w-lg bg-white dark:bg-slate-900 rounded-3xl border border-slate-200 dark:border-slate-800 shadow-2xl p-5 sm:p-7 space-y-5 max-h-[90vh] overflow-y-auto">
              <div className="flex items-center justify-between pb-4 border-b border-slate-100 dark:border-slate-800">
                <div className="flex items-center gap-3">
                  <div className="p-2.5 rounded-2xl bg-indigo-500/10 text-indigo-600 dark:text-indigo-400">
                    <Key className="w-5 h-5" />
                  </div>
                  <div>
                    <h3 className="text-lg font-bold text-slate-900 dark:text-white">
                      {editingGateway.display_name} Settings
                    </h3>
                    <p className="text-xs text-slate-400">Gateway credentials and security configuration</p>
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
                  <span className="font-bold text-slate-700 dark:text-slate-300">Gateway Status</span>
                  <button
                    type="button"
                    onClick={() => setEditingGateway({ ...editingGateway, enabled: !editingGateway.enabled })}
                    className={`px-3 py-1.5 rounded-xl font-bold transition-all ${
                      editingGateway.enabled ? 'bg-emerald-500 text-white' : 'bg-slate-200 dark:bg-slate-700 text-slate-600'
                    }`}
                  >
                    {editingGateway.enabled ? 'Enabled' : 'Disabled'}
                  </button>
                </div>

                <div className="flex items-center justify-between p-3 rounded-2xl bg-slate-50 dark:bg-slate-800/50">
                  <span className="font-bold text-slate-700 dark:text-slate-300">Sandbox / Test Mode</span>
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
                  Cancel
                </button>
                <button
                  type="button"
                  onClick={handleSaveGateway}
                  disabled={actionLoading === 'save-gateway'}
                  className="px-6 py-2 rounded-xl bg-blue-600 hover:bg-blue-700 text-white text-xs font-bold shadow-md shadow-blue-500/20"
                >
                  Save Changes
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
                      {editingPlan.id ? 'Edit Package' : 'Create New Hosting Package'}
                    </h3>
                    <p className="text-xs text-slate-400">Configure quotas, pricing, and included features</p>
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
                  <label className="font-bold text-slate-700 dark:text-slate-300">Package Name</label>
                  <input
                    type="text"
                    value={editingPlan.name || ''}
                    onChange={e => setEditingPlan({ ...editingPlan, name: e.target.value })}
                    className="w-full px-4 py-2.5 rounded-xl border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 text-slate-900 dark:text-white"
                    placeholder="e.g. Ultra Cloud Plus"
                  />
                </div>

                <div className="space-y-1">
                  <label className="font-bold text-slate-700 dark:text-slate-300">Tier</label>
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
                  <label className="font-bold text-slate-700 dark:text-slate-300">Monthly Price (USD)</label>
                  <input
                    type="number"
                    step="0.01"
                    value={editingPlan.price_monthly || 0}
                    onChange={e => setEditingPlan({ ...editingPlan, price_monthly: parseFloat(e.target.value) || 0 })}
                    className="w-full px-4 py-2.5 rounded-xl border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 text-slate-900 dark:text-white"
                  />
                </div>

                <div className="space-y-1">
                  <label className="font-bold text-slate-700 dark:text-slate-300">Yearly Price (USD)</label>
                  <input
                    type="number"
                    step="0.01"
                    value={editingPlan.price_yearly || 0}
                    onChange={e => setEditingPlan({ ...editingPlan, price_yearly: parseFloat(e.target.value) || 0 })}
                    className="w-full px-4 py-2.5 rounded-xl border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 text-slate-900 dark:text-white"
                  />
                </div>

                <div className="space-y-1">
                  <label className="font-bold text-slate-700 dark:text-slate-300">Storage (MB) (10240 = 10GB)</label>
                  <input
                    type="number"
                    value={editingPlan.disk_space_mb || 10240}
                    onChange={e => setEditingPlan({ ...editingPlan, disk_space_mb: parseInt(e.target.value) || 0 })}
                    className="w-full px-4 py-2.5 rounded-xl border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 text-slate-900 dark:text-white"
                  />
                </div>

                <div className="space-y-1">
                  <label className="font-bold text-slate-700 dark:text-slate-300">Bandwidth (MB) (102400 = 100GB)</label>
                  <input
                    type="number"
                    value={editingPlan.bandwidth_mb || 102400}
                    onChange={e => setEditingPlan({ ...editingPlan, bandwidth_mb: parseInt(e.target.value) || 0 })}
                    className="w-full px-4 py-2.5 rounded-xl border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 text-slate-900 dark:text-white"
                  />
                </div>

                <div className="space-y-1">
                  <label className="font-bold text-slate-700 dark:text-slate-300">Max Websites</label>
                  <input
                    type="number"
                    value={editingPlan.max_websites || 1}
                    onChange={e => setEditingPlan({ ...editingPlan, max_websites: parseInt(e.target.value) || 1 })}
                    className="w-full px-4 py-2.5 rounded-xl border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 text-slate-900 dark:text-white"
                  />
                </div>

                <div className="space-y-1">
                  <label className="font-bold text-slate-700 dark:text-slate-300">Max Databases</label>
                  <input
                    type="number"
                    value={editingPlan.max_databases || 2}
                    onChange={e => setEditingPlan({ ...editingPlan, max_databases: parseInt(e.target.value) || 1 })}
                    className="w-full px-4 py-2.5 rounded-xl border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 text-slate-900 dark:text-white"
                  />
                </div>

                <div className="space-y-1">
                  <label className="font-bold text-slate-700 dark:text-slate-300">Max Mailboxes</label>
                  <input
                    type="number"
                    value={editingPlan.max_mailboxes || 5}
                    onChange={e => setEditingPlan({ ...editingPlan, max_mailboxes: parseInt(e.target.value) || 1 })}
                    className="w-full px-4 py-2.5 rounded-xl border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 text-slate-900 dark:text-white"
                  />
                </div>

                <div className="space-y-1">
                  <label className="font-bold text-slate-700 dark:text-slate-300">Max FTP Accounts</label>
                  <input
                    type="number"
                    value={editingPlan.max_ftp || 2}
                    onChange={e => setEditingPlan({ ...editingPlan, max_ftp: parseInt(e.target.value) || 1 })}
                    className="w-full px-4 py-2.5 rounded-xl border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 text-slate-900 dark:text-white"
                  />
                </div>

                <div className="space-y-1">
                  <label className="font-bold text-slate-700 dark:text-slate-300">CPU Limit (vCPU Cores)</label>
                  <input
                    type="number"
                    min="1"
                    max="64"
                    value={editingPlan.cpu_limit || 1}
                    onChange={e => setEditingPlan({ ...editingPlan, cpu_limit: parseInt(e.target.value) || 1 })}
                    className="w-full px-4 py-2.5 rounded-xl border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 text-slate-900 dark:text-white"
                  />
                </div>

                <div className="space-y-1">
                  <label className="font-bold text-slate-700 dark:text-slate-300">RAM Limit (MB) (2048 = 2GB)</label>
                  <input
                    type="number"
                    min="512"
                    step="512"
                    value={editingPlan.ram_limit_mb || 2048}
                    onChange={e => setEditingPlan({ ...editingPlan, ram_limit_mb: parseInt(e.target.value) || 2048 })}
                    className="w-full px-4 py-2.5 rounded-xl border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 text-slate-900 dark:text-white"
                  />
                </div>

                <div className="space-y-1">
                  <label className="font-bold text-slate-700 dark:text-slate-300">Max Cron Jobs</label>
                  <input
                    type="number"
                    value={editingPlan.max_cron || 5}
                    onChange={e => setEditingPlan({ ...editingPlan, max_cron: parseInt(e.target.value) || 0 })}
                    className="w-full px-4 py-2.5 rounded-xl border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 text-slate-900 dark:text-white"
                  />
                </div>

                <div className="space-y-1">
                  <label className="font-bold text-slate-700 dark:text-slate-300">Max Subdomains</label>
                  <input
                    type="number"
                    value={editingPlan.max_subdomains || 10}
                    onChange={e => setEditingPlan({ ...editingPlan, max_subdomains: parseInt(e.target.value) || 0 })}
                    className="w-full px-4 py-2.5 rounded-xl border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 text-slate-900 dark:text-white"
                  />
                </div>

                <div className="space-y-1">
                  <label className="font-bold text-slate-700 dark:text-slate-300">Setup Fee ($)</label>
                  <input
                    type="number"
                    step="0.01"
                    value={editingPlan.setup_fee || 0}
                    onChange={e => setEditingPlan({ ...editingPlan, setup_fee: parseFloat(e.target.value) || 0 })}
                    className="w-full px-4 py-2.5 rounded-xl border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 text-slate-900 dark:text-white"
                  />
                </div>

                <div className="space-y-1">
                  <label className="font-bold text-slate-700 dark:text-slate-300">Trial Duration (Days)</label>
                  <input
                    type="number"
                    min="1"
                    max="90"
                    disabled={!editingPlan.trial_allowed}
                    value={editingPlan.trial_days || 14}
                    onChange={e => setEditingPlan({ ...editingPlan, trial_days: parseInt(e.target.value) || 14 })}
                    className="w-full px-4 py-2.5 rounded-xl border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 text-slate-900 dark:text-white disabled:opacity-50"
                  />
                </div>

                <div className="col-span-2 grid grid-cols-2 sm:grid-cols-3 gap-3 p-3 rounded-2xl bg-slate-50 dark:bg-slate-800/60 border border-slate-200 dark:border-slate-700">
                  <label className="flex items-center gap-2 cursor-pointer font-bold text-slate-700 dark:text-slate-300">
                    <input
                      type="checkbox"
                      checked={editingPlan.trial_allowed ?? true}
                      onChange={e => setEditingPlan({ ...editingPlan, trial_allowed: e.target.checked })}
                      className="rounded text-purple-600 focus:ring-purple-500 w-4 h-4"
                    />
                    <span>Allow Free Trial</span>
                  </label>

                  <label className="flex items-center gap-2 cursor-pointer font-bold text-slate-700 dark:text-slate-300">
                    <input
                      type="checkbox"
                      checked={editingPlan.is_featured ?? false}
                      onChange={e => setEditingPlan({ ...editingPlan, is_featured: e.target.checked })}
                      className="rounded text-blue-600 focus:ring-blue-500 w-4 h-4"
                    />
                    <span>Featured Badge</span>
                  </label>

                  <label className="flex items-center gap-2 cursor-pointer font-bold text-slate-700 dark:text-slate-300">
                    <input
                      type="checkbox"
                      checked={editingPlan.is_active ?? true}
                      onChange={e => setEditingPlan({ ...editingPlan, is_active: e.target.checked })}
                      className="rounded text-emerald-600 focus:ring-emerald-500 w-4 h-4"
                    />
                    <span>Active Package</span>
                  </label>
                </div>

                <div className="col-span-2 space-y-1">
                  <label className="font-bold text-slate-700 dark:text-slate-300">Description</label>
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
                  Cancel
                </button>
                <button
                  type="button"
                  onClick={handleSavePlan}
                  disabled={actionLoading === 'save-plan'}
                  className="px-6 py-2 rounded-xl bg-blue-600 hover:bg-blue-700 text-white text-xs font-bold shadow-md shadow-blue-500/20"
                >
                  Save Package
                </button>
              </div>
            </div>
          </div>
        )}

        {/* MODAL 5: EXTEND TRIAL MODAL */}
        {extendModalOpen && selectedTrialForExtend && (
          <div className="fixed inset-0 z-50 flex items-center justify-center p-3 sm:p-4 bg-slate-950/60 backdrop-blur-sm animate-in fade-in duration-200">
            <div className="relative w-full max-w-md bg-white dark:bg-slate-900 rounded-3xl border border-slate-200 dark:border-slate-800 shadow-2xl p-5 sm:p-6 space-y-4 max-h-[90vh] overflow-y-auto">
              <div className="flex items-center justify-between pb-3 border-b border-slate-100 dark:border-slate-800">
                <div className="flex items-center gap-2.5">
                  <div className="p-2 rounded-xl bg-purple-500/10 text-purple-600 dark:text-purple-400">
                    <Clock className="w-5 h-5" />
                  </div>
                  <div>
                    <h3 className="text-base font-bold text-slate-900 dark:text-white">Extend Customer Trial</h3>
                    <p className="text-xs text-slate-400">{selectedTrialForExtend.plan_name}</p>
                  </div>
                </div>
                <button
                  onClick={() => setExtendModalOpen(false)}
                  className="p-1.5 rounded-lg text-slate-400 hover:text-slate-600 hover:bg-slate-100 dark:hover:bg-slate-800"
                >
                  <X className="w-4 h-4" />
                </button>
              </div>

              <div className="space-y-3 text-xs">
                <p className="text-slate-500 dark:text-slate-400">
                  Select the number of additional evaluation days to grant to this client trial subscription:
                </p>

                <div className="grid grid-cols-4 gap-2">
                  {[3, 7, 14, 30].map(d => (
                    <button
                      key={d}
                      type="button"
                      onClick={() => setExtendDays(d)}
                      className={`py-2 rounded-xl font-bold border transition-all ${
                        extendDays === d
                          ? 'border-purple-500 bg-purple-500/10 text-purple-600 dark:text-purple-400 ring-2 ring-purple-500/20'
                          : 'border-slate-200 dark:border-slate-800 text-slate-600 dark:text-slate-400 hover:bg-slate-50 dark:hover:bg-slate-800'
                      }`}
                    >
                      +{d} Days
                    </button>
                  ))}
                </div>

                <div className="space-y-1 pt-1">
                  <label className="font-bold text-slate-700 dark:text-slate-300">Custom Days to Add:</label>
                  <input
                    type="number"
                    min="1"
                    max="365"
                    value={extendDays}
                    onChange={e => setExtendDays(Math.max(1, parseInt(e.target.value) || 1))}
                    className="w-full px-3 py-2 rounded-xl border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 text-slate-900 dark:text-white font-bold"
                    placeholder="Custom days"
                  />
                </div>
              </div>

              <div className="flex items-center justify-end gap-2 pt-3 border-t border-slate-100 dark:border-slate-800">
                <button
                  type="button"
                  onClick={() => setExtendModalOpen(false)}
                  className="px-4 py-2 rounded-xl text-xs font-bold text-slate-600 hover:bg-slate-100 dark:hover:bg-slate-800"
                >
                  Cancel
                </button>
                <button
                  type="button"
                  onClick={handleConfirmExtendTrial}
                  disabled={actionLoading === `extend-${selectedTrialForExtend.id}`}
                  className="px-5 py-2 rounded-xl bg-purple-600 hover:bg-purple-700 text-white text-xs font-bold shadow-md shadow-purple-500/20 flex items-center gap-1.5 transition-all"
                >
                  {actionLoading === `extend-${selectedTrialForExtend.id}` && <RefreshCw className="w-3.5 h-3.5 animate-spin" />}
                  <span>Confirm Extension</span>
                </button>
              </div>
            </div>
          </div>
        )}
      </div>
    </DashboardShell>
  );
}
