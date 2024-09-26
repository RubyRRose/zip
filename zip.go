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

var prefix = "MarsGoExe_"     // 文件夹和压缩包的前缀
var maxFilesPerFolder = 10    // 默认值
var deleteSourceFiles = false // 是否删除源文件

func organizeFilesAndCompress(sourceDir string, prefixStr string, maxFilesPerFolder int, deleteSource bool) error {
	prefix = prefixStr
	maxZipNum, exists, err, files := findMaxPrefixNumber(sourceDir, prefix)
	if err != nil {
		return err
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
		return nil
	}

	// 排序文件
	sort.Slice(fileEntries, func(i, j int) bool {
		return fileEntries[i].Name() < fileEntries[j].Name()
	})

	// 处理文件，每maxFilesPerFolder个放入一个文件夹并压缩
	startFolderNum := maxZipNum + 1
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
			return err
		}

		// 移动文件到新文件夹
		for _, file := range fileEntries[i:end] {
			oldPath := filepath.Join(sourceDir, file.Name())
			newPath := filepath.Join(folderPath, file.Name())
			err = os.Rename(oldPath, newPath)
			if err != nil {
				return err
			}
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
				// 注意：这里可以根据需要决定是否要返回错误或继续执行
				// 如果选择继续执行，则不返回错误
			}
			fmt.Printf("已删除文件夹 %s\n", folderPath)
		} else {
			fmt.Printf("未删除文件夹 %s\n", folderPath)
		}

		fmt.Printf("已压缩文件夹 %s\n", folderPath)
		startFolderNum++
	}

	return nil
}

func compressFolder(folderPath, zipFilePath string) error {
	zipFile, err := os.Create(zipFilePath)
	if err != nil {
		return err
	}
	defer zipFile.Close()

	w := zip.NewWriter(zipFile)
	defer w.Close()

	// 遍历文件夹中的所有文件
	files, err := os.ReadDir(folderPath)
	if err != nil {
		return err
	}

	for _, file := range files {
		filePath := filepath.Join(folderPath, file.Name())
		f, err := os.Open(filePath)
		if err != nil {
			return err
		}
		defer f.Close()

		// 创建一个ZIP文件条目，不包含文件夹路径
		zf, err := w.Create(file.Name())
		if err != nil {
			return err
		}

		// 将文件内容写入ZIP条目
		if _, err := io.Copy(zf, f); err != nil {
			return err
		}
	}

	return nil
}

