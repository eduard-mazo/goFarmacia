<script setup lang="ts">
import { ref, computed, onMounted, watch, nextTick } from "vue";
import { toast } from "vue-sonner";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import { Tabs, TabsList, TabsTrigger, TabsContent } from "@/components/ui/tabs";
import {
  Table,
  TableHeader,
  TableBody,
  TableRow,
  TableHead,
  TableCell,
} from "@/components/ui/table";
import {
  AlertDialog,
  AlertDialogContent,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogCancel,
  AlertDialogAction,
} from "@/components/ui/alert-dialog";
import {
  DropdownMenu,
  DropdownMenuTrigger,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
} from "@/components/ui/dropdown-menu";
import {
  Database,
  Table2,
  RefreshCcw,
  HardDriveDownload,
  FileUp,
  Download,
  FileCode,
  MoreHorizontal,
  Key,
  Link2,
  CheckCircle2,
  Minus,
  ChevronLeft,
  ChevronRight,
  Loader2,
  Search,
  Zap,
  Wrench,
  Trash2,
  Upload,
  Columns,
  LayoutList,
  ArrowUpDown,
  ArrowUp,
  ArrowDown,
  Lock,
  Pencil,
  Check,
  X,
  MoreVertical,
} from "lucide-vue-next";
import {
  GetTablas,
  GetEsquemaTabla,
  GetDatosTabla,
  ExportarTablaCSV,
  ExportarTablaSQL,
  ExportarBDSQL,
  ImportarTablaCSV,
  ImportarSQL,
  AnalizarTabla,
  VacuumTabla,
  TruncarTabla,
  ActualizarFilaTabla,
  EliminarFilaTabla,
} from "@/../wailsjs/go/backend/Db";

// ── Types ─────────────────────────────────────────────────────────────────────
interface TableInfo {
  Name: string;
  Schema: string;
  RowCount: number;
  SizeBytes: number;
  SizeHuman: string;
}
interface ColumnInfo {
  Position: number;
  Name: string;
  DataType: string;
  IsNullable: boolean;
  Default: string;
  IsPrimaryKey: boolean;
  IsForeignKey: boolean;
  MaxLength: number;
}
interface IndexInfo {
  Name: string;
  IsUnique: boolean;
  IsPrimary: boolean;
  Definition: string;
}
interface ConstraintInfo {
  Name: string;
  Type: string;
  Definition: string;
}
interface TableSchema {
  TableInfo: TableInfo;
  Columns: ColumnInfo[];
  Indexes: IndexInfo[];
  Constraints: ConstraintInfo[];
}
interface TablePreview {
  Columns: string[];
  Rows: Record<string, string>[];
  Total: number;
  Limit: number;
  Offset: number;
}
interface OperationResult {
  Success: boolean;
  Message: string;
}

// ── State ─────────────────────────────────────────────────────────────────────
const tables = ref<TableInfo[]>([]);
const selectedTable = ref<string | null>(null);
const schema = ref<TableSchema | null>(null);
const preview = ref<TablePreview | null>(null);
const activeTab = ref<string>("schema");
const tableSearch = ref("");
const dataPage = ref(0);
const PAGE_SIZE = 50;

const isLoadingTables = ref(false);
const isLoadingSchema = ref(false);
const isLoadingData = ref(false);
const busyOp = ref<string | null>(null);

// Sort state
const sortCol = ref<string>("");
const sortDir = ref<"asc" | "desc">("asc");

// Data tab filter
const dataFilter = ref("");

// Inline edit state
const editingCell = ref<{ rowIdx: number; col: string } | null>(null);
const editValue = ref<string>("");
const editInputRef = ref<HTMLInputElement | null>(null);
const savingCell = ref<string>(""); // "rowIdx_col" key while saving
const flashCell = ref<{ key: string; type: "ok" | "err" } | null>(null);

// Delete row dialog
const deleteRowDialog = ref<{ pkValue: string } | null>(null);
const deletingRow = ref(false);

// Truncate dialog
const showTruncateDialog = ref(false);

// ── Computed ──────────────────────────────────────────────────────────────────
const filteredTables = computed(() =>
  tables.value.filter((t) =>
    t.Name.toLowerCase().includes(tableSearch.value.toLowerCase())
  )
);

const totalSize = computed(() => {
  const b = tables.value.reduce((a, t) => a + t.SizeBytes, 0);
  if (b >= 1073741824) return `${(b / 1073741824).toFixed(1)} GB`;
  if (b >= 1048576) return `${(b / 1048576).toFixed(1)} MB`;
  if (b >= 1024) return `${(b / 1024).toFixed(1)} KB`;
  return `${b} B`;
});

const totalPages = computed(() =>
  preview.value ? Math.max(1, Math.ceil(preview.value.Total / PAGE_SIZE)) : 1
);

const showingRange = computed(() => {
  if (!preview.value) return "";
  const from = dataPage.value * PAGE_SIZE + 1;
  const to = Math.min((dataPage.value + 1) * PAGE_SIZE, preview.value.Total);
  return `${from}–${to} de ${preview.value.Total.toLocaleString()}`;
});

// The first PK column (for edit/delete operations)
const pkColumn = computed(
  () => schema.value?.Columns.find((c) => c.IsPrimaryKey)?.Name ?? null
);

const hasPK = computed(() => pkColumn.value !== null);

// Map column name → ColumnInfo for quick lookup
const columnMap = computed<Record<string, ColumnInfo>>(() => {
  const m: Record<string, ColumnInfo> = {};
  schema.value?.Columns.forEach((c) => (m[c.Name] = c));
  return m;
});

// Filtered preview rows (client-side filter on current page)
const filteredRows = computed(() => {
  if (!preview.value || !dataFilter.value.trim()) return preview.value?.Rows ?? [];
  const q = dataFilter.value.toLowerCase();
  return preview.value.Rows.filter((row) =>
    Object.values(row).some((v) => v.toLowerCase().includes(q))
  );
});

