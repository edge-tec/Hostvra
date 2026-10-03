'use client';

import React, { createContext, useContext, useEffect, useState, useCallback, useRef } from 'react';
import { useRouter } from 'next/navigation';
import {
  apiFetch,
  getStoredToken,
  getStoredWebmailToken,
  setStoredWebmailToken,
  clearStoredWebmailAuth,
} from '@/lib/api';

export interface WebmailAccount {
  id: string;
  email: string;
  name: string;
  token?: string;
  quotaBytes?: number;
  usedBytes?: number;
}

export interface ComposeInitialData {
  to?: string;
  cc?: string;
  bcc?: string;
  replyTo?: string;
  subject?: string;
  bodyHTML?: string;
  bodyText?: string;
  replyToMessageId?: string;
  attachments?: Array<{
    id: string;
    filename: string;
    content_type: string;
    size_bytes: number;
  }>;
}

export interface MailFilter {
  id: string;
  mailbox_id: string;
  name: string;
  criteria_field: string;
  criteria_pattern: string;
  action_type: string;
  action_target?: string;
  is_active: boolean;
  priority: number;
}

export interface MailContact {
  id: string;
  name: string;
  email: string;
  phone?: string;
  company?: string;
  group_name?: string;
  notes?: string;
}

export interface MailIdentity {
  id?: string;
  mailbox_id?: string;
  email: string;
  display_name: string;
  reply_to?: string;
  signature_html?: string;
  signature_text?: string;
  is_default: boolean;
}

export interface MailForwardingRule {
  id?: string;
  mailbox_id?: string;
  forward_to: string;
  keep_copy: boolean;
  is_active: boolean;
}

interface WebmailContextType {
  isAuthenticated: boolean;
  isAuthLoading: boolean;
  verifySession: () => Promise<boolean>;
  accounts: WebmailAccount[];
  activeAccount: WebmailAccount | null;
  activeEmail: string;
  folderCounts: Record<string, number>;
  isSyncing: boolean;
  searchQuery: string;
  setSearchQuery: React.Dispatch<React.SetStateAction<string>>;
  switchAccount: (email: string) => void;
  addAccount: (acc: WebmailAccount) => void;
  loginWithCredentials: (email: string, password: string) => Promise<boolean>;
  removeAccount: (email: string) => void;
  logoutCurrentAccount: () => void;
  logoutAllAccounts: () => void;
  refreshFolderCounts: () => Promise<void>;
  // Compose
  isComposeOpen: boolean;
  composeInitial: ComposeInitialData | null;
  openCompose: (initial?: ComposeInitialData) => void;
  closeCompose: () => void;
  // Add Account Modal
  isAddAccountOpen: boolean;
  openAddAccount: () => void;
  closeAddAccount: () => void;
  // Layout & General Preferences
  readingPaneLayout: 'split' | 'full';
  setReadingPaneLayout: (l: 'split' | 'full') => void;
  emailsPerPage: number;
  setEmailsPerPage: (c: number) => void;
  defaultFolder: string;
  setDefaultFolder: (f: string) => void;
  syncInterval: string;
  setSyncInterval: (i: string) => void;
  updatePreferences: (prefs: {
    readingPaneLayout?: 'split' | 'full';
    emailsPerPage?: number;
    defaultFolder?: string;
    syncInterval?: string;
  }) => Promise<boolean>;
  // Audio & Desktop alerts
  soundEnabled: boolean;
  setSoundEnabled: (enabled: boolean) => void;
  toggleSound: () => void;
  desktopNotifications: boolean;
  setDesktopNotifications: (enabled: boolean) => void;
  desktopNotificationsEnabled: boolean;
  requestNotificationPermission: () => Promise<boolean>;
  requestDesktopNotifications: () => Promise<boolean>;
  // Identities
  identities: MailIdentity[];
  fetchIdentities: () => Promise<void>;
  saveIdentity: (identity: MailIdentity) => Promise<boolean>;
  deleteIdentity: (id: string) => Promise<boolean>;
  // Filters
  filters: MailFilter[];
  fetchFilters: () => Promise<void>;
  saveFilter: (filter: Partial<MailFilter>) => Promise<boolean>;
  deleteFilter: (id: string) => Promise<boolean>;
  // Forwarding
  forwardingRule: MailForwardingRule | null;
  fetchForwarding: () => Promise<void>;
  saveForwarding: (forwardTo: string, keepCopy: boolean) => Promise<boolean>;
  deleteForwarding: () => Promise<boolean>;
  // Contacts
  contacts: MailContact[];
  fetchContacts: (q?: string) => Promise<void>;
  saveContact: (contact: Partial<MailContact>) => Promise<boolean>;
  deleteContact: (id: string) => Promise<boolean>;
}

