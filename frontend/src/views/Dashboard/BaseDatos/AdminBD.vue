<script setup lang="ts">
import { ref, computed, onMounted, watch } from "vue";
import { toast } from "vue-sonner";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import { Separator } from "@/components/ui/separator";
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
} from "@/../wailsjs/go/backend/Db";

// ── Types (mirrors Go structs) ────────────────────────────────────────────────
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
const busyOp = ref<string | null>(null); // tracks which op button is spinning

const showTruncateDialog = ref(false);

// ── Computed ──────────────────────────────────────────────────────────────────
const filteredTables = computed(() =>
  tables.value.filter((t) =>
    t.Name.toLowerCase().includes(tableSearch.value.toLowerCase())
  )
);

const totalSize = computed(() => {
  const bytes = tables.value.reduce((a, t) => a + t.SizeBytes, 0);
  if (bytes >= 1024 * 1024 * 1024) return `${(bytes / 1024 / 1024 / 1024).toFixed(2)} GB`;
  if (bytes >= 1024 * 1024) return `${(bytes / 1024 / 1024).toFixed(2)} MB`;
  if (bytes >= 1024) return `${(bytes / 1024).toFixed(2)} KB`;
  return `${bytes} B`;
});

const totalPages = computed(() =>
  preview.value ? Math.max(1, Math.ceil(preview.value.Total / PAGE_SIZE)) : 1
);

