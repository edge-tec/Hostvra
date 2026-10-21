'use client';

import React, { Suspense, useState, useEffect, useCallback, useMemo, useRef } from 'react';
import { useSearchParams, useRouter } from 'next/navigation';
import Link from 'next/link';
import {
  Database as DatabaseIcon,
  Table,
  Play,
  Download,
  Upload,
  Search,
  Settings,
  RefreshCw,
  ExternalLink,
  Plus,
  Trash2,
  Edit3,
  CheckCircle2,
  AlertCircle,
  Copy,
  Check,
  Eye,
  ArrowLeft,
  Server,
  Zap,
  Terminal,
  Home,
  LogOut,
  HelpCircle,
  Star,
  ChevronRight,
  ChevronDown,
  Columns,
  Key,
  Folder,
  Sliders,
  Filter,
  Layers,
  Clock,
  Code2,
  FileCode,
  Maximize2,
  Minimize2,
  X,
  MoreHorizontal,
  FolderTree,
  Shield,
  FileText,
  Workflow,
  ArrowUpDown,
  SortAsc,
  SortDesc,
  DatabaseZap,
  RotateCcw,
  Sparkles,
  Activity,
  HardDrive,
  ListFilter,
  CheckSquare,
  Square
} from 'lucide-react';
import { DashboardShell } from '@/components/DashboardShell';
import { apiFetch, Database } from '@/lib/api';

type TabType =
  | 'structure'
  | 'sql'
  | 'search'
  | 'export'
  | 'import'
  | 'operations'
  | 'routines'
  | 'events'
  | 'triggers'
  | 'views'
  | 'privileges'
  | 'browse';

// Types for live schema tree
interface TableNode {
  name: string;
  rows: number;
}

interface TriggerNode {
  name: string;
  table: string;
}

interface DatabaseTreeNode {
  name: string;
  tables: TableNode[];
  views: string[];
  procedures: string[];
  functions: string[];
  events: string[];
  triggers: TriggerNode[];
}

// Table overview detail
interface TableDetail {
  name: string;
  type: string;
  engine: string;
  collation: string;
  rows: number;
  data_size_kb: number;
  index_size_kb: number;
  total_size_kb: number;
  comment: string;
}

// Full table column schema
interface ColumnDetail {
  field: string;
  type: string;
  collation?: string;
  null: string;
  key: string;
  default?: string;
  extra: string;
  privileges: string;
  comment: string;
}

// Table index
interface IndexDetail {
  key_name: string;
  non_unique: number;
  column_name: string;
  index_type: string;
  seq_in_index: number;
  comment: string;
}

// Table foreign key
interface ForeignKeyDetail {
  constraint_name: string;
  column_name: string;
  ref_table: string;
  ref_column: string;
  on_update: string;
  on_delete: string;
}

// Table structure API response
interface TableStructureData {
  table: string;
  columns: ColumnDetail[];
  indexes: IndexDetail[];
  foreign_keys: ForeignKeyDetail[];
}

// Browse rows result
interface BrowseRowsData {
  columns: string[];
  rows: Record<string, any>[];
  primary_key: string;
  page: number;
  limit: number;
  total_rows: number;
  total_pages: number;
  execution_time: string;
}

// Routines, Events, Triggers, Views
interface RoutineItem {
  name: string;
  type: 'PROCEDURE' | 'FUNCTION';
  data_type?: string;
  parameters?: string;
  definition: string;
  security?: string;
}

interface EventItem {
  name: string;
  status: string;
  type: string;
  interval?: string;
  starts?: string;
  definition: string;
}

interface TriggerItem {
  name: string;
  table: string;
  timing: string;
  event: string;
  statement: string;
  definer?: string;
}

interface ViewItem {
  name: string;
  definition: string;
  check_option: string;
  is_updatable: string;
  security_type: string;
}

interface UserPrivilegeItem {
  user: string;
  host: string;
  type: string;
  privileges: string;
  grant: boolean;
}

