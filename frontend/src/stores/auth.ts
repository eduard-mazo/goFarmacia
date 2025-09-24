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

  const userInitials = computed(() => {
    if (!user.value) return "";
    const nombre = user.value.Nombre?.trim() || "";
    const apellido = user.value.Apellido?.trim() || "";
    const inicialNombre = nombre.charAt(0).toUpperCase();
    const inicialApellido = apellido.charAt(0).toUpperCase();
    return `${inicialNombre}${inicialApellido}`.trim();
  });

  // --- ACTIONS ---


  function setAuth(data: backend.LoginResponse) {
    user.value = data.vendedor;
    token.value = data.token;
    localStorage.setItem("authToken", data.token);
    localStorage.setItem("authUser", JSON.stringify(data.vendedor));
  }

  function clearAuth() {
    user.value = null;
    token.value = null;
    localStorage.removeItem("authToken");
    localStorage.removeItem("authUser");
  }

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

  async function login(email: string, contrasena: string) {
    try {
      const loginRequest: backend.LoginRequest = {
        Email: email,
        Contrasena: contrasena,
      };
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

  function logout() {
    clearAuth();
    router.push("/login");
  }

  function tryAutoLogin() {
    const storedToken = localStorage.getItem("authToken");
    const storedUser = localStorage.getItem("authUser");
    if (storedToken && storedUser) {
      token.value = storedToken;
      user.value = JSON.parse(storedUser);
    }
  }

  // NUEVA ACCIÓN: Para actualizar los datos del usuario en el estado
  function updateUser(updatedUserData: Partial<backend.Vendedor>) {
    if (user.value) {
      // Fusionamos los datos nuevos con los existentes
      user.value = { ...user.value, ...updatedUserData };
      // Actualizamos también el localStorage para persistencia
      localStorage.setItem("authUser", JSON.stringify(user.value));
    }
  }

  return {
    user,
    token,
    isAuthenticated,
    currentUser,
    userInitials,
    register,
    login,
    logout,
    tryAutoLogin,
    updateUser,
  };
});
