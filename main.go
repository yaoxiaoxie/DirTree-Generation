package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

func main() {
	// 使用 NewWithID 创建应用，并提供一个唯一的ID
	myApp := app.NewWithID("com.example.dircreator")
	// 1. 设置为浅色主题
	myApp.Settings().SetTheme(theme.LightTheme())

	myWindow := myApp.NewWindow("目录树生成工具")
	myWindow.Resize(fyne.NewSize(600, 450))

	// 用于存储选择的目标路径
	var targetPath string

	// GUI组件
	title := widget.NewLabel("=== 目录树生成工具 ===")
	title.TextStyle.Bold = true
	title.Alignment = fyne.TextAlignCenter

	pathLabel := widget.NewLabel("目标路径: 未选择")

	selectBtn := widget.NewButton("选择文件夹", func() {
		folderDialog := dialog.NewFolderOpen(func(uri fyne.ListableURI, err error) {
			if err != nil {
				dialog.ShowError(err, myWindow)
				return
			}
			if uri == nil {
				// 用户取消了选择
				return
			}
			targetPath = uri.Path()
			pathLabel.SetText("目标路径: " + targetPath)
		}, myWindow)

		// 尝试将对话框的起始位置设置为用户的主目录
		homeDir, err := os.UserHomeDir()
		if err == nil {
			uri, err := storage.ListerForURI(storage.NewFileURI(homeDir))
			if err == nil {
				folderDialog.SetLocation(uri)
			}
		}
		folderDialog.Show()
	})

	output := widget.NewMultiLineEntry()
	output.SetPlaceHolder("生成信息将显示在这里...")

	// ！！！修正点在这里！！！
	// 使用 SetMinRowsVisible() 方法，而不是直接访问字段
	output.SetMinRowsVisible(10)

	output.Wrapping = fyne.TextWrapWord
	output.Disable() // 在程序逻辑中我们是重新设置文本，所以可以禁用它防止用户输入

	createBtn := widget.NewButton("生成目录树", func() {
		if targetPath == "" {
			dialog.ShowError(fmt.Errorf("请先选择目标文件夹"), myWindow)
			return
		}

		// 完整且正确的目录结构定义
		dirStructure := map[string]interface{}{
			"app": map[string]interface{}{ // <-- 已修正
				"brower_tool": map[string]interface{}{
					"brower_app": map[string]interface{}{
						"chrome_dow":  nil,
						"chrome_fil":  nil,
						"edge_dow":    nil,
						"edge_fil":    nil,
						"firefox_dow": nil,
						"firefox_fil": nil,
					},
				},
				"cod_tool": map[string]interface{}{
					"bigModel_tool": map[string]interface{}{
						"bigModel_files": nil,
						"cherry_app":     nil,
						"cherry_fils":    nil,
						"ollama_app":     nil,
						"ollama_files":   nil,
						"lobe_app":       nil,
					},
				},
			},
			"wen": map[string]interface{}{ // <-- 已修正
				"relax_tool": map[string]interface{}{
					"game_tool": map[string]interface{}{
						"kuro": map[string]interface{}{
							"mc": map[string]interface{}{
								"子主题1212": nil,
							},
						},
						"odd": nil,
						"wy":  nil,
					},
					"odd_tool": nil,
				},
			},
		}

		output.Enable() // 先启用，再设置文本
		output.SetText("开始生成目录树...\n")
		logMessages := createDirs(targetPath, dirStructure)
		output.SetText(output.Text + strings.Join(logMessages, ""))
		output.SetText(output.Text + "\n目录树生成完成！")
		output.Disable() // 完成后再次禁用

		dialog.ShowInformation("成功", "目录树已成功生成！", myWindow)
	})

	// 使用 Border 布局来让 output 区域填满剩余空间
	topContent := container.NewVBox(
		title,
		pathLabel,
		container.NewGridWithColumns(2, selectBtn, createBtn),
		widget.NewSeparator(),
		widget.NewLabel("生成信息:"),
	)

	content := container.NewBorder(
		topContent,                  // Top
		nil,                         // Bottom
		nil,                         // Left
		nil,                         // Right
		container.NewScroll(output), // Center (will fill remaining space)
	)

	myWindow.SetContent(content)
	myWindow.ShowAndRun()
}

func createDirs(basePath string, structure map[string]interface{}) []string {
	var logs []string
	for dir, subDirs := range structure {
		fullPath := filepath.Join(basePath, dir)
		err := os.MkdirAll(fullPath, 0755)
		if err != nil {
			log.Printf("创建目录失败: %s (错误: %v)\n", fullPath, err)
			logs = append(logs, fmt.Sprintf("创建目录失败: %s (错误: %v)\n", fullPath, err))
		} else {
			log.Printf("创建目录: %s\n", fullPath)
			logs = append(logs, fmt.Sprintf("创建目录: %s\n", fullPath))
		}

		if subDirsMap, ok := subDirs.(map[string]interface{}); ok && subDirsMap != nil {
			logs = append(logs, createDirs(fullPath, subDirsMap)...)
		}
	}
	return logs
}
