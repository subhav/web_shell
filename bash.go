//go:generate gcc -shared -fPIC -o lib/inject_tcsetpgrp.so lib/inject_tcsetpgrp.c

package main

import (
	"context"
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
	"syscall"
)

var (
	shellPath    = flag.String("shell", "/run/current-system/sw/bin/bash", "Path to shell interpreter")
	shellScript  = flag.String("shell_script", "command_server.sh", "Command server script")
	shellPreload *string
)

func init() {
	cwd, _ := os.Getwd()
	injectPath := path.Join(cwd, "lib/inject_tcsetpgrp.so")
	shellPreload = flag.String("shell_preload", injectPath, "Path to shared object to link into the shell")
}

type BashShell struct {
	dir string

	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout io.ReadCloser

	exitCh chan int
	fgPgid int
}

func NewBashShell() (*BashShell, error) {
	shell := &BashShell{}

	shell.cmd = exec.Command(*shellPath, "--login", "-i", *shellScript)
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

	shell.cmd.Env = append(os.Environ(), "LD_PRELOAD="+*shellPreload, "SETPGRP_FD=24")

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
			Done bool
			Pgid int
			Exit int
			Dir  string
		}
		err := dec.Decode(&m)
		// TODO: We can end up getting stuck looping on this error. Limit retries or ignore this "element"
		if err != nil {
			log.Println(err)
			continue
		}

		log.Printf("Parsed Message: %+v", m)
		if m.Pgid > 0 {
			b.fgPgid = m.Pgid
		}
		if m.Dir != "" {
			b.dir = m.Dir
		}
		if m.Done {
			b.exitCh <- m.Exit
		}
	}

	log.Print("Read loop ended")
	close(b.exitCh)
}

// Attempt to send a sigint to foreground pgrp
func (b *BashShell) cancel() {
	var err error
	if b.fgPgid > 0 {
		err = syscall.Kill(-b.fgPgid, syscall.SIGINT)
	} else {
		err = b.cmd.Process.Signal(os.Interrupt)
	}
	if err != nil {
		log.Print(err)
	}
}

func (b *BashShell) Close() error {
	b.stdin.Close()
	b.stdout.Close()
	return b.cmd.Wait()
}

func (b *BashShell) Dir() string {
	return b.dir
}

func (b *BashShell) StdIO(in, out, err *os.File) error {
	fmt.Fprintln(b.stdin, "stdio")
	for _, f := range []*os.File{in, out, err} {
		if f == nil {
			fmt.Fprintln(b.stdin, os.DevNull)
		} else {
			fmt.Fprintln(b.stdin, f.Name())
		}
	}
	b.stdin.Write([]byte{0})
	return nil
}

func (b *BashShell) Run(ctx context.Context, cmd io.Reader) error {
	// TODO: error handle quit process

	fmt.Fprintln(b.stdin, "run")
	io.Copy(b.stdin, cmd)
	b.stdin.Write([]byte{0})
	// What if?
	// - bash is trying to read from us
	// - bash dies
	select {
	// TODO: if exitCh is closed, this will still read a 0
	case exit := <-b.exitCh:
		if exit != 0 {
			return errors.New("exit code: " + strconv.Itoa(exit))
		}
	case <-ctx.Done():
		b.cancel()
		exit := <-b.exitCh
		if exit != 0 {
			return errors.New("exit code: " + strconv.Itoa(exit))
		}
	}
	return nil
}

func (b *BashShell) Complete(ctx context.Context, cmd io.Reader) error {
	// TODO: error handle quit process

	fmt.Fprintln(b.stdin, "complete")
	io.Copy(b.stdin, cmd)
	b.stdin.Write([]byte{0})
	// What if?
	// - bash is trying to read from us
	// - bash dies
	select {
	// TODO: if exitCh is closed, this will still read a 0
	case exit := <-b.exitCh:
		if exit != 0 {
			return errors.New("exit code: " + strconv.Itoa(exit))
		}
	case <-ctx.Done():
		b.cancel()
		exit := <-b.exitCh
		if exit != 0 {
			return errors.New("exit code: " + strconv.Itoa(exit))
		}
	}
	return nil
}