const WebmailContext = createContext<WebmailContextType | null>(null);

const STORAGE_ACCOUNTS_KEY = 'hostvra_webmail_accounts';
const STORAGE_ACTIVE_KEY = 'hostvra_webmail_active_account';
const STORAGE_SOUND_KEY = 'hostvra_webmail_sound_enabled';

// Elegant Web Audio API gentle notification chime (no external asset needed)
function playGentleChime() {
  try {
    const AudioContextClass = window.AudioContext || (window as unknown as { webkitAudioContext: typeof AudioContext }).webkitAudioContext;
    if (!AudioContextClass) return;
    const ctx = new AudioContextClass();
    const now = ctx.currentTime;

    const osc1 = ctx.createOscillator();
    const osc2 = ctx.createOscillator();
    const gain = ctx.createGain();

    osc1.type = 'sine';
    osc1.frequency.setValueAtTime(587.33, now); // D5
    osc1.frequency.exponentialRampToValueAtTime(880.0, now + 0.15); // A5

    osc2.type = 'triangle';
    osc2.frequency.setValueAtTime(880.0, now + 0.15); // A5
    osc2.frequency.exponentialRampToValueAtTime(1174.66, now + 0.35); // D6

    gain.gain.setValueAtTime(0, now);
    gain.gain.linearRampToValueAtTime(0.2, now + 0.05);
    gain.gain.exponentialRampToValueAtTime(0.001, now + 0.6);

    osc1.connect(gain);
    osc2.connect(gain);
    gain.connect(ctx.destination);

    osc1.start(now);
    osc2.start(now + 0.12);
    osc1.stop(now + 0.35);
    osc2.stop(now + 0.65);
  } catch (err) {
    console.debug('Audio chime skipped:', err);
  }
}

