package images

import (
	"bytes"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/programzheng/go-ptt-crawler/pkg/aws"

	"github.com/gocolly/colly"
	"github.com/gocolly/colly/debug"
)

var JSON_FILE_DATE string = time.Now().Format("2006-01-02")

func checkFileExist(filePath string) bool {
	_, err := os.Stat(filePath)
	return !os.IsNotExist(err)
}

func getJsonFileData(fileName string) []string {
	var oldJsonData []string
	if checkFileExist(fileName) {
		oldJsonDataByte, err := os.ReadFile(aws.LambdaTmpDir() + fileName)
		if err != nil {
			log.Fatalf("get old images json file data error:\n%v", err)
		}
		json.Unmarshal(oldJsonDataByte, &oldJsonData)
	}
	return oldJsonData
}

func writeJsonFile(fileName string, data []string) int {
	oldJsonData := getJsonFileData(fileName)
	data = append(data, oldJsonData...)
	jsonData, err := json.Marshal(data)
	if err != nil {
		log.Fatalf("images to json data error:\n%v", err)
	}
	err = os.WriteFile(aws.LambdaTmpDir()+fileName, jsonData, 0644)
	if err != nil {
		log.Fatalf("images json data to json file error:\n%v", err)
	}

	return len(data)
}

// PttImageBoard is a function for crawling a specific board on PTT.
// Parameters:
//
//	board: The name of the board to crawl.
//	titlePrefix: The prefix used to filter article titles. It can be an empty string.
//	chunkSize: The number of articles to request per chunk.
//	limitSize: The maximum number of articles to crawl.
//	write: A boolean indicating whether to write the results to a file.
//
// Returns: A list of image links crawled.
func PttImageBoard(board string, titlePrefix string, chunkSize int, limitSize int, write bool) []string {
	titlePrefixMd5 := md5.Sum([]byte("_" + titlePrefix))
	fileName := fmt.Sprintf("ptt_images_%v_%x_%v.json", board, titlePrefixMd5, JSON_FILE_DATE)

	//make a []string slice len is 0 and cap is chunkSize
	images := make([]string, 0, chunkSize)
	currentImageTotal := 0

	baseUrl := "https://www.ptt.cc"
	url := fmt.Sprintf("/bbs/%v", board)

	c := colly.NewCollector(
		colly.Async(true),
	)
	if os.Getenv("DEBUG") == "true" {
		c = colly.NewCollector(
			colly.Debugger(&debug.LogDebugger{}),
			colly.Async(true),
		)
	}

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
			if !write {
				return
			}
			if limitSize != -1 && currentImageTotal >= limitSize {
				return
			}
			currentImageTotal = writeJsonFile(fileName, images)
			images = make([]string, 0, chunkSize)
		} else {
			images = append(images, link)
		}
	})

	c.OnRequest(func(r *colly.Request) {
		if !write && len(images) == cap(images) {
			r.Abort()
		}
		if limitSize != -1 && currentImageTotal >= limitSize {
			r.Abort()
		}
		//set cookie
		r.Headers.Set("Cookie", "over18=1")
	})

	c.Visit(baseUrl + url + "/index.html")

	c.Wait()

	return images
}

func PttRandomImageBoard(board string, titlePrefix string) string {
	resultCh := make(chan string, 1)
	titlePrefixMd5 := md5.Sum([]byte("_" + titlePrefix))
	fileName := fmt.Sprintf("ptt_images_%v_%x_%v.json", board, titlePrefixMd5, JSON_FILE_DATE)
	go func() {
		oldJsonData := getJsonFileData(fileName)
		if len(oldJsonData) == 0 {
			oldJsonData = PttImageBoard(board, titlePrefix, 30, 120, false)
		}
		rand.Seed(time.Now().Unix())
		resultCh <- oldJsonData[rand.Intn(len(oldJsonData))]
	}()
	return <-resultCh
}

func GetImageBufferBytesAndContentTypeByUrl(url string) ([]byte, string, error) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Printf("getImageBufferBytes url:%s error:%v", url, err)
		return nil, "", err
	}
	req.Header.Set("User-Agent", "")

	response, err := client.Do(req)
	if err != nil || response.StatusCode != http.StatusOK {
		log.Printf("getImageBufferBytesAndContentType http get image %s failed statusCode:%d error: %v", url, response.StatusCode, err)
		return nil, "", err
	}

	buf := new(bytes.Buffer)
	buf.ReadFrom(response.Body)
	reader := response.Body
	defer reader.Close()
	contentType := response.Header.Get("Content-Type")

	return buf.Bytes(), contentType, nil
}
