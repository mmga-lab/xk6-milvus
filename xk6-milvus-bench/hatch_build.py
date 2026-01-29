"""Hatch 构建钩子 - 在构建时编译 k6 二进制

这个钩子会在构建 wheel 时自动编译带有 xk6-milvus 扩展的 k6 二进制。
"""

from __future__ import annotations

import os
import platform
import shutil
import subprocess
import sys
from pathlib import Path
from typing import Any

from hatchling.builders.hooks.plugin.interface import BuildHookInterface

# 平台映射
PLATFORM_MAP = {
    ("Darwin", "arm64"): "darwin-arm64",
    ("Darwin", "x86_64"): "darwin-amd64",
    ("Linux", "x86_64"): "linux-amd64",
    ("Linux", "aarch64"): "linux-arm64",
    ("Windows", "AMD64"): "windows-amd64",
}

# 要包含的扩展
K6_EXTENSIONS = [
    "github.com/mmga-lab/xk6-milvus@latest",
    "github.com/mmga-lab/xk6-parquet@latest",
    "github.com/grafana/xk6-faker@latest",
]


class K6BuildHook(BuildHookInterface):
    """构建钩子：在构建 wheel 时编译 k6 二进制"""

    PLUGIN_NAME = "k6-builder"

    def initialize(self, version: str, build_data: dict[str, Any]) -> None:
        """初始化钩子 - 在构建开始时调用"""
        # 检查是否需要构建 k6
        if not self._should_build_k6():
            print("Skipping k6 build (BUILD_K6=0 or missing dependencies)")
            return

        # 获取目标平台
        target_platform = self._get_target_platform()
        if not target_platform:
            print(f"Unsupported platform: {platform.system()}-{platform.machine()}")
            return

        # 构建 k6
        bin_dir = Path(self.root) / "src" / "xk6_milvus_bench" / "bin"
        bin_dir.mkdir(parents=True, exist_ok=True)

        binary_name = f"k6-{target_platform}"
        if platform.system() == "Windows":
            binary_name += ".exe"

        output_path = bin_dir / binary_name

        # 如果二进制已存在且是最新的，跳过构建
        if output_path.exists() and not os.environ.get("FORCE_K6_BUILD"):
            print(f"k6 binary already exists: {output_path}")
            return

        try:
            self._build_k6(output_path)
            # 添加到构建数据，确保包含在 wheel 中
            if "force_include" not in build_data:
                build_data["force_include"] = {}
            build_data["force_include"][str(output_path)] = f"xk6_milvus_bench/bin/{binary_name}"
        except Exception as e:
            print(f"Warning: Failed to build k6: {e}")
            print("The package will be built without bundled k6 binary.")

    def _should_build_k6(self) -> bool:
        """检查是否应该构建 k6"""
        # 环境变量控制
        if os.environ.get("BUILD_K6", "1") == "0":
            return False

        # 检查 xk6 是否可用
        if not shutil.which("xk6"):
            # 尝试检查 go 是否可用
            if not shutil.which("go"):
                return False
            # 安装 xk6
            try:
                subprocess.run(
                    ["go", "install", "go.k6.io/xk6/cmd/xk6@latest"],
                    check=True,
                    capture_output=True,
                )
            except subprocess.CalledProcessError:
                return False

        return True

    def _get_target_platform(self) -> str | None:
        """获取目标平台标识"""
        system = platform.system()
        machine = platform.machine()
        return PLATFORM_MAP.get((system, machine))

    def _build_k6(self, output_path: Path) -> None:
        """构建 k6 二进制"""
        print(f"Building k6 with extensions: {K6_EXTENSIONS}")

        cmd = ["xk6", "build", "--output", str(output_path)]
        for ext in K6_EXTENSIONS:
            cmd.extend(["--with", ext])

        print(f"Running: {' '.join(cmd)}")

        result = subprocess.run(
            cmd,
            capture_output=True,
            text=True,
        )

        if result.returncode != 0:
            raise RuntimeError(f"xk6 build failed: {result.stderr}")

        # 确保二进制可执行
        output_path.chmod(0o755)
        print(f"k6 built successfully: {output_path}")
