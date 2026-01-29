"""Benchmark 数据集管理模块

支持标准向量数据库 benchmark 数据集的下载、转换和管理。
数据集将转换为 Parquet 格式，供 k6 通过 xk6-parquet 扩展读取。

支持的数据集：
- sift-128-euclidean (SIFT1M): 128维，1M向量
- gist-960-euclidean (GIST1M): 960维，1M向量
- glove-25-angular: 25维，1.2M向量
- glove-100-angular: 100维，1.2M向量
- fashion-mnist-784-euclidean: 784维，60K向量
- cohere-medium-22-angular: 768维，1M向量

注意：下载和转换数据集需要安装额外依赖：
    pip install xk6-milvus-bench[datasets]
    # 或
    pip install h5py pyarrow numpy
"""

from __future__ import annotations

import shutil
import struct
import tempfile
from pathlib import Path
from typing import TYPE_CHECKING

# numpy/pyarrow 只在实际下载转换时需要，不影响列出数据集
if TYPE_CHECKING:
    import numpy as np

# 数据集注册表
DATASETS = {
    "sift-128-euclidean": {
        "description": "SIFT1M - 128维 SIFT 特征向量",
        "dimension": 128,
        "metric": "L2",
        "train_size": 1_000_000,
        "test_size": 10_000,
        "url": "http://ann-benchmarks.com/sift-128-euclidean.hdf5",
        "format": "hdf5",
    },
    "gist-960-euclidean": {
        "description": "GIST1M - 960维 GIST 特征向量",
        "dimension": 960,
        "metric": "L2",
        "train_size": 1_000_000,
        "test_size": 1_000,
        "url": "http://ann-benchmarks.com/gist-960-euclidean.hdf5",
        "format": "hdf5",
    },
    "glove-25-angular": {
        "description": "GloVe-25 - 25维词向量",
        "dimension": 25,
        "metric": "COSINE",
        "train_size": 1_183_514,
        "test_size": 10_000,
        "url": "http://ann-benchmarks.com/glove-25-angular.hdf5",
        "format": "hdf5",
    },
    "glove-100-angular": {
        "description": "GloVe-100 - 100维词向量",
        "dimension": 100,
        "metric": "COSINE",
        "train_size": 1_183_514,
        "test_size": 10_000,
        "url": "http://ann-benchmarks.com/glove-100-angular.hdf5",
        "format": "hdf5",
    },
    "fashion-mnist-784-euclidean": {
        "description": "Fashion-MNIST - 784维图像特征",
        "dimension": 784,
        "metric": "L2",
        "train_size": 60_000,
        "test_size": 10_000,
        "url": "http://ann-benchmarks.com/fashion-mnist-784-euclidean.hdf5",
        "format": "hdf5",
    },
    "mnist-784-euclidean": {
        "description": "MNIST - 784维手写数字",
        "dimension": 784,
        "metric": "L2",
        "train_size": 60_000,
        "test_size": 10_000,
        "url": "http://ann-benchmarks.com/mnist-784-euclidean.hdf5",
        "format": "hdf5",
    },
}

# 默认数据目录
DEFAULT_DATA_DIR = Path.home() / ".xk6-milvus-bench" / "datasets"


def get_data_dir() -> Path:
    """获取数据目录"""
    data_dir = DEFAULT_DATA_DIR
    data_dir.mkdir(parents=True, exist_ok=True)
    return data_dir


def list_datasets() -> dict[str, dict]:
    """列出所有可用数据集"""
    return DATASETS.copy()


def get_dataset_info(name: str) -> dict | None:
    """获取数据集信息"""
    return DATASETS.get(name)


def get_dataset_path(name: str) -> Path:
    """获取数据集存储路径"""
    return get_data_dir() / name


def is_dataset_ready(name: str) -> bool:
    """检查数据集是否已准备好（转换为 parquet）"""
    dataset_dir = get_dataset_path(name)
    required_files = ["train.parquet", "test.parquet", "neighbors.parquet", "metadata.json"]
    return all((dataset_dir / f).exists() for f in required_files)


def download_file(url: str, dest: Path, show_progress: bool = True) -> None:
    """下载文件"""
    import urllib.request

    if show_progress:
        print(f"Downloading {url}...")

    # 简单下载，实际使用可以添加进度条
    urllib.request.urlretrieve(url, dest)

    if show_progress:
        print(f"Downloaded to {dest}")


