import re
import pandas as pd
import matplotlib.pyplot as plt
import argparse
from datetime import datetime
import plotly.graph_objects as go
from plotly.subplots import make_subplots

def parse_atop_log(file_path):
    """解析atop日志文件，提取内存和交换空间使用数据"""
    data = []
    current_timestamp = None
    
    with open(file_path, 'r') as file:
        for line in file:
            # 匹配时间戳行
            timestamp_match = re.match(r"ATOP - \w+\s+(\d{4}/\d{2}/\d{2}\s+\d{2}:\d{2}:\d{2})", line)
            if timestamp_match:
                current_timestamp = datetime.strptime(timestamp_match.group(1), "%Y/%m/%d %H:%M:%S")
                continue
                
            # 匹配MEM行
            mem_match = re.match(r"MEM \| tot\s+([\d.]+)(G|M) \| free\s+([\d.]+)(G|M)", line)
            if mem_match and current_timestamp:
                mem_tot = float(mem_match.group(1))
                if mem_match.group(2) == 'M':
                    mem_tot /= 1024
                mem_free = float(mem_match.group(3))
                if mem_match.group(4) == 'M':
                    mem_free /= 1024
                    
            # 匹配SWP行
            swp_match = re.match(r"SWP \| tot\s+([\d.]+)(G|M) \| free\s+([\d.]+)(G|M)", line)
            if swp_match and current_timestamp and 'mem_tot' in locals():
                swp_tot = float(swp_match.group(1))
                if swp_match.group(2) == 'M':
                    swp_tot /= 1024
                swp_free = float(swp_match.group(3))
                if swp_match.group(4) == 'M':
                    swp_free /= 1024
                
                # 添加到数据列表
                data.append({
                    'timestamp': current_timestamp,
                    'mem_tot': mem_tot,
                    'mem_free': mem_free,
                    'swp_tot': swp_tot,
                    'swp_free': swp_free
                })
                
    return pd.DataFrame(data)

def generate_report(data, output_prefix='memory_report', generate_html=False):
    """生成内存使用报告和图表"""
    if data.empty:
        print("没有找到有效数据")
        return
    
    # 保存CSV文件
    csv_file = f"{output_prefix}.csv"
    data.to_csv(csv_file, index=False)
    print(f"已保存CSV文件: {csv_file}")
    
    # 绘制内存使用图表（静态PNG）
    plt.figure(figsize=(12, 6))
    plt.plot(data['timestamp'], data['mem_tot'], label='MEM Total (GB)')
    plt.plot(data['timestamp'], data['mem_free'], label='MEM Free (GB)')
    plt.plot(data['timestamp'], data['swp_tot'], label='SWAP Total (GB)')
    plt.plot(data['timestamp'], data['swp_free'], label='SWAP Free (GB)')
    plt.title('Memory/Swap Usage Over Time')
    plt.xlabel('Time')
    plt.ylabel('Size (GB)')
    plt.legend()
    plt.grid(True)
    plt.xticks(rotation=45)
    plt.tight_layout()
    
    mem_chart_file = f"{output_prefix}_memory_swap.png"
    plt.savefig(mem_chart_file)
    print(f"已保存内存使用图表: {mem_chart_file}")
    plt.close()
    
    # 如果指定了generate_html，则生成交互式HTML报告
    if generate_html:
        fig = go.Figure()
        
        # 添加内存数据
        fig.add_trace(go.Scatter(
            x=data['timestamp'],
            y=data['mem_tot'],
            name='MEM Total (GB)',
            mode='lines',
            hovertemplate='%{x}<br>MEM Total: %{y:.2f} GB<extra></extra>'
        ))
        fig.add_trace(go.Scatter(
            x=data['timestamp'],
            y=data['mem_free'],
            name='MEM Free (GB)',
            mode='lines',
            hovertemplate='%{x}<br>MEM Free: %{y:.2f} GB<extra></extra>'
        ))
        
        # 添加交换空间数据
        fig.add_trace(go.Scatter(
            x=data['timestamp'],
            y=data['swp_tot'],
            name='SWAP Total (GB)',
            mode='lines',
            hovertemplate='%{x}<br>SWAP Total: %{y:.2f} GB<extra></extra>'
        ))
        fig.add_trace(go.Scatter(
            x=data['timestamp'],
            y=data['swp_free'],
            name='SWAP Free (GB)',
            mode='lines',
            hovertemplate='%{x}<br>SWAP Free: %{y:.2f} GB<extra></extra>'
        ))
        
        # 更新布局
        fig.update_layout(
            title='Memory/Swap Usage Over Time (Interactive)',
            xaxis_title='Time',
            yaxis_title='Size (GB)',
            hovermode='x unified',
            showlegend=True
        )
        
        # 保存为HTML文件
        html_file = f"{output_prefix}_memory_swap.html"
        fig.write_html(html_file)
        print(f"已保存交互式HTML报告: {html_file}")
    

if __name__ == "__main__":
    # 创建命令行参数解析器
    parser = argparse.ArgumentParser(description='解析atop日志文件并生成内存使用报告')
    parser.add_argument('--log_file', '-f', required=True, help='atop日志文件的路径')
    parser.add_argument('--output', '-o', default='memory_report',
                      help='输出文件前缀 (默认: memory_report)')
    parser.add_argument('--html', action='store_true',
                      help='生成交互式HTML报告，可查看每个时间点的详细数据')
    
    # 解析命令行参数
    args = parser.parse_args()
    
    try:
        data = parse_atop_log(args.log_file)
        if not data.empty:
            generate_report(data, args.output, args.html)
            print("报告生成完成！")
        else:
            print("日志文件中没有找到有效的内存数据")
    except FileNotFoundError:
        print(f"错误: 文件 {args.log_file} 未找到")
    except Exception as e:
        print(f"处理日志时发生错误: {str(e)}")