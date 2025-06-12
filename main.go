// Copyright 2025 The OpenCmd Authors. All rights reserved.
// Use of this source code is governed by a GPL-3.0
// license that can be found in the LICENSE file.
package main

import (
	"fmt"
	"github.com/opencommand/echochic"
	"opencmd-cookbook/config"
	"opencmd-cookbook/handler"
	"opencmd-cookbook/routes"
	octmpl "opencmd-cookbook/template"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/gin-gonic/gin"
	"sigs.k8s.io/yaml"
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

func renderStatusColor(code int) string {
	var styled lipgloss.Style

	switch {
	case code >= 200 && code < 300:
		styled = echochic.Green
	case code >= 300 && code < 400:
		styled = echochic.BlueDark
	case code >= 400 && code < 500:
		styled = echochic.Yellow
	case code >= 500:
		styled = echochic.Red
	default:
		styled = echochic.Grey
	}

	return styled.Render(fmt.Sprintf("%d", code))
}

func renderHTTPMethod(method string) string {
	var styled lipgloss.Style

	switch strings.ToUpper(method) {
	case "GET":
		styled = echochic.Green
	case "POST":
		styled = echochic.Blue
	case "PUT":
		styled = echochic.Yellow
	case "DELETE":
		styled = echochic.Red
	case "PATCH":
		styled = echochic.Pink
	case "OPTIONS":
		styled = echochic.GreyDark
	case "HEAD":
		styled = echochic.Grey
	default:
		styled = echochic.Bold
	}

	return styled.Render(method)
}

func main() {
	startTime := time.Now()
	var cfg config.Configure
	data, _ := os.ReadFile("boot.yaml")
	yaml.Unmarshal(data, &cfg)
	gin.SetMode(gin.ReleaseMode)
	go handler.WatchWeb("./src")

	router := gin.New()
	router.Use(gin.LoggerWithConfig(gin.LoggerConfig{
		Formatter: func(params gin.LogFormatterParams) string {
			return fmt.Sprintln(echochic.Styled().
				Grey(params.TimeStamp.Format("15:04:05")).
				Space().
				With(echochic.Bold, echochic.Blue).
				Text("[GIN]").
				Space().
				Text(renderStatusColor(params.StatusCode)).
				Space().
				Grey(params.Latency.String()).
				Space().
				GreyDark(params.ClientIP).
				Space().
				Text(renderHTTPMethod(params.Method)).
				Space().
				BlueDark(params.Path).
				String())
		},
	}), gin.Recovery())
	routes.Register(router, []routes.Route{
		&routes.StaticRoute{},
		&routes.PagesRoute{},
		&routes.HotfixRoute{},
	})
	uptime := time.Since(startTime).Milliseconds()
	
	fmt.Println(echochic.Styled().
		Newline().
		Indent(2).
		With(echochic.Bold, echochic.Green, echochic.Italic).
		Text("opencmd-cookbook").
		Space().
		Green("v1.0.0").
		Space(2).
		Grey("ready in").
		Space().
		Bold(strconv.Itoa(int(uptime))).
		Space().
		Text("ms").
		String())

	fmt.Println(echochic.Styled().
		Newline().
		Space(2).
		Green("➜").
		Space(2).
		Bold("Local:").
		Space(4).
		BlueDark("http://localhost:").
		With(echochic.Bold, echochic.Blue).
		Text(strconv.Itoa(int(cfg.Port))).
		String())

	fmt.Println(echochic.Styled().
		Space(2).
		GreenDark("➜").
		Space(2).
		With(echochic.Bold, echochic.Grey).
		Text("Network:").
		Space(2).
		Grey("set").
		Space().
		Bold("host / port").
		Space().
		Grey("in").
		Space().
		Bold("boot.yaml").
		Space().
		Grey("to expose").
		String())

	fmt.Println(echochic.Styled().
		Space(2).
		GreenDark("➜").
		Space(2).
		Grey("press").
		Space().
		Bold("ctrl + c").
		Space().
		Grey("to exit").
		String())
	fmt.Println()
	router.Run(fmt.Sprintf("%s:%d", cfg.Host, cfg.Port))
}
