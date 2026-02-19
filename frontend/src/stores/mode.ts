import { defineStore } from "pinia";
import { ref, computed, watch } from "vue";
import { useAuthStore } from "@/stores/auth";

export type AppMode = "pos" | "erp";

export const useModeStore = defineStore("mode", () => {
  const authStore = useAuthStore();

  const currentMode = ref<AppMode>(
    (localStorage.getItem("appMode") as AppMode) || "pos"
  );

  const isAdmin = computed(() => authStore.currentUser?.Role === "admin");

  const availableModes = computed<AppMode[]>(() => {
    if (isAdmin.value) return ["pos", "erp"];
    return ["pos"];
  });

  function setMode(mode: AppMode) {
    if (availableModes.value.includes(mode)) {
      currentMode.value = mode;
    }
  }

  // Force cashier to POS mode
  watch(
    () => authStore.currentUser?.Role,
    (role) => {
      if (role && role !== "admin" && currentMode.value === "erp") {
        currentMode.value = "pos";
      }
    }
  );

  watch(currentMode, (mode) => {
    localStorage.setItem("appMode", mode);
  });

  return {
    currentMode,
    isAdmin,
    availableModes,
    setMode,
  };
});
