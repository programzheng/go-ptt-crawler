package main

import (
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gocolly/colly/v2"
	"github.com/gocolly/colly/v2/debug"
)

func pttImageBoardHandler(ctx *gin.Context) {
	board := ctx.Param("board")
	baseUrl := "https://www.ptt.cc/"
	url := fmt.Sprintf("/bbs/%v", board)
	c := colly.NewCollector(colly.Debugger(&debug.LogDebugger{}))

	// Find and visit all article links
	c.OnHTML("div.r-ent a[href]", func(e *colly.HTMLElement) {
		//filter 公告
		title := e.Text
		link := e.Attr("href")
		if strings.HasPrefix(title, "[公告]") || !strings.HasPrefix(link, url+"/M.") {
			return
		}
		articleLink := baseUrl + link
		e.Request.Visit(articleLink)
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
