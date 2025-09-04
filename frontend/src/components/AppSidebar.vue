<script setup lang="ts">
import type { SidebarProps } from "@/components/ui/sidebar";
// import HeaderLog from '@/components/HeaderLog.vue';
import NavUser from "@/components/NavUser.vue";
import {
  Sidebar,
  SidebarContent,
  SidebarGroup,
  SidebarGroupContent,
  SidebarGroupLabel,
  SidebarHeader,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarRail,
} from "@/components/ui/sidebar";
import { useRoute } from "vue-router";
const route = useRoute();

const props = defineProps<SidebarProps>();

// This is sample data.
const data = {
  navMain: [
    {
      title: "Facturación",
      url: "/dashboard",
      items: [
        {
          title: "Punto de venta POS",
          url: "/dashboard",
        },
        {
          title: "Productos",
          url: "/dashboard/pos",
        },
        {
          title: "Vendedores",
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
      <SidebarGroup>
        <SidebarGroup v-for="item in data.navMain" :key="item.title">
          <SidebarGroupLabel>{{ item.title }}</SidebarGroupLabel>
          <SidebarGroupContent>
            <SidebarMenu>
              <SidebarMenuItem
                v-for="childItem in item.items"
                :key="childItem.title"
              >
                <SidebarMenuButton
                  as-child
                  :is-active="route.path === childItem.url"
                >
                  <router-link :to="childItem.url">{{
                    childItem.title
                  }}</router-link>
                </SidebarMenuButton>
              </SidebarMenuItem>
            </SidebarMenu>
          </SidebarGroupContent>
        </SidebarGroup>
        <SidebarGroupContent>
          <SidebarMenu> </SidebarMenu>
        </SidebarGroupContent>
      </SidebarGroup>
    </SidebarContent>
    <SidebarRail />
  </Sidebar>
</template>
