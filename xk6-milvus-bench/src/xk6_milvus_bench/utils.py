"""工具函数模块"""

import random
import string
from typing import Any


def generate_random_string(length: int = 8) -> str:
    """生成随机字符串"""
    return "".join(random.choices(string.ascii_lowercase + string.digits, k=length))


def format_duration(seconds: float) -> str:
    """格式化时长"""
    if seconds < 60:
        return f"{seconds:.1f}s"
    elif seconds < 3600:
        minutes = seconds / 60
        return f"{minutes:.1f}m"
    else:
        hours = seconds / 3600
        return f"{hours:.1f}h"


def parse_duration_to_seconds(duration: str) -> int:
    """解析时长字符串为秒数

    支持格式: 30s, 5m, 1h, 1h30m, 等
    """
    total_seconds = 0
    value = ""

    for char in duration:
        if char.isdigit():
            value += char
        elif char == "h":
            total_seconds += int(value) * 3600
            value = ""
        elif char == "m":
            total_seconds += int(value) * 60
            value = ""
        elif char == "s":
            total_seconds += int(value)
            value = ""

    return total_seconds if total_seconds > 0 else 300


def flatten_dict(d: dict, parent_key: str = "", sep: str = "_") -> dict:
    """扁平化嵌套字典"""
    items: list[tuple[str, Any]] = []
    for k, v in d.items():
        new_key = f"{parent_key}{sep}{k}" if parent_key else k
        if isinstance(v, dict):
            items.extend(flatten_dict(v, new_key, sep=sep).items())
        else:
            items.append((new_key, v))
    return dict(items)


def merge_dicts(base: dict, override: dict) -> dict:
    """深度合并两个字典"""
    result = base.copy()
    for key, value in override.items():
        if key in result and isinstance(result[key], dict) and isinstance(value, dict):
            result[key] = merge_dicts(result[key], value)
        else:
            result[key] = value
    return result


def validate_milvus_host(host: str) -> bool:
    """验证 Milvus 主机地址格式"""
    if not host:
        return False

    # 简单检查格式: host:port
    parts = host.rsplit(":", 1)
    if len(parts) != 2:
        return False

    try:
        port = int(parts[1])
        return 1 <= port <= 65535
    except ValueError:
        return False


def format_bytes(size: int) -> str:
    """格式化字节大小"""
    for unit in ["B", "KB", "MB", "GB", "TB"]:
        if size < 1024:
            return f"{size:.1f}{unit}"
        size /= 1024
    return f"{size:.1f}PB"


def calculate_percentile(values: list[float], percentile: float) -> float:
    """计算百分位数"""
    if not values:
        return 0.0

    sorted_values = sorted(values)
    index = int(len(sorted_values) * percentile / 100)
    index = min(index, len(sorted_values) - 1)

    return sorted_values[index]


def calculate_stats(values: list[float]) -> dict[str, float]:
    """计算统计值"""
    if not values:
        return {
            "min": 0.0,
            "max": 0.0,
            "avg": 0.0,
            "p50": 0.0,
            "p95": 0.0,
            "p99": 0.0,
        }

    sorted_values = sorted(values)
    n = len(sorted_values)

    return {
        "min": sorted_values[0],
        "max": sorted_values[-1],
        "avg": sum(values) / n,
        "p50": sorted_values[int(n * 0.5)],
        "p95": sorted_values[int(n * 0.95)] if n > 20 else sorted_values[-1],
        "p99": sorted_values[int(n * 0.99)] if n > 100 else sorted_values[-1],
    }
