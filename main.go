package main

import (
	"compress/gzip"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"time"
)

type Config struct {
	Commands []Command
}

func (c *Config) GetCommand(name string) *Command {
	var result *Command

	for _, cmd := range c.Commands {
		if cmd.Name == name {
			result = &cmd
			break
		}
	}

	return result
}

type Command struct {
	Name string
	Path string
	Args []string
}

var config Config

func main() {
	var configPath string
	var host string
	var port int

	flags := flag.NewFlagSet("remote", flag.ExitOnError)
	flags.Usage = printUsage
	flags.StringVar(&configPath, "config", "remote.config.json", "config file path")
	flags.StringVar(&host, "host", "localhost", "host to bind to")
	flags.IntVar(&port, "port", 5000, "port to bind to")

	if err := flags.Parse(os.Args[1:]); err != nil {
		printUsage()
		os.Exit(1)
	}

	readConfig(configPath)
	startServer(host, port)
}

func readConfig(path string) {
	file, err := os.Open(path)
	if os.IsNotExist(err) {
		log.Fatalf("Config file does not exist at %s", path)
	}
	if err != nil {
		log.Fatalf("Could not open config at %s\n%s", path, err.Error())
	}
	defer file.Close()

	decoder := json.NewDecoder(file)

	err = decoder.Decode(&config)
	if err != nil && err != io.EOF {
		log.Fatalf("Error parsing config at %s\n%s", path, err)
	}
}

func startServer(host string, port int) {
	address := host + ":" + strconv.Itoa(port)

	http.HandleFunc("/run", httpGet(httpCors(runHandler)))
	http.HandleFunc("/list", httpGet(httpCors(listHandler)))
	http.HandleFunc("/heartbeat", httpGet(httpCors(heartbeatHandler)))

	log.Printf("Remote server listening at http://%s", address)
	log.Fatal(http.ListenAndServe(address, nil))
}

func httpGet(h http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		h(w, r)
	})
}

func httpCors(h http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Expect simple requests without OPTIONS preflight
		// https://developer.mozilla.org/en-US/docs/Web/HTTP/CORS#Simple_requests
		w.Header().Set("Access-Control-Allow-Origin", "*")
		h(w, r)
	})
}

func runHandler(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	if len(name) == 0 {
		http.Error(w, "Query parameter [name] is required and cannot be empty.", http.StatusBadRequest)
		return
	}

	toRun := config.GetCommand(name)
	if toRun == nil {
		http.Error(w, "Specified command was not found.", http.StatusBadRequest)
		return
	}

	cmd := exec.Command(toRun.Path, toRun.Args...)
	if err := cmd.Start(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	go func() {
		if err := cmd.Wait(); err != nil {
			log.Printf("Failed to wait for command %s with error %s", toRun.Name, err.Error())
		}
	}()

	fmt.Fprintf(w, "Command %s started successfully.", toRun.Name)
}

func listHandler(w http.ResponseWriter, r *http.Request) {
	names := make([]string, 0, len(config.Commands))
	for _, cmd := range config.Commands {
		names = append(names, cmd.Name)
	}

	sort.Strings(names)

	w.Header().Set("Content-Encoding", "gzip")
	w.Header().Set("Content-Type", "application/json")

	zip := gzip.NewWriter(w)
	defer zip.Close()

	json.NewEncoder(zip).Encode(names)
}

func heartbeatHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Server Time: %s", time.Now().Format("2006-01-02 15:04:05"))
}

func printUsage() {
	fmt.Fprintf(os.Stderr, helpText)
}

const helpText = `Usage: remote [options]

  Remote exposes an endpoint to run commands on the host machine.

Options:
  -config "remote.config.json" Path to configuration file
  -host "localhost"            Host that the server should bind to
  -port 5000                   Port that the server should bind to

Configuration File:

  The configuration file contains an array of commands to expose.

  {
    "Commands": [
      {
        "Name": "command_name",
          "Path": "echo",
          "Args": [
            "Hello",
            "Remote
          ]
      }
    ]
  }

Command Execution:

  Execute commands by sending a GET request to /run.
  http://localhost:5000/run?name=command_name

`
