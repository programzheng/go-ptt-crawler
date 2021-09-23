package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gocolly/colly/v2"
	"github.com/gocolly/colly/v2/debug"
)

const PTT_IMAGE_BOARD_HANDLER_CHUNK_SIZE = 10
const PTT_IMAGE_BOARD_HANDLER_LIMIT_SIZE = 100

func checkFileExist(filePath string) bool {
	_, err := os.Stat(filePath)
	return !os.IsNotExist(err)
}

func writeJsonFile(fileName string, data []string) {
	if checkFileExist(fileName) {
		appendJsonDataByte, err := ioutil.ReadFile(fileName)
		if err != nil {
			log.Fatalf("get old images json file data error:\n%v", err)
		}
		var appendJsonData []string
		json.Unmarshal(appendJsonDataByte, &appendJsonData)
		appendJsonData = append(appendJsonData, data...)
		data = appendJsonData
	}
	jsonData, err := json.Marshal(data)
	if err != nil {
		log.Fatalf("images to json data error:\n%v", err)
	}
	err = ioutil.WriteFile(fileName, jsonData, 0644)
	if err != nil {
		log.Fatalf("images json data to json file error:\n%v", err)
	}
}

func pttImageBoardHandler(ctx *gin.Context) {
	var images []string

	board := ctx.Param("board")
	baseUrl := "https://www.ptt.cc/"
	url := fmt.Sprintf("/bbs/%v", board)
	c := colly.NewCollector(colly.Debugger(&debug.LogDebugger{}))

	// Find and visit all article links
	c.OnHTML("div.r-ent a[href]", func(e *colly.HTMLElement) {
		//filter 公告
		text := e.Text
		link := e.Attr("href")
		if strings.HasPrefix(text, "[公告]") || !strings.HasPrefix(link, url+"/M.") {
			return
		}
			return
		}
		articleLink := baseUrl + link
		e.Request.Visit(articleLink)
	})

	//Find previous page link
	c.OnHTML("div.btn-group.btn-group-paging a[href]", func(e *colly.HTMLElement) {
		text := e.Text
		link := e.Attr("href")
		if text == "‹ 上頁" {
			prevLink := baseUrl + link
			e.Request.Visit(prevLink)
		}
	})

	//Find article content image link
	c.OnHTML("a[href]", func(e *colly.HTMLElement) {
		parent := e.DOM.Parent()
		parentClass, _ := parent.Attr("class")
		//filter push message
		if strings.Contains(parentClass, "push-content") {
			return
		}
		link := e.Attr("href")
		//only get imgur image
		if !strings.HasPrefix(link, "https://i.imgur.com") {
			return
		}

		fmt.Println(link)
		//write json
		if len(images) > 0 && len(images)%PTT_IMAGE_BOARD_HANDLER_CHUNK_SIZE == 0 {
			fmt.Println(images)
			fileName := fmt.Sprintf("ptt_images_%v.json", board)
			writeJsonFile(fileName, images)
		} else if len(images) >= PTT_IMAGE_BOARD_HANDLER_LIMIT_SIZE {
			panic("Exit")
		} else {
			images = append(images, link)
		}
	})

	c.OnRequest(func(r *colly.Request) {
		//set cookie
		r.Headers.Set("Cookie", "over18=1")
		fmt.Println("Visiting", r.URL)
	})

	c.Visit(baseUrl + url + "/index.html")
}

func main() {
	router := gin.Default()
	//router
	apiGroup := router.Group("/api/v1")
	{
		apiGroup.GET("ptt/image/:board", pttImageBoardHandler)
	}
	router.Run()
}
