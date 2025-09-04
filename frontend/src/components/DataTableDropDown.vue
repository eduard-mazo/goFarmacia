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

// Define props to accept a Vendedor object
const props = defineProps<{
  vendedor: backend.Vendedor;
}>();

// Define emits for parent component to listen to
const emit = defineEmits<{
  (e: "edit", value: backend.Vendedor): void;
  (e: "delete", value: backend.Vendedor): void;
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
      <DropdownMenuItem @click="emit('edit', props.vendedor)">
        Editar Vendedor
      </DropdownMenuItem>
      <DropdownMenuItem
        @click="emit('delete', props.vendedor)"
        class="text-red-600"
      >
        Eliminar Vendedor
      </DropdownMenuItem>
    </DropdownMenuContent>
  </DropdownMenu>
</template>
