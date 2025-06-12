package main

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"image/color"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"time"

	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
)

// MemoryRecord 表示单条内存记录
type MemoryRecord struct {
	Timestamp time.Time
	MemTotal  float64
	MemFree   float64
	SwapTotal float64
	SwapFree  float64
}

// 编译正则表达式
var (
	timestampRegex = regexp.MustCompile(`ATOP - \w+\s+(\d{4}/\d{2}/\d{2}\s+\d{2}:\d{2}:\d{2})`)
	memRegex       = regexp.MustCompile(`MEM \| tot\s+([\d.]+)(G|M) \| free\s+([\d.]+)(G|M)`)
	swpRegex       = regexp.MustCompile(`SWP \| tot\s+([\d.]+)(G|M) \| free\s+([\d.]+)(G|M)`)
)

// parseAtopLog 解析单个atop日志文件
func parseAtopLog(filePath string) ([]MemoryRecord, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var data []MemoryRecord
	var currentTimestamp time.Time
	var memTot, memFree float64
	var memTotUnit, memFreeUnit string
	var hasMemData bool

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()

		// 匹配时间戳行
		if matches := timestampRegex.FindStringSubmatch(line); matches != nil {
			timestamp, err := time.Parse("2006/01/02 15:04:05", matches[1])
			if err != nil {
				continue
			}
			currentTimestamp = timestamp
			hasMemData = false
			continue
		}

		// 匹配MEM行
		if matches := memRegex.FindStringSubmatch(line); matches != nil && !currentTimestamp.IsZero() {
			memTot, _ = strconv.ParseFloat(matches[1], 64)
			memTotUnit = matches[2]
			if memTotUnit == "M" {
				memTot /= 1024
			}

			memFree, _ = strconv.ParseFloat(matches[3], 64)
			memFreeUnit = matches[4]
			if memFreeUnit == "M" {
				memFree /= 1024
			}
			hasMemData = true
			continue
		}

		// 匹配SWP行
		if matches := swpRegex.FindStringSubmatch(line); matches != nil && !currentTimestamp.IsZero() && hasMemData {
			swpTot, _ := strconv.ParseFloat(matches[1], 64)
			swpTotUnit := matches[2]
			if swpTotUnit == "M" {
				swpTot /= 1024
			}

			swpFree, _ := strconv.ParseFloat(matches[3], 64)
			swpFreeUnit := matches[4]
			if swpFreeUnit == "M" {
				swpFree /= 1024
			}

			// 添加到数据列表
			data = append(data, MemoryRecord{
				Timestamp: currentTimestamp,
				MemTotal:  memTot,
				MemFree:   memFree,
				SwapTotal: swpTot,
				SwapFree:  swpFree,
			})

			hasMemData = false
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return data, nil
}

// parseAtopDirectory 解析目录中的所有atop日志文件
func parseAtopDirectory(dirPath string) ([]MemoryRecord, error) {
	// 检查目录是否存在
	fileInfo, err := os.Stat(dirPath)
	if err != nil {
		return nil, fmt.Errorf("目录 %s 不存在: %v", dirPath, err)
	}
	if !fileInfo.IsDir() {
		return nil, fmt.Errorf("%s 不是一个目录", dirPath)
	}

	// 获取目录中的所有文件
	files, err := ioutil.ReadDir(dirPath)
	if err != nil {
		return nil, err
	}

	if len(files) == 0 {
		fmt.Printf("警告: 目录 %s 中没有找到文件\n", dirPath)
		return nil, nil
	}

	var allData []MemoryRecord
	var successfulFiles int

	// 解析每个文件
	for _, file := range files {
		if file.IsDir() {
			continue
		}

		filePath := filepath.Join(dirPath, file.Name())
		fileData, err := parseAtopLog(filePath)
		if err != nil {
			fmt.Printf("解析文件 %s 时出错: %v\n", file.Name(), err)
			continue
		}

		if len(fileData) > 0 {
			fmt.Printf("成功解析文件: %s, 找到 %d 条记录\n", file.Name(), len(fileData))
			allData = append(allData, fileData...)
			successfulFiles++
		} else {
			fmt.Printf("文件 %s 中没有找到有效数据\n", file.Name())
		}
	}

	if len(allData) == 0 {
		return nil, nil
	}

	// 按时间戳排序
	sort.Slice(allData, func(i, j int) bool {
		return allData[i].Timestamp.Before(allData[j].Timestamp)
	})

	fmt.Printf("总共从 %d 个文件中解析出 %d 条记录\n", successfulFiles, len(allData))
	return allData, nil
}

// generateReport 生成内存使用报告和图表
func generateReport(data []MemoryRecord, outputPrefix string, generateHTML bool) error {
	if len(data) == 0 {
		fmt.Println("没有找到有效数据")
		return nil
	}

	// 保存CSV文件
	csvFile := outputPrefix + ".csv"
	file, err := os.Create(csvFile)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// 写入CSV头
	if err := writer.Write([]string{"timestamp", "mem_tot", "mem_free", "swp_tot", "swp_free"}); err != nil {
		return err
	}

	// 写入数据
	for _, record := range data {
		row := []string{
			record.Timestamp.Format("2006-01-02 15:04:05"),
			fmt.Sprintf("%.2f", record.MemTotal),
			fmt.Sprintf("%.2f", record.MemFree),
			fmt.Sprintf("%.2f", record.SwapTotal),
			fmt.Sprintf("%.2f", record.SwapFree),
		}
		if err := writer.Write(row); err != nil {
			return err
		}
	}
	fmt.Printf("已保存CSV文件: %s\n", csvFile)

	// 绘制内存使用图表（静态PNG）
	p := plot.New()

	p.Title.Text = "Memory/Swap Usage Over Time"
	p.X.Label.Text = "Time"
	p.Y.Label.Text = "Size (GB)"

	// 准备数据点
	memTotalData := make(plotter.XYs, len(data))
	memFreeData := make(plotter.XYs, len(data))
	swpTotalData := make(plotter.XYs, len(data))
	swpFreeData := make(plotter.XYs, len(data))

	// 将时间转换为浮点数以便绘图
	baseTime := data[0].Timestamp
	for i, record := range data {
		timeOffset := record.Timestamp.Sub(baseTime).Hours()
		memTotalData[i].X = timeOffset
		memTotalData[i].Y = record.MemTotal
		memFreeData[i].X = timeOffset
		memFreeData[i].Y = record.MemFree
		swpTotalData[i].X = timeOffset
		swpTotalData[i].Y = record.SwapTotal
		swpFreeData[i].X = timeOffset
		swpFreeData[i].Y = record.SwapFree
	}

	// 添加线条
	memTotalLine, err := plotter.NewLine(memTotalData)
	if err != nil {
		return err
	}
	memTotalLine.Color = color.RGBA{R: 255, A: 255}
	p.Add(memTotalLine)
	p.Legend.Add("MEM Total (GB)", memTotalLine)

	memFreeLine, err := plotter.NewLine(memFreeData)
	if err != nil {
		return err
	}
	memFreeLine.Color = color.RGBA{G: 255, A: 255}
	p.Add(memFreeLine)
	p.Legend.Add("MEM Free (GB)", memFreeLine)

	swpTotalLine, err := plotter.NewLine(swpTotalData)
	if err != nil {
		return err
	}
	swpTotalLine.Color = color.RGBA{B: 255, A: 255}
	p.Add(swpTotalLine)
	p.Legend.Add("SWAP Total (GB)", swpTotalLine)

	swpFreeLine, err := plotter.NewLine(swpFreeData)
	if err != nil {
		return err
	}
	swpFreeLine.Color = color.RGBA{R: 255, G: 255, A: 255}
	p.Add(swpFreeLine)
	p.Legend.Add("SWAP Free (GB)", swpFreeLine)

	// 保存图表
	memChartFile := outputPrefix + "_memory_swap.png"
	if err := p.Save(8*vg.Inch, 4*vg.Inch, memChartFile); err != nil {
		return err
	}
	fmt.Printf("已保存内存使用图表: %s\n", memChartFile)

	// 如果指定了generateHTML，则生成交互式HTML报告
	if generateHTML {
		htmlFile := outputPrefix + "_memory_swap.html"
		if err := generateHTMLReport(data, htmlFile); err != nil {
			return err
		}
		fmt.Printf("已保存交互式HTML报告: %s\n", htmlFile)
	}

	return nil
}

// generateHTMLReport 生成交互式HTML报告
func generateHTMLReport(data []MemoryRecord, outputFile string) error {
	// 准备数据
	timestamps := make([]string, len(data))
	memTotal := make([]float64, len(data))
	memFree := make([]float64, len(data))
	swpTotal := make([]float64, len(data))
	swpFree := make([]float64, len(data))

	for i, record := range data {
		timestamps[i] = record.Timestamp.Format("2006-01-02 15:04:05")
		memTotal[i] = record.MemTotal
		memFree[i] = record.MemFree
		swpTotal[i] = record.SwapTotal
		swpFree[i] = record.SwapFree
	}

	// 生成HTML内容
	timestampsJSON, _ := json.Marshal(timestamps)
	memTotalJSON, _ := json.Marshal(memTotal)
	memFreeJSON, _ := json.Marshal(memFree)
	swpTotalJSON, _ := json.Marshal(swpTotal)
	swpFreeJSON, _ := json.Marshal(swpFree)

	htmlTemplate := `
<!DOCTYPE html>
<html>
<head>
    <title>Memory/Swap Usage Over Time</title>
    <script src="https://cdn.jsdelivr.net/npm/chart.js"></script>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        .chart-container { width: 80%; margin: 0 auto; }
    </style>
</head>
<body>
    <h1>Memory/Swap Usage Over Time (Interactive)</h1>
    <div class="chart-container">
        <canvas id="memoryChart"></canvas>
    </div>
    <script>
        const timestamps = %s;
        const memTotal = %s;
        const memFree = %s;
        const swpTotal = %s;
        const swpFree = %s;

        const ctx = document.getElementById('memoryChart').getContext('2d');
        const chart = new Chart(ctx, {
            type: 'line',
            data: {
                labels: timestamps,
                datasets: [
                    {
                        label: 'MEM Total (GB)',
                        data: memTotal,
                        borderColor: 'rgb(255, 0, 0)',
                        fill: false,
                        tension: 0.1
                    },
                    {
                        label: 'MEM Free (GB)',
                        data: memFree,
                        borderColor: 'rgb(0, 255, 0)',
                        fill: false,
                        tension: 0.1
                    },
                    {
                        label: 'SWAP Total (GB)',
                        data: swpTotal,
                        borderColor: 'rgb(0, 0, 255)',
                        fill: false,
                        tension: 0.1
                    },
                    {
                        label: 'SWAP Free (GB)',
                        data: swpFree,
                        borderColor: 'rgb(255, 255, 0)',
                        fill: false,
                        tension: 0.1
                    }
                ]
            },
            options: {
                responsive: true,
                plugins: {
                    title: {
                        display: true,
                        text: 'Memory/Swap Usage Over Time'
                    },
                    tooltip: {
                        mode: 'index',
                        intersect: false,
                    }
                },
                scales: {
                    x: {
                        title: {
                            display: true,
                            text: 'Time'
                        }
                    },
                    y: {
                        title: {
                            display: true,
                            text: 'Size (GB)'
                        }
                    }
                }
            }
        });
    </script>
</body>
</html>
`

	// 将数据填充到HTML模板中
	htmlContent := fmt.Sprintf(
		htmlTemplate,
		timestampsJSON,
		memTotalJSON,
		memFreeJSON,
		swpTotalJSON,
		swpFreeJSON,
	)

	// 写入HTML文件
	return os.WriteFile(outputFile, []byte(htmlContent), 0644)
}

func main() {
	// 创建命令行参数解析器
	logFile := flag.String("log_file", "", "单个atop日志文件的路径")
	logFileShort := flag.String("f", "", "单个atop日志文件的路径 (简写)")
	dirPath := flag.String("dir", "", "包含多个atop日志文件的目录路径")
	dirPathShort := flag.String("d", "", "包含多个atop日志文件的目录路径 (简写)")
	outputPrefix := flag.String("output", "memory_report", "输出文件前缀 (默认: memory_report)")
	outputPrefixShort := flag.String("o", "", "输出文件前缀 (简写)")
	generateHTML := flag.Bool("html", false, "生成交互式HTML报告，可查看每个时间点的详细数据")

	// 解析命令行参数
	flag.Parse()

	// 处理简写参数
	if *logFileShort != "" && *logFile == "" {
		*logFile = *logFileShort
	}
	if *dirPathShort != "" && *dirPath == "" {
		*dirPath = *dirPathShort
	}
	if *outputPrefixShort != "" {
		*outputPrefix = *outputPrefixShort
	}

	// 检查必需参数
	if *logFile == "" && *dirPath == "" {
		fmt.Println("错误: 必须指定 --log_file (-f) 或 --dir (-d) 参数")
		flag.Usage()
		os.Exit(1)
	}

	// 确保不同时指定两个输入源
	if *logFile != "" && *dirPath != "" {
		fmt.Println("错误: --log_file 和 --dir 参数不能同时使用")
		flag.Usage()
		os.Exit(1)
	}

	var data []MemoryRecord
	var err error

	try := func() {
		// 根据输入类型选择解析方法
		if *logFile != "" {
			fmt.Printf("解析单个日志文件: %s\n", *logFile)
			data, err = parseAtopLog(*logFile)
			if err != nil {
				fmt.Printf("错误: %v\n", err)
				os.Exit(1)
			}
		} else {
			fmt.Printf("解析目录中的所有日志文件: %s\n", *dirPath)
			data, err = parseAtopDirectory(*dirPath)
			if err != nil {
				fmt.Printf("错误: %v\n", err)
				os.Exit(1)
			}
		}

		if len(data) == 0 {
			fmt.Println("没有找到有效的内存数据")
			os.Exit(1)
		}

		err = generateReport(data, *outputPrefix, *generateHTML)
		if err != nil {
			fmt.Printf("生成报告时出错: %v\n", err)
			os.Exit(1)
		}

		fmt.Println("报告生成完成！")
	}

	try()
}
