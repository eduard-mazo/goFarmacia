<script setup lang="ts">
import { MoreHorizontal } from "lucide-vue-next";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import type { backend } from "../../wailsjs/go/models";

// Define props to accept a producto object
const props = defineProps<{
  producto: backend.Producto;
}>();

// Define emits for parent component to listen to
const emit = defineEmits<{
  (e: "edit", value: backend.Producto): void;
  (e: "delete", value: backend.Producto): void;
}>();
</script>

<template>
  <DropdownMenu>
    <DropdownMenuTrigger as-child>
      <Button variant="ghost" class="w-8 h-8 p-0">
        <span class="sr-only">Abrir menú</span>
        <MoreHorizontal class="w-4 h-4" />
      </Button>
    </DropdownMenuTrigger>
    <DropdownMenuContent align="end">
      <DropdownMenuLabel>Acciones</DropdownMenuLabel>
      <DropdownMenuSeparator />
      <DropdownMenuItem @click="emit('edit', props.producto)">
        Editar producto
      </DropdownMenuItem>
      <DropdownMenuItem
        @click="emit('delete', props.producto)"
        class="text-red-600"
      >
        Eliminar producto
      </DropdownMenuItem>
    </DropdownMenuContent>
  </DropdownMenu>
</template>
