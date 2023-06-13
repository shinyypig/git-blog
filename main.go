package main

import (
	"shinyypig/gitblog/router"
)

func main() {
	go router.AnaylzePosts()
	router.RunBlogServer()
}
