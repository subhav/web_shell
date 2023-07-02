// Prototype Shell UI, for the Go sh package and external shell interpreters.
//
// Each command is allocated a pty for stdout and a (named) pipe for stderr.
// An anonymous pipe would be better, but would require fd passing.
//
// This doesn't work for every command:
//   - If `less` can't open `/dev/tty`, it READS from stderr! Not stdin.
//     (because stdin might be the read end of a pipe)
//     alias less="less 2<&0" works, but wouldn't work in a pipe.
//   - sudo reads from /dev/tty by default, but you can tell it to use stdin
//     with `sudo -S`. alias sudo="sudo -S" works.
//
// Apparently according to POSIX, stderr is supposed to be open for both
// reading and writing...
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/buildkite/terminal-to-html/v3"
	"github.com/creack/pty"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
	"sync"
	"syscall"
)

var (
	host = flag.String("host", "localhost", "Hostname at which to run the server")
	port = flag.Int("port", 3000, "Port at which to run the server over HTTP")
	gosh = flag.Bool("gosh", false, "Use the sh package instead of bash")
	oil  = flag.Bool("oil", false, "Use oil instead of bash")
	fifo = flag.Bool("fifo", true, "Use named fifo instead of anonymous pipe")
	test = flag.Bool("test", false, "test a function ad-hoc, used for development")
)

var shell Shell

type Shell interface {
	StdIO(*os.File, *os.File, *os.File) error
	Run(context.Context, io.Reader) error
	Dir() string
}

type CommandOut struct {
	Dir            string
	Stdout, Stderr string
	Err            error
}

func main() {
	flag.Parse()
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	var err error

	if *gosh {
		shell, err = NewGoShell()
	} else if *oil {
		shell, err = NewFANOSShell()
	} else {
		shell, err = NewBashShell()
	}
	if err != nil {
		log.Fatal(err)
	}

	if *test {
		os.Stdout.Write([]byte("helo"))
		completion, _ := Complete(strings.NewReader("git s"))
		log.Println(completion)
		return
	}
	http.HandleFunc("/run", HandleRun)
	http.HandleFunc("/cancel", HandleCancel)
	http.Handle("/", http.FileServer(http.Dir("./web")))
	log.Fatal(http.ListenAndServe(*host+":"+strconv.Itoa(*port), nil))
}

var runMu sync.Mutex
var runCancel context.CancelFunc = func() {}

func Run(req io.Reader) (CommandOut, error) {
	var output CommandOut
	runMu.Lock()
	defer runMu.Unlock()
	var stdout, stderr bytes.Buffer
	var runCtx context.Context

	runCtx, runCancel = context.WithCancel(context.Background())
	defer runCancel()

	ptmx, pts, err := pty.Open()
	if err != nil {
		log.Println(err)
		return output, err
	}
	defer func() {
		ptmx.Close()
		pts.Close()
	}()
	go io.Copy(&stdout, ptmx)

	var pipe *os.File
	if *fifo {
		dir, _ := ioutil.TempDir("", "webshell-*")
		pipeName := path.Join(dir, "errpipe")
		syscall.Mkfifo(pipeName, 0600)
		// If you open only the read side, then you need to open with O_NONBLOCK
		// and clear that flag after opening.
		//	pipe, err := os.OpenFile(pipeName, os.O_RDONLY|syscall.O_NONBLOCK, 0600)
		pipe, err = os.OpenFile(pipeName, os.O_RDWR, 0600)
		if err != nil {
			log.Println(err)
			return output, err
		}
		defer func() {
			pipe.Close()
			os.Remove(pipeName)
			os.Remove(dir)
		}()
		go io.Copy(&stderr, pipe)
	} else {
		var rdPipe *os.File
		rdPipe, pipe, err = os.Pipe()
		if err != nil {
			log.Println(err)
			return output, err
		}
		go func() {
			io.Copy(&stderr, rdPipe)
			rdPipe.Close()
			pipe.Close()
		}()
	}

	// Reset stdio of runner before running a new command
	err = shell.StdIO(nil, pts, pipe)
	if err != nil {
		log.Println(err)
		return output, err
	}
	err = shell.Run(runCtx, req)
	if err != nil {
		log.Println(err)
	}

	output.Dir = shell.Dir()
	output.Stdout = string(terminal.Render(stdout.Bytes()))
	output.Stderr = stderr.String()
	output.Err = err
	return output, nil
}

func HandleRun(w http.ResponseWriter, req *http.Request) {
	go func() {
		output, err := Run(req.Body)
		if err != nil {
			log.Println(err)
			return
		}
		log.Println(req.Body)
		o, err := json.Marshal(output)
		if err != nil {
			log.Println(err)
		}
		_, err = w.Write(o)
		if err != nil {
			log.Println(err)
		}
	}()
}

func HandleCancel(w http.ResponseWriter, req *http.Request) {
	log.Print("Received cancel")
	runCancel()
}

// currently bash complete only, wrapper around "Run"
func Complete(req io.Reader) (string, error) {
	buf, err := ioutil.ReadAll(req)
	if err != nil {
		log.Println(err)
		return "", err
	}
	prog, _, hasArgs := strings.Cut(string(buf[:]), " ")
	if hasArgs {
		comm := fmt.Sprintf("complete -p '%s' | sed s/^complete/compgen/", prog)
		log.Println(comm)
		completion, _ := Run(strings.NewReader(comm))
		log.Println(completion.Stdout)
		out, _ := Run(strings.NewReader(completion.Stdout))
		log.Println(out.Stdout)
		return out.Stdout, nil
	} else {
		// TODO Search $path/bash builtins/etc.
		return "", errors.New("Not yet implemented")
	}
	//
}
