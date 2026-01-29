"""配置管理模块"""

import os
import re
from pathlib import Path
from typing import Any

import yaml
from pydantic import BaseModel, Field


class MilvusConfig(BaseModel):
    """Milvus 连接配置"""

    host: str = "localhost:19530"
    token: str = ""
    database: str = "default"


class IndexParams(BaseModel):
    """索引参数"""

    M: int = 16
    efConstruction: int = 200


class IndexConfig(BaseModel):
    """索引配置"""

    type: str = "HNSW"
    metric: str = "L2"
    params: IndexParams = Field(default_factory=IndexParams)


class CollectionConfig(BaseModel):
    """集合配置"""

    name_prefix: str = "perf_test_"
    vector_dim: int = 128
    index: IndexConfig = Field(default_factory=IndexConfig)


class StageConfig(BaseModel):
    """执行阶段配置"""

    duration: str
    target: int


class ExecutionConfig(BaseModel):
    """k6 执行配置"""

    vus: int = 10
    duration: str = "5m"
    stages: list[StageConfig] | None = None


class ThresholdsConfig(BaseModel):
    """性能阈值配置"""

    search_p95_ms: int = 100
    search_p99_ms: int = 200
    insert_p95_ms: int = 500
    min_recall: float = 0.95
    max_error_rate: float = 0.01


class ResourceRequests(BaseModel):
    """K8s 资源请求"""

    cpu: str = "500m"
    memory: str = "512Mi"


class ResourceLimits(BaseModel):
    """K8s 资源限制"""

    cpu: str = "2"
    memory: str = "2Gi"


class Resources(BaseModel):
    """K8s 资源配置"""

    requests: ResourceRequests = Field(default_factory=ResourceRequests)
    limits: ResourceLimits = Field(default_factory=ResourceLimits)


class KubernetesConfig(BaseModel):
    """K8s 配置"""

    namespace: str = "k6-tests"
    image: str = "harbor.milvus.io/milvus/k6-milvus:latest"
    parallelism: int = 1
    resources: Resources = Field(default_factory=Resources)


class PrometheusConfig(BaseModel):
    """Prometheus 配置"""

    enabled: bool = True
    remote_write_url: str = "http://prometheus:9090/api/v1/write"


class SetupConfig(BaseModel):
    """数据准备配置"""

    data_size: int = 10000
    batch_size: int = 1000


class SearchConfig(BaseModel):
    """搜索参数配置"""

    topk: int = 10
    output_fields: list[str] = Field(default_factory=lambda: ["category", "price"])
    nprobe: int = 16
    ef: int = 64


class BenchmarkConfig(BaseModel):
    """Benchmark 数据集配置"""

    dataset: str = "sift-128-euclidean"
    dimension: int = 128
    metric: str = "L2"
    train_path: str = ""
    test_path: str = ""
    neighbors_path: str = ""


class VariantConfig(BaseModel):
    """测试变体配置"""

    name: str
    filter: str | None = None
    topk: int | None = None


class ScenarioConfig(BaseModel):
    """场景配置"""

    name: str = "default"
    description: str = ""
    type: str = "search"
    setup: SetupConfig = Field(default_factory=SetupConfig)
    search: SearchConfig = Field(default_factory=SearchConfig)
    benchmark: BenchmarkConfig = Field(default_factory=BenchmarkConfig)
    variants: list[VariantConfig] = Field(
        default_factory=lambda: [
            VariantConfig(name="basic", filter=None),
            VariantConfig(name="filtered", filter="price > 50"),
        ]
    )


class Config(BaseModel):
    """完整配置"""

    milvus: MilvusConfig = Field(default_factory=MilvusConfig)
    collection: CollectionConfig = Field(default_factory=CollectionConfig)
    execution: ExecutionConfig = Field(default_factory=ExecutionConfig)
    thresholds: ThresholdsConfig = Field(default_factory=ThresholdsConfig)
    kubernetes: KubernetesConfig = Field(default_factory=KubernetesConfig)
    prometheus: PrometheusConfig = Field(default_factory=PrometheusConfig)
    scenario: ScenarioConfig = Field(default_factory=ScenarioConfig)
    matrix: dict[str, list[Any]] | None = None


def expand_env_vars(value: Any) -> Any:
    """递归展开环境变量"""
    if isinstance(value, str):
        # 匹配 ${VAR:-default} 或 ${VAR} 格式
        pattern = r"\$\{([^}:]+)(?::-([^}]*))?\}"

        def replace(match: re.Match) -> str:
            var_name = match.group(1)
            default = match.group(2) or ""
            return os.environ.get(var_name, default)

        return re.sub(pattern, replace, value)
    elif isinstance(value, dict):
        return {k: expand_env_vars(v) for k, v in value.items()}
    elif isinstance(value, list):
        return [expand_env_vars(item) for item in value]
    return value


def deep_merge(base: dict, override: dict) -> dict:
    """深度合并两个字典"""
    result = base.copy()
    for key, value in override.items():
        if key in result and isinstance(result[key], dict) and isinstance(value, dict):
            result[key] = deep_merge(result[key], value)
        else:
            result[key] = value
    return result


def get_templates_dir() -> Path:
    """获取模板目录路径"""
    return Path(__file__).parent.parent.parent / "templates"


def get_configs_dir() -> Path:
    """获取配置文件目录路径"""
    return Path(__file__).parent.parent.parent / "configs"


def load_yaml_config(path: Path) -> dict:
    """加载 YAML 配置文件"""
    with open(path) as f:
        data = yaml.safe_load(f) or {}
    return expand_env_vars(data)


def load_scenario_config(scenario: str, config_path: Path | None = None) -> dict:
    """加载场景配置

    优先级：
    1. 用户指定的配置文件 (-c)
    2. configs/scenarios/<scenario>.yaml
    3. configs/default.yaml
    """
    configs_dir = get_configs_dir()

    # 加载默认配置
    default_path = configs_dir / "default.yaml"
    if default_path.exists():
        base_config = load_yaml_config(default_path)
    else:
        base_config = {}

    # 加载场景配置
    scenario_path = configs_dir / "scenarios" / f"{scenario}.yaml"
    if scenario_path.exists():
        scenario_config = load_yaml_config(scenario_path)
        # 处理 extends
        if "extends" in scenario_config:
            extends_path = configs_dir / f"{scenario_config['extends']}.yaml"
            if extends_path.exists():
                extends_config = load_yaml_config(extends_path)
                base_config = deep_merge(base_config, extends_config)
            del scenario_config["extends"]
        base_config = deep_merge(base_config, scenario_config)

    # 加载用户指定的配置文件
    if config_path and config_path.exists():
        user_config = load_yaml_config(config_path)
        if "extends" in user_config:
            del user_config["extends"]
        base_config = deep_merge(base_config, user_config)

    # 设置场景名称
    if "scenario" not in base_config:
        base_config["scenario"] = {}
    if "name" not in base_config["scenario"]:
        base_config["scenario"]["name"] = scenario
    if "type" not in base_config["scenario"]:
        base_config["scenario"]["type"] = scenario

    return base_config


def load_config(scenario: str, config_path: Path | None = None) -> Config:
    """加载并验证配置"""
    raw_config = load_scenario_config(scenario, config_path)
    return Config(**raw_config)


def config_to_dict(config: Config) -> dict:
    """将配置对象转换为字典（用于模板渲染）"""
    return config.model_dump()
