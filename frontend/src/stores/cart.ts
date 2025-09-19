import { ref, computed, watch } from "vue";
import { defineStore } from "pinia";
import { backend } from "@/../wailsjs/go/models";
import { toast } from "vue-sonner";

// Interfaces para un tipado estricto
export interface ItemCarrito extends backend.Producto {
  cantidad: number;
}

export interface CarritoGuardado {
  id: number;
  nombre: string;
  items: ItemCarrito[];
  total: number;
  fecha: Date;
}

// Claves para el almacenamiento local
const ACTIVE_CART_STORAGE_KEY = "gofarmacia_active_cart_v2";
const SAVED_CARTS_STORAGE_KEY = "gofarmacia_saved_carts_v2";

export const useCartStore = defineStore("cart", () => {
  // --- ESTADO ---
  // Se intenta cargar el estado inicial desde localStorage.
  const activeCart = ref<ItemCarrito[]>(
    JSON.parse(localStorage.getItem(ACTIVE_CART_STORAGE_KEY) || "[]")
  );
  const savedCarts = ref<CarritoGuardado[]>(
    JSON.parse(localStorage.getItem(SAVED_CARTS_STORAGE_KEY) || "[]")
  );

  // --- GETTERS (Propiedades computadas) ---
  const activeCartTotal = computed(() =>
    activeCart.value.reduce(
      (acc, item) => acc + item.PrecioVenta * item.cantidad,
      0
    )
  );

  // --- [NUEVO] Lógica de persistencia manual ---
  // Observamos los cambios en los carritos y los guardamos automáticamente.
  watch(
    activeCart,
    (newCart) => {
      localStorage.setItem(ACTIVE_CART_STORAGE_KEY, JSON.stringify(newCart));
    },
    { deep: true }
  );

  watch(
    savedCarts,
    (newSavedCarts) => {
      localStorage.setItem(
        SAVED_CARTS_STORAGE_KEY,
        JSON.stringify(newSavedCarts)
      );
    },
    { deep: true }
  );

  // --- ACCIONES ---

  function addToCart(producto: backend.Producto) {
    if (!producto) return;

    const itemExistente = activeCart.value.find(
      (item) => item.id === producto.id
    );

    if (itemExistente) {
      if (itemExistente.cantidad < itemExistente.Stock) {
        itemExistente.cantidad++;
      } else {
        toast.warning("Stock máximo alcanzado", {
          description: `No hay más stock disponible para ${producto.Nombre}.`,
        });
      }
    } else {
      const nuevoProducto = {
        ...new backend.Producto(producto),
        cantidad: 1,
      } as ItemCarrito;
      activeCart.value.push(nuevoProducto);
    }
  }

  function removeFromCart(productoId: number) {
    activeCart.value = activeCart.value.filter((p) => p.id !== productoId);
  }

  function updateQuantity(productoId: number, nuevaCantidad: number) {
    const item = activeCart.value.find((p) => p.id === productoId);
    if (item) {
      if (nuevaCantidad > 0 && nuevaCantidad <= item.Stock) {
        item.cantidad = nuevaCantidad;
      } else if (nuevaCantidad > item.Stock) {
        item.cantidad = item.Stock;
        toast.warning("Stock máximo alcanzado", {
          description: `El stock disponible es de ${item.Stock} unidades.`,
        });
      } else if (nuevaCantidad <= 0) {
        removeFromCart(productoId);
      }
    }
  }

  function clearActiveCart() {
    activeCart.value = [];
  }

  function saveCurrentCart() {
    if (activeCart.value.length === 0) {
      toast.info("El carrito está vacío", {
        description: "No hay nada que guardar.",
      });
      return;
    }

    const newSavedCart: CarritoGuardado = {
      id: Date.now(),
      nombre: `Guardado a las ${new Date().toLocaleTimeString()}`,
      items: [...activeCart.value],
      total: activeCartTotal.value,
      fecha: new Date(),
    };

    savedCarts.value.unshift(newSavedCart);
    clearActiveCart();
    toast.success("Carrito guardado", {
      description: `Se ha guardado el carrito para continuar más tarde.`,
    });
  }

  function loadCart(cartId: number) {
    if (activeCart.value.length > 0) {
      toast.error("Hay una venta en curso", {
        description:
          "Por favor, finaliza o guarda la venta actual antes de cargar otra.",
      });
      return;
    }

    const cartToLoad = savedCarts.value.find((c) => c.id === cartId);
    if (cartToLoad) {
      activeCart.value = cartToLoad.items;
      savedCarts.value = savedCarts.value.filter((c) => c.id !== cartId);
      toast.success("Carrito cargado", {
        description: `Listo para continuar con la venta.`,
      });
    }
  }

  function deleteSavedCart(cartId: number) {
    savedCarts.value = savedCarts.value.filter((c) => c.id !== cartId);
    toast.info("Carrito en espera eliminado.");
  }

  return {
    activeCart,
    savedCarts,
    activeCartTotal,
    addToCart,
    removeFromCart,
    updateQuantity,
    clearActiveCart,
    saveCurrentCart,
    loadCart,
    deleteSavedCart,
  };
});
