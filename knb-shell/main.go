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

	"github.com/akristianlopez/run-action/knb-shell/consul"
)

func main() {
	// 1. Initialisation de slog
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	// 2. Définition des paramètres
	portStr := getEnv("PORT", "8080") //8080
	defaultPort, _ := strconv.Atoi(portStr)

	portPtr := flag.Int("port", defaultPort, "Port d'écoute du serveur")
	// consulAddrPtr := flag.String("consul", getEnv("CONSUL_ADDR", "localhost:8500"), "Adresse de l'agent Consul")
	consulAddrPtr := os.Getenv("CONSUL_ADDR")

	kvPathPtr := flag.String("kvpath", getEnv("CONFIG_PATH", "knb/services/shell"), "Chemin de la config dans Consul KV")
	serviceName := getEnv("SERVICE_NAME", "knb-shell")
	serviceDomain := getEnv("SERVICE_DOMAIN", "wosa.local")
	slog.Info("Démarrage du Shell KNB", "port", portStr, "service_name", serviceName)
	flag.Parse()

	// 3. Initialisation du client Consul
	consulClient, err := consul.NewClient(consulAddrPtr, logger)
	if err != nil {
		slog.Error("Impossible d'initialiser le client Consul", "erreur", err)
		os.Exit(1)
	}

	// 4. Lecture configuration KV
	configKV, err := consulClient.GetKV(*kvPathPtr)
	if err != nil {
		slog.Error("Aucune configuration trouvée ou erreur", "path", *kvPathPtr, "erreur", err)
	}

	// 5. Enregistrement du service
	serviceID := fmt.Sprintf("%s-%d", serviceName, *portPtr)
	consulClient.Deregister(serviceID)
	err = consulClient.Register(serviceName, serviceID, *portPtr, configKV)
	if err != nil {
		slog.Error("Échec de l'enregistrement", "service", serviceID, "erreur", err)
		os.Exit(1)
	}

	// 6. Graceful Shutdown
	go func() {
		stop := make(chan os.Signal, 1)
		signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
		<-stop
		slog.Info("Arrêt du service...")
		consulClient.Deregister(serviceID)
		os.Exit(0)
	}()

	// 7. Routes HTTP

	// Configuration du répertoire web (Vite génère dans web/dist)
	workDir, _ := os.Getwd()
	webDir := filepath.Join(workDir, "web", "dist")
	// webDir := "/root/web/dist"
	// Handler Santé
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"status": "UP", "service": serviceID})
	})

	http.HandleFunc("/health/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"status": "UP", "service": serviceID})
	})

	// Handler Discovery
	http.HandleFunc("/api/discovery", func(w http.ResponseWriter, r *http.Request) {
		slog.Info("Requête reçue", "path", r.URL.Path)
		remotes, err := consulClient.DiscoverRemotes(serviceDomain)
		if err != nil {
			http.Error(w, "Erreur de découverte", 500)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(remotes)
	})

	// Handler Statique avec INJECTION DYNAMIQUE
	fs := http.FileServer(http.Dir(webDir))
	http.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		slog.Info("Requête reçue", "path", r.URL.Path)

		// CAS 1 : Injection du Token dans l'index.html
		if r.URL.Path == "/" || r.URL.Path == "/index.html" {
			filePath := filepath.Join(webDir, "index.html")
			content, err := os.ReadFile(filePath)
			if err != nil {
				http.Error(w, "index.html non trouvé", http.StatusNotFound)
				return
			}

			// Récupération du token depuis les headers (injectés par Forward-Auth/Traefik)
			// Adaptez le nom du header selon votre config (ex: Authorization ou X-Auth-Token)
			token := r.Header.Get("X-Auth-Token")
			// slog.Info("Token récupéré pour injection", "header", r.Header)
			if token == "" {
				// Fallback sur le header Authorization si besoin
				authHeader := r.Header.Get("Authorization")
				token = strings.TrimPrefix(authHeader, "Bearer ")
			}
			if token == "" {
				token = r.Header.Get("X-Forwarded-Access-Token") // Courant avec Authentik/Traefik
			}
			// slog.Info("Token récupéré pour injection", "token_present", token != "")
			// Injection d'un script global pour que le main.js puisse lire le token sans fetch
			scriptInjection := fmt.Sprintf("<script>window.__KNB_TOKEN__ = '%s';</script>", token)
			modified := strings.Replace(string(content), "</head>", scriptInjection+"</head>", 1)

			w.Header().Set("Content-Type", "text/html")
			w.Write([]byte(modified))
			return
		}

		// CAS 2 : Gestion dynamique de remoteEntry.js
		if strings.HasSuffix(r.URL.Path, "remoteEntry.js") {
			// filePath := filepath.Join(webDir, "remoteEntry.js")
			filePath := filepath.Join(webDir, "assets", "remoteEntry.js")
			content, err := os.ReadFile(filePath)
			if err != nil {
				slog.Warn("Fichier remoteEntry.js manquant sur le disque", "path", filePath)
				fs.ServeHTTP(w, r)
				return
			}

			// Remplacement du placeholder par le vrai nom du service
			// On s'assure que le nom ne contient pas de caractères interdits en JS (comme les tirets)
			// ou on laisse Vite gérer si le plugin est flexible.
			modified := strings.ReplaceAll(string(content), "KNB_DYNAMIC_SERVICE_NAME", serviceName)

			w.Header().Set("Content-Type", "application/javascript")
			w.Write([]byte(modified))
			return
		}

		// Pour les autres fichiers JS, on force le type MIME si nécessaire
		if strings.HasSuffix(r.URL.Path, ".js") {
			w.Header().Set("Content-Type", "application/javascript")
		}

		fs.ServeHTTP(w, r)
	}))

	slog.Info("Démarrage du Shell KNB", "port", *portPtr, "dir", webDir)
	if err := http.ListenAndServe(fmt.Sprintf("0.0.0.0:%d", *portPtr), nil); err != nil {
		slog.Error("Erreur fatale", "erreur", err)
		os.Exit(1)
	}
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
