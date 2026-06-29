import { describe, it, expect, vi, beforeEach } from "vitest";
import { setActivePinia, createPinia } from "pinia";
import { useModeStore } from "@/stores/mode";
import { useAuthStore } from "@/stores/auth";

// Mock Wails bindings (auth store imports them)
vi.mock("@/../bridge/go/backend/Db", () => ({
  LoginVendedor: vi.fn(),
  VerificarLoginMFA: vi.fn(),
  RegistrarVendedor: vi.fn(),
}));

function mockVendedor(role: string) {
  return {
    UUID: "test-uuid",
    Nombre: "Test",
    Apellido: "User",
    Cedula: "123",
    Email: "test@test.com",
    Contrasena: "",
    MFAEnabled: false,
    Role: role,
    CreatedAt: "2024-01-01T00:00:00Z",
    UpdatedAt: "2024-01-01T00:00:00Z",
  };
}

describe("Mode Store", () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    localStorage.clear();
  });

  describe("Initial State", () => {
    it("defaults to 'pos' mode", () => {
      const modeStore = useModeStore();
      expect(modeStore.currentMode).toBe("pos");
    });

    it("restores mode from localStorage", () => {
      localStorage.setItem("appMode", "erp");
      setActivePinia(createPinia());
      // Need to set up admin user first so mode store has context
      const authStore = useAuthStore();
      authStore.setAuthenticated(mockVendedor("admin") as any, "token");
      const modeStore = useModeStore();
      expect(modeStore.currentMode).toBe("erp");
    });
  });

  describe("Admin Role", () => {
    it("shows both modes available for admin", () => {
      const authStore = useAuthStore();
      authStore.setAuthenticated(mockVendedor("admin") as any, "token");

      const modeStore = useModeStore();
      expect(modeStore.isAdmin).toBe(true);
      expect(modeStore.availableModes).toEqual(["pos", "erp"]);
    });

    it("allows admin to switch to ERP mode", () => {
      const authStore = useAuthStore();
      authStore.setAuthenticated(mockVendedor("admin") as any, "token");

      const modeStore = useModeStore();
      modeStore.setMode("erp");
      expect(modeStore.currentMode).toBe("erp");
    });

    it("allows admin to switch back to POS mode", () => {
      const authStore = useAuthStore();
      authStore.setAuthenticated(mockVendedor("admin") as any, "token");

      const modeStore = useModeStore();
      modeStore.setMode("erp");
      modeStore.setMode("pos");
      expect(modeStore.currentMode).toBe("pos");
    });
  });

  describe("Cajero Role", () => {
    it("only shows POS mode for cajero", () => {
      const authStore = useAuthStore();
      authStore.setAuthenticated(mockVendedor("cajero") as any, "token");

      const modeStore = useModeStore();
      expect(modeStore.isAdmin).toBe(false);
      expect(modeStore.availableModes).toEqual(["pos"]);
    });

    it("prevents cajero from switching to ERP mode", () => {
      const authStore = useAuthStore();
      authStore.setAuthenticated(mockVendedor("cajero") as any, "token");

      const modeStore = useModeStore();
      modeStore.setMode("erp");
      expect(modeStore.currentMode).toBe("pos"); // should NOT change
    });
  });

  describe("Persistence", () => {
    it("saves mode to localStorage on change", async () => {
      const authStore = useAuthStore();
      authStore.setAuthenticated(mockVendedor("admin") as any, "token");

      const modeStore = useModeStore();
      modeStore.setMode("erp");

      // Watchers are async - wait a tick
      await new Promise((r) => setTimeout(r, 0));
      expect(localStorage.getItem("appMode")).toBe("erp");
    });
  });

  describe("Role Change Watcher", () => {
    it("forces cashier back to POS when role downgrades while in ERP", async () => {
      const authStore = useAuthStore();
      authStore.setAuthenticated(mockVendedor("admin") as any, "token");

      const modeStore = useModeStore();
      modeStore.setMode("erp");
      expect(modeStore.currentMode).toBe("erp");

      // Simulate role change (e.g., user data updated)
      authStore.updateUser({ Role: "cajero" });

      // Watcher triggers async
      await new Promise((r) => setTimeout(r, 0));
      expect(modeStore.currentMode).toBe("pos");
    });
  });
});
