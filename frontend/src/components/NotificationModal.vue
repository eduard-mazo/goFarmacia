<template>
  <div
    v-if="show"
    class="fixed top-5 right-5 z-50 transition-transform duration-300 ease-out"
    :class="[
      containerClass,
      show ? 'translate-x-0 opacity-100' : 'translate-x-full opacity-0',
    ]"
  >
    <div
      class="flex items-center p-4 rounded-lg shadow-lg"
      :class="colorClasses"
    >
      <div class="mr-3">
        <svg
          v-if="type === 'success'"
          class="w-6 h-6 text-green-500"
          fill="currentColor"
          viewBox="0 0 20 20"
        >
          <path
            fill-rule="evenodd"
            d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z"
            clip-rule="evenodd"
          />
        </svg>
        <svg
          v-if="type === 'error'"
          class="w-6 h-6 text-red-500"
          fill="currentColor"
          viewBox="0 0 20 20"
        >
          <path
            fill-rule="evenodd"
            d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z"
            clip-rule="evenodd"
          />
        </svg>
      </div>
      <div class="text-sm font-medium">
        {{ message }}
      </div>
      <button
        @click="$emit('close')"
        class="ml-4 -mr-1 p-1 rounded-md focus:outline-none focus:ring-2 focus:ring-white"
      >
        <svg class="w-4 h-4" fill="currentColor" viewBox="0 0 20 20">
          <path
            fill-rule="evenodd"
            d="M4.293 4.293a1 1 0 011.414 0L10 8.586l4.293-4.293a1 1 0 111.414 1.414L11.414 10l4.293 4.293a1 1 0 01-1.414 1.414L10 11.414l-4.293 4.293a1 1 0 01-1.414-1.414L8.586 10 4.293 5.707a1 1 0 010-1.414z"
            clip-rule="evenodd"
          />
        </svg>
      </button>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { computed } from "vue";

const props = defineProps<{
  show: boolean;
  message: string;
  type: "success" | "error";
}>();

defineEmits(["close"]);

const containerClass = computed(() => {
  return props.show
    ? "translate-x-0 opacity-100"
    : "translate-x-full opacity-0";
});

const colorClasses = computed(() => {
  if (props.type === "success") {
    return "bg-green-100 text-green-800";
  }
  if (props.type === "error") {
    return "bg-red-100 text-red-800";
  }
  return "bg-gray-100 text-gray-800";
});
</script>
