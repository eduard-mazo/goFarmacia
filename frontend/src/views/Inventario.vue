<template>
  <div>
    <h1 class="text-3xl font-bold text-gray-800 mb-6">Carga de Inventario</h1>

    <div class="grid grid-cols-1 lg:grid-cols-5 gap-8">
      <div class="lg:col-span-3 bg-white p-6 rounded-lg shadow-md">
        <h3 class="text-xl font-semibold mb-4 border-b pb-3">
          Detalles de la Compra
        </h3>
        <div class="grid grid-cols-1 md:grid-cols-2 gap-4 mb-6">
          <div>
            <label for="proveedor" class="label">Proveedor</label>
            <select id="proveedor" v-model="compra.ProveedorID" class="input">
              <option v-for="p in proveedores" :key="p.id" :value="p.id">
                {{ p.Nombre }}
              </option>
            </select>
          </div>
          <div>
            <label for="factura" class="label">N° Factura Proveedor</label>
            <input
              id="factura"
              type="text"
              v-model="compra.FacturaNumero"
              class="input"
            />
          </div>
        </div>

        <div class="mb-4">
          <label for="scanner" class="label"
            >📦 Escanear Código de Producto</label
          >
          <input
            ref="scannerInput"
            id="scanner"
            type="text"
            v-model="codigoScaneado"
            @keydown.enter.prevent="handleScan"
            placeholder="Escanear o escribir código y presionar Enter"
            class="w-full text-lg p-3 border-gray-300 rounded-md shadow-sm focus:ring-indigo-500 focus:border-indigo-500"
          />
        </div>

        <div class="overflow-x-auto">
          <table class="min-w-full divide-y divide-gray-200">
            <thead class="bg-gray-50">
              <tr>
                <th class="th">Producto</th>
                <th class="th">Cantidad</th>
                <th class="th">Precio Compra</th>
                <th class="th">Subtotal</th>
                <th class="th text-right">Acción</th>
              </tr>
            </thead>
            <tbody class="bg-white divide-y divide-gray-200">
              <tr v-if="productosCompra.length === 0">
                <td colspan="5" class="text-center text-gray-500 py-8">
                  Añade productos a la compra
                </td>
              </tr>
              <tr
                v-for="(item, index) in productosCompra"
                :key="item.ProductoID"
              >
                <td class="td font-medium">
                  {{ getProductName(item.ProductoID) }}
                </td>
                <td class="td">
                  <input
                    type="number"
                    min="1"
                    v-model.number="item.Cantidad"
                    class="w-20 text-center input"
                  />
                </td>
                <td class="td">
                  <input
                    type="number"
                    step="0.01"
                    min="0"
                    v-model.number="item.PrecioCompraUnitario"
                    class="w-28 text-center input"
                  />
                </td>
                <td class="td font-semibold">
                  ${{ (item.Cantidad * item.PrecioCompraUnitario).toFixed(2) }}
                </td>
                <td class="td text-right">
                  <button
                    @click="productosCompra.splice(index, 1)"
                    class="btn-delete"
                  >
                    Quitar
                  </button>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>

      <div class="lg:col-span-2">
        <div class="bg-white p-6 rounded-lg shadow-md sticky top-6">
          <h3 class="text-xl font-semibold mb-4">Resumen</h3>
          <div class="text-4xl font-extrabold text-gray-800 text-right">
            Total:
            <span class="text-indigo-600">${{ totalCompra.toFixed(2) }}</span>
          </div>
          <button
            @click="finalizarCompra"
            :disabled="!isCompraValid"
            class="w-full btn-primary mt-6 py-4 text-lg"
          >
            {{ procesando ? "Procesando..." : "Registrar Compra" }}
          </button>
        </div>
      </div>
    </div>

    <div
      v-if="mostrarModalNuevoProducto"
      class="fixed inset-0 bg-black bg-opacity-50 flex justify-center items-center z-50"
    >
      <div class="bg-white p-8 rounded-lg shadow-2xl w-full max-w-md">
        <h3 class="text-2xl font-bold mb-6">Registrar Nuevo Producto</h3>
        <form @submit.prevent="guardarNuevoProducto">
          <div class="space-y-4">
            <div>
              <label class="label">Código</label>
              <input
                type="text"
                v-model="nuevoProducto.Codigo"
                class="input"
                required
                readonly
              />
            </div>
            <div>
              <label class="label">Nombre</label>
              <input
                type="text"
                v-model="nuevoProducto.Nombre"
                class="input"
                required
              />
            </div>
            <div>
              <label class="label">Precio de Venta</label>
              <input
                type="number"
                step="0.01"
                v-model.number="nuevoProducto.PrecioVenta"
                class="input"
                required
              />
            </div>
            <input type="hidden" v-model.number="nuevoProducto.Stock" />
          </div>
          <div class="flex justify-end space-x-4 mt-8">
            <button
              type="button"
              @click="mostrarModalNuevoProducto = false"
              class="btn-secondary"
            >
              Cancelar
            </button>
            <button type="submit" class="btn-primary">Guardar Producto</button>
          </div>
        </form>
      </div>
    </div>

    <NotificationModal
      :show="notification.show"
      :message="notification.message"
      :type="notification.type"
      @close="notification.show = false"
    />
  </div>
