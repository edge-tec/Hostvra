'use client';

import React, { useState, useEffect } from 'react';
import Link from 'next/link';
import {
  Flame,
  Check,
  ArrowRight,
  Shield,
  Zap,
  Server,
  Globe,
  Database,
  Mail,
  Lock,
  Cpu,
  HardDrive,
  Terminal,
  Activity,
  Layers,
  ChevronDown,
  ChevronRight,
  ExternalLink,
  Sparkles,
  Users,
  CheckCircle2,
  Clock,
  HelpCircle,
  Code2,
  Boxes,
} from 'lucide-react';
import { apiFetch, HostingPlan } from '@/lib/api';

const FALLBACK_PLANS: HostingPlan[] = [
  {
    id: '10000000-0000-0000-0000-000000000001',
    name: 'Starter Cloud',
    slug: 'starter-cloud',
    description: 'Perfect for personal websites, blogs, and lightweight web projects.',
    tier: 'starter',
    price_monthly: 4.99,
    price_yearly: 49.99,
    currency: 'USD',
    trial_allowed: true,
    trial_days: 14,
    is_featured: false,
    cpu_limit: 1.0,
    ram_limit_mb: 1024,
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
      '2 MariaDB / MySQL Databases',
      '5 Custom Domain Mailboxes',
      'Free Let\'s Encrypt Wildcard SSL',
      'Multi-PHP (7.4, 8.1, 8.2, 8.3)',
      'Automated Weekly Backups',
    ],
    is_active: true,
    sort_order: 1,
  },
  {
    id: '10000000-0000-0000-0000-000000000002',
    name: 'Business Cloud',
    slug: 'business-cloud',
    description: 'High-speed cloud performance for growing businesses, ecommerce, and web agencies.',
    tier: 'business',
    price_monthly: 11.99,
    price_yearly: 119.99,
    currency: 'USD',
    trial_allowed: true,
    trial_days: 14,
    is_featured: true,
    cpu_limit: 2.0,
    ram_limit_mb: 2048,
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
      '10 MariaDB & PostgreSQL Databases',
      '25 Custom Domain Mailboxes',
      'Free Let\'s Encrypt Wildcard SSL',
      'Node.js & Python Application Runner',
      'ModSecurity WAF & Anti-DDoS',
      'Automated Daily Offsite Backups',
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
    trial_allowed: false,
    trial_days: 0,
    is_featured: false,
    cpu_limit: 4.0,
    ram_limit_mb: 4096,
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
    trial_allowed: false,
    trial_days: 0,
    is_featured: false,
    cpu_limit: 8.0,
    ram_limit_mb: 16384,
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

const FAQS = [
  {
    q: 'How does the 14-day free hosting trial work?',
    a: 'You can register in under 60 seconds without entering a credit card. Your hosting account, DNS zone, and full control panel access are provisioned automatically. You get complete access to test websites, databases, PHP/Node.js stacks, and mail services for 14 days.',
  },
  {
    q: 'Can I host my own custom domains and get free SSL certificates?',
    a: 'Yes! You can point any domain or subdomain (e.g. yourbrand.com) to your Hostvra server. Our platform integrates automated Let\'s Encrypt ACME with zero-touch issuance and auto-renewal before expiration.',
  },
  {
    q: 'Does Hostvra include in-house email routing with webmail?',
    a: 'Absolutely. Hostvra runs an enterprise-grade mail subsystem powered by Postfix (MTA), Dovecot (IMAP/POP3), and Rspamd spam scoring. You get built-in webmail, DKIM/SPF/DMARC automated signing, and IMAP support for Outlook and Apple Mail.',
  },
  {
    q: 'Can I run PHP, Node.js, Python, and WordPress on the same server?',
    a: 'Yes. Hostvra features multi-version PHP isolation (7.4 through 8.3) with independent PHP-FPM pools, automated Node.js process orchestration, Python WSGI/ASGI proxies, and 1-click WordPress installation with Redis object caching.',
  },
  {
    q: 'Is my data isolated from other accounts on the server?',
    a: 'Security is paramount. Every customer account runs inside a hardened Linux Chroot boundary with kernel Cgroups resource throttling. Users can never traverse above their home directory or affect neighboring accounts.',
  },
  {
    q: 'Can I upgrade or downgrade my hosting plan at any time?',
    a: 'Yes. You can seamlessly scale up your resources (CPU, RAM, storage, website quota) from your billing dashboard at any time. Storage and resource quotas adjust instantly with zero downtime.',
  },
];

export default function LandingPage() {
  const [plans, setPlans] = useState<HostingPlan[]>(FALLBACK_PLANS);
  const [billingCycle, setBillingCycle] = useState<'monthly' | 'yearly'>('monthly');
  const [openFaq, setOpenFaq] = useState<number | null>(0);
  const [activePreviewTab, setActivePreviewTab] = useState<'vhosts' | 'databases' | 'mail' | 'terminal'>('vhosts');

  useEffect(() => {
    async function loadPlans() {
      try {
        const res = await apiFetch<HostingPlan[]>('/api/v1/billing/plans');
        if (res && Array.isArray(res) && res.length > 0) {
          setPlans(res);
        }
      } catch (err) {
        console.warn('Using default hosting catalog fallback:', err);
      }
    }
    loadPlans();
  }, []);

  return (
    <div className="min-h-screen bg-slate-50 text-slate-800 selection:bg-indigo-600 selection:text-white overflow-x-hidden font-sans">
      {/* Navigation Header */}
      <header className="sticky top-0 z-50 backdrop-blur-xl bg-white/95 border-b border-slate-200/90 shadow-sm transition-all">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 h-20 flex items-center justify-between">
          <Link href="/" className="flex items-center gap-3 group">
            <div className="w-10 h-10 rounded-xl bg-indigo-600 flex items-center justify-center shadow-md shadow-indigo-600/20 group-hover:scale-105 transition-transform">
              <Flame className="w-5 h-5 text-white fill-white" />
            </div>
            <div className="flex flex-col">
              <span className="text-xl font-black tracking-tight text-slate-950 flex items-center gap-1.5">
                HOSTVRA
                <span className="text-[10px] font-bold px-1.5 py-0.5 rounded bg-indigo-100 text-indigo-700 border border-indigo-200 uppercase tracking-widest">
                  Cloud
                </span>
              </span>
              <span className="text-[10px] text-slate-500 font-semibold tracking-wider uppercase -mt-0.5">
                Hosting Control Panel
              </span>
            </div>
          </Link>

          <nav className="hidden md:flex items-center gap-8 text-sm font-semibold text-slate-600">
            <a href="#features" className="hover:text-indigo-600 transition-colors">
              Features
            </a>
            <a href="#pricing" className="hover:text-indigo-600 transition-colors">
              Hosting Plans
            </a>
            <a href="#architecture" className="hover:text-indigo-600 transition-colors">
              Technology
            </a>
            <a href="#faq" className="hover:text-indigo-600 transition-colors">
              FAQ
            </a>
            <Link href="/webmail" className="hover:text-indigo-600 transition-colors flex items-center gap-1">
              Webmail <ExternalLink className="w-3 h-3 text-slate-400" />
            </Link>
          </nav>

          <div className="flex items-center gap-3">
            <Link
              href="/login"
              className="px-4 py-2 text-sm font-semibold text-slate-700 hover:text-indigo-600 transition-colors"
            >
              Sign In
            </Link>
            <Link
              href="/register?trial=true"
              className="px-5 py-2.5 rounded-xl bg-indigo-600 hover:bg-indigo-700 text-white text-sm font-bold shadow-md shadow-indigo-600/20 hover:shadow-indigo-600/30 transition-all flex items-center gap-2 group"
            >
              Start Free Trial
              <ArrowRight className="w-4 h-4 group-hover:translate-x-0.5 transition-transform" />
            </Link>
          </div>
        </div>
      </header>

      {/* Hero Section */}
      <section className="relative pt-16 pb-20 md:pt-24 md:pb-28 bg-gradient-to-b from-white via-slate-50 to-slate-100/70 border-b border-slate-200/80">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 text-center">
          {/* Badge */}
          <div className="inline-flex items-center gap-2 px-4 py-1.5 rounded-full bg-indigo-50 border border-indigo-200/80 shadow-sm mb-6">
            <Sparkles className="w-4 h-4 text-indigo-600" />
            <span className="text-xs font-bold text-indigo-900">
              Enterprise Hosting Control Panel • 99.99% Guaranteed SLA
            </span>
          </div>

          {/* Heading */}
          <h1 className="text-4xl sm:text-6xl lg:text-7xl font-black tracking-tight text-slate-950 max-w-5xl mx-auto leading-[1.12]">
            Powerful Cloud Hosting.{' '}
            <span className="text-transparent bg-clip-text bg-gradient-to-r from-indigo-600 via-purple-600 to-indigo-700">
              Effortless Management.
            </span>
          </h1>

          {/* Subtitle */}
          <p className="mt-6 text-lg sm:text-xl text-slate-600 max-w-3xl mx-auto font-normal leading-relaxed">
            Deploy high-traffic websites, manage isolated MariaDB & PostgreSQL databases, route enterprise emails, and protect your
            infrastructure with automated WAF—all from one lightning-fast control panel.
          </p>

          {/* CTAs */}
          <div className="mt-10 flex flex-col sm:flex-row items-center justify-center gap-4">
            <Link
              href="/register?trial=true"
              className="w-full sm:w-auto px-8 py-4 rounded-xl bg-indigo-600 hover:bg-indigo-700 text-white font-bold text-base shadow-lg shadow-indigo-600/25 hover:shadow-indigo-600/35 transition-all flex items-center justify-center gap-2.5 group"
            >
              Start 14-Day Free Trial
              <ArrowRight className="w-5 h-5 group-hover:translate-x-1 transition-transform" />
            </Link>
            <a
              href="#pricing"
              className="w-full sm:w-auto px-8 py-4 rounded-xl bg-white hover:bg-slate-50 border border-slate-300 text-slate-800 font-semibold text-base shadow-sm hover:shadow transition-all flex items-center justify-center gap-2"
            >
              Explore Hosting Plans
              <ChevronDown className="w-4 h-4 text-slate-500" />
            </a>
          </div>

          {/* Trust Indicators */}
          <div className="mt-12 flex flex-wrap items-center justify-center gap-6 sm:gap-10 text-xs sm:text-sm font-semibold text-slate-700">
            <div className="flex items-center gap-2">
              <CheckCircle2 className="w-4 h-4 text-emerald-600" />
              <span>No Credit Card Required</span>
            </div>
            <div className="flex items-center gap-2">
              <CheckCircle2 className="w-4 h-4 text-emerald-600" />
              <span>1-Click WordPress & Laravel</span>
            </div>
            <div className="flex items-center gap-2">
              <CheckCircle2 className="w-4 h-4 text-emerald-600" />
              <span>Pure NVMe Gen4 Storage</span>
            </div>
            <div className="flex items-center gap-2">
              <CheckCircle2 className="w-4 h-4 text-emerald-600" />
              <span>Instant Automated Provisioning</span>
            </div>
          </div>

          {/* Interactive Live Control Panel Preview Mockup */}
          <div className="mt-14 sm:mt-16 max-w-5xl mx-auto rounded-2xl bg-white border border-slate-200/90 shadow-2xl p-2 sm:p-4">
            <div className="flex items-center justify-between px-4 py-3 border-b border-slate-200 bg-slate-100/80 rounded-t-xl">
              <div className="flex items-center gap-2">
                <div className="w-3 h-3 rounded-full bg-rose-500" />
                <div className="w-3 h-3 rounded-full bg-amber-500" />
                <div className="w-3 h-3 rounded-full bg-emerald-500" />
                <span className="ml-2 text-xs font-mono text-slate-600 font-semibold">hostvra-control-plane:~/production</span>
              </div>
              <div className="flex items-center gap-2">
                <button
                  onClick={() => setActivePreviewTab('vhosts')}
                  className={`px-3 py-1.5 text-xs rounded-lg font-bold transition-all ${
                    activePreviewTab === 'vhosts'
                      ? 'bg-indigo-600 text-white shadow-sm'
                      : 'text-slate-600 hover:text-slate-950 hover:bg-slate-200/70'
                  }`}
                >
                  Websites
                </button>
                <button
                  onClick={() => setActivePreviewTab('databases')}
                  className={`px-3 py-1.5 text-xs rounded-lg font-bold transition-all ${
                    activePreviewTab === 'databases'
                      ? 'bg-indigo-600 text-white shadow-sm'
                      : 'text-slate-600 hover:text-slate-950 hover:bg-slate-200/70'
                  }`}
                >
                  Databases
                </button>
                <button
                  onClick={() => setActivePreviewTab('mail')}
                  className={`px-3 py-1.5 text-xs rounded-lg font-bold transition-all ${
                    activePreviewTab === 'mail'
                      ? 'bg-indigo-600 text-white shadow-sm'
                      : 'text-slate-600 hover:text-slate-950 hover:bg-slate-200/70'
                  }`}
                >
                  Mail Server
                </button>
                <button
                  onClick={() => setActivePreviewTab('terminal')}
                  className={`px-3 py-1.5 text-xs rounded-lg font-bold transition-all ${
                    activePreviewTab === 'terminal'
                      ? 'bg-indigo-600 text-white shadow-sm'
                      : 'text-slate-600 hover:text-slate-950 hover:bg-slate-200/70'
                  }`}
                >
                  Terminal
                </button>
              </div>
            </div>

            <div className="p-6 bg-slate-50/60 rounded-b-xl text-left">
              {activePreviewTab === 'vhosts' && (
                <div className="space-y-4">
                  <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                    <div className="p-4 rounded-xl bg-white border border-slate-200 shadow-sm">
                      <div className="flex items-center justify-between text-xs text-slate-500 font-medium">
                        <span>Active Websites</span>
                        <Globe className="w-4 h-4 text-indigo-600" />
                      </div>
                      <div className="text-2xl font-bold text-slate-900 mt-2">12 Live VHosts</div>
                      <div className="text-xs text-emerald-600 mt-1 flex items-center gap-1 font-semibold">
                        <span className="w-1.5 h-1.5 rounded-full bg-emerald-500 animate-pulse" />
                        Nginx HTTP/3 & TLS 1.3 Active
                      </div>
                    </div>
                    <div className="p-4 rounded-xl bg-white border border-slate-200 shadow-sm">
                      <div className="flex items-center justify-between text-xs text-slate-500 font-medium">
                        <span>PHP-FPM Engines</span>
                        <Code2 className="w-4 h-4 text-purple-600" />
                      </div>
                      <div className="text-2xl font-bold text-slate-900 mt-2">PHP 8.2 & 8.3</div>
                      <div className="text-xs text-indigo-700 mt-1 font-semibold">Multi-Version Pool Isolated</div>
                    </div>
                    <div className="p-4 rounded-xl bg-white border border-slate-200 shadow-sm">
                      <div className="flex items-center justify-between text-xs text-slate-500 font-medium">
                        <span>SSL Certificates</span>
                        <Lock className="w-4 h-4 text-emerald-600" />
                      </div>
                      <div className="text-2xl font-bold text-slate-900 mt-2">100% Protected</div>
                      <div className="text-xs text-emerald-600 mt-1 font-semibold">Automated Let&apos;s Encrypt Renewal</div>
                    </div>
                  </div>

                  <div className="border border-slate-200 rounded-xl overflow-hidden bg-white shadow-sm">
                    <table className="w-full text-xs text-left">
                      <thead className="bg-slate-100 text-slate-700 border-b border-slate-200 font-bold">
                        <tr>
                          <th className="p-3">Domain</th>
                          <th className="p-3">Stack</th>
                          <th className="p-3">SSL Status</th>
                          <th className="p-3">Traffic (24h)</th>
                          <th className="p-3">Status</th>
                        </tr>
                      </thead>
                      <tbody className="divide-y divide-slate-200 text-slate-700">
                        <tr>
                          <td className="p-3 font-bold text-slate-900">store.example.com</td>
                          <td className="p-3 font-medium">PHP 8.3 / Nginx</td>
                          <td className="p-3 text-emerald-700 font-bold">Valid (82 Days)</td>
                          <td className="p-3 font-medium">142.8 GB</td>
                          <td className="p-3">
                            <span className="px-2 py-0.5 rounded bg-emerald-100 text-emerald-800 font-bold">
                              ONLINE
                            </span>
                          </td>
                        </tr>
                        <tr>
                          <td className="p-3 font-bold text-slate-900">api.saasapp.io</td>
                          <td className="p-3 font-medium">Node.js SSR</td>
                          <td className="p-3 text-emerald-700 font-bold">Valid (68 Days)</td>
                          <td className="p-3 font-medium">582.4 GB</td>
                          <td className="p-3">
                            <span className="px-2 py-0.5 rounded bg-emerald-100 text-emerald-800 font-bold">
                              ONLINE
                            </span>
                          </td>
                        </tr>
                      </tbody>
                    </table>
                  </div>
                </div>
              )}

              {activePreviewTab === 'databases' && (
                <div className="space-y-4">
                  <div className="p-4 rounded-xl bg-white border border-slate-200 shadow-sm flex items-center justify-between">
                    <div>
                      <div className="text-sm font-bold text-slate-900">MariaDB & PostgreSQL Database Engine</div>
                      <div className="text-xs text-slate-500 mt-0.5 font-medium">
                        InnoDB Buffer Pool: 85% Hit Ratio • Connection Pool: Active
                      </div>
                    </div>
                    <span className="px-3 py-1 rounded-full bg-emerald-100 text-emerald-800 text-xs font-bold">
                      HEALTHY
                    </span>
                  </div>
                  <div className="grid grid-cols-2 gap-4 text-xs">
                    <div className="p-3 bg-white rounded-lg border border-slate-200 shadow-sm">
                      <div className="text-slate-500 font-medium">Query Latency:</div>
                      <div className="text-emerald-700 font-bold mt-1">99.4% Queries &lt; 0.8ms</div>
                    </div>
                    <div className="p-3 bg-white rounded-lg border border-slate-200 shadow-sm">
                      <div className="text-slate-500 font-medium">Automated Backup:</div>
                      <div className="text-indigo-700 font-bold mt-1">Snapshot Synchronized Daily</div>
                    </div>
                  </div>
                </div>
              )}

              {activePreviewTab === 'mail' && (
                <div className="space-y-4">
                  <div className="grid grid-cols-3 gap-3 text-center">
                    <div className="p-4 bg-white rounded-xl border border-slate-200 shadow-sm">
                      <div className="text-xs text-slate-500 font-semibold">Postfix SMTP</div>
                      <div className="text-emerald-700 font-bold mt-1 text-sm">Port 587/465 Active</div>
                    </div>
                    <div className="p-4 bg-white rounded-xl border border-slate-200 shadow-sm">
                      <div className="text-xs text-slate-500 font-semibold">Dovecot IMAP</div>
                      <div className="text-emerald-700 font-bold mt-1 text-sm">Port 993 Active</div>
                    </div>
                    <div className="p-4 bg-white rounded-xl border border-slate-200 shadow-sm">
                      <div className="text-xs text-slate-500 font-semibold">DKIM / SPF / DMARC</div>
                      <div className="text-emerald-700 font-bold mt-1 text-sm">10/10 Deliverability</div>
                    </div>
                  </div>
                </div>
              )}

              {activePreviewTab === 'terminal' && (
                <div className="font-mono text-xs text-slate-200 space-y-1 bg-slate-950 p-4 rounded-xl border border-slate-900 shadow-inner">
                  <div className="text-indigo-400">root@hostvra-cluster:~# systemctl status hostvra-agent nginx postfix</div>
                  <div className="text-emerald-400">● hostvra-agent.service - Hostvra Server Agent (Active: running)</div>
                  <div className="text-emerald-400">● nginx.service - A high performance web server (Active: running)</div>
                  <div className="text-emerald-400">● postfix.service - Postfix Mail Transport Agent (Active: running)</div>
                  <div className="text-slate-500 mt-2">root@hostvra-cluster:~# hostvra status --all-vhosts</div>
                  <div className="text-slate-300">[OK] 12 VHosts verified. Zero configuration anomalies detected.</div>
                </div>
              )}
            </div>
          </div>
        </div>
      </section>

      {/* Hosting Features Grid */}
      <section id="features" className="py-20 md:py-28 bg-white border-b border-slate-200">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="text-center max-w-3xl mx-auto">
            <h2 className="text-xs font-bold uppercase tracking-widest text-indigo-600 mb-2">
              Next-Generation Stack
            </h2>
            <p className="text-3xl sm:text-4xl font-extrabold text-slate-950 tracking-tight">
              Engineered for Extreme Speed, Security & Scalability
            </p>
            <p className="mt-4 text-base text-slate-600">
              Hostvra unites enterprise web servers, database clusters, mail servers, and modern developer tooling into
              a seamless, friction-free control panel.
            </p>
          </div>

          <div className="mt-16 grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
            <div className="p-6 rounded-2xl bg-slate-50 border border-slate-200 hover:border-indigo-500/50 hover:bg-white transition-all hover:shadow-lg group">
              <div className="w-12 h-12 rounded-xl bg-indigo-100 text-indigo-600 flex items-center justify-center mb-5 group-hover:scale-110 transition-transform">
                <Globe className="w-6 h-6" />
              </div>
              <h3 className="text-lg font-bold text-slate-900">Nginx & OpenLiteSpeed</h3>
              <p className="mt-2 text-sm text-slate-600 leading-relaxed">
                Supercharged web servers equipped with HTTP/3, Brotli compression, and Redis object cache for instant page
                loads.
              </p>
            </div>

            <div className="p-6 rounded-2xl bg-slate-50 border border-slate-200 hover:border-purple-500/50 hover:bg-white transition-all hover:shadow-lg group">
              <div className="w-12 h-12 rounded-xl bg-purple-100 text-purple-600 flex items-center justify-center mb-5 group-hover:scale-110 transition-transform">
                <Code2 className="w-6 h-6" />
              </div>
              <h3 className="text-lg font-bold text-slate-900">Multi-PHP 7.4 to 8.3</h3>
              <p className="mt-2 text-sm text-slate-600 leading-relaxed">
                Seamlessly run multiple PHP versions per domain with isolated PHP-FPM pools, custom php.ini editor, and
                extensions.
              </p>
            </div>

            <div className="p-6 rounded-2xl bg-slate-50 border border-slate-200 hover:border-blue-500/50 hover:bg-white transition-all hover:shadow-lg group">
              <div className="w-12 h-12 rounded-xl bg-blue-100 text-blue-600 flex items-center justify-center mb-5 group-hover:scale-110 transition-transform">
                <Boxes className="w-6 h-6" />
              </div>
              <h3 className="text-lg font-bold text-slate-900">Node.js & Python</h3>
              <p className="mt-2 text-sm text-slate-600 leading-relaxed">
                Deploy Next.js, Express, Django, and FastAPI apps effortlessly via automated reverse-proxy and process
                managers.
              </p>
            </div>

            <div className="p-6 rounded-2xl bg-slate-50 border border-slate-200 hover:border-emerald-500/50 hover:bg-white transition-all hover:shadow-lg group">
              <div className="w-12 h-12 rounded-xl bg-emerald-100 text-emerald-600 flex items-center justify-center mb-5 group-hover:scale-110 transition-transform">
                <Database className="w-6 h-6" />
              </div>
              <h3 className="text-lg font-bold text-slate-900">MySQL & PostgreSQL</h3>
              <p className="mt-2 text-sm text-slate-600 leading-relaxed">
                High-performance relational databases with web-based phpMyAdmin integration, remote host allowlists, and
                quotas.
              </p>
            </div>

            <div className="p-6 rounded-2xl bg-slate-50 border border-slate-200 hover:border-rose-500/50 hover:bg-white transition-all hover:shadow-lg group">
              <div className="w-12 h-12 rounded-xl bg-rose-100 text-rose-600 flex items-center justify-center mb-5 group-hover:scale-110 transition-transform">
                <Mail className="w-6 h-6" />
              </div>
              <h3 className="text-lg font-bold text-slate-900">In-House Mail & Webmail</h3>
              <p className="mt-2 text-sm text-slate-600 leading-relaxed">
                Independent mail platform (Postfix & Dovecot) with Rspamd spam filtering, automated DKIM/DMARC, and
                webmail.
              </p>
            </div>

            <div className="p-6 rounded-2xl bg-slate-50 border border-slate-200 hover:border-amber-500/50 hover:bg-white transition-all hover:shadow-lg group">
              <div className="w-12 h-12 rounded-xl bg-amber-100 text-amber-700 flex items-center justify-center mb-5 group-hover:scale-110 transition-transform">
                <Lock className="w-6 h-6" />
              </div>
              <h3 className="text-lg font-bold text-slate-900">Automated Free SSL</h3>
              <p className="mt-2 text-sm text-slate-600 leading-relaxed">
                1-Click Let&apos;s Encrypt SSL certificates for all root domains and subdomains with automated zero-touch
                renewal.
              </p>
            </div>

            <div className="p-6 rounded-2xl bg-slate-50 border border-slate-200 hover:border-cyan-500/50 hover:bg-white transition-all hover:shadow-lg group">
              <div className="w-12 h-12 rounded-xl bg-cyan-100 text-cyan-700 flex items-center justify-center mb-5 group-hover:scale-110 transition-transform">
                <Shield className="w-6 h-6" />
              </div>
              <h3 className="text-lg font-bold text-slate-900">Enterprise WAF & Jail</h3>
              <p className="mt-2 text-sm text-slate-600 leading-relaxed">
                Chroot file system isolation, Fail2ban brute-force protection, UFW firewall, and ModSecurity rule sets.
              </p>
            </div>

            <div className="p-6 rounded-2xl bg-slate-50 border border-slate-200 hover:border-indigo-500/50 hover:bg-white transition-all hover:shadow-lg group">
              <div className="w-12 h-12 rounded-xl bg-indigo-100 text-indigo-600 flex items-center justify-center mb-5 group-hover:scale-110 transition-transform">
                <Terminal className="w-6 h-6" />
              </div>
              <h3 className="text-lg font-bold text-slate-900">Web Terminal & Backups</h3>
              <p className="mt-2 text-sm text-slate-600 leading-relaxed">
                Secure browser-based PTY shell, drag-and-drop file manager, scheduled cron jobs, and 1-click cloud
                backups.
              </p>
            </div>
          </div>
        </div>
      </section>

      {/* Dynamic Hosting Plans & Pricing */}
      <section id="pricing" className="py-20 md:py-28 bg-slate-100/60 border-b border-slate-200">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="text-center max-w-3xl mx-auto">
            <h2 className="text-xs font-bold uppercase tracking-widest text-indigo-600 mb-2">
              Transparent Cloud Pricing
            </h2>
            <p className="text-3xl sm:text-5xl font-black text-slate-950 tracking-tight">
              Flexible Plans for Every Stage of Growth
            </p>
            <p className="mt-4 text-base text-slate-600">
              Choose the package that suits your projects. Try Starter or Business free for 14 days with zero risk.
            </p>

            {/* Monthly / Yearly Toggle */}
            <div className="mt-8 inline-flex items-center p-1.5 rounded-2xl bg-white border border-slate-200 shadow-sm">
              <button
                onClick={() => setBillingCycle('monthly')}
                className={`px-5 py-2.5 rounded-xl text-xs sm:text-sm font-bold transition-all ${
                  billingCycle === 'monthly'
                    ? 'bg-indigo-600 text-white shadow-sm'
                    : 'text-slate-600 hover:text-slate-950'
                }`}
              >
                Monthly Billing
              </button>
              <button
                onClick={() => setBillingCycle('yearly')}
                className={`px-5 py-2.5 rounded-xl text-xs sm:text-sm font-bold transition-all flex items-center gap-1.5 ${
                  billingCycle === 'yearly'
                    ? 'bg-indigo-600 text-white shadow-sm'
                    : 'text-slate-600 hover:text-slate-950'
                }`}
              >
                Yearly Billing
                <span className="text-[10px] uppercase font-black px-1.5 py-0.5 rounded bg-emerald-100 text-emerald-800 border border-emerald-200">
                  Save ~17%
                </span>
              </button>
            </div>
          </div>

          {/* Pricing Cards Grid */}
          <div className="mt-16 grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-8">
            {plans.map((plan) => {
              const price = billingCycle === 'yearly' ? plan.price_yearly / 12 : plan.price_monthly;
              const hasTrial = plan.trial_allowed;

              return (
                <div
                  key={plan.id}
                  className={`relative flex flex-col p-8 rounded-3xl bg-white border transition-all duration-300 hover:shadow-xl ${
                    plan.is_featured
                      ? 'border-2 border-indigo-600 shadow-xl shadow-indigo-600/10 ring-4 ring-indigo-50'
                      : 'border-slate-200/90 hover:border-slate-300 shadow-sm'
                  }`}
                >
                  {plan.is_featured && (
                    <div className="absolute -top-3.5 left-1/2 -translate-x-1/2 px-4 py-1 rounded-full bg-indigo-600 text-[11px] font-black uppercase tracking-wider text-white shadow-md">
                      Most Popular
                    </div>
                  )}

                  <div className="flex-1">
                    <h3 className="text-xl font-black text-slate-900">{plan.name}</h3>
                    <p className="mt-2 text-xs text-slate-600 leading-relaxed min-h-[36px]">{plan.description}</p>

                    <div className="mt-6 flex items-baseline gap-1">
                      <span className="text-4xl sm:text-5xl font-black text-slate-950 tracking-tight">
                        ${price.toFixed(2)}
                      </span>
                      <span className="text-xs text-slate-500 font-bold">/ month</span>
                    </div>
                    {billingCycle === 'yearly' && (
                      <div className="text-[11px] text-emerald-700 font-bold mt-1">
                        Billed annually (${plan.price_yearly.toFixed(2)} / year)
                      </div>
                    )}

                    {hasTrial && (
                      <div className="mt-4 px-3 py-1.5 rounded-xl bg-indigo-50 border border-indigo-200 text-indigo-800 text-xs font-bold flex items-center gap-1.5">
                        <Sparkles className="w-3.5 h-3.5 text-indigo-600" />
                        <span>Includes {plan.trial_days || 14}-Day Free Trial</span>
                      </div>
                    )}

                    {/* Limits Specs */}
                    <div className="mt-6 pt-6 border-t border-slate-100 space-y-3 text-xs">
                      <div className="flex items-center justify-between">
                        <span className="text-slate-500 font-medium">Websites</span>
                        <span className="font-bold text-slate-900">{plan.max_websites} Hosted</span>
                      </div>
                      <div className="flex items-center justify-between">
                        <span className="text-slate-500 font-medium">NVMe SSD Storage</span>
                        <span className="font-bold text-slate-900">{(plan.disk_space_mb / 1024).toFixed(0)} GB</span>
                      </div>
                      <div className="flex items-center justify-between">
                        <span className="text-slate-500 font-medium">Monthly Bandwidth</span>
                        <span className="font-bold text-slate-900">{(plan.bandwidth_mb / 1024).toFixed(0)} GB</span>
                      </div>
                      <div className="flex items-center justify-between">
                        <span className="text-slate-500 font-medium">Databases</span>
                        <span className="font-bold text-slate-900">{plan.max_databases} DBs</span>
                      </div>
                      <div className="flex items-center justify-between">
                        <span className="text-slate-500 font-medium">Mailboxes</span>
                        <span className="font-bold text-slate-900">{plan.max_mailboxes} Inboxes</span>
                      </div>
                    </div>

                    {/* Features List */}
                    <div className="mt-6 pt-6 border-t border-slate-100 space-y-2.5">
                      {plan.features.slice(0, 6).map((feat, idx) => (
                        <div key={idx} className="flex items-start gap-2.5 text-xs text-slate-700">
                          <Check className="w-4 h-4 text-emerald-600 flex-shrink-0 mt-0.5" />
                          <span className="font-medium">{feat}</span>
                        </div>
                      ))}
                    </div>
                  </div>

                  <div className="mt-8 pt-6 border-t border-slate-100">
                    <Link
                      href={`/register?plan=${plan.slug}&cycle=${billingCycle}${hasTrial ? '&trial=true' : ''}`}
                      className={`w-full py-3.5 px-4 rounded-xl text-center text-xs font-bold transition-all block shadow-sm ${
                        plan.is_featured
                          ? 'bg-indigo-600 hover:bg-indigo-700 text-white shadow-indigo-600/20'
                          : hasTrial
                          ? 'bg-indigo-50 hover:bg-indigo-100 text-indigo-700 border border-indigo-200'
                          : 'bg-slate-900 hover:bg-slate-800 text-white'
                      }`}
                    >
                      {hasTrial ? `Start ${plan.trial_days || 14}-Day Free Trial` : 'Choose Plan'}
                    </Link>
                  </div>
                </div>
              );
            })}
          </div>
        </div>
      </section>

      {/* Architecture & Zero-Trust Security Section */}
      <section id="architecture" className="py-20 md:py-28 bg-white border-b border-slate-200">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-12 items-center">
            <div>
              <div className="inline-flex items-center gap-2 px-3 py-1 rounded-full bg-indigo-50 border border-indigo-200 text-indigo-700 text-xs font-bold uppercase tracking-wider mb-4">
                Architecture & Security
              </div>
              <h2 className="text-3xl sm:text-4xl font-black text-slate-950 tracking-tight">
                Complete Isolation. Zero Shared Risk.
              </h2>
              <p className="mt-4 text-slate-600 text-sm sm:text-base leading-relaxed">
                Traditional shared hosting suffers from noisy neighbors and cross-account vulnerabilities. Hostvra
                replaces obsolete paradigms with Linux containerized isolation and Chroot environments.
              </p>

              <div className="mt-8 space-y-4 text-sm text-slate-700">
                <div className="flex items-start gap-3">
                  <div className="p-2 rounded-lg bg-indigo-50 text-indigo-600 mt-1 border border-indigo-100">
                    <Shield className="w-4 h-4" />
                  </div>
                  <div>
                    <strong className="text-slate-900 block font-bold">Chroot File Boundary</strong>
                    <span className="text-slate-600 text-xs">
                      Users can never traverse above their home directory. Path canonicalization blocks symlink escapes
                      and Zip Slip vulnerabilities.
                    </span>
                  </div>
                </div>

                <div className="flex items-start gap-3">
                  <div className="p-2 rounded-lg bg-purple-50 text-purple-600 mt-1 border border-purple-100">
                    <Cpu className="w-4 h-4" />
                  </div>
                  <div>
                    <strong className="text-slate-900 block font-bold">Dedicated CPU & RAM Cgroups</strong>
                    <span className="text-slate-600 text-xs">
                      Strict resource enforcement prevents a single traffic surge from affecting neighboring sites.
                    </span>
                  </div>
                </div>

                <div className="flex items-start gap-3">
                  <div className="p-2 rounded-lg bg-emerald-50 text-emerald-600 mt-1 border border-emerald-100">
                    <Activity className="w-4 h-4" />
                  </div>
                  <div>
                    <strong className="text-slate-900 block font-bold">Real-Time Threat Detection</strong>
                    <span className="text-slate-600 text-xs">
                      Automated Fail2ban, UFW dynamic rules, and ModSecurity block SQL injection, XSS, and brute-force
                      attacks instantly.
                    </span>
                  </div>
                </div>
              </div>
            </div>

            <div className="p-8 rounded-3xl bg-slate-50 border border-slate-200 shadow-xl relative">
              <div className="space-y-4">
                <div className="flex items-center justify-between p-4 rounded-xl bg-white border border-slate-200 shadow-sm">
                  <div className="flex items-center gap-3">
                    <div className="w-2.5 h-2.5 rounded-full bg-emerald-500 animate-ping" />
                    <div>
                      <div className="text-xs font-bold text-slate-900">Fail2ban Brute-Force Shield</div>
                      <div className="text-[11px] text-slate-500">Monitoring SSH, Postfix & HTTP</div>
                    </div>
                  </div>
                  <span className="text-xs font-mono text-emerald-700 font-bold">0 Breaches</span>
                </div>

                <div className="flex items-center justify-between p-4 rounded-xl bg-white border border-slate-200 shadow-sm">
                  <div className="flex items-center gap-3">
                    <div className="w-2.5 h-2.5 rounded-full bg-indigo-600" />
                    <div>
                      <div className="text-xs font-bold text-slate-900">Automated Let&apos;s Encrypt ACME</div>
                      <div className="text-[11px] text-slate-500">Zero-downtime certificate renewal</div>
                    </div>
                  </div>
                  <span className="text-xs font-mono text-indigo-700 font-bold">Auto-Renewed</span>
                </div>

                <div className="flex items-center justify-between p-4 rounded-xl bg-white border border-slate-200 shadow-sm">
                  <div className="flex items-center gap-3">
                    <div className="w-2.5 h-2.5 rounded-full bg-purple-600" />
                    <div>
                      <div className="text-xs font-bold text-slate-900">Postfix / Rspamd Mail Gateway</div>
                      <div className="text-[11px] text-slate-500">SPF, DKIM, DMARC cryptographically signed</div>
                    </div>
                  </div>
                  <span className="text-xs font-mono text-purple-700 font-bold">100% Inboxed</span>
                </div>
              </div>
            </div>
          </div>
        </div>
      </section>

      {/* Frequently Asked Questions */}
      <section id="faq" className="py-20 md:py-28 bg-slate-50 border-b border-slate-200">
        <div className="max-w-4xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="text-center">
            <h2 className="text-xs font-bold uppercase tracking-widest text-indigo-600 mb-2">Got Questions?</h2>
            <p className="text-3xl sm:text-4xl font-extrabold text-slate-950 tracking-tight">
              Frequently Asked Questions
            </p>
          </div>

          <div className="mt-12 space-y-4">
            {FAQS.map((faq, index) => {
              const isOpen = openFaq === index;
              return (
                <div
                  key={index}
                  className="rounded-2xl bg-white border border-slate-200/90 shadow-sm overflow-hidden transition-all"
                >
                  <button
                    onClick={() => setOpenFaq(isOpen ? null : index)}
                    className="w-full p-6 text-left flex items-center justify-between gap-4"
                  >
                    <span className="text-base font-bold text-slate-900">{faq.q}</span>
                    <ChevronDown
                      className={`w-5 h-5 text-slate-500 flex-shrink-0 transition-transform ${
                        isOpen ? 'rotate-180 text-indigo-600' : ''
                      }`}
                    />
                  </button>
                  {isOpen && (
                    <div className="px-6 pb-6 text-sm text-slate-600 leading-relaxed border-t border-slate-100 pt-4">
                      {faq.a}
                    </div>
                  )}
                </div>
              );
            })}
          </div>
        </div>
      </section>

      {/* Pre-Footer CTA */}
      <section className="py-20 bg-white relative overflow-hidden">
        <div className="max-w-5xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="relative rounded-3xl p-10 sm:p-16 bg-gradient-to-r from-indigo-600 via-indigo-700 to-purple-700 shadow-2xl text-center overflow-hidden">
            <h2 className="text-3xl sm:text-5xl font-black text-white tracking-tight">
              Ready to Upgrade Your Cloud Hosting?
            </h2>
            <p className="mt-4 text-base sm:text-lg text-indigo-100 max-w-2xl mx-auto font-medium">
              Launch your sites in seconds with automated SSL, isolated NVMe storage, and dedicated developer tooling.
            </p>

            <div className="mt-8 flex flex-col sm:flex-row items-center justify-center gap-4">
              <Link
                href="/register?trial=true"
                className="w-full sm:w-auto px-8 py-4 rounded-xl bg-white hover:bg-slate-100 text-indigo-900 font-extrabold text-sm sm:text-base shadow-xl transition-all"
              >
                Start 14-Day Free Trial
              </Link>
              <Link
                href="/login"
                className="w-full sm:w-auto px-8 py-4 rounded-xl bg-indigo-800/80 hover:bg-indigo-800 border border-indigo-400/50 text-white font-bold text-sm sm:text-base transition-all"
              >
                Sign In to Control Panel
              </Link>
            </div>
          </div>
        </div>
      </section>

      {/* Comprehensive Footer */}
      <footer className="border-t border-slate-200 bg-white py-16 text-xs text-slate-600">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 grid grid-cols-2 md:grid-cols-5 gap-8">
          <div className="col-span-2">
            <Link href="/" className="flex items-center gap-2.5">
              <div className="w-8 h-8 rounded-lg bg-indigo-600 flex items-center justify-center">
                <Flame className="w-4 h-4 text-white fill-white" />
              </div>
              <span className="text-lg font-black tracking-tight text-slate-950">HOSTVRA</span>
            </Link>
            <p className="mt-4 text-slate-600 leading-relaxed max-w-sm">
              Next-generation hosting control plane and server infrastructure manager. Built for high performance,
              security, and modern web applications.
            </p>
            <div className="mt-6 text-[11px] text-slate-500 font-medium">
              © {new Date().getFullYear()} Hostvra Inc. All rights reserved.
            </div>
          </div>

          <div>
            <h4 className="font-bold text-slate-900 uppercase tracking-wider mb-4">Hosting</h4>
            <ul className="space-y-2.5 font-medium">
              <li>
                <a href="#pricing" className="hover:text-indigo-600 transition-colors">
                  Web Hosting
                </a>
              </li>
              <li>
                <a href="#pricing" className="hover:text-indigo-600 transition-colors">
                  WordPress Cloud
                </a>
              </li>
              <li>
                <a href="#pricing" className="hover:text-indigo-600 transition-colors">
                  Node.js & Python
                </a>
              </li>
              <li>
                <a href="#pricing" className="hover:text-indigo-600 transition-colors">
                  Reseller Hosting
                </a>
              </li>
            </ul>
          </div>

          <div>
            <h4 className="font-bold text-slate-900 uppercase tracking-wider mb-4">Platform</h4>
            <ul className="space-y-2.5 font-medium">
              <li>
                <Link href="/login" className="hover:text-indigo-600 transition-colors">
                  Control Panel
                </Link>
              </li>
              <li>
                <Link href="/webmail" className="hover:text-indigo-600 transition-colors">
                  Webmail Client
                </Link>
              </li>
              <li>
                <a href="#features" className="hover:text-indigo-600 transition-colors">
                  In-House Mail
                </a>
              </li>
              <li>
                <a href="#architecture" className="hover:text-indigo-600 transition-colors">
                  WAF & Security
                </a>
              </li>
            </ul>
          </div>

          <div>
            <h4 className="font-bold text-slate-900 uppercase tracking-wider mb-4">Support</h4>
            <ul className="space-y-2.5 font-medium">
              <li>
                <Link href="/login" className="hover:text-indigo-600 transition-colors">
                  Help Desk
                </Link>
              </li>
              <li>
                <a href="#faq" className="hover:text-indigo-600 transition-colors">
                  Knowledgebase
                </a>
              </li>
              <li>
                <span className="inline-flex items-center gap-1.5 text-emerald-700 font-bold">
                  <span className="w-1.5 h-1.5 rounded-full bg-emerald-500 animate-pulse" />
                  Systems Operational
                </span>
              </li>
            </ul>
          </div>
        </div>
      </footer>
    </div>
  );
}
