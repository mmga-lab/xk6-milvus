"""k6 二进制管理模块

管理打包在 Python 包中的 k6 二进制文件。
"""

from __future__ import annotations

import platform
import shutil
import subprocess
from pathlib import Path

# k6 二进制存放目录
BIN_DIR = Path(__file__).parent / "bin"

# 平台映射
PLATFORM_MAP = {
    ("Darwin", "arm64"): "darwin-arm64",
    ("Darwin", "x86_64"): "darwin-amd64",
    ("Linux", "x86_64"): "linux-amd64",
    ("Linux", "aarch64"): "linux-arm64",
    ("Windows", "AMD64"): "windows-amd64",
}


def get_platform_suffix() -> str:
    """获取当前平台后缀"""
    system = platform.system()
    machine = platform.machine()

    suffix = PLATFORM_MAP.get((system, machine))
    if not suffix:
        raise RuntimeError(f"Unsupported platform: {system}-{machine}")

    return suffix


def get_k6_binary_name() -> str:
    """获取 k6 二进制文件名"""
    suffix = get_platform_suffix()
    if platform.system() == "Windows":
        return f"k6-{suffix}.exe"
    return f"k6-{suffix}"


def get_bundled_k6_path() -> Path | None:
    """获取打包的 k6 二进制路径

    Returns:
        k6 二进制路径，如果不存在则返回 None
    """
    binary_name = get_k6_binary_name()
    binary_path = BIN_DIR / binary_name

    if binary_path.exists():
        return binary_path

    # 尝试不带平台后缀的名称（开发模式）
    simple_path = BIN_DIR / "k6"
    if simple_path.exists():
        return simple_path

    return None


def find_k6_binary(user_specified: str | None = None) -> str:
    """查找 k6 二进制文件

    查找顺序：
    1. 用户指定的路径
    2. 打包在 Python 包中的二进制
    3. 当前目录的 k6
    4. xk6-milvus 项目根目录的 k6
    5. 系统 PATH 中的 k6

    Args:
        user_specified: 用户指定的 k6 路径

    Returns:
        k6 二进制路径

    Raises:
        FileNotFoundError: 找不到 k6 二进制
    """
    # 1. 用户指定
    if user_specified:
        path = Path(user_specified)
        if path.exists():
            return str(path.absolute())
        raise FileNotFoundError(f"Specified k6 binary not found: {user_specified}")

    # 2. 打包的二进制
    bundled = get_bundled_k6_path()
    if bundled:
        return str(bundled)

    # 3. 当前目录
    local_k6 = Path("./k6")
    if local_k6.exists() and local_k6.is_file():
        return str(local_k6.absolute())

    # 4. xk6-milvus 项目根目录 (xk6-milvus-bench 是子目录)
    project_k6 = Path(__file__).parent.parent.parent.parent / "k6"
    if project_k6.exists() and project_k6.is_file():
        return str(project_k6.absolute())

    # 5. 系统 PATH
    k6_in_path = shutil.which("k6")
    if k6_in_path:
        return k6_in_path

    raise FileNotFoundError(
        "k6 binary not found. Please either:\n"
        "  1. Install xk6-milvus-bench with bundled k6: pip install xk6-milvus-bench[k6]\n"
        "  2. Build k6 manually: xk6 build --with github.com/mmga-lab/xk6-milvus\n"
        "  3. Specify k6 path: --k6 /path/to/k6"
    )


def verify_k6_binary(k6_path: str) -> dict:
    """验证 k6 二进制并获取版本信息

    Args:
        k6_path: k6 二进制路径

    Returns:
        包含版本信息的字典
    """
    try:
        result = subprocess.run(
            [k6_path, "version"],
            capture_output=True,
            text=True,
            timeout=10,
        )

        version_output = result.stdout.strip()

        # 检查是否包含 xk6-milvus 扩展
        has_milvus = "xk6-milvus" in version_output or "milvus" in version_output.lower()

        return {
            "path": k6_path,
            "version": version_output,
            "has_milvus_extension": has_milvus,
            "valid": result.returncode == 0,
        }
    except Exception as e:
        return {
            "path": k6_path,
            "version": None,
            "has_milvus_extension": False,
            "valid": False,
            "error": str(e),
        }


def build_k6(
    output_dir: Path | None = None,
    extensions: list[str] | None = None,
) -> Path:
    """使用 xk6 构建 k6 二进制

    Args:
        output_dir: 输出目录，默认为当前目录
        extensions: 要包含的扩展列表

    Returns:
        构建的 k6 二进制路径
    """
    if output_dir is None:
        output_dir = Path.cwd()

    if extensions is None:
        extensions = [
            "github.com/mmga-lab/xk6-milvus@latest",
            "github.com/mmga-lab/xk6-parquet@latest",
            "github.com/grafana/xk6-faker@latest",
        ]

    # 检查 xk6 是否可用
    xk6_path = shutil.which("xk6")
    if not xk6_path:
        raise RuntimeError(
            "xk6 not found. Install it with: go install go.k6.io/xk6/cmd/xk6@latest"
        )

    # 构建命令
    output_name = "k6"
    if platform.system() == "Windows":
        output_name = "k6.exe"

    output_path = output_dir / output_name

    cmd = ["xk6", "build", "--output", str(output_path)]
    for ext in extensions:
        cmd.extend(["--with", ext])

    print(f"Building k6 with extensions: {extensions}")
    print(f"Command: {' '.join(cmd)}")

    result = subprocess.run(
        cmd,
        capture_output=True,
        text=True,
    )

    if result.returncode != 0:
        raise RuntimeError(f"Failed to build k6: {result.stderr}")

    print(f"k6 built successfully: {output_path}")
    return output_path