</template>

<script lang="ts" setup>
import { ref, onMounted, reactive, computed } from "vue";
import { backend } from "../../wailsjs/go/models";
import {
  BuscarProductoPorCodigo,
  ObtenerProveedores,
  RegistrarProducto,
  RegistrarCompra,
} from "../../wailsjs/go/backend/Db";
import NotificationModal from "../components/NotificationModal.vue";

type NotificationType = "success" | "error";

// --- STATE ---
const scannerInput = ref<HTMLInputElement | null>(null);
const proveedores = ref<backend.Proveedor[]>([]);
const todosLosProductos = ref<backend.Producto[]>([]); // Cache para nombres de producto
const codigoScaneado = ref("");
const productosCompra = ref<backend.ProductoCompraInfo[]>([]);
const compra = ref(new backend.CompraRequest());
const procesando = ref(false);
const mostrarModalNuevoProducto = ref(false);
const nuevoProducto = ref(new backend.Producto());

const notification = reactive({
  show: false,
  message: "",
  type: "success" as NotificationType,
});

// --- LIFECYCLE ---
onMounted(async () => {
  await cargarDatosIniciales();
  scannerInput.value?.focus();
});

// --- COMPUTED ---
const totalCompra = computed(() => {
  return productosCompra.value.reduce((total, item) => {
    return total + item.Cantidad * item.PrecioCompraUnitario;
  }, 0);
});

const isCompraValid = computed(() => {
  return (
    compra.value.ProveedorID &&
    compra.value.FacturaNumero &&
    productosCompra.value.length > 0 &&
    !procesando.value
  );
});

// --- METHODS ---
const showNotification = (
  message: string,
  type: NotificationType = "success",
  duration: number = 3000
) => {
  notification.message = message;
  notification.type = type;
  notification.show = true;
  if (duration > 0) {
    setTimeout(() => {
      notification.show = false;
    }, duration);
  }
};

const cargarDatosIniciales = async () => {
  try {
    proveedores.value = await ObtenerProveedores();
    if (proveedores.value.length > 0 && proveedores.value[0]?.id !== undefined) {
      compra.value.ProveedorID = proveedores.value[0].id!;
    }
  } catch (error) {
    showNotification(`Error al cargar proveedores: ${error}`, "error");
  }
};

const handleScan = async () => {
  if (!codigoScaneado.value.trim()) return;
  const codigo = codigoScaneado.value.trim();

  try {
    const producto = await BuscarProductoPorCodigo(codigo);
    // Producto existe, añadir a la lista de compra
    agregarProductoACompra(producto);
  } catch (error) {
    // Asumimos que el error es 'record not found'
    if (
      confirm(
        `El producto con código "${codigo}" no existe. ¿Desea crearlo ahora?`
      )
    ) {
      nuevoProducto.value = new backend.Producto();
      nuevoProducto.value.Codigo = codigo;
      nuevoProducto.value.Stock = 0; // El stock inicial será 0, se actualizará con la compra
      nuevoProducto.value.PrecioVenta = 0;
      mostrarModalNuevoProducto.value = true;
    }
  } finally {
    codigoScaneado.value = "";
  }
};

const agregarProductoACompra = (producto: backend.Producto) => {
  todosLosProductos.value.push(producto);
  const itemExistente = productosCompra.value.find(
    (p) => p.ProductoID === producto.id
  );
  if (itemExistente) {
    itemExistente.Cantidad++;
  } else {
    const nuevoItem = new backend.ProductoCompraInfo();
    nuevoItem.ProductoID = producto.id!;
    nuevoItem.Cantidad = 1;
    nuevoItem.PrecioCompraUnitario = 0; // Default price
    productosCompra.value.push(nuevoItem);
  }
  showNotification(`${producto.Nombre} añadido a la compra.`, "success");
};

const guardarNuevoProducto = async () => {
  try {
    await RegistrarProducto(nuevoProducto.value);
    showNotification(
      "Producto creado con éxito. Ahora puede añadirlo a la compra.",
      "success"
    );
    mostrarModalNuevoProducto.value = false;
    // Re-escanear para añadirlo a la lista
    codigoScaneado.value = nuevoProducto.value.Codigo;
    await handleScan();
  } catch (error) {
    showNotification(`Error al crear el producto: ${error}`, "error");
  }
};

const finalizarCompra = async () => {
  if (!isCompraValid.value) return;

  procesando.value = true;
  compra.value.Productos = productosCompra.value;

  try {
    const resultado = await RegistrarCompra(compra.value);
    showNotification(resultado, "success");
    // Resetear formulario
    productosCompra.value = [];
    compra.value.FacturaNumero = "";
  } catch (error) {
    showNotification(`Error al registrar la compra: ${error}`, "error");
  } finally {
    procesando.value = false;
    scannerInput.value?.focus();
  }
};

const getProductName = (id: number) => {
  return (
    todosLosProductos.value.find((p) => p.id === id)?.Nombre || "Desconocido"
  );
};
</script>

<style scoped>
/* Usar estilos globales definidos en style.css para consistencia */
@reference "../style.css";
</style>
