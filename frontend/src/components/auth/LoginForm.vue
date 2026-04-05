<script setup lang="ts">
import { ref, onMounted } from "vue";
import { useAuthStore } from "@/stores/auth";
import { backend } from "@/../wailsjs/go/models";
import { toast } from "vue-sonner";
import { useRoute } from "vue-router";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import { Alert, AlertDescription } from "@/components/ui/alert";
import {
  AlertCircle,
  Loader2,
  ShieldCheck,
  Eye,
  EyeOff,
  Settings,
} from "lucide-vue-next";
import { IsSetupMode } from "@/../wailsjs/go/backend/Db";

const authStore = useAuthStore();
const route = useRoute();
const isSetupMode = ref(false);

const credentials = ref<backend.LoginRequest>(new backend.LoginRequest());
const mfaCode = ref<string>("");

const isLoading = ref<boolean>(false);
const error = ref<string>("");
const loginStep = ref<number>(1);
const showPassword = ref<boolean>(false);

onMounted(async () => {
  if (route.query.registered === "true") {
    toast.success("¡Cuenta creada con éxito! Ahora puedes iniciar sesión.");
  }
  try {
    isSetupMode.value = await IsSetupMode();
  } catch {
    isSetupMode.value = false;
  }
});

async function handleLogin() {
  isLoading.value = true;
  error.value = "";
  try {
    const requiresMFA = await authStore.login(credentials.value);
    if (requiresMFA) loginStep.value = 2;
  } catch (err: any) {
    error.value = err.message || "Credenciales incorrectas.";
  } finally {
    isLoading.value = false;
  }
}

async function handleVerifyMFA() {
  if (mfaCode.value.length !== 6) {
    error.value = "El código debe tener 6 dígitos.";
    return;
  }
  isLoading.value = true;
  error.value = "";
  try {
    await authStore.verifyMfaAndFinishLogin(mfaCode.value);
  } catch (err: any) {
    error.value = err.message || "Código MFA incorrecto o expirado.";
  } finally {
    isLoading.value = false;
  }
}
</script>

