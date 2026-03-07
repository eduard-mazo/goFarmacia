<script setup lang="ts">
import { computed } from "vue";
import type { LucideIcon } from "lucide-vue-next";
import type { SidebarProps } from "@/components/ui/sidebar";
import {
  LayoutDashboard,
  Store,
  History,
  Package,
  Warehouse,
  Users,
  Contact,
  UserCog,
  Truck,
  BarChart3,
  PackageSearch,
  Settings,
  DatabaseZap,
  Wifi,
  WifiOff,
  AlertCircle,
  Mail,
  Landmark,
  HardDrive,
} from "lucide-vue-next";
import NavMain from "@/components/layout/NavMain.vue";
import NavUser from "@/components/layout/NavUser.vue";
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarHeader,
} from "@/components/ui/sidebar";
import { useModeStore } from "@/stores/mode";
import type { AppMode } from "@/stores/mode";
import { useDBStore } from "@/stores/dbStore";

const props = withDefaults(defineProps<SidebarProps>(), {
  collapsible: "icon",
});

const modeStore = useModeStore();
const dbStore = useDBStore();

type NavItem = {
  title: string;
  url?: string;
  icon?: LucideIcon;
  items?: { title: string; url: string; icon?: LucideIcon }[];
};

const posNav: NavItem[] = [
  { title: "Dashboard", url: "/dashboard", icon: LayoutDashboard },
  { title: "Punto de Venta", url: "/dashboard/pos", icon: Store },
  { title: "Facturas", url: "/dashboard/facturas", icon: History },
];

const erpNav: NavItem[] = [
  { title: "Dashboard ERP", url: "/dashboard/erp", icon: LayoutDashboard },
  {
    title: "Catálogo",
    icon: Package,
    items: [{ title: "Productos", url: "/dashboard/productos" }],
  },
  {
    title: "Inventario",
    icon: Warehouse,
    items: [{ title: "Control de Stock", url: "/dashboard/controlStock" }],
  },
  {
    title: "Personas",
    icon: Users,
    items: [
      { title: "Clientes", url: "/dashboard/clientes", icon: Contact },
      { title: "Vendedores", url: "/dashboard/vendedores", icon: UserCog },
      { title: "Proveedores", url: "/dashboard/proveedores", icon: Truck },
    ],
  },
  {
    title: "Compras",
    icon: Mail,
    items: [
      { title: "Facturas Electrónicas", url: "/dashboard/compras/facturas", icon: Mail },
    ],
  },
  {
    title: "Tesorería",
    icon: Landmark,
    items: [
      { title: "Transferencias Bancolombia", url: "/dashboard/tesoreria/bancolombia", icon: Landmark },
    ],
  },
  {
    title: "Reportes",
    icon: BarChart3,
    items: [
      { title: "Ventas", url: "/dashboard/reportes/ventas" },
      { title: "Inventario", url: "/dashboard/reportes/inventario", icon: PackageSearch },
    ],
  },
  {
    title: "Sistema",
    icon: Settings,
    items: [
      { title: "Configuración", url: "/dashboard/configuracion", icon: Settings },
      { title: "Backups Drive", url: "/dashboard/configuracion/backups", icon: HardDrive },
      { title: "Base de Datos", url: "/dashboard/basedatos", icon: DatabaseZap },
    ],
  },
];

const currentNav = computed(() =>
  modeStore.currentMode === "pos" ? posNav : erpNav
);

const dbStatus = computed(() => {
  if (dbStore.setupMode) return { label: "Sin configurar", icon: AlertCircle, color: "text-amber-500" };
  if (dbStore.connected) return { label: "Conectado", icon: Wifi, color: "text-emerald-500" };
  return { label: "Sin conexión", icon: WifiOff, color: "text-red-500" };
});
</script>

<template>
  <Sidebar v-bind="props">
    <SidebarHeader class="border-b border-sidebar-border pb-0">
      <NavUser />

      <!-- Mode switcher -->
      <div v-if="modeStore.isAdmin" class="px-2 pb-3 pt-1">
        <div class="flex gap-1 p-1 rounded-lg bg-muted/60">
          <button
            v-for="mode in (['pos', 'erp'] as AppMode[])"
            :key="mode"
            @click="modeStore.setMode(mode)"
            class="flex-1 h-7 rounded-md text-xs font-semibold tracking-wide transition-all duration-150"
            :class="modeStore.currentMode === mode
              ? 'bg-background text-foreground shadow-sm'
              : 'text-muted-foreground hover:text-foreground'"
          >
            {{ mode.toUpperCase() }}
          </button>
        </div>
      </div>
      <div v-else class="px-3 pb-3 pt-1">
        <span class="text-[10px] font-bold text-muted-foreground tracking-widest uppercase">
          Punto de Venta
        </span>
      </div>
    </SidebarHeader>

    <SidebarContent>
      <NavMain :items="currentNav" />
    </SidebarContent>

    <SidebarFooter class="border-t border-sidebar-border">
      <div class="px-3 py-2.5 flex items-center gap-2.5 text-xs">
        <div
          class="flex items-center justify-center h-5 w-5 rounded-full shrink-0"
          :class="{
            'bg-emerald-500/10': dbStore.connected,
            'bg-amber-500/10': dbStore.setupMode,
            'bg-red-500/10': !dbStore.connected && !dbStore.setupMode,
          }"
        >
          <component :is="dbStatus.icon" class="h-3 w-3 shrink-0" :class="dbStatus.color" />
        </div>
        <div class="min-w-0">
          <span class="truncate text-muted-foreground font-medium">{{ dbStatus.label }}</span>
        </div>
      </div>
    </SidebarFooter>
  </Sidebar>
</template>
