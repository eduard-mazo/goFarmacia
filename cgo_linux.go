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
    // Force X11 GDK backend so the Wayland compositor layer does not
    // install additional bare signal handlers that conflict with Go ≥ 1.23.
    // The SA_ONSTACK fix for WebKit/JSC is handled by LD_PRELOAD=libsigfix.so
    // (see sigfix.c and the goFarmacia.sh launcher).
    setenv("GDK_BACKEND", "x11", 0);
}
*/
import "C"
