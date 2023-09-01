package main

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"flag"
	"io"
	"log"
	"os"
	"os/exec"
	"strconv"
	"syscall"
)

var (
	fanosShellPath = flag.String("oils_path", "/usr/local/bin/osh", "Path to Oils interpreter (https://oils-for-unix.org/)")
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

func (b *FANOSShell) Complete(ctx context.Context, cmd io.Reader) ([]string, error) {
	return nil, errors.New("unimplemented")
}
