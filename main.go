// Copyright 2025 The OpenCmd Authors. All rights reserved.
// Use of this source code is governed by a GPL-3.0
// license that can be found in the LICENSE file.
package main

import (
	"fmt"
	"html/template"
	"net/http"
	"opencmd/model"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
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

	tmplMu.RLock()
	err = detailTemplate.ExecuteTemplate(ctx.Writer, "base", detailData)
	tmplMu.RUnlock()
	if err != nil {
		http.Error(ctx.Writer, err.Error(), http.StatusInternalServerError)
	}
}

var (
	tmplMu         sync.RWMutex
	tmpl           *template.Template
	detailTemplate *template.Template
	indexTmpl      *template.Template
	aboutTmpl      *template.Template
)

func loadTemplates() error {
	t, err := template.ParseFiles(
		"src/views/base.html",
		"src/views/explain.html",
		"src/components/macros.html",
	)
	if err != nil {
		return err
	}
	dt, err := template.ParseFiles(
		"src/views/base.html",
		"src/views/detail.html",
	)
	if err != nil {
		return err
	}
	it, err := template.ParseFiles(
		"src/components/base.html",
		"src/index.html",
		"src/components/macros.html",
		"src/components/footer.html")
	if err != nil {
		return err
	}
	at, err := template.ParseFiles(
		"src/views/about.html",
		"src/views/base.html",
		"src/components/macros.html",
		"src/components/footer.html",
	)
	if err != nil {
		return err
	}

	tmplMu.Lock()
	defer tmplMu.Unlock()
	tmpl = t
	detailTemplate = dt
	indexTmpl = it
	aboutTmpl = at
	return nil
}

func sendTemplateUpdateNotification() {
	if wsConn != nil {
		err := wsConn.WriteMessage(websocket.TextMessage, []byte("reload"))
		if err != nil {
			logrus.Error("Failed to send WebSocket message:", err)
		}
	}
}

// 在模板变动时调用
func watchTemplates() {
	updateTemplateModTime()
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		logrus.Fatalf("failed to create fsnotify watcher: %v", err)
	}
	defer watcher.Close()

	err = watcher.Add("src")
	if err != nil {
		logrus.Fatalf("failed to watch templates folder: %v", err)
	}

	logrus.Info("Watching template files for changes...")

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			if event.Op&fsnotify.Write == fsnotify.Write ||
				event.Op&fsnotify.Create == fsnotify.Create ||
				event.Op&fsnotify.Remove == fsnotify.Remove {
				logrus.Infof("Template change detected: %s", event.Name)
				ext := filepath.Ext(event.Name)
				switch ext {
				case ".ts":

				case ".html":
					if err := loadTemplates(); err != nil {
						logrus.Errorf("Failed to reload templates: %v", err)
					} else {
						updateTemplateModTime()
						logrus.Infof("Templates reloaded successfully")
						sendTemplateUpdateNotification()
					}
				}
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			logrus.Errorf("Watcher error: %v", err)
		}
	}
}

var tmplModTimeMu sync.RWMutex
var tmplModTime time.Time
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

var wsConn *websocket.Conn

func handleWebSocket(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		logrus.Error("WebSocket upgrade error:", err)
		return
	}
	wsConn = conn
}

func updateTemplateModTime() {
	// 遍历模板目录，找最新修改时间
	dir := "templates"
	var latest time.Time
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() && info.ModTime().After(latest) {
			latest = info.ModTime()
		}
		return nil
	})

	tmplModTimeMu.Lock()
	tmplModTime = latest
	tmplModTimeMu.Unlock()
}

func getTemplateModTime() time.Time {
	tmplModTimeMu.RLock()
	defer tmplModTimeMu.RUnlock()
	return tmplModTime
}

