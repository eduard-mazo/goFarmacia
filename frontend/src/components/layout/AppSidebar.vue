<script setup lang="ts">
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
  Receipt,
} from "lucide-vue-next";
import NavMain from "@/components/layout/NavMain.vue";
import NavUser from "@/components/layout/NavUser.vue";
import {
  Sidebar,
  SidebarContent,
  SidebarHeader,
  SidebarRail,
} from "@/components/ui/sidebar";
const props = withDefaults(defineProps<SidebarProps>(), {
  collapsible: "icon",
});

const data = {
  navMain: [
    {
      title: "Dashboard",
      url: "/dashboard",
      icon: LayoutDashboard,
    },
    {
      title: "Facturación",
      url: "/dashboard",
      icon: Receipt,
      items: [
        {
          title: "Punto de venta POS",
          url: "/dashboard/pos",
          icon: Store,
        },
        {
          title: "Facturas",
          url: "/dashboard/facturas",
          icon: History,
        },
      ],
    },
    {
      title: "Catalogo",
      url: "/dashboard",
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
      url: "/dashboard",
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
      url: "/dashboard",
      icon: Users,
      items: [
        {
          title: "Información Clientes",
          url: "/dashboard/clientes",
          icon: Contact,
        },
        {
          title: "Información Vendedores",
          url: "/dashboard/vendedores",
          icon: UserCog,
        },
      ],
    },
  ],
};
</script>
<template>
  <Sidebar v-bind="props">
    <SidebarHeader>
      <NavUser />
    </SidebarHeader>
    <SidebarContent>
      <NavMain :items="data.navMain" />
    </SidebarContent>
    <SidebarRail />
  </Sidebar>
</template>
