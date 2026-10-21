'use client';

import React from 'react';
import { Keyboard, X } from 'lucide-react';

interface KeyboardShortcutsModalProps {
  isOpen: boolean;
  onClose: () => void;
}

const SHORTCUTS = [
  { key: 'Ctrl + A / ⌘ + A', desc: 'Select all files and folders in current directory' },
  { key: 'Ctrl + C / ⌘ + C', desc: 'Copy selected files to clipboard' },
  { key: 'Ctrl + X / ⌘ + X', desc: 'Cut selected files to clipboard' },
  { key: 'Ctrl + V / ⌘ + V', desc: 'Paste files into current folder' },
  { key: 'Delete / Backspace', desc: 'Move selected files to Enterprise Trash Bin' },
  { key: 'F2', desc: 'Rename selected file or folder' },
  { key: 'Enter', desc: 'Open selected folder or edit selected file' },
  { key: 'Escape', desc: 'Clear selection or close active modal' },
  { key: 'Ctrl + S / ⌘ + S', desc: 'Save current file in code editor' },
  { key: 'Ctrl + F / ⌘ + F', desc: 'Open Find & Replace in code editor' },
  { key: 'Tab', desc: 'Indent 2 spaces in code editor' },
  { key: '?', desc: 'Open this Keyboard Shortcuts cheat sheet' },
];

export const KeyboardShortcutsModal: React.FC<KeyboardShortcutsModalProps> = ({
  isOpen,
  onClose,
}) => {
  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 z-50 bg-black/75 backdrop-blur-xs flex items-center justify-center p-4 animate-fadeIn">
      <div className="w-full max-w-lg bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-700 rounded-2xl shadow-2xl p-6 relative font-sans text-xs">
        <button
          onClick={onClose}
          className="absolute top-5 right-5 p-1 rounded-lg text-slate-400 hover:text-slate-700 dark:hover:text-white"
        >
          <X className="w-5 h-5" />
        </button>

        <div className="flex items-center gap-3 mb-4">
          <div className="w-10 h-10 rounded-xl bg-emerald-500/10 border border-emerald-500/20 flex items-center justify-center text-emerald-600">
            <Keyboard className="w-5 h-5" />
          </div>
          <div>
            <h3 className="text-sm font-bold text-slate-900 dark:text-white">
              Keyboard Shortcuts
            </h3>
            <p className="text-xs text-slate-500">Fast navigation & productivity shortcuts</p>
          </div>
        </div>

        <div className="divide-y divide-slate-100 dark:divide-surface-800">
          {SHORTCUTS.map((s) => (
            <div key={s.key} className="py-2.5 flex items-center justify-between gap-4">
              <span className="text-slate-600 dark:text-slate-300 font-medium">
                {s.desc}
              </span>
              <kbd className="px-2.5 py-1 rounded-md bg-slate-100 dark:bg-surface-800 border border-slate-300 dark:border-surface-700 font-mono text-[11px] font-bold text-slate-800 dark:text-slate-200 shrink-0 shadow-2xs">
                {s.key}
              </kbd>
            </div>
          ))}
        </div>

        <div className="mt-5 flex justify-end">
          <button
            type="button"
            onClick={onClose}
            className="px-4 py-2 rounded-xl bg-slate-100 hover:bg-slate-200 dark:bg-surface-800 dark:hover:bg-surface-700 text-slate-700 dark:text-slate-200 font-semibold"
          >
            Got it
          </button>
        </div>
      </div>
    </div>
  );
};
