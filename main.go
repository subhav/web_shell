package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"github.com/buildkite/terminal-to-html/v3"
	"github.com/creack/pty"
	"io"
	"io/ioutil"
	"log"
	"mvdan.cc/sh/v3/interp"
	"mvdan.cc/sh/v3/syntax"
	"net/http"
	"os"
	"path"
	"strconv"
	"sync"
	"syscall"
)

var (
	host = flag.String("host", "localhost", "Hostname at which to run the server")
	port = flag.Int("port", 3000, "Port at which to run the server over HTTP")
)

var runner *interp.Runner
var parser *syntax.Parser

var shell *BashShell

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	var err error
	runner, err = interp.New()
	if err != nil {
		log.Fatal(err)
	}

	parser = syntax.NewParser()

	shell, err = NewBashShell()
	if err != nil {
		log.Fatal(err)
	}

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
	go io.Copy(&stdout, ptmx)

	dir, _ := ioutil.TempDir("", "webshell-*")
	pipeName := path.Join(dir, "errpipe")
	syscall.Mkfifo(pipeName, 0600)
	// If you open only the read side, then you need to open with O_NONBLOCK
	// and clear that flag after opening.
	//	pipe, err := os.OpenFile(pipeName, os.O_RDONLY|syscall.O_NONBLOCK, 0600)
	pipe, err := os.OpenFile(pipeName, os.O_RDWR, 0600)
	if err != nil {
		log.Println(err)
		return
	}
	defer func() {
		pipe.Close()
		os.Remove(pipeName)
		os.Remove(dir)
	}()
	go io.Copy(&stderr, pipe)

	// Reset stdio of runner before running a new command
	err = shell.StdIO(nil, pts, pipe)
	if err != nil {
		log.Println(err)
		return
	}
	err = shell.Run(req.Body)
	if err != nil {
		log.Println(err)
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
