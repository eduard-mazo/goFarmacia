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
import type { PropType } from "vue";
import { computed } from "vue";

ChartJS.register(
  Title,
  Tooltip,
  Legend,
  BarElement,
  CategoryScale,
  LinearScale
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

const chartOptions = {
  responsive: true,
  maintainAspectRatio: false,
  plugins: {
    legend: {
      display: false,
    },
    tooltip: {
      callbacks: {
        title: function (context: any) {
          const hour = context[0].label;
          return `Ventas entre ${hour} y las ${String(
            parseInt(hour.split(":")[0]) + 1
          ).padStart(2, "0")}:00`;
        },
        label: function (context: any) {
          let label = "Total: ";
          if (context.parsed.y !== null) {
            label += new Intl.NumberFormat("es-CO", {
              style: "currency",
              currency: "COP",
              minimumFractionDigits: 0,
            }).format(context.parsed.y);
          }
          return label;
        },
      },
    },
  },
  scales: {
    y: {
      beginAtZero: true,
      ticks: {
        callback: function (value: any) {
          return new Intl.NumberFormat("es-CO", {
            style: "currency",
            currency: "COP",
            maximumFractionDigits: 0,
          }).format(value);
        },
      },
    },
    x: {
      grid: {
        display: false, // Oculta la rejilla vertical para un look más limpio
      },
    },
  },
};

const formattedChartData = computed(() => {
  if (!props.chartData || props.chartData.length === 0) {
    return { labels: [], datasets: [] };
  }

  const hourlySales = Array(24).fill(0);

  // Agrupa las ventas por hora
  props.chartData.forEach((sale) => {
    const hour = new Date(sale.timestamp).getHours();
    hourlySales[hour] += sale.total;
  });

  // Crea las etiquetas para el eje X (e.g., "00:00", "01:00", ...)
  const labels = Array.from(
    { length: 24 },
    (_, i) => `${String(i).padStart(2, "0")}:00`
  );

  return {
    labels: labels,
    datasets: [
      {
        label: "Ventas por Hora",
        backgroundColor: "#10b981",
        borderColor: "#059669",
        borderRadius: 4,
        borderWidth: 1,
        data: hourlySales,
      },
    ],
  };
});
</script>

<template>
  <div class="h-full w-full">
    <Bar
      v-if="chartData.length > 0"
      :data="formattedChartData"
      :options="chartOptions"
    />
    <div v-else class="flex items-center justify-center h-full">
      <p class="text-sm text-muted-foreground">
        No hay ventas registradas para mostrar.
      </p>
    </div>
  </div>
</template>
