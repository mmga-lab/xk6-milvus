"""CLI 入口模块"""

from pathlib import Path
from typing import Annotated

import typer
from rich.console import Console
from rich.panel import Panel

from . import __version__
from .config import load_config
from .generator import ScriptGenerator, expand_matrix, parse_matrix_arg
from .reporter import Reporter
from .runner import TestResult, get_runner

app = typer.Typer(
    name="xk6-milvus-bench",
    help="Milvus 性能/稳定性测试框架 - 基于 xk6-milvus",
    no_args_is_help=True,
)
console = Console()

# 可用场景列表
AVAILABLE_SCENARIOS = [
    "search",
    "insert",
    "mixed",
    "concurrent",
    "soak",
    "recall",
    "benchmark",  # 标准数据集 benchmark
]

# Dataset 子命令组
dataset_app = typer.Typer(help="管理 benchmark 数据集")


def version_callback(value: bool):
    if value:
        console.print(f"xk6-milvus-bench version {__version__}")
        raise typer.Exit()


@app.callback()
def main(
    version: Annotated[
        bool,
        typer.Option("--version", "-v", callback=version_callback, is_eager=True),
    ] = False,
):
    """Milvus 性能/稳定性测试框架"""
    pass


@app.command()
def run(
    scenario: Annotated[str, typer.Argument(help="测试场景名称")],
    config: Annotated[
        Path | None,
        typer.Option("-c", "--config", help="配置文件路径"),
    ] = None,
    mode: Annotated[
        str,
        typer.Option("--mode", help="运行模式: local/k8s"),
    ] = "local",
    host: Annotated[
        str | None,
        typer.Option("--host", help="Milvus 地址"),
    ] = None,
    token: Annotated[
        str | None,
        typer.Option("--token", help="Milvus 认证 token"),
    ] = None,
    vus: Annotated[
        int | None,
        typer.Option("--vus", help="虚拟用户数"),
    ] = None,
    duration: Annotated[
        str | None,
        typer.Option("--duration", help="测试时长 (如: 5m, 1h)"),
    ] = None,
    matrix: Annotated[
        list[str] | None,
        typer.Option("--matrix", help="测试矩阵参数 (如: topk=[10,50,100])"),
    ] = None,
    parallelism: Annotated[
        int | None,
        typer.Option("--parallelism", help="K8s 并行度"),
    ] = None,
    namespace: Annotated[
        str | None,
        typer.Option("--namespace", help="K8s 命名空间"),
    ] = None,
    output: Annotated[
        Path | None,
        typer.Option("-o", "--output", help="输出目录"),
    ] = None,
    k6_binary: Annotated[
        str | None,
        typer.Option("--k6", help="k6 二进制文件路径"),
    ] = None,
    dry_run: Annotated[
        bool,
        typer.Option("--dry-run", help="仅生成脚本，不运行"),
    ] = False,
):
    """运行性能测试"""
    # 验证场景
    if scenario not in AVAILABLE_SCENARIOS:
        console.print(f"[red]Error: Unknown scenario '{scenario}'[/red]")
        console.print(f"Available scenarios: {', '.join(AVAILABLE_SCENARIOS)}")
        raise typer.Exit(1)

    # 打印 banner
    console.print(
        Panel.fit(
            f"[bold blue]xk6-milvus-bench[/bold blue] v{__version__}\n"
            f"Scenario: [cyan]{scenario}[/cyan]",
            border_style="blue",
        )
    )

    # 加载配置
    try:
        cfg = load_config(scenario, config)
    except Exception as e:
        console.print(f"[red]Error loading config: {e}[/red]")
        raise typer.Exit(1) from None

    # 应用 CLI 覆盖
    if host:
        cfg.milvus.host = host
    if token:
        cfg.milvus.token = token
    if vus:
        cfg.execution.vus = vus
    if duration:
        cfg.execution.duration = duration
    if parallelism:
        cfg.kubernetes.parallelism = parallelism
    if namespace:
        cfg.kubernetes.namespace = namespace

    # 设置输出目录
    output_dir = output or Path("./output")
    output_dir.mkdir(parents=True, exist_ok=True)

    # 打印配置摘要
    console.print()
    console.print("[bold]Configuration:[/bold]")
    console.print(f"  Target: [cyan]{cfg.milvus.host}[/cyan]")
    console.print(f"  VUs: [cyan]{cfg.execution.vus}[/cyan]")
    console.print(f"  Duration: [cyan]{cfg.execution.duration}[/cyan]")
    console.print(f"  Mode: [cyan]{mode}[/cyan]")

    # 处理测试矩阵
    matrix_params_list = [{}]
    if matrix:
        parsed_matrix = parse_matrix_arg(matrix)
        if parsed_matrix:
            matrix_params_list = expand_matrix(parsed_matrix)
            console.print(f"  Matrix: [cyan]{len(matrix_params_list)} combinations[/cyan]")

    console.print()

    # 生成脚本
    generator = ScriptGenerator()
    results: list[TestResult] = []
    reporter = Reporter()

    for i, matrix_params in enumerate(matrix_params_list):
        if matrix_params:
            params_str = ", ".join(f"{k}={v}" for k, v in matrix_params.items())
            console.print(f"[bold]Running combination {i + 1}/{len(matrix_params_list)}:[/bold] {params_str}")

        # 生成脚本
        script_path = generator.generate_script(scenario, cfg, output_dir, matrix_params or None)
        console.print(f"  Generated script: [dim]{script_path}[/dim]")

        if dry_run:
            console.print("  [yellow]Dry run - skipping execution[/yellow]")
            continue

        # 获取运行器
        runner = get_runner(
            mode,
            k6_binary=k6_binary,
            namespace=cfg.kubernetes.namespace,
        )

        # 运行测试
        with console.status("[bold green]Running test..."):
            result = runner.run(script_path, cfg, matrix_params or None)

        results.append(result)
        reporter.print_result(result)

    # 打印矩阵摘要
    if len(results) > 1:
        reporter.print_matrix_summary(results)

    # 生成报告
    if results and not dry_run:
        # JSON 报告
        json_report = reporter.save_json_report(results, output_dir / "report.json")
        console.print(f"JSON report: [dim]{json_report}[/dim]")

        # HTML 报告
        html_report = reporter.generate_html_report(results, output_dir / "report.html")
        console.print(f"HTML report: [dim]{html_report}[/dim]")

    # 返回状态
    if all(r.status == "success" and r.thresholds_passed for r in results):
        console.print("\n[bold green]✅ All tests passed[/bold green]")
    else:
        console.print("\n[bold red]❌ Some tests failed[/bold red]")
        raise typer.Exit(1)


