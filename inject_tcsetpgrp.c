#include <unistd.h>
#include <stdio.h>
#include <stdlib.h>

// The functions here are called after redirections are applied.
// TODO: Hardcoding fd 24 is fragile. Either copy fd 1 in the script instead
//       or look for the right fd in the undo stack.
#define STDOUT_FD 24

// TODO: Print stuff in debug builds

// Need to unset LD_PRELOAD in the command server script instead of doing it here.
//
// The problem is bash redefines `unsetenv` to call its internal
// `unbind_variable`, so calling it in an __attribute__((constructor)) before
// bash has created internal variables does nothing.
//
// If we wanted to clear LD_PRELOAD here, we'd have to change environ directly.
// More info at: https://stackoverflow.com/questions/3275015/ld-preload-affects-new-child-even-after-unsetenvld-preload

static pid_t last_pgrp = 0;

pid_t tcgetpgrp(int fd) {
    if (last_pgrp == 0) {
        pid_t pgid = getpgid(0);
//        dprintf(STDOUT_FD, "[INJECT get first %d]", pgid);
        return pgid;
    }
//    dprintf(STDOUT_FD, "[current pgid %d]", getpgid(0));
//    dprintf(STDOUT_FD, "[INJECT get %d]", last_pgrp);
    return last_pgrp;
}

int tcsetpgrp(int fd, pid_t pgrp) {
//    dprintf(STDOUT_FD, "[INJECT set %d]", pgrp);
    dprintf(STDOUT_FD, "{\"Pgid\": %d}\n", pgrp);
    last_pgrp = pgrp;
    return 0;
}
