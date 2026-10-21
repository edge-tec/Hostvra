'use client';

import React, { useState } from 'react';
import {
  Globe,
  Folder,
  FileCode,
  ShieldCheck,
  Terminal,
  Settings,
  Sparkles,
  ExternalLink,
  ChevronDown,
  Wrench,
  X,
  Check,
  FileText,
  AlertCircle
} from 'lucide-react';
import { FileItem } from '@/app/files/page';

interface DomainWebsite {
  id: string;
  domain: string;
  document_root: string;
  status: string;
}

interface WebsiteFileHubProps {
  currentPath: string;
  domains: DomainWebsite[];
  activeDomain: string;
  onDomainSelect: (domain: string, docRoot: string) => void;
  onNavigate: (path: string) => void;
  onQuickConfigFile: (filename: string) => void;
  onFixPermissions: (targetPath: string) => void;
  fixingPermissions?: boolean;
}

const COMMON_CONFIG_FILES = [
  { name: 'wp-config.php', label: 'WordPress Config', desc: 'Database host, credentials & security keys', type: 'php' },
  { name: '.htaccess', label: 'Apache / LiteSpeed Rules', desc: 'URL rewrites, redirects & security headers', type: 'config' },
  { name: '.env', label: 'Environment Variables', desc: 'Laravel, Node.js, Next.js secrets & DB URLs', type: 'env' },
  { name: 'robots.txt', label: 'Robots.txt', desc: 'Search engine bot crawler rules', type: 'text' },
  { name: 'nginx.conf', label: 'Nginx VirtualHost', desc: 'Server block, SSL certificates & reverse proxy', type: 'config' },
  { name: 'composer.json', label: 'Composer Packages', desc: 'PHP dependency definitions and autoloading', type: 'json' },
];

