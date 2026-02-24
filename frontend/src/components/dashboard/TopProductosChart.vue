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

interface ProductoVendido {
  nombre: string;
  cantidad: number;
}

const props = defineProps({
  chartData: {
    type: Array as PropType<ProductoVendido[]>,
    required: true,
  },
});

const COLORS = [
  "rgba(16,185,129,0.85)",
  "rgba(37,99,235,0.85)",
  "rgba(245,158,11,0.85)",
  "rgba(139,92,246,0.85)",
  "rgba(244,63,94,0.85)",
];

const truncate = (s: string, n = 22) =>
  s.length > n ? s.slice(0, n) + "…" : s;

const chartOptions = {
  indexAxis: "y" as const,
  responsive: true,
  maintainAspectRatio: false,
  plugins: {
    legend: { display: false },
    tooltip: {
      callbacks: {
        label: (ctx: any) => ` ${ctx.parsed.x} unidades`,
        title: (ctx: any) => props.chartData[ctx[0].dataIndex]?.nombre ?? "",
      },
    },
  },
  scales: {
    x: {
      beginAtZero: true,
      grid: { color: "rgba(0,0,0,0.04)" },
      ticks: { font: { size: 10 }, precision: 0 },
    },
    y: {
      grid: { display: false },
      ticks: { font: { size: 11 } },
    },
  },
};

const formattedChartData = computed(() => ({
  labels: props.chartData.map((d) => truncate(d.nombre)),
  datasets: [
    {
      label: "Unidades",
      data: props.chartData.map((d) => d.cantidad),
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
      No se vendieron productos este día.
    </p>
  </div>
</template>
