package main

import (
	"github.com/RediSearch/redisearch-go/redisearch"
	"log"
	"os"
	"uoft-scraper/internal"
)

func main() {
	host, exists := os.LookupEnv("REDIS_HOST")

	if !exists {
		host = "127.0.0.1:6379"
	}

	rc := initializeRediSearch(host)
	parser := internal.NewCoursesParser(rc)

	parser.UpdateData()
}

func initializeRediSearch(address string) *redisearch.Client {
	rs := redisearch.NewClient(address, "course")

	sc := redisearch.NewSchema(redisearch.DefaultOptions).
		AddField(redisearch.NewSortableTextField("code", 1.0)).
		AddField(redisearch.NewTextField("name")).
		AddField(redisearch.NewTextFieldOptions("json", redisearch.TextFieldOptions{NoIndex: true}))

	if err := rs.CreateIndex(sc); err != nil {
		log.Println(err)
	}

	return rs
}
