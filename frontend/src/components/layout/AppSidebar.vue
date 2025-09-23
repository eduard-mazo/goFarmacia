<script setup lang="ts">
import type { SidebarProps } from "@/components/ui/sidebar";
// CAMBIO: Se importan nuevos íconos más descriptivos
import {
  LayoutDashboard,
  Store,
  History,
  Package,
  Warehouse,
  Users,
  Contact,
  UserCog,
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

// CAMBIO: Se actualiza la estructura de navegación con los nuevos íconos
const data = {
  navMain: [
    {
      title: "Facturación",
      url: "/dashboard",
      icon: LayoutDashboard, // Ícono más general para un dashboard
      items: [
        {
          title: "Punto de venta POS",
          url: "/dashboard/pos",
          icon: Store, // Ícono de tienda, más adecuado para POS
        },
        {
          title: "Facturas",
          url: "/dashboard/facturas",
          icon: History, // Ícono de historial, perfecto para facturas pasadas
        },
      ],
    },
    {
      title: "Catalogo",
      url: "/dashboard",
      icon: Package, // Ícono de paquete para productos
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
      icon: Warehouse, // Ícono de almacén para stock
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
      icon: Users, // Ícono de grupo para la sección principal
      items: [
        {
          title: "Información Clientes",
          url: "/dashboard/clientes",
          icon: Contact, // Ícono específico para clientes/contactos
        },
        {
          title: "Información Vendedores",
          url: "/dashboard/vendedores",
          icon: UserCog, // Ícono de usuario con engranaje para personal/vendedores
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
