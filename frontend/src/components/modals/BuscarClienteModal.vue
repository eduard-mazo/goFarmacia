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
import { Search } from "lucide-vue-next";
import { toast } from "vue-sonner";
import { ObtenerClientesPaginado } from "@/../wailsjs/go/backend/Db";
import { backend } from "@/../wailsjs/go/models";

// Props y Emits para la comunicación con el componente padre
const props = defineProps<{
  open: boolean;
}>();
const emit = defineEmits(["update:open", "cliente-seleccionado"]);

// Estado interno del modal
const busqueda = ref("");
const clientes = ref<backend.Cliente[]>([]);
const isLoading = ref(false);
const debounceTimer = ref<number | undefined>(undefined);

// Observador para la búsqueda con debounce
watch(busqueda, (nuevoValor) => {
  clearTimeout(debounceTimer.value);
  if (nuevoValor.length < 2) {
    clientes.value = [];
    return;
  }
  isLoading.value = true;
  debounceTimer.value = setTimeout(async () => {
    try {
      const resultado = await ObtenerClientesPaginado(
        1,
        15,
        nuevoValor,
        "",
        "asc"
      );
      clientes.value = (resultado.Records as backend.Cliente[]) || [];
    } catch (error) {
      console.error("Error al buscar clientes:", error);
      toast.error("Error de búsqueda", {
        description: "No se pudieron obtener los clientes.",
      });
    } finally {
      isLoading.value = false;
    }
  }, 350);
});

function seleccionarCliente(cliente: backend.Cliente) {
  emit("cliente-seleccionado", cliente);
  // Limpiamos para la próxima vez que se abra
  busqueda.value = "";
  clientes.value = [];
}
</script>

<template>
  <Dialog :open="props.open" @update:open="emit('update:open', $event)">
    <DialogContent class="sm:max-w-[625px]">
      <DialogHeader>
        <DialogTitle>Buscar Cliente</DialogTitle>
        <DialogDescription>
          Busca por nombre, apellido o número de identificación.
        </DialogDescription>
      </DialogHeader>
      <div class="relative mt-4">
        <Search
          class="absolute left-3 top-1/2 -translate-y-1/2 h-5 w-5 text-muted-foreground"
        />
        <Input
          v-model="busqueda"
          placeholder="Escribe para buscar..."
          class="pl-10 text-md h-10"
        />
      </div>
      <div class="mt-4 h-72 overflow-y-auto border rounded-md">
        <ul v-if="clientes.length > 0">
          <li
            v-for="cliente in clientes"
            :key="cliente.id"
            class="p-3 hover:bg-muted cursor-pointer"
            @click="seleccionarCliente(cliente)"
          >
            <p class="font-semibold">
              {{ cliente.Nombre }} {{ cliente.Apellido }}
            </p>
            <p class="text-sm text-muted-foreground">
              ID: {{ cliente.NumeroID }} | Tel: {{ cliente.Telefono }}
            </p>
          </li>
        </ul>
        <div
          v-else
          class="h-full flex items-center justify-center text-muted-foreground"
        >
          <p v-if="isLoading">Buscando...</p>
          <p v-else>No se encontraron clientes</p>
        </div>
      </div>
    </DialogContent>
  </Dialog>
</template>
