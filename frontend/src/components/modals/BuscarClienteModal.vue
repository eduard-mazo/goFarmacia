<script setup lang="ts">
import { ref, watch } from "vue";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Search, Loader2 } from "lucide-vue-next";
import { toast } from "vue-sonner";
import { ObtenerClientesPaginado } from "@/../bridge/go/backend/Db";
import { backend } from "@/../bridge/go/models";

const props = defineProps<{ open: boolean }>();
const emit = defineEmits(["update:open", "cliente-seleccionado"]);

const busqueda = ref("");
const clientes = ref<backend.Cliente[]>([]);
const isLoading = ref(false);
const debounceTimer = ref<number | undefined>(undefined);

async function cargarClientes(filtro = "") {
  isLoading.value = true;
  try {
    const resultado = await ObtenerClientesPaginado(1, 50, filtro, "nombre", "asc");
    clientes.value = (resultado.Records as backend.Cliente[]) || [];
  } catch {
    toast.error("Error al obtener clientes.");
  } finally {
    isLoading.value = false;
  }
}

// Cargar todos al abrir; limpiar al cerrar
watch(() => props.open, (open) => {
  if (open) {
    busqueda.value = "";
    cargarClientes();
  } else {
    busqueda.value = "";
    clientes.value = [];
  }
});

// Filtrar con debounce mientras se escribe
watch(busqueda, (valor) => {
  clearTimeout(debounceTimer.value);
  debounceTimer.value = setTimeout(() => cargarClientes(valor.trim()), 300);
});

function seleccionarCliente(cliente: backend.Cliente) {
  emit("cliente-seleccionado", cliente);
  emit("update:open", false);
}
</script>

<template>
  <Dialog :open="props.open" @update:open="emit('update:open', $event)">
    <DialogContent class="sm:max-w-[560px]">
      <DialogHeader>
        <DialogTitle>Buscar Cliente</DialogTitle>
        <DialogDescription>
          Filtra por nombre, apellido o número de identificación.
        </DialogDescription>
      </DialogHeader>

      <!-- Search bar -->
      <div class="relative mt-2">
        <Search class="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground pointer-events-none" />
        <Input
          v-model="busqueda"
          placeholder="Filtrar clientes..."
          class="pl-9 h-9"
          autofocus
        />
        <Loader2
          v-if="isLoading"
          class="absolute right-3 top-1/2 -translate-y-1/2 h-4 w-4 animate-spin text-muted-foreground"
        />
      </div>

      <!-- List -->
      <div class="h-72 overflow-y-auto border rounded-md mt-3">
        <ul v-if="clientes.length > 0">
          <li
            v-for="cliente in clientes"
            :key="cliente.UUID"
            class="flex items-center justify-between px-4 py-2.5 hover:bg-muted cursor-pointer border-b last:border-0 select-none"
            @click="seleccionarCliente(cliente)"
          >
            <div>
              <p class="font-medium text-sm">{{ cliente.Nombre }} {{ cliente.Apellido }}</p>
              <p class="text-xs text-muted-foreground">{{ cliente.NumeroID }}</p>
            </div>
            <p class="text-xs text-muted-foreground">{{ cliente.Telefono }}</p>
          </li>
        </ul>
        <div v-else class="h-full flex flex-col items-center justify-center gap-2 text-muted-foreground">
          <Search class="h-7 w-7 opacity-25" />
          <p class="text-sm">
            {{ isLoading ? "Cargando..." : busqueda ? `Sin resultados para "${busqueda}"` : "Sin clientes registrados" }}
          </p>
        </div>
      </div>
    </DialogContent>
  </Dialog>
</template>
