'use client';

import React, { useState, useEffect, useRef, useCallback, useMemo } from 'react';
import {
  Folder,
  FileText,
  Trash2,
  Edit,
  Download,
  FolderPlus,
  FilePlus,
  ChevronRight,
  RefreshCw,
  Search,
  X,
  Save,
  ShieldCheck,
  UploadCloud,
  Archive,
  FolderArchive,
  ArrowUp,
  FileCode,
  Lock,
  Copy,
  FolderInput,
  PenLine,
  Move,
  AlertCircle,
  CheckCircle2,
  Globe,
  SlidersHorizontal,
  LayoutGrid,
  List as ListIcon,
  PieChart,
  Clock,
  Terminal,
  RotateCcw,
  Check,
  Eye,
  MoreVertical,
  Scissors,
  Star,
  Plus,
  Sparkles,
  ExternalLink,
  ChevronDown,
  Keyboard,
  SortAsc,
  SortDesc,
  ArrowUpDown,
  Layers,
  PanelLeft,
  Image as ImageIcon,
  Film,
  Music,
  Database,
  File,
  HardDrive,
  Key,
  ShieldAlert,
  Server,
  Filter,
} from 'lucide-react';
import { DashboardShell } from '@/components/DashboardShell';
import { apiFetch, getApiBaseUrl } from '@/lib/api';
import { QuickAccessSidebar } from '@/components/filemanager/QuickAccessSidebar';
import { FilePreviewPanel } from '@/components/filemanager/FilePreviewPanel';
import { FileContextMenu } from '@/components/filemanager/FileContextMenu';
import { RestoreWizardModal, TrashItemData } from '@/components/filemanager/RestoreWizardModal';
import { EmptyTrashModal } from '@/components/filemanager/EmptyTrashModal';
import { MultiDomainMoveCopyModal } from '@/components/filemanager/MultiDomainMoveCopyModal';
import { UploadManagerDrawer, UploadTask } from '@/components/filemanager/UploadManagerDrawer';
import { ActivityLogsModal } from '@/components/filemanager/ActivityLogsModal';
import { StorageInfoModal } from '@/components/filemanager/StorageInfoModal';
import { WebsiteFileHub } from '@/components/filemanager/WebsiteFileHub';
import { AdvancedCodeEditor } from '@/components/filemanager/AdvancedCodeEditor';
import { KeyboardShortcutsModal } from '@/components/filemanager/KeyboardShortcutsModal';

export interface FileItem {
  name: string;
  path: string;
  size: number;
  mode: string;
  perm_octal: string;
  owner: string;
  group: string;
  uid: number;
  gid: number;
  is_dir: boolean;
  modified_at: string;
  extension: string;
}

interface FileListResponse {
  current_path: string;
  items: FileItem[];
  count: number;
}

interface DomainWebsite {
  id: string;
  domain: string;
  document_root: string;
  status: string;
}

