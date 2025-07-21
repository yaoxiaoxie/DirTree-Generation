package main

import (
	"encoding/json"
	"fmt"
	"image/color" // <-- 已添加
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
	"gopkg.in/yaml.v3" // <-- 1. 引入 YAML 库
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
	// 读取文件内容
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("读取文件失败: %w", err)
	}

	var structure map[string]interface{}
	// 根据文件后缀选择解析器
	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".json":
		err = json.Unmarshal(data, &structure)
		if err != nil {
			return nil, fmt.Errorf("解析 JSON 失败: %w", err)
		}
	case ".yaml", ".yml":
		err = yaml.Unmarshal(data, &structure)
		if err != nil {
			return nil, fmt.Errorf("解析 YAML 失败: %w", err)
		}
	default:
		return nil, fmt.Errorf("不支持的配置文件格式: %s (仅支持 .json, .yaml, .yml)", ext)
	}

	return structure, nil
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

func main() {
	myApp := app.NewWithID("com.example.dircreator.v3")

	// 应用我们的自定义主题
	myApp.Settings().SetTheme(&myTheme{})

	myWindow := myApp.NewWindow("目录树生成工具")
	myWindow.Resize(fyne.NewSize(600, 500)) // 稍微调高一点窗口以容纳新组件

	// --- 状态变量 ---
	var targetPath string
	var loadedDirStructure map[string]interface{} // <-- 2. 用于存储从文件加载的结构

	// --- GUI组件 ---
	title := widget.NewLabel("=== 目录树生成工具 ===")
	title.TextStyle.Bold = true
	title.Alignment = fyne.TextAlignCenter

	pathLabel := widget.NewLabel("目标路径: 未选择")
	configLabel := widget.NewLabel("配置文件: 未加载") // <-- 3. 新增标签显示配置文件路径

	// --- ↓↓↓ 这里是修改的部分 ↓↓↓ ---
	selectBtn := widget.NewButton("选择目标文件夹", func() {
		folderDialog := dialog.NewFolderOpen(func(uri fyne.ListableURI, err error) {
			if err != nil {
				dialog.ShowError(err, myWindow)
				return
			}
			if uri == nil {
				return
			}
			targetPath = uri.Path()
			pathLabel.SetText("目标路径: " + targetPath)
		}, myWindow)

		// 使用兼容旧版本的代码来定位根目录
		var rootPath string
		if runtime.GOOS == "windows" {
			// 在 Windows 上，假定 C:\ 是主根目录
			rootPath = "C:\\"
		} else {
			// 在 Linux 或 macOS 上，根目录是 /
			rootPath = "/"
		}

		// 检查路径是否存在，并尝试设置为默认位置
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
	// --- ↑↑↑ 修改结束 ↑↑↑ ---

	// <-- 4. 新增加载配置文件的按钮 -->
	loadConfigBtn := widget.NewButton("加载配置文件", func() {
		fileDialog := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
			if err != nil {
				dialog.ShowError(err, myWindow)
				return
			}
			if reader == nil {
				// 用户取消
				return
			}

			filePath := reader.URI().Path()
			structure, err := parseStructureFromFile(filePath)
			if err != nil {
				dialog.ShowError(err, myWindow)
				loadedDirStructure = nil // 解析失败则清空
				configLabel.SetText("配置文件: 加载失败")
				return
			}

			// 解析成功
			loadedDirStructure = structure
			configLabel.SetText("配置文件: " + filepath.Base(filePath)) // 只显示文件名，更简洁
			dialog.ShowInformation("成功", "配置文件已成功加载！", myWindow)

		}, myWindow)

		// 尝试获取用户桌面路径
		homeDir, err := os.UserHomeDir()
		if err == nil {
			desktopPath := filepath.Join(homeDir, "Desktop")
			uri, err := storage.ListerForURI(storage.NewFileURI(desktopPath))
			if err == nil {
				// 如果成功，将位置设置为桌面
				fileDialog.SetLocation(uri)
			} else {
				log.Println("无法定位到桌面目录:", err)
			}
		} else {
			log.Println("无法获取用户主目录:", err)
		}
		// --- ↑↑↑ 新增的核心逻辑 ↑↑↑ ---

		// 设置文件过滤器，只显示 JSON 和 YAML 文件
		fileDialog.SetFilter(storage.NewExtensionFileFilter([]string{".json", ".yaml", ".yml"}))
		fileDialog.Show()
	})

	output := widget.NewMultiLineEntry()
	output.SetPlaceHolder("生成信息将显示在这里...")
	output.SetMinRowsVisible(10)
	output.Wrapping = fyne.TextWrapWord
	output.Disable()

	createBtn := widget.NewButton("生成目录树", func() {
		// --- 5. 更新生成逻辑 ---
		if targetPath == "" {
			dialog.ShowError(fmt.Errorf("请先选择目标文件夹"), myWindow)
			return
		}
		// 检查配置是否已加载
		if loadedDirStructure == nil {
			dialog.ShowError(fmt.Errorf("请先加载一个有效的配置文件"), myWindow)
			return
		}

		output.Enable()
		output.SetText("开始生成目录树...\n")
		// 使用加载的结构，而不是硬编码的
		logMessages := createDirs(targetPath, loadedDirStructure)
		output.SetText(output.Text + strings.Join(logMessages, ""))
		output.SetText(output.Text + "\n目录树生成完成！")
		output.Disable()

		dialog.ShowInformation("成功", "目录树已成功生成！", myWindow)
	})

	// --- 6. 更新布局以包含新组件 ---
	topContent := container.NewVBox(
		title,
		pathLabel,
		configLabel, // 添加新标签
		container.NewGridWithColumns(2, selectBtn, loadConfigBtn), // 放入网格布局
		widget.NewSeparator(),
		createBtn, // 将生成按钮单独放在一行，更清晰
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