export const WebsiteFileHub: React.FC<WebsiteFileHubProps> = ({
  currentPath,
  domains,
  activeDomain,
  onDomainSelect,
  onNavigate,
  onQuickConfigFile,
  onFixPermissions,
  fixingPermissions = false,
}) => {
  const [configsModalOpen, setConfigsModalOpen] = useState(false);
  const [fixConfirmOpen, setFixConfirmOpen] = useState(false);

  const selectedDomainObj = domains.find((d) => d.domain === activeDomain);
  const docRoot = selectedDomainObj?.document_root || (currentPath.startsWith('/var/www') ? currentPath : '/var/www');

  return (
    <div className="bg-white dark:bg-surface-900 border-b border-slate-200 dark:border-surface-800 px-4 py-2 flex flex-wrap items-center justify-between gap-2.5 shadow-2xs">
      {/* Left: Domain Selector and Website Shortcuts */}
      <div className="flex items-center gap-2 flex-wrap">
        {/* Domain Switcher Pill */}
        <div className="flex items-center gap-1.5 px-3 py-1.5 rounded-xl bg-slate-50 dark:bg-surface-800 border border-slate-200 dark:border-surface-700 text-xs">
          <Globe className="w-4 h-4 text-emerald-600 shrink-0" />
          <span className="font-semibold text-slate-500 text-[11px] hidden sm:inline">Active Website:</span>
          <select
            value={activeDomain}
            onChange={(e) => {
              const dom = e.target.value;
              const matched = domains.find((d) => d.domain === dom);
              if (matched) {
                onDomainSelect(dom, matched.document_root);
              } else {
                onDomainSelect('', '/var/www');
              }
            }}
            className="bg-transparent text-slate-900 dark:text-white font-bold font-mono focus:outline-none cursor-pointer text-xs"
          >
            <option value="">All Websites / Server Root</option>
            {domains.map((d) => (
              <option key={d.domain} value={d.domain} className="bg-white dark:bg-surface-900 text-slate-900 dark:text-white">
                {d.domain}
              </option>
            ))}
          </select>
        </div>

        {/* Quick Directory Jump Pills */}
        {activeDomain && (
          <div className="flex items-center gap-1 flex-wrap text-xs">
            {/* Root */}
            <button
              type="button"
              onClick={() => onNavigate(docRoot)}
              className={`px-2.5 py-1 rounded-lg border font-semibold text-[11px] transition flex items-center gap-1 ${
                currentPath === docRoot
                  ? 'bg-emerald-600 text-white border-emerald-600 shadow-2xs'
                  : 'bg-white dark:bg-surface-850 border-slate-200 dark:border-surface-700 text-slate-700 dark:text-slate-300 hover:bg-slate-100'
              }`}
            >
              <Folder className="w-3 h-3 text-amber-500" />
              <span>Doc Root</span>
            </button>

            {/* public_html / public */}
            <button
              type="button"
              onClick={() => {
                const target = docRoot.endsWith('/public_html') ? docRoot : `${docRoot}/public_html`;
                onNavigate(target);
              }}
              className="px-2.5 py-1 rounded-lg bg-white dark:bg-surface-850 border border-slate-200 dark:border-surface-700 hover:bg-slate-100 text-slate-700 dark:text-slate-300 font-semibold text-[11px] transition flex items-center gap-1"
            >
              <span>public_html</span>
            </button>

            {/* wp-content */}
            <button
              type="button"
              onClick={() => onNavigate(`${docRoot}/wp-content`)}
              className="px-2.5 py-1 rounded-lg bg-white dark:bg-surface-850 border border-slate-200 dark:border-surface-700 hover:bg-slate-100 text-slate-700 dark:text-slate-300 font-semibold text-[11px] transition flex items-center gap-1"
            >
              <span className="text-purple-600 dark:text-purple-400 font-bold">wp-content</span>
            </button>

            {/* plugins */}
            <button
              type="button"
              onClick={() => onNavigate(`${docRoot}/wp-content/plugins`)}
              className="hidden md:flex px-2 py-1 rounded-lg bg-white dark:bg-surface-850 border border-slate-200 dark:border-surface-700 hover:bg-slate-100 text-slate-600 dark:text-slate-400 font-medium text-[11px] transition items-center gap-1"
            >
              <span>plugins</span>
            </button>

            {/* themes */}
            <button
              type="button"
              onClick={() => onNavigate(`${docRoot}/wp-content/themes`)}
              className="hidden md:flex px-2 py-1 rounded-lg bg-white dark:bg-surface-850 border border-slate-200 dark:border-surface-700 hover:bg-slate-100 text-slate-600 dark:text-slate-400 font-medium text-[11px] transition items-center gap-1"
            >
              <span>themes</span>
            </button>
          </div>
        )}
      </div>

      {/* Right: Quick Tools: Quick Configs & Fix Permissions */}
      <div className="flex items-center gap-2">
        {/* Quick Config Files Button */}
        <button
          type="button"
          onClick={() => setConfigsModalOpen(true)}
          className="px-3 py-1.5 rounded-xl bg-emerald-50 dark:bg-emerald-950/40 text-emerald-700 dark:text-emerald-300 border border-emerald-200 dark:border-emerald-800 hover:bg-emerald-100 font-bold text-xs transition flex items-center gap-1.5 cursor-pointer shadow-2xs"
          title="Instant access to wp-config, .htaccess, .env, robots.txt"
        >
          <FileCode className="w-3.5 h-3.5 text-emerald-600" />
          <span>Quick Configs</span>
        </button>

        {/* Fix Permissions 1-Click Button */}
        <button
          type="button"
          onClick={() => setFixConfirmOpen(true)}
          disabled={fixingPermissions}
          className="px-3 py-1.5 rounded-xl bg-white dark:bg-surface-800 border border-slate-200 dark:border-surface-700 hover:bg-slate-50 dark:hover:bg-surface-700 text-slate-700 dark:text-slate-200 font-bold text-xs transition flex items-center gap-1.5 cursor-pointer shadow-2xs disabled:opacity-50"
          title="Reset directory permissions to 0755 and file permissions to 0644"
        >
          <ShieldCheck className="w-3.5 h-3.5 text-blue-600" />
          <span className="hidden sm:inline">Fix Permissions</span>
        </button>

        {/* Open Terminal in current directory */}
        <a
          href={`/terminal?cwd=${encodeURIComponent(currentPath)}`}
          target="_blank"
          rel="noopener noreferrer"
          className="p-1.5 rounded-xl border border-slate-200 dark:border-surface-700 hover:bg-slate-50 dark:hover:bg-surface-700 text-slate-600 dark:text-slate-300 transition"
          title="Open web terminal at this directory"
        >
          <Terminal className="w-4 h-4" />
        </a>
      </div>

      {/* Quick Config Files Modal */}
      {configsModalOpen && (
        <div className="fixed inset-0 z-50 bg-black/75 backdrop-blur-xs flex items-center justify-center p-4 animate-fadeIn">
          <div className="w-full max-w-lg bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-700 rounded-2xl shadow-2xl p-6 relative font-sans">
            <button
              onClick={() => setConfigsModalOpen(false)}
              className="absolute top-5 right-5 p-1 rounded-lg text-slate-400 hover:text-slate-700 dark:hover:text-white"
            >
              <X className="w-5 h-5" />
            </button>

            <div className="flex items-center gap-3 mb-4">
              <div className="w-10 h-10 rounded-xl bg-emerald-500/10 border border-emerald-500/20 flex items-center justify-center text-emerald-600">
                <FileCode className="w-5 h-5" />
              </div>
              <div>
                <h3 className="text-sm font-bold text-slate-900 dark:text-white">
                  Quick Website Configurations
                </h3>
                <p className="text-xs text-slate-500 dark:text-slate-400 font-mono">
                  Target directory: {currentPath}
                </p>
              </div>
            </div>

            <p className="text-xs text-slate-600 dark:text-slate-300 mb-4">
              Select any core web configuration file to immediately open and edit it in the code editor:
            </p>

            <div className="space-y-2 max-h-[50vh] overflow-y-auto">
              {COMMON_CONFIG_FILES.map((cfg) => (
                <div
                  key={cfg.name}
                  className="p-3 rounded-xl bg-slate-50 dark:bg-surface-950 border border-slate-200 dark:border-surface-800 hover:border-emerald-500/40 transition flex items-center justify-between gap-3 text-xs"
                >
                  <div className="space-y-0.5 min-w-0">
                    <div className="flex items-center gap-2">
                      <span className="font-mono font-bold text-slate-900 dark:text-white">
                        {cfg.name}
                      </span>
                      <span className="px-1.5 py-0.2 rounded text-[10px] bg-slate-200 dark:bg-surface-800 text-slate-600 dark:text-slate-400 font-semibold">
                        {cfg.label}
                      </span>
                    </div>
                    <p className="text-[11px] text-slate-500 dark:text-slate-400 truncate">
                      {cfg.desc}
                    </p>
                  </div>

                  <button
                    type="button"
                    onClick={() => {
                      setConfigsModalOpen(false);
                      onQuickConfigFile(cfg.name);
                    }}
                    className="px-3 py-1.5 rounded-lg bg-emerald-600 hover:bg-emerald-700 text-white font-bold text-xs transition cursor-pointer shrink-0 shadow-2xs"
                  >
                    Edit File
                  </button>
                </div>
              ))}
            </div>

            <div className="mt-5 flex justify-end">
              <button
                type="button"
                onClick={() => setConfigsModalOpen(false)}
                className="px-4 py-2 rounded-xl bg-slate-100 hover:bg-slate-200 dark:bg-surface-800 dark:hover:bg-surface-700 text-slate-700 dark:text-slate-200 font-semibold text-xs"
              >
                Close
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Fix Web Permissions Modal */}
      {fixConfirmOpen && (
        <div className="fixed inset-0 z-50 bg-black/75 backdrop-blur-xs flex items-center justify-center p-4 animate-fadeIn">
          <div className="w-full max-w-md bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-700 rounded-2xl shadow-2xl p-6 relative font-sans text-xs">
            <button
              onClick={() => setFixConfirmOpen(false)}
              className="absolute top-5 right-5 p-1 rounded-lg text-slate-400 hover:text-slate-700 dark:hover:text-white"
            >
              <X className="w-5 h-5" />
            </button>

            <div className="flex items-center gap-3 mb-4">
              <div className="w-10 h-10 rounded-xl bg-blue-500/10 border border-blue-500/20 flex items-center justify-center text-blue-600">
                <ShieldCheck className="w-5 h-5" />
              </div>
              <div>
                <h3 className="text-sm font-bold text-slate-900 dark:text-white">
                  Fix Standard Web Permissions
                </h3>
                <p className="text-xs text-slate-500">Security & write access fix</p>
              </div>
            </div>

            <div className="space-y-3 mb-5 text-slate-600 dark:text-slate-300">
              <p>
                This operation will automatically reset all permissions in{' '}
                <strong className="font-mono text-slate-900 dark:text-white">{currentPath}</strong>:
              </p>
              <ul className="list-disc pl-5 space-y-1 font-mono text-[11px]">
                <li>Directories will be set to <strong className="text-emerald-600">0755</strong> (rwxr-xr-x)</li>
                <li>Files will be set to <strong className="text-emerald-600">0644</strong> (rw-r--r--)</li>
              </ul>
              <p className="text-[11px] text-slate-500">
                This fixes WordPress, Laravel, and PHP file upload errors, plugin update failures, and 403 Forbidden errors.
              </p>
            </div>

            <div className="flex items-center justify-end gap-2">
              <button
                type="button"
                onClick={() => setFixConfirmOpen(false)}
                className="px-4 py-2 rounded-xl bg-slate-100 hover:bg-slate-200 dark:bg-surface-800 dark:hover:bg-surface-700 text-slate-700 dark:text-slate-200 font-semibold"
              >
                Cancel
              </button>
              <button
                type="button"
                onClick={() => {
                  setFixConfirmOpen(false);
                  onFixPermissions(currentPath);
                }}
                disabled={fixingPermissions}
                className="px-4 py-2 rounded-xl bg-blue-600 hover:bg-blue-700 text-white font-bold transition shadow-xs flex items-center gap-1.5"
              >
                <ShieldCheck className="w-4 h-4" />
                <span>{fixingPermissions ? 'Fixing...' : 'Apply 0755 / 0644'}</span>
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};
