"""测试运行器模块"""

import json
import os
import subprocess
import tempfile
from abc import ABC, abstractmethod
from dataclasses import dataclass, field
from datetime import datetime
from pathlib import Path
from typing import Any

from rich.console import Console

from .config import Config


@dataclass
class TestResult:
    """测试结果"""

    status: str  # success, failed, error
    scenario: str
    duration_seconds: float
    metrics: dict[str, Any] = field(default_factory=dict)
    thresholds_passed: bool = True
    error_message: str | None = None
    output_file: Path | None = None
    matrix_params: dict[str, Any] | None = None

    def to_dict(self) -> dict:
        """转换为字典"""
        return {
            "status": self.status,
            "scenario": self.scenario,
            "duration_seconds": self.duration_seconds,
            "metrics": self.metrics,
            "thresholds_passed": self.thresholds_passed,
            "error_message": self.error_message,
            "output_file": str(self.output_file) if self.output_file else None,
            "matrix_params": self.matrix_params,
        }


class BaseRunner(ABC):
    """运行器基类"""

    def __init__(self):
        self.console = Console()

    @abstractmethod
    def run(
        self,
        script_path: Path,
        config: Config,
        matrix_params: dict[str, Any] | None = None,
    ) -> TestResult:
        """运行测试"""
        pass


class LocalRunner(BaseRunner):
    """本地运行器"""

    def __init__(self, k6_binary: str = "k6"):
        super().__init__()
        self.k6_binary = k6_binary

    def find_k6_binary(self) -> str:
        """查找 k6 二进制文件"""
        # 优先使用当前目录的 k6
        local_k6 = Path("./k6")
        if local_k6.exists() and local_k6.is_file():
            return str(local_k6.absolute())

        # 尝试 xk6-milvus 项目根目录
        project_k6 = Path(__file__).parent.parent.parent.parent / "k6"
        if project_k6.exists() and project_k6.is_file():
            return str(project_k6.absolute())

        # 使用系统 k6
        return self.k6_binary

    def run(
        self,
        script_path: Path,
        config: Config,
        matrix_params: dict[str, Any] | None = None,
    ) -> TestResult:
        """本地运行 k6 测试"""
        k6_path = self.find_k6_binary()
        start_time = datetime.now()

        # 构建命令
        cmd = [k6_path, "run"]

        # 添加 VUs 和 duration（如果没有 stages）
        if not config.execution.stages:
            cmd.extend(["--vus", str(config.execution.vus)])
            cmd.extend(["--duration", config.execution.duration])

        # 添加 JSON 输出
        with tempfile.NamedTemporaryFile(
            mode="w", suffix=".json", delete=False
        ) as json_output:
            json_output_path = Path(json_output.name)

        cmd.extend(["--out", f"json={json_output_path}"])

        # 添加环境变量
        env = os.environ.copy()
        env["MILVUS_HOST"] = config.milvus.host
        if config.milvus.token:
            env["MILVUS_TOKEN"] = config.milvus.token

        # 添加 Prometheus 输出（如果启用）
        if config.prometheus.enabled:
            cmd.extend(["--out", "experimental-prometheus-rw"])
            env["K6_PROMETHEUS_RW_SERVER_URL"] = config.prometheus.remote_write_url

        cmd.append(str(script_path))

        self.console.print(f"[bold blue]Running:[/bold blue] {' '.join(cmd)}")

        try:
            result = subprocess.run(
                cmd,
                env=env,
                capture_output=True,
                text=True,
                timeout=self._parse_duration(config.execution.duration) + 300,  # 额外 5 分钟
            )

            duration = (datetime.now() - start_time).total_seconds()

            # 解析输出
            metrics = self._parse_k6_output(result.stdout, json_output_path)

            if result.returncode == 0:
                return TestResult(
                    status="success",
                    scenario=config.scenario.name,
                    duration_seconds=duration,
                    metrics=metrics,
                    thresholds_passed=True,
                    output_file=json_output_path,
                    matrix_params=matrix_params,
                )
            else:
                # 检查是否只是阈值失败
                if "thresholds" in result.stdout.lower() and "failed" in result.stdout.lower():
                    return TestResult(
                        status="success",
                        scenario=config.scenario.name,
                        duration_seconds=duration,
                        metrics=metrics,
                        thresholds_passed=False,
                        output_file=json_output_path,
                        matrix_params=matrix_params,
                    )
                return TestResult(
                    status="failed",
                    scenario=config.scenario.name,
                    duration_seconds=duration,
                    metrics=metrics,
                    thresholds_passed=False,
                    error_message=result.stderr or result.stdout,
                    output_file=json_output_path,
                    matrix_params=matrix_params,
                )

        except subprocess.TimeoutExpired:
            duration = (datetime.now() - start_time).total_seconds()
            return TestResult(
                status="error",
                scenario=config.scenario.name,
                duration_seconds=duration,
                error_message="Test timed out",
                matrix_params=matrix_params,
            )
        except Exception as e:
            duration = (datetime.now() - start_time).total_seconds()
            return TestResult(
                status="error",
                scenario=config.scenario.name,
                duration_seconds=duration,
                error_message=str(e),
                matrix_params=matrix_params,
            )

    def _parse_duration(self, duration: str) -> int:
        """解析时长字符串为秒数"""
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
        return total_seconds if total_seconds > 0 else 300  # 默认 5 分钟

    def _parse_k6_output(self, stdout: str, json_path: Path) -> dict[str, Any]:
        """解析 k6 输出获取指标"""
        metrics = {}

        # 尝试从 JSON 输出解析
        if json_path.exists():
            try:
                with open(json_path) as f:
                    for line in f:
                        try:
                            data = json.loads(line)
                            if data.get("type") == "Point":
                                metric_name = data.get("metric")
                                value = data.get("data", {}).get("value")
                                if metric_name and value is not None:
                                    if metric_name not in metrics:
                                        metrics[metric_name] = []
                                    metrics[metric_name].append(value)
                        except json.JSONDecodeError:
                            continue
            except Exception:
                pass

        # 从标准输出解析摘要
        for line in stdout.split("\n"):
            line = line.strip()
            if "avg=" in line or "p(95)=" in line:
                # 解析类似: milvus_search_latency...avg=23.45ms min=12.3ms p(50)=21.2ms
                parts = line.split()
                if parts:
                    metric_name = parts[0].strip(".")
                    for part in parts[1:]:
                        if "=" in part:
                            key, value = part.split("=", 1)
                            # 移除单位
                            value = value.rstrip("ms").rstrip("s").rstrip("%")
                            try:
                                metrics[f"{metric_name}_{key}"] = float(value)
                            except ValueError:
                                pass

        return metrics


