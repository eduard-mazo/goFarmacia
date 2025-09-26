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
import { computed } from "vue"; // Se importa 'computed' para mejorar la reactividad

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
          if (context[0] && props.chartData) {
            const date = new Date(
              props.chartData[context[0].dataIndex]!.timestamp
            );
            return `Venta a las ${date.toLocaleTimeString("es-CO", {
              hour: "2-digit",
              minute: "2-digit",
              second: "2-digit",
            })}`;
          }
          return "";
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
      ticks: {
        maxRotation: 70,
        minRotation: 70,
      },
    },
  },
};

const formattedChartData = computed(() => {
  if (!props.chartData || props.chartData.length === 0) {
    return {
      labels: [],
      datasets: [
        {
          label: "Venta",
          backgroundColor: "#10b981",
          borderColor: "#059669",
          borderWidth: 1,
          data: [],
        },
      ],
    };
  }

  // Si hay datos, los formatea como antes
  return {
    labels: props.chartData.map((d) =>
      new Date(d.timestamp).toLocaleTimeString("es-CO", {
        hour: "2-digit",
        minute: "2-digit",
      })
    ),
    datasets: [
      {
        label: "Venta",
        backgroundColor: "#10b981",
        borderColor: "#059669",
        borderWidth: 1,
        data: props.chartData.map((d) => d.total),
      },
    ],
  };
});
</script>

<template>
  <div class="h-full w-full">
    <Bar :data="formattedChartData" :options="chartOptions" />
  </div>
</template>
