package main

import (
	"fmt"
	"github.com/RediSearch/redisearch-go/redisearch"
	"uoft-scraper/internal"
)

func main() {
	rc := initializeRediSearch("127.0.0.1:6379")
	parser := internal.NewCoursesParser(rc)

	parser.UpdateData()
}

func initializeRediSearch(address string) *redisearch.Client {
	rs := redisearch.NewClient(address, "course")
	redisearch.NewQuery("AUTH somepassword")

	sc := redisearch.NewSchema(redisearch.DefaultOptions).
		AddField(redisearch.NewTextFieldOptions("code", redisearch.TextFieldOptions{Sortable: true})).
		AddField(redisearch.NewTextField("name")).
		AddField(redisearch.NewTextFieldOptions("json", redisearch.TextFieldOptions{NoIndex: true}))

	if err := rs.CreateIndex(sc); err != nil {
		fmt.Println(err)
	}

	return rs
}
