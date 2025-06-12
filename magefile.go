//go:build mage

package main

import (
	"archive/zip"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/caarlos0/log"
	"github.com/magefile/mage/mg"
)

// Runs go mod download and then installs the binary.
func Build() error {
	if err := runCmd("go", "mod", "download"); err != nil {
		return err
	}
	return runCmd("go", "install", "./...")
}

func Setup() {
	if err := runCmd("pnpm", "install"); err != nil {
		log.WithError(err).Fatal("fail to resolve node dependencies")
	}
	mg.Deps(resolveStaticFiles, resolveBootstrap, resolveFontAwesome, resolveDepsFromNodeModules)
}

func runCmd(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	log.WithField("cmd", strings.Join(append([]string{name}, args...), " ")).Info("run command")
	return cmd.Run()
}

var bootstrapFiles = map[string]string{
	"https://cdnjs.cloudflare.com/ajax/libs/twitter-bootstrap/2.3.1/js/bootstrap.js":                    "./dist/static/js/bootstrap.js",
	"https://cdnjs.cloudflare.com/ajax/libs/twitter-bootstrap/2.3.1/js/bootstrap.min.js":                "./dist/static/js/bootstrap.min.js",
	"https://cdnjs.cloudflare.com/ajax/libs/twitter-bootstrap/2.3.1/css/bootstrap.css":                  "./dist/static/css/bootstrap.css",
	"https://cdnjs.cloudflare.com/ajax/libs/twitter-bootstrap/2.3.1/css/bootstrap.min.css":              "./dist/static/css/bootstrap.min.css",
	"https://cdnjs.cloudflare.com/ajax/libs/twitter-bootstrap/2.3.1/css/bootstrap.responsive.css":       "./dist/static/css/bootstrap.responsive.css",
	"https://cdnjs.cloudflare.com/ajax/libs/twitter-bootstrap/2.3.1/css/bootstrap.responsive.min.css":   "./dist/static/css/bootstrap.responsive.min.css",
	"https://raw.githubusercontent.com/thomaspark/bootswatch/refs/tags/v2.3.2/cyborg/bootstrap.min.css": "./dist/static/css/bootstrap-cyborg.min.css",
}

func resolveBootstrap() {
	for url, dist := range bootstrapFiles {
		if err := downloadFile(url, dist); err != nil {
			log.WithError(err).Fatal("fail to fetch file")
		}
	}
}

var extractMap = map[string]string{
	"Font-Awesome-3.2.1/css/font-awesome.css":          "dist/static/css/font-awesome.css",
	"Font-Awesome-3.2.1/css/font-awesome.min.css":      "dist/static/css/font-awesome.min.css",
	"Font-Awesome-3.2.1/font/FontAwesome.otf":          "dist/static/font/FontAwesome.otf",
	"Font-Awesome-3.2.1/font/fontawesome-webfont.woff": "dist/static/font/fontawesome-webfont.woff",
	"Font-Awesome-3.2.1/font/fontawesome-webfont.ttf":  "dist/static/font/fontawesome-webfont.ttf",
	"Font-Awesome-3.2.1/font/fontawesome-webfont.eot":  "dist/static/font/fontawesome-webfont.eot",
	"Font-Awesome-3.2.1/font/fontawesome-webfont.svg":  "dist/static/font/fontawesome-webfont.svg",
}

const (
	FontAwesomeUrl = "https://github.com/FortAwesome/Font-Awesome/archive/refs/tags/v3.2.1.zip"
	FontAwesomeZip = "font-awesome-3.2.1.zip"
)

func resolveFontAwesome() {
	err := downloadFile(FontAwesomeUrl, FontAwesomeZip)
	if err != nil {
		log.WithError(err).Fatal("fail to download")
	}
	defer os.Remove(FontAwesomeZip)

	err = extractSelectedFiles(FontAwesomeZip, extractMap)
	if err != nil {
		log.WithError(err).Fatal("fail to extract files")
	}
}

var jsscripts = map[string]string{
	"node_modules/jquery/jquery.js":         "dist/static/js/jquery.js",
	"node_modules/underscore/underscore.js": "dist/static/js/underscore.js",
	"node_modules/d3/d3.js":                 "dist/static/js/d3.v3.js",
	"node_modules/d3/d3.min.js":             "dist/static/js/d3.v3.min.js",
}

func resolveDepsFromNodeModules() {
	for url, dist := range jsscripts {
		if err := copyFile(url, dist); err != nil {
			log.WithError(err).Fatal("fail to copy file")
		}
	}
}

func resolveStaticFiles() {
	err := filepath.Walk("src/static", func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		dst := strings.Replace(path, "src", "dist", 1)
		os.MkdirAll(filepath.Dir(dst), os.ModePerm)
		err = copyFile(path, dst)
		if err != nil {
			return err
		}
		if filepath.Ext(path) == ".ts" {
			defer os.Remove(dst)
			err = runCmd("pnpm", "exec", "tsc", dst)
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		log.WithError(err).Fatal("resolve static files")
	}
}

// copyFile copies the file from src to dst
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return fmt.Errorf("failed to copy file content: %w", err)
	}

	err = destFile.Sync()
	if err != nil {
		return fmt.Errorf("failed to flush to disk: %w", err)
	}

	return nil
}

func downloadFile(url, filepath string) error {
	log.WithField("url", url).Info("downloding")
	resp, err := http.Get(url)
	if err != nil {
		log.WithError(err).
			WithField("suggestion", "You can manually download the file and place it in the dist directory.").
			Error("fail to donwload")
		return err
	}
	defer resp.Body.Close()

	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

func extractSelectedFiles(zipPath string, fileMap map[string]string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer r.Close()

	for zipEntryPath, targetPath := range fileMap {
		if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}

		for _, f := range r.File {
			if strings.HasSuffix(f.Name, zipEntryPath) {
				destFile, err := os.Create(targetPath)
				if err != nil {
					return fmt.Errorf("failed to create file: %w", err)
				}
				defer destFile.Close()

				srcFile, err := f.Open()
				if err != nil {
					return fmt.Errorf("failed to open ZIP file entry: %w", err)
				}
				defer srcFile.Close()

				if _, err := io.Copy(destFile, srcFile); err != nil {
					return fmt.Errorf("failed to copy file: %w", err)
				}

				log.WithField("src", f.Name).WithField("dst", targetPath).Info("extracted file")
				break
			}
		}
	}

	return nil
}
