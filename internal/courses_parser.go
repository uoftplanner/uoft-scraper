package internal

import (
	"encoding/json"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/RediSearch/redisearch-go/redisearch"
	"github.com/gocolly/colly/v2"
	"net/http"
	"reflect"
	"strconv"
	"strings"
)

const (
	courseURL     = "https://coursefinder.utoronto.ca/course-search/search/"
	courseListURL = "https://coursefinder.utoronto.ca/course-search/search/courseSearch/course/search" +
		"?queryText=&requirements=&campusParam=St.%20George,Scarborough,Mississauga"
)

type Course struct {
	Code                    string
	Name                    string
	Division                string `field:"Division"`
	Description             string `field:"Course Description"`
	Department              string `field:"Department"`
	Prerequisites           string `field:"Pre-requisites"`
	Corequisites            string `field:"Corequisite"`
	Exclusions              string `field:"Exclusion"`
	RecommendedPreparation  string `field:"Recommended Preparation"`
	Level                   string `field:"Course Level"`
	UTSCBreadth             string `field:"UTSC Breadth"`
	UTMDistribution         string `field:"UTM Distribution"`
	ArtsScienceBreadth      string `field:"Arts and Science Breadth"`
	ArtsScienceDistribution string `field:"Arts and Science Distribution"`
	APSCElectives           string `field:"APSC Electives"`
	Campus                  string `field:"Campus"`
	Term                    string `field:"Term"`
}

var indexingOptions = redisearch.IndexingOptions{Replace: true, Partial: true}
var fieldSelectors = make(map[string]string)

func init() {
	var c Course
	t := reflect.TypeOf(c)

	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)

		if v, ok := f.Tag.Lookup("field"); ok {
			// no need to check for empty since the tag only ever exists or does not
			fieldSelectors[f.Name] = getFieldSelector(v)
		}
	}
}

func getFieldSelector(f string) string {
	return `div[data-label='` + f + `'] span:nth-child(2)`
}

type CoursesParser struct {
	collector *colly.Collector
	redis     *redisearch.Client
}

func NewCoursesParser(rc *redisearch.Client) *CoursesParser {
	c := colly.NewCollector(
		colly.Async(true),
	)

	cp := &CoursesParser{c, rc}

	// TODO: option to change concurrent connections and timeout
	_ = c.Limit(&colly.LimitRule{
		DomainGlob:  "*",
		Parallelism: 50,
	})

	c.DisableCookies()

	c.OnHTML("#correctPage", func(e *colly.HTMLElement) {
		course := new(Course)

		if err := e.UnmarshalWithMap(course, fieldSelectors); err != nil {
			fmt.Println(err)
			return
		}

		course.Code = e.Request.Ctx.Get("code")
		course.Name = e.Request.Ctx.Get("name")

		cp.addCourseToDatabase(course)
	})

	c.OnError(func(response *colly.Response, err error) {
		fmt.Println(err.Error())
	})

	return cp
}

func (p *CoursesParser) addCourseToDatabase(course *Course) {
	doc := redisearch.NewDocument(course.Code, 1.0)

	if d, err := json.Marshal(course); err == nil {
		doc.Set("code", course.Code)
		doc.Set("name", course.Name)
		doc.Set("json", string(d))
	} else {
		fmt.Println(err)
		return
	}

	if err := p.redis.IndexOptions(indexingOptions, doc); err != nil {
		fmt.Println(err)
	}
}

func (p *CoursesParser) updateCourse(path string, code string, name string) {
	ctx := colly.NewContext()

	ctx.Put("code", code)
	ctx.Put("name", name)

	err := p.collector.Request(http.MethodGet, courseURL+path, nil, ctx, nil)

	if err != nil {
		fmt.Println(err)
	}
}

func (p *CoursesParser) UpdateData() {
	c := colly.NewCollector(
		colly.AllowURLRevisit(),
		colly.Async(true),
	)

	c.OnResponse(func(response *colly.Response) {
		// the response is a JSON object with only one JSON object 'aaData'
		var res map[string]interface{}

		if err := json.Unmarshal(response.Body, &res); err != nil {
			fmt.Println(err)
			return
		}

		// aaData is an array of course data
		aaData := res["aaData"].([]interface{})

		fmt.Println("Found " + strconv.Itoa(len(aaData)) + " courses")

		// course data is also stored in an array
		for _, course := range aaData {
			data := course.([]interface{})
			aTag := data[1].(string)

			d, err := goquery.NewDocumentFromReader(strings.NewReader(aTag))

			if err != nil {
				fmt.Println(err)
				continue
			}

			coursePath, _ := d.Find("a").Attr("href")
			courseCode := d.Text()
			courseName := data[2].(string)

			p.updateCourse(coursePath, courseCode, courseName)
		}
	})

	c.OnResponseHeaders(func(response *colly.Response) {
		// if response is successful we do not need to obtain a session and retry
		if response.StatusCode == 200 {
			return
		}

		// retry after obtaining session
		if err := response.Request.Visit(courseListURL); err != nil {
			fmt.Println(err)
		}

		// no need to parse body since we do not have a session
		response.Request.Abort()
	})

	if err := c.Visit(courseListURL); err != nil {
		// failed to retrieve courses
		fmt.Println(err)
		return
	}

	c.Wait()

	// wait until all courses are parsed
	p.collector.Wait()
}
