<script setup lang="ts">
import { ref, computed, watch, nextTick } from "vue";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Wallet, Banknote, CreditCard, ReceiptText } from "lucide-vue-next";

const props = defineProps<{
  open: boolean;
  total: number;
  cliente: string;
}>();

const emit = defineEmits(["update:open", "confirm"]);

const metodoPago = ref("efectivo");
const efectivoRecibido = ref<number | undefined>(undefined);
const inputEfectivoRef = ref<HTMLInputElement | null>(null);

const denominations = [2000, 5000, 10000, 20000, 50000, 100000];

const cambio = computed(() => {
  if (metodoPago.value === "efectivo" && efectivoRecibido.value && efectivoRecibido.value >= props.total) {
    return efectivoRecibido.value - props.total;
  }
  return 0;
});

const canConfirm = computed(() => {
  if (metodoPago.value === "efectivo") {
    return efectivoRecibido.value !== undefined && efectivoRecibido.value >= props.total;
  }
  return true;
});

watch(() => props.open, (isOpen) => {
  if (isOpen) {
    metodoPago.value = "efectivo";
    efectivoRecibido.value = props.total;
    nextTick(() => {
      setTimeout(() => {
        inputEfectivoRef.value?.focus();
        inputEfectivoRef.value?.select();
      }, 100);
    });
  }
});

function handleQuickPay(amount: number) {
    efectivoRecibido.value = (efectivoRecibido.value || 0) + amount;
}

function handleExactAmount() {
    efectivoRecibido.value = props.total;
}

function confirm() {
  if (!canConfirm.value) return;
  emit("confirm", {
    metodoPago: metodoPago.value,
    efectivoRecibido: efectivoRecibido.value,
  });
}

function handleKeydown(e: KeyboardEvent) {
    if (e.key === "Enter" && canConfirm.value) {
        confirm();
    }
}

</script>

<template>
  <Dialog :open="open" @update:open="(v) => emit('update:open', v)">
    <DialogContent class="sm:max-w-[600px] gap-0 p-0 overflow-hidden" @keydown="handleKeydown">
      <DialogHeader class="p-6 border-b bg-muted/20">
        <DialogTitle class="flex items-center gap-2 text-xl">
          <ReceiptText class="w-5 h-5 text-primary" />
          Finalizar Venta
        </DialogTitle>
      </DialogHeader>

      <div class="grid grid-cols-1 md:grid-cols-2">
        <!-- Resumen -->
        <div class="p-6 border-b md:border-b-0 md:border-r space-y-6">
          <div class="space-y-1">
            <Label class="text-xs text-muted-foreground uppercase tracking-wider">Cliente</Label>
            <p class="font-semibold">{{ cliente }}</p>
          </div>

          <div class="space-y-3">
            <Label class="text-xs text-muted-foreground uppercase tracking-wider">Método de Pago</Label>
            <div class="grid grid-cols-2 gap-2">
              <Button
                variant="outline"
                class="h-20 flex flex-col gap-2"
                :class="{ 'border-primary bg-primary/5 text-primary': metodoPago === 'efectivo' }"
                @click="metodoPago = 'efectivo'"
              >
                <Banknote class="w-6 h-6" />
                Efectivo
              </Button>
              <Button
                variant="outline"
                class="h-20 flex flex-col gap-2"
                :class="{ 'border-primary bg-primary/5 text-primary': metodoPago === 'transferencia' }"
                @click="metodoPago = 'transferencia'"
              >
                <CreditCard class="w-6 h-6" />
                Transferencia
              </Button>
            </div>
          </div>

          <div class="pt-4 mt-auto">
            <div class="flex items-end justify-between">
              <span class="text-sm font-medium text-muted-foreground">Total a Pagar</span>
              <span class="text-3xl font-bold font-mono text-primary">${{ total.toLocaleString() }}</span>
            </div>
          </div>
        </div>

        <!-- Pago en Efectivo -->
        <div class="p-6 bg-muted/5 space-y-6">
          <div v-if="metodoPago === 'efectivo'" class="space-y-6">
            <div class="space-y-2">
              <Label for="efectivo" class="text-xs text-muted-foreground uppercase tracking-wider">Efectivo Recibido</Label>
              <div class="relative">
                <span class="absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground font-mono">$</span>
                <Input
                  id="efectivo"
                  ref="inputEfectivoRef"
                  type="number"
                  v-model="efectivoRecibido"
                  class="pl-7 h-12 text-2xl font-mono text-right"
                  placeholder="0"
                />
              </div>
            </div>

            <div class="space-y-2">
              <Label class="text-xs text-muted-foreground uppercase tracking-wider">Acceso Rápido</Label>
              <div class="grid grid-cols-3 gap-2">
                <Button
                  v-for="amount in denominations"
                  :key="amount"
                  variant="outline"
                  size="sm"
                  class="font-mono text-xs"
                  @click="handleQuickPay(amount)"
                >
                  +{{ (amount/1000) }}k
                </Button>
                <Button
                    variant="secondary"
                    size="sm"
                    class="col-span-3 font-semibold"
                    @click="handleExactAmount"
                >
                    Valor Exacto
                </Button>
              </div>
            </div>

            <div v-if="efectivoRecibido && efectivoRecibido >= total" class="p-4 rounded-lg bg-emerald-500/10 border border-emerald-500/20">
                <div class="flex items-center justify-between">
                    <span class="text-sm font-medium text-emerald-700">Cambio</span>
                    <span class="text-2xl font-bold font-mono text-emerald-600">${{ cambio.toLocaleString() }}</span>
                </div>
            </div>
            <div v-else-if="efectivoRecibido && efectivoRecibido < total" class="p-4 rounded-lg bg-destructive/10 border border-destructive/20 text-center">
                <span class="text-sm font-medium text-destructive">Faltan ${{ (total - efectivoRecibido).toLocaleString() }}</span>
            </div>
          </div>

          <div v-else class="h-full flex flex-col items-center justify-center text-center space-y-4 py-8">
            <div class="w-16 h-16 rounded-full bg-primary/10 flex items-center justify-center">
                <Wallet class="w-8 h-8 text-primary" />
            </div>
            <div>
                <p class="font-semibold">Transferencia Bancaria</p>
                <p class="text-sm text-muted-foreground">Confirme que recibió el dinero en la cuenta antes de finalizar.</p>
            </div>
          </div>
        </div>
      </div>

      <DialogFooter class="p-6 border-t bg-muted/20 sm:justify-between flex-row gap-4 items-center">
        <Button variant="ghost" @click="emit('update:open', false)">Cancelar</Button>
        <Button
            size="lg"
            class="px-8 font-bold text-lg h-12 gap-2"
            :disabled="!canConfirm"
            @click="confirm"
        >
          Confirmar Venta
          <kbd class="ml-1 hidden sm:inline-flex h-5 select-none items-center gap-1 rounded border bg-primary-foreground/10 px-1.5 font-mono text-[10px] font-medium opacity-100">
            Enter
          </kbd>
        </Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>
</template>