func main() {
	if err := loadTemplates(); err != nil {
		logrus.Fatalf("failed to load templates: %v", err)
	}
	go watchTemplates()

	router := gin.Default()
	router.Static("/static", "./dist/static")
	router.GET("/", func(ctx *gin.Context) {
		tmplMu.RLock()
		err := indexTmpl.ExecuteTemplate(ctx.Writer, "base", nil)
		tmplMu.RUnlock()
		if err != nil {
			http.Error(ctx.Writer, err.Error(), http.StatusInternalServerError)
		}
	})
	router.POST("/explain", func(ctx *gin.Context) {
		fmt.Println(ctx.Request.FormValue("cmd"))
		data := PageData{
			GetArgs: "ls -l | grep 'test'",
			OriginalCommand: []OriginalCommand{
				{
					Name:       "ls",
					MatchClass: "1",
				},
				{
					Name:       "-l",
					MatchClass: "2",
				},
				{
					Name:       "|",
					MatchClass: "3",
				},
				{
					Name:       "grep",
					MatchClass: "4",
				},
				{
					Name:       `"test"`,
					MatchClass: "5",
				},
			},
			Matches: []Match{
				{
					ID:           "1",
					Name:         "ls",
					Section:      "",
					Source:       "ls",
					Match:        "ls",
					Spaces:       " ",
					CommandClass: "command",
					HelpClass:    "help1",
					Suggestions: []Suggestion{
						{Cmd: "ls -a", Text: "Show all files", Link: "ls-a"},
						{Cmd: "ls -la", Text: "Show all files in long format", Link: "ls-la"},
					},
				},
				{
					ID:           "2",
					Name:         "-l",
					Section:      "",
					Source:       "",
					Match:        "-l",
					Spaces:       " ",
					CommandClass: "command",
					HelpClass:    "help2",
					Suggestions:  []Suggestion{},
				},
				{
					ID:           "3",
					Name:         "|",
					Section:      "",
					Source:       "",
					Match:        "|",
					Spaces:       " ",
					CommandClass: "command",
					HelpClass:    "help3",
					Suggestions:  []Suggestion{},
				},
				{
					ID:           "4",
					Name:         "grep",
					Section:      "",
					Source:       "grep",
					Match:        "grep",
					Spaces:       " ",
					CommandClass: "command",
					HelpClass:    "help4",
					Suggestions: []Suggestion{
						{Cmd: "grep -i", Text: "Case-insensitive search", Link: "grep-i"},
						{Cmd: "grep -r", Text: "Recursive search", Link: "grep-r"},
					},
				},
				{
					ID:           "5",
					Name:         "'test'",
					Section:      "",
					Source:       "",
					Match:        "'test'",
					Spaces:       "",
					CommandClass: "command",
					HelpClass:    "help5",
					Suggestions:  []Suggestion{},
				},
			},
			HelpText: []HelpText{
				{
					ID:   "help1",
					Text: "List directory contents",
				},
				{
					ID:   "help2",
					Text: "Use a long listing format",
				},
				{
					ID:   "help3",
					Text: "Pipe operator - redirects output of one command to input of another",
				},
				{
					ID:   "help4",
					Text: "Search for patterns in files",
				},
				{
					ID:   "help5",
					Text: "Search pattern to match",
				},
			},
			Config: Config{
				DEBUG: true,
			},
		}

		tmplMu.RLock()
		err := tmpl.ExecuteTemplate(ctx.Writer, "base", data)
		tmplMu.RUnlock()
		if err != nil {
			http.Error(ctx.Writer, err.Error(), http.StatusInternalServerError)
		}
	})
	router.GET("/explain/:cmdName", CommandDocument)

	router.POST("/completion", func(c *gin.Context) {
		var json struct {
			Command string `json:"command"`
		}
		if err := c.ShouldBindJSON(&json); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		}
	})
	router.GET("/about", func(ctx *gin.Context) {
		tmplMu.RLock()
		err := aboutTmpl.ExecuteTemplate(ctx.Writer, "base", nil)
		tmplMu.RUnlock()
		if err != nil {
			http.Error(ctx.Writer, err.Error(), http.StatusInternalServerError)
		}
	})
	// WebSocket 连接处理
	router.GET("/ws", handleWebSocket)

	// 模板修改时间接口
	router.GET("/template-modtime", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"modtime": getTemplateModTime().Unix()})
	})
	router.Run(":8080")
}
