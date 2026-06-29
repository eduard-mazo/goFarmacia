import { describe, it, expect, vi, beforeEach } from "vitest";
import { setActivePinia, createPinia } from "pinia";
import { createRouter, createWebHashHistory } from "vue-router";
import { useAuthStore } from "@/stores/auth";

// Mock Wails bindings
vi.mock("@/../bridge/go/backend/Db", () => ({
  LoginVendedor: vi.fn(),
  VerificarLoginMFA: vi.fn(),
  RegistrarVendedor: vi.fn(),
}));

// Mock jwt-decode
vi.mock("jwt-decode", () => ({
  jwtDecode: vi.fn(() => ({
    exp: Math.floor(Date.now() / 1000) + 3600, // valid for 1 hour
  })),
}));

// Minimal component stub
const Stub = { template: "<div>stub</div>" };

function createTestRouter() {
  return createRouter({
    history: createWebHashHistory(),
    routes: [
      {
        path: "/login",
        name: "Login",
        component: Stub,
        meta: { public: true },
      },
      {
        path: "/Register",
        name: "Register",
        component: Stub,
        meta: { public: true },
      },
      {
        path: "/dashboard",
        component: Stub,
        meta: { requiresAuth: true },
        children: [
          { path: "", name: "DashboardHome", component: Stub },
          { path: "pos", name: "VentasPOS", component: Stub },
          { path: "facturas", name: "Facturas", component: Stub },
          {
            path: "erp",
            name: "DashboardERP",
            component: Stub,
            meta: { requiresAuth: true, role: "admin" },
          },
          {
            path: "productos",
            name: "Productos",
            component: Stub,
            meta: { requiresAuth: true, role: "admin" },
          },
          {
            path: "proveedores",
            name: "Proveedores",
            component: Stub,
            meta: { requiresAuth: true, role: "admin" },
          },
          {
            path: "reportes/ventas",
            name: "ReporteVentas",
            component: Stub,
            meta: { requiresAuth: true, role: "admin" },
          },
          {
            path: "configuracion",
            name: "Configuracion",
            component: Stub,
            meta: { requiresAuth: true, role: "admin" },
          },
        ],
      },
    ],
  });
}

function mockVendedor(role: string) {
  return {
    UUID: "v-uuid",
    Nombre: "Test",
    Apellido: "User",
    Cedula: "123",
    Email: "test@test.com",
    Contrasena: "",
    MFAEnabled: false,
    Role: role,
    CreatedAt: "2024-01-01",
    UpdatedAt: "2024-01-01",
  };
}

describe("Router Guards", () => {
  let router: ReturnType<typeof createTestRouter>;

  beforeEach(() => {
    setActivePinia(createPinia());
    localStorage.clear();
    router = createTestRouter();

    // Install the same guard logic from our router
    router.beforeEach((to, _, next) => {
      const authStore = useAuthStore();

      if (!authStore.isAuthenticated) {
        authStore.tryAutoLogin();
      }

      const requiresAuth = to.matched.some((r) => r.meta.requiresAuth);
      const isPublic = to.matched.some((r) => r.meta.public);

      if (requiresAuth && !authStore.isAuthenticated) {
        next({ name: "Login" });
      } else if (isPublic && authStore.isAuthenticated) {
        next({ name: "DashboardHome" });
      } else {
        const requiredRole = to.meta.role as string | undefined;
        if (requiredRole && authStore.currentUser?.Role !== requiredRole) {
          next({ name: "DashboardHome" });
        } else {
          next();
        }
      }
    });
  });

  describe("Unauthenticated user", () => {
    it("redirects to Login when accessing protected route", async () => {
      await router.push("/dashboard");
      await router.isReady();
      expect(router.currentRoute.value.name).toBe("Login");
    });

    it("allows access to Login page", async () => {
      await router.push("/login");
      await router.isReady();
      expect(router.currentRoute.value.name).toBe("Login");
    });

    it("allows access to Register page", async () => {
      await router.push("/Register");
      await router.isReady();
      expect(router.currentRoute.value.name).toBe("Register");
    });
  });

  describe("Authenticated admin", () => {
    beforeEach(() => {
      const authStore = useAuthStore();
      authStore.setAuthenticated(mockVendedor("admin") as any, "valid-token");
    });

    it("can access POS dashboard", async () => {
      await router.push("/dashboard");
      await router.isReady();
      expect(router.currentRoute.value.name).toBe("DashboardHome");
    });

    it("can access ERP dashboard", async () => {
      await router.push("/dashboard/erp");
      await router.isReady();
      expect(router.currentRoute.value.name).toBe("DashboardERP");
    });

    it("can access admin-only routes (Productos)", async () => {
      await router.push("/dashboard/productos");
      await router.isReady();
      expect(router.currentRoute.value.name).toBe("Productos");
    });

    it("can access Proveedores", async () => {
      await router.push("/dashboard/proveedores");
      await router.isReady();
      expect(router.currentRoute.value.name).toBe("Proveedores");
    });

    it("can access ReporteVentas", async () => {
      await router.push("/dashboard/reportes/ventas");
      await router.isReady();
      expect(router.currentRoute.value.name).toBe("ReporteVentas");
    });

    it("can access Configuracion", async () => {
      await router.push("/dashboard/configuracion");
      await router.isReady();
      expect(router.currentRoute.value.name).toBe("Configuracion");
    });

    it("redirects away from Login to Dashboard", async () => {
      await router.push("/login");
      await router.isReady();
      expect(router.currentRoute.value.name).toBe("DashboardHome");
    });
  });

  describe("Authenticated cajero (cashier)", () => {
    beforeEach(() => {
      const authStore = useAuthStore();
      authStore.setAuthenticated(mockVendedor("cajero") as any, "valid-token");
    });

    it("can access POS dashboard", async () => {
      await router.push("/dashboard");
      await router.isReady();
      expect(router.currentRoute.value.name).toBe("DashboardHome");
    });

    it("can access POS view", async () => {
      await router.push("/dashboard/pos");
      await router.isReady();
      expect(router.currentRoute.value.name).toBe("VentasPOS");
    });

    it("can access Facturas", async () => {
      await router.push("/dashboard/facturas");
      await router.isReady();
      expect(router.currentRoute.value.name).toBe("Facturas");
    });

    it("is blocked from ERP dashboard → redirected to DashboardHome", async () => {
      await router.push("/dashboard/erp");
      await router.isReady();
      expect(router.currentRoute.value.name).toBe("DashboardHome");
    });

    it("is blocked from Productos → redirected to DashboardHome", async () => {
      await router.push("/dashboard/productos");
      await router.isReady();
      expect(router.currentRoute.value.name).toBe("DashboardHome");
    });

    it("is blocked from Proveedores → redirected to DashboardHome", async () => {
      await router.push("/dashboard/proveedores");
      await router.isReady();
      expect(router.currentRoute.value.name).toBe("DashboardHome");
    });

    it("is blocked from ReporteVentas → redirected to DashboardHome", async () => {
      await router.push("/dashboard/reportes/ventas");
      await router.isReady();
      expect(router.currentRoute.value.name).toBe("DashboardHome");
    });

    it("is blocked from Configuracion → redirected to DashboardHome", async () => {
      await router.push("/dashboard/configuracion");
      await router.isReady();
      expect(router.currentRoute.value.name).toBe("DashboardHome");
    });
  });
});
