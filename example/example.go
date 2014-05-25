package main

import (
	"html/template"
	"strings"

	"github.com/codegangsta/martini"
	"github.com/gobuild/middleware"
)

func main() {
	m := martini.Classic()
	var funcMap = template.FuncMap{"title": strings.Title}
	m.Use(middleware.ContextWithCookieSecret("", middleware.Options{
		Funcs: []template.FuncMap{funcMap},
	}))

	m.Post("/hello", func(ctx *middleware.Context) {
		ctx.WriteString("Hello World!")
	})
	m.Get("/home", func(ctx *middleware.Context) {
		ctx.Data["Title"] = "home"
		ctx.HTML(200, "home")
	})
	m.Run()
}
