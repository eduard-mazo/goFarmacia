<script setup lang="ts">
import { Bar } from "vue-chartjs";
import {
  Chart as ChartJS,
  Title,
  Tooltip,
  Legend,
  BarElement,
  CategoryScale,
  LinearScale,
} from "chart.js";
import { computed } from "vue";
import type { PropType } from "vue";

ChartJS.register(Title, Tooltip, Legend, BarElement, CategoryScale, LinearScale);

interface MetodoPago {
  metodo_pago: string;
  count: number;
  monto: number;
}

const props = defineProps({
  chartData: {
    type: Array as PropType<MetodoPago[]>,
    required: true,
  },
});

const fmt = (v: number) =>
  new Intl.NumberFormat("es-CO", {
    style: "currency",
    currency: "COP",
    minimumFractionDigits: 0,
  }).format(v);

const COLORS = [
  "rgba(37,99,235,0.8)",
  "rgba(16,185,129,0.8)",
  "rgba(245,158,11,0.8)",
  "rgba(139,92,246,0.8)",
  "rgba(244,63,94,0.8)",
];

const chartOptions = computed(() => ({
  indexAxis: "y" as const,
  responsive: true,
  maintainAspectRatio: false,
  plugins: {
    legend: { display: false },
    tooltip: {
      callbacks: {
        label: (ctx: any) => {
          const item = props.chartData[ctx.dataIndex];
          return ` ${fmt(ctx.parsed.x)}  ·  ${item.count} transacción(es)`;
        },
      },
    },
  },
  scales: {
    x: {
      beginAtZero: true,
      grid: { color: "rgba(0,0,0,0.04)" },
      ticks: {
        font: { size: 10 },
        callback: (v: any) =>
          new Intl.NumberFormat("es-CO", {
            notation: "compact",
            currency: "COP",
            style: "currency",
            maximumFractionDigits: 0,
          }).format(v),
      },
    },
    y: {
      grid: { display: false },
      ticks: { font: { size: 11 } },
    },
  },
}));

const formattedChartData = computed(() => ({
  labels: props.chartData.map((d) => d.metodo_pago),
  datasets: [
    {
      label: "Monto",
      data: props.chartData.map((d) => d.monto),
      backgroundColor: props.chartData.map((_, i) => COLORS[i % COLORS.length]),
      borderRadius: 4,
      borderSkipped: false,
    },
  ],
}));
</script>

<template>
  <div class="h-full w-full">
    <Bar
      v-if="chartData.length > 0"
      :data="formattedChartData"
      :options="chartOptions"
    />
    <p v-else class="flex items-center justify-center h-full text-sm text-muted-foreground">
      Sin datos de pago para este día.
    </p>
  </div>
</template>
