package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"strconv"

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

func HandleRun(w http.ResponseWriter, req *http.Request) {
	var stdout, stderr bytes.Buffer

	commands, err := parser.Parse(req.Body, "")
	if err == nil {
		// Reset stdio of runner before running a new command
		err = interp.StdIO(nil, &stdout, &stderr)(runner)
		if err != nil {
			log.Println(err)
			return
		}
		err = runner.Run(context.Background(), commands)
		if err != nil {
			log.Println(err)
		}
	}

	b, _ := json.Marshal(struct {
		Stdout, Stderr string
		Err error
	}{stdout.String(), stderr.String(), err})

	_, err = w.Write(b)
	if err != nil {
		log.Println(err)
	}
}
