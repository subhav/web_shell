package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

var (
	shellPath   = flag.String("shell", "/bin/bash", "Path to shell interpreter")
	shellScript = flag.String("shell_script", "command_server.sh", "Command server script")
)

type BashShell struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout io.ReadCloser
}

func NewBashShell() (*BashShell, error) {
	shell := &BashShell{}

	shell.cmd = exec.Command(*shellPath, "-i", *shellScript)
	// Can't use setsid here. If bash is a session leader, then the operating
	// system will allocate any non-controlling terminal it opens as the
	// controlling terminal of that session. More concretely, if we call StdIO()
	// to have bash open a pts, then bash will exit if we ever close the master
	// end of the pty.
	// (This also means if webshell is running as a session leader, then it
	// should open any pts with O_NOCTTY set.)
	//	shell.cmd.SysProcAttr = &syscall.SysProcAttr{
	//		Setsid: true,
	//	}
	shell.stdin, _ = shell.cmd.StdinPipe()
	shell.stdout, _ = shell.cmd.StdoutPipe()
	shell.cmd.Stderr = os.Stderr
	err := shell.cmd.Start()
	if err != nil {
		return nil, err
	}

	return shell, nil
}

func (b *BashShell) Close() error {
	b.stdin.Close()
	b.stdout.Close()
	return b.cmd.Wait()
}

func (b *BashShell) StdIO(in, out, err *os.File) error {
	fmt.Fprintln(b.stdin, "stdio")
	for _, f := range []*os.File{in, out, err} {
		if f == nil {
			fmt.Fprintln(b.stdin, "/dev/null")
		} else {
			fmt.Fprintln(b.stdin, f.Name())
		}
	}
	b.stdin.Write([]byte{0})
	return nil
}

func (b *BashShell) Run(cmd io.Reader) error {
	fmt.Fprintln(b.stdin, "run")
	io.Copy(b.stdin, cmd)
	b.stdin.Write([]byte{0})
	// This read blocks. We should instead wait until either:
	// - we're able to read or get EOF
	// - bash is trying to read from us?
	// - a context expires
	exit, err := bufio.NewReader(b.stdout).ReadString('\n')
	if err != nil {
		return err
	}
	exit = strings.TrimSpace(exit)
	if exit != "0" {
		return errors.New("exit code: " + exit)
	}
	return err
}