<template>
  <div class="auth-root">

    <!-- ── LEFT BRAND PANEL ── -->
    <div class="brand-panel">
      <!-- Architectural grid pattern -->
      <div class="brand-grid" aria-hidden="true"></div>
      <!-- Floating cross accent -->
      <div class="brand-cross-bg" aria-hidden="true">
        <svg viewBox="0 0 120 120" fill="none" xmlns="http://www.w3.org/2000/svg">
          <rect x="48" y="8" width="24" height="104" rx="4" fill="currentColor"/>
          <rect x="8" y="48" width="104" height="24" rx="4" fill="currentColor"/>
        </svg>
      </div>

      <div class="brand-content">
        <!-- Logo mark -->
        <div class="brand-logo-row">
          <div class="brand-icon-wrap">
            <svg viewBox="0 0 32 32" fill="none" class="brand-icon-svg">
              <rect x="12" y="2" width="8" height="28" rx="2" fill="white"/>
              <rect x="2" y="12" width="28" height="8" rx="2" fill="white"/>
            </svg>
          </div>
          <div class="brand-name-block">
            <span class="brand-name">Droguería Luna</span>
            <span class="brand-sub">Sistema farmacéutico</span>
          </div>
        </div>

        <!-- Divider -->
        <div class="brand-divider"></div>

        <!-- Feature list -->
        <ul class="brand-features">
          <li v-for="f in ['Control de inventario en tiempo real', 'Punto de venta integrado', 'Facturación electrónica DIAN']" :key="f">
            <span class="feature-dot"></span>
            <span>{{ f }}</span>
          </li>
        </ul>
      </div>

      <p class="brand-copy">© {{ new Date().getFullYear() }} Droguería Luna</p>
    </div>

    <!-- ── RIGHT FORM PANEL ── -->
    <div class="form-panel">

      <!-- Setup mode banner -->
      <div v-if="isSetupMode" class="setup-banner">
        <Settings class="w-3.5 h-3.5 shrink-0" />
        <span>Modo configuración — usa cualquier email con contraseña <strong>admin</strong></span>
      </div>

      <Transition
        enter-active-class="transition-all duration-350 ease-out"
        enter-from-class="opacity-0 translate-y-3"
        enter-to-class="opacity-100 translate-y-0"
        leave-active-class="transition-all duration-200 ease-in"
        leave-from-class="opacity-100 translate-y-0"
        leave-to-class="opacity-0 -translate-y-3"
        mode="out-in"
      >

        <!-- STEP 1: Credentials -->
        <form v-if="loginStep === 1" key="credentials" @submit.prevent="handleLogin" class="auth-form">

          <div class="form-header">
            <h2 class="form-title">Bienvenido de nuevo</h2>
            <p class="form-subtitle">Ingresa tus credenciales para continuar</p>
          </div>

          <div class="fields-stack">
            <!-- Email -->
            <div class="field-group">
              <Label for="email" class="field-label">Correo electrónico</Label>
              <Input
                id="email"
                v-model="credentials.Email"
                type="email"
                placeholder="usuario@droguerialuna.com"
                required
                class="field-input"
              />
            </div>

            <!-- Password -->
            <div class="field-group">
              <Label for="password" class="field-label">Contraseña</Label>
              <div class="relative">
                <Input
                  id="password"
                  v-model="credentials.Contrasena"
                  :type="showPassword ? 'text' : 'password'"
                  placeholder="••••••••"
                  required
                  class="field-input pr-11"
                />
                <button
                  type="button"
                  @click="showPassword = !showPassword"
                  class="eye-toggle"
                  tabindex="-1"
                  :aria-label="showPassword ? 'Ocultar contraseña' : 'Mostrar contraseña'"
                >
                  <EyeOff v-if="showPassword" class="w-4 h-4" />
                  <Eye v-else class="w-4 h-4" />
                </button>
              </div>
            </div>

            <!-- Error -->
            <Alert v-if="error" class="error-alert">
              <AlertCircle class="w-3.5 h-3.5 shrink-0 mt-px" />
              <AlertDescription class="text-xs">{{ error }}</AlertDescription>
            </Alert>

            <!-- Submit -->
            <Button type="submit" class="submit-btn" :disabled="isLoading">
              <Loader2 v-if="isLoading" class="mr-2 h-4 w-4 animate-spin" />
              {{ isLoading ? "Ingresando…" : "Iniciar Sesión" }}
            </Button>

            <p class="form-footer-text">
              ¿No tienes una cuenta?
              <router-link to="/register" class="form-link">Regístrate</router-link>
            </p>
          </div>
        </form>

        <!-- STEP 2: MFA -->
        <form v-else key="mfa" @submit.prevent="handleVerifyMFA" class="auth-form">

          <div class="mfa-header">
            <div class="mfa-icon-wrap">
              <ShieldCheck class="w-6 h-6 text-white" />
            </div>
            <h2 class="form-title">Verificación en dos pasos</h2>
            <p class="form-subtitle">Ingresa el código de 6 dígitos de tu aplicación de autenticación</p>
          </div>

          <div class="fields-stack">
            <div class="field-group">
              <Label for="mfa-code" class="field-label">Código de autenticación</Label>
              <Input
                id="mfa-code"
                v-model="mfaCode"
                class="field-input text-center text-2xl tracking-[0.75em] font-mono h-14"
                maxlength="6"
                autocomplete="one-time-code"
                placeholder="——————"
                required
              />
            </div>

            <Alert v-if="error" class="error-alert">
              <AlertCircle class="w-3.5 h-3.5 shrink-0 mt-px" />
              <AlertDescription class="text-xs">{{ error }}</AlertDescription>
            </Alert>

            <Button type="submit" class="submit-btn" :disabled="isLoading">
              <Loader2 v-if="isLoading" class="mr-2 h-4 w-4 animate-spin" />
              {{ isLoading ? "Verificando…" : "Verificar Código" }}
            </Button>

            <button
              type="button"
              @click="loginStep = 1; error = ''"
              class="form-footer-text underline underline-offset-4 hover:text-foreground transition-colors text-center"
            >
              Volver al inicio de sesión
            </button>
          </div>
        </form>

      </Transition>
    </div>

  </div>
</template>

<style scoped>
/* ── Fonts ── */
@import url('https://fonts.googleapis.com/css2?family=Fraunces:ital,opsz,wght@0,9..144,300;0,9..144,600;1,9..144,300&family=Plus+Jakarta+Sans:wght@400;500;600&display=swap');

/* ── Root layout ── */
.auth-root {
  display: flex;
  height: 100vh;
  width: 100vw;
  overflow: hidden;
  font-family: 'Plus Jakarta Sans', sans-serif;
}