def read_hdf5_dataset(path: Path) -> dict:
    """读取 HDF5 格式数据集（ann-benchmarks 格式）"""
    try:
        import h5py
        import numpy as np
    except ImportError as e:
        raise ImportError(
            "需要安装 h5py 和 numpy: pip install xk6-milvus-bench[datasets]"
        ) from e

    data = {}
    with h5py.File(path, "r") as f:
        # ann-benchmarks 格式
        if "train" in f:
            data["train"] = np.array(f["train"])
        if "test" in f:
            data["test"] = np.array(f["test"])
        if "neighbors" in f:
            data["neighbors"] = np.array(f["neighbors"])
        if "distances" in f:
            data["distances"] = np.array(f["distances"])

    return data


def read_fvecs(path: Path):
    """读取 fvecs 格式向量文件"""
    try:
        import numpy as np
    except ImportError as e:
        raise ImportError(
            "需要安装 numpy: pip install xk6-milvus-bench[datasets]"
        ) from e

    with open(path, "rb") as f:
        # 读取第一个向量的维度
        d = struct.unpack("i", f.read(4))[0]
        f.seek(0)

        # 计算向量数量
        file_size = path.stat().st_size
        vector_size = 4 + d * 4  # dim(int) + d floats
        n = file_size // vector_size

        # 读取所有向量
        vectors = np.zeros((n, d), dtype=np.float32)
        for i in range(n):
            d_check = struct.unpack("i", f.read(4))[0]
            assert d_check == d, f"Dimension mismatch: {d_check} != {d}"
            vectors[i] = struct.unpack(f"{d}f", f.read(d * 4))

    return vectors


def read_ivecs(path: Path):
    """读取 ivecs 格式整数向量文件（用于 ground truth）"""
    try:
        import numpy as np
    except ImportError as e:
        raise ImportError(
            "需要安装 numpy: pip install xk6-milvus-bench[datasets]"
        ) from e

    with open(path, "rb") as f:
        # 读取第一个向量的维度
        d = struct.unpack("i", f.read(4))[0]
        f.seek(0)

        # 计算向量数量
        file_size = path.stat().st_size
        vector_size = 4 + d * 4  # dim(int) + d ints
        n = file_size // vector_size

        # 读取所有向量
        vectors = np.zeros((n, d), dtype=np.int32)
        for i in range(n):
            d_check = struct.unpack("i", f.read(4))[0]
            assert d_check == d, f"Dimension mismatch: {d_check} != {d}"
            vectors[i] = struct.unpack(f"{d}i", f.read(d * 4))

    return vectors


def convert_to_parquet(
    data: dict[str, np.ndarray],
    output_dir: Path,
    dataset_info: dict,
) -> None:
    """将数据集转换为 Parquet 格式"""
    try:
        import pyarrow as pa
        import pyarrow.parquet as pq
    except ImportError as e:
        raise ImportError("需要安装 pyarrow: pip install pyarrow") from e

    output_dir.mkdir(parents=True, exist_ok=True)

    # 转换训练数据（用于构建索引）
    if "train" in data:
        train_vectors = data["train"]
        # 将向量存储为 list 类型，每行一个向量
        train_table = pa.table({
            "id": pa.array(range(len(train_vectors)), type=pa.int64()),
            "vector": pa.array([v.tolist() for v in train_vectors], type=pa.list_(pa.float32())),
        })
        pq.write_table(train_table, output_dir / "train.parquet")
        print(f"  Created train.parquet: {len(train_vectors)} vectors")

    # 转换测试数据（查询向量）
    if "test" in data:
        test_vectors = data["test"]
        test_table = pa.table({
            "id": pa.array(range(len(test_vectors)), type=pa.int64()),
            "vector": pa.array([v.tolist() for v in test_vectors], type=pa.list_(pa.float32())),
        })
        pq.write_table(test_table, output_dir / "test.parquet")
        print(f"  Created test.parquet: {len(test_vectors)} vectors")

    # 转换 ground truth（neighbors）
    if "neighbors" in data:
        neighbors = data["neighbors"]
        neighbors_table = pa.table({
            "query_id": pa.array(range(len(neighbors)), type=pa.int64()),
            "neighbors": pa.array([n.tolist() for n in neighbors], type=pa.list_(pa.int64())),
        })
        pq.write_table(neighbors_table, output_dir / "neighbors.parquet")
        print(f"  Created neighbors.parquet: {len(neighbors)} queries")

    # 转换 distances（可选）
    if "distances" in data:
        distances = data["distances"]
        distances_table = pa.table({
            "query_id": pa.array(range(len(distances)), type=pa.int64()),
            "distances": pa.array([d.tolist() for d in distances], type=pa.list_(pa.float32())),
        })
        pq.write_table(distances_table, output_dir / "distances.parquet")
        print(f"  Created distances.parquet: {len(distances)} queries")

    # 创建元数据文件
    import json
    metadata = {
        "name": dataset_info.get("name", "unknown"),
        "dimension": dataset_info["dimension"],
        "metric": dataset_info["metric"],
        "train_size": len(data.get("train", [])),
        "test_size": len(data.get("test", [])),
        "neighbors_k": len(data["neighbors"][0]) if "neighbors" in data else 0,
    }
    with open(output_dir / "metadata.json", "w") as f:
        json.dump(metadata, f, indent=2)
    print("  Created metadata.json")


