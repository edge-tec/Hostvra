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
} from 'lucide-react';
import { apiFetch, Website, OneClickAppTemplate, InstalledAppInfo } from '@/lib/api';

interface OneClickAppModalProps {
  website: Website;
  isOpen: boolean;
  onClose: () => void;
  onSuccess: () => void;
  initialAppId?: string;
}

export function OneClickAppModal({ website, isOpen, onClose, onSuccess, initialAppId }: OneClickAppModalProps) {
  const [templates, setTemplates] = useState<OneClickAppTemplate[]>([]);
  const [selectedApp, setSelectedApp] = useState<string>(initialAppId || 'wordpress');
  const [loadingTemplates, setLoadingTemplates] = useState(false);
  const [siteTitle, setSiteTitle] = useState(`${website.primary_domain} Website`);
  const [adminUser, setAdminUser] = useState('admin');
  const [adminEmail, setAdminEmail] = useState(`admin@${website.primary_domain}`);
  const [adminPassword, setAdminPassword] = useState('');
  const [showPassword, setShowPassword] = useState(false);
  const [dbType, setDbType] = useState<'mysql' | 'postgresql' | 'sqlite'>('mysql');

  // Deployment state
  const [installing, setInstalling] = useState(false);
  const [progress, setProgress] = useState(0);
  const [currentStep, setCurrentStep] = useState('');
  const [deployedApp, setDeployedApp] = useState<InstalledAppInfo | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [copiedField, setCopiedField] = useState<string | null>(null);

  useEffect(() => {
    if (isOpen) {
      if (initialAppId) {
        setSelectedApp(initialAppId);
      }
      fetchTemplates();
      generatePassword();
      setDeployedApp(null);
      setError(null);
      setProgress(0);
    }
  }, [isOpen, initialAppId]);

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
    setProgress(15);
    setCurrentStep('Validating virtual host document root...');

    const stepTimer1 = setTimeout(() => {
      setProgress(40);
      setCurrentStep('Provisioning database & isolation credentials...');
    }, 400);

    const stepTimer2 = setTimeout(() => {
      setProgress(70);
      setCurrentStep(`Deploying ${selectedApp} core scaffolding & configuration...`);
    }, 800);

    try {
      const res = await apiFetch<InstalledAppInfo>(`/api/v1/websites/${website.id}/app/install`, {
        method: 'POST',
        body: JSON.stringify({
          app_id: selectedApp,
          site_title: siteTitle,
          admin_user: adminUser,
          admin_email: adminEmail,
          admin_password: adminPassword,
          db_type: dbType,
        }),
      });

      clearTimeout(stepTimer1);
      clearTimeout(stepTimer2);

      if (res.success && res.data) {
        setProgress(100);
        setCurrentStep('Deployment finalized successfully!');
        setDeployedApp(res.data);
        onSuccess();
      } else {
        setError(res.error?.message || 'Deployment failed');
      }
    } catch (err: any) {
      clearTimeout(stepTimer1);
      clearTimeout(stepTimer2);
      setError(err.message || 'Deployment error');
    } finally {
      setInstalling(false);
    }
  };

  if (!isOpen) return null;

  const currentTemplate = templates.find((t) => t.id === selectedApp);

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
                <h2 className="text-base font-bold text-slate-950 dark:text-white">1-Click App Installer</h2>
                <span className="px-2 py-0.5 rounded-full text-[10px] font-black bg-purple-500/10 text-purple-600 dark:text-purple-400 border border-purple-500/20">
                  {website.primary_domain}
                </span>
              </div>
              <p className="text-xs text-slate-500 dark:text-slate-400">Instant CMS & Framework Deployment</p>
            </div>
          </div>
          <button onClick={onClose} className="p-1.5 rounded-lg text-slate-400 hover:text-slate-600 dark:hover:text-white">
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
                  The application is live and configured with hardened security salts.
                </p>
              </div>
            </div>

            <div className="bg-slate-50 dark:bg-[#121824] p-4 rounded-xl border border-slate-200 dark:border-surface-700 space-y-3 text-xs">
              <div className="flex justify-between items-center py-1 border-b border-slate-200/60 dark:border-surface-800">
                <span className="text-slate-500 dark:text-slate-400">Site Title:</span>
                <span className="font-bold text-slate-950 dark:text-white">{siteTitle}</span>
              </div>

              <div className="flex justify-between items-center py-1 border-b border-slate-200/60 dark:border-surface-800">
                <span className="text-slate-500 dark:text-slate-400">Admin Username:</span>
                <span className="font-mono font-bold text-slate-950 dark:text-white">{adminUser}</span>
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
                <div className="flex justify-between items-center py-1 border-b border-slate-200/60 dark:border-surface-800">
                  <span className="text-slate-500 dark:text-slate-400">Provisioned Database:</span>
                  <span className="font-mono text-purple-600 dark:text-purple-400 font-bold">{deployedApp.db_name}</span>
                </div>
              )}

              <div className="flex justify-between items-center py-1">
                <span className="text-slate-500 dark:text-slate-400">Admin URL:</span>
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
                className="px-4 py-2 rounded-xl border border-slate-300 dark:border-surface-700 text-xs font-semibold text-slate-700 dark:text-slate-300"
              >
                Close
              </button>
              <a
                href={deployedApp.admin_url}
                target="_blank"
                rel="noreferrer"
                className="inline-flex items-center gap-1.5 px-5 py-2 rounded-xl bg-purple-600 hover:bg-purple-500 text-white text-xs font-bold shadow-md transition-all"
              >
                <span>Launch App Admin</span>
                <ExternalLink className="w-3.5 h-3.5" />
              </a>
            </div>
          </div>
        ) : (
          /* DEPLOYMENT FORM */
          <form onSubmit={handleDeploy} className="p-5 space-y-5">
            {error && (
              <div className="p-3.5 rounded-xl bg-rose-500/10 border border-rose-500/20 text-rose-700 dark:text-rose-400 text-xs font-medium flex items-center gap-2">
                <AlertTriangle className="w-4 h-4 flex-shrink-0" />
                <span>{error}</span>
              </div>
            )}

            {/* Template Selector Cards */}
            <div>
              <label className="block text-xs font-bold text-slate-800 dark:text-slate-200 mb-2">
                Select Application Template
              </label>
              <div className="grid grid-cols-2 sm:grid-cols-3 gap-2.5">
                {[
                  { id: 'wordpress', name: 'WordPress', badge: 'v6.7', icon: Globe, desc: 'World\'s #1 CMS' },
                  { id: 'laravel', name: 'Laravel', badge: 'v11', icon: Code2, desc: 'PHP MVC Framework' },
                  { id: 'nextjs', name: 'Next.js', badge: 'v15', icon: Cpu, desc: 'React SSR Fullstack' },
                  { id: 'drupal', name: 'Drupal', badge: 'v10', icon: Layers, desc: 'Enterprise CMS' },
                  { id: 'phpmyadmin', name: 'phpMyAdmin', badge: 'v5.2', icon: Database, desc: 'MySQL Web Admin' },
                ].map((tpl) => {
                  const Icon = tpl.icon;
                  const isSelected = selectedApp === tpl.id;

                  return (
                    <button
                      key={tpl.id}
                      type="button"
                      onClick={() => setSelectedApp(tpl.id)}
                      className={`p-3 rounded-xl border text-left transition-all ${
                        isSelected
                          ? 'border-purple-500 bg-purple-50/50 dark:bg-purple-950/20 ring-2 ring-purple-500/20'
                          : 'border-slate-200 dark:border-surface-700 hover:border-slate-300 dark:hover:border-surface-600 bg-slate-50 dark:bg-[#121824]'
                      }`}
                    >
                      <div className="flex items-center justify-between mb-1.5">
                        <Icon className={`w-4 h-4 ${isSelected ? 'text-purple-600 dark:text-purple-400' : 'text-slate-500'}`} />
                        <span className="text-[10px] font-black px-1.5 py-0.5 rounded bg-slate-200 dark:bg-surface-800 text-slate-700 dark:text-slate-300">
                          {tpl.badge}
                        </span>
                      </div>
                      <div className="font-bold text-xs text-slate-950 dark:text-white">{tpl.name}</div>
                      <div className="text-[10px] text-slate-500 dark:text-slate-400 mt-0.5">{tpl.desc}</div>
                    </button>
                  );
                })}
              </div>
            </div>

            {/* Application Configuration Fields */}
            <div className="space-y-3.5 bg-slate-50 dark:bg-[#121824] p-4 rounded-xl border border-slate-200 dark:border-surface-700">
              <div className="text-xs font-bold text-slate-900 dark:text-slate-100 flex items-center gap-1.5">
                <Sparkles className="w-3.5 h-3.5 text-purple-600 dark:text-purple-400" />
                Quick Installation Parameters
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
                  className="w-full px-3 py-2 rounded-lg border border-slate-300 dark:border-surface-700 bg-white dark:bg-surface-800 text-xs font-medium text-slate-950 dark:text-white"
                />
              </div>

              <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
                <div>
                  <label className="block text-[11px] font-bold text-slate-700 dark:text-slate-300 mb-1">
                    Administrator Username
                  </label>
                  <input
                    type="text"
                    required
                    value={adminUser}
                    onChange={(e) => setAdminUser(e.target.value)}
                    className="w-full px-3 py-2 rounded-lg border border-slate-300 dark:border-surface-700 bg-white dark:bg-surface-800 text-xs font-medium text-slate-950 dark:text-white"
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
                    className="w-full px-3 py-2 rounded-lg border border-slate-300 dark:border-surface-700 bg-white dark:bg-surface-800 text-xs font-medium text-slate-950 dark:text-white"
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
                    Generate Random
                  </button>
                </div>
                <div className="relative">
                  <input
                    type={showPassword ? 'text' : 'password'}
                    required
                    value={adminPassword}
                    onChange={(e) => setAdminPassword(e.target.value)}
                    className="w-full pl-3 pr-10 py-2 rounded-lg border border-slate-300 dark:border-surface-700 bg-white dark:bg-surface-800 text-xs font-mono text-slate-950 dark:text-white"
                  />
                  <button
                    type="button"
                    onClick={() => setShowPassword(!showPassword)}
                    className="absolute right-3 top-2.5 text-slate-400 hover:text-slate-600"
                  >
                    {showPassword ? <EyeOff className="w-3.5 h-3.5" /> : <Eye className="w-3.5 h-3.5" />}
                  </button>
                </div>
              </div>
            </div>

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
                className="px-4 py-2 rounded-xl border border-slate-300 dark:border-surface-700 text-xs font-semibold text-slate-700 dark:text-slate-300 disabled:opacity-50"
              >
                Cancel
              </button>
              <button
                type="submit"
                disabled={installing}
                className="inline-flex items-center gap-2 px-5 py-2 rounded-xl bg-purple-600 hover:bg-purple-500 text-white text-xs font-bold shadow-md transition-all disabled:opacity-50"
              >
                <Zap className="w-3.5 h-3.5" />
                <span>{installing ? 'Deploying...' : `Install ${selectedApp === 'wordpress' ? 'WordPress' : selectedApp}`}</span>
              </button>
            </div>
          </form>
        )}
      </div>
    </div>
  );
}
