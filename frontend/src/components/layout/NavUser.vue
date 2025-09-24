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
// CAMBIO: Asumiremos que crearemos una nueva función en el backend más segura
import { ActualizarPerfilVendedor } from "@/../wailsjs/go/backend/Db";
import { backend } from "@/../wailsjs/go/models";
import { toast } from "vue-sonner";

const authStore = useAuthStore();
const userAvatar = "https://avatar.iran.liara.run/public/boy";
const { user: authenticatedUser, userInitials } = storeToRefs(authStore);
const { isMobile } = useSidebar();

const isSettingsDialogOpen = ref(false);
const editableUser = ref(new backend.Vendedor());

// NUEVO: Refs para el manejo de contraseñas
const currentPassword = ref("");
const newPassword = ref("");
const confirmPassword = ref("");

watch(isSettingsDialogOpen, (isOpen) => {
  if (isOpen && authenticatedUser.value) {
    editableUser.value = { ...authenticatedUser.value };
    currentPassword.value = "";
    newPassword.value = "";
    confirmPassword.value = "";
  }
});

async function handleSaveChanges() {
  if (!editableUser.value) return;

  // Objeto para la solicitud, basado en un nuevo modelo que crearemos en Go
  const request = new backend.VendedorUpdateRequest();
  request.ID = editableUser.value.id;
  request.Nombre = editableUser.value.Nombre;
  request.Apellido = editableUser.value.Apellido;
  request.Cedula = editableUser.value.Cedula;
  request.Email = editableUser.value.Email;

  // --- Lógica de validación de contraseña ---
  if (newPassword.value || currentPassword.value) {
    if (!currentPassword.value) {
      toast.error(
        "Para cambiar la contraseña, debes ingresar tu contraseña actual."
      );
      return;
    }
    if (newPassword.value !== confirmPassword.value) {
      toast.error("La nueva contraseña y su confirmación no coinciden.");
      return;
    }
    if (newPassword.value.length < 6) {
      toast.error("La nueva contraseña debe tener al menos 6 caracteres.");
      return;
    }
    // Si todo es correcto, añadimos las contraseñas a la solicitud
    request.ContrasenaActual = currentPassword.value;
    request.ContrasenaNueva = newPassword.value;
  }

  try {
    // Llamamos a la nueva función segura del backend
    await ActualizarPerfilVendedor(request);
    toast.success("Perfil actualizado correctamente.");

    // Actualizamos el store localmente
    authStore.updateUser(editableUser.value);

    isSettingsDialogOpen.value = false;
  } catch (error) {
    toast.error("Error al actualizar el perfil", { description: `${error}` });
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
              <DialogContent class="w-11/12 md:max-w-[700px]">
                <DialogHeader>
                  <DialogTitle>Editar Perfil</DialogTitle>
                  <DialogDescription>
                    Realiza cambios a tu perfil aquí. Haz clic en guardar cuando
                    termines.
                  </DialogDescription>
                </DialogHeader>
                <div class="grid gap-4 py-4">
                  <div class="grid grid-cols-4 items-center gap-4">
                    <Label for="nombre" class="text-right">Nombre</Label>
                    <Input
                      id="nombre"
                      v-model="editableUser.Nombre"
                      class="col-span-3"
                    />
                  </div>
                  <div class="grid grid-cols-4 items-center gap-4">
                    <Label for="apellido" class="text-right">Apellido</Label>
                    <Input
                      id="apellido"
                      v-model="editableUser.Apellido"
                      class="col-span-3"
                    />
                  </div>
                  <div class="grid grid-cols-4 items-center gap-4">
                    <Label for="cedula" class="text-right">Cédula</Label>
                    <Input
                      id="cedula"
                      v-model="editableUser.Cedula"
                      class="col-span-3"
                    />
                  </div>
                  <div class="grid grid-cols-4 items-center gap-4">
                    <Label for="email" class="text-right">Email</Label>
                    <Input
                      id="email"
                      v-model="editableUser.Email"
                      class="col-span-3"
                    />
                  </div>
                  <hr class="my-2" />
                  <p
                    class="text-sm text-muted-foreground text-center col-span-full"
                  >
                    Para cambiar tu contraseña, completa los siguientes tres
                    campos.
                  </p>
                  <div class="grid grid-cols-4 items-center gap-4">
                    <Label for="currentPassword" class="text-right"
                      >Contraseña Actual</Label
                    >
                    <Input
                      id="currentPassword"
                      v-model="currentPassword"
                      type="password"
                      class="col-span-3"
                    />
                  </div>
                  <div class="grid grid-cols-4 items-center gap-4">
                    <Label for="newPassword" class="text-right"
                      >Nueva Contraseña</Label
                    >
                    <Input
                      id="newPassword"
                      v-model="newPassword"
                      type="password"
                      class="col-span-3"
                    />
                  </div>
                  <div class="grid grid-cols-4 items-center gap-4">
                    <Label for="confirmPassword" class="text-right"
                      >Confirmar Contraseña</Label
                    >
                    <Input
                      id="confirmPassword"
                      v-model="confirmPassword"
                      type="password"
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
