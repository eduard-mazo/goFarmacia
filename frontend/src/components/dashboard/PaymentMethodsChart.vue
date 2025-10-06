<script setup lang="ts">
import { Doughnut } from "vue-chartjs";
import { Chart as ChartJS, ArcElement, Tooltip, Legend } from "chart.js";
import { computed } from "vue";
import type { PropType } from "vue";

ChartJS.register(ArcElement, Tooltip, Legend);

interface MetodoPago {
  metodo_pago: string;
  count: number;
}

const props = defineProps({
  chartData: {
    type: Array as PropType<MetodoPago[]>,
    required: true,
  },
});

const chartOptions = {
  responsive: true,
  maintainAspectRatio: false,
  plugins: {
    legend: {
      position: "top" as const,
    },
    tooltip: {
      callbacks: {
        label: function (context: any) {
          let label = context.label || "";
          if (label) {
            label += ": ";
          }
          if (context.parsed !== null) {
            label += `${context.parsed} transacciones`;
          }
          return label;
        },
      },
    },
  },
};

const formattedChartData = computed(() => {
  const labels = props.chartData.map((item) => item.metodo_pago);
  const data = props.chartData.map((item) => item.count);

  return {
    labels,
    datasets: [
      {
        backgroundColor: [
          "#4ade80",
          "#fbbf24",
          "#60a5fa",
          "#f87171",
          "#c084fc",
        ],
        data,
      },
    ],
  };
});
</script>

<template>
  <Doughnut :data="formattedChartData" :options="chartOptions" />
</template>