// ── Init ──────────────────────────────────────────────────────────────────────
onMounted(async () => {
  await loadTables(true);
});

// ── Data loading ──────────────────────────────────────────────────────────────
async function loadTables(selectFirst = false) {
  isLoadingTables.value = true;
  try {
    tables.value = (await GetTablas()) as TableInfo[];
    if (selectFirst && tables.value.length > 0 && !selectedTable.value) {
      await selectTable(tables.value[0].Name);
    }
  } catch (e) {
    toast.error("Error al cargar tablas", { description: String(e) });
  } finally {
    isLoadingTables.value = false;
  }
}

async function selectTable(name: string) {
  cancelEdit();
  selectedTable.value = name;
  schema.value = null;
  preview.value = null;
  dataPage.value = 0;
  sortCol.value = "";
  sortDir.value = "asc";
  dataFilter.value = "";
  activeTab.value = "schema";
  await loadSchema(name);
}

async function loadSchema(name: string) {
  isLoadingSchema.value = true;
  try {
    schema.value = (await GetEsquemaTabla(name)) as TableSchema;
  } catch (e) {
    toast.error("Error al cargar esquema", { description: String(e) });
  } finally {
    isLoadingSchema.value = false;
  }
}

async function loadData(name: string, page: number) {
  isLoadingData.value = true;
  cancelEdit();
  try {
    preview.value = (await GetDatosTabla(
      name,
      PAGE_SIZE,
      page * PAGE_SIZE,
      sortCol.value,
      sortDir.value
    )) as TablePreview;
  } catch (e) {
    toast.error("Error al cargar datos", { description: String(e) });
  } finally {
    isLoadingData.value = false;
  }
}

// Lazy-load data when switching to data tab
watch(activeTab, async (tab) => {
  if (tab === "data" && selectedTable.value && !preview.value) {
    await loadData(selectedTable.value, 0);
  }
});

async function prevPage() {
  if (dataPage.value > 0 && selectedTable.value) {
    dataPage.value--;
    await loadData(selectedTable.value, dataPage.value);
  }
}

async function nextPage() {
  if (dataPage.value + 1 < totalPages.value && selectedTable.value) {
    dataPage.value++;
    await loadData(selectedTable.value, dataPage.value);
  }
}

// ── Sorting ────────────────────────────────────────────────────────────────────
async function toggleSort(col: string) {
  if (sortCol.value === col) {
    sortDir.value = sortDir.value === "asc" ? "desc" : "asc";
  } else {
    sortCol.value = col;
    sortDir.value = "asc";
  }
  dataPage.value = 0;
  if (selectedTable.value) await loadData(selectedTable.value, 0);
}

// ── Inline editing ────────────────────────────────────────────────────────────
function isEditable(col: string): boolean {
  const info = columnMap.value[col];
  return !!info && !info.IsPrimaryKey;
}

function isBool(col: string): boolean {
  const dt = columnMap.value[col]?.DataType?.toLowerCase() ?? "";
  return dt === "boolean" || dt === "bool";
}

function cellKey(rowIdx: number, col: string) {
  return `${rowIdx}_${col}`;
}

function isEditingThis(rowIdx: number, col: string) {
  return editingCell.value?.rowIdx === rowIdx && editingCell.value?.col === col;
}

function isSavingThis(rowIdx: number, col: string) {
  return savingCell.value === cellKey(rowIdx, col);
}

function isFlashingThis(rowIdx: number, col: string) {
  return flashCell.value?.key === cellKey(rowIdx, col);
}

function startEdit(rowIdx: number, col: string, currentVal: string) {
  if (!isEditable(col)) return;
  cancelEdit();

  if (isBool(col)) {
    // Toggle boolean directly without entering edit mode
    const newVal = currentVal === "true" ? "false" : "true";
    commitBoolEdit(rowIdx, col, newVal);
    return;
  }

  editingCell.value = { rowIdx, col };
  editValue.value = currentVal === "NULL" ? "" : currentVal;

  nextTick(() => {
    editInputRef.value?.focus();
    editInputRef.value?.select();
  });
}

function cancelEdit() {
  editingCell.value = null;
  editValue.value = "";
}

async function commitEdit() {
  if (!editingCell.value) return;
  const { rowIdx, col } = editingCell.value;
  await saveCell(rowIdx, col, editValue.value);
  editingCell.value = null;
}

async function commitBoolEdit(rowIdx: number, col: string, newVal: string) {
  await saveCell(rowIdx, col, newVal);
}

async function saveCell(rowIdx: number, col: string, value: string) {
  if (!selectedTable.value || !pkColumn.value) return;

  const row = preview.value!.Rows[rowIdx];
  const pkVal = row[pkColumn.value];
  const isNullable = columnMap.value[col]?.IsNullable ?? false;
  const isNull = isNullable && value.trim() === "";

  const key = cellKey(rowIdx, col);
  savingCell.value = key;

  try {
    const result = (await ActualizarFilaTabla(
      selectedTable.value,
      pkColumn.value,
      pkVal,
      col,
      value,
      isNull
    )) as OperationResult;

    if (result.Success) {
      // Update local data
      row[col] = isNull ? "NULL" : value;
      // Flash green
      flashCell.value = { key, type: "ok" };
      setTimeout(() => {
        if (flashCell.value?.key === key) flashCell.value = null;
      }, 800);
    } else {
      toast.error("Error al actualizar", { description: result.Message });
      flashCell.value = { key, type: "err" };
      setTimeout(() => {
        if (flashCell.value?.key === key) flashCell.value = null;
      }, 1000);
    }
  } catch (e) {
    toast.error("Error al actualizar", { description: String(e) });
    flashCell.value = { key, type: "err" };
    setTimeout(() => {
      if (flashCell.value?.key === key) flashCell.value = null;
    }, 1000);
  } finally {
    savingCell.value = "";
  }
}

