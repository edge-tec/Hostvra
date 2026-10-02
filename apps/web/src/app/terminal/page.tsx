'use client';

import React, { useState, useEffect, useRef, useCallback } from 'react';
import {
  Terminal as TerminalIcon,
  Trash2,
  Copy,
  Check,
  Server,
  Folder,
  User,
  Maximize2,
  Minimize2,
  Sparkles,
  RefreshCw,
  Sun,
  Moon,
  Zap,
  HelpCircle,
  ChevronDown,
  ChevronUp,
} from 'lucide-react';
import { DashboardShell } from '@/components/DashboardShell';
import { apiFetch, getStoredToken, TerminalInfo } from '@/lib/api';

type TerminalThemeMode = 'white' | 'dark' | 'matrix';

// Define terminal color themes at module scope
const TERMINAL_THEMES = {
  white: {
    background: '#f8fafc',
    foreground: '#0f172a',
    cursor: '#2563eb',
    cursorAccent: '#f8fafc',
    selectionBackground: '#bfdbfe',
    selectionForeground: '#1e3a8a',
    black: '#0f172a',
    red: '#dc2626',
    green: '#16a34a',
    yellow: '#d97706',
    blue: '#2563eb',
    magenta: '#9333ea',
    cyan: '#0891b2',
    white: '#f1f5f9',
    brightBlack: '#64748b',
    brightRed: '#ef4444',
    brightGreen: '#22c55e',
    brightYellow: '#f59e0b',
    brightBlue: '#3b82f6',
    brightMagenta: '#a855f7',
    brightCyan: '#06b6d4',
    brightWhite: '#000000',
  },
  dark: {
    background: '#090d16',
    foreground: '#f8fafc',
    cursor: '#38bdf8',
    cursorAccent: '#090d16',
    selectionBackground: '#1e293b',
    selectionForeground: '#ffffff',
    black: '#1e293b',
    red: '#f87171',
    green: '#4ade80',
    yellow: '#fbbf24',
    blue: '#60a5fa',
    magenta: '#c084fc',
    cyan: '#38bdf8',
    white: '#f8fafc',
    brightBlack: '#475569',
    brightRed: '#ef4444',
    brightGreen: '#22c55e',
    brightYellow: '#f59e0b',
    brightBlue: '#3b82f6',
    brightMagenta: '#a855f7',
    brightCyan: '#06b6d4',
    brightWhite: '#ffffff',
  },
  matrix: {
    background: '#040705',
    foreground: '#00ff66',
    cursor: '#00ff66',
    cursorAccent: '#040705',
    selectionBackground: '#003311',
    selectionForeground: '#00ff66',
    black: '#040705',
    red: '#ff3333',
    green: '#00ff66',
    yellow: '#ffcc00',
    blue: '#3399ff',
    magenta: '#cc33ff',
    cyan: '#00ffff',
    white: '#ffffff',
    brightBlack: '#1a3320',
    brightRed: '#ff6666',
    brightGreen: '#33ff88',
    brightYellow: '#ffdd33',
    brightBlue: '#66b2ff',
    brightMagenta: '#dd66ff',
    brightCyan: '#66ffff',
    brightWhite: '#ffffff',
  },
};

export type TerminalLifecycleState =
  | 'initializing'
  | 'connecting'
  | 'allocating'
  | 'ready'
  | 'disconnected'
  | 'error'
  | 'closed';

