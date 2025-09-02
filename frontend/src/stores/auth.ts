import { defineStore } from "pinia";
import { ref, computed } from "vue";
import { LoginVendedor, RegistrarVendedor } from "../../wailsjs/go/backend/Db";
import type { backend } from "../../wailsjs/go/models";
import { useRouter } from "vue-router";

export const useAuthStore = defineStore("auth", () => {
  const router = useRouter();

  // --- STATE ---
  const user = ref<backend.Vendedor | null>(null);
  const token = ref<string | null>(localStorage.getItem("authToken"));

  // --- GETTERS ---
  const isAuthenticated = computed(() => !!token.value && !!user.value);
  const currentUser = computed(() => user.value);

  // --- ACTIONS ---

  // Función para establecer el estado de autenticación
  function setAuth(data: backend.LoginResponse) {
    user.value = data.vendedor;
    token.value = data.token;
    localStorage.setItem("authToken", data.token);
    // Puedes también guardar el usuario, pero el token es lo esencial.
    localStorage.setItem("authUser", JSON.stringify(data.vendedor));
  }

  // Función para limpiar el estado de autenticación
  function clearAuth() {
    user.value = null;
    token.value = null;
    localStorage.removeItem("authToken");
    localStorage.removeItem("authUser");
  }

  // Acción para registrar un nuevo vendedor
  async function register(vendedorData: backend.Vendedor) {
    try {
      // La contraseña ya está en vendedorData.Contrasena
      await RegistrarVendedor(vendedorData);
    } catch (error) {
      console.error("Error en el registro:", error);
      // Re-lanzamos el error para que la vista pueda capturarlo y mostrarlo.
      throw new Error((error as string) || "Error desconocido en el registro");
    }
  }

  // Acción para iniciar sesión
  async function login(email: string, contrasena: string) {
    try {
      const loginRequest: backend.LoginRequest = { Email: email, Contrasena: contrasena };
      const response = await LoginVendedor(loginRequest);
      if (response.token && response.vendedor) {
        setAuth(response);
      } else {
        throw new Error("Respuesta de login inválida");
      }
    } catch (error) {
      console.error("Error en el login:", error);
      clearAuth();
      throw new Error((error as string) || "Error desconocido en el login");
    }
  }

  // Acción para cerrar sesión
  function logout() {
    clearAuth();
    // Redirigir al login
    router.push("/login");
  }

  // Acción para intentar auto-login al cargar la app
  function tryAutoLogin() {
    const storedToken = localStorage.getItem("authToken");
    const storedUser = localStorage.getItem("authUser");
    if (storedToken && storedUser) {
      token.value = storedToken;
      user.value = JSON.parse(storedUser);
    }
  }

  return {
    user,
    token,
    isAuthenticated,
    currentUser,
    register,
    login,
    logout,
    tryAutoLogin,
  };
});