// Tab to next editable cell
function handleEditKeydown(e: KeyboardEvent, rowIdx: number, col: string) {
  if (e.key === "Enter") {
    e.preventDefault();
    commitEdit();
  } else if (e.key === "Escape") {
    e.preventDefault();
    cancelEdit();
  } else if (e.key === "Tab") {
    e.preventDefault();
    commitEdit();
    // Move to next column
    const cols = preview.value?.Columns ?? [];
    const cur = cols.indexOf(col);
    const next = cols.slice(cur + 1).find((c) => isEditable(c));
    if (next) {
      nextTick(() => startEdit(rowIdx, next, preview.value!.Rows[rowIdx][next]));
    }
  }
}

// ── Delete row ────────────────────────────────────────────────────────────────
function askDeleteRow(pkValue: string) {
  deleteRowDialog.value = { pkValue };
}

async function confirmDeleteRow() {
  if (!deleteRowDialog.value || !selectedTable.value || !pkColumn.value) return;
  const { pkValue } = deleteRowDialog.value;
  deletingRow.value = true;
  try {
    const result = (await EliminarFilaTabla(
      selectedTable.value,
      pkColumn.value,
      pkValue
    )) as OperationResult;
    if (result.Success) {
      toast.success("Fila eliminada");
      // Remove from local array
      if (preview.value) {
        const idx = preview.value.Rows.findIndex(
          (r) => r[pkColumn.value!] === pkValue
        );
        if (idx !== -1) {
          preview.value.Rows.splice(idx, 1);
          preview.value.Total--;
        }
      }
    } else {
      toast.error("Error al eliminar", { description: result.Message });
    }
  } catch (e) {
    toast.error("Error al eliminar", { description: String(e) });
  } finally {
    deletingRow.value = false;
    deleteRowDialog.value = null;
  }
}

// ── Operations ────────────────────────────────────────────────────────────────
async function runOp(label: string, fn: () => Promise<OperationResult>, afterFn?: () => Promise<void>) {
  busyOp.value = label;
  try {
    const res = await fn();
    if (res.Message) {
      res.Success
        ? toast.success(label, { description: res.Message })
        : toast.info(label, { description: res.Message });
    }
    if (res.Success && afterFn) await afterFn();
  } catch (e) {
    toast.error(`Error: ${label}`, { description: String(e) });
  } finally {
    busyOp.value = null;
  }
}

function exportCSV() {
  const t = selectedTable.value!;
  runOp("Exportar CSV", () => ExportarTablaCSV(t) as Promise<OperationResult>);
}
function exportSQL() {
  const t = selectedTable.value!;
  runOp("Exportar SQL", () => ExportarTablaSQL(t) as Promise<OperationResult>);
}
function doBackup() {
  runOp("Backup BD", () => ExportarBDSQL() as Promise<OperationResult>);
}
function importCSV() {
  const t = selectedTable.value!;
  runOp("Importar CSV", () => ImportarTablaCSV(t) as Promise<OperationResult>, async () => {
    if (selectedTable.value) await loadData(selectedTable.value, dataPage.value);
  });
}
function doImportSQL() {
  runOp("Importar SQL", () => ImportarSQL() as Promise<OperationResult>, async () => {
    await loadTables();
  });
}
function analizar() {
  const t = selectedTable.value!;
  runOp("Analizar tabla", () => AnalizarTabla(t) as Promise<OperationResult>);
}
function vacuum() {
  const t = selectedTable.value!;
  runOp("VACUUM ANALYZE", () => VacuumTabla(t) as Promise<OperationResult>);
}
async function confirmTruncar() {
  showTruncateDialog.value = false;
  const t = selectedTable.value!;
  await runOp("Truncar tabla", () => TruncarTabla(t) as Promise<OperationResult>, async () => {
    if (selectedTable.value) {
      await loadSchema(selectedTable.value);
      preview.value = null;
      if (activeTab.value === "data") await loadData(selectedTable.value, 0);
    }
  });
}

// ── Visual helpers ────────────────────────────────────────────────────────────
function typeColor(dt: string): string {
  const d = dt.toLowerCase();
  if (d.includes("char") || d.includes("text") || d === "uuid" || d === "name")
    return "bg-sky-100 text-sky-700 border-sky-200 dark:bg-sky-950 dark:text-sky-300";
  if (d.includes("int") || d.includes("serial") || d.includes("numeric") || d.includes("float") || d.includes("double") || d === "real" || d === "money")
    return "bg-orange-100 text-orange-700 border-orange-200";
  if (d === "boolean" || d === "bool")
    return "bg-emerald-100 text-emerald-700 border-emerald-200";
  if (d.includes("time") || d.includes("date") || d.includes("interval"))
    return "bg-violet-100 text-violet-700 border-violet-200";
  if (d.includes("json"))
    return "bg-pink-100 text-pink-700 border-pink-200";
  return "bg-gray-100 text-gray-600 border-gray-200";
}

function constraintBadge(type: string) {
  switch (type) {
    case "PRIMARY KEY": return "bg-amber-100 text-amber-700 border-amber-200";
    case "FOREIGN KEY": return "bg-sky-100 text-sky-700 border-sky-200";
    case "UNIQUE":      return "bg-emerald-100 text-emerald-700 border-emerald-200";
    case "CHECK":       return "bg-violet-100 text-violet-700 border-violet-200";
    default:            return "bg-gray-100 text-gray-600 border-gray-200";
  }
}

function cellBg(rowIdx: number, col: string, value: string): string {
  const key = cellKey(rowIdx, col);
  const flash = flashCell.value;
  if (flash?.key === key) {
    return flash.type === "ok" ? "bg-emerald-100" : "bg-red-100";
  }
  if (isEditingThis(rowIdx, col)) return "bg-blue-50 ring-1 ring-inset ring-blue-400";
  if (isSavingThis(rowIdx, col)) return "bg-yellow-50";
  if (columnMap.value[col]?.IsPrimaryKey) return "bg-amber-50/60";
  return "";
}
</script>

