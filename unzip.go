package main

import (
	"archive/zip"
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	reader := bufio.NewReader(os.Stdin)

	fmt.Print("Enter the root directory to start searching (请输入要开始搜索的根目录): ")
	rootDir, _ := reader.ReadString('\n')
	rootDir = strings.TrimSpace(rootDir) // Remove any trailing newline or spaces

	deleteZips := shouldDeleteAllZips(reader)

	err := filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && filepath.Ext(path) == ".zip" {
			fmt.Printf("Processing %s\n", path)
			err = unzipFile(path)
			if err != nil {
				fmt.Printf("Failed to unzip %s: %v\n", path, err)
			} else if deleteZips {
				if err := os.Remove(path); err != nil {
					fmt.Printf("Failed to delete zip file %s: %v\n", path, err)
				} else {
					fmt.Printf("Deleted zip file %s\n", path)
				}
			}
		}
		return nil
	})

	if err != nil {
		fmt.Printf("Error walking the path %q: %v\n", rootDir, err)
	}
}

func unzipFile(zipFilePath string) error {
	zipReader, err := zip.OpenReader(zipFilePath)
	if err != nil {
		return err
	}
	defer zipReader.Close()

	zipDir := filepath.Dir(zipFilePath)

	for _, file := range zipReader.File {
		filePath := filepath.Join(zipDir, file.Name)

		if file.FileInfo().IsDir() {
			os.MkdirAll(filePath, file.Mode())
			continue
		}

		// 如果文件已存在，则重命名新文件
		if _, err := os.Stat(filePath); err == nil {
			filePath = generateUniqueFilename(filePath)
		} else if os.IsNotExist(err) {
			// 文件不存在，可以继续创建
		} else {
			return err
		}

		if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
			return err
		}

		fileReader, err := file.Open()
		if err != nil {
			return err
		}
		defer fileReader.Close()

		targetFile, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.Mode())
		if err != nil {
			return err
		}
		defer targetFile.Close()

		_, err = io.Copy(targetFile, fileReader)
		if err != nil {
			return err
		}
	}

	return nil
}

func generateUniqueFilename(filename string) string {
	ext := filepath.Ext(filename)
	base := filename[:len(filename)-len(ext)]

	counter := 1
	for {
		newName := fmt.Sprintf("%s_%d%s", base, counter, ext)
		if _, err := os.Stat(newName); os.IsNotExist(err) {
			return newName
		}
		counter++
	}
}

func shouldDeleteAllZips(reader *bufio.Reader) bool {
	fmt.Printf("Do you want to delete all zip files after extraction? (y/n) (解压后是否要删除所有的压缩包？(y/n)): ")
	response, _ := reader.ReadString('\n')
	response = strings.TrimSpace(response)

	if response == "y" || response == "Y" {
		fmt.Println("Please confirm deletion by typing 'confirm' (请通过输入 'confirm' 来确认删除):")
		confirmation, _ := reader.ReadString('\n')
		confirmation = strings.TrimSpace(confirmation)

		if confirmation == "confirm" {
			fmt.Println("All zip files will be deleted after extraction. (所有压缩包将在解压后被删除。)")
			return true
		} else {
			fmt.Println("Deletion not confirmed. Keeping all zip files. (未确认删除。保留所有的压缩包。)")
			return false
		}
	}

	fmt.Println("All zip files will be kept. (所有的压缩包都将被保留。)")
	return false
}