@app.command()
def generate(
    scenario: Annotated[str, typer.Argument(help="测试场景名称")],
    config: Annotated[
        Path | None,
        typer.Option("-c", "--config", help="配置文件路径"),
    ] = None,
    output: Annotated[
        Path,
        typer.Option("-o", "--output", help="输出目录"),
    ] = Path("./generated"),
    include_k8s: Annotated[
        bool,
        typer.Option("--k8s", help="生成 K8s 资源"),
    ] = True,
):
    """生成测试脚本（不运行）"""
    # 验证场景
    if scenario not in AVAILABLE_SCENARIOS:
        console.print(f"[red]Error: Unknown scenario '{scenario}'[/red]")
        console.print(f"Available scenarios: {', '.join(AVAILABLE_SCENARIOS)}")
        raise typer.Exit(1)

    # 加载配置
    cfg = load_config(scenario, config)

    # 生成文件
    generator = ScriptGenerator()
    files = generator.generate_all(scenario, cfg, output, include_k8s)

    console.print("[bold green]Generated files:[/bold green]")
    for name, path in files.items():
        console.print(f"  {name}: [dim]{path}[/dim]")


@app.command("list")
def list_scenarios():
    """列出可用的测试场景"""
    console.print("[bold]Available scenarios:[/bold]")
    console.print()

    scenarios_info = {
        "search": "向量搜索性能测试",
        "insert": "数据插入性能测试",
        "mixed": "混合负载测试 (读写混合)",
        "concurrent": "高并发测试",
        "soak": "稳定性/持久性测试",
        "recall": "召回率验证测试",
        "benchmark": "标准数据集 benchmark（需要先下载数据集）",
    }

    for name, desc in scenarios_info.items():
        console.print(f"  [cyan]{name:12}[/cyan] - {desc}")


