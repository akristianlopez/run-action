package webapi

import (
	"fmt"
	"os"

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

	// 2. Route pour le fichier remoteEntry.js dynamique
	// Remplace 'KNB_DYNAMIC_SERVICE_NAME' par la valeur de serviceName au runtime
	router.GET("/ui/remoteEntry.js", func(c *gin.Context) {
		var err error
		once.Do(func() {
			filePath := "./ui/dist/assets/remoteEntry.js"
			content, readErr := os.ReadFile(filePath)
			if readErr != nil {
				err = readErr
				return
			}

			// Injection du nom réel du service dans le code JavaScript
			modified := strings.ReplaceAll(string(content), "KNB_DYNAMIC_SERVICE_NAME", serviceName)
			cachedRemoteJS = []byte(modified)
			log.Printf("✅ Module [%s] injecté et mis en cache", serviceName)
		})

		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Fichier remoteEntry.js introuvable. Vérifiez le build frontend (npm run build)."})
			return
		}

		c.Header("Content-Type", "application/javascript")
		c.Header("Cache-Control", "no-cache") // Utile pour le développement
		c.Data(http.StatusOK, "application/javascript", cachedRemoteJS)
	})

	// 3. Service des assets statiques (CSS, chunks JS générés par Vite)
	router.Static("/ui/assets", "./ui/dist/assets")

	// 4. Intégration de vos routes métier existantes
	addRoutes(router, *store)

	log.Printf("🚀 Serveur [%s] en écoute sur le port %d", serviceName, port)
	return router.Run(fmt.Sprintf(":%d", port))
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
