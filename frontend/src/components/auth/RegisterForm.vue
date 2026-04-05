<script setup lang="ts">
import { ref } from "vue";
import { useRouter } from "vue-router";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { AlertCircle, Loader2, Eye, EyeOff } from "lucide-vue-next";
import { useAuthStore } from "@/stores/auth";
import { backend } from "@/../wailsjs/go/models";

const registerPayload = ref<backend.Vendedor>(new backend.Vendedor());
const error = ref<string | null>(null);
const isLoading = ref(false);
const showPassword = ref<boolean>(false);
const router = useRouter();
const authStore = useAuthStore();

const handleRegister = async () => {
  isLoading.value = true;
  error.value = "";
  try {
    await authStore.register(registerPayload.value);
    router.push({ path: "/login", query: { registered: "true" } });
  } catch (err: any) {
    error.value = err.message || "Ocurrió un error durante el registro.";
  } finally {
    isLoading.value = false;
  }
};
</script>

<template>
  <div class="auth-root">

    <!-- ── LEFT BRAND PANEL ── -->
    <div class="brand-panel">
      <div class="brand-grid" aria-hidden="true"></div>
      <div class="brand-cross-bg" aria-hidden="true">
        <svg viewBox="0 0 120 120" fill="none" xmlns="http://www.w3.org/2000/svg">
          <rect x="48" y="8" width="24" height="104" rx="4" fill="currentColor"/>
          <rect x="8" y="48" width="104" height="24" rx="4" fill="currentColor"/>
        </svg>
      </div>

      <div class="brand-content">
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

        <div class="brand-divider"></div>

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
      <form @submit.prevent="handleRegister" class="auth-form">

        <div class="form-header">
          <h2 class="form-title">Crea tu cuenta</h2>
          <p class="form-subtitle">Completa el formulario para unirte al sistema</p>
        </div>

        <div class="fields-stack">

          <!-- Nombre + Apellido -->
          <div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
            <div class="field-group">
              <Label for="nombre" class="field-label">Nombres</Label>
              <Input id="nombre" v-model="registerPayload.Nombre" type="text"
                placeholder="Juan" required class="field-input" />
            </div>
            <div class="field-group">
              <Label for="apellido" class="field-label">Apellidos</Label>
              <Input id="apellido" v-model="registerPayload.Apellido" type="text"
                placeholder="Rodríguez" required class="field-input" />
            </div>
          </div>

          <!-- Email + Cédula -->
          <div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
            <div class="field-group">
              <Label for="email" class="field-label">Correo</Label>
              <Input id="email" v-model="registerPayload.Email" type="email"
                placeholder="usuario@ejemplo.com" required class="field-input" />
            </div>
            <div class="field-group">
              <Label for="cedula" class="field-label">Cédula</Label>
              <Input id="cedula" v-model="registerPayload.Cedula" type="text"
                placeholder="1020430991" required class="field-input" />
            </div>
          </div>

          <!-- Contraseña -->
          <div class="field-group">
            <Label for="password" class="field-label">Contraseña</Label>
            <div class="relative">
              <Input
                id="password"
                v-model="registerPayload.Contrasena"
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
            {{ isLoading ? "Creando cuenta…" : "Crear Cuenta" }}
          </Button>

          <p class="form-footer-text">
            ¿Ya tienes una cuenta?
            <router-link to="/login" class="form-link">Inicia sesión</router-link>
          </p>

        </div>
      </form>
    </div>

  </div>
</template>

<style scoped>
@import url('https://fonts.googleapis.com/css2?family=Fraunces:opsz,wght@9..144,300;9..144,600&family=Plus+Jakarta+Sans:wght@400;500;600&display=swap');

.auth-root {
  display: flex;
  height: 100vh;
  width: 100vw;
  overflow: hidden;
  font-family: 'Plus Jakarta Sans', sans-serif;
}

/* ── Brand panel (identical to Login) ── */
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

.brand-grid {
  position: absolute;
  inset: 0;
  background-image:
    linear-gradient(rgba(99,179,237,0.06) 1px, transparent 1px),
    linear-gradient(90deg, rgba(99,179,237,0.06) 1px, transparent 1px);
  background-size: 40px 40px;
  mask-image: radial-gradient(ellipse 80% 80% at 50% 50%, black 30%, transparent 100%);
}

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

.brand-icon-svg { width: 22px; height: 22px; }

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
  padding: 2.5rem 3.5rem;
  overflow-y: auto;
  animation: formPanelIn 0.6s 0.1s cubic-bezier(0.22, 1, 0.36, 1) both;
}

@keyframes formPanelIn {
  from { opacity: 0; }
  to   { opacity: 1; }
}

.auth-form {
  width: 100%;
  max-width: 400px;
  animation: fadeUp 0.5s 0.25s cubic-bezier(0.22, 1, 0.36, 1) both;
}

@keyframes fadeUp {
  from { opacity: 0; transform: translateY(12px); }
  to   { opacity: 1; transform: translateY(0); }
}

.form-header {
  margin-bottom: 2rem;
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
  line-height: 1.5;
}

.fields-stack {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.field-group {
  display: flex;
  flex-direction: column;
  gap: 0.4rem;
}

.field-label {
  font-size: 0.78rem;
  font-weight: 600;
  color: #334155;
  letter-spacing: 0.01em;
}

.field-input {
  height: 2.6rem !important;
  background: white;
  border-color: #e2e8f0;
  font-size: 0.85rem;
  font-family: 'Plus Jakarta Sans', sans-serif;
  color: #0f172a;
  border-radius: 8px;
  transition: border-color 0.2s, box-shadow 0.2s;
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

/* ── Mobile (≤ 640 px): hide brand panel, show only the form ── */
@media (max-width: 640px) {
  .brand-panel { display: none; }
  .form-panel  { padding: 3rem 1.5rem 3.5rem; }
  .auth-form   { max-width: 100%; }
}

/* ── Small tablet (641–900 px): narrow the brand panel ── */
@media (min-width: 641px) and (max-width: 900px) {
  .brand-panel { width: 34%; padding: 2.5rem 2rem; }
  .form-panel  { padding: 2rem 2.5rem; }
}
</style>
