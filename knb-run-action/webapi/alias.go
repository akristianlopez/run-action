package webapi

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func getScreen(ctx *gin.Context, s Action) {
	payload := &RequestData{}
	if err := ctx.BindJSON(payload); err != nil {
		ctx.AbortWithError(http.StatusBadRequest, err)
		return
	}
	str, err := s.GetInterface(ctx, *payload)
	res := &ResponseData{}
	res.Error = 0
	if err != nil {
		res.Error = 1
		res.Data["Error"] = err.Error()
		ctx.JSON(http.StatusOK, res)
	}
	res.Data["Screen"] = str
	ctx.JSON(http.StatusOK, res)
}
func runAction(ctx *gin.Context, s Action) {
	payload := &RequestData{}
	if err := ctx.BindJSON(payload); err != nil {
		ctx.AbortWithError(http.StatusBadRequest, err)
		return
	}
	res, _ := s.Run(ctx, *payload)
	ctx.JSON(http.StatusOK, res)
}
func fetchAction(ctx *gin.Context, s Action) {
	payload := &RequestData{}
	if err := ctx.BindJSON(payload); err != nil {
		ctx.AbortWithError(http.StatusBadRequest, err)
		return
	}
	res, _ := s.Fetch(ctx, *payload)
	ctx.JSON(http.StatusOK, res)
}
func checkAction(ctx *gin.Context, s Action) {
	id := ctx.Param("id")
	table := ctx.Param("table")
	newName := ctx.Param("name")
	payload := &RequestData{}
	if err := ctx.BindJSON(payload); err != nil {
		ctx.AbortWithError(http.StatusBadRequest, err)
		return
	}
	res, _, _ := s.Check(ctx, *payload, id, table, newName)
	ctx.JSON(http.StatusOK, res)
}
func health(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, "OK")
}

// func createTodo(ctx *gin.Context, s Store) {
// 	payload := &CreatePayload{}
// 	if err := ctx.BindJSON(payload); err != nil {
// 		ctx.AbortWithError(http.StatusBadRequest, err)
// 		return
// 	}

// 	ctx.JSON(http.StatusOK, CreateResponse{
// 		Todo: s.Create(payload.Description),
// 	})
// }

// func deleteTodo(ctx *gin.Context, s Store) {
// 	id := ctx.Param("id")
// 	s.Delete(id)
// }

// func checkTodo(ctx *gin.Context, s Store) {
// 	id := ctx.Param("id")

// 	payload := &CheckPayload{}
// 	if err := ctx.BindJSON(payload); err != nil {
// 		ctx.AbortWithError(http.StatusBadRequest, err)
// 		return
// 	}

// 	t, err := s.Check(id, payload.Completed)
// 	if err != nil {
// 		ctx.AbortWithError(http.StatusNotFound, err)
// 		return
// 	}

// 	ctx.JSON(http.StatusOK, CheckResponse{
// 		Todo: t,
// 	})
// }

// func listTodos(ctx *gin.Context, s Store) {
// 	ctx.JSON(http.StatusOK, ListResponse{
// 		Todos: s.List(),
// 	})
// }
