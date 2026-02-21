import { defineStore } from "pinia";
import { ref } from "vue";
import {
  GetDBStatus,
  TestDBConnection,
  ConfigurarDB,
} from "@/../wailsjs/go/backend/Db";

export const useDBStore = defineStore("db", () => {
  const connected = ref(false);
  const setupMode = ref(false);
  const message = ref("");
  const dsnHint = ref("");
  let pollInterval: ReturnType<typeof setInterval> | null = null;

  async function fetchStatus() {
    try {
      const s = await GetDBStatus();
      connected.value = s.Connected;
      setupMode.value = s.SetupMode;
      message.value = s.Message;
      dsnHint.value = s.DSNHint;
    } catch {
      connected.value = false;
      setupMode.value = false;
      message.value = "Error al obtener estado de la base de datos";
    }
  }

  function startPolling(intervalMs = 15000) {
    fetchStatus();
    pollInterval = setInterval(fetchStatus, intervalMs);
  }

  function stopPolling() {
    if (pollInterval !== null) {
      clearInterval(pollInterval);
      pollInterval = null;
    }
  }

  /** Returns null on success, or an error string. */
  async function testConnection(dsn: string): Promise<string | null> {
    try {
      await TestDBConnection(dsn);
      return null;
    } catch (e: unknown) {
      return String(e);
    }
  }

  /** Saves the DSN, creates the DB if needed, connects, and runs migrations. */
  async function configureDB(dsn: string): Promise<string | null> {
    try {
      await ConfigurarDB(dsn);
      await fetchStatus();
      return null;
    } catch (e: unknown) {
      return String(e);
    }
  }

  return {
    connected,
    setupMode,
    message,
    dsnHint,
    fetchStatus,
    startPolling,
    stopPolling,
    testConnection,
    configureDB,
  };
});
