package webapi

import (
	"github.com/gin-gonic/gin"
)

func addRoutes(r *gin.Engine, act Action) {

	api := r.Group("/action") //fmt.Sprintf("/%s", ConfigClient.Params["service_name"].(string))
	api.GET("/run", func(ctx *gin.Context) { getScreen(ctx, act) })
	api.POST("/run", func(ctx *gin.Context) { runAction(ctx, act) })
	api.PUT("/check/:id/:table/:name", func(ctx *gin.Context) { checkAction(ctx, act) })
	api.PUT("/fetch", func(ctx *gin.Context) { fetchAction(ctx, act) })
	api.GET("/ping", func(ctx *gin.Context) { health(ctx, ConfigClient.Params["service_name"].(string)) })
	// api.POST("/data", func(ctx *gin.Context) { dataHandler(ctx) })
	api.GET("/config", func(ctx *gin.Context) { refresh(ctx) })
	api.GET("/contract/:service/:name/:proc/:goal/:role", func(ctx *gin.Context) { signature(ctx, act) })
	api.POST("/contract", func(ctx *gin.Context) { execContract(ctx, act) })
	// uig := r.Group("/ui")
	// workDir, _ := os.Getwd()
	// webDir := filepath.Join(workDir, "web", "dist")
	// uig.StaticFS("/ui", http.Dir(webDir))
	// fs := http.FileServer(http.Dir(webDir))
	// uig.GET("/remoteEntry.js", func(ctx *gin.Context) {
	// 	ui.ServeFile(ctx, fs, webDir,
	// 		ConfigClient.Params["service_name"].(string))
	// })
}
