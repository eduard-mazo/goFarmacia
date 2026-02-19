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

const props = defineProps<{ open: boolean }>();
const emit = defineEmits(["update:open", "cliente-seleccionado"]);

const busqueda = ref("");
const clientes = ref<backend.Cliente[]>([]);
const isLoading = ref(false);
const hasSearched = ref(false);
const debounceTimer = ref<number | undefined>(undefined);

watch(busqueda, (nuevoValor) => {
  clearTimeout(debounceTimer.value);
  if (!nuevoValor.trim()) {
    clientes.value = [];
    hasSearched.value = false;
    return;
  }
  isLoading.value = true;
  debounceTimer.value = setTimeout(async () => {
    try {
      const resultado = await ObtenerClientesPaginado(1, 20, nuevoValor, "", "asc");
      clientes.value = (resultado.Records as backend.Cliente[]) || [];
      hasSearched.value = true;
    } catch (error) {
      toast.error("Error de búsqueda", { description: "No se pudieron obtener los clientes." });
    } finally {
      isLoading.value = false;
    }
  }, 300);
});

// Reset al cerrar
watch(() => props.open, (open) => {
  if (!open) {
    busqueda.value = "";
    clientes.value = [];
    hasSearched.value = false;
  }
});

function seleccionarCliente(cliente: backend.Cliente) {
  emit("cliente-seleccionado", cliente);
  emit("update:open", false);
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
        <!-- Results list -->
        <ul v-if="clientes.length > 0">
          <li
            v-for="cliente in clientes"
            :key="cliente.UUID"
            class="flex items-center justify-between px-4 py-3 hover:bg-muted cursor-pointer border-b last:border-0"
            @click="seleccionarCliente(cliente)"
          >
            <div>
              <p class="font-semibold text-sm">{{ cliente.Nombre }} {{ cliente.Apellido }}</p>
              <p class="text-xs text-muted-foreground">{{ cliente.NumeroID }}</p>
            </div>
            <p class="text-xs text-muted-foreground">{{ cliente.Telefono }}</p>
          </li>
        </ul>
        <!-- Empty states -->
        <div v-else class="h-full flex flex-col items-center justify-center gap-2 text-muted-foreground">
          <Search class="h-8 w-8 opacity-30" />
          <p v-if="isLoading" class="text-sm">Buscando...</p>
          <p v-else-if="hasSearched" class="text-sm">Sin resultados para "<span class="font-medium">{{ busqueda }}</span>"</p>
          <p v-else class="text-sm">Escribe para buscar clientes</p>
        </div>
      </div>
    </DialogContent>
  </Dialog>
</template>
