<script setup lang="ts">
import type { LucideIcon } from "lucide-vue-next";
import { ChevronRight } from "lucide-vue-next";
import { useRoute } from "vue-router";
import { RouterLink } from "vue-router";
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@/components/ui/collapsible";
import {
  SidebarGroup,
  SidebarGroupLabel,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarMenuSub,
  SidebarMenuSubItem,
  SidebarMenuSubButton,
  useSidebar,
} from "@/components/ui/sidebar";

defineProps<{
  items: {
    title: string;
    url?: string;
    icon?: LucideIcon;
    items?: {
      title: string;
      url: string;
      icon?: LucideIcon;
    }[];
  }[];
}>();

const { state, setOpen } = useSidebar();
const route = useRoute();

const isParentActive = (item: { items?: { url: string }[] }): boolean => {
  if (!item.items) return false;
  return item.items.some((subItem) => route.path === subItem.url);
};

function handleMenuClick() {
  if (state.value === "collapsed") setOpen(true);
}
</script>

<template>
  <SidebarGroup>
    <SidebarGroupLabel class="text-[10px] uppercase tracking-widest font-semibold text-muted-foreground/60 px-3 mb-1">
      Navegación
    </SidebarGroupLabel>
    <SidebarMenu>
      <template v-for="item in items" :key="item.title">
        <Collapsible
          v-if="item.items && item.items.length > 0"
          as-child
          :default-open="isParentActive(item)"
          class="group/collapsible"
        >
          <SidebarMenuItem>
            <CollapsibleTrigger as-child @click="handleMenuClick">
              <SidebarMenuButton
                :tooltip="item.title"
                class="rounded-md"
                :class="isParentActive(item) ? 'text-primary font-medium' : 'text-muted-foreground hover:text-foreground'"
              >
                <component :is="item.icon" v-if="item.icon" class="h-4 w-4 shrink-0" />
                <span>{{ item.title }}</span>
                <ChevronRight
                  class="ml-auto h-3.5 w-3.5 transition-transform duration-150 group-data-[state=open]/collapsible:rotate-90 text-muted-foreground/60"
                />
              </SidebarMenuButton>
            </CollapsibleTrigger>
            <CollapsibleContent>
              <SidebarMenuSub class="ml-6 border-l border-border pl-3 space-y-0.5">
                <SidebarMenuSubItem
                  v-for="subItem in item.items"
                  :key="subItem.title"
                >
                  <SidebarMenuSubButton as-child class="rounded">
                    <RouterLink
                      :to="subItem.url"
                      class="flex items-center gap-2 text-sm text-muted-foreground transition-colors"
                      active-class="!text-primary font-medium"
                    >
                      <component
                        :is="subItem.icon"
                        v-if="subItem.icon"
                        class="h-3.5 w-3.5"
                      />
                      <span>{{ subItem.title }}</span>
                    </RouterLink>
                  </SidebarMenuSubButton>
                </SidebarMenuSubItem>
              </SidebarMenuSub>
            </CollapsibleContent>
          </SidebarMenuItem>
        </Collapsible>

        <SidebarMenuItem v-else>
          <SidebarMenuButton as-child :tooltip="item.title" class="rounded-md">
            <RouterLink
              :to="item.url!"
              class="flex items-center gap-2 text-muted-foreground transition-colors"
              active-class="!text-primary !font-medium bg-primary/8"
              exact-active-class="!text-primary !font-medium bg-primary/8"
              exact
            >
              <component :is="item.icon" v-if="item.icon" class="h-4 w-4 shrink-0" />
              <span>{{ item.title }}</span>
            </RouterLink>
          </SidebarMenuButton>
        </SidebarMenuItem>
      </template>
    </SidebarMenu>
  </SidebarGroup>
</template>
