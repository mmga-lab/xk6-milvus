"""工具函数模块测试"""

import pytest

from xk6_milvus_bench.utils import (
    generate_random_string,
    format_duration,
    parse_duration_to_seconds,
    flatten_dict,
    merge_dicts,
    validate_milvus_host,
    format_bytes,
    calculate_percentile,
    calculate_stats,
)


class TestGenerateRandomString:
    """随机字符串生成测试"""

    def test_default_length(self):
        """测试默认长度"""
        result = generate_random_string()
        assert len(result) == 8

    def test_custom_length(self):
        """测试自定义长度"""
        result = generate_random_string(16)
        assert len(result) == 16

    def test_uniqueness(self):
        """测试唯一性"""
        results = [generate_random_string() for _ in range(100)]
        assert len(set(results)) == 100


class TestFormatDuration:
    """时长格式化测试"""

    def test_format_seconds(self):
        """测试秒格式化"""
        assert format_duration(30) == "30.0s"
        assert format_duration(59.9) == "59.9s"

    def test_format_minutes(self):
        """测试分钟格式化"""
        assert format_duration(60) == "1.0m"
        assert format_duration(300) == "5.0m"

    def test_format_hours(self):
        """测试小时格式化"""
        assert format_duration(3600) == "1.0h"
        assert format_duration(7200) == "2.0h"


class TestParseDurationToSeconds:
    """时长解析测试"""

    def test_parse_seconds(self):
        """测试解析秒"""
        assert parse_duration_to_seconds("30s") == 30

    def test_parse_minutes(self):
        """测试解析分钟"""
        assert parse_duration_to_seconds("5m") == 300

    def test_parse_hours(self):
        """测试解析小时"""
        assert parse_duration_to_seconds("1h") == 3600

    def test_parse_combined(self):
        """测试解析组合格式"""
        assert parse_duration_to_seconds("1h30m") == 5400
        assert parse_duration_to_seconds("2h30m15s") == 9015

    def test_parse_invalid(self):
        """测试解析无效格式"""
        # 无效格式返回默认值 300
        assert parse_duration_to_seconds("invalid") == 300


class TestFlattenDict:
    """字典扁平化测试"""

    def test_flatten_simple(self):
        """测试简单字典扁平化"""
        d = {"a": 1, "b": 2}
        result = flatten_dict(d)
        assert result == {"a": 1, "b": 2}

    def test_flatten_nested(self):
        """测试嵌套字典扁平化"""
        d = {"a": {"b": {"c": 1}}}
        result = flatten_dict(d)
        assert result == {"a_b_c": 1}

    def test_flatten_mixed(self):
        """测试混合字典扁平化"""
        d = {"a": 1, "b": {"c": 2, "d": {"e": 3}}}
        result = flatten_dict(d)
        assert result == {"a": 1, "b_c": 2, "b_d_e": 3}


class TestMergeDicts:
    """字典合并测试"""

    def test_merge_simple(self):
        """测试简单合并"""
        base = {"a": 1}
        override = {"b": 2}
        result = merge_dicts(base, override)
        assert result == {"a": 1, "b": 2}

    def test_merge_override(self):
        """测试覆盖合并"""
        base = {"a": 1}
        override = {"a": 2}
        result = merge_dicts(base, override)
        assert result == {"a": 2}

    def test_merge_nested(self):
        """测试嵌套合并"""
        base = {"a": {"b": 1, "c": 2}}
        override = {"a": {"b": 3}}
        result = merge_dicts(base, override)
        assert result == {"a": {"b": 3, "c": 2}}


class TestValidateMilvusHost:
    """Milvus 主机验证测试"""

    def test_valid_host(self):
        """测试有效主机"""
        assert validate_milvus_host("localhost:19530") is True
        assert validate_milvus_host("192.168.1.1:19530") is True
        assert validate_milvus_host("milvus.example.com:19530") is True

    def test_invalid_host(self):
        """测试无效主机"""
        assert validate_milvus_host("") is False
        assert validate_milvus_host("localhost") is False
        assert validate_milvus_host("localhost:") is False
        assert validate_milvus_host("localhost:invalid") is False
        assert validate_milvus_host("localhost:0") is False
        assert validate_milvus_host("localhost:99999") is False


class TestFormatBytes:
    """字节格式化测试"""

    def test_format_bytes(self):
        """测试字节格式化"""
        assert format_bytes(512) == "512.0B"
        assert format_bytes(1024) == "1.0KB"
        assert format_bytes(1024 * 1024) == "1.0MB"
        assert format_bytes(1024 * 1024 * 1024) == "1.0GB"


class TestCalculatePercentile:
    """百分位数计算测试"""

    def test_calculate_percentile(self):
        """测试百分位数计算"""
        values = list(range(1, 101))  # 1-100
        # 百分位数计算有不同实现方式，这里检查是否在合理范围内
        assert 50 <= calculate_percentile(values, 50) <= 51
        assert 95 <= calculate_percentile(values, 95) <= 96
        assert 99 <= calculate_percentile(values, 99) <= 100

    def test_empty_list(self):
        """测试空列表"""
        assert calculate_percentile([], 50) == 0.0


class TestCalculateStats:
    """统计值计算测试"""

    def test_calculate_stats(self):
        """测试统计值计算"""
        values = list(range(1, 101))  # 1-100
        stats = calculate_stats(values)
        assert stats["min"] == 1
        assert stats["max"] == 100
        assert stats["avg"] == 50.5
        # p50 应该在 50-51 范围内（取决于计算方式）
        assert 50 <= stats["p50"] <= 51
        assert 95 <= stats["p95"] <= 96

    def test_empty_list(self):
        """测试空列表"""
        stats = calculate_stats([])
        assert stats["min"] == 0.0
        assert stats["max"] == 0.0
        assert stats["avg"] == 0.0
