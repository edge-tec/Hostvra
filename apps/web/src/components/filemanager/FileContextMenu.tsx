'use client';

import React, { useEffect, useRef } from 'react';
import {
  FolderOpen,
  ExternalLink,
  Edit2,
  Copy,
  Scissors,
  Files,
  Archive,
  FolderArchive,
  Download,
  Star,
  Trash2,
  Info,
  Tag,
  Lock,
} from 'lucide-react';
import { FileItem } from '@/app/files/page';

interface FileContextMenuProps {
  x: number;
  y: number;
  item: FileItem;
  onClose: () => void;
  onOpen: (item: FileItem) => void;
  onOpenNewTab: (item: FileItem) => void;
  onRename: (item: FileItem) => void;
  onCopy: (item: FileItem) => void;
  onCut: (item: FileItem) => void;
  onDuplicate: (item: FileItem) => void;
  onCompress: (item: FileItem) => void;
  onExtract: (item: FileItem) => void;
  onDownload: (item: FileItem) => void;
  onFavorite: (item: FileItem) => void;
  onLabelColor: (item: FileItem, color: string) => void;
  onDelete: (item: FileItem) => void;
  onProperties: (item: FileItem) => void;
  onChmod: (item: FileItem) => void;
}

export const FileContextMenu: React.FC<FileContextMenuProps> = ({
  x,
  y,
  item,
  onClose,
  onOpen,
  onOpenNewTab,
  onRename,
  onCopy,
  onCut,
  onDuplicate,
  onCompress,
  onExtract,
  onDownload,
  onFavorite,
  onLabelColor,
  onDelete,
  onProperties,
  onChmod,
}) => {
  const menuRef = useRef<HTMLDivElement>(null);

  // Close on click outside or escape
  useEffect(() => {
    const handleOutside = (e: MouseEvent) => {
      if (menuRef.current && !menuRef.current.contains(e.target as Node)) {
        onClose();
      }
    };
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onClose();
    };

    window.addEventListener('mousedown', handleOutside);
    window.addEventListener('keydown', handleKeyDown);
    return () => {
      window.removeEventListener('mousedown', handleOutside);
      window.removeEventListener('keydown', handleKeyDown);
    };
  }, [onClose]);

  // Adjust coordinates if menu overflows window
  const menuWidth = 220;
  const menuHeight = 440;
  const adjustedX = Math.min(x, window.innerWidth - menuWidth - 10);
  const adjustedY = Math.min(y, window.innerHeight - menuHeight - 10);

  const ext = (item.extension || '').toLowerCase();
  const isArchive = ['zip', 'tar', 'gz', 'bz2', 'rar', '7z'].includes(ext);

  return (
    <div
      ref={menuRef}
      style={{ left: `${adjustedX}px`, top: `${adjustedY}px` }}
      className="fixed z-50 w-56 bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-700 rounded-xl shadow-2xl py-1 text-xs text-slate-700 dark:text-slate-200 font-sans animate-scaleIn backdrop-blur-md"
    >
      {/* Item title preview */}
      <div className="px-3 py-1.5 border-b border-slate-100 dark:border-surface-800 text-[11px] font-bold text-slate-900 dark:text-white truncate">
        {item.name}
      </div>

      {/* Main Actions */}
      <div className="py-1">
        <button
          type="button"
          onClick={() => {
            onOpen(item);
            onClose();
          }}
          className="w-full flex items-center gap-2.5 px-3 py-1.5 hover:bg-slate-100 dark:hover:bg-surface-800 text-left cursor-pointer font-medium"
        >
          <FolderOpen className="w-4 h-4 text-emerald-600" />
          <span>Open</span>
        </button>

        <button
          type="button"
          onClick={() => {
            onOpenNewTab(item);
            onClose();
          }}
          className="w-full flex items-center gap-2.5 px-3 py-1.5 hover:bg-slate-100 dark:hover:bg-surface-800 text-left cursor-pointer"
        >
          <ExternalLink className="w-4 h-4 text-slate-400" />
          <span>Open in New Tab</span>
        </button>
      </div>

      <div className="h-px bg-slate-100 dark:bg-surface-800 my-1" />

      {/* File Management Actions */}
      <div className="py-1">
        <button
          type="button"
          onClick={() => {
            onRename(item);
            onClose();
          }}
          className="w-full flex items-center justify-between px-3 py-1.5 hover:bg-slate-100 dark:hover:bg-surface-800 text-left cursor-pointer"
        >
          <span className="flex items-center gap-2.5">
            <Edit2 className="w-4 h-4 text-slate-400" />
            <span>Rename</span>
          </span>
          <span className="text-[10px] text-slate-400 font-mono">F2</span>
        </button>

        <button
          type="button"
          onClick={() => {
            onCopy(item);
            onClose();
          }}
          className="w-full flex items-center justify-between px-3 py-1.5 hover:bg-slate-100 dark:hover:bg-surface-800 text-left cursor-pointer"
        >
          <span className="flex items-center gap-2.5">
            <Copy className="w-4 h-4 text-slate-400" />
            <span>Copy</span>
          </span>
          <span className="text-[10px] text-slate-400 font-mono">Ctrl+C</span>
        </button>

        <button
          type="button"
          onClick={() => {
            onCut(item);
            onClose();
          }}
          className="w-full flex items-center justify-between px-3 py-1.5 hover:bg-slate-100 dark:hover:bg-surface-800 text-left cursor-pointer"
        >
          <span className="flex items-center gap-2.5">
            <Scissors className="w-4 h-4 text-slate-400" />
            <span>Cut / Move</span>
          </span>
          <span className="text-[10px] text-slate-400 font-mono">Ctrl+X</span>
        </button>

        <button
          type="button"
          onClick={() => {
            onDuplicate(item);
            onClose();
          }}
          className="w-full flex items-center gap-2.5 px-3 py-1.5 hover:bg-slate-100 dark:hover:bg-surface-800 text-left cursor-pointer"
        >
          <Files className="w-4 h-4 text-slate-400" />
          <span>Duplicate</span>
        </button>
      </div>

      <div className="h-px bg-slate-100 dark:bg-surface-800 my-1" />

      {/* Compression & Downloads */}
      <div className="py-1">
        <button
          type="button"
          onClick={() => {
            onCompress(item);
            onClose();
          }}
          className="w-full flex items-center gap-2.5 px-3 py-1.5 hover:bg-slate-100 dark:hover:bg-surface-800 text-left cursor-pointer"
        >
          <Archive className="w-4 h-4 text-indigo-500" />
          <span>Compress to ZIP</span>
        </button>

        {isArchive && (
          <button
            type="button"
            onClick={() => {
              onExtract(item);
              onClose();
            }}
            className="w-full flex items-center gap-2.5 px-3 py-1.5 hover:bg-slate-100 dark:hover:bg-surface-800 text-left cursor-pointer text-indigo-600 font-semibold"
          >
            <FolderArchive className="w-4 h-4 text-indigo-500" />
            <span>Extract Archive</span>
          </button>
        )}

        {!item.is_dir && (
          <button
            type="button"
            onClick={() => {
              onDownload(item);
              onClose();
            }}
            className="w-full flex items-center gap-2.5 px-3 py-1.5 hover:bg-slate-100 dark:hover:bg-surface-800 text-left cursor-pointer"
          >
            <Download className="w-4 h-4 text-slate-400" />
            <span>Download</span>
          </button>
        )}
      </div>

      <div className="h-px bg-slate-100 dark:bg-surface-800 my-1" />

      {/* Favorites & Folder Colors */}
      <div className="py-1">
        <button
          type="button"
          onClick={() => {
            onFavorite(item);
            onClose();
          }}
          className="w-full flex items-center gap-2.5 px-3 py-1.5 hover:bg-slate-100 dark:hover:bg-surface-800 text-left cursor-pointer"
        >
          <Star className="w-4 h-4 text-amber-500 fill-amber-500" />
          <span>Add to Favorites</span>
        </button>

        {/* Color Labels */}
        <div className="px-3 py-1.5">
          <span className="text-[10px] text-slate-400 block mb-1">Color Tag</span>
          <div className="flex items-center gap-1.5">
            {['blue', 'emerald', 'amber', 'rose', 'purple', 'indigo'].map((col) => (
              <button
                key={col}
                type="button"
                onClick={() => {
                  onLabelColor(item, col);
                  onClose();
                }}
                className={`w-3.5 h-3.5 rounded-full transition hover:scale-125 cursor-pointer ${
                  col === 'blue'
                    ? 'bg-sky-500'
                    : col === 'emerald'
                    ? 'bg-emerald-500'
                    : col === 'amber'
                    ? 'bg-amber-500'
                    : col === 'rose'
                    ? 'bg-rose-500'
                    : col === 'purple'
                    ? 'bg-purple-500'
                    : 'bg-indigo-500'
                }`}
              />
            ))}
          </div>
        </div>
      </div>

      <div className="h-px bg-slate-100 dark:bg-surface-800 my-1" />

      {/* Security & Deletion */}
      <div className="py-1">
        <button
          type="button"
          onClick={() => {
            onChmod(item);
            onClose();
          }}
          className="w-full flex items-center gap-2.5 px-3 py-1.5 hover:bg-slate-100 dark:hover:bg-surface-800 text-left cursor-pointer"
        >
          <Lock className="w-4 h-4 text-slate-400" />
          <span>Permissions</span>
        </button>

        <button
          type="button"
          onClick={() => {
            onProperties(item);
            onClose();
          }}
          className="w-full flex items-center gap-2.5 px-3 py-1.5 hover:bg-slate-100 dark:hover:bg-surface-800 text-left cursor-pointer"
        >
          <Info className="w-4 h-4 text-slate-400" />
          <span>Properties</span>
        </button>

        <button
          type="button"
          onClick={() => {
            onDelete(item);
            onClose();
          }}
          className="w-full flex items-center justify-between px-3 py-1.5 hover:bg-rose-50 dark:hover:bg-rose-950/40 text-rose-600 text-left cursor-pointer font-medium"
        >
          <span className="flex items-center gap-2.5">
            <Trash2 className="w-4 h-4" />
            <span>Move to Trash</span>
          </span>
          <span className="text-[10px] text-rose-400 font-mono">Del</span>
        </button>
      </div>
    </div>
  );
};
