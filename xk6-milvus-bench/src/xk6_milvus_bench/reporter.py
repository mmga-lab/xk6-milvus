"""报告生成模块"""

import json
from datetime import datetime
from pathlib import Path
from typing import Any

from jinja2 import Environment, FileSystemLoader
from rich.console import Console
from rich.table import Table

from .config import get_templates_dir
from .runner import TestResult


class Reporter:
    """测试报告生成器"""

    def __init__(self, templates_dir: Path | None = None):
        self.templates_dir = templates_dir or get_templates_dir()
        self.console = Console()
        self.env = Environment(
            loader=FileSystemLoader(str(self.templates_dir)),
            trim_blocks=True,
            lstrip_blocks=True,
        )

    def print_result(self, result: TestResult) -> None:
        """打印单个测试结果"""
        status_color = {
            "success": "green",
            "failed": "red",
            "error": "red",
        }.get(result.status, "yellow")

        self.console.print()
        self.console.rule(f"[bold]Results: {result.scenario}[/bold]")
        self.console.print()

        # 状态
        status_icon = "✅" if result.status == "success" else "❌"
        self.console.print(f"  Status: [{status_color}]{status_icon} {result.status}[/]")
        self.console.print(f"  Duration: {result.duration_seconds:.2f}s")

        if result.matrix_params:
            params_str = ", ".join(f"{k}={v}" for k, v in result.matrix_params.items())
            self.console.print(f"  Parameters: {params_str}")

        # 阈值
        if result.thresholds_passed:
            self.console.print("  Thresholds: [green]✅ All passed[/]")
        else:
            self.console.print("  Thresholds: [red]❌ Some failed[/]")

        # 错误信息
        if result.error_message:
            self.console.print(f"  [red]Error: {result.error_message}[/]")

        # 指标
        if result.metrics:
            self.console.print()
            self.console.print("  [bold]Metrics:[/bold]")

            # 按类别组织指标
            latency_metrics = {}
            throughput_metrics = {}
            quality_metrics = {}
            other_metrics = {}

            for key, value in result.metrics.items():
                if isinstance(value, list):
                    # 计算统计值
                    if value:
                        avg = sum(value) / len(value)
                        p95_idx = int(len(value) * 0.95)
                        sorted_values = sorted(value)
                        p95 = sorted_values[p95_idx] if p95_idx < len(value) else sorted_values[-1]
                        value = {"avg": avg, "p95": p95, "count": len(value)}

                if "latency" in key.lower() or "duration" in key.lower():
                    latency_metrics[key] = value
                elif "ops" in key.lower() or "qps" in key.lower() or "rate" in key.lower():
                    throughput_metrics[key] = value
                elif "recall" in key.lower() or "error" in key.lower():
                    quality_metrics[key] = value
                else:
                    other_metrics[key] = value

            if latency_metrics:
                self.console.print("    [cyan]Latency:[/cyan]")
                for key, value in latency_metrics.items():
                    self._print_metric(key, value)

            if throughput_metrics:
                self.console.print("    [cyan]Throughput:[/cyan]")
                for key, value in throughput_metrics.items():
                    self._print_metric(key, value)

            if quality_metrics:
                self.console.print("    [cyan]Quality:[/cyan]")
                for key, value in quality_metrics.items():
                    self._print_metric(key, value)

            if other_metrics:
                self.console.print("    [cyan]Other:[/cyan]")
                for key, value in other_metrics.items():
                    self._print_metric(key, value)

        self.console.print()

    def _print_metric(self, key: str, value: Any) -> None:
        """打印单个指标"""
        if isinstance(value, dict):
            parts = [f"{k}={v:.2f}" if isinstance(v, float) else f"{k}={v}" for k, v in value.items()]
            self.console.print(f"      {key}: {', '.join(parts)}")
        elif isinstance(value, float):
            self.console.print(f"      {key}: {value:.2f}")
        else:
            self.console.print(f"      {key}: {value}")

    def print_matrix_summary(self, results: list[TestResult]) -> None:
        """打印矩阵测试摘要"""
        if not results:
            return

        self.console.print()
        self.console.rule("[bold]Matrix Results Summary[/bold]")
        self.console.print()

        # 创建表格
        table = Table(title=f"Matrix Results ({len(results)} combinations)")

        # 获取所有参数名
        param_names = set()
        for r in results:
            if r.matrix_params:
                param_names.update(r.matrix_params.keys())
        param_names = sorted(param_names)

        # 添加列
        for name in param_names:
            table.add_column(name, style="cyan")
        table.add_column("Status", style="bold")
        table.add_column("Duration", style="dim")
        table.add_column("Thresholds")

        # 添加行
        for r in results:
            row = []
            for name in param_names:
                value = r.matrix_params.get(name, "-") if r.matrix_params else "-"
                row.append(str(value))

            status_style = "green" if r.status == "success" else "red"
            row.append(f"[{status_style}]{r.status}[/]")
            row.append(f"{r.duration_seconds:.1f}s")

            if r.thresholds_passed:
                row.append("[green]✅[/]")
            else:
                row.append("[red]❌[/]")

            table.add_row(*row)

        self.console.print(table)

        # 统计
        success_count = sum(1 for r in results if r.status == "success")
        failed_count = len(results) - success_count
        passed_count = sum(1 for r in results if r.thresholds_passed)

        self.console.print()
        self.console.print(f"  Total: {len(results)} | Success: {success_count} | Failed: {failed_count}")
        self.console.print(f"  Thresholds Passed: {passed_count}/{len(results)}")

        # 找出最佳配置
        successful_results = [r for r in results if r.status == "success"]
        if successful_results:
            # 按某个指标排序（如 p95 延迟）
            best = min(
                successful_results,
                key=lambda r: r.metrics.get("milvus_search_latency_p95", float("inf")),
            )
            if best.matrix_params:
                params_str = ", ".join(f"{k}={v}" for k, v in best.matrix_params.items())
                self.console.print()
                self.console.print(f"  [bold green]Best Configuration:[/bold green] {params_str}")

        self.console.print()

    def generate_html_report(
        self,
        results: list[TestResult],
        output_path: Path,
        title: str = "Milvus Bench Report",
    ) -> Path:
        """生成 HTML 报告"""
        try:
            template = self.env.get_template("reports/summary.html.j2")
        except Exception:
            # 使用内置模板
            return self._generate_simple_html_report(results, output_path, title)

        context = {
            "title": title,
            "generated_at": datetime.now().isoformat(),
            "results": [r.to_dict() for r in results],
            "summary": {
                "total": len(results),
                "success": sum(1 for r in results if r.status == "success"),
                "failed": sum(1 for r in results if r.status != "success"),
                "thresholds_passed": sum(1 for r in results if r.thresholds_passed),
            },
        }

        content = template.render(**context)
        output_path.parent.mkdir(parents=True, exist_ok=True)
        output_path.write_text(content)

        return output_path

    def _generate_simple_html_report(
        self,
        results: list[TestResult],
        output_path: Path,
        title: str,
    ) -> Path:
        """生成简单 HTML 报告（当模板不可用时）"""
        html = f"""<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <title>{title}</title>
    <style>
        body {{ font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; margin: 40px; }}
        h1 {{ color: #333; }}
        table {{ border-collapse: collapse; width: 100%; margin-top: 20px; }}
        th, td {{ border: 1px solid #ddd; padding: 12px; text-align: left; }}
        th {{ background-color: #4CAF50; color: white; }}
        tr:nth-child(even) {{ background-color: #f2f2f2; }}
        .success {{ color: green; }}
        .failed {{ color: red; }}
        .summary {{ background-color: #f9f9f9; padding: 20px; border-radius: 8px; margin-bottom: 20px; }}
    </style>
</head>
<body>
    <h1>{title}</h1>
    <p>Generated at: {datetime.now().isoformat()}</p>

    <div class="summary">
        <h2>Summary</h2>
        <p>Total Tests: {len(results)}</p>
        <p>Success: {sum(1 for r in results if r.status == 'success')}</p>
        <p>Failed: {sum(1 for r in results if r.status != 'success')}</p>
        <p>Thresholds Passed: {sum(1 for r in results if r.thresholds_passed)}</p>
    </div>

    <h2>Results</h2>
    <table>
        <tr>
            <th>Scenario</th>
            <th>Status</th>
            <th>Duration</th>
            <th>Thresholds</th>
            <th>Parameters</th>
        </tr>
"""
        for r in results:
            status_class = "success" if r.status == "success" else "failed"
            params = ", ".join(f"{k}={v}" for k, v in (r.matrix_params or {}).items()) or "-"
            thresholds = "✅" if r.thresholds_passed else "❌"
            html += f"""        <tr>
            <td>{r.scenario}</td>
            <td class="{status_class}">{r.status}</td>
            <td>{r.duration_seconds:.2f}s</td>
            <td>{thresholds}</td>
            <td>{params}</td>
        </tr>
"""

        html += """    </table>
</body>
</html>
"""
        output_path.parent.mkdir(parents=True, exist_ok=True)
        output_path.write_text(html)
        return output_path

    def save_json_report(self, results: list[TestResult], output_path: Path) -> Path:
        """保存 JSON 报告"""
        report = {
            "generated_at": datetime.now().isoformat(),
            "results": [r.to_dict() for r in results],
            "summary": {
                "total": len(results),
                "success": sum(1 for r in results if r.status == "success"),
                "failed": sum(1 for r in results if r.status != "success"),
                "thresholds_passed": sum(1 for r in results if r.thresholds_passed),
            },
        }

        output_path.parent.mkdir(parents=True, exist_ok=True)
        with open(output_path, "w") as f:
            json.dump(report, f, indent=2)

        return output_path
