<script setup lang="ts">
import type { SidebarProps } from "@/components/ui/sidebar";
import { Wallet, HandCoins, Users } from "lucide-vue-next";
import NavMain from "@/components/NavMain.vue";
import NavUser from "@/components/NavUser.vue";
import {
  Sidebar,
  SidebarContent,
  SidebarHeader,
  SidebarRail,
} from "@/components/ui/sidebar";
const props = withDefaults(defineProps<SidebarProps>(), {
  collapsible: "icon",
});
// This is sample data.
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
      ],
    },
    {
      title: "Usuarios",
      url: "/dashboard",
      icon: Users,
      items: [
        {
          title: "Productos",
          url: "/dashboard/productos",
        },
        {
          title: "Vendedores",
          url: "/dashboard/vendedores",
        },
        {
          title: "Clientes",
          url: "/dashboard/clientes",
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
