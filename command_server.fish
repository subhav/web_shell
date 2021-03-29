# I should try to abuse fish's lack of subshells somehow. Things that look like subshells
# in fish like command substitutions run synchronously and change state.

set -e LD_PRELOAD

function __cmd_trap_int --on-signal INT
    return
end

status --job-control full

set __cmd_stdin ""
set __cmd_stdout ""
set __cmd_stderr ""
function __cmd_stdio -a args
	string split -n " " $args | read --null __cmd_stdin __cmd_stdout __cmd_stderr
	# Before fish 3.1:
#	string split -n $args | read --null __cmd_stdin __cmd_stdout __cmd_stderr _
end

function __cmd_restore_status
	return $argv[1]
end

set __cmd_last_status $status
# eval in fish used to be a function which piped the command into source.
# We can do the same thing, simplifying its behavior.
function __cmd_run -a command --no-scope-shadowing
	__cmd_restore_status $__cmd_last_status
	echo "begin $command "\n" ;end <$__cmd_stdin" | source >$__cmd_stdout 2>$__cmd_stderr
	set -g __cmd_last_status $status
	echo "{\"Exit\": $__cmd_last_status}"
end

while read --null method args
	switch $method
		case "stdio"
			__cmd_stdio $args
		case "run"
			__cmd_run $args
		case "exit"
			exit 0
		case "*"
			echo "Unknown method: $method" >&2
	end
end
