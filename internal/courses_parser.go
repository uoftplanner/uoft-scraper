package internal

import (
	"encoding/json"
	"github.com/PuerkitoBio/goquery"
	"github.com/RediSearch/redisearch-go/redisearch"
	"github.com/gocolly/colly/v2"
	"log"
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
	Code                    string     `json:"code"`
	Name                    string     `json:"name"`
	Division                string     `json:"division" field:"Division"`
	Description             string     `json:"description" field:"Course Description"`
	Department              string     `json:"department" field:"Department"`
	Prerequisites           string     `json:"prerequisites" field:"Pre-requisites"`
	Corequisites            string     `json:"corequisites" field:"Corequisite"`
	Exclusions              string     `json:"exclusions" field:"Exclusion"`
	RecommendedPreparation  string     `json:"recommendedPrep" field:"Recommended Preparation"`
	Level                   string     `json:"level" field:"Course Level"`
	UTSCBreadth             string     `json:"utscBreadth" field:"UTSC Breadth"`
	UTMDistribution         string     `json:"utmDistribution" field:"UTM Distribution"`
	ArtsScienceBreadth      string     `json:"artsSciBreadth" field:"Arts and Science Breadth"`
	ArtsScienceDistribution string     `json:"artsSciDistribution" field:"Arts and Science Distribution"`
	APSCElectives           string     `json:"apscElectives" field:"APSC Electives"`
	Campus                  string     `json:"campus" field:"Campus"`
	Term                    string     `json:"term" field:"Term"`
	Schedule                []Activity `json:"schedule"`
}

type Activity struct {
	Name       string `json:"name"`
	DayAndTime string `json:"dayAndTime"`
	Instructor string `json:"instructor"`
	Location   string `json:"location"`
	ClassSize  int    `json:"classSize"`
	Enrolment  int    `json:"enrolment"`
	Waitlist   bool   `json:"waitlist"`
	Delivery   string `json:"delivery"`
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
			log.Println(err)
			return
		}

		course.Code = e.Request.Ctx.Get("code")
		course.Name = e.Request.Ctx.Get("name")

		log.Println("Found " + course.Code)

		e.DOM.Find("tbody tr").Each(func(i int, s *goquery.Selection) {
			columns := s.Children()

			name := strings.TrimSpace(columns.Eq(0).Text())
			dayAndTime := strings.TrimSpace(columns.Eq(1).Text())
			instructor := strings.TrimSpace(columns.Eq(2).Text())
			location := strings.TrimSpace(columns.Eq(3).Text())
			classSize, _ := strconv.Atoi(strings.TrimSpace(columns.Eq(4).Text()))
			enrolment, _ := strconv.Atoi(strings.TrimSpace(columns.Eq(5).Text()))
			waitlist := false
			delivery := strings.TrimSpace(columns.Eq(7).Text())

			waitlistImage, _ := columns.Eq(6).Find("img").Attr("src")

			if strings.Contains(waitlistImage, "checkmark") {
				waitlist = true
			}

			course.Schedule = append(course.Schedule, Activity{
				name,
				dayAndTime,
				instructor,
				location,
				classSize,
				enrolment,
				waitlist,
				delivery,
			})
		})

		cp.addCourseToDatabase(course)
	})

	c.OnError(func(response *colly.Response, err error) {
		log.Println(err.Error())
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
		log.Println(err)
		return
	}

	if err := p.redis.IndexOptions(indexingOptions, doc); err != nil {
		log.Println(err)
	}
}

func (p *CoursesParser) updateCourse(path string, code string, name string) {
	ctx := colly.NewContext()

	ctx.Put("code", code)
	ctx.Put("name", name)

	err := p.collector.Request(http.MethodGet, courseURL+path, nil, ctx, nil)

	if err != nil {
		log.Println(err)
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
			log.Println(err)
			return
		}

		// aaData is an array of course data
		aaData := res["aaData"].([]interface{})

		log.Println("Found " + strconv.Itoa(len(aaData)) + " courses")

		// course data is also stored in an array
		for _, course := range aaData {
			data := course.([]interface{})
			aTag := data[1].(string)

			d, err := goquery.NewDocumentFromReader(strings.NewReader(aTag))

			if err != nil {
				log.Println(err)
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
			log.Println(err)
		}

		// no need to parse body since we do not have a session
		response.Request.Abort()
	})

	if err := c.Visit(courseListURL); err != nil {
		// failed to retrieve courses
		log.Println(err)
		return
	}

	c.Wait()

	// wait until all courses are parsed
	p.collector.Wait()
}
