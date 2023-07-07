package main

import (
	"context"
	"io"
	"log"
	"mvdan.cc/sh/v3/interp"
	"mvdan.cc/sh/v3/syntax"
	"os"
)

// Wraps the sh package into our (worse) Shell interface.
type GoShell struct {
	*interp.Runner
	*syntax.Parser
}

func NewGoShell() (*GoShell, error) {
	runner, err := interp.New()
	if err != nil {
		log.Fatal(err)
	}

	parser := syntax.NewParser()

	return &GoShell{runner, parser}, err
}

func (s *GoShell) StdIO(in, out, err *os.File) error {
	return interp.StdIO(in, out, err)(s.Runner)
}
func (s *GoShell) Run(ctx context.Context, r io.Reader) error {
	// This is a regression from being able to parse before doing stdio setup.
	f, err := s.Parser.Parse(r, "")
	if err != nil {
		return err
	}
	return s.Runner.Run(ctx, f)
}
func (s *GoShell) Dir() string {
	return s.Runner.Dir
}
func (b *GoShell) Complete(ctx context.Context, cmd io.Reader) error {
	return nil
}
