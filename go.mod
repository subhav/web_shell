module github.com/subhav/web_shell

go 1.15

//replace mvdan.cc/sh/v3 => ../mvdan-sh

require (
	github.com/alessio/shellescape v1.4.2 // indirect
	github.com/buildkite/terminal-to-html/v3 v3.6.1
	github.com/creack/pty v1.1.11
	github.com/evanw/esbuild v0.19.7
	golang.org/x/sync v0.0.0-20201207232520-09787c993a3a // indirect
	golang.org/x/term v0.0.0-20201117132131-f5c789dd3221 // indirect
	gopkg.in/alessio/shellescape.v1 v1.0.0-20170105083845-52074bc9df61
	mvdan.cc/sh/v3 v3.2.2
)