/* ── Mobile (≤ 640 px) ── */
@media (max-width: 640px) {
  .auth-root {
    flex-direction: column;
    height: auto;
    min-height: 100dvh;
    overflow-y: auto;
    overflow-x: hidden;
  }

  .brand-panel {
    width: 100%;
    min-height: auto;
    flex-direction: row;
    align-items: center;
    padding: 1.1rem 1.25rem;
    animation: none;
  }

  .brand-content { flex-direction: row; align-items: center; gap: 0; }
  .brand-logo-row { animation: none; }
  .brand-divider,
  .brand-features,
  .brand-copy,
  .brand-cross-bg { display: none; }

  .form-panel {
    flex: 1;
    padding: 2rem 1.25rem 3rem;
    justify-content: flex-start;
    animation: none;
  }

  /* On mobile the banner stacks inline instead of floating at the top */
  .setup-banner {
    position: relative;
    top: auto; left: auto; right: auto;
    border-radius: 8px;
    border: 1px solid #fde68a;
    margin-bottom: 1.25rem;
  }

  .auth-form {
    max-width: 100%;
    animation: none;
  }
}

/* ── Small tablet (641 px – 900 px): narrow the brand panel ── */
@media (min-width: 641px) and (max-width: 900px) {
  .brand-panel { width: 36%; padding: 2.5rem 2rem; }
  .form-panel  { padding: 2rem 2.5rem; }
}

