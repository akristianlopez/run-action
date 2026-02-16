package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/hashicorp/consul/api"
)

// RemoteModule définit la structure d'un micro-frontend découvert
type RemoteModule struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

func main() {
	// 1. Configuration du client Consul
	consulAddr := os.Getenv("CONSUL_HTTP_ADDR")
	if consulAddr == "" {
		consulAddr = "localhost:8500"
	}

	config := api.DefaultConfig()
	config.Address = consulAddr
	client, err := api.NewClient(config)
	if err != nil {
		log.Fatalf("Erreur client Consul: %v", err)
	}

	// 2. Endpoint de découverte dynamique
	http.HandleFunc("/api/discovery", func(w http.ResponseWriter, r *http.Request) {
		// On cherche tous les services ayant le tag "knb-ui=true"
		// On utilise le catalogue pour lister les services
		services, _, err := client.Catalog().Services(nil)
		if err != nil {
			http.Error(w, "Erreur catalogue", 500)
			return
		}

		var remotes []RemoteModule
		for sName, tags := range services {
			isUI := false
			for _, t := range tags {
				if t == "knb-ui=true" {
					isUI = true
					break
				}
			}

			if isUI {
				// On construit l'URL basée sur la convention de nommage Traefik
				// Dans ton cas : organization.wosa.local
				remotes = append(remotes, RemoteModule{
					Name: sName,
					URL:  fmt.Sprintf("https://%s.wosa.local/ui/remoteEntry.js", strings.ReplaceAll(sName, "knb-", "")),
				})
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(remotes)
	})

	// 3. Serveur de fichiers statiques (pour le Shell lui-même)
	// En développement, on pointe sur le dossier de build de Vite
	fs := http.FileServer(http.Dir("./web/dist"))
	http.Handle("/", fs)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Shell KNB démarré sur le port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
