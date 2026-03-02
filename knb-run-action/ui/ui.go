package ui

import (
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

func ServeFile(ctx *gin.Context, fs http.Handler, webDir string, serviceName string) {
	slog.Info("Requête reçue", "path", ctx.Request.URL.Path)
	if strings.HasSuffix(ctx.Request.URL.Path, "remoteEntry.js") {
		// filePath := filepath.Join(webDir, "remoteEntry.js")
		filePath := filepath.Join(webDir, "assets", "remoteEntry.js")
		content, err := os.ReadFile(filePath)
		if err != nil {
			slog.Warn("Fichier remoteEntry.js manquant sur le disque", "path", filePath)
			fs.ServeHTTP(ctx.Writer, ctx.Request)
			return
		}

		// Remplacement du placeholder par le vrai nom du service
		// On s'assure que le nom ne contient pas de caractères interdits en JS (comme les tirets)
		// ou on laisse Vite gérer si le plugin est flexible.
		modified := strings.ReplaceAll(string(content), "KNB_DYNAMIC_SERVICE_NAME", serviceName)

		ctx.Writer.Header().Set("Content-Type", "application/javascript")
		ctx.Writer.Write([]byte(modified))
		return
	}

	// Pour les autres fichiers JS, on force le type MIME si nécessaire
	if strings.HasSuffix(ctx.Request.URL.Path, ".js") {
		ctx.Writer.Header().Set("Content-Type", "application/javascript")
	}
	fs.ServeHTTP(ctx.Writer, ctx.Request)
}
