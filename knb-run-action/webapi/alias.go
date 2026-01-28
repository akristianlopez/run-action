package webapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/akristianlopez/action/object"
	"github.com/gin-gonic/gin"
)

func getScreen(ctx *gin.Context, s Action) {
	select {
	case <-ctx.Done():
		return
	default:
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
}
func runAction(ctx *gin.Context, s Action) {
	select {
	case <-ctx.Done():
		return
	default:
		payload := &RequestData{}
		if err := ctx.BindJSON(payload); err != nil {
			ctx.AbortWithError(http.StatusBadRequest, err)
			return
		}
		res, _ := s.Run(ctx, *payload)
		ctx.JSON(http.StatusOK, res)
	}
}
func fetchAction(ctx *gin.Context, s Action) {
	select {
	case <-ctx.Done():
		return
	default:
		payload := &RequestData{}
		if err := ctx.BindJSON(payload); err != nil {
			ctx.AbortWithError(http.StatusBadRequest, err)
			return
		}
		res, _ := s.Fetch(ctx, *payload)
		ctx.JSON(http.StatusOK, res)
	}
}
func checkAction(ctx *gin.Context, s Action) {
	select {
	case <-ctx.Done():
		return
	default:
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
}
func health(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, "OK")
}

func refresh(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, "OK")
}
func signature(ctx *gin.Context, s Action) {
	select {
	case <-ctx.Done():
		return
	default:
		// Get the contract signature
		req := &RequestData{}
		if err := ctx.BindJSON(req); err != nil {
			ctx.AbortWithError(http.StatusBadRequest, err)
			return
		}
		ctx.JSON(http.StatusOK, s.Signature(ctx, *req))
	}
}
func execContract(ctx *gin.Context, s Action) {
	select {
	case <-ctx.Done():
		return
	default:
		req := &RequestData{}
		req.Data = make(map[string]interface{})
		req.Data["service"] = ctx.Param("service")
		req.Data["contract"] = ctx.Param("name")
		req.Proc = ctx.Param("proc")
		req.Knowledge = ctx.Param("goal")
		req.Role = ctx.Param("role")
		rValue, ok := s.ExecContract(ctx, *req)
		resp := ResponseData{Error: 0, Data: make(map[string]interface{})}
		if !ok {
			resp.Error = 1
			resp.Data["msg"] = rValue.Inspect()
		}
		resp.Data["result"] = rValue
		ctx.JSON(http.StatusOK, resp)
	}
}
func newError(format string, a ...interface{}) *object.Error {
	return &object.Error{Message: fmt.Sprintf(format, a...)}
}
func send2Eval(ctx *gin.Context, url string, data RequestData) object.Object {
	select {
	case <-ctx.Done():
		return newError("Request cancelled by client")
	default:
		jsonData, err := json.Marshal(data)
		if err != nil {
			return newError("Error encoding JSON: %v", err)
		}
		resp, err := http.Post(fmt.Sprintf("http://%s", url), "application/json",
			bytes.NewBuffer(jsonData))
		if err != nil {
			return newError("Error sending request: %v", err)
		}
		defer resp.Body.Close()

		// Read response
		res, err := io.ReadAll(resp.Body)
		if err != nil {
			return newError("Error reading response: %v", err)
		}
		var responseData ResponseData
		err = json.Unmarshal(res, &responseData)
		if err != nil {
			return newError("Error decoding JSON response: %v", err)
		}
		if responseData.Error != 0 {
			return newError("Error from eval server: %v", responseData.Data["msg"])
		}
		if val, ok := responseData.Data["result"]; ok {
			return val.(object.Object)
		}
		return object.NULL
	}
}
func send2getSignature(ctx *gin.Context, url string) (*ResponseData, error) {
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("Request cancelled by client")
	default:
		resp, err := http.Get(fmt.Sprintf("http://%s", url))
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		// Read response
		res, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		var responseData ResponseData
		err = json.Unmarshal(res, &responseData)
		if err != nil {
			return nil, err
		}
		if responseData.Error != 0 {
			return nil, fmt.Errorf("Error from eval server: %v", responseData.Data["msg"])
		}
		if val, ok := responseData.Data["signature"]; ok {
			return val.(*ResponseData), nil
		}
		return nil, fmt.Errorf("Error from eval server: %v", responseData.Data["msg"])
	}
}
