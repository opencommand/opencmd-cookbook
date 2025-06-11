// Copyright 2025 The OpenCmd Authors. All rights reserved.
// Use of this source code is governed by a GPL-3.0
// license that can be found in the LICENSE file.
package main

import (
	"opencmd-cookbook/handler"
	"opencmd-cookbook/routes"
	octmpl "opencmd-cookbook/template"

	"github.com/gin-gonic/gin"
)

func init() {
	if err := octmpl.Reg.Load("detail", []string{"src/views/base.html", "src/views/detail.html"}); err != nil {
		panic(err)
	}
	if err := octmpl.Reg.Load("home", []string{"src/components/base.html", "src/index.html", "src/components/macros.html", "src/components/footer.html"}); err != nil {
		panic(err)
	}
	if err := octmpl.Reg.Load("explain", []string{"src/views/base.html", "src/views/explain.html", "src/components/macros.html"}); err != nil {
		panic(err)
	}
	if err := octmpl.Reg.Load("about", []string{"src/views/about.html", "src/views/base.html", "src/components/macros.html", "src/components/footer.html"}); err != nil {
		panic(err)
	}
}

func main() {
	go handler.WatchWeb("./src")

	router := gin.Default()
	routes.Register(router, []routes.Route{
		&routes.StaticRoute{},
		&routes.PagesRoute{},
		&routes.HotfixRoute{},
	})

	router.Run(":8080")
}
