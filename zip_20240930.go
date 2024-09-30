package main

import (
	"archive/zip"
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

var prefix = "MarsGoExe_"      // 文件夹和压缩包的前缀
var maxFilesPerFolder = 10     // 默认值
var deleteSourceFiles = false  // 是否删除源文件
var sourceDirectory string     // 源目录
var deleteEmptyFolders = false // 是否删除已提取的空文件夹

// compressFolder 压缩指定文件夹为 zip 文件
func compressFolder(folderPath, zipFilePath string) error {
	zipFile, err := os.Create(zipFilePath)
	if err != nil {
		return err
	}
	defer zipFile.Close()
	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	err = filepath.Walk(folderPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}
		// 修改这里的路径处理，使其直接包含文件，不包含多层目录
		header.Name = filepath.Base(path)
		if info.IsDir() {
			header.Name += "/"
		} else {
			header.Method = zip.Deflate
		}
		writer, err := zipWriter.CreateHeader(header)
		if err != nil {
			return err
		}
		if !info.IsDir() {
			file, err := os.Open(path)
			if err != nil {
				return err
			}
			defer file.Close()
			_, err = io.Copy(writer, file)
			if err != nil {
				return err
			}
		}
		return nil
	})

	return err
}

// findMaxPrefixNumber 查找具有给定前缀的最大编号
func findMaxPrefixNumber(sourceDir, prefix string) (int, bool, error, []os.DirEntry) {
	var maxNum int
	exists := false
	files, err := os.ReadDir(sourceDir)
	if err != nil {
		return 0, false, err, nil
	}
	for _, file := range files {
		name := file.Name()
		if file.IsDir() && strings.HasPrefix(name, prefix) {
			numStr := strings.TrimPrefix(name, prefix)
			if num, err := strconv.Atoi(numStr); err == nil {
				if num > maxNum {
					maxNum = num
					exists = true
				}
			}
		} else if !file.IsDir() && strings.HasSuffix(name, ".zip") && strings.HasPrefix(strings.TrimSuffix(name, filepath.Ext(name)), prefix) {
			numStr := strings.TrimPrefix(strings.TrimSuffix(name, filepath.Ext(name)), prefix)
			if num, err := strconv.Atoi(numStr); err == nil {
				if num > maxNum {
					maxNum = num
					exists = true
				}
			}
		}
	}
	return maxNum, exists, nil, files
}

// organizeFilesAndCompress 组织文件并压缩
func organizeFilesAndCompress(sourceDir, prefixStr string, maxFilesPerFolder int, deleteSource bool) (int, error) {
	prefix = prefixStr
	maxZipNum, exists, err, files := findMaxPrefixNumber(sourceDir, prefix)
	if err != nil {
		return 0, err
	}
	if !exists {
		maxZipNum = 0 // 如果没有找到任何以该前缀命名的文件或文件夹，则最大编号为0
	}

	// 过滤出文件（忽略文件夹、.exe文件和.zip文件）
	var fileEntries []os.DirEntry
	for _, entry := range files {
		if !entry.IsDir() && !strings.HasSuffix(entry.Name(), ".exe") && !strings.HasSuffix(entry.Name(), ".zip") {
			fileEntries = append(fileEntries, entry)
		}
	}

	// 如果没有文件，则直接返回
	if len(fileEntries) == 0 {
		fmt.Println("源目录下没有文件。")
		return 0, nil
	}

	// 排序文件
	sort.Slice(fileEntries, func(i, j int) bool {
		return fileEntries[i].Name() < fileEntries[j].Name()
	})

	// 处理文件，每maxFilesPerFolder个放入一个文件夹并压缩
	startFolderNum := maxZipNum + 1
	finalFolderNum := startFolderNum
	for i := 0; i < len(fileEntries); i += maxFilesPerFolder {
		end := i + maxFilesPerFolder
		if end > len(fileEntries) {
			end = len(fileEntries)
		}
		folderName := fmt.Sprintf("%s%d", prefix, startFolderNum)
		folderPath := filepath.Join(sourceDir, folderName)

		// 创建文件夹
		err = os.MkdirAll(folderPath, 0777)
		if err != nil && !os.IsExist(err) {
			return 0, err
		}

		// 移动文件到新文件夹
		for _, file := range fileEntries[i:end] {
			oldPath := filepath.Join(sourceDir, file.Name())
			newPath := filepath.Join(folderPath, file.Name())
			err = os.Rename(oldPath, newPath)
			if err != nil {
				return 0, err
			}
			fmt.Printf("移动文件: %s -> %s\n", oldPath, newPath)
		}

		// 压缩文件夹
		zipFilePath := filepath.Join(sourceDir, fmt.Sprintf("%s.zip", folderName))
		err = compressFolder(folderPath, zipFilePath)
		if err != nil {
			fmt.Printf("压缩文件夹 %s 失败: %v\n", folderPath, err)
			continue // 跳过删除文件夹，因为压缩失败
		}

		// 删除文件夹
		// 根据用户选择是否删除源文件
		if deleteSource {
			err = os.RemoveAll(folderPath)
			if err != nil {
				fmt.Printf("删除文件夹 %s 失败: %v\n", folderPath, err)
			} else {
				fmt.Printf("已删除文件夹 %s\n", folderPath)
			}
		}

		startFolderNum++
		finalFolderNum = startFolderNum - 1
	}

	return finalFolderNum, nil
}

