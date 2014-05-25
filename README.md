# middleware.Context
origin from **martini-contrib/web**

[API Reference](http://godoc.org/github.com/gobuild/middleware)

## Description
`middleware.Context` provides a render(forked from martini-contrib/render), beego like ctx.

## Usage

ref: <https://github.com/martini-contrib/render>

~~~ go
package main

import (
	"html/template"
	"strings"

	"github.com/go-martini/martini"
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
		ctx.HTML(200, "home") // use templates by default
	})
	m.Run()
}
~~~