<script setup lang="ts">
import { ref, watch } from "vue";
import { storeToRefs } from "pinia";
import { useAuthStore } from "@/stores/auth";
import { ActualizarPerfilVendedor } from "@/../bridge/go/backend/Db";
import { backend } from "@/../bridge/go/models";
import { toast } from "vue-sonner";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
  CardDescription,
} from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Separator } from "@/components/ui/separator";
import { Badge } from "@/components/ui/badge";
import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import MFASetup from "@/components/auth/MFASetup.vue";
import {
  User,
  Mail,
  CreditCard,
  Lock,
  Shield,
  Save,
  Loader2,
  Eye,
  EyeOff,
} from "lucide-vue-next";

const authStore = useAuthStore();
const { currentUser, userInitials } = storeToRefs(authStore);

// ── Personal info ──────────────────────────────────────────
const editableUser = ref(new backend.Vendedor());
const isSavingInfo = ref(false);

function loadUser() {
  if (currentUser.value) {
    editableUser.value = Object.assign(new backend.Vendedor(), currentUser.value);
  }
}
loadUser();
watch(currentUser, loadUser);

async function savePersonalInfo() {
  isSavingInfo.value = true;
  try {
    const request = new backend.VendedorUpdateRequest();
    request.UUID = editableUser.value.UUID;
    request.Nombre = editableUser.value.Nombre;
    request.Apellido = editableUser.value.Apellido;
    request.Cedula = editableUser.value.Cedula;
    request.Email = editableUser.value.Email;
    await ActualizarPerfilVendedor(request);
    authStore.updateUser({
      Nombre: editableUser.value.Nombre,
      Apellido: editableUser.value.Apellido,
      Cedula: editableUser.value.Cedula,
      Email: editableUser.value.Email,
    });
    toast.success("Información actualizada correctamente.");
  } catch (err) {
    toast.error("Error al actualizar la información", { description: `${err}` });
  } finally {
    isSavingInfo.value = false;
  }
}

// ── Password change ────────────────────────────────────────
const currentPassword = ref("");
const newPassword = ref("");
const confirmPassword = ref("");
const showCurrent = ref(false);
const showNew = ref(false);
const showConfirm = ref(false);
const isSavingPassword = ref(false);