// organizeFilesOnly 组织文件但不压缩
func organizeFilesOnly(sourceDir, prefixStr string, maxFilesPerFolder int) (int, error) {
	prefix = prefixStr
	maxZipNum, exists, err, files := findMaxPrefixNumber(sourceDir, prefix)
	if err != nil {
		return 0, err
	}
	if !exists {
		maxZipNum = 0 // 如果没有找到任何以该前缀命名的文件或文件夹，则最大编号为0
	}

	// 过滤出文件（忽略文件夹、.exe文件和.zip文件）
	var fileEntries []os.DirEntry
	for _, entry := range files {
		if !entry.IsDir() && !strings.HasSuffix(entry.Name(), ".exe") && !strings.HasSuffix(entry.Name(), ".zip") {
			fileEntries = append(fileEntries, entry)
		}
	}

	// 如果没有文件，则直接返回
	if len(fileEntries) == 0 {
		fmt.Println("源目录下没有文件。")
		return 0, nil
	}

	// 排序文件
	sort.Slice(fileEntries, func(i, j int) bool {
		return fileEntries[i].Name() < fileEntries[j].Name()
	})

	// 处理文件，每maxFilesPerFolder个放入一个文件夹
	startFolderNum := maxZipNum + 1
	finalFolderNum := startFolderNum
	for i := 0; i < len(fileEntries); i += maxFilesPerFolder {
		end := i + maxFilesPerFolder
		if end > len(fileEntries) {
			end = len(fileEntries)
		}
		folderName := fmt.Sprintf("%s%d", prefix, startFolderNum)
		folderPath := filepath.Join(sourceDir, folderName)

		// 创建文件夹
		err = os.MkdirAll(folderPath, 0777)
		if err != nil && !os.IsExist(err) {
			return 0, err
		}

		// 移动文件到新文件夹
		for _, file := range fileEntries[i:end] {
			oldPath := filepath.Join(sourceDir, file.Name())
			newPath := filepath.Join(folderPath, file.Name())
			err = os.Rename(oldPath, newPath)
			if err != nil {
				return 0, err
			}
			fmt.Printf("移动文件: %s -> %s\n", oldPath, newPath)
		}

		startFolderNum++
		finalFolderNum = startFolderNum - 1
	}

	return finalFolderNum, nil
}

// getUserInput 获取用户输入
func getUserInput(reader *bufio.Reader, prompt, defaultValue string) (string, error) {
	fmt.Print(prompt)
	input, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	input = strings.TrimSpace(input)
	if input == "" {
		input = defaultValue
	}
	return input, nil
}

// extractFromFolder 从文件夹中提取文件
func extractFromFolder(folderPath, destinationPath string) error {
	files, err := os.ReadDir(folderPath)
	if err != nil {
		return err
	}
	for _, file := range files {
		oldPath := filepath.Join(folderPath, file.Name())
		newPath := filepath.Join(destinationPath, file.Name())
		if file.IsDir() {
			err = os.MkdirAll(newPath, 0777)
			if err != nil {
				return err
			}
			err = extractFromFolder(oldPath, newPath)
			if err != nil {
				return err
			}
		} else {
			err = os.Rename(oldPath, newPath)
			if err != nil {
				return err
			}
			fmt.Printf("移动文件: %s -> %s\n", oldPath, newPath)
		}
	}
	return nil
}

