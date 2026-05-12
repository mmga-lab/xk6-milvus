"""生成器模块测试"""

import tempfile
from pathlib import Path

from xk6_milvus_bench.config import load_config
from xk6_milvus_bench.generator import (
    ScriptGenerator,
    expand_matrix,
    parse_matrix_arg,
)


class TestExpandMatrix:
    """矩阵展开测试"""

    def test_expand_empty_matrix(self):
        """测试空矩阵展开"""
        result = expand_matrix({})
        assert result == [{}]

    def test_expand_single_param(self):
        """测试单参数矩阵展开"""
        matrix = {"topk": [10, 50, 100]}
        result = expand_matrix(matrix)
        assert len(result) == 3
        assert {"topk": 10} in result
        assert {"topk": 50} in result
        assert {"topk": 100} in result

    def test_expand_multiple_params(self):
        """测试多参数矩阵展开"""
        matrix = {"topk": [10, 50], "vus": [5, 10]}
        result = expand_matrix(matrix)
        assert len(result) == 4  # 2 * 2 = 4
        assert {"topk": 10, "vus": 5} in result
        assert {"topk": 10, "vus": 10} in result
        assert {"topk": 50, "vus": 5} in result
        assert {"topk": 50, "vus": 10} in result

    def test_expand_three_params(self):
        """测试三参数矩阵展开"""
        matrix = {"a": [1, 2], "b": [3, 4], "c": [5, 6]}
        result = expand_matrix(matrix)
        assert len(result) == 8  # 2 * 2 * 2 = 8


class TestParseMatrixArg:
    """矩阵参数解析测试"""

    def test_parse_single_param(self):
        """测试解析单个参数"""
        args = ["topk=[10,50,100]"]
        result = parse_matrix_arg(args)
        assert result == {"topk": [10, 50, 100]}

    def test_parse_multiple_params(self):
        """测试解析多个参数"""
        args = ["topk=[10,50]", "vus=[5,10,20]"]
        result = parse_matrix_arg(args)
        assert result == {"topk": [10, 50], "vus": [5, 10, 20]}

    def test_parse_float_values(self):
        """测试解析浮点数值"""
        args = ["threshold=[0.9,0.95,0.99]"]
        result = parse_matrix_arg(args)
        assert result == {"threshold": [0.9, 0.95, 0.99]}

    def test_parse_string_values(self):
        """测试解析字符串值"""
        args = ["metric=[L2,IP]"]
        result = parse_matrix_arg(args)
        assert result == {"metric": ["L2", "IP"]}

    def test_parse_single_value(self):
        """测试解析单值参数"""
        args = ["topk=10"]
        result = parse_matrix_arg(args)
        assert result == {"topk": [10]}

    def test_parse_invalid_format(self):
        """测试解析无效格式"""
        args = ["invalid"]
        result = parse_matrix_arg(args)
        assert result == {}


class TestScriptGenerator:
    """脚本生成器测试"""

    def test_generate_script(self):
        """测试生成脚本"""
        config = load_config("search", None)

        with tempfile.TemporaryDirectory() as tmpdir:
            output_dir = Path(tmpdir)
            generator = ScriptGenerator()

            script_path = generator.generate_script("search", config, output_dir)

            assert script_path.exists()
            assert script_path.suffix == ".js"

            content = script_path.read_text()
            assert "import milvus from 'k6/x/milvus'" in content
            assert "export const options" in content
            assert "export function setup()" in content
            assert "export default function" in content

    def test_generate_script_with_matrix_params(self):
        """测试生成带矩阵参数的脚本"""
        config = load_config("search", None)

        with tempfile.TemporaryDirectory() as tmpdir:
            output_dir = Path(tmpdir)
            generator = ScriptGenerator()

            matrix_params = {"topk": 50, "vus": 20}
            script_path = generator.generate_script(
                "search", config, output_dir, matrix_params
            )

            assert script_path.exists()
            assert "topk50" in script_path.name
            assert "vus20" in script_path.name

    def test_generate_struct_array_script(self):
        """测试生成 Struct Array benchmark 脚本"""
        config = load_config("struct-array", None)

        with tempfile.TemporaryDirectory() as tmpdir:
            output_dir = Path(tmpdir)
            generator = ScriptGenerator()

            script_path = generator.generate_script("struct-array", config, output_dir)

            assert script_path.exists()
            content = script_path.read_text()
            assert "structA[embedding]" in content
            assert "MAX_SIM_COSINE" in content
            assert "element_filter(structA" in content
            assert "MATCH_ANY(structA" in content
            assert "MATCH_LEAST(structA" in content
            assert "array_contains(structA[color]" in content
            assert "array_contains_any(structA[category]" in content
            assert "array_length(structA[color])" in content
            assert "structA[0][color] in" in content
            assert "groupByField: 'id'" in content

    def test_generate_k8s_resources(self):
        """测试生成 K8s 资源"""
        config = load_config("search", None)

        with tempfile.TemporaryDirectory() as tmpdir:
            output_dir = Path(tmpdir)
            generator = ScriptGenerator()

            resources = generator.generate_k8s_resources(
                "search", config, output_dir, "search.js"
            )

            assert "testrun" in resources
            assert "configmap" in resources
            assert resources["testrun"].exists()
            assert resources["configmap"].exists()

            testrun_content = resources["testrun"].read_text()
            assert "apiVersion: k6.io/v1alpha1" in testrun_content
            assert "kind: TestRun" in testrun_content

    def test_generate_all(self):
        """测试生成所有资源"""
        config = load_config("search", None)

        with tempfile.TemporaryDirectory() as tmpdir:
            output_dir = Path(tmpdir)
            generator = ScriptGenerator()

            files = generator.generate_all("search", config, output_dir)

            assert "script" in files
            assert "testrun" in files
            assert "configmap" in files
            assert all(f.exists() for f in files.values())

    def test_generate_all_without_k8s(self):
        """测试仅生成脚本（不含 K8s 资源）"""
        config = load_config("search", None)

        with tempfile.TemporaryDirectory() as tmpdir:
            output_dir = Path(tmpdir)
            generator = ScriptGenerator()

            files = generator.generate_all(
                "search", config, output_dir, include_k8s=False
            )

            assert "script" in files
            assert "testrun" not in files
            assert "configmap" not in files
