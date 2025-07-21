package main

import (
	"encoding/json"
	"fmt"
	"image/color"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"gopkg.in/yaml.v3"
)

// myTheme is a custom theme that inherits from the light theme.
type myTheme struct{}

var _ fyne.Theme = (*myTheme)(nil)

func (m *myTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	if name == theme.ColorNameDisabled {
		// Use a much darker color for disabled text instead of the default light gray.
		return color.NRGBA{R: 0x40, G: 0x40, B: 0x40, A: 0xff}
	}
	return theme.LightTheme().Color(name, variant)
}
func (m *myTheme) Font(style fyne.TextStyle) fyne.Resource    { return theme.LightTheme().Font(style) }
func (m *myTheme) Icon(name fyne.ThemeIconName) fyne.Resource { return theme.LightTheme().Icon(name) }
func (m *myTheme) Size(name fyne.ThemeSizeName) float32       { return theme.LightTheme().Size(name) }

// parseStructureFromFile 解析 JSON 或 YAML 文件为 map[string]interface{}
func parseStructureFromFile(filePath string) (map[string]interface{}, error) {
	// 检查文件是否存在
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("配置文件不存在：%s\n请确认文件路径是否正确", filePath)
	}

	// 读取文件内容
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("无法读取配置文件：%s\n错误原因：%v\n请检查文件权限是否足够", filePath, err)
	}

	// 检查文件是否为空
	if len(data) == 0 {
		return nil, fmt.Errorf("配置文件为空：%s\n请确认文件包含有效的配置内容", filePath)
	}

	var structure map[string]interface{}
	// 根据文件后缀选择解析器
	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".json":
		err = json.Unmarshal(data, &structure)
		if err != nil {
			return nil, fmt.Errorf("JSON 格式解析失败：%s\n错误详情：%v\n\n请检查：\n• JSON 语法是否正确\n• 括号、引号是否匹配\n• 是否有多余的逗号", filePath, err)
		}
	case ".yaml", ".yml":
		err = yaml.Unmarshal(data, &structure)
		if err != nil {
			return nil, fmt.Errorf("YAML 格式解析失败：%s\n错误详情：%v\n\n请检查：\n• YAML 缩进是否正确（使用空格，不使用制表符）\n• 冒号后是否有空格\n• 特殊字符是否需要引号", filePath, err)
		}
	default:
		return nil, fmt.Errorf("不支持的配置文件格式：%s\n\n支持的格式：\n• .json - JSON 格式\n• .yaml - YAML 格式\n• .yml - YAML 格式\n\n请将文件保存为支持的格式后重试", ext)
	}

	// 检查解析后的结构是否为空
	if len(structure) == 0 {
		return nil, fmt.Errorf("配置文件解析后为空：%s\n请确认文件包含有效的目录结构配置", filePath)
	}

	return structure, nil
}

func createDirs(basePath string, structure map[string]interface{}, enablePrefix bool, prefix string) []string {
	var logs []string

	// 检查基础路径是否有效
	if basePath == "" {
		logs = append(logs, "错误：目标路径为空，无法创建目录\n")
		return logs
	}

	// 检查基础路径是否存在
	if _, err := os.Stat(basePath); os.IsNotExist(err) {
		logs = append(logs, fmt.Sprintf("警告：目标路径不存在，将尝试创建：%s\n", basePath))
		if err := os.MkdirAll(basePath, 0755); err != nil {
			logs = append(logs, fmt.Sprintf("错误：无法创建目标路径 %s\n原因：%v\n", basePath, err))
			return logs
		}
		logs = append(logs, fmt.Sprintf("成功：已创建目标路径 %s\n", basePath))
	}

	for dir, subDirs := range structure {
		// 验证目录名称
		if dir == "" {
			logs = append(logs, "跳过：发现空的目录名称\n")
			continue
		}

		// 应用前缀
		finalDirName := dir
		if enablePrefix && prefix != "" {
			finalDirName = prefix + dir
			logs = append(logs, fmt.Sprintf("应用前缀：\"%s\" -> \"%s\"\n", dir, finalDirName))
		}

		// 检查目录名称中的非法字符（使用最终的目录名）
		if strings.ContainsAny(finalDirName, `<>:"|?*`) {
			logs = append(logs, fmt.Sprintf("跳过：目录名包含非法字符 \"%s\"\n", finalDirName))
			continue
		}

		fullPath := filepath.Join(basePath, finalDirName)

		// 检查路径长度（Windows 限制）
		if runtime.GOOS == "windows" && len(fullPath) > 260 {
			logs = append(logs, fmt.Sprintf("跳过：路径过长（超过260字符）\"%s\"\n", fullPath))
			continue
		}

		err := os.MkdirAll(fullPath, 0755)
		if err != nil {
			// 详细的错误分析
			errorMsg := fmt.Sprintf("创建目录失败：%s\n", fullPath)
			if os.IsPermission(err) {
				errorMsg += "原因：权限不足，请检查是否有写入权限\n"
			} else if os.IsExist(err) {
				errorMsg += "原因：目录已存在（这通常不是错误）\n"
				logs = append(logs, fmt.Sprintf("目录已存在：%s\n", fullPath))
				// 如果目录已存在，继续处理子目录
				if subDirsMap, ok := subDirs.(map[string]interface{}); ok && subDirsMap != nil {
					logs = append(logs, createDirs(fullPath, subDirsMap, enablePrefix, prefix)...)
				}
				continue
			} else {
				errorMsg += fmt.Sprintf("原因：%v\n", err)
			}
			log.Printf(errorMsg)
			logs = append(logs, errorMsg)
		} else {
			log.Printf("创建目录：%s\n", fullPath)
			logs = append(logs, fmt.Sprintf("✓ 成功创建：%s\n", fullPath))
		}

		// 递归处理子目录
		if subDirsMap, ok := subDirs.(map[string]interface{}); ok && subDirsMap != nil {
			logs = append(logs, createDirs(fullPath, subDirsMap, enablePrefix, prefix)...)
		}
	}
	return logs
}

