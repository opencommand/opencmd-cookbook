//go:build mage

package main

import (
	"archive/zip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
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
	mg.Deps(ResolveDeps)
	return nil
}

func ResolveDeps() error {
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
	return nil
}

func downloadFile(url, filepath string) error {
	resp, err := http.Get(url)
	if err != nil {
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
