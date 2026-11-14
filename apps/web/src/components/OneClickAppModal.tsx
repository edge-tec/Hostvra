'use client';

import React, { useState, useEffect } from 'react';
import {
  Sparkles,
  X,
  Globe,
  Database,
  Cpu,
  Layers,
  Code2,
  CheckCircle2,
  AlertTriangle,
  RefreshCw,
  ExternalLink,
  Eye,
  EyeOff,
  Copy,
  Check,
  Zap,
  Server,
  Key,
  Folder,
  Plus,
  ChevronDown,
} from 'lucide-react';
import { apiFetch, Website, OneClickAppTemplate, InstalledAppInfo } from '@/lib/api';

interface OneClickAppModalProps {
  website?: Website | null;
  websites?: Website[];
  isOpen: boolean;
  onClose: () => void;
  onSuccess: (targetSite?: Website) => void;
  initialAppId?: string;
  onCreatedWebsite?: (site: Website) => void;
}

export function OneClickAppModal({
  website,
  websites = [],
  isOpen,
  onClose,
  onSuccess,
  initialAppId,
  onCreatedWebsite,
}: OneClickAppModalProps) {
  const [templates, setTemplates] = useState<OneClickAppTemplate[]>([]);
  const [selectedApp, setSelectedApp] = useState<string>(initialAppId || 'wordpress');
  const [loadingTemplates, setLoadingTemplates] = useState(false);

  // Target website selection (existing or new)
  const [targetMode, setTargetMode] = useState<'existing' | 'new'>(website ? 'existing' : websites.length > 0 ? 'existing' : 'new');
  const [selectedSiteId, setSelectedSiteId] = useState<string>(website?.id || (websites.length > 0 ? websites[0].id : ''));
  const [newDomain, setNewDomain] = useState('');
  const [newPhpVersion, setNewPhpVersion] = useState('8.2');

  // Application parameters
  const [siteTitle, setSiteTitle] = useState('');
  const [adminUser, setAdminUser] = useState('admin');
  const [adminEmail, setAdminEmail] = useState('');
  const [adminPassword, setAdminPassword] = useState('');
  const [showPassword, setShowPassword] = useState(false);

  // Database parameters (aaPanel 1-Click Database Parity)
  const [showDbConfig, setShowDbConfig] = useState(true);
  const [dbType, setDbType] = useState<'mysql' | 'postgresql' | 'sqlite'>('mysql');
  const [dbName, setDbName] = useState('');
  const [dbUser, setDbUser] = useState('');
  const [dbPassword, setDbPassword] = useState('');
  const [showDbPassword, setShowDbPassword] = useState(false);
  const [tablePrefix, setTablePrefix] = useState('wp_');

  // Deployment state
  const [installing, setInstalling] = useState(false);
  const [progress, setProgress] = useState(0);
  const [currentStep, setCurrentStep] = useState('');
  const [deployedApp, setDeployedApp] = useState<InstalledAppInfo | null>(null);
  const [deployedSite, setDeployedSite] = useState<Website | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [copiedField, setCopiedField] = useState<string | null>(null);

  // Determine active target domain
  const effectiveWebsite = website || websites.find((w) => w.id === selectedSiteId) || null;
  const currentDomain = targetMode === 'existing' ? effectiveWebsite?.primary_domain || '' : newDomain.trim();

  useEffect(() => {
    if (isOpen) {
      if (initialAppId) {
        setSelectedApp(initialAppId);
      }
      fetchTemplates();
      generatePassword();
      setDeployedApp(null);
      setDeployedSite(null);
      setError(null);
      setProgress(0);

      if (website) {
        setSelectedSiteId(website.id);
        setTargetMode('existing');
      } else if (websites.length > 0) {
        setSelectedSiteId(websites[0].id);
        setTargetMode('existing');
      } else {
        setTargetMode('new');
      }
    }
  }, [isOpen, initialAppId, website]);

  // Update default titles & emails when domain changes
  useEffect(() => {
    if (currentDomain) {
      const sanitized = currentDomain.replace(/[^a-zA-Z0-9]/g, '_').toLowerCase().slice(0, 8);
      const randHex = Math.random().toString(36).substring(2, 6);
      if (!siteTitle || siteTitle.includes('Website')) {
        setSiteTitle(`${currentDomain} Portal`);
      }
      if (!adminEmail || adminEmail.includes('@')) {
        setAdminEmail(`admin@${currentDomain}`);
      }
      setDbName(`db_${sanitized}_${randHex}`);
      setDbUser(`u_${sanitized}_${randHex}`);
    }
  }, [currentDomain]);

  useEffect(() => {
    if (selectedApp === 'wordpress') {
      setTablePrefix('wp_');
    } else if (selectedApp === 'joomla') {
      setTablePrefix('jos_');
    } else if (selectedApp === 'typecho') {
      setTablePrefix('typecho_');
    } else {
      setTablePrefix('app_');
    }
  }, [selectedApp]);

  const fetchTemplates = async () => {
    try {
      setLoadingTemplates(true);
      const res = await apiFetch<OneClickAppTemplate[]>('/api/v1/installer/templates');
      if (res.success && res.data) {
        setTemplates(res.data);
      }
    } catch {
      // Fallback
    } finally {
      setLoadingTemplates(false);
    }
  };

  const generatePassword = () => {
    const chars = 'abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*()_+-=';
    let pwd = '';
    for (let i = 0; i < 18; i++) {
      pwd += chars.charAt(Math.floor(Math.random() * chars.length));
    }
    setAdminPassword(pwd);

    let dbPwd = '';
    for (let i = 0; i < 16; i++) {
      dbPwd += chars.charAt(Math.floor(Math.random() * chars.length));
    }
    setDbPassword(dbPwd);
  };

  const handleCopy = (text: string, field: string) => {
    navigator.clipboard.writeText(text);
    setCopiedField(field);
    setTimeout(() => setCopiedField(null), 2000);
  };

  const handleDeploy = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);
    setInstalling(true);
    setProgress(10);
    setCurrentStep('Validating installation target & domain configuration...');

    let targetSite: Website | null = effectiveWebsite;

    try {
      // Step 1: If creating a new site on the fly (aaPanel 1-Click Deployment)
      if (targetMode === 'new') {
        const domainClean = newDomain.trim().toLowerCase();
        if (!domainClean) {
          throw new Error('Please enter a valid domain name for the new website.');
        }

        setProgress(20);
        setCurrentStep(`Provisioning new virtual host for ${domainClean}...`);

        const siteRes = await apiFetch<Website>('/api/v1/websites', {
          method: 'POST',
          body: JSON.stringify({
            primary_domain: domainClean,
            document_root: `/var/www/${domainClean}/public_html`,
            php_version: newPhpVersion,
            app_type: 'php',
            web_server_type: 'nginx',
          }),
        });

        if (!siteRes.success || !siteRes.data) {
          throw new Error(siteRes.error?.message || 'Failed to create virtual host for new website');
        }

        targetSite = siteRes.data;
        setDeployedSite(targetSite);
        onCreatedWebsite?.(targetSite);
      }

      if (!targetSite) {
        throw new Error('No target website selected.');
      }

      setDeployedSite(targetSite);
      setProgress(40);
      setCurrentStep('Provisioning isolated MySQL database & user privileges...');

      const stepTimer1 = setTimeout(() => {
        setProgress(65);
        setCurrentStep(`Deploying ${selectedApp.toUpperCase()} core files & configuration...`);
      }, 500);

      const stepTimer2 = setTimeout(() => {
        setProgress(85);
        setCurrentStep('Configuring security salts, permissions & virtual host routing...');
      }, 1000);

      const res = await apiFetch<InstalledAppInfo>(`/api/v1/websites/${targetSite.id}/app/install`, {
        method: 'POST',
        body: JSON.stringify({
          app_id: selectedApp,
          site_title: siteTitle || `${targetSite.primary_domain} Website`,
          admin_user: adminUser,
          admin_email: adminEmail,
          admin_password: adminPassword,
          db_type: dbType,
          db_name: dbName,
          db_user: dbUser,
          db_password: dbPassword,
          table_prefix: tablePrefix,
        }),
      });

      clearTimeout(stepTimer1);
      clearTimeout(stepTimer2);

      if (res.success && res.data) {
        setProgress(100);
        setCurrentStep('Application deployed and running live!');
        setDeployedApp(res.data);
        onSuccess(targetSite);
      } else {
        setError(res.error?.message || 'Deployment failed');
      }
    } catch (err: any) {
      setError(err.message || 'Deployment error');
    } finally {
      setInstalling(false);
    }
  };

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/60 backdrop-blur-xs animate-fadeIn">
      <div className="bg-white dark:bg-[#10141d] border border-slate-200 dark:border-surface-800 rounded-2xl w-full max-w-2xl max-h-[90vh] overflow-y-auto shadow-2xl">
        {/* Header */}
        <div className="flex items-center justify-between p-5 border-b border-slate-200 dark:border-surface-800">
          <div className="flex items-center gap-2.5">
            <div className="p-2 rounded-xl bg-purple-500/10 text-purple-600 dark:text-purple-400">
              <Zap className="w-5 h-5" />
            </div>
            <div>
              <div className="flex items-center gap-2">
                <h2 className="text-base font-bold text-slate-950 dark:text-white">1-Click App Deployment</h2>
                <span className="px-2 py-0.5 rounded-full text-[10px] font-black bg-purple-500/10 text-purple-600 dark:text-purple-400 border border-purple-500/20">
                  aaPanel Parity
                </span>
              </div>
              <p className="text-xs text-slate-500 dark:text-slate-400">
                Instant CMS, Framework & Database Provisioning
              </p>
            </div>
          </div>
          <button
            onClick={onClose}
            className="p-1.5 rounded-lg text-slate-400 hover:text-slate-600 dark:hover:text-white hover:bg-slate-100 dark:hover:bg-surface-800 transition"
          >
            <X className="w-4 h-4" />
          </button>
        </div>

        {deployedApp ? (
          /* SUCCESS SUMMARY VIEW */
          <div className="p-6 space-y-5">
            <div className="p-4 rounded-2xl bg-emerald-500/10 border border-emerald-500/20 flex items-center gap-3">
              <div className="p-2.5 rounded-full bg-emerald-500/20 text-emerald-600 dark:text-emerald-400 flex-shrink-0">
                <CheckCircle2 className="w-6 h-6" />
              </div>
              <div>
                <h3 className="text-sm font-bold text-emerald-800 dark:text-emerald-300">
                  {deployedApp.name} Successfully Deployed!
                </h3>
                <p className="text-xs text-emerald-700/80 dark:text-emerald-400/80 mt-0.5">
                  Virtual host, database, and security configurations are active and live.
                </p>
              </div>
            </div>

            <div className="bg-slate-50 dark:bg-[#121824] p-4 rounded-xl border border-slate-200 dark:border-surface-700 space-y-3 text-xs">
              <div className="flex justify-between items-center py-1 border-b border-slate-200/60 dark:border-surface-800">
                <span className="text-slate-500 dark:text-slate-400">Target Website:</span>
                <span className="font-bold text-slate-950 dark:text-white">
                  {deployedSite?.primary_domain || currentDomain}
                </span>
              </div>

              <div className="flex justify-between items-center py-1 border-b border-slate-200/60 dark:border-surface-800">
                <span className="text-slate-500 dark:text-slate-400">Admin Username:</span>
                <div className="flex items-center gap-1.5 font-mono font-bold text-slate-950 dark:text-white">
                  <span>{adminUser}</span>
                  <button
                    type="button"
                    onClick={() => handleCopy(adminUser, 'user')}
                    className="text-slate-400 hover:text-slate-600 dark:hover:text-white p-0.5"
                  >
                    {copiedField === 'user' ? <Check className="w-3.5 h-3.5 text-emerald-500" /> : <Copy className="w-3.5 h-3.5" />}
                  </button>
                </div>
              </div>

              <div className="flex justify-between items-center py-1 border-b border-slate-200/60 dark:border-surface-800">
                <span className="text-slate-500 dark:text-slate-400">Admin Password:</span>
                <div className="flex items-center gap-1.5 font-mono font-bold text-slate-950 dark:text-white">
                  <span>{adminPassword}</span>
                  <button
                    type="button"
                    onClick={() => handleCopy(adminPassword, 'pwd')}
                    className="text-slate-400 hover:text-slate-600 dark:hover:text-white p-0.5"
                  >
                    {copiedField === 'pwd' ? <Check className="w-3.5 h-3.5 text-emerald-500" /> : <Copy className="w-3.5 h-3.5" />}
                  </button>
                </div>
              </div>

              {deployedApp.db_name && (
                <>
                  <div className="flex justify-between items-center py-1 border-b border-slate-200/60 dark:border-surface-800">
                    <span className="text-slate-500 dark:text-slate-400">Database Name:</span>
                    <div className="flex items-center gap-1.5 font-mono font-bold text-purple-600 dark:text-purple-400">
                      <span>{deployedApp.db_name}</span>
                      <button
                        type="button"
                        onClick={() => handleCopy(deployedApp.db_name!, 'dbname')}
                        className="text-slate-400 hover:text-slate-600 dark:hover:text-white p-0.5"
                      >
                        {copiedField === 'dbname' ? <Check className="w-3.5 h-3.5 text-emerald-500" /> : <Copy className="w-3.5 h-3.5" />}
                      </button>
                    </div>
                  </div>

                  {deployedApp.db_user && (
                    <div className="flex justify-between items-center py-1 border-b border-slate-200/60 dark:border-surface-800">
                      <span className="text-slate-500 dark:text-slate-400">Database User:</span>
                      <div className="flex items-center gap-1.5 font-mono text-slate-900 dark:text-slate-100 font-bold">
                        <span>{deployedApp.db_user}</span>
                        <button
                          type="button"
                          onClick={() => handleCopy(deployedApp.db_user!, 'dbuser')}
                          className="text-slate-400 hover:text-slate-600 dark:hover:text-white p-0.5"
                        >
                          {copiedField === 'dbuser' ? <Check className="w-3.5 h-3.5 text-emerald-500" /> : <Copy className="w-3.5 h-3.5" />}
                        </button>
                      </div>
                    </div>
                  )}
                </>
              )}

              <div className="flex justify-between items-center py-1">
                <span className="text-slate-500 dark:text-slate-400">Admin Dashboard:</span>
                <a
                  href={deployedApp.admin_url}
                  target="_blank"
                  rel="noreferrer"
                  className="font-bold text-indigo-600 dark:text-indigo-400 hover:underline flex items-center gap-1"
                >
                  <span>{deployedApp.admin_url}</span>
                  <ExternalLink className="w-3 h-3" />
                </a>
              </div>
            </div>

            <div className="flex items-center justify-end gap-2.5 pt-2">
              <button
                type="button"
                onClick={onClose}
                className="px-4 py-2 rounded-xl border border-slate-300 dark:border-surface-700 text-xs font-semibold text-slate-700 dark:text-slate-300 hover:bg-slate-50 dark:hover:bg-surface-800 transition"
              >
                Done
              </button>
              <a
                href={deployedApp.admin_url}
                target="_blank"
                rel="noreferrer"
                className="inline-flex items-center gap-1.5 px-5 py-2 rounded-xl bg-purple-600 hover:bg-purple-500 text-white text-xs font-bold shadow-md transition-all"
              >
                <span>Launch Admin</span>
                <ExternalLink className="w-3.5 h-3.5" />
              </a>
            </div>
          </div>
        ) : (
          /* DEPLOYMENT FORM */
          <form onSubmit={handleDeploy} className="p-5 space-y-4">
            {error && (
              <div className="p-3.5 rounded-xl bg-rose-500/10 border border-rose-500/20 text-rose-700 dark:text-rose-400 text-xs font-medium flex items-center gap-2">
                <AlertTriangle className="w-4 h-4 flex-shrink-0" />
                <span>{error}</span>
              </div>
            )}

            {/* Target Website Choice (Existing vs New) */}
            <div className="bg-slate-50 dark:bg-[#121824] p-3.5 rounded-xl border border-slate-200 dark:border-surface-700 space-y-2.5">
              <div className="flex items-center justify-between">
                <label className="text-xs font-bold text-slate-900 dark:text-slate-100 flex items-center gap-1.5">
                  <Globe className="w-3.5 h-3.5 text-purple-600 dark:text-purple-400" />
                  Target Website / Virtual Host
                </label>
                {!website && (
                  <div className="inline-flex rounded-lg bg-slate-200 dark:bg-surface-800 p-0.5 text-[11px] font-semibold">
                    <button
                      type="button"
                      onClick={() => setTargetMode('existing')}
                      className={`px-2.5 py-1 rounded-md transition ${
                        targetMode === 'existing'
                          ? 'bg-white dark:bg-surface-700 text-slate-900 dark:text-white shadow-2xs font-bold'
                          : 'text-slate-600 dark:text-slate-400'
                      }`}
                    >
                      Existing Site ({websites.length})
                    </button>
                    <button
                      type="button"
                      onClick={() => setTargetMode('new')}
                      className={`px-2.5 py-1 rounded-md transition ${
                        targetMode === 'new'
                          ? 'bg-white dark:bg-surface-700 text-slate-900 dark:text-white shadow-2xs font-bold'
                          : 'text-slate-600 dark:text-slate-400'
                      }`}
                    >
                      + Deploy to New Domain
                    </button>
                  </div>
                )}
              </div>

              {targetMode === 'existing' ? (
                <div>
                  {website ? (
                    <div className="flex items-center justify-between px-3 py-2 bg-white dark:bg-surface-800 rounded-lg border border-slate-300 dark:border-surface-700 text-xs">
                      <span className="font-bold text-slate-900 dark:text-white">{website.primary_domain}</span>
                      <span className="text-[11px] text-slate-500 font-mono">{website.document_root}</span>
                    </div>
                  ) : websites.length > 0 ? (
                    <select
                      value={selectedSiteId}
                      onChange={(e) => setSelectedSiteId(e.target.value)}
                      className="w-full px-3 py-2 rounded-lg border border-slate-300 dark:border-surface-700 bg-white dark:bg-surface-800 text-xs font-medium text-slate-950 dark:text-white"
                    >
                      {websites.map((w) => (
                        <option key={w.id} value={w.id}>
                          {w.primary_domain} ({w.document_root})
                        </option>
                      ))}
                    </select>
                  ) : (
                    <p className="text-xs text-amber-600 dark:text-amber-400">
                      No websites found. Please switch to &quot;Deploy to New Domain&quot; to auto-create one.
                    </p>
                  )}
                </div>
              ) : (
                <div className="grid grid-cols-1 sm:grid-cols-3 gap-2.5">
                  <div className="sm:col-span-2">
                    <input
                      type="text"
                      required
                      placeholder="e.g. blog.yourdomain.com"
                      value={newDomain}
                      onChange={(e) => setNewDomain(e.target.value)}
                      className="w-full px-3 py-2 rounded-lg border border-slate-300 dark:border-surface-700 bg-white dark:bg-surface-800 text-xs font-medium text-slate-950 dark:text-white"
                    />
                  </div>
                  <div>
                    <select
                      value={newPhpVersion}
                      onChange={(e) => setNewPhpVersion(e.target.value)}
                      className="w-full px-3 py-2 rounded-lg border border-slate-300 dark:border-surface-700 bg-white dark:bg-surface-800 text-xs font-medium text-slate-950 dark:text-white"
                    >
                      <option value="8.3">PHP 8.3</option>
                      <option value="8.2">PHP 8.2 (Recommended)</option>
                      <option value="8.1">PHP 8.1</option>
                      <option value="8.0">PHP 8.0</option>
                      <option value="7.4">PHP 7.4</option>
                    </select>
                  </div>
                </div>
              )}
            </div>

            {/* Template Selector Cards (aaPanel 1-Click Suite) */}
            <div>
              <label className="block text-xs font-bold text-slate-800 dark:text-slate-200 mb-2">
                Select Application Template
              </label>
              <div className="grid grid-cols-2 sm:grid-cols-3 gap-2">
                {[
                  { id: 'wordpress', name: 'WordPress', badge: 'v6.7 LTS', icon: Globe, desc: 'World #1 CMS' },
                  { id: 'laravel', name: 'Laravel', badge: 'v11.x', icon: Code2, desc: 'PHP MVC Framework' },
                  { id: 'nextjs', name: 'Next.js', badge: 'v15.1', icon: Cpu, desc: 'React SSR Fullstack' },
                  { id: 'drupal', name: 'Drupal', badge: 'v10.3', icon: Layers, desc: 'Enterprise CMS' },
                  { id: 'joomla', name: 'Joomla', badge: 'v5.x', icon: Layers, desc: 'Corporate CMS' },
                  { id: 'typecho', name: 'Typecho', badge: 'v1.2', icon: Globe, desc: 'Lightweight Blog' },
                  { id: 'phpmyadmin', name: 'phpMyAdmin', badge: 'v5.2', icon: Database, desc: 'MySQL Web Admin' },
                ].map((tpl) => {
                  const Icon = tpl.icon;
                  const isSelected = selectedApp === tpl.id;

                  return (
                    <button
                      key={tpl.id}
                      type="button"
                      onClick={() => setSelectedApp(tpl.id)}
                      className={`p-2.5 rounded-xl border text-left transition-all ${
                        isSelected
                          ? 'border-purple-500 bg-purple-50/50 dark:bg-purple-950/20 ring-2 ring-purple-500/20'
                          : 'border-slate-200 dark:border-surface-700 hover:border-slate-300 dark:hover:border-surface-600 bg-slate-50 dark:bg-[#121824]'
                      }`}
                    >
                      <div className="flex items-center justify-between mb-1">
                        <Icon className={`w-4 h-4 ${isSelected ? 'text-purple-600 dark:text-purple-400' : 'text-slate-500'}`} />
                        <span className="text-[9px] font-black px-1.5 py-0.5 rounded bg-slate-200 dark:bg-surface-800 text-slate-700 dark:text-slate-300">
                          {tpl.badge}
                        </span>
                      </div>
                      <div className="font-bold text-xs text-slate-950 dark:text-white">{tpl.name}</div>
                      <div className="text-[10px] text-slate-500 dark:text-slate-400 mt-0.5 truncate">{tpl.desc}</div>
                    </button>
                  );
                })}
              </div>
            </div>

            {/* Application Configuration Fields */}
            <div className="space-y-3 bg-slate-50 dark:bg-[#121824] p-3.5 rounded-xl border border-slate-200 dark:border-surface-700">
              <div className="text-xs font-bold text-slate-900 dark:text-slate-100 flex items-center gap-1.5">
                <Sparkles className="w-3.5 h-3.5 text-purple-600 dark:text-purple-400" />
                Site & Admin Account Credentials
              </div>

              <div>
                <label className="block text-[11px] font-bold text-slate-700 dark:text-slate-300 mb-1">
                  Site / Application Title
                </label>
                <input
                  type="text"
                  required
                  value={siteTitle}
                  onChange={(e) => setSiteTitle(e.target.value)}
                  className="w-full px-3 py-1.5 rounded-lg border border-slate-300 dark:border-surface-700 bg-white dark:bg-surface-800 text-xs font-medium text-slate-950 dark:text-white"
                />
              </div>

              <div className="grid grid-cols-1 sm:grid-cols-2 gap-2.5">
                <div>
                  <label className="block text-[11px] font-bold text-slate-700 dark:text-slate-300 mb-1">
                    Administrator Username
                  </label>
                  <input
                    type="text"
                    required
                    value={adminUser}
                    onChange={(e) => setAdminUser(e.target.value)}
                    className="w-full px-3 py-1.5 rounded-lg border border-slate-300 dark:border-surface-700 bg-white dark:bg-surface-800 text-xs font-medium text-slate-950 dark:text-white"
                  />
                </div>

                <div>
                  <label className="block text-[11px] font-bold text-slate-700 dark:text-slate-300 mb-1">
                    Administrator Email
                  </label>
                  <input
                    type="email"
                    required
                    value={adminEmail}
                    onChange={(e) => setAdminEmail(e.target.value)}
                    className="w-full px-3 py-1.5 rounded-lg border border-slate-300 dark:border-surface-700 bg-white dark:bg-surface-800 text-xs font-medium text-slate-950 dark:text-white"
                  />
                </div>
              </div>

              <div>
                <div className="flex items-center justify-between mb-1">
                  <label className="text-[11px] font-bold text-slate-700 dark:text-slate-300">
                    Administrator Password
                  </label>
                  <button
                    type="button"
                    onClick={generatePassword}
                    className="text-[10px] text-purple-600 dark:text-purple-400 font-bold hover:underline"
                  >
                    Regenerate All Passwords
                  </button>
                </div>
                <div className="relative">
                  <input
                    type={showPassword ? 'text' : 'password'}
                    required
                    value={adminPassword}
                    onChange={(e) => setAdminPassword(e.target.value)}
                    className="w-full pl-3 pr-10 py-1.5 rounded-lg border border-slate-300 dark:border-surface-700 bg-white dark:bg-surface-800 text-xs font-mono text-slate-950 dark:text-white"
                  />
                  <button
                    type="button"
                    onClick={() => setShowPassword(!showPassword)}
                    className="absolute right-3 top-2 text-slate-400 hover:text-slate-600"
                  >
                    {showPassword ? <EyeOff className="w-3.5 h-3.5" /> : <Eye className="w-3.5 h-3.5" />}
                  </button>
                </div>
              </div>
            </div>

            {/* Database Configuration (aaPanel Parity) */}
            {selectedApp !== 'nextjs' && (
              <div className="space-y-3 bg-slate-50 dark:bg-[#121824] p-3.5 rounded-xl border border-slate-200 dark:border-surface-700">
                <div className="flex items-center justify-between">
                  <div className="text-xs font-bold text-slate-900 dark:text-slate-100 flex items-center gap-1.5">
                    <Database className="w-3.5 h-3.5 text-indigo-600 dark:text-indigo-400" />
                    MySQL Database Auto-Provisioning
                  </div>
                  <span className="text-[10px] px-2 py-0.5 rounded-full font-bold bg-emerald-50 text-emerald-700 dark:bg-emerald-950/40 dark:text-emerald-400 border border-emerald-200 dark:border-emerald-800">
                    Auto Provisioned
                  </span>
                </div>

                <div className="grid grid-cols-1 sm:grid-cols-2 gap-2.5">
                  <div>
                    <label className="block text-[11px] font-bold text-slate-700 dark:text-slate-300 mb-1">
                      Database Name
                    </label>
                    <input
                      type="text"
                      value={dbName}
                      onChange={(e) => setDbName(e.target.value)}
                      className="w-full px-3 py-1.5 rounded-lg border border-slate-300 dark:border-surface-700 bg-white dark:bg-surface-800 text-xs font-mono text-slate-950 dark:text-white"
                    />
                  </div>

                  <div>
                    <label className="block text-[11px] font-bold text-slate-700 dark:text-slate-300 mb-1">
                      Database Username
                    </label>
                    <input
                      type="text"
                      value={dbUser}
                      onChange={(e) => setDbUser(e.target.value)}
                      className="w-full px-3 py-1.5 rounded-lg border border-slate-300 dark:border-surface-700 bg-white dark:bg-surface-800 text-xs font-mono text-slate-950 dark:text-white"
                    />
                  </div>
                </div>

                <div className="grid grid-cols-1 sm:grid-cols-2 gap-2.5">
                  <div>
                    <label className="block text-[11px] font-bold text-slate-700 dark:text-slate-300 mb-1">
                      Database Password
                    </label>
                    <div className="relative">
                      <input
                        type={showDbPassword ? 'text' : 'password'}
                        value={dbPassword}
                        onChange={(e) => setDbPassword(e.target.value)}
                        className="w-full pl-3 pr-10 py-1.5 rounded-lg border border-slate-300 dark:border-surface-700 bg-white dark:bg-surface-800 text-xs font-mono text-slate-950 dark:text-white"
                      />
                      <button
                        type="button"
                        onClick={() => setShowDbPassword(!showDbPassword)}
                        className="absolute right-3 top-2 text-slate-400 hover:text-slate-600"
                      >
                        {showDbPassword ? <EyeOff className="w-3.5 h-3.5" /> : <Eye className="w-3.5 h-3.5" />}
                      </button>
                    </div>
                  </div>

                  <div>
                    <label className="block text-[11px] font-bold text-slate-700 dark:text-slate-300 mb-1">
                      Table Prefix
                    </label>
                    <input
                      type="text"
                      value={tablePrefix}
                      onChange={(e) => setTablePrefix(e.target.value)}
                      className="w-full px-3 py-1.5 rounded-lg border border-slate-300 dark:border-surface-700 bg-white dark:bg-surface-800 text-xs font-mono text-slate-950 dark:text-white"
                    />
                  </div>
                </div>
              </div>
            )}

            {/* Installation Progress Bar */}
            {installing && (
              <div className="space-y-2 p-3.5 rounded-xl bg-purple-500/10 border border-purple-500/20 animate-fadeIn">
                <div className="flex items-center justify-between text-xs font-bold text-purple-700 dark:text-purple-300">
                  <div className="flex items-center gap-2">
                    <RefreshCw className="w-3.5 h-3.5 animate-spin" />
                    <span>{currentStep}</span>
                  </div>
                  <span>{progress}%</span>
                </div>
                <div className="w-full bg-purple-200 dark:bg-purple-950/40 rounded-full h-2 overflow-hidden">
                  <div
                    className="h-full bg-purple-600 dark:bg-purple-400 transition-all duration-300"
                    style={{ width: `${progress}%` }}
                  />
                </div>
              </div>
            )}

            {/* Footer Actions */}
            <div className="flex items-center justify-end gap-2.5 pt-3 border-t border-slate-200 dark:border-surface-800">
              <button
                type="button"
                onClick={onClose}
                disabled={installing}
                className="px-4 py-2 rounded-xl border border-slate-300 dark:border-surface-700 text-xs font-semibold text-slate-700 dark:text-slate-300 disabled:opacity-50 hover:bg-slate-50 dark:hover:bg-surface-800 transition"
              >
                Cancel
              </button>
              <button
                type="submit"
                disabled={installing}
                className="inline-flex items-center gap-2 px-5 py-2 rounded-xl bg-purple-600 hover:bg-purple-500 text-white text-xs font-bold shadow-md transition-all disabled:opacity-50 cursor-pointer"
              >
                <Zap className="w-3.5 h-3.5" />
                <span>
                  {installing ? 'Deploying...' : `One-Click Deploy ${selectedApp.toUpperCase()}`}
                </span>
              </button>
            </div>
          </form>
        )}
      </div>
    </div>
  );
}
