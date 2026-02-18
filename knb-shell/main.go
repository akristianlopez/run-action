package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/akristianlopez/run-action/knb-shell/consul" // Assurez-vous que le nom du module dans go.mod est knb-shell
	// "gopkg.in/yaml.v2"
)

func main() {
	// 1. Initialisation de slog (format JSON pour les conteneurs)
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	// 2. Définition des paramètres (Priorité : Flag > Env > Default)
	portStr := getEnv("PORT", "8080")
	defaultPort, _ := strconv.Atoi(portStr)

	portPtr := flag.Int("port", defaultPort, "Port d'écoute du serveur")
	consulAddrPtr := flag.String("consul", getEnv("CONSUL_ADDR", "localhost:8500"), "Adresse de l'agent Consul")
	kvPathPtr := flag.String("kvpath", getEnv("CONFIG_PATH", "knb/services/shell"), "Chemin de la config dans Consul KV")
	serviceName := getEnv("SERVICE_NAME", "knb-shell")
	serviceDomain := getEnv("SERVICE_DOMAIN", "wosa.local")
	flag.Parse()

	// 3. Initialisation du package Consul
	consulClient, err := consul.NewClient(*consulAddrPtr, logger)
	if err != nil {
		slog.Error("Impossible d'initialiser le client Consul", "erreur", err)
		os.Exit(1)
	}

	// 4. Lecture facultative d'une configuration dans le KV Store
	configKV, err := consulClient.GetKV(*kvPathPtr)
	if err != nil {
		slog.Error("Aucune configuration trouvée dans le KV Store ou erreur", "path", *kvPathPtr, "erreur", err)
		os.Exit(1)
	}

	//	consulClient.Deregister("DESKTOP-FD38FG2")
	// 5. Enregistrement du service pour Traefik
	serviceID := fmt.Sprintf("%s-%d", serviceName, *portPtr)
	// serviceID, err := os.Hostname()
	consulClient.Deregister(serviceID)
	// if err != nil {
	// 	slog.Error("Échec de l'enregistrement du service", "serviceID", serviceID, "erreur", err)
	// 	os.Exit(1)
	// }

	err = consulClient.Register(serviceName, serviceID, *portPtr, configKV)
	if err != nil {
		slog.Error("Échec de l'enregistrement du service", "service", serviceID, "erreur", err)
		os.Exit(1)
	}

	// 6. Gestion du signal d'arrêt (Graceful Shutdown)
	go func() {
		stop := make(chan os.Signal, 1)
		signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
		<-stop
		slog.Info("Signal d'arrêt reçu, nettoyage de Consul...")
		consulClient.Deregister(serviceID)
		os.Exit(0)
	}()

	// 7. Routes HTTP

	// Healthbite : Endpoint de santé pour Consul et Traefik
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "UP",
			"message": "Healthbite OK",
			"service": serviceID,
		})
	})

	// API Discovery : Découverte dynamique des micro-frontends
	http.HandleFunc("/api/discovery", func(w http.ResponseWriter, r *http.Request) {
		remotes, err := consulClient.DiscoverRemotes(serviceDomain)
		if err != nil {
			slog.Error("Erreur de découverte", "erreur", err)
			http.Error(w, "Erreur interne", 500)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(remotes)
	})

	// Serveur de fichiers statiques (UI du Shell)
	workDir, _ := os.Getwd()
	webDir := filepath.Join(workDir, "web")
	fs := http.FileServer(http.Dir(webDir))
	http.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// On s'assure que le JS est servi avec le bon type MIME
		if strings.HasSuffix(r.URL.Path, ".js") {
			w.Header().Set("Content-Type", "application/javascript")
		}
		fs.ServeHTTP(w, r)
	}))

	// 8. Lancement du serveur
	slog.Info("Démarrage du Shell KNB", "port", *portPtr) //, "consul", *consulAddrPtr)
	if err := http.ListenAndServe(fmt.Sprintf(":%d", *portPtr), nil); err != nil {
		slog.Error("Erreur fatale du serveur", "erreur", err)
		os.Exit(1)
	}
}

// getEnv est un utilitaire pour récupérer une variable d'environnement ou une valeur par défaut
func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
