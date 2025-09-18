<script setup lang="ts">
import type { SidebarProps } from "@/components/ui/sidebar";
import { Wallet, HandCoins, Users } from "lucide-vue-next";
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
      title: "Facturación",
      url: "/dashboard",
      icon: Wallet,
      items: [
        {
          title: "Punto de venta POS",
          url: "/dashboard/pos",
          icon: HandCoins,
        },
        {
          title: "Facturas",
          url: "/dashboard/facturas",
          icon: HandCoins,
        },
      ],
    },
    {
      title: "Catalogo",
      url: "/dashboard",
      icon: Users,
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
      icon: Users,
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
        },
        {
          title: "Información Vendedores",
          url: "/dashboard/vendedores",
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
