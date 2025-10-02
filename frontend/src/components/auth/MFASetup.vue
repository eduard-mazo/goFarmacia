<script setup lang="ts">
import { ref } from "vue";
import { useAuthStore } from "@/stores/auth";
import { GenerarMFA, HabilitarMFA } from "@/../wailsjs/go/backend/Db";
import { toast } from "vue-sonner";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Loader2, ShieldCheck } from "lucide-vue-next";
import { storeToRefs } from "pinia";

const authStore = useAuthStore();
const { user: currentUser } = storeToRefs(authStore);

// --- Estados del componente ---
const isLoading = ref(false); // Para la carga inicial del QR
const isVerifying = ref(false); // Para la verificación del código
const error = ref("");
const qrCodeImage = ref(""); // Almacenará la imagen base64 del QR
const secretKey = ref(""); // Para mostrar como alternativa al QR
const verificationCode = ref("");

// --- Lógica del componente ---

// Se llama cuando el usuario decide iniciar la configuración de 2FA
async function handleGenerateMFA() {
  isLoading.value = true;
  error.value = "";
  try {
    if (!currentUser.value?.Email) {
      throw new Error(
        "No se pudo obtener el email del usuario. Inicia sesión de nuevo."
      );
    }
    const response = await GenerarMFA(currentUser.value.Email);
    qrCodeImage.value = response.image_url;
    secretKey.value = response.secret;
  } catch (err: any) {
    error.value = `Error al generar el código MFA: ${err.message || err}`;
  } finally {
    isLoading.value = false;
  }
}

// Se llama para verificar el código y completar la activación
async function verifyAndEnableMFA() {
  if (verificationCode.value.length !== 6) {
    error.value = "El código de verificación debe tener 6 dígitos.";
    return;
  }
  isVerifying.value = true;
  error.value = "";
  try {
    if (!currentUser.value?.Email)
      throw new Error("Sesión de usuario no encontrada.");

    const success = await HabilitarMFA(
      currentUser.value.Email,
      verificationCode.value
    );

    if (success) {
      toast.success("¡Autenticación de Dos Factores habilitada correctamente!");
      authStore.updateUser({ mfa_enabled: true }); // Actualiza el estado global
    }
  } catch (err: any) {
    error.value = `Error al verificar el código: ${err.message || err}`;
    toast.error("Error al verificar el código", { description: `${err}` });
  } finally {
    isVerifying.value = false;
  }
}
</script>

<template>
  <div class="py-4">
    <div
      v-if="currentUser?.mfa_enabled"
      class="flex flex-col items-center justify-center h-full p-6 border rounded-lg bg-secondary/50"
    >
      <ShieldCheck class="w-16 h-16 text-green-500 mb-4" />
      <h3 class="text-lg font-semibold">2FA está Activo</h3>
      <p class="text-sm text-muted-foreground text-center mt-2">
        La autenticación de dos factores ya está protegiendo tu cuenta.
      </p>
    </div>

    <div v-else-if="!qrCodeImage" class="text-center space-y-4">
      <p class="text-sm text-muted-foreground">
        Protege tu cuenta contra accesos no autorizados. Al activarlo,
        necesitarás un código de tu app de autenticación para iniciar sesión.
      </p>
      <Button @click="handleGenerateMFA" :disabled="isLoading">
        <Loader2 v-if="isLoading" class="mr-2 h-4 w-4 animate-spin" />
        {{ isLoading ? "Generando..." : "Activar 2FA" }}
      </Button>
    </div>

    <div v-else class="space-y-6">
      <Alert v-if="error" variant="destructive">
        <AlertTitle>Error</AlertTitle>
        <AlertDescription>{{ error }}</AlertDescription>
      </Alert>

      <div class="flex flex-col items-center gap-4">
        <p class="text-sm font-semibold">1. Escanea el código QR</p>
        <div class="border rounded-lg p-2 bg-white">
          <img :src="qrCodeImage" alt="Código QR para 2FA" />
        </div>
        <div class="text-center">
          <p class="text-xs text-muted-foreground">
            ¿No puedes escanear? Ingresa esta clave manualmente:
          </p>
          <p
            class="text-sm font-mono bg-secondary p-2 mt-1 rounded-md break-all"
          >
            {{ secretKey }}
          </p>
        </div>
      </div>

      <form @submit.prevent="verifyAndEnableMFA" class="space-y-4">
        <div class="space-y-2">
          <Label for="mfa-code" class="font-semibold"
            >2. Ingresa el código de verificación</Label
          >
          <Input
            id="mfa-code"
            v-model="verificationCode"
            type="text"
            placeholder="123456"
            maxlength="6"
            required
            autocomplete="off"
            class="text-center text-lg tracking-[0.5em]"
          />
        </div>
        <Button type="submit" class="w-full" :disabled="isVerifying">
          <Loader2 v-if="isVerifying" class="mr-2 h-4 w-4 animate-spin" />
          {{ isVerifying ? "Verificando..." : "Habilitar y Verificar" }}
        </Button>
      </form>
    </div>
  </div>
</template>
