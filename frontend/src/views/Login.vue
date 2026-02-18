<script lang="ts">
export const description = "A login page with form and image.";
</script>

<script setup lang="ts">
import LoginForm from "@/components/auth/LoginForm.vue";
</script>

<template>
  <div class="h-screen w-screen overflow-hidden">
    <LoginForm />
  </div>
</template>
