package main

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"go-ptt-crawler/pkg/images"

	"github.com/gin-gonic/gin"
)

func pttImageBoardHandler(ctx *gin.Context) {
	chunkSize, err := strconv.Atoi(ctx.DefaultQuery("chunk_size", "30"))
	if err != nil {
		log.Fatalf("chunk size strconv.Atoi failed: %v", err)
	}
	limitSize, err := strconv.Atoi(ctx.DefaultQuery("limit_size", "1"))
	if err != nil {
		log.Fatalf("limit size strconv.Atoi failed: %v", err)
	}
	board := ctx.Param("board")
	titlePrefix := fmt.Sprintf("[%v]", ctx.Query("prefix"))

	images.PttImageBoard(board, titlePrefix, chunkSize, limitSize)
}

func pttRandomImageBoardHandler(ctx *gin.Context) {
	board := ctx.Param("board")
	titlePrefix := fmt.Sprintf("[%v]", ctx.Query("prefix"))
	image := images.PttRandomImageBoard(board, titlePrefix)

	response, err := http.Get(image)
	if err != nil || response.StatusCode != http.StatusOK {
		ctx.Status(http.StatusServiceUnavailable)
		return
	}

	buf := new(bytes.Buffer)
	buf.ReadFrom(response.Body)
	reader := response.Body
	defer reader.Close()
	contentType := response.Header.Get("Content-Type")

	ctx.Data(http.StatusOK, contentType, buf.Bytes())
	return
}

func main() {
	router := gin.Default()
	//router
	apiGroup := router.Group("/api/v1")
	{
		apiGroup.GET("ptt/image/:board", pttImageBoardHandler)
		apiGroup.GET("ptt/image/:board/random", pttRandomImageBoardHandler)
	}
	router.Run()
}
