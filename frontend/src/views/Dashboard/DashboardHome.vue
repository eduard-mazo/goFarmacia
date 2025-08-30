<script setup lang="ts">
import { toast } from 'vue-sonner'
import { Button } from '@/components/ui/button'

import { ResetearTodaLaData } from "../../../wailsjs/go/backend/Db";

async function resetDB() {
  toast.warning('BASE DE DATOS', {
    description: "Iniciando limpieza"
  })
  var res = await ResetearTodaLaData();
  toast.success('BASE DE DATOS', {
    description: res
  })

}
</script>

<template>
  <Button variant="outline" @click="resetDB">
    Add to calendar
  </Button>
</template>