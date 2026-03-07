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
  return item.items.some((subItem) => route.path.startsWith(subItem.url));
};

function handleMenuClick() {
  if (state.value === "collapsed") setOpen(true);
}
</script>

<template>
  <SidebarGroup class="py-1">
    <SidebarMenu class="gap-0.5">
      <template v-for="item in items" :key="item.title">

        <!-- Collapsible group -->
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
                class="h-8 rounded-md gap-2 px-2 transition-colors"
                :class="isParentActive(item)
                  ? 'text-foreground font-medium'
                  : 'text-muted-foreground hover:text-foreground hover:bg-accent'"
              >
                <component
                  :is="item.icon"
                  v-if="item.icon"
                  class="h-4 w-4 shrink-0"
                  :class="isParentActive(item) ? 'text-primary' : ''"
                />
                <span class="text-sm">{{ item.title }}</span>
                <ChevronRight
                  class="ml-auto h-3.5 w-3.5 transition-transform duration-200 ease-in-out group-data-[state=open]/collapsible:rotate-90"
                  :class="isParentActive(item) ? 'text-primary/60' : 'text-muted-foreground/40'"
                />
              </SidebarMenuButton>
            </CollapsibleTrigger>
            <CollapsibleContent>
              <SidebarMenuSub class="mx-3 my-0.5 border-l border-border/60 pl-3 gap-0.5">
                <SidebarMenuSubItem
                  v-for="subItem in item.items"
                  :key="subItem.title"
                >
                  <SidebarMenuSubButton as-child class="h-7 rounded-md">
                    <RouterLink
                      :to="subItem.url"
                      class="flex items-center gap-2 text-[13px] text-muted-foreground transition-colors px-2"
                      active-class="!text-foreground font-medium !bg-accent"
                    >
                      <component
                        :is="subItem.icon"
                        v-if="subItem.icon"
                        class="h-3.5 w-3.5 shrink-0"
                      />
                      <span class="truncate">{{ subItem.title }}</span>
                    </RouterLink>
                  </SidebarMenuSubButton>
                </SidebarMenuSubItem>
              </SidebarMenuSub>
            </CollapsibleContent>
          </SidebarMenuItem>
        </Collapsible>

        <!-- Direct link -->
        <SidebarMenuItem v-else>
          <SidebarMenuButton
            as-child
            :tooltip="item.title"
            class="h-8 rounded-md gap-2 px-2 transition-colors"
          >
            <RouterLink
              :to="item.url!"
              class="flex items-center gap-2 text-sm text-muted-foreground transition-colors"
              active-class="!text-foreground !font-medium !bg-accent"
              exact-active-class="!text-foreground !font-medium !bg-accent"
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
