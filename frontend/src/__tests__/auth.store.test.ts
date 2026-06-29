import { describe, it, expect, vi, beforeEach } from "vitest";
import { setActivePinia, createPinia } from "pinia";
import { useAuthStore } from "@/stores/auth";

// Mock the backend bindings
const mockLoginVendedor = vi.fn();
const mockVerificarLoginMFA = vi.fn();
const mockRegistrarVendedor = vi.fn();

vi.mock("@/../bridge/go/backend/Db", () => ({
  LoginVendedor: (...args: any[]) => mockLoginVendedor(...args),
  VerificarLoginMFA: (...args: any[]) => mockVerificarLoginMFA(...args),
  RegistrarVendedor: (...args: any[]) => mockRegistrarVendedor(...args),
}));

// Helper to create a mock Vendedor from backend
function mockVendedor(overrides: Record<string, any> = {}) {
  return {
    UUID: "test-uuid-123",
    Nombre: "Juan",
    Apellido: "Perez",
    Cedula: "12345678",
    Email: "juan@test.com",
    Contrasena: "",
    MFAEnabled: false,
    Role: "admin",
    CreatedAt: "2024-01-01T00:00:00Z",
    UpdatedAt: "2024-01-01T00:00:00Z",
    ...overrides,
  };
}

describe("Auth Store", () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    vi.clearAllMocks();
    localStorage.clear();
  });

  describe("Initial State", () => {
    it("starts unauthenticated with no user", () => {
      const store = useAuthStore();
      expect(store.isAuthenticated).toBe(false);
      expect(store.currentUser).toBeNull();
      expect(store.token).toBeNull();
      expect(store.isAdmin).toBe(false);
    });

    it("loads token from localStorage on creation", () => {
      localStorage.setItem("authToken", "saved-token");
      setActivePinia(createPinia());
      const store = useAuthStore();
      expect(store.token).toBe("saved-token");
      // Still not authenticated because user is null
      expect(store.isAuthenticated).toBe(false);
    });
  });

  describe("login()", () => {
    it("calls LoginVendedor and sets auth state on success (no MFA)", async () => {
      const vendedor = mockVendedor();
      mockLoginVendedor.mockResolvedValue({
        MFARequired: false,
        Token: "jwt-token-abc",
        Vendedor: vendedor,
      });

      const store = useAuthStore();
      const mfaRequired = await store.login({
        Email: "juan@test.com",
        Contrasena: "password123",
      });

      expect(mfaRequired).toBe(false);
      expect(mockLoginVendedor).toHaveBeenCalledWith({
        Email: "juan@test.com",
        Contrasena: "password123",
      });
      expect(store.isAuthenticated).toBe(true);
      expect(store.token).toBe("jwt-token-abc");
      expect(store.currentUser?.UUID).toBe("test-uuid-123");
      expect(store.currentUser?.Role).toBe("admin");
      expect(localStorage.getItem("authToken")).toBe("jwt-token-abc");
      expect(localStorage.getItem("authUser")).toContain("test-uuid-123");
    });

    it("returns true when MFA is required and stores temp token", async () => {
      mockLoginVendedor.mockResolvedValue({
        MFARequired: true,
        Token: "temp-mfa-token",
        Vendedor: mockVendedor(),
      });

      const store = useAuthStore();
      const mfaRequired = await store.login({
        Email: "juan@test.com",
        Contrasena: "password123",
      });

      expect(mfaRequired).toBe(true);
      // Should NOT be fully authenticated yet
      expect(store.token).toBeNull();
    });

    it("propagates backend errors", async () => {
      mockLoginVendedor.mockRejectedValue(
        "vendedor no encontrado o credenciales incorrectas"
      );

      const store = useAuthStore();
      await expect(
        store.login({ Email: "bad@test.com", Contrasena: "wrong" })
      ).rejects.toEqual(
        "vendedor no encontrado o credenciales incorrectas"
      );

      expect(store.isAuthenticated).toBe(false);
    });
  });

  describe("verifyMfaAndFinishLogin()", () => {
    it("throws if no temp MFA token exists", async () => {
      const store = useAuthStore();
      await expect(store.verifyMfaAndFinishLogin("123456")).rejects.toThrow(
        "No se encontró un token de MFA temporal"
      );
    });
  });

  describe("register()", () => {
    it("calls RegistrarVendedor and returns the result", async () => {
      const vendedor = mockVendedor({ Role: "cajero" });
      mockRegistrarVendedor.mockResolvedValue(vendedor);

      const store = useAuthStore();
      const result = await store.register(vendedor as any);

      expect(mockRegistrarVendedor).toHaveBeenCalledWith(vendedor);
      expect(result.UUID).toBe("test-uuid-123");
      expect(result.Role).toBe("cajero");
    });

    it("wraps backend registration errors", async () => {
      mockRegistrarVendedor.mockRejectedValue(
        "la cédula o el email ya están registrados"
      );

      const store = useAuthStore();
      await expect(store.register({} as any)).rejects.toThrow(
        "la cédula o el email ya están registrados"
      );
    });
  });

  describe("RBAC: isAdmin getter", () => {
    it("returns true for admin role", () => {
      const store = useAuthStore();
      store.setAuthenticated(mockVendedor({ Role: "admin" }) as any, "token");
      expect(store.isAdmin).toBe(true);
    });

    it("returns false for cajero role", () => {
      const store = useAuthStore();
      store.setAuthenticated(mockVendedor({ Role: "cajero" }) as any, "token");
      expect(store.isAdmin).toBe(false);
    });

    it("returns false when no user", () => {
      const store = useAuthStore();
      expect(store.isAdmin).toBe(false);
    });
  });

  describe("userInitials", () => {
    it("returns initials from first and last name", () => {
      const store = useAuthStore();
      store.setAuthenticated(
        mockVendedor({ Nombre: "Maria", Apellido: "Garcia" }) as any,
        "token"
      );
      expect(store.userInitials).toBe("MG");
    });

    it("returns empty string when no user", () => {
      const store = useAuthStore();
      expect(store.userInitials).toBe("");
    });
  });

  describe("logout()", () => {
    it("clears all auth state and localStorage", () => {
      const store = useAuthStore();
      store.setAuthenticated(mockVendedor() as any, "token-abc");
      expect(store.isAuthenticated).toBe(true);

      store.logout();

      expect(store.isAuthenticated).toBe(false);
      expect(store.token).toBeNull();
      expect(store.currentUser).toBeNull();
      expect(localStorage.getItem("authToken")).toBeNull();
      expect(localStorage.getItem("authUser")).toBeNull();
    });
  });

  describe("tryAutoLogin()", () => {
    it("restores auth from localStorage", () => {
      const vendedor = mockVendedor();
      localStorage.setItem("authToken", "stored-token");
      localStorage.setItem("authUser", JSON.stringify(vendedor));

      setActivePinia(createPinia());
      const store = useAuthStore();
      store.tryAutoLogin();

      expect(store.isAuthenticated).toBe(true);
      expect(store.currentUser?.UUID).toBe("test-uuid-123");
      expect(store.currentUser?.Role).toBe("admin");
    });

    it("accepts tokenOverride and stores it", () => {
      const vendedor = mockVendedor();
      localStorage.setItem("authUser", JSON.stringify(vendedor));

      setActivePinia(createPinia());
      const store = useAuthStore();
      store.tryAutoLogin("override-token");

      expect(store.token).toBe("override-token");
      expect(localStorage.getItem("authToken")).toBe("override-token");
    });

    it("clears auth when stored user is invalid JSON", () => {
      localStorage.setItem("authToken", "some-token");
      localStorage.setItem("authUser", "not-valid-json{{");

      setActivePinia(createPinia());
      const store = useAuthStore();
      store.tryAutoLogin();

      expect(store.isAuthenticated).toBe(false);
      expect(localStorage.getItem("authToken")).toBeNull();
    });
  });

  describe("updateUser()", () => {
    it("merges partial user data and persists to localStorage", () => {
      const store = useAuthStore();
      store.setAuthenticated(mockVendedor() as any, "token");

      store.updateUser({ Nombre: "Carlos", Role: "cajero" });

      expect(store.currentUser?.Nombre).toBe("Carlos");
      expect(store.currentUser?.Role).toBe("cajero");
      expect(store.currentUser?.Apellido).toBe("Perez"); // unchanged
      const stored = JSON.parse(localStorage.getItem("authUser")!);
      expect(stored.Nombre).toBe("Carlos");
    });

    it("does nothing when no user is set", () => {
      const store = useAuthStore();
      store.updateUser({ Nombre: "Carlos" });
      expect(store.currentUser).toBeNull();
    });
  });
});