@app.command()
def report(
    result_dir: Annotated[Path, typer.Argument(help="结果目录")],
    output: Annotated[
        Path,
        typer.Option("-o", "--output", help="报告输出路径"),
    ] = Path("./report.html"),
    format: Annotated[
        str,
        typer.Option("--format", "-f", help="报告格式: html/json"),
    ] = "html",
):
    """从结果目录生成报告"""
    import json

    # 加载结果
    json_files = list(result_dir.glob("*.json"))
    if not json_files:
        console.print(f"[red]No JSON result files found in {result_dir}[/red]")
        raise typer.Exit(1)

    results = []
    for json_file in json_files:
        try:
            with open(json_file) as f:
                data = json.load(f)
                if "results" in data:
                    # 从报告文件加载
                    for r in data["results"]:
                        results.append(TestResult(**r))
                else:
                    # 从单个结果加载
                    results.append(TestResult(**data))
        except Exception as e:
            console.print(f"[yellow]Warning: Failed to load {json_file}: {e}[/yellow]")

    if not results:
        console.print("[red]No valid results found[/red]")
        raise typer.Exit(1)

    reporter = Reporter()

    if format == "html":
        report_path = reporter.generate_html_report(results, output)
    else:
        report_path = reporter.save_json_report(results, output)

    console.print(f"[green]Report generated:[/green] {report_path}")


@app.command()
def deploy(
    namespace: Annotated[
        str,
        typer.Option("--namespace", "-n", help="K8s 命名空间"),
    ] = "k6-tests",
    install_operator: Annotated[
        bool,
        typer.Option("--install-operator", help="安装 k6 operator"),
    ] = False,
):
    """部署到 Kubernetes"""
    import subprocess

    if install_operator:
        console.print("[bold]Installing k6 operator...[/bold]")
        try:
            # 使用 Helm 安装 k6 operator
            subprocess.run(
                [
                    "helm",
                    "repo",
                    "add",
                    "grafana",
                    "https://grafana.github.io/helm-charts",
                ],
                check=True,
            )
            subprocess.run(["helm", "repo", "update"], check=True)
            subprocess.run(
                [
                    "helm",
                    "install",
                    "k6-operator",
                    "grafana/k6-operator",
                    "-n",
                    namespace,
                    "--create-namespace",
                ],
                check=True,
            )
            console.print("[green]k6 operator installed successfully[/green]")
        except subprocess.CalledProcessError as e:
            console.print(f"[red]Failed to install k6 operator: {e}[/red]")
            raise typer.Exit(1) from None
    else:
        console.print("[yellow]Skipping operator installation. Use --install-operator to install.[/yellow]")

    # 创建命名空间
    console.print(f"[bold]Creating namespace {namespace}...[/bold]")
    try:
        subprocess.run(
            ["kubectl", "create", "namespace", namespace],
            capture_output=True,
        )
        console.print(f"[green]Namespace {namespace} ready[/green]")
    except subprocess.CalledProcessError:
        console.print(f"[yellow]Namespace {namespace} may already exist[/yellow]")

    console.print()
    console.print("[bold green]Deployment complete![/bold green]")
    console.print()
    console.print("Next steps:")
    console.print("  1. Generate test resources: xk6-milvus-bench generate search -o ./output")
    console.print(f"  2. Run in K8s: xk6-milvus-bench run search --mode k8s -n {namespace}")


# ============================================================================
# Dataset 子命令
# ============================================================================


