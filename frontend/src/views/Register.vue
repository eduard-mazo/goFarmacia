<script lang="ts">
export const description = "A login page with form and image.";
</script>

<script setup lang="ts">
import RegisterForm from "@/components/auth/RegisterForm.vue";
</script>

<template>
  <div class="h-screen w-screen overflow-hidden">
    <RegisterForm />
  </div>
</template>
