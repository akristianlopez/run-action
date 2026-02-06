package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
)

func main() {
	// On interroge l'endpoint de santé interne de notre propre conteneur
	n := os.Getenv("APP_PORT")
	if n == "" {
		n = "4000"
	}
	_, err := strconv.Atoi(n)
	if err != nil {
		log.Fatalf("Invalid application port '%s'", n)
	}
	resp, err := http.Get(fmt.Sprintf("http://localhost:%s/ping", n))
	if err != nil || resp.StatusCode != http.StatusOK {
		os.Exit(1) // Échec pour Docker
	}
	os.Exit(0) // Succès pour Docker
}
