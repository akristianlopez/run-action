package webapi

import (
	"fmt"
	"os"

	"github.com/gin-gonic/gin"
)

func Start(port int) error {
	store := newAction()
	router := gin.Default()
	addRoutes(router, *store)

	return router.Run(fmt.Sprintf(":%d", port))
}
func StartTLS(port int, certFile, keyFile string) error {
	store := newAction()
	router := gin.Default()
	addRoutes(router, *store)

	return router.RunTLS(fmt.Sprintf(":%d", port), certFile, keyFile)
}
func Stop(port int) {
	os.Exit(0)
}
