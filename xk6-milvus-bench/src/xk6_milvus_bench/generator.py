"""脚本/资源生成器模块"""

import itertools
import json
from datetime import datetime
from pathlib import Path
from typing import Any

from jinja2 import Environment, FileSystemLoader, select_autoescape

from .config import Config, config_to_dict, get_templates_dir


class ScriptGenerator:
    """k6 脚本和 K8s 资源生成器"""

    def __init__(self, templates_dir: Path | None = None):
        self.templates_dir = templates_dir or get_templates_dir()
        self.env = Environment(
            loader=FileSystemLoader(str(self.templates_dir)),
            trim_blocks=True,
            lstrip_blocks=True,
            autoescape=select_autoescape(default=False),
        )
        # 添加自定义过滤器
        self.env.filters["tojson"] = lambda x: json.dumps(x)

    def generate_script(
        self,
        scenario: str,
        config: Config,
        output_dir: Path,
        matrix_params: dict[str, Any] | None = None,
    ) -> Path:
        """生成 k6 测试脚本

        Args:
            scenario: 场景名称
            config: 配置对象
            output_dir: 输出目录
            matrix_params: 矩阵参数（可选）

        Returns:
            生成的脚本路径
        """
        template_name = f"scripts/{scenario}.js.j2"
        try:
            template = self.env.get_template(template_name)
        except Exception:
            # 如果场景模板不存在，使用通用模板
            template = self.env.get_template("scripts/base.js.j2")

        # 准备模板上下文
        context = config_to_dict(config)
        context["generated_at"] = datetime.now().isoformat()
        context["scenario_type"] = scenario

        # 合并矩阵参数
        if matrix_params:
            # 更新执行配置
            if "vus" in matrix_params:
                context["execution"]["vus"] = matrix_params["vus"]
            # 更新搜索参数
            if "topk" in matrix_params:
                context["scenario"]["search"]["topk"] = matrix_params["topk"]
            if "ef" in matrix_params:
                context["scenario"]["search"]["ef"] = matrix_params["ef"]
            if "nprobe" in matrix_params:
                context["scenario"]["search"]["nprobe"] = matrix_params["nprobe"]

        content = template.render(**context)

        # 生成文件名
        output_dir.mkdir(parents=True, exist_ok=True)
        if matrix_params:
            param_str = "_".join(f"{k}{v}" for k, v in sorted(matrix_params.items()))
            filename = f"{scenario}_{param_str}.js"
        else:
            filename = f"{scenario}.js"

        output_path = output_dir / filename
        output_path.write_text(content)

        return output_path

    def generate_k8s_resources(
        self,
        scenario: str,
        config: Config,
        output_dir: Path,
        script_filename: str | None = None,
    ) -> dict[str, Path]:
        """生成 K8s 资源文件

        Args:
            scenario: 场景名称
            config: 配置对象
            output_dir: 输出目录
            script_filename: 脚本文件名

        Returns:
            生成的资源文件路径字典
        """
        output_dir.mkdir(parents=True, exist_ok=True)
        context = config_to_dict(config)
        timestamp = datetime.now().strftime("%Y%m%d%H%M%S")

        context["testrun_name"] = f"milvus-{scenario}-{timestamp}"
        context["configmap_name"] = f"milvus-test-{scenario}"
        context["script_filename"] = script_filename or f"{scenario}.js"
        context["generated_at"] = datetime.now().isoformat()

        results = {}

        # 生成 TestRun
        testrun_template = self.env.get_template("k8s/testrun.yaml.j2")
        testrun_content = testrun_template.render(**context)
        testrun_path = output_dir / f"testrun-{scenario}.yaml"
        testrun_path.write_text(testrun_content)
        results["testrun"] = testrun_path

        # 生成 ConfigMap
        configmap_template = self.env.get_template("k8s/configmap.yaml.j2")
        configmap_content = configmap_template.render(**context)
        configmap_path = output_dir / f"configmap-{scenario}.yaml"
        configmap_path.write_text(configmap_content)
        results["configmap"] = configmap_path

        return results

    def generate_all(
        self,
        scenario: str,
        config: Config,
        output_dir: Path,
        include_k8s: bool = True,
    ) -> dict[str, Path]:
        """生成所有资源文件

        Args:
            scenario: 场景名称
            config: 配置对象
            output_dir: 输出目录
            include_k8s: 是否生成 K8s 资源

        Returns:
            生成的文件路径字典
        """
        results = {}

        # 生成脚本
        script_path = self.generate_script(scenario, config, output_dir)
        results["script"] = script_path

        # 生成 K8s 资源
        if include_k8s:
            k8s_resources = self.generate_k8s_resources(
                scenario, config, output_dir, script_path.name
            )
            results.update(k8s_resources)

        return results


def expand_matrix(matrix: dict[str, list[Any]]) -> list[dict[str, Any]]:
    """展开测试矩阵为参数组合列表

    Args:
        matrix: 参数矩阵，如 {"topk": [10, 50], "vus": [5, 10]}

    Returns:
        参数组合列表，如 [{"topk": 10, "vus": 5}, {"topk": 10, "vus": 10}, ...]
    """
    if not matrix:
        return [{}]

    keys = list(matrix.keys())
    values = list(matrix.values())

    combinations = []
    for combo in itertools.product(*values):
        combinations.append(dict(zip(keys, combo, strict=True)))

    return combinations


def parse_matrix_arg(matrix_args: list[str]) -> dict[str, list[Any]]:
    """解析命令行矩阵参数

    Args:
        matrix_args: 命令行参数列表，如 ["topk=[10,50,100]", "vus=[5,10]"]

    Returns:
        解析后的矩阵字典
    """
    matrix = {}
    for arg in matrix_args:
        if "=" not in arg:
            continue
        key, value_str = arg.split("=", 1)
        key = key.strip()
        value_str = value_str.strip()

        # 解析列表格式 [1,2,3]
        if value_str.startswith("[") and value_str.endswith("]"):
            value_str = value_str[1:-1]
            values = []
            for v in value_str.split(","):
                v = v.strip()
                # 尝试转换为数字
                try:
                    if "." in v:
                        values.append(float(v))
                    else:
                        values.append(int(v))
                except ValueError:
                    values.append(v)
            matrix[key] = values
        else:
            # 单个值
            try:
                if "." in value_str:
                    matrix[key] = [float(value_str)]
                else:
                    matrix[key] = [int(value_str)]
            except ValueError:
                matrix[key] = [value_str]

    return matrix
