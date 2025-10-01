import { defineStore } from "pinia";
import { ref, computed } from "vue";
import type { backend } from "@/../wailsjs/go/models";
import { useRouter } from "vue-router";
import { RegistrarVendedor } from "@/../wailsjs/go/backend/Db";

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
    localStorage.removeItem("authToken");
    localStorage.removeItem("authUser");
    router.push("/login");
  }

  async function register(vendedorData: backend.Vendedor) {
    try {
      const newUser = await RegistrarVendedor(vendedorData);
      return newUser;
    } catch (error: any) {
      throw new Error(error || "Error desconocido durante el registro.");
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
      try {
        user.value = JSON.parse(storedUser);
      } catch (e) {
        console.error("Failed to parse user from localStorage", e);
        clearAuth();
      }
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
    register,
    setAuthenticated,
    logout,
    tryAutoLogin,
    updateUser,
  };
});
