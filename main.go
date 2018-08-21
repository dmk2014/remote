package main

import (
	"fmt"
	"log"
	"net/http"
	"os/exec"
)

func main() {

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
