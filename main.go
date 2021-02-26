package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"io"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/buildkite/terminal-to-html/v3"
	"github.com/creack/pty"
	"mvdan.cc/sh/v3/interp"
	"mvdan.cc/sh/v3/syntax"
)

var (
	host = flag.String("host", "localhost", "Hostname at which to run the server")
	port = flag.Int("port", 3000, "Port at which to run the server over HTTP")
)

var runner *interp.Runner
var parser *syntax.Parser

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	var err error
	runner, err = interp.New()
	if err != nil {
		log.Fatal(err)
	}

	parser = syntax.NewParser()

	http.HandleFunc("/run", HandleRun)
	http.Handle("/", http.FileServer(http.Dir("./web")))
	log.Fatal(http.ListenAndServe(*host+":"+strconv.Itoa(*port), nil))
}

var runMu sync.Mutex

func HandleRun(w http.ResponseWriter, req *http.Request) {
	runMu.Lock()
	defer runMu.Unlock()
	var stdout, stderr bytes.Buffer

	ptmx, pts, err := pty.Open()
	if err != nil {
		log.Println(err)
		return
	}
	defer func() {
		ptmx.Close()
		pts.Close()
	}()
	go func() {
		io.Copy(&stdout, ptmx)
	}()

	commands, err := parser.Parse(req.Body, "")
	if err == nil {
		// Reset stdio of runner before running a new command
		err = interp.StdIO(nil, pts, &stderr)(runner)
		if err != nil {
			log.Println(err)
			return
		}
		// TODO: serve partial output instead of blocking
		ctx, cancel := context.WithTimeout(req.Context(), 10*time.Second)
		defer cancel()
		err = runner.Run(ctx, commands)
		if err != nil {
			log.Println(err)
		}
	}

	b, _ := json.Marshal(struct {
		Dir            string
		Stdout, Stderr string
		Err            error
	}{
		runner.Dir,
		string(terminal.Render(stdout.Bytes())),
		stderr.String(),
		err,
	})

	_, err = w.Write(b)
	if err != nil {
		log.Println(err)
	}
}
