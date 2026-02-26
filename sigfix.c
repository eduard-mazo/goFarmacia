/* sigfix.c — LD_PRELOAD shim: adds SA_ONSTACK to every sigaction() call.
 *
 * Go ≥ 1.23 panics when it receives a signal whose handler was installed
 * without SA_ONSTACK (runtime.sigNotOnStack). WebKit's JavaScriptCore
 * installs SIGSEGV/SIGBUS handlers without that flag for JIT null-checks.
 * This shim intercepts every sigaction() call in the process and adds
 * SA_ONSTACK automatically, making GTK/WebKit/JSC handlers Go-compatible.
 *
 * Build:
 *   gcc -shared -fPIC -O2 -o libsigfix.so sigfix.c -ldl
 * Use:
 *   LD_PRELOAD=./libsigfix.so ./goFarmacia
 * The launcher script goFarmacia.sh does this automatically.
 */
#define _GNU_SOURCE
#include <signal.h>
#include <dlfcn.h>

typedef int (*real_sigaction_t)(int, const struct sigaction *, struct sigaction *);

int sigaction(int signum, const struct sigaction *act, struct sigaction *oldact) {
    static real_sigaction_t _real = NULL;
    if (!_real)
        _real = (real_sigaction_t)dlsym(RTLD_NEXT, "sigaction");

    if (act != NULL) {
        struct sigaction patched = *act;
        patched.sa_flags |= SA_ONSTACK;
        return _real(signum, &patched, oldact);
    }
    return _real(signum, act, oldact);
}