/* ── Brand panel ── */
.brand-panel {
  position: relative;
  width: 42%;
  flex-shrink: 0;
  background: linear-gradient(155deg, #0a1628 0%, #0d1f3c 35%, #091729 65%, #060e1c 100%);
  display: flex;
  flex-direction: column;
  justify-content: space-between;
  padding: 3.5rem 3rem;
  overflow: hidden;
  color: white;
  animation: panelIn 0.7s cubic-bezier(0.22, 1, 0.36, 1) both;
}

@keyframes panelIn {
  from { opacity: 0; transform: translateX(-24px); }
  to   { opacity: 1; transform: translateX(0); }
}

/* Grid pattern */
.brand-grid {
  position: absolute;
  inset: 0;
  background-image:
    linear-gradient(rgba(99,179,237,0.06) 1px, transparent 1px),
    linear-gradient(90deg, rgba(99,179,237,0.06) 1px, transparent 1px);
  background-size: 40px 40px;
  mask-image: radial-gradient(ellipse 80% 80% at 50% 50%, black 30%, transparent 100%);
}

/* Large background cross */
.brand-cross-bg {
  position: absolute;
  bottom: -60px;
  right: -60px;
  width: 340px;
  height: 340px;
  color: rgba(147, 197, 253, 0.05);
  animation: floatCross 8s ease-in-out infinite;
}
@keyframes floatCross {
  0%, 100% { transform: translateY(0) rotate(0deg); }
  50%       { transform: translateY(-12px) rotate(3deg); }
}

.brand-content {
  display: flex;
  flex-direction: column;
  gap: 2.5rem;
  position: relative;
  z-index: 1;
}

.brand-logo-row {
  display: flex;
  align-items: center;
  gap: 1rem;
  animation: fadeUp 0.6s 0.2s cubic-bezier(0.22, 1, 0.36, 1) both;
}

.brand-icon-wrap {
  width: 52px;
  height: 52px;
  border-radius: 14px;
  background: linear-gradient(135deg, rgba(96,165,250,0.25), rgba(99,102,241,0.15));
  border: 1px solid rgba(147, 197, 253, 0.2);
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.brand-icon-svg {
  width: 22px;
  height: 22px;
}

.brand-name-block {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.brand-name {
  font-family: 'Fraunces', Georgia, serif;
  font-size: 1.35rem;
  font-weight: 600;
  letter-spacing: -0.02em;
  color: #fff;
  line-height: 1.2;
}

.brand-sub {
  font-size: 0.7rem;
  font-weight: 500;
  letter-spacing: 0.12em;
  text-transform: uppercase;
  color: rgba(147,197,253,0.7);
}

.brand-divider {
  width: 40px;
  height: 2px;
  background: linear-gradient(90deg, rgba(96,165,250,0.6), transparent);
  animation: fadeUp 0.6s 0.3s cubic-bezier(0.22, 1, 0.36, 1) both;
}

.brand-features {
  list-style: none;
  padding: 0;
  margin: 0;
  display: flex;
  flex-direction: column;
  gap: 1.1rem;
  animation: fadeUp 0.6s 0.4s cubic-bezier(0.22, 1, 0.36, 1) both;
}

.brand-features li {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  font-size: 0.85rem;
  color: rgba(226, 232, 240, 0.8);
  font-weight: 400;
}

.feature-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: #60a5fa;
  flex-shrink: 0;
  box-shadow: 0 0 8px rgba(96, 165, 250, 0.5);
}

.brand-copy {
  position: relative;
  z-index: 1;
  font-size: 0.7rem;
  color: rgba(147,197,253,0.35);
  letter-spacing: 0.04em;
  animation: fadeUp 0.6s 0.5s cubic-bezier(0.22, 1, 0.36, 1) both;
}

/* ── Form panel ── */
.form-panel {
  flex: 1;
  display: flex;
  flex-direction: column;
  justify-content: center;
  align-items: center;
  background: #fafaf8;
  padding: 3rem 4rem;
  overflow-y: auto;
  position: relative;
  animation: formPanelIn 0.6s 0.1s cubic-bezier(0.22, 1, 0.36, 1) both;
}

@keyframes formPanelIn {
  from { opacity: 0; }
  to   { opacity: 1; }
}

/* Setup banner */
.setup-banner {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.6rem 1.5rem;
  font-size: 0.72rem;
  background: #fffbeb;
  color: #92400e;
  border-bottom: 1px solid #fde68a;
}

/* Auth form */
.auth-form {
  width: 100%;
  max-width: 360px;
  animation: fadeUp 0.5s 0.25s cubic-bezier(0.22, 1, 0.36, 1) both;
}

@keyframes fadeUp {
  from { opacity: 0; transform: translateY(12px); }
  to   { opacity: 1; transform: translateY(0); }
}

.form-header {
  margin-bottom: 2.25rem;
}

.form-title {
  font-family: 'Fraunces', Georgia, serif;
  font-size: 1.9rem;
  font-weight: 600;
  color: #0f172a;
  letter-spacing: -0.03em;
  line-height: 1.15;
  margin-bottom: 0.4rem;
}

.form-subtitle {
  font-size: 0.84rem;
  color: #64748b;
  font-weight: 400;
  line-height: 1.5;
}

.fields-stack {
  display: flex;
  flex-direction: column;
  gap: 1.1rem;
}

.field-group {
  display: flex;
  flex-direction: column;
  gap: 0.45rem;
}

.field-label {
  font-size: 0.78rem;
  font-weight: 600;
  color: #334155;
  letter-spacing: 0.01em;
}

.field-input {
  height: 2.75rem !important;
  background: white;
  border-color: #e2e8f0;
  font-size: 0.875rem;
  font-family: 'Plus Jakarta Sans', sans-serif;
  color: #0f172a;
  transition: border-color 0.2s, box-shadow 0.2s;
  border-radius: 8px;
}

.field-input:focus {
  border-color: #3b82f6;
  box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.1);
}

.eye-toggle {
  position: absolute;
  top: 0;
  bottom: 0;
  right: 0;
  display: flex;
  align-items: center;
  padding-right: 0.75rem;
  color: #94a3b8;
  transition: color 0.15s;
  background: none;
  border: none;
  cursor: pointer;
}
.eye-toggle:hover { color: #475569; }

.error-alert {
  display: flex;
  align-items: flex-start;
  gap: 0.5rem;
  padding: 0.65rem 0.85rem;
  border-radius: 8px;
  background: #fef2f2;
  border: 1px solid #fecaca;
  color: #dc2626;
}

.submit-btn {
  width: 100%;
  height: 2.75rem;
  font-family: 'Plus Jakarta Sans', sans-serif;
  font-weight: 600;
  font-size: 0.875rem;
  letter-spacing: 0.01em;
  border-radius: 8px;
  margin-top: 0.25rem;
  background: #1e3a5f;
  color: white;
  transition: background 0.2s, transform 0.1s, box-shadow 0.2s;
  box-shadow: 0 2px 8px rgba(30, 58, 95, 0.3);
}

.submit-btn:hover:not(:disabled) {
  background: #152d4a;
  box-shadow: 0 4px 16px rgba(30, 58, 95, 0.35);
}

.submit-btn:active:not(:disabled) {
  transform: translateY(1px);
}

.form-footer-text {
  text-align: center;
  font-size: 0.8rem;
  color: #94a3b8;
  margin-top: 0.25rem;
}

.form-link {
  color: #1e3a5f;
  font-weight: 600;
  text-decoration: underline;
  text-underline-offset: 3px;
  transition: color 0.15s;
}
.form-link:hover { color: #3b82f6; }

/* MFA header */
.mfa-header {
  display: flex;
  flex-direction: column;
  align-items: center;
  text-align: center;
  margin-bottom: 2rem;
  gap: 0.75rem;
}

.mfa-icon-wrap {
  width: 56px;
  height: 56px;
  border-radius: 16px;
  background: linear-gradient(135deg, #1e3a5f, #1d4ed8);
  display: flex;
  align-items: center;
  justify-content: center;
  box-shadow: 0 4px 16px rgba(30, 58, 95, 0.35);
  margin-bottom: 0.25rem;
}
</style>