// removeEmptyFolders 删除空文件夹
func removeEmptyFolders(folderPath, prefix string) error {
	files, err := os.ReadDir(folderPath)
	if err != nil {
		return err
	}
	if len(files) == 0 {
		return os.Remove(folderPath)
	}
	for _, file := range files {
		if file.IsDir() {
			subFolderPath := filepath.Join(folderPath, file.Name())
			err := removeEmptyFolders(subFolderPath, prefix)
			if err != nil {
				return err
			}
		}
	}
	files, _ = os.ReadDir(folderPath)
	if len(files) == 0 {
		return os.Remove(folderPath)
	}
	return nil
}

// extractFromZip 从压缩包中提取文件
func extractFromZip(zipFilePath, destinationPath string) error {
	reader, err := zip.OpenReader(zipFilePath)
	if err != nil {
		return err
	}
	defer reader.Close()

	for _, file := range reader.File {
		filePath := filepath.Join(destinationPath, file.Name)
		if file.FileInfo().IsDir() {
			os.MkdirAll(filePath, os.ModePerm)
			continue
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

// findNextAvailableFolder 查找下一个可用的同名文件夹
func findNextAvailableFolder(baseFolder string) (string, error) {
	baseName := filepath.Base(baseFolder)
	dir := filepath.Dir(baseFolder)
	num := 1
	for {
		folderPath := filepath.Join(dir, fmt.Sprintf("%s_%d", baseName, num))
		_, err := os.Stat(folderPath)
		if os.IsNotExist(err) {
			return folderPath, nil
		}
		num++
	}
}

// getDestinationFolder 确定解压目标文件夹
func getDestinationFolder(sourceDir, baseFolder string) (string, error) {
	destinationPath := filepath.Join(sourceDir, baseFolder)
	// 检查是否存在同名文件夹
	_, err := os.Stat(destinationPath)
	if os.IsNotExist(err) {
		// 文件夹不存在，创建它
		err = os.MkdirAll(destinationPath, 0777)
		if err != nil {
			return "", err
		}
		return destinationPath, nil
	}

	// 存在同名文件夹，检查是否为空
	entries, _ := os.ReadDir(destinationPath)
	if len(entries) == 0 {
		// 文件夹为空，直接使用
		return destinationPath, nil
	}

	// 文件夹不为空，查找下一个可用的同名文件夹
	maxNum, _, _, _ := findMaxPrefixNumber(sourceDir, baseFolder+"_")
	nextNum := maxNum + 1
	destinationPath = filepath.Join(sourceDir, fmt.Sprintf("%s_%d", baseFolder, nextNum))
	err = os.MkdirAll(destinationPath, 0777)
	if err != nil {
		return "", err
	}
	return destinationPath, nil
}

// extractFromZips 从压缩包中提取文件
func extractFromZips(sourceDir, prefix string, onlyWithPrefix bool) error {
	files, err := os.ReadDir(sourceDir)
	if err != nil {
		return err
	}

	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".zip") && (!onlyWithPrefix || strings.HasPrefix(strings.TrimSuffix(file.Name(), filepath.Ext(file.Name())), prefix)) {
			zipFilePath := filepath.Join(sourceDir, file.Name())
			fmt.Printf("正在处理压缩包: %s\n", zipFilePath)

			// 确定解压目标路径
			baseFolder := strings.TrimSuffix(file.Name(), filepath.Ext(file.Name()))
			destinationPath, err := getDestinationFolder(sourceDir, baseFolder)
			if err != nil {
				return err
			}

			// 解压文件
			err = extractFromZip(zipFilePath, destinationPath)
			if err != nil {
				fmt.Printf("从压缩包 %s 提取文件时发生错误: %v\n", zipFilePath, err)
			} else {
				fmt.Printf("从压缩包 %s 提取文件完成。\n", zipFilePath)
			}
		}
	}
	return nil
}

func main() {
	reader := bufio.NewReader(os.Stdin)
	ex, err := os.Executable()
	if err != nil {
		panic(err)
	}
	sourceDirectory = filepath.Dir(ex) // 或者你可以指定其他目录

	// 程序开始提示
	fmt.Println("！！！======================================================================== ！！！")
	fmt.Println("！！！输入框留空回车将使用程序默认值且无法回退只能关闭窗口重新进入，请谨慎操作 ！！！")
	fmt.Println("！！！======================================================================== ！！！")

	// 输入文件前缀
	inputPrefix, _ := getUserInput(reader, "请输入文件夹和压缩包的前缀（直接回车将使用默认值MarsGoExe_）: ", "MarsGoExe_")
	prefix = inputPrefix

	// 提示用户选择操作
	action, _ := getUserInput(reader, "请选择操作：\n1. 压缩文件\n2. 仅组织文件\n3. 从文件夹或压缩包中提取文件\n请输入数字(1, 2 或 3): ", "")
	switch action {
	case "1":
		// 压缩文件
		maxFilesPerFolderStr, _ := getUserInput(reader, "请输入每个文件夹中的最大文件数（正整数，空行回车将使用默认值10）: ", "10")
		maxFilesPerFolder, _ := strconv.Atoi(maxFilesPerFolderStr)
		deleteConfirm, _ := getUserInput(reader, "压缩完成后是否需要删除源文件？(y/n, 直接回车将使用默认值n): ", "n")
		deleteSourceFiles := strings.ToLower(deleteConfirm) == "y"

		// 组织文件并压缩
		finalFolderNum, err := organizeFilesAndCompress(sourceDirectory, prefix, maxFilesPerFolder, deleteSourceFiles)
		if err != nil {
			fmt.Printf("处理文件时发生错误: %v\n", err)
			return
		}
		fmt.Printf("文件组织、压缩完成。最后的文件夹编号是 %d。\n", finalFolderNum)
	case "2":
		// 仅组织文件
		maxFilesPerFolderStr, _ := getUserInput(reader, "请输入每个文件夹中的最大文件数（正整数，空行回车将使用默认值10）: ", "10")
		maxFilesPerFolder, _ := strconv.Atoi(maxFilesPerFolderStr)

		// 仅组织文件
		finalFolderNum, err := organizeFilesOnly(sourceDirectory, prefix, maxFilesPerFolder)
		if err != nil {
			fmt.Printf("处理文件时发生错误: %v\n", err)
			return
		}
		fmt.Printf("文件组织完成。最后的文件夹编号是 %d。\n", finalFolderNum)
	case "3":
		// 从文件夹或压缩包中提取文件
		extractFromOption, _ := getUserInput(reader, "请选择提取方式：\n1. 从文件夹中提取文件\n2. 从压缩包中提取文件\n请输入数字(1 或 2): ", "")
		switch extractFromOption {
		case "1":
			// 从文件夹中提取文件
			deleteEmptyFoldersConfirm, _ := getUserInput(reader, "是否删除已提取的空文件夹？(y/n, 直接回车将使用默认值n): ", "n")
			deleteEmptyFolders = strings.ToLower(deleteEmptyFoldersConfirm) == "y"
			onlyWithPrefixConfirm, _ := getUserInput(reader, "是否只从指定前缀的文件夹中提取文件？(y/n, 直接回车将使用默认值n): ", "n")
			onlyWithPrefix := strings.ToLower(onlyWithPrefixConfirm) == "y"

			// 从文件夹中提取文件
			files, _ := os.ReadDir(sourceDirectory)
			for _, file := range files {
				if file.IsDir() && (!onlyWithPrefix || strings.HasPrefix(file.Name(), prefix)) {
					folderPath := filepath.Join(sourceDirectory, file.Name())
					fmt.Printf("正在处理文件夹: %s\n", folderPath)
					err := extractFromFolder(folderPath, sourceDirectory)
					if err != nil {
						fmt.Printf("从文件夹 %s 提取文件时发生错误: %v\n", folderPath, err)
					} else {
						fmt.Printf("从文件夹 %s 提取文件完成。\n", folderPath)
					}

					// 删除空文件夹
					if deleteEmptyFolders && strings.HasPrefix(file.Name(), prefix) {
						err := removeEmptyFolders(folderPath, prefix)
						if err != nil {
							fmt.Printf("删除空文件夹 %s 时发生错误: %v\n", folderPath, err)
						} else {
							fmt.Printf("已删除空文件夹 %s\n", folderPath)
						}
					}
				}
			}
		case "2":
			// 从压缩包中提取文件
			onlyWithPrefixConfirm, _ := getUserInput(reader, "是否只从指定前缀的压缩包中提取文件？(y/n, 直接回车将使用默认值y): ", "y")
			onlyWithPrefix := strings.ToLower(onlyWithPrefixConfirm) == "y"

			// 从压缩包中提取文件
			err := extractFromZips(sourceDirectory, prefix, onlyWithPrefix)
			if err != nil {
				fmt.Printf("从压缩包中提取文件时发生错误: %v\n", err)
			} else {
				fmt.Println("从压缩包中提取文件完成。")
			}
		default:
			fmt.Println("无效的选择，请重新运行程序并选择有效的选项。")
		}
	default:
		fmt.Println("无效的选择，请重新运行程序并选择有效的选项。")
	}
}
