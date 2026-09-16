'use client';

import React, { useState, useEffect, useRef } from 'react';
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
  X,
  Minus,
  Plus,
} from 'lucide-react';
import { DashboardShell } from '@/components/DashboardShell';
import { apiFetch, TerminalInfo, TerminalExecutionResult } from '@/lib/api';

interface TerminalEntry {
  id: string;
  command: string;
  cwd: string;
  stdout: string;
  stderr: string;
  exitCode: number;
  durationMs: number;
  timestamp: string;
  aborted?: boolean;
}

// Convert ANSI escape codes to styled React elements with light/dark awareness
function renderAnsi(text: string, isDark: boolean): React.ReactNode {
  if (!text) return null;

  // Match ANSI escape codes like \u001b[32m or \033[1;34m
  const parts = text.split(/(\u001b\[[0-9;]*m)/g);
  if (parts.length === 1) return text;

  const colorMap: Record<string, string> = isDark
    ? {
        '0': '', // reset
        '1': 'font-bold',
        '2': 'opacity-60',
        '3': 'italic',
        '4': 'underline',
        // Standard Foreground (Dark)
        '30': 'text-slate-400',
        '31': 'text-rose-400 font-semibold',
        '32': 'text-emerald-400 font-semibold',
        '33': 'text-amber-400 font-semibold',
        '34': 'text-sky-400 font-semibold',
        '35': 'text-fuchsia-400 font-semibold',
        '36': 'text-cyan-400 font-semibold',
        '37': 'text-slate-200',
        // High Intensity (Dark)
        '90': 'text-slate-500',
        '91': 'text-red-400 font-bold',
        '92': 'text-emerald-300 font-bold',
        '93': 'text-yellow-300 font-bold',
        '94': 'text-blue-300 font-bold',
        '95': 'text-pink-400 font-bold',
        '96': 'text-cyan-300 font-bold',
        '97': 'text-white font-bold',
      }
    : {
        '0': '', // reset
        '1': 'font-bold',
        '2': 'opacity-70',
        '3': 'italic',
        '4': 'underline',
        // Standard Foreground (Light/White theme)
        '30': 'text-slate-900',
        '31': 'text-rose-700 font-semibold',
        '32': 'text-emerald-700 font-semibold',
        '33': 'text-amber-700 font-semibold',
        '34': 'text-blue-700 font-semibold',
        '35': 'text-purple-700 font-semibold',
        '36': 'text-teal-700 font-semibold',
        '37': 'text-slate-700', // Never pure white on white background!
        // High Intensity (Light/White theme)
        '90': 'text-slate-500',
        '91': 'text-red-700 font-bold',
        '92': 'text-emerald-700 font-bold',
        '93': 'text-amber-800 font-bold',
        '94': 'text-blue-800 font-bold',
        '95': 'text-purple-800 font-bold',
        '96': 'text-teal-800 font-bold',
        '97': 'text-slate-950 font-bold',
      };

  let activeStyles = new Set<string>();
  const elements: React.ReactNode[] = [];

  for (let i = 0; i < parts.length; i++) {
    const part = parts[i];
    const match = part.match(/^\u001b\[([0-9;]*m)$/);
    if (match) {
      const codes = (match[1] || '0').split(';');
      for (const code of codes) {
        if (code === '0' || code === '') {
          activeStyles.clear();
        } else if (colorMap[code]) {
          activeStyles.add(colorMap[code]);
        }
      }
    } else if (part) {
      const className = Array.from(activeStyles).join(' ');
      elements.push(
        className ? (
          <span key={i} className={className}>
            {part}
          </span>
        ) : (
          part
        )
      );
    }
  }

  return elements;
}

// Shorten /root or /home/user paths to ~ like real Unix shells
function formatPath(path: string): string {
  if (!path) return '~';
  if (path === '/root') return '~';
  if (path.startsWith('/root/')) return '~/' + path.slice(6);
  if (path.startsWith('/home/')) {
    const parts = path.split('/');
    if (parts.length >= 3) {
      return '~' + path.slice(parts.slice(0, 3).join('/').length);
    }
  }
  return path;
}

// Robust clipboard copy that works on both HTTPS and plain HTTP IP addresses
async function copyToClipboard(text: string): Promise<boolean> {
  if (!text) return false;

  // Try modern navigator.clipboard first (if available and secure context)
  if (typeof window !== 'undefined' && window.isSecureContext && navigator.clipboard?.writeText) {
    try {
      await navigator.clipboard.writeText(text);
      return true;
    } catch {
      // fallback to execCommand below
    }
  }

  // Reliable fallback for non-secure HTTP connections (e.g. http://13.140.157.238:3000)
  try {
    const textArea = document.createElement('textarea');
    textArea.value = text;
    textArea.style.position = 'fixed';
    textArea.style.left = '-999999px';
    textArea.style.top = '-999999px';
    textArea.setAttribute('readonly', '');
    document.body.appendChild(textArea);
    textArea.focus();
    textArea.select();
    const successful = document.execCommand('copy');
    document.body.removeChild(textArea);
    return successful;
  } catch (err) {
    console.error('Fallback clipboard copy failed:', err);
    return false;
  }
}

type TerminalTheme = 'white' | 'macos' | 'ubuntu' | 'matrix';

export default function TerminalPage() {
  const [info, setInfo] = useState<TerminalInfo | null>(null);
  const [cwd, setCwd] = useState<string>('/root/Hostvra');
  const [command, setCommand] = useState('');
  const [history, setHistory] = useState<TerminalEntry[]>([]);
  const [commandHistory, setCommandHistory] = useState<string[]>([]);
  const [historyIndex, setHistoryIndex] = useState<number>(-1);
  const [isExecuting, setIsExecuting] = useState(false);
  const [copiedId, setCopiedId] = useState<string | null>(null);
  const [copiedAll, setCopiedAll] = useState(false);
  const [isFullscreen, setIsFullscreen] = useState(false);
  const [autoScroll, setAutoScroll] = useState(true);
  // Default to pure white background + black text as requested!
  const [theme, setTheme] = useState<TerminalTheme>('white');
  const [fontSize, setFontSize] = useState<number>(13);

  const inputRef = useRef<HTMLInputElement>(null);
  const terminalEndRef = useRef<HTMLDivElement>(null);
  const terminalContainerRef = useRef<HTMLDivElement>(null);
  const abortControllerRef = useRef<AbortController | null>(null);

  // Fetch initial terminal environment metadata
  useEffect(() => {
    async function loadInfo() {
      const res = await apiFetch<TerminalInfo>('/api/v1/terminal/info');
      if (res.success && res.data) {
        setInfo(res.data);
        if (res.data.default_cwd) {
          setCwd(res.data.default_cwd);
        }
      }
    }
    loadInfo();

    if (typeof window !== 'undefined') {
      const params = new URLSearchParams(window.location.search);
      const initialCmd = params.get('cmd');
      if (initialCmd) {
        setCommand(initialCmd);
      }
    }
  }, []);

  // Auto-scroll terminal to bottom when new entries arrive
  useEffect(() => {
    if (autoScroll && terminalEndRef.current) {
      terminalEndRef.current.scrollIntoView({ behavior: 'smooth' });
    }
  }, [history, isExecuting, autoScroll, command]);

  // Keep focus on the terminal prompt automatically
  useEffect(() => {
    inputRef.current?.focus();
  }, [isExecuting, history]);

  const handleCancel = () => {
    if (abortControllerRef.current) {
      abortControllerRef.current.abort();
      abortControllerRef.current = null;
    }
    setIsExecuting(false);
    setHistory((prev) => [
      ...prev,
      {
        id: Math.random().toString(),
        command: command || (commandHistory[commandHistory.length - 1] || ''),
        cwd,
        stdout: '',
        stderr: '^C',
        exitCode: 130,
        durationMs: 0,
        timestamp: new Date().toLocaleTimeString(),
        aborted: true,
      },
    ]);
    setCommand('');
  };

  // Helper to execute a single command line against the backend
  const executeSingleCommand = async (rawCmd: string, execCwd: string): Promise<string> => {
    const startTs = new Date().toLocaleTimeString();
    const controller = new AbortController();
    abortControllerRef.current = controller;

    try {
      const res = await apiFetch<TerminalExecutionResult>('/api/v1/terminal/execute', {
        method: 'POST',
        signal: controller.signal,
        body: JSON.stringify({
          command: rawCmd,
          cwd: execCwd,
        }),
      });

      if (res.success && res.data) {
        const result = res.data;
        const resultingCwd = result.cwd || execCwd;
        setHistory((prev) => [
          ...prev,
          {
            id: Math.random().toString(),
            command: rawCmd,
            cwd: resultingCwd,
            stdout: result.stdout,
            stderr: result.stderr,
            exitCode: result.exit_code,
            durationMs: result.duration_ms,
            timestamp: startTs,
          },
        ]);
        return resultingCwd;
      } else {
        setHistory((prev) => [
          ...prev,
          {
            id: Math.random().toString(),
            command: rawCmd,
            cwd: execCwd,
            stdout: '',
            stderr: res.error?.message || 'bash: command failed or rejected by server API',
            exitCode: 1,
            durationMs: 0,
            timestamp: startTs,
          },
        ]);
        return execCwd;
      }
    } catch (err: any) {
      if (err.name === 'AbortError' || err.message?.includes('aborted')) {
        return execCwd;
      }
      let errMsg = err.message || 'Network communication error';
      if (
        errMsg === 'Load failed' ||
        errMsg.includes('Failed to fetch') ||
        errMsg.includes('NetworkError')
      ) {
        errMsg =
          'Connection reset or closed by host server. If you executed a service restart (e.g. systemctl restart hostvra-web), the server restarted. Please refresh your browser page.';
      }
      setHistory((prev) => [
        ...prev,
        {
          id: Math.random().toString(),
          command: rawCmd,
          cwd: execCwd,
          stdout: '',
          stderr: errMsg,
          exitCode: 1,
          durationMs: 0,
          timestamp: startTs,
        },
      ]);
      return execCwd;
    }
  };

  // Smart execution that seamlessly handles single, sequential, or multi-line pasted commands
  const handleExecute = async (cmdToRun?: string) => {
    const rawInput = (cmdToRun !== undefined ? cmdToRun : command).trim();
    if (!rawInput || isExecuting) return;

    // Built-in client commands
    if (rawInput === 'clear') {
      setHistory([]);
      setCommand('');
      return;
    }

    if (rawInput === 'help') {
      setHistory((prev) => [
        ...prev,
        {
          id: Math.random().toString(),
          command: 'help',
          cwd,
          stdout: `Hostvra Cloud OS Shell (${info?.os || 'linux'} ${info?.arch || 'amd64'})
==============================================
Available commands & features:
  • Any Linux command : git, npm, pm2, systemctl, mariadb, nginx, docker, etc.
  • cd <directory>    : Navigate directories (state is preserved across commands)
  • Multi-command     : Paste multi-line scripts or separate commands; all execute sequentially!
  • clear / Ctrl+L    : Clear terminal screen
  • Ctrl+C            : Interrupt running process or cancel line (copies text if highlighted)
  • Up / Down Arrow   : Browse command history
  • Tab               : Autocomplete shell commands
  • Copy All / Copy   : 1-click clipboard copying (compatible with HTTP IP addresses)`,
          stderr: '',
          exitCode: 0,
          durationMs: 0,
          timestamp: new Date().toLocaleTimeString(),
        },
      ]);
      setCommand('');
      return;
    }

    if (rawInput === 'history') {
      const historyList = commandHistory
        .map((cmd, idx) => `  ${String(idx + 1).padStart(4, ' ')}  ${cmd}`)
        .join('\n');
      setHistory((prev) => [
        ...prev,
        {
          id: Math.random().toString(),
          command: 'history',
          cwd,
          stdout: historyList || 'No command history recorded yet.',
          stderr: '',
          exitCode: 0,
          durationMs: 0,
          timestamp: new Date().toLocaleTimeString(),
        },
      ]);
      setCommand('');
      return;
    }

    // Split lines
    const lines = rawInput
      .split(/\r?\n/)
      .map((l) => l.trim())
      .filter((l) => l.length > 0 && !l.startsWith('#'));

    // Check if it's a compound script (e.g. heredoc `<< EOF`, unclosed quotes, or control flow)
    const isCompoundScript =
      rawInput.includes('<<') ||
      rawInput.includes('\\') ||
      (rawInput.match(/"/g) || []).length % 2 !== 0 ||
      (rawInput.match(/'/g) || []).length % 2 !== 0 ||
      /^\s*(if|for|while|case)\b/.test(rawInput);

    setIsExecuting(true);
    setCommand('');
    setHistoryIndex(-1);

    if (isCompoundScript || lines.length <= 1) {
      // Execute as a single script
      setCommandHistory((prev) => [...prev, rawInput]);
      const newCwd = await executeSingleCommand(rawInput, cwd);
      setCwd(newCwd);
    } else {
      // Execute multi-line commands sequentially, maintaining cwd between commands
      let activeDir = cwd;
      for (const line of lines) {
        setCommandHistory((prev) => [...prev, line]);
        activeDir = await executeSingleCommand(line, activeDir);
        setCwd(activeDir);
      }
    }

    setIsExecuting(false);
    abortControllerRef.current = null;
    setTimeout(() => inputRef.current?.focus(), 30);
  };

  // Intercept paste to properly handle multi-line commands
  const handlePaste = async (e: React.ClipboardEvent<HTMLInputElement>) => {
    const pastedText = e.clipboardData.getData('text');
    if (!pastedText) return;

    if (pastedText.includes('\n')) {
      e.preventDefault();
      // Execute the pasted multi-line commands
      await handleExecute(pastedText);
    }
  };

  const commonCommands = [
    'systemctl status',
    'systemctl restart',
    'git status',
    'git pull origin main',
    'git log -n 5',
    'npm run build',
    'pm2 status',
    'pm2 restart all',
    'docker ps',
    'df -h',
    'free -m',
    'uptime',
    'ls -la',
    'cat',
    'nano',
    'mkdir',
    'chmod',
    'chown',
    'mariadb',
    'nginx -t',
    'journalctl -xeu',
    'clear',
    'help',
  ];

  const handleKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    // Ctrl + C / Cmd + C Handling:
    if ((e.ctrlKey || e.metaKey) && (e.key === 'c' || e.key === 'C')) {
      // If user has highlighted/selected text on the page, DO NOT cancel. Allow browser to copy!
      const selection = window.getSelection();
      if (selection && selection.toString().trim().length > 0) {
        return; // Allow native copy
      }

      // Otherwise, act as terminal interrupt ^C
      e.preventDefault();
      if (isExecuting) {
        handleCancel();
      } else {
        // Echo ^C and create a fresh line
        setHistory((prev) => [
          ...prev,
          {
            id: Math.random().toString(),
            command: command,
            cwd,
            stdout: '',
            stderr: '^C',
            exitCode: 130,
            durationMs: 0,
            timestamp: new Date().toLocaleTimeString(),
            aborted: true,
          },
        ]);
        setCommand('');
      }
      return;
    }

    // Ctrl + L (clear buffer)
    if (e.ctrlKey && (e.key === 'l' || e.key === 'L')) {
      e.preventDefault();
      setHistory([]);
      return;
    }

    // Tab (Autocompletion)
    if (e.key === 'Tab') {
      e.preventDefault();
      const current = command.trim();
      if (!current) return;
      const match = commonCommands.find((c) => c.startsWith(current) && c !== current);
      if (match) {
        setCommand(match);
      }
      return;
    }

    // Enter (Execute)
    if (e.key === 'Enter') {
      e.preventDefault();
      handleExecute();
      return;
    }

    // History navigation with Up/Down arrows
    if (e.key === 'ArrowUp') {
      e.preventDefault();
      if (commandHistory.length === 0) return;
      const nextIndex =
        historyIndex === -1 ? commandHistory.length - 1 : Math.max(0, historyIndex - 1);
      setHistoryIndex(nextIndex);
      setCommand(commandHistory[nextIndex]);
      return;
    }

    if (e.key === 'ArrowDown') {
      e.preventDefault();
      if (commandHistory.length === 0 || historyIndex === -1) return;
      const nextIndex = historyIndex + 1;
      if (nextIndex >= commandHistory.length) {
        setHistoryIndex(-1);
        setCommand('');
      } else {
        setHistoryIndex(nextIndex);
        setCommand(commandHistory[nextIndex]);
      }
      return;
    }
  };

  const handleCopySingle = async (id: string, text: string) => {
    const ok = await copyToClipboard(text);
    if (ok) {
      setCopiedId(id);
      setTimeout(() => setCopiedId(null), 2000);
    }
  };

  const handleCopyAll = async () => {
    const allText = history
      .map((h) => {
        let block = `${currentUser}@${currentHost}:${h.cwd}# ${h.command}`;
        if (h.stdout) block += `\n${h.stdout}`;
        if (h.stderr) block += `\n${h.stderr}`;
        return block;
      })
      .join('\n\n');

    if (!allText) return;
    const ok = await copyToClipboard(allText);
    if (ok) {
      setCopiedAll(true);
      setTimeout(() => setCopiedAll(false), 2000);
    }
  };

  const quickCommands = [
    { label: 'Git Status', cmd: 'git status' },
    { label: 'Git Pull', cmd: 'git pull origin main' },
    { label: 'API Service', cmd: 'systemctl status hostvra-api' },
    { label: 'Web UI Service', cmd: 'systemctl status hostvra-web' },
    { label: 'Uptime', cmd: 'uptime' },
    { label: 'Memory (RAM)', cmd: 'free -m' },
    { label: 'Disk Space', cmd: 'df -h' },
    { label: 'Docker Containers', cmd: 'docker ps' },
  ];

  // Theme styling configurations (White background + black text as default!)
  const themeStyles = {
    white: {
      bg: 'bg-white',
      border: 'border border-slate-300 shadow-xl',
      headerBg: 'bg-[#f4f5f7] border-b border-slate-200',
      titleText: 'text-slate-800',
      welcomeDate: 'text-slate-500',
      welcomeOs: 'text-slate-600',
      welcomeHelp: 'text-slate-500',
      welcomeHighlight: 'text-amber-700 font-semibold',
      welcomeDivider: 'border-slate-200',
      userColor: 'text-emerald-700',
      pathColor: 'text-blue-700',
      promptChar: 'text-slate-900',
      commandText: 'text-slate-950',
      inputText: 'text-slate-950',
      stdoutText: 'text-slate-900',
      stderrText: 'text-rose-700 bg-rose-50 border border-rose-200 px-2 py-0.5 rounded',
      caretColor: '#000000',
      selection: 'selection:bg-blue-100 selection:text-slate-950',
      toolbarBtn: 'text-slate-600 hover:text-slate-950 hover:bg-slate-200/80',
      isDark: false,
    },
    macos: {
      bg: 'bg-[#18181b]',
      border: 'border border-slate-800/90 shadow-2xl',
      headerBg: 'bg-[#27272a]/90 border-b border-[#3f3f46]',
      titleText: 'text-slate-300',
      welcomeDate: 'text-slate-400',
      welcomeOs: 'text-slate-500',
      welcomeHelp: 'text-slate-600',
      welcomeHighlight: 'text-amber-400 font-semibold',
      welcomeDivider: 'border-white/5',
      userColor: 'text-emerald-400',
      pathColor: 'text-sky-400',
      promptChar: 'text-slate-300',
      commandText: 'text-white',
      inputText: 'text-white',
      stdoutText: 'text-slate-200',
      stderrText: 'text-rose-400',
      caretColor: '#34d399',
      selection: 'selection:bg-emerald-500/40 selection:text-white',
      toolbarBtn: 'text-slate-400 hover:text-white hover:bg-white/10',
      isDark: true,
    },
    ubuntu: {
      bg: 'bg-[#300a24]',
      border: 'border border-[#5a1b47] shadow-2xl',
      headerBg: 'bg-[#3e1130] border-b border-[#5a1b47]',
      titleText: 'text-slate-200',
      welcomeDate: 'text-slate-400',
      welcomeOs: 'text-slate-400',
      welcomeHelp: 'text-slate-400',
      welcomeHighlight: 'text-amber-300 font-semibold',
      welcomeDivider: 'border-[#5a1b47]',
      userColor: 'text-[#8ae234]',
      pathColor: 'text-[#729fcf]',
      promptChar: 'text-white',
      commandText: 'text-[#f5f5f5]',
      inputText: 'text-[#f5f5f5]',
      stdoutText: 'text-[#f5f5f5]',
      stderrText: 'text-rose-400',
      caretColor: '#8ae234',
      selection: 'selection:bg-purple-500/40 selection:text-white',
      toolbarBtn: 'text-slate-300 hover:text-white hover:bg-white/10',
      isDark: true,
    },
    matrix: {
      bg: 'bg-[#0a0e14]',
      border: 'border border-emerald-950/80 shadow-2xl',
      headerBg: 'bg-[#0f141c] border-b border-[#1f2937]',
      titleText: 'text-emerald-400',
      welcomeDate: 'text-emerald-600',
      welcomeOs: 'text-emerald-500',
      welcomeHelp: 'text-emerald-600',
      welcomeHighlight: 'text-emerald-300 font-semibold',
      welcomeDivider: 'border-emerald-950',
      userColor: 'text-emerald-400 font-bold',
      pathColor: 'text-cyan-400 font-bold',
      promptChar: 'text-emerald-500 font-bold',
      commandText: 'text-emerald-300',
      inputText: 'text-emerald-300',
      stdoutText: 'text-emerald-400',
      stderrText: 'text-rose-400',
      caretColor: '#10b981',
      selection: 'selection:bg-emerald-900/60 selection:text-emerald-200',
      toolbarBtn: 'text-emerald-500 hover:text-emerald-300 hover:bg-emerald-950/40',
      isDark: true,
    },
  }[theme];

  const currentUser = info?.user || 'root';
  const currentHost = info?.hostname || 'vmi3561516';
  const isRoot = currentUser === 'root';

  return (
    <DashboardShell>
      <div
        className={`space-y-4 transition-all ${
          isFullscreen
            ? 'fixed inset-0 z-50 bg-slate-950 p-4 sm:p-6 flex flex-col'
            : ''
        }`}
      >
        {/* Header Title and Server Environment Badges */}
        {!isFullscreen && (
          <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3">
            <div>
              <h1 className="text-2xl font-bold tracking-tight text-slate-900 dark:text-white flex items-center gap-2.5">
                <span className="p-2 rounded-xl bg-indigo-500/10 text-indigo-500 dark:text-indigo-400">
                  <TerminalIcon className="w-5 h-5" />
                </span>
                Web Terminal
                <span className="text-[11px] px-2.5 py-0.5 rounded-full bg-emerald-500/10 text-emerald-700 dark:text-emerald-400 border border-emerald-500/20 font-semibold flex items-center gap-1.5">
                  <span className="w-1.5 h-1.5 rounded-full bg-emerald-500 animate-pulse" />
                  Root Shell
                </span>
              </h1>
              <p className="text-xs text-slate-500 dark:text-slate-400 mt-0.5">
                Authentic Unix terminal stream with multi-command execution, sequential directory tracking, and ANSI color rendering.
              </p>
            </div>

            {/* Server Badges */}
            <div className="flex flex-wrap items-center gap-2 text-xs">
              <div className="flex items-center gap-2 px-3 py-1.5 rounded-xl bg-white dark:bg-surface-800 border border-slate-200 dark:border-surface-700 text-slate-700 dark:text-slate-200 shadow-2xs">
                <Server className="w-3.5 h-3.5 text-indigo-500" />
                <span className="font-mono font-semibold">{currentHost}</span>
              </div>
              <div className="flex items-center gap-2 px-3 py-1.5 rounded-xl bg-white dark:bg-surface-800 border border-slate-200 dark:border-surface-700 text-slate-700 dark:text-slate-200 shadow-2xs">
                <User className="w-3.5 h-3.5 text-emerald-500" />
                <span className="font-mono font-semibold">{currentUser}</span>
              </div>
              <div className="flex items-center gap-2 px-3 py-1.5 rounded-xl bg-white dark:bg-surface-800 border border-slate-200 dark:border-surface-700 text-slate-700 dark:text-slate-200 shadow-2xs">
                <Folder className="w-3.5 h-3.5 text-sky-500" />
                <span className="font-mono font-semibold truncate max-w-[150px]">{formatPath(cwd)}</span>
              </div>
            </div>
          </div>
        )}

        {/* Quick Command Chips */}
        {!isFullscreen && (
          <div className="flex items-center gap-1.5 overflow-x-auto pb-1 text-xs no-scrollbar">
            <span className="text-slate-500 dark:text-slate-400 font-bold text-[11px] uppercase tracking-wider whitespace-nowrap mr-1 flex items-center gap-1">
              <Sparkles className="w-3.5 h-3.5 text-indigo-500" /> Quick:
            </span>
            {quickCommands.map((q) => (
              <button
                key={q.cmd}
                onClick={() => handleExecute(q.cmd)}
                disabled={isExecuting}
                className="px-2.5 py-1 rounded-lg bg-white hover:bg-slate-50 dark:bg-surface-800 dark:hover:bg-surface-700 text-slate-700 dark:text-slate-200 border border-slate-200 dark:border-surface-700 text-[11px] font-mono whitespace-nowrap transition-all shadow-2xs active:scale-95 disabled:opacity-40 cursor-pointer"
              >
                {q.label}
              </button>
            ))}
          </div>
        )}

        {/* Real Mac / Linux Terminal Window */}
        <div
          ref={terminalContainerRef}
          className={`flex-1 rounded-2xl overflow-hidden ${themeStyles.border} ${themeStyles.bg} flex flex-col font-mono select-text transition-colors duration-150 ${
            isFullscreen ? 'h-full min-h-0' : 'min-h-[580px] max-h-[76vh]'
          }`}
          style={{ fontSize: `${fontSize}px` }}
        >
          {/* Authentic macOS Window Titlebar with Traffic Lights */}
          <div className={`h-10 px-4 ${themeStyles.headerBg} flex items-center justify-between select-none`}>
            {/* Window Traffic Lights */}
            <div className="flex items-center gap-2 group/dots">
              <button
                type="button"
                onClick={() => setHistory([])}
                title="Clear Terminal Output (Ctrl+L)"
                className="w-3 h-3 rounded-full bg-[#ff5f56] hover:brightness-110 flex items-center justify-center transition cursor-pointer shadow-xs"
              >
                <X className="w-2 h-2 text-black/70 opacity-0 group-hover/dots:opacity-100 transition-opacity" />
              </button>
              <button
                type="button"
                onClick={() => setHistory([])}
                title="Minimize Buffer"
                className="w-3 h-3 rounded-full bg-[#ffbd2e] hover:brightness-110 flex items-center justify-center transition cursor-pointer shadow-xs"
              >
                <Minus className="w-2 h-2 text-black/70 opacity-0 group-hover/dots:opacity-100 transition-opacity" />
              </button>
              <button
                type="button"
                onClick={() => setIsFullscreen(!isFullscreen)}
                title={isFullscreen ? 'Exit Fullscreen' : 'Maximize / Fullscreen'}
                className="w-3 h-3 rounded-full bg-[#27c93f] hover:brightness-110 flex items-center justify-center transition cursor-pointer shadow-xs"
              >
                <Plus className="w-2 h-2 text-black/70 opacity-0 group-hover/dots:opacity-100 transition-opacity" />
              </button>
            </div>

            {/* Window Title (Center) */}
            <div className={`text-xs ${themeStyles.titleText} font-mono font-medium flex items-center gap-2 truncate max-w-[260px] sm:max-w-md`}>
              <span className={`${themeStyles.userColor} font-bold`}>{currentUser}@{currentHost}</span>
              <span className="opacity-50">:</span>
              <span className={`${themeStyles.pathColor} font-semibold`}>{formatPath(cwd)}</span>
              <span className="opacity-50 hidden sm:inline">— bash — 80×24</span>
            </div>

            {/* Terminal Controls & Theme Selector (Right) */}
            <div className="flex items-center gap-1.5 text-xs">
              {/* Theme Selector */}
              <div className={`flex items-center rounded-lg p-0.5 border text-[10px] ${
                theme === 'white'
                  ? 'bg-slate-200/80 border-slate-300'
                  : 'bg-black/40 border-white/10'
              }`}>
                {(['white', 'macos', 'ubuntu', 'matrix'] as TerminalTheme[]).map((t) => (
                  <button
                    key={t}
                    type="button"
                    onClick={() => setTheme(t)}
                    className={`px-2 py-0.5 rounded capitalize transition cursor-pointer ${
                      theme === t
                        ? theme === 'white'
                          ? 'bg-white text-slate-900 font-bold shadow-xs'
                          : 'bg-white/20 text-white font-bold'
                        : theme === 'white'
                          ? 'text-slate-600 hover:text-slate-900'
                          : 'text-slate-400 hover:text-white'
                    }`}
                  >
                    {t === 'white' ? 'White' : t === 'macos' ? 'Dark' : t === 'ubuntu' ? 'Linux' : 'Matrix'}
                  </button>
                ))}
              </div>

              {/* Font Size Adjust */}
              <div className={`hidden sm:flex items-center gap-1 px-1.5 py-0.5 rounded-lg border text-[11px] ${
                theme === 'white'
                  ? 'bg-slate-200/80 border-slate-300 text-slate-700'
                  : 'bg-black/40 border-white/10 text-slate-300'
              }`}>
                <button
                  type="button"
                  onClick={() => setFontSize((f) => Math.max(11, f - 1))}
                  className="px-1 cursor-pointer hover:font-bold"
                  title="Smaller Font"
                >
                  A-
                </button>
                <span className="text-[9px] opacity-70 font-mono">{fontSize}</span>
                <button
                  type="button"
                  onClick={() => setFontSize((f) => Math.min(18, f + 1))}
                  className="px-1 cursor-pointer hover:font-bold"
                  title="Larger Font"
                >
                  A+
                </button>
              </div>

              {/* Copy All Terminal Output */}
              <button
                type="button"
                onClick={handleCopyAll}
                title="Copy Terminal History to Clipboard"
                className={`p-1 rounded-lg transition cursor-pointer flex items-center gap-1 ${themeStyles.toolbarBtn}`}
              >
                {copiedAll ? (
                  <Check className="w-3.5 h-3.5 text-emerald-600" />
                ) : (
                  <Copy className="w-3.5 h-3.5" />
                )}
                <span className="hidden md:inline text-[10px] font-sans font-medium">
                  {copiedAll ? 'Copied!' : 'Copy'}
                </span>
              </button>

              {/* Clear Output */}
              <button
                type="button"
                onClick={() => setHistory([])}
                title="Clear Terminal Screen (Ctrl+L)"
                className={`p-1 rounded-lg transition cursor-pointer ${themeStyles.toolbarBtn}`}
              >
                <Trash2 className="w-3.5 h-3.5" />
              </button>

              {/* Fullscreen Toggle */}
              <button
                type="button"
                onClick={() => setIsFullscreen(!isFullscreen)}
                title={isFullscreen ? 'Exit Fullscreen' : 'Fullscreen'}
                className={`p-1 rounded-lg transition cursor-pointer ${themeStyles.toolbarBtn}`}
              >
                {isFullscreen ? <Minimize2 className="w-3.5 h-3.5" /> : <Maximize2 className="w-3.5 h-3.5" />}
              </button>
            </div>
          </div>

          {/* Terminal Viewport / Screen (Click anywhere to focus) */}
          <div
            onClick={() => {
              // Don't focus input if user is selecting text
              const sel = window.getSelection();
              if (sel && sel.toString().trim().length > 0) return;
              inputRef.current?.focus();
            }}
            className={`flex-1 overflow-y-auto p-4 space-y-2 cursor-text leading-relaxed font-mono ${themeStyles.selection}`}
          >
            {/* Authentic Unix Welcome Header */}
            <div className={`text-xs space-y-0.5 pb-2 border-b ${themeStyles.welcomeDivider} select-text`}>
              <p className={themeStyles.welcomeDate}>Last login: {new Date().toLocaleDateString()} on pts/0</p>
              <p className={themeStyles.welcomeOs}>
                Hostvra Cloud OS ({info?.os || 'GNU/Linux'} {info?.arch || 'x86_64'}) • {info?.shell || 'bash'}
              </p>
              <p className={themeStyles.welcomeHelp}>
                Type <span className={themeStyles.welcomeHighlight}>help</span> for commands,{' '}
                <span className={themeStyles.welcomeHighlight}>clear</span> (or Ctrl+L) to wipe screen. You can paste multi-line commands.
              </p>
            </div>

            {/* Historical Command & Output Log */}
            {history.map((entry) => (
              <div key={entry.id} className="space-y-0.5 leading-snug group/entry">
                {/* Command Prompt Line */}
                <div className="flex items-center justify-between flex-wrap gap-x-2">
                  <div className="flex items-baseline flex-wrap">
                    <span className={`${themeStyles.userColor} font-bold mr-0.5 select-none`}>
                      {currentUser}@{currentHost}
                    </span>
                    <span className="opacity-50 mr-0.5 select-none">:</span>
                    <span className={`${themeStyles.pathColor} font-semibold mr-1.5 select-none`}>
                      {formatPath(entry.cwd)}
                    </span>
                    <span className={`${themeStyles.promptChar} font-bold mr-2 select-none`}>
                      {isRoot ? '#' : '$'}
                    </span>
                    <span className={`${themeStyles.commandText} font-semibold break-all select-text`}>
                      {entry.command}
                    </span>
                    {entry.aborted && <span className="text-rose-600 font-bold ml-1">^C</span>}
                  </div>

                  <div className="opacity-0 group-hover/entry:opacity-100 transition-opacity flex items-center gap-2 text-[10px] text-slate-500">
                    {entry.durationMs > 0 && <span>{entry.durationMs}ms</span>}
                    <button
                      type="button"
                      onClick={(e) => {
                        e.stopPropagation();
                        handleCopySingle(entry.id, entry.stdout || entry.stderr);
                      }}
                      className="p-0.5 rounded cursor-pointer hover:bg-slate-200/60 dark:hover:bg-slate-800"
                      title="Copy Output"
                    >
                      {copiedId === entry.id ? (
                        <Check className="w-3 h-3 text-emerald-600" />
                      ) : (
                        <Copy className="w-3 h-3" />
                      )}
                    </button>
                    {entry.exitCode !== 0 && !entry.aborted && (
                      <span className="text-rose-700 bg-rose-100 dark:text-rose-400 dark:bg-rose-950/40 px-1 rounded border border-rose-300 dark:border-rose-800/40 font-mono text-[9px]">
                        [{entry.exitCode}]
                      </span>
                    )}
                  </div>
                </div>

                {/* Raw Stdout Stream */}
                {entry.stdout && (
                  <div className={`whitespace-pre-wrap break-all ${themeStyles.stdoutText} py-0.5 pl-0 leading-relaxed select-text font-mono`}>
                    {renderAnsi(entry.stdout, themeStyles.isDark)}
                  </div>
                )}

                {/* Raw Stderr Stream */}
                {entry.stderr && !entry.aborted && (
                  <div
                    className={`whitespace-pre-wrap break-all py-0.5 pl-0 leading-relaxed select-text font-mono ${
                      entry.exitCode === 0 ? themeStyles.stdoutText : themeStyles.stderrText
                    }`}
                  >
                    {renderAnsi(entry.stderr, themeStyles.isDark)}
                  </div>
                )}
              </div>
            ))}

            {/* Currently Executing Spinner Banner */}
            {isExecuting && (
              <div className={`flex items-center justify-between text-xs py-1 px-2.5 rounded-lg my-1 ${
                theme === 'white'
                  ? 'bg-emerald-50 border border-emerald-300 text-emerald-800'
                  : 'bg-emerald-950/30 border border-emerald-800/40 text-emerald-400'
              }`}>
                <div className="flex items-center gap-2">
                  <span className="w-2 h-2 rounded-full bg-emerald-500 animate-ping" />
                  <span className="font-semibold">Running command on host...</span>
                </div>
                <button
                  type="button"
                  onClick={handleCancel}
                  className="flex items-center gap-1 px-2 py-0.5 text-[11px] rounded bg-rose-100 hover:bg-rose-200 border border-rose-300 text-rose-800 dark:bg-rose-950 dark:hover:bg-rose-900 dark:border-rose-800 dark:text-rose-300 font-mono cursor-pointer transition shadow-2xs"
                >
                  <span>Interrupt</span>
                  <kbd className="text-[9px] bg-black/10 dark:bg-black/40 px-1 rounded">Ctrl+C</kbd>
                </button>
              </div>
            )}

            {/* Active Inline Command Prompt (Seamless terminal stream like Mac/Linux) */}
            <div className="flex items-baseline flex-wrap leading-snug pt-1">
              <span className={`${themeStyles.userColor} font-bold mr-0.5 select-none`}>
                {currentUser}@{currentHost}
              </span>
              <span className="opacity-50 mr-0.5 select-none">:</span>
              <span className={`${themeStyles.pathColor} font-semibold mr-1.5 select-none`}>
                {formatPath(cwd)}
              </span>
              <span className={`${themeStyles.promptChar} font-bold mr-2 select-none`}>
                {isRoot ? '#' : '$'}
              </span>

              {/* Native Continuous Command Input with Blinking Cursor and 0 border/outline */}
              <div className="relative inline-flex items-center flex-1 min-w-[200px]">
                <input
                  ref={inputRef}
                  type="text"
                  value={command}
                  onChange={(e) => setCommand(e.target.value)}
                  onKeyDown={handleKeyDown}
                  onPaste={handlePaste}
                  disabled={isExecuting}
                  className={`w-full bg-transparent ${themeStyles.inputText} font-mono focus:outline-none focus:ring-0 outline-none border-none ring-0 p-0 m-0 shadow-none leading-none appearance-none`}
                  style={{
                    caretColor: themeStyles.caretColor,
                    fontSize: `${fontSize}px`,
                  }}
                  autoFocus
                  spellCheck={false}
                  autoComplete="off"
                  autoCorrect="off"
                  autoCapitalize="off"
                />
              </div>
            </div>

            <div ref={terminalEndRef} />
          </div>
        </div>
      </div>
    </DashboardShell>
  );
}
