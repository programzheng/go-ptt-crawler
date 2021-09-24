package main

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gocolly/colly/v2"
)

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
	chunkSize, err := strconv.Atoi(ctx.DefaultQuery("chunk_size", "30"))
	if err != nil {
		log.Fatalf("chunk size strconv.Atoi failed: %v", err)
	}
	limitSize, err := strconv.Atoi(ctx.DefaultQuery("limit_size", "1"))
	if err != nil {
		log.Fatalf("limit size strconv.Atoi failed: %v", err)
	}

	//make a []string slice len is 0 and cap is chunkSize
	images := make([]string, 0, chunkSize)
	currentImageTotal := 0

	board := ctx.Param("board")
	titlePrefix := fmt.Sprintf("[%v]", ctx.Query("prefix"))
	titlePrefixMd5 := md5.Sum([]byte("_" + titlePrefix))
	fileName := fmt.Sprintf("ptt_images_%v_%x.json", board, titlePrefixMd5)
	baseUrl := "https://www.ptt.cc"
	url := fmt.Sprintf("/bbs/%v", board)
	c := colly.NewCollector(
		colly.Async(true),
	)
	c.Limit(&colly.LimitRule{
		DomainGlob:  "*",
		Parallelism: chunkSize,
	})

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
			currentImageTotal = writeJsonFile(fileName, images)
			images = make([]string, 0, chunkSize)
		} else {
			images = append(images, link)
		}
	})

	c.OnRequest(func(r *colly.Request) {
		if limitSize != -1 && currentImageTotal >= limitSize {
			r.Abort()
		}
		//set cookie
		r.Headers.Set("Cookie", "over18=1")
		fmt.Println("Visiting", r.URL)
	})

	c.Visit(baseUrl + url + "/index.html")

	c.Wait()
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
