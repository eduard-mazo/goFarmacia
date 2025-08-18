import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { useRouter } from 'vue-router'
import { backend } from '../../wailsjs/go/models'
import { LoginVendedor } from '../../wailsjs/go/backend/Db'

export const useAuthStore = defineStore('auth', () => {
  const router = useRouter()
  const vendedor = ref<backend.Vendedor | null>(null)

  const isAuthenticated = computed(() => !!vendedor.value)
  const vendedorNombre = computed(() => vendedor.value ? `${vendedor.value.Nombre} ${vendedor.value.Apellido}` : 'N/A')
  const vendedorId = computed(() => vendedor.value?.id)

  async function login(credenciales: backend.LoginRequest) {
    try {
      const vendedorData = await LoginVendedor(credenciales)
      vendedor.value = vendedorData
      router.push({ name: 'POS' })
    } catch (error) {
      console.error('Login failed:', error)
      throw error // Lanza el error para que el componente de login lo maneje
    }
  }

  function logout() {
    vendedor.value = null
    router.push({ name: 'Login' })
  }

  return {
    vendedor,
    isAuthenticated,
    vendedorNombre,
    vendedorId,
    login,
    logout,
  }
})