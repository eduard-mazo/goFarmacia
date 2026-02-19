Plan to implement                                                                                                         │
│                                                                                                                           │
│ Plan: Modern Login & Register UI Redesign                                                                                 │
│                                                                                                                           │
│ Context                                                                                                                   │
│                                                                                                                           │
│ The current login/register forms are functional but visually basic — generic 2-column cards with a side image,            │
│ placeholder English text ("Welcome back", "Acme Inc"), and no polish. The goal is a professional, modern design befitting │
│  a pharmacy POS desktop app (Wails), using the already-installed stack: shadcn-vue + TailwindCSS v4 + Lucide icons +      │
│ tw-animate-css.                                                                                                           │
│                                                                                                                           │
│ ---                                                                                                                       │
│ Design Direction: Full-Screen Split Layout                                                                                │
│                                                                                                                           │
│ ┌──────────────────────────────────────────────────────────┐                                                              │
│ │  LEFT BRAND PANEL (dark blue-slate gradient, ~40% width) │                                                              │
│ │                                                          │                                                              │
│ │   🏪 [Building2 icon — large, white]                     │                                                              │
│ │                                                          │                                                              │
│ │   Droguería Luna                                         │                                                              │
│ │   Sistema de gestión farmacéutica                        │                                                              │
│ │                                                          │                                                              │
│ │   ✓ Control de inventario                                │                                                              │
│ │   ✓ Punto de venta integrado                             │                                                              │
│ │   ✓ Reportes en tiempo real                              │                                                              │
│ │                                                          │                                                              │
│ └──────────────────────────────────────────────────────────┘                                                              │
│ │  RIGHT FORM PANEL (white/bg-background, ~60% width)      │                                                              │
│ │                                                          │                                                              │
│ │   Bienvenido de nuevo      [small logo repeat optional]  │                                                              │
│ │   Ingresa tus credenciales para acceder                  │                                                              │
│ │                                                          │                                                              │
│ │   Email ____________________                             │                                                              │
│ │   Contraseña ______________ 👁                           │                                                               │
│ │   [Error alert]                                          │                                                              │
│ │   [Iniciar Sesión button — full width, primary]          │                                                              │
│ │   ¿No tienes cuenta? → Regístrate                        │                                                              │
│ │                                                          │                                                              │
│ └──────────────────────────────────────────────────────────┘                                                              │
│                                                                                                                           │
│                                                                                                                           │
│ MFA step slides/fades in on the right panel with a ShieldCheck icon and digit input.                                      │
│                                                                                                                           │
│ Register mirrors the same split, form panel has 2-column grid (Nombre/Apellido, Email/Cedula, Password full-width).       │
│                                                                                                                           │
│ ---                                                                                                                       │
│ Files to Modify                                                                                                           │
│                                                                                                                           │
│ File: frontend/src/views/Login.vue                                                                                        │
│ Change: Replace generic wrapper with full-screen flex container                                                           │
│ ────────────────────────────────────────                                                                                  │
│ File: frontend/src/views/Register.vue                                                                                     │
│ Change: Same as Login wrapper                                                                                             │
│ ────────────────────────────────────────                                                                                  │
│ File: frontend/src/components/auth/LoginForm.vue                                                                          │
│ Change: Full redesign — split panel, password toggle, animated step transition                                            │
│ ────────────────────────────────────────                                                                                  │
│ File: frontend/src/components/auth/RegisterForm.vue                                                                       │
│ Change: Full redesign — fix placeholder text, add Loader2, cleaner layout                                                 │
│                                                                                                                           │
│ No new files are needed. No router or store changes.                                                                      │
│                                                                                                                           │
│ ---                                                                                                                       │
│ Implementation Details                                                                                                    │
│                                                                                                                           │
│ 1. Login.vue & Register.vue                                                                                               │
│                                                                                                                           │
│ - Change wrapper to <div class="h-screen w-screen overflow-hidden"> so the split layout fills the full Wails window.      │
│ - Remove max-w-sm md:max-w-3xl constraint.                                                                                │
│                                                                                                                           │
│ 2. LoginForm.vue — redesign                                                                                               │
│                                                                                                                           │
│ Structure: <div class="flex h-screen"> → left panel + right panel                                                         │
│                                                                                                                           │
│ Left panel (w-2/5 bg-gradient-to-br from-slate-900 via-blue-950 to-indigo-900 text-white flex flex-col justify-center     │
│ px-12):                                                                                                                   │
│ - Building2 icon (lucide, 56px, white, slight opacity bg circle)                                                          │
│ - <h1> "Droguería Luna" large bold                                                                                        │
│ - <p> "Sistema integral de gestión farmacéutica"                                                                          │
│ - Feature list with CheckCircle2 icons                                                                                    │
│                                                                                                                           │
│ Right panel (w-3/5 flex flex-col justify-center px-16 bg-background):                                                     │
│ - Step 1 (credentials): form with email + password (+ eye icon toggle using ref<boolean> showPassword)                    │
│ - Step 2 (MFA): centered with ShieldCheck icon, clean OTP input                                                           │
│ - v-show / Transition with animate-in fade-in (from tw-animate-css) between steps                                         │
│ - Error displayed inline with Alert destructive variant                                                                   │
│ - Footer link to Register                                                                                                 │
│                                                                                                                           │
│ Password toggle: <button type="button" @click="showPassword = !showPassword"> with Eye/EyeOff icons absolutely positioned │
│  inside the input wrapper.                                                                                                │
│                                                                                                                           │
│ 3. RegisterForm.vue — redesign                                                                                            │
│                                                                                                                           │
│ Same left panel as Login (reuse the exact same markup for brand consistency).                                             │
│                                                                                                                           │
│ Right panel form:                                                                                                         │
│ - Header: "Crea tu cuenta" / "Completa el formulario para unirte"                                                         │
│ - 2-col grid: Nombre | Apellido                                                                                           │
│ - 2-col grid: Email | Cédula                                                                                              │
│ - Full-width: Contraseña (with show/hide toggle)                                                                          │
│ - Full-width: Button with Loader2 spinner                                                                                 │
│ - Footer link back to Login                                                                                               │
│                                                                                                                           │
│ Fix all placeholder text (remove "Welcome back", "Acme Inc").                                                             │
│                                                                                                                           │
│ ---                                                                                                                       │
│ Lucide Icons to Use                                                                                                       │
│                                                                                                                           │
│ - Building2 — brand icon                                                                                                  │
│ - CheckCircle2 — feature list bullets                                                                                     │
│ - ShieldCheck — MFA step                                                                                                  │
│ - Eye / EyeOff — password toggle                                                                                          │
│ - Loader2 — loading spinner (already used)                                                                                │
│ - AlertCircle — error alert (already used)                                                                                │
│ - Mail, Lock — optional field icons                                                                                       │
│                                                                                                                           │
│ ---                                                                                                                       │
│ Verification                                                                                                              │
│                                                                                                                           │
│ 1. Run wails dev -tags webkit2_41 — app window should show full-screen split on login                                     │
│ 2. Enter wrong credentials → red Alert appears inline                                                                     │
│ 3. Enter correct credentials with MFA → step 2 slides in with shield icon                                                 │
│ 4. Navigate to /Register → same brand panel, registration form                                                            │
│ 5. Register → redirects to login with success toast                                                                       │
│ 6. No TypeScript errors in console                                                                                        │
╰───────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────╯
