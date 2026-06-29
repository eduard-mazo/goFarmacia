import { describe, it, expect, vi, beforeEach } from "vitest";
import { setActivePinia, createPinia } from "pinia";
import { useCartStore } from "@/stores/cart";

// Mock toast since cart store uses it
vi.mock("vue-sonner", () => ({
  toast: {
    warning: vi.fn(),
    info: vi.fn(),
    success: vi.fn(),
    error: vi.fn(),
  },
}));

// Mock the Wails backend models
vi.mock("@/../bridge/go/models", () => ({
  backend: {
    Producto: class Producto {
      UUID = "";
      Nombre = "";
      Codigo = "";
      PrecioVenta = 0;
      Stock = 0;
      CreatedAt = "";
      UpdatedAt = "";
      constructor(source: any = {}) {
        Object.assign(this, source);
      }
      static createFrom(source: any) {
        return new Producto(source);
      }
    },
  },
}));

// Mock Wails bindings (needed because auth store is imported transitively)
vi.mock("@/../bridge/go/backend/Db", () => ({
  LoginVendedor: vi.fn(),
  VerificarLoginMFA: vi.fn(),
  RegistrarVendedor: vi.fn(),
}));

function mockProducto(overrides: Record<string, any> = {}) {
  return {
    UUID: "prod-uuid-1",
    Nombre: "Paracetamol",
    Codigo: "PAR001",
    PrecioVenta: 5000,
    Stock: 50,
    CreatedAt: "2024-01-01T00:00:00Z",
    UpdatedAt: "2024-01-01T00:00:00Z",
    ...overrides,
  };
}