function PhpMyAdminCore() {
  const searchParams = useSearchParams();
  const router = useRouter();
  const dbParam = searchParams.get('db') || '';

  // Core Database & Selection State
  const [currentDb, setCurrentDb] = useState<string>(dbParam);
  const [selectedTable, setSelectedTable] = useState<string>('');
  const [activeTab, setActiveTab] = useState<TabType>('structure');

  // Tree Navigator State
  const [treeNodes, setTreeNodes] = useState<DatabaseTreeNode[]>([]);
  const [loadingTree, setLoadingTree] = useState<boolean>(false);
  const [treeSearch, setTreeSearch] = useState<string>('');
  const [expandedDbs, setExpandedDbs] = useState<Record<string, boolean>>({});
  const [expandedSections, setExpandedSections] = useState<Record<string, boolean>>({});

  // Database Overview / Tables List State
  const [tableDetails, setTableDetails] = useState<TableDetail[]>([]);
  const [loadingTableDetails, setLoadingTableDetails] = useState<boolean>(false);
  const [selectedTableNames, setSelectedTableNames] = useState<string[]>([]);
  const [tableFilterWord, setTableFilterWord] = useState<string>('');
  const [tableSortCol, setTableSortCol] = useState<'name' | 'rows' | 'engine' | 'collation' | 'size'>('name');
  const [tableSortOrder, setTableSortOrder] = useState<'ASC' | 'DESC'>('ASC');
  const [treeFilterType, setTreeFilterType] = useState<'all' | 'tables' | 'views'>('all');
  const [hudExpanded, setHudExpanded] = useState<boolean>(true);

  // Table Structure State
  const [tableStructure, setTableStructure] = useState<TableStructureData | null>(null);
  const [loadingStructure, setLoadingStructure] = useState<boolean>(false);

  // Browse Rows State
  const [browseData, setBrowseData] = useState<BrowseRowsData | null>(null);
  const [loadingBrowse, setLoadingBrowse] = useState<boolean>(false);
  const [browsePage, setBrowsePage] = useState<number>(1);
  const [browseLimit, setBrowseLimit] = useState<number>(25);
  const [browseSortCol, setBrowseSortCol] = useState<string>('');
  const [browseSortOrder, setBrowseSortOrder] = useState<'ASC' | 'DESC'>('ASC');
  const [browseSearch, setBrowseSearch] = useState<string>('');

  // Interactive SQL Query State
  const [sqlQuery, setSqlQuery] = useState<string>('');
  const [runningQuery, setRunningQuery] = useState<boolean>(false);
  const [queryResult, setQueryResult] = useState<{
    columns: string[];
    rows: Record<string, any>[];
    rows_affected: number;
    execution_time: string;
    error?: string;
  } | null>(null);
  const [queryError, setQueryError] = useState<string | null>(null);

  // Routines, Events, Triggers, Views State
  const [routines, setRoutines] = useState<RoutineItem[]>([]);
  const [events, setEvents] = useState<EventItem[]>([]);
  const [triggers, setTriggers] = useState<TriggerItem[]>([]);
  const [views, setViews] = useState<ViewItem[]>([]);
  const [loadingEntities, setLoadingEntities] = useState<boolean>(false);

  // Privileges State
  const [privileges, setPrivileges] = useState<UserPrivilegeItem[]>([]);
  const [loadingPrivileges, setLoadingPrivileges] = useState<boolean>(false);

  // Export State
  const [exportFormat, setExportFormat] = useState<'sql' | 'csv' | 'json'>('sql');
  const [exportSelectedTables, setExportSelectedTables] = useState<string[]>([]);

  // Import State
  const [importing, setImporting] = useState<boolean>(false);
  const [importResult, setImportResult] = useState<{
    successful: number;
    failed: number;
    total: number;
    tables?: number;
    errors?: string[];
  } | null>(null);
  const [importFile, setImportFile] = useState<File | null>(null);
  const [importSqlText, setImportSqlText] = useState<string>('');
  const [isDraggingImport, setIsDraggingImport] = useState<boolean>(false);
  const [isDraggingOverPage, setIsDraggingOverPage] = useState<boolean>(false);
  const [importProgress, setImportProgress] = useState<number>(0);
  const [importStatusText, setImportStatusText] = useState<string>('');

  // Operations State
  const [opCollation, setOpCollation] = useState<string>('utf8mb4_unicode_ci');
  const [opRenameTable, setOpRenameTable] = useState<string>('');
  const [opCopyTable, setOpCopyTable] = useState<string>('');
  const [opCopyData, setOpCopyData] = useState<boolean>(true);
  const [opMaintenanceMsg, setOpMaintenanceMsg] = useState<string | null>(null);
  const [opRunning, setOpRunning] = useState<boolean>(false);

  // Modals State
  const [createDbModalOpen, setCreateDbModalOpen] = useState(false);
  const [newDbName, setNewDbName] = useState('');
  const [newDbCollation, setNewDbCollation] = useState('utf8mb4_unicode_ci');

  const [createTableModalOpen, setCreateTableModalOpen] = useState(false);
  const [newTableName, setNewTableName] = useState('');
  const [newTableColsCount, setNewTableColsCount] = useState(4);

  const [insertRowModalOpen, setInsertRowModalOpen] = useState(false);
  const [insertRowValues, setInsertRowValues] = useState<Record<string, string>>({});

  const [editRowModalOpen, setEditRowModalOpen] = useState(false);
  const [editRowOriginal, setEditRowOriginal] = useState<Record<string, any> | null>(null);
  const [editRowValues, setEditRowValues] = useState<Record<string, string>>({});

  const [addColumnModalOpen, setAddColumnModalOpen] = useState(false);
  const [newColName, setNewColName] = useState('');
  const [newColType, setNewColType] = useState('varchar(255)');
  const [newColCollation, setNewColCollation] = useState('utf8mb4_unicode_ci');
  const [newColNull, setNewColNull] = useState('YES');
  const [newColDefault, setNewColDefault] = useState('');
  const [newColExtra, setNewColExtra] = useState('');
  const [newColAfter, setNewColAfter] = useState('');

  const [modifyColModalOpen, setModifyColModalOpen] = useState(false);
  const [modColTarget, setModColTarget] = useState<ColumnDetail | null>(null);
  const [modColName, setModColName] = useState('');
  const [modColType, setModColType] = useState('');
  const [modColCollation, setModColCollation] = useState('');
  const [modColNull, setModColNull] = useState('YES');
  const [modColDefault, setModColDefault] = useState('');
  const [modColExtra, setModColExtra] = useState('');

  const [addIndexModalOpen, setAddIndexModalOpen] = useState(false);
  const [newIndexName, setNewIndexName] = useState('');
  const [newIndexType, setNewIndexType] = useState<'PRIMARY' | 'INDEX' | 'UNIQUE' | 'FULLTEXT'>('INDEX');
  const [newIndexColumns, setNewIndexColumns] = useState<string[]>([]);

  const [addRoutineModalOpen, setAddRoutineModalOpen] = useState(false);
  const [routineName, setRoutineName] = useState('');
  const [routineType, setRoutineType] = useState<'PROCEDURE' | 'FUNCTION'>('PROCEDURE');
  const [routineParams, setRoutineParams] = useState('');
  const [routineReturns, setRoutineReturns] = useState('');
  const [routineBody, setRoutineBody] = useState('BEGIN\n  -- routine SQL body\nEND');

  const [addEventModalOpen, setAddEventModalOpen] = useState(false);
  const [eventName, setEventName] = useState('');
  const [eventSchedule, setEventSchedule] = useState('EVERY 1 DAY');
  const [eventBody, setEventBody] = useState('DO BEGIN\n  -- scheduled SQL\nEND');

  const [addTriggerModalOpen, setAddTriggerModalOpen] = useState(false);
  const [triggerName, setTriggerName] = useState('');
  const [triggerTable, setTriggerTable] = useState('');
  const [triggerTiming, setTriggerTiming] = useState<'BEFORE' | 'AFTER'>('AFTER');
  const [triggerEvent, setTriggerEvent] = useState<'INSERT' | 'UPDATE' | 'DELETE'>('INSERT');
  const [triggerBody, setTriggerBody] = useState('BEGIN\n  -- trigger logic\nEND');

  const [addUserModalOpen, setAddUserModalOpen] = useState(false);
  const [newUsername, setNewUsername] = useState('');
  const [newUserHost, setNewUserHost] = useState('localhost');
  const [newUserPassword, setNewUserPassword] = useState('');

  const [toastMessage, setToastMessage] = useState<string | null>(null);
  const showToast = (msg: string, _isError?: boolean) => {
    setToastMessage(msg);
    setTimeout(() => setToastMessage(null), 4000);
  };

  // 1. Fetch Real Database Hierarchy Tree
  const fetchTree = useCallback(async () => {
    setLoadingTree(true);
    try {
      const res = await apiFetch<DatabaseTreeNode[]>('/api/v1/databases/tree');
      if (res && res.success && Array.isArray(res.data)) {
        setTreeNodes(res.data);
        // If currentDb is empty or invalid, default to first non-system database
        if (!currentDb && res.data.length > 0) {
          const first = res.data.find(
            (d) => d.name !== 'information_schema' && d.name !== 'mysql' && d.name !== 'performance_schema' && d.name !== 'sys'
          ) || res.data[0];
          setCurrentDb(first.name);
          setExpandedDbs((prev) => ({ ...prev, [first.name]: true }));
        } else if (currentDb) {
          setExpandedDbs((prev) => ({ ...prev, [currentDb]: true }));
        }
      }
    } catch {
      // Tree network error
    } finally {
      setLoadingTree(false);
    }
  }, [currentDb]);

  useEffect(() => {
    fetchTree();
  }, [fetchTree]);

  // 2. Fetch Live Tables / Overview for Active Database
  const fetchTableDetails = useCallback(async (db: string) => {
    if (!db) return;
    setLoadingTableDetails(true);
    try {
      const res = await apiFetch<{ database: string; tables: TableDetail[] }>(
        `/api/v1/databases/tables/details?db=${encodeURIComponent(db)}`
      );
      if (res && res.success && res.data && Array.isArray(res.data.tables)) {
        setTableDetails(res.data.tables);
        setExportSelectedTables(res.data.tables.map((t) => t.name));
      } else {
        setTableDetails([]);
      }
    } catch {
      setTableDetails([]);
    } finally {
      setLoadingTableDetails(false);
    }
  }, []);

  // 3. Fetch Full Table Structure (Columns, Indexes, Foreign Keys)
  const fetchTableStructure = useCallback(async (db: string, tbl: string) => {
    if (!db || !tbl) return;
    setLoadingStructure(true);
    try {
      const res = await apiFetch<TableStructureData>(
        `/api/v1/databases/tables/structure?db=${encodeURIComponent(db)}&table=${encodeURIComponent(tbl)}`
      );
      if (res && res.success && res.data) {
        setTableStructure(res.data);
      } else {
        setTableStructure(null);
      }
    } catch {
      setTableStructure(null);
    } finally {
      setLoadingStructure(false);
    }
  }, []);

  // 4. Fetch Paginated Rows for Browse Tab
  const fetchBrowseRows = useCallback(async (db: string, tbl: string, page = 1, limit = 25, sortCol = '', sortOrder = 'ASC', search = '') => {
    if (!db || !tbl) return;
    setLoadingBrowse(true);
    try {
      const q = new URLSearchParams({
        db,
        table: tbl,
        page: String(page),
        limit: String(limit),
        sort_column: sortCol,
        sort_order: sortOrder,
        search,
      });
      const res = await apiFetch<BrowseRowsData>(`/api/v1/databases/tables/rows?${q.toString()}`);
      if (res && res.success && res.data) {
        setBrowseData(res.data);
        setBrowsePage(res.data.page);
        setBrowseLimit(res.data.limit);
      } else {
        setBrowseData(null);
      }
    } catch {
      setBrowseData(null);
    } finally {
      setLoadingBrowse(false);
    }
  }, []);

  // 5. Fetch Routines, Events, Triggers, Views
  const fetchAuxEntities = useCallback(async (db: string) => {
    if (!db) return;
    setLoadingEntities(true);
    try {
      const [rRes, eRes, tRes, vRes] = await Promise.all([
        apiFetch<RoutineItem[]>(`/api/v1/databases/routines?db=${encodeURIComponent(db)}`),
        apiFetch<EventItem[]>(`/api/v1/databases/events?db=${encodeURIComponent(db)}`),
        apiFetch<TriggerItem[]>(`/api/v1/databases/triggers?db=${encodeURIComponent(db)}`),
        apiFetch<ViewItem[]>(`/api/v1/databases/views?db=${encodeURIComponent(db)}`),
      ]);
      if (rRes && rRes.data) setRoutines(rRes.data);
      if (eRes && eRes.data) setEvents(eRes.data);
      if (tRes && tRes.data) setTriggers(tRes.data);
      if (vRes && vRes.data) setViews(vRes.data);
    } catch {
      // Ignored
    } finally {
      setLoadingEntities(false);
    }
  }, []);

  // 6. Fetch User Privileges
  const fetchPrivileges = useCallback(async (db: string) => {
    setLoadingPrivileges(true);
    try {
      const res = await apiFetch<{ columns: string[]; rows: Record<string, string>[] }>('/api/v1/databases/query', {
        method: 'POST',
        body: JSON.stringify({
          database: 'mysql',
          query: 'SELECT User as user, Host as host, Select_priv, Insert_priv, Update_priv, Delete_priv, Create_priv, Drop_priv, Grant_priv FROM mysql.user ORDER BY User ASC;',
        }),
      });
      if (res && res.data && Array.isArray(res.data.rows)) {
        const mapped: UserPrivilegeItem[] = res.data.rows.map((r) => {
          const privs: string[] = [];
          if (r.Select_priv === 'Y') privs.push('SELECT');
          if (r.Insert_priv === 'Y') privs.push('INSERT');
          if (r.Update_priv === 'Y') privs.push('UPDATE');
          if (r.Delete_priv === 'Y') privs.push('DELETE');
          if (r.Create_priv === 'Y') privs.push('CREATE');
          if (r.Drop_priv === 'Y') privs.push('DROP');
          return {
            user: r.user,
            host: r.host,
            type: 'global',
            privileges: privs.length === 6 ? 'ALL PRIVILEGES' : privs.join(', ') || 'USAGE',
            grant: r.Grant_priv === 'Y',
          };
        });
        setPrivileges(mapped);
      }
    } catch {
      setPrivileges([]);
    } finally {
      setLoadingPrivileges(false);
    }
  }, []);

  // Synchronize on currentDb change
  useEffect(() => {
    if (currentDb) {
      fetchTableDetails(currentDb);
      fetchAuxEntities(currentDb);
      if (activeTab === 'privileges') fetchPrivileges(currentDb);
    }
  }, [currentDb, fetchTableDetails, fetchAuxEntities, fetchPrivileges, activeTab]);

  // Synchronize on selectedTable change
  useEffect(() => {
    if (currentDb && selectedTable) {
      if (activeTab === 'structure') {
        fetchTableStructure(currentDb, selectedTable);
      } else if (activeTab === 'browse') {
        fetchBrowseRows(currentDb, selectedTable, browsePage, browseLimit, browseSortCol, browseSortOrder, browseSearch);
      }
    }
  }, [currentDb, selectedTable, activeTab, fetchTableStructure, fetchBrowseRows, browsePage, browseLimit, browseSortCol, browseSortOrder, browseSearch]);

  // Global drag-and-drop prevention + full-page drop overlay
  useEffect(() => {
    let dragCounter = 0;
    const handleDragOver = (e: DragEvent) => {
      e.preventDefault();
      e.stopPropagation();
    };
    const handleDragEnter = (e: DragEvent) => {
      e.preventDefault();
      e.stopPropagation();
      dragCounter++;
      if (e.dataTransfer?.types?.includes('Files')) {
        setIsDraggingOverPage(true);
      }
    };
    const handleDragLeave = (e: DragEvent) => {
      e.preventDefault();
      e.stopPropagation();
      dragCounter--;
      if (dragCounter <= 0) {
        dragCounter = 0;
        setIsDraggingOverPage(false);
      }
    };
    const handleDrop = (e: DragEvent) => {
      e.preventDefault();
      e.stopPropagation();
      dragCounter = 0;
      setIsDraggingOverPage(false);
      if (e.dataTransfer?.files && e.dataTransfer.files.length > 0) {
        const file = e.dataTransfer.files[0];
        const name = file.name.toLowerCase();
        if (name.endsWith('.sql') || name.endsWith('.txt') || name.endsWith('.csv')) {
          setImportFile(file);
          setActiveTab('import');
        }
      }
    };
    document.addEventListener('dragover', handleDragOver);
    document.addEventListener('dragenter', handleDragEnter);
    document.addEventListener('dragleave', handleDragLeave);
    document.addEventListener('drop', handleDrop);
    return () => {
      document.removeEventListener('dragover', handleDragOver);
      document.removeEventListener('dragenter', handleDragEnter);
      document.removeEventListener('dragleave', handleDragLeave);
      document.removeEventListener('drop', handleDrop);
    };
  }, []);

  // Switch Active Database
  const handleSwitchDb = (db: string) => {
    setCurrentDb(db);
    setSelectedTable('');
    setSelectedTableNames([]);
    router.replace(`/phpmyadmin?db=${encodeURIComponent(db)}`);
    setExpandedDbs((prev) => ({ ...prev, [db]: true }));
  };

  // Open Table in Browse Tab
  const handleOpenTableBrowse = (tbl: string) => {
    setSelectedTable(tbl);
    setActiveTab('browse');
    setBrowsePage(1);
    setBrowseSearch('');
    setBrowseSortCol('');
  };

  // Open Table in Structure Tab
  const handleOpenTableStructure = (tbl: string) => {
    setSelectedTable(tbl);
    setActiveTab('structure');
  };

  // Run SQL Statement
  const handleExecuteSql = async (overrideQuery?: string) => {
    const q = (overrideQuery !== undefined ? overrideQuery : sqlQuery).trim();
    if (!q) {
      setQueryError('Please enter a SQL statement to execute.');
      return;
    }
    setQueryError(null);
    setRunningQuery(true);
    try {
      const res = await apiFetch<any>('/api/v1/databases/query', {
        method: 'POST',
        body: JSON.stringify({
          database: currentDb,
          query: q,
        }),
      });
      if (res && res.data) {
        if (res.data.error) {
          setQueryError(res.data.error);
          setQueryResult(null);
        } else {
          setQueryResult(res.data);
          showToast(`Query executed in ${res.data.execution_time || '0.001s'}`);
          // Refresh schema tree and tables if DDL/DML
          const upper = q.toUpperCase();
          if (
            upper.includes('CREATE') ||
            upper.includes('DROP') ||
            upper.includes('ALTER') ||
            upper.includes('TRUNCATE') ||
            upper.includes('INSERT') ||
            upper.includes('UPDATE') ||
            upper.includes('DELETE')
          ) {
            fetchTree();
            fetchTableDetails(currentDb);
            if (selectedTable) {
              fetchTableStructure(currentDb, selectedTable);
              fetchBrowseRows(currentDb, selectedTable, browsePage, browseLimit, browseSortCol, browseSortOrder, browseSearch);
            }
          }
        }
      } else {
        setQueryError('Database query execution failed.');
      }
    } catch (err: any) {
      setQueryError(err?.message || 'Database query error');
    } finally {
      setRunningQuery(false);
    }
  };

  // Drop / Delete Database Functionality
  const handleDropDatabase = async (dbNameToDrop?: string) => {
    const target = dbNameToDrop || currentDb;
    if (!target) return;
    if (['mysql', 'information_schema', 'performance_schema', 'sys'].includes(target.toLowerCase())) {
      showToast('Cannot drop MySQL system databases.');
      return;
    }

    if (!confirm(`Are you sure you want to permanently DROP database '${target}'? All tables and data will be destroyed from MySQL immediately.`)) {
      return;
    }

    try {
      const res = await apiFetch<any>('/api/v1/databases/query', {
        method: 'POST',
        body: JSON.stringify({
          database: target === currentDb ? 'mysql' : currentDb,
          query: `DROP DATABASE IF EXISTS \`${target}\`;`,
        }),
      });

      if (res && res.data && res.data.error) {
        showToast(`Failed to drop database: ${res.data.error}`);
        return;
      }

      showToast(`Database '${target}' successfully dropped.`);

      // Also clean up Hostvra metadata store if it was registered there
      try {
        const allMetaRes = await apiFetch<any[]>('/api/v1/databases');
        if (allMetaRes && Array.isArray(allMetaRes.data)) {
          const match = allMetaRes.data.find((d: any) => d && d.name === target);
          if (match && match.id) {
            await apiFetch(`/api/v1/databases/${match.id}`, { method: 'DELETE' });
          }
        }
      } catch (e) {
        // metadata cleanup error can be ignored
      }

      // Immediately refresh tree from live MySQL schema
      const treeRes = await apiFetch<DatabaseTreeNode[]>('/api/v1/databases/tree');
      if (treeRes && treeRes.success && Array.isArray(treeRes.data)) {
        setTreeNodes(treeRes.data);
        if (currentDb === target) {
          const nextDb = treeRes.data.find(
            (d) => d.name !== 'information_schema' && d.name !== 'mysql' && d.name !== 'performance_schema' && d.name !== 'sys'
          ) || treeRes.data[0];
          if (nextDb) {
            setCurrentDb(nextDb.name);
            setSelectedTable('');
            fetchTableDetails(nextDb.name);
          } else {
            setCurrentDb('');
            setSelectedTable('');
            setTableDetails([]);
          }
        }
      }
    } catch (err: any) {
      showToast(err?.message || 'Error dropping database');
    }
  };

  // Row Delete in Browse Tab
  const handleDeleteRow = async (row: Record<string, any>) => {
    if (!browseData || !browseData.primary_key) {
      showToast('Cannot delete: No primary key detected for table.');
      return;
    }
    const pk = browseData.primary_key;
    const pkVal = row[pk];
    if (pkVal === undefined) return;

    if (!confirm(`Do you really want to DELETE row where ${pk} = ${pkVal}?`)) return;

    try {
      const res = await apiFetch<any>('/api/v1/databases/tables/rows', {
        method: 'DELETE',
        body: JSON.stringify({
          database: currentDb,
          table: selectedTable,
          primary_key_col: pk,
          primary_key_vals: [pkVal],
        }),
      });
      if (res && res.success) {
        showToast(`Row deleted successfully.`);
        fetchBrowseRows(currentDb, selectedTable, browsePage, browseLimit, browseSortCol, browseSortOrder, browseSearch);
        fetchTableDetails(currentDb);
      } else {
        showToast(res?.error?.message || 'Failed to delete row.');
      }
    } catch (e: any) {
      showToast(e?.message || 'Delete error');
    }
  };

  // Open Edit Modal for a Row
  const handleOpenEditRow = (row: Record<string, any>) => {
    setEditRowOriginal(row);
    const initialVals: Record<string, string> = {};
    for (const col of browseData?.columns || []) {
      initialVals[col] = row[col] !== undefined && row[col] !== null ? String(row[col]) : '';
    }
    setEditRowValues(initialVals);
    setEditRowModalOpen(true);
  };

  // Submit Row Edit
  const handleSubmitEditRow = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!browseData || !browseData.primary_key || !editRowOriginal) return;

    const pk = browseData.primary_key;
    const pkVal = editRowOriginal[pk];

    try {
      const res = await apiFetch<any>('/api/v1/databases/tables/rows', {
        method: 'PUT',
        body: JSON.stringify({
          database: currentDb,
          table: selectedTable,
          primary_key_col: pk,
          primary_key_val: pkVal,
          values: editRowValues,
        }),
      });
      if (res && res.success) {
        showToast('Row updated successfully.');
        setEditRowModalOpen(false);
        fetchBrowseRows(currentDb, selectedTable, browsePage, browseLimit, browseSortCol, browseSortOrder, browseSearch);
      } else {
        showToast(res?.error?.message || 'Failed to update row.');
      }
    } catch (err: any) {
      showToast(err?.message || 'Update row error');
    }
  };

  // Open Insert Row Modal
  const handleOpenInsertRow = () => {
    const initialVals: Record<string, string> = {};
    for (const col of browseData?.columns || tableStructure?.columns.map((c) => c.field) || []) {
      initialVals[col] = '';
    }
    setInsertRowValues(initialVals);
    setInsertRowModalOpen(true);
  };

  // Submit Row Insert
  const handleSubmitInsertRow = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      const res = await apiFetch<any>('/api/v1/databases/tables/rows', {
        method: 'POST',
        body: JSON.stringify({
          database: currentDb,
          table: selectedTable,
          values: insertRowValues,
        }),
      });
      if (res && res.success) {
        showToast(`Row inserted successfully. Insert ID: ${res.data?.last_insert_id || 'OK'}`);
        setInsertRowModalOpen(false);
        fetchBrowseRows(currentDb, selectedTable, browsePage, browseLimit, browseSortCol, browseSortOrder, browseSearch);
        fetchTableDetails(currentDb);
      } else {
        showToast(res?.error?.message || 'Failed to insert row.');
      }
    } catch (err: any) {
      showToast(err?.message || 'Insert row error');
    }
  };

  // Drop Column
  const handleDropColumn = async (colName: string) => {
    if (!confirm(`Do you really want to DROP column \`${colName}\` from table \`${selectedTable}\`? All data in this column will be lost!`)) return;

    try {
      const res = await apiFetch<any>('/api/v1/databases/columns/modify', {
        method: 'POST',
        body: JSON.stringify({
          database: currentDb,
          table: selectedTable,
          action: 'drop',
          column: colName,
        }),
      });
      if (res && res.success) {
        showToast(`Column \`${colName}\` dropped.`);
        fetchTableStructure(currentDb, selectedTable);
        fetchBrowseRows(currentDb, selectedTable, browsePage, browseLimit, browseSortCol, browseSortOrder, browseSearch);
      } else {
        showToast(res?.error?.message || 'Failed to drop column');
      }
    } catch (e: any) {
      showToast(e?.message || 'Drop column error');
    }
  };

  // Submit Add Column
  const handleAddColumnSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!newColName.trim()) return;

    try {
      const res = await apiFetch<any>('/api/v1/databases/columns/modify', {
        method: 'POST',
        body: JSON.stringify({
          database: currentDb,
          table: selectedTable,
          action: 'add',
          column: newColName.trim(),
          type_def: newColType,
          collation: newColCollation,
          null: newColNull,
          default: newColDefault,
          extra: newColExtra,
          after_col: newColAfter,
        }),
      });
      if (res && res.success) {
        showToast(`Column \`${newColName}\` added successfully.`);
        setAddColumnModalOpen(false);
        setNewColName('');
        fetchTableStructure(currentDb, selectedTable);
        fetchBrowseRows(currentDb, selectedTable, browsePage, browseLimit, browseSortCol, browseSortOrder, browseSearch);
      } else {
        showToast(res?.error?.message || 'Failed to add column');
      }
    } catch (e: any) {
      showToast(e?.message || 'Add column error');
    }
  };

  // Open Modify Column Modal
  const handleOpenModifyCol = (col: ColumnDetail) => {
    setModColTarget(col);
    setModColName(col.field);
    setModColType(col.type);
    setModColCollation(col.collation || 'utf8mb4_unicode_ci');
    setModColNull(col.null);
    setModColDefault(col.default || '');
    setModColExtra(col.extra || '');
    setModifyColModalOpen(true);
  };

  // Submit Modify Column
  const handleModifyColumnSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!modColTarget) return;

    try {
      const res = await apiFetch<any>('/api/v1/databases/columns/modify', {
        method: 'POST',
        body: JSON.stringify({
          database: currentDb,
          table: selectedTable,
          action: 'modify',
          column: modColTarget.field,
          new_name: modColName.trim(),
          type_def: modColType,
          collation: modColCollation,
          null: modColNull,
          default: modColDefault,
          extra: modColExtra,
        }),
      });
      if (res && res.success) {
        showToast(`Column \`${modColName}\` modified successfully.`);
        setModifyColModalOpen(false);
        fetchTableStructure(currentDb, selectedTable);
        fetchBrowseRows(currentDb, selectedTable, browsePage, browseLimit, browseSortCol, browseSortOrder, browseSearch);
      } else {
        showToast(res?.error?.message || 'Failed to modify column');
      }
    } catch (e: any) {
      showToast(e?.message || 'Modify column error');
    }
  };

  // Drop Index
  const handleDropIndex = async (idxName: string) => {
    if (!confirm(`Do you really want to DROP index \`${idxName}\` from table \`${selectedTable}\`?`)) return;

    try {
      const res = await apiFetch<any>('/api/v1/databases/indexes/modify', {
        method: 'POST',
        body: JSON.stringify({
          database: currentDb,
          table: selectedTable,
          action: 'drop',
          index_name: idxName,
        }),
      });
      if (res && res.success) {
        showToast(`Index \`${idxName}\` dropped.`);
        fetchTableStructure(currentDb, selectedTable);
      } else {
        showToast(res?.error?.message || 'Failed to drop index');
      }
    } catch (e: any) {
      showToast(e?.message || 'Drop index error');
    }
  };

  // Submit Add Index
  const handleAddIndexSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!newIndexName.trim() && newIndexType !== 'PRIMARY') return;
    if (newIndexColumns.length === 0) {
      showToast('Select at least one column for index');
      return;
    }

    try {
      const res = await apiFetch<any>('/api/v1/databases/indexes/modify', {
        method: 'POST',
        body: JSON.stringify({
          database: currentDb,
          table: selectedTable,
          action: 'add',
          index_name: newIndexName.trim(),
          index_type: newIndexType,
          columns: newIndexColumns,
        }),
      });
      if (res && res.success) {
        showToast(`Index created successfully.`);
        setAddIndexModalOpen(false);
        setNewIndexName('');
        setNewIndexColumns([]);
        fetchTableStructure(currentDb, selectedTable);
      } else {
        showToast(res?.error?.message || 'Failed to add index');
      }
    } catch (e: any) {
      showToast(e?.message || 'Add index error');
    }
  };

  // Table Operations: Rename, Copy, Truncate, Drop, Maintenance
  const handleTableOperation = async (action: string) => {
    if (!selectedTable && action !== 'collation') {
      showToast('Please select a table first.');
      return;
    }

    if (action === 'truncate' && !confirm(`TRUNCATE table \`${selectedTable}\`? All rows will be permanently deleted!`)) return;
    if (action === 'drop' && !confirm(`DROP table \`${selectedTable}\`? The table and all data will be permanently removed!`)) return;

    setOpRunning(true);
    setOpMaintenanceMsg(null);
    try {
      const res = await apiFetch<any>('/api/v1/databases/tables/operations', {
        method: 'POST',
        body: JSON.stringify({
          database: currentDb,
          table: selectedTable,
          action,
          new_name: action === 'rename' ? opRenameTable : action === 'copy' ? opCopyTable : '',
          copy_data: opCopyData,
          collation: opCollation,
        }),
      });
      if (res && res.success) {
        if (res.data?.maintenance) {
          setOpMaintenanceMsg(res.data.maintenance);
        }
        showToast(`Operation ${action.toUpperCase()} completed successfully.`);

        if (action === 'rename') {
          setSelectedTable(opRenameTable);
          setOpRenameTable('');
        } else if (action === 'drop') {
          setSelectedTable('');
          setActiveTab('structure');
        }
        fetchTree();
        fetchTableDetails(currentDb);
        if (selectedTable && action !== 'drop') {
          fetchTableStructure(currentDb, selectedTable);
          fetchBrowseRows(currentDb, selectedTable, browsePage, browseLimit, browseSortCol, browseSortOrder, browseSearch);
        }
      } else {
        showToast(res?.error?.message || `Operation failed.`);
      }
    } catch (e: any) {
      showToast(e?.message || 'Operation error');
    } finally {
      setOpRunning(false);
    }
  };

  // Export File Download
  const handleTriggerExport = () => {
    const tblParam = exportSelectedTables.length > 0 ? exportSelectedTables.join(',') : '';
    const url = `/api/v1/databases/export?db=${encodeURIComponent(currentDb)}&format=${exportFormat}&tables=${encodeURIComponent(tblParam)}`;
    const link = document.createElement('a');
    link.href = url;
    link.setAttribute('download', `${currentDb}_export.${exportFormat}`);
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
    showToast(`Export started for ${currentDb} (${exportFormat.toUpperCase()})`);
  };

  // Import SQL via Multipart or Raw Query
  const handleStartImport = async () => {
    if (!importFile && !importSqlText.trim()) {
      showToast('Please select a .sql file or enter SQL commands.');
      return;
    }

    setImporting(true);
    setImportResult(null);
    setImportProgress(5);
    setImportStatusText('Preparing import...');

    const progressTimer = setInterval(() => {
      setImportProgress((prev) => {
        if (prev < 20) return prev + 5;
        if (prev < 50) return prev + 3;
        if (prev < 80) return prev + 2;
        if (prev < 95) return prev + 0.5;
        return prev;
      });
    }, 500);

    try {
      if (importFile) {
        setImportStatusText(`Uploading & restoring ${importFile.name} (${(importFile.size / (1024 * 1024)).toFixed(1)} MB)...`);
        const formData = new FormData();
        formData.append('file', importFile);
        const res = await apiFetch<any>(`/api/v1/databases/import?db=${encodeURIComponent(currentDb)}`, {
          method: 'POST',
          body: formData,
        });
        clearInterval(progressTimer);
        setImportProgress(100);

        if (res && res.success) {
          setImportStatusText(`Import complete! ${res.data?.successful || 0} statements executed.`);
          setImportResult(res.data);
          showToast(`Import finished: ${res.data?.successful || 0} executed, ${res.data?.failed || 0} failed.`);
          fetchTree();
          fetchTableDetails(currentDb);
        } else {
          setImportStatusText('Import failed.');
          showToast(res?.error?.message || 'Import failed.');
        }
      } else if (importSqlText.trim()) {
        setImportStatusText('Executing raw SQL statements...');
        // For raw SQL, use the import endpoint with text/plain Content-Type
        const res = await apiFetch<any>(`/api/v1/databases/import?db=${encodeURIComponent(currentDb)}`, {
          method: 'POST',
          headers: { 'Content-Type': 'text/plain' },
          body: importSqlText,
        });
        clearInterval(progressTimer);
        setImportProgress(100);

        if (res && res.success) {
          setImportStatusText(`Execution complete! ${res.data?.successful || 0} statements executed.`);
          setImportResult(res.data);
          showToast(`SQL executed: ${res.data?.successful || 0} succeeded, ${res.data?.failed || 0} failed.`);
          fetchTree();
          fetchTableDetails(currentDb);
        } else {
          setImportStatusText('SQL execution failed.');
          showToast(res?.error?.message || res?.data?.error || 'SQL execution failed.');
        }
      }
    } catch (e: any) {
      clearInterval(progressTimer);
      setImportProgress(0);
      setImportStatusText('Import error.');
      showToast(e?.message || 'Import error');
    } finally {
      clearInterval(progressTimer);
      setTimeout(() => {
        setImporting(false);
      }, 600);
    }
  };

  // Create Database
  const handleCreateDatabase = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!newDbName.trim()) return;
    const clean = newDbName.trim().toLowerCase().replace(/[^a-z0-9_]/g, '_');
    try {
      const res = await apiFetch<any>('/api/v1/databases/query', {
        method: 'POST',
        body: JSON.stringify({
          database: 'information_schema',
          query: `CREATE DATABASE \`${clean}\` CHARACTER SET utf8mb4 COLLATE ${newDbCollation};`,
        }),
      });
      if (res && res.data && !res.data.error) {
        showToast(`Database \`${clean}\` created successfully.`);
        setCreateDbModalOpen(false);
        setNewDbName('');
        await fetchTree();
        handleSwitchDb(clean);
      } else {
        showToast(res?.data?.error || 'Failed to create database');
      }
    } catch (e: any) {
      showToast(e?.message || 'Database creation error');
    }
  };

  // Create Table
  const handleCreateTable = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!newTableName.trim()) return;
    const clean = newTableName.trim().toLowerCase().replace(/[^a-z0-9_]/g, '_');
    try {
      const query = `CREATE TABLE \`${clean}\` (
        \`id\` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
        \`name\` varchar(255) NOT NULL,
        \`created_at\` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
        \`updated_at\` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
        PRIMARY KEY (\`id\`)
      ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;`;

      const res = await apiFetch<any>('/api/v1/databases/query', {
        method: 'POST',
        body: JSON.stringify({
          database: currentDb,
          query,
        }),
      });
      if (res && res.data && !res.data.error) {
        showToast(`Table \`${clean}\` created.`);
        setCreateTableModalOpen(false);
        setNewTableName('');
        await fetchTree();
        fetchTableDetails(currentDb);
        handleOpenTableStructure(clean);
      } else {
        showToast(res?.data?.error || 'Failed to create table');
      }
    } catch (e: any) {
      showToast(e?.message || 'Table creation error');
    }
  };

  // Create Stored Routine
  const handleCreateRoutine = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!routineName.trim()) return;
    try {
      let q = '';
      if (routineType === 'PROCEDURE') {
        q = `CREATE PROCEDURE \`${routineName.trim()}\`(${routineParams}) \n${routineBody};`;
      } else {
        q = `CREATE FUNCTION \`${routineName.trim()}\`(${routineParams}) RETURNS ${routineReturns || 'INT'} \n${routineBody};`;
      }
      const res = await apiFetch<any>('/api/v1/databases/query', {
        method: 'POST',
        body: JSON.stringify({ database: currentDb, query: q }),
      });
      if (res && res.data && !res.data.error) {
        showToast(`Routine \`${routineName}\` created.`);
        setAddRoutineModalOpen(false);
        setRoutineName('');
        fetchAuxEntities(currentDb);
        fetchTree();
      } else {
        showToast(res?.data?.error || 'Failed to create routine');
      }
    } catch (e: any) {
      showToast(e?.message || 'Create routine error');
    }
  };

  // Drop Routine
  const handleDropRoutine = async (r: RoutineItem) => {
    if (!confirm(`Do you really want to DROP ${r.type} \`${r.name}\`?`)) return;
    try {
      const q = `DROP ${r.type} IF EXISTS \`${r.name}\`;`;
      const res = await apiFetch<any>('/api/v1/databases/query', {
        method: 'POST',
        body: JSON.stringify({ database: currentDb, query: q }),
      });
      if (res && res.data && !res.data.error) {
        showToast(`${r.type} \`${r.name}\` dropped.`);
        fetchAuxEntities(currentDb);
        fetchTree();
      } else {
        showToast(res?.data?.error || 'Failed to drop routine');
      }
    } catch (e: any) {
      showToast(e?.message || 'Drop routine error');
    }
  };

  // Create Event
  const handleCreateEvent = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!eventName.trim()) return;
    try {
      const q = `CREATE EVENT \`${eventName.trim()}\` ON SCHEDULE ${eventSchedule} ${eventBody};`;
      const res = await apiFetch<any>('/api/v1/databases/query', {
        method: 'POST',
        body: JSON.stringify({ database: currentDb, query: q }),
      });
      if (res && res.data && !res.data.error) {
        showToast(`Event \`${eventName}\` created.`);
        setAddEventModalOpen(false);
        setEventName('');
        fetchAuxEntities(currentDb);
        fetchTree();
      } else {
        showToast(res?.data?.error || 'Failed to create event');
      }
    } catch (e: any) {
      showToast(e?.message || 'Create event error');
    }
  };

  // Drop Event
  const handleDropEvent = async (evName: string) => {
    if (!confirm(`Drop event \`${evName}\`?`)) return;
    try {
      const q = `DROP EVENT IF EXISTS \`${evName}\`;`;
      const res = await apiFetch<any>('/api/v1/databases/query', {
        method: 'POST',
        body: JSON.stringify({ database: currentDb, query: q }),
      });
      if (res && res.data && !res.data.error) {
        showToast(`Event \`${evName}\` dropped.`);
        fetchAuxEntities(currentDb);
        fetchTree();
      }
    } catch (e: any) {
      showToast(e?.message || 'Drop event error');
    }
  };

  // Create Trigger
  const handleCreateTrigger = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!triggerName.trim() || !triggerTable.trim()) return;
    try {
      const q = `CREATE TRIGGER \`${triggerName.trim()}\` ${triggerTiming} ${triggerEvent} ON \`${triggerTable.trim()}\` FOR EACH ROW \n${triggerBody};`;
      const res = await apiFetch<any>('/api/v1/databases/query', {
        method: 'POST',
        body: JSON.stringify({ database: currentDb, query: q }),
      });
      if (res && res.data && !res.data.error) {
        showToast(`Trigger \`${triggerName}\` created.`);
        setAddTriggerModalOpen(false);
        setTriggerName('');
        fetchAuxEntities(currentDb);
        fetchTree();
      } else {
        showToast(res?.data?.error || 'Failed to create trigger');
      }
    } catch (e: any) {
      showToast(e?.message || 'Create trigger error');
    }
  };

  // Drop Trigger
  const handleDropTrigger = async (trgName: string) => {
    if (!confirm(`Drop trigger \`${trgName}\`?`)) return;
    try {
      const q = `DROP TRIGGER IF EXISTS \`${trgName}\`;`;
      const res = await apiFetch<any>('/api/v1/databases/query', {
        method: 'POST',
        body: JSON.stringify({ database: currentDb, query: q }),
      });
      if (res && res.data && !res.data.error) {
        showToast(`Trigger \`${trgName}\` dropped.`);
        fetchAuxEntities(currentDb);
        fetchTree();
      }
    } catch (e: any) {
      showToast(e?.message || 'Drop trigger error');
    }
  };

  // Filtered tree nodes
  const filteredTreeNodes = useMemo(() => {
    const term = treeSearch.trim().toLowerCase();
    return treeNodes
      .map((node) => {
        const matchDb = !term || node.name.toLowerCase().includes(term);
        const matchTables = (treeFilterType === 'views' ? [] : (node.tables || [])).filter(
          (t) => !term || matchDb || t.name.toLowerCase().includes(term)
        );
        const matchViews = (treeFilterType === 'tables' ? [] : (node.views || [])).filter(
          (v) => !term || matchDb || v.toLowerCase().includes(term)
        );
        const matchProcs = (treeFilterType !== 'all' ? [] : (node.procedures || [])).filter(
          (p) => !term || matchDb || p.toLowerCase().includes(term)
        );
        const matchFuncs = (treeFilterType !== 'all' ? [] : (node.functions || [])).filter(
          (f) => !term || matchDb || f.toLowerCase().includes(term)
        );
        const matchEvents = (treeFilterType !== 'all' ? [] : (node.events || [])).filter(
          (e) => !term || matchDb || e.toLowerCase().includes(term)
        );
        const matchTriggers = (treeFilterType !== 'all' ? [] : (node.triggers || [])).filter(
          (tr) => !term || matchDb || tr.name.toLowerCase().includes(term)
        );

        if (
          matchDb ||
          matchTables.length > 0 ||
          matchViews.length > 0 ||
          matchProcs.length > 0 ||
          matchFuncs.length > 0 ||
          matchEvents.length > 0 ||
          matchTriggers.length > 0
        ) {
          return {
            ...node,
            tables: matchTables,
            views: matchViews,
            procedures: matchProcs,
            functions: matchFuncs,
            events: matchEvents,
            triggers: matchTriggers,
          };
        }
        return null;
      })
      .filter((n): n is DatabaseTreeNode => n !== null);
  }, [treeNodes, treeSearch, treeFilterType]);

  // Sorted and filtered table details for Table Overview
  const sortedTableDetails = useMemo(() => {
    return [...tableDetails]
      .filter((t) => t.name.toLowerCase().includes(tableFilterWord.toLowerCase()))
      .sort((a, b) => {
        let cmp = 0;
        if (tableSortCol === 'name') cmp = a.name.localeCompare(b.name);
        else if (tableSortCol === 'rows') cmp = (a.rows || 0) - (b.rows || 0);
        else if (tableSortCol === 'engine') cmp = (a.engine || a.type || '').localeCompare(b.engine || b.type || '');
        else if (tableSortCol === 'collation') cmp = (a.collation || '').localeCompare(b.collation || '');
        else if (tableSortCol === 'size') cmp = (a.total_size_kb || 0) - (b.total_size_kb || 0);
        return tableSortOrder === 'ASC' ? cmp : -cmp;
      });
  }, [tableDetails, tableFilterWord, tableSortCol, tableSortOrder]);

  // Aggregate database statistics
  const dbMetrics = useMemo(() => {
    const count = tableDetails.length;
    const rows = tableDetails.reduce((sum, t) => sum + (t.rows || 0), 0);
    const sizeKb = tableDetails.reduce((sum, t) => sum + (t.total_size_kb || 0), 0);
    const sizeFormatted = sizeKb >= 1024 ? `${(sizeKb / 1024).toFixed(1)} MB` : `${sizeKb.toLocaleString()} KB`;
    const defaultCollation = tableDetails.length > 0 ? tableDetails[0].collation : 'utf8mb4_unicode_ci';
    return { count, rows, sizeKb, sizeFormatted, defaultCollation };
  }, [tableDetails]);

  // Realtime Database-Wide Optimization
  const handleOptimizeAllTables = async () => {
    if (!currentDb || tableDetails.length === 0) return;
    try {
      showToast(`Optimizing all ${tableDetails.length} tables in '${currentDb}'...`);
      const sql = `OPTIMIZE TABLE ${tableDetails.map((t) => '`' + t.name + '`').join(',')};`;
      await handleExecuteSql(sql);
      await fetchTableDetails(currentDb);
      showToast(`Successfully optimized all tables in '${currentDb}'!`);
    } catch (err: any) {
      showToast(err.message || 'Failed to optimize tables', true);
    }
  };

  // Realtime Database-Wide Check
  const handleCheckAllTables = async () => {
    if (!currentDb || tableDetails.length === 0) return;
    try {
      showToast(`Running integrity check across all tables in '${currentDb}'...`);
      const sql = `CHECK TABLE ${tableDetails.map((t) => '`' + t.name + '`').join(',')};`;
      await handleExecuteSql(sql);
      showToast(`Table integrity check completed for '${currentDb}'.`);
    } catch (err: any) {
      showToast(err.message || 'Failed to check tables', true);
    }
  };

  // Realtime Database-Wide Repair
  const handleRepairAllTables = async () => {
    if (!currentDb || tableDetails.length === 0) return;
    try {
      showToast(`Repairing tables in '${currentDb}'...`);
      const sql = `REPAIR TABLE ${tableDetails.map((t) => '`' + t.name + '`').join(',')};`;
      await handleExecuteSql(sql);
      await fetchTableDetails(currentDb);
      showToast(`Repair routine finished for '${currentDb}'.`);
    } catch (err: any) {
      showToast(err.message || 'Failed to repair tables', true);
    }
  };

  // Realtime Bulk Operations for Selected Tables
  const handleEmptySelectedTables = async () => {
    if (selectedTableNames.length === 0) return;
    if (!confirm(`TRUNCATE ${selectedTableNames.length} selected tables in '${currentDb}'? All data will be deleted.`)) return;
    try {
      for (const t of selectedTableNames) {
        await handleExecuteSql(`TRUNCATE TABLE \`${t}\`;`);
      }
      await fetchTableDetails(currentDb);
      setSelectedTableNames([]);
      showToast(`Successfully truncated ${selectedTableNames.length} tables.`);
    } catch (err: any) {
      showToast(err.message || 'Failed to truncate tables', true);
    }
  };

  const handleDropSelectedTables = async () => {
    if (selectedTableNames.length === 0) return;
    if (!confirm(`DROP ${selectedTableNames.length} selected tables in '${currentDb}'? This cannot be undone!`)) return;
    try {
      await handleExecuteSql(`DROP TABLE IF EXISTS ${selectedTableNames.map((t) => '`' + t + '`').join(',')};`);
      await fetchTableDetails(currentDb);
      await fetchTree();
      setSelectedTableNames([]);
      showToast(`Dropped ${selectedTableNames.length} tables successfully.`);
    } catch (err: any) {
      showToast(err.message || 'Failed to drop tables', true);
    }
  };

  return (
    <DashboardShell>
      {/* Full-page drag overlay */}
      {isDraggingOverPage && (
        <div className="fixed inset-0 z-[100] bg-amber-500/20 backdrop-blur-xs flex items-center justify-center pointer-events-none">
          <div className="bg-white rounded-3xl p-10 shadow-2xl border-4 border-dashed border-amber-500 text-center animate-pulse">
            <Upload className="w-16 h-16 mx-auto mb-4 text-amber-500" />
            <p className="text-xl font-bold text-slate-900">Drop your SQL file here</p>
            <p className="text-sm text-slate-500 mt-2">Release to import into <code className="font-mono text-amber-600">{currentDb || 'database'}</code></p>
          </div>
        </div>
      )}

      {/* Toast Notification */}
      {toastMessage && (
        <div className="fixed bottom-6 right-6 z-50 bg-slate-900 text-white px-4 py-3 rounded-2xl shadow-xl border border-slate-700 flex items-center gap-3 animate-fadeIn text-xs">
          <CheckCircle2 className="w-4 h-4 text-emerald-400 shrink-0" />
          <span>{toastMessage}</span>
        </div>
      )}

      {/* =========================================================================
          1. ADVANCED EXECUTIVE DATABASE TELEMETRY HUD (Pure Solid White - Zero Blur)
          ========================================================================= */}
      <div className="rounded-2xl bg-white text-slate-900 p-5 sm:p-6 border border-slate-200 shadow-xs mb-4">
        {/* Header Row */}
        <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3.5 pb-4 border-b border-slate-100">
          <div className="flex items-center gap-3">
            <div className="w-10 h-10 rounded-xl bg-amber-50 border border-amber-200 flex items-center justify-center text-amber-600 shadow-2xs shrink-0">
              <DatabaseIcon className="w-5 h-5 animate-pulse" />
            </div>
            <div>
              <div className="flex items-center gap-2 flex-wrap">
                <h1 className="text-sm font-black uppercase tracking-wider text-slate-900 font-mono">
                  Hostvra Database Control Center
                </h1>
                <span className="inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full text-[10px] bg-emerald-50 text-emerald-700 border border-emerald-200 font-mono font-bold">
                  <span className="w-1.5 h-1.5 rounded-full bg-emerald-500 animate-ping" />
                  LIVE CONNECTED
                </span>
                <span className="inline-flex items-center gap-1 px-2.5 py-0.5 rounded-full text-[10px] bg-slate-100 text-slate-700 border border-slate-200 font-mono font-bold">
                  MySQL 8.0 / MariaDB • Port 3306
                </span>
              </div>
              <p className="text-xs text-slate-500 font-medium mt-0.5">
                InnoDB ACID Engine • utf8mb4 Multi-byte Collation • Zero-Downtime Replication
              </p>
            </div>
          </div>

          {/* Quick Cockpit Action Buttons & Database Switcher */}
          <div className="flex items-center gap-2 flex-wrap">
            {/* Database Switcher dropdown */}
            <div className="relative">
              <select
                value={currentDb}
                onChange={(e) => {
                  const dbName = e.target.value;
                  if (dbName && dbName !== currentDb) {
                    handleSwitchDb(dbName);
                  }
                }}
                className="pl-3 pr-8 py-1.5 rounded-xl bg-slate-50 hover:bg-slate-100 text-slate-900 border border-slate-200 text-xs font-mono font-bold cursor-pointer transition shadow-2xs focus:outline-none focus:border-amber-500 appearance-none"
              >
                <option value="" disabled>Select Database...</option>
                {treeNodes.map((n) => (
                  <option key={n.name} value={n.name}>
                    {n.name} ({n.tables?.length || 0} tbls)
                  </option>
                ))}
              </select>
              <ChevronDown className="w-3.5 h-3.5 text-slate-400 absolute right-2.5 top-2.5 pointer-events-none" />
            </div>

            <button
              type="button"
              onClick={() => setCreateTableModalOpen(true)}
              className="px-3 py-1.5 rounded-xl bg-emerald-600 hover:bg-emerald-700 active:bg-emerald-800 text-white text-xs font-bold transition flex items-center gap-1.5 cursor-pointer shadow-xs"
              title="Create new database table"
            >
              <Plus className="w-3.5 h-3.5" />
              <span>Create Table</span>
            </button>

            <button
              type="button"
              onClick={handleOptimizeAllTables}
              className="px-3 py-1.5 rounded-xl bg-slate-50 hover:bg-slate-100 active:bg-slate-200 text-slate-700 border border-slate-200 text-xs font-bold transition flex items-center gap-1.5 cursor-pointer shadow-2xs"
              title="Run OPTIMIZE TABLE on all tables in active database"
            >
              <Sparkles className="w-3.5 h-3.5 text-amber-500" />
              <span>Optimize DB</span>
            </button>

            <button
              type="button"
              onClick={handleRepairAllTables}
              className="px-3 py-1.5 rounded-xl bg-slate-50 hover:bg-slate-100 active:bg-slate-200 text-slate-700 border border-slate-200 text-xs font-bold transition flex items-center gap-1.5 cursor-pointer shadow-2xs"
              title="Run REPAIR TABLE on tables in active database"
            >
              <Zap className="w-3.5 h-3.5 text-indigo-500" />
              <span>Repair</span>
            </button>

            <button
              type="button"
              onClick={() => {
                fetchTree();
                if (currentDb) {
                  fetchTableDetails(currentDb);
                  if (selectedTable) {
                    fetchTableStructure(currentDb, selectedTable);
                    fetchBrowseRows(currentDb, selectedTable, browsePage, browseLimit, browseSortCol, browseSortOrder, browseSearch);
                  }
                }
                showToast('Refreshed database metadata from host.');
              }}
              className="px-3 py-1.5 rounded-xl bg-slate-50 hover:bg-slate-100 active:bg-slate-200 text-slate-700 border border-slate-200 text-xs font-bold transition flex items-center gap-1.5 cursor-pointer shadow-2xs"
              title="Refresh Schema"
            >
              <RefreshCw className="w-3.5 h-3.5 text-slate-500" />
              <span>Refresh</span>
            </button>

            <Link
              href="/databases"
              className="px-3 py-1.5 rounded-xl bg-slate-50 hover:bg-slate-100 active:bg-slate-200 text-slate-700 border border-slate-200 text-xs font-bold transition flex items-center gap-1.5 cursor-pointer shadow-2xs"
            >
              <ArrowLeft className="w-3.5 h-3.5 text-slate-500" />
              <span>Databases</span>
            </Link>

            <button
              type="button"
              onClick={() => setHudExpanded(!hudExpanded)}
              className="px-2.5 py-1.5 rounded-xl bg-slate-100 hover:bg-slate-200 text-slate-600 border border-slate-200 text-xs font-semibold flex items-center gap-1 transition cursor-pointer shadow-2xs"
            >
              <span>{hudExpanded ? 'Hide' : 'Stats'}</span>
              <ChevronDown className={`w-3.5 h-3.5 transition-transform duration-200 ${hudExpanded ? 'rotate-180' : ''}`} />
            </button>
          </div>
        </div>

        {/* 4-Tile Live Database Metrics Grid */}
        {hudExpanded && (
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-3.5 pt-4">
            {/* Tile 1: Active Database */}
            <div className="p-4 rounded-xl bg-white border border-slate-200 hover:border-amber-500 hover:shadow-xs transition-all flex flex-col justify-between">
              <div className="flex items-center justify-between mb-2">
                <div className="w-9 h-9 rounded-lg bg-amber-50 border border-amber-200 flex items-center justify-center text-amber-600">
                  <DatabaseIcon className="w-4 h-4" />
                </div>
                <button
                  type="button"
                  onClick={() => {
                    if (currentDb) {
                      navigator.clipboard.writeText(currentDb);
                      showToast(`Copied database '${currentDb}' to clipboard`);
                    }
                  }}
                  className="text-[10px] px-2 py-0.5 rounded-full bg-amber-50 hover:bg-amber-100 text-amber-800 font-bold border border-amber-200 font-mono flex items-center gap-1 cursor-pointer transition"
                  title="Copy database name"
                >
                  <Copy className="w-2.5 h-2.5" />
                  <span>Copy</span>
                </button>
              </div>
              <div>
                <span className="text-[11px] text-slate-500 font-bold block uppercase tracking-wider">Active Database</span>
                <div className="flex items-baseline gap-2 mt-0.5 truncate">
                  <span className="text-xl font-black font-mono text-slate-900 truncate">{currentDb || 'None'}</span>
                </div>
                <div className="w-full bg-slate-100 h-1.5 rounded-full overflow-hidden mt-2.5">
                  <div className="bg-amber-500 h-full rounded-full w-full" />
                </div>
                <span className="text-[10px] text-slate-500 font-medium block mt-1.5">
                  Charset: {dbMetrics.defaultCollation}
                </span>
              </div>
            </div>

            {/* Tile 2: Live Tables */}
            <div className="p-4 rounded-xl bg-white border border-slate-200 hover:border-emerald-500 hover:shadow-xs transition-all flex flex-col justify-between">
              <div className="flex items-center justify-between mb-2">
                <div className="w-9 h-9 rounded-lg bg-emerald-50 border border-emerald-200 flex items-center justify-center text-emerald-600">
                  <Table className="w-4 h-4" />
                </div>
                <span className="text-[10px] px-2 py-0.5 rounded-full bg-emerald-50 text-emerald-700 font-bold border border-emerald-200 font-mono">
                  {dbMetrics.count} Tables
                </span>
              </div>
              <div>
                <span className="text-[11px] text-slate-500 font-bold block uppercase tracking-wider">Tables in Schema</span>
                <div className="flex items-baseline gap-2 mt-0.5">
                  <span className="text-2xl font-black font-mono text-slate-900">{dbMetrics.count}</span>
                  <span className="text-xs text-slate-500 font-medium">Relational Tables</span>
                </div>
                <div className="w-full bg-slate-100 h-1.5 rounded-full overflow-hidden mt-2.5">
                  <div className="bg-emerald-500 h-full rounded-full w-full" />
                </div>
                <span className="text-[10px] text-slate-500 font-medium block mt-1.5">
                  Views: {views.length} • Routines: {routines.length}
                </span>
              </div>
            </div>

            {/* Tile 3: Total Records / Rows */}
            <div className="p-4 rounded-xl bg-white border border-slate-200 hover:border-sky-500 hover:shadow-xs transition-all flex flex-col justify-between">
              <div className="flex items-center justify-between mb-2">
                <div className="w-9 h-9 rounded-lg bg-sky-50 border border-sky-200 flex items-center justify-center text-sky-600">
                  <Activity className="w-4 h-4" />
                </div>
                <span className="text-[10px] px-2 py-0.5 rounded-full bg-sky-50 text-sky-700 font-bold border border-sky-200 font-mono">
                  Realtime Count
                </span>
              </div>
              <div>
                <span className="text-[11px] text-slate-500 font-bold block uppercase tracking-wider">Total Records / Rows</span>
                <div className="flex items-baseline gap-2 mt-0.5">
                  <span className="text-2xl font-black font-mono text-slate-900">{dbMetrics.rows.toLocaleString()}</span>
                  <span className="text-xs text-slate-500 font-medium">Rows</span>
                </div>
                <div className="w-full bg-slate-100 h-1.5 rounded-full overflow-hidden mt-2.5">
                  <div className="bg-sky-500 h-full rounded-full w-full" />
                </div>
                <span className="text-[10px] text-slate-500 font-medium block mt-1.5">
                  Aggregate table cardinality
                </span>
              </div>
            </div>

            {/* Tile 4: Storage Footprint */}
            <div className="p-4 rounded-xl bg-white border border-slate-200 hover:border-indigo-500 hover:shadow-xs transition-all flex flex-col justify-between">
              <div className="flex items-center justify-between mb-2">
                <div className="w-9 h-9 rounded-lg bg-indigo-50 border border-indigo-200 flex items-center justify-center text-indigo-600">
                  <HardDrive className="w-4 h-4" />
                </div>
                <span className="text-[10px] px-2 py-0.5 rounded-full bg-indigo-50 text-indigo-700 font-bold border border-indigo-200 font-mono">
                  Disk Footprint
                </span>
              </div>
              <div>
                <span className="text-[11px] text-slate-500 font-bold block uppercase tracking-wider">Data + Index Storage</span>
                <div className="flex items-baseline gap-2 mt-0.5">
                  <span className="text-2xl font-black font-mono text-indigo-700">{dbMetrics.sizeFormatted}</span>
                </div>
                <div className="w-full bg-slate-100 h-1.5 rounded-full overflow-hidden mt-2.5">
                  <div className="bg-indigo-500 h-full rounded-full w-full" />
                </div>
                <span className="text-[10px] text-slate-500 font-medium block mt-1.5">
                  Allocated MySQL tablespace
                </span>
              </div>
            </div>
          </div>
        )}
      </div>

      {/* =========================================================================
          2. TOP PHPMYADMIN NAVIGATION TABS (Solid Pure White Strip)
          ========================================================================= */}
      <div className="bg-white border border-slate-200 rounded-2xl p-2 shadow-2xs mb-4 flex flex-wrap items-center justify-between gap-3">
        <div className="flex items-center gap-1.5 overflow-x-auto no-scrollbar max-w-full">
          {selectedTable && (
            <button
              onClick={() => setActiveTab('browse')}
              className={`flex items-center gap-1.5 px-3.5 py-1.5 rounded-xl text-xs font-bold transition-all cursor-pointer select-none shrink-0 ${
                activeTab === 'browse'
                  ? 'bg-emerald-600 text-white shadow-xs'
                  : 'text-slate-600 hover:text-slate-900 hover:bg-slate-100/80'
              }`}
            >
              <Eye className="w-3.5 h-3.5" />
              <span>Browse: {selectedTable}</span>
            </button>
          )}

          <button
            onClick={() => setActiveTab('structure')}
            className={`flex items-center gap-1.5 px-3.5 py-1.5 rounded-xl text-xs font-bold transition-all cursor-pointer select-none shrink-0 ${
              activeTab === 'structure'
                ? 'bg-emerald-600 text-white shadow-xs'
                : 'text-slate-600 hover:text-slate-900 hover:bg-slate-100/80'
            }`}
          >
            <Columns className="w-3.5 h-3.5" />
            <span>Structure</span>
          </button>

          <button
            onClick={() => setActiveTab('sql')}
            className={`flex items-center gap-1.5 px-3.5 py-1.5 rounded-xl text-xs font-bold transition-all cursor-pointer select-none shrink-0 ${
              activeTab === 'sql'
                ? 'bg-emerald-600 text-white shadow-xs'
                : 'text-slate-600 hover:text-slate-900 hover:bg-slate-100/80'
            }`}
          >
            <Terminal className="w-3.5 h-3.5" />
            <span>SQL Console</span>
          </button>

          <button
            onClick={() => setActiveTab('export')}
            className={`flex items-center gap-1.5 px-3.5 py-1.5 rounded-xl text-xs font-bold transition-all cursor-pointer select-none shrink-0 ${
              activeTab === 'export'
                ? 'bg-emerald-600 text-white shadow-xs'
                : 'text-slate-600 hover:text-slate-900 hover:bg-slate-100/80'
            }`}
          >
            <Download className="w-3.5 h-3.5" />
            <span>Export</span>
          </button>

          <button
            onClick={() => setActiveTab('import')}
            className={`flex items-center gap-1.5 px-3.5 py-1.5 rounded-xl text-xs font-bold transition-all cursor-pointer select-none shrink-0 ${
              activeTab === 'import'
                ? 'bg-emerald-600 text-white shadow-xs'
                : 'text-slate-600 hover:text-slate-900 hover:bg-slate-100/80'
            }`}
          >
            <Upload className="w-3.5 h-3.5" />
            <span>Import</span>
          </button>

          <button
            onClick={() => setActiveTab('operations')}
            className={`flex items-center gap-1.5 px-3.5 py-1.5 rounded-xl text-xs font-bold transition-all cursor-pointer select-none shrink-0 ${
              activeTab === 'operations'
                ? 'bg-emerald-600 text-white shadow-xs'
                : 'text-slate-600 hover:text-slate-900 hover:bg-slate-100/80'
            }`}
          >
            <Sliders className="w-3.5 h-3.5" />
            <span>Operations</span>
          </button>

          <button
            onClick={() => setActiveTab('routines')}
            className={`flex items-center gap-1.5 px-3.5 py-1.5 rounded-xl text-xs font-bold transition-all cursor-pointer select-none shrink-0 ${
              activeTab === 'routines'
                ? 'bg-emerald-600 text-white shadow-xs'
                : 'text-slate-600 hover:text-slate-900 hover:bg-slate-100/80'
            }`}
          >
            <Code2 className="w-3.5 h-3.5" />
            <span>Routines ({routines.length})</span>
          </button>

          <button
            onClick={() => setActiveTab('events')}
            className={`flex items-center gap-1.5 px-3.5 py-1.5 rounded-xl text-xs font-bold transition-all cursor-pointer select-none shrink-0 ${
              activeTab === 'events'
                ? 'bg-emerald-600 text-white shadow-xs'
                : 'text-slate-600 hover:text-slate-900 hover:bg-slate-100/80'
            }`}
          >
            <Clock className="w-3.5 h-3.5" />
            <span>Events ({events.length})</span>
          </button>

          <button
            onClick={() => setActiveTab('triggers')}
            className={`flex items-center gap-1.5 px-3.5 py-1.5 rounded-xl text-xs font-bold transition-all cursor-pointer select-none shrink-0 ${
              activeTab === 'triggers'
                ? 'bg-emerald-600 text-white shadow-xs'
                : 'text-slate-600 hover:text-slate-900 hover:bg-slate-100/80'
            }`}
          >
            <Zap className="w-3.5 h-3.5" />
            <span>Triggers ({triggers.length})</span>
          </button>

          <button
            onClick={() => setActiveTab('views')}
            className={`flex items-center gap-1.5 px-3.5 py-1.5 rounded-xl text-xs font-bold transition-all cursor-pointer select-none shrink-0 ${
              activeTab === 'views'
                ? 'bg-emerald-600 text-white shadow-xs'
                : 'text-slate-600 hover:text-slate-900 hover:bg-slate-100/80'
            }`}
          >
            <Layers className="w-3.5 h-3.5" />
            <span>Views ({views.length})</span>
          </button>

          <button
            onClick={() => setActiveTab('privileges')}
            className={`flex items-center gap-1.5 px-3.5 py-1.5 rounded-xl text-xs font-bold transition-all cursor-pointer select-none shrink-0 ${
              activeTab === 'privileges'
                ? 'bg-emerald-600 text-white shadow-xs'
                : 'text-slate-600 hover:text-slate-900 hover:bg-slate-100/80'
            }`}
          >
            <Shield className="w-3.5 h-3.5" />
            <span>Privileges</span>
          </button>
        </div>

        {/* Current Database Indicator */}
        <div className="flex items-center gap-1.5 text-xs font-semibold text-slate-500 pr-2 hidden md:flex">
          <Server className="w-3.5 h-3.5 text-amber-500" />
          <span>localhost</span>
          <ChevronRight className="w-3 h-3 text-slate-400" />
          <DatabaseIcon className="w-3.5 h-3.5 text-emerald-500" />
          <span className="font-mono font-bold text-slate-900">{currentDb || 'none'}</span>
          {selectedTable && (
            <>
              <ChevronRight className="w-3 h-3 text-slate-400" />
              <Table className="w-3.5 h-3.5 text-blue-500" />
              <span className="font-mono font-bold text-slate-900">{selectedTable}</span>
            </>
          )}
        </div>
      </div>

      {/* Main Two-Column Layout */}
      <div className="grid grid-cols-1 lg:grid-cols-12 gap-5 items-start">
        {/* Left Column: Authentic Database Hierarchy Tree Navigator (Pure Solid White) */}
        <div className="lg:col-span-3 bg-white border border-slate-200 rounded-2xl p-4 shadow-xs space-y-3 sticky top-4">
          <div className="flex items-center justify-between border-b border-slate-100 pb-2.5">
            <div className="flex items-center gap-2">
              <div className="w-7 h-7 rounded-lg bg-amber-50 border border-amber-200 flex items-center justify-center text-amber-600 shadow-2xs shrink-0">
                <FolderTree className="w-4 h-4" />
              </div>
              <div>
                <span className="font-bold text-xs text-slate-900 block leading-tight">Database Tree</span>
                <span className="text-[10px] text-slate-400 font-mono leading-none mt-0.5 block">{treeNodes.length} databases</span>
              </div>
            </div>
            <div className="flex items-center gap-1">
              <button
                onClick={() => fetchTree()}
                className="p-1 rounded-lg hover:bg-slate-100 text-slate-400 hover:text-slate-700 transition cursor-pointer"
                title="Refresh database tree from MySQL"
              >
                <RotateCcw className="w-3.5 h-3.5" />
              </button>
              <button
                onClick={() => setCreateDbModalOpen(true)}
                className="p-1 rounded-lg hover:bg-emerald-50 text-emerald-600 hover:text-emerald-700 transition cursor-pointer"
                title="Create New Database"
              >
                <Plus className="w-3.5 h-3.5" />
              </button>
            </div>
          </div>

          {/* Filter Segment Tabs: All / Tables / Views */}
          <div className="flex items-center bg-slate-100 p-0.5 rounded-xl text-[11px] font-semibold">
            <button
              type="button"
              onClick={() => setTreeFilterType('all')}
              className={`flex-1 py-1 rounded-lg text-center transition cursor-pointer ${
                treeFilterType === 'all' ? 'bg-white text-slate-900 shadow-2xs font-bold' : 'text-slate-500 hover:text-slate-900'
              }`}
            >
              All
            </button>
            <button
              type="button"
              onClick={() => setTreeFilterType('tables')}
              className={`flex-1 py-1 rounded-lg text-center transition cursor-pointer ${
                treeFilterType === 'tables' ? 'bg-white text-slate-900 shadow-2xs font-bold' : 'text-slate-500 hover:text-slate-900'
              }`}
            >
              Tables
            </button>
            <button
              type="button"
              onClick={() => setTreeFilterType('views')}
              className={`flex-1 py-1 rounded-lg text-center transition cursor-pointer ${
                treeFilterType === 'views' ? 'bg-white text-slate-900 shadow-2xs font-bold' : 'text-slate-500 hover:text-slate-900'
              }`}
            >
              Views
            </button>
          </div>

          {/* Tree Search Box */}
          <div className="relative">
            <Search className="w-3.5 h-3.5 absolute left-2.5 top-2.5 text-slate-400" />
            <input
              type="text"
              value={treeSearch}
              onChange={(e) => setTreeSearch(e.target.value)}
              placeholder="Search tables & databases..."
              className="w-full pl-8 pr-7 py-1.5 rounded-xl bg-slate-50 border border-slate-200 text-xs font-mono text-slate-900 placeholder:text-slate-400 focus:outline-none focus:border-amber-500 focus:bg-white transition"
            />
            {treeSearch && (
              <button
                onClick={() => setTreeSearch('')}
                className="absolute right-2 top-2 text-slate-400 hover:text-slate-600 cursor-pointer"
              >
                <X className="w-3 h-3" />
              </button>
            )}
          </div>

          {/* Tree View List */}
          <div className="max-h-[640px] overflow-y-auto space-y-1 text-xs pr-1">
            {loadingTree ? (
              <div className="p-4 text-center text-slate-400 text-xs">Loading database tree...</div>
            ) : filteredTreeNodes.length === 0 ? (
              <div className="p-4 text-center text-slate-400 text-xs">No databases found</div>
            ) : (
              filteredTreeNodes.map((node) => {
                const isCurrent = currentDb === node.name;
                const isExpanded = expandedDbs[node.name] || false;

                return (
                  <div key={node.name} className="space-y-0.5">
                    {/* Database Node */}
                    <div
                      className={`flex items-center justify-between px-2.5 py-1.5 rounded-xl transition group ${
                        isCurrent
                          ? 'bg-amber-50 text-amber-900 font-bold border border-amber-200/70 shadow-2xs'
                          : 'hover:bg-slate-50 text-slate-700'
                      }`}
                    >
                      <div
                        className="flex items-center gap-1.5 truncate flex-1 cursor-pointer"
                        onClick={() => {
                          if (currentDb !== node.name) handleSwitchDb(node.name);
                          setExpandedDbs((prev) => ({ ...prev, [node.name]: !isExpanded }));
                        }}
                      >
                        <span className="text-slate-400 font-mono text-[10px]">
                          {isExpanded ? '[-]' : '[+]'}
                        </span>
                        <DatabaseIcon className={`w-3.5 h-3.5 shrink-0 ${isCurrent ? 'text-amber-600' : 'text-slate-400'}`} />
                        <span className="truncate font-mono">{node.name}</span>
                      </div>

                      <div className="flex items-center gap-1 flex-shrink-0">
                        <span className="text-[10px] font-mono px-1.5 py-0.5 rounded-md bg-white border border-slate-200 text-slate-600 shadow-2xs">
                          {node.tables?.length || 0}
                        </span>

                        {!['mysql', 'information_schema', 'performance_schema', 'sys'].includes(node.name.toLowerCase()) && (
                          <button
                            onClick={(e) => {
                              e.stopPropagation();
                              handleDropDatabase(node.name);
                            }}
                            className="p-1 rounded text-slate-400 hover:text-rose-600 hover:bg-rose-50 opacity-0 group-hover:opacity-100 transition-opacity cursor-pointer"
                            title={`Drop database ${node.name}`}
                          >
                            <Trash2 className="w-3.5 h-3.5" />
                          </button>
                        )}
                      </div>
                    </div>

                    {/* Expanded Database Hierarchy */}
                    {isExpanded && (
                      <div className="pl-3.5 space-y-0.5 border-l-2 border-slate-200 ml-2.5 my-1">
                        {/* New Table button inside active database */}
                        {isCurrent && (
                          <button
                            onClick={() => setCreateTableModalOpen(true)}
                            className="w-full flex items-center gap-1.5 px-2 py-1 rounded-lg text-emerald-700 hover:bg-emerald-50 font-bold text-[11px] cursor-pointer transition mb-1"
                          >
                            <Plus className="w-3 h-3" />
                            <span>New Table</span>
                          </button>
                        )}

                        {/* Tables */}
                        {(node.tables || []).map((tbl) => {
                          const isTblActive = isCurrent && selectedTable === tbl.name;
                          return (
                            <div
                              key={tbl.name}
                              onClick={() => {
                                if (currentDb !== node.name) handleSwitchDb(node.name);
                                handleOpenTableBrowse(tbl.name);
                              }}
                              className={`group flex items-center justify-between px-2 py-1 rounded-lg cursor-pointer transition text-xs font-mono ${
                                isTblActive
                                  ? 'bg-emerald-50 text-emerald-800 font-bold border border-emerald-200/60 shadow-2xs'
                                  : 'hover:bg-slate-100 text-slate-600 hover:text-slate-900'
                              }`}
                            >
                              <div className="flex items-center gap-1.5 truncate flex-1 min-w-0 mr-1">
                                <Table className={`w-3 h-3 shrink-0 ${isTblActive ? 'text-emerald-600' : 'text-slate-400 group-hover:text-emerald-500'}`} />
                                <span className="truncate">{tbl.name}</span>
                              </div>

                              {/* Hover Quick Action Buttons */}
                              <div className="hidden group-hover:flex items-center gap-0.5 shrink-0">
                                <button
                                  onClick={(e) => {
                                    e.stopPropagation();
                                    if (currentDb !== node.name) handleSwitchDb(node.name);
                                    handleOpenTableBrowse(tbl.name);
                                  }}
                                  className="p-1 rounded hover:bg-emerald-100 text-emerald-700"
                                  title="Browse table rows"
                                >
                                  <Eye className="w-3 h-3" />
                                </button>
                                <button
                                  onClick={(e) => {
                                    e.stopPropagation();
                                    if (currentDb !== node.name) handleSwitchDb(node.name);
                                    handleOpenTableStructure(tbl.name);
                                  }}
                                  className="p-1 rounded hover:bg-sky-100 text-sky-700"
                                  title="View structure"
                                >
                                  <Columns className="w-3 h-3" />
                                </button>
                                <button
                                  onClick={(e) => {
                                    e.stopPropagation();
                                    navigator.clipboard.writeText(tbl.name);
                                    showToast(`Copied table '${tbl.name}'`);
                                  }}
                                  className="p-1 rounded hover:bg-slate-200 text-slate-600"
                                  title="Copy name"
                                >
                                  <Copy className="w-3 h-3" />
                                </button>
                              </div>

                              <span className="text-[10px] text-slate-400 group-hover:hidden shrink-0">
                                {tbl.rows.toLocaleString()}
                              </span>
                            </div>
                          );
                        })}

                        {/* Views */}
                        {node.views && node.views.length > 0 && (
                          <div className="pt-1">
                            <div className="text-[10px] uppercase font-bold text-slate-400 px-2 py-0.5 flex items-center gap-1">
                              <Layers className="w-3 h-3" />
                              <span>Views ({node.views.length})</span>
                            </div>
                            {node.views.map((v) => (
                              <div
                                key={v}
                                onClick={() => {
                                  if (currentDb !== node.name) handleSwitchDb(node.name);
                                  handleOpenTableBrowse(v);
                                }}
                                className="flex items-center gap-1.5 px-2 py-0.5 text-xs text-slate-500 hover:text-slate-900 cursor-pointer truncate font-mono"
                              >
                                <span className="w-1.5 h-1.5 rounded-full bg-cyan-500"></span>
                                <span className="truncate">{v}</span>
                              </div>
                            ))}
                          </div>
                        )}

                        {/* Procedures & Functions */}
                        {((node.procedures && node.procedures.length > 0) || (node.functions && node.functions.length > 0)) && (
                          <div className="pt-1">
                            <div className="text-[10px] uppercase font-bold text-slate-400 px-2 py-0.5 flex items-center gap-1">
                              <Code2 className="w-3 h-3" />
                              <span>Routines</span>
                            </div>
                            {(node.procedures || []).map((p) => (
                              <div
                                key={p}
                                onClick={() => {
                                  if (currentDb !== node.name) handleSwitchDb(node.name);
                                  setActiveTab('routines');
                                }}
                                className="flex items-center gap-1.5 px-2 py-0.5 text-xs text-slate-500 hover:text-purple-600 cursor-pointer truncate font-mono"
                              >
                                <span className="text-[10px] text-purple-600 font-bold">P</span>
                                <span className="truncate">{p}</span>
                              </div>
                            ))}
                            {(node.functions || []).map((f) => (
                              <div
                                key={f}
                                onClick={() => {
                                  if (currentDb !== node.name) handleSwitchDb(node.name);
                                  setActiveTab('routines');
                                }}
                                className="flex items-center gap-1.5 px-2 py-0.5 text-xs text-slate-500 hover:text-purple-600 cursor-pointer truncate font-mono"
                              >
                                <span className="text-[10px] text-purple-600 font-bold">F</span>
                                <span className="truncate">{f}</span>
                              </div>
                            ))}
                          </div>
                        )}
                      </div>
                    )}
                  </div>
                );
              })
            )}
          </div>
        </div>

        {/* Right Column: Active Tab Content Area */}
        <div className="lg:col-span-9 space-y-4">
          {/* TAB 1: STRUCTURE VIEW */}
          {activeTab === 'structure' && (
            <div className="bg-white border border-slate-200 rounded-2xl shadow-xs overflow-hidden">
              {/* If a specific table is selected, show its full Column, Index, FK structure */}
              {selectedTable ? (
                <div className="p-5 space-y-6">
                  <div className="flex flex-wrap items-center justify-between gap-3 border-b border-slate-100 pb-3">
                    <div>
                      <h2 className="text-sm font-bold text-slate-900 flex items-center gap-2">
                        <Table className="w-4 h-4 text-emerald-500" />
                        <span>Table Structure: <code className="font-mono text-emerald-600">{selectedTable}</code></span>
                      </h2>
                      <p className="text-xs text-slate-500">Columns, data types, indexes, and foreign key constraints</p>
                    </div>
                    <div className="flex items-center gap-2">
                      <button
                        onClick={() => handleOpenTableBrowse(selectedTable)}
                        className="px-3.5 py-1.5 rounded-xl bg-emerald-600 hover:bg-emerald-700 text-white font-bold text-xs cursor-pointer shadow-xs transition"
                      >
                        Browse Table Rows
                      </button>
                      <button
                        onClick={() => setAddColumnModalOpen(true)}
                        className="flex items-center gap-1.5 px-3 py-1.5 rounded-xl bg-slate-100 hover:bg-slate-200 text-slate-800 font-bold text-xs cursor-pointer transition"
                      >
                        <Plus className="w-3.5 h-3.5" />
                        <span>Add Column</span>
                      </button>
                    </div>
                  </div>

                  {loadingStructure ? (
                    <div className="p-8 text-center text-slate-400 text-xs">Loading table structure...</div>
                  ) : !tableStructure ? (
                    <div className="p-8 text-center text-slate-400 text-xs">Unable to load table structure.</div>
                  ) : (
                    <>
                      {/* Columns Table */}
                      <div className="space-y-2">
                        <h3 className="font-bold text-xs text-slate-700">Columns ({tableStructure.columns?.length || 0})</h3>
                        <div className="overflow-x-auto border border-slate-200 rounded-xl shadow-2xs">
                          <table className="w-full text-left text-xs border-collapse">
                            <thead>
                              <tr className="border-b border-slate-200 bg-slate-50 text-slate-700 font-bold text-[11px]">
                                <th className="p-2.5">Field</th>
                                <th className="p-2.5">Type</th>
                                <th className="p-2.5">Collation</th>
                                <th className="p-2.5">Null</th>
                                <th className="p-2.5">Key</th>
                                <th className="p-2.5">Default</th>
                                <th className="p-2.5">Extra</th>
                                <th className="p-2.5 text-center">Action</th>
                              </tr>
                            </thead>
                            <tbody className="divide-y divide-slate-100 font-mono text-xs">
                              {tableStructure.columns.map((col) => (
                                <tr key={col.field} className="hover:bg-slate-50 transition">
                                  <td className="p-2.5 font-bold text-slate-900 flex items-center gap-1.5">
                                    {col.key === 'PRI' && <Key className="w-3.5 h-3.5 text-amber-500" />}
                                    <span>{col.field}</span>
                                  </td>
                                  <td className="p-2.5 text-emerald-600 font-bold">{col.type}</td>
                                  <td className="p-2.5 text-slate-500">{col.collation || '-'}</td>
                                  <td className="p-2.5">{col.null}</td>
                                  <td className="p-2.5 font-bold text-amber-600">{col.key}</td>
                                  <td className="p-2.5 text-slate-500">{col.default !== null && col.default !== undefined ? String(col.default) : <em className="text-slate-400">NULL</em>}</td>
                                  <td className="p-2.5 text-purple-600">{col.extra}</td>
                                  <td className="p-2.5 font-sans text-center">
                                    <div className="flex items-center justify-center gap-1.5">
                                      <button
                                        onClick={() => handleOpenModifyCol(col)}
                                        className="inline-flex items-center gap-1 px-2.5 py-1 rounded-lg text-xs font-semibold bg-sky-50 text-sky-700 hover:bg-sky-100 border border-sky-200 transition shadow-2xs cursor-pointer"
                                        title={`Modify column ${col.field}`}
                                      >
                                        <Edit3 className="w-3 h-3 text-sky-600" />
                                        <span>Change</span>
                                      </button>
                                      <button
                                        onClick={() => handleDropColumn(col.field)}
                                        className="inline-flex items-center gap-1 px-2.5 py-1 rounded-lg text-xs font-semibold bg-rose-50 text-rose-700 hover:bg-rose-100 border border-rose-200 transition shadow-2xs cursor-pointer"
                                        title={`Drop column ${col.field}`}
                                      >
                                        <Trash2 className="w-3 h-3 text-rose-600" />
                                        <span>Drop</span>
                                      </button>
                                    </div>
                                  </td>
                                </tr>
                              ))}
                            </tbody>
                          </table>
                        </div>
                      </div>

                      {/* Indexes Table */}
                      <div className="space-y-2 pt-2">
                        <div className="flex items-center justify-between">
                          <h3 className="font-bold text-xs text-slate-700">Indexes ({tableStructure.indexes?.length || 0})</h3>
                          <button
                            onClick={() => setAddIndexModalOpen(true)}
                            className="inline-flex items-center gap-1 text-xs text-blue-600 hover:text-blue-800 font-bold cursor-pointer"
                          >
                            <Plus className="w-3 h-3" />
                            <span>Create Index</span>
                          </button>
                        </div>
                        <div className="overflow-x-auto border border-slate-200 rounded-xl shadow-2xs">
                          <table className="w-full text-left text-xs border-collapse">
                            <thead>
                              <tr className="border-b border-slate-200 bg-slate-50 text-slate-700 font-bold text-[11px]">
                                <th className="p-2.5">Key Name</th>
                                <th className="p-2.5">Type</th>
                                <th className="p-2.5">Unique</th>
                                <th className="p-2.5">Column</th>
                                <th className="p-2.5 text-center">Action</th>
                              </tr>
                            </thead>
                            <tbody className="divide-y divide-slate-100 font-mono text-xs">
                              {tableStructure.indexes.map((idx, i) => (
                                <tr key={i} className="hover:bg-slate-50 transition">
                                  <td className="p-2.5 font-bold text-slate-900">{idx.key_name}</td>
                                  <td className="p-2.5">{idx.index_type}</td>
                                  <td className="p-2.5">{idx.non_unique === 0 ? 'Yes' : 'No'}</td>
                                  <td className="p-2.5 text-amber-600">{idx.column_name}</td>
                                  <td className="p-2.5 font-sans text-center">
                                    <button
                                      onClick={() => handleDropIndex(idx.key_name)}
                                      className="inline-flex items-center gap-1 px-2.5 py-1 rounded-lg text-xs font-semibold bg-rose-50 text-rose-700 hover:bg-rose-100 border border-rose-200 transition shadow-2xs cursor-pointer"
                                      title={`Drop index ${idx.key_name}`}
                                    >
                                      <Trash2 className="w-3 h-3 text-rose-600" />
                                      <span>Drop</span>
                                    </button>
                                  </td>
                                </tr>
                              ))}
                            </tbody>
                          </table>
                        </div>
                      </div>

                      {/* Foreign Keys Table */}
                      {tableStructure.foreign_keys && tableStructure.foreign_keys.length > 0 && (
                        <div className="space-y-2 pt-2">
                          <h3 className="font-bold text-xs text-slate-700">Foreign Keys</h3>
                          <div className="overflow-x-auto border border-slate-200 rounded-xl shadow-2xs">
                            <table className="w-full text-left text-xs border-collapse font-mono">
                              <thead className="bg-slate-50 font-bold text-[11px]">
                                <tr className="border-b border-slate-200">
                                  <th className="p-2.5">Constraint</th>
                                  <th className="p-2.5">Column</th>
                                  <th className="p-2.5">Ref Table</th>
                                  <th className="p-2.5">Ref Column</th>
                                  <th className="p-2.5">On Update</th>
                                  <th className="p-2.5">On Delete</th>
                                </tr>
                              </thead>
                              <tbody className="divide-y divide-slate-100">
                                {tableStructure.foreign_keys.map((fk, i) => (
                                  <tr key={i} className="hover:bg-slate-50">
                                    <td className="p-2.5 font-bold">{fk.constraint_name}</td>
                                    <td className="p-2.5 text-amber-600">{fk.column_name}</td>
                                    <td className="p-2.5 text-emerald-600">{fk.ref_table}</td>
                                    <td className="p-2.5 text-emerald-600">{fk.ref_column}</td>
                                    <td className="p-2.5 text-slate-500">{fk.on_update}</td>
                                    <td className="p-2.5 text-slate-500">{fk.on_delete}</td>
                                  </tr>
                                ))}
                              </tbody>
                            </table>
                          </div>
                        </div>
                      )}
                    </>
                  )}
                </div>
              ) : (
                /* Database-level Table Overview (High-Tech Executive Cockpit) */
                <div className="p-5 space-y-4 bg-white">
                  {/* Top Header of Table Overview */}
                  <div className="flex flex-wrap items-center justify-between gap-3 pb-3 border-b border-slate-100">
                    <div>
                      <div className="flex items-center gap-2">
                        <div className="w-8 h-8 rounded-lg bg-amber-50 border border-amber-200 flex items-center justify-center text-amber-600 shadow-2xs">
                          <DatabaseIcon className="w-4 h-4" />
                        </div>
                        <div>
                          <div className="flex items-center gap-2">
                            <h2 className="text-base font-black text-slate-900 font-mono tracking-tight">
                              {currentDb}
                            </h2>
                            <span className="px-2 py-0.5 rounded-full text-[11px] font-bold bg-amber-50 text-amber-800 border border-amber-200 font-mono">
                              {sortedTableDetails.length} tables
                            </span>
                            <span className="px-2 py-0.5 rounded-full text-[11px] font-bold bg-emerald-50 text-emerald-800 border border-emerald-200 font-mono">
                              {dbMetrics.rows.toLocaleString()} rows
                            </span>
                            <span className="px-2 py-0.5 rounded-full text-[11px] font-bold bg-indigo-50 text-indigo-800 border border-indigo-200 font-mono">
                              {dbMetrics.sizeFormatted}
                            </span>
                          </div>
                          <p className="text-xs text-slate-500 mt-0.5">
                            Realtime table schema, record telemetry, and storage metrics
                          </p>
                        </div>
                      </div>
                    </div>

                    <div className="flex items-center gap-2 flex-wrap">
                      <button
                        onClick={() => setCreateTableModalOpen(true)}
                        className="inline-flex items-center gap-1.5 px-3.5 py-1.5 rounded-xl bg-emerald-600 hover:bg-emerald-700 text-white font-bold text-xs shadow-xs cursor-pointer transition"
                      >
                        <Plus className="w-3.5 h-3.5" />
                        <span>Create Table</span>
                      </button>

                      <button
                        onClick={handleOptimizeAllTables}
                        className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-xl bg-white hover:bg-slate-50 text-slate-700 border border-slate-200 text-xs font-bold transition shadow-2xs cursor-pointer"
                        title="Optimize all tables in database"
                      >
                        <Sparkles className="w-3.5 h-3.5 text-amber-500" />
                        <span>Optimize DB</span>
                      </button>

                      {!['mysql', 'information_schema', 'performance_schema', 'sys'].includes((currentDb || '').toLowerCase()) && (
                        <button
                          onClick={() => handleDropDatabase(currentDb)}
                          className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-xl bg-rose-50 hover:bg-rose-100 text-rose-700 border border-rose-200 text-xs font-bold transition shadow-2xs cursor-pointer"
                          title={`Permanently drop database ${currentDb}`}
                        >
                          <Trash2 className="w-3.5 h-3.5 text-rose-600" />
                          <span>Drop Database</span>
                        </button>
                      )}
                    </div>
                  </div>

                  {/* Filter & Batch Actions Bar */}
                  <div className="flex flex-wrap items-center justify-between gap-3 text-xs pt-1">
                    <div className="flex items-center gap-2.5 flex-1 min-w-[280px]">
                      <div className="relative flex-1 max-w-xs">
                        <Search className="w-3.5 h-3.5 absolute left-2.5 top-2.5 text-slate-400" />
                        <input
                          type="text"
                          placeholder="Filter tables in database..."
                          value={tableFilterWord}
                          onChange={(e) => setTableFilterWord(e.target.value)}
                          className="w-full pl-8 pr-7 py-1.5 rounded-xl bg-slate-50 border border-slate-200 font-mono text-xs text-slate-900 placeholder:text-slate-400 focus:outline-none focus:border-amber-500 focus:bg-white transition"
                        />
                        {tableFilterWord && (
                          <button
                            onClick={() => setTableFilterWord('')}
                            className="absolute right-2 top-2 text-slate-400 hover:text-slate-600 cursor-pointer"
                          >
                            <X className="w-3 h-3" />
                          </button>
                        )}
                      </div>

                      <div className="flex items-center gap-1 bg-slate-100 p-0.5 rounded-xl text-[11px] font-semibold">
                        <button
                          type="button"
                          onClick={() => setTableFilterWord('')}
                          className={`px-2.5 py-1 rounded-lg transition cursor-pointer ${
                            !tableFilterWord ? 'bg-white text-slate-900 shadow-2xs font-bold' : 'text-slate-500 hover:text-slate-900'
                          }`}
                        >
                          All ({tableDetails.length})
                        </button>
                      </div>
                    </div>

                    {/* Batch Actions Toolbar when tables are selected */}
                    {selectedTableNames.length > 0 && (
                      <div className="flex items-center gap-1.5 p-1 px-2.5 rounded-xl bg-amber-50 border border-amber-200 text-xs font-bold animate-fadeIn">
                        <span className="text-amber-900 font-mono mr-1">
                          {selectedTableNames.length} selected:
                        </span>
                        <button
                          onClick={async () => {
                            if (!confirm(`Optimize ${selectedTableNames.length} selected tables?`)) return;
                            await handleExecuteSql(`OPTIMIZE TABLE ${selectedTableNames.map((t) => '`' + t + '`').join(',')};`);
                            await fetchTableDetails(currentDb);
                            showToast(`Optimized ${selectedTableNames.length} tables.`);
                          }}
                          className="px-2.5 py-1 rounded-lg bg-emerald-600 hover:bg-emerald-700 text-white shadow-2xs cursor-pointer transition"
                        >
                          Optimize
                        </button>
                        <button
                          onClick={async () => {
                            if (!confirm(`CHECK ${selectedTableNames.length} selected tables?`)) return;
                            await handleExecuteSql(`CHECK TABLE ${selectedTableNames.map((t) => '`' + t + '`').join(',')};`);
                            showToast(`Table check finished.`);
                          }}
                          className="px-2.5 py-1 rounded-lg bg-sky-600 hover:bg-sky-700 text-white shadow-2xs cursor-pointer transition"
                        >
                          Check
                        </button>
                        <button
                          onClick={handleEmptySelectedTables}
                          className="px-2.5 py-1 rounded-lg bg-amber-600 hover:bg-amber-700 text-white shadow-2xs cursor-pointer transition"
                        >
                          Empty (Truncate)
                        </button>
                        <button
                          onClick={handleDropSelectedTables}
                          className="px-2.5 py-1 rounded-lg bg-rose-600 hover:bg-rose-700 text-white shadow-2xs cursor-pointer transition"
                        >
                          Drop
                        </button>
                        <button
                          onClick={() => setSelectedTableNames([])}
                          className="px-2 py-1 text-slate-500 hover:text-slate-800 cursor-pointer ml-1"
                        >
                          Clear
                        </button>
                      </div>
                    )}
                  </div>

                  {/* Database Tables Table */}
                  <div className="overflow-x-auto border border-slate-200 rounded-xl shadow-2xs">
                    <table className="w-full text-left text-xs border-collapse">
                      <thead>
                        <tr className="border-b border-slate-200 bg-slate-50 text-slate-700 font-bold text-[11px] select-none">
                          <th className="p-3 w-10 text-center">
                            <input
                              type="checkbox"
                              checked={selectedTableNames.length === sortedTableDetails.length && sortedTableDetails.length > 0}
                              onChange={(e) => {
                                if (e.target.checked) setSelectedTableNames(sortedTableDetails.map((t) => t.name));
                                else setSelectedTableNames([]);
                              }}
                              className="rounded border-slate-300 text-emerald-600 focus:ring-emerald-500 cursor-pointer w-4 h-4"
                            />
                          </th>
                          <th
                            onClick={() => {
                              if (tableSortCol === 'name') setTableSortOrder(prev => prev === 'ASC' ? 'DESC' : 'ASC');
                              else { setTableSortCol('name'); setTableSortOrder('ASC'); }
                            }}
                            className="p-3 cursor-pointer hover:bg-slate-100 transition"
                          >
                            <div className="flex items-center gap-1.5">
                              <span>Table</span>
                              {tableSortCol === 'name' ? (
                                tableSortOrder === 'ASC' ? <SortAsc className="w-3.5 h-3.5 text-emerald-600" /> : <SortDesc className="w-3.5 h-3.5 text-emerald-600" />
                              ) : (
                                <ArrowUpDown className="w-3 h-3 text-slate-400" />
                              )}
                            </div>
                          </th>
                          <th className="p-3 font-sans text-center">Action</th>
                          <th
                            onClick={() => {
                              if (tableSortCol === 'rows') setTableSortOrder(prev => prev === 'ASC' ? 'DESC' : 'ASC');
                              else { setTableSortCol('rows'); setTableSortOrder('DESC'); }
                            }}
                            className="p-3 text-right cursor-pointer hover:bg-slate-100 transition"
                          >
                            <div className="flex items-center justify-end gap-1.5">
                              <span>Rows</span>
                              {tableSortCol === 'rows' ? (
                                tableSortOrder === 'ASC' ? <SortAsc className="w-3.5 h-3.5 text-emerald-600" /> : <SortDesc className="w-3.5 h-3.5 text-emerald-600" />
                              ) : (
                                <ArrowUpDown className="w-3 h-3 text-slate-400" />
                              )}
                            </div>
                          </th>
                          <th
                            onClick={() => {
                              if (tableSortCol === 'engine') setTableSortOrder(prev => prev === 'ASC' ? 'DESC' : 'ASC');
                              else { setTableSortCol('engine'); setTableSortOrder('ASC'); }
                            }}
                            className="p-3 cursor-pointer hover:bg-slate-100 transition"
                          >
                            <div className="flex items-center gap-1.5">
                              <span>Type</span>
                              {tableSortCol === 'engine' ? (
                                tableSortOrder === 'ASC' ? <SortAsc className="w-3.5 h-3.5 text-emerald-600" /> : <SortDesc className="w-3.5 h-3.5 text-emerald-600" />
                              ) : (
                                <ArrowUpDown className="w-3 h-3 text-slate-400" />
                              )}
                            </div>
                          </th>
                          <th
                            onClick={() => {
                              if (tableSortCol === 'collation') setTableSortOrder(prev => prev === 'ASC' ? 'DESC' : 'ASC');
                              else { setTableSortCol('collation'); setTableSortOrder('ASC'); }
                            }}
                            className="p-3 cursor-pointer hover:bg-slate-100 transition"
                          >
                            <div className="flex items-center gap-1.5">
                              <span>Collation</span>
                              {tableSortCol === 'collation' ? (
                                tableSortOrder === 'ASC' ? <SortAsc className="w-3.5 h-3.5 text-emerald-600" /> : <SortDesc className="w-3.5 h-3.5 text-emerald-600" />
                              ) : (
                                <ArrowUpDown className="w-3 h-3 text-slate-400" />
                              )}
                            </div>
                          </th>
                          <th
                            onClick={() => {
                              if (tableSortCol === 'size') setTableSortOrder(prev => prev === 'ASC' ? 'DESC' : 'ASC');
                              else { setTableSortCol('size'); setTableSortOrder('DESC'); }
                            }}
                            className="p-3 text-right cursor-pointer hover:bg-slate-100 transition"
                          >
                            <div className="flex items-center justify-end gap-1.5">
                              <span>Size</span>
                              {tableSortCol === 'size' ? (
                                tableSortOrder === 'ASC' ? <SortAsc className="w-3.5 h-3.5 text-emerald-600" /> : <SortDesc className="w-3.5 h-3.5 text-emerald-600" />
                              ) : (
                                <ArrowUpDown className="w-3 h-3 text-slate-400" />
                              )}
                            </div>
                          </th>
                          <th className="p-3">Comment</th>
                        </tr>
                      </thead>
                      <tbody className="divide-y divide-slate-100 font-mono text-xs">
                        {loadingTableDetails ? (
                          <tr>
                            <td colSpan={8} className="p-10 text-center text-slate-400 font-sans">
                              <div className="flex flex-col items-center justify-center gap-2">
                                <DatabaseIcon className="w-6 h-6 text-amber-500 animate-spin" />
                                <span>Loading database tables from MySQL...</span>
                              </div>
                            </td>
                          </tr>
                        ) : sortedTableDetails.length === 0 ? (
                          <tr>
                            <td colSpan={8} className="p-10 text-center text-slate-400 font-sans">
                              <p className="font-semibold text-slate-600">No tables match your filter criteria.</p>
                              <p className="text-xs text-slate-400 mt-1">Click &quot;Create Table&quot; to add a new table to `{currentDb}`.</p>
                            </td>
                          </tr>
                        ) : (
                          sortedTableDetails.map((t) => {
                            const isChecked = selectedTableNames.includes(t.name);
                            return (
                              <tr
                                key={t.name}
                                className={`group hover:bg-slate-50 transition ${
                                  isChecked ? 'bg-amber-50/40' : ''
                                }`}
                              >
                                <td className="p-2.5 text-center">
                                  <input
                                    type="checkbox"
                                    checked={isChecked}
                                    onChange={(e) => {
                                      if (e.target.checked) setSelectedTableNames((prev) => [...prev, t.name]);
                                      else setSelectedTableNames((prev) => prev.filter((n) => n !== t.name));
                                    }}
                                    className="rounded border-slate-300 text-emerald-600 focus:ring-emerald-500 cursor-pointer w-4 h-4"
                                  />
                                </td>
                                <td className="p-2.5 font-bold text-slate-900">
                                  <div className="flex items-center gap-1.5">
                                    <button
                                      onClick={() => handleOpenTableBrowse(t.name)}
                                      className="hover:text-emerald-700 text-left cursor-pointer flex items-center gap-1.5 transition group-hover:underline"
                                    >
                                      <Table className="w-3.5 h-3.5 text-slate-400 group-hover:text-emerald-600 shrink-0" />
                                      <span className="font-bold">{t.name}</span>
                                    </button>
                                    <button
                                      onClick={(e) => {
                                        e.stopPropagation();
                                        navigator.clipboard.writeText(t.name);
                                        showToast(`Copied table '${t.name}'`);
                                      }}
                                      className="opacity-0 group-hover:opacity-100 p-1 text-slate-400 hover:text-slate-700 rounded transition cursor-pointer"
                                      title="Copy table name"
                                    >
                                      <Copy className="w-3 h-3" />
                                    </button>
                                  </div>
                                </td>
                                <td className="p-2.5 font-sans text-center">
                                  <div className="flex items-center justify-center gap-1.5 flex-nowrap">
                                    {/* Browse Pill */}
                                    <button
                                      onClick={() => handleOpenTableBrowse(t.name)}
                                      className="inline-flex items-center gap-1 px-2.5 py-1 rounded-lg text-xs font-bold bg-emerald-50 text-emerald-700 hover:bg-emerald-100 border border-emerald-200 transition shadow-2xs cursor-pointer"
                                      title="Browse table rows"
                                    >
                                      <Eye className="w-3.5 h-3.5 text-emerald-600" />
                                      <span>Browse</span>
                                    </button>

                                    {/* Structure Pill */}
                                    <button
                                      onClick={() => handleOpenTableStructure(t.name)}
                                      className="inline-flex items-center gap-1 px-2.5 py-1 rounded-lg text-xs font-bold bg-sky-50 text-sky-700 hover:bg-sky-100 border border-sky-200 transition shadow-2xs cursor-pointer"
                                      title="View table structure"
                                    >
                                      <Columns className="w-3.5 h-3.5 text-sky-600" />
                                      <span>Structure</span>
                                    </button>

                                    {/* SQL Pill */}
                                    <button
                                      onClick={() => {
                                        setSqlQuery(`SELECT * FROM \`${t.name}\` LIMIT 25;`);
                                        setActiveTab('sql');
                                      }}
                                      className="inline-flex items-center gap-1 px-2 py-1 rounded-lg text-xs font-bold bg-indigo-50 text-indigo-700 hover:bg-indigo-100 border border-indigo-200 transition shadow-2xs cursor-pointer"
                                      title="Query table in SQL console"
                                    >
                                      <Terminal className="w-3.5 h-3.5 text-indigo-600" />
                                      <span>SQL</span>
                                    </button>

                                    {/* Empty Pill */}
                                    <button
                                      onClick={async () => {
                                        if (!confirm(`Empty (TRUNCATE) table \`${t.name}\`? All rows will be permanently deleted.`)) return;
                                        await handleExecuteSql(`TRUNCATE TABLE \`${t.name}\`;`);
                                        await fetchTableDetails(currentDb);
                                        showToast(`Table \`${t.name}\` emptied successfully.`);
                                      }}
                                      className="inline-flex items-center gap-1 px-2 py-1 rounded-lg text-xs font-bold bg-amber-50 text-amber-700 hover:bg-amber-100 border border-amber-200 transition shadow-2xs cursor-pointer"
                                      title="Truncate table (empty rows)"
                                    >
                                      <RotateCcw className="w-3.5 h-3.5 text-amber-600" />
                                      <span>Empty</span>
                                    </button>

                                    {/* Drop Pill */}
                                    <button
                                      onClick={async () => {
                                        if (!confirm(`Are you sure you want to permanently DROP table \`${t.name}\`? Table structure and data will be destroyed.`)) return;
                                        await handleExecuteSql(`DROP TABLE IF EXISTS \`${t.name}\`;`);
                                        await fetchTableDetails(currentDb);
                                        await fetchTree();
                                        showToast(`Table \`${t.name}\` dropped.`);
                                      }}
                                      className="inline-flex items-center gap-1 px-2 py-1 rounded-lg text-xs font-bold bg-rose-50 text-rose-700 hover:bg-rose-100 border border-rose-200 transition shadow-2xs cursor-pointer"
                                      title="Drop table permanently"
                                    >
                                      <Trash2 className="w-3.5 h-3.5 text-rose-600" />
                                      <span>Drop</span>
                                    </button>
                                  </div>
                                </td>
                                <td className="p-2.5 text-right font-mono">
                                  {t.rows > 0 ? (
                                    <span className="font-bold text-slate-900 bg-slate-100 px-2 py-0.5 rounded-md border border-slate-200">
                                      {t.rows.toLocaleString()}
                                    </span>
                                  ) : (
                                    <span className="text-slate-400">0</span>
                                  )}
                                </td>
                                <td className="p-2.5">
                                  <span className="px-2 py-0.5 rounded-md text-[11px] font-mono bg-slate-100 text-slate-700 border border-slate-200 font-medium">
                                    {t.engine || t.type || 'InnoDB'}
                                  </span>
                                </td>
                                <td className="p-2.5 text-slate-500 font-mono text-[11px]">{t.collation}</td>
                                <td className="p-2.5 text-right font-mono font-medium">
                                  {t.total_size_kb >= 1024 ? (
                                    <span className="font-bold text-indigo-700">
                                      {(t.total_size_kb / 1024).toFixed(2)} MB
                                    </span>
                                  ) : (
                                    <span className="text-slate-600">
                                      {t.total_size_kb.toLocaleString()} KB
                                    </span>
                                  )}
                                </td>
                                <td className="p-2.5 text-slate-400 truncate max-w-xs">{t.comment || '-'}</td>
                              </tr>
                            );
                          })
                        )}
                      </tbody>
                      {/* Aggregate Summary Footer */}
                      {sortedTableDetails.length > 0 && (
                        <tfoot className="bg-slate-50 border-t-2 border-slate-200 font-mono text-xs text-slate-700">
                          <tr>
                            <td className="p-3 text-center">
                              <CheckSquare className="w-4 h-4 text-slate-400 mx-auto" />
                            </td>
                            <td className="p-3 font-bold text-slate-900">
                              Total: {sortedTableDetails.length} tables
                            </td>
                            <td className="p-3 text-center font-sans">
                              <span className="text-slate-500 text-[11px]">
                                {selectedTableNames.length > 0 ? `${selectedTableNames.length} selected` : 'Click rows to select'}
                              </span>
                            </td>
                            <td className="p-3 text-right font-bold text-emerald-700">
                              {dbMetrics.rows.toLocaleString()} rows
                            </td>
                            <td className="p-3 text-slate-500">
                              InnoDB / MySQL 8.0
                            </td>
                            <td className="p-3 text-slate-500 text-[11px]">
                              {dbMetrics.defaultCollation}
                            </td>
                            <td className="p-3 text-right font-bold text-indigo-700">
                              {dbMetrics.sizeFormatted}
                            </td>
                            <td className="p-3 text-slate-400">-</td>
                          </tr>
                        </tfoot>
                      )}
                    </table>
                  </div>
                </div>
              )}
            </div>
          )}

          {/* TAB 2: BROWSE ROWS */}
          {activeTab === 'browse' && (
            <div className="bg-white border border-slate-200 rounded-2xl shadow-xs overflow-hidden space-y-3">
              {/* Browse Header */}
              <div className="p-4 border-b border-slate-100 flex flex-wrap items-center justify-between gap-3">
                <div>
                  <h2 className="font-bold text-sm text-slate-900 flex items-center gap-2">
                    <Eye className="w-4 h-4 text-emerald-500" />
                    <span>Browse: <code className="font-mono text-emerald-600">{currentDb}.{selectedTable}</code></span>
                  </h2>
                  <p className="text-xs text-slate-500">
                    Showing rows {browseData ? (browseData.page - 1) * browseData.limit + 1 : 0} -{' '}
                    {browseData ? Math.min(browseData.page * browseData.limit, browseData.total_rows) : 0} of{' '}
                    {browseData?.total_rows || 0} (Execution: {browseData?.execution_time || '0.001s'})
                  </p>
                </div>
                <div className="flex items-center gap-2">
                  <button
                    onClick={handleOpenInsertRow}
                    className="flex items-center gap-1.5 px-3 py-1.5 rounded-xl bg-emerald-600 hover:bg-emerald-700 text-white font-bold text-xs shadow-xs cursor-pointer transition"
                  >
                    <Plus className="w-3.5 h-3.5" />
                    <span>Insert Row</span>
                  </button>
                  <button
                    onClick={() => handleOpenTableStructure(selectedTable)}
                    className="px-3 py-1.5 rounded-xl bg-slate-100 hover:bg-slate-200 text-slate-700 font-bold text-xs cursor-pointer transition"
                  >
                    View Structure
                  </button>
                </div>
              </div>

              {/* Filter / Search Bar */}
              <div className="px-4 flex flex-wrap items-center justify-between gap-3 text-xs">
                <div className="relative w-72">
                  <Search className="w-3.5 h-3.5 absolute left-2.5 top-2.5 text-slate-400" />
                  <input
                    type="text"
                    placeholder="Search in table..."
                    value={browseSearch}
                    onChange={(e) => {
                      setBrowseSearch(e.target.value);
                      fetchBrowseRows(currentDb, selectedTable, 1, browseLimit, browseSortCol, browseSortOrder, e.target.value);
                    }}
                    className="w-full pl-8 pr-3 py-1.5 rounded-xl bg-slate-50 border border-slate-200 font-mono text-xs focus:outline-none focus:border-amber-500"
                  />
                </div>
                <div className="flex items-center gap-2">
                  <span className="text-slate-500">Number of rows:</span>
                  <select
                    value={browseLimit}
                    onChange={(e) => {
                      const newLim = Number(e.target.value);
                      setBrowseLimit(newLim);
                      fetchBrowseRows(currentDb, selectedTable, 1, newLim, browseSortCol, browseSortOrder, browseSearch);
                    }}
                    className="px-2 py-1 rounded-lg bg-slate-50 border border-slate-200 font-mono text-xs"
                  >
                    <option value={25}>25</option>
                    <option value={50}>50</option>
                    <option value={100}>100</option>
                    <option value={250}>250</option>
                  </select>
                </div>
              </div>

              {/* Live Rows Table */}
              <div className="overflow-x-auto">
                <table className="w-full text-left text-xs border-collapse font-mono">
                  <thead>
                    <tr className="border-b border-slate-200 bg-slate-50 text-slate-700 font-bold text-[11px]">
                      <th className="p-2.5 w-24 text-center font-sans">Action</th>
                      {(browseData?.columns || []).map((col) => {
                        const isSorted = browseSortCol === col;
                        return (
                          <th
                            key={col}
                            onClick={() => {
                              const newOrder = isSorted && browseSortOrder === 'ASC' ? 'DESC' : 'ASC';
                              setBrowseSortCol(col);
                              setBrowseSortOrder(newOrder);
                              fetchBrowseRows(currentDb, selectedTable, 1, browseLimit, col, newOrder, browseSearch);
                            }}
                            className="p-2.5 whitespace-nowrap cursor-pointer hover:bg-slate-100 select-none"
                          >
                            <div className="flex items-center gap-1">
                              <span>{col}</span>
                              {isSorted ? (
                                browseSortOrder === 'ASC' ? (
                                  <SortAsc className="w-3 h-3 text-amber-500" />
                                ) : (
                                  <SortDesc className="w-3 h-3 text-amber-500" />
                                )
                              ) : (
                                <ArrowUpDown className="w-3 h-3 text-slate-400" />
                              )}
                            </div>
                          </th>
                        );
                      })}
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-slate-100">
                    {loadingBrowse ? (
                      <tr>
                        <td colSpan={(browseData?.columns.length || 0) + 1} className="p-8 text-center text-slate-400 font-sans">
                          Querying live table rows...
                        </td>
                      </tr>
                    ) : !browseData || browseData.rows.length === 0 ? (
                      <tr>
                        <td colSpan={(browseData?.columns.length || 0) + 1} className="p-8 text-center text-slate-400 font-sans">
                          Table `{selectedTable}` has 0 records.
                        </td>
                      </tr>
                    ) : (
                      browseData.rows.map((row, idx) => (
                        <tr key={idx} className="hover:bg-slate-50 transition">
                          <td className="p-2 text-center font-sans">
                            <div className="flex items-center justify-center gap-1.5">
                              <button
                                onClick={() => handleOpenEditRow(row)}
                                className="inline-flex items-center gap-1 px-2 py-0.5 rounded-md text-[11px] font-bold bg-sky-50 text-sky-700 hover:bg-sky-100 border border-sky-200 cursor-pointer shadow-2xs transition"
                                title="Edit row"
                              >
                                <Edit3 className="w-3 h-3 text-sky-600" />
                                <span>Edit</span>
                              </button>
                              <button
                                onClick={() => handleDeleteRow(row)}
                                className="inline-flex items-center gap-1 px-2 py-0.5 rounded-md text-[11px] font-bold bg-rose-50 text-rose-700 hover:bg-rose-100 border border-rose-200 cursor-pointer shadow-2xs transition"
                                title="Delete row"
                              >
                                <Trash2 className="w-3 h-3 text-rose-600" />
                                <span>Delete</span>
                              </button>
                            </div>
                          </td>
                          {browseData.columns.map((col) => {
                            const val = row[col];
                            return (
                              <td key={col} className="p-2.5 whitespace-nowrap text-slate-800">
                                {val !== null && val !== undefined ? (
                                  String(val)
                                ) : (
                                  <em className="text-amber-500/80 font-serif">NULL</em>
                                )}
                              </td>
                            );
                          })}
                        </tr>
                      ))
                    )}
                  </tbody>
                </table>
              </div>

              {/* Pagination Controls */}
              {browseData && browseData.total_pages > 1 && (
                <div className="p-3 bg-white border-t border-slate-200 flex items-center justify-between text-xs font-semibold text-slate-500">
                  <span>
                    Page {browseData.page} of {browseData.total_pages}
                  </span>
                  <div className="flex items-center gap-1 font-mono">
                    <button
                      disabled={browseData.page <= 1}
                      onClick={() => fetchBrowseRows(currentDb, selectedTable, 1, browseLimit, browseSortCol, browseSortOrder, browseSearch)}
                      className="px-2.5 py-1 rounded-lg bg-white hover:bg-slate-50 border border-slate-200 text-slate-700 disabled:opacity-40 cursor-pointer shadow-2xs transition"
                    >
                      &lt;&lt;
                    </button>
                    <button
                      disabled={browseData.page <= 1}
                      onClick={() => fetchBrowseRows(currentDb, selectedTable, browseData.page - 1, browseLimit, browseSortCol, browseSortOrder, browseSearch)}
                      className="px-2.5 py-1 rounded-lg bg-white hover:bg-slate-50 border border-slate-200 text-slate-700 disabled:opacity-40 cursor-pointer shadow-2xs transition"
                    >
                      &lt;
                    </button>
                    <span className="px-3 py-1 font-bold text-slate-900">{browseData.page}</span>
                    <button
                      disabled={browseData.page >= browseData.total_pages}
                      onClick={() => fetchBrowseRows(currentDb, selectedTable, browseData.page + 1, browseLimit, browseSortCol, browseSortOrder, browseSearch)}
                      className="px-2.5 py-1 rounded-lg bg-white hover:bg-slate-50 border border-slate-200 text-slate-700 disabled:opacity-40 cursor-pointer shadow-2xs transition"
                    >
                      &gt;
                    </button>
                    <button
                      disabled={browseData.page >= browseData.total_pages}
                      onClick={() => fetchBrowseRows(currentDb, selectedTable, browseData.total_pages, browseLimit, browseSortCol, browseSortOrder, browseSearch)}
                      className="px-2.5 py-1 rounded-lg bg-white hover:bg-slate-50 border border-slate-200 text-slate-700 disabled:opacity-40 cursor-pointer shadow-2xs transition"
                    >
                      &gt;&gt;
                    </button>
                  </div>
                </div>
              )}
            </div>
          )}

          {/* TAB 3: SQL CONSOLE */}
          {activeTab === 'sql' && (
            <div className="bg-white border border-slate-200 rounded-2xl p-6 shadow-xs space-y-4">
              <div className="flex flex-wrap items-center justify-between gap-3 pb-3 border-b border-slate-100">
                <div>
                  <h2 className="font-bold text-sm text-slate-900 flex items-center gap-2">
                    <Terminal className="w-4 h-4 text-amber-500" />
                    <span>SQL Query Console: <code className="font-mono text-amber-600">{currentDb}</code></span>
                  </h2>
                  <p className="text-xs text-slate-500">Run native SQL commands directly against the database engine</p>
                </div>
                {/* Quick query buttons */}
                <div className="flex flex-wrap gap-1.5 text-xs font-mono">
                  <button
                    onClick={() => setSqlQuery(`SELECT * FROM \`${selectedTable || 'table'}\` LIMIT 50;`)}
                    className="px-2.5 py-1 rounded-lg bg-slate-50 hover:bg-slate-100 border border-slate-200 text-slate-700 font-semibold cursor-pointer shadow-2xs transition"
                  >
                    SELECT *
                  </button>
                  <button
                    onClick={() => setSqlQuery(`SELECT COUNT(*) FROM \`${selectedTable || 'table'}\`;`)}
                    className="px-2.5 py-1 rounded-lg bg-slate-50 hover:bg-slate-100 border border-slate-200 text-slate-700 font-semibold cursor-pointer shadow-2xs transition"
                  >
                    SELECT COUNT
                  </button>
                  <button
                    onClick={() => setSqlQuery(`SHOW CREATE TABLE \`${selectedTable || 'table'}\`;`)}
                    className="px-2.5 py-1 rounded-lg bg-slate-50 hover:bg-slate-100 border border-slate-200 text-slate-700 font-semibold cursor-pointer shadow-2xs transition"
                  >
                    SHOW CREATE
                  </button>
                </div>
              </div>

              {/* SQL Textarea */}
              <div className="relative">
                <textarea
                  rows={6}
                  value={sqlQuery}
                  onChange={(e) => setSqlQuery(e.target.value)}
                  placeholder="Enter your SQL query here... (e.g. SELECT * FROM users WHERE status = 'active';)"
                  className="w-full p-3 font-mono text-xs bg-slate-950 text-emerald-400 rounded-xl border border-slate-800 focus:outline-none focus:ring-1 focus:ring-emerald-500"
                />
              </div>

              <div className="flex justify-between items-center">
                <span className="text-xs text-slate-500 font-mono">Delimiter: ;</span>
                <div className="flex gap-2">
                  <button
                    onClick={() => setSqlQuery('')}
                    className="px-4 py-2 rounded-xl bg-slate-100 hover:bg-slate-200 text-slate-700 text-xs font-bold border border-slate-200 cursor-pointer shadow-2xs transition"
                  >
                    Clear
                  </button>
                  <button
                    onClick={() => handleExecuteSql()}
                    disabled={runningQuery}
                    className="flex items-center gap-1.5 px-5 py-2 rounded-xl bg-emerald-600 hover:bg-emerald-700 text-white text-xs font-bold cursor-pointer shadow-xs disabled:opacity-50 transition"
                  >
                    <Play className="w-3.5 h-3.5 fill-current" />
                    <span>{runningQuery ? 'Executing...' : 'Go / Execute'}</span>
                  </button>
                </div>
              </div>

              {/* Error banner */}
              {queryError && (
                <div className="p-3.5 rounded-xl bg-rose-50 border border-rose-200 text-rose-800 text-xs flex items-start gap-2">
                  <AlertCircle className="w-4 h-4 shrink-0 text-rose-500 mt-0.5" />
                  <div className="font-mono whitespace-pre-wrap">{queryError}</div>
                </div>
              )}

              {/* Query Result */}
              {queryResult && (
                <div className="space-y-2 pt-2 border-t border-slate-100">
                  <div className="flex items-center justify-between text-xs text-slate-500">
                    <span>
                      {queryResult.rows_affected} rows affected / returned ({queryResult.execution_time})
                    </span>
                  </div>
                  {queryResult.columns && queryResult.columns.length > 0 && queryResult.rows && (
                    <div className="overflow-x-auto max-h-96 border border-slate-200 rounded-xl shadow-2xs">
                      <table className="w-full text-left text-xs font-mono border-collapse">
                        <thead className="bg-slate-50 sticky top-0">
                          <tr className="border-b border-slate-200 text-slate-700 font-bold">
                            {queryResult.columns.map((c) => (
                              <th key={c} className="p-2.5 whitespace-nowrap">
                                {c}
                              </th>
                            ))}
                          </tr>
                        </thead>
                        <tbody className="divide-y divide-slate-100">
                          {queryResult.rows.map((r, i) => (
                            <tr key={i} className="hover:bg-slate-50 transition">
                              {queryResult.columns.map((c) => (
                                <td key={c} className="p-2.5 whitespace-nowrap text-slate-800">
                                  {r[c] !== null && r[c] !== undefined ? (
                                    String(r[c])
                                  ) : (
                                    <em className="text-amber-500 font-serif">NULL</em>
                                  )}
                                </td>
                              ))}
                            </tr>
                          ))}
                        </tbody>
                      </table>
                    </div>
                  )}
                </div>
              )}
            </div>
          )}

          {/* TAB 4: EXPORT */}
          {activeTab === 'export' && (
            <div className="bg-white border border-slate-200 rounded-2xl p-6 shadow-xs space-y-6">
              <div className="pb-3 border-b border-slate-100">
                <h2 className="font-bold text-sm text-slate-900">Export Database / Tables</h2>
                <p className="text-xs text-slate-500">
                  Generate and stream live database dumps from schema <code className="font-mono text-amber-600">{currentDb}</code>
                </p>
              </div>

              <div className="space-y-4 text-xs">
                {/* Format selection */}
                <div className="space-y-1.5">
                  <label className="font-bold text-slate-700 block">Export Format</label>
                  <div className="flex gap-4">
                    {(['sql', 'csv', 'json'] as const).map((fmt) => (
                      <label key={fmt} className="flex items-center gap-2 cursor-pointer font-mono font-semibold text-slate-700">
                        <input
                          type="radio"
                          name="format"
                          value={fmt}
                          checked={exportFormat === fmt}
                          onChange={() => setExportFormat(fmt)}
                          className="text-amber-500 focus:ring-amber-500"
                        />
                        <span>{fmt.toUpperCase()}</span>
                      </label>
                    ))}
                  </div>
                </div>

                {/* Table selection */}
                <div className="space-y-1.5">
                  <div className="flex items-center justify-between">
                    <label className="font-bold text-slate-700">Select Tables to Export</label>
                    <button
                      type="button"
                      onClick={() => {
                        if (exportSelectedTables.length === tableDetails.length) setExportSelectedTables([]);
                        else setExportSelectedTables(tableDetails.map((t) => t.name));
                      }}
                      className="text-xs text-blue-600 hover:underline font-semibold cursor-pointer"
                    >
                      {exportSelectedTables.length === tableDetails.length ? 'Deselect All' : 'Select All'}
                    </button>
                  </div>
                  <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 gap-2 max-h-48 overflow-y-auto p-3 border border-slate-200 rounded-xl bg-slate-50 font-mono text-xs">
                    {tableDetails.map((t) => (
                      <label key={t.name} className="flex items-center gap-2 truncate cursor-pointer text-slate-800 hover:text-emerald-700">
                        <input
                          type="checkbox"
                          checked={exportSelectedTables.includes(t.name)}
                          onChange={(e) => {
                            if (e.target.checked) setExportSelectedTables((prev) => [...prev, t.name]);
                            else setExportSelectedTables((prev) => prev.filter((n) => n !== t.name));
                          }}
                          className="rounded text-amber-500 focus:ring-amber-500"
                        />
                        <span className="truncate">{t.name}</span>
                      </label>
                    ))}
                  </div>
                </div>

                <div className="pt-2">
                  <button
                    onClick={handleTriggerExport}
                    className="flex items-center gap-2 px-6 py-2.5 rounded-xl bg-emerald-600 hover:bg-emerald-700 text-white font-bold cursor-pointer shadow-xs transition"
                  >
                    <Download className="w-4 h-4" />
                    <span>Download {exportFormat.toUpperCase()} Export</span>
                  </button>
                </div>
              </div>
            </div>
          )}

          {/* TAB 5: IMPORT */}
          {activeTab === 'import' && (
            <div className="bg-white border border-slate-200 rounded-2xl p-6 shadow-xs space-y-6">
              <div className="pb-3 border-b border-slate-100">
                <h2 className="font-bold text-sm text-slate-900">Import into Database</h2>
                <p className="text-xs text-slate-500">
                  Restore SQL statements directly into schema <code className="font-mono text-amber-600">{currentDb}</code>
                </p>
              </div>

               <div className="space-y-4 text-xs">
                {/* File Upload Area */}
                <div
                  onDragOver={(e) => {
                    e.preventDefault();
                    e.stopPropagation();
                    setIsDraggingImport(true);
                  }}
                  onDragEnter={(e) => {
                    e.preventDefault();
                    e.stopPropagation();
                    setIsDraggingImport(true);
                  }}
                  onDragLeave={(e) => {
                    e.preventDefault();
                    e.stopPropagation();
                    setIsDraggingImport(false);
                  }}
                  onDrop={(e) => {
                    e.preventDefault();
                    e.stopPropagation();
                    setIsDraggingImport(false);
                    if (e.dataTransfer.files && e.dataTransfer.files[0]) {
                      const file = e.dataTransfer.files[0];
                      setImportFile(file);
                    }
                  }}
                  className={`border-2 border-dashed rounded-2xl p-8 text-center transition cursor-pointer ${
                    isDraggingImport
                      ? 'border-amber-500 bg-amber-50/50 scale-[1.01]'
                      : importFile
                        ? 'border-emerald-500 bg-emerald-50/40'
                        : 'border-slate-300 bg-slate-50/60 hover:border-amber-500 hover:bg-slate-50'
                  }`}
                >
                  <Upload className={`w-10 h-10 mx-auto mb-3 transition ${isDraggingImport ? 'text-amber-500 animate-bounce' : importFile ? 'text-emerald-500' : 'text-slate-400'}`} />
                  {importFile ? (
                    <>
                      <p className="font-bold text-emerald-700 text-sm">
                        {importFile.name}
                      </p>
                      <p className="text-slate-500 mt-1">
                        Size: {(importFile.size / (1024 * 1024)).toFixed(2)} MB
                      </p>
                      <button
                        type="button"
                        onClick={(e) => { e.stopPropagation(); setImportFile(null); }}
                        className="mt-2 inline-block px-3 py-1.5 rounded-lg bg-red-50 border border-red-200 text-red-600 font-bold hover:bg-red-100 cursor-pointer text-[11px] transition shadow-2xs"
                      >
                        Remove File
                      </button>
                    </>
                  ) : (
                    <>
                      <p className="font-bold text-slate-900 text-sm">
                        {isDraggingImport ? 'Drop your .sql file here' : 'Drag & drop .sql file here'}
                      </p>
                      <p className="text-[11px] text-slate-500 mt-1">Maximum upload size: 128 MB</p>
                    </>
                  )}
                  <input
                    type="file"
                    accept=".sql,.txt"
                    id="sql-file-input"
                    className="hidden"
                    onChange={(e) => {
                      if (e.target.files && e.target.files[0]) {
                        setImportFile(e.target.files[0]);
                      }
                    }}
                  />
                  {!importFile && (
                    <label
                      htmlFor="sql-file-input"
                      className="mt-3 inline-block px-5 py-2.5 rounded-xl bg-white border border-slate-300 font-bold hover:bg-slate-50 cursor-pointer text-slate-800 shadow-2xs transition"
                    >
                      Browse File
                    </label>
                  )}
                </div>

                {/* Or paste SQL */}
                <div className="space-y-1">
                  <label className="font-bold text-slate-700">Or Paste Raw SQL Statements:</label>
                  <textarea
                    rows={6}
                    value={importSqlText}
                    onChange={(e) => setImportSqlText(e.target.value)}
                    placeholder="CREATE TABLE ...; INSERT INTO ...;"
                    className="w-full p-3 font-mono text-xs bg-slate-50 border border-slate-300 rounded-xl focus:outline-none focus:border-amber-500 text-slate-900"
                  />
                </div>

                {importing && (
                  <div className="p-4 rounded-xl bg-slate-50 border border-slate-200 space-y-2.5">
                    <div className="flex items-center justify-between text-xs font-bold">
                      <div className="flex items-center gap-2">
                        <RefreshCw className="w-3.5 h-3.5 text-amber-500 animate-spin" />
                        <span className="text-slate-800">{importStatusText || 'Importing SQL dump into database...'}</span>
                      </div>
                      <span className="text-amber-600 font-mono text-sm">{importProgress}%</span>
                    </div>
                    <div className="w-full bg-slate-200 h-2.5 rounded-full overflow-hidden">
                      <div
                        className="bg-gradient-to-r from-amber-500 via-amber-400 to-emerald-500 h-full transition-all duration-300 rounded-full"
                        style={{ width: `${importProgress}%` }}
                      />
                    </div>
                    <p className="text-[11px] text-slate-500">Executing tables &amp; rows restoration. Please keep this tab open.</p>
                  </div>
                )}

                {importResult && (
                  <div className={`p-4 rounded-xl border space-y-2 ${
                    importResult.failed > 0
                      ? 'bg-amber-50 border-amber-200 text-amber-800'
                      : 'bg-emerald-50 border-emerald-200 text-emerald-800'
                  }`}>
                    <p className="font-bold flex items-center gap-2">
                      <span>Import Completed:</span>
                    </p>
                    <div className="grid grid-cols-2 sm:grid-cols-4 gap-2 text-xs">
                      <div className="p-2.5 rounded-lg bg-white border border-slate-200 shadow-2xs">
                        <span className="text-slate-500 block">Total Statements</span>
                        <span className="font-mono font-bold text-sm text-slate-900">{importResult.total}</span>
                      </div>
                      <div className="p-2.5 rounded-lg bg-white border border-slate-200 shadow-2xs">
                        <span className="text-slate-500 block">Successful</span>
                        <span className="font-mono font-bold text-sm text-emerald-600">{importResult.successful}</span>
                      </div>
                      <div className="p-2.5 rounded-lg bg-white border border-slate-200 shadow-2xs">
                        <span className="text-slate-500 block">Failed</span>
                        <span className="font-mono font-bold text-sm text-rose-600">{importResult.failed}</span>
                      </div>
                      {importResult.tables !== undefined && (
                        <div className="p-2.5 rounded-lg bg-white border border-slate-200 shadow-2xs">
                          <span className="text-slate-500 block">Live Tables</span>
                          <span className="font-mono font-bold text-sm text-cyan-600">{importResult.tables}</span>
                        </div>
                      )}
                    </div>

                    {importResult.errors && importResult.errors.length > 0 && (
                      <div className="mt-3 text-xs bg-rose-50 p-3 rounded-lg border border-rose-200 max-h-48 overflow-y-auto font-mono text-rose-700 space-y-1">
                        <p className="font-bold text-rose-800">Execution Errors ({importResult.errors.length}):</p>
                        {importResult.errors.map((err, i) => (
                          <p key={i} className="break-all">• {err}</p>
                        ))}
                      </div>
                    )}
                  </div>
                )}

                <div className="pt-2 flex items-center gap-3">
                  <button
                    onClick={handleStartImport}
                    disabled={importing || (!importFile && !importSqlText.trim())}
                    className="flex items-center gap-2 px-6 py-2.5 rounded-xl bg-emerald-600 hover:bg-emerald-700 text-white font-bold cursor-pointer shadow-xs disabled:opacity-50 transition"
                  >
                    <Upload className="w-4 h-4" />
                    <span>{importing ? 'Importing Statements...' : 'Go / Start Import'}</span>
                  </button>
                  {(importFile || importSqlText.trim()) && !importing && (
                    <button
                      onClick={() => { setImportFile(null); setImportSqlText(''); setImportResult(null); setImportProgress(0); setImportStatusText(''); }}
                      className="flex items-center gap-1.5 px-4 py-2.5 rounded-xl bg-slate-100 hover:bg-slate-200 text-slate-700 font-bold border border-slate-200 cursor-pointer shadow-2xs transition"
                    >
                      <RefreshCw className="w-3.5 h-3.5" />
                      <span>Clear</span>
                    </button>
                  )}
                </div>
              </div>
            </div>
          )}

          {/* TAB 6: OPERATIONS */}
          {activeTab === 'operations' && (
            <div className="bg-white border border-slate-200 rounded-2xl p-6 shadow-xs space-y-6">
              <div className="pb-3 border-b border-slate-100">
                <h2 className="font-bold text-sm text-slate-900">Database &amp; Table Operations</h2>
                <p className="text-xs text-slate-500">
                  Maintenance, collation alteration, table rename, copy, and optimization
                </p>
              </div>

              {opMaintenanceMsg && (
                <div className="p-4 rounded-xl bg-slate-950 text-emerald-400 font-mono text-xs whitespace-pre-wrap border border-slate-800 shadow-2xs">
                  {opMaintenanceMsg}
                </div>
              )}

              <div className="grid grid-cols-1 md:grid-cols-2 gap-4 text-xs">
                {/* Database Collation */}
                <div className="p-5 rounded-2xl border border-slate-200 bg-white shadow-2xs space-y-3">
                  <h3 className="font-bold text-slate-900 text-sm">Database Collation</h3>
                  <p className="text-slate-500 text-[11px]">Alter default character collation for {currentDb}</p>
                  <div className="flex gap-2">
                    <select
                      value={opCollation}
                      onChange={(e) => setOpCollation(e.target.value)}
                      className="px-3 py-2 rounded-xl bg-white border border-slate-300 text-slate-800 font-mono text-xs flex-1 focus:ring-1 focus:ring-emerald-500"
                    >
                      <option value="utf8mb4_unicode_ci">utf8mb4_unicode_ci</option>
                      <option value="utf8mb4_general_ci">utf8mb4_general_ci</option>
                      <option value="utf8_general_ci">utf8_general_ci</option>
                      <option value="latin1_swedish_ci">latin1_swedish_ci</option>
                    </select>
                    <button
                      onClick={() => handleTableOperation('collation')}
                      disabled={opRunning}
                      className="px-4 py-2 rounded-xl bg-emerald-600 hover:bg-emerald-700 text-white font-bold cursor-pointer shadow-xs transition"
                    >
                      Save
                    </button>
                  </div>
                </div>

                {/* Table Maintenance */}
                {selectedTable && (
                  <div className="p-5 rounded-2xl border border-slate-200 bg-white shadow-2xs space-y-3">
                    <h3 className="font-bold text-slate-900 text-sm">Table Maintenance: `{selectedTable}`</h3>
                    <p className="text-slate-500 text-[11px]">Defragment indexes and verify storage consistency</p>
                    <div className="flex flex-wrap gap-2">
                      <button
                        onClick={() => handleTableOperation('optimize')}
                        disabled={opRunning}
                        className="px-3.5 py-1.5 rounded-xl bg-emerald-50 hover:bg-emerald-100 text-emerald-700 border border-emerald-200 font-bold cursor-pointer shadow-2xs transition"
                      >
                        Optimize Table
                      </button>
                      <button
                        onClick={() => handleTableOperation('check')}
                        disabled={opRunning}
                        className="px-3.5 py-1.5 rounded-xl bg-sky-50 hover:bg-sky-100 text-sky-700 border border-sky-200 font-bold cursor-pointer shadow-2xs transition"
                      >
                        Check Table
                      </button>
                      <button
                        onClick={() => handleTableOperation('analyze')}
                        disabled={opRunning}
                        className="px-3.5 py-1.5 rounded-xl bg-purple-50 hover:bg-purple-100 text-purple-700 border border-purple-200 font-bold cursor-pointer shadow-2xs transition"
                      >
                        Analyze Table
                      </button>
                      <button
                        onClick={() => handleTableOperation('repair')}
                        disabled={opRunning}
                        className="px-3.5 py-1.5 rounded-xl bg-amber-50 hover:bg-amber-100 text-amber-700 border border-amber-200 font-bold cursor-pointer shadow-2xs transition"
                      >
                        Repair Table
                      </button>
                    </div>
                  </div>
                )}

                {/* Table Rename */}
                {selectedTable && (
                  <div className="p-5 rounded-2xl border border-slate-200 bg-white shadow-2xs space-y-3">
                    <h3 className="font-bold text-slate-900 text-sm">Rename Table to:</h3>
                    <div className="flex gap-2">
                      <input
                        type="text"
                        placeholder="new_table_name"
                        value={opRenameTable}
                        onChange={(e) => setOpRenameTable(e.target.value)}
                        className="px-3 py-2 rounded-xl bg-white border border-slate-300 text-slate-800 font-mono text-xs flex-1 focus:ring-1 focus:ring-emerald-500"
                      />
                      <button
                        onClick={() => handleTableOperation('rename')}
                        disabled={opRunning || !opRenameTable.trim()}
                        className="px-4 py-2 rounded-xl bg-emerald-600 hover:bg-emerald-700 text-white font-bold cursor-pointer shadow-xs disabled:opacity-50 transition"
                      >
                        Rename
                      </button>
                    </div>
                  </div>
                )}

                {/* Table Copy */}
                {selectedTable && (
                  <div className="p-5 rounded-2xl border border-slate-200 bg-white shadow-2xs space-y-3">
                    <h3 className="font-bold text-slate-900 text-sm">Copy Table to:</h3>
                    <div className="space-y-2">
                      <input
                        type="text"
                        placeholder="copy_table_name"
                        value={opCopyTable}
                        onChange={(e) => setOpCopyTable(e.target.value)}
                        className="w-full px-3 py-2 rounded-xl bg-white border border-slate-300 text-slate-800 font-mono text-xs focus:ring-1 focus:ring-emerald-500"
                      />
                      <label className="flex items-center gap-2 cursor-pointer font-semibold text-slate-700">
                        <input
                          type="checkbox"
                          checked={opCopyData}
                          onChange={(e) => setOpCopyData(e.target.checked)}
                          className="rounded text-amber-500 focus:ring-amber-500"
                        />
                        <span>Copy table data and rows</span>
                      </label>
                      <button
                        onClick={() => handleTableOperation('copy')}
                        disabled={opRunning || !opCopyTable.trim()}
                        className="px-4 py-2 rounded-xl bg-emerald-600 hover:bg-emerald-700 text-white font-bold cursor-pointer shadow-xs disabled:opacity-50 transition"
                      >
                        Copy Table
                      </button>
                    </div>
                  </div>
                )}

                {/* Truncate / Drop Table */}
                {selectedTable && (
                  <div className="p-5 rounded-2xl border border-rose-200 bg-rose-50/40 space-y-3 md:col-span-2">
                    <h3 className="font-bold text-rose-800 text-sm">Danger Zone: `{selectedTable}`</h3>
                    <div className="flex flex-wrap gap-3">
                      <button
                        onClick={() => handleTableOperation('truncate')}
                        disabled={opRunning}
                        className="px-4 py-2 rounded-xl bg-amber-600 hover:bg-amber-700 text-white font-bold cursor-pointer shadow-xs transition"
                      >
                        Empty Table (TRUNCATE)
                      </button>
                      <button
                        onClick={() => handleTableOperation('drop')}
                        disabled={opRunning}
                        className="px-4 py-2 rounded-xl bg-rose-700 hover:bg-rose-800 text-white font-bold cursor-pointer shadow-xs transition"
                      >
                        Delete Table (DROP)
                      </button>
                    </div>
                  </div>
                )}

                {/* Database Danger Zone: Drop Entire Database */}
                {!['mysql', 'information_schema', 'performance_schema', 'sys'].includes(currentDb.toLowerCase()) && (
                  <div className="p-5 rounded-2xl border border-red-300 bg-red-50/60 space-y-3 md:col-span-2">
                    <div className="flex items-center gap-2">
                      <Trash2 className="w-4 h-4 text-red-600" />
                      <h3 className="font-bold text-red-900 text-sm">Database Danger Zone: `{currentDb}`</h3>
                    </div>
                    <p className="text-xs text-red-700">
                      Permanently drop database <strong className="font-mono">{currentDb}</strong> and all of its tables, routines, and data from MySQL. This action is irreversible.
                    </p>
                    <button
                      onClick={() => handleDropDatabase(currentDb)}
                      disabled={opRunning}
                      className="px-4 py-2 rounded-xl bg-red-700 hover:bg-red-800 text-white font-bold text-xs shadow-xs cursor-pointer flex items-center gap-2 transition disabled:opacity-50"
                    >
                      <Trash2 className="w-3.5 h-3.5" />
                      <span>Drop Database ({currentDb})</span>
                    </button>
                  </div>
                )}
              </div>
            </div>
          )}

          {/* TAB 7: ROUTINES */}
          {activeTab === 'routines' && (
            <div className="bg-white border border-slate-200 rounded-2xl p-5 shadow-xs space-y-4">
              <div className="flex items-center justify-between pb-3 border-b border-slate-100">
                <div>
                  <h2 className="font-bold text-sm text-slate-900">Stored Procedures &amp; Functions</h2>
                  <p className="text-xs text-slate-500">Manage database routines for {currentDb}</p>
                </div>
                <button
                  onClick={() => setAddRoutineModalOpen(true)}
                  className="flex items-center gap-1.5 px-3.5 py-1.5 rounded-xl bg-emerald-600 hover:bg-emerald-700 text-white font-bold text-xs shadow-xs cursor-pointer transition"
                >
                  <Plus className="w-3.5 h-3.5" />
                  <span>Add Routine</span>
                </button>
              </div>

              <div className="overflow-x-auto border border-slate-200 rounded-xl shadow-2xs">
                <table className="w-full text-left text-xs border-collapse">
                  <thead>
                    <tr className="border-b border-slate-200 bg-slate-50 font-bold text-[11px] text-slate-700">
                      <th className="p-2.5">Name</th>
                      <th className="p-2.5">Type</th>
                      <th className="p-2.5">Data Type</th>
                      <th className="p-2.5">Security</th>
                      <th className="p-2.5 text-center">Action</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-slate-100 font-mono text-xs">
                    {loadingEntities ? (
                      <tr>
                        <td colSpan={5} className="p-6 text-center text-slate-400 font-sans">
                          Loading routines...
                        </td>
                      </tr>
                    ) : routines.length === 0 ? (
                      <tr>
                        <td colSpan={5} className="p-6 text-center text-slate-400 font-sans">
                          No stored procedures or functions found in `{currentDb}`.
                        </td>
                      </tr>
                    ) : (
                      routines.map((r) => (
                        <tr key={r.name} className="hover:bg-slate-50 transition">
                          <td className="p-2.5 font-bold text-slate-900">{r.name}</td>
                          <td className="p-2.5">
                            <span className="px-2 py-0.5 rounded text-[10px] font-bold bg-purple-100 text-purple-800">
                              {r.type}
                            </span>
                          </td>
                          <td className="p-2.5 text-slate-500">{r.data_type || '-'}</td>
                          <td className="p-2.5 text-slate-500">{r.security || 'DEFINER'}</td>
                          <td className="p-2.5 font-sans text-center">
                            <div className="flex items-center justify-center gap-1.5">
                              <button
                                onClick={() => {
                                  setSqlQuery(`CALL \`${r.name}\`();`);
                                  setActiveTab('sql');
                                }}
                                className="inline-flex items-center gap-1 px-2.5 py-1 rounded-lg text-xs font-semibold bg-emerald-50 text-emerald-700 hover:bg-emerald-100 border border-emerald-200 transition shadow-2xs cursor-pointer"
                                title="Execute routine"
                              >
                                <Zap className="w-3 h-3 text-emerald-600" />
                                <span>Execute</span>
                              </button>
                              <button
                                onClick={() => handleDropRoutine(r)}
                                className="inline-flex items-center gap-1 px-2.5 py-1 rounded-lg text-xs font-semibold bg-rose-50 text-rose-700 hover:bg-rose-100 border border-rose-200 transition shadow-2xs cursor-pointer"
                                title="Drop routine"
                              >
                                <Trash2 className="w-3 h-3 text-rose-600" />
                                <span>Drop</span>
                              </button>
                            </div>
                          </td>
                        </tr>
                      ))
                    )}
                  </tbody>
                </table>
              </div>
            </div>
          )}

          {/* TAB 8: EVENTS */}
          {activeTab === 'events' && (
            <div className="bg-white border border-slate-200 rounded-2xl p-5 shadow-xs space-y-4">
              <div className="flex items-center justify-between pb-3 border-b border-slate-100">
                <div>
                  <h2 className="font-bold text-sm text-slate-900">Scheduled MySQL Events</h2>
                  <p className="text-xs text-slate-500">Cron-like background schedules executed by MySQL</p>
                </div>
                <button
                  onClick={() => setAddEventModalOpen(true)}
                  className="flex items-center gap-1.5 px-3.5 py-1.5 rounded-xl bg-emerald-600 hover:bg-emerald-700 text-white font-bold text-xs shadow-xs cursor-pointer transition"
                >
                  <Plus className="w-3.5 h-3.5" />
                  <span>Add Event</span>
                </button>
              </div>

              <div className="overflow-x-auto border border-slate-200 rounded-xl shadow-2xs">
                <table className="w-full text-left text-xs border-collapse">
                  <thead>
                    <tr className="border-b border-slate-200 bg-slate-50 font-bold text-[11px] text-slate-700">
                      <th className="p-2.5">Name</th>
                      <th className="p-2.5">Status</th>
                      <th className="p-2.5">Type</th>
                      <th className="p-2.5">Interval / Schedule</th>
                      <th className="p-2.5 text-center">Action</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-slate-100 font-mono text-xs">
                    {loadingEntities ? (
                      <tr>
                        <td colSpan={5} className="p-6 text-center text-slate-400 font-sans">
                          Loading events...
                        </td>
                      </tr>
                    ) : events.length === 0 ? (
                      <tr>
                        <td colSpan={5} className="p-6 text-center text-slate-400 font-sans">
                          No scheduled events found in `{currentDb}`.
                        </td>
                      </tr>
                    ) : (
                      events.map((ev) => (
                        <tr key={ev.name} className="hover:bg-slate-50 transition">
                          <td className="p-2.5 font-bold text-slate-900">{ev.name}</td>
                          <td className="p-2.5">
                            <span className="px-2 py-0.5 rounded text-[10px] font-bold bg-emerald-100 text-emerald-800">
                              {ev.status}
                            </span>
                          </td>
                          <td className="p-2.5">{ev.type}</td>
                          <td className="p-2.5 text-slate-500">{ev.interval || ev.starts || '-'}</td>
                          <td className="p-2.5 font-sans text-center">
                            <button
                              onClick={() => handleDropEvent(ev.name)}
                              className="inline-flex items-center gap-1 px-2.5 py-1 rounded-lg text-xs font-semibold bg-rose-50 text-rose-700 hover:bg-rose-100 border border-rose-200 transition shadow-2xs cursor-pointer"
                              title="Drop event"
                            >
                              <Trash2 className="w-3 h-3 text-rose-600" />
                              <span>Drop</span>
                            </button>
                          </td>
                        </tr>
                      ))
                    )}
                  </tbody>
                </table>
              </div>
            </div>
          )}

          {/* TAB 9: TRIGGERS */}
          {activeTab === 'triggers' && (
            <div className="bg-white border border-slate-200 rounded-2xl p-5 shadow-xs space-y-4">
              <div className="flex items-center justify-between pb-3 border-b border-slate-100">
                <div>
                  <h2 className="font-bold text-sm text-slate-900">Database Triggers</h2>
                  <p className="text-xs text-slate-500">Automated triggers on INSERT, UPDATE, or DELETE</p>
                </div>
                <button
                  onClick={() => {
                    setTriggerTable(selectedTable || (tableDetails[0] ? tableDetails[0].name : ''));
                    setAddTriggerModalOpen(true);
                  }}
                  className="flex items-center gap-1.5 px-3.5 py-1.5 rounded-xl bg-emerald-600 hover:bg-emerald-700 text-white font-bold text-xs shadow-xs cursor-pointer transition"
                >
                  <Plus className="w-3.5 h-3.5" />
                  <span>Add Trigger</span>
                </button>
              </div>

              <div className="overflow-x-auto border border-slate-200 rounded-xl shadow-2xs">
                <table className="w-full text-left text-xs border-collapse">
                  <thead>
                    <tr className="border-b border-slate-200 bg-slate-50 font-bold text-[11px] text-slate-700">
                      <th className="p-2.5">Name</th>
                      <th className="p-2.5">Table</th>
                      <th className="p-2.5">Timing</th>
                      <th className="p-2.5">Event</th>
                      <th className="p-2.5 text-center">Action</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-slate-100 font-mono text-xs">
                    {loadingEntities ? (
                      <tr>
                        <td colSpan={5} className="p-6 text-center text-slate-400 font-sans">
                          Loading triggers...
                        </td>
                      </tr>
                    ) : triggers.length === 0 ? (
                      <tr>
                        <td colSpan={5} className="p-6 text-center text-slate-400 font-sans">
                          No triggers found in `{currentDb}`.
                        </td>
                      </tr>
                    ) : (
                      triggers.map((trg) => (
                        <tr key={trg.name} className="hover:bg-slate-50 transition">
                          <td className="p-2.5 font-bold text-slate-900">{trg.name}</td>
                          <td className="p-2.5 text-amber-600">{trg.table}</td>
                          <td className="p-2.5">{trg.timing}</td>
                          <td className="p-2.5">{trg.event}</td>
                          <td className="p-2.5 font-sans text-center">
                            <button
                              onClick={() => handleDropTrigger(trg.name)}
                              className="inline-flex items-center gap-1 px-2.5 py-1 rounded-lg text-xs font-semibold bg-rose-50 text-rose-700 hover:bg-rose-100 border border-rose-200 transition shadow-2xs cursor-pointer"
                              title="Drop trigger"
                            >
                              <Trash2 className="w-3 h-3 text-rose-600" />
                              <span>Drop</span>
                            </button>
                          </td>
                        </tr>
                      ))
                    )}
                  </tbody>
                </table>
              </div>
            </div>
          )}

          {/* TAB 10: VIEWS */}
          {activeTab === 'views' && (
            <div className="bg-white border border-slate-200 rounded-2xl p-5 shadow-xs space-y-4">
              <div className="flex items-center justify-between pb-3 border-b border-slate-100">
                <div>
                  <h2 className="font-bold text-sm text-slate-900">Database Views</h2>
                  <p className="text-xs text-slate-500">Virtual tables defined by stored queries</p>
                </div>
              </div>

              <div className="overflow-x-auto border border-slate-200 rounded-xl shadow-2xs">
                <table className="w-full text-left text-xs border-collapse">
                  <thead>
                    <tr className="border-b border-slate-200 bg-slate-50 font-bold text-[11px] text-slate-700">
                      <th className="p-2.5">View Name</th>
                      <th className="p-2.5">Updatable</th>
                      <th className="p-2.5">Security</th>
                      <th className="p-2.5 text-center">Action</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-slate-100 font-mono text-xs">
                    {loadingEntities ? (
                      <tr>
                        <td colSpan={4} className="p-6 text-center text-slate-400 font-sans">
                          Loading views...
                        </td>
                      </tr>
                    ) : views.length === 0 ? (
                      <tr>
                        <td colSpan={4} className="p-6 text-center text-slate-400 font-sans">
                          No views found in `{currentDb}`.
                        </td>
                      </tr>
                    ) : (
                      views.map((v) => (
                        <tr key={v.name} className="hover:bg-slate-50 transition">
                          <td className="p-2.5 font-bold text-slate-900">{v.name}</td>
                          <td className="p-2.5">{v.is_updatable}</td>
                          <td className="p-2.5 text-slate-500">{v.security_type}</td>
                          <td className="p-2.5 font-sans text-center">
                            <div className="flex items-center justify-center gap-1.5">
                              <button
                                onClick={() => handleOpenTableBrowse(v.name)}
                                className="inline-flex items-center gap-1 px-2.5 py-1 rounded-lg text-xs font-semibold bg-emerald-50 text-emerald-700 hover:bg-emerald-100 border border-emerald-200 transition shadow-2xs cursor-pointer"
                                title="Browse view rows"
                              >
                                <Eye className="w-3 h-3 text-emerald-600" />
                                <span>Browse</span>
                              </button>
                              <button
                                onClick={async () => {
                                  if (!confirm(`DROP view \`${v.name}\`?`)) return;
                                  await handleExecuteSql(`DROP VIEW IF EXISTS \`${v.name}\`;`);
                                  fetchAuxEntities(currentDb);
                                  showToast(`View \`${v.name}\` dropped.`);
                                }}
                                className="inline-flex items-center gap-1 px-2.5 py-1 rounded-lg text-xs font-semibold bg-rose-50 text-rose-700 hover:bg-rose-100 border border-rose-200 transition shadow-2xs cursor-pointer"
                                title="Drop view"
                              >
                                <Trash2 className="w-3 h-3 text-rose-600" />
                                <span>Drop</span>
                              </button>
                            </div>
                          </td>
                        </tr>
                      ))
                    )}
                  </tbody>
                </table>
              </div>
            </div>
          )}

          {/* TAB 11: PRIVILEGES */}
          {activeTab === 'privileges' && (
            <div className="bg-white border border-slate-200 rounded-2xl p-5 shadow-xs space-y-4">
              <div className="flex items-center justify-between pb-3 border-b border-slate-100">
                <div>
                  <h2 className="font-bold text-sm text-slate-900">User Accounts &amp; Privileges</h2>
                  <p className="text-xs text-slate-500">MySQL user accounts and permissions</p>
                </div>
                <button
                  onClick={() => setAddUserModalOpen(true)}
                  className="flex items-center gap-1.5 px-3.5 py-1.5 rounded-xl bg-emerald-600 hover:bg-emerald-700 text-white font-bold text-xs shadow-xs cursor-pointer transition"
                >
                  <Plus className="w-3.5 h-3.5" />
                  <span>Add User</span>
                </button>
              </div>

              <div className="overflow-x-auto border border-slate-200 rounded-xl shadow-2xs">
                <table className="w-full text-left text-xs border-collapse">
                  <thead>
                    <tr className="border-b border-slate-200 bg-slate-50 font-bold text-[11px] text-slate-700">
                      <th className="p-2.5">User</th>
                      <th className="p-2.5">Host</th>
                      <th className="p-2.5">Type</th>
                      <th className="p-2.5">Privileges</th>
                      <th className="p-2.5">Grant</th>
                      <th className="p-2.5 text-center">Action</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-slate-100 font-mono text-xs">
                    {loadingPrivileges ? (
                      <tr>
                        <td colSpan={6} className="p-6 text-center text-slate-400 font-sans">
                          Loading user accounts...
                        </td>
                      </tr>
                    ) : privileges.length === 0 ? (
                      <tr>
                        <td colSpan={6} className="p-6 text-center text-slate-400 font-sans">
                          No users retrieved.
                        </td>
                      </tr>
                    ) : (
                      privileges.map((p, i) => (
                        <tr key={i} className="hover:bg-slate-50 transition">
                          <td className="p-2.5 font-bold text-slate-900">{p.user}</td>
                          <td className="p-2.5 text-amber-600">{p.host}</td>
                          <td className="p-2.5">{p.type}</td>
                          <td className="p-2.5 text-slate-500">{p.privileges}</td>
                          <td className="p-2.5">{p.grant ? 'Yes' : 'No'}</td>
                          <td className="p-2.5 font-sans text-center">
                            <button
                              onClick={async () => {
                                if (!confirm(`DROP user '${p.user}'@'${p.host}'?`)) return;
                                await handleExecuteSql(`DROP USER '${p.user}'@'${p.host}';`);
                                fetchPrivileges(currentDb);
                                showToast(`User dropped.`);
                              }}
                              className="inline-flex items-center gap-1 px-2.5 py-1 rounded-lg text-xs font-semibold bg-rose-50 text-rose-700 hover:bg-rose-100 border border-rose-200 transition shadow-2xs cursor-pointer"
                              title="Drop user"
                            >
                              <Trash2 className="w-3 h-3 text-rose-600" />
                              <span>Drop</span>
                            </button>
                          </td>
                        </tr>
                      ))
                    )}
                  </tbody>
                </table>
              </div>
            </div>
          )}
        </div>
      </div>

      {/* MODAL 1: CREATE DATABASE */}
      {createDbModalOpen && (
        <div className="fixed inset-0 z-50 bg-slate-900/40 backdrop-blur-xs flex items-center justify-center p-4 animate-fadeIn">
          <div className="w-full max-w-md bg-white border border-slate-200 rounded-2xl shadow-2xl p-6 text-xs space-y-4">
            <h3 className="text-base font-bold text-slate-900">Create New Database</h3>
            <p className="text-slate-500">Execute real `CREATE DATABASE` on MySQL engine</p>
            <form onSubmit={handleCreateDatabase} className="space-y-4">
              <div>
                <label className="block font-semibold mb-1 text-slate-700">Database Name</label>
                <input
                  type="text"
                  required
                  placeholder="new_db_name"
                  value={newDbName}
                  onChange={(e) => setNewDbName(e.target.value)}
                  className="w-full p-2.5 rounded-xl border bg-white border-slate-300 text-slate-900 font-mono focus:border-emerald-500 focus:ring-1 focus:ring-emerald-500"
                />
              </div>
              <div>
                <label className="block font-semibold mb-1 text-slate-700">Collation</label>
                <select
                  value={newDbCollation}
                  onChange={(e) => setNewDbCollation(e.target.value)}
                  className="w-full p-2.5 rounded-xl border bg-white border-slate-300 text-slate-900 font-mono focus:border-emerald-500 focus:ring-1 focus:ring-emerald-500"
                >
                  <option value="utf8mb4_unicode_ci">utf8mb4_unicode_ci</option>
                  <option value="utf8mb4_general_ci">utf8mb4_general_ci</option>
                  <option value="utf8_general_ci">utf8_general_ci</option>
                </select>
              </div>
              <div className="flex justify-end gap-2 pt-2">
                <button
                  type="button"
                  onClick={() => setCreateDbModalOpen(false)}
                  className="px-4 py-2 rounded-xl bg-slate-100 hover:bg-slate-200 text-slate-700 font-semibold border border-slate-200 cursor-pointer shadow-2xs transition"
                >
                  Cancel
                </button>
                <button type="submit" className="px-4 py-2 rounded-xl bg-emerald-600 hover:bg-emerald-700 text-white font-bold cursor-pointer shadow-xs transition">
                  Create Database
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* MODAL 2: CREATE TABLE */}
      {createTableModalOpen && (
        <div className="fixed inset-0 z-50 bg-slate-900/40 backdrop-blur-xs flex items-center justify-center p-4 animate-fadeIn">
          <div className="w-full max-w-md bg-white border border-slate-200 rounded-2xl shadow-2xl p-6 text-xs space-y-4">
            <h3 className="text-base font-bold text-slate-900">Create Table in `{currentDb}`</h3>
            <form onSubmit={handleCreateTable} className="space-y-4">
              <div>
                <label className="block font-semibold mb-1 text-slate-700">Table Name</label>
                <input
                  type="text"
                  required
                  placeholder="table_name"
                  value={newTableName}
                  onChange={(e) => setNewTableName(e.target.value)}
                  className="w-full p-2.5 rounded-xl border bg-white border-slate-300 text-slate-900 font-mono focus:border-emerald-500 focus:ring-1 focus:ring-emerald-500"
                />
              </div>
              <div className="flex justify-end gap-2 pt-2">
                <button
                  type="button"
                  onClick={() => setCreateTableModalOpen(false)}
                  className="px-4 py-2 rounded-xl bg-slate-100 hover:bg-slate-200 text-slate-700 font-semibold border border-slate-200 cursor-pointer shadow-2xs transition"
                >
                  Cancel
                </button>
                <button type="submit" className="px-4 py-2 rounded-xl bg-emerald-600 hover:bg-emerald-700 text-white font-bold cursor-pointer shadow-xs transition">
                  Create Table
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* MODAL 3: INSERT ROW */}
      {insertRowModalOpen && (
        <div className="fixed inset-0 z-50 bg-slate-900/40 backdrop-blur-xs flex items-center justify-center p-4 animate-fadeIn">
          <div className="w-full max-w-lg bg-white border border-slate-200 rounded-2xl shadow-2xl p-6 text-xs space-y-4 max-h-[90vh] overflow-y-auto">
            <h3 className="text-base font-bold text-slate-900">Insert Row into `{selectedTable}`</h3>
            <form onSubmit={handleSubmitInsertRow} className="space-y-3">
              {(browseData?.columns || []).map((col) => (
                <div key={col}>
                  <label className="block font-mono font-semibold text-slate-700 mb-1">
                    {col}
                  </label>
                  <input
                    type="text"
                    placeholder="Value or leave blank for NULL/DEFAULT"
                    value={insertRowValues[col] || ''}
                    onChange={(e) => setInsertRowValues({ ...insertRowValues, [col]: e.target.value })}
                    className="w-full p-2.5 rounded-xl border bg-white border-slate-300 text-slate-900 font-mono focus:border-emerald-500 focus:ring-1 focus:ring-emerald-500"
                  />
                </div>
              ))}
              <div className="flex justify-end gap-2 pt-2">
                <button
                  type="button"
                  onClick={() => setInsertRowModalOpen(false)}
                  className="px-4 py-2 rounded-xl bg-slate-100 hover:bg-slate-200 text-slate-700 font-semibold border border-slate-200 cursor-pointer shadow-2xs transition"
                >
                  Cancel
                </button>
                <button type="submit" className="px-4 py-2 rounded-xl bg-emerald-600 hover:bg-emerald-700 text-white font-bold cursor-pointer shadow-xs transition">
                  Save Row
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* MODAL 4: EDIT ROW */}
      {editRowModalOpen && editRowOriginal && (
        <div className="fixed inset-0 z-50 bg-slate-900/40 backdrop-blur-xs flex items-center justify-center p-4 animate-fadeIn">
          <div className="w-full max-w-lg bg-white border border-slate-200 rounded-2xl shadow-2xl p-6 text-xs space-y-4 max-h-[90vh] overflow-y-auto">
            <h3 className="text-base font-bold text-slate-900">Edit Row in `{selectedTable}`</h3>
            <form onSubmit={handleSubmitEditRow} className="space-y-3">
              {(browseData?.columns || []).map((col) => {
                const isPk = browseData?.primary_key === col;
                return (
                  <div key={col}>
                    <label className="block font-mono font-semibold text-slate-700 mb-1 flex items-center gap-1">
                      {isPk && <Key className="w-3 h-3 text-amber-500" />}
                      <span>{col}</span>
                      {isPk && <span className="text-[10px] text-amber-500 font-sans">(Primary Key)</span>}
                    </label>
                    <input
                      type="text"
                      disabled={isPk}
                      value={editRowValues[col] !== undefined ? editRowValues[col] : ''}
                      onChange={(e) => setEditRowValues({ ...editRowValues, [col]: e.target.value })}
                      className="w-full p-2.5 rounded-xl border bg-white border-slate-300 text-slate-900 font-mono disabled:opacity-60 focus:border-emerald-500 focus:ring-1 focus:ring-emerald-500"
                    />
                  </div>
                );
              })}
              <div className="flex justify-end gap-2 pt-2">
                <button
                  type="button"
                  onClick={() => setEditRowModalOpen(false)}
                  className="px-4 py-2 rounded-xl bg-slate-100 hover:bg-slate-200 text-slate-700 font-semibold border border-slate-200 cursor-pointer shadow-2xs transition"
                >
                  Cancel
                </button>
                <button type="submit" className="px-4 py-2 rounded-xl bg-emerald-600 hover:bg-emerald-700 text-white font-bold cursor-pointer shadow-xs transition">
                  Update Row
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* MODAL 5: ADD COLUMN */}
      {addColumnModalOpen && (
        <div className="fixed inset-0 z-50 bg-slate-900/40 backdrop-blur-xs flex items-center justify-center p-4 animate-fadeIn">
          <div className="w-full max-w-md bg-white border border-slate-200 rounded-2xl shadow-2xl p-6 text-xs space-y-4">
            <h3 className="text-base font-bold text-slate-900">Add Column to `{selectedTable}`</h3>
            <form onSubmit={handleAddColumnSubmit} className="space-y-3">
              <div>
                <label className="block font-semibold mb-1 text-slate-700">Column Name</label>
                <input
                  type="text"
                  required
                  placeholder="column_name"
                  value={newColName}
                  onChange={(e) => setNewColName(e.target.value)}
                  className="w-full p-2.5 rounded-xl border bg-white border-slate-300 text-slate-900 font-mono focus:border-emerald-500 focus:ring-1 focus:ring-emerald-500"
                />
              </div>
              <div>
                <label className="block font-semibold mb-1 text-slate-700">Type</label>
                <input
                  type="text"
                  required
                  placeholder="varchar(255), int(11), text..."
                  value={newColType}
                  onChange={(e) => setNewColType(e.target.value)}
                  className="w-full p-2.5 rounded-xl border bg-white border-slate-300 text-slate-900 font-mono focus:border-emerald-500 focus:ring-1 focus:ring-emerald-500"
                />
              </div>
              <div className="grid grid-cols-2 gap-2">
                <div>
                  <label className="block font-semibold mb-1 text-slate-700">Null</label>
                  <select
                    value={newColNull}
                    onChange={(e) => setNewColNull(e.target.value)}
                    className="w-full p-2.5 rounded-xl border bg-white border-slate-300 text-slate-900 font-mono focus:border-emerald-500 focus:ring-1 focus:ring-emerald-500"
                  >
                    <option value="YES">NULL</option>
                    <option value="NO">NOT NULL</option>
                  </select>
                </div>
                <div>
                  <label className="block font-semibold mb-1 text-slate-700">Default</label>
                  <input
                    type="text"
                    placeholder="NULL or default value"
                    value={newColDefault}
                    onChange={(e) => setNewColDefault(e.target.value)}
                    className="w-full p-2.5 rounded-xl border bg-white border-slate-300 text-slate-900 font-mono focus:border-emerald-500 focus:ring-1 focus:ring-emerald-500"
                  />
                </div>
              </div>
              <div>
                <label className="block font-semibold mb-1 text-slate-700">After Column</label>
                <select
                  value={newColAfter}
                  onChange={(e) => setNewColAfter(e.target.value)}
                  className="w-full p-2.5 rounded-xl border bg-white border-slate-300 text-slate-900 font-mono focus:border-emerald-500 focus:ring-1 focus:ring-emerald-500"
                >
                  <option value="">At End of Table</option>
                  {(tableStructure?.columns || []).map((c) => (
                    <option key={c.field} value={c.field}>After {c.field}</option>
                  ))}
                </select>
              </div>
              <div className="flex justify-end gap-2 pt-2">
                <button
                  type="button"
                  onClick={() => setAddColumnModalOpen(false)}
                  className="px-4 py-2 rounded-xl bg-slate-100 hover:bg-slate-200 text-slate-700 font-semibold border border-slate-200 cursor-pointer shadow-2xs transition"
                >
                  Cancel
                </button>
                <button type="submit" className="px-4 py-2 rounded-xl bg-emerald-600 hover:bg-emerald-700 text-white font-bold cursor-pointer shadow-xs transition">
                  Add Column
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* MODAL 6: MODIFY COLUMN */}
      {modifyColModalOpen && modColTarget && (
        <div className="fixed inset-0 z-50 bg-slate-900/40 backdrop-blur-xs flex items-center justify-center p-4 animate-fadeIn">
          <div className="w-full max-w-md bg-white border border-slate-200 rounded-2xl shadow-2xl p-6 text-xs space-y-4">
            <h3 className="text-base font-bold text-slate-900">Modify Column: `{modColTarget.field}`</h3>
            <form onSubmit={handleModifyColumnSubmit} className="space-y-3">
              <div>
                <label className="block font-semibold mb-1 text-slate-700">Column Name</label>
                <input
                  type="text"
                  required
                  value={modColName}
                  onChange={(e) => setModColName(e.target.value)}
                  className="w-full p-2.5 rounded-xl border bg-white border-slate-300 text-slate-900 font-mono focus:border-emerald-500 focus:ring-1 focus:ring-emerald-500"
                />
              </div>
              <div>
                <label className="block font-semibold mb-1 text-slate-700">Type</label>
                <input
                  type="text"
                  required
                  value={modColType}
                  onChange={(e) => setModColType(e.target.value)}
                  className="w-full p-2.5 rounded-xl border bg-white border-slate-300 text-slate-900 font-mono focus:border-emerald-500 focus:ring-1 focus:ring-emerald-500"
                />
              </div>
              <div className="grid grid-cols-2 gap-2">
                <div>
                  <label className="block font-semibold mb-1 text-slate-700">Null</label>
                  <select
                    value={modColNull}
                    onChange={(e) => setModColNull(e.target.value)}
                    className="w-full p-2.5 rounded-xl border bg-white border-slate-300 text-slate-900 font-mono focus:border-emerald-500 focus:ring-1 focus:ring-emerald-500"
                  >
                    <option value="YES">NULL</option>
                    <option value="NO">NOT NULL</option>
                  </select>
                </div>
                <div>
                  <label className="block font-semibold mb-1 text-slate-700">Default</label>
                  <input
                    type="text"
                    value={modColDefault}
                    onChange={(e) => setModColDefault(e.target.value)}
                    className="w-full p-2.5 rounded-xl border bg-white border-slate-300 text-slate-900 font-mono focus:border-emerald-500 focus:ring-1 focus:ring-emerald-500"
                  />
                </div>
              </div>
              <div className="flex justify-end gap-2 pt-2">
                <button
                  type="button"
                  onClick={() => setModifyColModalOpen(false)}
                  className="px-4 py-2 rounded-xl bg-slate-100 hover:bg-slate-200 text-slate-700 font-semibold border border-slate-200 cursor-pointer shadow-2xs transition"
                >
                  Cancel
                </button>
                <button type="submit" className="px-4 py-2 rounded-xl bg-emerald-600 hover:bg-emerald-700 text-white font-bold cursor-pointer shadow-xs transition">
                  Save Changes
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* MODAL 7: ADD INDEX */}
      {addIndexModalOpen && (
        <div className="fixed inset-0 z-50 bg-slate-900/40 backdrop-blur-xs flex items-center justify-center p-4 animate-fadeIn">
          <div className="w-full max-w-md bg-white border border-slate-200 rounded-2xl shadow-2xl p-6 text-xs space-y-4">
            <h3 className="text-base font-bold text-slate-900">Add Index to `{selectedTable}`</h3>
            <form onSubmit={handleAddIndexSubmit} className="space-y-3">
              <div>
                <label className="block font-semibold mb-1 text-slate-700">Index Name</label>
                <input
                  type="text"
                  placeholder="idx_column_name"
                  value={newIndexName}
                  onChange={(e) => setNewIndexName(e.target.value)}
                  className="w-full p-2.5 rounded-xl border bg-white border-slate-300 text-slate-900 font-mono focus:border-emerald-500 focus:ring-1 focus:ring-emerald-500"
                />
              </div>
              <div>
                <label className="block font-semibold mb-1 text-slate-700">Index Type</label>
                <select
                  value={newIndexType}
                  onChange={(e: any) => setNewIndexType(e.target.value)}
                  className="w-full p-2.5 rounded-xl border bg-white border-slate-300 text-slate-900 font-mono focus:border-emerald-500 focus:ring-1 focus:ring-emerald-500"
                >
                  <option value="INDEX">INDEX</option>
                  <option value="UNIQUE">UNIQUE</option>
                  <option value="PRIMARY">PRIMARY</option>
                  <option value="FULLTEXT">FULLTEXT</option>
                </select>
              </div>
              <div>
                <label className="block font-semibold mb-1 text-slate-700">Columns</label>
                <div className="max-h-36 overflow-y-auto border border-slate-200 rounded-xl p-2.5 space-y-1 bg-slate-50 font-mono">
                  {(tableStructure?.columns || []).map((c) => (
                    <label key={c.field} className="flex items-center gap-2 cursor-pointer text-slate-800 hover:text-emerald-700">
                      <input
                        type="checkbox"
                        checked={newIndexColumns.includes(c.field)}
                        onChange={(e) => {
                          if (e.target.checked) setNewIndexColumns((prev) => [...prev, c.field]);
                          else setNewIndexColumns((prev) => prev.filter((col) => col !== c.field));
                        }}
                        className="rounded text-amber-500 focus:ring-amber-500"
                      />
                      <span>{c.field}</span>
                    </label>
                  ))}
                </div>
              </div>
              <div className="flex justify-end gap-2 pt-2">
                <button
                  type="button"
                  onClick={() => setAddIndexModalOpen(false)}
                  className="px-4 py-2 rounded-xl bg-slate-100 hover:bg-slate-200 text-slate-700 font-semibold border border-slate-200 cursor-pointer shadow-2xs transition"
                >
                  Cancel
                </button>
                <button type="submit" className="px-4 py-2 rounded-xl bg-emerald-600 hover:bg-emerald-700 text-white font-bold cursor-pointer shadow-xs transition">
                  Create Index
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* MODAL 8: ADD ROUTINE */}
      {addRoutineModalOpen && (
        <div className="fixed inset-0 z-50 bg-slate-900/40 backdrop-blur-xs flex items-center justify-center p-4 animate-fadeIn">
          <div className="w-full max-w-lg bg-white border border-slate-200 rounded-2xl shadow-2xl p-6 text-xs space-y-4">
            <h3 className="text-base font-bold text-slate-900">Add Stored Routine</h3>
            <form onSubmit={handleCreateRoutine} className="space-y-3">
              <div>
                <label className="block font-semibold mb-1 text-slate-700">Routine Name</label>
                <input
                  type="text"
                  required
                  placeholder="routine_name"
                  value={routineName}
                  onChange={(e) => setRoutineName(e.target.value)}
                  className="w-full p-2.5 rounded-xl border bg-white border-slate-300 text-slate-900 font-mono focus:border-emerald-500 focus:ring-1 focus:ring-emerald-500"
                />
              </div>
              <div className="grid grid-cols-2 gap-2">
                <div>
                  <label className="block font-semibold mb-1 text-slate-700">Type</label>
                  <select
                    value={routineType}
                    onChange={(e: any) => setRoutineType(e.target.value)}
                    className="w-full p-2.5 rounded-xl border bg-white border-slate-300 text-slate-900 font-mono focus:border-emerald-500 focus:ring-1 focus:ring-emerald-500"
                  >
                    <option value="PROCEDURE">PROCEDURE</option>
                    <option value="FUNCTION">FUNCTION</option>
                  </select>
                </div>
                {routineType === 'FUNCTION' && (
                  <div>
                    <label className="block font-semibold mb-1 text-slate-700">Return Type</label>
                    <input
                      type="text"
                      placeholder="INT, VARCHAR(255)..."
                      value={routineReturns}
                      onChange={(e) => setRoutineReturns(e.target.value)}
                      className="w-full p-2.5 rounded-xl border bg-white border-slate-300 text-slate-900 font-mono focus:border-emerald-500 focus:ring-1 focus:ring-emerald-500"
                    />
                  </div>
                )}
              </div>
              <div>
                <label className="block font-semibold mb-1 text-slate-700">Parameters (e.g. IN p_id INT, OUT p_cnt INT)</label>
                <input
                  type="text"
                  placeholder="param list"
                  value={routineParams}
                  onChange={(e) => setRoutineParams(e.target.value)}
                  className="w-full p-2.5 rounded-xl border bg-white border-slate-300 text-slate-900 font-mono focus:border-emerald-500 focus:ring-1 focus:ring-emerald-500"
                />
              </div>
              <div>
                <label className="block font-semibold mb-1 text-slate-700">Routine Body (SQL)</label>
                <textarea
                  rows={6}
                  value={routineBody}
                  onChange={(e) => setRoutineBody(e.target.value)}
                  className="w-full p-2.5 rounded-xl border bg-white border-slate-300 text-slate-900 font-mono text-xs focus:border-emerald-500 focus:ring-1 focus:ring-emerald-500"
                />
              </div>
              <div className="flex justify-end gap-2 pt-2">
                <button
                  type="button"
                  onClick={() => setAddRoutineModalOpen(false)}
                  className="px-4 py-2 rounded-xl bg-slate-100 hover:bg-slate-200 text-slate-700 font-semibold border border-slate-200 cursor-pointer shadow-2xs transition"
                >
                  Cancel
                </button>
                <button type="submit" className="px-4 py-2 rounded-xl bg-emerald-600 hover:bg-emerald-700 text-white font-bold cursor-pointer shadow-xs transition">
                  Save Routine
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* MODAL 9: ADD EVENT */}
      {addEventModalOpen && (
        <div className="fixed inset-0 z-50 bg-slate-900/40 backdrop-blur-xs flex items-center justify-center p-4 animate-fadeIn">
          <div className="w-full max-w-md bg-white border border-slate-200 rounded-2xl shadow-2xl p-6 text-xs space-y-4">
            <h3 className="text-base font-bold text-slate-900">Create Scheduled Event</h3>
            <form onSubmit={handleCreateEvent} className="space-y-3">
              <div>
                <label className="block font-semibold mb-1 text-slate-700">Event Name</label>
                <input
                  type="text"
                  required
                  placeholder="event_name"
                  value={eventName}
                  onChange={(e) => setEventName(e.target.value)}
                  className="w-full p-2.5 rounded-xl border bg-white border-slate-300 text-slate-900 font-mono focus:border-emerald-500 focus:ring-1 focus:ring-emerald-500"
                />
              </div>
              <div>
                <label className="block font-semibold mb-1 text-slate-700">Schedule Expression</label>
                <input
                  type="text"
                  required
                  placeholder="EVERY 1 DAY"
                  value={eventSchedule}
                  onChange={(e) => setEventSchedule(e.target.value)}
                  className="w-full p-2.5 rounded-xl border bg-white border-slate-300 text-slate-900 font-mono focus:border-emerald-500 focus:ring-1 focus:ring-emerald-500"
                />
              </div>
              <div>
                <label className="block font-semibold mb-1 text-slate-700">SQL Body</label>
                <textarea
                  rows={4}
                  value={eventBody}
                  onChange={(e) => setEventBody(e.target.value)}
                  className="w-full p-2.5 rounded-xl border bg-white border-slate-300 text-slate-900 font-mono text-xs focus:border-emerald-500 focus:ring-1 focus:ring-emerald-500"
                />
              </div>
              <div className="flex justify-end gap-2 pt-2">
                <button
                  type="button"
                  onClick={() => setAddEventModalOpen(false)}
                  className="px-4 py-2 rounded-xl bg-slate-100 hover:bg-slate-200 text-slate-700 font-semibold border border-slate-200 cursor-pointer shadow-2xs transition"
                >
                  Cancel
                </button>
                <button type="submit" className="px-4 py-2 rounded-xl bg-emerald-600 hover:bg-emerald-700 text-white font-bold cursor-pointer shadow-xs transition">
                  Create Event
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* MODAL 10: ADD TRIGGER */}
      {addTriggerModalOpen && (
        <div className="fixed inset-0 z-50 bg-slate-900/40 backdrop-blur-xs flex items-center justify-center p-4 animate-fadeIn">
          <div className="w-full max-w-md bg-white border border-slate-200 rounded-2xl shadow-2xl p-6 text-xs space-y-4">
            <h3 className="text-base font-bold text-slate-900">Create Trigger</h3>
            <form onSubmit={handleCreateTrigger} className="space-y-3">
              <div>
                <label className="block font-semibold mb-1 text-slate-700">Trigger Name</label>
                <input
                  type="text"
                  required
                  placeholder="trigger_name"
                  value={triggerName}
                  onChange={(e) => setTriggerName(e.target.value)}
                  className="w-full p-2.5 rounded-xl border bg-white border-slate-300 text-slate-900 font-mono focus:border-emerald-500 focus:ring-1 focus:ring-emerald-500"
                />
              </div>
              <div>
                <label className="block font-semibold mb-1 text-slate-700">Table</label>
                <select
                  value={triggerTable}
                  onChange={(e) => setTriggerTable(e.target.value)}
                  className="w-full p-2.5 rounded-xl border bg-white border-slate-300 text-slate-900 font-mono focus:border-emerald-500 focus:ring-1 focus:ring-emerald-500"
                >
                  {tableDetails.map((t) => (
                    <option key={t.name} value={t.name}>
                      {t.name}
                    </option>
                  ))}
                </select>
              </div>
              <div className="grid grid-cols-2 gap-2">
                <div>
                  <label className="block font-semibold mb-1 text-slate-700">Timing</label>
                  <select
                    value={triggerTiming}
                    onChange={(e: any) => setTriggerTiming(e.target.value)}
                    className="w-full p-2.5 rounded-xl border bg-white border-slate-300 text-slate-900 font-mono focus:border-emerald-500 focus:ring-1 focus:ring-emerald-500"
                  >
                    <option value="BEFORE">BEFORE</option>
                    <option value="AFTER">AFTER</option>
                  </select>
                </div>
                <div>
                  <label className="block font-semibold mb-1 text-slate-700">Event</label>
                  <select
                    value={triggerEvent}
                    onChange={(e: any) => setTriggerEvent(e.target.value)}
                    className="w-full p-2.5 rounded-xl border bg-white border-slate-300 text-slate-900 font-mono focus:border-emerald-500 focus:ring-1 focus:ring-emerald-500"
                  >
                    <option value="INSERT">INSERT</option>
                    <option value="UPDATE">UPDATE</option>
                    <option value="DELETE">DELETE</option>
                  </select>
                </div>
              </div>
              <div>
                <label className="block font-semibold mb-1 text-slate-700">Trigger SQL Body</label>
                <textarea
                  rows={4}
                  value={triggerBody}
                  onChange={(e) => setTriggerBody(e.target.value)}
                  className="w-full p-2.5 rounded-xl border bg-white border-slate-300 text-slate-900 font-mono text-xs focus:border-emerald-500 focus:ring-1 focus:ring-emerald-500"
                />
              </div>
              <div className="flex justify-end gap-2 pt-2">
                <button
                  type="button"
                  onClick={() => setAddTriggerModalOpen(false)}
                  className="px-4 py-2 rounded-xl bg-slate-100 hover:bg-slate-200 text-slate-700 font-semibold border border-slate-200 cursor-pointer shadow-2xs transition"
                >
                  Cancel
                </button>
                <button type="submit" className="px-4 py-2 rounded-xl bg-emerald-600 hover:bg-emerald-700 text-white font-bold cursor-pointer shadow-xs transition">
                  Create Trigger
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* MODAL 11: ADD USER ACCOUNT */}
      {addUserModalOpen && (
        <div className="fixed inset-0 z-50 bg-slate-900/40 backdrop-blur-xs flex items-center justify-center p-4 animate-fadeIn">
          <div className="w-full max-w-md bg-white border border-slate-200 rounded-2xl shadow-2xl p-6 text-xs space-y-4">
            <h3 className="text-base font-bold text-slate-900">Add Database User</h3>
            <form
              onSubmit={async (e) => {
                e.preventDefault();
                if (!newUsername.trim() || !newUserPassword.trim()) return;
                try {
                  const q = `CREATE USER '${newUsername.trim()}'@'${newUserHost.trim()}' IDENTIFIED BY '${newUserPassword}'; GRANT ALL PRIVILEGES ON \`${currentDb}\`.* TO '${newUsername.trim()}'@'${newUserHost.trim()}'; FLUSH PRIVILEGES;`;
                  const res = await apiFetch<any>('/api/v1/databases/query', {
                    method: 'POST',
                    body: JSON.stringify({ database: 'mysql', query: q }),
                  });
                  if (res && res.data && !res.data.error) {
                    showToast(`User '${newUsername}' created with privileges on ${currentDb}.`);
                    setAddUserModalOpen(false);
                    setNewUsername('');
                    setNewUserPassword('');
                    fetchPrivileges(currentDb);
                  } else {
                    showToast(res?.data?.error || 'Failed to create database user');
                  }
                } catch (err: any) {
                  showToast(err?.message || 'Create user error');
                }
              }}
              className="space-y-3"
            >
              <div>
                <label className="block font-semibold mb-1 text-slate-700">Username</label>
                <input
                  type="text"
                  required
                  placeholder="username"
                  value={newUsername}
                  onChange={(e) => setNewUsername(e.target.value)}
                  className="w-full p-2.5 rounded-xl border bg-white border-slate-300 text-slate-900 font-mono focus:border-emerald-500 focus:ring-1 focus:ring-emerald-500"
                />
              </div>
              <div>
                <label className="block font-semibold mb-1 text-slate-700">Host</label>
                <input
                  type="text"
                  required
                  value={newUserHost}
                  onChange={(e) => setNewUserHost(e.target.value)}
                  className="w-full p-2.5 rounded-xl border bg-white border-slate-300 text-slate-900 font-mono focus:border-emerald-500 focus:ring-1 focus:ring-emerald-500"
                />
              </div>
              <div>
                <label className="block font-semibold mb-1 text-slate-700">Password</label>
                <input
                  type="password"
                  required
                  placeholder="••••••••"
                  value={newUserPassword}
                  onChange={(e) => setNewUserPassword(e.target.value)}
                  className="w-full p-2.5 rounded-xl border bg-white border-slate-300 text-slate-900 font-mono focus:border-emerald-500 focus:ring-1 focus:ring-emerald-500"
                />
              </div>
              <div className="flex justify-end gap-2 pt-2">
                <button
                  type="button"
                  onClick={() => setAddUserModalOpen(false)}
                  className="px-4 py-2 rounded-xl bg-slate-100 hover:bg-slate-200 text-slate-700 font-semibold border border-slate-200 cursor-pointer shadow-2xs transition"
                >
                  Cancel
                </button>
                <button type="submit" className="px-4 py-2 rounded-xl bg-emerald-600 hover:bg-emerald-700 text-white font-bold cursor-pointer shadow-xs transition">
                  Create User
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </DashboardShell>
  );
}

export default function PhpMyAdminPage() {
  return (
    <Suspense fallback={<div className="p-8 text-center text-slate-500">Loading phpMyAdmin Database Manager...</div>}>
      <PhpMyAdminCore />
    </Suspense>
  );
}
