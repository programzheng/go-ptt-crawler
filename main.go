package main

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gocolly/colly/v2"
)

const PTT_IMAGE_BOARD_HANDLER_CHUNK_SIZE = 10
const PTT_IMAGE_BOARD_HANDLER_LIMIT_SIZE = 100

func checkFileExist(filePath string) bool {
	_, err := os.Stat(filePath)
	return !os.IsNotExist(err)
}

func writeJsonFile(fileName string, data []string) int {
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

	return len(data)
}

func pttImageBoardHandler(ctx *gin.Context) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("exit crawl", r)
		}
	}()

	//make a []string slice len is 0 and cap is PTT_IMAGE_BOARD_HANDLER_CHUNK_SIZE
	images := make([]string, 0, PTT_IMAGE_BOARD_HANDLER_CHUNK_SIZE)

	board := ctx.Param("board")
	titlePrefix := fmt.Sprintf("[%v]", ctx.Query("prefix"))
	titlePrefixMd5 := md5.Sum([]byte("_" + titlePrefix))
	fileName := fmt.Sprintf("ptt_images_%v_%x.json", board, titlePrefixMd5)
	baseUrl := "https://www.ptt.cc"
	url := fmt.Sprintf("/bbs/%v", board)
	c := colly.NewCollector()

	// Find and visit all article links
	c.OnHTML("div.r-ent a[href]", func(e *colly.HTMLElement) {
		text := e.Text
		link := e.Attr("href")

		//filter text and link
		if strings.HasPrefix(text, "[公告]") || !strings.HasPrefix(link, url+"/M.") {
			return
		}
		if titlePrefix != "[]" && !strings.HasPrefix(text, titlePrefix) {
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

		//write json
		if len(images) == cap(images) {
			currentImageNumber := writeJsonFile(fileName, images)
			images = make([]string, 0, PTT_IMAGE_BOARD_HANDLER_CHUNK_SIZE)
			if currentImageNumber >= PTT_IMAGE_BOARD_HANDLER_LIMIT_SIZE {
				panic("images write json is finish")
			}
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
