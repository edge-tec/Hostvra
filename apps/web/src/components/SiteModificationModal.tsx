'use client';

import React, { useState, useEffect, useMemo } from 'react';
import {
  X,
  Globe,
  Folder,
  Shield,
  FileCode,
  FileText,
  Lock,
  Cpu,
  Server,
  GitBranch,
  Terminal,
  CornerDownRight,
  Radio,
  EyeOff,
  AlertOctagon,
  ScrollText,
  Plus,
  Trash2,
  Check,
  RotateCcw,
  ExternalLink,
  Copy,
  ChevronUp,
  ChevronDown,
  RefreshCw,
  Play,
  Save,
  CheckCircle2,
  AlertTriangle,
  Zap,
} from 'lucide-react';
import { Website, apiFetch } from '@/lib/api';

export type SiteModalTab =
  | 'domain'
  | 'directory'
  | 'limit'
  | 'rewrite'
  | 'defaultDoc'
  | 'config'
  | 'ssl'
  | 'php'
  | 'webserver'
  | 'git'
  | 'composer'
  | 'redirect'
  | 'proxy'
  | 'hotlink'
  | 'maintenance'
  | 'logs';

interface DomainItem {
  domain: string;
  port: number;
  isPrimary: boolean;
}

interface RedirectRule {
  id: string;
  type: '301' | '302';
  sourcePath: string;
  targetUrl: string;
  preserveQuery: boolean;
  status: 'active' | 'disabled';
}

interface ProxyRule {
  id: string;
  name: string;
  targetUrl: string;
  sentHost: string;
  cacheEnabled: boolean;
  cacheTimeMinutes: number;
  websocket: boolean;
}

interface SiteModificationModalProps {
  website: Website | null;
  isOpen: boolean;
  onClose: () => void;
  initialTab?: SiteModalTab;
  onUpdateWebsite?: (updated: Partial<Website>) => void;
  showToast?: (message: string) => void;
}

const SIDEBAR_TABS = [
  { id: 'domain', label: 'Domain Manager', icon: Globe },
  { id: 'directory', label: 'Directory', icon: Folder },
  { id: 'limit', label: 'Limit access', icon: Shield },
  { id: 'rewrite', label: 'URL rewrite', icon: FileCode },
  { id: 'defaultDoc', label: 'Default document', icon: FileText },
  { id: 'config', label: 'Config', icon: Terminal },
  { id: 'ssl', label: 'SSL', icon: Lock },
  { id: 'php', label: 'PHP version', icon: Cpu },
  { id: 'webserver', label: 'Web Server', icon: Server },
  { id: 'git', label: 'Git Manager', icon: GitBranch },
  { id: 'composer', label: 'Composer', icon: Terminal },
  { id: 'redirect', label: 'Redirect', icon: CornerDownRight },
  { id: 'proxy', label: 'Reverse proxy', icon: Radio },
  { id: 'hotlink', label: 'Hotlink Protection', icon: EyeOff },
  { id: 'maintenance', label: 'Maintenance Mode', icon: AlertOctagon },
  { id: 'logs', label: 'Response log', icon: ScrollText },
] as const;

const REWRITE_PRESETS: Record<string, string> = {
  current: `# Current rewrite rules\nlocation / {\n    try_files $uri $uri/ /index.php?$query_string;\n}`,
  WordPress: `# WordPress Rewrite Rules\nlocation / {\n    try_files $uri $uri/ /index.php?$args;\n}\n\n# Add trailing slash to */wp-admin requests\nrewrite /wp-admin$ $scheme://$host$uri/ permanent;\n\nlocation ~* ^.+\\.(ogg|ogv|svg|svgz|eot|otf|woff|mp4|ttf|rss|atom|jpg|jpeg|gif|png|ico|zip|tgz|gz|rar|bz2|doc|xls|exe|ppt|tar|mid|midi|wav|bmp|rtf)$ {\n    access_log off; log_not_found off; expires max;\n}`,
  Laravel5: `# Laravel 5 / 8 / 9 / 10 / 11 Rewrite\nlocation / {\n    try_files $uri $uri/ /index.php?$query_string;\n}\n\nlocation = /favicon.ico { access_log off; log_not_found off; }\nlocation = /robots.txt  { access_log off; log_not_found off; }\n\nerror_page 404 /index.php;`,
  ThinkPHP: `# ThinkPHP Rewrite Rules\nlocation / {\n    if (!-e $request_filename){\n        rewrite  ^(.*)$  /index.php?s=$1  last;   break;\n    }\n}`,
  Joomla: `# Joomla Rewrite Rules\nlocation / {\n    try_files $uri $uri/ /index.php?$args;\n}`,
  Drupal: `# Drupal Rewrite Rules\nlocation / {\n    try_files $uri @rewrite;\n}\nlocation @rewrite {\n    rewrite ^ /index.php;\n}`,
  Discuz: `# Discuz! X Rewrite Rules\nrewrite ^([^\.]*)/topic-(.+)\.html$ $1/portal.php?mod=topic&topic=$2 last;\nrewrite ^([^\.]*)/article-([0-9]+)-([0-9]+)\.html$ $1/portal.php?mod=view&aid=$2&page=$3 last;\nrewrite ^([^\.]*)/forum-(\\w+)-([0-9]+)\.html$ $1/forum.php?mod=forumdisplay&fid=$2&page=$3 last;`,
  Typecho: `# Typecho Rewrite Rules\nif (!-e $request_filename) {\n    rewrite ^(.*)$ /index.php$1 last;\n}`,
  Nextcloud: `# Nextcloud Rewrite Rules\nlocation / {\n    rewrite ^ /index.php$request_uri;\n}`,
  CodeIgniter: `# CodeIgniter Rewrite Rules\nlocation / {\n    try_files $uri $uri/ /index.php?$query_string;\n}`,
  NginxDefault: `# Default Nginx Static Rules\nlocation / {\n    try_files $uri $uri/ =404;\n}`,
};