export function WebmailProvider({ children }: { children: React.ReactNode }) {
  const router = useRouter();
  const [accounts, setAccounts] = useState<WebmailAccount[]>([]);
  const [activeAccount, setActiveAccount] = useState<WebmailAccount | null>(null);
  const [folderCounts, setFolderCounts] = useState<Record<string, number>>({});
  const [isSyncing, setIsSyncing] = useState<boolean>(false);
  const [searchQuery, setSearchQuery] = useState<string>('');
  
  // Compose modal state
  const [isComposeOpen, setIsComposeOpen] = useState<boolean>(false);
  const [composeInitial, setComposeInitial] = useState<ComposeInitialData | null>(null);
  
  // Add Account modal
  const [isAddAccountOpen, setIsAddAccountOpen] = useState<boolean>(false);

  // Sound and Desktop alerts
  const [soundEnabled, setSoundEnabledState] = useState<boolean>(true);
  const [desktopNotifications, setDesktopNotifications] = useState<boolean>(false);

  // Layout & General Preferences
  const [readingPaneLayout, setReadingPaneLayoutState] = useState<'split' | 'full'>('split');
  const [emailsPerPage, setEmailsPerPageState] = useState<number>(50);
  const [defaultFolder, setDefaultFolderState] = useState<string>('inbox');
  const [syncInterval, setSyncIntervalState] = useState<string>('realtime');

  // Settings & entities state
  const [identities, setIdentities] = useState<MailIdentity[]>([]);
  const [filters, setFilters] = useState<MailFilter[]>([]);
  const [forwardingRule, setForwardingRule] = useState<MailForwardingRule | null>(null);
  const [contacts, setContacts] = useState<MailContact[]>([]);

  const prevUnreadRef = useRef<number>(-1);

  const [isAuthenticated, setIsAuthenticated] = useState<boolean>(false);
  const [isAuthLoading, setIsAuthLoading] = useState<boolean>(true);

  // Strict Webmail session verification from server
  const verifySession = useCallback(async (): Promise<boolean> => {
    try {
      const res = await apiFetch<{
        authenticated: boolean;
        token: string;
        mailbox: {
          id: string;
          email: string;
          name: string;
          quota_bytes: number;
          used_bytes: number;
        };
      }>('/api/v1/webmail/session');

      if (res.data && res.data.authenticated && res.data.mailbox) {
        const mb = res.data.mailbox;
        const token = res.data.token;
        if (token) {
          setStoredWebmailToken(token);
        }
        const acc: WebmailAccount = {
          id: mb.id,
          email: mb.email,
          name: mb.name || mb.email.split('@')[0],
          token: token,
          quotaBytes: mb.quota_bytes,
          usedBytes: mb.used_bytes,
        };
        setActiveAccount(acc);
        let existingAccounts: WebmailAccount[] = [];
        try {
          const raw = localStorage.getItem(STORAGE_ACCOUNTS_KEY);
          if (raw) existingAccounts = JSON.parse(raw);
        } catch {}
        const merged = [acc, ...existingAccounts.filter(a => a.email.toLowerCase() !== acc.email.toLowerCase())];
        setAccounts(merged);
        localStorage.setItem(STORAGE_ACCOUNTS_KEY, JSON.stringify(merged));
        setIsAuthenticated(true);
        return true;
      } else {
        clearStoredWebmailAuth();
        setActiveAccount(null);
        setAccounts([]);
        setIsAuthenticated(false);
        return false;
      }
    } catch {
      clearStoredWebmailAuth();
      setActiveAccount(null);
      setAccounts([]);
      setIsAuthenticated(false);
      return false;
    } finally {
      setIsAuthLoading(false);
    }
  }, []);

  // Initialize and attach bfcache back-button defense
  useEffect(() => {
    if (typeof window === 'undefined') return;

    if ('Notification' in window && Notification.permission === 'granted') {
      setDesktopNotifications(true);
    }

    const soundPref = localStorage.getItem(STORAGE_SOUND_KEY);
    if (soundPref !== null) {
      setSoundEnabledState(soundPref === 'true');
    }

    verifySession();

    // Browser back button defense: revalidate session when page is shown from bfcache
    const handlePageShow = (event: PageTransitionEvent) => {
      if (event.persisted) {
        verifySession();
      }
    };

    window.addEventListener('pageshow', handlePageShow);
    return () => {
      window.removeEventListener('pageshow', handlePageShow);
    };
  }, [verifySession]);

  // Update document title with unread badge e.g. "(3) Hostvra Webmail"
  useEffect(() => {
    if (typeof document === 'undefined') return;
    const unread = folderCounts['inboxUnread'] || 0;
    if (unread > 0) {
      document.title = `(${unread}) Hostvra Webmail - ${activeAccount?.email || ''}`;
    } else {
      document.title = activeAccount ? `Hostvra Webmail - ${activeAccount.email}` : 'Hostvra Webmail';
    }
  }, [folderCounts, activeAccount]);

  // Refresh counts
  const refreshFolderCounts = useCallback(async () => {
    if (!activeAccount) return;
    setIsSyncing(true);
    try {
      const res = await apiFetch<Record<string, number>>(
        `/api/v1/webmail/counts?mailbox_id=${activeAccount.id}&account_email=${encodeURIComponent(activeAccount.email)}`
      );
      if (res.data) {
        setFolderCounts(res.data);
      }
    } catch (err) {
      console.error('Failed refreshing folder counts:', err);
    } finally {
      setIsSyncing(false);
    }
  }, [activeAccount]);

  // Periodic refresh & SSE
  useEffect(() => {
    if (!activeAccount) return;
    refreshFolderCounts();

    const interval = setInterval(() => {
      refreshFolderCounts();
    }, 45000);

    let eventSource: EventSource | null = null;
    try {
      const sseUrl = `/api/v1/webmail/events?mailbox_id=${activeAccount.id}&account_email=${encodeURIComponent(activeAccount.email)}`;
      eventSource = new EventSource(sseUrl);

      eventSource.addEventListener('message_count', (e: MessageEvent) => {
        try {
          const counts = JSON.parse(e.data);
          setFolderCounts(counts);
          const newInboxUnread = counts.inboxUnread || 0;
          if (prevUnreadRef.current !== -1 && newInboxUnread > prevUnreadRef.current) {
            if (soundEnabled) {
              playGentleChime();
            }
            if ('Notification' in window && Notification.permission === 'granted') {
              new Notification(`New email for ${activeAccount.email}`, {
                body: `You have new messages in your inbox.`,
                icon: '/favicon.ico',
              });
            }
          }
          prevUnreadRef.current = newInboxUnread;
        } catch (err) {
          console.error('Failed parsing SSE message_count:', err);
        }
      });

      eventSource.addEventListener('new_mail', (e: MessageEvent) => {
        try {
          const data = JSON.parse(e.data);
          if (soundEnabled) {
            playGentleChime();
          }
          if ('Notification' in window && Notification.permission === 'granted') {
            new Notification(`New Email: ${data.subject || 'No Subject'}`, {
              body: `From: ${data.from || 'Unknown Sender'}`,
              icon: '/favicon.ico',
            });
          }
          refreshFolderCounts();
        } catch (err) {
          console.error('Failed parsing new_mail event:', err);
        }
      });
    } catch (err) {
      console.debug('SSE connection skipped or unsupported:', err);
    }

    return () => {
      clearInterval(interval);
      if (eventSource) {
        eventSource.close();
      }
    };
  }, [activeAccount, refreshFolderCounts, soundEnabled]);

  // Account switching
  const switchAccount = useCallback((email: string) => {
    const target = accounts.find(a => a.email.toLowerCase() === email.toLowerCase());
    if (target) {
      if (target.token) {
        setStoredWebmailToken(target.token);
      }
      setActiveAccount(target);
      localStorage.setItem(STORAGE_ACTIVE_KEY, target.email);
      prevUnreadRef.current = -1;
    }
  }, [accounts]);

  // Add account object
  const addAccount = useCallback((newAcc: WebmailAccount) => {
    setAccounts(prev => {
      const filtered = prev.filter(a => a.email.toLowerCase() !== newAcc.email.toLowerCase());
      const updated = [...filtered, newAcc];
      localStorage.setItem(STORAGE_ACCOUNTS_KEY, JSON.stringify(updated));
      return updated;
    });
    setActiveAccount(newAcc);
    if (newAcc.token) {
      setStoredWebmailToken(newAcc.token);
    }
    localStorage.setItem(STORAGE_ACTIVE_KEY, newAcc.email);
    setIsAddAccountOpen(false);
  }, []);

  // Login with credentials directly
  const loginWithCredentials = useCallback(async (email: string, password: string): Promise<boolean> => {
    try {
      const res = await apiFetch<{
        token: string;
        mailbox: {
          id: string;
          email: string;
          name: string;
          quota_bytes: number;
          used_bytes: number;
        };
      }>('/api/v1/webmail/auth', {
        method: 'POST',
        body: JSON.stringify({ email: email.trim().toLowerCase(), password }),
      });

      if (res.data && res.data.mailbox && res.data.token) {
        setStoredWebmailToken(res.data.token);
        const newAcc: WebmailAccount = {
          id: res.data.mailbox.id,
          email: res.data.mailbox.email,
          name: res.data.mailbox.name || res.data.mailbox.email.split('@')[0],
          token: res.data.token,
          quotaBytes: res.data.mailbox.quota_bytes,
          usedBytes: res.data.mailbox.used_bytes,
        };
        addAccount(newAcc);
        setIsAuthenticated(true);
        return true;
      }
      return false;
    } catch (err) {
      console.error('Login error:', err);
      return false;
    }
  }, [addAccount]);

  // Remove saved account
  const removeAccount = useCallback((email: string) => {
    setAccounts(prev => {
      const updated = prev.filter(a => a.email.toLowerCase() !== email.toLowerCase());
      localStorage.setItem(STORAGE_ACCOUNTS_KEY, JSON.stringify(updated));
      if (activeAccount?.email.toLowerCase() === email.toLowerCase()) {
        const next = updated[0] || null;
        setActiveAccount(next);
        if (next) {
          localStorage.setItem(STORAGE_ACTIVE_KEY, next.email);
          if (next.token) {
            setStoredWebmailToken(next.token);
          }
        } else {
          localStorage.removeItem(STORAGE_ACTIVE_KEY);
          clearStoredWebmailAuth();
        }
      }
      return updated;
    });
  }, [activeAccount]);

  // Layout & General Preference Setters
  const setReadingPaneLayout = useCallback((l: 'split' | 'full') => {
    setReadingPaneLayoutState(l);
    localStorage.setItem('hostvra_webmail_reading_pane_layout', l);
  }, []);

  const setEmailsPerPage = useCallback((c: number) => {
    setEmailsPerPageState(c);
    localStorage.setItem('hostvra_webmail_page_size', String(c));
  }, []);

  const setDefaultFolder = useCallback((f: string) => {
    setDefaultFolderState(f);
    localStorage.setItem('hostvra_webmail_default_folder', f);
  }, []);

  const setSyncInterval = useCallback((i: string) => {
    setSyncIntervalState(i);
    localStorage.setItem('hostvra_webmail_sync_interval', i);
  }, []);

  const updatePreferences = useCallback(async (prefs: {
    readingPaneLayout?: 'split' | 'full';
    emailsPerPage?: number;
    defaultFolder?: string;
    syncInterval?: string;
  }): Promise<boolean> => {
    if (prefs.readingPaneLayout) setReadingPaneLayout(prefs.readingPaneLayout);
    if (prefs.emailsPerPage) setEmailsPerPage(prefs.emailsPerPage);
    if (prefs.defaultFolder) setDefaultFolder(prefs.defaultFolder);
    if (prefs.syncInterval) setSyncInterval(prefs.syncInterval);

    if (activeAccount) {
      try {
        await apiFetch('/api/v1/webmail/preferences', {
          method: 'PUT',
          body: JSON.stringify({
            mailbox_id: activeAccount.id,
            page_size: prefs.emailsPerPage || emailsPerPage,
            auto_refresh_seconds: prefs.syncInterval === '60' ? 60 : prefs.syncInterval === '300' ? 300 : 30,
          }),
        });
      } catch (e) {
        console.debug('Failed to sync preferences to backend:', e);
      }
    }
    return true;
  }, [activeAccount, emailsPerPage, setReadingPaneLayout, setEmailsPerPage, setDefaultFolder, setSyncInterval]);

  // Logout all accounts
  const logoutAllAccounts = useCallback(async () => {
    try {
      await apiFetch('/api/v1/webmail/logout', { method: 'POST' });
    } catch (e) {
      console.debug('Webmail backend logout notification skipped:', e);
    }
    clearStoredWebmailAuth();
    setIsAuthenticated(false);
    setActiveAccount(null);
    setAccounts([]);
    setFolderCounts({});
    setContacts([]);
    setFilters([]);
    setIdentities([]);
    setForwardingRule(null);
    if (typeof window !== 'undefined') {
      sessionStorage.clear();
    }
    router.push('/webmail/login');
  }, [router]);

  // Logout current account
  const logoutCurrentAccount = useCallback(() => {
    logoutAllAccounts();
  }, [logoutAllAccounts]);

  // Compose actions
  const openCompose = useCallback((initial?: ComposeInitialData) => {
    setComposeInitial(initial || null);
    setIsComposeOpen(true);
  }, []);

  const closeCompose = useCallback(() => {
    setIsComposeOpen(false);
    setComposeInitial(null);
  }, []);

  // Add Account modal actions
  const openAddAccount = useCallback(() => {
    setIsAddAccountOpen(true);
  }, []);

  const closeAddAccount = useCallback(() => {
    setIsAddAccountOpen(false);
  }, []);

  // Audio chime toggle
  const toggleSound = useCallback(() => {
    setSoundEnabledState(prev => {
      const next = !prev;
      localStorage.setItem(STORAGE_SOUND_KEY, String(next));
      if (next) {
        playGentleChime();
      }
      return next;
    });
  }, []);

  const setSoundEnabled = useCallback((v: boolean) => {
    setSoundEnabledState(v);
    localStorage.setItem(STORAGE_SOUND_KEY, String(v));
    if (v) playGentleChime();
  }, []);

  // Request desktop notification permission
  const requestNotificationPermission = useCallback(async (): Promise<boolean> => {
    if (typeof window === 'undefined' || !('Notification' in window)) return false;
    try {
      const perm = await Notification.requestPermission();
      if (perm === 'granted') {
        setDesktopNotifications(true);
        new Notification('Hostvra Webmail Notifications Active', {
          body: 'You will receive instant desktop alerts for incoming messages.',
          icon: '/favicon.ico',
        });
        return true;
      }
      return false;
    } catch (e) {
      console.error('Error requesting notification permission:', e);
      return false;
    }
  }, []);

  // --- Identities ---
  const fetchIdentities = useCallback(async () => {
    if (!activeAccount) return;
    try {
      const res = await apiFetch<MailIdentity[]>(`/api/v1/webmail/identities?mailbox_id=${activeAccount.id}`);
      if (res.data) setIdentities(res.data);
    } catch (e) {
      console.error('Failed fetching identities:', e);
    }
  }, [activeAccount]);

  const saveIdentity = useCallback(async (identity: MailIdentity): Promise<boolean> => {
    if (!activeAccount) return false;
    try {
      const res = await apiFetch<{ identity: MailIdentity }>(`/api/v1/webmail/identities?mailbox_id=${activeAccount.id}`, {
        method: 'POST',
        body: JSON.stringify(identity),
      });
      if (res.data) {
        await fetchIdentities();
        return true;
      }
      return false;
    } catch (e) {
      console.error('Failed saving identity:', e);
      return false;
    }
  }, [activeAccount, fetchIdentities]);

  const deleteIdentity = useCallback(async (id: string): Promise<boolean> => {
    if (!activeAccount) return false;
    try {
      await apiFetch(`/api/v1/webmail/identities?mailbox_id=${activeAccount.id}&id=${id}`, {
        method: 'DELETE',
      });
      await fetchIdentities();
      return true;
    } catch (e) {
      console.error('Failed deleting identity:', e);
      return false;
    }
  }, [activeAccount, fetchIdentities]);

  // --- Filters ---
  const fetchFilters = useCallback(async () => {
    if (!activeAccount) return;
    try {
      const res = await apiFetch<MailFilter[]>(`/api/v1/webmail/filters?mailbox_id=${activeAccount.id}`);
      if (res.data) setFilters(res.data);
    } catch (e) {
      console.error('Failed fetching filters:', e);
    }
  }, [activeAccount]);

  const saveFilter = useCallback(async (rule: Partial<MailFilter>): Promise<boolean> => {
    if (!activeAccount) return false;
    try {
      const res = await apiFetch<{ filter: MailFilter }>(`/api/v1/webmail/filters?mailbox_id=${activeAccount.id}`, {
        method: 'POST',
        body: JSON.stringify(rule),
      });
      if (res.data) {
        await fetchFilters();
        return true;
      }
      return false;
    } catch (e) {
      console.error('Failed saving filter:', e);
      return false;
    }
  }, [activeAccount, fetchFilters]);

  const deleteFilter = useCallback(async (id: string): Promise<boolean> => {
    if (!activeAccount) return false;
    try {
      await apiFetch(`/api/v1/webmail/filters?mailbox_id=${activeAccount.id}&id=${id}`, {
        method: 'DELETE',
      });
      await fetchFilters();
      return true;
    } catch (e) {
      console.error('Failed deleting filter:', e);
      return false;
    }
  }, [activeAccount, fetchFilters]);

  // --- Forwarding ---
  const fetchForwarding = useCallback(async () => {
    if (!activeAccount) return;
    try {
      const res = await apiFetch<MailForwardingRule>(`/api/v1/webmail/forwarding?mailbox_id=${activeAccount.id}`);
      if (res.data) setForwardingRule(res.data);
    } catch (e) {
      setForwardingRule(null);
    }
  }, [activeAccount]);

  const saveForwarding = useCallback(async (forwardTo: string, keepCopy: boolean): Promise<boolean> => {
    if (!activeAccount) return false;
    try {
      const res = await apiFetch<{ rule: MailForwardingRule }>(`/api/v1/webmail/forwarding?mailbox_id=${activeAccount.id}`, {
        method: 'POST',
        body: JSON.stringify({ forward_to: forwardTo, keep_copy: keepCopy, is_active: true }),
      });
      if (res.data) {
        setForwardingRule(res.data.rule);
        return true;
      }
      return false;
    } catch (e) {
      console.error('Failed saving forwarding:', e);
      return false;
    }
  }, [activeAccount]);

  const deleteForwarding = useCallback(async (): Promise<boolean> => {
    if (!activeAccount) return false;
    try {
      await apiFetch(`/api/v1/webmail/forwarding?mailbox_id=${activeAccount.id}`, {
        method: 'DELETE',
      });
      setForwardingRule(null);
      return true;
    } catch (e) {
      console.error('Failed deleting forwarding:', e);
      return false;
    }
  }, [activeAccount]);

  // --- Contacts ---
  const fetchContacts = useCallback(async (q?: string) => {
    if (!activeAccount) return;
    try {
      const url = q
        ? `/api/v1/webmail/contacts?mailbox_id=${activeAccount.id}&q=${encodeURIComponent(q)}`
        : `/api/v1/webmail/contacts?mailbox_id=${activeAccount.id}`;
      const res = await apiFetch<MailContact[]>(url);
      if (res.data) setContacts(res.data);
    } catch (e) {
      console.error('Failed fetching contacts:', e);
    }
  }, [activeAccount]);

  const saveContact = useCallback(async (contact: Partial<MailContact>): Promise<boolean> => {
    if (!activeAccount) return false;
    try {
      const res = await apiFetch<{ contact: MailContact }>(`/api/v1/webmail/contacts?mailbox_id=${activeAccount.id}`, {
        method: 'POST',
        body: JSON.stringify(contact),
      });
      if (res.data) {
        await fetchContacts();
        return true;
      }
      return false;
    } catch (e) {
      console.error('Failed saving contact:', e);
      return false;
    }
  }, [activeAccount, fetchContacts]);

  const deleteContact = useCallback(async (id: string): Promise<boolean> => {
    if (!activeAccount) return false;
    try {
      await apiFetch(`/api/v1/webmail/contacts?mailbox_id=${activeAccount.id}&id=${id}`, {
        method: 'DELETE',
      });
      await fetchContacts();
      return true;
    } catch (e) {
      console.error('Failed deleting contact:', e);
      return false;
    }
  }, [activeAccount, fetchContacts]);

  return (
    <WebmailContext.Provider
      value={{
        isAuthenticated,
        isAuthLoading,
        verifySession,
        accounts,
        activeAccount,
        activeEmail: activeAccount?.email || '',
        folderCounts,
        isSyncing,
        searchQuery,
        setSearchQuery,
        switchAccount,
        addAccount,
        loginWithCredentials,
        removeAccount,
        logoutCurrentAccount,
        logoutAllAccounts,
        refreshFolderCounts,
        isComposeOpen,
        composeInitial,
        openCompose,
        closeCompose,
        isAddAccountOpen,
        openAddAccount,
        closeAddAccount,
        readingPaneLayout,
        setReadingPaneLayout,
        emailsPerPage,
        setEmailsPerPage,
        defaultFolder,
        setDefaultFolder,
        syncInterval,
        setSyncInterval,
        updatePreferences,
        soundEnabled,
        setSoundEnabled,
        toggleSound,
        desktopNotifications,
        setDesktopNotifications,
        desktopNotificationsEnabled: desktopNotifications,
        requestNotificationPermission,
        requestDesktopNotifications: requestNotificationPermission,
        identities,
        fetchIdentities,
        saveIdentity,
        deleteIdentity,
        filters,
        fetchFilters,
        saveFilter,
        deleteFilter,
        forwardingRule,
        fetchForwarding,
        saveForwarding,
        deleteForwarding,
        contacts,
        fetchContacts,
        saveContact,
        deleteContact,
      }}
    >
      {children}
    </WebmailContext.Provider>
  );
}

export function useWebmail() {
  const ctx = useContext(WebmailContext);
  if (!ctx) {
    throw new Error('useWebmail must be used within a WebmailProvider');
  }
  return ctx;
}