describe("Cart Store", () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    localStorage.clear();
  });

  describe("Initial State", () => {
    it("starts with an empty cart", () => {
      const store = useCartStore();
      expect(store.activeCart).toEqual([]);
      expect(store.activeCartTotal).toBe(0);
      expect(store.savedCarts).toEqual([]);
    });
  });

  describe("addToCart()", () => {
    it("adds a product to the cart with quantity 1", () => {
      const store = useCartStore();
      store.addToCart(mockProducto() as any);

      expect(store.activeCart).toHaveLength(1);
      expect(store.activeCart[0].UUID).toBe("prod-uuid-1");
      expect(store.activeCart[0].cantidad).toBe(1);
    });

    it("increments quantity if product already in cart", () => {
      const store = useCartStore();
      const prod = mockProducto();
      store.addToCart(prod as any);
      store.addToCart(prod as any);

      expect(store.activeCart).toHaveLength(1);
      expect(store.activeCart[0].cantidad).toBe(2);
    });

    it("does not exceed stock limit", () => {
      const store = useCartStore();
      const prod = mockProducto({ Stock: 2 });
      store.addToCart(prod as any);
      store.addToCart(prod as any);
      store.addToCart(prod as any); // Should be capped at 2

      expect(store.activeCart[0].cantidad).toBe(2);
    });

    it("handles multiple different products", () => {
      const store = useCartStore();
      store.addToCart(mockProducto({ UUID: "prod-1" }) as any);
      store.addToCart(mockProducto({ UUID: "prod-2", Nombre: "Ibuprofeno" }) as any);

      expect(store.activeCart).toHaveLength(2);
    });

    it("does nothing if product is null/undefined", () => {
      const store = useCartStore();
      store.addToCart(null as any);
      expect(store.activeCart).toHaveLength(0);
    });
  });

  describe("activeCartTotal", () => {
    it("calculates total correctly", () => {
      const store = useCartStore();
      store.addToCart(mockProducto({ UUID: "p1", PrecioVenta: 5000 }) as any);
      store.addToCart(mockProducto({ UUID: "p2", PrecioVenta: 3000 }) as any);
      store.addToCart(mockProducto({ UUID: "p1", PrecioVenta: 5000 }) as any); // qty = 2

      // p1: 5000 * 2 = 10000, p2: 3000 * 1 = 3000
      expect(store.activeCartTotal).toBe(13000);
    });
  });

  describe("removeFromCart()", () => {
    it("removes a product by UUID", () => {
      const store = useCartStore();
      store.addToCart(mockProducto({ UUID: "p1" }) as any);
      store.addToCart(mockProducto({ UUID: "p2" }) as any);

      store.removeFromCart("p1");
      expect(store.activeCart).toHaveLength(1);
      expect(store.activeCart[0].UUID).toBe("p2");
    });

    it("does nothing if UUID not found", () => {
      const store = useCartStore();
      store.addToCart(mockProducto() as any);
      store.removeFromCart("nonexistent");
      expect(store.activeCart).toHaveLength(1);
    });
  });

  describe("updateQuantity()", () => {
    it("updates product quantity", () => {
      const store = useCartStore();
      store.addToCart(mockProducto({ UUID: "p1", Stock: 100 }) as any);
      store.updateQuantity("p1", 10);
      expect(store.activeCart[0].cantidad).toBe(10);
    });

    it("caps quantity at available stock", () => {
      const store = useCartStore();
      store.addToCart(mockProducto({ UUID: "p1", Stock: 5 }) as any);
      store.updateQuantity("p1", 50);
      expect(store.activeCart[0].cantidad).toBe(5);
    });

    it("removes item if quantity set to negative", () => {
      const store = useCartStore();
      store.addToCart(mockProducto({ UUID: "p1" }) as any);
      store.updateQuantity("p1", -1);
      expect(store.activeCart).toHaveLength(0);
    });
  });

  describe("clearActiveCart()", () => {
    it("empties the active cart", () => {
      const store = useCartStore();
      store.addToCart(mockProducto({ UUID: "p1" }) as any);
      store.addToCart(mockProducto({ UUID: "p2" }) as any);

      store.clearActiveCart();
      expect(store.activeCart).toHaveLength(0);
      expect(store.activeCartTotal).toBe(0);
    });
  });

  describe("saveCurrentCart() / loadCart()", () => {
    it("saves current cart and clears active cart", () => {
      const store = useCartStore();
      store.addToCart(mockProducto({ UUID: "p1" }) as any);

      store.saveCurrentCart();
      expect(store.activeCart).toHaveLength(0);
      expect(store.savedCarts).toHaveLength(1);
      expect(store.savedCarts[0].items).toHaveLength(1);
    });

    it("does nothing if cart is empty", () => {
      const store = useCartStore();
      store.saveCurrentCart();
      expect(store.savedCarts).toHaveLength(0);
    });

    it("loads a saved cart into active cart", () => {
      const store = useCartStore();
      store.addToCart(mockProducto({ UUID: "p1" }) as any);
      store.saveCurrentCart();

      const savedId = store.savedCarts[0].id;
      store.loadCart(savedId);

      expect(store.activeCart).toHaveLength(1);
      expect(store.savedCarts).toHaveLength(0);
    });

    it("refuses to load if active cart is not empty", () => {
      const store = useCartStore();
      store.addToCart(mockProducto({ UUID: "p1" }) as any);
      store.saveCurrentCart();

      // Add another item to active cart
      store.addToCart(mockProducto({ UUID: "p2" }) as any);

      const savedId = store.savedCarts[0].id;
      store.loadCart(savedId);

      // Should NOT have loaded - active cart still has p2
      expect(store.activeCart).toHaveLength(1);
      expect(store.activeCart[0].UUID).toBe("p2");
      expect(store.savedCarts).toHaveLength(1); // still saved
    });
  });

  describe("deleteSavedCart()", () => {
    it("removes a saved cart by ID", () => {
      const store = useCartStore();
      store.addToCart(mockProducto({ UUID: "p1" }) as any);
      store.saveCurrentCart();

      const savedId = store.savedCarts[0].id;
      store.deleteSavedCart(savedId);
      expect(store.savedCarts).toHaveLength(0);
    });
  });
});
