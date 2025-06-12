package handler

import (
	"fmt"
	"net/http"
	"opencmd-cookbook/template"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/opencommand/echochic"
	"github.com/sirupsen/logrus"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

var client *websocket.Conn

var RELOAD = []byte("reload")

func WatchWeb(path string) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		logrus.Fatalf("failed to create fsnotify watcher: %v", err)
	}
	defer watcher.Close()

	err = addWatchRecursive(watcher, "./src")
	if err != nil {
		logrus.Fatalf("failed to recursively watch templates folder: %v", err)
	}

	fmt.Println(echochic.Styled().
		Grey(time.Now().Format("15:04:05")).
		Space().
		With(echochic.Bold, echochic.Blue).
		Text("[FS]").
		Space().
		Green("INFO").
		Space().
		Grey("Watching template files for changes...").
		String())

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
					// require clear cache
					continue
					dst := strings.Replace(event.Name, path, "dist", 1)
					cmd := exec.Command("pnpm", "exec", "tsc", event.Name, dst)
					if err = cmd.Run(); err != nil {
						logrus.Error("Fail to exec command")
					}
				case ".html":
					affectedTemplates, err := template.Reg.GetTemplatesAffectedBy(event.Name)
					if err != nil {
						logrus.Errorf("Failed to get affected templates for %s: %v", event.Name, err)
						continue
					}
					fmt.Println(affectedTemplates)
					for _, name := range affectedTemplates {
						if err := template.Reg.Reload(name); err != nil {
							logrus.Errorf("Failed to reload template %s: %v", name, err)
						} else {
							logrus.Infof("Template %s reloaded", name)
						}
					}
					if client != nil {
						err = client.WriteMessage(websocket.TextMessage, RELOAD)
						if err != nil {
							logrus.Error("Failed to send WebSocket message:", err)
						}
					}
				default:

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

func Hotfix(ctx *gin.Context) {
	var err error
	client, err = upgrader.Upgrade(ctx.Writer, ctx.Request, nil)
	if err != nil {
		logrus.Error("WebSocket upgrade error:", err)
		return
	}
}

func addWatchRecursive(watcher *fsnotify.Watcher, root string) error {
	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			if err := watcher.Add(path); err != nil {
				logrus.Errorf("Failed to watch directory: %s, err: %v", path, err)
			} else {
				// logrus.Infof("Watching: %s", path)
			}
		}
		return nil
	})
}
