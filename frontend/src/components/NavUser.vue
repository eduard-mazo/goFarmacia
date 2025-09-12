<script setup lang="ts">
import { ref, watch } from "vue";
import { ChevronsUpDown, LogOut, Settings } from "lucide-vue-next";
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
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

const authStore = useAuthStore();
const userAvatar = "https://avatar.iran.liara.run/public/boy";
const { user: authenticatedUser, userInitials } = storeToRefs(authStore);
const { isMobile } = useSidebar();

// --- Lógica para el diálogo de ajustes ---

// Controla la visibilidad del diálogo
const isSettingsDialogOpen = ref(false);
// Almacena el nombre que se está editando
const editableName = ref("");

// Observa el estado del diálogo. Cuando se abre, carga el nombre del usuario.
watch(isSettingsDialogOpen, (isOpen) => {
  if (isOpen) {
    editableName.value = authenticatedUser.value?.Nombre || "";
  }
});

// Función para guardar los cambios del nombre
function handleSaveChanges() {
  if (editableName.value && editableName.value.trim() !== "") {
    // Aquí llamarías a una acción en tu store para actualizar el nombre del usuario
    // Ejemplo: authStore.updateUserName(editableName.value);
    console.log(`Guardando nuevo nombre: ${editableName.value}`);

    // Cierra el diálogo después de guardar
    isSettingsDialogOpen.value = false;
  }
}

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
            <Dialog v-model:open="isSettingsDialogOpen">
              <DialogTrigger as-child>
                <DropdownMenuItem @select.prevent>
                  <Settings class="mr-2 size-4" />
                  <span>Ajustes</span>
                </DropdownMenuItem>
              </DialogTrigger>
              <DialogContent class="sm:max-w-[425px]">
                <DialogHeader>
                  <DialogTitle>Editar nombre de usuario</DialogTitle>
                  <DialogDescription>
                    Realiza cambios a tu nombre aquí. Haz clic en guardar cuando
                    termines.
                  </DialogDescription>
                </DialogHeader>
                <div class="grid gap-4 py-4">
                  <div class="grid grid-cols-4 items-center gap-4">
                    <Label for="name" class="text-right"> Nombre </Label>
                    <Input
                      id="name"
                      v-model="editableName"
                      class="col-span-3"
                    />
                  </div>
                </div>
                <DialogFooter>
                  <Button type="button" @click="handleSaveChanges">
                    Guardar cambios
                  </Button>
                </DialogFooter>
              </DialogContent>
            </Dialog>
          </DropdownMenuGroup>
          <DropdownMenuSeparator />
          <DropdownMenuItem @click="handleLogOut">
            <LogOut class="mr-2 size-4" />
            Cerrar Sesión
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>
    </SidebarMenuItem>
  </SidebarMenu>
</template>