func main() {
	myApp := app.NewWithID("com.example.dircreator.v3")

	// 应用我们的自定义主题
	myApp.Settings().SetTheme(&myTheme{})

	myWindow := myApp.NewWindow("目录树生成工具")
	myWindow.Resize(fyne.NewSize(600, 500))

	// --- 状态变量 ---
	var targetPath string
	var loadedDirStructure map[string]interface{}
	var enablePrefix bool
	var prefix string

	// --- GUI组件 ---
	title := widget.NewLabel("=== 目录树生成工具 ===")
	title.TextStyle.Bold = true
	title.Alignment = fyne.TextAlignCenter

	pathLabel := widget.NewLabel("目标路径: 未选择")
	configLabel := widget.NewLabel("配置文件: 未加载")

	// 前缀功能组件
	prefixCheck := widget.NewCheck("为所有文件夹添加前缀", nil)
	prefixEntry := widget.NewEntry()
	prefixEntry.SetPlaceHolder("输入前缀（例如：C_）")
	prefixEntry.Disable() // 默认禁用

	// 前缀勾选框事件
	prefixCheck.OnChanged = func(checked bool) {
		enablePrefix = checked
		if checked {
			prefixEntry.Enable()
		} else {
			prefixEntry.Disable()
			prefixEntry.SetText("")
			prefix = ""
		}
	}

	// 前缀输入框事件
	prefixEntry.OnChanged = func(text string) {
		prefix = text
	}

	selectBtn := widget.NewButton("选择目标文件夹", func() {
		folderDialog := dialog.NewFolderOpen(func(uri fyne.ListableURI, err error) {
			if err != nil {
				// 优化文件夹选择错误提示
				errorMsg := fmt.Sprintf("选择文件夹时发生错误：\n%v\n\n请重试或选择其他文件夹", err)
				dialog.ShowError(fmt.Errorf(errorMsg), myWindow)
				return
			}
			if uri == nil {
				return
			}

			// 验证选择的路径
			selectedPath := uri.Path()
			if selectedPath == "" {
				dialog.ShowError(fmt.Errorf("选择的路径无效\n请重新选择一个有效的文件夹"), myWindow)
				return
			}

			// 检查路径权限
			testFile := filepath.Join(selectedPath, ".permission_test")
			if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
				dialog.ShowError(fmt.Errorf("选择的文件夹没有写入权限：\n%s\n\n请选择其他文件夹或检查权限设置", selectedPath), myWindow)
				return
			}
			os.Remove(testFile) // 清理测试文件

			targetPath = selectedPath
			pathLabel.SetText("目标路径: " + targetPath)
		}, myWindow)

		// 使用兼容旧版本的代码来定位根目录
		var rootPath string
		if runtime.GOOS == "windows" {
			rootPath = "C:\\"
		} else {
			rootPath = "/"
		}

		if _, err := os.Stat(rootPath); err == nil {
			uri, err := storage.ListerForURI(storage.NewFileURI(rootPath))
			if err == nil {
				folderDialog.SetLocation(uri)
			} else {
				log.Println("无法为根目录创建URI:", err)
			}
		}

		folderDialog.Show()
	})

	loadConfigBtn := widget.NewButton("加载配置文件", func() {
		fileDialog := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
			if err != nil {
				// 优化文件打开错误提示
				errorMsg := fmt.Sprintf("打开配置文件时发生错误：\n%v\n\n请检查：\n• 文件是否存在\n• 是否有读取权限\n• 文件是否被其他程序占用", err)
				dialog.ShowError(fmt.Errorf(errorMsg), myWindow)
				return
			}
			if reader == nil {
				// 用户取消选择
				return
			}

			filePath := reader.URI().Path()
			if filePath == "" {
				dialog.ShowError(fmt.Errorf("获取文件路径失败\n请重新选择配置文件"), myWindow)
				return
			}

			structure, err := parseStructureFromFile(filePath)
			if err != nil {
				// 显示详细的解析错误信息
				dialog.ShowError(err, myWindow)
				loadedDirStructure = nil
				configLabel.SetText("配置文件: 加载失败")
				return
			}

			// 解析成功
			loadedDirStructure = structure
			fileName := filepath.Base(filePath)
			configLabel.SetText("配置文件: " + fileName)

			// 显示加载成功信息，包含统计
			totalDirs := countTotalDirectories(structure)
			successMsg := fmt.Sprintf("配置文件加载成功！\n\n文件：%s\n预计创建目录数量：%d", fileName, totalDirs)
			dialog.ShowInformation("加载成功", successMsg, myWindow)

		}, myWindow)

		// 尝试获取用户桌面路径
		homeDir, err := os.UserHomeDir()
		if err == nil {
			desktopPath := filepath.Join(homeDir, "Desktop")
			if _, err := os.Stat(desktopPath); err == nil {
				uri, err := storage.ListerForURI(storage.NewFileURI(desktopPath))
				if err == nil {
					fileDialog.SetLocation(uri)
				}
			}
		}

		// 设置文件过滤器
		fileDialog.SetFilter(storage.NewExtensionFileFilter([]string{".json", ".yaml", ".yml"}))
		fileDialog.Show()
	})

	output := widget.NewMultiLineEntry()
	output.SetPlaceHolder("生成信息将显示在这里...")
	output.SetMinRowsVisible(10)
	output.Wrapping = fyne.TextWrapWord
	output.Disable()

	createBtn := widget.NewButton("生成目录树", func() {
		// 验证必要条件
		if targetPath == "" {
			dialog.ShowError(fmt.Errorf("请先选择目标文件夹\n\n步骤：\n1. 点击 \"选择目标文件夹\" 按钮\n2. 选择要创建目录树的位置\n3. 确认选择"), myWindow)
			return
		}

		if loadedDirStructure == nil {
			dialog.ShowError(fmt.Errorf("请先加载配置文件\n\n步骤：\n1. 点击 \"加载配置文件\" 按钮\n2. 选择 JSON 或 YAML 格式的配置文件\n3. 确认文件加载成功"), myWindow)
			return
		}

		// 最终确认
		prefixInfo := ""
		if enablePrefix && prefix != "" {
			prefixInfo = fmt.Sprintf("\n前缀设置：为所有文件夹添加前缀 \"%s\"", prefix)
		}

		confirmMsg := fmt.Sprintf("即将在以下位置创建目录树：\n%s\n\n预计创建 %d 个目录%s\n\n是否继续？", targetPath, countTotalDirectories(loadedDirStructure), prefixInfo)
		confirmDialog := dialog.NewConfirm("确认创建", confirmMsg, func(confirmed bool) {
			if !confirmed {
				return
			}

			output.Enable()
			output.SetText("开始生成目录树...\n\n")

			// 显示前缀设置信息
			if enablePrefix && prefix != "" {
				output.SetText(output.Text + fmt.Sprintf("前缀设置：为所有文件夹添加前缀 \"%s\"\n\n", prefix))
			}

			logMessages := createDirs(targetPath, loadedDirStructure, enablePrefix, prefix)
			allLogs := strings.Join(logMessages, "")
			output.SetText(output.Text + allLogs)

			// 统计结果
			successCount := strings.Count(allLogs, "✓ 成功创建")
			errorCount := strings.Count(allLogs, "错误：") + strings.Count(allLogs, "跳过：")

			summary := fmt.Sprintf("\n========== 生成完成 ==========\n成功创建：%d 个目录\n", successCount)
			if errorCount > 0 {
				summary += fmt.Sprintf("跳过/失败：%d 个目录\n", errorCount)
			}
			summary += "=============================\n"

			output.SetText(output.Text + summary)
			output.Disable()

			if errorCount == 0 {
				dialog.ShowInformation("生成成功", fmt.Sprintf("目录树已成功生成！\n\n共创建了 %d 个目录", successCount), myWindow)
			} else {
				dialog.ShowInformation("生成完成", fmt.Sprintf("目录树生成完成！\n\n成功：%d 个目录\n跳过/失败：%d 个目录\n\n请查看详细信息了解具体情况", successCount, errorCount), myWindow)
			}
		}, myWindow)

		confirmDialog.Show()
	})

	// 布局
	topContent := container.NewVBox(
		title,
		pathLabel,
		configLabel,
		widget.NewSeparator(),
		// 前缀功能区域
		widget.NewLabel("前缀设置:"),
		prefixCheck,
		container.NewBorder(nil, nil, widget.NewLabel("前缀:"), nil, prefixEntry),
		widget.NewSeparator(),
		container.NewGridWithColumns(2, selectBtn, loadConfigBtn),
		widget.NewSeparator(),
		createBtn,
		widget.NewSeparator(),
		widget.NewLabel("生成信息:"),
	)

	content := container.NewBorder(
		topContent,
		nil,
		nil,
		nil,
		container.NewScroll(output),
	)

	myWindow.SetContent(content)
	myWindow.ShowAndRun()
}

// 辅助函数：计算总目录数量
func countTotalDirectories(structure map[string]interface{}) int {
	count := 0
	for _, subDirs := range structure {
		count++
		if subDirsMap, ok := subDirs.(map[string]interface{}); ok && subDirsMap != nil {
			count += countTotalDirectories(subDirsMap)
		}
	}
	return count
}