@dataset_app.command("list")
def dataset_list(
    downloaded_only: Annotated[
        bool,
        typer.Option("--downloaded", "-d", help="仅显示已下载的数据集"),
    ] = False,
):
    """列出可用的 benchmark 数据集"""
    from rich.table import Table

    from .datasets import DATASETS, list_downloaded_datasets

    if downloaded_only:
        # 仅显示已下载的
        downloaded = list_downloaded_datasets()
        if not downloaded:
            console.print("[yellow]No datasets downloaded yet.[/yellow]")
            console.print("Use 'xk6-milvus-bench dataset download <name>' to download.")
            return

        table = Table(title="Downloaded Datasets")
        table.add_column("Name", style="cyan")
        table.add_column("Dimension", justify="right")
        table.add_column("Metric")
        table.add_column("Train Size", justify="right")
        table.add_column("Test Size", justify="right")

        for ds in downloaded:
            table.add_row(
                ds.get("name", "unknown"),
                str(ds.get("dimension", "-")),
                ds.get("metric", "-"),
                f"{ds.get('train_size', 0):,}",
                f"{ds.get('test_size', 0):,}",
            )

        console.print(table)
    else:
        # 显示所有可用数据集
        table = Table(title="Available Benchmark Datasets")
        table.add_column("Name", style="cyan")
        table.add_column("Description")
        table.add_column("Dimension", justify="right")
        table.add_column("Metric")
        table.add_column("Train Size", justify="right")
        table.add_column("Status")

        downloaded = {ds["name"] for ds in list_downloaded_datasets()}

        for name, info in DATASETS.items():
            status = "[green]✓ Ready[/green]" if name in downloaded else "[dim]Not downloaded[/dim]"
            table.add_row(
                name,
                info["description"],
                str(info["dimension"]),
                info["metric"],
                f"{info['train_size']:,}",
                status,
            )

        console.print(table)
        console.print()
        console.print("[dim]Use 'xk6-milvus-bench dataset download <name>' to download a dataset.[/dim]")


@dataset_app.command("download")
def dataset_download(
    name: Annotated[str, typer.Argument(help="数据集名称")],
    force: Annotated[
        bool,
        typer.Option("--force", "-f", help="强制重新下载"),
    ] = False,
):
    """下载并准备 benchmark 数据集"""
    from .datasets import DATASETS, download_dataset, is_dataset_ready

    if name not in DATASETS:
        console.print(f"[red]Unknown dataset: {name}[/red]")
        console.print(f"Available datasets: {', '.join(DATASETS.keys())}")
        raise typer.Exit(1)

    if is_dataset_ready(name) and not force:
        console.print(f"[green]Dataset '{name}' is already ready.[/green]")
        console.print("Use --force to re-download.")
        return

    console.print(f"[bold]Downloading dataset: {name}[/bold]")

    try:
        output_dir = download_dataset(name, force=force)
        console.print(f"[green]Dataset ready at: {output_dir}[/green]")
    except ImportError as e:
        console.print(f"[red]Missing dependency: {e}[/red]")
        console.print("Install with: pip install h5py pyarrow")
        raise typer.Exit(1) from None
    except Exception as e:
        console.print(f"[red]Failed to download dataset: {e}[/red]")
        raise typer.Exit(1) from None


@dataset_app.command("delete")
def dataset_delete(
    name: Annotated[str, typer.Argument(help="数据集名称")],
    confirm: Annotated[
        bool,
        typer.Option("--yes", "-y", help="跳过确认"),
    ] = False,
):
    """删除已下载的数据集"""
    from .datasets import delete_dataset, is_dataset_ready

    if not is_dataset_ready(name):
        console.print(f"[yellow]Dataset '{name}' is not downloaded.[/yellow]")
        return

    if not confirm:
        confirm = typer.confirm(f"Are you sure you want to delete dataset '{name}'?")
        if not confirm:
            console.print("Cancelled.")
            return

    if delete_dataset(name):
        console.print(f"[green]Dataset '{name}' deleted.[/green]")
    else:
        console.print(f"[red]Failed to delete dataset '{name}'.[/red]")


