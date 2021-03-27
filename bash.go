//go:generate gcc -shared -fPIC -o inject_tcsetpgrp.so inject_tcsetpgrp.c

package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path"
	"strconv"
)

var (
	shellPath   = flag.String("shell", "/bin/bash", "Path to shell interpreter")
	shellScript = flag.String("shell_script", "command_server.sh", "Command server script")
)

type BashShell struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout io.ReadCloser

	exitCh chan int
	fgPgid int
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
//		//Setsid: true,		// new session doesn't work because it steals term
//		Setpgid: true,		// creating a new pgid doesn't work if pg leader
//		Pgid: 0,
//	}

	cwd, _ := os.Getwd()
	injectPath := path.Join(cwd, "inject_tcsetpgrp.so")
	shell.cmd.Env = append(os.Environ(), "LD_PRELOAD="+injectPath)

	shell.stdin, _ = shell.cmd.StdinPipe()
	shell.stdout, _ = shell.cmd.StdoutPipe()
	shell.cmd.Stderr = os.Stderr
	err := shell.cmd.Start()
	if err != nil {
		return nil, err
	}

	shell.exitCh = make(chan int)
	go shell.readLoop()

	return shell, nil
}

// Read different types of messages from the command server's stdout.
func (b *BashShell) readLoop() {
	dec := json.NewDecoder(b.stdout)
	for dec.More() {
		var m struct {
			Pgid int
			Exit int
		}
		err := dec.Decode(&m)
		if err != nil {
			close(b.exitCh)
			log.Println(err)
			//return
		}

		// Switch on message type
		if m.Pgid > 0 {
			log.Print("Foreground PGID set to ", m.Pgid)
			b.fgPgid = m.Pgid
		} else {
			b.exitCh <- m.Exit
			b.fgPgid = m.Pgid
		}
	}

	close(b.exitCh)
}

func readstringloop(r io.Reader, exitCh chan<- string) {
//	for {
//		exit, err := bufio.NewReader(r).ReadString('\n')
//		if err != nil {
//			close(exitCh)
//			log.Println(err)
//			return
//		}
//		exit = strings.TrimSpace(exit)
//		exitCh <- exit
//	}
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
	exit := <-b.exitCh
	if exit != 0 {
		return errors.New("exit code: " + strconv.Itoa(exit))
	}
	return nil
}
