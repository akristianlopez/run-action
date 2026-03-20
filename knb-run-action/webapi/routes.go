package webapi

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

func addRoutes(r *gin.Engine, act Action, serviceName string) {

	api := r.Group("/action") //fmt.Sprintf("/%s", ConfigClient.Params["service_name"].(string))
	api.GET("/ping", func(ctx *gin.Context) { health(ctx, ConfigClient.Params["service_name"].(string)) })

	uig := r.Group(fmt.Sprintf("/%s", ConfigClient.Params["service_name"].(string)))
	uig.GET("/run", func(ctx *gin.Context) { getScreen(ctx, act) })
	uig.POST("/run", func(ctx *gin.Context) { runAction(ctx, act) })
	uig.PUT("/check", func(ctx *gin.Context) { checkAction(ctx, act) })
	uig.PUT("/fetch", func(ctx *gin.Context) { fetchAction(ctx, act) })
	uig.GET("/desc/:role/:proc/:goal/:object", func(ctx *gin.Context) { describeObject(ctx, act) })

	uig.GET("/config", func(ctx *gin.Context) { refresh(ctx) })
	uig.GET("/contract/:service/:name/:proc/:goal/:role", func(ctx *gin.Context) { signature(ctx, act) })
	uig.POST("/contract", func(ctx *gin.Context) { execContract(ctx, act) })

	uig.GET("/api/v1/mfe-setup", func(ctx *gin.Context) {
		ctx.JSON(200, act.getMFEConfig())
	})

	// Définition de la route avec un wildcard pour capturer tous les assets
	// uig.GET(fmt.Sprintf("/ui/*filepath", serviceName), func(c *gin.Context) {
	// 	path := c.Param("filepath")
	// 	fullPath := filepath.Join("./ui/dist/assets", path)

	// 	// Cas spécifique : Traitement et mise en cache du remoteEntry.js
	// 	if strings.HasSuffix(path, "remoteEntry.js") {
	// 		var err error
	// 		once.Do(func() {
	// 			// On cherche le fichier dans ./ui/dist/ ou ./ui/dist/assets/ selon votre build
	// 			content, readErr := os.ReadFile(fullPath)
	// 			if readErr != nil {
	// 				// Tentative de secours dans /assets/ si le chemin direct échoue
	// 				content, readErr = os.ReadFile(filepath.Join("./ui/dist/", "remoteEntry.js"))
	// 			}

	// 			if readErr != nil {
	// 				err = readErr
	// 				return
	// 			}

	// 			// Injection dynamique du nom du service
	// 			modified := strings.ReplaceAll(string(content), "KNB_DYNAMIC_SERVICE_NAME", serviceName)
	// 			cachedRemoteJS = []byte(modified)
	// 			log.Printf("✅ Module Federation [%s] injecté et mis en cache", serviceName)
	// 		})

	// 		if err != nil {
	// 			c.JSON(http.StatusNotFound, gin.H{"error": "remoteEntry.js introuvable"})
	// 			return
	// 		}

	// 		serveJS(c, cachedRemoteJS, ".js")
	// 		return
	// 	}

	// 	// Cas général : Servir les autres fichiers JS (ex: __federation_expose_App.js)
	// 	if strings.HasSuffix(path, ".js") {
	// 		content, err := os.ReadFile(fullPath)
	// 		if err != nil {
	// 			c.Status(http.StatusNotFound)
	// 			return
	// 		}
	// 		serveJS(c, content, ".js")
	// 		return
	// 	}
	// 	if strings.HasSuffix(path, ".css") {
	// 		content, err := os.ReadFile(fullPath)
	// 		if err != nil {
	// 			c.Status(http.StatusNotFound)
	// 			return
	// 		}

	// 		serveJS(c, content, ".css")
	// 		return
	// 	}

	// 	// Par défaut, laisser Gin servir le fichier statique normalement
	// 	fullPath = filepath.Join("./ui/dist", path)
	// 	c.File(fullPath)
	// })

	uig.GET("/ui/*filepath", func(c *gin.Context) {
		path := c.Param("filepath")

		// 1. Déterminer le dossier probable
		// On essaie d'abord dans /assets, puis à la racine de dist
		fullPath := filepath.Join("./ui/dist/assets", path)
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			fullPath = filepath.Join("./ui/dist", path)
		}

		// 2. Traitement spécial remoteEntry.js
		if strings.HasSuffix(path, "remoteEntry.js") {
			once.Do(func() {
				content, err := os.ReadFile(fullPath)
				if err == nil {
					modified := strings.ReplaceAll(string(content), "KNB_DYNAMIC_SERVICE_NAME", serviceName)
					cachedRemoteJS = []byte(modified)
				}
			})
			if cachedRemoteJS != nil {
				serveJS(c, cachedRemoteJS, ".js")
				return
			}
		}

		// 3. Servir les fichiers avec le bon type MIME
		ext := filepath.Ext(path)
		if ext == ".js" || ext == ".css" {
			content, err := os.ReadFile(fullPath)
			if err != nil {
				log.Printf("❌ Fichier introuvable : %s", fullPath)
				c.AbortWithStatus(http.StatusNotFound)
				return
			}
			serveJS(c, content, ext)
			return
		}

		// 4. Fallback pour le reste (images, etc.)
		c.File(fullPath)
	})

	// Routes pour servir les assets du Micro-frontend
	// Note : Assurez-vous que le chemin correspond à votre build de React
	// Exemple : Si votre build place les fichiers dans ./ui/dist/assets, ajustez en conséquence
	// Le wildcard *filepath permet de servir tous les fichiers nécessaires (JS, CSS, images, etc.)
	// r.GET(fmt.Sprintf("/%s/ui/*filepath", ConfigClient.Params["service_name"].(string)), func(c *gin.Context) {
	// 	path := c.Param("filepath")
	// 	fullPath := filepath.Join("./ui/dist/assets", path)
	// workDir, _ := os.Getwd()
	// webDir := filepath.Join(workDir, "web", "dist")
	// uig.StaticFS("/ui", http.Dir(webDir))
	// fs := http.FileServer(http.Dir(webDir))
	// uig.GET("/remoteEntry.js", func(ctx *gin.Context) {
	// 	ui.ServeFile(ctx, fs, webDir,
	// 		ConfigClient.Params["service_name"].(string))
	// })
}
