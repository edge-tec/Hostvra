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
    q: 'How does the 14-day free trial work?',
    a: 'You can start your free trial on our Starter or Business Cloud plans instantly without entering a credit card. You get full access to all control panel features, website deployment, MySQL databases, and email hosting for 14 days.',
  },
  {
    q: 'Can I host WordPress, Laravel, or custom Node.js apps?',
    a: 'Yes! Hostvra includes native 1-click support for WordPress, Laravel, Node.js SSR/Express, Python WSGI/ASGI, and raw PHP with multiple switchable PHP versions (7.4 through 8.3).',
  },
  {
    q: 'Are email accounts and webmail included?',
    a: 'Yes, Hostvra features an in-house enterprise mail platform powered by Postfix and Dovecot with automated DKIM, SPF, and DMARC signing, plus modern webmail access for your team.',
  },
  {
    q: 'Can I upgrade or downgrade my plan later?',
    a: 'Absolutely. You can seamlessly upgrade your hosting package at any time from your customer billing dashboard with instant quota adjustments and automated prorated billing.',
  },
  {
    q: 'Is my data isolated from other users on the server?',
    a: 'Yes. Hostvra implements strict Linux Chroot jail and system user isolation. Every customer account is strictly quarantined with dedicated resource quotas (CPU, RAM, Disk).',
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
        if (res.success && res.data && res.data.length > 0) {
          setPlans(res.data);
        }
      } catch {
        // Fallback to initial plans
      }
    }
    loadPlans();
  }, []);

  return (
    <div className="min-h-screen bg-[#07090e] text-slate-100 selection:bg-indigo-500 selection:text-white overflow-x-hidden font-sans">
      {/* Dynamic Background Glows */}
      <div className="fixed inset-0 pointer-events-none z-0 overflow-hidden">
        <div className="absolute -top-40 left-1/2 -translate-x-1/2 w-[1000px] h-[600px] bg-gradient-to-b from-indigo-600/20 via-purple-600/10 to-transparent blur-[140px] rounded-full" />
        <div className="absolute top-[40%] -left-64 w-[600px] h-[600px] bg-blue-600/10 blur-[160px] rounded-full" />
        <div className="absolute top-[70%] -right-64 w-[600px] h-[600px] bg-indigo-500/10 blur-[160px] rounded-full" />
      </div>

      {/* Navigation Header */}
      <header className="sticky top-0 z-50 backdrop-blur-xl bg-[#07090e]/80 border-b border-slate-800/80 transition-all">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 h-20 flex items-center justify-between">
          <Link href="/" className="flex items-center gap-3 group">
            <div className="w-10 h-10 rounded-xl bg-gradient-to-tr from-indigo-600 via-indigo-500 to-purple-500 flex items-center justify-center shadow-lg shadow-indigo-500/25 group-hover:scale-105 transition-transform">
              <Flame className="w-5 h-5 text-white fill-white" />
            </div>
            <div className="flex flex-col">
              <span className="text-xl font-black tracking-tight text-white flex items-center gap-1.5">
                HOSTVRA
                <span className="text-[10px] font-bold px-1.5 py-0.5 rounded bg-indigo-500/20 text-indigo-400 border border-indigo-500/30 uppercase tracking-widest">
                  Cloud
                </span>
              </span>
              <span className="text-[10px] text-slate-400 font-medium tracking-wider uppercase -mt-0.5">
                Hosting Control Panel
              </span>
            </div>
          </Link>

          <nav className="hidden md:flex items-center gap-8 text-sm font-semibold text-slate-300">
            <a href="#features" className="hover:text-white transition-colors">
              Features
            </a>
            <a href="#pricing" className="hover:text-white transition-colors">
              Hosting Plans
            </a>
            <a href="#architecture" className="hover:text-white transition-colors">
              Technology
            </a>
            <a href="#faq" className="hover:text-white transition-colors">
              FAQ
            </a>
            <Link href="/webmail" className="hover:text-white transition-colors flex items-center gap-1">
              Webmail <ExternalLink className="w-3 h-3 text-slate-500" />
            </Link>
          </nav>

          <div className="flex items-center gap-3">
            <Link
              href="/login"
              className="px-4 py-2 text-sm font-semibold text-slate-300 hover:text-white transition-colors"
            >
              Sign In
            </Link>
            <Link
              href="/register?trial=true"
              className="px-5 py-2.5 rounded-xl bg-gradient-to-r from-indigo-600 to-purple-600 hover:from-indigo-500 hover:to-purple-500 text-white text-sm font-bold shadow-lg shadow-indigo-600/30 hover:shadow-indigo-600/50 transition-all flex items-center gap-2 group"
            >
              Start Free Trial
              <ArrowRight className="w-4 h-4 group-hover:translate-x-0.5 transition-transform" />
            </Link>
          </div>
        </div>
      </header>

      {/* Hero Section */}
      <section className="relative pt-24 pb-20 md:pt-32 md:pb-32 z-10">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 text-center">
          <div className="inline-flex items-center gap-2.5 px-4 py-1.5 rounded-full bg-slate-900/90 border border-indigo-500/30 shadow-inner mb-8 backdrop-blur-md">
            <Sparkles className="w-4 h-4 text-indigo-400" />
            <span className="text-xs font-semibold text-slate-300">
              Enterprise Hosting Control Panel • 99.99% Guaranteed SLA
            </span>
          </div>

          <h1 className="text-4xl sm:text-6xl lg:text-7xl font-black tracking-tight text-white max-w-5xl mx-auto leading-[1.1]">
            Powerful Cloud Hosting.{' '}
            <span className="text-transparent bg-clip-text bg-gradient-to-r from-indigo-400 via-purple-300 to-pink-400">
              Effortless Management.
            </span>
          </h1>

          <p className="mt-6 text-lg sm:text-xl text-slate-400 max-w-3xl mx-auto font-normal leading-relaxed">
            Deploy websites, manage isolated MySQL & PostgreSQL databases, route enterprise emails, and protect your
            infrastructure with automated WAF—all from one lightning-fast control panel.
          </p>

          <div className="mt-10 flex flex-col sm:flex-row items-center justify-center gap-4">
            <Link
              href="/register?trial=true"
              className="w-full sm:w-auto px-8 py-4 rounded-xl bg-gradient-to-r from-indigo-600 via-indigo-500 to-purple-600 hover:from-indigo-500 hover:to-purple-500 text-white font-bold text-base shadow-xl shadow-indigo-600/30 hover:shadow-indigo-600/50 transition-all flex items-center justify-center gap-2.5 group"
            >
              Start 14-Day Free Trial
              <ArrowRight className="w-5 h-5 group-hover:translate-x-1 transition-transform" />
            </Link>
            <a
              href="#pricing"
              className="w-full sm:w-auto px-8 py-4 rounded-xl bg-slate-900/80 hover:bg-slate-800/80 border border-slate-700/80 text-slate-200 font-semibold text-base backdrop-blur-md transition-all flex items-center justify-center gap-2"
            >
              Explore Hosting Plans
              <ChevronDown className="w-4 h-4 text-slate-400" />
            </a>
          </div>

          {/* Trust Indicators */}
          <div className="mt-12 flex flex-wrap items-center justify-center gap-6 sm:gap-10 text-xs sm:text-sm font-medium text-slate-400">
            <div className="flex items-center gap-2">
              <CheckCircle2 className="w-4 h-4 text-emerald-400" />
              <span>No Credit Card Required</span>
            </div>
            <div className="flex items-center gap-2">
              <CheckCircle2 className="w-4 h-4 text-emerald-400" />
              <span>1-Click WordPress & Laravel</span>
            </div>
            <div className="flex items-center gap-2">
              <CheckCircle2 className="w-4 h-4 text-emerald-400" />
              <span>Pure NVMe Gen4 Storage</span>
            </div>
            <div className="flex items-center gap-2">
              <CheckCircle2 className="w-4 h-4 text-emerald-400" />
              <span>Instant Automated Provisioning</span>
            </div>
          </div>

          {/* Interactive Live Control Panel Preview Mockup */}
          <div className="mt-16 sm:mt-20 max-w-5xl mx-auto rounded-2xl bg-slate-900/70 border border-slate-800 shadow-2xl p-2 sm:p-4 backdrop-blur-xl">
            <div className="flex items-center justify-between px-4 py-3 border-b border-slate-800/80 bg-slate-950/60 rounded-t-xl">
              <div className="flex items-center gap-2">
                <div className="w-3 h-3 rounded-full bg-rose-500/80" />
                <div className="w-3 h-3 rounded-full bg-amber-500/80" />
                <div className="w-3 h-3 rounded-full bg-emerald-500/80" />
                <span className="ml-2 text-xs font-mono text-slate-500">hostvra-control-plane:~/production</span>
              </div>
              <div className="flex items-center gap-2">
                <button
                  onClick={() => setActivePreviewTab('vhosts')}
                  className={`px-3 py-1 text-xs rounded-lg font-medium transition-all ${
                    activePreviewTab === 'vhosts'
                      ? 'bg-indigo-600/30 text-indigo-300 border border-indigo-500/40'
                      : 'text-slate-400 hover:text-white'
                  }`}
                >
                  Websites
                </button>
                <button
                  onClick={() => setActivePreviewTab('databases')}
                  className={`px-3 py-1 text-xs rounded-lg font-medium transition-all ${
                    activePreviewTab === 'databases'
                      ? 'bg-indigo-600/30 text-indigo-300 border border-indigo-500/40'
                      : 'text-slate-400 hover:text-white'
                  }`}
                >
                  Databases
                </button>
                <button
                  onClick={() => setActivePreviewTab('mail')}
                  className={`px-3 py-1 text-xs rounded-lg font-medium transition-all ${
                    activePreviewTab === 'mail'
                      ? 'bg-indigo-600/30 text-indigo-300 border border-indigo-500/40'
                      : 'text-slate-400 hover:text-white'
                  }`}
                >
                  Mail Server
                </button>
                <button
                  onClick={() => setActivePreviewTab('terminal')}
                  className={`px-3 py-1 text-xs rounded-lg font-medium transition-all ${
                    activePreviewTab === 'terminal'
                      ? 'bg-indigo-600/30 text-indigo-300 border border-indigo-500/40'
                      : 'text-slate-400 hover:text-white'
                  }`}
                >
                  Terminal
                </button>
              </div>
            </div>

            <div className="p-6 bg-slate-950/80 rounded-b-xl text-left">
              {activePreviewTab === 'vhosts' && (
                <div className="space-y-4">
                  <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                    <div className="p-4 rounded-xl bg-slate-900/60 border border-slate-800">
                      <div className="flex items-center justify-between text-xs text-slate-400">
                        <span>Active Websites</span>
                        <Globe className="w-4 h-4 text-indigo-400" />
                      </div>
                      <div className="text-2xl font-bold text-white mt-2">12 Live VHosts</div>
                      <div className="text-xs text-emerald-400 mt-1 flex items-center gap-1">
                        <span className="w-1.5 h-1.5 rounded-full bg-emerald-400 animate-pulse" />
                        Nginx HTTP/3 & TLS 1.3 Active
                      </div>
                    </div>
                    <div className="p-4 rounded-xl bg-slate-900/60 border border-slate-800">
                      <div className="flex items-center justify-between text-xs text-slate-400">
                        <span>PHP-FPM Engines</span>
                        <Code2 className="w-4 h-4 text-purple-400" />
                      </div>
                      <div className="text-2xl font-bold text-white mt-2">PHP 8.2 & 8.3</div>
                      <div className="text-xs text-indigo-400 mt-1">Multi-Version Pool Isolated</div>
                    </div>
                    <div className="p-4 rounded-xl bg-slate-900/60 border border-slate-800">
                      <div className="flex items-center justify-between text-xs text-slate-400">
                        <span>SSL Certificates</span>
                        <Lock className="w-4 h-4 text-emerald-400" />
                      </div>
                      <div className="text-2xl font-bold text-white mt-2">100% Protected</div>
                      <div className="text-xs text-emerald-400 mt-1">Automated Let&apos;s Encrypt Renewal</div>
                    </div>
                  </div>

                  <div className="border border-slate-800 rounded-xl overflow-hidden">
                    <table className="w-full text-xs text-left">
                      <thead className="bg-slate-900/80 text-slate-400 border-b border-slate-800">
                        <tr>
                          <th className="p-3">Domain</th>
                          <th className="p-3">Stack</th>
                          <th className="p-3">SSL Status</th>
                          <th className="p-3">Traffic (24h)</th>
                          <th className="p-3">Status</th>
                        </tr>
                      </thead>
                      <tbody className="divide-y divide-slate-800/60 text-slate-300">
                        <tr>
                          <td className="p-3 font-semibold text-white">store.example.com</td>
                          <td className="p-3">PHP 8.3 / Nginx</td>
                          <td className="p-3 text-emerald-400 font-medium">Valid (82 Days)</td>
                          <td className="p-3">142.8 GB</td>
                          <td className="p-3">
                            <span className="px-2 py-0.5 rounded bg-emerald-500/20 text-emerald-300 font-bold">
                              ONLINE
                            </span>
                          </td>
                        </tr>
                        <tr>
                          <td className="p-3 font-semibold text-white">api.saasapp.io</td>
                          <td className="p-3">Node.js SSR</td>
                          <td className="p-3 text-emerald-400 font-medium">Valid (68 Days)</td>
                          <td className="p-3">582.4 GB</td>
                          <td className="p-3">
                            <span className="px-2 py-0.5 rounded bg-emerald-500/20 text-emerald-300 font-bold">
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
                  <div className="p-4 rounded-xl bg-slate-900/60 border border-slate-800 flex items-center justify-between">
                    <div>
                      <div className="text-sm font-bold text-white">MariaDB & PostgreSQL Database Engine</div>
                      <div className="text-xs text-slate-400 mt-0.5">
                        InnoDB Buffer Pool: 85% Hit Ratio • Connection Pool: Active
                      </div>
                    </div>
                    <span className="px-3 py-1 rounded-full bg-emerald-500/20 text-emerald-400 text-xs font-bold">
                      HEALTHY
                    </span>
                  </div>
                  <div className="grid grid-cols-2 gap-4 text-xs font-mono">
                    <div className="p-3 bg-slate-900/40 rounded-lg border border-slate-800">
                      <div className="text-slate-500">Query Cache:</div>
                      <div className="text-emerald-400 font-bold mt-1">99.4% Latency &lt; 0.8ms</div>
                    </div>
                    <div className="p-3 bg-slate-900/40 rounded-lg border border-slate-800">
                      <div className="text-slate-500">Automated Backup:</div>
                      <div className="text-indigo-400 font-bold mt-1">Snapshot Synchronized Daily</div>
                    </div>
                  </div>
                </div>
              )}

              {activePreviewTab === 'mail' && (
                <div className="space-y-4">
                  <div className="grid grid-cols-3 gap-3 text-center">
                    <div className="p-3 bg-slate-900/50 rounded-xl border border-slate-800">
                      <div className="text-xs text-slate-400">Postfix SMTP</div>
                      <div className="text-emerald-400 font-bold mt-1">Port 587/465 Active</div>
                    </div>
                    <div className="p-3 bg-slate-900/50 rounded-xl border border-slate-800">
                      <div className="text-xs text-slate-400">Dovecot IMAP</div>
                      <div className="text-emerald-400 font-bold mt-1">Port 993 Active</div>
                    </div>
                    <div className="p-3 bg-slate-900/50 rounded-xl border border-slate-800">
                      <div className="text-xs text-slate-400">DKIM / SPF / DMARC</div>
                      <div className="text-emerald-400 font-bold mt-1">10/10 Score</div>
                    </div>
                  </div>
                </div>
              )}

              {activePreviewTab === 'terminal' && (
                <div className="font-mono text-xs text-slate-300 space-y-1 bg-black/60 p-4 rounded-xl border border-slate-800/80">
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
      <section id="features" className="py-24 border-t border-slate-800/80 relative z-10">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="text-center max-w-3xl mx-auto">
            <h2 className="text-xs font-bold uppercase tracking-widest text-indigo-400 mb-2">
              Next-Generation Stack
            </h2>
            <p className="text-3xl sm:text-4xl font-extrabold text-white tracking-tight">
              Engineered for Extreme Speed, Security & Scalability
            </p>
            <p className="mt-4 text-base text-slate-400">
              Hostvra unites enterprise web servers, database clusters, mail servers, and modern developer tooling into
              a seamless, friction-free control panel.
            </p>
          </div>

          <div className="mt-16 grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
            <div className="p-6 rounded-2xl bg-slate-900/60 border border-slate-800 hover:border-indigo-500/50 transition-all hover:shadow-xl hover:shadow-indigo-500/10 group">
              <div className="w-12 h-12 rounded-xl bg-indigo-600/20 text-indigo-400 flex items-center justify-center mb-5 group-hover:scale-110 transition-transform">
                <Globe className="w-6 h-6" />
              </div>
              <h3 className="text-lg font-bold text-white">Nginx & OpenLiteSpeed</h3>
              <p className="mt-2 text-sm text-slate-400 leading-relaxed">
                Supercharged web servers equipped with HTTP/3, Brotli compression, and Redis object cache for instant page
                loads.
              </p>
            </div>

            <div className="p-6 rounded-2xl bg-slate-900/60 border border-slate-800 hover:border-purple-500/50 transition-all hover:shadow-xl hover:shadow-purple-500/10 group">
              <div className="w-12 h-12 rounded-xl bg-purple-600/20 text-purple-400 flex items-center justify-center mb-5 group-hover:scale-110 transition-transform">
                <Code2 className="w-6 h-6" />
              </div>
              <h3 className="text-lg font-bold text-white">Multi-PHP 7.4 to 8.3</h3>
              <p className="mt-2 text-sm text-slate-400 leading-relaxed">
                Seamlessly run multiple PHP versions per domain with isolated PHP-FPM pools, custom php.ini editor, and
                extensions.
              </p>
            </div>

            <div className="p-6 rounded-2xl bg-slate-900/60 border border-slate-800 hover:border-blue-500/50 transition-all hover:shadow-xl hover:shadow-blue-500/10 group">
              <div className="w-12 h-12 rounded-xl bg-blue-600/20 text-blue-400 flex items-center justify-center mb-5 group-hover:scale-110 transition-transform">
                <Boxes className="w-6 h-6" />
              </div>
              <h3 className="text-lg font-bold text-white">Node.js & Python</h3>
              <p className="mt-2 text-sm text-slate-400 leading-relaxed">
                Deploy Next.js, Express, Django, and FastAPI apps effortlessly via automated reverse-proxy and process
                managers.
              </p>
            </div>

            <div className="p-6 rounded-2xl bg-slate-900/60 border border-slate-800 hover:border-emerald-500/50 transition-all hover:shadow-xl hover:shadow-emerald-500/10 group">
              <div className="w-12 h-12 rounded-xl bg-emerald-600/20 text-emerald-400 flex items-center justify-center mb-5 group-hover:scale-110 transition-transform">
                <Database className="w-6 h-6" />
              </div>
              <h3 className="text-lg font-bold text-white">MySQL & PostgreSQL</h3>
              <p className="mt-2 text-sm text-slate-400 leading-relaxed">
                High-performance relational databases with web-based phpMyAdmin integration, remote host allowlists, and
                quotas.
              </p>
            </div>

            <div className="p-6 rounded-2xl bg-slate-900/60 border border-slate-800 hover:border-rose-500/50 transition-all hover:shadow-xl hover:shadow-rose-500/10 group">
              <div className="w-12 h-12 rounded-xl bg-rose-600/20 text-rose-400 flex items-center justify-center mb-5 group-hover:scale-110 transition-transform">
                <Mail className="w-6 h-6" />
              </div>
              <h3 className="text-lg font-bold text-white">In-House Mail & Webmail</h3>
              <p className="mt-2 text-sm text-slate-400 leading-relaxed">
                Independent mail platform (Postfix & Dovecot) with Rspamd spam filtering, automated DKIM/DMARC, and
                webmail.
              </p>
            </div>

            <div className="p-6 rounded-2xl bg-slate-900/60 border border-slate-800 hover:border-amber-500/50 transition-all hover:shadow-xl hover:shadow-amber-500/10 group">
              <div className="w-12 h-12 rounded-xl bg-amber-600/20 text-amber-400 flex items-center justify-center mb-5 group-hover:scale-110 transition-transform">
                <Lock className="w-6 h-6" />
              </div>
              <h3 className="text-lg font-bold text-white">Automated Free SSL</h3>
              <p className="mt-2 text-sm text-slate-400 leading-relaxed">
                1-Click Let&apos;s Encrypt SSL certificates for all root domains and subdomains with automated zero-touch
                renewal.
              </p>
            </div>

            <div className="p-6 rounded-2xl bg-slate-900/60 border border-slate-800 hover:border-cyan-500/50 transition-all hover:shadow-xl hover:shadow-cyan-500/10 group">
              <div className="w-12 h-12 rounded-xl bg-cyan-600/20 text-cyan-400 flex items-center justify-center mb-5 group-hover:scale-110 transition-transform">
                <Shield className="w-6 h-6" />
              </div>
              <h3 className="text-lg font-bold text-white">Enterprise WAF & Jail</h3>
              <p className="mt-2 text-sm text-slate-400 leading-relaxed">
                Chroot file system isolation, Fail2ban brute-force protection, UFW firewall, and ModSecurity rule sets.
              </p>
            </div>

            <div className="p-6 rounded-2xl bg-slate-900/60 border border-slate-800 hover:border-indigo-500/50 transition-all hover:shadow-xl hover:shadow-indigo-500/10 group">
              <div className="w-12 h-12 rounded-xl bg-indigo-600/20 text-indigo-400 flex items-center justify-center mb-5 group-hover:scale-110 transition-transform">
                <Terminal className="w-6 h-6" />
              </div>
              <h3 className="text-lg font-bold text-white">Web Terminal & Backups</h3>
              <p className="mt-2 text-sm text-slate-400 leading-relaxed">
                Secure browser-based PTY shell, drag-and-drop file manager, scheduled cron jobs, and 1-click cloud
                backups.
              </p>
            </div>
          </div>
        </div>
      </section>

      {/* Dynamic Hosting Plans & Pricing */}
      <section id="pricing" className="py-24 border-t border-slate-800/80 bg-slate-950/40 relative z-10">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="text-center max-w-3xl mx-auto">
            <h2 className="text-xs font-bold uppercase tracking-widest text-indigo-400 mb-2">
              Transparent Cloud Pricing
            </h2>
            <p className="text-3xl sm:text-5xl font-black text-white tracking-tight">
              Flexible Plans for Every Stage of Growth
            </p>
            <p className="mt-4 text-base text-slate-400">
              Choose the package that suits your projects. Try Starter or Business free for 14 days with zero risk.
            </p>

            {/* Monthly / Yearly Toggle */}
            <div className="mt-8 inline-flex items-center p-1.5 rounded-2xl bg-slate-900 border border-slate-800">
              <button
                onClick={() => setBillingCycle('monthly')}
                className={`px-5 py-2 rounded-xl text-xs sm:text-sm font-bold transition-all ${
                  billingCycle === 'monthly'
                    ? 'bg-indigo-600 text-white shadow-lg shadow-indigo-600/25'
                    : 'text-slate-400 hover:text-white'
                }`}
              >
                Monthly Billing
              </button>
              <button
                onClick={() => setBillingCycle('yearly')}
                className={`px-5 py-2 rounded-xl text-xs sm:text-sm font-bold transition-all flex items-center gap-1.5 ${
                  billingCycle === 'yearly'
                    ? 'bg-indigo-600 text-white shadow-lg shadow-indigo-600/25'
                    : 'text-slate-400 hover:text-white'
                }`}
              >
                Yearly Billing
                <span className="text-[10px] uppercase font-black px-1.5 py-0.5 rounded bg-emerald-500/20 text-emerald-400 border border-emerald-500/30">
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
                  className={`relative flex flex-col p-8 rounded-3xl bg-slate-900/70 border transition-all duration-300 hover:scale-[1.02] ${
                    plan.is_featured
                      ? 'border-indigo-500 shadow-2xl shadow-indigo-500/20 ring-1 ring-indigo-500 bg-slate-900/90'
                      : 'border-slate-800 hover:border-slate-700 shadow-xl'
                  }`}
                >
                  {plan.is_featured && (
                    <div className="absolute -top-3.5 left-1/2 -translate-x-1/2 px-4 py-1 rounded-full bg-gradient-to-r from-indigo-500 to-purple-500 text-[11px] font-black uppercase tracking-wider text-white shadow-lg">
                      Most Popular
                    </div>
                  )}

                  <div className="flex-1">
                    <h3 className="text-xl font-bold text-white">{plan.name}</h3>
                    <p className="mt-2 text-xs text-slate-400 leading-relaxed min-h-[36px]">{plan.description}</p>

                    <div className="mt-6 flex items-baseline gap-1">
                      <span className="text-4xl sm:text-5xl font-black text-white tracking-tight">
                        ${price.toFixed(2)}
                      </span>
                      <span className="text-xs text-slate-400 font-semibold">/ month</span>
                    </div>
                    {billingCycle === 'yearly' && (
                      <div className="text-[11px] text-emerald-400 font-medium mt-1">
                        Billed annually (${plan.price_yearly.toFixed(2)} / year)
                      </div>
                    )}

                    {hasTrial && (
                      <div className="mt-4 px-3 py-1.5 rounded-xl bg-indigo-500/10 border border-indigo-500/20 text-indigo-300 text-xs font-semibold flex items-center gap-1.5">
                        <Sparkles className="w-3.5 h-3.5 text-indigo-400" />
                        <span>Includes {plan.trial_days || 14}-Day Free Trial</span>
                      </div>
                    )}

                    {/* Limits Specs */}
                    <div className="mt-6 pt-6 border-t border-slate-800/80 space-y-3 text-xs text-slate-300">
                      <div className="flex items-center justify-between">
                        <span className="text-slate-400">Websites</span>
                        <span className="font-bold text-white">{plan.max_websites} Hosted</span>
                      </div>
                      <div className="flex items-center justify-between">
                        <span className="text-slate-400">NVMe SSD Storage</span>
                        <span className="font-bold text-white">{(plan.disk_space_mb / 1024).toFixed(0)} GB</span>
                      </div>
                      <div className="flex items-center justify-between">
                        <span className="text-slate-400">Monthly Bandwidth</span>
                        <span className="font-bold text-white">{(plan.bandwidth_mb / 1024).toFixed(0)} GB</span>
                      </div>
                      <div className="flex items-center justify-between">
                        <span className="text-slate-400">Databases</span>
                        <span className="font-bold text-white">{plan.max_databases} DBs</span>
                      </div>
                      <div className="flex items-center justify-between">
                        <span className="text-slate-400">Mailboxes</span>
                        <span className="font-bold text-white">{plan.max_mailboxes} Inboxes</span>
                      </div>
                    </div>

                    {/* Features List */}
                    <div className="mt-6 pt-6 border-t border-slate-800/80 space-y-2.5">
                      {plan.features.slice(0, 6).map((feat, idx) => (
                        <div key={idx} className="flex items-start gap-2.5 text-xs text-slate-300">
                          <Check className="w-4 h-4 text-emerald-400 flex-shrink-0 mt-0.5" />
                          <span>{feat}</span>
                        </div>
                      ))}
                    </div>
                  </div>

                  <div className="mt-8 pt-6 border-t border-slate-800/80">
                    <Link
                      href={`/register?plan=${plan.slug}&cycle=${billingCycle}${hasTrial ? '&trial=true' : ''}`}
                      className={`w-full py-3.5 px-4 rounded-xl text-center text-xs font-bold transition-all block ${
                        plan.is_featured
                          ? 'bg-gradient-to-r from-indigo-600 to-purple-600 hover:from-indigo-500 hover:to-purple-500 text-white shadow-lg shadow-indigo-600/25'
                          : hasTrial
                          ? 'bg-indigo-600/20 hover:bg-indigo-600/30 text-indigo-300 border border-indigo-500/40'
                          : 'bg-slate-800 hover:bg-slate-700 text-white border border-slate-700'
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
      <section id="architecture" className="py-24 border-t border-slate-800/80 relative z-10">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-12 items-center">
            <div>
              <div className="inline-flex items-center gap-2 px-3 py-1 rounded-full bg-indigo-500/10 border border-indigo-500/20 text-indigo-400 text-xs font-bold uppercase tracking-wider mb-4">
                Architecture & Security
              </div>
              <h2 className="text-3xl sm:text-4xl font-black text-white tracking-tight">
                Complete Isolation. Zero Shared Risk.
              </h2>
              <p className="mt-4 text-slate-400 text-sm sm:text-base leading-relaxed">
                Traditional shared hosting suffers from noisy neighbors and cross-account vulnerabilities. Hostvra
                replaces obsolete paradigms with Linux containerized isolation and Chroot environments.
              </p>

              <div className="mt-8 space-y-4 text-sm text-slate-300">
                <div className="flex items-start gap-3">
                  <div className="p-2 rounded-lg bg-indigo-500/20 text-indigo-400 mt-1">
                    <Shield className="w-4 h-4" />
                  </div>
                  <div>
                    <strong className="text-white block font-semibold">Chroot File Boundary</strong>
                    <span className="text-slate-400 text-xs">
                      Users can never traverse above their home directory. Path canonicalization blocks symlink escapes
                      and Zip Slip vulnerabilities.
                    </span>
                  </div>
                </div>

                <div className="flex items-start gap-3">
                  <div className="p-2 rounded-lg bg-purple-500/20 text-purple-400 mt-1">
                    <Cpu className="w-4 h-4" />
                  </div>
                  <div>
                    <strong className="text-white block font-semibold">Dedicated CPU & RAM Cgroups</strong>
                    <span className="text-slate-400 text-xs">
                      Strict resource enforcement prevents a single traffic surge from affecting neighboring sites.
                    </span>
                  </div>
                </div>

                <div className="flex items-start gap-3">
                  <div className="p-2 rounded-lg bg-emerald-500/20 text-emerald-400 mt-1">
                    <Activity className="w-4 h-4" />
                  </div>
                  <div>
                    <strong className="text-white block font-semibold">Real-Time Threat Detection</strong>
                    <span className="text-slate-400 text-xs">
                      Automated Fail2ban, UFW dynamic rules, and ModSecurity block SQL injection, XSS, and brute-force
                      attacks instantly.
                    </span>
                  </div>
                </div>
              </div>
            </div>

            <div className="p-8 rounded-3xl bg-slate-900/60 border border-slate-800 shadow-2xl relative">
              <div className="space-y-4">
                <div className="flex items-center justify-between p-4 rounded-xl bg-slate-950/80 border border-slate-800">
                  <div className="flex items-center gap-3">
                    <div className="w-2.5 h-2.5 rounded-full bg-emerald-400 animate-ping" />
                    <div>
                      <div className="text-xs font-bold text-white">Fail2ban Brute-Force Shield</div>
                      <div className="text-[11px] text-slate-400">Monitoring SSH, Postfix & HTTP</div>
                    </div>
                  </div>
                  <span className="text-xs font-mono text-emerald-400 font-bold">0 Breaches</span>
                </div>

                <div className="flex items-center justify-between p-4 rounded-xl bg-slate-950/80 border border-slate-800">
                  <div className="flex items-center gap-3">
                    <div className="w-2.5 h-2.5 rounded-full bg-indigo-400" />
                    <div>
                      <div className="text-xs font-bold text-white">Automated Let&apos;s Encrypt ACME</div>
                      <div className="text-[11px] text-slate-400">Zero-downtime certificate renewal</div>
                    </div>
                  </div>
                  <span className="text-xs font-mono text-indigo-400 font-bold">Auto-Renewed</span>
                </div>

                <div className="flex items-center justify-between p-4 rounded-xl bg-slate-950/80 border border-slate-800">
                  <div className="flex items-center gap-3">
                    <div className="w-2.5 h-2.5 rounded-full bg-purple-400" />
                    <div>
                      <div className="text-xs font-bold text-white">Postfix / Rspamd Mail Gateway</div>
                      <div className="text-[11px] text-slate-400">SPF, DKIM, DMARC cryptographically signed</div>
                    </div>
                  </div>
                  <span className="text-xs font-mono text-purple-400 font-bold">100% Inboxed</span>
                </div>
              </div>
            </div>
          </div>
        </div>
      </section>

      {/* Frequently Asked Questions */}
      <section id="faq" className="py-24 border-t border-slate-800/80 bg-slate-950/30 relative z-10">
        <div className="max-w-4xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="text-center">
            <h2 className="text-xs font-bold uppercase tracking-widest text-indigo-400 mb-2">Got Questions?</h2>
            <p className="text-3xl sm:text-4xl font-extrabold text-white tracking-tight">
              Frequently Asked Questions
            </p>
          </div>

          <div className="mt-12 space-y-4">
            {FAQS.map((faq, index) => {
              const isOpen = openFaq === index;
              return (
                <div
                  key={index}
                  className="rounded-2xl bg-slate-900/60 border border-slate-800 overflow-hidden transition-all"
                >
                  <button
                    onClick={() => setOpenFaq(isOpen ? null : index)}
                    className="w-full p-6 text-left flex items-center justify-between gap-4"
                  >
                    <span className="text-base font-bold text-white">{faq.q}</span>
                    <ChevronDown
                      className={`w-5 h-5 text-slate-400 flex-shrink-0 transition-transform ${
                        isOpen ? 'rotate-180 text-indigo-400' : ''
                      }`}
                    />
                  </button>
                  {isOpen && (
                    <div className="px-6 pb-6 text-sm text-slate-300 leading-relaxed border-t border-slate-800/50 pt-4">
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
      <section className="py-20 border-t border-slate-800/80 relative z-10 overflow-hidden">
        <div className="max-w-5xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="relative rounded-3xl p-10 sm:p-16 bg-gradient-to-tr from-indigo-900/40 via-purple-900/20 to-slate-900/80 border border-indigo-500/30 shadow-2xl text-center overflow-hidden">
            <div className="absolute top-0 right-0 w-80 h-80 bg-indigo-500/10 blur-[100px] rounded-full pointer-events-none" />

            <h2 className="text-3xl sm:text-5xl font-black text-white tracking-tight">
              Ready to Upgrade Your Cloud Hosting?
            </h2>
            <p className="mt-4 text-base sm:text-lg text-slate-300 max-w-2xl mx-auto">
              Launch your sites in seconds with automated SSL, isolated NVMe storage, and dedicated developer tooling.
            </p>

            <div className="mt-8 flex flex-col sm:flex-row items-center justify-center gap-4">
              <Link
                href="/register?trial=true"
                className="w-full sm:w-auto px-8 py-4 rounded-xl bg-white hover:bg-slate-100 text-slate-950 font-extrabold text-sm sm:text-base shadow-xl transition-all"
              >
                Start 14-Day Free Trial
              </Link>
              <Link
                href="/login"
                className="w-full sm:w-auto px-8 py-4 rounded-xl bg-slate-900/80 hover:bg-slate-800/80 border border-slate-700 text-white font-semibold text-sm sm:text-base transition-all"
              >
                Sign In to Control Panel
              </Link>
            </div>
          </div>
        </div>
      </section>

      {/* Comprehensive Footer */}
      <footer className="border-t border-slate-800/80 bg-[#05070a] py-16 relative z-10 text-xs text-slate-400">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 grid grid-cols-2 md:grid-cols-5 gap-8">
          <div className="col-span-2">
            <Link href="/" className="flex items-center gap-2.5">
              <div className="w-8 h-8 rounded-lg bg-indigo-600 flex items-center justify-center">
                <Flame className="w-4 h-4 text-white fill-white" />
              </div>
              <span className="text-lg font-black tracking-tight text-white">HOSTVRA</span>
            </Link>
            <p className="mt-4 text-slate-400 leading-relaxed max-w-sm">
              Next-generation hosting control plane and server infrastructure manager. Built for high performance,
              security, and modern web applications.
            </p>
            <div className="mt-6 text-[11px] text-slate-400">
              © {new Date().getFullYear()} Hostvra Inc. All rights reserved.
            </div>
          </div>

          <div>
            <h4 className="font-bold text-white uppercase tracking-wider mb-4">Hosting</h4>
            <ul className="space-y-2.5">
              <li>
                <a href="#pricing" className="hover:text-white transition-colors">
                  Web Hosting
                </a>
              </li>
              <li>
                <a href="#pricing" className="hover:text-white transition-colors">
                  WordPress Cloud
                </a>
              </li>
              <li>
                <a href="#pricing" className="hover:text-white transition-colors">
                  Node.js & Python
                </a>
              </li>
              <li>
                <a href="#pricing" className="hover:text-white transition-colors">
                  Reseller Hosting
                </a>
              </li>
            </ul>
          </div>

          <div>
            <h4 className="font-bold text-white uppercase tracking-wider mb-4">Platform</h4>
            <ul className="space-y-2.5">
              <li>
                <Link href="/login" className="hover:text-white transition-colors">
                  Control Panel
                </Link>
              </li>
              <li>
                <Link href="/webmail" className="hover:text-white transition-colors">
                  Webmail Client
                </Link>
              </li>
              <li>
                <a href="#features" className="hover:text-white transition-colors">
                  In-House Mail
                </a>
              </li>
              <li>
                <a href="#architecture" className="hover:text-white transition-colors">
                  WAF & Security
                </a>
              </li>
            </ul>
          </div>

          <div>
            <h4 className="font-bold text-white uppercase tracking-wider mb-4">Support</h4>
            <ul className="space-y-2.5">
              <li>
                <Link href="/login" className="hover:text-white transition-colors">
                  Help Desk
                </Link>
              </li>
              <li>
                <a href="#faq" className="hover:text-white transition-colors">
                  Knowledgebase
                </a>
              </li>
              <li>
                <span className="inline-flex items-center gap-1.5 text-emerald-400 font-medium">
                  <span className="w-1.5 h-1.5 rounded-full bg-emerald-400 animate-pulse" />
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
