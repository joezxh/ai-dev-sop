package console

import "github.com/gin-gonic/gin"

type envelope struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data any    `json:"data"`
}

func OK(c *gin.Context, data any) {
	c.JSON(200, envelope{Code: 0, Msg: "", Data: data})
}

func Fail(c *gin.Context, status, code int, msg string) {
	c.JSON(status, envelope{Code: code, Msg: msg, Data: nil})
}
