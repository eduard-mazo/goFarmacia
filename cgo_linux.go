//go:build linux

package main

/*
#include <stdlib.h>

// This constructor runs before Go's runtime, GTK, and WebKit initialize.
//
// Problem: JavaScriptCore (WebKit) installs a SIGSEGV handler WITHOUT the
// SA_ONSTACK flag. The handler is used to detect null-pointer dereferences
// in JIT-compiled JavaScript. Go ≥ 1.23 panics if it receives any signal
// whose current handler was not installed with SA_ONSTACK.
//
// Fix: move JSC's GC signal away from SIGUSR1 (signal 10), which Go also
// uses for async goroutine preemption, and force the X11 GDK backend so
// the Wayland compositor layer does not install additional bare handlers.
static void __attribute__((constructor)) earlyGtkSignalSetup(void) {
    setenv("JSC_SIGNAL_FOR_GC", "12", 0);     // use SIGUSR2 instead of SIGUSR1
    setenv("GDK_BACKEND", "x11", 0);          // avoid Wayland compositor signals
}
*/
import "C"
