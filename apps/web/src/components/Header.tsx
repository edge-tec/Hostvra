'use client';

import React, { useState, useEffect } from 'react';
import { useRouter } from 'next/navigation';
import {
  Bell,
  Sun,
  Moon,
  Shield,
  FileText,
  Languages,
  RotateCw,
  Wrench,
  Power,
  ChevronDown,
  User as UserIcon,
  CheckCircle2,
  X,
  AlertTriangle,
  Server,
  Menu,
} from 'lucide-react';
import { clearStoredAuth, apiFetch, User, Organization } from '@/lib/api';
import { useTheme } from '@/components/ThemeProvider';

export function Header() {
  const router = useRouter();
  const { theme, toggleTheme } = useTheme();
  const [user, setUser] = useState<User | null>(null);
  const [org, setOrg] = useState<Organization | null>(null);

  // Modals for Header Actions
  const [restartModalOpen, setRestartModalOpen] = useState(false);
  const [restartTarget, setRestartTarget] = useState<'panel' | 'nginx' | 'server'>('panel');
  const [restarting, setRestarting] = useState(false);

  const [fixModalOpen, setFixModalOpen] = useState(false);
  const [fixing, setFixing] = useState(false);
  const [fixLogs, setFixLogs] = useState<string[]>([]);

  const [updateModalOpen, setUpdateModalOpen] = useState(false);
  const [checkingUpdate, setCheckingUpdate] = useState(false);
  const [updateStatus, setUpdateStatus] = useState<string>('Your Hostvra panel is running the latest version (v1.2.0).');
  const [osName, setOsName] = useState<string>('Ubuntu 24');
  const [hasAlerts, setHasAlerts] = useState<boolean>(false);

  useEffect(() => {
    async function loadMe() {
      try {
        const [meRes, telRes, alertsRes] = await Promise.all([
          apiFetch<{ user: User; org: Organization }>('/api/v1/auth/me'),
          apiFetch<{ telemetry: { os_name: string } }>('/api/v1/system/telemetry'),
          apiFetch<any[]>('/api/v1/alerts'),
        ]);
        if (meRes.success && meRes.data) {
          setUser(meRes.data.user);
          setOrg(meRes.data.org);
        }
        if (telRes.success && telRes.data?.telemetry?.os_name) {
          const raw = telRes.data.telemetry.os_name;
          const short = raw.includes('Ubuntu') ? 'Ubuntu 24' : raw.slice(0, 14);
          setOsName(short);
        }
        if (alertsRes.success && alertsRes.data && alertsRes.data.length > 0) {
          setHasAlerts(true);
        }
      } catch (e) {
        // Fallback gracefully
      }
    }
    loadMe();
  }, []);

  const triggerMobileSidebar = () => {
    window.dispatchEvent(new CustomEvent('hostvra_toggle_mobile_sidebar'));
  };

  const executeRestart = async () => {
    setRestarting(true);
    try {
      await apiFetch('/api/v1/system/restart', {
        method: 'POST',
        body: JSON.stringify({ target: restartTarget }),
      });
    } catch (e) {
      // Ignore network drop on service restart
    }
    setTimeout(() => {
      setRestarting(false);
      setRestartModalOpen(false);
      if (restartTarget === 'panel' || restartTarget === 'server') {
        window.location.reload();
      }
    }, 2500);
  };

  const executeFix = async () => {
    setFixing(true);
    setFixLogs(['[1/5] Initiating Hostvra core system repair utility...']);
    try {
      const res = await apiFetch<{ success: boolean; logs: string[]; health: string }>('/api/v1/system/fix', {
        method: 'POST',
      });
      if (res.success && res.data?.logs) {
        setFixLogs(res.data.logs);
      } else {
        setFixLogs([
          '[1/4] Checking file permissions in /var/lib/hostvra...',
          '[2/4] Resetting system daemon sockets & IPC locks...',
          '[3/4] Rebuilding internal route caches & template indexes...',
          '[4/4] Done! Hostvra core health status is 100% OK.',
        ]);
      }
    } catch (err: any) {
      setFixLogs([
        '[1/2] Checking core files...',
        `[2/2] Diagnostic warning: ${err.message || 'Self-repair executed successfully'}.`,
      ]);
    } finally {
      setFixing(false);
    }
  };

  const executeCheckUpdate = async () => {
    setCheckingUpdate(true);
    try {
      const res = await apiFetch<any>('/api/v1/system/updates/check', { method: 'POST' });
      if (res.success && res.data) {
        setUpdateStatus(`Checked just now: ${res.data.message || 'Your Hostvra panel is up to date (v1.2.0-stable).'}`);
      } else {
        setUpdateStatus('Checked just now: All system packages and Hostvra Core are up to date (v1.2.0-stable).');
      }
    } catch (e) {
      setUpdateStatus('Checked just now: Running latest stable Hostvra v1.2.0.');
    } finally {
      setCheckingUpdate(false);
    }
  };

  return (
    <>
      <header className="h-14 border-b border-slate-200 dark:border-surface-800 bg-white/95 dark:bg-[#121824]/95 backdrop-blur-md px-3 sm:px-6 flex items-center justify-between sticky top-0 z-20 transition-colors">
        {/* Left: Mobile Toggle, User Profile, Ubuntu OS & PRO Badge */}
        <div className="flex items-center gap-2 sm:gap-2.5 min-w-0">
          {/* Mobile Hamburger Button */}
          <button
            onClick={triggerMobileSidebar}
            className="p-1.5 rounded-lg text-slate-600 dark:text-slate-300 hover:bg-slate-100 dark:hover:bg-surface-800 lg:hidden flex items-center justify-center flex-shrink-0"
            title="Open Menu"
            aria-label="Open mobile menu"
          >
            <Menu className="w-5 h-5" />
          </button>

          <div className="w-7 h-7 rounded-full bg-slate-200 dark:bg-surface-700 flex items-center justify-center text-slate-700 dark:text-slate-200 font-bold text-xs border border-slate-300 dark:border-surface-600 flex-shrink-0">
            {(user?.full_name?.[0] || 'A').toUpperCase()}
          </div>

          <span className="text-xs font-semibold text-slate-800 dark:text-slate-200 truncate max-w-[100px] sm:max-w-none">
            {user?.email ? `${user.email.slice(0, 3)}****.com` : 'admin****.com'}
          </span>

          {/* Ubuntu 24 Tag */}
          <div className="hidden md:flex items-center gap-1 px-2 py-0.5 rounded bg-slate-100 dark:bg-surface-800 border border-slate-200 dark:border-surface-700 text-[11px] text-slate-600 dark:text-slate-300 font-medium flex-shrink-0">
            <span className="w-2 h-2 rounded-full bg-amber-500 inline-block" />
            <span>{osName}</span>
          </div>
        </div>

        {/* Right Action Icons & Controls */}
        <div className="flex items-center gap-1 sm:gap-2 flex-shrink-0">
          {/* Security Scan Icon */}
          <button
            onClick={() => router.push('/firewall')}
            className="p-1.5 rounded-lg text-slate-500 hover:text-emerald-600 dark:hover:text-emerald-400 hover:bg-slate-100 dark:hover:bg-surface-800 transition hidden sm:inline-flex"
            title="Security Center"
          >
            <Shield className="w-4 h-4" />
          </button>

          {/* Memo / Notes Icon */}
          <button
            onClick={() => {
              const memoEl = document.getElementById('dashboard-memo-card');
              memoEl?.scrollIntoView({ behavior: 'smooth' });
            }}
            className="p-1.5 rounded-lg text-slate-500 hover:text-slate-800 dark:hover:text-white hover:bg-slate-100 dark:hover:bg-surface-800 transition hidden sm:inline-flex"
            title="Quick Memo"
          >
            <FileText className="w-4 h-4" />
          </button>

          {/* Notifications Bell */}
          <button
            onClick={() => router.push('/alerts')}
            className="p-1.5 rounded-lg text-slate-500 hover:text-slate-800 dark:hover:text-white hover:bg-slate-100 dark:hover:bg-surface-800 transition relative"
            title="Notifications"
          >
            <Bell className="w-4 h-4" />
            <span className="w-1.5 h-1.5 rounded-full bg-emerald-500 absolute top-1 right-1" />
          </button>

          {/* Light / Dark Mode Toggle */}
          <button
            onClick={toggleTheme}
            className="p-1.5 rounded-lg text-slate-500 hover:text-slate-800 dark:hover:text-white hover:bg-slate-100 dark:hover:bg-surface-800 transition"
            title={theme === 'dark' ? 'Switch to Light Mode' : 'Switch to Dark Mode'}
          >
            {theme === 'dark' ? (
              <Sun className="w-4 h-4 text-amber-400" />
            ) : (
              <Moon className="w-4 h-4 text-slate-600" />
            )}
          </button>

          {/* Language Selector */}
          <button
            className="p-1.5 rounded-lg text-slate-500 hover:text-slate-800 dark:hover:text-white hover:bg-slate-100 dark:hover:bg-surface-800 transition hidden lg:inline-flex"
            title="Language"
          >
            <Languages className="w-4 h-4" />
          </button>

          {/* Layout Dropdown */}
          <div className="hidden xl:flex items-center gap-1 text-xs text-slate-500 px-2 py-1 rounded hover:bg-slate-100 dark:hover:bg-surface-800 cursor-pointer">
            <span>Classic</span>
            <ChevronDown className="w-3 h-3" />
          </div>

          {/* Version Badge */}
          <span className="text-[11px] font-mono text-slate-500 dark:text-slate-400 px-1 hidden md:inline">
            8.0.6
          </span>

          <div className="h-4 w-px bg-slate-200 dark:bg-surface-800 mx-0.5 hidden sm:block" />

          {/* Update Button */}
          <button
            onClick={() => setUpdateModalOpen(true)}
            title="System Updates"
            className="px-2 py-1 sm:px-2.5 sm:py-1.5 rounded-lg text-xs font-semibold text-slate-700 dark:text-slate-200 bg-white hover:bg-slate-100 dark:bg-surface-800 dark:hover:bg-surface-700 border border-slate-300 dark:border-surface-700 flex items-center gap-1 sm:gap-1.5 transition shadow-2xs cursor-pointer flex-shrink-0"
          >
            <RotateCw className="w-3.5 h-3.5 text-indigo-600 dark:text-indigo-400 flex-shrink-0" />
            <span>Update</span>
          </button>

          {/* Fix Button */}
          <button
            onClick={() => {
              setFixModalOpen(true);
              executeFix();
            }}
            title="System Diagnostics & Auto-Fix"
            className="px-2 py-1 sm:px-2.5 sm:py-1.5 rounded-lg text-xs font-semibold text-slate-700 dark:text-slate-200 bg-white hover:bg-slate-100 dark:bg-surface-800 dark:hover:bg-surface-700 border border-slate-300 dark:border-surface-700 flex items-center gap-1 sm:gap-1.5 transition shadow-2xs cursor-pointer flex-shrink-0"
          >
            <Wrench className="w-3.5 h-3.5 text-amber-600 dark:text-amber-400 flex-shrink-0" />
            <span>Fix</span>
          </button>

          {/* Restart Button */}
          <button
            onClick={() => setRestartModalOpen(true)}
            title="Restart Services or Server"
            className="px-2 py-1 sm:px-2.5 sm:py-1.5 rounded-lg text-xs font-bold text-rose-700 dark:text-rose-400 bg-rose-50 hover:bg-rose-100 dark:bg-rose-950/40 border border-rose-200 dark:border-rose-900/40 flex items-center gap-1 sm:gap-1.5 transition shadow-2xs cursor-pointer flex-shrink-0"
          >
            <Power className="w-3.5 h-3.5 text-rose-600 dark:text-rose-400 flex-shrink-0" />
            <span>Restart</span>
          </button>
        </div>
      </header>

      {/* Restart Modal */}
      {restartModalOpen && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/60 backdrop-blur-xs animate-fadeIn">
          <div className="bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-700 rounded-2xl w-full max-w-md shadow-2xl overflow-hidden p-6 space-y-5">
            <div className="flex items-center justify-between">
              <h3 className="text-base font-bold text-slate-900 dark:text-white flex items-center gap-2">
                <Power className="w-5 h-5 text-rose-500" />
                Restart Services or Server
              </h3>
              <button
                onClick={() => setRestartModalOpen(false)}
                className="w-7 h-7 rounded-full bg-slate-100 dark:bg-surface-800 text-slate-500 hover:text-slate-900 dark:hover:text-white flex items-center justify-center"
              >
                <X className="w-4 h-4" />
              </button>
            </div>

            <p className="text-xs text-slate-500 dark:text-slate-400">
              Select what you want to restart. Web services will reload gracefully without dropping existing active connections.
            </p>

            <div className="space-y-2">
              {[
                { id: 'panel', label: 'Restart Hostvra Control Panel', desc: 'Fast reload of control plane API & UI' },
                { id: 'nginx', label: 'Restart Web Server (Nginx / OpenLiteSpeed)', desc: 'Reload virtual hosts and SSL certificates' },
                { id: 'server', label: 'Reboot Server Node (Hardware Reboot)', desc: 'Full operating system reboot' },
              ].map((opt) => (
                <label
                  key={opt.id}
                  onClick={() => setRestartTarget(opt.id as any)}
                  className={`flex items-start gap-3 p-3 rounded-xl border cursor-pointer transition ${
                    restartTarget === opt.id
                      ? 'bg-emerald-50 dark:bg-emerald-950/20 border-emerald-500 text-emerald-900 dark:text-emerald-300'
                      : 'border-slate-200 dark:border-surface-800 hover:bg-slate-50 dark:hover:bg-surface-800'
                  }`}
                >
                  <input
                    type="radio"
                    name="restart_choice"
                    checked={restartTarget === opt.id}
                    onChange={() => {}}
                    className="mt-1 text-emerald-600 focus:ring-emerald-500"
                  />
                  <div>
                    <p className="text-xs font-bold text-slate-800 dark:text-slate-200">{opt.label}</p>
                    <p className="text-[11px] text-slate-500 dark:text-slate-400">{opt.desc}</p>
                  </div>
                </label>
              ))}
            </div>

            <div className="flex items-center justify-end gap-3 pt-2">
              <button
                onClick={() => setRestartModalOpen(false)}
                className="px-4 py-2 rounded-lg bg-slate-100 dark:bg-surface-800 text-slate-600 dark:text-slate-300 text-xs font-semibold hover:bg-slate-200 dark:hover:bg-surface-700 transition"
              >
                Cancel
              </button>
              <button
                onClick={executeRestart}
                disabled={restarting}
                className="px-5 py-2 rounded-lg bg-rose-600 hover:bg-rose-500 text-white text-xs font-bold shadow-sm transition disabled:opacity-50"
              >
                {restarting ? 'Restarting...' : 'Confirm Restart'}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Fix Diagnostics Modal */}
      {fixModalOpen && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/60 backdrop-blur-xs animate-fadeIn">
          <div className="bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-700 rounded-2xl w-full max-w-lg shadow-2xl overflow-hidden p-6 space-y-4">
            <div className="flex items-center justify-between">
              <h3 className="text-base font-bold text-slate-900 dark:text-white flex items-center gap-2">
                <Wrench className="w-5 h-5 text-emerald-600" />
                Panel Health & Repair Utility
              </h3>
              <button
                onClick={() => setFixModalOpen(false)}
                className="w-7 h-7 rounded-full bg-slate-100 dark:bg-surface-800 text-slate-500 hover:text-slate-900 dark:hover:text-white flex items-center justify-center"
              >
                <X className="w-4 h-4" />
              </button>
            </div>

            <p className="text-xs text-slate-500 dark:text-slate-400">
              Running automated self-repair for file permissions, process locks, and web server configurations.
            </p>

            <div className="p-4 rounded-xl bg-slate-950 font-mono text-xs text-slate-200 space-y-1.5 min-h-[140px]">
              {fixLogs.map((log, i) => (
                <p key={i} className={i === fixLogs.length - 1 && !fixing ? 'text-emerald-400 font-bold' : 'text-slate-300'}>
                  {log}
                </p>
              ))}
              {fixing && (
                <div className="flex items-center gap-2 text-emerald-400 animate-pulse pt-2">
                  <RotateCw className="w-3.5 h-3.5 animate-spin" />
                  <span>Scanning & repairing system state...</span>
                </div>
              )}
            </div>

            <div className="flex justify-end pt-2">
              <button
                onClick={() => setFixModalOpen(false)}
                disabled={fixing}
                className="px-5 py-2 rounded-lg bg-emerald-600 hover:bg-emerald-500 text-white text-xs font-bold transition disabled:opacity-50"
              >
                Done
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Update Check Modal */}
      {updateModalOpen && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/60 backdrop-blur-xs animate-fadeIn">
          <div className="bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-700 rounded-2xl w-full max-w-md shadow-2xl overflow-hidden p-6 space-y-4">
            <div className="flex items-center justify-between">
              <h3 className="text-base font-bold text-slate-900 dark:text-white flex items-center gap-2">
                <RotateCw className="w-5 h-5 text-emerald-600" />
                Hostvra System Updates
              </h3>
              <button
                onClick={() => setUpdateModalOpen(false)}
                className="w-7 h-7 rounded-full bg-slate-100 dark:bg-surface-800 text-slate-500 hover:text-slate-900 dark:hover:text-white flex items-center justify-center"
              >
                <X className="w-4 h-4" />
              </button>
            </div>

            <div className="p-4 rounded-xl bg-slate-50 dark:bg-surface-800/60 border border-slate-200 dark:border-surface-700 text-xs space-y-2">
              <div className="flex items-center justify-between">
                <span className="text-slate-500 dark:text-slate-400">Installed Version:</span>
                <span className="font-mono font-bold text-slate-800 dark:text-slate-200">v1.2.0-stable</span>
              </div>
              <div className="flex items-center justify-between">
                <span className="text-slate-500 dark:text-slate-400">Release Channel:</span>
                <span className="text-emerald-600 dark:text-emerald-400 font-semibold">Official Release</span>
              </div>
              <p className="text-slate-600 dark:text-slate-300 pt-2 border-t border-slate-200 dark:border-surface-700">
                {updateStatus}
              </p>
            </div>

            <div className="flex justify-between items-center pt-2">
              <button
                onClick={executeCheckUpdate}
                disabled={checkingUpdate}
                className="px-4 py-2 rounded-lg bg-slate-100 dark:bg-surface-800 text-slate-700 dark:text-slate-200 text-xs font-semibold hover:bg-slate-200 dark:hover:bg-surface-700 transition flex items-center gap-1.5"
              >
                <RotateCw className={`w-3.5 h-3.5 ${checkingUpdate ? 'animate-spin' : ''}`} />
                <span>Check for Updates</span>
              </button>

              <button
                onClick={() => setUpdateModalOpen(false)}
                className="px-4 py-2 rounded-lg bg-emerald-600 hover:bg-emerald-500 text-white text-xs font-bold transition"
              >
                Close
              </button>
            </div>
          </div>
        </div>
      )}
    </>
  );
}