async function savePassword() {
  if (!currentPassword.value) {
    toast.error("Ingresa tu contraseña actual.");
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
  isSavingPassword.value = true;
  try {
    const request = new backend.VendedorUpdateRequest();
    request.UUID = editableUser.value.UUID;
    request.Nombre = editableUser.value.Nombre;
    request.Apellido = editableUser.value.Apellido;
    request.Cedula = editableUser.value.Cedula;
    request.Email = editableUser.value.Email;
    request.ContrasenaActual = currentPassword.value;
    request.ContrasenaNueva = newPassword.value;
    await ActualizarPerfilVendedor(request);
    toast.success("Contraseña actualizada correctamente.");
    currentPassword.value = "";
    newPassword.value = "";
    confirmPassword.value = "";
  } catch (err) {
    toast.error("Error al cambiar la contraseña", { description: `${err}` });
  } finally {
    isSavingPassword.value = false;
  }
}
</script>

<template>
  <div class="p-6 max-w-4xl space-y-6">
    <!-- Header -->
    <div>
      <h1 class="text-2xl font-semibold tracking-tight">Mi Perfil</h1>
      <p class="text-sm text-muted-foreground mt-0.5">
        Gestiona tu información personal y seguridad
      </p>
    </div>

    <!-- Profile identity card -->
    <Card class="">
      <CardContent class="pt-6">
        <div class="flex items-center gap-5">
          <Avatar class="h-16 w-16 rounded-xl text-xl">
            <AvatarFallback class="rounded-xl bg-primary text-primary-foreground font-semibold text-lg">
              {{ userInitials }}
            </AvatarFallback>
          </Avatar>
          <div class="space-y-1">
            <p class="text-lg font-semibold leading-none">
              {{ currentUser?.Nombre }} {{ currentUser?.Apellido }}
            </p>
            <p class="text-sm text-muted-foreground">{{ currentUser?.Email }}</p>
            <Badge
              :class="currentUser?.Role === 'admin'
                ? 'bg-blue-100 text-blue-800 border-blue-200'
                : 'bg-slate-100 text-slate-700 border-slate-200'"
              class="mt-1 text-xs font-medium border capitalize"
            >
              {{ currentUser?.Role === 'admin' ? 'Administrador' : 'Cajero' }}
            </Badge>
          </div>
        </div>
      </CardContent>
    </Card>

    <div class="grid grid-cols-1 lg:grid-cols-2 gap-6">
      <!-- Personal info -->
      <Card class="">
        <CardHeader class="pb-3">
          <CardTitle class="text-base font-semibold flex items-center gap-2">
            <User class="h-4 w-4 text-muted-foreground" />
            Información Personal
          </CardTitle>
          <CardDescription>Actualiza tu nombre, email y cédula</CardDescription>
        </CardHeader>
        <CardContent class="space-y-4">
          <div class="grid grid-cols-2 gap-3">
            <div class="space-y-1.5">
              <Label for="pNombre">Nombre</Label>
              <Input id="pNombre" v-model="editableUser.Nombre" class="h-9" />
            </div>
            <div class="space-y-1.5">
              <Label for="pApellido">Apellido</Label>
              <Input id="pApellido" v-model="editableUser.Apellido" class="h-9" />
            </div>
          </div>
          <div class="space-y-1.5">
            <Label for="pEmail">
              <Mail class="inline h-3.5 w-3.5 mr-1 text-muted-foreground" />
              Email
            </Label>
            <Input id="pEmail" v-model="editableUser.Email" type="email" class="h-9" />
          </div>
          <div class="space-y-1.5">
            <Label for="pCedula">
              <CreditCard class="inline h-3.5 w-3.5 mr-1 text-muted-foreground" />
              Cédula
            </Label>
            <Input id="pCedula" v-model="editableUser.Cedula" class="h-9" />
          </div>
          <Button @click="savePersonalInfo" :disabled="isSavingInfo" class="w-full h-9 gap-2">
            <Loader2 v-if="isSavingInfo" class="h-4 w-4 animate-spin" />
            <Save v-else class="h-4 w-4" />
            {{ isSavingInfo ? "Guardando..." : "Guardar cambios" }}
          </Button>
        </CardContent>
      </Card>

      <!-- Password change -->
      <Card class="">
        <CardHeader class="pb-3">
          <CardTitle class="text-base font-semibold flex items-center gap-2">
            <Lock class="h-4 w-4 text-muted-foreground" />
            Cambiar Contraseña
          </CardTitle>
          <CardDescription>Usa al menos 6 caracteres</CardDescription>
        </CardHeader>
        <CardContent class="space-y-4">
          <div class="space-y-1.5">
            <Label for="pCurPass">Contraseña actual</Label>
            <div class="relative">
              <Input
                id="pCurPass"
                v-model="currentPassword"
                :type="showCurrent ? 'text' : 'password'"
                class="h-9 pr-9"
              />
              <button
                type="button"
                class="absolute right-2 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground"
                @click="showCurrent = !showCurrent"
              >
                <Eye v-if="!showCurrent" class="h-4 w-4" />
                <EyeOff v-else class="h-4 w-4" />
              </button>
            </div>
          </div>
          <div class="space-y-1.5">
            <Label for="pNewPass">Nueva contraseña</Label>
            <div class="relative">
              <Input
                id="pNewPass"
                v-model="newPassword"
                :type="showNew ? 'text' : 'password'"
                class="h-9 pr-9"
              />
              <button
                type="button"
                class="absolute right-2 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground"
                @click="showNew = !showNew"
              >
                <Eye v-if="!showNew" class="h-4 w-4" />
                <EyeOff v-else class="h-4 w-4" />
              </button>
            </div>
          </div>
          <div class="space-y-1.5">
            <Label for="pConfPass">Confirmar contraseña</Label>
            <div class="relative">
              <Input
                id="pConfPass"
                v-model="confirmPassword"
                :type="showConfirm ? 'text' : 'password'"
                class="h-9 pr-9"
              />
              <button
                type="button"
                class="absolute right-2 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground"
                @click="showConfirm = !showConfirm"
              >
                <Eye v-if="!showConfirm" class="h-4 w-4" />
                <EyeOff v-else class="h-4 w-4" />
              </button>
            </div>
          </div>
          <Button @click="savePassword" :disabled="isSavingPassword" class="w-full h-9 gap-2">
            <Loader2 v-if="isSavingPassword" class="h-4 w-4 animate-spin" />
            <Lock v-else class="h-4 w-4" />
            {{ isSavingPassword ? "Actualizando..." : "Cambiar contraseña" }}
          </Button>
        </CardContent>
      </Card>
    </div>

    <!-- 2FA section -->
    <Card class="">
      <CardHeader class="pb-3">
        <CardTitle class="text-base font-semibold flex items-center gap-2">
          <Shield class="h-4 w-4 text-muted-foreground" />
          Autenticación de Dos Factores (2FA)
        </CardTitle>
        <CardDescription>
          Añade una capa extra de seguridad usando Google Authenticator u otra app TOTP
        </CardDescription>
      </CardHeader>
      <Separator />
      <CardContent class="pt-4">
        <MFASetup />
      </CardContent>
    </Card>
  </div>
</template>
