package webapi

import (
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"log"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
)

// Variables pour optimiser les performances du Micro-frontend
var (
	cachedRemoteJS []byte
	once           sync.Once
)

func InitDB() (*sql.DB, error) {
	db, err := sql.Open(Db_connect_params.Kind, getConnectionString())
	if err != nil {
		return nil, err
	}

	// Configuration cruciale pour éviter "too many clients"
	db.SetMaxOpenConns(30)                 // Limite stricte de connexions simultanées
	db.SetMaxIdleConns(10)                 // Garde des connexions prêtes sous le coude
	db.SetConnMaxLifetime(3 * time.Minute) // Évite les connexions "fantômes" trop vieilles

	// Vérifie si la connexion est réellement établie
	if err = db.Ping(); err != nil {
		return nil, err
	}

	return db, nil
}

// Start initialise et lance le serveur avec le support Micro-frontend dynamique
// Ajout du paramètre serviceName pour le remplacement dynamique
func Start(serviceName string, port int) error {
	// Initialisation de votre store existant
	db, err := InitDB()
	if err != nil {
		slog.Error(err.Error())
		return err
	}
	defer db.Close()
	store := newAction(db)

	router := gin.Default()
	// router.SetTrustedProxies(nil)

	// // 1. Middleware CORS : Crucial pour que le Shell charge ce Micro-frontend
	// router.Use(func(c *gin.Context) {
	// 	c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
	// 	c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	// 	c.Writer.Header().Set("Access-Control-Allow-Headers", "Origin, Content-Type")

	// 	if c.Request.Method == "OPTIONS" {
	// 		c.AbortWithStatus(204)
	// 		return
	// 	}
	// 	c.Next()
	// })

	// 4. Intégration de vos routes métier existantes
	addRoutes(router, *store, serviceName)

	log.Printf("🚀 Serveur [%s] en écoute sur le port %d", serviceName, port)
	return router.Run(fmt.Sprintf("0.0.0.0:%d", port))
}

// func serveJS(c *gin.Context, content []byte, ext string) {
// 	c.Header("Content-Type", "application/javascript")
// 	c.Header("Access-Control-Allow-Origin", "*") // Indispensable pour la Fédération
// 	c.Header("Cache-Control", "no-cache")

// 	switch ext {
// 	case ".js":
// 		c.Header("Content-Type", "application/javascript")
// 	case ".css":
// 		c.Header("Content-Type", "text/css")
// 	}
// 	c.Data(http.StatusOK, c.GetHeader("Content-Type"), content)
// }

func serveJS(c *gin.Context, content []byte, ext string) {
	contentType := "text/javascript" // Valeur par défaut application

	if ext == ".css" {
		contentType = "text/css"
	}

	c.Header("Content-Type", contentType)
	c.Header("Access-Control-Allow-Origin", "*")
	c.Header("Cache-Control", "no-cache")

	// On utilise la variable contentType locale
	c.Data(http.StatusOK, contentType, content)
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