export default function FileManagerPage() {
  // Navigation & Directory State
  const [currentPath, setCurrentPath] = useState('/var/www');
  const [directPathInput, setDirectPathInput] = useState('/var/www');
  const [isEditingPath, setIsEditingPath] = useState(false);
  const [files, setFiles] = useState<FileItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [errorMsg, setErrorMsg] = useState<string | null>(null);
  const [activeDomain, setActiveDomain] = useState<string>('');
  const [domains, setDomains] = useState<DomainWebsite[]>([]);
  const [viewMode, setViewMode] = useState<'list' | 'grid'>('list');
  const [showHidden, setShowHidden] = useState(false);
  const [refreshKey, setRefreshKey] = useState(0);
  const [sidebarCollapsed, setSidebarCollapsed] = useState(false);

  // Sorting & Category Filters State
  const [activeCategoryFilter, setActiveCategoryFilter] = useState<string>('all');
  const [sortField, setSortField] = useState<'name' | 'size' | 'modified' | 'perm'>('name');
  const [sortDirection, setSortDirection] = useState<'asc' | 'desc'>('asc');
  const [foldersFirst, setFoldersFirst] = useState(true);

  // Productivity & Website Tools Modals
  const [shortcutsModalOpen, setShortcutsModalOpen] = useState(false);
  const [fixingPermissions, setFixingPermissions] = useState(false);

  // Trash Mode State
  const [isTrashActive, setIsTrashActive] = useState(false);
  const [trashItems, setTrashItems] = useState<TrashItemData[]>([]);
  const [trashLoading, setTrashLoading] = useState(false);
  const [trashStats, setTrashStats] = useState({ totalFiles: 0, totalFolders: 0, totalSize: 0 });

  // Selection & Clipboard
  const [selectedPaths, setSelectedPaths] = useState<Set<string>>(new Set());
  const [lastSelectedPath, setLastSelectedPath] = useState<string | null>(null);
  const [clipboard, setClipboard] = useState<{ paths: string[]; action: 'copy' | 'cut' } | null>(null);

  // Inspector Preview Panel
  const [previewItem, setPreviewItem] = useState<FileItem | null>(null);
  const [previewOpen, setPreviewOpen] = useState(false);

  // Context Menu
  const [contextMenu, setContextMenu] = useState<{
    x: number;
    y: number;
    item: FileItem;
  } | null>(null);

  // Search & Filter State
  const [searchQuery, setSearchQuery] = useState('');
  const [searchFilterType, setSearchFilterType] = useState('all');
  const [searchAllDomains, setSearchAllDomains] = useState(false);
  const [searchFiltersOpen, setSearchFiltersOpen] = useState(false);
  const [isSearching, setIsSearching] = useState(false);
  const [searchResults, setSearchResults] = useState<FileItem[] | null>(null);

  // Drag and Drop
  const [draggedItem, setDraggedItem] = useState<FileItem | null>(null);
  const [dragOverPath, setDragOverPath] = useState<string | null>(null);
  const [undoAction, setUndoAction] = useState<{
    message: string;
    undo: () => Promise<void>;
  } | null>(null);

  // Modals State
  const [editorOpen, setEditorOpen] = useState(false);
  const [editingFile, setEditingFile] = useState<FileItem | null>(null);
  const [fileContent, setFileContent] = useState('');
  const [savingFile, setSavingFile] = useState(false);
  const [saveSuccess, setSaveSuccess] = useState(false);

  const [createModalOpen, setCreateModalOpen] = useState(false);
  const [createType, setCreateType] = useState<'file' | 'folder'>('file');
  const [createName, setCreateName] = useState('');
  const [creating, setCreating] = useState(false);

  const [renameModalOpen, setRenameModalOpen] = useState(false);
  const [itemToRename, setItemToRename] = useState<FileItem | null>(null);
  const [newName, setNewName] = useState('');

  const [permModalOpen, setPermModalOpen] = useState(false);
  const [permTargetPaths, setPermTargetPaths] = useState<string[]>([]);
  const [permMode, setPermMode] = useState('0755');
  const [permSaving, setPermSaving] = useState(false);

  const [archiveModalOpen, setArchiveModalOpen] = useState(false);
  const [archiveTargetPaths, setArchiveTargetPaths] = useState<string[]>([]);
  const [archiveName, setArchiveName] = useState('archive.zip');
  const [archiveProcessing, setArchiveProcessing] = useState(false);

  const [extractModalOpen, setExtractModalOpen] = useState(false);
  const [extractArchiveItem, setExtractArchiveItem] = useState<FileItem | null>(null);
  const [extractDestDir, setExtractDestDir] = useState('');
  const [extracting, setExtracting] = useState(false);

  const [moveCopyModalOpen, setMoveCopyModalOpen] = useState(false);
  const [moveCopyMode, setMoveCopyMode] = useState<'move' | 'copy'>('move');
  const [moveCopyItems, setMoveCopyItems] = useState<FileItem[]>([]);

  const [restoreWizardOpen, setRestoreWizardOpen] = useState(false);
  const [itemsToRestore, setItemsToRestore] = useState<TrashItemData[]>([]);

  const [emptyTrashModalOpen, setEmptyTrashModalOpen] = useState(false);
  const [storageModalOpen, setStorageModalOpen] = useState(false);
  const [activityModalOpen, setActivityModalOpen] = useState(false);
  const [folderToDelete, setFolderToDelete] = useState<string | null>(null);

  // Upload Manager Tasks
  const [uploadTasks, setUploadTasks] = useState<UploadTask[]>([]);
  const fileInputRef = useRef<HTMLInputElement>(null);
  const folderInputRef = useRef<HTMLInputElement>(null);

  // Mobile navigation
  const [mobileSidebarOpen, setMobileSidebarOpen] = useState(false);

  // Toast notifications
  const [toast, setToast] = useState<{ message: string; isError?: boolean } | null>(null);
  const showToast = (message: string, isError = false) => {
    setToast({ message, isError });
    setTimeout(() => setToast(null), 3500);
  };

  // Read URL query parameters on initial mount
  useEffect(() => {
    if (typeof window === 'undefined') return;
    const params = new URLSearchParams(window.location.search);
    const pathParam = params.get('path');
    const domainParam = params.get('domain');
    const viewParam = params.get('view');

    if (pathParam) {
      setCurrentPath(pathParam);
    }
    if (domainParam) {
      setActiveDomain(domainParam);
    }
    if (viewParam === 'trash') {
      setIsTrashActive(true);
    }
  }, []);

  // Fetch website domains for multi-domain switcher
  const fetchDomains = useCallback(async () => {
    try {
      const res = await apiFetch<DomainWebsite[]>('/api/v1/filemanager/domains');
      if (res.success && res.data) {
        setDomains(res.data);
      } else {
        const altRes = await apiFetch<any[]>('/api/v1/websites');
        if (altRes.success && altRes.data) {
          setDomains(
            altRes.data.map((w) => ({
              id: w.id,
              domain: w.primary_domain,
              document_root: w.document_root,
              status: w.status,
            }))
          );
        }
      }
    } catch (err) {
      console.error('Failed to load domains', err);
    }
  }, []);

  useEffect(() => {
    fetchDomains();
  }, [fetchDomains]);

  // Fetch directory listing
  const fetchDirectory = useCallback(async (path: string) => {
    try {
      setLoading(true);
      setErrorMsg(null);
      setSelectedPaths(new Set());
      setSearchResults(null);
      setIsTrashActive(false);

      const res = await apiFetch<FileListResponse>(
        `/api/v1/files/list?path=${encodeURIComponent(path)}`
      );
      if (res.success && res.data) {
        setFiles(res.data.items || []);
        const loadedPath = res.data.current_path || path;
        setCurrentPath(loadedPath);
        setDirectPathInput(loadedPath);

        // Record recent folder visit
        apiFetch('/api/v1/filemanager/recent', {
          method: 'POST',
          body: JSON.stringify({ path: loadedPath, domain: activeDomain }),
        }).catch(() => {});
      } else {
        setErrorMsg(res.error?.message || 'Failed to load directory');
      }
    } catch (err: any) {
      setErrorMsg(err.message || 'Error communicating with host filesystem API');
    } finally {
      setLoading(false);
    }
  }, [activeDomain]);

  // Fetch trash listing
  const fetchTrash = useCallback(async () => {
    try {
      setTrashLoading(true);
      setErrorMsg(null);
      setSelectedPaths(new Set());
      setIsTrashActive(true);

      const res = await apiFetch<{
        items: TrashItemData[];
        total_files: number;
        total_folders: number;
        total_size: number;
      }>('/api/v1/filemanager/trash');

      if (res.success && res.data) {
        setTrashItems(res.data.items || []);
        setTrashStats({
          totalFiles: res.data.total_files || 0,
          totalFolders: res.data.total_folders || 0,
          totalSize: res.data.total_size || 0,
        });
      }
    } catch (err: any) {
      setErrorMsg(err.message || 'Error loading trash bin');
    } finally {
      setTrashLoading(false);
    }
  }, []);

  useEffect(() => {
    if (!isTrashActive) {
      fetchDirectory(currentPath);
    } else {
      fetchTrash();
    }
  }, [fetchDirectory, fetchTrash, isTrashActive, refreshKey]);

  // Search handler
  const handleSearch = async (e?: React.FormEvent) => {
    if (e) e.preventDefault();
    if (!searchQuery.trim()) {
      setSearchResults(null);
      return;
    }

    try {
      setIsSearching(true);
      const params = new URLSearchParams({
        query: searchQuery,
        path: currentPath,
        type: searchFilterType,
        all_domains: searchAllDomains ? 'true' : 'false',
      });
      const res = await apiFetch<{ results: FileItem[]; count: number }>(
        `/api/v1/filemanager/search?${params.toString()}`
      );
      if (res.success && res.data) {
        setSearchResults(res.data.results || []);
      }
    } catch (err: any) {
      showToast(err.message || 'Search failed', true);
    } finally {
      setIsSearching(false);
    }
  };

  // Navigation helpers
  const navigateTo = (path: string, domain?: string) => {
    setIsTrashActive(false);
    if (domain) setActiveDomain(domain);
    fetchDirectory(path);
  };

  const navigateUp = () => {
    if (currentPath === '/' || currentPath === '') return;
    const parts = currentPath.split('/').filter(Boolean);
    if (parts.length <= 1) {
      fetchDirectory('/');
    } else {
      parts.pop();
      fetchDirectory('/' + parts.join('/'));
    }
  };

  // Open file or directory
  const openFileEditor = async (file: FileItem) => {
    if (file.is_dir) return;
    try {
      setEditingFile(file);
      setSaveSuccess(false);
      setFileContent('Loading file content...');
      setEditorOpen(true);
      const res = await apiFetch<{ content: string }>(
        `/api/v1/files/content?path=${encodeURIComponent(file.path)}`
      );
      if (res.success && res.data) {
        setFileContent(res.data.content);
      } else {
        showToast(res.error?.message || 'Failed to read file content', true);
        setEditorOpen(false);
      }
    } catch (err: any) {
      showToast(err.message || 'Cannot open file', true);
      setEditorOpen(false);
    }
  };

  const handleItemClick = async (file: FileItem) => {
    if (file.is_dir) {
      fetchDirectory(file.path);
    } else {
      // Select for preview panel
      setPreviewItem(file);
      setPreviewOpen(true);
      const ext = (file.extension || '').toLowerCase().replace(/^\./, '') || file.name.split('.').pop()?.toLowerCase() || '';
      const isBinary = ['zip', 'tar', 'gz', 'bz2', 'rar', '7z', 'png', 'jpg', 'jpeg', 'gif', 'webp', 'ico', 'pdf', 'mp4', 'webm', 'mov', 'mp3', 'wav'].includes(ext);

      if (!isBinary) {
        openFileEditor(file);
      }
    }
  };

  // Save file content
  const handleSaveContent = async () => {
    if (!editingFile) return;
    try {
      setSavingFile(true);
      const res = await apiFetch('/api/v1/files/content', {
        method: 'PUT',
        body: JSON.stringify({
          path: editingFile.path,
          content: fileContent,
        }),
      });

      if (res.success) {
        setSaveSuccess(true);
        setTimeout(() => setSaveSuccess(false), 3000);
        showToast(`Saved ${editingFile.name} successfully`);
        fetchDirectory(currentPath);
      } else {
        showToast(res.error?.message || 'Failed to save file', true);
      }
    } catch (err: any) {
      showToast(err.message || 'Failed to save file', true);
    } finally {
      setSavingFile(false);
    }
  };

  // Move items to Trash
  const handleMoveToTrash = async (paths: string[]) => {
    if (paths.length === 0) return;
    try {
      const res = await apiFetch<{ count: number; trashed: boolean }>(
        '/api/v1/filemanager/trash',
        {
          method: 'POST',
          body: JSON.stringify({
            paths: paths,
            domain: activeDomain,
          }),
        }
      );

      if (res.success) {
        showToast(`Moved ${paths.length} ${paths.length === 1 ? 'item' : 'items'} to Trash Bin`);
        setSelectedPaths(new Set());
        if (previewOpen && previewItem && paths.includes(previewItem.path)) {
          setPreviewOpen(false);
          setPreviewItem(null);
        }
        fetchDirectory(currentPath);
        setRefreshKey((k) => k + 1);
      } else {
        showToast(res.error?.message || 'Failed to move to trash', true);
      }
    } catch (err: any) {
      showToast(err.message || 'Error moving to trash', true);
    }
  };

  // Keyboard Shortcuts handler
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      // Don't intercept when inside input, textarea, or modals
      const target = e.target as HTMLElement;
      if (target.tagName === 'INPUT' || target.tagName === 'TEXTAREA' || editorOpen) {
        return;
      }

      // Delete key -> Move to Trash
      if (e.key === 'Delete' || e.key === 'Backspace') {
        if (selectedPaths.size > 0 && !isTrashActive) {
          e.preventDefault();
          handleMoveToTrash(Array.from(selectedPaths));
        }
      }

      // Ctrl+C / Cmd+C -> Copy
      if ((e.ctrlKey || e.metaKey) && e.key === 'c') {
        if (selectedPaths.size > 0) {
          e.preventDefault();
          setClipboard({ paths: Array.from(selectedPaths), action: 'copy' });
          showToast(`Copied ${selectedPaths.size} ${selectedPaths.size === 1 ? 'item' : 'items'} to clipboard`);
        }
      }

      // Ctrl+X / Cmd+X -> Cut
      if ((e.ctrlKey || e.metaKey) && e.key === 'x') {
        if (selectedPaths.size > 0) {
          e.preventDefault();
          setClipboard({ paths: Array.from(selectedPaths), action: 'cut' });
          showToast(`Cut ${selectedPaths.size} ${selectedPaths.size === 1 ? 'item' : 'items'} to clipboard`);
        }
      }

      // Ctrl+V / Cmd+V -> Paste
      if ((e.ctrlKey || e.metaKey) && e.key === 'v') {
        if (clipboard && clipboard.paths.length > 0) {
          e.preventDefault();
          handlePaste();
        }
      }

      // Ctrl+A / Cmd+A -> Select All
      if ((e.ctrlKey || e.metaKey) && e.key === 'a') {
        e.preventDefault();
        const allPaths = new Set(displayedItems.map((i) => i.path));
        setSelectedPaths(allPaths);
      }

      // F2 -> Rename selected item
      if (e.key === 'F2') {
        if (selectedPaths.size === 1) {
          e.preventDefault();
          const targetPath = Array.from(selectedPaths)[0];
          const item = displayedItems.find((i) => i.path === targetPath);
          if (item) {
            setItemToRename(item);
            setNewName(item.name);
            setRenameModalOpen(true);
          }
        }
      }

      // Esc -> Clear selection / close inspector
      if (e.key === 'Escape') {
        setSelectedPaths(new Set());
        setPreviewOpen(false);
      }
    };

    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [selectedPaths, clipboard, currentPath, isTrashActive, editorOpen]);

  // Paste action
  const handlePaste = async () => {
    if (!clipboard || clipboard.paths.length === 0) return;
    const endpoint =
      clipboard.action === 'cut' ? '/api/v1/filemanager/move' : '/api/v1/filemanager/copy';

    try {
      const res = await apiFetch<{ count: number }>(endpoint, {
        method: 'POST',
        body: JSON.stringify({
          src_paths: clipboard.paths,
          dest_path: currentPath,
          dest_domain: activeDomain,
          conflict_strategy: 'rename',
        }),
      });

      if (res.success) {
        showToast(
          `${clipboard.action === 'cut' ? 'Moved' : 'Copied'} ${clipboard.paths.length} items to current directory`
        );
        if (clipboard.action === 'cut') setClipboard(null);
        fetchDirectory(currentPath);
      } else {
        showToast(res.error?.message || 'Failed to paste items', true);
      }
    } catch (err: any) {
      showToast(err.message || 'Paste failed', true);
    }
  };

  // Drag and Drop files onto directory or Trash
  const handleDropOnFolder = async (targetDestPath: string, isTrash = false) => {
    if (isTrash) {
      if (draggedItem) {
        handleMoveToTrash([draggedItem.path]);
      } else if (selectedPaths.size > 0) {
        handleMoveToTrash(Array.from(selectedPaths));
      }
      return;
    }

    const itemsToMove = draggedItem
      ? [draggedItem.path]
      : Array.from(selectedPaths);

    if (itemsToMove.length === 0 || itemsToMove.includes(targetDestPath)) return;

    try {
      const res = await apiFetch<{ count: number }>('/api/v1/filemanager/move', {
        method: 'POST',
        body: JSON.stringify({
          src_paths: itemsToMove,
          dest_path: targetDestPath,
          conflict_strategy: 'rename',
        }),
      });

      if (res.success) {
        showToast(`Moved ${itemsToMove.length} items to ${targetDestPath}`);
        // Set undo option
        setUndoAction({
          message: `Moved ${itemsToMove.length} items to ${targetDestPath}`,
          undo: async () => {
            const movedDestPaths = itemsToMove.map(
              (p) => `${targetDestPath}/${p.split('/').pop()}`
            );
            await apiFetch('/api/v1/filemanager/move', {
              method: 'POST',
              body: JSON.stringify({
                src_paths: movedDestPaths,
                dest_path: currentPath,
                conflict_strategy: 'replace',
              }),
            });
            fetchDirectory(currentPath);
            showToast('Undo completed');
          },
        });
        setTimeout(() => setUndoAction(null), 8000);
        fetchDirectory(currentPath);
      } else {
        showToast(res.error?.message || 'Move failed', true);
      }
    } catch (err: any) {
      showToast(err.message || 'Drop move failed', true);
    } finally {
      setDraggedItem(null);
    }
  };

  // Upload Handlers (Regular & Chunked)
  const handleUploadFiles = async (filesToUpload: FileList | File[]) => {
    const fileArray = Array.from(filesToUpload);
    if (fileArray.length === 0) return;

    const newTasks: UploadTask[] = fileArray.map((f) => ({
      id: `${Date.now()}_${Math.random().toString(36).substr(2, 9)}`,
      file: f,
      targetDir: currentPath,
      progress: 0,
      uploadedBytes: 0,
      totalBytes: f.size,
      speed: '0 MB/s',
      status: 'queued',
    }));

    setUploadTasks((prev) => [...prev, ...newTasks]);

    // Process tasks sequentially or in parallel
    for (const task of newTasks) {
      await processUploadTask(task);
    }
    fetchDirectory(currentPath);
  };

  const processUploadTask = async (task: UploadTask) => {
    const file = task.file;
    const chunkSize = 2 * 1024 * 1024; // 2MB chunks
    const totalChunks = Math.ceil(file.size / chunkSize);
    const uploadId = task.id;

    setUploadTasks((prev) =>
      prev.map((t) => (t.id === task.id ? { ...t, status: 'uploading' } : t))
    );

    const startTime = Date.now();

    for (let chunkIndex = 0; chunkIndex < totalChunks; chunkIndex++) {
      const start = chunkIndex * chunkSize;
      const end = Math.min(start + chunkSize, file.size);
      const chunkBlob = file.slice(start, end);

      const formData = new FormData();
      formData.append('chunk', chunkBlob, file.name);
      formData.append('upload_id', uploadId);
      formData.append('filename', file.name);
      formData.append('target_dir', task.targetDir);
      formData.append('chunk_index', chunkIndex.toString());
      formData.append('total_chunks', totalChunks.toString());

      try {
        const res = await apiFetch<{ complete: boolean }>('/api/v1/filemanager/upload/chunk', {
          method: 'POST',
          body: formData,
        });

        if (!res.success) {
          setUploadTasks((prev) =>
            prev.map((t) =>
              t.id === task.id ? { ...t, status: 'failed', error: res.error?.message } : t
            )
          );
          return;
        }

        const elapsedSec = (Date.now() - startTime) / 1000;
        const uploadedBytes = end;
        const speedMBs = elapsedSec > 0 ? (uploadedBytes / (1024 * 1024) / elapsedSec).toFixed(1) : '0';
        const progress = Math.round((uploadedBytes / file.size) * 100);

        setUploadTasks((prev) =>
          prev.map((t) =>
            t.id === task.id
              ? {
                  ...t,
                  progress,
                  uploadedBytes,
                  speed: `${speedMBs} MB/s`,
                  status: res.data?.complete ? 'completed' : 'uploading',
                }
              : t
          )
        );
      } catch (err: any) {
        setUploadTasks((prev) =>
          prev.map((t) =>
            t.id === task.id ? { ...t, status: 'failed', error: err.message } : t
          )
        );
        return;
      }
    }
  };

  // Streaming Download ZIP for selected files
  const handleDownloadZip = () => {
    if (selectedPaths.size === 0) return;
    const pathsArray = Array.from(selectedPaths);
    const url = `/api/v1/filemanager/download-zip?paths_csv=${encodeURIComponent(pathsArray.join(','))}`;
    window.location.href = url;
  };

  // Filtered and Sorted Items
  const displayedItems = useMemo(() => {
    const source = searchResults || files;
    let filtered = source.filter((item) => {
      if (!showHidden && item.name.startsWith('.')) return false;
      if (activeCategoryFilter === 'folders') return item.is_dir;
      if (activeCategoryFilter === 'code') {
        const ext = (item.extension || item.name.split('.').pop() || '').toLowerCase();
        return !item.is_dir && ['php', 'js', 'ts', 'jsx', 'tsx', 'html', 'css', 'json', 'sql', 'sh', 'py'].includes(ext);
      }
      if (activeCategoryFilter === 'images') {
        const ext = (item.extension || item.name.split('.').pop() || '').toLowerCase();
        return !item.is_dir && ['png', 'jpg', 'jpeg', 'gif', 'svg', 'webp', 'ico'].includes(ext);
      }
      if (activeCategoryFilter === 'archives') {
        const ext = (item.extension || item.name.split('.').pop() || '').toLowerCase();
        return !item.is_dir && ['zip', 'tar', 'gz', 'tgz', 'rar', '7z'].includes(ext);
      }
      if (activeCategoryFilter === 'configs') {
        return !item.is_dir && (item.name.startsWith('.') || ['conf', 'ini', 'json', 'yaml', 'yml', 'env'].some((e) => item.name.endsWith(e)));
      }
      return true;
    });

    return filtered.sort((a, b) => {
      if (foldersFirst) {
        if (a.is_dir && !b.is_dir) return -1;
        if (!a.is_dir && b.is_dir) return 1;
      }
      let aVal: any = '';
      let bVal: any = '';
      if (sortField === 'name') {
        aVal = a.name.toLowerCase();
        bVal = b.name.toLowerCase();
      } else if (sortField === 'size') {
        aVal = a.size || 0;
        bVal = b.size || 0;
      } else if (sortField === 'modified') {
        aVal = new Date(a.modified_at || 0).getTime();
        bVal = new Date(b.modified_at || 0).getTime();
      } else if (sortField === 'perm') {
        aVal = a.perm_octal || '';
        bVal = b.perm_octal || '';
      }
      if (aVal < bVal) return sortDirection === 'asc' ? -1 : 1;
      if (aVal > bVal) return sortDirection === 'asc' ? 1 : -1;
      return 0;
    });
  }, [files, searchResults, showHidden, activeCategoryFilter, sortField, sortDirection, foldersFirst]);

  // Toggle sorting on column click
  const toggleSort = (field: 'name' | 'size' | 'modified' | 'perm') => {
    if (sortField === field) {
      setSortDirection((prev) => (prev === 'asc' ? 'desc' : 'asc'));
    } else {
      setSortField(field);
      setSortDirection('asc');
    }
  };

  // 1-Click Fix Web Server Permissions (0755 for dirs, 0644 for files)
  const handleFixWebPermissions = async (targetPath: string) => {
    try {
      setFixingPermissions(true);
      showToast(`Normalizing permissions (0755 dirs / 0644 files) for ${targetPath}...`);
      await apiFetch('/api/v1/files/permissions', {
        method: 'POST',
        body: JSON.stringify({
          paths: [targetPath],
          mode: '0755',
          recursive: true,
        }),
      });
      showToast(`Permissions successfully set to standard 0755 / 0644!`);
      fetchDirectory(currentPath);
    } catch (err: any) {
      showToast(`Permission fix error: ${err.message}`, true);
    } finally {
      setFixingPermissions(false);
    }
  };

  // Quick Open Common Config Files (wp-config.php, .htaccess, .env, robots.txt, nginx.conf)
  const handleOpenQuickConfigFile = async (filename: string) => {
    const targetPath = `${currentPath.replace(/\/+$/, '')}/${filename}`;
    const existing = files.find((f) => f.name === filename || f.path === targetPath);
    if (existing) {
      openFileEditor(existing);
    } else {
      try {
        const res = await apiFetch<{ content: string }>(`/api/v1/files/content?path=${encodeURIComponent(targetPath)}`);
        if (res.success && res.data) {
          const syntheticItem: FileItem = {
            name: filename,
            path: targetPath,
            size: res.data.content.length,
            mode: '-rw-r--r--',
            perm_octal: '0644',
            owner: 'www-data',
            group: 'www-data',
            uid: 33,
            gid: 33,
            is_dir: false,
            modified_at: new Date().toISOString(),
            extension: filename.split('.').pop() || '',
          };
          setEditingFile(syntheticItem);
          setFileContent(res.data.content);
          setEditorOpen(true);
        } else {
          if (confirm(`'${filename}' does not exist in ${currentPath}. Would you like to create it now?`)) {
            await apiFetch('/api/v1/files/content', {
              method: 'POST',
              body: JSON.stringify({
                path: targetPath,
                content: `# ${filename} configuration\n`,
              }),
            });
            await fetchDirectory(currentPath);
            const syntheticItem: FileItem = {
              name: filename,
              path: targetPath,
              size: 25,
              mode: '-rw-r--r--',
              perm_octal: '0644',
              owner: 'www-data',
              group: 'www-data',
              uid: 33,
              gid: 33,
              is_dir: false,
              modified_at: new Date().toISOString(),
              extension: filename.split('.').pop() || '',
            };
            setEditingFile(syntheticItem);
            setFileContent(`# ${filename} configuration\n`);
            setEditorOpen(true);
          }
        }
      } catch (err: any) {
        showToast(`Could not open ${filename}: ${err.message}`, true);
      }
    }
  };

  // Breadcrumbs builder
  const breadcrumbs = useMemo(() => {
    const clean = currentPath.replace(/\/+/g, '/').replace(/^\/|\/$/g, '');
    if (!clean) return [{ name: 'Root', path: '/' }];
    const segments = clean.split('/');
    const result = [{ name: 'Root', path: '/' }];
    let acc = '';
    for (const seg of segments) {
      acc += '/' + seg;
      result.push({ name: seg, path: acc });
    }
    return result;
  }, [currentPath]);

  const formatBytes = (bytes: number) => {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i];
  };

  // Category counts based on active filtered folder items
  const categoryCounts = useMemo(() => {
    const source = (searchResults || files).filter(
      (item) => showHidden || !item.name.startsWith('.')
    );
    let folders = 0, code = 0, images = 0, archives = 0, configs = 0;
    for (const item of source) {
      if (item.is_dir) {
        folders++;
      } else {
        const ext = (item.extension || item.name.split('.').pop() || '').toLowerCase();
        if (['php', 'js', 'ts', 'jsx', 'tsx', 'html', 'css', 'json', 'sql', 'sh', 'py'].includes(ext)) code++;
        if (['png', 'jpg', 'jpeg', 'gif', 'svg', 'webp', 'ico'].includes(ext)) images++;
        if (['zip', 'tar', 'gz', 'tgz', 'rar', '7z'].includes(ext)) archives++;
        if (item.name.startsWith('.') || ['conf', 'ini', 'json', 'yaml', 'yml', 'env'].some((e) => item.name.endsWith(e))) configs++;
      }
    }
    return { all: source.length, folders, code, images, archives, configs };
  }, [files, searchResults, showHidden]);

  const [copiedPath, setCopiedPath] = useState(false);
  const handleCopyPath = () => {
    navigator.clipboard.writeText(currentPath);
    setCopiedPath(true);
    showToast('Full directory path copied to clipboard');
    setTimeout(() => setCopiedPath(false), 2000);
  };

  const formatDate = (isoString?: string) => {
    if (!isoString) return '-';
    try {
      const d = new Date(isoString);
      if (isNaN(d.getTime())) return isoString;
      return d.toLocaleDateString(undefined, {
        month: 'short',
        day: 'numeric',
        year: 'numeric',
        hour: '2-digit',
        minute: '2-digit',
      });
    } catch {
      return isoString;
    }
  };

  const renderFileIcon = (item: FileItem, size: 'sm' | 'md' | 'lg' = 'sm') => {
    if (item.is_dir) {
      return (
        <div className={`${size === 'lg' ? 'w-14 h-14' : size === 'md' ? 'w-10 h-10' : 'w-8 h-8'} rounded-xl bg-gradient-to-br from-amber-400/20 to-amber-500/10 text-amber-500 border border-amber-500/25 flex items-center justify-center shrink-0 shadow-2xs group-hover:scale-105 transition-transform duration-150`}>
          <Folder className={`${size === 'lg' ? 'w-8 h-8' : size === 'md' ? 'w-5 h-5' : 'w-4 h-4'} fill-amber-500/30 text-amber-500`} />
        </div>
      );
    }

    const ext = (item.extension || '').toLowerCase().replace(/^\./, '') || item.name.split('.').pop()?.toLowerCase() || '';

    if (['php'].includes(ext)) {
      return (
        <div className={`${size === 'lg' ? 'w-14 h-14' : size === 'md' ? 'w-10 h-10' : 'w-8 h-8'} rounded-xl bg-gradient-to-br from-indigo-500/20 to-indigo-600/10 text-indigo-600 dark:text-indigo-400 border border-indigo-500/30 flex items-center justify-center shrink-0 shadow-2xs font-extrabold ${size === 'lg' ? 'text-sm' : size === 'md' ? 'text-xs' : 'text-[10px]'} font-mono group-hover:scale-105 transition-transform duration-150`}>
          PHP
        </div>
      );
    }
    if (['js', 'ts', 'jsx', 'tsx', 'mjs'].includes(ext)) {
      return (
        <div className={`${size === 'lg' ? 'w-14 h-14' : size === 'md' ? 'w-10 h-10' : 'w-8 h-8'} rounded-xl bg-gradient-to-br from-amber-500/20 to-yellow-500/10 text-amber-600 dark:text-amber-400 border border-amber-500/30 flex items-center justify-center shrink-0 shadow-2xs font-extrabold ${size === 'lg' ? 'text-xs' : size === 'md' ? 'text-[11px]' : 'text-[9px]'} font-mono group-hover:scale-105 transition-transform duration-150`}>
          {ext.slice(0, 3).toUpperCase()}
        </div>
      );
    }
    if (['html', 'htm'].includes(ext)) {
      return (
        <div className={`${size === 'lg' ? 'w-14 h-14' : size === 'md' ? 'w-10 h-10' : 'w-8 h-8'} rounded-xl bg-gradient-to-br from-orange-500/20 to-rose-500/10 text-orange-600 dark:text-orange-400 border border-orange-500/30 flex items-center justify-center shrink-0 shadow-2xs font-extrabold ${size === 'lg' ? 'text-xs' : size === 'md' ? 'text-[11px]' : 'text-[9px]'} font-mono group-hover:scale-105 transition-transform duration-150`}>
          HTML
        </div>
      );
    }
    if (['css', 'scss', 'sass', 'less'].includes(ext)) {
      return (
        <div className={`${size === 'lg' ? 'w-14 h-14' : size === 'md' ? 'w-10 h-10' : 'w-8 h-8'} rounded-xl bg-gradient-to-br from-sky-500/20 to-blue-500/10 text-sky-600 dark:text-sky-400 border border-sky-500/30 flex items-center justify-center shrink-0 shadow-2xs font-extrabold ${size === 'lg' ? 'text-xs' : size === 'md' ? 'text-[11px]' : 'text-[9px]'} font-mono group-hover:scale-105 transition-transform duration-150`}>
          CSS
        </div>
      );
    }
    if (['json', 'yaml', 'yml', 'xml', 'toml'].includes(ext)) {
      return (
        <div className={`${size === 'lg' ? 'w-14 h-14' : size === 'md' ? 'w-10 h-10' : 'w-8 h-8'} rounded-xl bg-gradient-to-br from-emerald-500/20 to-teal-500/10 text-emerald-600 dark:text-emerald-400 border border-emerald-500/30 flex items-center justify-center shrink-0 shadow-2xs font-extrabold ${size === 'lg' ? 'text-xs' : size === 'md' ? 'text-[11px]' : 'text-[9px]'} font-mono group-hover:scale-105 transition-transform duration-150`}>
          {ext.slice(0, 3).toUpperCase()}
        </div>
      );
    }
    if (['sql', 'sqlite', 'db'].includes(ext)) {
      return (
        <div className={`${size === 'lg' ? 'w-14 h-14' : size === 'md' ? 'w-10 h-10' : 'w-8 h-8'} rounded-xl bg-gradient-to-br from-emerald-500/20 to-emerald-600/10 text-emerald-600 dark:text-emerald-400 border border-emerald-500/30 flex items-center justify-center shrink-0 shadow-2xs group-hover:scale-105 transition-transform duration-150`}>
          <Database className={`${size === 'lg' ? 'w-7 h-7' : size === 'md' ? 'w-5 h-5' : 'w-4 h-4'}`} />
        </div>
      );
    }
    if (['zip', 'tar', 'gz', 'bz2', 'rar', '7z', 'tgz'].includes(ext)) {
      return (
        <div className={`${size === 'lg' ? 'w-14 h-14' : size === 'md' ? 'w-10 h-10' : 'w-8 h-8'} rounded-xl bg-gradient-to-br from-amber-600/20 to-amber-700/10 text-amber-700 dark:text-amber-400 border border-amber-600/30 flex items-center justify-center shrink-0 shadow-2xs group-hover:scale-105 transition-transform duration-150`}>
          <Archive className={`${size === 'lg' ? 'w-7 h-7' : size === 'md' ? 'w-5 h-5' : 'w-4 h-4'}`} />
        </div>
      );
    }
    if (['png', 'jpg', 'jpeg', 'gif', 'svg', 'webp', 'ico'].includes(ext)) {
      return (
        <div className={`${size === 'lg' ? 'w-14 h-14' : size === 'md' ? 'w-10 h-10' : 'w-8 h-8'} rounded-xl bg-gradient-to-br from-rose-500/20 to-pink-500/10 text-rose-600 dark:text-rose-400 border border-rose-500/30 flex items-center justify-center shrink-0 shadow-2xs group-hover:scale-105 transition-transform duration-150`}>
          <ImageIcon className={`${size === 'lg' ? 'w-7 h-7' : size === 'md' ? 'w-5 h-5' : 'w-4 h-4'}`} />
        </div>
      );
    }
    if (['mp4', 'mov', 'avi', 'mkv', 'webm'].includes(ext)) {
      return (
        <div className={`${size === 'lg' ? 'w-14 h-14' : size === 'md' ? 'w-10 h-10' : 'w-8 h-8'} rounded-xl bg-gradient-to-br from-violet-500/20 to-purple-500/10 text-violet-600 dark:text-violet-400 border border-violet-500/30 flex items-center justify-center shrink-0 shadow-2xs group-hover:scale-105 transition-transform duration-150`}>
          <Film className={`${size === 'lg' ? 'w-7 h-7' : size === 'md' ? 'w-5 h-5' : 'w-4 h-4'}`} />
        </div>
      );
    }
    if (['mp3', 'wav', 'ogg', 'm4a', 'flac'].includes(ext)) {
      return (
        <div className={`${size === 'lg' ? 'w-14 h-14' : size === 'md' ? 'w-10 h-10' : 'w-8 h-8'} rounded-xl bg-gradient-to-br from-fuchsia-500/20 to-pink-500/10 text-fuchsia-600 dark:text-fuchsia-400 border border-fuchsia-500/30 flex items-center justify-center shrink-0 shadow-2xs group-hover:scale-105 transition-transform duration-150`}>
          <Music className={`${size === 'lg' ? 'w-7 h-7' : size === 'md' ? 'w-5 h-5' : 'w-4 h-4'}`} />
        </div>
      );
    }
    if (item.name.startsWith('.env') || item.name === '.htaccess' || item.name.endsWith('.conf') || item.name.endsWith('.ini')) {
      return (
        <div className={`${size === 'lg' ? 'w-14 h-14' : size === 'md' ? 'w-10 h-10' : 'w-8 h-8'} rounded-xl bg-gradient-to-br from-teal-500/20 to-emerald-500/10 text-teal-600 dark:text-teal-400 border border-teal-500/30 flex items-center justify-center shrink-0 shadow-2xs group-hover:scale-105 transition-transform duration-150`}>
          <ShieldCheck className={`${size === 'lg' ? 'w-7 h-7' : size === 'md' ? 'w-5 h-5' : 'w-4 h-4'}`} />
        </div>
      );
    }
    if (['pdf', 'doc', 'docx', 'txt', 'md', 'log'].includes(ext)) {
      return (
        <div className={`${size === 'lg' ? 'w-14 h-14' : size === 'md' ? 'w-10 h-10' : 'w-8 h-8'} rounded-xl bg-gradient-to-br from-slate-500/20 to-slate-600/10 text-slate-600 dark:text-slate-400 border border-slate-500/30 flex items-center justify-center shrink-0 shadow-2xs group-hover:scale-105 transition-transform duration-150`}>
          <FileText className={`${size === 'lg' ? 'w-7 h-7' : size === 'md' ? 'w-5 h-5' : 'w-4 h-4'}`} />
        </div>
      );
    }

    return (
      <div className={`${size === 'lg' ? 'w-14 h-14' : size === 'md' ? 'w-10 h-10' : 'w-8 h-8'} rounded-xl bg-slate-100 dark:bg-surface-800 text-slate-500 dark:text-slate-400 border border-slate-200 dark:border-surface-700 flex items-center justify-center shrink-0 shadow-2xs group-hover:scale-105 transition-transform duration-150`}>
        <File className={`${size === 'lg' ? 'w-7 h-7' : size === 'md' ? 'w-5 h-5' : 'w-4 h-4'}`} />
      </div>
    );
  };

  return (
    <DashboardShell>
      <div className="flex h-[calc(100vh-4rem)] bg-white dark:bg-surface-950 font-sans text-slate-800 dark:text-slate-200 overflow-hidden relative">
        {/* Hidden inputs for uploads */}
        <input
          type="file"
          ref={fileInputRef}
          multiple
          className="hidden"
          onChange={(e) => e.target.files && handleUploadFiles(e.target.files)}
        />
        <input
          type="file"
          ref={folderInputRef}
          // @ts-ignore
          webkitdirectory="true"
          multiple
          className="hidden"
          onChange={(e) => e.target.files && handleUploadFiles(e.target.files)}
        />

        {/* 1. SMART QUICK ACCESS SIDEBAR */}
        <QuickAccessSidebar
          currentPath={currentPath}
          onNavigate={navigateTo}
          activeDomain={activeDomain}
          onDomainChange={(dom) => setActiveDomain(dom)}
          refreshKey={refreshKey}
          isMobileOpen={mobileSidebarOpen}
          onCloseMobile={() => setMobileSidebarOpen(false)}
          onTrashClick={fetchTrash}
          isTrashActive={isTrashActive}
          onDropItem={handleDropOnFolder}
          onDeleteFolder={(path) => setFolderToDelete(path)}
          isCollapsed={sidebarCollapsed}
          onToggleCollapse={() => setSidebarCollapsed(!sidebarCollapsed)}
        />

        {/* 2. MAIN FILE EXPLORER WORKSPACE */}
        <main className="flex-1 flex flex-col min-w-0 bg-slate-50/50 dark:bg-surface-950 overflow-hidden">
          {/* TOP GLOBAL TOOLBAR */}
          <div className="bg-white dark:bg-surface-900 border-b border-slate-200 dark:border-surface-800 p-2 sm:p-2.5 flex flex-wrap items-center justify-between gap-2 shadow-xs">
            {/* Left Tools & Multi-Domain Directory Switcher */}
            <div className="flex items-center gap-1.5 sm:gap-2 flex-wrap">
              {/* Dual Mobile / Desktop Sidebar Collapse Toggle */}
              <button
                type="button"
                onClick={() => {
                  if (typeof window !== 'undefined' && window.innerWidth < 768) {
                    setMobileSidebarOpen(true);
                  } else {
                    setSidebarCollapsed(!sidebarCollapsed);
                  }
                }}
                className={`p-1.5 rounded-lg border transition cursor-pointer shrink-0 ${
                  sidebarCollapsed
                    ? 'bg-emerald-50 dark:bg-emerald-950/40 text-emerald-600 border-emerald-300'
                    : 'border-slate-200 dark:border-surface-700 text-slate-600 dark:text-slate-300 hover:bg-slate-100 dark:hover:bg-surface-800'
                }`}
                title={sidebarCollapsed ? 'Show Quick Access Sidebar' : 'Hide Quick Access Sidebar'}
              >
                <PanelLeft className="w-4 h-4" />
              </button>

              {/* Multi-Domain Selector Dropdown (Phase 4) */}
              <div className="flex items-center gap-1.5 px-2 sm:px-2.5 py-1 rounded-xl bg-slate-100 dark:bg-surface-800 border border-slate-200 dark:border-surface-700 text-xs font-semibold shrink-0">
                <Globe className="w-3.5 h-3.5 text-blue-600 shrink-0" />
                <span className="text-[11px] text-slate-400 hidden sm:inline">Domain:</span>
                <select
                  value={activeDomain}
                  onChange={(e) => {
                    const dom = e.target.value;
                    setActiveDomain(dom);
                    const selected = domains.find((d) => d.domain === dom);
                    if (selected) navigateTo(selected.document_root, dom);
                  }}
                  className="bg-transparent text-slate-800 dark:text-slate-100 font-bold focus:outline-hidden cursor-pointer max-w-[120px] sm:max-w-[170px] truncate text-xs"
                >
                  <option value="">All Domains / Server</option>
                  {domains.map((d) => (
                    <option key={d.domain} value={d.domain}>
                      {d.domain}
                    </option>
                  ))}
                </select>
              </div>

              {/* Action Buttons: New File, New Folder, Upload */}
              {!isTrashActive && (
                <div className="flex items-center gap-1">
                  <button
                    type="button"
                    onClick={() => {
                      setCreateType('file');
                      setCreateName('');
                      setCreateModalOpen(true);
                    }}
                    className="flex items-center gap-1 px-2.5 py-1.5 rounded-lg bg-emerald-600 hover:bg-emerald-700 text-white font-bold text-xs transition shadow-xs cursor-pointer"
                  >
                    <FilePlus className="w-3.5 h-3.5" />
                    <span className="hidden sm:inline">New File</span>
                  </button>

                  <button
                    type="button"
                    onClick={() => {
                      setCreateType('folder');
                      setCreateName('');
                      setCreateModalOpen(true);
                    }}
                    className="flex items-center gap-1 px-2.5 py-1.5 rounded-lg border border-slate-200 dark:border-surface-700 hover:bg-slate-100 dark:hover:bg-surface-800 text-slate-700 dark:text-slate-300 font-bold text-xs transition cursor-pointer"
                  >
                    <FolderPlus className="w-3.5 h-3.5 text-amber-500" />
                    <span className="hidden sm:inline">New Folder</span>
                  </button>

                  {/* Upload button with Dropdown for File or Folder */}
                  <div className="relative group">
                    <button
                      type="button"
                      onClick={() => fileInputRef.current?.click()}
                      className="flex items-center gap-1 px-2.5 py-1.5 rounded-lg border border-slate-200 dark:border-surface-700 hover:bg-slate-100 dark:hover:bg-surface-800 text-slate-700 dark:text-slate-300 font-bold text-xs transition cursor-pointer"
                    >
                      <UploadCloud className="w-3.5 h-3.5 text-sky-500" />
                      <span className="hidden sm:inline">Upload</span>
                    </button>
                  </div>
                </div>
              )}

              {/* Empty Trash button when inside Trash Bin */}
              {isTrashActive && (
                <button
                  type="button"
                  onClick={() => setEmptyTrashModalOpen(true)}
                  disabled={trashItems.length === 0}
                  className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-rose-600 hover:bg-rose-700 text-white font-bold text-xs transition shadow-xs disabled:opacity-50 cursor-pointer"
                >
                  <Trash2 className="w-3.5 h-3.5" />
                  <span>Empty Trash (Permanent)</span>
                </button>
              )}
            </div>

            {/* Right Tools: Global Search, View Switcher, Storage, Terminal */}
            <div className="flex items-center gap-1.5 sm:gap-2 flex-wrap justify-end">
              {/* Global Search Bar (Phase 10) */}
              <form onSubmit={handleSearch} className="relative flex items-center">
                <Search className="w-3.5 h-3.5 absolute left-2.5 text-slate-400" />
                <input
                  type="text"
                  value={searchQuery}
                  onChange={(e) => setSearchQuery(e.target.value)}
                  placeholder="Global search files..."
                  className="w-28 sm:w-36 md:w-48 lg:w-56 pl-8 pr-7 py-1 text-xs rounded-xl bg-slate-50 dark:bg-surface-800 border border-slate-200 dark:border-surface-700 text-slate-800 dark:text-slate-200 focus:outline-hidden focus:ring-1 focus:ring-emerald-500"
                />
                <button
                  type="button"
                  onClick={() => setSearchFiltersOpen(!searchFiltersOpen)}
                  title="Search Filters"
                  className={`absolute right-2 p-0.5 rounded text-slate-400 hover:text-emerald-600 ${
                    searchFilterType !== 'all' || searchAllDomains ? 'text-emerald-600' : ''
                  }`}
                >
                  <SlidersHorizontal className="w-3 h-3" />
                </button>

                {/* Search Filters Popover */}
                {searchFiltersOpen && (
                  <div className="absolute top-full right-0 mt-1.5 w-64 bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-700 rounded-xl shadow-xl p-3 z-30 text-xs space-y-2.5">
                    <div className="flex items-center justify-between font-bold text-slate-900 dark:text-white">
                      <span>Search Filters</span>
                      <button
                        type="button"
                        onClick={() => setSearchFiltersOpen(false)}
                        className="text-slate-400 hover:text-slate-600"
                      >
                        <X className="w-3.5 h-3.5" />
                      </button>
                    </div>

                    <div>
                      <span className="text-[10px] text-slate-400 block mb-1">File Type</span>
                      <select
                        value={searchFilterType}
                        onChange={(e) => setSearchFilterType(e.target.value)}
                        className="w-full px-2 py-1 rounded-lg bg-slate-50 dark:bg-surface-800 border border-slate-200 dark:border-surface-700 text-xs font-semibold"
                      >
                        <option value="all">All File Types</option>
                        <option value="php">PHP Scripts</option>
                        <option value="image">Images (PNG, JPG, SVG, WebP)</option>
                        <option value="archive">Archives (ZIP, TAR, GZ)</option>
                        <option value="js">JavaScript / TypeScript</option>
                        <option value="html">HTML Documents</option>
                        <option value="css">CSS / SCSS Stylesheets</option>
                        <option value="json">JSON Configs</option>
                        <option value="document">PDF / Text Documents</option>
                        <option value="video">Videos</option>
                        <option value="audio">Audio Files</option>
                      </select>
                    </div>

                    <label className="flex items-center gap-2 cursor-pointer text-slate-700 dark:text-slate-300">
                      <input
                        type="checkbox"
                        checked={searchAllDomains}
                        onChange={(e) => setSearchAllDomains(e.target.checked)}
                        className="rounded text-emerald-600 focus:ring-emerald-500"
                      />
                      <span>Search All Domains on Server</span>
                    </label>

                    <button
                      type="button"
                      onClick={() => {
                        setSearchFiltersOpen(false);
                        handleSearch();
                      }}
                      className="w-full py-1.5 rounded-lg bg-emerald-600 text-white font-bold transition hover:bg-emerald-700"
                    >
                      Apply Search
                    </button>
                  </div>
                )}
              </form>

              {/* Hidden Files Toggle */}
              <button
                type="button"
                onClick={() => setShowHidden(!showHidden)}
                title={showHidden ? 'Hide dotfiles' : 'Show hidden dotfiles (.env, .htaccess)'}
                className={`p-1.5 rounded-lg border transition cursor-pointer text-xs font-mono font-bold ${
                  showHidden
                    ? 'bg-emerald-50 text-emerald-700 border-emerald-300 dark:bg-emerald-950/40 dark:text-emerald-300'
                    : 'border-slate-200 dark:border-surface-700 text-slate-600 hover:bg-slate-100'
                }`}
              >
                .*
              </button>

              {/* View Switcher: List vs Grid */}
              <div className="flex items-center border border-slate-200 dark:border-surface-700 rounded-lg overflow-hidden">
                <button
                  type="button"
                  onClick={() => setViewMode('list')}
                  className={`p-1.5 transition ${
                    viewMode === 'list'
                      ? 'bg-slate-200 dark:bg-surface-700 text-slate-900 dark:text-white'
                      : 'text-slate-400 hover:bg-slate-100'
                  }`}
                  title="List View"
                >
                  <ListIcon className="w-3.5 h-3.5" />
                </button>
                <button
                  type="button"
                  onClick={() => setViewMode('grid')}
                  className={`p-1.5 transition ${
                    viewMode === 'grid'
                      ? 'bg-slate-200 dark:bg-surface-700 text-slate-900 dark:text-white'
                      : 'text-slate-400 hover:bg-slate-100'
                  }`}
                  title="Grid View"
                >
                  <LayoutGrid className="w-3.5 h-3.5" />
                </button>
              </div>

              {/* Storage Telemetry */}
              <button
                type="button"
                onClick={() => setStorageModalOpen(true)}
                title="Disk Storage Telemetry"
                className="flex p-1.5 rounded-lg border border-slate-200 dark:border-surface-700 text-slate-600 hover:text-emerald-600 hover:bg-slate-100 transition cursor-pointer"
              >
                <PieChart className="w-3.5 h-3.5" />
              </button>

              {/* Activity & Audit Logs */}
              <button
                type="button"
                onClick={() => setActivityModalOpen(true)}
                title="Activity & Audit Logs"
                className="flex p-1.5 rounded-lg border border-slate-200 dark:border-surface-700 text-slate-600 hover:text-emerald-600 hover:bg-slate-100 transition cursor-pointer"
              >
                <Clock className="w-3.5 h-3.5" />
              </button>

              {/* Keyboard Shortcuts Cheat Sheet */}
              <button
                type="button"
                onClick={() => setShortcutsModalOpen(true)}
                title="Keyboard Shortcuts (?)"
                className="flex p-1.5 rounded-lg border border-slate-200 dark:border-surface-700 text-slate-600 hover:text-emerald-600 hover:bg-slate-100 transition cursor-pointer"
              >
                <Keyboard className="w-3.5 h-3.5" />
              </button>

              {/* Web Terminal Shortcut */}
              <a
                href={`/terminal?cwd=${encodeURIComponent(currentPath)}`}
                title="Open Terminal at Current Directory"
                className="p-1.5 rounded-lg border border-slate-200 dark:border-surface-700 text-slate-600 hover:text-emerald-600 hover:bg-slate-100 transition"
              >
                <Terminal className="w-3.5 h-3.5" />
              </a>

              {/* Refresh Button */}
              <button
                type="button"
                onClick={() => {
                  if (isTrashActive) fetchTrash();
                  else fetchDirectory(currentPath);
                }}
                title="Refresh Folder"
                className="p-1.5 rounded-lg border border-slate-200 dark:border-surface-700 text-slate-600 hover:bg-slate-100 transition cursor-pointer"
              >
                <RefreshCw className={`w-3.5 h-3.5 ${loading || trashLoading ? 'animate-spin' : ''}`} />
              </button>
            </div>
          </div>

          {/* ADVANCED WEBSITE FILE HUB (Document roots, quick configs, permissions) */}
          {!isTrashActive && (
            <WebsiteFileHub
              currentPath={currentPath}
              domains={domains}
              activeDomain={activeDomain}
              onDomainSelect={(dom, docRoot) => {
                setActiveDomain(dom);
                navigateTo(docRoot, dom);
              }}
              onNavigate={navigateTo}
              onQuickConfigFile={handleOpenQuickConfigFile}
              onFixPermissions={handleFixWebPermissions}
              fixingPermissions={fixingPermissions}
            />
          )}

          {/* BREADCRUMB NAVIGATION & DIRECT PATH BAR (Phase 9) */}
          <div className="bg-white/80 dark:bg-surface-900/80 backdrop-blur-xs border-b border-slate-200 dark:border-surface-800 px-3 sm:px-4 py-2 flex items-center justify-between gap-2 sm:gap-3 text-xs min-w-0">
            {/* Clickable Breadcrumbs OR Direct Path Input Form */}
            <div className="flex items-center gap-1.5 overflow-x-auto no-scrollbar min-w-0 font-medium flex-1">
              <button
                type="button"
                onClick={navigateUp}
                disabled={currentPath === '/' || isTrashActive}
                className="p-1 rounded text-slate-400 hover:text-slate-700 disabled:opacity-30 disabled:cursor-not-allowed shrink-0 cursor-pointer"
                title="Up One Folder"
              >
                <ArrowUp className="w-3.5 h-3.5" />
              </button>

              {isTrashActive ? (
                <div className="flex items-center gap-1.5 text-rose-600 font-bold shrink-0">
                  <Trash2 className="w-3.5 h-3.5" />
                  <span>Enterprise Trash Bin</span>
                </div>
              ) : isEditingPath ? (
                <form
                  onSubmit={(e) => {
                    e.preventDefault();
                    if (directPathInput.trim()) {
                      setIsEditingPath(false);
                      navigateTo(directPathInput.trim());
                    }
                  }}
                  className="flex items-center gap-1.5 w-full max-w-lg"
                >
                  <input
                    type="text"
                    value={directPathInput}
                    onChange={(e) => setDirectPathInput(e.target.value)}
                    placeholder="/var/www/..."
                    className="flex-1 px-2.5 py-1 text-xs rounded-lg bg-slate-50 dark:bg-surface-800 border border-emerald-500 font-mono text-slate-900 dark:text-white focus:outline-none"
                    autoFocus
                  />
                  <button
                    type="submit"
                    className="px-2.5 py-1 rounded-lg bg-emerald-600 text-white font-bold text-xs cursor-pointer"
                  >
                    Go
                  </button>
                  <button
                    type="button"
                    onClick={() => {
                      setDirectPathInput(currentPath);
                      setIsEditingPath(false);
                    }}
                    className="p-1 text-slate-400 hover:text-slate-600 cursor-pointer"
                  >
                    <X className="w-3.5 h-3.5" />
                  </button>
                </form>
              ) : (
                <div className="flex items-center gap-1.5 overflow-x-auto no-scrollbar min-w-0">
                  {breadcrumbs.map((b, idx) => (
                    <React.Fragment key={b.path}>
                      {idx > 0 && <ChevronRight className="w-3 h-3 text-slate-400 shrink-0" />}
                      <button
                        type="button"
                        onClick={() => navigateTo(b.path)}
                        className={`truncate max-w-[120px] sm:max-w-[160px] hover:text-emerald-600 hover:underline transition shrink-0 cursor-pointer ${
                          idx === breadcrumbs.length - 1
                            ? 'font-bold text-slate-900 dark:text-white'
                            : 'text-slate-500'
                        }`}
                      >
                        {b.name}
                      </button>
                    </React.Fragment>
                  ))}
                  <button
                    type="button"
                    onClick={() => {
                      setDirectPathInput(currentPath);
                      setIsEditingPath(true);
                    }}
                    className="p-0.5 rounded text-slate-400 hover:text-slate-700 dark:hover:text-white ml-1 shrink-0 cursor-pointer"
                    title="Type or paste path directly"
                  >
                    <PenLine className="w-3 h-3" />
                  </button>
                </div>
              )}
            </div>

            {/* Breadcrumb Right Controls & Category Filter Chips */}
            {!isTrashActive && (
              <div className="flex items-center gap-2 shrink-0 text-[11px] text-slate-500">
                {/* Responsive Category Filter Chips with live item counts */}
                <div className="flex items-center gap-1 overflow-x-auto no-scrollbar max-w-[280px] sm:max-w-none py-0.5">
                  {[
                    { id: 'all', label: 'All', count: categoryCounts.all },
                    { id: 'folders', label: 'Folders', count: categoryCounts.folders },
                    { id: 'code', label: 'Code', count: categoryCounts.code },
                    { id: 'images', label: 'Images', count: categoryCounts.images },
                    { id: 'archives', label: 'Archives', count: categoryCounts.archives },
                    { id: 'configs', label: 'Configs', count: categoryCounts.configs },
                  ].map((chip) => (
                    <button
                      key={chip.id}
                      type="button"
                      onClick={() => setActiveCategoryFilter(chip.id)}
                      className={`px-2 py-0.5 rounded-lg text-[10px] font-semibold transition cursor-pointer flex items-center gap-1 shrink-0 ${
                        activeCategoryFilter === chip.id
                          ? 'bg-emerald-600 text-white font-bold shadow-2xs'
                          : 'bg-slate-100 dark:bg-surface-800 text-slate-600 dark:text-slate-400 hover:bg-slate-200 dark:hover:bg-surface-700'
                      }`}
                    >
                      <span>{chip.label}</span>
                      <span className={`text-[9px] px-1 py-0.2 rounded-full font-mono ${
                        activeCategoryFilter === chip.id
                          ? 'bg-emerald-700/60 text-white'
                          : 'bg-slate-200/80 dark:bg-surface-700 text-slate-500 dark:text-slate-400'
                      }`}>
                        {chip.count}
                      </span>
                    </button>
                  ))}
                </div>

                <span className="h-3 w-px bg-slate-200 dark:bg-surface-700 hidden sm:inline shrink-0" />

                {/* Copy Path with checkmark animation */}
                <button
                  type="button"
                  onClick={handleCopyPath}
                  className={`hover:text-slate-700 dark:hover:text-white flex items-center gap-1 font-mono shrink-0 cursor-pointer transition ${
                    copiedPath ? 'text-emerald-600 font-bold' : ''
                  }`}
                  title="Copy Full Directory Path"
                >
                  {copiedPath ? <Check className="w-3 h-3 text-emerald-600" /> : <Copy className="w-3 h-3" />}
                  <span className="hidden md:inline">{copiedPath ? 'Copied!' : 'Copy Path'}</span>
                </button>

                <span className="h-3 w-px bg-slate-200 dark:bg-surface-700 shrink-0" />

                <span className="font-mono text-emerald-600 font-bold shrink-0 whitespace-nowrap text-[11px] bg-emerald-50 dark:bg-emerald-950/40 px-2 py-0.5 rounded-md border border-emerald-200/60 dark:border-emerald-800/60">
                  {displayedItems.length} items
                </span>
              </div>
            )}
          </div>

          {/* BULK OPERATIONS TOOLBAR (Phase 11) */}
          {selectedPaths.size > 0 && (
            <div className="bg-emerald-50 dark:bg-emerald-950/40 border-b border-emerald-200 dark:border-emerald-800 px-4 py-2 flex items-center justify-between gap-3 text-xs animate-fadeIn">
              <div className="flex items-center gap-2 font-semibold text-emerald-900 dark:text-emerald-200">
                <CheckCircle2 className="w-4 h-4 text-emerald-600" />
                <span>
                  {selectedPaths.size} {selectedPaths.size === 1 ? 'item' : 'items'} selected
                </span>
                <button
                  type="button"
                  onClick={() => setSelectedPaths(new Set())}
                  className="text-xs text-emerald-700 hover:underline ml-2"
                >
                  Deselect All
                </button>
              </div>

              {/* Bulk Action Buttons */}
              <div className="flex items-center gap-1.5 flex-wrap">
                {!isTrashActive ? (
                  <>
                    <button
                      type="button"
                      onClick={() => {
                        const items = displayedItems.filter((i) => selectedPaths.has(i.path));
                        setMoveCopyItems(items);
                        setMoveCopyMode('move');
                        setMoveCopyModalOpen(true);
                      }}
                      className="px-2.5 py-1 rounded-lg bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-700 font-bold hover:bg-slate-50 transition cursor-pointer"
                    >
                      Move
                    </button>

                    <button
                      type="button"
                      onClick={() => {
                        const items = displayedItems.filter((i) => selectedPaths.has(i.path));
                        setMoveCopyItems(items);
                        setMoveCopyMode('copy');
                        setMoveCopyModalOpen(true);
                      }}
                      className="px-2.5 py-1 rounded-lg bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-700 font-bold hover:bg-slate-50 transition cursor-pointer"
                    >
                      Copy
                    </button>

                    {/* Edit button when 1 non-directory file is selected */}
                    {(() => {
                      const selectedFiles = displayedItems.filter((i) => selectedPaths.has(i.path) && !i.is_dir);
                      if (selectedFiles.length !== 1) return null;
                      return (
                        <button
                          type="button"
                          onClick={() => openFileEditor(selectedFiles[0])}
                          className="px-2.5 py-1 rounded-lg bg-emerald-50 dark:bg-emerald-950/40 text-emerald-600 dark:text-emerald-400 border border-emerald-200 dark:border-emerald-800 font-bold hover:bg-emerald-100 dark:hover:bg-emerald-900/50 transition cursor-pointer flex items-center gap-1 shadow-2xs"
                        >
                          <Edit className="w-3.5 h-3.5" />
                          <span>Edit</span>
                        </button>
                      );
                    })()}

                    {/* Unzip / Extract button */}
                    {(() => {
                      const selectedArchives = displayedItems.filter(
                        (i) =>
                          selectedPaths.has(i.path) &&
                          !i.is_dir &&
                          /\.(zip|tar\.gz|tgz|tar|gz)$/i.test(i.name)
                      );
                      const hasArchive = selectedArchives.length > 0;
                      return (
                        <button
                          type="button"
                          onClick={() => {
                            if (hasArchive) {
                              setExtractArchiveItem(selectedArchives[0]);
                              setExtractDestDir(currentPath);
                              setExtractModalOpen(true);
                            }
                          }}
                          disabled={!hasArchive}
                          className={`px-2.5 py-1 rounded-lg border font-bold transition flex items-center gap-1 cursor-pointer ${
                            hasArchive
                              ? 'bg-indigo-600 hover:bg-indigo-700 text-white border-indigo-600 shadow-xs'
                              : 'bg-white dark:bg-surface-900 border-slate-200 dark:border-surface-700 text-slate-400 opacity-50 cursor-not-allowed'
                          }`}
                          title={hasArchive ? 'Unzip / Extract selected archive' : 'Select a ZIP or TAR archive to extract'}
                        >
                          <FolderArchive className="w-3.5 h-3.5" />
                          <span>Unzip / Extract</span>
                        </button>
                      );
                    })()}

                    <button
                      type="button"
                      onClick={() => {
                        setArchiveTargetPaths(Array.from(selectedPaths));
                        setArchiveName('archive.zip');
                        setArchiveModalOpen(true);
                      }}
                      className="px-2.5 py-1 rounded-lg bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-700 font-bold hover:bg-slate-50 transition cursor-pointer"
                    >
                      Compress ZIP
                    </button>

                    <button
                      type="button"
                      onClick={handleDownloadZip}
                      className="px-2.5 py-1 rounded-lg bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-700 font-bold hover:bg-slate-50 transition cursor-pointer"
                    >
                      Download ZIP
                    </button>

                    <button
                      type="button"
                      onClick={() => {
                        setPermTargetPaths(Array.from(selectedPaths));
                        setPermMode('0755');
                        setPermModalOpen(true);
                      }}
                      className="px-2.5 py-1 rounded-lg bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-700 font-bold hover:bg-slate-50 transition cursor-pointer"
                    >
                      Permissions
                    </button>

                    <button
                      type="button"
                      onClick={() => handleMoveToTrash(Array.from(selectedPaths))}
                      className="px-2.5 py-1 rounded-lg bg-rose-600 hover:bg-rose-700 text-white font-bold transition shadow-xs cursor-pointer flex items-center gap-1"
                    >
                      <Trash2 className="w-3.5 h-3.5" />
                      <span>Move to Trash</span>
                    </button>
                  </>
                ) : (
                  <>
                    <button
                      type="button"
                      onClick={() => {
                        const items = trashItems.filter((i) => selectedPaths.has(i.id));
                        setItemsToRestore(items);
                        setRestoreWizardOpen(true);
                      }}
                      className="px-3 py-1 rounded-lg bg-emerald-600 hover:bg-emerald-700 text-white font-bold transition flex items-center gap-1"
                    >
                      <RotateCcw className="w-3.5 h-3.5" />
                      <span>Restore Selected</span>
                    </button>
                  </>
                )}
              </div>
            </div>
          )}

          {/* UNDO NOTIFICATION BANNER */}
          {undoAction && (
            <div className="bg-slate-900 text-white px-4 py-2 flex items-center justify-between text-xs animate-slideDown shadow-lg">
              <span>{undoAction.message}</span>
              <button
                type="button"
                onClick={async () => {
                  await undoAction.undo();
                  setUndoAction(null);
                }}
                className="px-3 py-1 bg-emerald-600 hover:bg-emerald-700 text-white rounded font-bold transition"
              >
                Undo
              </button>
            </div>
          )}

          {/* WORKSPACE FILE LISTING / GRID / TRASH */}
          <div
            className="flex-1 overflow-auto p-3"
            onDragOver={(e) => e.preventDefault()}
            onDrop={(e) => {
              e.preventDefault();
              if (e.dataTransfer.files && e.dataTransfer.files.length > 0) {
                handleUploadFiles(e.dataTransfer.files);
              }
            }}
          >
            {/* TRASH VIEW */}
            {isTrashActive ? (
              <div className="space-y-3">
                <div className="flex items-center justify-between p-3 rounded-xl bg-rose-50/60 dark:bg-rose-950/20 border border-rose-200 dark:border-rose-900/60">
                  <div className="flex items-center gap-3">
                    <Trash2 className="w-5 h-5 text-rose-600" />
                    <div>
                      <h3 className="font-bold text-slate-900 dark:text-white text-xs">
                        Enterprise Trash Bin ({trashItems.length} items)
                      </h3>
                      <p className="text-[11px] text-slate-500">
                        Total storage held: {formatBytes(trashStats.totalSize)} • Restore items with Restore Wizard
                      </p>
                    </div>
                  </div>
                </div>

                {trashLoading ? (
                  <div className="py-20 text-center text-slate-400 text-xs">
                    <RefreshCw className="w-6 h-6 animate-spin text-emerald-600 mx-auto mb-2" />
                    <span>Loading trash items...</span>
                  </div>
                ) : trashItems.length === 0 ? (
                  <div className="py-20 text-center text-slate-400 text-xs">
                    <Trash2 className="w-8 h-8 text-slate-300 dark:text-slate-700 mx-auto mb-2" />
                    <p className="font-semibold text-slate-700 dark:text-slate-300">Trash Bin is Empty</p>
                    <p className="text-[11px] text-slate-400">Deleted files will safely stay here until emptied.</p>
                  </div>
                ) : (
                  <div className="rounded-xl border border-slate-200 dark:border-surface-800 bg-white dark:bg-surface-900 overflow-x-auto custom-scrollbar shadow-xs">
                    <table className="w-full text-left text-xs min-w-[600px] lg:min-w-full">
                      <thead className="bg-slate-50 dark:bg-surface-800 text-slate-500 font-bold uppercase text-[10px] tracking-wider border-b border-slate-200 dark:border-surface-700">
                        <tr>
                          <th className="py-2.5 px-3 w-8">
                            <input
                              type="checkbox"
                              checked={
                                selectedPaths.size === trashItems.length && trashItems.length > 0
                              }
                              onChange={(e) => {
                                if (e.target.checked) {
                                  setSelectedPaths(new Set(trashItems.map((i) => i.id)));
                                } else {
                                  setSelectedPaths(new Set());
                                }
                              }}
                              className="rounded text-emerald-600 focus:ring-emerald-500 cursor-pointer"
                            />
                          </th>
                          <th className="py-2.5 px-3 min-w-[160px]">Name & Original Path</th>
                          <th className="py-2.5 px-3 min-w-[70px]">Size</th>
                          <th className="py-2.5 px-3 min-w-[100px]">Deleted By</th>
                          <th className="py-2.5 px-3 min-w-[120px]">Deleted At</th>
                          <th className="py-2.5 px-3 text-right">Actions</th>
                        </tr>
                      </thead>
                      <tbody className="divide-y divide-slate-100 dark:divide-surface-800">
                        {trashItems.map((t) => (
                          <tr
                            key={t.id}
                            className={`hover:bg-slate-50/80 dark:hover:bg-surface-800/60 transition ${
                              selectedPaths.has(t.id) ? 'bg-emerald-50/60 dark:bg-emerald-950/30' : ''
                            }`}
                          >
                            <td className="py-2.5 px-3">
                              <input
                                type="checkbox"
                                checked={selectedPaths.has(t.id)}
                                onChange={(e) => {
                                  const next = new Set(selectedPaths);
                                  if (e.target.checked) next.add(t.id);
                                  else next.delete(t.id);
                                  setSelectedPaths(next);
                                }}
                                className="rounded text-emerald-600 focus:ring-emerald-500 cursor-pointer"
                              />
                            </td>
                            <td className="py-2.5 px-3 min-w-[160px]">
                              <div className="font-semibold text-slate-900 dark:text-white">
                                {t.name}
                              </div>
                              <div className="text-[10px] text-slate-400 font-mono truncate max-w-sm">
                                {t.original_path}
                              </div>
                            </td>
                            <td className="py-2.5 px-3 font-mono text-slate-600 dark:text-slate-400 whitespace-nowrap">
                              {formatBytes(t.size)}
                            </td>
                            <td className="py-2.5 px-3 text-slate-600 dark:text-slate-400 whitespace-nowrap">
                              {t.deleted_by || 'Admin'}
                            </td>
                            <td className="py-2.5 px-3 text-slate-500 text-[11px] whitespace-nowrap">
                              {t.deleted_at ? new Date(t.deleted_at).toLocaleString() : 'N/A'}
                            </td>
                            <td className="py-2.5 px-3 text-right">
                              <button
                                type="button"
                                onClick={() => {
                                  setItemsToRestore([t]);
                                  setRestoreWizardOpen(true);
                                }}
                                className="px-2.5 py-1 rounded-lg bg-emerald-50 text-emerald-700 dark:bg-emerald-950/40 dark:text-emerald-300 font-bold hover:bg-emerald-100 transition mr-2 cursor-pointer"
                              >
                                Restore
                              </button>
                            </td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>
                )}
              </div>
            ) : (
              /* REGULAR FILE LISTING (LIST OR GRID) */
              <div>
                {loading ? (
                  <div className="py-24 text-center text-slate-400 text-xs">
                    <RefreshCw className="w-6 h-6 animate-spin text-emerald-600 mx-auto mb-2" />
                    <span>Loading directory contents...</span>
                  </div>
                ) : displayedItems.length === 0 ? (
                  <div className="py-24 text-center text-slate-400 text-xs">
                    <Folder className="w-10 h-10 text-slate-300 dark:text-slate-700 mx-auto mb-2" />
                    <p className="font-semibold text-slate-700 dark:text-slate-300">
                      This folder is empty
                    </p>
                    <p className="text-[11px] text-slate-400 mt-1">
                      Drag files here, or click New File / Upload to get started.
                    </p>
                  </div>
                ) : viewMode === 'list' ? (
                  /* TABLE LIST VIEW */
                  <div className="rounded-xl border border-slate-200 dark:border-surface-800 bg-white dark:bg-surface-900 overflow-x-auto custom-scrollbar shadow-xs">
                    <table className="w-full text-left text-xs min-w-[620px] lg:min-w-full">
                      <thead className="bg-slate-50 dark:bg-surface-800 text-slate-500 font-bold uppercase text-[10px] tracking-wider border-b border-slate-200 dark:border-surface-700">
                        <tr>
                          <th className="py-2.5 px-3 w-8">
                            <input
                              type="checkbox"
                              checked={
                                selectedPaths.size === displayedItems.length &&
                                displayedItems.length > 0
                              }
                              onChange={(e) => {
                                if (e.target.checked) {
                                  setSelectedPaths(new Set(displayedItems.map((i) => i.path)));
                                } else {
                                  setSelectedPaths(new Set());
                                }
                              }}
                              className="rounded text-emerald-600 focus:ring-emerald-500 cursor-pointer"
                            />
                          </th>
                          <th
                            onClick={() => toggleSort('name')}
                            className="py-2.5 px-3 cursor-pointer hover:text-emerald-600 select-none min-w-[160px]"
                          >
                            <div className="flex items-center gap-1">
                              <span>Name</span>
                              {sortField === 'name' ? (
                                sortDirection === 'asc' ? <SortAsc className="w-3.5 h-3.5 text-emerald-600" /> : <SortDesc className="w-3.5 h-3.5 text-emerald-600" />
                              ) : (
                                <ArrowUpDown className="w-3 h-3 text-slate-400" />
                              )}
                            </div>
                          </th>
                          <th
                            onClick={() => toggleSort('size')}
                            className="py-2.5 px-3 cursor-pointer hover:text-emerald-600 select-none min-w-[70px]"
                          >
                            <div className="flex items-center gap-1">
                              <span>Size</span>
                              {sortField === 'size' ? (
                                sortDirection === 'asc' ? <SortAsc className="w-3.5 h-3.5 text-emerald-600" /> : <SortDesc className="w-3.5 h-3.5 text-emerald-600" />
                              ) : (
                                <ArrowUpDown className="w-3 h-3 text-slate-400" />
                              )}
                            </div>
                          </th>
                          <th
                            onClick={() => toggleSort('perm')}
                            className="py-2.5 px-3 cursor-pointer hover:text-emerald-600 select-none min-w-[85px]"
                          >
                            <div className="flex items-center gap-1">
                              <span>Permissions</span>
                              {sortField === 'perm' ? (
                                sortDirection === 'asc' ? <SortAsc className="w-3.5 h-3.5 text-emerald-600" /> : <SortDesc className="w-3.5 h-3.5 text-emerald-600" />
                              ) : (
                                <ArrowUpDown className="w-3 h-3 text-slate-400" />
                              )}
                            </div>
                          </th>
                          <th className="py-2.5 px-3 min-w-[110px]">Owner:Group</th>
                          <th
                            onClick={() => toggleSort('modified')}
                            className="py-2.5 px-3 cursor-pointer hover:text-emerald-600 select-none min-w-[130px]"
                          >
                            <div className="flex items-center gap-1">
                              <span>Last Modified</span>
                              {sortField === 'modified' ? (
                                sortDirection === 'asc' ? <SortAsc className="w-3.5 h-3.5 text-emerald-600" /> : <SortDesc className="w-3.5 h-3.5 text-emerald-600" />
                              ) : (
                                <ArrowUpDown className="w-3 h-3 text-slate-400" />
                              )}
                            </div>
                          </th>
                          <th className="py-2.5 px-3 text-right w-10 shrink-0">Actions</th>
                        </tr>
                      </thead>
                      <tbody className="divide-y divide-slate-100 dark:divide-surface-800">
                        {displayedItems.map((file) => {
                          const isSelected = selectedPaths.has(file.path);
                          const isDragTarget = dragOverPath === file.path;

                          return (
                            <tr
                              key={file.path}
                              draggable
                              onDragStart={() => setDraggedItem(file)}
                              onDragOver={(e) => {
                                if (file.is_dir) {
                                  e.preventDefault();
                                  setDragOverPath(file.path);
                                }
                              }}
                              onDragLeave={() => setDragOverPath(null)}
                              onDrop={(e) => {
                                if (file.is_dir) {
                                  e.preventDefault();
                                  setDragOverPath(null);
                                  handleDropOnFolder(file.path);
                                }
                              }}
                              onContextMenu={(e) => {
                                e.preventDefault();
                                setContextMenu({
                                  x: e.clientX,
                                  y: e.clientY,
                                  item: file,
                                });
                              }}
                              onClick={() => {
                                const next = new Set(selectedPaths);
                                if (next.has(file.path)) next.delete(file.path);
                                else next.add(file.path);
                                setSelectedPaths(next);
                                setPreviewItem(file);
                              }}
                              onDoubleClick={() => handleItemClick(file)}
                              className={`group cursor-pointer transition-colors duration-100 select-none ${
                                isSelected
                                  ? 'bg-emerald-50/80 dark:bg-emerald-950/40 text-emerald-950 dark:text-emerald-200'
                                  : 'hover:bg-slate-50/90 dark:hover:bg-surface-800/60'
                              } ${isDragTarget ? 'ring-2 ring-emerald-500 bg-emerald-100/70' : ''}`}
                            >
                              {/* Selection Checkbox */}
                              <td className="py-2.5 px-3 w-8" onClick={(e) => e.stopPropagation()}>
                                <input
                                  type="checkbox"
                                  checked={isSelected}
                                  onChange={(e) => {
                                    const next = new Set(selectedPaths);
                                    if (e.target.checked) next.add(file.path);
                                    else next.delete(file.path);
                                    setSelectedPaths(next);
                                  }}
                                  className="rounded text-emerald-600 focus:ring-emerald-500 cursor-pointer"
                                />
                              </td>

                              {/* File / Folder Name with Rich Icon */}
                              <td className="py-2.5 px-3 min-w-[200px]">
                                <div className="flex items-center gap-2.5 min-w-0">
                                  {renderFileIcon(file, 'sm')}
                                  <div className="min-w-0 flex-1">
                                    <span
                                      className="font-bold text-xs text-slate-900 dark:text-white truncate block group-hover:text-emerald-600 dark:group-hover:text-emerald-400 transition-colors"
                                      title={file.name}
                                    >
                                      {file.name}
                                    </span>
                                    {searchResults && (
                                      <span className="block text-[10px] text-slate-400 font-mono truncate max-w-xs">
                                        {file.path}
                                      </span>
                                    )}
                                  </div>
                                </div>
                              </td>

                              {/* File Size */}
                              <td className="py-2.5 px-3 font-mono text-slate-500 dark:text-slate-400 whitespace-nowrap min-w-[75px]">
                                {file.is_dir ? (
                                  <span className="text-slate-400 text-xs">—</span>
                                ) : (
                                  <span className="font-semibold text-slate-700 dark:text-slate-300 text-xs">
                                    {formatBytes(file.size)}
                                  </span>
                                )}
                              </td>

                              {/* Interactive Permissions Pill */}
                              <td className="py-2.5 px-3 font-mono whitespace-nowrap min-w-[85px]">
                                <button
                                  type="button"
                                  onClick={(e) => {
                                    e.stopPropagation();
                                    setPermTargetPaths([file.path]);
                                    setPermMode(file.perm_octal || (file.is_dir ? '0755' : '0644'));
                                    setPermModalOpen(true);
                                  }}
                                  className={`px-2 py-0.5 rounded-md text-[11px] font-semibold transition cursor-pointer flex items-center gap-1 ${
                                    file.perm_octal === '0777' || file.perm_octal === '0666'
                                      ? 'bg-rose-500/10 text-rose-600 dark:text-rose-400 border border-rose-500/30 hover:bg-rose-500/20'
                                      : file.is_dir
                                      ? 'bg-amber-500/10 text-amber-700 dark:text-amber-300 border border-amber-500/25 hover:bg-amber-500/20'
                                      : 'bg-slate-100 dark:bg-surface-800 text-slate-600 dark:text-slate-300 border border-slate-200 dark:border-surface-700 hover:border-emerald-500/40'
                                  }`}
                                  title={`Permissions: ${file.perm_octal || (file.is_dir ? '0755' : '0644')} (Click to edit)`}
                                >
                                  <Lock className="w-2.5 h-2.5 opacity-60" />
                                  <span>{file.perm_octal || (file.is_dir ? '0755' : '0644')}</span>
                                </button>
                              </td>

                              {/* Owner & Group */}
                              <td className="py-2.5 px-3 font-mono text-slate-500 max-w-[130px] truncate" title={`${file.owner}:${file.group}`}>
                                <span className="px-1.5 py-0.5 rounded text-[11px] bg-slate-100 dark:bg-surface-800 text-slate-600 dark:text-slate-400 border border-slate-200 dark:border-surface-700">
                                  {file.owner}:{file.group}
                                </span>
                              </td>

                              {/* Last Modified */}
                              <td className="py-2.5 px-3 text-slate-500 dark:text-slate-400 font-mono text-[11px] whitespace-nowrap min-w-[140px]">
                                {formatDate(file.modified_at)}
                              </td>

                              {/* Actions: Fast Inline Hover Actions + More Button */}
                              <td className="py-2.5 px-3 text-right shrink-0" onClick={(e) => e.stopPropagation()}>
                                <div className="flex items-center justify-end gap-1">
                                  {/* Fast Inline Hover Actions */}
                                  <div className="hidden sm:flex items-center gap-0.5 opacity-0 group-hover:opacity-100 transition-opacity">
                                    {!file.is_dir ? (
                                      <button
                                        type="button"
                                        onClick={() => openFileEditor(file)}
                                        title="Quick Edit File"
                                        className="p-1 rounded-lg text-emerald-600 hover:text-emerald-700 hover:bg-emerald-50 dark:hover:bg-emerald-950/40 transition cursor-pointer"
                                      >
                                        <Edit className="w-3.5 h-3.5" />
                                      </button>
                                    ) : (
                                      <button
                                        type="button"
                                        onClick={() => handleItemClick(file)}
                                        title="Open Folder"
                                        className="p-1 rounded-lg text-emerald-600 hover:text-emerald-700 hover:bg-emerald-50 dark:hover:bg-emerald-950/40 transition cursor-pointer"
                                      >
                                        <Folder className="w-3.5 h-3.5" />
                                      </button>
                                    )}

                                    {!file.is_dir && (
                                      <button
                                        type="button"
                                        onClick={() => {
                                          window.location.href = `/api/v1/files/download?path=${encodeURIComponent(file.path)}`;
                                        }}
                                        title="Download File"
                                        className="p-1 rounded-lg text-slate-500 hover:text-slate-800 dark:hover:text-white hover:bg-slate-100 dark:hover:bg-surface-800 transition cursor-pointer"
                                      >
                                        <Download className="w-3.5 h-3.5" />
                                      </button>
                                    )}

                                    <button
                                      type="button"
                                      onClick={() => {
                                        setItemToRename(file);
                                        setNewName(file.name);
                                        setRenameModalOpen(true);
                                      }}
                                      title="Rename"
                                      className="p-1 rounded-lg text-slate-500 hover:text-slate-800 dark:hover:text-white hover:bg-slate-100 dark:hover:bg-surface-800 transition cursor-pointer"
                                    >
                                      <PenLine className="w-3.5 h-3.5" />
                                    </button>

                                    <button
                                      type="button"
                                      onClick={() => handleMoveToTrash([file.path])}
                                      title="Move to Trash"
                                      className="p-1 rounded-lg text-rose-500 hover:text-rose-600 hover:bg-rose-50 dark:hover:bg-rose-950/40 transition cursor-pointer"
                                    >
                                      <Trash2 className="w-3.5 h-3.5" />
                                    </button>
                                  </div>

                                  {/* Full Context Menu Button */}
                                  <button
                                    type="button"
                                    onClick={(e) => {
                                      const rect = e.currentTarget.getBoundingClientRect();
                                      setContextMenu({
                                        x: rect.left,
                                        y: rect.bottom,
                                        item: file,
                                      });
                                    }}
                                    className="p-1 rounded-lg text-slate-400 hover:text-slate-700 dark:hover:text-white hover:bg-slate-100 dark:hover:bg-surface-800 transition cursor-pointer"
                                    title="More Options"
                                  >
                                    <MoreVertical className="w-4 h-4" />
                                  </button>
                                </div>
                              </td>
                            </tr>
                          );
                        })}
                      </tbody>
                    </table>
                  </div>
                ) : (
                  /* MODERN GRID / CARD VIEW */
                  <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 xl:grid-cols-6 gap-3.5">
                    {displayedItems.map((file) => {
                      const isSelected = selectedPaths.has(file.path);
                      const ext = (file.extension || file.name.split('.').pop() || '').toUpperCase();
                      return (
                        <div
                          key={file.path}
                          draggable
                          onDragStart={() => setDraggedItem(file)}
                          onContextMenu={(e) => {
                            e.preventDefault();
                            setContextMenu({
                              x: e.clientX,
                              y: e.clientY,
                              item: file,
                            });
                          }}
                          onClick={() => {
                            const next = new Set(selectedPaths);
                            if (next.has(file.path)) next.delete(file.path);
                            else next.add(file.path);
                            setSelectedPaths(next);
                            setPreviewItem(file);
                          }}
                          onDoubleClick={() => handleItemClick(file)}
                          className={`group relative p-3 rounded-2xl border flex flex-col justify-between cursor-pointer transition-all duration-150 select-none hover:shadow-md hover:-translate-y-0.5 ${
                            isSelected
                              ? 'border-emerald-500 bg-emerald-50/80 dark:bg-emerald-950/40 shadow-xs'
                              : 'border-slate-200 dark:border-surface-800 bg-white dark:bg-surface-900 hover:border-slate-300 dark:hover:border-surface-700'
                          }`}
                        >
                          {/* Top Row: Checkbox & Type Badge */}
                          <div className="flex items-center justify-between w-full mb-1">
                            <input
                              type="checkbox"
                              checked={isSelected}
                              onChange={(e) => {
                                e.stopPropagation();
                                const next = new Set(selectedPaths);
                                if (e.target.checked) next.add(file.path);
                                else next.delete(file.path);
                                setSelectedPaths(next);
                              }}
                              onClick={(e) => e.stopPropagation()}
                              className={`rounded text-emerald-600 focus:ring-emerald-500 cursor-pointer ${
                                isSelected ? 'opacity-100' : 'opacity-0 group-hover:opacity-100'
                              } transition-opacity`}
                            />
                            <span className="px-1.5 py-0.2 rounded text-[9px] font-mono font-bold bg-slate-100 dark:bg-surface-800 text-slate-500 dark:text-slate-400">
                              {file.is_dir ? 'DIR' : ext || 'FILE'}
                            </span>
                          </div>

                          {/* Center Icon & Filename */}
                          <div className="flex flex-col items-center text-center my-2">
                            <div className="mb-2">
                              {renderFileIcon(file, 'md')}
                            </div>
                            <span
                              className="font-bold text-xs text-slate-900 dark:text-white truncate w-full group-hover:text-emerald-600 dark:group-hover:text-emerald-400 transition-colors"
                              title={file.name}
                            >
                              {file.name}
                            </span>
                            <span className="text-[10px] text-slate-400 font-mono mt-0.5">
                              {file.is_dir ? 'Folder' : formatBytes(file.size)}
                            </span>
                          </div>

                          {/* Bottom Row: Quick Action Bar visible on hover */}
                          <div className="mt-1 pt-1.5 border-t border-slate-100 dark:border-surface-800 flex items-center justify-between text-[10px] text-slate-400 font-mono">
                            <span>{file.perm_octal || (file.is_dir ? '0755' : '0644')}</span>
                            <div className="flex items-center gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
                              {!file.is_dir && (
                                <button
                                  type="button"
                                  onClick={(e) => {
                                    e.stopPropagation();
                                    openFileEditor(file);
                                  }}
                                  title="Edit"
                                  className="p-0.5 hover:text-emerald-600"
                                >
                                  <Edit className="w-3 h-3" />
                                </button>
                              )}
                              <button
                                type="button"
                                onClick={(e) => {
                                  e.stopPropagation();
                                  const rect = e.currentTarget.getBoundingClientRect();
                                  setContextMenu({
                                    x: rect.left,
                                    y: rect.bottom,
                                    item: file,
                                  });
                                }}
                                title="More"
                                className="p-0.5 hover:text-slate-700 dark:hover:text-white"
                              >
                                <MoreVertical className="w-3 h-3" />
                              </button>
                            </div>
                          </div>
                        </div>
                      );
                    })}
                  </div>
                )}
              </div>
            )}
          </div>
        </main>

        {/* 3. RIGHT SIDE FILE PREVIEW PANEL (Phase 12) */}
        {previewOpen && previewItem && (
          <FilePreviewPanel
            item={previewItem}
            onClose={() => setPreviewOpen(false)}
            onEdit={(f) => openFileEditor(f)}
            onDownload={(f) => {
              window.location.href = `/api/v1/files/download?path=${encodeURIComponent(f.path)}`;
            }}
            onDelete={(f) => handleMoveToTrash([f.path])}
            onChmod={(f) => {
              setPermTargetPaths([f.path]);
              setPermMode(f.perm_octal || '0755');
              setPermModalOpen(true);
            }}
            onRename={(f) => {
              setItemToRename(f);
              setNewName(f.name);
              setRenameModalOpen(true);
            }}
            activeDomain={activeDomain}
          />
        )}

        {/* 4. CONTEXT MENU (Phase 3) */}
        {contextMenu && (
          <FileContextMenu
            x={contextMenu.x}
            y={contextMenu.y}
            item={contextMenu.item}
            onClose={() => setContextMenu(null)}
            onOpen={(item) => handleItemClick(item)}
            onEdit={(item) => openFileEditor(item)}
            onOpenNewTab={(item) => {
              if (item.is_dir) {
                window.open(`/files?path=${encodeURIComponent(item.path)}`, '_blank');
              } else {
                window.open(`/api/v1/files/download?path=${encodeURIComponent(item.path)}`, '_blank');
              }
            }}
            onRename={(item) => {
              setItemToRename(item);
              setNewName(item.name);
              setRenameModalOpen(true);
            }}
            onCopy={(item) => {
              setClipboard({ paths: [item.path], action: 'copy' });
              showToast(`Copied ${item.name} to clipboard`);
            }}
            onCut={(item) => {
              setClipboard({ paths: [item.path], action: 'cut' });
              showToast(`Cut ${item.name} to clipboard`);
            }}
            onDuplicate={async (item) => {
              const res = await apiFetch('/api/v1/filemanager/copy', {
                method: 'POST',
                body: JSON.stringify({
                  src_path: item.path,
                  dest_path: currentPath,
                  conflict_strategy: 'rename',
                }),
              });
              if (res.success) {
                showToast(`Duplicated ${item.name}`);
                fetchDirectory(currentPath);
              }
            }}
            onCompress={(item) => {
              setArchiveTargetPaths([item.path]);
              setArchiveName(`${item.name}.zip`);
              setArchiveModalOpen(true);
            }}
            onExtract={(item) => {
              setExtractArchiveItem(item);
              setExtractDestDir(currentPath);
              setExtractModalOpen(true);
            }}
            onDownload={(item) => {
              window.location.href = `/api/v1/files/download?path=${encodeURIComponent(item.path)}`;
            }}
            onFavorite={async (item) => {
              const res = await apiFetch('/api/v1/filemanager/favorite', {
                method: 'POST',
                body: JSON.stringify({
                  path: item.path,
                  name: item.name,
                  domain: activeDomain,
                  color: 'emerald',
                }),
              });
              if (res.success) {
                showToast(`Added ${item.name} to Favorites`);
                setRefreshKey((k) => k + 1);
              }
            }}
            onLabelColor={async (item, color) => {
              await apiFetch('/api/v1/filemanager/label', {
                method: 'POST',
                body: JSON.stringify({ path: item.path, color, domain: activeDomain }),
              });
              showToast(`Tagged with ${color}`);
              setRefreshKey((k) => k + 1);
            }}
            onDelete={(item) => handleMoveToTrash([item.path])}
            onProperties={(item) => {
              setPreviewItem(item);
              setPreviewOpen(true);
            }}
            onChmod={(item) => {
              setPermTargetPaths([item.path]);
              setPermMode(item.perm_octal || '0755');
              setPermModalOpen(true);
            }}
          />
        )}

        {/* 5. UPLOAD MANAGER FLOATING DRAWER (Phase 15) */}
        <UploadManagerDrawer
          tasks={uploadTasks}
          onCancel={(id) => {
            setUploadTasks((prev) =>
              prev.map((t) => (t.id === id ? { ...t, status: 'cancelled' } : t))
            );
          }}
          onRetry={(id) => {
            const task = uploadTasks.find((t) => t.id === id);
            if (task) processUploadTask(task);
          }}
          onClearCompleted={() => {
            setUploadTasks((prev) => prev.filter((t) => t.status !== 'completed'));
          }}
        />

        {/* 6. MODALS */}

        {/* ADVANCED PROFESSIONAL CODE EDITOR MODAL */}
        <AdvancedCodeEditor
          isOpen={editorOpen}
          file={editingFile}
          content={fileContent}
          onContentChange={setFileContent}
          onSave={handleSaveContent}
          onClose={() => setEditorOpen(false)}
          saving={savingFile}
          saveSuccess={saveSuccess}
        />

        {/* NEW FILE / FOLDER MODAL */}
        {createModalOpen && (
          <div className="fixed inset-0 z-50 bg-black/60 backdrop-blur-xs flex items-center justify-center p-4 animate-fadeIn">
            <div className="w-full max-w-md bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-700 rounded-2xl shadow-2xl p-6 relative">
              <button
                type="button"
                onClick={() => setCreateModalOpen(false)}
                className="absolute top-5 right-5 p-1 rounded-lg text-slate-400 hover:text-slate-700"
              >
                <X className="w-5 h-5" />
              </button>

              <h3 className="font-bold text-base text-slate-900 dark:text-white mb-1">
                Create New {createType === 'file' ? 'File' : 'Folder'}
              </h3>
              <p className="text-xs text-slate-500 mb-4">Location: {currentPath}</p>

              <form
                onSubmit={async (e) => {
                  e.preventDefault();
                  if (!createName.trim()) return;
                  try {
                    setCreating(true);
                    const target = `${currentPath}/${createName.trim()}`;
                    const endpoint =
                      createType === 'file' ? '/api/v1/files/content' : '/api/v1/files/mkdir';
                    const method = createType === 'file' ? 'PUT' : 'POST';
                    const body =
                      createType === 'file'
                        ? JSON.stringify({ path: target, content: '' })
                        : JSON.stringify({ path: target });

                    const res = await apiFetch(endpoint, { method, body });
                    if (res.success) {
                      showToast(`Created ${createType} ${createName}`);
                      setCreateModalOpen(false);
                      fetchDirectory(currentPath);
                    } else {
                      showToast(res.error?.message || 'Creation failed', true);
                    }
                  } catch (err: any) {
                    showToast(err.message || 'Creation failed', true);
                  } finally {
                    setCreating(false);
                  }
                }}
                className="space-y-4 text-xs"
              >
                <div>
                  <label className="font-bold text-slate-800 dark:text-white block mb-1">
                    {createType === 'file' ? 'File Name' : 'Folder Name'}
                  </label>
                  <input
                    type="text"
                    value={createName}
                    onChange={(e) => setCreateName(e.target.value)}
                    placeholder={createType === 'file' ? 'index.php' : 'uploads'}
                    autoFocus
                    required
                    className="w-full px-3 py-2 rounded-xl bg-slate-50 dark:bg-surface-800 border border-slate-300 dark:border-surface-700 font-mono text-xs focus:ring-2 focus:ring-emerald-500"
                  />
                </div>

                <div className="flex items-center justify-end gap-2 pt-2">
                  <button
                    type="button"
                    onClick={() => setCreateModalOpen(false)}
                    className="px-4 py-2 rounded-xl border border-slate-200 dark:border-surface-700 font-bold"
                  >
                    Cancel
                  </button>
                  <button
                    type="submit"
                    disabled={creating}
                    className="px-5 py-2 rounded-xl bg-emerald-600 hover:bg-emerald-700 text-white font-bold disabled:opacity-50"
                  >
                    {creating ? 'Creating...' : 'Create'}
                  </button>
                </div>
              </form>
            </div>
          </div>
        )}

        {/* RENAME MODAL */}
        {renameModalOpen && itemToRename && (
          <div className="fixed inset-0 z-50 bg-black/60 backdrop-blur-xs flex items-center justify-center p-4 animate-fadeIn">
            <div className="w-full max-w-md bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-700 rounded-2xl shadow-2xl p-6 relative">
              <button
                type="button"
                onClick={() => setRenameModalOpen(false)}
                className="absolute top-5 right-5 p-1 rounded-lg text-slate-400 hover:text-slate-700"
              >
                <X className="w-5 h-5" />
              </button>

              <h3 className="font-bold text-base text-slate-900 dark:text-white mb-1">Rename Item</h3>
              <p className="text-xs text-slate-500 mb-4">{itemToRename.path}</p>

              <form
                onSubmit={async (e) => {
                  e.preventDefault();
                  if (!newName.trim() || newName === itemToRename.name) return;
                  const newPath = `${currentPath}/${newName.trim()}`;
                  const res = await apiFetch('/api/v1/files/rename', {
                    method: 'POST',
                    body: JSON.stringify({
                      old_path: itemToRename.path,
                      new_path: newPath,
                    }),
                  });
                  if (res.success) {
                    showToast(`Renamed to ${newName}`);
                    setRenameModalOpen(false);
                    fetchDirectory(currentPath);
                  } else {
                    showToast(res.error?.message || 'Rename failed', true);
                  }
                }}
                className="space-y-4 text-xs"
              >
                <div>
                  <label className="font-bold text-slate-800 dark:text-white block mb-1">
                    New Name
                  </label>
                  <input
                    type="text"
                    value={newName}
                    onChange={(e) => setNewName(e.target.value)}
                    autoFocus
                    required
                    className="w-full px-3 py-2 rounded-xl bg-slate-50 dark:bg-surface-800 border border-slate-300 dark:border-surface-700 font-mono text-xs focus:ring-2 focus:ring-emerald-500"
                  />
                </div>

                <div className="flex items-center justify-end gap-2 pt-2">
                  <button
                    type="button"
                    onClick={() => setRenameModalOpen(false)}
                    className="px-4 py-2 rounded-xl border border-slate-200 font-bold"
                  >
                    Cancel
                  </button>
                  <button
                    type="submit"
                    className="px-5 py-2 rounded-xl bg-emerald-600 hover:bg-emerald-700 text-white font-bold"
                  >
                    Save Changes
                  </button>
                </div>
              </form>
            </div>
          </div>
        )}

        {/* CHMOD PERMISSIONS MODAL */}
        {permModalOpen && permTargetPaths.length > 0 && (
          <div className="fixed inset-0 z-50 bg-black/60 backdrop-blur-xs flex items-center justify-center p-4 animate-fadeIn">
            <div className="w-full max-w-md bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-700 rounded-2xl shadow-2xl p-6 relative">
              <button
                type="button"
                onClick={() => setPermModalOpen(false)}
                className="absolute top-5 right-5 p-1 rounded-lg text-slate-400 hover:text-slate-700"
              >
                <X className="w-5 h-5" />
              </button>

              <h3 className="font-bold text-base text-slate-900 dark:text-white mb-1">
                Change Permissions (chmod)
              </h3>
              <p className="text-xs text-slate-500 mb-4">
                Applying to {permTargetPaths.length} items
              </p>

              <form
                onSubmit={async (e) => {
                  e.preventDefault();
                  try {
                    setPermSaving(true);
                    const res = await apiFetch('/api/v1/files/permissions', {
                      method: 'POST',
                      body: JSON.stringify({
                        paths: permTargetPaths,
                        mode: permMode,
                      }),
                    });
                    if (res.success) {
                      showToast(`Permissions updated to ${permMode}`);
                      setPermModalOpen(false);
                      fetchDirectory(currentPath);
                    } else {
                      showToast(res.error?.message || 'Failed to update permissions', true);
                    }
                  } finally {
                    setPermSaving(false);
                  }
                }}
                className="space-y-4 text-xs"
              >
                <div>
                  <label className="font-bold text-slate-800 dark:text-white block mb-1">
                    Octal Permission Mode (e.g. 0755, 0644)
                  </label>
                  <input
                    type="text"
                    value={permMode}
                    onChange={(e) => setPermMode(e.target.value)}
                    required
                    className="w-full px-3 py-2 rounded-xl bg-slate-50 dark:bg-surface-800 border border-slate-300 dark:border-surface-700 font-mono text-xs focus:ring-2 focus:ring-emerald-500"
                  />
                </div>

                <div className="grid grid-cols-3 gap-2">
                  {['0755', '0644', '0777', '0750', '0600', '0700'].map((preset) => (
                    <button
                      key={preset}
                      type="button"
                      onClick={() => setPermMode(preset)}
                      className={`p-2 rounded-lg border font-mono font-bold transition ${
                        permMode === preset
                          ? 'border-emerald-600 bg-emerald-50 text-emerald-700'
                          : 'border-slate-200 hover:bg-slate-50'
                      }`}
                    >
                      {preset}
                    </button>
                  ))}
                </div>

                <div className="flex items-center justify-end gap-2 pt-2">
                  <button
                    type="button"
                    onClick={() => setPermModalOpen(false)}
                    className="px-4 py-2 rounded-xl border border-slate-200 font-bold"
                  >
                    Cancel
                  </button>
                  <button
                    type="submit"
                    disabled={permSaving}
                    className="px-5 py-2 rounded-xl bg-emerald-600 hover:bg-emerald-700 text-white font-bold disabled:opacity-50"
                  >
                    {permSaving ? 'Updating...' : 'Apply Permissions'}
                  </button>
                </div>
              </form>
            </div>
          </div>
        )}

        {/* EXTRACT ARCHIVE MODAL */}
        {extractModalOpen && extractArchiveItem && (
          <div className="fixed inset-0 z-50 bg-black/60 backdrop-blur-xs flex items-center justify-center p-4 animate-fadeIn">
            <div className="w-full max-w-md bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-700 rounded-2xl shadow-2xl p-6 relative">
              <button
                type="button"
                onClick={() => setExtractModalOpen(false)}
                className="absolute top-5 right-5 p-1 rounded-lg text-slate-400 hover:text-slate-700 dark:hover:text-white"
              >
                <X className="w-5 h-5" />
              </button>

              <div className="flex items-center gap-3 mb-4">
                <div className="w-10 h-10 rounded-xl bg-indigo-50 dark:bg-indigo-950/40 text-indigo-600 dark:text-indigo-400 flex items-center justify-center shrink-0">
                  <FolderArchive className="w-5 h-5" />
                </div>
                <div>
                  <h3 className="font-bold text-base text-slate-900 dark:text-white">
                    Extract Archive
                  </h3>
                  <p className="text-xs text-slate-500">
                    Uncompress archive contents into destination folder
                  </p>
                </div>
              </div>

              <form
                onSubmit={async (e) => {
                  e.preventDefault();
                  if (!extractArchiveItem) return;
                  try {
                    setExtracting(true);
                    const res = await apiFetch<{ extracted: boolean; dest_dir: string }>('/api/v1/files/extract', {
                      method: 'POST',
                      body: JSON.stringify({
                        archive_path: extractArchiveItem.path,
                        dest_dir: extractDestDir || currentPath,
                      }),
                    });
                    if (res.success) {
                      showToast(`Extracted ${extractArchiveItem.name} successfully!`);
                      setExtractModalOpen(false);
                      fetchDirectory(currentPath);
                    } else {
                      showToast(res.error?.message || 'Extraction failed', true);
                    }
                  } catch (err: any) {
                    showToast(err.message || 'Extraction failed', true);
                  } finally {
                    setExtracting(false);
                  }
                }}
                className="space-y-4 text-xs"
              >
                <div>
                  <label className="font-bold text-slate-800 dark:text-white block mb-1">
                    Archive File:
                  </label>
                  <div className="p-2.5 rounded-xl bg-slate-50 dark:bg-surface-800 border border-slate-200 dark:border-surface-700 font-mono text-xs text-slate-700 dark:text-slate-300 break-all font-semibold flex items-center gap-2">
                    <FolderArchive className="w-4 h-4 text-indigo-500 shrink-0" />
                    <span>{extractArchiveItem.name}</span>
                  </div>
                </div>

                <div>
                  <label className="font-bold text-slate-800 dark:text-white block mb-1">
                    Extract Destination Directory:
                  </label>
                  <input
                    type="text"
                    value={extractDestDir}
                    onChange={(e) => setExtractDestDir(e.target.value)}
                    required
                    className="w-full px-3 py-2 rounded-xl bg-slate-50 dark:bg-surface-800 border border-slate-300 dark:border-surface-700 font-mono text-xs focus:ring-2 focus:ring-indigo-500 text-slate-900 dark:text-white"
                  />
                  <p className="text-[11px] text-slate-400 mt-1">
                    Files will be extracted directly into this directory.
                  </p>
                </div>

                <div className="flex items-center justify-end gap-2 pt-2">
                  <button
                    type="button"
                    onClick={() => setExtractModalOpen(false)}
                    className="px-4 py-2 rounded-xl border border-slate-200 dark:border-surface-700 font-bold hover:bg-slate-50 dark:hover:bg-surface-800 transition text-slate-700 dark:text-slate-300"
                  >
                    Cancel
                  </button>
                  <button
                    type="submit"
                    disabled={extracting}
                    className="px-5 py-2 rounded-xl bg-indigo-600 hover:bg-indigo-700 text-white font-bold transition flex items-center gap-1.5 shadow-xs cursor-pointer disabled:opacity-50"
                  >
                    {extracting ? (
                      <>
                        <RefreshCw className="w-3.5 h-3.5 animate-spin" />
                        <span>Extracting...</span>
                      </>
                    ) : (
                      <>
                        <FolderArchive className="w-3.5 h-3.5" />
                        <span>Extract Here</span>
                      </>
                    )}
                  </button>
                </div>
              </form>
            </div>
          </div>
        )}

        {/* COMPRESS ARCHIVE MODAL */}
        {archiveModalOpen && archiveTargetPaths.length > 0 && (
          <div className="fixed inset-0 z-50 bg-black/60 backdrop-blur-xs flex items-center justify-center p-4 animate-fadeIn">
            <div className="w-full max-w-md bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-700 rounded-2xl shadow-2xl p-6 relative">
              <button
                type="button"
                onClick={() => setArchiveModalOpen(false)}
                className="absolute top-5 right-5 p-1 rounded-lg text-slate-400 hover:text-slate-700 dark:hover:text-white"
              >
                <X className="w-5 h-5" />
              </button>

              <div className="flex items-center gap-3 mb-4">
                <div className="w-10 h-10 rounded-xl bg-purple-50 dark:bg-purple-950/40 text-purple-600 dark:text-purple-400 flex items-center justify-center shrink-0">
                  <Archive className="w-5 h-5" />
                </div>
                <div>
                  <h3 className="font-bold text-base text-slate-900 dark:text-white">
                    Compress to Archive
                  </h3>
                  <p className="text-xs text-slate-500">
                    Create a ZIP archive of selected items
                  </p>
                </div>
              </div>

              <form
                onSubmit={async (e) => {
                  e.preventDefault();
                  try {
                    setArchiveProcessing(true);
                    const cleanName = archiveName.endsWith('.zip') ? archiveName : `${archiveName}.zip`;
                    const destPath = `${currentPath}/${cleanName}`;
                    const res = await apiFetch('/api/v1/files/archive', {
                      method: 'POST',
                      body: JSON.stringify({
                        paths: archiveTargetPaths,
                        dest_path: destPath,
                        format: 'zip',
                      }),
                    });
                    if (res.success) {
                      showToast(`Compressed ${archiveTargetPaths.length} item(s) to ${cleanName}`);
                      setArchiveModalOpen(false);
                      fetchDirectory(currentPath);
                    } else {
                      showToast(res.error?.message || 'Compression failed', true);
                    }
                  } catch (err: any) {
                    showToast(err.message || 'Compression failed', true);
                  } finally {
                    setArchiveProcessing(false);
                  }
                }}
                className="space-y-4 text-xs"
              >
                <div>
                  <label className="font-bold text-slate-800 dark:text-white block mb-1">
                    Items to Compress:
                  </label>
                  <div className="p-2.5 rounded-xl bg-slate-50 dark:bg-surface-800 border border-slate-200 dark:border-surface-700 text-xs text-slate-700 dark:text-slate-300">
                    <span className="font-bold">{archiveTargetPaths.length}</span> item(s) selected
                  </div>
                </div>

                <div>
                  <label className="font-bold text-slate-800 dark:text-white block mb-1">
                    Archive File Name:
                  </label>
                  <input
                    type="text"
                    value={archiveName}
                    onChange={(e) => setArchiveName(e.target.value)}
                    required
                    className="w-full px-3 py-2 rounded-xl bg-slate-50 dark:bg-surface-800 border border-slate-300 dark:border-surface-700 font-mono text-xs focus:ring-2 focus:ring-purple-500 text-slate-900 dark:text-white"
                  />
                </div>

                <div className="flex items-center justify-end gap-2 pt-2">
                  <button
                    type="button"
                    onClick={() => setArchiveModalOpen(false)}
                    className="px-4 py-2 rounded-xl border border-slate-200 dark:border-surface-700 font-bold hover:bg-slate-50 dark:hover:bg-surface-800 transition text-slate-700 dark:text-slate-300"
                  >
                    Cancel
                  </button>
                  <button
                    type="submit"
                    disabled={archiveProcessing}
                    className="px-5 py-2 rounded-xl bg-purple-600 hover:bg-purple-700 text-white font-bold transition flex items-center gap-1.5 shadow-xs cursor-pointer disabled:opacity-50"
                  >
                    {archiveProcessing ? (
                      <>
                        <RefreshCw className="w-3.5 h-3.5 animate-spin" />
                        <span>Compressing...</span>
                      </>
                    ) : (
                      <>
                        <Archive className="w-3.5 h-3.5" />
                        <span>Create ZIP</span>
                      </>
                    )}
                  </button>
                </div>
              </form>
            </div>
          </div>
        )}

        {/* RESTORE WIZARD MODAL (Phase 8) */}
        {restoreWizardOpen && itemsToRestore.length > 0 && (
          <RestoreWizardModal
            items={itemsToRestore}
            domains={domains}
            onClose={() => setRestoreWizardOpen(false)}
            onSuccess={(count) => {
              showToast(`Restored ${count} items successfully!`);
              fetchTrash();
              setRefreshKey((k) => k + 1);
            }}
          />
        )}

        {/* EMPTY TRASH MODAL (Phase 7) */}
        {emptyTrashModalOpen && (
          <EmptyTrashModal
            totalFiles={trashStats.totalFiles}
            totalFolders={trashStats.totalFolders}
            totalSize={trashStats.totalSize}
            domain={activeDomain}
            onClose={() => setEmptyTrashModalOpen(false)}
            onSuccess={(freed, count) => {
              showToast(`Trash permanently emptied! Freed ${formatBytes(freed)} across ${count} items.`);
              fetchTrash();
              setRefreshKey((k) => k + 1);
            }}
          />
        )}

        {/* MULTI-DOMAIN MOVE / COPY MODAL (Phase 4) */}
        {moveCopyModalOpen && moveCopyItems.length > 0 && (
          <MultiDomainMoveCopyModal
            mode={moveCopyMode}
            items={moveCopyItems}
            currentDomain={activeDomain}
            domains={domains}
            onClose={() => setMoveCopyModalOpen(false)}
            onSuccess={(m, count) => {
              showToast(`${m === 'move' ? 'Moved' : 'Copied'} ${count} items successfully!`);
              fetchDirectory(currentPath);
            }}
          />
        )}

        {/* STORAGE INFO MODAL (Phase 17) */}
        {storageModalOpen && (
          <StorageInfoModal
            currentPath={currentPath}
            domain={activeDomain}
            onClose={() => setStorageModalOpen(false)}
          />
        )}

        {/* ACTIVITY LOGS MODAL (Phase 13) */}
        {activityModalOpen && (
          <ActivityLogsModal
            domain={activeDomain}
            onClose={() => setActivityModalOpen(false)}
          />
        )}

        {/* FOLDER DELETE CONFIRMATION MODAL */}
        {folderToDelete && (
          <div className="fixed inset-0 z-50 bg-black/60 backdrop-blur-xs flex items-center justify-center p-4 animate-fadeIn">
            <div className="w-full max-w-md bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-700 rounded-2xl shadow-2xl p-6 relative">
              <button
                type="button"
                onClick={() => setFolderToDelete(null)}
                className="absolute top-5 right-5 p-1 rounded-lg text-slate-400 hover:text-slate-700 dark:hover:text-white"
              >
                <X className="w-5 h-5" />
              </button>

              <div className="flex items-center gap-3 mb-4">
                <div className="w-10 h-10 rounded-xl bg-rose-50 dark:bg-rose-950/40 text-rose-600 flex items-center justify-center shrink-0">
                  <Trash2 className="w-5 h-5" />
                </div>
                <div>
                  <h3 className="font-bold text-base text-slate-900 dark:text-white">
                    Move Folder to Trash
                  </h3>
                  <p className="text-xs text-slate-500">
                    Enterprise Trash Bin Protection
                  </p>
                </div>
              </div>

              <div className="p-3 rounded-xl bg-slate-50 dark:bg-surface-800 border border-slate-200 dark:border-surface-700 mb-4">
                <div className="text-xs font-semibold text-slate-700 dark:text-slate-300 mb-1">
                  Folder to Delete:
                </div>
                <div className="text-xs font-mono text-rose-600 dark:text-rose-400 break-all font-semibold">
                  {folderToDelete}
                </div>
              </div>

              <p className="text-xs text-slate-600 dark:text-slate-300 mb-5 leading-relaxed">
                This folder and all its contents will be moved to the <strong>Trash Bin</strong>. You can restore it anytime from the Trash Bin view.
              </p>

              <div className="flex items-center justify-end gap-2">
                <button
                  type="button"
                  onClick={() => setFolderToDelete(null)}
                  className="px-4 py-2 rounded-xl text-xs font-bold text-slate-700 dark:text-slate-300 hover:bg-slate-100 dark:hover:bg-surface-800 transition cursor-pointer"
                >
                  Cancel
                </button>
                <button
                  type="button"
                  onClick={async () => {
                    const target = folderToDelete;
                    setFolderToDelete(null);
                    await handleMoveToTrash([target]);
                    if (currentPath === target || currentPath.startsWith(target + '/')) {
                      navigateTo('/var/www');
                    }
                  }}
                  className="px-4 py-2 rounded-xl text-xs font-bold bg-rose-600 hover:bg-rose-700 text-white transition flex items-center gap-1.5 shadow-xs cursor-pointer"
                >
                  <Trash2 className="w-3.5 h-3.5" />
                  <span>Move to Trash</span>
                </button>
              </div>
            </div>
          </div>
        )}

        {/* TOAST POPUP */}
        {toast && (
          <div
            className={`fixed bottom-6 left-1/2 -translate-x-1/2 z-50 px-4 py-2.5 rounded-xl shadow-2xl text-xs font-bold flex items-center gap-2 animate-slideUp ${
              toast.isError
                ? 'bg-rose-600 text-white'
                : 'bg-slate-900 text-white border border-slate-700'
            }`}
          >
            {toast.isError ? (
              <AlertCircle className="w-4 h-4 text-rose-200" />
            ) : (
              <CheckCircle2 className="w-4 h-4 text-emerald-400" />
            )}
            <span>{toast.message}</span>
          </div>
        )}

        {/* KEYBOARD SHORTCUTS MODAL */}
        <KeyboardShortcutsModal
          isOpen={shortcutsModalOpen}
          onClose={() => setShortcutsModalOpen(false)}
        />
      </div>
    </DashboardShell>
  );
}