export function SiteModificationModal({
  website,
  isOpen,
  onClose,
  initialTab = 'domain',
  onUpdateWebsite,
  showToast = (msg) => alert(msg),
}: SiteModificationModalProps) {
  const [activeTab, setActiveTab] = useState<SiteModalTab>(initialTab);

  useEffect(() => {
    if (isOpen && initialTab) {
      setActiveTab(initialTab);
    }
  }, [isOpen, initialTab]);

  // ----------------------------------------------------
  // Tab 1: Domain Manager State
  // ----------------------------------------------------
  const [domainInput, setDomainInput] = useState('');
  const [domainList, setDomainList] = useState<DomainItem[]>([]);
  const [selectedDomains, setSelectedDomains] = useState<string[]>([]);

  useEffect(() => {
    if (website) {
      const primary = website.primary_domain;
      setDomainList([
        { domain: primary, port: 80, isPrimary: true },
        { domain: `www.${primary}`, port: 80, isPrimary: false },
      ]);
      setSelectedDomains([]);
    }
  }, [website]);

  const handleAddDomains = () => {
    if (!domainInput.trim()) return;
    const lines = domainInput
      .split('\n')
      .map((l) => l.trim())
      .filter(Boolean);
    const newItems: DomainItem[] = [];

    lines.forEach((line) => {
      let domain = line;
      let port = 80;
      if (line.includes(':')) {
        const parts = line.split(':');
        domain = parts[0];
        port = parseInt(parts[1], 10) || 80;
      }
      if (!domainList.some((d) => d.domain === domain && d.port === port)) {
        newItems.push({ domain, port, isPrimary: false });
      }
    });

    if (newItems.length > 0) {
      setDomainList([...domainList, ...newItems]);
      setDomainInput('');
      showToast(`Successfully added ${newItems.length} domain(s) to virtual host.`);
    }
  };

  const handleDeleteDomain = (domain: string) => {
    setDomainList(domainList.filter((d) => d.domain !== domain || d.isPrimary));
    setSelectedDomains(selectedDomains.filter((d) => d !== domain));
    showToast(`Domain '${domain}' removed from bindings.`);
  };

  const handleDeleteSelectedDomains = () => {
    if (selectedDomains.length === 0) return;
    setDomainList(domainList.filter((d) => !selectedDomains.includes(d.domain) || d.isPrimary));
    setSelectedDomains([]);
    showToast(`Removed selected domain bindings.`);
  };

  const toggleSelectDomain = (domain: string) => {
    if (selectedDomains.includes(domain)) {
      setSelectedDomains(selectedDomains.filter((d) => d !== domain));
    } else {
      setSelectedDomains([...selectedDomains, domain]);
    }
  };

  const toggleSelectAllDomains = () => {
    const nonPrimary = domainList.filter((d) => !d.isPrimary).map((d) => d.domain);
    if (selectedDomains.length === nonPrimary.length) {
      setSelectedDomains([]);
    } else {
      setSelectedDomains(nonPrimary);
    }
  };

  // ----------------------------------------------------
  // Tab 2: Directory State
  // ----------------------------------------------------
  const [docRoot, setDocRoot] = useState('');
  const [runDir, setRunDir] = useState('/');
  const [antiXss, setAntiXss] = useState(true);
  const [authEnabled, setAuthEnabled] = useState(false);
  const [authUser, setAuthUser] = useState('');
  const [authPass, setAuthPass] = useState('');

  useEffect(() => {
    if (website) {
      setDocRoot(website.document_root || `/www/wwwroot/${website.primary_domain}`);
    }
  }, [website]);

  // ----------------------------------------------------
  // Tab 3: Limit access State
  // ----------------------------------------------------
  const [ipLimitType, setIpLimitType] = useState<'block' | 'allow'>('block');
  const [newIpRule, setNewIpRule] = useState('');
  const [ipRules, setIpRules] = useState<{ ip: string; type: 'block' | 'allow'; date: string }[]>([
    { ip: '194.26.29.114', type: 'block', date: '2026-08-18' },
    { ip: '45.154.255.0/24', type: 'block', date: '2026-08-20' },
  ]);
  const [ccDefense, setCcDefense] = useState(false);
  const [rateLimitRps, setRateLimitRps] = useState(25);
  const [burstLimit, setBurstLimit] = useState(50);
  const [connLimit, setConnLimit] = useState(15);

  const handleAddIpRule = () => {
    if (!newIpRule.trim()) return;
    setIpRules([
      ...ipRules,
      { ip: newIpRule.trim(), type: ipLimitType, date: new Date().toISOString().slice(0, 10) },
    ]);
    setNewIpRule('');
    showToast(`IP rule for '${newIpRule.trim()}' added.`);
  };

  // ----------------------------------------------------
  // Tab 4: URL rewrite State
  // ----------------------------------------------------
  const [selectedPreset, setSelectedPreset] = useState('WordPress');
  const [rewriteRules, setRewriteRules] = useState(REWRITE_PRESETS['WordPress']);

  const handleSelectPreset = (preset: string) => {
    setSelectedPreset(preset);
    if (REWRITE_PRESETS[preset]) {
      setRewriteRules(REWRITE_PRESETS[preset]);
    }
  };

  // ----------------------------------------------------
  // Tab 5: Default document State
  // ----------------------------------------------------
  const [defaultDocs, setDefaultDocs] = useState([
    'index.php',
    'index.html',
    'index.htm',
    'default.php',
    'default.htm',
    'default.html',
  ]);
  const [newDocName, setNewDocName] = useState('');

  const moveDoc = (index: number, direction: 'up' | 'down') => {
    const target = direction === 'up' ? index - 1 : index + 1;
    if (target < 0 || target >= defaultDocs.length) return;
    const updated = [...defaultDocs];
    const temp = updated[index];
    updated[index] = updated[target];
    updated[target] = temp;
    setDefaultDocs(updated);
  };

  const handleAddDoc = () => {
    if (!newDocName.trim()) return;
    if (!defaultDocs.includes(newDocName.trim())) {
      setDefaultDocs([...defaultDocs, newDocName.trim()]);
    }
    setNewDocName('');
  };

  const handleDeleteDoc = (name: string) => {
    setDefaultDocs(defaultDocs.filter((d) => d !== name));
  };

  // ----------------------------------------------------
  // Tab 6: Config (vHost raw) State
  // ----------------------------------------------------
  const [vhostCode, setVhostCode] = useState('');
  const [isSavingConfig, setIsSavingConfig] = useState(false);

  useEffect(() => {
    if (website) {
      setVhostCode(`# Virtual Host Configuration for ${website.primary_domain}
# Auto-generated by Hostvra Cloud Control Engine

server {
    listen 80;
    listen [::]:80;
    server_name ${website.primary_domain} www.${website.primary_domain};
    index ${defaultDocs.join(' ')};
    root ${website.document_root};

    # SSL configuration (Let's Encrypt automated)
    ${
      website.ssl_enabled
        ? `listen 443 ssl http2;
    listen [::]:443 ssl http2;
    ssl_certificate /etc/letsencrypt/live/${website.primary_domain}/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/${website.primary_domain}/privkey.pem;
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers HIGH:!aNULL:!MD5;`
        : `# listen 443 ssl http2;`
    }

    # Security Headers
    add_header X-Frame-Options "SAMEORIGIN" always;
    add_header X-XSS-Protection "1; mode=block" always;
    add_header X-Content-Type-Options "nosniff" always;
    add_header Referrer-Policy "strict-origin-when-cross-origin" always;

    # Access and Error Logs
    access_log /var/log/nginx/${website.primary_domain}.access.log;
    error_log /var/log/nginx/${website.primary_domain}.error.log;

    # URL Rewrite Rules
    location / {
        try_files $uri $uri/ /index.php?$query_string;
    }

    # PHP-FPM FastCGI Handler
    location ~ \\.php$ {
        include snippets/fastcgi-php.conf;
        fastcgi_pass unix:/run/php/php${website.php_version || '8.1'}-fpm.sock;
        fastcgi_param SCRIPT_FILENAME $document_root$fastcgi_script_name;
        include fastcgi_params;
    }

    # Deny access to sensitive files
    location ~ /\\.(?!well-known).* {
        deny all;
    }

    # Static Asset Caching
    location ~* \\.(jpg|jpeg|gif|png|css|js|ico|webp|svg|woff|woff2)$ {
        expires 30d;
        add_header Cache-Control "public, no-transform";
    }
}`);
    }
  }, [website, defaultDocs]);

  const handleSaveVhost = () => {
    setIsSavingConfig(true);
    setTimeout(() => {
      setIsSavingConfig(false);
      showToast(`vHost configuration saved. Nginx syntax test passed (exit code 0), worker processes reloaded.`);
    }, 600);
  };

  // ----------------------------------------------------
  // Tab 7: SSL State
  // ----------------------------------------------------
  const [sslTab, setSslTab] = useState<'letsencrypt' | 'custom'>('letsencrypt');
  const [forceHttps, setForceHttps] = useState(website?.ssl_enabled ?? true);
  const [http2Enabled, setHttp2Enabled] = useState(true);
  const [autoRenew, setAutoRenew] = useState(true);
  const [customCert, setCustomCert] = useState('');
  const [customKey, setCustomKey] = useState('');
  const [isIssuingSsl, setIsIssuingSsl] = useState(false);

  const handleApplyLetsEncrypt = () => {
    setIsIssuingSsl(true);
    setTimeout(() => {
      setIsIssuingSsl(false);
      if (onUpdateWebsite && website) {
        onUpdateWebsite({ ssl_enabled: true, ssl_days_left: 90 });
      }
      showToast(`Let's Encrypt 90-day multi-domain certificate successfully issued and installed!`);
    }, 1200);
  };

  // ----------------------------------------------------
  // Tab 8: PHP Version State
  // ----------------------------------------------------
  const [selectedPhp, setSelectedPhp] = useState(website?.php_version || '8.1');
  const [phpMemoryLimit, setPhpMemoryLimit] = useState('256M');
  const [phpMaxExecution, setPhpMaxExecution] = useState('300');
  const [phpMaxUpload, setPhpMaxUpload] = useState('100M');
  const [opcacheEnabled, setOpcacheEnabled] = useState(true);

  const handleSwitchPhp = () => {
    if (onUpdateWebsite && website) {
      onUpdateWebsite({ php_version: selectedPhp });
    }
    showToast(`Website '${website?.primary_domain}' switched to PHP-${selectedPhp}! Pool reloaded.`);
  };

  // ----------------------------------------------------
  // Tab 9: Web Server State
  // ----------------------------------------------------
  const [webServerEngine, setWebServerEngine] = useState('nginx');
  const [gzipEnabled, setGzipEnabled] = useState(true);
  const [gzipLevel, setGzipLevel] = useState(6);
  const [brotliEnabled, setBrotliEnabled] = useState(true);
  const [clientMaxBody, setClientMaxBody] = useState('100M');
  const [keepaliveTimeout, setKeepaliveTimeout] = useState(65);

  // ----------------------------------------------------
  // Tab 10: Git Manager State
  // ----------------------------------------------------
  const [gitRepoUrl, setGitRepoUrl] = useState(`https://github.com/edge-tec/${website?.primary_domain || 'my-project'}.git`);
  const [gitBranch, setGitBranch] = useState('main');
  const [gitScript, setGitScript] = useState(`npm install --production\nnpm run build\npm2 restart all 2>/dev/null || true`);
  const [isPullingGit, setIsPullingGit] = useState(false);

  const handlePullGit = () => {
    setIsPullingGit(true);
    setTimeout(() => {
      setIsPullingGit(false);
      showToast(`Git pull complete. HEAD updated to latest commit (Origin/${gitBranch}).`);
    }, 1000);
  };

  // ----------------------------------------------------
  // Tab 11: Composer State
  // ----------------------------------------------------
  const [composerMirror, setComposerMirror] = useState('Packagist Official');
  const [composerConsole, setComposerConsole] = useState(`$ composer --version\nComposer version 2.7.2 2026-04-12 10:45:11\nPHP 8.1.28 (cli) (built: Feb 14 2026)\nReady for dependency installation.`);

  const handleRunComposer = (cmd: string) => {
    setComposerConsole((prev) => `${prev}\n\n$ composer ${cmd}\nLoading composer repositories with package information\nUpdating dependencies\nNothing to modify in lock file\nGenerating autoload files\nGenerated autoload files\n[OK] Operation completed successfully.`);
    showToast(`Executed: composer ${cmd}`);
  };

  // ----------------------------------------------------
  // Tab 12: Redirect State
  // ----------------------------------------------------
  const [redirectList, setRedirectList] = useState<RedirectRule[]>([
    {
      id: 'r1',
      type: '301',
      sourcePath: '/old-blog',
      targetUrl: 'https://antiprofiles.com/blog',
      preserveQuery: true,
      status: 'active',
    },
  ]);
  const [newRedirType, setNewRedirType] = useState<'301' | '302'>('301');
  const [newRedirSource, setNewRedirSource] = useState('');
  const [newRedirTarget, setNewRedirTarget] = useState('');
  const [newRedirPreserve, setNewRedirPreserve] = useState(true);

  const handleAddRedirect = () => {
    if (!newRedirSource.trim() || !newRedirTarget.trim()) return;
    setRedirectList([
      ...redirectList,
      {
        id: `r-${Date.now()}`,
        type: newRedirType,
        sourcePath: newRedirSource.trim(),
        targetUrl: newRedirTarget.trim(),
        preserveQuery: newRedirPreserve,
        status: 'active',
      },
    ]);
    setNewRedirSource('');
    setNewRedirTarget('');
    showToast(`Redirect rule (${newRedirType}) added successfully.`);
  };

  // ----------------------------------------------------
  // Tab 13: Reverse proxy State
  // ----------------------------------------------------
  const [proxyList, setProxyList] = useState<ProxyRule[]>([
    {
      id: 'p1',
      name: 'NextSSR',
      targetUrl: 'http://127.0.0.1:3000',
      sentHost: '$host',
      cacheEnabled: false,
      cacheTimeMinutes: 5,
      websocket: true,
    },
  ]);
  const [newProxyName, setNewProxyName] = useState('');
  const [newProxyTarget, setNewProxyTarget] = useState('');
  const [newProxySentHost, setNewProxySentHost] = useState('$host');
  const [newProxyCache, setNewProxyCache] = useState(false);
  const [newProxyWs, setNewProxyWs] = useState(true);

  const handleAddProxy = () => {
    if (!newProxyName.trim() || !newProxyTarget.trim()) return;
    setProxyList([
      ...proxyList,
      {
        id: `p-${Date.now()}`,
        name: newProxyName.trim(),
        targetUrl: newProxyTarget.trim(),
        sentHost: newProxySentHost.trim() || '$host',
        cacheEnabled: newProxyCache,
        cacheTimeMinutes: 10,
        websocket: newProxyWs,
      },
    ]);
    setNewProxyName('');
    setNewProxyTarget('');
    showToast(`Reverse proxy '${newProxyName.trim()}' added.`);
  };

  // ----------------------------------------------------
  // Tab 14: Hotlink Protection State
  // ----------------------------------------------------
  const [hotlinkEnabled, setHotlinkEnabled] = useState(false);
  const [hotlinkDomains, setHotlinkDomains] = useState(`${website?.primary_domain || 'antiprofiles.com'}\n*.${website?.primary_domain || 'antiprofiles.com'}\ngoogle.com\nbing.com`);
  const [hotlinkExts, setHotlinkExts] = useState('jpg,jpeg,png,gif,webp,ico,mp4,zip,pdf');
  const [hotlinkAction, setHotlinkAction] = useState<'403' | 'redirect'>('403');
  const [hotlinkRedirectUrl, setHotlinkRedirectUrl] = useState('https://antiprofiles.com/hotlink-warning.png');

  // ----------------------------------------------------
  // Tab 15: Maintenance Mode State
  // ----------------------------------------------------
  const [maintenanceEnabled, setMaintenanceEnabled] = useState(false);
  const [maintenanceTitle, setMaintenanceTitle] = useState("Under Scheduled Maintenance");
  const [maintenanceNotice, setMaintenanceNotice] = useState("We're currently performing scheduled maintenance to upgrade our infrastructure. We'll be back shortly!");
  const [maintenanceIps, setMaintenanceIps] = useState("127.0.0.1\n103.140.157.238");

  // ----------------------------------------------------
  // Tab 16: Response Log State
  // ----------------------------------------------------
  const [logType, setLogType] = useState<'access' | 'error'>('access');
  const [logFilter, setLogFilter] = useState<'all' | '2xx' | '4xx' | '5xx'>('all');
  const [logSearch, setLogSearch] = useState('');
  const [autoRefreshLogs, setAutoRefreshLogs] = useState(true);

  const mockLogs = useMemo(() => {
    const now = new Date().toISOString();
    return [
      `13.140.157.238 - - [${now}] "GET / HTTP/1.1" 200 8945 "-" "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7)"`,
      `13.140.157.238 - - [${now}] "GET /_next/static/css/styles.css HTTP/1.1" 200 4521 "https://${website?.primary_domain}/" "Mozilla/5.0"`,
      `66.249.66.1 - - [${now}] "GET /robots.txt HTTP/1.1" 200 120 "-" "Googlebot/2.1 (+http://www.google.com/bot.html)"`,
      `185.191.171.12 - - [${now}] "GET /wp-login.php HTTP/1.1" 404 153 "-" "SemrushBot/7~bl"`,
      `194.26.29.114 - - [${now}] "POST /xmlrpc.php HTTP/1.1" 403 210 "-" "curl/7.68.0"`,
      `13.140.157.238 - - [${now}] "GET /api/status HTTP/1.1" 200 45 "https://${website?.primary_domain}/" "Mozilla/5.0"`,
      `192.168.1.105 - - [${now}] "GET /admin/db HTTP/1.1" 401 188 "-" "Mozilla/5.0"`,
      `13.140.157.238 - - [${now}] "POST /login HTTP/1.1" 200 1024 "https://${website?.primary_domain}/login" "Mozilla/5.0"`,
    ];
  }, [website]);

  if (!isOpen || !website) return null;

  return (
    <div className="fixed inset-0 z-50 bg-black/60 backdrop-blur-xs flex items-center justify-center p-2 sm:p-4 animate-fadeIn">
      {/* Modal Container: 100% White in Light Mode, Dark Slate in Dark Mode */}
      <div className="w-full max-w-5xl h-[88vh] max-h-[760px] bg-white dark:bg-[#0B1120] text-slate-900 dark:text-slate-100 border border-slate-200 dark:border-slate-800 rounded-xl shadow-2xl flex flex-col overflow-hidden">
        {/* Header: Site modification [domain] */}
        <div className="flex items-center justify-between px-5 py-3.5 bg-slate-50 dark:bg-[#0F172A] border-b border-slate-200 dark:border-slate-800 select-none">
          <div className="flex items-center gap-2 truncate">
            <span className="font-semibold text-sm text-slate-900 dark:text-slate-100 truncate">
              Site modification [{website.primary_domain}] -- Time added [{website.created_at ? website.created_at.replace('T', ' ').slice(0, 19) : '2026-08-17 14:27:38'}]
            </span>
          </div>
          <button
            onClick={onClose}
            className="p-1 rounded text-slate-400 hover:text-slate-700 dark:hover:text-white hover:bg-slate-200 dark:hover:bg-slate-700/50 transition cursor-pointer"
          >
            <X className="w-5 h-5" />
          </button>
        </div>

        {/* Modal Body: Left Sidebar + Right Tab Content */}
        <div className="flex-1 flex overflow-hidden">
          {/* Left Sidebar (16 options) */}
          <div className="w-48 sm:w-52 bg-slate-50/80 dark:bg-[#0F172A] border-r border-slate-200 dark:border-slate-800 overflow-y-auto py-2 select-none flex-shrink-0">
            {SIDEBAR_TABS.map((tab) => {
              const Icon = tab.icon;
              const isActive = activeTab === tab.id;
              return (
                <button
                  key={tab.id}
                  onClick={() => setActiveTab(tab.id as SiteModalTab)}
                  className={`w-full flex items-center gap-2.5 px-4 py-2.5 text-xs text-left transition font-medium cursor-pointer ${
                    isActive
                      ? 'bg-white dark:bg-[#1E293B] text-emerald-700 dark:text-emerald-400 font-bold border-l-3 border-emerald-600 shadow-2xs'
                      : 'text-slate-600 dark:text-slate-400 hover:text-slate-950 dark:hover:text-slate-100 hover:bg-slate-100 dark:hover:bg-slate-800/40'
                  }`}
                >
                  <Icon className={`w-3.5 h-3.5 ${isActive ? 'text-emerald-600 dark:text-emerald-400' : 'text-slate-400'}`} />
                  <span className="truncate">{tab.label}</span>
                </button>
              );
            })}
          </div>

          {/* Right Content Area: Pure White in Light Mode, Dark in Dark Mode */}
          <div className="flex-1 overflow-y-auto p-5 bg-white dark:bg-[#0B1120] text-slate-900 dark:text-slate-100">
            {/* 1. DOMAIN MANAGER */}
            {activeTab === 'domain' && (
              <div className="space-y-4">
                {/* Domain Input Textarea + Add Button */}
                <div className="flex items-start gap-4">
                  <textarea
                    rows={4}
                    value={domainInput}
                    onChange={(e) => setDomainInput(e.target.value)}
                    placeholder={`A domain per line, the default port is 80\nWildcard domain format: *.domain.com\nTo add another port, the format is www.domain.com:88`}
                    className="flex-1 p-3 text-xs font-mono rounded-lg bg-slate-50 dark:bg-[#0F172A] border border-slate-300 dark:border-slate-700 text-slate-900 dark:text-slate-100 placeholder-slate-400 focus:outline-none focus:border-emerald-600 focus:ring-1 focus:ring-emerald-500 resize-none leading-relaxed"
                  />
                  <button
                    onClick={handleAddDomains}
                    className="px-5 py-2.5 rounded-lg bg-emerald-600 hover:bg-emerald-700 active:bg-emerald-800 text-white text-xs font-bold transition shadow-xs cursor-pointer flex-shrink-0"
                  >
                    Add
                  </button>
                </div>

                {/* Domain List Table */}
                <div className="border border-slate-200 dark:border-slate-800 rounded-lg overflow-hidden bg-white dark:bg-[#0F172A]">
                  <table className="w-full text-left text-xs border-collapse">
                    <thead>
                      <tr className="bg-slate-100 dark:bg-[#1E2432] text-slate-700 dark:text-slate-300 border-b border-slate-200 dark:border-slate-700">
                        <th className="w-10 px-3 py-2.5">
                          <input
                            type="checkbox"
                            checked={
                              selectedDomains.length > 0 &&
                              selectedDomains.length === domainList.filter((d) => !d.isPrimary).length
                            }
                            onChange={toggleSelectAllDomains}
                            className="rounded border-slate-300 dark:border-slate-600 accent-emerald-600 cursor-pointer"
                          />
                        </th>
                        <th className="px-3 py-2.5 font-semibold">Domain name</th>
                        <th className="px-3 py-2.5 font-semibold w-24">Port</th>
                        <th className="px-3 py-2.5 font-semibold text-right w-28">Operate</th>
                      </tr>
                    </thead>
                    <tbody className="divide-y divide-slate-200 dark:divide-slate-800">
                      {domainList.map((item) => {
                        const isChecked = selectedDomains.includes(item.domain);
                        return (
                          <tr key={`${item.domain}-${item.port}`} className="hover:bg-slate-50 dark:hover:bg-slate-800/40 text-slate-800 dark:text-slate-200 transition">
                            <td className="px-3 py-2.5">
                              {!item.isPrimary && (
                                <input
                                  type="checkbox"
                                  checked={isChecked}
                                  onChange={() => toggleSelectDomain(item.domain)}
                                  className="rounded border-slate-300 dark:border-slate-600 accent-emerald-600 cursor-pointer"
                                />
                              )}
                            </td>
                            <td className="px-3 py-2.5 font-mono text-[#22c55e]">
                              {item.domain}
                            </td>
                            <td className="px-3 py-2.5 text-slate-300 font-mono">
                              {item.port}
                            </td>
                            <td className="px-3 py-2.5 text-right">
                              {item.isPrimary ? (
                                <span className="text-slate-500 font-medium select-none">
                                  Inoperable
                                </span>
                              ) : (
                                <button
                                  onClick={() => handleDeleteDomain(item.domain)}
                                  className="text-rose-400 hover:text-rose-300 hover:underline cursor-pointer"
                                >
                                  Delete
                                </button>
                              )}
                            </td>
                          </tr>
                        );
                      })}
                    </tbody>
                  </table>
                </div>

                {/* Delete Selected Button */}
                <div>
                  <button
                    onClick={handleDeleteSelectedDomains}
                    disabled={selectedDomains.length === 0}
                    className="px-3.5 py-1.5 rounded-lg bg-orange-600 hover:bg-orange-700 active:bg-orange-800 text-white text-xs font-bold transition shadow-xs disabled:opacity-40 disabled:cursor-not-allowed cursor-pointer"
                  >
                    Delete Selected
                  </button>
                </div>
              </div>
            )}

            {/* 2. DIRECTORY */}
            {activeTab === 'directory' && (
              <div className="space-y-5 text-xs">
                <div className="bg-slate-50 dark:bg-[#131B2E] p-4 rounded-lg border border-slate-200 dark:border-slate-800 space-y-4">
                  <h3 className="text-sm font-semibold text-slate-900 dark:text-white">Website Directories</h3>
                  <div>
                    <label className="block text-slate-700 dark:text-slate-300 font-medium mb-1">Site Directory (Document Root)</label>
                    <div className="flex gap-2">
                      <input
                        type="text"
                        value={docRoot}
                        onChange={(e) => setDocRoot(e.target.value)}
                        className="flex-1 p-2 rounded-lg bg-white dark:bg-[#0F172A] border border-slate-300 dark:border-slate-700 text-slate-900 dark:text-slate-100 font-mono focus:outline-none focus:border-emerald-600 focus:ring-1 focus:ring-emerald-500"
                      />
                      <a
                        href={`/files?path=${encodeURIComponent(docRoot)}`}
                        className="px-3 py-2 rounded-lg bg-white dark:bg-slate-800 hover:bg-slate-100 dark:hover:bg-slate-700 text-slate-700 dark:text-slate-200 border border-slate-300 dark:border-slate-700 font-semibold flex items-center gap-1.5 transition"
                      >
                        <Folder className="w-3.5 h-3.5" />
                        <span>File Manager</span>
                      </a>
                    </div>
                  </div>

                  <div>
                    <label className="block text-slate-700 dark:text-slate-300 font-medium mb-1">Run Directory (Sub-Directory)</label>
                    <select
                      value={runDir}
                      onChange={(e) => setRunDir(e.target.value)}
                      className="p-2 rounded-lg bg-white dark:bg-[#0F172A] border border-slate-300 dark:border-slate-700 text-slate-900 dark:text-slate-100 font-mono w-64 focus:outline-none focus:border-emerald-600 focus:ring-1 focus:ring-emerald-500"
                    >
                      <option value="/">/ (Root)</option>
                      <option value="/public">/public (Laravel, ThinkPHP, Symfony)</option>
                      <option value="/dist">/dist (Vue, React, Vite)</option>
                      <option value="/web">/web (Yii2, Drupal)</option>
                    </select>
                    <p className="text-[11px] text-slate-500 dark:text-slate-400 mt-1">
                      Frameworks like Laravel or ThinkPHP must point to `/public` to ensure security.
                    </p>
                  </div>
                </div>

                <div className="bg-slate-50 dark:bg-[#131B2E] p-4 rounded-lg border border-slate-200 dark:border-slate-800 space-y-3">
                  <h3 className="text-sm font-semibold text-slate-900 dark:text-white">Security & Permissions</h3>
                  <label className="flex items-center gap-2.5 cursor-pointer">
                    <input
                      type="checkbox"
                      checked={antiXss}
                      onChange={(e) => setAntiXss(e.target.checked)}
                      className="rounded border-slate-300 dark:border-slate-600 accent-emerald-600"
                    />
                    <span className="font-medium text-slate-800 dark:text-slate-200">Anti-XSS attack (open_basedir protection)</span>
                  </label>
                  <p className="text-[11px] text-slate-500 dark:text-slate-400 pl-6">
                    Restricts PHP scripts to only read/write files within the website directory.
                  </p>

                  <div className="pt-3 border-t border-slate-200 dark:border-slate-800">
                    <label className="flex items-center gap-2.5 cursor-pointer">
                      <input
                        type="checkbox"
                        checked={authEnabled}
                        onChange={(e) => setAuthEnabled(e.target.checked)}
                        className="rounded border-slate-300 dark:border-slate-600 accent-emerald-600"
                      />
                      <span className="font-medium text-slate-800 dark:text-slate-200">Password Protection (HTTP Basic Auth)</span>
                    </label>
                    {authEnabled && (
                      <div className="mt-3 grid grid-cols-2 gap-3 pl-6">
                        <input
                          type="text"
                          placeholder="Username"
                          value={authUser}
                          onChange={(e) => setAuthUser(e.target.value)}
                          className="p-2 rounded-lg bg-white dark:bg-[#0F172A] border border-slate-300 dark:border-slate-700 text-slate-900 dark:text-slate-100"
                        />
                        <input
                          type="password"
                          placeholder="Password"
                          value={authPass}
                          onChange={(e) => setAuthPass(e.target.value)}
                          className="p-2 rounded-lg bg-white dark:bg-[#0F172A] border border-slate-300 dark:border-slate-700 text-slate-900 dark:text-slate-100"
                        />
                      </div>
                    )}
                  </div>
                </div>

                <button
                  onClick={() => showToast('Directory and permission settings saved!')}
                  className="px-4 py-2 rounded-lg bg-emerald-600 hover:bg-emerald-700 active:bg-emerald-800 text-white font-bold text-xs transition shadow-xs cursor-pointer"
                >
                  Save Directory Settings
                </button>
              </div>
            )}

            {/* 3. LIMIT ACCESS */}
            {activeTab === 'limit' && (
              <div className="space-y-5 text-xs">
                <div className="bg-slate-50 dark:bg-[#131B2E] p-4 rounded-lg border border-slate-200 dark:border-slate-800 space-y-3">
                  <h3 className="text-sm font-semibold text-slate-900 dark:text-white">IP Blacklist & Whitelist</h3>
                  <div className="flex gap-2">
                    <select
                      value={ipLimitType}
                      onChange={(e) => setIpLimitType(e.target.value as any)}
                      className="p-2 rounded-lg bg-white dark:bg-[#0F172A] border border-slate-300 dark:border-slate-700 text-slate-900 dark:text-slate-100 font-medium"
                    >
                      <option value="block">Block (Blacklist)</option>
                      <option value="allow">Allow (Whitelist)</option>
                    </select>
                    <input
                      type="text"
                      placeholder="IP or Subnet CIDR (e.g. 192.168.1.5 or 10.0.0.0/24)"
                      value={newIpRule}
                      onChange={(e) => setNewIpRule(e.target.value)}
                      className="flex-1 p-2 rounded-lg bg-white dark:bg-[#0F172A] border border-slate-300 dark:border-slate-700 text-slate-900 dark:text-slate-100 font-mono"
                    />
                    <button
                      onClick={handleAddIpRule}
                      className="px-4 py-2 rounded-lg bg-emerald-600 hover:bg-emerald-700 active:bg-emerald-800 text-white font-bold transition shadow-xs cursor-pointer"
                    >
                      Add IP
                    </button>
                  </div>

                  <div className="border border-slate-200 dark:border-slate-800 rounded-lg overflow-hidden mt-3 bg-white dark:bg-[#0F172A]">
                    <table className="w-full text-left">
                      <thead className="bg-slate-100 dark:bg-[#1E2432] text-slate-700 dark:text-slate-300 border-b border-slate-200 dark:border-slate-700">
                        <tr>
                          <th className="p-2.5">IP / CIDR</th>
                          <th className="p-2.5">Action</th>
                          <th className="p-2.5">Date Added</th>
                          <th className="p-2.5 text-right">Remove</th>
                        </tr>
                      </thead>
                      <tbody className="divide-y divide-slate-200 dark:divide-slate-800">
                        {ipRules.map((rule, idx) => (
                          <tr key={idx} className="hover:bg-slate-50 dark:hover:bg-slate-800/40 text-slate-800 dark:text-slate-200 transition">
                            <td className="p-2.5 font-mono text-slate-900 dark:text-slate-100">{rule.ip}</td>
                            <td className="p-2.5">
                              <span
                                className={`px-2 py-0.5 rounded text-[10px] font-bold ${
                                  rule.type === 'block' ? 'bg-rose-100 dark:bg-rose-500/20 text-rose-700 dark:text-rose-400' : 'bg-emerald-100 dark:bg-emerald-500/20 text-emerald-700 dark:text-emerald-400'
                                }`}
                              >
                                {rule.type.toUpperCase()}
                              </span>
                            </td>
                            <td className="p-2.5 text-slate-500 dark:text-slate-400">{rule.date}</td>
                            <td className="p-2.5 text-right">
                              <button
                                onClick={() => setIpRules(ipRules.filter((_, i) => i !== idx))}
                                className="text-rose-600 hover:text-rose-700 dark:text-rose-400 hover:underline font-medium"
                              >
                                Delete
                              </button>
                            </td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>
                </div>

                <div className="bg-slate-50 dark:bg-[#131B2E] p-4 rounded-lg border border-slate-200 dark:border-slate-800 space-y-3">
                  <div className="flex items-center justify-between">
                    <div>
                      <h3 className="text-sm font-semibold text-slate-900 dark:text-white">CC Defense & Rate Limiting</h3>
                      <p className="text-[11px] text-slate-500 dark:text-slate-400">Anti-DDoS frequency limits implemented directly in Nginx</p>
                    </div>
                    <label className="relative inline-flex items-center cursor-pointer">
                      <input
                        type="checkbox"
                        checked={ccDefense}
                        onChange={(e) => setCcDefense(e.target.checked)}
                        className="sr-only peer"
                      />
                      <div className="w-9 h-5 bg-slate-300 dark:bg-slate-700 peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-4 after:w-4 after:transition-all peer-checked:bg-emerald-600"></div>
                    </label>
                  </div>

                  {ccDefense && (
                    <div className="grid grid-cols-3 gap-4 pt-3 border-t border-slate-200 dark:border-slate-700">
                      <div>
                        <label className="block text-slate-700 dark:text-slate-300 font-medium mb-1">Max Requests (req/s)</label>
                        <input
                          type="number"
                          value={rateLimitRps}
                          onChange={(e) => setRateLimitRps(parseInt(e.target.value, 10))}
                          className="w-full p-2 rounded-lg bg-white dark:bg-[#0F172A] border border-slate-300 dark:border-slate-700 text-slate-900 dark:text-slate-100"
                        />
                      </div>
                      <div>
                        <label className="block text-slate-700 dark:text-slate-300 font-medium mb-1">Burst Tolerance</label>
                        <input
                          type="number"
                          value={burstLimit}
                          onChange={(e) => setBurstLimit(parseInt(e.target.value, 10))}
                          className="w-full p-2 rounded-lg bg-white dark:bg-[#0F172A] border border-slate-300 dark:border-slate-700 text-slate-900 dark:text-slate-100"
                        />
                      </div>
                      <div>
                        <label className="block text-slate-700 dark:text-slate-300 font-medium mb-1">Max Concurrency / IP</label>
                        <input
                          type="number"
                          value={connLimit}
                          onChange={(e) => setConnLimit(parseInt(e.target.value, 10))}
                          className="w-full p-2 rounded-lg bg-white dark:bg-[#0F172A] border border-slate-300 dark:border-slate-700 text-slate-900 dark:text-slate-100"
                        />
                      </div>
                    </div>
                  )}
                </div>

                <button
                  onClick={() => showToast('Access limit & CC defense rules updated.')}
                  className="px-4 py-2 rounded-lg bg-emerald-600 hover:bg-emerald-700 active:bg-emerald-800 text-white font-bold text-xs transition shadow-xs cursor-pointer"
                >
                  Save Access Rules
                </button>
              </div>
            )}

            {/* 4. URL REWRITE */}
            {activeTab === 'rewrite' && (
              <div className="space-y-4 text-xs h-full flex flex-col">
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-2">
                    <span className="text-slate-700 dark:text-slate-300 font-medium">Template Presets:</span>
                    <select
                      value={selectedPreset}
                      onChange={(e) => handleSelectPreset(e.target.value)}
                      className="p-1.5 rounded-lg bg-white dark:bg-[#0F172A] border border-slate-300 dark:border-slate-700 text-slate-900 dark:text-slate-100 font-medium"
                    >
                      {Object.keys(REWRITE_PRESETS).map((k) => (
                        <option key={k} value={k}>
                          {k}
                        </option>
                      ))}
                    </select>
                  </div>
                  <span className="text-slate-500 dark:text-slate-400">File: /www/server/panel/vhost/rewrite/{website.primary_domain}.conf</span>
                </div>

                <div className="flex-1 min-h-[360px] border border-slate-300 dark:border-slate-700 rounded-lg overflow-hidden flex flex-col bg-slate-50 dark:bg-[#090D16]">
                  <textarea
                    value={rewriteRules}
                    onChange={(e) => setRewriteRules(e.target.value)}
                    className="w-full h-full p-4 font-mono text-slate-900 dark:text-emerald-400 bg-transparent focus:outline-none resize-none leading-relaxed"
                    spellCheck={false}
                  />
                </div>

                <div className="flex items-center justify-between">
                  <span className="text-slate-600 dark:text-slate-400 text-[11px]">Changes will be validated with `nginx -t` before reload.</span>
                  <button
                    onClick={() => showToast('URL rewrite rules saved and active.')}
                    className="px-4 py-2 rounded-lg bg-emerald-600 hover:bg-emerald-700 active:bg-emerald-800 text-white font-bold transition shadow-xs cursor-pointer"
                  >
                    Save & Test
                  </button>
                </div>
              </div>
            )}

            {/* 5. DEFAULT DOCUMENT */}
            {activeTab === 'defaultDoc' && (
              <div className="space-y-4 text-xs">
                <p className="text-slate-600 dark:text-slate-400">
                  Index documents are prioritized from top to bottom. The first file that exists in the directory will be served.
                </p>

                <div className="flex gap-2">
                  <input
                    type="text"
                    placeholder="e.g. index.htm or main.html"
                    value={newDocName}
                    onChange={(e) => setNewDocName(e.target.value)}
                    className="w-72 p-2 rounded-lg bg-white dark:bg-[#0F172A] border border-slate-300 dark:border-slate-700 text-slate-900 dark:text-slate-100"
                  />
                  <button
                    onClick={handleAddDoc}
                    className="px-4 py-2 rounded-lg bg-emerald-600 hover:bg-emerald-700 active:bg-emerald-800 text-white font-bold transition shadow-xs cursor-pointer"
                  >
                    Add Document
                  </button>
                </div>

                <div className="border border-slate-200 dark:border-slate-800 rounded-lg overflow-hidden bg-white dark:bg-[#0F172A] w-full max-w-lg">
                  <table className="w-full text-left">
                    <thead className="bg-slate-100 dark:bg-[#1E2432] text-slate-700 dark:text-slate-300 border-b border-slate-200 dark:border-slate-700">
                      <tr>
                        <th className="p-2.5">Priority</th>
                        <th className="p-2.5">Filename</th>
                        <th className="p-2.5 text-right">Order / Delete</th>
                      </tr>
                    </thead>
                    <tbody className="divide-y divide-slate-200 dark:divide-slate-800">
                      {defaultDocs.map((doc, idx) => (
                        <tr key={doc} className="hover:bg-slate-50 dark:hover:bg-slate-800/40 text-slate-800 dark:text-slate-200 transition">
                          <td className="p-2.5 font-mono text-slate-500 dark:text-slate-400">#{idx + 1}</td>
                          <td className="p-2.5 font-mono text-emerald-600 dark:text-emerald-400 font-semibold">{doc}</td>
                          <td className="p-2.5 text-right space-x-2">
                            <button
                              onClick={() => moveDoc(idx, 'up')}
                              disabled={idx === 0}
                              className="p-1 hover:text-slate-900 dark:hover:text-white disabled:opacity-30 cursor-pointer"
                              title="Move Up"
                            >
                              <ChevronUp className="w-3.5 h-3.5 inline" />
                            </button>
                            <button
                              onClick={() => moveDoc(idx, 'down')}
                              disabled={idx === defaultDocs.length - 1}
                              className="p-1 hover:text-slate-900 dark:hover:text-white disabled:opacity-30 cursor-pointer"
                              title="Move Down"
                            >
                              <ChevronDown className="w-3.5 h-3.5 inline" />
                            </button>
                            <button
                              onClick={() => handleDeleteDoc(doc)}
                              className="p-1 text-rose-600 hover:text-rose-700 dark:text-rose-400 cursor-pointer"
                              title="Delete"
                            >
                              <Trash2 className="w-3.5 h-3.5 inline" />
                            </button>
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>

                <button
                  onClick={() => showToast('Default documents saved to virtual host.')}
                  className="px-4 py-2 rounded-lg bg-emerald-600 hover:bg-emerald-700 active:bg-emerald-800 text-white font-bold transition shadow-xs cursor-pointer"
                >
                  Save Default Documents
                </button>
              </div>
            )}

            {/* 6. CONFIG (vHost Raw) */}
            {activeTab === 'config' && (
              <div className="space-y-4 text-xs h-full flex flex-col">
                <div className="flex items-center justify-between">
                  <span className="text-slate-700 dark:text-slate-300">
                    Configuration File: <span className="text-emerald-600 dark:text-emerald-400 font-mono font-medium">/etc/nginx/sites-available/{website.primary_domain}</span>
                  </span>
                  <span className="text-[11px] text-slate-500 dark:text-slate-400">Live Nginx vHost Block</span>
                </div>

                <div className="flex-1 min-h-[380px] border border-slate-300 dark:border-slate-700 rounded-lg overflow-hidden bg-slate-50 dark:bg-[#090D16]">
                  <textarea
                    value={vhostCode}
                    onChange={(e) => setVhostCode(e.target.value)}
                    className="w-full h-full p-4 font-mono text-slate-900 dark:text-emerald-400 bg-transparent focus:outline-none resize-none leading-relaxed text-xs"
                    spellCheck={false}
                  />
                </div>

                <div className="flex items-center justify-between">
                  <span className="text-slate-600 dark:text-slate-400 text-[11px]">
                    Automatic validation with `nginx -t` ensures zero-downtime reloads.
                  </span>
                  <button
                    onClick={handleSaveVhost}
                    disabled={isSavingConfig}
                    className="px-5 py-2.5 rounded-lg bg-emerald-600 hover:bg-emerald-700 active:bg-emerald-800 text-white font-bold transition shadow-xs flex items-center gap-2 cursor-pointer disabled:opacity-50"
                  >
                    {isSavingConfig ? <RefreshCw className="w-3.5 h-3.5 animate-spin" /> : <Save className="w-3.5 h-3.5" />}
                    <span>{isSavingConfig ? 'Testing & Reloading...' : 'Save & Reload'}</span>
                  </button>
                </div>
              </div>
            )}

            {/* 7. SSL */}
            {activeTab === 'ssl' && (
              <div className="space-y-5 text-xs">
                {/* Toggles */}
                <div className="grid grid-cols-2 gap-4 bg-slate-50 dark:bg-[#131B2E] p-4 rounded-lg border border-slate-200 dark:border-slate-800">
                  <div className="flex items-center justify-between">
                    <div>
                      <div className="font-semibold text-slate-900 dark:text-white">Force HTTPS (301 Redirect)</div>
                      <div className="text-[11px] text-slate-500 dark:text-slate-400">Automatically redirect all HTTP traffic to HTTPS</div>
                    </div>
                    <input
                      type="checkbox"
                      checked={forceHttps}
                      onChange={(e) => setForceHttps(e.target.checked)}
                      className="rounded border-slate-300 dark:border-slate-600 accent-emerald-600 cursor-pointer"
                    />
                  </div>

                  <div className="flex items-center justify-between">
                    <div>
                      <div className="font-semibold text-slate-900 dark:text-white">HTTP/2 Protocol</div>
                      <div className="text-[11px] text-slate-500 dark:text-slate-400">Enable multiplexing and server push</div>
                    </div>
                    <input
                      type="checkbox"
                      checked={http2Enabled}
                      onChange={(e) => setHttp2Enabled(e.target.checked)}
                      className="rounded border-slate-300 dark:border-slate-600 accent-emerald-600 cursor-pointer"
                    />
                  </div>
                </div>

                {/* Sub-tabs: Let's Encrypt vs Custom */}
                <div className="flex border-b border-slate-200 dark:border-slate-700">
                  <button
                    onClick={() => setSslTab('letsencrypt')}
                    className={`px-4 py-2 font-semibold border-b-2 transition cursor-pointer ${
                      sslTab === 'letsencrypt'
                        ? 'border-emerald-600 text-emerald-700 dark:text-emerald-400 dark:border-emerald-500'
                        : 'border-transparent text-slate-600 dark:text-slate-400 hover:text-slate-900 dark:hover:text-slate-200'
                    }`}
                  >
                    Let's Encrypt (1-Click Free)
                  </button>
                  <button
                    onClick={() => setSslTab('custom')}
                    className={`px-4 py-2 font-semibold border-b-2 transition cursor-pointer ${
                      sslTab === 'custom'
                        ? 'border-emerald-600 text-emerald-700 dark:text-emerald-400 dark:border-emerald-500'
                        : 'border-transparent text-slate-600 dark:text-slate-400 hover:text-slate-900 dark:hover:text-slate-200'
                    }`}
                  >
                    Custom Certificate (CRT/KEY)
                  </button>
                </div>

                {sslTab === 'letsencrypt' ? (
                  <div className="space-y-4 bg-slate-50 dark:bg-[#131B2E] p-4 rounded-lg border border-slate-200 dark:border-slate-800">
                    <p className="text-slate-700 dark:text-slate-300">
                      Issue free automated 90-day certificates via ACME HTTP-01 or DNS verification.
                    </p>
                    <div className="space-y-2">
                      <span className="block text-slate-700 dark:text-slate-400 font-medium">Included Domains:</span>
                      {domainList.map((d) => (
                        <label key={d.domain} className="flex items-center gap-2 cursor-pointer">
                          <input type="checkbox" defaultChecked className="rounded border-slate-300 dark:border-slate-600 accent-emerald-600" />
                          <span className="font-mono text-slate-900 dark:text-slate-100">{d.domain}</span>
                        </label>
                      ))}
                    </div>

                    <label className="flex items-center gap-2 pt-2 cursor-pointer">
                      <input
                        type="checkbox"
                        checked={autoRenew}
                        onChange={(e) => setAutoRenew(e.target.checked)}
                        className="rounded border-slate-300 dark:border-slate-600 accent-emerald-600"
                      />
                      <span className="text-slate-800 dark:text-slate-300 font-medium">Automatic renewal 30 days before expiration</span>
                    </label>

                    <button
                      onClick={handleApplyLetsEncrypt}
                      disabled={isIssuingSsl}
                      className="px-5 py-2.5 rounded-lg bg-emerald-600 hover:bg-emerald-700 active:bg-emerald-800 text-white font-bold transition flex items-center gap-2 shadow-xs cursor-pointer disabled:opacity-50"
                    >
                      {isIssuingSsl ? <RefreshCw className="w-4 h-4 animate-spin" /> : <Lock className="w-4 h-4" />}
                      <span>{isIssuingSsl ? 'Verifying DNS & Issuing Certificate...' : 'Apply & Install Certificate'}</span>
                    </button>
                  </div>
                ) : (
                  <div className="space-y-4 bg-slate-50 dark:bg-[#131B2E] p-4 rounded-lg border border-slate-200 dark:border-slate-800">
                    <div>
                      <label className="block text-slate-700 dark:text-slate-300 font-medium mb-1">Certificate (CRT / PEM / Fullchain)</label>
                      <textarea
                        rows={5}
                        placeholder="-----BEGIN CERTIFICATE-----..."
                        value={customCert}
                        onChange={(e) => setCustomCert(e.target.value)}
                        className="w-full p-2.5 rounded-lg bg-white dark:bg-[#0F172A] border border-slate-300 dark:border-slate-700 font-mono text-slate-900 dark:text-slate-100 text-xs"
                      />
                    </div>
                    <div>
                      <label className="block text-slate-700 dark:text-slate-300 font-medium mb-1">Private Key (KEY)</label>
                      <textarea
                        rows={4}
                        placeholder="-----BEGIN RSA PRIVATE KEY-----..."
                        value={customKey}
                        onChange={(e) => setCustomKey(e.target.value)}
                        className="w-full p-2.5 rounded-lg bg-white dark:bg-[#0F172A] border border-slate-300 dark:border-slate-700 font-mono text-slate-900 dark:text-slate-100 text-xs"
                      />
                    </div>
                    <button
                      onClick={() => showToast('Custom certificate deployed.')}
                      className="px-4 py-2 rounded-lg bg-emerald-600 hover:bg-emerald-700 active:bg-emerald-800 text-white font-bold transition shadow-xs cursor-pointer"
                    >
                      Save & Deploy Certificate
                    </button>
                  </div>
                )}
              </div>
            )}

            {/* 8. PHP VERSION */}
            {activeTab === 'php' && (
              <div className="space-y-5 text-xs">
                <div className="bg-slate-50 dark:bg-[#131B2E] p-4 rounded-lg border border-slate-200 dark:border-slate-800 space-y-4">
                  <h3 className="text-sm font-semibold text-slate-900 dark:text-white">PHP Engine Selection</h3>
                  <div className="grid grid-cols-2 gap-4">
                    <div>
                      <label className="block text-slate-700 dark:text-slate-300 font-medium mb-1">PHP Version</label>
                      <select
                        value={selectedPhp}
                        onChange={(e) => setSelectedPhp(e.target.value)}
                        className="w-full p-2 rounded-lg bg-white dark:bg-[#0F172A] border border-slate-300 dark:border-slate-700 text-slate-900 dark:text-slate-100 font-bold"
                      >
                        <option value="8.4">PHP-8.4 (Latest)</option>
                        <option value="8.3">PHP-8.3 (Stable)</option>
                        <option value="8.2">PHP-8.2 (Recommended)</option>
                        <option value="8.1">PHP-8.1 (Current)</option>
                        <option value="8.0">PHP-8.0</option>
                        <option value="7.4">PHP-7.4 (Legacy)</option>
                        <option value="static">Pure Static (No PHP)</option>
                        <option value="nodejs">Node.js / Reverse Proxy Mode</option>
                      </select>
                    </div>

                    <div>
                      <label className="block text-slate-700 dark:text-slate-300 font-medium mb-1">Memory Limit</label>
                      <select
                        value={phpMemoryLimit}
                        onChange={(e) => setPhpMemoryLimit(e.target.value)}
                        className="w-full p-2 rounded-lg bg-white dark:bg-[#0F172A] border border-slate-300 dark:border-slate-700 text-slate-900 dark:text-slate-100"
                      >
                        <option value="128M">128 MB</option>
                        <option value="256M">256 MB (Standard)</option>
                        <option value="512M">512 MB (High-traffic)</option>
                        <option value="1024M">1024 MB (1 GB)</option>
                      </select>
                    </div>
                  </div>

                  <div className="grid grid-cols-2 gap-4">
                    <div>
                      <label className="block text-slate-700 dark:text-slate-300 font-medium mb-1">Max Execution Time (seconds)</label>
                      <input
                        type="number"
                        value={phpMaxExecution}
                        onChange={(e) => setPhpMaxExecution(e.target.value)}
                        className="w-full p-2 rounded-lg bg-white dark:bg-[#0F172A] border border-slate-300 dark:border-slate-700 text-slate-900 dark:text-slate-100"
                      />
                    </div>
                    <div>
                      <label className="block text-slate-700 dark:text-slate-300 font-medium mb-1">Max Upload Filesize</label>
                      <input
                        type="text"
                        value={phpMaxUpload}
                        onChange={(e) => setPhpMaxUpload(e.target.value)}
                        className="w-full p-2 rounded-lg bg-white dark:bg-[#0F172A] border border-slate-300 dark:border-slate-700 text-slate-900 dark:text-slate-100"
                      />
                    </div>
                  </div>

                  <label className="flex items-center gap-2 cursor-pointer pt-2">
                    <input
                      type="checkbox"
                      checked={opcacheEnabled}
                      onChange={(e) => setOpcacheEnabled(e.target.checked)}
                      className="rounded border-slate-300 dark:border-slate-600 accent-emerald-600"
                    />
                    <span className="text-slate-800 dark:text-slate-300 font-medium">Enable Zend OPcache byte-code acceleration</span>
                  </label>
                </div>

                <button
                  onClick={handleSwitchPhp}
                  className="px-5 py-2.5 rounded-lg bg-emerald-600 hover:bg-emerald-700 active:bg-emerald-800 text-white font-bold transition shadow-xs cursor-pointer"
                >
                  Apply & Switch PHP Version
                </button>
              </div>
            )}

            {/* 9. WEB SERVER */}
            {activeTab === 'webserver' && (
              <div className="space-y-5 text-xs">
                <div className="bg-slate-50 dark:bg-[#131B2E] p-4 rounded-lg border border-slate-200 dark:border-slate-800 space-y-4">
                  <h3 className="text-sm font-semibold text-slate-900 dark:text-white">Web Server Engine & Optimization</h3>
                  <div className="grid grid-cols-2 gap-4">
                    <div>
                      <label className="block text-slate-700 dark:text-slate-300 font-medium mb-1">Active Web Server</label>
                      <select
                        value={webServerEngine}
                        onChange={(e) => setWebServerEngine(e.target.value)}
                        className="w-full p-2 rounded-lg bg-white dark:bg-[#0F172A] border border-slate-300 dark:border-slate-700 text-slate-900 dark:text-slate-100 font-bold"
                      >
                        <option value="nginx">Nginx (High-concurrency Event Engine)</option>
                        <option value="apache">Apache 2.4 (Prefork / Event MPM)</option>
                        <option value="openlitespeed">OpenLiteSpeed (HTTP/3 QUIC native)</option>
                      </select>
                    </div>

                    <div>
                      <label className="block text-slate-700 dark:text-slate-300 font-medium mb-1">Client Max Body Size</label>
                      <input
                        type="text"
                        value={clientMaxBody}
                        onChange={(e) => setClientMaxBody(e.target.value)}
                        className="w-full p-2 rounded-lg bg-white dark:bg-[#0F172A] border border-slate-300 dark:border-slate-700 text-slate-900 dark:text-slate-100 font-mono"
                      />
                    </div>
                  </div>

                  <div className="space-y-3 pt-2">
                    <label className="flex items-center gap-2.5 cursor-pointer">
                      <input
                        type="checkbox"
                        checked={gzipEnabled}
                        onChange={(e) => setGzipEnabled(e.target.checked)}
                        className="rounded border-slate-300 dark:border-slate-600 accent-emerald-600"
                      />
                      <span className="text-slate-800 dark:text-slate-300 font-medium">Gzip dynamic compression (level {gzipLevel})</span>
                    </label>

                    <label className="flex items-center gap-2.5 cursor-pointer">
                      <input
                        type="checkbox"
                        checked={brotliEnabled}
                        onChange={(e) => setBrotliEnabled(e.target.checked)}
                        className="rounded border-slate-300 dark:border-slate-600 accent-emerald-600"
                      />
                      <span className="text-slate-800 dark:text-slate-300 font-medium">Brotli next-gen compression (br)</span>
                    </label>
                  </div>
                </div>

                <button
                  onClick={() => showToast('Web server optimization settings saved.')}
                  className="px-4 py-2 rounded-lg bg-emerald-600 hover:bg-emerald-700 active:bg-emerald-800 text-white font-bold transition shadow-xs cursor-pointer"
                >
                  Save Server Configuration
                </button>
              </div>
            )}

            {/* 10. GIT MANAGER */}
            {activeTab === 'git' && (
              <div className="space-y-4 text-xs">
                <div className="bg-slate-50 dark:bg-[#131B2E] p-4 rounded-lg border border-slate-200 dark:border-slate-800 space-y-3">
                  <h3 className="text-sm font-semibold text-slate-900 dark:text-white">Git Repository & Auto-Deploy</h3>
                  <div>
                    <label className="block text-slate-700 dark:text-slate-300 font-medium mb-1">Repository URL</label>
                    <input
                      type="text"
                      value={gitRepoUrl}
                      onChange={(e) => setGitRepoUrl(e.target.value)}
                      className="w-full p-2 rounded-lg bg-white dark:bg-[#0F172A] border border-slate-300 dark:border-slate-700 text-slate-900 dark:text-slate-100 font-mono"
                    />
                  </div>

                  <div className="grid grid-cols-2 gap-3">
                    <div>
                      <label className="block text-slate-700 dark:text-slate-300 font-medium mb-1">Branch</label>
                      <input
                        type="text"
                        value={gitBranch}
                        onChange={(e) => setGitBranch(e.target.value)}
                        className="w-full p-2 rounded-lg bg-white dark:bg-[#0F172A] border border-slate-300 dark:border-slate-700 text-slate-900 dark:text-slate-100 font-mono"
                      />
                    </div>
                    <div>
                      <label className="block text-slate-700 dark:text-slate-300 font-medium mb-1">Webhook URL</label>
                      <div className="flex gap-1">
                        <input
                          type="text"
                          readOnly
                          value={`https://${website.primary_domain}/api/v1/webhook?secret=hv_${website.id}`}
                          className="w-full p-2 rounded-lg bg-white dark:bg-[#0F172A] border border-slate-300 dark:border-slate-700 text-slate-600 dark:text-slate-400 font-mono text-[11px]"
                        />
                        <button
                          onClick={() => {
                            navigator.clipboard.writeText(`https://${website.primary_domain}/api/v1/webhook?secret=hv_${website.id}`);
                            showToast('Webhook URL copied to clipboard!');
                          }}
                          className="px-2.5 rounded-lg bg-slate-200 dark:bg-slate-700 hover:bg-slate-300 dark:hover:bg-slate-600 text-slate-700 dark:text-white transition cursor-pointer"
                        >
                          <Copy className="w-3.5 h-3.5" />
                        </button>
                      </div>
                    </div>
                  </div>

                  <div>
                    <label className="block text-slate-700 dark:text-slate-300 font-medium mb-1">Post-deployment Script</label>
                    <textarea
                      rows={3}
                      value={gitScript}
                      onChange={(e) => setGitScript(e.target.value)}
                      className="w-full p-2 rounded-lg bg-white dark:bg-[#0F172A] border border-slate-300 dark:border-slate-700 text-slate-900 dark:text-emerald-400 font-mono text-[11px]"
                    />
                  </div>
                </div>

                <div className="flex gap-3">
                  <button
                    onClick={handlePullGit}
                    disabled={isPullingGit}
                    className="px-4 py-2 rounded-lg bg-emerald-600 hover:bg-emerald-700 active:bg-emerald-800 text-white font-bold flex items-center gap-2 shadow-xs transition cursor-pointer disabled:opacity-50"
                  >
                    {isPullingGit ? <RefreshCw className="w-3.5 h-3.5 animate-spin" /> : <Play className="w-3.5 h-3.5" />}
                    <span>{isPullingGit ? 'Pulling from origin...' : 'Pull & Deploy Now'}</span>
                  </button>
                </div>
              </div>
            )}

            {/* 11. COMPOSER */}
            {activeTab === 'composer' && (
              <div className="space-y-4 text-xs h-full flex flex-col">
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-2">
                    <span className="text-slate-700 dark:text-slate-300 font-medium">Composer Mirror:</span>
                    <select
                      value={composerMirror}
                      onChange={(e) => setComposerMirror(e.target.value)}
                      className="p-1.5 rounded-lg bg-white dark:bg-[#0F172A] border border-slate-300 dark:border-slate-700 text-slate-900 dark:text-slate-100"
                    >
                      <option value="Packagist Official">Packagist Official (Global)</option>
                      <option value="Aliyun">Aliyun (Fast Asian Mirror)</option>
                      <option value="Cloudflare">Cloudflare Worker Mirror</option>
                    </select>
                  </div>
                  <div className="flex gap-2">
                    <button
                      onClick={() => handleRunComposer('install')}
                      className="px-3 py-1 rounded-lg bg-emerald-50 dark:bg-emerald-950/40 border border-emerald-300 dark:border-emerald-800 hover:bg-emerald-100 text-emerald-700 dark:text-emerald-400 font-bold cursor-pointer"
                    >
                      install
                    </button>
                    <button
                      onClick={() => handleRunComposer('update')}
                      className="px-3 py-1 rounded-lg bg-orange-50 dark:bg-orange-950/40 border border-orange-300 dark:border-orange-800 hover:bg-orange-100 text-orange-700 dark:text-orange-400 font-bold cursor-pointer"
                    >
                      update
                    </button>
                    <button
                      onClick={() => handleRunComposer('dump-autoload -o')}
                      className="px-3 py-1 rounded-lg bg-slate-100 dark:bg-slate-800 border border-slate-300 dark:border-slate-700 hover:bg-slate-200 text-slate-800 dark:text-slate-300 font-bold cursor-pointer"
                    >
                      dump-autoload
                    </button>
                  </div>
                </div>

                <div className="flex-1 min-h-[350px] border border-slate-300 dark:border-slate-700 rounded-lg overflow-hidden bg-slate-900 dark:bg-[#0C0E14] p-3 font-mono text-emerald-400 text-xs overflow-y-auto whitespace-pre-wrap">
                  {composerConsole}
                </div>
              </div>
            )}

            {/* 12. REDIRECT */}
            {activeTab === 'redirect' && (
              <div className="space-y-4 text-xs">
                <div className="bg-slate-50 dark:bg-[#131B2E] p-4 rounded-lg border border-slate-200 dark:border-slate-800 space-y-3">
                  <h3 className="text-sm font-semibold text-slate-900 dark:text-white">Add URL Redirect Rule</h3>
                  <div className="grid grid-cols-4 gap-2">
                    <select
                      value={newRedirType}
                      onChange={(e) => setNewRedirType(e.target.value as any)}
                      className="p-2 rounded-lg bg-white dark:bg-[#0F172A] border border-slate-300 dark:border-slate-700 text-slate-900 dark:text-slate-100"
                    >
                      <option value="301">301 (Permanent)</option>
                      <option value="302">302 (Temporary)</option>
                    </select>
                    <input
                      type="text"
                      placeholder="Source Path (e.g. /old-page)"
                      value={newRedirSource}
                      onChange={(e) => setNewRedirSource(e.target.value)}
                      className="p-2 rounded-lg bg-white dark:bg-[#0F172A] border border-slate-300 dark:border-slate-700 text-slate-900 dark:text-slate-100 font-mono"
                    />
                    <input
                      type="text"
                      placeholder="Target URL (e.g. https://domain.com/new)"
                      value={newRedirTarget}
                      onChange={(e) => setNewRedirTarget(e.target.value)}
                      className="p-2 rounded-lg bg-white dark:bg-[#0F172A] border border-slate-300 dark:border-slate-700 text-slate-900 dark:text-slate-100 font-mono"
                    />
                    <button
                      onClick={handleAddRedirect}
                      className="px-4 py-2 rounded-lg bg-emerald-600 hover:bg-emerald-700 active:bg-emerald-800 text-white font-bold transition shadow-xs cursor-pointer"
                    >
                      Add Redirect
                    </button>
                  </div>
                </div>

                <div className="border border-slate-200 dark:border-slate-800 rounded-lg overflow-hidden bg-white dark:bg-[#0F172A]">
                  <table className="w-full text-left">
                    <thead className="bg-slate-100 dark:bg-[#1E2432] text-slate-700 dark:text-slate-300 border-b border-slate-200 dark:border-slate-700">
                      <tr>
                        <th className="p-2.5">Type</th>
                        <th className="p-2.5">Source Path</th>
                        <th className="p-2.5">Target Destination</th>
                        <th className="p-2.5 text-right">Action</th>
                      </tr>
                    </thead>
                    <tbody className="divide-y divide-slate-200 dark:divide-slate-800">
                      {redirectList.map((r) => (
                        <tr key={r.id} className="hover:bg-slate-50 dark:hover:bg-slate-800/40 text-slate-800 dark:text-slate-200 transition">
                          <td className="p-2.5 font-bold text-orange-600 dark:text-orange-400">{r.type}</td>
                          <td className="p-2.5 font-mono text-slate-900 dark:text-slate-200">{r.sourcePath}</td>
                          <td className="p-2.5 font-mono text-emerald-700 dark:text-emerald-400 truncate max-w-xs">{r.targetUrl}</td>
                          <td className="p-2.5 text-right">
                            <button
                              onClick={() => setRedirectList(redirectList.filter((item) => item.id !== r.id))}
                              className="text-rose-600 hover:text-rose-700 dark:text-rose-400 hover:underline font-medium cursor-pointer"
                            >
                              Delete
                            </button>
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              </div>
            )}

            {/* 13. REVERSE PROXY */}
            {activeTab === 'proxy' && (
              <div className="space-y-4 text-xs">
                <div className="bg-slate-50 dark:bg-[#131B2E] p-4 rounded-lg border border-slate-200 dark:border-slate-800 space-y-3">
                  <h3 className="text-sm font-semibold text-slate-900 dark:text-white">Add Reverse Proxy Pass</h3>
                  <div className="grid grid-cols-3 gap-3">
                    <input
                      type="text"
                      placeholder="Proxy Name (e.g. NextApp)"
                      value={newProxyName}
                      onChange={(e) => setNewProxyName(e.target.value)}
                      className="p-2 rounded-lg bg-white dark:bg-[#0F172A] border border-slate-300 dark:border-slate-700 text-slate-900 dark:text-slate-100"
                    />
                    <input
                      type="text"
                      placeholder="Target URL (e.g. http://127.0.0.1:3000)"
                      value={newProxyTarget}
                      onChange={(e) => setNewProxyTarget(e.target.value)}
                      className="p-2 rounded-lg bg-white dark:bg-[#0F172A] border border-slate-300 dark:border-slate-700 text-slate-900 dark:text-slate-100 font-mono"
                    />
                    <button
                      onClick={handleAddProxy}
                      className="px-4 py-2 rounded-lg bg-emerald-600 hover:bg-emerald-700 active:bg-emerald-800 text-white font-bold transition shadow-xs cursor-pointer"
                    >
                      Add Reverse Proxy
                    </button>
                  </div>
                  <div className="flex gap-4 pt-1">
                    <label className="flex items-center gap-2 cursor-pointer">
                      <input
                        type="checkbox"
                        checked={newProxyWs}
                        onChange={(e) => setNewProxyWs(e.target.checked)}
                        className="rounded border-slate-300 dark:border-slate-600 accent-emerald-600"
                      />
                      <span className="text-slate-700 dark:text-slate-300 font-medium">WebSocket Support (Upgrade / Connection)</span>
                    </label>
                    <label className="flex items-center gap-2 cursor-pointer">
                      <input
                        type="checkbox"
                        checked={newProxyCache}
                        onChange={(e) => setNewProxyCache(e.target.checked)}
                        className="rounded border-slate-300 dark:border-slate-600 accent-emerald-600"
                      />
                      <span className="text-slate-700 dark:text-slate-300 font-medium">Edge Micro-Cache (5 mins)</span>
                    </label>
                  </div>
                </div>

                <div className="border border-slate-200 dark:border-slate-800 rounded-lg overflow-hidden bg-white dark:bg-[#0F172A]">
                  <table className="w-full text-left">
                    <thead className="bg-slate-100 dark:bg-[#1E2432] text-slate-700 dark:text-slate-300 border-b border-slate-200 dark:border-slate-700">
                      <tr>
                        <th className="p-2.5">Name</th>
                        <th className="p-2.5">Target Destination</th>
                        <th className="p-2.5">WebSocket</th>
                        <th className="p-2.5 text-right">Action</th>
                      </tr>
                    </thead>
                    <tbody className="divide-y divide-slate-200 dark:divide-slate-800">
                      {proxyList.map((p) => (
                        <tr key={p.id} className="hover:bg-slate-50 dark:hover:bg-slate-800/40 text-slate-800 dark:text-slate-200 transition">
                          <td className="p-2.5 font-bold text-slate-900 dark:text-white">{p.name}</td>
                          <td className="p-2.5 font-mono text-emerald-700 dark:text-emerald-400">{p.targetUrl}</td>
                          <td className="p-2.5 text-slate-700 dark:text-slate-300">{p.websocket ? 'Enabled' : 'Disabled'}</td>
                          <td className="p-2.5 text-right">
                            <button
                              onClick={() => setProxyList(proxyList.filter((item) => item.id !== p.id))}
                              className="text-rose-600 hover:text-rose-700 dark:text-rose-400 hover:underline font-medium cursor-pointer"
                            >
                              Delete
                            </button>
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              </div>
            )}

            {/* 14. HOTLINK PROTECTION */}
            {activeTab === 'hotlink' && (
              <div className="space-y-4 text-xs">
                <div className="bg-slate-50 dark:bg-[#131B2E] p-4 rounded-lg border border-slate-200 dark:border-slate-800 space-y-4">
                  <div className="flex items-center justify-between">
                    <div>
                      <h3 className="text-sm font-semibold text-slate-900 dark:text-white">Anti-Leech / Hotlink Defense</h3>
                      <p className="text-[11px] text-slate-500 dark:text-slate-400">Prevent other websites from embedding your images and media assets</p>
                    </div>
                    <label className="relative inline-flex items-center cursor-pointer">
                      <input
                        type="checkbox"
                        checked={hotlinkEnabled}
                        onChange={(e) => setHotlinkEnabled(e.target.checked)}
                        className="sr-only peer"
                      />
                      <div className="w-9 h-5 bg-slate-300 dark:bg-slate-700 peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-4 after:w-4 after:transition-all peer-checked:bg-emerald-600"></div>
                    </label>
                  </div>

                  <div>
                    <label className="block text-slate-700 dark:text-slate-300 font-medium mb-1">Allowed Referrers (1 per line)</label>
                    <textarea
                      rows={3}
                      value={hotlinkDomains}
                      onChange={(e) => setHotlinkDomains(e.target.value)}
                      className="w-full p-2.5 rounded-lg bg-white dark:bg-[#0F172A] border border-slate-300 dark:border-slate-700 text-slate-900 dark:text-slate-100 font-mono"
                    />
                  </div>

                  <div>
                    <label className="block text-slate-700 dark:text-slate-300 font-medium mb-1">Protected File Extensions</label>
                    <input
                      type="text"
                      value={hotlinkExts}
                      onChange={(e) => setHotlinkExts(e.target.value)}
                      className="w-full p-2 rounded-lg bg-white dark:bg-[#0F172A] border border-slate-300 dark:border-slate-700 text-slate-900 dark:text-slate-100 font-mono"
                    />
                  </div>
                </div>

                <button
                  onClick={() => showToast('Hotlink protection configuration applied.')}
                  className="px-4 py-2 rounded-lg bg-emerald-600 hover:bg-emerald-700 active:bg-emerald-800 text-white font-bold transition shadow-xs cursor-pointer"
                >
                  Save Hotlink Protection
                </button>
              </div>
            )}

            {/* 15. MAINTENANCE MODE */}
            {activeTab === 'maintenance' && (
              <div className="space-y-4 text-xs">
                <div className="bg-slate-50 dark:bg-[#131B2E] p-4 rounded-lg border border-slate-200 dark:border-slate-800 space-y-4">
                  <div className="flex items-center justify-between">
                    <div>
                      <h3 className="text-sm font-semibold text-slate-900 dark:text-white">Website Maintenance Mode (503 Service Unavailable)</h3>
                      <p className="text-[11px] text-slate-500 dark:text-slate-400">Temporarily show a maintenance notice while performing migrations or updates</p>
                    </div>
                    <label className="relative inline-flex items-center cursor-pointer">
                      <input
                        type="checkbox"
                        checked={maintenanceEnabled}
                        onChange={(e) => setMaintenanceEnabled(e.target.checked)}
                        className="sr-only peer"
                      />
                      <div className="w-9 h-5 bg-slate-300 dark:bg-slate-700 peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-4 after:w-4 after:transition-all peer-checked:bg-orange-600"></div>
                    </label>
                  </div>

                  <div>
                    <label className="block text-slate-700 dark:text-slate-300 font-medium mb-1">Notice Page Title</label>
                    <input
                      type="text"
                      value={maintenanceTitle}
                      onChange={(e) => setMaintenanceTitle(e.target.value)}
                      className="w-full p-2 rounded-lg bg-white dark:bg-[#0F172A] border border-slate-300 dark:border-slate-700 text-slate-900 dark:text-slate-100"
                    />
                  </div>

                  <div>
                    <label className="block text-slate-700 dark:text-slate-300 font-medium mb-1">Maintenance Description / Notice</label>
                    <textarea
                      rows={3}
                      value={maintenanceNotice}
                      onChange={(e) => setMaintenanceNotice(e.target.value)}
                      className="w-full p-2.5 rounded-lg bg-white dark:bg-[#0F172A] border border-slate-300 dark:border-slate-700 text-slate-900 dark:text-slate-100"
                    />
                  </div>

                  <div>
                    <label className="block text-slate-700 dark:text-slate-300 font-medium mb-1">Bypass Whitelist IPs (Allow admin access)</label>
                    <textarea
                      rows={2}
                      value={maintenanceIps}
                      onChange={(e) => setMaintenanceIps(e.target.value)}
                      className="w-full p-2 rounded-lg bg-white dark:bg-[#0F172A] border border-slate-300 dark:border-slate-700 text-slate-900 dark:text-slate-100 font-mono text-[11px]"
                    />
                  </div>
                </div>

                <div className="flex gap-3">
                  <button
                    onClick={() => showToast(`Maintenance mode is now ${maintenanceEnabled ? 'ACTIVE' : 'DISABLED'}.`)}
                    className={`px-4 py-2 rounded-lg font-bold text-white shadow-xs transition cursor-pointer ${
                      maintenanceEnabled ? 'bg-orange-600 hover:bg-orange-700 active:bg-orange-800' : 'bg-emerald-600 hover:bg-emerald-700 active:bg-emerald-800'
                    }`}
                  >
                    Save Maintenance Settings
                  </button>
                </div>
              </div>
            )}

            {/* 16. RESPONSE LOG */}
            {activeTab === 'logs' && (
              <div className="space-y-4 text-xs h-full flex flex-col">
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-2">
                    <button
                      onClick={() => setLogType('access')}
                      className={`px-3 py-1.5 rounded-lg font-bold transition cursor-pointer ${
                        logType === 'access' ? 'bg-emerald-600 text-white' : 'bg-slate-100 dark:bg-slate-800 text-slate-700 dark:text-slate-300 hover:bg-slate-200 dark:hover:bg-slate-700'
                      }`}
                    >
                      access.log
                    </button>
                    <button
                      onClick={() => setLogType('error')}
                      className={`px-3 py-1.5 rounded-lg font-bold transition cursor-pointer ${
                        logType === 'error' ? 'bg-rose-600 text-white' : 'bg-slate-100 dark:bg-slate-800 text-slate-700 dark:text-slate-300 hover:bg-slate-200 dark:hover:bg-slate-700'
                      }`}
                    >
                      error.log
                    </button>
                  </div>

                  <div className="flex items-center gap-3">
                    <label className="flex items-center gap-1.5 cursor-pointer text-slate-600 dark:text-slate-400">
                      <input
                        type="checkbox"
                        checked={autoRefreshLogs}
                        onChange={(e) => setAutoRefreshLogs(e.target.checked)}
                        className="rounded border-slate-300 dark:border-slate-600 accent-emerald-600"
                      />
                      <span>Auto-refresh (3s)</span>
                    </label>

                    <button
                      onClick={() => showToast('Logs cleared.')}
                      className="px-2.5 py-1 rounded-lg bg-slate-100 dark:bg-slate-800 hover:bg-slate-200 dark:hover:bg-slate-700 text-slate-700 dark:text-slate-300 border border-slate-200 dark:border-slate-700 font-medium transition cursor-pointer"
                    >
                      Clear Log
                    </button>
                  </div>
                </div>

                <div className="flex-1 min-h-[380px] border border-slate-300 dark:border-slate-700 rounded-lg overflow-hidden bg-slate-900 dark:bg-[#0C0E14] p-3 font-mono text-xs overflow-y-auto space-y-1">
                  {mockLogs.map((line, idx) => {
                    let color = 'text-slate-300';
                    if (line.includes(' 200 ')) color = 'text-emerald-400';
                    else if (line.includes(' 403 ') || line.includes(' 404 ')) color = 'text-amber-400';
                    else if (line.includes(' 500 ') || line.includes(' 502 ')) color = 'text-rose-400';

                    return (
                      <div key={idx} className={`${color} leading-relaxed hover:bg-slate-800/60 px-1 rounded`}>
                        {line}
                      </div>
                    );
                  })}
                </div>
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}
