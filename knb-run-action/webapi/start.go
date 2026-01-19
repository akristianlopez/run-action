package webapi

import (
	"fmt"

	"github.com/gin-gonic/gin"
)

func Start(port int) {
	store := newAction()
	router := gin.Default()
	addRoutes(router, *store)

	router.Run(fmt.Sprintf(":%d", port))
}
