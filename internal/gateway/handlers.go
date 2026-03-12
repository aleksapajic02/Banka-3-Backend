package gateway

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type Request struct {
	Henlo string `json:"henlo" binding:"required"`
}

type Response struct {
	Hi string `json:"hi"`
}

func (s *Server) test(c *gin.Context) {
	// all microservices should be available here behind s

	// function will return with error if the request json can't be bound to the Request struct
	var req Request
	err := c.BindJSON(&req)
	if err != nil {
		return
	}
	str := "HI " + req.Henlo

	// return struct serialized to JSON
	resp := Response{Hi: str}
	c.JSON(http.StatusOK, resp)
}

func SetupApi(router *gin.Engine, s *Server) {
	api := router.Group("/api")
	api.GET("/test", s.test)
}
