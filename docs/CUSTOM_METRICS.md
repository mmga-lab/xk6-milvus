# 自定义 Metrics 指南

本文档介绍如何在 xk6-milvus 扩展中使用和创建自定义指标（metrics）。

## 概述

k6 提供了多种方式来收集和报告自定义指标：

1. **JavaScript 层面的指标** - 在 k6 脚本中使用 `Counter`, `Trend`, `Rate`, `Gauge`
2. **Go 扩展层面的指标** - 在 Go 代码中直接创建和推送指标
3. **混合方式** - 结合两种方法

## 1. JavaScript 层面的自定义指标（推荐方式）

这是最简单和推荐的方式，在 k6 脚本中直接创建指标：

### 示例：基础指标追踪

```javascript
import milvus from 'k6/x/milvus';
import { check } from 'k6';
import { Counter, Trend, Rate, Gauge } from 'k6/metrics';

// 创建自定义指标
const milvusInserts = new Counter('milvus_inserts_total');
const milvusSearches = new Counter('milvus_searches_total');
const milvusLatency = new Trend('milvus_operation_duration');
const milvusErrors = new Rate('milvus_error_rate');
const milvusVectorDimension = new Gauge('milvus_vector_dimension');

export default function() {
    const client = milvus.client('localhost:19530');
    
    // 记录向量维度
    const dimension = 128;
    milvusVectorDimension.add(dimension);
    
    // 测量插入操作
    const start = new Date();
    const vectors = [Array(dimension).fill(0).map(() => Math.random())];
    
    try {
        const ids = client.insertVectors('test_collection', vectors);
        const duration = new Date() - start;
        
        // 记录成功指标
        milvusInserts.add(vectors.length);
        milvusLatency.add(duration);
        milvusErrors.add(0); // 成功操作
        
        check(ids, {
            'insert successful': (ids) => ids && ids.length > 0,
        });
        
    } catch (error) {
        // 记录错误指标
        milvusErrors.add(1);
        console.error('Insert failed:', error);
    }
    
    client.close();
}
```

### 指标类型说明

- **Counter**: 累计计数器，只能增加
- **Trend**: 趋势指标，用于统计时间、大小等数值的分布
- **Rate**: 比率指标，用于计算错误率、成功率等
- **Gauge**: 仪表指标，用于记录当前值

## 2. 高级指标追踪

### 带标签的指标

```javascript
import { Counter } from 'k6/metrics';

const milvusOperations = new Counter('milvus_operations_total');

export default function() {
    const client = milvus.client('localhost:19530');
    
    // 为不同操作类型添加标签
    milvusOperations.add(1, { operation: 'insert', collection: 'test_collection' });
    milvusOperations.add(1, { operation: 'search', collection: 'test_collection' });
}
```

### 复杂性能分析

```javascript
import { Trend, Counter } from 'k6/metrics';

const insertLatency = new Trend('milvus_insert_latency');
const searchLatency = new Trend('milvus_search_latency');
const vectorsProcessed = new Counter('milvus_vectors_processed');

export default function() {
    const client = milvus.client('localhost:19530');
    
    // 批量插入性能测试
    const batchSizes = [10, 50, 100];
    
    for (const batchSize of batchSizes) {
        const vectors = [];
        for (let i = 0; i < batchSize; i++) {
            vectors.push(Array(128).fill(0).map(() => Math.random()));
        }
        
        const insertStart = new Date();
        const ids = client.insertVectors('perf_test', vectors);
        const insertDuration = new Date() - insertStart;
        
        // 记录按批次大小分类的指标
        insertLatency.add(insertDuration, { batch_size: batchSize.toString() });
        vectorsProcessed.add(batchSize, { operation: 'insert' });
    }
}
```

## 3. 设置阈值（Thresholds）

在 k6 选项中设置性能阈值：

```javascript
export const options = {
    vus: 10,
    duration: '30s',
    thresholds: {
        // 自定义指标的阈值
        'milvus_operation_duration': [
            'p(95)<500',      // 95% 的操作应在 500ms 内完成
            'p(99)<1000',     // 99% 的操作应在 1s 内完成
        ],
        'milvus_error_rate': [
            'rate<0.05',      // 错误率应低于 5%
        ],
        'milvus_inserts_total': [
            'count>100',      // 应执行超过 100 次插入
        ],
    },
};
```

## 4. Go 扩展层面的指标（高级）

虽然当前实现还未完全集成，但这里展示概念：

### 在 Go 代码中创建指标

```go
import (
    "go.k6.io/k6/metrics"
    "go.k6.io/k6/js/modules"
)

// 在扩展中定义指标
var (
    insertDuration = metrics.NewMetric("milvus_insert_duration", metrics.Trend, metrics.Time)
    insertCount    = metrics.NewMetric("milvus_insert_count", metrics.Counter)
)

func (c *Client) Insert(collectionName string, data map[string]interface{}) ([]int64, error) {
    start := time.Now()
    
    // 执行实际操作
    result, err := c.performInsert(collectionName, data)
    
    // 获取 VU 上下文并推送指标
    if vu := modules.GetVU(); vu != nil {
        duration := time.Since(start)
        
        vu.State().Samples <- metrics.ConnectedSamples{
            Samples: []metrics.Sample{
                {
                    TimeSeries: metrics.TimeSeries{
                        Metric: insertDuration,
                        Tags:   metrics.NewSampleTags().Set("collection", collectionName),
                    },
                    Value: metrics.D(duration),
                    Time:  time.Now(),
                },
            },
        }
    }
    
    return result, err
}
```

## 5. 指标输出和可视化

### 输出到不同格式

```bash
# 输出到 JSON
./k6 run --out json=metrics.json script.js

# 输出到 InfluxDB
./k6 run --out influxdb=http://localhost:8086/mydb script.js

# 输出到 Prometheus
./k6 run --out experimental-prometheus-rw script.js
```

### 实时监控

```bash
# 使用 k6 的实时输出
./k6 run --out json script.js | jq '.metrics | select(.metric == "milvus_insert_latency")'
```

## 6. 最佳实践

### 1. 指标命名约定
- 使用小写和下划线：`milvus_insert_duration`
- 包含单位：`milvus_insert_duration_ms`
- 使用明确的前缀：`milvus_` 前缀表示 Milvus 相关指标

### 2. 合理使用标签
```javascript
// 好的做法
milvusOperations.add(1, { 
    operation: 'search', 
    collection: 'products',
    index_type: 'hnsw'
});

// 避免高基数标签
// milvusOperations.add(1, { user_id: userId }); // 可能有数百万个不同值
```

### 3. 性能考虑
- 不要在每次操作中都创建新指标对象
- 批量记录指标而不是单个记录
- 合理设置采样率对于高频操作

### 4. 错误处理
```javascript
export default function() {
    const client = milvus.client('localhost:19530');
    
    try {
        // 执行操作
        const result = client.someOperation();
        successMetric.add(1);
    } catch (error) {
        errorMetric.add(1, { error_type: error.name });
        console.error('Operation failed:', error);
    } finally {
        totalOperations.add(1);
        client.close();
    }
}
```

## 7. 示例脚本

参考 `examples/advanced/metrics-example.js` 查看完整的指标使用示例。

## 总结

1. **推荐使用 JavaScript 层面的指标** - 简单、灵活、功能完整
2. **合理命名和使用标签** - 便于后续分析和监控
3. **设置有意义的阈值** - 确保性能测试的有效性
4. **选择合适的输出格式** - 根据监控系统选择相应的输出格式

通过这些方法，你可以全面监控 Milvus 的性能表现，识别瓶颈，并优化系统配置。