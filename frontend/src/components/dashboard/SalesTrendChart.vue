<script setup lang="ts">
import { Line } from "vue-chartjs";
import {
  Chart as ChartJS,
  Title,
  Tooltip,
  Legend,
  LineElement,
  PointElement,
  CategoryScale,
  LinearScale,
  Filler,
} from "chart.js";
import type { PropType } from "vue";
import { computed } from "vue";

ChartJS.register(
  Title, Tooltip, Legend,
  LineElement, PointElement,
  CategoryScale, LinearScale, Filler
);

interface VentaIndividual {
  timestamp: string;
  total: number;
}

const props = defineProps({
  chartData: {
    type: Array as PropType<VentaIndividual[]>,
    required: true,
  },
});

const fmt = (v: number) =>
  new Intl.NumberFormat("es-CO", {
    style: "currency",
    currency: "COP",
    minimumFractionDigits: 0,
  }).format(v);

const chartOptions = {
  responsive: true,
  maintainAspectRatio: false,
  interaction: { mode: "index" as const, intersect: false },
  plugins: {
    legend: { display: false },
    tooltip: {
      callbacks: {
        title: (ctx: any) => {
          const h = parseInt(ctx[0].label);
          return `${String(h).padStart(2, "0")}:00 – ${String(h + 1).padStart(2, "0")}:00`;
        },
        label: (ctx: any) => {
          if (ctx.datasetIndex === 0) return ` Ventas: ${fmt(ctx.parsed.y)}`;
          return ` Transacciones: ${ctx.parsed.y}`;
        },
      },
    },
  },
  scales: {
    x: {
      grid: { display: false },
      ticks: { font: { size: 10 } },
    },
    y: {
      beginAtZero: true,
      grid: { color: "rgba(0,0,0,0.05)" },
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
    y2: {
      display: false,
      beginAtZero: true,
      position: "right" as const,
    },
  },
};

const formattedChartData = computed(() => {
  const hourlyRevenue = Array(24).fill(0);
  const hourlyCount = Array(24).fill(0);

  props.chartData.forEach((sale) => {
    // timestamp from Go is local time (after timezone fix), parse safely
    const h = new Date(sale.timestamp.replace(" ", "T")).getHours();
    hourlyRevenue[h] += sale.total;
    hourlyCount[h]++;
  });

  const labels = Array.from({ length: 24 }, (_, i) =>
    String(i).padStart(2, "0")
  );

  // Build gradient lazily using a canvas reference
  const gradient = {
    backgroundColor: "rgba(37,99,235,0.15)",
    borderColor: "rgba(37,99,235,0.9)",
  };

  return {
    labels,
    datasets: [
      {
        label: "Ingresos",
        data: hourlyRevenue,
        fill: true,
        tension: 0.4,
        pointRadius: 2,
        pointHoverRadius: 5,
        borderWidth: 2,
        ...gradient,
        yAxisID: "y",
      },
      {
        label: "Transacciones",
        data: hourlyCount,
        fill: false,
        tension: 0.4,
        pointRadius: 0,
        borderWidth: 1.5,
        borderColor: "rgba(16,185,129,0.7)",
        borderDash: [4, 3],
        yAxisID: "y2",
      },
    ],
  };
});
</script>

<template>
  <div class="h-full w-full">
    <Line
      v-if="chartData.length > 0"
      :data="formattedChartData"
      :options="chartOptions"
    />
    <div v-else class="flex items-center justify-center h-full text-sm text-muted-foreground">
      No hay ventas registradas para mostrar.
    </div>
  </div>
</template>
