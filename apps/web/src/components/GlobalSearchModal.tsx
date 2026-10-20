'use client';

import React, { useState, useEffect, useRef, useMemo } from 'react';
import { useRouter } from 'next/navigation';
import {
  Search,
  Globe,
  Database,
  Folder,
  Shield,
  Terminal,
  Cpu,
  Mail,
  HardDrive,
  Settings,
  RefreshCw,
  Wrench,
  Power,
  X,
  ArrowRight,
  Layers,
  Lock,
  PlusCircle,
  Sparkles,
  Zap,
  Clock,
  Activity,
  FileCode,
  Share2,
} from 'lucide-react';
import { apiFetch, Website, RegisteredDomain } from '@/lib/api';

interface SearchItem {
  id: string;
  title: string;
  subtitle?: string;
  category: 'Pages & Tools' | 'Websites & Domains' | 'Quick Actions';
  href?: string;
  action?: () => void;
  icon: React.ComponentType<{ className?: string }>;
  badge?: string;
  badgeColor?: string;
  keywords?: string[];
}

interface GlobalSearchModalProps {
  isOpen: boolean;
  onClose: () => void;
  onTriggerRestart?: () => void;
  onTriggerFix?: () => void;
  onTriggerUpdate?: () => void;
}

export function GlobalSearchModal({
  isOpen,
  onClose,
  onTriggerRestart,
  onTriggerFix,
  onTriggerUpdate,
}: GlobalSearchModalProps) {
  const router = useRouter();
  const inputRef = useRef<HTMLInputElement>(null);
  const [query, setQuery] = useState('');
  const [selectedIndex, setSelectedIndex] = useState(0);

  // Live data
  const [websites, setWebsites] = useState<Website[]>([]);
  const [domains, setDomains] = useState<RegisteredDomain[]>([]);

  // Fetch live sites & domains on mount or when opened
  useEffect(() => {
    if (!isOpen) return;

    setQuery('');
    setSelectedIndex(0);

    // Auto-focus input
    setTimeout(() => {
      inputRef.current?.focus();
    }, 50);

    async function loadData() {
      try {
        const [sitesRes, domRes] = await Promise.all([
          apiFetch<Website[]>('/api/v1/websites'),
          apiFetch<RegisteredDomain[]>('/api/v1/domains'),
        ]);
        if (sitesRes.success && sitesRes.data) {
          setWebsites(sitesRes.data);
        }
        if (domRes.success && domRes.data) {
          setDomains(domRes.data);
        }
      } catch (e) {
        // Fallback gracefully
      }
    }
    loadData();
  }, [isOpen]);

  // Static Platform Features & Pages
  const staticItems = useMemo<SearchItem[]>(() => [
    {
      id: 'page-websites',
      title: 'Websites & Applications',
      subtitle: 'Manage WordPress, PHP, Node.js sites & virtual hosts',
      category: 'Pages & Tools',
      href: '/websites',
      icon: Globe,
      badge: 'Core',
      badgeColor: 'bg-emerald-500/10 text-emerald-600',
      keywords: ['sites', 'wordpress', 'php', 'vhost', 'domain', 'web', 'nginx', 'apache'],
    },
    {
      id: 'page-domains',
      title: 'Domains & DNS Manager',
      subtitle: 'Domain registration, WHOIS, DNS records & registrar settings',
      category: 'Pages & Tools',
      href: '/domains',
      icon: Layers,
      badge: 'Reseller',
      badgeColor: 'bg-blue-500/10 text-blue-600',
      keywords: ['domain', 'whois', 'registration', 'reseller', 'tld', 'nameservers', 'epp', 'transfer'],
    },
    {
      id: 'page-databases',
      title: 'Databases & Users',
      subtitle: 'MySQL, PostgreSQL, MariaDB database provisioning',
      category: 'Pages & Tools',
      href: '/databases',
      icon: Database,
      keywords: ['db', 'mysql', 'postgres', 'sql', 'mariadb', 'tables'],
    },
    {
      id: 'page-phpmyadmin',
      title: 'phpMyAdmin',
      subtitle: 'Web-based graphical database administration',
      category: 'Pages & Tools',
      href: '/phpmyadmin',
      icon: FileCode,
      badge: 'Direct',
      badgeColor: 'bg-amber-500/10 text-amber-600',
      keywords: ['pma', 'mysql', 'tables', 'browse', 'sql'],
    },
    {
      id: 'page-files',
      title: 'File Manager',
      subtitle: 'Browse, edit, zip, unzip, and upload website files',
      category: 'Pages & Tools',
      href: '/files',
      icon: Folder,
      keywords: ['file', 'explorer', 'upload', 'code', 'editor', 'public_html'],
    },
    {
      id: 'page-ssl',
      title: 'SSL / TLS Certificates',
      subtitle: "Let's Encrypt auto-renewal, Cloudflare Origin & custom certs",
      category: 'Pages & Tools',
      href: '/ssl',
      icon: Lock,
      badge: 'HTTPS',
      badgeColor: 'bg-emerald-500/10 text-emerald-600',
      keywords: ['ssl', 'tls', 'https', 'cert', 'letsencrypt', 'cloudflare', 'certificate'],
    },
    {
      id: 'page-dns',
      title: 'DNS Zone Editor',
      subtitle: 'Configure A, CNAME, MX, TXT, SPF, DKIM records',
      category: 'Pages & Tools',
      href: '/dns',
      icon: Share2,
      keywords: ['dns', 'records', 'zone', 'nameserver', 'mx', 'txt', 'cname'],
    },
    {
      id: 'page-terminal',
      title: 'Web Terminal (SSH Console)',
      subtitle: 'Direct root shell access right in your browser',
      category: 'Pages & Tools',
      href: '/terminal',
      icon: Terminal,
      badge: 'SSH',
      badgeColor: 'bg-purple-500/10 text-purple-600',
      keywords: ['ssh', 'bash', 'cli', 'console', 'root', 'command'],
    },
    {
      id: 'page-email',
      title: 'Email Accounts & Webmail',
      subtitle: 'Create mailboxes, webmail client, forwarders & DKIM',
      category: 'Pages & Tools',
      href: '/email',
      icon: Mail,
      keywords: ['mail', 'inbox', 'webmail', 'smtp', 'imap', 'roundcube'],
    },
    {
      id: 'page-firewall',
      title: 'Firewall & Network Security',
      subtitle: 'Port filtering, IP blacklists, UFW rules & intrusion defense',
      category: 'Pages & Tools',
      href: '/firewall',
      icon: Shield,
      keywords: ['firewall', 'ufw', 'port', 'block', 'security'],
    },
    {
      id: 'page-waf',
      title: 'Web Application Firewall (WAF)',
      subtitle: 'ModSecurity OWASP CRS, DDoS mitigation & rate limits',
      category: 'Pages & Tools',
      href: '/waf',
      icon: Zap,
      badge: 'Enterprise',
      badgeColor: 'bg-rose-500/10 text-rose-600',
      keywords: ['waf', 'modsecurity', 'ddos', 'rate limit', 'rules'],
    },
    {
      id: 'page-backups',
      title: 'System & Website Backups',
      subtitle: 'One-click snapshot creation, restores & scheduled S3 storage',
      category: 'Pages & Tools',
      href: '/backups',
      icon: HardDrive,
      keywords: ['backup', 'snapshot', 'restore', 'archive', 's3'],
    },
    {
      id: 'page-cron',
      title: 'Cron Jobs & Scheduler',
      subtitle: 'Automated background tasks and scheduled bash commands',
      category: 'Pages & Tools',
      href: '/cron',
      icon: Clock,
      keywords: ['cron', 'schedule', 'crontab', 'automation'],
    },
    {
      id: 'page-php',
      title: 'PHP Versions & Extensions',
      subtitle: 'Configure PHP 7.4 - 8.4, opcache, memory limits & php.ini',
      category: 'Pages & Tools',
      href: '/php',
      icon: Cpu,
      keywords: ['php', 'fpm', 'php.ini', 'extensions', 'opcache'],
    },
    {
      id: 'page-app-store',
      title: '1-Click App Store',
      subtitle: 'Install WordPress, Node.js, Redis, Composer & developer packages',
      category: 'Pages & Tools',
      href: '/app-store',
      icon: Sparkles,
      badge: 'Installer',
      badgeColor: 'bg-amber-500/10 text-amber-600',
      keywords: ['apps', 'install', 'wordpress', 'node', 'redis', 'oneclick'],
    },
    {
      id: 'page-audit',
      title: 'Audit Logs & Security Stream',
      subtitle: 'Real-time security telemetry and administrative events',
      category: 'Pages & Tools',
      href: '/audit-logs',
      icon: Activity,
      keywords: ['logs', 'audit', 'security', 'events', 'history'],
    },
    {
      id: 'page-settings',
      title: 'Panel Settings & API Keys',
      subtitle: 'Two-factor auth (2FA), credentials, system users & themes',
      category: 'Pages & Tools',
      href: '/settings',
      icon: Settings,
      keywords: ['settings', 'config', 'profile', '2fa', 'theme', 'api'],
    },
  ], []);

  // Quick System Actions
  const actionItems = useMemo<SearchItem[]>(() => [
    {
      id: 'action-new-site',
      title: 'Create New Website',
      subtitle: 'Deploy a new virtual host with custom domain and PHP pool',
      category: 'Quick Actions',
      href: '/websites',
      icon: PlusCircle,
      badge: 'Action',
      badgeColor: 'bg-emerald-500/10 text-emerald-600',
      keywords: ['create', 'add website', 'deploy'],
    },
    {
      id: 'action-search-domain',
      title: 'Search & Register Domain',
      subtitle: 'Check real-time availability and prices across top extensions',
      category: 'Quick Actions',
      href: '/domains?tab=search',
      icon: Search,
      badge: 'Realtime',
      badgeColor: 'bg-blue-500/10 text-blue-600',
      keywords: ['search domain', 'register domain', 'buy domain'],
    },
    {
      id: 'action-fix',
      title: 'Run Self-Repair Utility',
      subtitle: 'Re-align socket permissions, Nginx config & system health',
      category: 'Quick Actions',
      action: () => {
        onClose();
        if (onTriggerFix) onTriggerFix();
      },
      icon: Wrench,
      badge: 'Repair',
      badgeColor: 'bg-orange-500/10 text-orange-600',
      keywords: ['fix', 'repair', 'permissions', 'health'],
    },
    {
      id: 'action-restart',
      title: 'Restart Web Services (Nginx / Panel)',
      subtitle: 'Gracefully reload web server and application daemon',
      category: 'Quick Actions',
      action: () => {
        onClose();
        if (onTriggerRestart) onTriggerRestart();
      },
      icon: Power,
      badge: 'System',
      badgeColor: 'bg-rose-500/10 text-rose-600',
      keywords: ['restart', 'reload', 'reboot', 'nginx'],
    },
    {
      id: 'action-update',
      title: 'Check for Hostvra Updates',
      subtitle: 'Scan for latest stable panel packages and security patches',
      category: 'Quick Actions',
      action: () => {
        onClose();
        if (onTriggerUpdate) onTriggerUpdate();
      },
      icon: RefreshCw,
      badge: 'v1.2.0',
      badgeColor: 'bg-emerald-500/10 text-emerald-600',
      keywords: ['update', 'upgrade', 'check update', 'version'],
    },
  ], [onClose, onTriggerFix, onTriggerRestart, onTriggerUpdate]);

  // Dynamic live website & domain items
  const dynamicItems = useMemo<SearchItem[]>(() => {
    const list: SearchItem[] = [];

    websites.forEach((w) => {
      list.push({
        id: `site-${w.id}`,
        title: w.primary_domain,
        subtitle: `Website (${w.app_type || 'php'}) • Document Root: ${w.document_root || '/var/www/' + w.primary_domain}`,
        category: 'Websites & Domains',
        href: `/websites?domain=${encodeURIComponent(w.primary_domain)}`,
        icon: Globe,
        badge: w.ssl_enabled ? 'SSL Active' : 'HTTP',
        badgeColor: w.ssl_enabled ? 'bg-emerald-500/10 text-emerald-600' : 'bg-slate-100 text-slate-500',
        keywords: [w.primary_domain, w.app_type || 'php', 'site'],
      });
    });

    domains.forEach((d) => {
      // Don't duplicate if already listed in websites
      if (!websites.some((w) => w.primary_domain === d.domain_name)) {
        list.push({
          id: `dom-${d.id}`,
          title: d.domain_name,
          subtitle: `Registered Domain (${d.tld}) • Status: ${d.status}`,
          category: 'Websites & Domains',
          href: `/domains?domain=${encodeURIComponent(d.domain_name)}`,
          icon: Layers,
          badge: d.status,
          badgeColor: d.status === 'active' ? 'bg-emerald-500/10 text-emerald-600' : 'bg-amber-500/10 text-amber-600',
          keywords: [d.domain_name, d.tld, 'registered'],
        });
      }
    });

    return list;
  }, [websites, domains]);

  // Filtered Results
  const filteredItems = useMemo(() => {
    const q = query.trim().toLowerCase();
    if (!q) {
      // Show default top items
      return [
        ...actionItems.slice(0, 3),
        ...dynamicItems.slice(0, 4),
        ...staticItems.slice(0, 8),
      ];
    }

    const all = [...actionItems, ...dynamicItems, ...staticItems];

    // Check if query looks like a domain to offer instant domain check
    const domainCheckItem: SearchItem[] = [];
    if (q.length >= 2 && !q.includes(' ')) {
      const cleanQ = q.replace(/^https?:\/\//, '').replace(/^www\./, '');
      domainCheckItem.push({
        id: `search-domain-live-${cleanQ}`,
        title: `Check availability for "${cleanQ}"`,
        subtitle: `Search real-time pricing and instant registration across .com, .net, .org, .xyz`,
        category: 'Quick Actions',
        href: `/domains?tab=search&q=${encodeURIComponent(cleanQ)}`,
        icon: Search,
        badge: 'Instant Check',
        badgeColor: 'bg-emerald-500/10 text-emerald-600',
        keywords: [cleanQ, 'register', 'domain'],
      });
    }

    const matches = all.filter((item) => {
      const matchTitle = item.title.toLowerCase().includes(q);
      const matchSub = item.subtitle ? item.subtitle.toLowerCase().includes(q) : false;
      const matchKeywords = item.keywords ? item.keywords.some((k) => k.toLowerCase().includes(q)) : false;
      return matchTitle || matchSub || matchKeywords;
    });

    return [...domainCheckItem, ...matches];
  }, [query, actionItems, dynamicItems, staticItems]);

  // Keep selected index within bounds
  useEffect(() => {
    setSelectedIndex(0);
  }, [query]);

  // Keyboard navigation
  useEffect(() => {
    if (!isOpen) return;

    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'ArrowDown') {
        e.preventDefault();
        setSelectedIndex((prev) => (prev + 1 < filteredItems.length ? prev + 1 : 0));
      } else if (e.key === 'ArrowUp') {
        e.preventDefault();
        setSelectedIndex((prev) => (prev - 1 >= 0 ? prev - 1 : filteredItems.length - 1));
      } else if (e.key === 'Enter') {
        e.preventDefault();
        const selected = filteredItems[selectedIndex];
        if (selected) {
          executeSelection(selected);
        }
      } else if (e.key === 'Escape') {
        e.preventDefault();
        onClose();
      }
    };

    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [isOpen, filteredItems, selectedIndex, onClose]);

  const executeSelection = (item: SearchItem) => {
    onClose();
    if (item.action) {
      item.action();
    } else if (item.href) {
      router.push(item.href);
    }
  };

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 z-50 flex items-start justify-center pt-16 sm:pt-24 px-4 bg-slate-950/60 backdrop-blur-xs transition-opacity animate-in fade-in duration-150">
      <div
        className="relative w-full max-w-2xl bg-white dark:bg-[#0F172A] rounded-2xl shadow-2xl border border-slate-200 dark:border-slate-800 overflow-hidden flex flex-col max-h-[80vh] transition-all"
        onClick={(e) => e.stopPropagation()}
      >
        {/* Search Input Bar */}
        <div className="relative flex items-center px-4 py-3.5 border-b border-slate-200 dark:border-slate-800">
          <Search className="w-5 h-5 text-slate-400 dark:text-slate-500 mr-3 flex-shrink-0" />
          <input
            ref={inputRef}
            type="text"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder="Type a domain, command, or feature... (e.g. onlyflrt, ssl, terminal)"
            className="w-full bg-transparent text-sm sm:text-base text-slate-900 dark:text-white placeholder-slate-400 focus:outline-none"
          />
          {query ? (
            <button
              onClick={() => setQuery('')}
              className="p-1 rounded-md text-slate-400 hover:text-slate-600 dark:hover:text-slate-200 transition cursor-pointer"
            >
              <X className="w-4 h-4" />
            </button>
          ) : (
            <kbd className="hidden sm:inline-flex items-center px-2 py-0.5 text-[11px] font-mono text-slate-400 bg-slate-100 dark:bg-slate-800 border border-slate-200 dark:border-slate-700 rounded shadow-2xs">
              ESC
            </kbd>
          )}
        </div>

        {/* Results Stream */}
        <div className="overflow-y-auto flex-1 p-2 divide-y divide-slate-100 dark:divide-slate-800/60">
          {filteredItems.length === 0 ? (
            <div className="py-12 text-center text-slate-400">
              <Search className="w-8 h-8 mx-auto mb-2 opacity-30" />
              <p className="text-sm font-medium">No results found for &quot;{query}&quot;</p>
              <p className="text-xs text-slate-500 mt-1">
                Try searching for domains, SSL, terminal, or WordPress
              </p>
            </div>
          ) : (
            <div className="space-y-1">
              {filteredItems.map((item, idx) => {
                const isSelected = idx === selectedIndex;
                const IconComponent = item.icon;

                return (
                  <div
                    key={item.id}
                    onMouseEnter={() => setSelectedIndex(idx)}
                    onClick={() => executeSelection(item)}
                    className={`flex items-center justify-between p-3 rounded-xl cursor-pointer transition-all ${
                      isSelected
                        ? 'bg-emerald-500/10 dark:bg-emerald-500/15 border border-emerald-500/20 text-emerald-950 dark:text-emerald-100'
                        : 'hover:bg-slate-50 dark:hover:bg-slate-800/50 border border-transparent text-slate-700 dark:text-slate-200'
                    }`}
                  >
                    <div className="flex items-center gap-3 min-w-0">
                      <div
                        className={`w-9 h-9 rounded-lg flex items-center justify-center flex-shrink-0 transition ${
                          isSelected
                            ? 'bg-emerald-600 text-white shadow-xs'
                            : 'bg-slate-100 dark:bg-slate-800 text-slate-600 dark:text-slate-400'
                        }`}
                      >
                        <IconComponent className="w-4 h-4" />
                      </div>
                      <div className="min-w-0">
                        <div className="flex items-center gap-2">
                          <span className="text-sm font-bold truncate">{item.title}</span>
                          {item.badge && (
                            <span
                              className={`px-1.5 py-0.5 rounded text-[10px] font-bold tracking-wide uppercase border border-current/10 ${
                                item.badgeColor || 'bg-slate-100 text-slate-500'
                              }`}
                            >
                              {item.badge}
                            </span>
                          )}
                        </div>
                        {item.subtitle && (
                          <p className="text-xs text-slate-500 dark:text-slate-400 truncate">
                            {item.subtitle}
                          </p>
                        )}
                      </div>
                    </div>

                    <div className="flex items-center gap-2 pl-3 flex-shrink-0">
                      <span className="text-[10px] font-medium text-slate-400 dark:text-slate-500 hidden sm:inline">
                        {item.category}
                      </span>
                      <ArrowRight
                        className={`w-4 h-4 transition-transform ${
                          isSelected ? 'translate-x-0.5 text-emerald-600 dark:text-emerald-400' : 'text-slate-300 dark:text-slate-600'
                        }`}
                      />
                    </div>
                  </div>
                );
              })}
            </div>
          )}
        </div>

        {/* Footer Shortcut Helper */}
        <div className="px-4 py-2.5 bg-slate-50 dark:bg-[#0B1120] border-t border-slate-200 dark:border-slate-800 flex items-center justify-between text-[11px] text-slate-500">
          <div className="flex items-center gap-3">
            <span className="flex items-center gap-1">
              <kbd className="px-1.5 py-0.5 bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 rounded shadow-2xs font-mono">
                ↑
              </kbd>
              <kbd className="px-1.5 py-0.5 bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 rounded shadow-2xs font-mono">
                ↓
              </kbd>
              Navigate
            </span>
            <span className="flex items-center gap-1">
              <kbd className="px-1.5 py-0.5 bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 rounded shadow-2xs font-mono">
                ↵
              </kbd>
              Select
            </span>
          </div>
          <span className="text-slate-400">Hostvra Real-Time OmniSearch</span>
        </div>
      </div>
    </div>
  );
}
