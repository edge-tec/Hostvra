'use client';

import React, { useState, useEffect, useCallback } from 'react';
import {
  Globe,
  ExternalLink,
  Zap,
  RefreshCw,
  Shield,
  ShieldCheck,
  Database,
  AlertTriangle,
  CheckCircle2,
  Key,
  Layers,
  Search,
  Plus,
  Trash2,
  Check,
  Copy,
  Sparkles,
  Cpu,
  FileCode,
  Lock,
  Eye,
  EyeOff,
  Sliders,
  Terminal,
} from 'lucide-react';
import {
  Website,
  WordPressDetails,
  WordPressPlugin,
  WordPressTheme,
  PopularPlugin,
  WordPressStagingRecord,
  getWordPressDetails,
  toggleWordPressMaintenance,
  toggleWordPressDebug,
  flushWordPressCache,
  optimizeWordPressDB,
  resetWordPressPassword,
  listWordPressPlugins,
  toggleWordPressPlugin,
  installWordPressPlugin,
  deleteWordPressPlugin,
  listWordPressThemes,
  activateWordPressTheme,
  updateWordPressCore,
  getPopularWordPressPlugins,
  installWordPress,
  getWordPressStaging,
  createWordPressStaging,
  pushWordPressStagingToLive,
  deleteWordPressStaging,
} from '@/lib/api';

interface WordPressManagerTabProps {
  website: Website;
  showToast: (msg: string) => void;
}

