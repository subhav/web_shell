#include <unistd.h>
#include <stdio.h>

// The functions here are called after redirections are applied.
// TODO: Hardcoding fd 24 is fragile. Either copy fd 1 in the script instead
//       or look for the right fd in the bash undo stack.
// Fish does the sane thing and doesn't move its own file descriptors around.
// Script stdout is still fd 1 and command's out,err,in are fds 6,7,8
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

pid_t tcgetpgrp(int fd) {
//    dprintf(STDOUT_FD, "[INJECT get %d]", getpgid(0));
    return getpgid(0);
}

int tcsetpgrp(int fd, pid_t pgrp) {
//    dprintf(STDOUT_FD, "[INJECT set %d]", pgrp);
    dprintf(STDOUT_FD, "{\"Pgid\": %d}\n", pgrp);
    return 0;
}
