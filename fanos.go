package main

import (
	"bufio"
	"bytes"
	"context"
	"flag"
	"github.com/creack/pty"
	"gopkg.in/alessio/shellescape.v1"
	"io"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
)

var (
	fanosShellPath = flag.String("oil_path", "/usr/bin/oil", "Path to Oil shell interpreter")
)

type FANOSShell struct {
	cmd    *exec.Cmd
	socket *os.File

	in, out, err *os.File
}

func NewFANOSShell() (*FANOSShell, error) {
	shell := &FANOSShell{}
	shell.cmd = exec.Command(*fanosShellPath, "--headless")

	fds, err := syscall.Socketpair(syscall.AF_UNIX, syscall.SOCK_STREAM, 0)
	if err != nil {
		return nil, err
	}
	shell.socket = os.NewFile(uintptr(fds[0]), "fanos_client")
	server := os.NewFile(uintptr(fds[1]), "fanos_server")
	shell.cmd.Stdin = server
	shell.cmd.Stdout = server

	shell.cmd.Stderr = os.Stderr

	return shell, shell.cmd.Start()
}

func (s *FANOSShell) StdIO(in, out, err *os.File) error {
	// Save these for the next Run
	s.in, s.out, s.err = in, out, err
	if s.in == nil {
		s.in, _ = os.Open(os.DevNull)
	}
	if s.out == nil {
		s.out, _ = os.Open(os.DevNull)
	}
	if s.err == nil {
		s.err, _ = os.Open(os.DevNull)
	}

	return nil
}

// Run calls the FANOS EVAL method
func (s *FANOSShell) Run(ctx context.Context, r io.Reader) error {
	rights := syscall.UnixRights(int(s.in.Fd()), int(s.out.Fd()), int(s.err.Fd()))

	var buf bytes.Buffer
	buf.WriteString("EVAL ")
	_, err := io.Copy(&buf, r)
	if err != nil {
		return err
	}

	_, err = s.socket.Write([]byte(strconv.Itoa(buf.Len()) + ":"))
	if err != nil {
		return err
	}
	err = syscall.Sendmsg(int(s.socket.Fd()), buf.Bytes(), rights, nil, 0)
	if err != nil {
		return err
	}
	_, err = s.socket.Write([]byte(","))
	if err != nil {
		return err
	}

	// TODO: Actually read netstring instead of reading until ','
	sockReader := bufio.NewReader(s.socket)
	msg, err := sockReader.ReadString(',')
	if err != nil {
		return err
	}
	log.Println(msg)

	return nil
}

func (s *FANOSShell) Dir() string {
	return ""
}

func (s *FANOSShell) Complete(ctx context.Context, r CompletionReq) (*CompletionResult, error) {
	comps := CompletionResult{}
	comps.To = len(r.Text)
	//// STDIO stuff
	var stdout, stderr bytes.Buffer
	ptmx, pts, err := pty.Open()
	if err != nil {
		log.Println(err)
		return &comps, err
	}
	defer func() {
		ptmx.Close()
		pts.Close()
	}()
	go io.Copy(&stdout, ptmx)

	var pipe *os.File
	var rdPipe *os.File
	rdPipe, pipe, err = os.Pipe()
	if err != nil {
		log.Println(err)
		return &comps, err
	}
	go func() {
		io.Copy(&stderr, rdPipe)
		rdPipe.Close()
		pipe.Close()
	}()
	// Reset stdio of runner before running a new command
	err = shell.StdIO(nil, pts, pipe)
	if err != nil {
		log.Println(err)
		return &comps, err
	}

	//// Run stuff
	rights := syscall.UnixRights(int(s.in.Fd()), int(s.out.Fd()), int(s.err.Fd()))
	s.StdIO(nil, nil, nil)
	var buf bytes.Buffer
	buf.WriteString("EVAL ")
	_, err = io.Copy(&buf, strings.NewReader("compexport -c "+shellescape.Quote(r.Text)))
	if err != nil {
		log.Println(err)
		return nil, err
	}
	_, err = s.socket.Write([]byte(strconv.Itoa(buf.Len()) + ":"))
	if err != nil {
		log.Println(err)
		return nil, err
	}
	err = syscall.Sendmsg(int(s.socket.Fd()), buf.Bytes(), rights, nil, 0)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	_, err = s.socket.Write([]byte(","))
	if err != nil {
		log.Println(err)
		return nil, err
	}

	// TODO: Actually read netstring instead of reading until ','
	sockReader := bufio.NewReader(s.socket)
	msg, err := sockReader.ReadString(',')
	if err != nil {
		log.Println(err)
		return nil, err
	}
	log.Println(msg)

	completions := strings.Split(stdout.String(), "\n")

	comps.Options = make([]Completion, len(completions))
	for i, completion := range completions {
		log.Println(completion)
		if len(completion) > 2 {
			comps.Options[i] = Completion{
				Label: completion[1 : len(completion)-2],
			}
		}
	}
	return &comps, nil
}
