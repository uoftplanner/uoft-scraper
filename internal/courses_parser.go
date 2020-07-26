package internal

import (
	"encoding/json"
	"fmt"
	"github.com/gocolly/colly/v2"
	"reflect"
	"strings"
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

var fieldSelectors = make(map[string]string)

func init() {
	var c Course
	t := reflect.TypeOf(c)

	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)

		if v, ok := f.Tag.Lookup("field"); ok {
			// no need to check for empty since the tag only ever exists or does not
			fieldSelectors[f.Name] = getFieldSelector(v)
			fmt.Println(f.Name + ": " + getFieldSelector(v))
		}
	}
}

func getFieldSelector(f string) string {
	return `div[data-label='` + f + `'] span:nth-child(2)`
}

type CoursesParser struct {
	collector *colly.Collector
}

func NewCoursesParser(db *DatabaseHandler) *CoursesParser {
	c := colly.NewCollector()

	c.OnHTML("#correctPage", func(e *colly.HTMLElement) {
		course := new(Course)

		if err := e.UnmarshalWithMap(course, fieldSelectors); err != nil {
			// TODO: proper error handling
			fmt.Println(err)
			return
		}

		title := strings.Split(e.ChildText(".uif-headerText-span"), ":")
		course.Code = title[0]
		course.Name = strings.TrimLeft(title[1], " ")

		if d, err := json.Marshal(course); err == nil {
			(*db).Put(course.Code, string(d))
		} else {
			// TODO: proper error handling
			fmt.Println(err)
			return
		}

		fmt.Println(course.Name)
		fmt.Println(course.Division)
		fmt.Println(course.Description)
		fmt.Println(course.Department)
		fmt.Println(course.Prerequisites)
		fmt.Println(course.Corequisites)
		fmt.Println(course.Level)
		fmt.Println(course.ArtsScienceBreadth)
		fmt.Println(course.ArtsScienceDistribution)
		fmt.Println(course.Campus)
		fmt.Println(course.Term)
	})

	return &CoursesParser{c}
}

func (p *CoursesParser) updateCourse(url string) {
	if err := p.collector.Visit(url); err != nil {
		// TODO: proper error handling
		fmt.Println(err)
	}
}

func (p *CoursesParser) UpdateData() {
	// TODO: visit all courses on uoft coursefinder
	p.updateCourse("https://coursefinder.utoronto.ca/course-search/search/courseInquiry?methodToCall=start&viewId=CourseDetails-InquiryView&courseId=MAT240H1F20209")
}
