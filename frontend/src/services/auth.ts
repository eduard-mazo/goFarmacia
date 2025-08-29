import { ref, computed } from "vue";
import { useRouter } from "vue-router";

// Simulamos una base de datos de usuarios
const USERS_DB = [
  {
    id: 1,
    name: "Admin User",
    email: "admin@example.com",
    password: "password123",
  },
];

interface User {
  id: number;
  name: string;
  email: string;
}

const currentUser = ref<User | null>(null);

// Intentamos cargar el usuario desde localStorage al iniciar
const userFromStorage = localStorage.getItem("user");
if (userFromStorage) {
  currentUser.value = JSON.parse(userFromStorage);
}

export function useAuth() {
  const router = useRouter();

  const isAuthenticated = computed(() => !!currentUser.value);

  const login = (email: string, password: string): Promise<User> => {
    return new Promise((resolve, reject) => {
      setTimeout(() => {
        // Simula una llamada a API
        const user = USERS_DB.find(
          (u) => u.email === email && u.password === password
        );
        if (user) {
          const userData = { id: user.id, name: user.name, email: user.email };
          currentUser.value = userData;
          localStorage.setItem("user", JSON.stringify(userData));
          resolve(userData);
        } else {
          reject(new Error("Credenciales inválidas"));
        }
      }, 500);
    });
  };

  const register = (
    name: string,
    email: string,
    password: string
  ): Promise<User> => {
    return new Promise((resolve, reject) => {
      setTimeout(() => {
        const userExists = USERS_DB.find((u) => u.email === email);
        if (userExists) {
          reject(new Error("El correo electrónico ya está registrado"));
          return;
        }
        const newUser = { id: Date.now(), name, email, password };
        USERS_DB.push(newUser);

        const userData = {
          id: newUser.id,
          name: newUser.name,
          email: newUser.email,
        };
        currentUser.value = userData;
        localStorage.setItem("user", JSON.stringify(userData));
        resolve(userData);
      }, 500);
    });
  };

  const logout = () => {
    currentUser.value = null;
    localStorage.removeItem("user");
    router.push("/login");
  };

  return {
    user: currentUser,
    isAuthenticated,
    login,
    register,
    logout,
  };
}
