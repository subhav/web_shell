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

source ./complete.sh

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
# Responses are printed to stdout as JSON and delimited by a newline.
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
#
# Style guidelines:
# -   Stick to builtins as much as possible, such as for string manipulation
# -   If you need to run an external command, do so in a subshell, to avoid
#     messing with the state of running jobs.
################################################################################

unset LD_PRELOAD
unset SETPGRP_FD

# Enable job control
set -m
# (Even if we don't do this, as long as you start bash with `-i`, process groups
# still seem to get created unless you explicitly disable monitor mode with
# `set +m`. Though, bash explicitly says "no job control" and monitor mode is
# reportedly turned off,  Weird.)
# Notify (`set -b`) does not seem to work in a script like this.

# Using return works because we're using source instead of eval.
# Without this trap, bash exits if a child process gets a SIGINT, too.
trap "return 130" SIGINT

# Make the ERR trap work in functions
set -o errtrace

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
	# Save the command to history. This saves multi-line history literally, the
	# same as using the shopt "lithist".
	[[ $SHELLOPTS =~ (^|:)history($|:) ]] && history -s "$1"

	# Propagate cancellations out of functions. The shopt "errtrace" needs to
	# be on for this to work.
	# Check FUNCNAME to prevent it from short-circuiting after the source
	# command itself.
	trap '[[ $? == 130 ]] && [[ ${FUNCNAME[0]} != __cmd_run ]] && return 130' ERR

	__cmd_restore_status "$__cmd_last_status" # Reset $? for the eval

	# The command group forces the syntax to be checked before execution.
	#
	# Using source instead of eval makes `return` not break things
	# But, now the echo runs in a subshell and there's an extra pipe :/
	#
	# TODO: Redirections in the eval directly on the command group would make
	# syntax errors report to the outer stderr. But, it would also make it
	# easier to break and accidentally write to the script's stdout/err. Those
	# syntax errors get mangled by the command group anyway.
	#
	# The script's stdio gets temporarily copied to new fds (23-25) and the
	# saved fds get copied even though we mark them as closed here.
	# So, fds 0-2,10-12(copies of 20-22),23-25(copies of 0-2),63(proc subst) are
	# all set in the eval. However, all the extra fds are set to close on exec
	# this way.
	source <(echo "{
$1
}") <&20- >&21- 2>&22-

	__cmd_last_status=$? # Capture $?
	# Should we capture $PIPESTATUS? Is it possible to restore it?

	trap - ERR

	# TODO: Discover new jobs and report them, so the UI knows if the command
	#       has completed and what to wait for.

	# Make $PWD a more valid JSON string by escaping quotes and backslashes.
	local dir="${PWD//\\/\\\\}"
	dir="${dir//\"/\\\"}"
	dir="${dir//$'\n'/\n}"
	dir="${dir//$'\t'/\t}"

	echo "{\"Done\": true, \"Exit\": $__cmd_last_status, \"Dir\": \"$dir\"}"
}

__cmd_complete() {
	local compgen_out
	compgen_out=$(__completion_print "$*")
	compgen_out="${compgen_out//$'\n'/\\n}"
	compgen_out="${compgen_out// /}"
	#echo "{\"Complete\": \"$compgen_out\"}" >&2
    # handle "no completion" as a single empty completion
    # otherwise we have to detect if a completion command was run or not
	echo "{\"Done\": true, \"Complete\": \"$compgen_out\"}"
}
# Main Loop
#
# Turn on history in the same line as the loop so that it's the last command in
# the script.
# This won't write back multi-line commands correctly without HISTTIMEFORMAT
# set, but we can leave it to the user to do that.
[[ -v __cmd_writeback_history ]] && set -o history; \
# Without the IFS, it might split the arg
# TODO: remove \n as (trimmable) IFS aswell and use Fprint instead of Fprintln
while IFS=$'\n\0' read -r -d $'\0' method args; do
	case $method in
	"stdio")
		# Reset IFS in case it gets overridden.
		IFS=$' \t\n' __cmd_stdio $args
		;;
	"run")
		__cmd_run "$args"
		;;
	"complete")
		__cmd_complete "$args"
		;;
	"dir")
		dir="${PWD//\\/\\\\}"
		dir="${dir//\"/\\\"}"
		dir="${dir//$'\n'/\n}"
		dir="${dir//$'\t'/\t}"
		echo "{\"Dir\": \"$dir\"}"
		;;
	# For debugging
	"vars")
		declare -p
		;;
	"jobs")
		jobs -l
		;;
	"lsof")
		lsof -p $PPID
		lsof -p $$
		;;
	"exit")
		exit 0
		;;
	*)
		echo "Unknown method: $method" >&2
		;;
	esac
done
