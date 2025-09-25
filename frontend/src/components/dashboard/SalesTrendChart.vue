<script setup lang="ts">
import { Line } from "vue-chartjs";
import {
  Chart as ChartJS,
  Title,
  Tooltip,
  Legend,
  LineElement,
  CategoryScale,
  LinearScale,
  PointElement,
} from "chart.js";
import type { PropType } from "vue";

// Registrar los componentes necesarios de Chart.js
ChartJS.register(
  Title,
  Tooltip,
  Legend,
  LineElement,
  CategoryScale,
  LinearScale,
  PointElement
);

// Definir la estructura de los datos que esperamos
interface VentaHora {
  hora: number;
  total: number;
}

// Props del componente
const props = defineProps({
  chartData: {
    type: Array as PropType<VentaHora[]>,
    required: true,
  },
});

// Opciones de configuración del gráfico
const chartOptions = {
  responsive: true,
  maintainAspectRatio: false,
  plugins: {
    legend: {
      display: false, // Ocultamos la leyenda para un look más limpio
    },
    tooltip: {
      callbacks: {
        label: function (context: any) {
          let label = context.dataset.label || "";
          if (label) {
            label += ": ";
          }
          if (context.parsed.y !== null) {
            label += new Intl.NumberFormat("es-CO", {
              style: "currency",
              currency: "COP",
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
  },
};

// Formatear los datos para que Chart.js los entienda
const formattedChartData = {
  labels: props.chartData.map((d) => `${String(d.hora).padStart(2, "0")}:00`),
  datasets: [
    {
      label: "Ventas",
      backgroundColor: "#10b981", // Color de la línea
      borderColor: "#10b981",
      data: props.chartData.map((d) => d.total),
      tension: 0.2, // Ligeramente curvado para suavidad
    },
  ],
};
</script>

<template>
  <div class="h-full w-full">
    <Line :data="formattedChartData" :options="chartOptions" />
  </div>
</template>
