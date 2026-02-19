// Global test setup — mock Wails runtime
// Wails injects window['go'] at runtime; in tests we mock the binding modules.

import { vi } from "vitest";

// Clear localStorage between tests
beforeEach(() => {
  localStorage.clear();
});

// Suppress vue-router warnings in tests
vi.mock("vue-router", async () => {
  const actual = await vi.importActual<typeof import("vue-router")>("vue-router");
  return {
    ...actual,
    useRouter: () => ({
      push: vi.fn(),
      replace: vi.fn(),
      back: vi.fn(),
    }),
    useRoute: () => ({
      path: "/dashboard",
      meta: {},
      matched: [],
    }),
  };
});
