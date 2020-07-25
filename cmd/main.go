package main

import (
	"uoft-scraper/internal"
)

func main() {
	var db internal.DatabaseHandler
	db = internal.NewMemoryDatabase()
	parser := internal.NewCoursesParser(&db)

	parser.UpdateData()
}
