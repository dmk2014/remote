package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
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
	readConfig()

	startServer()
}

func readConfig() {
	path := "remote.config.json"

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

func startServer() {
	address := "localhost:5000"

	http.HandleFunc("/helloremote", func(w http.ResponseWriter, r *http.Request) {
		cmd := exec.Command("echo", "Hello Remote")

		if err := cmd.Run(); err != nil {
			fmt.Fprintf(w, "Error: %s", err.Error())
			return
		}

		fmt.Fprintf(w, "Success!")
	})

	http.HandleFunc("/error", func(w http.ResponseWriter, r *http.Request) {
		cmd := exec.Command("idonotexist")

		if err := cmd.Run(); err != nil {
			fmt.Fprintf(w, "Error: %s", err.Error())
			return
		}

		fmt.Fprintf(w, "Unexpected success.")
	})

	log.Printf("Remote server listening at http://%s", address)
	log.Fatal(http.ListenAndServe(address, nil))
}
