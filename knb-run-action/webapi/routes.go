package webapi

import (
	"fmt"

	"github.com/gin-gonic/gin"
)

func addRoutes(r *gin.Engine, act Action) {

	api := r.Group("/action") //fmt.Sprintf("/%s", ConfigClient.Params["service_name"].(string))
	api.GET("/ping", func(ctx *gin.Context) { health(ctx, ConfigClient.Params["service_name"].(string)) })

	uig := r.Group(fmt.Sprintf("/%s", ConfigClient.Params["service_name"].(string)))
	uig.GET("/run", func(ctx *gin.Context) { getScreen(ctx, act) })
	uig.POST("/run", func(ctx *gin.Context) { runAction(ctx, act) })
	uig.PUT("/check/:id/:table/:name", func(ctx *gin.Context) { checkAction(ctx, act) })
	uig.PUT("/fetch", func(ctx *gin.Context) { fetchAction(ctx, act) })

	uig.GET("/config", func(ctx *gin.Context) { refresh(ctx) })
	uig.GET("/contract/:service/:name/:proc/:goal/:role", func(ctx *gin.Context) { signature(ctx, act) })
	uig.POST("/contract", func(ctx *gin.Context) { execContract(ctx, act) })

	uig.GET("/api/v1/mfe-setup", func(ctx *gin.Context) {
		ctx.JSON(200, act.getMFEConfig())
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
