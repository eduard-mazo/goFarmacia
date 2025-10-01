<script setup>
import { ref, onMounted } from "vue";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { toast } from "vue-sonner";

// Importa las funciones del backend generadas por Wails
import { GenerarMFA, HabilitarMFA } from "@/../wailsjs/go/backend/Db";

// Asumimos un store de Pinia para obtener el email del usuario actual
import { useAuthStore } from "@/stores/auth";

const userStore = useAuthStore();
const emit = defineEmits(["mfa-enabled"]);

// --- Estados del componente ---
const isLoading = ref(true);
const isVerifying = ref(false);
const error = ref("");
const qrCodeImage = ref("");
const secretKey = ref("");
const verificationCode = ref("");

// --- Lógica del componente ---

onMounted(async () => {
  try {
    if (!userStore.currentUser?.Email) {
      throw new Error(
        "No se pudo obtener el email del usuario. Inicia sesión de nuevo."
      );
    }
    const response = await GenerarMFA(userStore.currentUser.Email);
    qrCodeImage.value = response.image_url;
    secretKey.value = response.secret;
  } catch (err) {
    error.value = `Error al generar el código MFA: ${err}`;
  } finally {
    isLoading.value = false;
  }
});

const verifyAndEnableMFA = async () => {
  if (verificationCode.value.length !== 6) {
    error.value = "El código debe tener 6 dígitos.";
    return;
  }
  isVerifying.value = true;
  error.value = "";
  try {
    const success = await HabilitarMFA(
      userStore.currentUser.Email,
      verificationCode.value
    );
    if (success) {
      toast.success("¡Autenticación de Dos Factores habilitada correctamente!");
      // --- INICIO DE LA MODIFICACIÓN: Lógica de estado correcta ---
      userStore.updateUser({ mfa_enabled: true });
      emit("mfa-enabled"); // Notificar al componente padre
      // --- FIN DE LA MODIFICACIÓN ---
    }
  } catch (err) {
    error.value = `Error al verificar el código: ${err}`;
    toast.error("Error al verificar el código", { description: `${err}` });
  } finally {
    isVerifying.value = false;
  }
};
</script>

<template>
  <Card class="w-full max-w-md border-0 shadow-none">
    <CardHeader>
      <CardTitle class="text-xl font-bold text-center"
        >Configurar 2FA</CardTitle
      >
      <CardDescription class="text-center text-xs">
        Escanea el QR con tu app de autenticación (e.g., Google Authenticator).
      </CardDescription>
    </CardHeader>

    <CardContent class="space-y-6">
      <Alert v-if="error" variant="destructive">
        <AlertTitle>Error</AlertTitle>
        <AlertDescription>{{ error }}</AlertDescription>
      </Alert>

      <div v-if="isLoading" class="text-center text-sm text-muted-foreground">
        Cargando QR...
      </div>

      <div v-if="qrCodeImage" class="flex flex-col items-center gap-4">
        <img
          :src="qrCodeImage"
          alt="Código QR para MFA"
          class="border rounded-lg p-2 bg-white"
        />
        <div class="text-center">
          <p class="text-xs text-muted-foreground">¿No puedes escanear?</p>
          <p
            class="text-sm font-mono bg-secondary p-2 mt-1 rounded-md break-all"
          >
            {{ secretKey }}
          </p>
        </div>
      </div>

      <form
        v-if="qrCodeImage"
        @submit.prevent="verifyAndEnableMFA"
        class="space-y-4"
      >
        <div class="space-y-2">
          <Label for="mfa-code">Código de Verificación</Label>
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
          {{ isVerifying ? "Verificando..." : "Habilitar y Verificar" }}
        </Button>
      </form>
    </CardContent>
  </Card>
</template>
