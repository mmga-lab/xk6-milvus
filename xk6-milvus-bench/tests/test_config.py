"""配置模块测试"""

import os
import tempfile
from pathlib import Path

import pytest
import yaml

from xk6_milvus_bench.config import (
    Config,
    expand_env_vars,
    deep_merge,
    load_yaml_config,
    load_config,
    config_to_dict,
)


class TestExpandEnvVars:
    """环境变量展开测试"""

    def test_expand_simple_env_var(self):
        """测试简单环境变量展开"""
        os.environ["TEST_VAR"] = "test_value"
        result = expand_env_vars("${TEST_VAR}")
        assert result == "test_value"
        del os.environ["TEST_VAR"]

    def test_expand_env_var_with_default(self):
        """测试带默认值的环境变量展开"""
        # 未设置环境变量时使用默认值
        result = expand_env_vars("${UNSET_VAR:-default_value}")
        assert result == "default_value"

        # 已设置环境变量时使用环境变量值
        os.environ["SET_VAR"] = "actual_value"
        result = expand_env_vars("${SET_VAR:-default_value}")
        assert result == "actual_value"
        del os.environ["SET_VAR"]

    def test_expand_in_dict(self):
        """测试字典中的环境变量展开"""
        os.environ["HOST"] = "milvus.example.com"
        data = {
            "milvus": {
                "host": "${HOST:-localhost}:19530",
                "token": "${TOKEN:-}",
            }
        }
        result = expand_env_vars(data)
        assert result["milvus"]["host"] == "milvus.example.com:19530"
        assert result["milvus"]["token"] == ""
        del os.environ["HOST"]

    def test_expand_in_list(self):
        """测试列表中的环境变量展开"""
        os.environ["ITEM"] = "value"
        data = ["${ITEM}", "${UNSET:-default}"]
        result = expand_env_vars(data)
        assert result == ["value", "default"]
        del os.environ["ITEM"]


class TestDeepMerge:
    """深度合并测试"""

    def test_simple_merge(self):
        """测试简单合并"""
        base = {"a": 1, "b": 2}
        override = {"b": 3, "c": 4}
        result = deep_merge(base, override)
        assert result == {"a": 1, "b": 3, "c": 4}

    def test_nested_merge(self):
        """测试嵌套合并"""
        base = {
            "milvus": {"host": "localhost", "port": 19530},
            "execution": {"vus": 10},
        }
        override = {
            "milvus": {"host": "remote.host"},
            "execution": {"duration": "5m"},
        }
        result = deep_merge(base, override)
        assert result["milvus"]["host"] == "remote.host"
        assert result["milvus"]["port"] == 19530
        assert result["execution"]["vus"] == 10
        assert result["execution"]["duration"] == "5m"

    def test_override_non_dict_with_dict(self):
        """测试用字典覆盖非字典值"""
        base = {"a": "string"}
        override = {"a": {"nested": "dict"}}
        result = deep_merge(base, override)
        assert result["a"] == {"nested": "dict"}


class TestLoadYamlConfig:
    """YAML 配置加载测试"""

    def test_load_simple_yaml(self):
        """测试加载简单 YAML"""
        with tempfile.NamedTemporaryFile(mode="w", suffix=".yaml", delete=False) as f:
            yaml.dump({"key": "value", "number": 42}, f)
            f.flush()
            path = Path(f.name)

        try:
            result = load_yaml_config(path)
            assert result["key"] == "value"
            assert result["number"] == 42
        finally:
            path.unlink()

    def test_load_yaml_with_env_vars(self):
        """测试加载带环境变量的 YAML"""
        os.environ["TEST_HOST"] = "test.example.com"

        with tempfile.NamedTemporaryFile(mode="w", suffix=".yaml", delete=False) as f:
            yaml.dump({"host": "${TEST_HOST:-localhost}"}, f)
            f.flush()
            path = Path(f.name)

        try:
            result = load_yaml_config(path)
            assert result["host"] == "test.example.com"
        finally:
            path.unlink()
            del os.environ["TEST_HOST"]


class TestConfig:
    """配置模型测试"""

    def test_default_config(self):
        """测试默认配置"""
        config = Config()
        assert config.milvus.host == "localhost:19530"
        assert config.execution.vus == 10
        assert config.execution.duration == "5m"
        assert config.thresholds.min_recall == 0.95

    def test_config_from_dict(self):
        """测试从字典创建配置"""
        data = {
            "milvus": {"host": "custom.host:19530"},
            "execution": {"vus": 50, "duration": "10m"},
        }
        config = Config(**data)
        assert config.milvus.host == "custom.host:19530"
        assert config.execution.vus == 50
        assert config.execution.duration == "10m"

    def test_config_to_dict(self):
        """测试配置转换为字典"""
        config = Config()
        result = config_to_dict(config)
        assert isinstance(result, dict)
        assert "milvus" in result
        assert "execution" in result
        assert result["milvus"]["host"] == "localhost:19530"


class TestLoadConfig:
    """配置加载测试"""

    def test_load_config_with_scenario(self):
        """测试加载场景配置"""
        # 这个测试依赖于实际的配置文件存在
        # 在没有配置文件的情况下应该返回默认配置
        config = load_config("search", None)
        assert config.scenario.type == "search"
