package response

import "github.com/gin-gonic/gin"

type Response struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

func Success(c *gin.Context, data interface{}) {
	c.JSON(200, Response{
		Success: true,
		Message: "success",
		Data:    data,
	})
}

func Error(c *gin.Context, code int, message string, errCode string) {
	c.JSON(code, Response{
		Success: false,
		Message: message,
		Error:   errCode,
	})
}
