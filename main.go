// Copyright 2025 The OpenCmd Authors. All rights reserved.
// Use of this source code is governed by a GPL-3.0
// license that can be found in the LICENSE file.
package main

import (
	"html/template"
	"net/http"
	"opencmd/model"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"sigs.k8s.io/yaml"
)

// Suggestion 表示建议的结构
type Suggestion struct {
	Cmd  string
	Text string
	Link string
}

type OriginalCommand struct {
	Name       string
	MatchClass string
}

// Match 表示命令匹配的结构
type Match struct {
	ID           string
	Name         string
	Section      string
	Source       string
	Match        string
	Spaces       string
	CommandClass string
	HelpClass    string
	Suggestions  []Suggestion
}

// HelpText 表示帮助文本的结构
type HelpText struct {
	ID   string
	Text string
}

// Config 表示配置结构
type Config struct {
	DEBUG bool
}

// PageData 表示页面数据的结构
type PageData struct {
	GetArgs         string
	OriginalCommand []OriginalCommand
	Matches         []Match
	HelpText        []HelpText
	Config          Config
}

// DetailHelpText 表示详情页帮助文本的结构
type DetailHelpText struct {
	Option      string
	Description string
}

// DetailData 表示详情页数据的结构
type DetailData struct {
	Synopsis    string
	Description string
	HelpText    []DetailHelpText
	Config      Config
}

func CommandDocument(ctx *gin.Context) {
	// TODO cache
	cmdName := ctx.Param("cmdName")

	if len(cmdName) == 0 {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Command name is required"})
		return
	}

	filePath := filepath.Join("docs", cmdName+".yaml")

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "Command document not found"})
		return
	}

	yamlFile, err := os.ReadFile(filePath)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read command document"})
		return
	}

	var commandSchema model.CommandSchema
	err = yaml.Unmarshal(yamlFile, &commandSchema)
	if err != nil {
		logrus.Error(err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to unmarshal command document"})
		return
	}

	if len(commandSchema.Commands) == 0 {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "Command document not found"})
		return
	}

	helps := []DetailHelpText{}

	for _, help := range commandSchema.Commands[0].Options {
		helps = append(helps, DetailHelpText{
			Option:      strings.Join(help.Alias, ", "),
			Description: help.Description,
		})
	}

	detailData := DetailData{
		Synopsis:    commandSchema.Commands[0].Synopsis,
		Description: commandSchema.Commands[0].Description,
		HelpText:    helps,
	}

	if err := detailTemplate.ExecuteTemplate(ctx.Writer, "base", detailData); err != nil {
		http.Error(ctx.Writer, err.Error(), http.StatusInternalServerError)
	}
}

var detailTemplate = template.Must(template.ParseFiles(
	"templates/base.tmpl",
	"templates/detail.tmpl",
))

func main() {
	// tmpl := template.Must(template.ParseFiles(
	// 	"templates/base.tmpl",
	// 	"templates/explain.tmpl",
	// 	"templates/macros.tmpl",
	// ))

	// http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
	// 	data := PageData{
	// 		GetArgs: "ls -l | grep 'test'",
	// 		OriginalCommand: []OriginalCommand{
	// 			{
	// 				Name:       "ls",
	// 				MatchClass: "1",
	// 			},
	// 			{
	// 				Name:       "-l",
	// 				MatchClass: "2",
	// 			},
	// 			{
	// 				Name:       "|",
	// 				MatchClass: "3",
	// 			},
	// 			{
	// 				Name:       "grep",
	// 				MatchClass: "4",
	// 			},
	// 			{
	// 				Name:       `"test"`,
	// 				MatchClass: "5",
	// 			},
	// 		},
	// 		Matches: []Match{
	// 			{
	// 				ID:           "1",
	// 				Name:         "ls",
	// 				Section:      "",
	// 				Source:       "ls",
	// 				Match:        "ls",
	// 				Spaces:       " ",
	// 				CommandClass: "command",
	// 				HelpClass:    "help1",
	// 				Suggestions: []Suggestion{
	// 					{Cmd: "ls -a", Text: "Show all files", Link: "ls-a"},
	// 					{Cmd: "ls -la", Text: "Show all files in long format", Link: "ls-la"},
	// 				},
	// 			},
	// 			{
	// 				ID:           "2",
	// 				Name:         "-l",
	// 				Section:      "",
	// 				Source:       "",
	// 				Match:        "-l",
	// 				Spaces:       " ",
	// 				CommandClass: "command",
	// 				HelpClass:    "help2",
	// 				Suggestions:  []Suggestion{},
	// 			},
	// 			{
	// 				ID:           "3",
	// 				Name:         "|",
	// 				Section:      "",
	// 				Source:       "",
	// 				Match:        "|",
	// 				Spaces:       " ",
	// 				CommandClass: "command",
	// 				HelpClass:    "help3",
	// 				Suggestions:  []Suggestion{},
	// 			},
	// 			{
	// 				ID:           "4",
	// 				Name:         "grep",
	// 				Section:      "",
	// 				Source:       "grep",
	// 				Match:        "grep",
	// 				Spaces:       " ",
	// 				CommandClass: "command",
	// 				HelpClass:    "help4",
	// 				Suggestions: []Suggestion{
	// 					{Cmd: "grep -i", Text: "Case-insensitive search", Link: "grep-i"},
	// 					{Cmd: "grep -r", Text: "Recursive search", Link: "grep-r"},
	// 				},
	// 			},
	// 			{
	// 				ID:           "5",
	// 				Name:         "'test'",
	// 				Section:      "",
	// 				Source:       "",
	// 				Match:        "'test'",
	// 				Spaces:       "",
	// 				CommandClass: "command",
	// 				HelpClass:    "help5",
	// 				Suggestions:  []Suggestion{},
	// 			},
	// 		},
	// 		HelpText: []HelpText{
	// 			{
	// 				ID:   "help1",
	// 				Text: "List directory contents",
	// 			},
	// 			{
	// 				ID:   "help2",
	// 				Text: "Use a long listing format",
	// 			},
	// 			{
	// 				ID:   "help3",
	// 				Text: "Pipe operator - redirects output of one command to input of another",
	// 			},
	// 			{
	// 				ID:   "help4",
	// 				Text: "Search for patterns in files",
	// 			},
	// 			{
	// 				ID:   "help5",
	// 				Text: "Search pattern to match",
	// 			},
	// 		},
	// 		Config: Config{
	// 			DEBUG: true,
	// 		},
	// 	}

	// 	if err := tmpl.ExecuteTemplate(w, "base", data); err != nil {
	// 		http.Error(w, err.Error(), http.StatusInternalServerError)
	// 	}
	// })

	router := gin.Default()
	router.Static("/static", "./static")
	router.GET("/docs/:cmdName", CommandDocument)
	router.POST("/analyze", func(c *gin.Context) {
		var json struct {
			Command string `json:"command"`
		}
		if err := c.ShouldBindJSON(&json); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	})
	router.POST("/completion", func(c *gin.Context) {
		var json struct {
			Command string `json:"command"`
		}
		if err := c.ShouldBindJSON(&json); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		}
	})

	router.Run(":8080")
}
