# Atop Analyze

Atop Analyze 是一个用于解析和分析 atop 日志文件的工具，专注于系统内存使用情况的分析。该工具提供了 Go 和 Python 两个版本的实现，可以生成详细的内存使用报告和可视化图表。

## 功能特性

- 解析 atop 日志文件中的内存使用数据
- 支持单个或多个日志文件的批量处理
- 生成 CSV 格式的数据报告
- 创建内存使用趋势的可视化图表（PNG格式）
- 生成交互式 HTML 报告
- 支持内存和交换空间使用情况的分析

## 安装

### Go 版本

1. 确保已安装 Go 1.16 或更高版本
2. 克隆仓库：
```bash
git clone https://github.com/ltarcher/Atop_analyze.git
cd Atop_analyze
```
3. 安装依赖：
```bash
go mod download
```
4. 编译程序：
```bash
go build atop_parser_mem.go
```

### Python 版本

1. 确保已安装 Python 3.6 或更高版本
2. 克隆仓库后直接使用 Python 脚本即可

## 使用方法

1. 先将atop日志转为txt文件，例如: cat atop_xxx.atop > atop_20250611.txt

### Go 版本

```bash
# Windows
atop_parser_mem.exe -f path/to/atop/logs/atop_20250611.txt -o atop_name_prefix --html

# 指定atop日志目录
atop_parser_mem.exe -d path/to/atop/logs -o atop_name_prefix --html

# Linux/Mac
./atop_parser_mem -f path/to/atop/logs/atop_20250611.txt -o atop_name_prefix --html

# 指定atop日志目录
./atop_parser_mem -d path/to/atop/logs -o atop_name_prefix --html

```

### Python 版本

```bash
python atop_parser_mem.py -f path/to/atop/logs/atop_20250611.txt -o atop_name_prefix --html

python atop_parser_mem.py -d path/to/atop/logs -o atop_name_prefix --html
```

## 输入文件格式

工具接受标准的 atop 日志文件作为输入。atop 日志文件应包含系统内存使用的相关信息。

## 输出说明

1. CSV 报告：包含时间序列的内存使用数据
2. PNG 图表：可视化展示内存使用趋势
3. HTML 报告：交互式的内存使用分析报告

## 目录结构

```
.
├── atop_parser_mem.go    # Go 版本实现
├── atop_parser_mem.py    # Python 版本实现
├── atop_analyze_mem.exe  # 编译后的可执行文件
├── data/                 # 示例数据目录
├── go.mod               # Go 模块定义
└── go.sum               # Go 依赖版本锁定文件
```

## 注意事项

- 确保输入的 atop 日志文件格式正确
- 对于大型日志文件，建议预留足够的系统内存
- 生成的图表和报告会保存在程序运行目录下

## 许可证

[添加许可证信息]

## 贡献

欢迎提交 Issue 和 Pull Request！