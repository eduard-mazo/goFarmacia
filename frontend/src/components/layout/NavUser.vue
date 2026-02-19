<script setup lang="ts">
import { ref } from "vue";
import { useRouter } from "vue-router";
import {
  ChevronsUpDown,
  LogOut,
  User,
  Shield,
} from "lucide-vue-next";
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import {
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  useSidebar,
} from "@/components/ui/sidebar";
import { storeToRefs } from "pinia";
import { useAuthStore } from "@/stores/auth";
import { toast } from "vue-sonner";
import MFASetup from "@/components/auth/MFASetup.vue";

const authStore = useAuthStore();
const router = useRouter();
const { user: authenticatedUser, userInitials } = storeToRefs(authStore);
const { isMobile } = useSidebar();

const isMFADialogOpen = ref(false);

function handleLogOut() {
  authStore.logout();
}
</script>

<template>
  <SidebarMenu>
    <SidebarMenuItem>
      <DropdownMenu>
        <DropdownMenuTrigger as-child>
          <SidebarMenuButton
            size="lg"
            class="data-[state=open]:bg-sidebar-accent data-[state=open]:text-sidebar-accent-foreground"
          >
            <Avatar class="h-8 w-8 rounded-lg">
              <AvatarImage :src="userAvatar" :alt="authenticatedUser?.Nombre" />
              <AvatarFallback class="rounded-lg">
                {{ userInitials }}
              </AvatarFallback>
            </Avatar>
            <div class="grid flex-1 text-left text-sm leading-tight">
              <span class="truncate font-semibold">{{
                authenticatedUser?.Nombre
              }}</span>
              <span class="truncate text-xs">{{
                authenticatedUser?.Email
              }}</span>
            </div>
            <ChevronsUpDown class="ml-auto size-4" />
          </SidebarMenuButton>
        </DropdownMenuTrigger>
        <DropdownMenuContent
          class="w-[--reka-dropdown-menu-trigger-width] min-w-56 rounded-lg"
          :side="isMobile ? 'bottom' : 'right'"
          align="end"
          :side-offset="4"
        >
          <DropdownMenuLabel class="p-0 font-normal">
            <div class="flex items-center gap-2 px-1 py-1.5 text-left text-sm">
              <Avatar class="h-8 w-8 rounded-lg">
                <AvatarImage
                  :src="userAvatar"
                  :alt="authenticatedUser?.Nombre"
                />
                <AvatarFallback class="rounded-lg">
                  {{ userInitials }}
                </AvatarFallback>
              </Avatar>
              <div class="grid flex-1 text-left text-sm leading-tight">
                <span class="truncate font-semibold">{{
                  authenticatedUser?.Nombre
                }}</span>
                <span class="truncate text-xs">{{
                  authenticatedUser?.Email
                }}</span>
              </div>
            </div>
          </DropdownMenuLabel>
          <DropdownMenuSeparator />
          <DropdownMenuGroup>
            <DropdownMenuItem @click="router.push('/dashboard/perfil')">
              <User class="mr-2 size-4" />
              <span>Mi Perfil</span>
            </DropdownMenuItem>

            <DropdownMenuItem @click="isMFADialogOpen = true">
              <Shield class="mr-2 size-4" />
              <span>Seguridad (2FA)</span>
            </DropdownMenuItem>
          </DropdownMenuGroup>
          <DropdownMenuSeparator />
          <DropdownMenuItem @click="handleLogOut">
            <LogOut class="mr-2 size-4" />
            Cerrar Sesión
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>

      <Dialog v-model:open="isMFADialogOpen">
        <DialogContent class="w-11/12 md:max-w-md">
          <DialogHeader>
            <DialogTitle
              >Configurar Autenticación de Dos Factores (2FA)</DialogTitle
            >
            <DialogDescription>
              Añade una capa extra de seguridad a tu cuenta usando una app como
              Google Authenticator.
            </DialogDescription>
          </DialogHeader>
          <MFASetup />
        </DialogContent>
      </Dialog>
    </SidebarMenuItem>
  </SidebarMenu>
</template>