def download_dataset(name: str, force: bool = False) -> Path:
    """下载并准备数据集

    Args:
        name: 数据集名称
        force: 是否强制重新下载

    Returns:
        数据集目录路径
    """
    if name not in DATASETS:
        available = ", ".join(DATASETS.keys())
        raise ValueError(f"Unknown dataset: {name}. Available: {available}")

    dataset_info = DATASETS[name]
    dataset_info["name"] = name
    output_dir = get_dataset_path(name)

    # 检查是否已存在
    if is_dataset_ready(name) and not force:
        print(f"Dataset {name} already prepared at {output_dir}")
        return output_dir

    print(f"Preparing dataset: {name}")
    print(f"  Description: {dataset_info['description']}")
    print(f"  Dimension: {dataset_info['dimension']}")
    print(f"  Metric: {dataset_info['metric']}")

    # 创建临时目录下载
    with tempfile.TemporaryDirectory() as tmpdir:
        tmpdir = Path(tmpdir)

        # 下载数据集
        url = dataset_info["url"]
        filename = url.split("/")[-1]
        download_path = tmpdir / filename

        download_file(url, download_path)

        # 读取数据
        fmt = dataset_info["format"]
        if fmt == "hdf5":
            data = read_hdf5_dataset(download_path)
        else:
            raise ValueError(f"Unsupported format: {fmt}")

        # 转换为 parquet
        print("Converting to Parquet format...")
        convert_to_parquet(data, output_dir, dataset_info)

    print(f"Dataset ready at: {output_dir}")
    return output_dir


def load_dataset_metadata(name: str) -> dict:
    """加载数据集元数据"""
    import json

    dataset_dir = get_dataset_path(name)
    metadata_path = dataset_dir / "metadata.json"

    if not metadata_path.exists():
        raise FileNotFoundError(f"Dataset {name} not found. Run 'xk6-milvus-bench dataset download {name}' first.")

    with open(metadata_path) as f:
        return json.load(f)


def delete_dataset(name: str) -> bool:
    """删除数据集"""
    dataset_dir = get_dataset_path(name)
    if dataset_dir.exists():
        shutil.rmtree(dataset_dir)
        return True
    return False


def list_downloaded_datasets() -> list[dict]:
    """列出已下载的数据集"""
    data_dir = get_data_dir()
    datasets = []

    for dataset_dir in data_dir.iterdir():
        if dataset_dir.is_dir():
            metadata_path = dataset_dir / "metadata.json"
            if metadata_path.exists():
                import json
                with open(metadata_path) as f:
                    metadata = json.load(f)
                    metadata["path"] = str(dataset_dir)
                    datasets.append(metadata)

    return datasets


def get_parquet_paths(name: str) -> dict[str, Path]:
    """获取数据集的 parquet 文件路径"""
    dataset_dir = get_dataset_path(name)

    if not is_dataset_ready(name):
        raise FileNotFoundError(f"Dataset {name} not ready. Run download first.")

    return {
        "train": dataset_dir / "train.parquet",
        "test": dataset_dir / "test.parquet",
        "neighbors": dataset_dir / "neighbors.parquet",
        "metadata": dataset_dir / "metadata.json",
    }
