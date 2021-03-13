#!/bin/bash -i
if [[ $SHELLOPTS =~ (^|:)history($|:) ]]; then
	if [[ $HISTCMD == 3 ]]; then
		# history -d -2 fails if there are only two entries :-/
		history -c
	else
		# Clear last two entries (-2 to -1), hashbang and this if statement
		history -d -2--1
	fi
	set +o history # Turn off history (for lines in this script)
	__cmd_writeback_history=1
fi

################################################################################
# Simple debug wrapper which exposes the shell state to a controlling process.
#
# This script can be run interactively using:
#     sed -u "s/\$/\x0/g" | setsid bash -i command_server.sh
#
# NOTE: If you use setsid to make bash a session leader and open a terminal not
#       allocated to a session already (i.e. using the stdio method), then the
#       OS will allocate that terminal as the controlling terminal of the new
#       session. Bash will close if the terminal is removed.
#       (Apparently, zsh opens files with O_NOCTTY, but I haven't tested this.)
#
# Each request begins with a method name and is terminated with a NUL character.
#
# Methods:
#     stdio <stdin> <stdout> <stderr>
#     run <command>
#         Returns exit status of command
#
# It will always be really easy to break. For example,
# -   messing with the debug vars (e.g. unset __cmd_run)
# -   running exec ("exec bash", though maybe this should be intended behavior)
# -   writing to fd 23 or 24
#
# Security isn't really a concern, but reliability is.
################################################################################

# Enable job control
set -m
# (Even if we don't do this, as long as you start bash with `-i`, then process
# groups still seem to still get created unless you explicitly disable monitor
# mode with `set +m`. Though, bash explicitly says "no job control" and monitor
# mode is reportedly turned off,  Weird.)
# Notify (`set -b`) does not seem to work in a script like this.

# Open command's stdin/out/err on fds 20-22
__cmd_stdio() {
	# Open files
	exec 20<"$1" 21>"$2" 22>"$3"
}

__cmd_restore_status() {
	return "$1"
}

__cmd_last_status=0
__cmd_run() {
	[[ $SHELLOPTS =~ (^|:)history($|:) ]] && history -s "$1"

	__cmd_restore_status "$__cmd_last_status" # Reset $? for the eval

	# The command group forces the syntax to be checked before execution.
	#
	# TODO: Redirections in the eval directly on the command group would make
	# syntax errors report to the outer stderr. But, it would also make it
	# easier to break and accidentally write to the script's stdout/err. Those
	# syntax errors get mangled by the command group anyway.
	#
	# The script's stdio gets temporarily copied to new fds (23-25) and the
	# saved fds get copied even though we mark them as closed here.
	# So, fds 0-2,10-12(copies of 20-22),23-25(copies of 0-2) are all set in
	# the eval. However, all the extra fds are set to close on exec this way.
	eval "{
$1
}" <&20- >&21- 2>&22-
	__cmd_last_status=$? # Capture $?
	# (No need to capture $!, because we don't run background jobs in this script)

	echo "$__cmd_last_status"
}

# Main Loop
#
# Turn on history in the same line as the loop so that it's the last command in
# the script.
# This won't write back multi-line commands correctly without HISTTIMEFORMAT
# set, but we can leave it to the user to do that.
# This is the same behavior as setting the shopt "lithist", with or without
# HISTTIMEFORMAT.
[[ -v __cmd_writeback_history ]] && set -o history
while read -r -d $'\0' method args; do
	case $method in
	"stdio")
		# Reset IFS in case it gets overridden.
		IFS=$' \t\n' __cmd_stdio $args
		;;
	"run")
		__cmd_run "$args"
		;;
	"exit")
		exit 0
		;;
	*)
		echo "Unknown method: $method" >&2
		;;
	esac
done