// findMaxPrefixNumber 查找给定前缀在目录中的文件夹和压缩包的最大编号
func findMaxPrefixNumber(dir, prefix string) (int, bool, error, []os.DirEntry) {
	maxNum := 0
	exists := false
	files, err := os.ReadDir(dir)
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

func main() {
	ex, err := os.Executable()
	if err != nil {
		panic(err)
	}
	exePath := filepath.Dir(ex)
	sourceDirectory := exePath // 或者你可以指定其他目录
	fmt.Println("！！！======================================================================== ！！！")
	fmt.Println("！！！输入框留空回车将使用程序默认值且无法回退只能关闭窗口重新进入，请谨慎操作 ！！！")
	fmt.Println("！！！======================================================================== ！！！")
	var inputPrefix string
	for {
		fmt.Print("请输入文件夹和压缩包的前缀（直接回车将使用默认值MarsGoExe_）: ")
		reader := bufio.NewReader(os.Stdin)
		inputPrefix, err = reader.ReadString('\n')
		if err != nil {
			fmt.Println("读取输入时发生错误:", err)
			return
		}
		inputPrefix = strings.TrimSpace(inputPrefix)
		if inputPrefix == "" {
			inputPrefix = "MarsGoExe_"
			break
		}

		// 检查前缀是否已存在，并找到最大编号
		maxNum, exists, err, _ := findMaxPrefixNumber(sourceDirectory, inputPrefix)
		if err != nil {
			fmt.Println("检查前缀时发生错误:", err)
			return
		}

		if exists {
			fmt.Printf("警告：已存在以该前缀命名的文件夹或压缩包。是否继续使用当前前缀并从最大编号%d后继续？(y/n): ", maxNum+1)
			confirm, err := reader.ReadString('\n')
			if err != nil {
				fmt.Println("读取输入时发生错误:", err)
				return
			}
			confirm = strings.TrimSpace(confirm)
			if strings.ToLower(confirm) != "y" {
				fmt.Println("请重新输入前缀。")
				continue
			}
		}
		break
	}

	for {
		// 提示用户输入
		fmt.Print("请输入每个文件夹中的最大文件数（正整数，空行回车将使用默认值10）: ")

		// 使用bufio读取一行输入
		reader := bufio.NewReader(os.Stdin)
		inputmax, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("读取输入时发生错误:", err)
			return
		}

		// 去除输入末尾的换行符
		inputmax = strings.TrimSpace(inputmax)

		// 检查输入是否为空
		if inputmax == "" {
			fmt.Println("使用默认值10作为每个文件夹中的最大文件数。")
			maxFilesPerFolder = 10 // 直接设置默认值并跳出循环
			break
		}

		// 尝试将输入转换为整数
		if num, err := strconv.Atoi(inputmax); err == nil && num > 0 {
			// 如果转换成功且是正整数，询问用户是否确认
			fmt.Printf("确认使用每个文件夹中的最大文件数 %d 吗？(y/n): ", num)
			confirm, err := reader.ReadString('\n')
			if err != nil {
				fmt.Println("读取输入时发生错误:", err)
				return
			}
			confirm = strings.TrimSpace(confirm)
			if strings.ToLower(confirm) == "y" {
				// 如果用户确认，则更新maxFilesPerFolder并跳出循环
				maxFilesPerFolder = num
				break
			} else {
				// 如果用户未确认，则不更新maxFilesPerFolder，提示重新输入
				fmt.Println("未确认，请重新输入。")
			}
		} else {
			// 转换失败或不是正整数，提示重新输入
			fmt.Println("输入错误：请输入一个有效的正整数或留空使用默认值。")
		}
	}

	// 使用maxFilesPerFolder
	fmt.Printf("每个文件夹中的最大文件数设置为：%d\n", maxFilesPerFolder)

	// 询问用户是否需要删除源文件
	for {
		fmt.Print("压缩完成后是否需要删除源文件？(y/n，空行回车将默认为不删除源文件，如果删除源文件则无法恢复！！！): ")
		reader := bufio.NewReader(os.Stdin)
		inputDelete, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("读取输入时发生错误:", err)
			return
		}
		inputDelete = strings.TrimSpace(inputDelete)

		if strings.ToLower(inputDelete) == "y" {
			fmt.Printf("确认删除源文件吗？(y/n，如果删除源文件则无法恢复！！！): ")
			confirmDelete, err := reader.ReadString('\n')
			if err != nil {
				fmt.Println("读取输入时发生错误:", err)
				return
			}
			confirmDelete = strings.TrimSpace(confirmDelete)

			if strings.ToLower(confirmDelete) == "y" {
				deleteSourceFiles = true
				break
			} else {
				fmt.Println("未确认删除源文件，请重新选择。")
				continue // 回到询问是否删除源文件的循环开始
			}
		} else if strings.ToLower(inputDelete) == "n" || inputDelete == "" {
			deleteSourceFiles = false
			break
		} else {
			fmt.Println("无效输入，请重新输入(y/n)。")
		}
	}

	// 使用maxFilesPerFolder和deleteSourceFiles
	fmt.Printf("压缩完成后将%s删除源文件。\n", func() string {
		if deleteSourceFiles {
			return "会"
		}
		return "不会"
	}())

	err = organizeFilesAndCompress(sourceDirectory, inputPrefix, maxFilesPerFolder, deleteSourceFiles)
	if err != nil {
		fmt.Printf("处理文件时发生错误: %v\n", err)
		return
	}
	fmt.Println("文件组织、压缩和删除完成。")
}

//   for /d %%X in (*) do "D:\7zip\7z.exe" a "%%X.7z" "%%X\"  保存为BAT,放在需要压缩文件夹的目录,请不要使用管理员权限运行该批处理文件,否则会把文件压缩到windows/system32目录下.
