package main

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	"go-ptt-crawler/pkg/aws"
	"go-ptt-crawler/pkg/images"

	"github.com/apex/gateway/v2"
	"github.com/gin-gonic/gin"

	_ "github.com/joho/godotenv/autoload"
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

	ctx.JSON(200, gin.H{
		"status": "success",
	})
}

func pttRandomImageBoardHandler(ctx *gin.Context) {
	board := ctx.Param("board")
	titlePrefix := fmt.Sprintf("[%v]", ctx.Query("prefix"))
	image := images.PttRandomImageBoard(board, titlePrefix)
	if aws.InLambda() {
		fmt.Println(image)
		ctx.JSON(200, gin.H{
			"image": image,
		})
		return
	}

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

func setupRouter() *gin.Engine {
	router := gin.Default()
	//router
	apiGroup := router.Group("/api/v1")
	{
		apiGroup.GET("ptt/image/:board", pttImageBoardHandler)
		apiGroup.GET("ptt/image/:board/random", pttRandomImageBoardHandler)
	}
	return router
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	addr := ":" + port
	log.Println("=======================================")
	log.Println("Runinng gin-lambda server in " + addr)
	log.Println("=======================================")
	if aws.InLambda() {
		log.Fatal(gateway.ListenAndServe(addr, setupRouter()))
	} else {
		log.Fatal(http.ListenAndServe(addr, setupRouter()))
	}
}