class K8sRunner(BaseRunner):
    """Kubernetes 运行器"""

    def __init__(
        self,
        namespace: str = "k6-tests",
        kubeconfig: str | None = None,
    ):
        super().__init__()
        self.namespace = namespace
        self.kubeconfig = kubeconfig

    def run(
        self,
        script_path: Path,
        config: Config,
        matrix_params: dict[str, Any] | None = None,
    ) -> TestResult:
        """在 K8s 中运行测试"""
        from .generator import ScriptGenerator

        start_time = datetime.now()

        try:
            # 生成 K8s 资源
            generator = ScriptGenerator()
            output_dir = script_path.parent

            k8s_resources = generator.generate_k8s_resources(
                config.scenario.type, config, output_dir, script_path.name
            )

            # 创建 ConfigMap
            configmap_cmd = [
                "kubectl",
                "create",
                "configmap",
                f"milvus-test-{config.scenario.type}",
                f"--from-file={script_path.name}={script_path}",
                "-n",
                self.namespace,
                "--dry-run=client",
                "-o",
                "yaml",
            ]

            if self.kubeconfig:
                configmap_cmd.extend(["--kubeconfig", self.kubeconfig])

            # 应用 ConfigMap
            configmap_result = subprocess.run(
                configmap_cmd,
                capture_output=True,
                text=True,
            )

            apply_cmd = ["kubectl", "apply", "-f", "-", "-n", self.namespace]
            if self.kubeconfig:
                apply_cmd.extend(["--kubeconfig", self.kubeconfig])

            subprocess.run(
                apply_cmd,
                input=configmap_result.stdout,
                capture_output=True,
                text=True,
            )

            # 应用 TestRun
            apply_testrun_cmd = [
                "kubectl",
                "apply",
                "-f",
                str(k8s_resources["testrun"]),
                "-n",
                self.namespace,
            ]
            if self.kubeconfig:
                apply_testrun_cmd.extend(["--kubeconfig", self.kubeconfig])

            subprocess.run(apply_testrun_cmd, capture_output=True, text=True)

            # 等待完成并获取结果
            testrun_name = f"milvus-{config.scenario.type}-"
            self.console.print(f"[bold blue]TestRun created:[/bold blue] {testrun_name}")
            self.console.print(
                "[yellow]Note: Use 'kubectl get testrun -n {self.namespace}' to check status[/yellow]"
            )

            duration = (datetime.now() - start_time).total_seconds()

            return TestResult(
                status="success",
                scenario=config.scenario.name,
                duration_seconds=duration,
                metrics={},
                thresholds_passed=True,
                output_file=k8s_resources["testrun"],
                matrix_params=matrix_params,
            )

        except Exception as e:
            duration = (datetime.now() - start_time).total_seconds()
            return TestResult(
                status="error",
                scenario=config.scenario.name,
                duration_seconds=duration,
                error_message=str(e),
                matrix_params=matrix_params,
            )


def get_runner(mode: str, **kwargs) -> BaseRunner:
    """获取运行器实例"""
    if mode == "local":
        return LocalRunner(k6_binary=kwargs.get("k6_binary", "k6"))
    elif mode == "k8s":
        return K8sRunner(
            namespace=kwargs.get("namespace", "k6-tests"),
            kubeconfig=kwargs.get("kubeconfig"),
        )
    else:
        raise ValueError(f"Unknown runner mode: {mode}")