<template>
  <!-- Delete row confirmation -->
  <AlertDialog
    :open="!!deleteRowDialog"
    @update:open="(v) => { if (!v) deleteRowDialog = null }"
  >
    <AlertDialogContent>
      <AlertDialogHeader>
        <AlertDialogTitle class="flex items-center gap-2 text-destructive">
          <Trash2 class="h-4 w-4" /> Eliminar fila
        </AlertDialogTitle>
        <AlertDialogDescription>
          Esta operación eliminará permanentemente la fila con
          <code class="bg-muted px-1 rounded font-mono text-xs">{{ pkColumn }} = {{ deleteRowDialog?.pkValue }}</code>.
          No se puede deshacer.
        </AlertDialogDescription>
      </AlertDialogHeader>
      <AlertDialogFooter>
        <AlertDialogCancel :disabled="deletingRow">Cancelar</AlertDialogCancel>
        <AlertDialogAction
          class="bg-destructive text-destructive-foreground hover:bg-destructive/90"
          :disabled="deletingRow"
          @click="confirmDeleteRow"
        >
          <Loader2 v-if="deletingRow" class="h-4 w-4 animate-spin mr-2" />
          Eliminar
        </AlertDialogAction>
      </AlertDialogFooter>
    </AlertDialogContent>
  </AlertDialog>

  <!-- Truncate confirmation -->
  <AlertDialog :open="showTruncateDialog" @update:open="showTruncateDialog = $event">
    <AlertDialogContent>
      <AlertDialogHeader>
        <AlertDialogTitle class="flex items-center gap-2 text-destructive">
          <Trash2 class="h-5 w-5" /> Vaciar "{{ selectedTable }}"
        </AlertDialogTitle>
        <AlertDialogDescription>
          Elimina <strong>todas las filas</strong> con
          <code class="bg-muted px-1 rounded text-xs">TRUNCATE RESTART IDENTITY CASCADE</code>.
          Esta acción <strong>no se puede deshacer</strong>.
        </AlertDialogDescription>
      </AlertDialogHeader>
      <AlertDialogFooter>
        <AlertDialogCancel>Cancelar</AlertDialogCancel>
        <AlertDialogAction
          class="bg-destructive text-destructive-foreground hover:bg-destructive/90"
          @click="confirmTruncar"
        >
          Sí, vaciar tabla
        </AlertDialogAction>
      </AlertDialogFooter>
    </AlertDialogContent>
  </AlertDialog>

  <!-- ═══ ROOT LAYOUT ════════════════════════════════════════════════════════ -->
  <div class="flex h-full overflow-hidden">

    <!-- ═══ LEFT PANEL ══════════════════════════════════════════════════════ -->
    <aside class="w-60 shrink-0 border-r flex flex-col overflow-hidden bg-sidebar">

      <!-- Header -->
      <div class="px-3 pt-3.5 pb-2.5 border-b">
        <div class="flex items-center justify-between">
          <div class="flex items-center gap-2">
            <div class="h-6 w-6 rounded bg-primary/10 flex items-center justify-center">
              <Database class="h-3.5 w-3.5 text-primary" />
            </div>
            <span class="font-semibold text-sm">Base de Datos</span>
          </div>
          <Button
            variant="ghost" size="icon" class="h-6 w-6 rounded"
            :disabled="isLoadingTables"
            @click="loadTables()"
            title="Recargar"
          >
            <Loader2 v-if="isLoadingTables" class="h-3.5 w-3.5 animate-spin" />
            <RefreshCcw v-else class="h-3.5 w-3.5" />
          </Button>
        </div>
        <div class="flex items-center gap-2 mt-1.5 text-[11px] text-muted-foreground font-mono">
          <span>{{ tables.length }} tablas</span>
          <span class="text-muted-foreground/40">·</span>
          <span>{{ totalSize }}</span>
        </div>
      </div>

      <!-- Search -->
      <div class="px-2 pt-2 pb-1.5 border-b">
        <div class="relative">
          <Search class="absolute left-2 top-1/2 -translate-y-1/2 h-3 w-3 text-muted-foreground/60" />
          <Input
            v-model="tableSearch"
            placeholder="Filtrar tablas..."
            class="pl-6 h-7 text-xs bg-background"
          />
        </div>
      </div>

      <!-- Table list -->
      <div class="flex-1 overflow-y-auto py-1 space-y-px">
        <button
          v-for="t in filteredTables"
          :key="t.Name"
          class="w-full flex items-center gap-2 px-2.5 py-1.5 text-left transition-all relative group rounded-sm mx-1"
          :style="{ width: 'calc(100% - 8px)' }"
          :class="
            selectedTable === t.Name
              ? 'bg-primary/10 text-foreground'
              : 'text-muted-foreground hover:bg-muted/60 hover:text-foreground'
          "
          @click="selectTable(t.Name)"
        >
          <!-- Active strip -->
          <span
            v-if="selectedTable === t.Name"
            class="absolute left-0 top-1 bottom-1 w-0.5 rounded-full bg-primary -ml-px"
          />
          <Table2 class="h-3.5 w-3.5 shrink-0 opacity-60" />
          <span class="flex-1 text-xs font-mono truncate">{{ t.Name }}</span>
          <span class="text-[10px] tabular-nums opacity-50 shrink-0">
            {{ t.RowCount.toLocaleString() }}
          </span>
        </button>

        <p v-if="!isLoadingTables && filteredTables.length === 0"
           class="text-center text-xs text-muted-foreground/60 py-8">
          Sin resultados
        </p>
      </div>

      <!-- Global actions -->
      <div class="border-t p-1.5 space-y-0.5">
        <Button
          variant="ghost" size="sm"
          class="w-full justify-start gap-2 h-7 text-xs text-muted-foreground hover:text-foreground px-2"
          :disabled="!!busyOp"
          @click="doBackup"
        >
          <Loader2 v-if="busyOp === 'Backup BD'" class="h-3.5 w-3.5 animate-spin" />
          <HardDriveDownload v-else class="h-3.5 w-3.5" />
          Backup completo
        </Button>
        <Button
          variant="ghost" size="sm"
          class="w-full justify-start gap-2 h-7 text-xs text-muted-foreground hover:text-foreground px-2"
          :disabled="!!busyOp"
          @click="doImportSQL"
        >
          <Loader2 v-if="busyOp === 'Importar SQL'" class="h-3.5 w-3.5 animate-spin" />
          <FileUp v-else class="h-3.5 w-3.5" />
          Importar SQL
        </Button>
      </div>
    </aside>

    <!-- ═══ RIGHT PANEL ════════════════════════════════════════════════════ -->
    <div class="flex-1 flex flex-col min-w-0 overflow-hidden bg-background">

      <!-- Empty state -->
      <div
        v-if="!selectedTable"
        class="flex-1 flex flex-col items-center justify-center gap-4 text-center px-8 select-none"
      >
        <div class="h-20 w-20 rounded-2xl bg-muted/40 flex items-center justify-center">
          <Database class="h-10 w-10 text-muted-foreground/20" />
        </div>
        <div>
          <p class="text-sm font-medium text-muted-foreground">Selecciona una tabla</p>
          <p class="text-xs text-muted-foreground/50 mt-0.5">
            Explora esquema, edita datos, gestiona índices y restricciones
          </p>
        </div>
      </div>

      <!-- Table content -->
      <template v-else>

        <!-- Table header bar -->
        <div class="border-b px-4 py-2.5 flex items-center gap-3 shrink-0 bg-background">
          <div class="flex-1 min-w-0">
            <div class="flex items-center gap-1 text-sm font-mono leading-none">
              <span class="text-muted-foreground/50 text-xs">public</span>
              <span class="text-muted-foreground/30 mx-0.5">›</span>
              <span class="font-semibold text-foreground">{{ selectedTable }}</span>
            </div>
            <div class="flex items-center gap-2 mt-1 text-[11px] text-muted-foreground">
              <template v-if="schema && !isLoadingSchema">
                <span>{{ schema.TableInfo.RowCount.toLocaleString() }} filas</span>
                <span class="opacity-40">·</span>
                <span>{{ schema.TableInfo.SizeHuman }}</span>
                <span class="opacity-40">·</span>
                <span>{{ schema.Columns.length }} columnas</span>
                <span v-if="!hasPK" class="text-amber-500 ml-1">· sin PK — edición deshabilitada</span>
              </template>
              <Loader2 v-else class="h-3 w-3 animate-spin" />
            </div>
          </div>

          <!-- Action buttons -->
          <div class="flex items-center gap-1 shrink-0">
            <Button
              variant="outline" size="sm" class="h-7 gap-1.5 text-xs"
              :disabled="!!busyOp" @click="importCSV"
            >
              <Loader2 v-if="busyOp === 'Importar CSV'" class="h-3.5 w-3.5 animate-spin" />
              <Upload v-else class="h-3.5 w-3.5" />
              Import CSV
            </Button>
            <Button
              variant="outline" size="sm" class="h-7 gap-1.5 text-xs"
              :disabled="!!busyOp" @click="exportCSV"
            >
              <Download class="h-3.5 w-3.5" />CSV
            </Button>
            <Button
              variant="outline" size="sm" class="h-7 gap-1.5 text-xs"
              :disabled="!!busyOp" @click="exportSQL"
            >
              <FileCode class="h-3.5 w-3.5" />SQL
            </Button>
            <DropdownMenu>
              <DropdownMenuTrigger as-child>
                <Button variant="outline" size="icon" class="h-7 w-7">
                  <MoreHorizontal class="h-4 w-4" />
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end" class="w-44">
                <DropdownMenuItem class="gap-2 text-xs cursor-pointer" @click="loadSchema(selectedTable!)">
                  <RefreshCcw class="h-3.5 w-3.5" /> Recargar esquema
                </DropdownMenuItem>
                <DropdownMenuSeparator />
                <DropdownMenuItem class="gap-2 text-xs cursor-pointer" @click="analizar">
                  <Zap class="h-3.5 w-3.5" /> ANALYZE
                </DropdownMenuItem>
                <DropdownMenuItem class="gap-2 text-xs cursor-pointer" @click="vacuum">
                  <Wrench class="h-3.5 w-3.5" /> VACUUM ANALYZE
                </DropdownMenuItem>
                <DropdownMenuSeparator />
                <DropdownMenuItem
                  class="gap-2 text-xs cursor-pointer text-destructive focus:text-destructive"
                  @click="showTruncateDialog = true"
                >
                  <Trash2 class="h-3.5 w-3.5" /> Truncar tabla…
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          </div>
        </div>

        <!-- Tabs -->
        <Tabs v-model="activeTab" class="flex flex-col flex-1 min-h-0">
          <TabsList class="rounded-none border-b bg-transparent h-9 px-4 justify-start gap-0 shrink-0 w-full">
            <TabsTrigger
              v-for="{ value, icon: Icon, label } in [
                { value: 'schema', icon: Columns, label: 'Esquema' },
                { value: 'data',   icon: LayoutList, label: 'Datos' },
                { value: 'indexes', icon: Zap, label: 'Índices' },
                { value: 'constraints', icon: Link2, label: 'Restricciones' },
              ]"
              :key="value"
              :value="value"
              class="rounded-none border-b-2 border-transparent data-[state=active]:border-primary data-[state=active]:bg-transparent h-9 px-3 gap-1.5 text-xs font-normal data-[state=active]:font-medium"
            >
              <component :is="Icon" class="h-3.5 w-3.5" />
              {{ label }}
            </TabsTrigger>
          </TabsList>

          <!-- ── SCHEMA TAB ──────────────────────────────────────────────── -->
          <TabsContent value="schema" class="flex-1 overflow-y-auto mt-0">
            <div v-if="isLoadingSchema" class="flex items-center justify-center py-16">
              <Loader2 class="h-6 w-6 animate-spin text-muted-foreground" />
            </div>
            <div v-else-if="schema" class="p-4 max-w-4xl">
              <table class="w-full text-xs border-collapse">
                <thead>
                  <tr class="text-left border-b">
                    <th class="pb-2 pr-3 text-[10px] font-medium text-muted-foreground w-8">#</th>
                    <th class="pb-2 pr-6 text-[10px] font-medium text-muted-foreground">Columna</th>
                    <th class="pb-2 pr-6 text-[10px] font-medium text-muted-foreground">Tipo</th>
                    <th class="pb-2 pr-6 text-[10px] font-medium text-muted-foreground w-20 text-center">Nullable</th>
                    <th class="pb-2 text-[10px] font-medium text-muted-foreground">Default</th>
                  </tr>
                </thead>
                <tbody>
                  <tr
                    v-for="col in schema.Columns"
                    :key="col.Name"
                    class="group border-b border-dashed border-muted last:border-0 hover:bg-muted/30 transition-colors"
                  >
                    <td class="py-2.5 pr-3 text-muted-foreground/50 tabular-nums font-mono">
                      {{ col.Position }}
                    </td>
                    <td class="py-2.5 pr-6">
                      <div class="flex items-center gap-1.5">
                        <Key
                          v-if="col.IsPrimaryKey"
                          class="h-3 w-3 text-amber-500 shrink-0"
                          title="Primary Key"
                        />
                        <Link2
                          v-else-if="col.IsForeignKey"
                          class="h-3 w-3 text-sky-500 shrink-0"
                          title="Foreign Key"
                        />
                        <span
                          class="font-mono"
                          :class="col.IsPrimaryKey ? 'font-semibold text-foreground' : ''"
                        >{{ col.Name }}</span>
                        <Lock
                          v-if="col.IsPrimaryKey"
                          class="h-2.5 w-2.5 text-muted-foreground/30 opacity-0 group-hover:opacity-100 transition-opacity"
                        />
                      </div>
                    </td>
                    <td class="py-2.5 pr-6">
                      <Badge
                        variant="outline"
                        class="text-[10px] px-1.5 py-0 font-mono font-normal leading-5"
                        :class="typeColor(col.DataType)"
                      >
                        {{ col.DataType }}
                        <span v-if="col.MaxLength > 0" class="opacity-50">({{ col.MaxLength }})</span>
                      </Badge>
                    </td>
                    <td class="py-2.5 pr-6 text-center">
                      <CheckCircle2 v-if="col.IsNullable" class="h-3.5 w-3.5 text-emerald-500 mx-auto" />
                      <Minus v-else class="h-3.5 w-3.5 text-muted-foreground/25 mx-auto" />
                    </td>
                    <td class="py-2.5 max-w-[220px]">
                      <span
                        v-if="col.Default"
                        class="font-mono text-[10px] text-muted-foreground truncate block"
                        :title="col.Default"
                      >{{ col.Default }}</span>
                      <span v-else class="text-muted-foreground/25 text-[10px]">—</span>
                    </td>
                  </tr>
                </tbody>
              </table>
            </div>
          </TabsContent>

          <!-- ── DATA TAB ───────────────────────────────────────────────── -->
          <TabsContent value="data" class="flex flex-col flex-1 min-h-0 mt-0 overflow-hidden">

            <!-- Data tab toolbar -->
            <div class="shrink-0 border-b px-3 py-2 flex items-center gap-2 bg-muted/10">
              <div class="relative flex-1 max-w-56">
                <Search class="absolute left-2 top-1/2 -translate-y-1/2 h-3 w-3 text-muted-foreground/50" />
                <Input
                  v-model="dataFilter"
                  placeholder="Filtrar en esta página..."
                  class="pl-6 h-7 text-xs"
                />
              </div>
              <Button
                variant="ghost" size="sm"
                class="h-7 gap-1.5 text-xs text-muted-foreground hover:text-foreground"
                :disabled="isLoadingData"
                @click="selectedTable && loadData(selectedTable, dataPage)"
              >
                <Loader2 v-if="isLoadingData" class="h-3.5 w-3.5 animate-spin" />
                <RefreshCcw v-else class="h-3.5 w-3.5" />
                Recargar
              </Button>
              <div v-if="sortCol" class="flex items-center gap-1 text-xs text-muted-foreground border rounded px-2 py-0.5 bg-background">
                <span>Orden:</span>
                <span class="font-mono">{{ sortCol }}</span>
                <ArrowUp v-if="sortDir === 'asc'" class="h-3 w-3" />
                <ArrowDown v-else class="h-3 w-3" />
                <button @click="sortCol = ''; selectedTable && loadData(selectedTable, 0)" class="hover:text-foreground ml-0.5">
                  <X class="h-3 w-3" />
                </button>
              </div>
              <div v-if="hasPK" class="ml-auto flex items-center gap-1 text-[11px] text-muted-foreground/60">
                <Pencil class="h-3 w-3" />
                <span>Clic en celda para editar</span>
              </div>
            </div>

            <!-- Loading -->
            <div v-if="isLoadingData" class="flex-1 flex items-center justify-center">
              <Loader2 class="h-6 w-6 animate-spin text-muted-foreground" />
            </div>

            <template v-else-if="preview">
              <!-- Data grid -->
              <div class="flex-1 overflow-auto">
                <table class="text-[11px] border-collapse w-max min-w-full">
                  <!-- Sticky header -->
                  <thead class="sticky top-0 z-20">
                    <tr class="bg-muted">
                      <!-- Row number column -->
                      <th class="px-2 py-2 border-b border-r text-muted-foreground/50 font-normal text-[10px] w-8 select-none sticky left-0 bg-muted">
                        #
                      </th>
                      <th
                        v-for="col in preview.Columns"
                        :key="col"
                        class="px-3 py-2 border-b border-r last:border-r-0 text-left font-medium text-muted-foreground whitespace-nowrap cursor-pointer hover:bg-muted select-none group/th transition-colors"
                        :class="sortCol === col ? 'text-foreground bg-muted/70' : ''"
                        @click="toggleSort(col)"
                      >
                        <div class="flex items-center gap-1">
                          <Key v-if="columnMap[col]?.IsPrimaryKey" class="h-2.5 w-2.5 text-amber-500 shrink-0" />
                          <Link2 v-else-if="columnMap[col]?.IsForeignKey" class="h-2.5 w-2.5 text-sky-400 shrink-0" />
                          <span class="font-mono">{{ col }}</span>
                          <ArrowUp v-if="sortCol === col && sortDir === 'asc'" class="h-3 w-3 ml-0.5 text-primary" />
                          <ArrowDown v-else-if="sortCol === col && sortDir === 'desc'" class="h-3 w-3 ml-0.5 text-primary" />
                          <ArrowUpDown v-else class="h-3 w-3 ml-0.5 opacity-0 group-hover/th:opacity-40 transition-opacity" />
                        </div>
                      </th>
                      <!-- Actions column (delete) -->
                      <th v-if="hasPK" class="px-2 py-2 border-b w-8 text-muted-foreground/40 font-normal text-[10px]">
                        ···
                      </th>
                    </tr>
                  </thead>

                  <tbody>
                    <tr
                      v-for="(row, i) in filteredRows"
                      :key="i"
                      class="border-b hover:bg-primary/4 transition-colors group/row"
                      :class="i % 2 === 0 ? 'bg-background' : 'bg-muted/10'"
                    >
                      <!-- Row number -->
                      <td class="px-2 py-1 border-r text-center text-muted-foreground/30 tabular-nums text-[10px] select-none sticky left-0 bg-inherit">
                        {{ dataPage * PAGE_SIZE + i + 1 }}
                      </td>

                      <!-- Data cells -->
                      <td
                        v-for="col in preview.Columns"
                        :key="col"
                        class="border-r last:border-r-0 relative min-w-[80px] max-w-[240px] transition-colors"
                        :class="[
                          cellBg(i, col, row[col]),
                          isEditable(col) && hasPK
                            ? 'cursor-text hover:bg-blue-50/50 group/cell'
                            : columnMap[col]?.IsPrimaryKey
                            ? 'cursor-default'
                            : ''
                        ]"
                        @click="hasPK && startEdit(i, col, row[col])"
                      >
                        <!-- Boolean cell toggle -->
                        <template v-if="isBool(col) && hasPK && !isEditingThis(i, col)">
                          <div class="px-3 py-1.5 flex items-center gap-1.5">
                            <button
                              class="flex items-center gap-1.5 hover:opacity-80 transition-opacity"
                              :disabled="!hasPK || !isEditable(col) || isSavingThis(i, col)"
                              @click.stop="hasPK && startEdit(i, col, row[col])"
                            >
                              <Loader2 v-if="isSavingThis(i, col)" class="h-3 w-3 animate-spin text-muted-foreground" />
                              <template v-else>
                                <span
                                  class="inline-block h-3.5 w-7 rounded-full transition-colors flex-shrink-0"
                                  :class="row[col] === 'true' ? 'bg-emerald-400' : 'bg-muted'"
                                />
                                <span class="font-mono text-[10px]" :class="row[col] === 'true' ? 'text-emerald-700' : 'text-muted-foreground/60'">
                                  {{ row[col] === 'true' ? 'true' : 'false' }}
                                </span>
                              </template>
                            </button>
                          </div>
                        </template>

                        <!-- Editing cell input -->
                        <template v-else-if="isEditingThis(i, col)">
                          <div class="flex items-center px-1 py-0.5">
                            <input
                              ref="editInputRef"
                              v-model="editValue"
                              class="flex-1 bg-transparent outline-none font-mono text-[11px] text-foreground px-1 py-0.5 min-w-0"
                              :placeholder="columnMap[col]?.IsNullable ? 'vacio = NULL' : ''"
                              @keydown="(e) => handleEditKeydown(e, i, col)"
                              @blur="commitEdit"
                            />
                            <button
                              class="h-5 w-5 rounded flex items-center justify-center text-emerald-600 hover:bg-emerald-100 shrink-0 transition-colors"
                              @mousedown.prevent="commitEdit"
                              title="Guardar (Enter)"
                            >
                              <Check class="h-3 w-3" />
                            </button>
                            <button
                              class="h-5 w-5 rounded flex items-center justify-center text-muted-foreground hover:bg-muted shrink-0 transition-colors"
                              @mousedown.prevent="cancelEdit"
                              title="Cancelar (Escape)"
                            >
                              <X class="h-3 w-3" />
                            </button>
                          </div>
                        </template>

                        <!-- Static cell display -->
                        <template v-else>
                          <div class="px-3 py-1.5 flex items-center gap-1.5">
                            <Loader2 v-if="isSavingThis(i, col)" class="h-3 w-3 animate-spin text-muted-foreground shrink-0" />
                            <span
                              v-if="row[col] === 'NULL'"
                              class="text-muted-foreground/30 italic font-mono"
                            >null</span>
                            <span
                              v-else
                              class="font-mono truncate"
                              :class="columnMap[col]?.IsPrimaryKey ? 'text-amber-700/80 text-[10px]' : ''"
                              :title="row[col]"
                            >{{ row[col] }}</span>
                            <!-- Edit hint icon on hover -->
                            <Pencil
                              v-if="isEditable(col) && hasPK"
                              class="h-2.5 w-2.5 text-muted-foreground/20 shrink-0 opacity-0 group-hover/cell:opacity-100 ml-auto transition-opacity"
                            />
                          </div>
                        </template>
                      </td>

                      <!-- Delete row button -->
                      <td v-if="hasPK" class="px-1 py-1 text-center">
                        <button
                          class="h-5 w-5 rounded flex items-center justify-center text-muted-foreground/0 group-hover/row:text-muted-foreground/40 hover:!text-destructive hover:bg-red-50 transition-all mx-auto"
                          :title="'Eliminar fila'"
                          @click.stop="pkColumn && askDeleteRow(row[pkColumn])"
                        >
                          <Trash2 class="h-3 w-3" />
                        </button>
                      </td>
                    </tr>

                    <tr v-if="filteredRows.length === 0">
                      <td
                        :colspan="(preview.Columns.length) + (hasPK ? 2 : 1)"
                        class="py-16 text-center text-sm text-muted-foreground/50"
                      >
                        <Database class="h-8 w-8 mx-auto mb-2 opacity-20" />
                        {{ dataFilter ? 'Sin resultados para el filtro' : 'La tabla está vacía' }}
                      </td>
                    </tr>
                  </tbody>
                </table>
              </div>

              <!-- Pagination bar -->
              <div class="shrink-0 border-t px-4 py-2 flex items-center justify-between bg-muted/20">
                <span class="text-[11px] text-muted-foreground">
                  Mostrando <span class="tabular-nums font-medium text-foreground">{{ showingRange }}</span> filas
                  <span v-if="dataFilter" class="ml-1 text-muted-foreground/60">(filtrado: {{ filteredRows.length }})</span>
                </span>
                <div class="flex items-center gap-1.5">
                  <Button
                    variant="outline" size="icon" class="h-6 w-6"
                    :disabled="dataPage === 0 || isLoadingData"
                    @click="prevPage"
                  >
                    <ChevronLeft class="h-3 w-3" />
                  </Button>
                  <span class="text-[11px] text-muted-foreground tabular-nums px-1">
                    {{ dataPage + 1 }} / {{ totalPages }}
                  </span>
                  <Button
                    variant="outline" size="icon" class="h-6 w-6"
                    :disabled="dataPage + 1 >= totalPages || isLoadingData"
                    @click="nextPage"
                  >
                    <ChevronRight class="h-3 w-3" />
                  </Button>
                </div>
              </div>
            </template>

            <div v-else class="flex-1 flex items-center justify-center text-sm text-muted-foreground">
              Cargando…
            </div>
          </TabsContent>

          <!-- ── INDEXES TAB ─────────────────────────────────────────────── -->
          <TabsContent value="indexes" class="flex-1 overflow-y-auto mt-0 p-4">
            <div v-if="isLoadingSchema" class="flex items-center justify-center py-12">
              <Loader2 class="h-5 w-5 animate-spin text-muted-foreground" />
            </div>
            <div v-else-if="!schema?.Indexes?.length" class="flex flex-col items-center justify-center py-16 gap-2 text-muted-foreground/40">
              <Zap class="h-8 w-8" />
              <span class="text-sm">Sin índices definidos</span>
            </div>
            <div v-else class="space-y-2 max-w-3xl">
              <div
                v-for="idx in schema?.Indexes"
                :key="idx.Name"
                class="border rounded-lg p-3 hover:bg-muted/30 transition-colors"
              >
                <div class="flex items-center gap-2 mb-2">
                  <span class="font-mono text-sm font-medium">{{ idx.Name }}</span>
                  <Badge v-if="idx.IsPrimary" variant="outline" class="text-[10px] px-1.5 py-0 bg-amber-100 text-amber-700 border-amber-200">PRIMARY</Badge>
                  <Badge v-else-if="idx.IsUnique" variant="outline" class="text-[10px] px-1.5 py-0 bg-emerald-100 text-emerald-700 border-emerald-200">UNIQUE</Badge>
                </div>
                <code class="block text-[11px] text-muted-foreground bg-muted/40 rounded px-2.5 py-2 font-mono leading-relaxed break-all">{{ idx.Definition }}</code>
              </div>
            </div>
          </TabsContent>

          <!-- ── CONSTRAINTS TAB ─────────────────────────────────────────── -->
          <TabsContent value="constraints" class="flex-1 overflow-y-auto mt-0 p-4">
            <div v-if="isLoadingSchema" class="flex items-center justify-center py-12">
              <Loader2 class="h-5 w-5 animate-spin text-muted-foreground" />
            </div>
            <div v-else-if="!schema?.Constraints?.length" class="flex flex-col items-center justify-center py-16 gap-2 text-muted-foreground/40">
              <Link2 class="h-8 w-8" />
              <span class="text-sm">Sin restricciones definidas</span>
            </div>
            <div v-else class="space-y-2 max-w-3xl">
              <div
                v-for="con in schema?.Constraints"
                :key="con.Name"
                class="border rounded-lg p-3 hover:bg-muted/30 transition-colors"
              >
                <div class="flex items-center gap-2 mb-2">
                  <span class="font-mono text-sm font-medium">{{ con.Name }}</span>
                  <Badge variant="outline" class="text-[10px] px-1.5 py-0" :class="constraintBadge(con.Type)">
                    {{ con.Type }}
                  </Badge>
                </div>
                <code v-if="con.Definition" class="block text-[11px] text-muted-foreground bg-muted/40 rounded px-2.5 py-2 font-mono leading-relaxed break-all">
                  {{ con.Definition }}
                </code>
              </div>
            </div>
          </TabsContent>
        </Tabs>
      </template>
    </div>
  </div>
</template>