const showingRange = computed(() => {
  if (!preview.value) return "";
  const from = dataPage.value * PAGE_SIZE + 1;
  const to = Math.min((dataPage.value + 1) * PAGE_SIZE, preview.value.Total);
  return `${from}–${to} de ${preview.value.Total}`;
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
  selectedTable.value = name;
  schema.value = null;
  preview.value = null;
  dataPage.value = 0;
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
  try {
    preview.value = (await GetDatosTabla(
      name,
      PAGE_SIZE,
      page * PAGE_SIZE
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

// ── Operations ────────────────────────────────────────────────────────────────
async function runOp(
  label: string,
  fn: () => Promise<OperationResult>,
  afterFn?: () => Promise<void>
) {
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
  if (!selectedTable.value) return;
  const t = selectedTable.value;
  runOp("Exportar CSV", () => ExportarTablaCSV(t) as Promise<OperationResult>);
}

function exportSQL() {
  if (!selectedTable.value) return;
  const t = selectedTable.value;
  runOp("Exportar SQL", () => ExportarTablaSQL(t) as Promise<OperationResult>);
}

function doBackup() {
  runOp("Backup BD", () => ExportarBDSQL() as Promise<OperationResult>);
}

function importCSV() {
  if (!selectedTable.value) return;
  const t = selectedTable.value;
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
  if (!selectedTable.value) return;
  const t = selectedTable.value;
  runOp("Analizar tabla", () => AnalizarTabla(t) as Promise<OperationResult>);
}

function vacuum() {
  if (!selectedTable.value) return;
  const t = selectedTable.value;
  runOp("VACUUM ANALYZE", () => VacuumTabla(t) as Promise<OperationResult>);
}

async function confirmTruncar() {
  showTruncateDialog.value = false;
  if (!selectedTable.value) return;
  const t = selectedTable.value;
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
    return "bg-blue-100 text-blue-700 border-blue-200";
  if (
    d.includes("int") ||
    d.includes("serial") ||
    d.includes("numeric") ||
    d.includes("float") ||
    d.includes("double") ||
    d === "real" ||
    d === "money"
  )
    return "bg-orange-100 text-orange-700 border-orange-200";
  if (d === "boolean" || d === "bool")
    return "bg-green-100 text-green-700 border-green-200";
  if (d.includes("time") || d.includes("date") || d.includes("interval"))
    return "bg-purple-100 text-purple-700 border-purple-200";
  if (d.includes("json"))
    return "bg-pink-100 text-pink-700 border-pink-200";
  return "bg-gray-100 text-gray-600 border-gray-200";
}

function constraintBadge(type: string) {
  switch (type) {
    case "PRIMARY KEY": return "bg-amber-100 text-amber-700 border-amber-200";
    case "FOREIGN KEY": return "bg-blue-100 text-blue-700 border-blue-200";
    case "UNIQUE":      return "bg-green-100 text-green-700 border-green-200";
    case "CHECK":       return "bg-purple-100 text-purple-700 border-purple-200";
    default:            return "bg-gray-100 text-gray-600 border-gray-200";
  }
}
</script>

<template>
  <!-- Truncate confirmation dialog -->
  <AlertDialog :open="showTruncateDialog" @update:open="showTruncateDialog = $event">
    <AlertDialogContent>
      <AlertDialogHeader>
        <AlertDialogTitle class="flex items-center gap-2 text-destructive">
          <Trash2 class="h-5 w-5" />
          Vaciar tabla "{{ selectedTable }}"
        </AlertDialogTitle>
        <AlertDialogDescription>
          Esta acción eliminará <strong>todas las filas</strong> de la tabla
          <code class="bg-muted px-1 rounded">{{ selectedTable }}</code> con
          <code>RESTART IDENTITY CASCADE</code>. Esta operación
          <strong>no se puede deshacer</strong>.
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

  <!-- Main layout -->
  <div class="flex h-full overflow-hidden font-sans">

    <!-- ═══ LEFT PANEL ═══════════════════════════════════════════════════════ -->
    <aside class="w-64 shrink-0 border-r flex flex-col bg-muted/20 overflow-hidden">

      <!-- Panel header -->
      <div class="px-3 pt-4 pb-3 border-b">
        <div class="flex items-center justify-between mb-0.5">
          <div class="flex items-center gap-2">
            <Database class="h-4 w-4 text-primary shrink-0" />
            <span class="font-semibold text-sm tracking-tight">Base de Datos</span>
          </div>
          <Button
            variant="ghost"
            size="icon"
            class="h-6 w-6"
            :disabled="isLoadingTables"
            @click="loadTables()"
            title="Recargar tablas"
          >
            <Loader2 v-if="isLoadingTables" class="h-3.5 w-3.5 animate-spin" />
            <RefreshCcw v-else class="h-3.5 w-3.5" />
          </Button>
        </div>
        <p class="text-[11px] text-muted-foreground">
          {{ tables.length }} tablas · {{ totalSize }}
        </p>
      </div>

      <!-- Search -->
      <div class="px-2 py-2 border-b">
        <div class="relative">
          <Search class="absolute left-2.5 top-1/2 -translate-y-1/2 h-3.5 w-3.5 text-muted-foreground" />
          <Input
            v-model="tableSearch"
            placeholder="Buscar tabla..."
            class="pl-8 h-7 text-xs"
          />
        </div>
      </div>

      <!-- Table list -->
      <div class="flex-1 overflow-y-auto py-1">
        <div
          v-if="isLoadingTables && tables.length === 0"
          class="flex items-center justify-center py-8"
        >
          <Loader2 class="h-5 w-5 animate-spin text-muted-foreground" />
        </div>

        <button
          v-for="t in filteredTables"
          :key="t.Name"
          class="w-full flex items-center gap-2 px-2 py-1.5 text-sm transition-colors relative group"
          :class="
            selectedTable === t.Name
              ? 'bg-primary/8 text-foreground font-medium'
              : 'text-muted-foreground hover:bg-muted hover:text-foreground'
          "
          @click="selectTable(t.Name)"
        >
          <!-- Active indicator -->
          <span
            v-if="selectedTable === t.Name"
            class="absolute left-0 top-1 bottom-1 w-0.5 rounded-r bg-primary"
          />
          <Table2 class="h-3.5 w-3.5 shrink-0 text-muted-foreground" />
          <span class="flex-1 text-left truncate text-xs font-mono">{{ t.Name }}</span>
          <span class="text-[10px] text-muted-foreground tabular-nums">
            {{ t.RowCount.toLocaleString() }}
          </span>
        </button>

        <p
          v-if="!isLoadingTables && filteredTables.length === 0"
          class="text-center text-xs text-muted-foreground py-6"
        >
          Sin resultados
        </p>
      </div>

      <!-- Global actions -->
      <div class="border-t p-2 space-y-0.5">
        <Button
          variant="ghost"
          size="sm"
          class="w-full justify-start gap-2 h-8 text-xs text-muted-foreground hover:text-foreground"
          :disabled="busyOp === 'Backup BD'"
          @click="doBackup"
        >
          <Loader2 v-if="busyOp === 'Backup BD'" class="h-3.5 w-3.5 animate-spin" />
          <HardDriveDownload v-else class="h-3.5 w-3.5" />
          Backup base de datos
        </Button>
        <Button
          variant="ghost"
          size="sm"
          class="w-full justify-start gap-2 h-8 text-xs text-muted-foreground hover:text-foreground"
          :disabled="busyOp === 'Importar SQL'"
          @click="doImportSQL"
        >
          <Loader2 v-if="busyOp === 'Importar SQL'" class="h-3.5 w-3.5 animate-spin" />
          <FileUp v-else class="h-3.5 w-3.5" />
          Importar SQL
        </Button>
      </div>
    </aside>

    <!-- ═══ RIGHT PANEL ══════════════════════════════════════════════════════ -->
    <div class="flex-1 flex flex-col min-w-0 overflow-hidden bg-background">

      <!-- Empty state -->
      <div
        v-if="!selectedTable"
        class="flex-1 flex flex-col items-center justify-center gap-3 text-center px-8"
      >
        <Database class="h-16 w-16 text-muted-foreground/20" />
        <div>
          <p class="text-base font-medium text-muted-foreground">Selecciona una tabla</p>
          <p class="text-sm text-muted-foreground/60 mt-0.5">
            Explora su estructura, datos, índices y restricciones
          </p>
        </div>
      </div>

      <!-- Table content -->
      <template v-else>

        <!-- Table header bar -->
        <div class="border-b px-4 py-3 flex items-center gap-3 shrink-0">
          <div class="flex-1 min-w-0">
            <div class="flex items-center gap-1.5 text-sm font-mono">
              <span class="text-muted-foreground text-xs">public</span>
              <span class="text-muted-foreground/50">›</span>
              <span class="font-semibold text-foreground">{{ selectedTable }}</span>
            </div>
            <p class="text-[11px] text-muted-foreground mt-0.5">
              <template v-if="schema">
                {{ schema.TableInfo.RowCount.toLocaleString() }} filas
                · {{ schema.TableInfo.SizeHuman }}
                · {{ schema.Columns.length }} columnas
              </template>
              <Loader2 v-else class="h-3 w-3 animate-spin inline" />
            </p>
          </div>

          <!-- Per-table action buttons -->
          <div class="flex items-center gap-1.5 shrink-0">
            <Button
              variant="outline"
              size="sm"
              class="h-7 gap-1.5 text-xs"
              :disabled="!!busyOp"
              @click="importCSV"
            >
              <Loader2 v-if="busyOp === 'Importar CSV'" class="h-3.5 w-3.5 animate-spin" />
              <Upload v-else class="h-3.5 w-3.5" />
              Importar CSV
            </Button>

            <Button
              variant="outline"
              size="sm"
              class="h-7 gap-1.5 text-xs"
              :disabled="!!busyOp"
              @click="exportCSV"
            >
              <Loader2 v-if="busyOp === 'Exportar CSV'" class="h-3.5 w-3.5 animate-spin" />
              <Download v-else class="h-3.5 w-3.5" />
              CSV
            </Button>

            <Button
              variant="outline"
              size="sm"
              class="h-7 gap-1.5 text-xs"
              :disabled="!!busyOp"
              @click="exportSQL"
            >
              <Loader2 v-if="busyOp === 'Exportar SQL'" class="h-3.5 w-3.5 animate-spin" />
              <FileCode v-else class="h-3.5 w-3.5" />
              SQL
            </Button>

            <!-- More actions dropdown -->
            <DropdownMenu>
              <DropdownMenuTrigger as-child>
                <Button variant="outline" size="icon" class="h-7 w-7">
                  <MoreHorizontal class="h-4 w-4" />
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end" class="w-44">
                <DropdownMenuItem class="gap-2 text-xs cursor-pointer" @click="loadSchema(selectedTable!)">
                  <RefreshCcw class="h-3.5 w-3.5" />
                  Recargar esquema
                </DropdownMenuItem>
                <DropdownMenuSeparator />
                <DropdownMenuItem class="gap-2 text-xs cursor-pointer" @click="analizar">
                  <Zap class="h-3.5 w-3.5" />
                  ANALYZE
                </DropdownMenuItem>
                <DropdownMenuItem class="gap-2 text-xs cursor-pointer" @click="vacuum">
                  <Wrench class="h-3.5 w-3.5" />
                  VACUUM ANALYZE
                </DropdownMenuItem>
                <DropdownMenuSeparator />
                <DropdownMenuItem
                  class="gap-2 text-xs cursor-pointer text-destructive focus:text-destructive"
                  @click="showTruncateDialog = true"
                >
                  <Trash2 class="h-3.5 w-3.5" />
                  Truncar tabla…
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          </div>
        </div>

        <!-- Tabs -->
        <Tabs v-model="activeTab" class="flex flex-col flex-1 min-h-0">
          <TabsList class="rounded-none border-b bg-transparent h-9 px-4 justify-start gap-0 shrink-0">
            <TabsTrigger
              value="schema"
              class="rounded-none border-b-2 border-transparent data-[state=active]:border-primary data-[state=active]:bg-transparent h-9 px-3 gap-1.5 text-xs"
            >
              <Columns class="h-3.5 w-3.5" />
              Esquema
            </TabsTrigger>
            <TabsTrigger
              value="data"
              class="rounded-none border-b-2 border-transparent data-[state=active]:border-primary data-[state=active]:bg-transparent h-9 px-3 gap-1.5 text-xs"
            >
              <LayoutList class="h-3.5 w-3.5" />
              Datos
            </TabsTrigger>
            <TabsTrigger
              value="indexes"
              class="rounded-none border-b-2 border-transparent data-[state=active]:border-primary data-[state=active]:bg-transparent h-9 px-3 gap-1.5 text-xs"
            >
              <Zap class="h-3.5 w-3.5" />
              Índices
            </TabsTrigger>
            <TabsTrigger
              value="constraints"
              class="rounded-none border-b-2 border-transparent data-[state=active]:border-primary data-[state=active]:bg-transparent h-9 px-3 gap-1.5 text-xs"
            >
              <Link2 class="h-3.5 w-3.5" />
              Restricciones
            </TabsTrigger>
          </TabsList>

          <!-- ── Schema tab ─────────────────────────────────────────────── -->
          <TabsContent value="schema" class="flex-1 overflow-y-auto mt-0">
            <div v-if="isLoadingSchema" class="flex items-center justify-center py-16">
              <Loader2 class="h-6 w-6 animate-spin text-muted-foreground" />
            </div>

            <div v-else-if="schema" class="p-4">
              <Table class="text-xs">
                <TableHeader>
                  <TableRow class="hover:bg-transparent border-b">
                    <TableHead class="w-8 text-[10px] text-muted-foreground py-2">#</TableHead>
                    <TableHead class="text-[10px] text-muted-foreground py-2">Columna</TableHead>
                    <TableHead class="text-[10px] text-muted-foreground py-2">Tipo</TableHead>
                    <TableHead class="w-20 text-[10px] text-muted-foreground py-2 text-center">Nullable</TableHead>
                    <TableHead class="text-[10px] text-muted-foreground py-2">Valor por defecto</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  <TableRow
                    v-for="col in schema.Columns"
                    :key="col.Name"
                    class="hover:bg-muted/50 border-b border-dashed border-muted"
                  >
                    <TableCell class="py-2 text-muted-foreground tabular-nums">
                      {{ col.Position }}
                    </TableCell>
                    <TableCell class="py-2">
                      <div class="flex items-center gap-1.5">
                        <Key
                          v-if="col.IsPrimaryKey"
                          class="h-3 w-3 text-amber-500 shrink-0"
                          title="Primary Key"
                        />
                        <Link2
                          v-else-if="col.IsForeignKey"
                          class="h-3 w-3 text-blue-500 shrink-0"
                          title="Foreign Key"
                        />
                        <span
                          class="font-mono"
                          :class="col.IsPrimaryKey ? 'font-semibold text-foreground' : 'text-foreground/80'"
                        >
                          {{ col.Name }}
                        </span>
                      </div>
                    </TableCell>
                    <TableCell class="py-2">
                      <Badge
                        variant="outline"
                        class="text-[10px] px-1.5 py-0 font-mono font-normal"
                        :class="typeColor(col.DataType)"
                      >
                        {{ col.DataType }}
                        <span v-if="col.MaxLength > 0" class="opacity-60">({{ col.MaxLength }})</span>
                      </Badge>
                    </TableCell>
                    <TableCell class="py-2 text-center">
                      <CheckCircle2
                        v-if="col.IsNullable"
                        class="h-3.5 w-3.5 text-green-500 mx-auto"
                      />
                      <Minus v-else class="h-3.5 w-3.5 text-muted-foreground/40 mx-auto" />
                    </TableCell>
                    <TableCell class="py-2 max-w-[240px]">
                      <span
                        v-if="col.Default"
                        class="font-mono text-[10px] text-muted-foreground truncate block"
                        :title="col.Default"
                      >
                        {{ col.Default }}
                      </span>
                      <span v-else class="text-muted-foreground/30 text-[10px]">—</span>
                    </TableCell>
                  </TableRow>
                </TableBody>
              </Table>
            </div>
          </TabsContent>

          <!-- ── Data tab ───────────────────────────────────────────────── -->
          <TabsContent value="data" class="flex flex-col flex-1 min-h-0 mt-0 overflow-hidden">
            <div v-if="isLoadingData" class="flex-1 flex items-center justify-center">
              <Loader2 class="h-6 w-6 animate-spin text-muted-foreground" />
            </div>

            <template v-else-if="preview">
              <!-- Scrollable table -->
              <div class="flex-1 overflow-auto">
                <table class="w-full text-[11px] border-collapse">
                  <thead class="sticky top-0 z-10 bg-muted/80 backdrop-blur-sm">
                    <tr>
                      <th
                        v-for="col in preview.Columns"
                        :key="col"
                        class="text-left px-3 py-2 font-mono font-medium text-muted-foreground border-b border-r last:border-r-0 whitespace-nowrap"
                      >
                        {{ col }}
                      </th>
                    </tr>
                  </thead>
                  <tbody>
                    <tr
                      v-for="(row, i) in preview.Rows"
                      :key="i"
                      class="border-b hover:bg-primary/5 transition-colors"
                      :class="i % 2 === 0 ? 'bg-background' : 'bg-muted/20'"
                    >
                      <td
                        v-for="col in preview.Columns"
                        :key="col"
                        class="px-3 py-1.5 border-r last:border-r-0 max-w-[200px]"
                      >
                        <span
                          v-if="row[col] === 'NULL'"
                          class="text-muted-foreground/40 italic"
                        >null</span>
                        <span
                          v-else
                          class="font-mono truncate block"
                          :title="row[col]"
                        >{{ row[col] }}</span>
                      </td>
                    </tr>
                    <tr v-if="preview.Rows.length === 0">
                      <td
                        :colspan="preview.Columns.length"
                        class="text-center py-12 text-muted-foreground"
                      >
                        La tabla está vacía
                      </td>
                    </tr>
                  </tbody>
                </table>
              </div>

              <!-- Pagination bar -->
              <div class="shrink-0 border-t px-4 py-2 flex items-center justify-between bg-muted/20">
                <span class="text-[11px] text-muted-foreground">
                  Mostrando {{ showingRange }}
                </span>
                <div class="flex items-center gap-2">
                  <Button
                    variant="outline"
                    size="icon"
                    class="h-6 w-6"
                    :disabled="dataPage === 0 || isLoadingData"
                    @click="prevPage"
                  >
                    <ChevronLeft class="h-3 w-3" />
                  </Button>
                  <span class="text-[11px] text-muted-foreground tabular-nums">
                    {{ dataPage + 1 }} / {{ totalPages }}
                  </span>
                  <Button
                    variant="outline"
                    size="icon"
                    class="h-6 w-6"
                    :disabled="dataPage + 1 >= totalPages || isLoadingData"
                    @click="nextPage"
                  >
                    <ChevronRight class="h-3 w-3" />
                  </Button>
                </div>
              </div>
            </template>

            <div v-else class="flex-1 flex items-center justify-center">
              <p class="text-sm text-muted-foreground">Cargando datos…</p>
            </div>
          </TabsContent>

          <!-- ── Indexes tab ─────────────────────────────────────────────── -->
          <TabsContent value="indexes" class="flex-1 overflow-y-auto mt-0 p-4">
            <div v-if="isLoadingSchema" class="flex items-center justify-center py-16">
              <Loader2 class="h-6 w-6 animate-spin text-muted-foreground" />
            </div>

            <div v-else-if="schema?.Indexes?.length === 0" class="text-center py-12">
              <Zap class="h-8 w-8 text-muted-foreground/20 mx-auto mb-2" />
              <p class="text-sm text-muted-foreground">Sin índices definidos</p>
            </div>

            <div v-else class="space-y-3">
              <div
                v-for="idx in schema?.Indexes"
                :key="idx.Name"
                class="border rounded-lg p-3 bg-muted/20 hover:bg-muted/40 transition-colors"
              >
                <div class="flex items-center gap-2 mb-2">
                  <span class="font-mono text-sm font-medium">{{ idx.Name }}</span>
                  <Badge
                    v-if="idx.IsPrimary"
                    variant="outline"
                    class="text-[10px] px-1.5 py-0 bg-amber-100 text-amber-700 border-amber-200"
                  >
                    PRIMARY
                  </Badge>
                  <Badge
                    v-else-if="idx.IsUnique"
                    variant="outline"
                    class="text-[10px] px-1.5 py-0 bg-green-100 text-green-700 border-green-200"
                  >
                    UNIQUE
                  </Badge>
                </div>
                <code class="block text-[11px] text-muted-foreground bg-muted/50 rounded px-2 py-1.5 font-mono leading-relaxed break-all">
                  {{ idx.Definition }}
                </code>
              </div>
            </div>
          </TabsContent>

          <!-- ── Constraints tab ─────────────────────────────────────────── -->
          <TabsContent value="constraints" class="flex-1 overflow-y-auto mt-0 p-4">
            <div v-if="isLoadingSchema" class="flex items-center justify-center py-16">
              <Loader2 class="h-6 w-6 animate-spin text-muted-foreground" />
            </div>

            <div v-else-if="schema?.Constraints?.length === 0" class="text-center py-12">
              <Link2 class="h-8 w-8 text-muted-foreground/20 mx-auto mb-2" />
              <p class="text-sm text-muted-foreground">Sin restricciones definidas</p>
            </div>

            <div v-else class="space-y-3">
              <div
                v-for="con in schema?.Constraints"
                :key="con.Name"
                class="border rounded-lg p-3 bg-muted/20 hover:bg-muted/40 transition-colors"
              >
                <div class="flex items-center gap-2 mb-2">
                  <span class="font-mono text-sm font-medium">{{ con.Name }}</span>
                  <Badge
                    variant="outline"
                    class="text-[10px] px-1.5 py-0"
                    :class="constraintBadge(con.Type)"
                  >
                    {{ con.Type }}
                  </Badge>
                </div>
                <code
                  v-if="con.Definition"
                  class="block text-[11px] text-muted-foreground bg-muted/50 rounded px-2 py-1.5 font-mono leading-relaxed break-all"
                >
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