export function WordPressManagerTab({ website, showToast }: WordPressManagerTabProps) {
  const [details, setDetails] = useState<WordPressDetails | null>(null);
  const [plugins, setPlugins] = useState<WordPressPlugin[]>([]);
  const [themes, setThemes] = useState<WordPressTheme[]>([]);
  const [popularPlugins, setPopularPlugins] = useState<PopularPlugin[]>([]);
  const [loading, setLoading] = useState(true);
  const [submitting, setSubmitting] = useState<string | null>(null);

  // Active sub-tab: overview, plugins, themes, security, staging
  const [activeSubTab, setActiveSubTab] = useState<'overview' | 'plugins' | 'themes' | 'security' | 'staging'>('overview');

  // Staging environment state
  const [staging, setStaging] = useState<WordPressStagingRecord | null>(null);
  const [stagingLoading, setStagingLoading] = useState(false);
  const [creatingStaging, setCreatingStaging] = useState(false);
  const [pushingLive, setPushingLive] = useState(false);
  const [stagingSubdomain, setStagingSubdomain] = useState('staging');
  const [stagingAuthUser, setStagingAuthUser] = useState('stg_admin');
  const [stagingAuthPass, setStagingAuthPass] = useState('StagingPass2026!');
  const [showStagingModal, setShowStagingModal] = useState(false);
  const [pushSuccessBackup, setPushSuccessBackup] = useState<string | null>(null);

  const fetchStaging = useCallback(async () => {
    if (!website?.id) return;
    setStagingLoading(true);
    try {
      const res = await getWordPressStaging(website.id);
      if (res.success && res.data) {
        setStaging(res.data);
      } else {
        setStaging(null);
      }
    } catch {
      // Ignored
    } finally {
      setStagingLoading(false);
    }
  }, [website?.id]);

  useEffect(() => {
    if (activeSubTab === 'staging') {
      fetchStaging();
    }
  }, [activeSubTab, fetchStaging]);

  const handleCreateStaging = async (e: React.FormEvent) => {
    e.preventDefault();
    setCreatingStaging(true);
    try {
      const res = await createWordPressStaging(website.id, {
        staging_subdomain: stagingSubdomain.trim() || 'staging',
        auth_user: stagingAuthUser.trim() || 'stg_admin',
        auth_password: stagingAuthPass.trim() || 'StagingPass2026!',
      });
      if (res.success) {
        showToast('WordPress staging environment created successfully!');
        setShowStagingModal(false);
        fetchStaging();
      } else {
        showToast(res.error?.message || 'Failed to create staging environment');
      }
    } catch (err: any) {
      showToast(err?.message || 'Failed to create staging environment');
    } finally {
      setCreatingStaging(false);
    }
  };

  const handlePushToLive = async () => {
    if (!confirm('Promote staging changes to PRODUCTION? An automated pre-flight safety backup of production will be taken before overwrite.')) return;
    setPushingLive(true);
    try {
      const res = await pushWordPressStagingToLive(website.id);
      if (res.success && res.data) {
        setPushSuccessBackup(res.data.backup_path);
        showToast('Staging pushed to live production successfully!');
        fetchStaging();
      } else {
        showToast(res.error?.message || 'Failed to push to live');
      }
    } catch (err: any) {
      showToast(err?.message || 'Push to live failed');
    } finally {
      setPushingLive(false);
    }
  };

  const handleDeleteStaging = async () => {
    if (!confirm('Delete this staging environment? All staging files and database clone will be removed.')) return;
    try {
      const res = await deleteWordPressStaging(website.id);
      if (res.success) {
        setStaging(null);
        showToast('Staging environment deleted.');
      } else {
        showToast(res.error?.message || 'Failed to delete staging');
      }
    } catch (err: any) {
      showToast(err?.message || 'Failed to delete staging');
    }
  };

  // Search filter for plugins
  const [pluginSearch, setPluginSearch] = useState('');
  const [pluginFilter, setPluginFilter] = useState<'all' | 'active' | 'inactive'>('all');

  // Password reset modal
  const [resetModalOpen, setResetModalOpen] = useState(false);
  const [resetUsername, setResetUsername] = useState('admin');
  const [resetPassword, setResetPassword] = useState('');
  const [showPassword, setShowPassword] = useState(false);

  // 1-Click Installation state
  const [isInstalling, setIsInstalling] = useState(false);
  const [installProgress, setInstallProgress] = useState(0);
  const [installStep, setInstallStep] = useState('');
  const [installError, setInstallError] = useState<string | null>(null);
  const [installSuccess, setInstallSuccess] = useState<WordPressDetails | null>(null);
  const [showAdvancedInstall, setShowAdvancedInstall] = useState(false);

  const cleanDomain = (website.primary_domain || 'site').replace(/[^a-zA-Z0-9]/g, '_').slice(0, 10);
  const [siteTitle, setSiteTitle] = useState(`${website.primary_domain}`);
  const [adminUser, setAdminUser] = useState('admin');
  const [adminEmail, setAdminEmail] = useState(`admin@${website.primary_domain}`);
  const [adminPassword, setAdminPassword] = useState('');
  const [showInstallPassword, setShowInstallPassword] = useState(false);
  const [dbName, setDbName] = useState(`wp_${cleanDomain}`);
  const [selectedPlugins, setSelectedPlugins] = useState<string[]>(['litespeed-cache', 'wordpress-seo']);
  const [copiedField, setCopiedField] = useState<string | null>(null);

  useEffect(() => {
    const chars = 'abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*()_+';
    let pwd = '';
    for (let i = 0; i < 18; i++) {
      pwd += chars.charAt(Math.floor(Math.random() * chars.length));
    }
    setAdminPassword(pwd);
  }, []);

  const handleGeneratePassword = () => {
    const chars = 'abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*()_+';
    let pwd = '';
    for (let i = 0; i < 18; i++) {
      pwd += chars.charAt(Math.floor(Math.random() * chars.length));
    }
    setAdminPassword(pwd);
  };

  const handleInstallWordPress = async () => {
    try {
      setIsInstalling(true);
      setInstallError(null);
      setInstallProgress(15);
      setInstallStep('Validating environment & document root...');

      const timer1 = setTimeout(() => {
        setInstallProgress(35);
        setInstallStep('Provisioning isolated MySQL database & credentials...');
      }, 500);

      const timer2 = setTimeout(() => {
        setInstallProgress(65);
        setInstallStep('Downloading official WordPress core & verifying archive...');
      }, 1200);

      const timer3 = setTimeout(() => {
        setInstallProgress(85);
        setInstallStep('Generating wp-config.php, salts & running core install...');
      }, 2200);

      const res = await installWordPress(website.id, {
        site_title: siteTitle,
        admin_user: adminUser,
        admin_email: adminEmail,
        admin_password: adminPassword,
        db_name: dbName,
        install_plugins: selectedPlugins,
      });

      clearTimeout(timer1);
      clearTimeout(timer2);
      clearTimeout(timer3);

      if (res.success && res.data) {
        setInstallProgress(100);
        setInstallStep('WordPress installed and verified successfully!');
        setInstallSuccess(res.data);
        showToast('WordPress installed successfully!');
      } else {
        throw new Error(res.error?.message || 'WordPress installation failed');
      }
    } catch (err: any) {
      setInstallError(err.message || 'Installation encountered an unexpected error');
      showToast(err.message || 'Installation failed');
    } finally {
      setIsInstalling(false);
    }
  };

  const loadAllData = useCallback(async () => {
    try {
      setLoading(true);
      const [detRes, plugRes, thmRes, popRes] = await Promise.all([
        getWordPressDetails(website.id),
        listWordPressPlugins(website.id),
        listWordPressThemes(website.id),
        getPopularWordPressPlugins(),
      ]);

      if (detRes.success && detRes.data) {
        setDetails(detRes.data);
      }
      if (plugRes.success && plugRes.data) {
        setPlugins(plugRes.data);
      }
      if (thmRes.success && thmRes.data) {
        setThemes(thmRes.data);
      }
      if (popRes.success && popRes.data) {
        setPopularPlugins(popRes.data);
      }
    } catch (err: any) {
      showToast(err.message || 'Failed to load WordPress details');
    } finally {
      setLoading(false);
    }
  }, [website.id, showToast]);

  useEffect(() => {
    loadAllData();
  }, [loadAllData]);

  const handleToggleMaintenance = async () => {
    if (!details) return;
    try {
      setSubmitting('maintenance');
      const targetState = !details.maintenance_mode;
      const res = await toggleWordPressMaintenance(website.id, targetState);
      if (res.success) {
        setDetails({ ...details, maintenance_mode: targetState });
        showToast(targetState ? 'Maintenance mode enabled' : 'Maintenance mode disabled');
      } else {
        showToast(res.error?.message || 'Failed to toggle maintenance mode');
      }
    } catch (err: any) {
      showToast(err.message || 'Action failed');
    } finally {
      setSubmitting(null);
    }
  };

  const handleToggleDebug = async () => {
    if (!details) return;
    try {
      setSubmitting('debug');
      const targetState = !details.debug_mode;
      const res = await toggleWordPressDebug(website.id, targetState);
      if (res.success) {
        setDetails({ ...details, debug_mode: targetState });
        showToast(targetState ? 'WP_DEBUG enabled' : 'WP_DEBUG disabled');
      } else {
        showToast(res.error?.message || 'Failed to toggle debug mode');
      }
    } catch (err: any) {
      showToast(err.message || 'Action failed');
    } finally {
      setSubmitting(null);
    }
  };

  const handleFlushCache = async () => {
    try {
      setSubmitting('cache');
      const res = await flushWordPressCache(website.id);
      if (res.success) {
        showToast(res.data?.message || 'WordPress cache cleared successfully!');
      } else {
        showToast(res.error?.message || 'Cache flush failed');
      }
    } catch (err: any) {
      showToast(err.message || 'Cache flush failed');
    } finally {
      setSubmitting(null);
    }
  };

  const handleOptimizeDB = async () => {
    try {
      setSubmitting('db');
      const res = await optimizeWordPressDB(website.id);
      if (res.success) {
        showToast(res.data?.message || 'Database tables optimized!');
      } else {
        showToast(res.error?.message || 'Database optimization failed');
      }
    } catch (err: any) {
      showToast(err.message || 'Database optimization failed');
    } finally {
      setSubmitting(null);
    }
  };

  const handleUpdateCore = async () => {
    try {
      setSubmitting('core');
      const res = await updateWordPressCore(website.id);
      if (res.success) {
        showToast(res.data?.message || 'WordPress core updated to latest version!');
        loadAllData();
      } else {
        showToast(res.error?.message || 'Core update failed');
      }
    } catch (err: any) {
      showToast(err.message || 'Core update failed');
    } finally {
      setSubmitting(null);
    }
  };

  const handleResetPassword = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!resetPassword) {
      showToast('Please enter a new password');
      return;
    }
    try {
      setSubmitting('reset-pwd');
      const res = await resetWordPressPassword(website.id, resetUsername, resetPassword);
      if (res.success) {
        showToast(`Password for user '${resetUsername}' has been reset!`);
        setResetModalOpen(false);
        setResetPassword('');
      } else {
        showToast(res.error?.message || 'Password reset failed');
      }
    } catch (err: any) {
      showToast(err.message || 'Password reset failed');
    } finally {
      setSubmitting(null);
    }
  };

  const handleTogglePlugin = async (slug: string, currentStatus: string) => {
    const activate = currentStatus !== 'active';
    try {
      setSubmitting(`plug-${slug}`);
      const res = await toggleWordPressPlugin(website.id, slug, activate);
      if (res.success) {
        setPlugins((prev) =>
          prev.map((p) =>
            p.name === slug ? { ...p, status: activate ? 'active' : 'inactive' } : p
          )
        );
        showToast(`Plugin ${activate ? 'activated' : 'deactivated'}: ${slug}`);
      } else {
        showToast(res.error?.message || 'Plugin state change failed');
      }
    } catch (err: any) {
      showToast(err.message || 'Action failed');
    } finally {
      setSubmitting(null);
    }
  };

  const handleInstallPlugin = async (slug: string) => {
    try {
      setSubmitting(`inst-${slug}`);
      const res = await installWordPressPlugin(website.id, slug, true);
      if (res.success) {
        showToast(`Plugin installed and activated: ${slug}`);
        const plugRes = await listWordPressPlugins(website.id);
        if (plugRes.success && plugRes.data) setPlugins(plugRes.data);
      } else {
        showToast(res.error?.message || 'Plugin install failed');
      }
    } catch (err: any) {
      showToast(err.message || 'Plugin install failed');
    } finally {
      setSubmitting(null);
    }
  };

  const handleDeletePlugin = async (slug: string) => {
    if (!confirm(`Are you sure you want to delete plugin '${slug}'?`)) return;
    try {
      setSubmitting(`del-${slug}`);
      const res = await deleteWordPressPlugin(website.id, slug);
      if (res.success) {
        setPlugins((prev) => prev.filter((p) => p.name !== slug));
        showToast(`Plugin deleted: ${slug}`);
      } else {
        showToast(res.error?.message || 'Plugin deletion failed');
      }
    } catch (err: any) {
      showToast(err.message || 'Plugin deletion failed');
    } finally {
      setSubmitting(null);
    }
  };

  const handleActivateTheme = async (slug: string) => {
    try {
      setSubmitting(`thm-${slug}`);
      const res = await activateWordPressTheme(website.id, slug);
      if (res.success) {
        setThemes((prev) =>
          prev.map((t) => ({
            ...t,
            status: t.name === slug ? 'active' : 'inactive',
          }))
        );
        showToast(`Theme activated: ${slug}`);
      } else {
        showToast(res.error?.message || 'Theme activation failed');
      }
    } catch (err: any) {
      showToast(err.message || 'Action failed');
    } finally {
      setSubmitting(null);
    }
  };

  const generateRandomPassword = () => {
    const chars = 'abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*()_+';
    let pwd = '';
    for (let i = 0; i < 18; i++) {
      pwd += chars.charAt(Math.floor(Math.random() * chars.length));
    }
    setResetPassword(pwd);
  };

  if (loading) {
    return (
      <div className="py-16 text-center text-slate-400 animate-fadeIn">
        <RefreshCw className="w-8 h-8 animate-spin mx-auto mb-3 text-purple-500" />
        <p className="text-sm font-semibold">Inspecting WordPress runtime & WP-CLI environment...</p>
      </div>
    );
  }

  if (installSuccess) {
    return (
      <div className="p-6 rounded-2xl bg-gradient-to-br from-emerald-500/10 via-surface-900/40 to-emerald-500/5 border border-emerald-500/30 space-y-6 animate-fadeIn">
        <div className="flex items-start gap-4">
          <div className="w-12 h-12 rounded-xl bg-emerald-500/20 text-emerald-600 dark:text-emerald-400 flex items-center justify-center flex-shrink-0">
            <CheckCircle2 className="w-7 h-7" />
          </div>
          <div>
            <h3 className="text-base font-bold text-slate-900 dark:text-white">WordPress Successfully Installed!</h3>
            <p className="text-xs text-slate-500 dark:text-slate-400 mt-1">
              Your WordPress website on <strong className="text-emerald-500">{website.primary_domain}</strong> is live and fully verified.
            </p>
          </div>
        </div>

        <div className="grid grid-cols-1 md:grid-cols-2 gap-3 p-4 rounded-xl bg-slate-50 dark:bg-surface-800/80 border border-slate-200 dark:border-surface-700/60 font-mono text-xs">
          <div>
            <span className="text-slate-400 text-[10px] uppercase font-bold block mb-1">Site URL</span>
            <a href={`https://${website.primary_domain}`} target="_blank" rel="noopener noreferrer" className="text-purple-600 dark:text-purple-400 hover:underline flex items-center gap-1.5 truncate">
              https://{website.primary_domain} <ExternalLink className="w-3 h-3" />
            </a>
          </div>
          <div>
            <span className="text-slate-400 text-[10px] uppercase font-bold block mb-1">Admin Login URL</span>
            <a href={installSuccess.admin_url || `https://${website.primary_domain}/wp-admin`} target="_blank" rel="noopener noreferrer" className="text-emerald-600 dark:text-emerald-400 hover:underline flex items-center gap-1.5 truncate">
              {installSuccess.admin_url || `https://${website.primary_domain}/wp-admin`} <ExternalLink className="w-3 h-3" />
            </a>
          </div>
          <div>
            <span className="text-slate-400 text-[10px] uppercase font-bold block mb-1">Admin Username</span>
            <span className="text-slate-800 dark:text-slate-200 font-bold">{adminUser}</span>
          </div>
          <div>
            <span className="text-slate-400 text-[10px] uppercase font-bold block mb-1">Admin Password</span>
            <div className="flex items-center gap-2">
              <span className="text-slate-800 dark:text-slate-200 font-bold">{adminPassword}</span>
              <button
                type="button"
                onClick={() => {
                  navigator.clipboard.writeText(adminPassword);
                  setCopiedField('password');
                  setTimeout(() => setCopiedField(null), 2000);
                }}
                className="text-xs text-purple-500 hover:text-purple-400 transition"
                title="Copy Password"
              >
                {copiedField === 'password' ? <Check className="w-3.5 h-3.5 text-emerald-500" /> : <Copy className="w-3.5 h-3.5" />}
              </button>
            </div>
          </div>
        </div>

        <div className="flex flex-wrap items-center gap-3 pt-2">
          <a
            href={installSuccess.admin_url || `https://${website.primary_domain}/wp-admin`}
            target="_blank"
            rel="noopener noreferrer"
            className="px-4 py-2.5 rounded-xl bg-purple-600 hover:bg-purple-700 text-white font-semibold text-xs shadow-lg shadow-purple-500/20 transition inline-flex items-center gap-2"
          >
            <span>Open WP-Admin</span> <ExternalLink className="w-3.5 h-3.5" />
          </a>
          <button
            type="button"
            onClick={() => {
              setInstallSuccess(null);
              loadAllData();
            }}
            className="px-4 py-2.5 rounded-xl border border-slate-300 dark:border-surface-700 hover:bg-slate-100 dark:hover:bg-surface-800 text-slate-700 dark:text-slate-200 font-semibold text-xs transition inline-flex items-center gap-2"
          >
            <Sliders className="w-3.5 h-3.5 text-purple-500" />
            <span>Manage in WordPress Toolkit</span>
          </button>
        </div>
      </div>
    );
  }

  if (!details?.is_installed) {
    return (
      <div className="p-6 rounded-2xl bg-gradient-to-br from-purple-500/5 via-slate-50 dark:via-surface-900/40 to-indigo-500/5 border border-purple-500/20 shadow-sm animate-fadeIn space-y-6">
        {/* Header */}
        <div className="flex flex-col md:flex-row md:items-center justify-between gap-4 pb-4 border-b border-slate-200 dark:border-surface-800">
          <div className="flex items-center gap-3.5">
            <div className="w-12 h-12 rounded-xl bg-purple-600 text-white flex items-center justify-center font-black text-2xl shadow-lg shadow-purple-500/30 flex-shrink-0">
              W
            </div>
            <div>
              <h3 className="text-base font-bold text-slate-900 dark:text-white">1-Click WordPress Installation</h3>
              <p className="text-xs text-slate-500 dark:text-slate-400 mt-0.5">
                Deploy an official, high-performance WordPress CMS on <span className="font-semibold text-purple-600 dark:text-purple-400">{website.primary_domain}</span>
              </p>
            </div>
          </div>
          <div className="flex items-center gap-2 text-xs">
            <span className="px-2.5 py-1 rounded-lg bg-slate-100 dark:bg-surface-800 text-slate-600 dark:text-slate-400 border border-slate-200 dark:border-surface-700 font-mono">
              {website.document_root}
            </span>
          </div>
        </div>

        {/* Installation Error Banner */}
        {installError && (
          <div className="p-3.5 rounded-xl bg-rose-500/10 border border-rose-500/30 text-rose-600 dark:text-rose-400 text-xs flex items-center gap-2.5">
            <AlertTriangle className="w-4 h-4 flex-shrink-0" />
            <span className="flex-1 font-medium">{installError}</span>
            <button
              onClick={() => setInstallError(null)}
              className="text-xs font-bold hover:underline"
            >
              Dismiss
            </button>
          </div>
        )}

        {/* Progress Bar during installation */}
        {isInstalling && (
          <div className="p-4 rounded-xl bg-purple-500/10 border border-purple-500/30 space-y-2.5 animate-pulse">
            <div className="flex items-center justify-between text-xs font-semibold text-purple-700 dark:text-purple-300">
              <span className="flex items-center gap-2">
                <RefreshCw className="w-3.5 h-3.5 animate-spin" />
                {installStep}
              </span>
              <span>{installProgress}%</span>
            </div>
            <div className="w-full h-2 rounded-full bg-slate-200 dark:bg-surface-800 overflow-hidden">
              <div
                className="h-full bg-gradient-to-r from-purple-500 to-indigo-500 transition-all duration-500 rounded-full"
                style={{ width: `${installProgress}%` }}
              />
            </div>
          </div>
        )}

        {/* Form Fields */}
        <form onSubmit={(e) => { e.preventDefault(); handleInstallWordPress(); }} className="space-y-4">
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div>
              <label className="block text-xs font-semibold text-slate-700 dark:text-slate-300 mb-1.5">
                Site Title
              </label>
              <input
                type="text"
                value={siteTitle}
                onChange={(e) => setSiteTitle(e.target.value)}
                disabled={isInstalling}
                placeholder="My WordPress Site"
                className="w-full px-3 py-2 rounded-xl bg-white dark:bg-surface-800 border border-slate-300 dark:border-surface-700 text-xs text-slate-900 dark:text-white focus:outline-none focus:border-purple-500"
                required
              />
            </div>

            <div>
              <label className="block text-xs font-semibold text-slate-700 dark:text-slate-300 mb-1.5">
                Admin Username
              </label>
              <input
                type="text"
                value={adminUser}
                onChange={(e) => setAdminUser(e.target.value)}
                disabled={isInstalling}
                placeholder="admin"
                className="w-full px-3 py-2 rounded-xl bg-white dark:bg-surface-800 border border-slate-300 dark:border-surface-700 text-xs text-slate-900 dark:text-white focus:outline-none focus:border-purple-500"
                required
              />
            </div>

            <div>
              <label className="block text-xs font-semibold text-slate-700 dark:text-slate-300 mb-1.5">
                Admin Email
              </label>
              <input
                type="email"
                value={adminEmail}
                onChange={(e) => setAdminEmail(e.target.value)}
                disabled={isInstalling}
                placeholder="admin@domain.com"
                className="w-full px-3 py-2 rounded-xl bg-white dark:bg-surface-800 border border-slate-300 dark:border-surface-700 text-xs text-slate-900 dark:text-white focus:outline-none focus:border-purple-500"
                required
              />
            </div>

            <div>
              <div className="flex items-center justify-between mb-1.5">
                <label className="block text-xs font-semibold text-slate-700 dark:text-slate-300">
                  Admin Password
                </label>
                <button
                  type="button"
                  onClick={handleGeneratePassword}
                  disabled={isInstalling}
                  className="text-[11px] text-purple-600 dark:text-purple-400 hover:underline inline-flex items-center gap-1 font-semibold"
                >
                  <RefreshCw className="w-3 h-3" /> Generate New
                </button>
              </div>
              <div className="relative">
                <input
                  type={showInstallPassword ? 'text' : 'password'}
                  value={adminPassword}
                  onChange={(e) => setAdminPassword(e.target.value)}
                  disabled={isInstalling}
                  placeholder="Strong password"
                  className="w-full pl-3 pr-16 py-2 rounded-xl bg-white dark:bg-surface-800 border border-slate-300 dark:border-surface-700 text-xs text-slate-900 dark:text-white focus:outline-none focus:border-purple-500 font-mono"
                  required
                />
                <div className="absolute right-2 top-1/2 -translate-y-1/2 flex items-center gap-1">
                  <button
                    type="button"
                    onClick={() => setShowInstallPassword(!showInstallPassword)}
                    className="p-1 text-slate-400 hover:text-slate-600 dark:hover:text-slate-200"
                  >
                    {showInstallPassword ? <EyeOff className="w-3.5 h-3.5" /> : <Eye className="w-3.5 h-3.5" />}
                  </button>
                  <button
                    type="button"
                    onClick={() => {
                      navigator.clipboard.writeText(adminPassword);
                      setCopiedField('form_pwd');
                      setTimeout(() => setCopiedField(null), 2000);
                    }}
                    className="p-1 text-slate-400 hover:text-slate-600 dark:hover:text-slate-200"
                    title="Copy Password"
                  >
                    {copiedField === 'form_pwd' ? <Check className="w-3.5 h-3.5 text-emerald-500" /> : <Copy className="w-3.5 h-3.5" />}
                  </button>
                </div>
              </div>
            </div>
          </div>

          {/* Advanced Options Accordion */}
          <div className="pt-2">
            <button
              type="button"
              onClick={() => setShowAdvancedInstall(!showAdvancedInstall)}
              className="text-xs text-slate-500 hover:text-slate-700 dark:text-slate-400 dark:hover:text-slate-200 font-semibold inline-flex items-center gap-1.5 transition"
            >
              <Sliders className="w-3.5 h-3.5 text-purple-500" />
              <span>{showAdvancedInstall ? 'Hide Database & Plugin Options' : 'Show Database & Plugin Options'}</span>
            </button>

            {showAdvancedInstall && (
              <div className="mt-3 p-4 rounded-xl bg-slate-100/60 dark:bg-surface-800/50 border border-slate-200 dark:border-surface-700/60 space-y-3.5 animate-fadeIn">
                <div>
                  <label className="block text-xs font-semibold text-slate-700 dark:text-slate-300 mb-1">
                    Database Name
                  </label>
                  <input
                    type="text"
                    value={dbName}
                    onChange={(e) => setDbName(e.target.value)}
                    disabled={isInstalling}
                    placeholder="wp_database"
                    className="w-full max-w-sm px-3 py-1.5 rounded-lg bg-white dark:bg-surface-800 border border-slate-300 dark:border-surface-700 text-xs font-mono text-slate-900 dark:text-white focus:outline-none focus:border-purple-500"
                  />
                  <p className="text-[11px] text-slate-400 mt-1">
                    A dedicated MySQL database and isolated user will be provisioned automatically.
                  </p>
                </div>

                <div>
                  <label className="block text-xs font-semibold text-slate-700 dark:text-slate-300 mb-1.5">
                    Pre-install Popular Plugins
                  </label>
                  <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 gap-2">
                    {[
                      { id: 'litespeed-cache', label: 'LiteSpeed Cache' },
                      { id: 'wordpress-seo', label: 'Yoast SEO' },
                      { id: 'elementor', label: 'Elementor Builder' },
                      { id: 'woocommerce', label: 'WooCommerce' },
                      { id: 'wordfence', label: 'Wordfence Security' },
                    ].map((plug) => {
                      const isChecked = selectedPlugins.includes(plug.id);
                      return (
                        <label
                          key={plug.id}
                          className={`flex items-center gap-2 p-2 rounded-lg border text-xs cursor-pointer transition ${
                            isChecked
                              ? 'bg-purple-500/10 border-purple-500/30 text-purple-700 dark:text-purple-300 font-semibold'
                              : 'bg-white dark:bg-surface-800 border-slate-200 dark:border-surface-700 text-slate-600 dark:text-slate-400'
                          }`}
                        >
                          <input
                            type="checkbox"
                            checked={isChecked}
                            onChange={(e) => {
                              if (e.target.checked) {
                                setSelectedPlugins([...selectedPlugins, plug.id]);
                              } else {
                                setSelectedPlugins(selectedPlugins.filter((id) => id !== plug.id));
                              }
                            }}
                            disabled={isInstalling}
                            className="rounded text-purple-600 focus:ring-purple-500"
                          />
                          <span>{plug.label}</span>
                        </label>
                      );
                    })}
                  </div>
                </div>
              </div>
            )}
          </div>

          {/* Action Bar */}
          <div className="pt-2 flex items-center gap-3">
            <button
              type="submit"
              disabled={isInstalling}
              className="px-5 py-2.5 rounded-xl bg-purple-600 hover:bg-purple-700 active:bg-purple-800 disabled:opacity-50 text-white font-semibold text-xs shadow-lg shadow-purple-500/20 transition-all inline-flex items-center gap-2 cursor-pointer"
            >
              {isInstalling ? (
                <>
                  <RefreshCw className="w-4 h-4 animate-spin" />
                  <span>Installing WordPress...</span>
                </>
              ) : (
                <>
                  <Sparkles className="w-4 h-4" />
                  <span>Install WordPress Now</span>
                </>
              )}
            </button>
            <span className="text-[11px] text-slate-400">
              Includes latest WordPress core, automated database schema, and security salts.
            </span>
          </div>
        </form>
      </div>
    );
  }

  const filteredPlugins = plugins.filter((p) => {
    if (pluginFilter === 'active' && p.status !== 'active') return false;
    if (pluginFilter === 'inactive' && p.status === 'active') return false;
    if (pluginSearch && !p.name.toLowerCase().includes(pluginSearch.toLowerCase()) && !p.title.toLowerCase().includes(pluginSearch.toLowerCase())) {
      return false;
    }
    return true;
  });

  const adminURL = details.admin_url || `https://${website.primary_domain}/wp-admin`;

  return (
    <div className="space-y-6 animate-fadeIn text-slate-800 dark:text-slate-200">
      {/* 1. Header Banner */}
      <div className="p-5 rounded-2xl bg-gradient-to-r from-purple-900/20 via-indigo-900/20 to-purple-900/10 border border-purple-500/30 flex flex-col md:flex-row md:items-center justify-between gap-4">
        <div className="flex items-center gap-3.5">
          <div className="w-12 h-12 rounded-xl bg-purple-600 text-white flex items-center justify-center font-black text-2xl shadow-lg shadow-purple-500/30 flex-shrink-0">
            W
          </div>
          <div>
            <div className="flex items-center gap-2">
              <h2 className="text-lg font-bold text-slate-900 dark:text-white">
                {details.site_title || website.primary_domain}
              </h2>
              <span className="px-2 py-0.5 rounded-full text-[10px] font-black bg-purple-500/20 text-purple-600 dark:text-purple-300 border border-purple-500/30">
                v{details.version || '6.7.x'}
              </span>
              {details.wp_cli_installed && (
                <span className="px-2 py-0.5 rounded-full text-[10px] font-bold bg-emerald-500/20 text-emerald-600 dark:text-emerald-400 border border-emerald-500/30 flex items-center gap-1">
                  <Terminal className="w-3 h-3" /> WP-CLI
                </span>
              )}
            </div>
            <p className="text-xs text-slate-500 dark:text-slate-400 mt-0.5 font-mono">
              DB: {details.db_name} (prefix: {details.db_prefix || 'wp_'})
            </p>
          </div>
        </div>

        <div className="flex items-center gap-2">
          <a
            href={adminURL}
            target="_blank"
            rel="noopener noreferrer"
            className="px-3.5 py-2 rounded-xl bg-purple-600 hover:bg-purple-700 text-white font-semibold text-xs shadow-md shadow-purple-500/20 transition-all inline-flex items-center gap-1.5"
          >
            <span>Open WP-Admin</span>
            <ExternalLink className="w-3.5 h-3.5" />
          </a>
          <button
            onClick={loadAllData}
            disabled={loading}
            className="p-2 rounded-xl border border-slate-200 dark:border-surface-700 hover:bg-slate-100 dark:hover:bg-surface-800 text-slate-500 transition-colors"
            title="Refresh State"
          >
            <RefreshCw className={`w-4 h-4 ${loading ? 'animate-spin' : ''}`} />
          </button>
        </div>
      </div>

      {/* 2. Sub-Navigation Tabs */}
      <div className="flex items-center gap-1 border-b border-slate-200 dark:border-surface-800 pb-2">
        <button
          onClick={() => setActiveSubTab('overview')}
          className={`px-3 py-1.5 rounded-lg text-xs font-semibold transition-colors ${activeSubTab === 'overview'
              ? 'bg-purple-500/10 text-purple-600 dark:text-purple-400 border border-purple-500/20'
              : 'text-slate-500 hover:text-slate-800 dark:hover:text-white'
            }`}
        >
          <Sliders className="w-3.5 h-3.5 inline mr-1.5" />
          Dashboard & Tools
        </button>
        <button
          onClick={() => setActiveSubTab('plugins')}
          className={`px-3 py-1.5 rounded-lg text-xs font-semibold transition-colors ${activeSubTab === 'plugins'
              ? 'bg-purple-500/10 text-purple-600 dark:text-purple-400 border border-purple-500/20'
              : 'text-slate-500 hover:text-slate-800 dark:hover:text-white'
            }`}
        >
          <Layers className="w-3.5 h-3.5 inline mr-1.5" />
          Plugins ({plugins.length})
        </button>
        <button
          onClick={() => setActiveSubTab('themes')}
          className={`px-3 py-1.5 rounded-lg text-xs font-semibold transition-colors ${activeSubTab === 'themes'
              ? 'bg-purple-500/10 text-purple-600 dark:text-purple-400 border border-purple-500/20'
              : 'text-slate-500 hover:text-slate-800 dark:hover:text-white'
            }`}
        >
          <Globe className="w-3.5 h-3.5 inline mr-1.5" />
          Themes ({themes.length})
        </button>
        <button
          onClick={() => setActiveSubTab('security')}
          className={`px-3 py-1.5 rounded-lg text-xs font-semibold transition-colors ${activeSubTab === 'security'
              ? 'bg-purple-500/10 text-purple-600 dark:text-purple-400 border border-purple-500/20'
              : 'text-slate-500 hover:text-slate-800 dark:hover:text-white'
            }`}
        >
          <Shield className="w-3.5 h-3.5 inline mr-1.5" />
          Security & Admin
        </button>
        <button
          onClick={() => setActiveSubTab('staging')}
          className={`px-3 py-1.5 rounded-lg text-xs font-semibold transition-colors ${activeSubTab === 'staging'
              ? 'bg-purple-500/10 text-purple-600 dark:text-purple-400 border border-purple-500/20'
              : 'text-slate-500 hover:text-slate-800 dark:hover:text-white'
            }`}
        >
          <RefreshCw className="w-3.5 h-3.5 inline mr-1.5" />
          Staging & Cloning
        </button>
      </div>

      {/* 3. Sub-Tab Content: OVERVIEW */}
      {activeSubTab === 'overview' && (
        <div className="space-y-5">
          {/* Status Alert if in Maintenance Mode */}
          {details.maintenance_mode && (
            <div className="p-3.5 rounded-xl bg-amber-500/10 border border-amber-500/20 flex items-center gap-3 text-xs text-amber-700 dark:text-amber-300 font-semibold">
              <AlertTriangle className="w-4 h-4 text-amber-500 flex-shrink-0" />
              <span>
                Maintenance mode is active! Visitors see a "Briefly unavailable for scheduled maintenance" message.
              </span>
            </div>
          )}

          {/* Quick Toggles Grid */}
          <div className="grid grid-cols-1 md:grid-cols-3 gap-3">
            {/* Maintenance Mode */}
            <div className="p-4 rounded-xl border border-slate-200 dark:border-surface-800 bg-slate-50/50 dark:bg-surface-900 flex items-center justify-between">
              <div>
                <div className="text-xs font-bold text-slate-900 dark:text-white flex items-center gap-1.5">
                  <ShieldCheck className="w-3.5 h-3.5 text-purple-500" />
                  Maintenance Mode
                </div>
                <div className="text-[11px] text-slate-500 mt-0.5">Toggle .maintenance lock</div>
              </div>
              <button
                onClick={handleToggleMaintenance}
                disabled={submitting === 'maintenance'}
                className={`relative inline-flex h-5 w-10 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-hidden ${details.maintenance_mode ? 'bg-amber-500' : 'bg-slate-300 dark:bg-surface-700'
                  }`}
              >
                <span
                  className={`pointer-events-none inline-block h-4 w-4 transform rounded-full bg-white shadow-sm ring-0 transition duration-200 ease-in-out ${details.maintenance_mode ? 'translate-x-5' : 'translate-x-0'
                    }`}
                />
              </button>
            </div>

            {/* WP_DEBUG Mode */}
            <div className="p-4 rounded-xl border border-slate-200 dark:border-surface-800 bg-slate-50/50 dark:bg-surface-900 flex items-center justify-between">
              <div>
                <div className="text-xs font-bold text-slate-900 dark:text-white flex items-center gap-1.5">
                  <FileCode className="w-3.5 h-3.5 text-purple-500" />
                  WP_DEBUG Mode
                </div>
                <div className="text-[11px] text-slate-500 mt-0.5">Display PHP warnings & notices</div>
              </div>
              <button
                onClick={handleToggleDebug}
                disabled={submitting === 'debug'}
                className={`relative inline-flex h-5 w-10 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-hidden ${details.debug_mode ? 'bg-purple-600' : 'bg-slate-300 dark:bg-surface-700'
                  }`}
              >
                <span
                  className={`pointer-events-none inline-block h-4 w-4 transform rounded-full bg-white shadow-sm ring-0 transition duration-200 ease-in-out ${details.debug_mode ? 'translate-x-5' : 'translate-x-0'
                    }`}
                />
              </button>
            </div>

            {/* Auto Core Updates */}
            <div className="p-4 rounded-xl border border-slate-200 dark:border-surface-800 bg-slate-50/50 dark:bg-surface-900 flex items-center justify-between">
              <div>
                <div className="text-xs font-bold text-slate-900 dark:text-white flex items-center gap-1.5">
                  <Zap className="w-3.5 h-3.5 text-emerald-500" />
                  Auto Updates
                </div>
                <div className="text-[11px] text-slate-500 mt-0.5">Automated minor security patches</div>
              </div>
              <span className="px-2 py-0.5 rounded-full text-[10px] font-black bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 border border-emerald-500/20">
                ACTIVE
              </span>
            </div>
          </div>

          {/* Quick Action Operations */}
          <div className="p-5 rounded-2xl border border-slate-200 dark:border-surface-800 bg-white dark:bg-surface-900 space-y-4">
            <h3 className="text-xs font-bold uppercase tracking-wider text-slate-500 dark:text-slate-400">
              Performance & Maintenance Tools
            </h3>
            <div className="grid grid-cols-1 sm:grid-cols-3 gap-3">
              <button
                onClick={handleFlushCache}
                disabled={submitting === 'cache'}
                className="p-3 rounded-xl border border-slate-200 dark:border-surface-700 hover:border-purple-500 hover:bg-purple-500/5 text-left transition-all flex items-start gap-3 group"
              >
                <div className="p-2 rounded-lg bg-purple-500/10 text-purple-600 dark:text-purple-400 group-hover:scale-110 transition-transform">
                  <Zap className="w-4 h-4" />
                </div>
                <div>
                  <div className="text-xs font-bold text-slate-900 dark:text-white">Flush Cache</div>
                  <div className="text-[11px] text-slate-500 mt-0.5">Purge object & transients cache</div>
                </div>
              </button>

              <button
                onClick={handleOptimizeDB}
                disabled={submitting === 'db'}
                className="p-3 rounded-xl border border-slate-200 dark:border-surface-700 hover:border-purple-500 hover:bg-purple-500/5 text-left transition-all flex items-start gap-3 group"
              >
                <div className="p-2 rounded-lg bg-indigo-500/10 text-indigo-600 dark:text-indigo-400 group-hover:scale-110 transition-transform">
                  <Database className="w-4 h-4" />
                </div>
                <div>
                  <div className="text-xs font-bold text-slate-900 dark:text-white">Optimize Database</div>
                  <div className="text-[11px] text-slate-500 mt-0.5">Defragment MySQL tables</div>
                </div>
              </button>

              <button
                onClick={handleUpdateCore}
                disabled={submitting === 'core'}
                className="p-3 rounded-xl border border-slate-200 dark:border-surface-700 hover:border-purple-500 hover:bg-purple-500/5 text-left transition-all flex items-start gap-3 group"
              >
                <div className="p-2 rounded-lg bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 group-hover:scale-110 transition-transform">
                  <RefreshCw className={`w-4 h-4 ${submitting === 'core' ? 'animate-spin' : ''}`} />
                </div>
                <div>
                  <div className="text-xs font-bold text-slate-900 dark:text-white">Update Core</div>
                  <div className="text-[11px] text-slate-500 mt-0.5">Upgrade to latest WP release</div>
                </div>
              </button>
            </div>
          </div>
        </div>
      )}

      {/* 4. Sub-Tab Content: PLUGINS */}
      {activeSubTab === 'plugins' && (
        <div className="space-y-5">
          {/* Toolbar */}
          <div className="flex flex-col sm:flex-row items-stretch sm:items-center justify-between gap-3">
            <div className="flex items-center gap-2">
              <div className="relative flex-1 sm:w-64">
                <Search className="w-3.5 h-3.5 absolute left-3 top-1/2 -translate-y-1/2 text-slate-400" />
                <input
                  type="text"
                  placeholder="Search plugins..."
                  value={pluginSearch}
                  onChange={(e) => setPluginSearch(e.target.value)}
                  className="w-full pl-8 pr-3 py-1.5 rounded-lg border border-slate-200 dark:border-surface-700 bg-white dark:bg-surface-900 text-xs text-slate-800 dark:text-slate-200"
                />
              </div>
              <div className="flex rounded-lg border border-slate-200 dark:border-surface-700 p-0.5 text-xs">
                {(['all', 'active', 'inactive'] as const).map((mode) => (
                  <button
                    key={mode}
                    onClick={() => setPluginFilter(mode)}
                    className={`px-2.5 py-1 rounded capitalize font-medium ${pluginFilter === mode
                        ? 'bg-purple-600 text-white'
                        : 'text-slate-500 hover:text-slate-800 dark:hover:text-white'
                      }`}
                  >
                    {mode}
                  </button>
                ))}
              </div>
            </div>
          </div>

          {/* Installed Plugins Table */}
          <div className="rounded-xl border border-slate-200 dark:border-surface-800 overflow-hidden">
            <table className="w-full text-left border-collapse text-xs">
              <thead>
                <tr className="bg-slate-50 dark:bg-surface-800/60 border-b border-slate-200 dark:border-surface-700 text-slate-500 font-bold uppercase tracking-wider text-[10px]">
                  <th className="py-2.5 px-3">Plugin</th>
                  <th className="py-2.5 px-3">Status</th>
                  <th className="py-2.5 px-3">Version</th>
                  <th className="py-2.5 px-3 text-right">Actions</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-200 dark:divide-surface-800">
                {filteredPlugins.length === 0 ? (
                  <tr>
                    <td colSpan={4} className="py-8 text-center text-slate-400">
                      No plugins match your filter.
                    </td>
                  </tr>
                ) : (
                  filteredPlugins.map((plug) => {
                    const isActive = plug.status === 'active';
                    return (
                      <tr key={plug.name} className="hover:bg-slate-50/50 dark:hover:bg-surface-800/30 transition-colors">
                        <td className="py-3 px-3">
                          <div className="font-bold text-slate-900 dark:text-white">{plug.title || plug.name}</div>
                          <div className="text-[11px] text-slate-400 font-mono">{plug.name}</div>
                        </td>
                        <td className="py-3 px-3">
                          <span
                            className={`px-2 py-0.5 rounded-full text-[10px] font-bold ${isActive
                                ? 'bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 border border-emerald-500/20'
                                : 'bg-slate-200 dark:bg-surface-700 text-slate-600 dark:text-slate-400'
                              }`}
                          >
                            {plug.status.toUpperCase()}
                          </span>
                        </td>
                        <td className="py-3 px-3 text-slate-500 font-mono">
                          {plug.version || '—'}
                          {plug.update_available && (
                            <span className="ml-2 text-amber-500 font-bold text-[10px]">(Update: {plug.update_version})</span>
                          )}
                        </td>
                        <td className="py-3 px-3 text-right space-x-2">
                          <button
                            onClick={() => handleTogglePlugin(plug.name, plug.status)}
                            disabled={submitting === `plug-${plug.name}`}
                            className={`px-2.5 py-1 rounded-md text-[11px] font-semibold transition-colors ${isActive
                                ? 'bg-slate-100 hover:bg-slate-200 dark:bg-surface-800 dark:hover:bg-surface-700 text-slate-700 dark:text-slate-300'
                                : 'bg-purple-600 hover:bg-purple-700 text-white'
                              }`}
                          >
                            {isActive ? 'Deactivate' : 'Activate'}
                          </button>
                          <button
                            onClick={() => handleDeletePlugin(plug.name)}
                            disabled={submitting === `del-${plug.name}`}
                            className="p-1 rounded-md text-slate-400 hover:text-rose-500 hover:bg-rose-500/10 transition-colors"
                            title="Delete Plugin"
                          >
                            <Trash2 className="w-3.5 h-3.5" />
                          </button>
                        </td>
                      </tr>
                    );
                  })
                )}
              </tbody>
            </table>
          </div>

          {/* Popular Plugins Catalog Recommendations */}
          {popularPlugins.length > 0 && (
            <div className="pt-4 space-y-3">
              <h4 className="text-xs font-bold uppercase tracking-wider text-slate-500 dark:text-slate-400 flex items-center gap-1.5">
                <Sparkles className="w-3.5 h-3.5 text-purple-500" />
                1-Click Popular Plugin Installations
              </h4>
              <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3">
                {popularPlugins.map((pop) => {
                  const alreadyInstalled = plugins.some((p) => p.name === pop.slug);
                  return (
                    <div
                      key={pop.slug}
                      className="p-3.5 rounded-xl border border-slate-200 dark:border-surface-800 bg-white dark:bg-surface-900 flex flex-col justify-between gap-3"
                    >
                      <div>
                        <div className="flex items-center justify-between">
                          <div className="font-bold text-xs text-slate-900 dark:text-white">{pop.name}</div>
                          <span className="text-[10px] font-semibold px-1.5 py-0.5 rounded bg-slate-100 dark:bg-surface-800 text-slate-500">
                            {pop.category}
                          </span>
                        </div>
                        <p className="text-[11px] text-slate-500 dark:text-slate-400 mt-1 line-clamp-2">
                          {pop.description}
                        </p>
                      </div>

                      <div className="pt-2 border-t border-slate-100 dark:border-surface-800/80 flex items-center justify-between">
                        <span className="text-[10px] font-mono text-slate-400">{pop.slug}</span>
                        {alreadyInstalled ? (
                          <span className="text-[11px] font-bold text-emerald-600 dark:text-emerald-400 flex items-center gap-1">
                            <Check className="w-3 h-3" /> Installed
                          </span>
                        ) : (
                          <button
                            onClick={() => handleInstallPlugin(pop.slug)}
                            disabled={submitting === `inst-${pop.slug}`}
                            className="px-2.5 py-1 rounded-lg bg-purple-600 hover:bg-purple-700 text-white text-[11px] font-semibold transition-colors flex items-center gap-1"
                          >
                            <Plus className="w-3 h-3" /> Install
                          </button>
                        )}
                      </div>
                    </div>
                  );
                })}
              </div>
            </div>
          )}
        </div>
      )}

      {/* 5. Sub-Tab Content: THEMES */}
      {activeSubTab === 'themes' && (
        <div className="space-y-4">
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
            {themes.map((thm) => {
              const isActive = thm.status === 'active';
              return (
                <div
                  key={thm.name}
                  className={`p-4 rounded-xl border transition-all ${isActive
                      ? 'border-purple-500/60 bg-purple-500/5 shadow-md shadow-purple-500/10'
                      : 'border-slate-200 dark:border-surface-800 bg-white dark:bg-surface-900'
                    }`}
                >
                  <div className="flex items-center justify-between mb-2">
                    <div className="font-bold text-sm text-slate-900 dark:text-white">{thm.title || thm.name}</div>
                    {isActive ? (
                      <span className="px-2 py-0.5 rounded-full text-[10px] font-black bg-purple-600 text-white">
                        ACTIVE
                      </span>
                    ) : (
                      <span className="text-[11px] text-slate-400">Inactive</span>
                    )}
                  </div>
                  <p className="text-xs text-slate-500 font-mono">Folder: {thm.name}</p>
                  <div className="mt-4 pt-3 border-t border-slate-100 dark:border-surface-800 flex items-center justify-between">
                    <span className="text-xs text-slate-400">v{thm.version || '1.0'}</span>
                    {!isActive && (
                      <button
                        onClick={() => handleActivateTheme(thm.name)}
                        disabled={submitting === `thm-${thm.name}`}
                        className="px-3 py-1 rounded-lg bg-purple-600 hover:bg-purple-700 text-white text-xs font-semibold transition-colors"
                      >
                        Activate
                      </button>
                    )}
                  </div>
                </div>
              );
            })}
          </div>
        </div>
      )}

      {/* 6. Sub-Tab Content: SECURITY & ADMIN */}
      {activeSubTab === 'security' && (
        <div className="space-y-5">
          <div className="p-5 rounded-2xl border border-slate-200 dark:border-surface-800 bg-white dark:bg-surface-900 space-y-4">
            <div className="flex items-center gap-3">
              <div className="p-2.5 rounded-xl bg-purple-500/10 text-purple-600 dark:text-purple-400">
                <Key className="w-5 h-5" />
              </div>
              <div>
                <h3 className="text-sm font-bold text-slate-900 dark:text-white">Emergency Admin Password Reset</h3>
                <p className="text-xs text-slate-500 mt-0.5">
                  Reset any WordPress administrator password directly without requiring email delivery or phpMyAdmin access.
                </p>
              </div>
            </div>

            <form onSubmit={handleResetPassword} className="space-y-3 pt-2 max-w-md">
              <div>
                <label className="block text-xs font-bold text-slate-700 dark:text-slate-300 mb-1">
                  WordPress Username
                </label>
                <input
                  type="text"
                  value={resetUsername}
                  onChange={(e) => setResetUsername(e.target.value)}
                  placeholder="e.g. admin"
                  required
                  className="w-full px-3 py-2 rounded-xl border border-slate-200 dark:border-surface-700 bg-white dark:bg-surface-900 text-xs font-mono"
                />
              </div>

              <div>
                <div className="flex items-center justify-between mb-1">
                  <label className="block text-xs font-bold text-slate-700 dark:text-slate-300">New Password</label>
                  <button
                    type="button"
                    onClick={generateRandomPassword}
                    className="text-[11px] text-purple-600 dark:text-purple-400 font-bold hover:underline"
                  >
                    Generate Strong
                  </button>
                </div>
                <div className="relative">
                  <input
                    type={showPassword ? 'text' : 'password'}
                    value={resetPassword}
                    onChange={(e) => setResetPassword(e.target.value)}
                    placeholder="Enter new password"
                    required
                    className="w-full px-3 py-2 pr-10 rounded-xl border border-slate-200 dark:border-surface-700 bg-white dark:bg-surface-900 text-xs font-mono"
                  />
                  <button
                    type="button"
                    onClick={() => setShowPassword(!showPassword)}
                    className="absolute right-3 top-1/2 -translate-y-1/2 text-slate-400 hover:text-slate-600"
                  >
                    {showPassword ? <EyeOff className="w-3.5 h-3.5" /> : <Eye className="w-3.5 h-3.5" />}
                  </button>
                </div>
              </div>

              <div className="pt-2">
                <button
                  type="submit"
                  disabled={submitting === 'reset-pwd'}
                  className="px-4 py-2 rounded-xl bg-purple-600 hover:bg-purple-700 text-white font-semibold text-xs shadow-md shadow-purple-500/20 transition-all flex items-center gap-1.5"
                >
                  <Key className="w-3.5 h-3.5" />
                  {submitting === 'reset-pwd' ? 'Updating...' : 'Reset Password'}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* 7. Sub-Tab Content: STAGING & CLONING */}
      {activeSubTab === 'staging' && (
        <div className="space-y-5">
          {/* Header Banner */}
          <div className="p-4 rounded-xl bg-purple-50/50 dark:bg-purple-950/20 border border-purple-200 dark:border-purple-900/30 flex items-center justify-between">
            <div className="flex items-center gap-3">
              <div className="p-2.5 rounded-xl bg-purple-600 text-white shadow-sm">
                <RefreshCw className="w-5 h-5" />
              </div>
              <div>
                <h3 className="text-sm font-bold text-slate-900 dark:text-white flex items-center gap-2">
                  1-Click Staging & Safe Cutover Engine
                  <span className="px-2 py-0.5 rounded text-[10px] font-bold bg-purple-500/15 text-purple-600 dark:text-purple-400 border border-purple-500/30">
                    Serialization-Aware
                  </span>
                </h3>
                <p className="text-xs text-slate-500 mt-0.5">
                  Spin up an isolated staging environment with cloned database, serialized URL replacement, and HTTP Basic Auth protection.
                </p>
              </div>
            </div>

            {!staging && (
              <button
                onClick={() => setShowStagingModal(true)}
                className="px-4 py-2 rounded-xl bg-purple-600 hover:bg-purple-700 text-white font-bold text-xs shadow-md shadow-purple-500/20 transition cursor-pointer flex items-center gap-1.5"
              >
                <Plus className="w-4 h-4" />
                <span>Create Staging</span>
              </button>
            )}
          </div>

          {/* Safety Backup Alert Banner after Push to Live */}
          {pushSuccessBackup && (
            <div className="p-4 rounded-xl bg-emerald-50 dark:bg-emerald-950/20 border border-emerald-300 dark:border-emerald-800 flex items-start gap-3 text-xs text-emerald-800 dark:text-emerald-200">
              <CheckCircle2 className="w-5 h-5 text-emerald-500 shrink-0 mt-0.5" />
              <div>
                <p className="font-bold">Production Updated Successfully!</p>
                <p className="text-[11px] mt-0.5">
                  Automated pre-flight production safety backup created at:{' '}
                  <code className="bg-emerald-100 dark:bg-emerald-900/50 px-1 py-0.5 rounded font-mono font-semibold">
                    {pushSuccessBackup}
                  </code>
                </p>
              </div>
            </div>
          )}

          {stagingLoading ? (
            <div className="flex flex-col items-center justify-center p-12 text-slate-400">
              <RefreshCw className="w-8 h-8 animate-spin mb-3 text-purple-500" />
              <p className="text-xs font-mono">Loading staging environment details...</p>
            </div>
          ) : staging ? (
            <div className="p-5 rounded-2xl border border-slate-200 dark:border-surface-800 bg-white dark:bg-surface-900 space-y-5">
              {/* Staging Info Grid */}
              <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
                <div className="p-3.5 rounded-xl bg-slate-50 dark:bg-[#131B2E] border border-slate-200 dark:border-slate-800">
                  <span className="text-[10px] uppercase font-bold text-slate-400 block mb-1">Staging Domain</span>
                  <div className="flex items-center justify-between">
                    <a
                      href={`https://${staging.staging_domain}`}
                      target="_blank"
                      rel="noreferrer"
                      className="font-bold text-purple-600 dark:text-purple-400 hover:underline flex items-center gap-1 truncate text-xs"
                    >
                      <span>{staging.staging_domain}</span>
                      <ExternalLink className="w-3 h-3 shrink-0" />
                    </a>
                  </div>
                </div>

                <div className="p-3.5 rounded-xl bg-slate-50 dark:bg-[#131B2E] border border-slate-200 dark:border-slate-800">
                  <span className="text-[10px] uppercase font-bold text-slate-400 block mb-1">Staging Document Root</span>
                  <span className="font-mono text-xs text-slate-700 dark:text-slate-300 truncate block">
                    {staging.staging_doc_root}
                  </span>
                </div>

                <div className="p-3.5 rounded-xl bg-slate-50 dark:bg-[#131B2E] border border-slate-200 dark:border-slate-800">
                  <span className="text-[10px] uppercase font-bold text-slate-400 block mb-1">HTTP Basic Auth Access</span>
                  <div className="flex items-center gap-2 text-xs font-mono text-slate-700 dark:text-slate-300">
                    <Lock className="w-3.5 h-3.5 text-amber-500 shrink-0" />
                    <span>User: <strong>{staging.auth_user}</strong></span>
                    <span className="text-slate-300 dark:text-slate-700">•</span>
                    <span>Pass: {staging.auth_password || '••••••••'}</span>
                  </div>
                </div>
              </div>

              {/* Safeguards Summary */}
              <div className="p-4 rounded-xl bg-slate-50 dark:bg-[#0F172A] border border-slate-200 dark:border-slate-800 space-y-2">
                <span className="text-xs font-bold text-slate-900 dark:text-white block">
                  Active Staging Safeguards
                </span>
                <div className="grid grid-cols-1 sm:grid-cols-3 gap-3 text-xs">
                  <div className="flex items-center gap-2 text-emerald-600 dark:text-emerald-400 font-semibold">
                    <CheckCircle2 className="w-4 h-4" />
                    <span>Search Indexing Blocked (noindex)</span>
                  </div>
                  <div className="flex items-center gap-2 text-emerald-600 dark:text-emerald-400 font-semibold">
                    <CheckCircle2 className="w-4 h-4" />
                    <span>Outbound Emails Disabled</span>
                  </div>
                  <div className="flex items-center gap-2 text-emerald-600 dark:text-emerald-400 font-semibold">
                    <CheckCircle2 className="w-4 h-4" />
                    <span>Password Gate (.htpasswd)</span>
                  </div>
                </div>
              </div>

              {/* Action Buttons */}
              <div className="pt-2 flex items-center justify-between border-t border-slate-200 dark:border-surface-800">
                <div className="text-xs text-slate-400">
                  {staging.last_sync_at ? (
                    <span>Last Pushed to Live: {staging.last_sync_at.replace('T', ' ').slice(0, 19)}</span>
                  ) : (
                    <span>Not pushed to live yet.</span>
                  )}
                </div>

                <div className="flex items-center gap-3">
                  <button
                    onClick={handleDeleteStaging}
                    className="px-3.5 py-2 rounded-xl text-rose-600 dark:text-rose-400 hover:bg-rose-50 dark:hover:bg-rose-950/20 font-bold text-xs transition cursor-pointer"
                  >
                    Delete Staging
                  </button>

                  <button
                    onClick={handlePushToLive}
                    disabled={pushingLive}
                    className="px-5 py-2 rounded-xl bg-purple-600 hover:bg-purple-700 text-white font-bold text-xs shadow-md shadow-purple-500/20 transition cursor-pointer disabled:opacity-50 flex items-center gap-2"
                  >
                    <Zap className={`w-3.5 h-3.5 ${pushingLive ? 'animate-spin' : ''}`} />
                    <span>{pushingLive ? 'Promoting to Live...' : '1-Click Push to Live'}</span>
                  </button>
                </div>
              </div>
            </div>
          ) : (
            <div className="p-10 rounded-2xl border border-dashed border-slate-300 dark:border-surface-700 text-center space-y-3">
              <RefreshCw className="w-10 h-10 mx-auto text-purple-400 opacity-40" />
              <h4 className="font-bold text-slate-800 dark:text-slate-200 text-sm">
                No Staging Environment Created
              </h4>
              <p className="text-xs text-slate-500 max-w-md mx-auto">
                Create a password-protected staging copy with automated PHP serialized string replacement in database options and posts.
              </p>
              <button
                onClick={() => setShowStagingModal(true)}
                className="px-4 py-2 rounded-xl bg-purple-600 hover:bg-purple-700 text-white font-bold text-xs shadow-md transition cursor-pointer inline-flex items-center gap-1.5"
              >
                <Plus className="w-4 h-4" />
                <span>Create WordPress Staging Environment</span>
              </button>
            </div>
          )}
        </div>
      )}

      {/* Create Staging Modal */}
      {showStagingModal && (
        <div className="fixed inset-0 z-60 bg-black/70 backdrop-blur-xs flex items-center justify-center p-4">
          <div className="w-full max-w-lg bg-white dark:bg-[#0F172A] border border-slate-200 dark:border-slate-800 rounded-2xl shadow-2xl overflow-hidden animate-fadeIn">
            <div className="flex items-center justify-between px-5 py-4 border-b border-slate-200 dark:border-slate-800 bg-slate-50 dark:bg-[#0B1120]">
              <h3 className="font-bold text-sm text-slate-900 dark:text-white flex items-center gap-2">
                <RefreshCw className="w-4 h-4 text-purple-500" />
                Create WordPress Staging Clone
              </h3>
              <button
                onClick={() => setShowStagingModal(false)}
                className="text-slate-400 hover:text-slate-600 dark:hover:text-white cursor-pointer"
              >
                ✕
              </button>
            </div>

            <form onSubmit={handleCreateStaging} className="p-5 space-y-4 text-xs">
              <div>
                <label className="block font-bold text-slate-700 dark:text-slate-300 mb-1">
                  Staging Subdomain
                </label>
                <div className="flex items-center gap-2 font-mono">
                  <input
                    type="text"
                    value={stagingSubdomain}
                    onChange={(e) => setStagingSubdomain(e.target.value)}
                    placeholder="staging"
                    required
                    className="flex-1 p-2 rounded-lg border border-slate-300 dark:border-slate-700 bg-slate-50 dark:bg-[#131B2E] text-slate-900 dark:text-white font-bold"
                  />
                  <span className="text-slate-500 font-semibold">.{website.primary_domain}</span>
                </div>
              </div>

              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label className="block font-bold text-slate-700 dark:text-slate-300 mb-1">
                    HTTP Basic Auth User
                  </label>
                  <input
                    type="text"
                    value={stagingAuthUser}
                    onChange={(e) => setStagingAuthUser(e.target.value)}
                    required
                    className="w-full p-2 rounded-lg border border-slate-300 dark:border-slate-700 bg-slate-50 dark:bg-[#131B2E] text-slate-900 dark:text-white font-mono"
                  />
                </div>
                <div>
                  <label className="block font-bold text-slate-700 dark:text-slate-300 mb-1">
                    HTTP Basic Auth Password
                  </label>
                  <input
                    type="text"
                    value={stagingAuthPass}
                    onChange={(e) => setStagingAuthPass(e.target.value)}
                    required
                    className="w-full p-2 rounded-lg border border-slate-300 dark:border-slate-700 bg-slate-50 dark:bg-[#131B2E] text-slate-900 dark:text-white font-mono"
                  />
                </div>
              </div>

              <div className="p-3 rounded-xl bg-purple-50 dark:bg-purple-950/20 border border-purple-200 dark:border-purple-900/30 text-[11px] text-purple-800 dark:text-purple-300 space-y-1">
                <p className="font-bold">What happens next?</p>
                <ul className="list-disc pl-4 space-y-0.5 text-slate-600 dark:text-slate-400">
                  <li>Copies all live WordPress core files, themes, and uploads.</li>
                  <li>Clones live database into isolated staging database.</li>
                  <li>Performs PHP serialization-aware search-and-replace for domain URLs.</li>
                  <li>Disables outbound emails and blocks web crawlers.</li>
                </ul>
              </div>

              <div className="pt-2 flex justify-end gap-3">
                <button
                  type="button"
                  onClick={() => setShowStagingModal(false)}
                  className="px-4 py-2 rounded-xl text-slate-600 dark:text-slate-400 font-semibold"
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  disabled={creatingStaging}
                  className="px-5 py-2 rounded-xl bg-purple-600 hover:bg-purple-700 text-white font-bold transition shadow-md cursor-pointer disabled:opacity-50 flex items-center gap-1.5"
                >
                  {creatingStaging ? <RefreshCw className="w-3.5 h-3.5 animate-spin" /> : <Plus className="w-3.5 h-3.5" />}
                  <span>{creatingStaging ? 'Cloning & Provisioning...' : 'Create Staging Clone'}</span>
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
}
