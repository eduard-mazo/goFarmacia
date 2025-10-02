<script setup lang="ts">
import { ref, watch } from "vue";
import {
  ChevronsUpDown,
  LogOut,
  Settings,
  Loader2,
  Shield,
} from "lucide-vue-next";
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
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
import { ActualizarPerfilVendedor } from "@/../wailsjs/go/backend/Db";
import { backend } from "@/../wailsjs/go/models";
import { toast } from "vue-sonner";
import MFASetup from "@/components/auth/MFASetup.vue";

const authStore = useAuthStore();
const userAvatar = "https://avatar.iran.liara.run/public/boy";
const { user: authenticatedUser, userInitials } = storeToRefs(authStore);
const { isMobile } = useSidebar();

const isSettingsDialogOpen = ref(false);
const isMFADialogOpen = ref(false);
const editableUser = ref<backend.Vendedor>(new backend.Vendedor());

editableUser.value.Nombre = "";
editableUser.value.Apellido = "";
editableUser.value.Cedula = "";
editableUser.value.Email = "";

const currentPassword = ref("");
const newPassword = ref("");
const confirmPassword = ref("");
const isSaving = ref(false);

watch(isSettingsDialogOpen, (isOpen) => {
  if (isOpen && authenticatedUser.value) {
    editableUser.value = Object.assign(
      new backend.Vendedor(),
      authenticatedUser.value
    );
    currentPassword.value = "";
    newPassword.value = "";
    confirmPassword.value = "";
  } else {
    editableUser.value = new backend.Vendedor();
  }
});

async function handleSaveChanges() {
  if (!editableUser.value) return;

  isSaving.value = true;
  const request = new backend.VendedorUpdateRequest();
  request.ID = editableUser.value.id;
  request.Nombre = editableUser.value.Nombre;
  request.Apellido = editableUser.value.Apellido;
  request.Cedula = editableUser.value.Cedula;
  request.Email = editableUser.value.Email;

  if (newPassword.value || currentPassword.value) {
    if (!currentPassword.value) {
      toast.error(
        "Para cambiar la contraseña, debes ingresar tu contraseña actual."
      );
      isSaving.value = false;
      return;
    }
    if (newPassword.value !== confirmPassword.value) {
      toast.error("La nueva contraseña y su confirmación no coinciden.");
      isSaving.value = false;
      return;
    }
    if (newPassword.value.length < 6) {
      toast.error("La nueva contraseña debe tener al menos 6 caracteres.");
      isSaving.value = false;
      return;
    }
    request.ContrasenaActual = currentPassword.value;
    request.ContrasenaNueva = newPassword.value;
  }

  try {
    await ActualizarPerfilVendedor(request);
    toast.success("Perfil actualizado correctamente.");
    authStore.updateUser({
      Nombre: editableUser.value.Nombre,
      Apellido: editableUser.value.Apellido,
      Cedula: editableUser.value.Cedula,
      Email: editableUser.value.Email,
    });
    isSettingsDialogOpen.value = false;
  } catch (error) {
    toast.error("Error al actualizar el perfil", { description: `${error}` });
  } finally {
    isSaving.value = false;
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
            <DropdownMenuItem @click="isSettingsDialogOpen = true">
              <Settings class="mr-2 size-4" />
              <span>Ajustes de Perfil</span>
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

      <Dialog v-model:open="isSettingsDialogOpen">
        <DialogContent class="w-11/12 md:max-w-[700px]">
          <DialogHeader>
            <DialogTitle>Editar Perfil</DialogTitle>
            <DialogDescription>
              Realiza cambios a tu información personal aquí.
            </DialogDescription>
          </DialogHeader>
          <div class="space-y-6 py-4">
            <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
              <div class="grid gap-2">
                <Label for="nombre">Nombre</Label>
                <Input id="nombre" v-model="editableUser.Nombre" />
              </div>
              <div class="grid gap-2">
                <Label for="apellido">Apellido</Label>
                <Input id="apellido" v-model="editableUser.Apellido" />
              </div>
              <div class="grid gap-2">
                <Label for="cedula">Cédula</Label>
                <Input id="cedula" v-model="editableUser.Cedula" />
              </div>
              <div class="grid gap-2">
                <Label for="email">Email</Label>
                <Input id="email" v-model="editableUser.Email" type="email" />
              </div>
            </div>
            <div class="space-y-4 border-t pt-4">
              <p class="text-sm text-muted-foreground text-center">
                Para cambiar tu contraseña, completa los siguientes campos.
              </p>
              <div class="grid grid-cols-1 sm:grid-cols-3 gap-4">
                <div class="grid gap-2">
                  <Label for="currentPassword">Contraseña Actual</Label>
                  <Input
                    id="currentPassword"
                    v-model="currentPassword"
                    type="password"
                  />
                </div>
                <div class="grid gap-2">
                  <Label for="newPassword">Nueva Contraseña</Label>
                  <Input
                    id="newPassword"
                    v-model="newPassword"
                    type="password"
                  />
                </div>
                <div class="grid gap-2">
                  <Label for="confirmPassword">Confirmar Contraseña</Label>
                  <Input
                    id="confirmPassword"
                    v-model="confirmPassword"
                    type="password"
                  />
                </div>
              </div>
            </div>
          </div>
          <DialogFooter>
            <Button
              type="button"
              @click="handleSaveChanges"
              :disabled="isSaving"
            >
              <Loader2 v-if="isSaving" class="mr-2 h-4 w-4 animate-spin" />
              {{ isSaving ? "Guardando..." : "Guardar cambios" }}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

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
