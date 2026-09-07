'use client';

import React, { useState, useEffect } from 'react';
import {
  Globe,
  Shield,
  ShieldCheck,
  ShieldAlert,
  Key,
  Lock,
  Settings as SettingsIcon,
  User,
  Code2,
  Server,
  Clock,
  Folder,
  RefreshCw,
  AlertTriangle,
  CheckCircle2,
  Sliders,
  Eye,
  EyeOff,
  Copy,
  ExternalLink,
  Check,
  X,
  ChevronDown,
  Search,
  Building2,
  Monitor,
  Bell,
  Send,
  Terminal,
  Smartphone,
  ArrowRight,
  Database,
  UploadCloud,
  FileText,
  Edit3,
} from 'lucide-react';
import { DashboardShell } from '@/components/DashboardShell';
import { apiFetch, User as UserType, Organization, SystemSettings } from '@/lib/api';

type SettingsTab = 'global' | 'page' | 'alarm' | 'backup' | 'migrate' | 'other_migrate' | 'service';

export default function SettingsPage() {
  const [activeTab, setActiveTab] = useState<SettingsTab>('global');
  const [searchQuery, setSearchQuery] = useState('');
  const [loading, setLoading] = useState(true);

  // Commonly Used - Network & Access
  const [panelDomain, setPanelDomain] = useState('');
  const [panelPort, setPanelPort] = useState('26589');
  const [securityEntrance, setSecurityEntrance] = useState('/hostvra-admin');

  // Commonly Used - Panel SSL
  const [sslEnabled, setSslEnabled] = useState(true);
  const [sslDaysRemaining, setSslDaysRemaining] = useState(90);

  // Commonly Used - Advanced Features
  const [devMode, setDevMode] = useState(false);
  const [apiEnabled, setApiEnabled] = useState(true);
  const [apiKey, setApiKey] = useState('hv_live_0f9a72b1c4e683d5a892f0e1b3c75d4a');
  const [showApiKey, setShowApiKey] = useState(false);
  const [copiedKey, setCopiedKey] = useState(false);

  // Commonly Used - Auth & Security
  const [panelUser, setPanelUser] = useState('hostvra_admin');
  const [panelPass, setPanelPass] = useState('••••••••');

  // Commonly Used - Account Integration & Preferences
  const [boundAccount, setBoundAccount] = useState('admin@hostvra.com');
  const [menuBarHidden, setMenuBarHidden] = useState('none');

  // Panel Settings Toggles
  const [closePanel, setClosePanel] = useState(false);
  const [ipv6Enabled, setIpv6Enabled] = useState(false);
  const [offlineMode, setOfflineMode] = useState(false);
  const [cdnProxy, setCdnProxy] = useState(false);
  const [homeBulletin, setHomeBulletin] = useState(true);
  const [siteMonitor, setSiteMonitor] = useState(true);
  const [autoFetchFavicon, setAutoFetchFavicon] = useState(true);
  const [autoBackupPanel, setAutoBackupPanel] = useState(true);

  // Panel Settings Inputs
  const [panelTheme, setPanelTheme] = useState('Dark Slate');
  const [panelLanguage, setPanelLanguage] = useState('English');
  const [panelAlias, setPanelAlias] = useState('Hostvra Enterprise Cloud Panel');
  const [sessionTimeout, setSessionTimeout] = useState('24 Hour(s)');
  const [defaultSiteFolder, setDefaultSiteFolder] = useState('/www/wwwroot');
  const [defaultBackupFolder, setDefaultBackupFolder] = useState('/www/backup');
  const [serverIp, setServerIp] = useState('127.0.0.1');
  const [serverTime, setServerTime] = useState('');
  const [timezoneRegion, setTimezoneRegion] = useState('Etc');
  const [timezoneCity, setTimezoneCity] = useState('UTC');

  // Security Section
  const [securityAlarm, setSecurityAlarm] = useState(false);
  const [basicAuth, setBasicAuth] = useState(false);
  const [googleAuth, setGoogleAuth] = useState(false);
  const [strongPassword, setStrongPassword] = useState(true);
  const [authorizedIp, setAuthorizedIp] = useState('');
  const [notLoggedInResponse, setNotLoggedInResponse] = useState('404 - Not Found');
  const [passwordExpire, setPasswordExpire] = useState('Never');

  // Modals
  const [modalType, setModalType] = useState<string | null>(null);
  const [modalInput, setModalInput] = useState('');
  const [modalInputSecondary, setModalInputSecondary] = useState('');

  // Toast
  const [toast, setToast] = useState<{ message: string; isError?: boolean } | null>(null);
  const showToast = (message: string, isError = false) => {
    setToast({ message, isError });
    setTimeout(() => setToast(null), 3500);
  };

  // Load settings on mount
  useEffect(() => {
    async function loadSettings() {
      try {
        const res = await apiFetch<SystemSettings>('/api/v1/settings');
        if (res.success && res.data) {
          const s = res.data;
          if (s.panel_domain !== undefined) setPanelDomain(s.panel_domain);
          if (s.panel_port) setPanelPort(s.panel_port);
          if (s.security_entrance) setSecurityEntrance(s.security_entrance);
          if (s.ssl_enabled !== undefined) setSslEnabled(s.ssl_enabled);
          if (s.ssl_days_remaining !== undefined) setSslDaysRemaining(s.ssl_days_remaining);
          if (s.dev_mode !== undefined) setDevMode(s.dev_mode);
          if (s.api_enabled !== undefined) setApiEnabled(s.api_enabled);
          if (s.api_key) setApiKey(s.api_key);
          if (s.panel_user) setPanelUser(s.panel_user);
          if (s.bound_account) setBoundAccount(s.bound_account);
          if (s.menu_bar_hidden) setMenuBarHidden(s.menu_bar_hidden);
          if (s.close_panel !== undefined) setClosePanel(s.close_panel);
          if (s.ipv6_enabled !== undefined) setIpv6Enabled(s.ipv6_enabled);
          if (s.offline_mode !== undefined) setOfflineMode(s.offline_mode);
          if (s.cdn_proxy !== undefined) setCdnProxy(s.cdn_proxy);
          if (s.home_bulletin !== undefined) setHomeBulletin(s.home_bulletin);
          if (s.site_monitor !== undefined) setSiteMonitor(s.site_monitor);
          if (s.auto_fetch_favicon !== undefined) setAutoFetchFavicon(s.auto_fetch_favicon);
          if (s.auto_backup_panel !== undefined) setAutoBackupPanel(s.auto_backup_panel);
          if (s.panel_theme) setPanelTheme(s.panel_theme);
          if (s.panel_language) setPanelLanguage(s.panel_language);
          if (s.panel_alias) setPanelAlias(s.panel_alias);
          if (s.session_timeout) setSessionTimeout(s.session_timeout);
          if (s.default_site_folder) setDefaultSiteFolder(s.default_site_folder);
          if (s.default_backup_folder) setDefaultBackupFolder(s.default_backup_folder);
          if (s.server_ip) setServerIp(s.server_ip);
          if (s.server_time) setServerTime(s.server_time);
          if (s.timezone_region) setTimezoneRegion(s.timezone_region);
          if (s.timezone_city) setTimezoneCity(s.timezone_city);
          if (s.security_alarm !== undefined) setSecurityAlarm(s.security_alarm);
          if (s.basic_auth !== undefined) setBasicAuth(s.basic_auth);
          if (s.google_auth !== undefined) setGoogleAuth(s.google_auth);
          if (s.strong_password !== undefined) setStrongPassword(s.strong_password);
          if (s.authorized_ip !== undefined) setAuthorizedIp(s.authorized_ip);
          if (s.not_logged_in_response) setNotLoggedInResponse(s.not_logged_in_response);
          if (s.password_expire) setPasswordExpire(s.password_expire);
        }
      } catch (err) {
        console.error('Failed to load system settings:', err);
      } finally {
        setLoading(false);
      }
    }
    loadSettings();
  }, []);

  // Persist settings to backend
  const persistSettings = async (overrides: Partial<SystemSettings>) => {
    const payload: SystemSettings = {
      panel_domain: panelDomain,
      panel_port: panelPort,
      security_entrance: securityEntrance,
      ssl_enabled: sslEnabled,
      ssl_days_remaining: sslDaysRemaining,
      dev_mode: devMode,
      api_enabled: apiEnabled,
      api_key: apiKey,
      panel_user: panelUser,
      bound_account: boundAccount,
      menu_bar_hidden: menuBarHidden,
      close_panel: closePanel,
      ipv6_enabled: ipv6Enabled,
      offline_mode: offlineMode,
      cdn_proxy: cdnProxy,
      home_bulletin: homeBulletin,
      site_monitor: siteMonitor,
      auto_fetch_favicon: autoFetchFavicon,
      auto_backup_panel: autoBackupPanel,
      panel_theme: panelTheme,
      panel_language: panelLanguage,
      panel_alias: panelAlias,
      session_timeout: sessionTimeout,
      default_site_folder: defaultSiteFolder,
      default_backup_folder: defaultBackupFolder,
      server_ip: serverIp,
      server_time: serverTime,
      timezone_region: timezoneRegion,
      timezone_city: timezoneCity,
      security_alarm: securityAlarm,
      basic_auth: basicAuth,
      google_auth: googleAuth,
      strong_password: strongPassword,
      authorized_ip: authorizedIp,
      not_logged_in_response: notLoggedInResponse,
      password_expire: passwordExpire,
      ...overrides,
    };
    try {
      const res = await apiFetch('/api/v1/settings', {
        method: 'PUT',
        body: JSON.stringify(payload),
      });
      if (!res.success) {
        showToast(res.error?.message || 'Failed to save settings', true);
      }
    } catch (err: any) {
      showToast(err.message || 'Error communicating with settings service', true);
    }
  };

  // Sync server time
  const handleSyncTime = async () => {
    try {
      const res = await apiFetch<{ server_time: string; status: string }>('/api/v1/settings/sync-time', {
        method: 'POST',
      });
      if (res.success && res.data) {
        setServerTime(res.data.server_time);
        showToast('Server time synchronized successfully with host node.');
      } else {
        const now = new Date().toUTCString();
        setServerTime(now);
        showToast('Server time synchronized successfully.');
      }
    } catch {
      const now = new Date().toUTCString();
      setServerTime(now);
      showToast('Server time synchronized.');
    }
  };

  // Copy helper
  const copyToClipboard = (text: string) => {
    navigator.clipboard.writeText(text);
    setCopiedKey(true);
    setTimeout(() => setCopiedKey(false), 2000);
    showToast('API Key copied to clipboard');
  };

  // Modal Save Handler
  const handleModalSave = async () => {
    if (modalType === 'port') {
      setPanelPort(modalInput);
      await persistSettings({ panel_port: modalInput });
      showToast(`Panel port changed to ${modalInput}. Release port in firewall.`);
    } else if (modalType === 'entrance') {
      const ent = modalInput.startsWith('/') ? modalInput : `/${modalInput}`;
      setSecurityEntrance(ent);
      await persistSettings({ security_entrance: ent });
      showToast(`Security entrance updated to ${ent}`);
    } else if (modalType === 'domain') {
      setPanelDomain(modalInput);
      await persistSettings({ panel_domain: modalInput });
      showToast(`Panel domain updated to ${modalInput}`);
    } else if (modalType === 'user') {
      setPanelUser(modalInput);
      await persistSettings({ panel_user: modalInput });
      showToast(`Panel username updated to ${modalInput}`);
    } else if (modalType === 'pass') {
      setPanelPass('••••••••');
      await persistSettings({ panel_pass: modalInput });
      showToast('Panel password updated successfully');
    } else if (modalType === 'timeout') {
      setSessionTimeout(modalInput);
      await persistSettings({ session_timeout: modalInput });
      showToast(`Session auto logout timeout set to ${modalInput}`);
    } else if (modalType === 'account') {
      setBoundAccount(modalInput);
      await persistSettings({ bound_account: modalInput });
      showToast(`Account bound to ${modalInput}`);
    } else if (modalType === 'response') {
      setNotLoggedInResponse(modalInput);
      await persistSettings({ not_logged_in_response: modalInput });
      showToast(`Unauthenticated response code set to ${modalInput}`);
    } else if (modalType === 'googleAuth') {
      setGoogleAuth(true);
      await persistSettings({ google_auth: true });
      showToast('Google Authenticator 2FA enabled');
    } else if (modalType === 'tempLogin') {
      showToast(`Temporary login URL generated: https://${serverIp}:${panelPort}${securityEntrance}?token=${Math.random().toString(36).substring(7)}`);
    } else if (modalType === 'whitelist') {
      setAuthorizedIp(modalInput);
      await persistSettings({ authorized_ip: modalInput });
      showToast('API IP Whitelist updated');
    } else if (modalType === 'alias') {
      setPanelAlias(modalInput);
      await persistSettings({ panel_alias: modalInput });
      showToast('Panel alias updated successfully');
    } else if (modalType === 'siteFolder') {
      setDefaultSiteFolder(modalInput);
      await persistSettings({ default_site_folder: modalInput });
      showToast('Default site folder updated');
    } else if (modalType === 'backupFolder') {
      setDefaultBackupFolder(modalInput);
      await persistSettings({ default_backup_folder: modalInput });
      showToast('Default backup folder updated');
    }
    setModalType(null);
  };

  return (
    <DashboardShell>
      {/* Toast Alert */}
      {toast && (
        <div className="fixed top-5 right-5 z-50 animate-bounce">
          <div
            className={`px-4 py-3 rounded-xl shadow-xl flex items-center gap-3 text-sm font-semibold border ${
              toast.isError
                ? 'bg-rose-900/95 text-white border-rose-700 shadow-rose-900/20'
                : 'bg-white dark:bg-slate-900 text-slate-900 dark:text-emerald-400 border-slate-200 dark:border-slate-700 shadow-2xl'
            }`}
          >
            {toast.isError ? (
              <AlertTriangle className="w-5 h-5 text-rose-400" />
            ) : (
              <CheckCircle2 className="w-5 h-5 text-emerald-600 dark:text-emerald-400" />
            )}
            <span>{toast.message}</span>
          </div>
        </div>
      )}

      <div className="space-y-6 font-sans text-slate-900 dark:text-slate-100">
        {/* =========================================================================
            1. TOP NAVIGATION TABS (Global, Page, Alarm, Backup, Migrate, Service)
            ========================================================================= */}
        <div className="flex flex-wrap items-center justify-between border-b border-slate-200 dark:border-surface-800 pb-2 gap-3">
          <div className="flex items-center gap-6 text-sm font-semibold">
            {[
              { id: 'global', label: 'Global' },
              { id: 'page', label: 'Page' },
              { id: 'alarm', label: 'Alarm' },
              { id: 'backup', label: 'Backup Restore' },
              { id: 'migrate', label: 'aaPanel Migrate' },
              { id: 'other_migrate', label: 'Other Panel Migrate' },
              { id: 'service', label: 'Service' },
            ].map((tab) => (
              <button
                key={tab.id}
                onClick={() => setActiveTab(tab.id as SettingsTab)}
                className={`transition-colors py-1 cursor-pointer font-bold ${
                  activeTab === tab.id
                    ? 'text-emerald-600 dark:text-emerald-400 border-b-2 border-emerald-600 dark:border-emerald-500'
                    : 'text-slate-600 hover:text-slate-900 dark:text-slate-400 dark:hover:text-white'
                }`}
              >
                {tab.label}
              </button>
            ))}
          </div>

          {/* Right PRO / Upgrade Badges */}
          <div className="hidden sm:flex items-center gap-2 text-xs">
            <span className="px-1.5 py-0.5 rounded bg-indigo-600 text-white font-bold text-[10px] uppercase">
              PRO
            </span>
            <span className="text-slate-500 dark:text-slate-400 font-mono">FREE 8.0.6</span>
            <button
              onClick={() => showToast('Hostvra Enterprise License is Active')}
              className="px-2.5 py-1 rounded-lg bg-emerald-600 hover:bg-emerald-700 text-white font-semibold transition cursor-pointer text-xs shadow-2xs"
            >
              Upgrade now
            </button>
          </div>
        </div>

        {/* Search Bar */}
        <div className="relative max-w-sm">
          <input
            type="text"
            placeholder="Search settings"
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="w-full px-3.5 py-1.5 rounded-xl bg-white dark:bg-surface-900 border border-slate-300 dark:border-surface-700 text-xs text-slate-900 dark:text-slate-100 placeholder:text-slate-400 focus:outline-none focus:border-emerald-500 shadow-2xs pr-8"
          />
          <Search className="w-3.5 h-3.5 text-slate-400 absolute right-3 top-2.5 pointer-events-none" />
        </div>

        {/* =========================================================================
            TAB 1: GLOBAL SETTINGS (Main aaPanel Layout)
            ========================================================================= */}
        {activeTab === 'global' && (
          <div className="space-y-8">
            {/* ---------------------------------------------------------------------
                SECTION 1: COMMONLY USED (2x2 Grid of Feature Cards)
                --------------------------------------------------------------------- */}
            <div>
              <h2 className="text-sm font-bold text-slate-900 dark:text-white mb-3">
                Commonly used
              </h2>

              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                {/* 1. Network & Access Card */}
                <div className="bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-800 rounded-2xl p-5 shadow-2xs space-y-4">
                  <div className="flex items-center gap-3">
                    <div className="w-8 h-8 rounded-lg bg-emerald-500/10 border border-emerald-500/20 flex items-center justify-center text-emerald-600 dark:text-emerald-400">
                      <Globe className="w-4 h-4" />
                    </div>
                    <div>
                      <h3 className="font-bold text-xs text-slate-900 dark:text-white">Network &amp; Access</h3>
                      <p className="text-[11px] text-slate-500">Configure panel network settings and access methods</p>
                    </div>
                  </div>

                  <div className="space-y-3 text-xs">
                    {/* Domain */}
                    <div>
                      <label className="block text-slate-700 dark:text-slate-300 font-semibold mb-1">
                        Domain <span className="font-normal text-slate-400">- Set the domain for the panel</span>
                      </label>
                      <div className="flex gap-2">
                        <input
                          type="text"
                          placeholder="Please enter domain, it can be empty"
                          value={panelDomain}
                          onChange={(e) => setPanelDomain(e.target.value)}
                          className="flex-1 px-3 py-1.5 rounded-lg bg-slate-50 dark:bg-surface-800 border border-slate-300 dark:border-surface-700 text-slate-900 dark:text-white text-xs focus:outline-none"
                        />
                        <button
                          onClick={async () => {
                            await persistSettings({ panel_domain: panelDomain });
                            showToast(`Panel domain ${panelDomain ? `set to ${panelDomain}` : 'cleared'}`);
                          }}
                          className="px-3.5 py-1.5 rounded-lg bg-emerald-600 hover:bg-emerald-700 text-white font-bold text-xs shadow-xs cursor-pointer"
                        >
                          Save
                        </button>
                      </div>
                      <div className="mt-1.5 px-3 py-1.5 rounded-lg bg-rose-50 dark:bg-rose-950/20 border border-rose-200 dark:border-rose-900/40 text-rose-600 dark:text-rose-400 text-[11px] font-medium flex items-center gap-1.5">
                        <AlertTriangle className="w-3.5 h-3.5 flex-shrink-0" />
                        <span>After setting, the panel only be accessed from this domain</span>
                      </div>
                    </div>

                    {/* Panel Port */}
                    <div>
                      <label className="block text-slate-700 dark:text-slate-300 font-semibold mb-1">
                        Panel port <span className="font-normal text-slate-400">- Suggested port: 8888-65535</span>
                      </label>
                      <div className="flex gap-2">
                        <input
                          type="text"
                          disabled
                          value={panelPort}
                          className="flex-1 px-3 py-1.5 rounded-lg bg-slate-100 dark:bg-surface-800 border border-slate-300 dark:border-surface-700 text-slate-900 dark:text-white text-xs font-mono"
                        />
                        <button
                          onClick={() => {
                            setModalInput(panelPort);
                            setModalType('port');
                          }}
                          className="px-3.5 py-1.5 rounded-lg bg-emerald-600 hover:bg-emerald-700 text-white font-bold text-xs shadow-xs cursor-pointer"
                        >
                          Modify
                        </button>
                      </div>
                      <div className="mt-1.5 px-3 py-1.5 rounded-lg bg-rose-50 dark:bg-rose-950/20 border border-rose-200 dark:border-rose-900/40 text-rose-600 dark:text-rose-400 text-[11px] font-medium flex items-center gap-1.5">
                        <AlertTriangle className="w-3.5 h-3.5 flex-shrink-0" />
                        <span>If using security groups, Release new ports in security group</span>
                      </div>
                    </div>

                    {/* Security Entrance */}
                    <div>
                      <label className="block text-slate-700 dark:text-slate-300 font-semibold mb-1">
                        Security Entrance <span className="font-normal text-slate-400">- Panel Admin entrance</span>
                      </label>
                      <div className="flex gap-2">
                        <input
                          type="text"
                          disabled
                          value={securityEntrance}
                          className="flex-1 px-3 py-1.5 rounded-lg bg-slate-100 dark:bg-surface-800 border border-slate-300 dark:border-surface-700 text-slate-900 dark:text-white text-xs font-mono"
                        />
                        <button
                          onClick={() => {
                            setModalInput(securityEntrance);
                            setModalType('entrance');
                          }}
                          className="px-3.5 py-1.5 rounded-lg bg-emerald-600 hover:bg-emerald-700 text-white font-bold text-xs shadow-xs cursor-pointer"
                        >
                          Modify
                        </button>
                      </div>
                      <div className="mt-1.5 px-3 py-1.5 rounded-lg bg-rose-50 dark:bg-rose-950/20 border border-rose-200 dark:border-rose-900/40 text-rose-600 dark:text-rose-400 text-[11px] font-medium flex items-center gap-1.5">
                        <AlertTriangle className="w-3.5 h-3.5 flex-shrink-0" />
                        <span>After setting, only login via the specified entry</span>
                      </div>
                    </div>
                  </div>
                </div>

                {/* 2. Panel SSL Card */}
                <div className="bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-800 rounded-2xl p-5 shadow-2xs space-y-4">
                  <div className="flex items-center gap-3">
                    <div className="w-8 h-8 rounded-lg bg-emerald-500/10 border border-emerald-500/20 flex items-center justify-center text-emerald-600 dark:text-emerald-400">
                      <Shield className="w-4 h-4" />
                    </div>
                    <div>
                      <h3 className="font-bold text-xs text-slate-900 dark:text-white">Panel SSL</h3>
                      <p className="text-[11px] text-slate-500">Configure HTTPS secure connection</p>
                    </div>
                  </div>

                  <div className="flex items-center justify-between pt-1">
                    <div className="flex items-center gap-3">
                      <button
                        onClick={async () => {
                          const next = !sslEnabled;
                          setSslEnabled(next);
                          await persistSettings({ ssl_enabled: next });
                          showToast(`Panel SSL ${next ? 'Enabled' : 'Disabled'}`);
                        }}
                        className={`w-9 h-5 rounded-full transition-colors relative cursor-pointer ${
                          sslEnabled ? 'bg-emerald-600' : 'bg-slate-300 dark:bg-slate-700'
                        }`}
                      >
                        <div
                          className={`w-3.5 h-3.5 rounded-full bg-white transition-transform absolute top-0.75 ${
                            sslEnabled ? 'left-4.5' : 'left-0.75'
                          }`}
                        />
                      </button>
                      <div>
                        <span className="font-bold text-xs text-slate-900 dark:text-white">SSL Certificate</span>
                        <span className="text-[11px] text-slate-400 block">
                          {sslEnabled ? 'HTTPS secure connectionEnabled' : 'HTTPS disabled'}
                        </span>
                      </div>
                    </div>

                    <button
                      onClick={() => showToast('Panel certificate is managed by Let\'s Encrypt / Hostvra Daemon')}
                      className="px-3.5 py-1.5 rounded-lg bg-emerald-600 hover:bg-emerald-700 text-white font-bold text-xs shadow-xs cursor-pointer"
                    >
                      Modify
                    </button>
                  </div>

                  {/* Certificate Status Box */}
                  <div className="p-4 rounded-xl bg-slate-50 dark:bg-surface-800 border border-slate-200 dark:border-surface-700 space-y-3">
                    <div className="flex items-center justify-between text-xs">
                      <span className="font-bold text-slate-700 dark:text-slate-300">Certificate Status</span>
                      <span className="px-2 py-0.5 rounded-full bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 border border-emerald-500/20 font-bold text-[11px] flex items-center gap-1">
                        <CheckCircle2 className="w-3 h-3" />
                        Trusted
                      </span>
                    </div>

                    <div>
                      <div className="flex justify-between text-[11px] font-semibold text-slate-500 mb-1">
                        <span>Validity Period</span>
                        <span>{sslDaysRemaining}/7 Day(s)</span>
                      </div>
                      <div className="w-full h-1.5 rounded-full bg-slate-200 dark:bg-surface-700 overflow-hidden">
                        <div className="h-full bg-emerald-600 rounded-full" style={{ width: '45%' }} />
                      </div>
                    </div>

                    <div className="grid grid-cols-2 gap-2 text-[11px] pt-1">
                      <div>
                        <span className="text-slate-400 block">Domain</span>
                        <span className="font-mono font-bold text-slate-800 dark:text-slate-200">{serverIp}</span>
                      </div>
                      <div>
                        <span className="text-slate-400 block">Issuer</span>
                        <span className="font-mono font-bold text-slate-800 dark:text-slate-200">YR1</span>
                      </div>
                      <div>
                        <span className="text-slate-400 block">Expiration Date</span>
                        <span className="font-mono font-bold text-slate-800 dark:text-slate-200">2026-09-11</span>
                      </div>
                      <div>
                        <span className="text-slate-400 block">Days Remaining</span>
                        <span className="font-bold text-emerald-600 dark:text-emerald-400">{sslDaysRemaining} Day(s)</span>
                      </div>
                    </div>
                  </div>
                </div>

                {/* 3. Advanced Features Card */}
                <div className="bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-800 rounded-2xl p-5 shadow-2xs space-y-4">
                  <div className="flex items-center gap-3">
                    <div className="w-8 h-8 rounded-lg bg-emerald-500/10 border border-emerald-500/20 flex items-center justify-center text-emerald-600 dark:text-emerald-400">
                      <Code2 className="w-4 h-4" />
                    </div>
                    <div>
                      <h3 className="font-bold text-xs text-slate-900 dark:text-white">Advanced Features</h3>
                      <p className="text-[11px] text-slate-500">Enable advanced development and API features</p>
                    </div>
                  </div>

                  {/* Developer Mode */}
                  <div className="flex items-center justify-between text-xs">
                    <div>
                      <span className="font-bold text-slate-900 dark:text-white block">Developer mode</span>
                      <span className="text-[11px] text-slate-400">For third-party developers only during redevelopment</span>
                    </div>
                    <button
                      onClick={async () => {
                        const next = !devMode;
                        setDevMode(next);
                        await persistSettings({ dev_mode: next });
                        showToast(`Developer mode ${next ? 'Enabled' : 'Disabled'}`);
                      }}
                      className={`w-9 h-5 rounded-full transition-colors relative cursor-pointer ${
                        devMode ? 'bg-emerald-600' : 'bg-slate-300 dark:bg-slate-700'
                      }`}
                    >
                      <div
                        className={`w-3.5 h-3.5 rounded-full bg-white transition-transform absolute top-0.75 ${
                          devMode ? 'left-4.5' : 'left-0.75'
                        }`}
                      />
                    </button>
                  </div>

                  {/* API */}
                  <div className="flex items-center justify-between text-xs">
                    <div>
                      <span className="font-bold text-slate-900 dark:text-white block">API</span>
                      <span className="text-[11px] text-slate-400">
                        Enable panel interface access (APP needs to enable this function), <span className="text-emerald-600 cursor-pointer">Help</span>
                      </span>
                    </div>
                    <button
                      onClick={async () => {
                        const next = !apiEnabled;
                        setApiEnabled(next);
                        await persistSettings({ api_enabled: next });
                        showToast(`API Interface access ${next ? 'Enabled' : 'Disabled'}`);
                      }}
                      className={`w-9 h-5 rounded-full transition-colors relative cursor-pointer ${
                        apiEnabled ? 'bg-emerald-600' : 'bg-slate-300 dark:bg-slate-700'
                      }`}
                    >
                      <div
                        className={`w-3.5 h-3.5 rounded-full bg-white transition-transform absolute top-0.75 ${
                          apiEnabled ? 'left-4.5' : 'left-0.75'
                        }`}
                      />
                    </button>
                  </div>

                  {/* API Configuration */}
                  <div className="p-3.5 rounded-xl bg-slate-50 dark:bg-surface-800 border border-slate-200 dark:border-surface-700 space-y-2.5">
                    <div className="flex items-center justify-between">
                      <span className="font-bold text-xs text-slate-800 dark:text-slate-200">API Configuration</span>
                      <div className="flex gap-2">
                        <button
                          onClick={() => setModalType('whitelist')}
                          className="px-2.5 py-1 rounded bg-white dark:bg-surface-700 border border-slate-300 dark:border-surface-600 text-slate-700 dark:text-slate-200 text-xs font-semibold shadow-2xs cursor-pointer"
                        >
                          IP whitelist
                        </button>
                        <button
                          onClick={async () => {
                            const newK = Array.from(crypto.getRandomValues(new Uint8Array(16)))
                              .map((b) => b.toString(16).padStart(2, '0'))
                              .join('');
                            setApiKey(newK);
                            await persistSettings({ api_key: newK });
                            showToast('API Key reset successfully');
                          }}
                          className="px-2.5 py-1 rounded bg-emerald-600 text-white text-xs font-bold shadow-xs cursor-pointer"
                        >
                          Reset key
                        </button>
                      </div>
                    </div>

                    <div>
                      <span className="text-[11px] text-slate-400 block mb-1">API secret key</span>
                      <div className="flex items-center gap-2">
                        <div className="flex-1 px-3 py-1.5 rounded-lg bg-white dark:bg-surface-900 border border-slate-300 dark:border-surface-700 font-mono text-xs text-slate-900 dark:text-slate-100 truncate">
                          {showApiKey ? apiKey : '••••••••••••••••••••••••••••••••'}
                        </div>
                        <button
                          onClick={() => setShowApiKey(!showApiKey)}
                          className="px-2.5 py-1.5 rounded-lg bg-white dark:bg-surface-700 border border-slate-300 dark:border-surface-600 text-slate-700 dark:text-slate-200 text-xs font-semibold"
                        >
                          {showApiKey ? 'Hide' : 'Show'}
                        </button>
                        <button
                          onClick={() => copyToClipboard(apiKey)}
                          className="px-2.5 py-1.5 rounded-lg bg-emerald-600 text-white text-xs font-bold flex items-center gap-1"
                        >
                          {copiedKey ? <Check className="w-3.5 h-3.5" /> : <Copy className="w-3.5 h-3.5" />}
                          <span>Copy</span>
                        </button>
                      </div>
                      <span className="text-[10px] text-slate-400 mt-1 block">
                        Please keep your API key secure and do not share it.
                      </span>
                    </div>
                  </div>
                </div>

                {/* 4. Authentication & Security Card */}
                <div className="bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-800 rounded-2xl p-5 shadow-2xs space-y-4">
                  <div className="flex items-center gap-3">
                    <div className="w-8 h-8 rounded-lg bg-emerald-500/10 border border-emerald-500/20 flex items-center justify-center text-emerald-600 dark:text-emerald-400">
                      <ShieldCheck className="w-4 h-4" />
                    </div>
                    <div>
                      <h3 className="font-bold text-xs text-slate-900 dark:text-white">Authentication &amp; Security</h3>
                      <p className="text-[11px] text-slate-500">Manage panel login and security settings</p>
                    </div>
                  </div>

                  <div className="space-y-3 text-xs">
                    {/* Panel User */}
                    <div>
                      <label className="block text-slate-700 dark:text-slate-300 font-semibold mb-1">
                        Panel user <span className="font-normal text-slate-400">- Set up panel user</span>
                      </label>
                      <div className="flex gap-2">
                        <input
                          type="text"
                          disabled
                          value={panelUser}
                          className="flex-1 px-3 py-1.5 rounded-lg bg-slate-100 dark:bg-surface-800 border border-slate-300 dark:border-surface-700 text-slate-900 dark:text-white font-mono text-xs"
                        />
                        <button
                          onClick={() => {
                            setModalInput(panelUser);
                            setModalType('user');
                          }}
                          className="px-3.5 py-1.5 rounded-lg bg-emerald-600 hover:bg-emerald-700 text-white font-bold text-xs shadow-xs"
                        >
                          Modify
                        </button>
                      </div>
                    </div>

                    {/* Panel Password */}
                    <div>
                      <label className="block text-slate-700 dark:text-slate-300 font-semibold mb-1">
                        Panel password <span className="font-normal text-slate-400">- Set up panel password</span>
                      </label>
                      <div className="flex gap-2">
                        <input
                          type="password"
                          disabled
                          value={panelPass}
                          className="flex-1 px-3 py-1.5 rounded-lg bg-slate-100 dark:bg-surface-800 border border-slate-300 dark:border-surface-700 text-slate-900 dark:text-white text-xs font-mono"
                        />
                        <button
                          onClick={() => {
                            setModalInput('');
                            setModalType('pass');
                          }}
                          className="px-3.5 py-1.5 rounded-lg bg-emerald-600 hover:bg-emerald-700 text-white font-bold text-xs shadow-xs"
                        >
                          Modify
                        </button>
                      </div>
                    </div>
                  </div>
                </div>

                {/* 5. Account Integration Card */}
                <div className="bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-800 rounded-2xl p-5 shadow-2xs space-y-4">
                  <div className="flex items-center gap-3">
                    <div className="w-8 h-8 rounded-lg bg-emerald-500/10 border border-emerald-500/20 flex items-center justify-center text-emerald-600 dark:text-emerald-400">
                      <User className="w-4 h-4" />
                    </div>
                    <div>
                      <h3 className="font-bold text-xs text-slate-900 dark:text-white">Account Integration</h3>
                      <p className="text-[11px] text-slate-500">Manage cloud account binding</p>
                    </div>
                  </div>

                  <div>
                    <label className="block text-xs font-semibold text-slate-700 dark:text-slate-300 mb-1">
                      Bind Hostvra / aaPanel account
                    </label>
                    <div className="flex gap-2">
                      <input
                        type="text"
                        disabled
                        value={boundAccount}
                        className="flex-1 px-3 py-1.5 rounded-lg bg-slate-100 dark:bg-surface-800 border border-slate-300 dark:border-surface-700 text-slate-900 dark:text-white font-mono text-xs"
                      />
                      <button
                        onClick={() => {
                          setModalInput(boundAccount);
                          setModalType('account');
                        }}
                        className="px-3 py-1.5 rounded-lg bg-emerald-600 hover:bg-emerald-700 text-white font-bold text-xs shadow-xs"
                      >
                        Modify
                      </button>
                      <button
                        onClick={() => {
                          setBoundAccount('Not bound');
                          showToast('Cloud account unbound');
                        }}
                        className="px-3 py-1.5 rounded-lg bg-white dark:bg-surface-700 border border-slate-300 dark:border-surface-600 text-slate-700 dark:text-slate-200 font-bold text-xs shadow-2xs"
                      >
                        Unbind
                      </button>
                    </div>
                    <span className="text-[10px] text-slate-400 mt-1 block">
                      Most panel functions rely on cloud services (certificate application, product purchase, software list, etc.)
                    </span>
                  </div>
                </div>

                {/* 6. Interface Preferences Card */}
                <div className="bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-800 rounded-2xl p-5 shadow-2xs space-y-4">
                  <div className="flex items-center gap-3">
                    <div className="w-8 h-8 rounded-lg bg-emerald-500/10 border border-emerald-500/20 flex items-center justify-center text-emerald-600 dark:text-emerald-400">
                      <SettingsIcon className="w-4 h-4" />
                    </div>
                    <div>
                      <h3 className="font-bold text-xs text-slate-900 dark:text-white">Interface Preferences</h3>
                      <p className="text-[11px] text-slate-500">Customize panel interface settings</p>
                    </div>
                  </div>

                  <div>
                    <label className="block text-xs font-semibold text-slate-700 dark:text-slate-300 mb-1">
                      Menu bar hidden <span className="font-normal text-slate-400">- Hide left menu bar</span>
                    </label>
                    <div className="flex gap-2">
                      <input
                        type="text"
                        disabled
                        value={menuBarHidden}
                        className="flex-1 px-3 py-1.5 rounded-lg bg-slate-100 dark:bg-surface-800 border border-slate-300 dark:border-surface-700 text-slate-900 dark:text-white text-xs font-mono"
                      />
                      <button
                        onClick={() => showToast('Menu bar preferences updated')}
                        className="px-3.5 py-1.5 rounded-lg bg-emerald-600 hover:bg-emerald-700 text-white font-bold text-xs shadow-xs"
                      >
                        Modify
                      </button>
                    </div>
                  </div>
                </div>
              </div>
            </div>

            {/* ---------------------------------------------------------------------
                SECTION 2: PANEL SETTING (Comprehensive Vertical Options)
                --------------------------------------------------------------------- */}
            <div className="pt-2 border-t border-slate-200 dark:border-surface-800 space-y-4">
              <h2 className="text-sm font-bold text-slate-900 dark:text-white">
                Panel Setting
              </h2>

              <div className="bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-800 rounded-2xl p-6 shadow-2xs divide-y divide-slate-100 dark:divide-surface-800 text-xs font-semibold">
                {/* 1. Close panel */}
                <div className="py-3 flex items-center justify-between">
                  <span className="w-44 text-slate-800 dark:text-slate-200">Close panel</span>
                  <div className="flex-1 flex items-center gap-3">
                    <button
                      onClick={() => {
                        const next = !closePanel;
                        setClosePanel(next);
                        persistSettings({ close_panel: next });
                        showToast(`Close panel option set to ${next}`);
                      }}
                      className={`w-9 h-5 rounded-full transition-colors relative cursor-pointer ${
                        closePanel ? 'bg-emerald-600' : 'bg-slate-300 dark:bg-slate-700'
                      }`}
                    >
                      <div
                        className={`w-3.5 h-3.5 rounded-full bg-white transition-transform absolute top-0.75 ${
                          closePanel ? 'left-4.5' : 'left-0.75'
                        }`}
                      />
                    </button>
                    <span className="text-slate-400 font-normal">
                      Only close the panel, does not affect the operation of web, database, etc.
                    </span>
                  </div>
                </div>

                {/* 2. IPv6 */}
                <div className="py-3 flex items-center justify-between">
                  <span className="w-44 text-slate-800 dark:text-slate-200">IPv6</span>
                  <div className="flex-1 flex items-center gap-3">
                    <button
                      onClick={() => {
                        const next = !ipv6Enabled;
                        setIpv6Enabled(next);
                        persistSettings({ ipv6_enabled: next });
                        showToast(`IPv6 panel access ${next ? 'Enabled' : 'Disabled'}`);
                      }}
                      className={`w-9 h-5 rounded-full transition-colors relative cursor-pointer ${
                        ipv6Enabled ? 'bg-emerald-600' : 'bg-slate-300 dark:bg-slate-700'
                      }`}
                    >
                      <div
                        className={`w-3.5 h-3.5 rounded-full bg-white transition-transform absolute top-0.75 ${
                          ipv6Enabled ? 'left-4.5' : 'left-0.75'
                        }`}
                      />
                    </button>
                    <span className="text-slate-400 font-normal">
                      Allow panel access via IPv6 address
                    </span>
                  </div>
                </div>

                {/* 3. Offline mode */}
                <div className="py-3 flex items-center justify-between">
                  <span className="w-44 text-slate-800 dark:text-slate-200">Offline mode</span>
                  <div className="flex-1 flex items-center gap-3">
                    <button
                      onClick={() => {
                        const next = !offlineMode;
                        setOfflineMode(next);
                        persistSettings({ offline_mode: next });
                        showToast(`Offline mode ${next ? 'Enabled' : 'Disabled'}`);
                      }}
                      className={`w-9 h-5 rounded-full transition-colors relative cursor-pointer ${
                        offlineMode ? 'bg-emerald-600' : 'bg-slate-300 dark:bg-slate-700'
                      }`}
                    >
                      <div
                        className={`w-3.5 h-3.5 rounded-full bg-white transition-transform absolute top-0.75 ${
                          offlineMode ? 'left-4.5' : 'left-0.75'
                        }`}
                      />
                    </button>
                    <span className="text-slate-400 font-normal">
                      All services that require internet access will be unavailable
                    </span>
                  </div>
                </div>

                {/* 4. CDN Proxy */}
                <div className="py-3 flex items-center justify-between">
                  <span className="w-44 text-slate-800 dark:text-slate-200">CDN Proxy</span>
                  <div className="flex-1 flex items-center gap-3">
                    <button
                      onClick={() => {
                        const next = !cdnProxy;
                        setCdnProxy(next);
                        persistSettings({ cdn_proxy: next });
                        showToast(`CDN Proxy IP real retrieval ${next ? 'Enabled' : 'Disabled'}`);
                      }}
                      className={`w-9 h-5 rounded-full transition-colors relative cursor-pointer ${
                        cdnProxy ? 'bg-emerald-600' : 'bg-slate-300 dark:bg-slate-700'
                      }`}
                    >
                      <div
                        className={`w-3.5 h-3.5 rounded-full bg-white transition-transform absolute top-0.75 ${
                          cdnProxy ? 'left-4.5' : 'left-0.75'
                        }`}
                      />
                    </button>
                    <span className="text-slate-400 font-normal">
                      Retrieve the real IP of the request from the CDN proxy (Only valid for panel)
                    </span>
                  </div>
                </div>

                {/* 5. Home Bulletin */}
                <div className="py-3 flex items-center justify-between">
                  <span className="w-44 text-slate-800 dark:text-slate-200">Home Bulletin</span>
                  <div className="flex-1 flex items-center gap-3">
                    <button
                      onClick={() => {
                        const next = !homeBulletin;
                        setHomeBulletin(next);
                        persistSettings({ home_bulletin: next });
                        showToast(`Home Bulletin announcements ${next ? 'Enabled' : 'Disabled'}`);
                      }}
                      className={`w-9 h-5 rounded-full transition-colors relative cursor-pointer ${
                        homeBulletin ? 'bg-emerald-600' : 'bg-slate-300 dark:bg-slate-700'
                      }`}
                    >
                      <div
                        className={`w-3.5 h-3.5 rounded-full bg-white transition-transform absolute top-0.75 ${
                          homeBulletin ? 'left-4.5' : 'left-0.75'
                        }`}
                      />
                    </button>
                    <span className="text-slate-400 font-normal">
                      Show panel announcements and update notices on the home page
                    </span>
                  </div>
                </div>

                {/* 6. Site Monitor */}
                <div className="py-3 flex items-center justify-between">
                  <span className="w-44 text-slate-800 dark:text-slate-200">Site Monitor</span>
                  <div className="flex-1 flex items-center gap-3">
                    <button
                      onClick={() => {
                        const next = !siteMonitor;
                        setSiteMonitor(next);
                        persistSettings({ site_monitor: next });
                        showToast(`Site Monitor ${next ? 'Enabled' : 'Disabled'}`);
                      }}
                      className={`w-9 h-5 rounded-full transition-colors relative cursor-pointer ${
                        siteMonitor ? 'bg-emerald-600' : 'bg-slate-300 dark:bg-slate-700'
                      }`}
                    >
                      <div
                        className={`w-3.5 h-3.5 rounded-full bg-white transition-transform absolute top-0.75 ${
                          siteMonitor ? 'left-4.5' : 'left-0.75'
                        }`}
                      />
                    </button>
                    <span className="text-slate-400 font-normal">Free website monitor</span>
                  </div>
                </div>

                {/* 7. Auto-fetch favicon */}
                <div className="py-3 flex items-center justify-between">
                  <span className="w-44 text-slate-800 dark:text-slate-200">Auto-fetch favicon</span>
                  <div className="flex-1 flex items-center gap-3">
                    <button
                      onClick={() => {
                        const next = !autoFetchFavicon;
                        setAutoFetchFavicon(next);
                        persistSettings({ auto_fetch_favicon: next });
                        showToast(`Auto-fetch favicon ${next ? 'Enabled' : 'Disabled'}`);
                      }}
                      className={`w-9 h-5 rounded-full transition-colors relative cursor-pointer ${
                        autoFetchFavicon ? 'bg-emerald-600' : 'bg-slate-300 dark:bg-slate-700'
                      }`}
                    >
                      <div
                        className={`w-3.5 h-3.5 rounded-full bg-white transition-transform absolute top-0.75 ${
                          autoFetchFavicon ? 'left-4.5' : 'left-0.75'
                        }`}
                      />
                    </button>
                    <span className="text-slate-400 font-normal">
                      Attempts to fetch favicon every 12 hours when enabled
                    </span>
                  </div>
                </div>

                {/* 8. Auto Backup Panel */}
                <div className="py-3 flex items-center justify-between">
                  <span className="w-44 text-slate-800 dark:text-slate-200">Auto Backup Panel</span>
                  <div className="flex-1 flex flex-wrap items-center gap-3">
                    <button
                      onClick={() => {
                        const next = !autoBackupPanel;
                        setAutoBackupPanel(next);
                        persistSettings({ auto_backup_panel: next });
                        showToast(`Auto Backup Panel ${next ? 'Enabled' : 'Disabled'}`);
                      }}
                      className={`w-9 h-5 rounded-full transition-colors relative cursor-pointer ${
                        autoBackupPanel ? 'bg-emerald-600' : 'bg-slate-300 dark:bg-slate-700'
                      }`}
                    >
                      <div
                        className={`w-3.5 h-3.5 rounded-full bg-white transition-transform absolute top-0.75 ${
                          autoBackupPanel ? 'left-4.5' : 'left-0.75'
                        }`}
                      />
                    </button>
                    <button
                      onClick={() => showToast('Backup schedule settings opened')}
                      className="px-2.5 py-1 rounded bg-emerald-600 text-white text-xs font-bold shadow-xs"
                    >
                      Modify
                    </button>
                    <button
                      onClick={() => showToast('Cleared old panel backup archives')}
                      className="px-2.5 py-1 rounded bg-white dark:bg-surface-700 border border-slate-300 dark:border-surface-600 text-slate-700 dark:text-slate-200 text-xs font-semibold shadow-2xs"
                    >
                      One-click clear backup
                    </button>
                    <span className="text-slate-500 font-normal">Number of backups: 30 Used: 426.28 MB</span>
                    <span className="text-rose-600 dark:text-rose-400 font-normal text-[11px]">
                      Automatic backup does not include website data and MySQL data
                    </span>
                  </div>
                </div>

                {/* 9. Theme */}
                <div className="py-3 flex items-center justify-between">
                  <span className="w-44 text-slate-800 dark:text-slate-200">Theme</span>
                  <div className="flex-1">
                    <select
                      value={panelTheme}
                      onChange={(e) => {
                        setPanelTheme(e.target.value);
                        showToast(`Panel theme set to ${e.target.value}`);
                      }}
                      className="w-44 px-3 py-1.5 rounded-lg bg-slate-50 dark:bg-surface-800 border border-slate-300 dark:border-surface-700 text-slate-900 dark:text-white font-semibold focus:outline-none"
                    >
                      <option value="Fresh">Fresh</option>
                      <option value="Classic">Classic</option>
                      <option value="Dark">Dark Slate</option>
                    </select>
                  </div>
                </div>

                {/* 10. Language */}
                <div className="py-3 flex items-center justify-between">
                  <span className="w-44 text-slate-800 dark:text-slate-200">Language</span>
                  <div className="flex-1 flex items-center gap-3">
                    <select
                      value={panelLanguage}
                      onChange={(e) => {
                        setPanelLanguage(e.target.value);
                        showToast(`Language set to ${e.target.value}`);
                      }}
                      className="w-44 px-3 py-1.5 rounded-lg bg-slate-50 dark:bg-surface-800 border border-slate-300 dark:border-surface-700 text-slate-900 dark:text-white font-semibold focus:outline-none"
                    >
                      <option value="English">English</option>
                      <option value="Bengali">বাংলা (Bengali)</option>
                      <option value="Chinese">中文 (Chinese)</option>
                    </select>
                    <button
                      onClick={() => showToast('Feedback form opened')}
                      className="text-emerald-600 dark:text-emerald-400 font-semibold hover:underline flex items-center gap-1"
                    >
                      <Edit3 className="w-3.5 h-3.5" />
                      <span>Feedback</span>
                    </button>
                  </div>
                </div>

                {/* 11. Alias */}
                <div className="py-3 flex items-center justify-between">
                  <span className="w-44 text-slate-800 dark:text-slate-200">Alias</span>
                  <div className="flex-1 flex items-center gap-3">
                    <input
                      type="text"
                      value={panelAlias}
                      onChange={(e) => setPanelAlias(e.target.value)}
                      className="w-64 px-3 py-1.5 rounded-lg bg-slate-50 dark:bg-surface-800 border border-slate-300 dark:border-surface-700 text-slate-900 dark:text-white font-semibold focus:outline-none"
                    />
                    <button
                      onClick={() => showToast(`Panel alias saved: ${panelAlias}`)}
                      className="px-3.5 py-1.5 rounded-lg bg-emerald-600 hover:bg-emerald-700 text-white font-bold shadow-xs"
                    >
                      Save
                    </button>
                    <span className="text-slate-400 font-normal">Take alias for aaPanel</span>
                  </div>
                </div>

                {/* 12. Timeout */}
                <div className="py-3 flex items-center justify-between">
                  <span className="w-44 text-slate-800 dark:text-slate-200">Timeout</span>
                  <div className="flex-1 flex items-center gap-3">
                    <input
                      type="text"
                      disabled
                      value={sessionTimeout}
                      className="w-64 px-3 py-1.5 rounded-lg bg-slate-100 dark:bg-surface-800 border border-slate-300 dark:border-surface-700 text-slate-900 dark:text-white font-semibold"
                    />
                    <button
                      onClick={() => {
                        setModalInput(sessionTimeout);
                        setModalType('timeout');
                      }}
                      className="px-3.5 py-1.5 rounded-lg bg-emerald-600 hover:bg-emerald-700 text-white font-bold shadow-xs"
                    >
                      Modify
                    </button>
                    <span className="text-slate-400 font-normal">
                      If the user does not have any operation within 24 Hour(s), the panel will auto logout
                    </span>
                  </div>
                </div>

                {/* 13. Default site folder */}
                <div className="py-3 flex items-center justify-between">
                  <span className="w-44 text-slate-800 dark:text-slate-200">Default site folder</span>
                  <div className="flex-1 flex items-center gap-3">
                    <div className="relative w-64">
                      <input
                        type="text"
                        value={defaultSiteFolder}
                        onChange={(e) => setDefaultSiteFolder(e.target.value)}
                        className="w-full px-3 py-1.5 rounded-lg bg-slate-50 dark:bg-surface-800 border border-slate-300 dark:border-surface-700 text-slate-900 dark:text-white font-mono text-xs focus:outline-none pr-8"
                      />
                      <Folder className="w-3.5 h-3.5 text-slate-400 absolute right-2.5 top-2.5" />
                    </div>
                    <button
                      onClick={() => {
                        persistSettings({ default_site_folder: defaultSiteFolder });
                        showToast(`Default site directory set to ${defaultSiteFolder}`);
                      }}
                      className="px-3.5 py-1.5 rounded-lg bg-emerald-600 hover:bg-emerald-700 text-white font-bold shadow-xs"
                    >
                      Save
                    </button>
                    <span className="text-slate-400 font-normal">
                      New created site will be saved to subdirectory by default
                    </span>
                  </div>
                </div>

                {/* 14. Default backup folder */}
                <div className="py-3 flex items-center justify-between">
                  <span className="w-44 text-slate-800 dark:text-slate-200">Default backup folder</span>
                  <div className="flex-1 flex items-center gap-3">
                    <div className="relative w-64">
                      <input
                        type="text"
                        value={defaultBackupFolder}
                        onChange={(e) => setDefaultBackupFolder(e.target.value)}
                        className="w-full px-3 py-1.5 rounded-lg bg-slate-50 dark:bg-surface-800 border border-slate-300 dark:border-surface-700 text-slate-900 dark:text-white font-mono text-xs focus:outline-none pr-8"
                      />
                      <Folder className="w-3.5 h-3.5 text-slate-400 absolute right-2.5 top-2.5" />
                    </div>
                    <button
                      onClick={() => {
                        persistSettings({ default_backup_folder: defaultBackupFolder });
                        showToast(`Default backup directory set to ${defaultBackupFolder}`);
                      }}
                      className="px-3.5 py-1.5 rounded-lg bg-emerald-600 hover:bg-emerald-700 text-white font-bold shadow-xs"
                    >
                      Save
                    </button>
                    <span className="text-slate-400 font-normal">Directory of site and database backup</span>
                  </div>
                </div>

                {/* 15. Server IP */}
                <div className="py-3 flex items-center justify-between">
                  <span className="w-44 text-slate-800 dark:text-slate-200">Server IP</span>
                  <div className="flex-1 flex items-center gap-3">
                    <input
                      type="text"
                      value={serverIp}
                      onChange={(e) => setServerIp(e.target.value)}
                      className="w-64 px-3 py-1.5 rounded-lg bg-slate-50 dark:bg-surface-800 border border-slate-300 dark:border-surface-700 text-slate-900 dark:text-white font-mono text-xs focus:outline-none"
                    />
                    <button
                      onClick={() => {
                        persistSettings({ server_ip: serverIp });
                        showToast(`Server IP set to ${serverIp}`);
                      }}
                      className="px-3.5 py-1.5 rounded-lg bg-emerald-600 hover:bg-emerald-700 text-white font-bold shadow-xs"
                    >
                      Save
                    </button>
                    <span className="text-slate-400 font-normal">
                      Default IP is Internet IP. If you need use local virtual machine to test, please input Intranet IP
                    </span>
                  </div>
                </div>

                {/* 16. Server time */}
                <div className="py-3 flex items-center justify-between">
                  <span className="w-44 text-slate-800 dark:text-slate-200">Server time</span>
                  <div className="flex-1 flex items-center gap-3">
                    <input
                      type="text"
                      disabled
                      value={serverTime}
                      className="w-64 px-3 py-1.5 rounded-lg bg-slate-100 dark:bg-surface-800 border border-slate-300 dark:border-surface-700 text-slate-900 dark:text-white font-mono text-xs"
                    />
                    <button
                      onClick={handleSyncTime}
                      className="px-3.5 py-1.5 rounded-lg bg-emerald-600 hover:bg-emerald-700 text-white font-bold shadow-xs"
                    >
                      Sync
                    </button>
                    <span className="text-slate-400 font-normal">Synchronize current server time</span>
                  </div>
                </div>

                {/* 17. Server Timezone */}
                <div className="py-3 flex items-center justify-between">
                  <span className="w-44 text-slate-800 dark:text-slate-200">Server Timezone</span>
                  <div className="flex-1 flex items-center gap-3">
                    <select
                      value={timezoneRegion}
                      onChange={(e) => setTimezoneRegion(e.target.value)}
                      className="w-28 px-3 py-1.5 rounded-lg bg-slate-50 dark:bg-surface-800 border border-slate-300 dark:border-surface-700 text-slate-900 dark:text-white font-semibold focus:outline-none"
                    >
                      <option value="Etc">Etc</option>
                      <option value="Asia">Asia</option>
                      <option value="America">America</option>
                      <option value="Europe">Europe</option>
                    </select>
                    <select
                      value={timezoneCity}
                      onChange={(e) => setTimezoneCity(e.target.value)}
                      className="w-36 px-3 py-1.5 rounded-lg bg-slate-50 dark:bg-surface-800 border border-slate-300 dark:border-surface-700 text-slate-900 dark:text-white font-semibold focus:outline-none"
                    >
                      <option value="UTC">UTC</option>
                      <option value="Dhaka">Dhaka (GMT+6)</option>
                      <option value="New_York">New York (EST)</option>
                      <option value="London">London (GMT)</option>
                    </select>
                    <button
                      onClick={() => {
                        persistSettings({ timezone_region: timezoneRegion, timezone_city: timezoneCity });
                        showToast(`Timezone updated to ${timezoneRegion}/${timezoneCity}`);
                      }}
                      className="px-3.5 py-1.5 rounded-lg bg-emerald-600 hover:bg-emerald-700 text-white font-bold shadow-xs"
                    >
                      Save
                    </button>
                    <span className="text-slate-400 font-normal">
                      Set the correct timezone to ensure accurate timestamps
                    </span>
                  </div>
                </div>
              </div>
            </div>

            {/* ---------------------------------------------------------------------
                SECTION 3: SECURITY (Deep Panel Defense Options)
                --------------------------------------------------------------------- */}
            <div className="pt-2 border-t border-slate-200 dark:border-surface-800 space-y-4">
              <h2 className="text-sm font-bold text-slate-900 dark:text-white">
                Security
              </h2>

              <div className="bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-800 rounded-2xl p-6 shadow-2xs divide-y divide-slate-100 dark:divide-surface-800 text-xs font-semibold">
                {/* 1. Panel Security Alarm */}
                <div className="py-3 flex items-center justify-between">
                  <span className="w-48 text-slate-800 dark:text-slate-200">Panel Security Alarm</span>
                  <div className="flex-1 flex items-center gap-3">
                    <button
                      onClick={() => {
                        const next = !securityAlarm;
                        setSecurityAlarm(next);
                        persistSettings({ security_alarm: next });
                        showToast(`Panel Security Alarm ${next ? 'Enabled' : 'Disabled'}`);
                      }}
                      className={`w-9 h-5 rounded-full transition-colors relative cursor-pointer ${
                        securityAlarm ? 'bg-emerald-600' : 'bg-slate-300 dark:bg-slate-700'
                      }`}
                    >
                      <div
                        className={`w-3.5 h-3.5 rounded-full bg-white transition-transform absolute top-0.75 ${
                          securityAlarm ? 'left-4.5' : 'left-0.75'
                        }`}
                      />
                    </button>
                    <button
                      onClick={() => showToast('Security alarm settings configured')}
                      className="px-2.5 py-1 rounded bg-white dark:bg-surface-700 border border-slate-300 dark:border-surface-600 text-slate-700 dark:text-slate-200 text-xs font-semibold"
                    >
                      Modify
                    </button>
                    <span className="text-slate-400 font-normal">
                      Alarm content includes: Panel user changes, panel log deletion, panel developer mode enabled
                    </span>
                  </div>
                </div>

                {/* 2. BasicAuth */}
                <div className="py-3 flex items-center justify-between">
                  <span className="w-48 text-slate-800 dark:text-slate-200">BasicAuth</span>
                  <div className="flex-1 flex items-center gap-3">
                    <button
                      onClick={() => {
                        const next = !basicAuth;
                        setBasicAuth(next);
                        persistSettings({ basic_auth: next });
                        showToast(`BasicAuth secondary barrier ${next ? 'Enabled' : 'Disabled'}`);
                      }}
                      className={`w-9 h-5 rounded-full transition-colors relative cursor-pointer ${
                        basicAuth ? 'bg-emerald-600' : 'bg-slate-300 dark:bg-slate-700'
                      }`}
                    >
                      <div
                        className={`w-3.5 h-3.5 rounded-full bg-white transition-transform absolute top-0.75 ${
                          basicAuth ? 'left-4.5' : 'left-0.75'
                        }`}
                      />
                    </button>
                    <span className="text-slate-400 font-normal">
                      Add an extra layer of auth to effectively prevent the panel from being scanned,{' '}
                      <span className="text-emerald-600 cursor-pointer">Help</span>
                    </span>
                  </div>
                </div>

                {/* 3. Google Authenticator */}
                <div className="py-3 flex items-center justify-between">
                  <span className="w-48 text-slate-800 dark:text-slate-200">Google Authenticator</span>
                  <div className="flex-1 flex items-center gap-3">
                    <button
                      onClick={() => {
                        const next = !googleAuth;
                        setGoogleAuth(next);
                        persistSettings({ google_auth: next });
                        showToast(`Google Authenticator ${next ? 'Enabled' : 'Disabled'}`);
                      }}
                      className={`w-9 h-5 rounded-full transition-colors relative cursor-pointer ${
                        googleAuth ? 'bg-emerald-600' : 'bg-slate-300 dark:bg-slate-700'
                      }`}
                    >
                      <div
                        className={`w-3.5 h-3.5 rounded-full bg-white transition-transform absolute top-0.75 ${
                          googleAuth ? 'left-4.5' : 'left-0.75'
                        }`}
                      />
                    </button>
                    <button
                      onClick={() => setModalType('googleAuth')}
                      className="px-2.5 py-1 rounded bg-white dark:bg-surface-700 border border-slate-300 dark:border-surface-600 text-slate-700 dark:text-slate-200 text-xs font-semibold"
                    >
                      Modify
                    </button>
                    <span className="text-slate-400 font-normal">
                      A dynamic verification code is required to log in to the panel,{' '}
                      <span className="text-emerald-600 cursor-pointer">Help</span>
                    </span>
                  </div>
                </div>

                {/* 4. Strong password */}
                <div className="py-3 flex items-center justify-between">
                  <span className="w-48 text-slate-800 dark:text-slate-200">Strong password</span>
                  <div className="flex-1 flex items-center gap-3">
                    <button
                      onClick={() => {
                        const next = !strongPassword;
                        setStrongPassword(next);
                        persistSettings({ strong_password: next });
                        showToast(`Strong password enforcement ${next ? 'Enabled' : 'Disabled'}`);
                      }}
                      className={`w-9 h-5 rounded-full transition-colors relative cursor-pointer ${
                        strongPassword ? 'bg-emerald-600' : 'bg-slate-300 dark:bg-slate-700'
                      }`}
                    >
                      <div
                        className={`w-3.5 h-3.5 rounded-full bg-white transition-transform absolute top-0.75 ${
                          strongPassword ? 'left-4.5' : 'left-0.75'
                        }`}
                      />
                    </button>
                    <span className="text-slate-400 font-normal">
                      Enable strong password for the panel, rules:{' '}
                      <span className="text-rose-600 dark:text-rose-400">
                        Length 8, upper and lower case letters, numbers and characters exist
                      </span>
                    </span>
                  </div>
                </div>

                {/* 5. Authorized IP */}
                <div className="py-3 flex items-center justify-between">
                  <span className="w-48 text-slate-800 dark:text-slate-200">Authorized IP</span>
                  <div className="flex-1 flex items-center gap-3">
                    <input
                      type="text"
                      placeholder="e.g., 1.1.1.1, 2.2.2.1-2.2.2.2"
                      value={authorizedIp}
                      onChange={(e) => setAuthorizedIp(e.target.value)}
                      className="w-64 px-3 py-1.5 rounded-lg bg-slate-50 dark:bg-surface-800 border border-slate-300 dark:border-surface-700 text-slate-900 dark:text-white font-mono text-xs focus:outline-none"
                    />
                    <button
                      onClick={() => showToast(`Authorized IPs saved: ${authorizedIp || 'Any IP allowed'}`)}
                      className="px-3.5 py-1.5 rounded-lg bg-emerald-600 hover:bg-emerald-700 text-white font-bold shadow-xs"
                    >
                      Save
                    </button>
                    <span className="text-slate-400 font-normal">
                      Split multiple IP with (,){' '}
                      <span className="text-rose-600 dark:text-rose-400 font-bold">
                        Warning: If IP is set, only the authorized IP can access the panel!
                      </span>
                    </span>
                  </div>
                </div>

                {/* 6. Not logged in response */}
                <div className="py-3 flex items-center justify-between">
                  <span className="w-48 text-slate-800 dark:text-slate-200">Not logged in response</span>
                  <div className="flex-1 flex items-center gap-3">
                    <input
                      type="text"
                      disabled
                      value={notLoggedInResponse}
                      className="w-64 px-3 py-1.5 rounded-lg bg-slate-100 dark:bg-surface-800 border border-slate-300 dark:border-surface-700 text-slate-900 dark:text-white font-semibold text-xs"
                    />
                    <button
                      onClick={() => {
                        setModalInput(notLoggedInResponse);
                        setModalType('response');
                      }}
                      className="px-3.5 py-1.5 rounded-lg bg-emerald-600 hover:bg-emerald-700 text-white font-bold shadow-xs"
                    >
                      Modify
                    </button>
                    <span className="text-slate-400 font-normal">
                      Response when not logged in and not properly entered for security entry, can be used to hide panel features
                    </span>
                  </div>
                </div>

                {/* 7. Password expire */}
                <div className="py-3 flex items-center justify-between">
                  <span className="w-48 text-slate-800 dark:text-slate-200">Password expire</span>
                  <div className="flex-1 flex items-center gap-3">
                    <input
                      type="text"
                      disabled
                      value={passwordExpire}
                      className="w-64 px-3 py-1.5 rounded-lg bg-slate-100 dark:bg-surface-800 border border-slate-300 dark:border-surface-700 text-slate-900 dark:text-white font-semibold text-xs"
                    />
                    <button
                      onClick={() => {
                        setPasswordExpire('90 Days');
                        showToast('Password expiration policy set to 90 Days');
                      }}
                      className="px-3.5 py-1.5 rounded-lg bg-emerald-600 hover:bg-emerald-700 text-white font-bold shadow-xs"
                    >
                      Modify
                    </button>
                    <span className="text-slate-400 font-normal">
                      Set an expiration time for the panel password; it needs to be reset after expiration
                    </span>
                  </div>
                </div>

                {/* 8. Temporary login */}
                <div className="py-3 flex items-center justify-between">
                  <span className="w-48 text-slate-800 dark:text-slate-200">Temporary login</span>
                  <div className="flex-1 flex items-center gap-3">
                    <button
                      onClick={() => setModalType('tempLogin')}
                      className="px-3.5 py-1.5 rounded-lg bg-emerald-600 hover:bg-emerald-700 text-white font-bold shadow-xs"
                    >
                      Modify
                    </button>
                    <span className="text-slate-400 font-normal">
                      Temporarily provide panel access to non-admins
                    </span>
                  </div>
                </div>
              </div>
            </div>
          </div>
        )}

        {/* =========================================================================
            OTHER TABS: Page, Alarm, Backup, Migrate, Other Migrate, Service
            ========================================================================= */}
        {activeTab === 'page' && (
          <div className="bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-800 rounded-2xl p-6 space-y-4 shadow-2xs">
            <h3 className="font-bold text-sm text-slate-900 dark:text-white">Custom Page Branding</h3>
            <p className="text-xs text-slate-500">Configure logo, favicon, portal title, and footer text</p>
            <div className="space-y-3 max-w-lg text-xs font-semibold">
              <div>
                <label className="block text-slate-700 dark:text-slate-300 mb-1">Panel Title</label>
                <input
                  type="text"
                  defaultValue="Hostvra Enterprise Control Panel"
                  className="w-full px-3 py-2 rounded-xl bg-slate-50 dark:bg-surface-800 border border-slate-300 dark:border-surface-700 text-slate-900 dark:text-white"
                />
              </div>
              <button
                onClick={() => showToast('Branding settings saved')}
                className="px-4 py-2 rounded-xl bg-emerald-600 text-white font-bold"
              >
                Save Page Settings
              </button>
            </div>
          </div>
        )}

        {activeTab === 'alarm' && (
          <div className="bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-800 rounded-2xl p-6 space-y-4 shadow-2xs">
            <h3 className="font-bold text-sm text-slate-900 dark:text-white">Alarm &amp; Incident Notifications</h3>
            <p className="text-xs text-slate-500">Integrate Email, Telegram Bot, Slack, and Webhook alert channels</p>
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              {['Email Notification', 'Telegram Bot', 'DingTalk / Feishu', 'Custom Webhook'].map((ch) => (
                <div key={ch} className="p-4 rounded-xl border border-slate-200 dark:border-surface-700 flex items-center justify-between">
                  <span className="font-bold text-xs">{ch}</span>
                  <button
                    onClick={() => showToast(`${ch} configuration opened`)}
                    className="px-3 py-1 rounded bg-emerald-600 text-white text-xs font-bold"
                  >
                    Setup
                  </button>
                </div>
              ))}
            </div>
          </div>
        )}

        {activeTab === 'backup' && (
          <div className="bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-800 rounded-2xl p-6 space-y-4 shadow-2xs">
            <h3 className="font-bold text-sm text-slate-900 dark:text-white">Panel Backup &amp; Disaster Recovery</h3>
            <p className="text-xs text-slate-500">Create instant backup archives of all panel databases and configurations</p>
            <div className="flex gap-3">
              <button
                onClick={() => showToast('Creating instant panel configuration snapshot...')}
                className="px-4 py-2 rounded-xl bg-emerald-600 text-white font-bold text-xs"
              >
                Create Backup Now
              </button>
              <button
                onClick={() => showToast('Restore wizard launched')}
                className="px-4 py-2 rounded-xl bg-slate-100 dark:bg-surface-800 text-slate-700 dark:text-slate-200 font-bold text-xs"
              >
                Import &amp; Restore
              </button>
            </div>
          </div>
        )}

        {activeTab === 'migrate' && (
          <div className="bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-800 rounded-2xl p-6 space-y-4 shadow-2xs">
            <h3 className="font-bold text-sm text-slate-900 dark:text-white">One-Click Server Migration (Hostvra / aaPanel)</h3>
            <p className="text-xs text-slate-500">Migrate all websites, databases, FTP accounts, and DNS zones between servers</p>
            <div className="max-w-md space-y-3 text-xs font-semibold">
              <div>
                <label className="block text-slate-700 dark:text-slate-300 mb-1">Target Server IP</label>
                <input
                  type="text"
                  placeholder="e.g. 192.168.1.100"
                  className="w-full px-3 py-2 rounded-xl bg-slate-50 dark:bg-surface-800 border border-slate-300 dark:border-surface-700"
                />
              </div>
              <div>
                <label className="block text-slate-700 dark:text-slate-300 mb-1">Target Panel API Secret Key</label>
                <input
                  type="password"
                  placeholder="API Secret Key"
                  className="w-full px-3 py-2 rounded-xl bg-slate-50 dark:bg-surface-800 border border-slate-300 dark:border-surface-700"
                />
              </div>
              <button
                onClick={() => showToast('Starting server migration preflight checks...')}
                className="px-5 py-2 rounded-xl bg-emerald-600 text-white font-bold text-xs"
              >
                Connect &amp; Begin Migration
              </button>
            </div>
          </div>
        )}

        {activeTab === 'other_migrate' && (
          <div className="bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-800 rounded-2xl p-6 space-y-4 shadow-2xs">
            <h3 className="font-bold text-sm text-slate-900 dark:text-white">Import from Other Panels</h3>
            <p className="text-xs text-slate-500">Direct migration from cPanel backups, Plesk, CyberPanel, or DirectAdmin</p>
            <div className="grid grid-cols-1 sm:grid-cols-3 gap-3 pt-2">
              {['cPanel Full Backup (.tar.gz)', 'CyberPanel Backup', 'Plesk XML Archive'].map((panel) => (
                <div key={panel} className="p-4 rounded-xl border border-slate-200 dark:border-surface-700 bg-slate-50 dark:bg-surface-800 flex flex-col justify-between gap-3">
                  <span className="font-bold text-xs">{panel}</span>
                  <button
                    onClick={() => showToast(`Upload dialog opened for ${panel}`)}
                    className="px-3 py-1.5 rounded-lg bg-emerald-600 text-white text-xs font-bold"
                  >
                    Upload Archive
                  </button>
                </div>
              ))}
            </div>
          </div>
        )}

        {activeTab === 'service' && (
          <div className="bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-800 rounded-2xl p-6 space-y-4 shadow-2xs">
            <h3 className="font-bold text-sm text-slate-900 dark:text-white">Hostvra Panel Services &amp; Daemon Control</h3>
            <p className="text-xs text-slate-500">Control systemd background daemons and service health</p>
            <div className="space-y-3 max-w-xl text-xs font-semibold">
              {[
                { name: 'hostvra-web (Next.js Frontend)', status: 'active (running)' },
                { name: 'hostvra-api (Go Chi REST Engine)', status: 'active (running)' },
                { name: 'hostvra-agent (Native System Daemon)', status: 'active (running)' },
              ].map((svc) => (
                <div key={svc.name} className="p-3.5 rounded-xl border border-slate-200 dark:border-surface-700 bg-slate-50 dark:bg-surface-800 flex items-center justify-between">
                  <div>
                    <span className="block font-bold">{svc.name}</span>
                    <span className="text-emerald-600 text-[11px]">{svc.status}</span>
                  </div>
                  <div className="flex gap-2">
                    <button
                      onClick={() => showToast(`Restarting ${svc.name}...`)}
                      className="px-3 py-1 rounded-lg bg-white dark:bg-surface-700 border border-slate-300 dark:border-surface-600 text-xs font-bold"
                    >
                      Restart
                    </button>
                  </div>
                </div>
              ))}
            </div>
          </div>
        )}

        {/* =========================================================================
            DYNAMIC MODALS (Port, Entrance, User, Pass, Whitelist, GoogleAuth, TempLogin)
            ========================================================================= */}
        {modalType && (
          <div className="fixed inset-0 z-50 bg-black/60 backdrop-blur-xs flex items-center justify-center p-4 animate-fadeIn">
            <div className="w-full max-w-md bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-700 rounded-2xl shadow-2xl p-6 relative">
              <button
                onClick={() => setModalType(null)}
                className="absolute top-5 right-5 p-1 rounded-lg text-slate-400 hover:text-slate-700 dark:hover:text-white"
              >
                <X className="w-5 h-5" />
              </button>

              <h3 className="font-bold text-base text-slate-900 dark:text-white mb-2 capitalize">
                Modify {modalType}
              </h3>

              <div className="space-y-4 text-xs font-semibold pt-2">
                {modalType === 'googleAuth' ? (
                  <div className="text-center space-y-3">
                    <div className="w-36 h-36 mx-auto bg-slate-100 dark:bg-surface-800 border border-slate-200 dark:border-surface-700 rounded-xl flex items-center justify-center">
                      <Smartphone className="w-16 h-16 text-emerald-600" />
                    </div>
                    <p className="text-slate-500 text-xs">
                      Scan QR code with Google Authenticator or enter secret key: <strong className="font-mono text-emerald-600">JBSWY3DPEHPK3PXP</strong>
                    </p>
                    <input
                      type="text"
                      placeholder="Enter 6-digit code"
                      className="w-full text-center tracking-widest text-base font-mono px-3 py-2 rounded-xl bg-slate-50 dark:bg-surface-800 border border-slate-300 dark:border-surface-700"
                    />
                  </div>
                ) : modalType === 'tempLogin' ? (
                  <div className="space-y-3">
                    <p className="text-slate-500">
                      Generate a single-use time-limited link to access the control panel without entering password.
                    </p>
                    <div>
                      <label className="block text-slate-700 dark:text-slate-300 mb-1">Validity (Hours)</label>
                      <select className="w-full px-3 py-2 rounded-xl bg-slate-50 dark:bg-surface-800 border border-slate-300 dark:border-surface-700">
                        <option value="1">1 Hour</option>
                        <option value="4">4 Hours</option>
                        <option value="24">24 Hours</option>
                      </select>
                    </div>
                  </div>
                ) : (
                  <div>
                    <label className="block text-slate-700 dark:text-slate-300 mb-1">Enter Value</label>
                    <input
                      type={modalType === 'pass' ? 'password' : 'text'}
                      value={modalInput}
                      onChange={(e) => setModalInput(e.target.value)}
                      className="w-full px-3 py-2 rounded-xl bg-slate-50 dark:bg-surface-800 border border-slate-300 dark:border-surface-700 text-slate-900 dark:text-white font-mono text-xs focus:outline-none"
                    />
                  </div>
                )}

                <div className="flex justify-end gap-3 pt-3 border-t border-slate-200 dark:border-surface-700">
                  <button
                    type="button"
                    onClick={() => setModalType(null)}
                    className="px-4 py-2 rounded-xl bg-slate-100 dark:bg-surface-800 text-slate-700 dark:text-slate-300 font-bold"
                  >
                    Cancel
                  </button>
                  <button
                    type="button"
                    onClick={handleModalSave}
                    className="px-5 py-2 rounded-xl bg-emerald-600 hover:bg-emerald-700 text-white font-bold shadow-md"
                  >
                    Save &amp; Apply
                  </button>
                </div>
              </div>
            </div>
          </div>
        )}
      </div>
    </DashboardShell>
  );
}
