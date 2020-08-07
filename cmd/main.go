package main

import (
	"uoft-scraper/internal"
)

func main() {
	var db internal.DatabaseHandler
	db = internal.NewRedisDatabase("127.0.0.1:6379", "", 0)
	parser := internal.NewCoursesParser(&db)

	parser.UpdateData()
}
