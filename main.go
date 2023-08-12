package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-co-op/gocron"
	"github.com/programzheng/go-ptt-crawler/pkg/aws"
	"github.com/programzheng/go-ptt-crawler/pkg/images"

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

	images.PttImageBoard(board, titlePrefix, chunkSize, limitSize, true)

	ctx.JSON(200, gin.H{
		"status": "success",
	})
}

func pttRandomImageBoardHandler(ctx *gin.Context) {
	board := ctx.Param("board")
	titlePrefix := fmt.Sprintf("[%v]", ctx.Query("prefix"))
	image := images.PttRandomImageBoard(board, titlePrefix)
	bufBytes, contentType, err := images.GetImageBufferBytesAndContentTypeByUrl(image)
	if err != nil {
		log.Printf("pttRandomImageBoardHandler images.GetImageBufferBytesAndContentTypeByUrl(%s) error: %v", image, err)
	}

	ctx.Data(http.StatusOK, contentType, bufBytes)
}

func pttRandomImageUrlBoardHandler(ctx *gin.Context) {
	board := ctx.Param("board")
	titlePrefix := fmt.Sprintf("[%v]", ctx.Query("prefix"))
	image := images.PttRandomImageBoard(board, titlePrefix)

	ctx.JSON(http.StatusOK, gin.H{
		"status": "success",
		"url":    image,
	})
}

func setupRouter() *gin.Engine {
	router := gin.Default()
	//router
	apiGroup := router.Group("/api/v1")
	{
		apiGroup.GET("ptt/image/:board", pttImageBoardHandler)
	}
	router.GET("ptt/image/:board/random", pttRandomImageBoardHandler)
	router.GET("ptt/image_url/:board/random", pttRandomImageUrlBoardHandler)
	return router
}

func main() {
	if os.Getenv("SCHEDULE") == "true" {
		runSchedules()
	}

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

func runSchedules() {
	s := gocron.NewScheduler(time.Now().Local().Location())
	job, err := s.Cron("0 0 * * *").Do(func() {
		scheduleBoards := os.Getenv("SCHEDULE_BOARDS")
		scheduleBoardLimit, err := strconv.Atoi(os.Getenv("SCHEDULE_BOARD_LIMIT"))
		if err != nil {
			log.Printf("scheduleBoard strconv.Atoi(os.Getenv(\"SCHEDULE_BOARD_LIMIT\")) error: %v", err)
		}
		if scheduleBoards != "" {
			for _, board := range strings.Split(scheduleBoards, ",") {
				images.PttImageBoard(board, "", 1000, scheduleBoardLimit, true)
				log.Printf("[runSchedules] %s", board)
			}
		}
	}) // every daily
	if err != nil {
		job.SingletonMode()
	}
	s.StartAsync()
}
