package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strconv"
)

type Config struct {
	Commands []Command
	Chains   []Chain
}

type Command struct {
	Name string
	Path string
	Args []string
}

type Chain struct {
	Name    string
	Command []Command
}

var config Config

func main() {
	var configPath string
	var port int

	flags := flag.NewFlagSet("remote", flag.ExitOnError)
	flags.Usage = printUsage
	flags.StringVar(&configPath, "config", "remote.config.json", "config file path")
	flags.IntVar(&port, "port", 5000, "port to bind to")

	if err := flags.Parse(os.Args[1:]); err != nil {
		printUsage()
		os.Exit(1)
	}

	readConfig(configPath)
	startServer(port)
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

func startServer(port int) {
	address := "localhost:" + strconv.Itoa(port)

	http.HandleFunc("/run", func(w http.ResponseWriter, r *http.Request) {
		name := r.URL.Query().Get("name")
		var toRun *Command

		for _, cmd := range config.Commands {
			if cmd.Name == name {
				toRun = &cmd
				break
			}
		}

		if toRun == nil {
			fmt.Fprintf(w, "Command %s was not found.", name)
			return
		}

		cmd := exec.Command(toRun.Path, toRun.Args...)
		if err := cmd.Run(); err != nil {
			fmt.Fprintf(w, "Error: %s", err.Error())
			return
		}

		fmt.Fprintf(w, "Command %s executed successfully.", toRun.Name)
	})

	log.Printf("Remote server listening at http://%s", address)
	log.Fatal(http.ListenAndServe(address, nil))
}

func printUsage() {
	fmt.Fprintf(os.Stderr, helpText)
}

const helpText = `Usage: remote [options]

  Remote exposes an endpoint to run commands on the host machine.

Options:
  -config "remote.config.json" Path to configuration file
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
