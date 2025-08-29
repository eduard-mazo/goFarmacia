import { defineStore } from "pinia";

export type CartItem = {
  id: number;
  nombre: string;
  precio: number;
  cantidad: number;
};

const STORAGE_KEY = "gofarmacia_cart_v1";

export const useCartStore = defineStore("cart", {
  state: () => ({
    items: [] as CartItem[],
  }),
  getters: {
    total: (state) =>
      state.items.reduce((acc, i) => acc + i.precio * i.cantidad, 0),
    totalCantidad: (state) => state.items.reduce((a, i) => a + i.cantidad, 0),
  },
  actions: {
    _persist() {
      localStorage.setItem(STORAGE_KEY, JSON.stringify(this.items));
    },
    _load() {
      const raw = localStorage.getItem(STORAGE_KEY);
      if (raw) {
        try {
          this.items = JSON.parse(raw) as CartItem[];
        } catch {
          this.items = [];
        }
      }
    },
    init() {
      this._load();
    },
    addItem(
      producto: { id: number; nombre: string; precio: number },
      cantidad = 1
    ) {
      const found = this.items.find((x) => x.id === producto.id);
      if (found) {
        found.cantidad += cantidad;
      } else {
        this.items.push({ ...producto, cantidad });
      }
      this._persist();
    },
    setCantidad(id: number, cantidad: number) {
      const it = this.items.find((x) => x.id === id);
      if (!it) return;
      it.cantidad = Math.max(1, cantidad);
      this._persist();
    },
    removeItem(id: number) {
      this.items = this.items.filter((x) => x.id !== id);
      this._persist();
    },
    clearCart() {
      this.items = [];
      this._persist();
    },
  },
});
