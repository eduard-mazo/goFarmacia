import { defineStore } from "pinia";
import { ref, computed } from "vue";
import type { backend } from "@/../wailsjs/go/models";
import { useRouter } from "vue-router";
import {
  LoginVendedor,
  VerificarLoginMFA,
  RegistrarVendedor,
} from "@/../wailsjs/go/backend/Db";

export const useAuthStore = defineStore("auth", () => {
  const router = useRouter();

  // --- STATE ---
  const user = ref<backend.Vendedor | null>(null);
  const token = ref<string | null>(localStorage.getItem("authToken"));
  const tempMFAToken = ref<string | null>(null);

  // --- GETTERS ---
  const isAuthenticated = computed(() => !!token.value && !!user.value);
  const currentUser = computed(() => user.value);

  const userInitials = computed(() => {
    if (!user.value) return "";
    const nombre = user.value.Nombre?.trim() || "";
    const apellido = user.value.Apellido?.trim() || "";
    const inicialNombre = nombre.charAt(0).toUpperCase();
    const inicialApellido = apellido.charAt(0).toUpperCase();
    return `${inicialNombre}${inicialApellido}`.trim();
  });

  // --- ACTIONS ---

  // Inicia el proceso de login, puede requerir un segundo paso
  async function login(credentials: { Email: string; Contrasena: string }) {
    const response = await LoginVendedor(credentials);

    if (response.mfa_required) {
      tempMFAToken.value = response.token; // Guarda el token temporal
      return true; // Indica al componente que se requiere MFA
    } else {
      setAuthenticated(response.vendedor, response.token);
      router.push("/dashboard");
      return false; // Indica al componente que el login está completo
    }
  }

  // Completa el login después de una verificación MFA exitosa
  async function verifyMfaAndFinishLogin(mfaCode: string) {
    if (!tempMFAToken.value) {
      throw new Error(
        "No se encontró un token de MFA temporal. Por favor, inicie sesión de nuevo."
      );
    }
    const finalToken = await VerificarLoginMFA(tempMFAToken.value, mfaCode);

    // Para obtener los datos del usuario, podríamos decodificar el nuevo token o hacer otra llamada.
    // Por simplicidad, asumimos que el usuario que inició el MFA es el correcto.
    // Una implementación más robusta podría hacer que VerificarLoginMFA devuelva el objeto Vendedor.
    tryAutoLogin(finalToken); // Re-usa tryAutoLogin para popular el estado con el nuevo token final
    router.push("/dashboard");
  }

  function setAuthenticated(
    vendedorData: backend.Vendedor,
    finalToken: string
  ) {
    user.value = vendedorData;
    token.value = finalToken;
    localStorage.setItem("authToken", finalToken);
    localStorage.setItem("authUser", JSON.stringify(vendedorData));
  }

  function clearAuth() {
    user.value = null;
    token.value = null;
    tempMFAToken.value = null;
    localStorage.removeItem("authToken");
    localStorage.removeItem("authUser");
  }

  function logout() {
    clearAuth();
    router.push("/login");
  }

  function tryAutoLogin(tokenOverride?: string) {
    const storedToken = tokenOverride || localStorage.getItem("authToken");
    const storedUser = localStorage.getItem("authUser");
    if (storedToken && storedUser) {
      token.value = storedToken;
      try {
        user.value = JSON.parse(storedUser);
        if (tokenOverride) {
          localStorage.setItem("authToken", tokenOverride);
        }
      } catch (e) {
        console.error("Failed to parse user from localStorage", e);
        clearAuth();
      }
    }
  }

  async function register(vendedorData: backend.Vendedor) {
    try {
      return await RegistrarVendedor(vendedorData);
    } catch (error: any) {
      throw new Error(error || "Error desconocido durante el registro.");
    }
  }

  function updateUser(updatedUserData: Partial<backend.Vendedor>) {
    if (user.value) {
      user.value = { ...user.value, ...updatedUserData };
      localStorage.setItem("authUser", JSON.stringify(user.value));
    }
  }

  return {
    user,
    token,
    isAuthenticated,
    currentUser,
    userInitials,
    login,
    verifyMfaAndFinishLogin,
    register,
    setAuthenticated,
    logout,
    tryAutoLogin,
    updateUser,
  };
});
