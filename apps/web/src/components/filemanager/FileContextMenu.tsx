'use client';

import React, { useEffect, useLayoutEffect, useRef, useState } from 'react';
import {
  FolderOpen,
  ExternalLink,
  Edit2,
  FileCode,
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
  onEdit?: (item: FileItem) => void;
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
  onEdit,
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

  // Initial position calculation to prevent layout flash
  const [pos, setPos] = useState(() => {
    if (typeof window === 'undefined') return { left: x, top: y };
    const estWidth = 240;
    const estHeight = 540;
    const pad = 12;
    const vpW = window.innerWidth;
    const vpH = window.innerHeight;

    let initLeft = x;
    if (x + estWidth > vpW - pad) {
      initLeft = Math.max(pad, vpW - estWidth - pad);
    } else {
      initLeft = Math.max(pad, x);
    }

    let initTop = y;
    if (y + estHeight > vpH - pad) {
      if (y - estHeight >= pad) {
        initTop = y - estHeight;
      } else {
        initTop = Math.max(pad, vpH - estHeight - pad);
      }
    } else {
      initTop = Math.max(pad, y);
    }

    return { left: initLeft, top: initTop };
  });

  // Dynamic collision detection and viewport clamping after DOM measurement
  useLayoutEffect(() => {
    if (!menuRef.current) return;
    const rect = menuRef.current.getBoundingClientRect();
    const menuWidth = rect.width || 240;
    const menuHeight = rect.height || 540;
    const pad = 12;
    const vpW = window.innerWidth;
    const vpH = window.innerHeight;

    // Horizontal placement
    let newLeft = x;
    if (x + menuWidth > vpW - pad) {
      newLeft = Math.max(pad, vpW - menuWidth - pad);
    } else {
      newLeft = Math.max(pad, x);
    }

    // Vertical placement: if opening downwards overflows screen, flip upward or clamp to viewport
    let newTop = y;
    if (y + menuHeight > vpH - pad) {
      if (y - menuHeight >= pad) {
        newTop = y - menuHeight;
      } else {
        newTop = Math.max(pad, vpH - menuHeight - pad);
      }
    } else {
      newTop = Math.max(pad, y);
    }

    setPos({ left: newLeft, top: newTop });
  }, [x, y]);

  // Close on click outside, scroll, resize, or escape key
  useEffect(() => {
    const handleOutside = (e: MouseEvent) => {
      if (menuRef.current && !menuRef.current.contains(e.target as Node)) {
        onClose();
      }
    };
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onClose();
    };
    const handleScrollOrResize = () => {
      onClose();
    };

    window.addEventListener('mousedown', handleOutside);
    window.addEventListener('keydown', handleKeyDown);
    window.addEventListener('resize', handleScrollOrResize);
    window.addEventListener('scroll', handleScrollOrResize, true);
    return () => {
      window.removeEventListener('mousedown', handleOutside);
      window.removeEventListener('keydown', handleKeyDown);
      window.removeEventListener('resize', handleScrollOrResize);
      window.removeEventListener('scroll', handleScrollOrResize, true);
    };
  }, [onClose]);

  const ext = (item.extension || '').toLowerCase().replace(/^\./, '') || item.name.split('.').pop()?.toLowerCase() || '';
  const isArchive = ['zip', 'tar', 'gz', 'bz2', 'rar', '7z'].includes(ext);

  return (
    <div
      ref={menuRef}
      style={{ left: `${pos.left}px`, top: `${pos.top}px` }}
      className="fixed z-[9999] w-60 max-h-[calc(100vh-24px)] overflow-y-auto bg-white/95 dark:bg-surface-900/95 border border-slate-200 dark:border-surface-700 rounded-xl shadow-2xl py-1 text-xs text-slate-700 dark:text-slate-200 font-sans animate-scaleIn backdrop-blur-md select-none overscroll-contain custom-scrollbar"
    >
      {/* Item title preview */}
      <div className="px-3 py-1.5 border-b border-slate-100 dark:border-surface-800 text-[11px] font-bold text-slate-900 dark:text-white truncate sticky top-0 bg-white dark:bg-surface-900 z-10 flex items-center justify-between">
        <span className="truncate flex-1" title={item.name}>
          {item.name}
        </span>
      </div>

      {/* Main Actions */}
      <div className="py-1">
        {!item.is_dir && onEdit && (
          <button
            type="button"
            onClick={() => {
              onEdit(item);
              onClose();
            }}
            className="w-full flex items-center gap-2.5 px-3 py-1.5 hover:bg-slate-100 dark:hover:bg-surface-800 text-left cursor-pointer font-bold text-emerald-600 dark:text-emerald-400"
          >
            <FileCode className="w-4 h-4 text-emerald-600 dark:text-emerald-400" />
            <span>Edit File</span>
          </button>
        )}

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