@dataset_app.command("info")
def dataset_info(
    name: Annotated[str, typer.Argument(help="数据集名称")],
):
    """显示数据集详细信息"""
    from .datasets import DATASETS, get_parquet_paths, is_dataset_ready, load_dataset_metadata

    if name not in DATASETS:
        console.print(f"[red]Unknown dataset: {name}[/red]")
        raise typer.Exit(1)

    info = DATASETS[name]

    console.print(f"[bold]Dataset: {name}[/bold]")
    console.print()
    console.print(f"  Description: {info['description']}")
    console.print(f"  Dimension:   {info['dimension']}")
    console.print(f"  Metric:      {info['metric']}")
    console.print(f"  Train Size:  {info['train_size']:,}")
    console.print(f"  Test Size:   {info['test_size']:,}")
    console.print(f"  Source:      {info['url']}")

    if is_dataset_ready(name):
        console.print()
        console.print("[green]Status: Downloaded and ready[/green]")

        try:
            load_dataset_metadata(name)  # Verify metadata exists
            paths = get_parquet_paths(name)
            console.print()
            console.print("[bold]Local files:[/bold]")
            for key, path in paths.items():
                if path.exists():
                    size_mb = path.stat().st_size / 1024 / 1024
                    console.print(f"  {key}: {path} ({size_mb:.1f} MB)")
        except Exception:
            pass
    else:
        console.print()
        console.print("[yellow]Status: Not downloaded[/yellow]")
        console.print(f"Run: xk6-milvus-bench dataset download {name}")


# 注册 dataset 子命令组
app.add_typer(dataset_app, name="dataset")


# ============================================================================
# k6 子命令
# ============================================================================

k6_app = typer.Typer(help="管理 k6 二进制")


@k6_app.command("info")
def k6_info():
    """显示 k6 二进制信息"""
    from .k6 import find_k6_binary, get_bundled_k6_path, verify_k6_binary

    # 检查打包的 k6
    bundled = get_bundled_k6_path()
    if bundled:
        console.print(f"[green]Bundled k6:[/green] {bundled}")
    else:
        console.print("[yellow]Bundled k6:[/yellow] Not found")

    # 查找可用的 k6
    try:
        k6_path = find_k6_binary()
        console.print(f"[green]Active k6:[/green] {k6_path}")

        # 验证
        info = verify_k6_binary(k6_path)
        if info["valid"]:
            console.print(f"[dim]Version:[/dim] {info['version']}")
            if info["has_milvus_extension"]:
                console.print("[green]xk6-milvus extension: ✓[/green]")
            else:
                console.print("[yellow]xk6-milvus extension: Not detected[/yellow]")
        else:
            console.print(f"[red]Validation failed:[/red] {info.get('error', 'Unknown error')}")
    except FileNotFoundError as e:
        console.print(f"[red]No k6 found:[/red] {e}")


@k6_app.command("build")
def k6_build(
    output: Annotated[
        Path | None,
        typer.Option("-o", "--output", help="输出目录"),
    ] = None,
    install: Annotated[
        bool,
        typer.Option("--install", "-i", help="安装到包目录"),
    ] = False,
):
    """构建 k6 二进制（带 xk6-milvus 扩展）"""
    from .k6 import BIN_DIR, build_k6, get_k6_binary_name

    if output is None:
        output = Path.cwd()

    try:
        k6_path = build_k6(output)
        console.print(f"[green]k6 built successfully:[/green] {k6_path}")

        if install:
            # 安装到包目录
            BIN_DIR.mkdir(parents=True, exist_ok=True)
            binary_name = get_k6_binary_name()
            dest = BIN_DIR / binary_name

            import shutil
            shutil.copy2(k6_path, dest)
            dest.chmod(0o755)
            console.print(f"[green]Installed to:[/green] {dest}")

    except Exception as e:
        console.print(f"[red]Build failed:[/red] {e}")
        raise typer.Exit(1) from None


@k6_app.command("path")
def k6_path():
    """显示 k6 二进制路径"""
    from .k6 import find_k6_binary

    try:
        path = find_k6_binary()
        console.print(path)
    except FileNotFoundError as e:
        console.print(f"[red]Error:[/red] {e}", err=True)
        raise typer.Exit(1) from None


# 注册 k6 子命令组
app.add_typer(k6_app, name="k6")


if __name__ == "__main__":
    app()
