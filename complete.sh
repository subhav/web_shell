#!/usr/bin/env bash
# Source this file and then...

# TODO: use after running the function
compopt () {
        #echo "Compopt run: $*" >&2
        :
}

__completion_export_vars() {
        local prog args
        export COMP_LINE="$*"
        prog="${COMP_LINE%% *}"
        args="${COMP_LINE#* }"
        # split completion line by space/newline
        export COMP_WORDS=($COMP_LINE)
        # "doubletap" expansion
        export COMP_TYPE='9'
        # always at the end of the line
        export COMP_POINT=${#COMP_LINE}
        # handle space at the end of string
        if [[ "${COMP_LINE: -1}" = " " ]]; then
                export COMP_CWORD=${#COMP_WORDS[@]}
        else
                export COMP_CWORD=$((${#COMP_WORDS[@]}-1))
        fi
}

__completion_load() {
        # try to load all possible completions
        if [[ -n "$XDG_DATA_DIRS" ]]; then
                while read -d: -r path; do
                        for com_path in "$path/bash-completion/completions" "${path}/bash_completions.d"; do
                                if [[ -d "$com_path" ]]; then
                                        for file in "$com_path"/*; do
                                                . "$file"
                                        done
                                fi
                        done
                # TODO: Maybe there's also usr/lib paths depending on OS
                done <<< "/etc:${XDG_DATA_DIRS}"
        fi
}


__completion_print() {
        local prog args
        __completion_export_vars "$*"
        prog="${COMP_LINE%% *}"
        args="${COMP_LINE#* }"
        #echo "line is '$COMP_LINE' (${#COMP_LINE}) and prog is '$prog' (${#prog})" >&2
        # If we're still on first word completion
        if [[ "${#COMP_LINE}" -eq "${#prog}" ]]; then
                #echo "complete first word" >&2
                compgen -abc $prog | sort -u
                return
        fi

        #example out: compgen -o bashdefault -o default -o nospace -F __git_wrap__git_main git s
        comp_command="$(complete -p "$prog")"
        if [[ -z "$comp_command" ]]; then
                #echo no completions >&2
                return
        fi
        is_func=false
        for arg in $comp_command; do
                # just run -F functions separately
                if $is_func; then
                        export MY_FUNCNAME="$arg"
                        is_func=false
                        echo eval "$arg" "$prog" "$COMP_CWORD" "${COMP_WORDS[$((${#COMP_WORDS[@]}-2))]}" >&2
                        eval "$arg" "$prog" "$COMP_CWORD" "${COMP_WORDS[$((${#COMP_WORDS[@]}-2))]}"
                        (IFS=$'\n'; echo "${COMPREPLY[*]}")
                        # TODO: compgen filters with -X, prefix/suffix with -P and -S
                        # E.g. compgen -P abc_ -F _myfunc
                        # TODO: if empty result, use the -o options (if -o bashdefault or -o default)
                        #exit
                        return
                fi
                case "$arg" in
                        # next argument is a function we wanna execute
                        "-F" | "-C")
                                is_func=true
                                ;;
                        # run with parameters
                        "-u" | "-a" | "-v")
                                compgen "$arg" "$args"
                                # If we vomplete comp
                                ;;
                esac
                #echo $arg
        done
        comp_command="${comp_command/complete/compgen}"
        eval $comp_command
}

__completion_load
