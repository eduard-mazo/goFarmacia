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
} from "lucide-vue-next";
import NavMain from "@/components/layout/NavMain.vue";
import NavUser from "@/components/layout/NavUser.vue";
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarHeader,
  SidebarRail,
} from "@/components/ui/sidebar";
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
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
  {
    title: "Dashboard",
    url: "/dashboard",
    icon: LayoutDashboard,
  },
  {
    title: "Punto de Venta",
    url: "/dashboard/pos",
    icon: Store,
  },
  {
    title: "Facturas",
    url: "/dashboard/facturas",
    icon: History,
  },
];

const erpNav: NavItem[] = [
  {
    title: "Dashboard ERP",
    url: "/dashboard/erp",
    icon: LayoutDashboard,
  },
  {
    title: "Catálogo",
    icon: Package,
    items: [
      {
        title: "Productos",
        url: "/dashboard/productos",
      },
    ],
  },
  {
    title: "Inventario",
    icon: Warehouse,
    items: [
      {
        title: "Control de Stock",
        url: "/dashboard/controlStock",
      },
    ],
  },
  {
    title: "Personas",
    icon: Users,
    items: [
      {
        title: "Clientes",
        url: "/dashboard/clientes",
        icon: Contact,
      },
      {
        title: "Vendedores",
        url: "/dashboard/vendedores",
        icon: UserCog,
      },
      {
        title: "Proveedores",
        url: "/dashboard/proveedores",
        icon: Truck,
      },
    ],
  },
  {
    title: "Reportes",
    icon: BarChart3,
    items: [
      {
        title: "Ventas",
        url: "/dashboard/reportes/ventas",
      },
      {
        title: "Inventario",
        url: "/dashboard/reportes/inventario",
        icon: PackageSearch,
      },
    ],
  },
  {
    title: "Base de Datos",
    url: "/dashboard/basedatos",
    icon: DatabaseZap,
  },
  {
    title: "Configuración",
    url: "/dashboard/configuracion",
    icon: Settings,
  },
];

const currentNav = computed(() =>
  modeStore.currentMode === "pos" ? posNav : erpNav
);

const dbStatusLabel = computed(() => {
  if (dbStore.setupMode) return "Sin configurar";
  if (dbStore.connected) return "Base de datos OK";
  return "Sin conexión";
});
</script>

<template>
  <Sidebar v-bind="props">
    <SidebarHeader>
      <NavUser />
      <div class="px-2 pb-2">
        <Tabs
          :model-value="modeStore.currentMode"
          @update:model-value="(v) => modeStore.setMode(v as AppMode)"
          class="w-full"
        >
          <TabsList
            class="w-full"
            :class="modeStore.isAdmin ? 'grid grid-cols-2' : 'grid grid-cols-1'"
          >
            <TabsTrigger value="pos">POS</TabsTrigger>
            <TabsTrigger v-if="modeStore.isAdmin" value="erp">ERP</TabsTrigger>
          </TabsList>
        </Tabs>
      </div>
    </SidebarHeader>
    <SidebarContent>
      <NavMain :items="currentNav" />
    </SidebarContent>
    <SidebarFooter>
      <div class="px-3 py-2 flex items-center gap-2 text-xs text-muted-foreground border-t">
        <!-- Status dot -->
        <span
          class="h-2 w-2 rounded-full shrink-0 transition-colors duration-500"
          :class="{
            'bg-green-500': dbStore.connected,
            'bg-amber-400 animate-pulse': dbStore.setupMode,
            'bg-red-500': !dbStore.connected && !dbStore.setupMode,
          }"
        />
        <span class="truncate">{{ dbStatusLabel }}</span>
      </div>
    </SidebarFooter>
    <SidebarRail />
  </Sidebar>
</template>
