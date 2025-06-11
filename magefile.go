//go:build mage

package main

import (
	"archive/zip"
	"fmt"
	"github.com/caarlos0/log"
	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const FontAwesomeUrl = "https://github.com/FortAwesome/Font-Awesome/archive/refs/tags/v3.2.1.zip"

// Runs go mod download and then installs the binary.
func Build() error {
	if err := sh.Run("go", "mod", "download"); err != nil {
		return err
	}
	return sh.Run("go", "install", "./...")
}

func Setup() error {
	mg.Deps(resolveStaticFiles, resolveBootstrap, resolveDeps)
	return nil
}

func resolveNPMDeps() error {
	if err := sh.Run("pnpm", "i"); err != nil {
		return err
	}
	return nil
}

func resolveBootstrap() error {
	downloadFile("https://cdnjs.cloudflare.com/ajax/libs/twitter-bootstrap/2.3.1/js/bootstrap.js", "./dist/static/js/bootstrap.js")
	downloadFile("https://cdnjs.cloudflare.com/ajax/libs/twitter-bootstrap/2.3.1/js/bootstrap.min.js", "./dist/static/js/bootstrap.min.js")
	downloadFile("https://cdnjs.cloudflare.com/ajax/libs/twitter-bootstrap/2.3.1/css/bootstrap.css", "./dist/static/css/bootstrap.css")
	downloadFile("https://cdnjs.cloudflare.com/ajax/libs/twitter-bootstrap/2.3.1/css/bootstrap.min.css", "./dist/static/css/bootstrap.min.css")
	downloadFile("https://cdnjs.cloudflare.com/ajax/libs/twitter-bootstrap/2.3.1/css/bootstrap.responsive.css", "./dist/static/css/bootstrap.responsive.css")
	downloadFile("https://cdnjs.cloudflare.com/ajax/libs/twitter-bootstrap/2.3.1/css/bootstrap.responsive.min.css", "./dist/static/css/bootstrap.responsive.min.css")
	downloadFile("https://raw.githubusercontent.com/thomaspark/bootswatch/refs/tags/v2.3.2/cyborg/bootstrap.min.css", "./dist/static/css/bootstrap-cyborg.min.css")
	return nil
}

func resolveDeps() error {
	mg.Deps(resolveNPMDeps)
	zipPath := "font-awesome-3.2.1.zip"
	err := downloadFile(FontAwesomeUrl, zipPath)
	if err != nil {
		return fmt.Errorf("下载失败: %w", err)
	}
	defer os.Remove(zipPath)

	// 定义需要提取的文件映射：ZIP中的路径 -> 目标路径
	extractMap := map[string]string{
		"Font-Awesome-3.2.1/css/font-awesome.css":          "dist/static/css/font-awesome.css",
		"Font-Awesome-3.2.1/css/font-awesome.min.css":      "dist/static/css/font-awesome.min.css",
		"Font-Awesome-3.2.1/font/FontAwesome.otf":          "dist/static/font/FontAwesome.otf",
		"Font-Awesome-3.2.1/font/fontawesome-webfont.woff": "dist/static/font/fontawesome-webfont.woff",
		"Font-Awesome-3.2.1/font/fontawesome-webfont.ttf":  "dist/static/font/fontawesome-webfont.ttf",
		"Font-Awesome-3.2.1/font/fontawesome-webfont.eot":  "dist/static/font/fontawesome-webfont.eot",
		"Font-Awesome-3.2.1/font/fontawesome-webfont.svg":  "dist/static/font/fontawesome-webfont.svg",
	}

	err = extractSelectedFiles(zipPath, extractMap)
	if err != nil {
		return fmt.Errorf("解压失败: %w", err)
	}

	fmt.Println("成功下载并提取Font Awesome 3.2.1!")

	copyFile("node_modules/jquery/jquery.js", "dist/static/js/jquery.js")
	copyFile("node_modules/underscore/underscore.js", "dist/static/js/underscore.js")
	copyFile("node_modules/d3/d3.js", "dist/static/js/d3.v3.js")
	copyFile("node_modules/d3/d3.min.js", "dist/static/js/d3.v3.min.js")
	return nil
}

func resolveStaticFiles() error {
	return filepath.Walk("src/static", func(path string, info fs.FileInfo, err error) error {
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
			err = sh.Run("pnpm", "exec", "tsc", dst)
			if err != nil {
				return err
			}
		}
		return nil
	})
}

// copyFile copies the file from src to dst
func copyFile(src, dst string) error {
	// 打开源文件
	sourceFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer sourceFile.Close()

	// 创建目标文件（若文件已存在将被覆盖）
	destFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer destFile.Close()

	// 复制内容
	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return fmt.Errorf("failed to copy file content: %w", err)
	}

	// 保证写入磁盘
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
		log.WithError(err).WithField("suggestion", "You can manually download the file and place it in the dist directory.").Error("fail to donwload")
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

	for zipPath, targetPath := range fileMap {
		// 确保目标目录存在
		if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
			return fmt.Errorf("创建目录失败: %w", err)
		}

		// 在ZIP文件中查找匹配的文件
		for _, f := range r.File {
			if strings.HasSuffix(f.Name, zipPath) {
				// 创建目标文件
				destFile, err := os.Create(targetPath)
				if err != nil {
					return fmt.Errorf("创建文件失败: %w", err)
				}
				defer destFile.Close()

				// 打开ZIP中的文件
				srcFile, err := f.Open()
				if err != nil {
					return fmt.Errorf("打开ZIP文件失败: %w", err)
				}
				defer srcFile.Close()

				// 复制内容
				if _, err := io.Copy(destFile, srcFile); err != nil {
					return fmt.Errorf("复制文件失败: %w", err)
				}

				fmt.Printf("已提取: %s -> %s\n", f.Name, targetPath)
				break
			}
		}
	}

	return nil
}