export default function TerminalPage() {
  const [info, setInfo] = useState<TerminalInfo | null>(null);
  const [terminalState, setTerminalState] = useState<TerminalLifecycleState>('initializing');
  const [isFullscreen, setIsFullscreen] = useState(false);
  const [themeMode, setThemeMode] = useState<TerminalThemeMode>('dark');
  const [fontSize, setFontSize] = useState<number>(14);
  const [copiedNotification, setCopiedNotification] = useState(false);
  const [showShortcuts, setShowShortcuts] = useState(false);
  const [disconnectInfo, setDisconnectInfo] = useState<{
    title: string;
    description: string;
    type: 'shell_exit' | 'network_drop' | 'auth_expired' | 'error';
    exitCode?: number;
  } | null>(null);

  const terminalContainerNodeRef = useRef<HTMLDivElement | null>(null);
  const [containerMounted, setContainerMounted] = useState(false);
  const xtermInstanceRef = useRef<any>(null);
  const fitAddonRef = useRef<any>(null);
  const wsRef = useRef<WebSocket | null>(null);
  const pingIntervalRef = useRef<NodeJS.Timeout | null>(null);
  const themeModeRef = useRef(themeMode);
  themeModeRef.current = themeMode;
  const fontSizeRef = useRef(fontSize);
  fontSizeRef.current = fontSize;

  const setTerminalContainerRef = useCallback((node: HTMLDivElement | null) => {
    if (node) {
      console.log('[TERMINAL] DOM container attached');
      terminalContainerNodeRef.current = node;
      setContainerMounted(true);
    } else {
      terminalContainerNodeRef.current = null;
      setContainerMounted(false);
    }
  }, []);

  // Fetch system environment metadata
  useEffect(() => {
    async function loadInfo() {
      try {
        const res = await apiFetch<TerminalInfo>('/api/v1/terminal/info');
        if (res.success && res.data) {
          setInfo(res.data);
        }
      } catch (err) {
        console.warn('Could not load terminal metadata:', err);
      }
    }
    loadInfo();
  }, []);

  // Initialize xterm and WebSocket connection
  const initTerminal = useCallback(async () => {
    const container = terminalContainerNodeRef.current;
    if (!container) {
      console.warn('[TERMINAL] Container ref not yet available');
      return;
    }

    console.log('[TERMINAL] Component mounted, starting terminal initialization...');

    // Cleanup previous instance if any
    if (pingIntervalRef.current) {
      clearInterval(pingIntervalRef.current);
      pingIntervalRef.current = null;
    }
    if (wsRef.current) {
      try {
        wsRef.current.close();
      } catch {}
      wsRef.current = null;
    }
    if (xtermInstanceRef.current) {
      try {
        xtermInstanceRef.current.dispose();
      } catch {}
      xtermInstanceRef.current = null;
    }
    container.innerHTML = '';

    setTerminalState('initializing');
    setDisconnectInfo(null);

    try {
      // Dynamically load @xterm/xterm and @xterm/addon-fit for SSR safety
      const xtermMod: any = await import('@xterm/xterm');
      const TerminalClass = xtermMod.Terminal || (xtermMod.default && (xtermMod.default.Terminal || xtermMod.default));

      const fitMod: any = await import('@xterm/addon-fit');
      const FitAddonClass = fitMod.FitAddon || (fitMod.default && (fitMod.default.FitAddon || fitMod.default));

      if (!TerminalClass || !FitAddonClass) {
        throw new Error('Terminal or FitAddon module could not be initialized');
      }

      const fitAddon = new FitAddonClass();
      fitAddonRef.current = fitAddon;

      const currentTheme = TERMINAL_THEMES[themeModeRef.current] || TERMINAL_THEMES.dark;
      const term = new TerminalClass({
        cursorBlink: true,
        cursorStyle: 'block',
        fontSize: fontSizeRef.current,
        fontFamily: 'JetBrains Mono, Menlo, Monaco, Consolas, "Liberation Mono", "Courier New", monospace',
        theme: currentTheme,
        convertEol: true,
        allowProposedApi: true,
        scrollback: 10000,
      });

      term.loadAddon(fitAddon);
      term.open(container);
      xtermInstanceRef.current = term;
      console.log('[TERMINAL] xterm initialized and attached to DOM');

      // Small delay for DOM layout before initial fit
      setTimeout(() => {
        try {
          fitAddon.fit();
        } catch {
          // Container sizing grace period
        }
      }, 50);

      // Resolve WebSocket connection endpoint and authentication token
      const isHttps = typeof window !== 'undefined' && window.location.protocol === 'https:';
      const wsProtocol = isHttps ? 'wss:' : 'ws:';
      let host = typeof window !== 'undefined' ? window.location.host : 'localhost:8080';

      if (typeof window !== 'undefined') {
        const isLocal = window.location.hostname === 'localhost' || window.location.hostname === '127.0.0.1';
        if (isLocal) {
          host = `${window.location.hostname}:8080`;
        } else if (window.location.port === '3000') {
          // If remote access on port 3000, strip port 3000 so WebSocket connects through standard Nginx reverse proxy on 80/443
          host = window.location.hostname;
        }
      }

      let token = getStoredToken() || '';
      if (!token && typeof window !== 'undefined') {
        token = localStorage.getItem('hostvra_access_token') || localStorage.getItem('token') || localStorage.getItem('access_token') || '';
        if (!token) {
          const match = document.cookie.match(/(?:hostvra_token|access_token|token)=([^;]+)/);
          if (match) token = decodeURIComponent(match[1]);
        }
      }

      if (!token) {
        console.error('[TERMINAL] Authentication token missing in storage/cookies');
        setTerminalState('error');
        setDisconnectInfo({
          title: 'Authentication Required',
          description: 'Authentication token not found. Please log in to Hostvra to open a terminal.',
          type: 'auth_expired',
        });
        term.write('\x1b[1;31m[Hostvra: Authentication token not found. Please log in to Hostvra.]\x1b[0m\r\n');
        return;
      }

      const queryCwd = typeof window !== 'undefined' ? new URLSearchParams(window.location.search).get('cwd') || '' : '';
      const cwdParam = queryCwd ? `&cwd=${encodeURIComponent(queryCwd)}` : '';
      const wsUrl = `${wsProtocol}//${host}/api/v1/terminal/ws?token=${encodeURIComponent(token)}&rows=${term.rows || 24}&cols=${term.cols || 80}${cwdParam}`;

      console.log('[TERMINAL] WebSocket connecting to:', `${wsProtocol}//${host}/api/v1/terminal/ws`);
      term.write('\x1b[1;36m[Hostvra]\x1b[0m Initializing interactive pseudo-terminal (PTY/TTY)...\r\n');
      term.write(`\x1b[90mConnecting to ${wsProtocol}//${host}/api/v1/terminal/ws ...\x1b[0m\r\n`);
      setTerminalState('connecting');

      const ws = new WebSocket(wsUrl);
      ws.binaryType = 'arraybuffer';
      wsRef.current = ws;

      const connectTimeout = setTimeout(() => {
        if (ws.readyState === WebSocket.CONNECTING) {
          console.warn('[TERMINAL] WebSocket connecting timeout watchdog triggered');
          term.write('\r\n\x1b[1;33m[Hostvra: Connection taking longer than expected...]\x1b[0m\r\n');
          term.write('\x1b[90mEnsure hostvra-api service is running: systemctl status hostvra-api\x1b[0m\r\n');
          setTerminalState('error');
          setDisconnectInfo({
            title: 'Connection Timed Out',
            description: 'Could not connect to Hostvra API. Please verify hostvra-api service is running.',
            type: 'error',
          });
        }
      }, 7000);

      ws.onopen = () => {
        console.log('[TERMINAL] WebSocket connected, PTY allocation started');
        setTerminalState('allocating');
        term.write('\x1b[90mWebSocket handshake verified. Allocating Linux PTY session...\x1b[0m\r\n');
        try {
          fitAddon.fit();
          ws.send(JSON.stringify({ type: 'resize', cols: term.cols, rows: term.rows }));
        } catch (err) {
          console.warn('Failed initial terminal resize:', err);
        }
      };

      ws.onmessage = async (event: MessageEvent) => {
        if (typeof event.data === 'string') {
          if (event.data.startsWith('{')) {
            try {
              const msg = JSON.parse(event.data);
              if (msg.type === 'ready') {
                clearTimeout(connectTimeout);
                console.log('[TERMINAL] PTY allocated & shell started - READY:', msg);
                setTerminalState('ready');
                setDisconnectInfo(null);
                term.write('\x1b[1;32m[Hostvra]\x1b[0m Interactive shell ready.\r\n\r\n');
                try {
                  fitAddon.fit();
                  ws.send(JSON.stringify({ type: 'resize', cols: term.cols, rows: term.rows }));
                } catch {}
                term.focus();
                console.log('[TERMINAL] terminal input enabled');

                // Start client-side heartbeat to ping every 20 seconds during idle periods
                if (pingIntervalRef.current) clearInterval(pingIntervalRef.current);
                pingIntervalRef.current = setInterval(() => {
                  if (wsRef.current && wsRef.current.readyState === WebSocket.OPEN) {
                    try {
                      wsRef.current.send(JSON.stringify({ type: 'ping' }));
                    } catch {}
                  }
                }, 20000);

                return;
              }
              if (msg.type === 'error') {
                clearTimeout(connectTimeout);
                console.error('[TERMINAL] Backend error:', msg);
                setTerminalState('error');
                setDisconnectInfo({
                  title: 'PTY Allocation Error',
                  description: msg.message || 'PTY allocation failed on server.',
                  type: 'error',
                });
                term.write(`\r\n\x1b[1;31m[Hostvra Error: ${msg.message || 'PTY allocation failed'}]\x1b[0m\r\n`);
                return;
              }
              if (msg.type === 'exit') {
                clearTimeout(connectTimeout);
                if (pingIntervalRef.current) {
                  clearInterval(pingIntervalRef.current);
                  pingIntervalRef.current = null;
                }
                const exitCode = typeof msg.exit_code === 'number' ? msg.exit_code : 0;
                console.log('[TERMINAL] Shell process exited:', exitCode);
                setTerminalState('closed');
                setDisconnectInfo({
                  title: 'Shell Process Ended',
                  description: `The interactive Linux shell session ended with exit status code ${exitCode}.`,
                  type: 'shell_exit',
                  exitCode,
                });
                term.write(`\r\n\x1b[33m[Hostvra: Shell process exited with code ${exitCode}]\x1b[0m\r\n`);
                return;
              }
              if (msg.type === 'pong') {
                // Heartbeat response verified
                return;
              }
            } catch {
              // Not a control message, write raw string
            }
          }
          term.write(event.data);
        } else if (event.data instanceof ArrayBuffer) {
          term.write(new Uint8Array(event.data));
        } else if (event.data instanceof Blob) {
          const buf = await event.data.arrayBuffer();
          term.write(new Uint8Array(buf));
        }
      };

      ws.onerror = (err) => {
        clearTimeout(connectTimeout);
        if (pingIntervalRef.current) {
          clearInterval(pingIntervalRef.current);
          pingIntervalRef.current = null;
        }
        console.error('[TERMINAL] WebSocket error:', err);
        setDisconnectInfo({
          title: 'Connection Error',
          description: 'Unable to reach the Hostvra terminal service. Please ensure hostvra-api is active.',
          type: 'error',
        });
        term.write('\r\n\x1b[1;31m[Hostvra: WebSocket connection error]\x1b[0m\r\n');
        term.write('\x1b[90mEnsure hostvra-api is active: systemctl restart hostvra-api\x1b[0m\r\n');
        setTerminalState('error');
      };

      ws.onclose = (event) => {
        clearTimeout(connectTimeout);
        if (pingIntervalRef.current) {
          clearInterval(pingIntervalRef.current);
          pingIntervalRef.current = null;
        }
        console.log('[TERMINAL] WebSocket closed:', event.code, event.reason);
        if (event.code === 1008 || (event.reason && event.reason.includes('Unauthorized'))) {
          term.write('\r\n\x1b[1;31m[Hostvra: Authentication failed (HTTP 401). Please re-login to Hostvra]\x1b[0m\r\n');
          setDisconnectInfo({
            title: 'Authentication Required',
            description: 'Your login session has expired or is unauthorized. Please re-login to Hostvra.',
            type: 'auth_expired',
          });
          setTerminalState('error');
        } else {
          setTerminalState((prev) => {
            if (prev === 'closed') {
              return prev;
            }
            setDisconnectInfo({
              title: 'Connection Interrupted',
              description: 'The terminal connection was interrupted. The server may have restarted or network was dropped.',
              type: 'network_drop',
            });
            return 'disconnected';
          });
          term.write('\r\n\x1b[33m[Hostvra: Terminal session closed]\x1b[0m\r\n');
        }
      };

      // Forward terminal keyboard input directly to PTY stdin
      term.onData((data: string) => {
        if (ws.readyState === WebSocket.OPEN) {
          ws.send(data);
        }
      });

      // Notify backend PTY of window dimension changes
      term.onResize(({ cols, rows }: { cols: number; rows: number }) => {
        if (ws.readyState === WebSocket.OPEN) {
          ws.send(JSON.stringify({ type: 'resize', cols, rows }));
        }
      });

      // Handle Copy / Paste keystrokes seamlessly
      term.attachCustomKeyEventHandler((e: KeyboardEvent) => {
        // Ctrl+Shift+C or Cmd+C when text is selected -> Copy selection
        if ((e.ctrlKey || e.metaKey) && e.key === 'c' && term.hasSelection()) {
          navigator.clipboard.writeText(term.getSelection());
          return false;
        }
        // Ctrl+Shift+V or Cmd+V -> Paste from clipboard
        if ((e.ctrlKey || e.metaKey) && e.key === 'v') {
          navigator.clipboard.readText().then((text) => {
            if (text && ws.readyState === WebSocket.OPEN) {
              ws.send(text);
            }
          });
          return false;
        }
        return true;
      });

      // Focus terminal
      setTimeout(() => {
        try {
          fitAddon.fit();
          term.focus();
        } catch {
          // ignore
        }
      }, 150);
    } catch (err: any) {
      console.error('[TERMINAL] Failed to initialize terminal emulator:', err);
      setTerminalState('error');
      if (container) {
        container.innerHTML = `
          <div class="p-6 text-center text-rose-500 font-mono text-sm">
            <p class="font-bold mb-2">Failed to initialize terminal emulator</p>
            <p class="text-xs text-slate-500">${err?.message || 'Unknown error'}</p>
          </div>
        `;
      }
    }
  }, []);

  // Trigger terminal initialization whenever container mounts into the DOM
  useEffect(() => {
    if (containerMounted && terminalContainerNodeRef.current) {
      initTerminal();
    }

    const handleWindowResize = () => {
      if (fitAddonRef.current) {
        try {
          fitAddonRef.current.fit();
        } catch {
          // ignore
        }
      }
    };

    const handleVisibilityChange = () => {
      if (document.visibilityState === 'visible') {
        if (wsRef.current && wsRef.current.readyState === WebSocket.OPEN) {
          try {
            wsRef.current.send(JSON.stringify({ type: 'ping' }));
            fitAddonRef.current?.fit();
            xtermInstanceRef.current?.focus();
          } catch {
            // ignore
          }
        }
      }
    };

    window.addEventListener('resize', handleWindowResize);
    document.addEventListener('visibilitychange', handleVisibilityChange);

    return () => {
      window.removeEventListener('resize', handleWindowResize);
      document.removeEventListener('visibilitychange', handleVisibilityChange);
      if (pingIntervalRef.current) {
        clearInterval(pingIntervalRef.current);
        pingIntervalRef.current = null;
      }
      if (wsRef.current) {
        try {
          wsRef.current.close();
        } catch {}
        wsRef.current = null;
      }
      if (xtermInstanceRef.current) {
        try {
          xtermInstanceRef.current.dispose();
        } catch {}
        xtermInstanceRef.current = null;
      }
    };
  }, [containerMounted, initTerminal]);

  // Update theme dynamically
  useEffect(() => {
    if (xtermInstanceRef.current) {
      xtermInstanceRef.current.options.theme = TERMINAL_THEMES[themeMode];
    }
  }, [themeMode]);

  // Update font size dynamically
  useEffect(() => {
    if (xtermInstanceRef.current && fitAddonRef.current) {
      xtermInstanceRef.current.options.fontSize = fontSize;
      try {
        fitAddonRef.current.fit();
      } catch {
        // ignore
      }
    }
  }, [fontSize]);

  // Quick Command Launcher
  const sendCommand = (cmd: string) => {
    if (wsRef.current && wsRef.current.readyState === WebSocket.OPEN) {
      wsRef.current.send(cmd + '\n');
      xtermInstanceRef.current?.focus();
    }
  };

  // Clear Terminal
  const handleClear = () => {
    if (xtermInstanceRef.current) {
      xtermInstanceRef.current.clear();
      xtermInstanceRef.current.focus();
    }
  };

  // Copy Selection to Clipboard
  const handleCopySelection = () => {
    if (xtermInstanceRef.current) {
      const selection = xtermInstanceRef.current.getSelection();
      if (selection) {
        navigator.clipboard.writeText(selection);
        setCopiedNotification(true);
        setTimeout(() => setCopiedNotification(false), 2000);
      }
    }
  };

  const getStatusBadge = () => {
    switch (terminalState) {
      case 'ready':
        return {
          label: 'Connected (Live PTY)',
          badgeClass:
            'bg-emerald-50 text-emerald-700 border border-emerald-200 dark:bg-emerald-950/40 dark:text-emerald-400 dark:border-emerald-800',
          dotClass: 'bg-emerald-500',
          headerText: 'LIVE PTY',
        };
      case 'allocating':
        return {
          label: 'Allocating PTY...',
          badgeClass:
            'bg-amber-50 text-amber-700 border border-amber-200 dark:bg-amber-950/40 dark:text-amber-400 dark:border-amber-800 animate-pulse',
          dotClass: 'bg-amber-500',
          headerText: 'ALLOCATING PTY',
        };
      case 'connecting':
        return {
          label: 'Connecting to API...',
          badgeClass:
            'bg-amber-50 text-amber-700 border border-amber-200 dark:bg-amber-950/40 dark:text-amber-400 dark:border-amber-800 animate-pulse',
          dotClass: 'bg-amber-500',
          headerText: 'CONNECTING...',
        };
      case 'initializing':
        return {
          label: 'Starting terminal...',
          badgeClass:
            'bg-blue-50 text-blue-700 border border-blue-200 dark:bg-blue-950/40 dark:text-blue-400 dark:border-blue-800 animate-pulse',
          dotClass: 'bg-blue-500',
          headerText: 'STARTING...',
        };
      case 'error':
        return {
          label: 'Connection Error',
          badgeClass:
            'bg-rose-50 text-rose-700 border border-rose-200 dark:bg-rose-950/40 dark:text-rose-400 dark:border-rose-800',
          dotClass: 'bg-rose-500',
          headerText: 'ERROR',
        };
      case 'closed':
        return {
          label: 'Session Closed',
          badgeClass:
            'bg-slate-100 text-slate-700 border border-slate-300 dark:bg-slate-800 dark:text-slate-400 dark:border-slate-700',
          dotClass: 'bg-slate-400',
          headerText: 'CLOSED',
        };
      case 'disconnected':
      default:
        return {
          label: 'Disconnected',
          badgeClass:
            'bg-rose-50 text-rose-700 border border-rose-200 dark:bg-rose-950/40 dark:text-rose-400 dark:border-rose-800',
          dotClass: 'bg-rose-500',
          headerText: 'OFFLINE',
        };
    }
  };

  const statusBadge = getStatusBadge();

  return (
    <DashboardShell>
      <div className={`space-y-4 ${isFullscreen ? 'fixed inset-0 z-50 bg-white dark:bg-slate-950 p-4' : 'pb-16'}`}>
        {/* Terminal Header & Server Information */}
        <div className="flex flex-col lg:flex-row lg:items-center lg:justify-between gap-4 bg-white dark:bg-slate-900 p-5 rounded-2xl border border-slate-200 dark:border-slate-800 shadow-sm">
          <div className="flex items-center gap-3">
            <div className="w-10 h-10 rounded-xl bg-blue-50 dark:bg-blue-950/50 border border-blue-200 dark:border-blue-900 flex items-center justify-center text-blue-600 dark:text-blue-400">
              <TerminalIcon className="w-5 h-5" />
            </div>
            <div>
              <div className="flex items-center gap-2">
                <h1 className="text-lg font-bold text-slate-900 dark:text-white">Web Terminal</h1>
                {/* Real-time Dynamic Connection Status Badge */}
                <div
                  className={`inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full text-xs font-semibold ${statusBadge.badgeClass}`}
                >
                  <span className={`w-2 h-2 rounded-full ${statusBadge.dotClass}`} />
                  <span>{statusBadge.label}</span>
                </div>
              </div>
              <p className="text-xs text-slate-500 dark:text-slate-400 mt-0.5">
                Full-duplex Linux pseudo-terminal with interactive ANSI, nano, vim, top, and signals.
              </p>
            </div>
          </div>

          {/* Node Metadata & Controls */}
          <div className="flex flex-wrap items-center gap-2">
            {info && (
              <div className="hidden sm:flex items-center gap-3 px-3 py-1.5 rounded-xl bg-slate-50 dark:bg-slate-800/60 border border-slate-200 dark:border-slate-700 text-xs text-slate-600 dark:text-slate-300">
                <div className="flex items-center gap-1">
                  <Server className="w-3.5 h-3.5 text-blue-500" />
                  <span className="font-semibold">{info.hostname}</span>
                </div>
                <div className="flex items-center gap-1">
                  <User className="w-3.5 h-3.5 text-emerald-500" />
                  <span>{info.user}</span>
                </div>
                <div className="flex items-center gap-1">
                  <Folder className="w-3.5 h-3.5 text-amber-500" />
                  <span>{info.shell}</span>
                </div>
              </div>
            )}

            {/* Theme Selector */}
            <div className="flex items-center rounded-xl border border-slate-200 dark:border-slate-700 p-0.5 bg-slate-50 dark:bg-slate-800">
              <button
                type="button"
                onClick={() => setThemeMode('white')}
                title="Light White Theme"
                className={`px-2.5 py-1 text-xs font-semibold rounded-lg transition-all flex items-center gap-1 ${
                  themeMode === 'white'
                    ? 'bg-white text-slate-900 shadow-sm font-bold'
                    : 'text-slate-600 dark:text-slate-400 hover:text-slate-900'
                }`}
              >
                <Sun className="w-3.5 h-3.5 text-amber-500" />
                <span>Light</span>
              </button>
              <button
                type="button"
                onClick={() => setThemeMode('dark')}
                title="Obsidian Dark Theme"
                className={`px-2.5 py-1 text-xs font-semibold rounded-lg transition-all flex items-center gap-1 ${
                  themeMode === 'dark'
                    ? 'bg-slate-900 text-white shadow-sm font-bold'
                    : 'text-slate-600 dark:text-slate-400 hover:text-slate-900'
                }`}
              >
                <Moon className="w-3.5 h-3.5 text-blue-400" />
                <span>Dark</span>
              </button>
              <button
                type="button"
                onClick={() => setThemeMode('matrix')}
                title="Matrix Green Theme"
                className={`px-2.5 py-1 text-xs font-semibold rounded-lg transition-all flex items-center gap-1 ${
                  themeMode === 'matrix'
                    ? 'bg-black text-emerald-400 shadow-sm font-bold border border-emerald-500/40'
                    : 'text-slate-600 dark:text-slate-400 hover:text-slate-900'
                }`}
              >
                <Zap className="w-3.5 h-3.5 text-emerald-500" />
                <span>Matrix</span>
              </button>
            </div>

            {/* Font Size Adjusters */}
            <div className="flex items-center rounded-xl border border-slate-200 dark:border-slate-700 bg-slate-50 dark:bg-slate-800 text-xs font-semibold">
              <button
                type="button"
                onClick={() => setFontSize((prev) => Math.max(11, prev - 1))}
                className="px-2.5 py-1.5 hover:bg-slate-100 dark:hover:bg-slate-700 rounded-l-xl text-slate-600 dark:text-slate-300"
                title="Decrease Font Size"
              >
                A-
              </button>
              <span className="px-2 text-slate-500 dark:text-slate-400 font-mono text-[11px]">{fontSize}px</span>
              <button
                type="button"
                onClick={() => setFontSize((prev) => Math.min(20, prev + 1))}
                className="px-2.5 py-1.5 hover:bg-slate-100 dark:hover:bg-slate-700 rounded-r-xl text-slate-600 dark:text-slate-300"
                title="Increase Font Size"
              >
                A+
              </button>
            </div>

            {/* Copy Selection */}
            <button
              type="button"
              onClick={handleCopySelection}
              className="p-2 rounded-xl border border-slate-200 dark:border-slate-700 bg-slate-50 dark:bg-slate-800 hover:bg-slate-100 dark:hover:bg-slate-700 text-slate-600 dark:text-slate-300 transition-colors"
              title="Copy Selected Text"
            >
              {copiedNotification ? <Check className="w-4 h-4 text-emerald-500" /> : <Copy className="w-4 h-4" />}
            </button>

            {/* Clear Screen */}
            <button
              type="button"
              onClick={handleClear}
              className="p-2 rounded-xl border border-slate-200 dark:border-slate-700 bg-slate-50 dark:bg-slate-800 hover:bg-slate-100 dark:hover:bg-slate-700 text-slate-600 dark:text-slate-300 transition-colors"
              title="Clear Terminal Screen (Ctrl+L)"
            >
              <Trash2 className="w-4 h-4" />
            </button>

            {/* Reconnect Button */}
            <button
              type="button"
              onClick={initTerminal}
              className="p-2 rounded-xl border border-slate-200 dark:border-slate-700 bg-slate-50 dark:bg-slate-800 hover:bg-slate-100 dark:hover:bg-slate-700 text-slate-600 dark:text-slate-300 transition-colors"
              title="Reconnect Terminal Session"
            >
              <RefreshCw className="w-4 h-4" />
            </button>

            {/* Fullscreen Toggle */}
            <button
              type="button"
              onClick={() => {
                setIsFullscreen((prev) => !prev);
                setTimeout(() => fitAddonRef.current?.fit(), 100);
              }}
              className="p-2 rounded-xl border border-slate-200 dark:border-slate-700 bg-slate-50 dark:bg-slate-800 hover:bg-slate-100 dark:hover:bg-slate-700 text-slate-600 dark:text-slate-300 transition-colors"
              title={isFullscreen ? 'Exit Fullscreen' : 'Fullscreen Terminal'}
            >
              {isFullscreen ? <Minimize2 className="w-4 h-4" /> : <Maximize2 className="w-4 h-4" />}
            </button>
          </div>
        </div>

        {/* Real Interactive xterm.js Terminal Container */}
        <div
          className={`relative rounded-2xl border transition-all overflow-hidden shadow-md flex flex-col ${
            themeMode === 'white'
              ? 'bg-[#f8fafc] border-slate-300 shadow-slate-200/50'
              : themeMode === 'matrix'
              ? 'bg-[#040705] border-emerald-950 shadow-emerald-950/20'
              : 'bg-[#090d16] border-slate-800 shadow-slate-900/50'
          }`}
          style={{ height: isFullscreen ? 'calc(100vh - 120px)' : '620px' }}
        >
          {/* Terminal Window Header Bar / Chrome */}
          <div className="flex items-center justify-between px-4 py-2 bg-slate-100 dark:bg-slate-800/90 border-b border-slate-200 dark:border-slate-700 select-none shrink-0">
            <div className="flex items-center gap-2">
              <span className="w-3 h-3 rounded-full bg-rose-500 inline-block shadow-sm" />
              <span className="w-3 h-3 rounded-full bg-amber-500 inline-block shadow-sm" />
              <span className="w-3 h-3 rounded-full bg-emerald-500 inline-block shadow-sm" />
              <span className="text-xs font-mono font-medium text-slate-700 dark:text-slate-300 ml-2">
                {info ? `${info.user}@${info.hostname}: ~ (${info.shell})` : 'root@hostvra: ~ (/bin/bash)'}
              </span>
            </div>
            <div className="flex items-center gap-2 text-[11px] font-mono text-slate-500 dark:text-slate-400">
              <span className={`w-2 h-2 rounded-full ${statusBadge.dotClass}`} />
              <span>{statusBadge.headerText}</span>
            </div>
          </div>

          {/* Terminal Screen Mount Point */}
          <div
            ref={setTerminalContainerRef}
            className="w-full flex-1 p-2 overflow-hidden"
            style={{ height: 'calc(100% - 40px)', minHeight: '350px' }}
          />

          {/* Connection Overlay when disconnected, closed, or error */}
          {(terminalState === 'disconnected' || terminalState === 'closed' || terminalState === 'error') && (
            <div className="absolute inset-0 bg-slate-900/40 backdrop-blur-sm flex items-center justify-center z-20">
              <div className="bg-white dark:bg-slate-900 p-6 rounded-2xl border border-slate-200 dark:border-slate-800 shadow-2xl text-center max-w-sm">
                <div
                  className={`w-12 h-12 rounded-full mx-auto flex items-center justify-center mb-3 ${
                    terminalState === 'closed'
                      ? 'bg-slate-100 dark:bg-slate-800 text-slate-500'
                      : terminalState === 'error'
                      ? 'bg-rose-50 dark:bg-rose-950/50 text-rose-500'
                      : 'bg-amber-50 dark:bg-amber-950/50 text-amber-500'
                  }`}
                >
                  <TerminalIcon className="w-6 h-6" />
                </div>
                <h3 className="text-sm font-bold text-slate-900 dark:text-white mb-1">
                  {disconnectInfo?.title ||
                    (terminalState === 'closed'
                      ? 'Terminal Session Ended'
                      : terminalState === 'error'
                      ? 'Connection Error'
                      : 'Connection Interrupted')}
                </h3>
                <p className="text-xs text-slate-500 dark:text-slate-400 mb-4">
                  {disconnectInfo?.description ||
                    (terminalState === 'closed'
                      ? 'The interactive shell process has exited.'
                      : terminalState === 'error'
                      ? 'Unable to establish WebSocket connection to Hostvra API.'
                      : 'The terminal connection was interrupted. Click below to reconnect.')}
                </p>
                <button
                  type="button"
                  onClick={initTerminal}
                  className={`w-full py-2.5 px-4 rounded-xl text-white font-semibold text-xs shadow-lg flex items-center justify-center gap-2 transition-all ${
                    terminalState === 'closed'
                      ? 'bg-emerald-600 hover:bg-emerald-700 shadow-emerald-500/20'
                      : 'bg-blue-600 hover:bg-blue-700 shadow-blue-500/20'
                  }`}
                >
                  <RefreshCw className="w-3.5 h-3.5" />
                  <span>{terminalState === 'closed' ? 'Start New Session' : 'Reconnect Terminal'}</span>
                </button>
              </div>
            </div>
          )}
        </div>

        {/* Quick Commands & Operations Bar */}
        <div className="bg-white dark:bg-slate-900 p-4 rounded-2xl border border-slate-200 dark:border-slate-800 shadow-sm space-y-3">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2 text-xs font-bold text-slate-700 dark:text-slate-300">
              <Sparkles className="w-4 h-4 text-blue-500" />
              <span>Interactive Quick Commands</span>
            </div>
            <button
              type="button"
              onClick={() => setShowShortcuts((prev) => !prev)}
              className="text-xs font-semibold text-blue-600 dark:text-blue-400 hover:underline flex items-center gap-1"
            >
              <HelpCircle className="w-3.5 h-3.5" />
              <span>{showShortcuts ? 'Hide Keyboard Shortcuts' : 'Show Keyboard Shortcuts'}</span>
              {showShortcuts ? <ChevronUp className="w-3 h-3" /> : <ChevronDown className="w-3 h-3" />}
            </button>
          </div>

          <div className="flex flex-wrap items-center gap-2">
            {[
              { label: 'git status', cmd: 'git status' },
              { label: 'nano .env', cmd: 'nano .env' },
              { label: 'top', cmd: 'top' },
              { label: 'uptime', cmd: 'uptime' },
              { label: 'status api', cmd: 'systemctl status hostvra-api' },
              { label: 'status web', cmd: 'systemctl status hostvra-web' },
              { label: 'docker ps', cmd: 'docker ps' },
              { label: 'free -m', cmd: 'free -m' },
              { label: 'df -h', cmd: 'df -h' },
            ].map((q) => (
              <button
                key={q.cmd}
                type="button"
                onClick={() => sendCommand(q.cmd)}
                className="px-3 py-1.5 rounded-xl border border-slate-200 dark:border-slate-700 bg-slate-50 dark:bg-slate-800/80 hover:bg-blue-50 hover:border-blue-300 hover:text-blue-600 dark:hover:bg-blue-950/40 dark:hover:border-blue-800 dark:hover:text-blue-400 text-xs font-mono font-medium text-slate-700 dark:text-slate-300 transition-all"
              >
                {q.label}
              </button>
            ))}
          </div>

          {/* Keyboard Shortcuts Reference Guide */}
          {showShortcuts && (
            <div className="mt-3 pt-3 border-t border-slate-100 dark:border-slate-800 grid grid-cols-1 md:grid-cols-3 gap-3 text-xs text-slate-600 dark:text-slate-400">
              <div className="p-2.5 rounded-xl bg-slate-50 dark:bg-slate-800/50 border border-slate-200 dark:border-slate-700">
                <div className="font-bold text-slate-800 dark:text-slate-200 mb-1">Process Controls</div>
                <ul className="space-y-1">
                  <li><kbd className="px-1.5 py-0.5 rounded bg-white dark:bg-slate-700 border text-[11px] font-mono">Ctrl + C</kbd> Interrupt running task</li>
                  <li><kbd className="px-1.5 py-0.5 rounded bg-white dark:bg-slate-700 border text-[11px] font-mono">Ctrl + D</kbd> Send EOF / Close shell</li>
                  <li><kbd className="px-1.5 py-0.5 rounded bg-white dark:bg-slate-700 border text-[11px] font-mono">Ctrl + Z</kbd> Suspend foreground task</li>
                </ul>
              </div>
              <div className="p-2.5 rounded-xl bg-slate-50 dark:bg-slate-800/50 border border-slate-200 dark:border-slate-700">
                <div className="font-bold text-slate-800 dark:text-slate-200 mb-1">Editors & Navigation</div>
                <ul className="space-y-1">
                  <li><kbd className="px-1.5 py-0.5 rounded bg-white dark:bg-slate-700 border text-[11px] font-mono">nano file</kbd> Interactive Nano editor</li>
                  <li><kbd className="px-1.5 py-0.5 rounded bg-white dark:bg-slate-700 border text-[11px] font-mono">Ctrl + O / X</kbd> Nano save &amp; exit</li>
                  <li><kbd className="px-1.5 py-0.5 rounded bg-white dark:bg-slate-700 border text-[11px] font-mono">vim / vi</kbd> Full Vim editor support</li>
                </ul>
              </div>
              <div className="p-2.5 rounded-xl bg-slate-50 dark:bg-slate-800/50 border border-slate-200 dark:border-slate-700">
                <div className="font-bold text-slate-800 dark:text-slate-200 mb-1">Terminal Navigation</div>
                <ul className="space-y-1">
                  <li><kbd className="px-1.5 py-0.5 rounded bg-white dark:bg-slate-700 border text-[11px] font-mono">Ctrl + L</kbd> Clear terminal screen</li>
                  <li><kbd className="px-1.5 py-0.5 rounded bg-white dark:bg-slate-700 border text-[11px] font-mono">Tab</kbd> Autocomplete commands</li>
                  <li><kbd className="px-1.5 py-0.5 rounded bg-white dark:bg-slate-700 border text-[11px] font-mono">↑ / ↓</kbd> Browse command history</li>
                </ul>
              </div>
            </div>
          )}
        </div>
      </div>
    </DashboardShell>
  );
}
