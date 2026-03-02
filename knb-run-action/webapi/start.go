package webapi

import (
	"fmt"
	"os"
	"path/filepath"

	"log"
	"net/http"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
)

// Variables pour optimiser les performances du Micro-frontend
var (
	cachedRemoteJS []byte
	once           sync.Once
)

// Start initialise et lance le serveur avec le support Micro-frontend dynamique
// Ajout du paramètre serviceName pour le remplacement dynamique
func Start(serviceName string, port int) error {
	// Initialisation de votre store existant
	store := newAction()

	router := gin.Default()
	router.SetTrustedProxies(nil)

	// 1. Middleware CORS : Crucial pour que le Shell charge ce Micro-frontend
	router.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Origin, Content-Type")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	// Définition de la route avec un wildcard pour capturer tous les assets
	router.GET(fmt.Sprintf("/%s/ui/*filepath", serviceName), func(c *gin.Context) {
		path := c.Param("filepath")
		fullPath := filepath.Join("./ui/dist/assets", path)

		// Cas spécifique : Traitement et mise en cache du remoteEntry.js
		if strings.HasSuffix(path, "remoteEntry.js") {
			var err error
			once.Do(func() {
				// On cherche le fichier dans ./ui/dist/ ou ./ui/dist/assets/ selon votre build
				content, readErr := os.ReadFile(fullPath)
				if readErr != nil {
					// Tentative de secours dans /assets/ si le chemin direct échoue
					content, readErr = os.ReadFile(filepath.Join("./ui/dist/", "remoteEntry.js"))
				}

				if readErr != nil {
					err = readErr
					return
				}

				// Injection dynamique du nom du service
				modified := strings.ReplaceAll(string(content), "KNB_DYNAMIC_SERVICE_NAME", serviceName)
				cachedRemoteJS = []byte(modified)
				log.Printf("✅ Module Federation [%s] injecté et mis en cache", serviceName)
			})

			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": "remoteEntry.js introuvable"})
				return
			}

			serveJS(c, cachedRemoteJS)
			return
		}

		// Cas général : Servir les autres fichiers JS (ex: __federation_expose_App.js)
		if strings.HasSuffix(path, ".js") {
			content, err := os.ReadFile(fullPath)
			if err != nil {
				c.Status(http.StatusNotFound)
				return
			}
			serveJS(c, content)
			return
		}

		// Par défaut, laisser Gin servir le fichier statique normalement
		fullPath = filepath.Join("./ui/dist", path)
		c.File(fullPath)
	})

	// 4. Intégration de vos routes métier existantes
	addRoutes(router, *store)

	log.Printf("🚀 Serveur [%s] en écoute sur le port %d", serviceName, port)
	return router.Run(fmt.Sprintf("0.0.0.0:%d", port))
}

func serveJS(c *gin.Context, content []byte) {
	c.Header("Content-Type", "application/javascript")
	c.Header("Access-Control-Allow-Origin", "*") // Indispensable pour la Fédération
	c.Header("Cache-Control", "no-cache")
	c.Data(http.StatusOK, "application/javascript", content)
}

// func Start(port int) error {
// 	store := newAction()
// 	router := gin.Default()
// 	router.SetTrustedProxies(nil)

// 	addRoutes(router, *store)

// 	return router.Run(fmt.Sprintf(":%d", port))
// }
// func StartTLS(port int, certFile, keyFile string) error {
// 	store := newAction()
// 	router := gin.Default()
// 	addRoutes(router, *store)

// 	return router.RunTLS(fmt.Sprintf(":%d", port), certFile, keyFile)
// }
// func Stop(port int) {
// 	os.Exit(0)
// }
