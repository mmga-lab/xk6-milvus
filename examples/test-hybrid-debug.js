import milvus from 'k6/x/milvus';
import { sleep } from 'k6';

export const options = {
    vus: 1,
    iterations: 1,
};

const MILVUS_HOST = __ENV.MILVUS_HOST || 'localhost:19530';

export default function() {
    const client = milvus.client(MILVUS_HOST);

    // Create collection
    const schema = {
        name: 'hybrid_debug',
        fields: [
            { name: 'id', dataType: 'Int64', isPrimaryKey: true, isAutoID: true },
            { name: 'dense', dataType: 'FloatVector', dimension: 4 },
            { name: 'sparse', dataType: 'FloatVector', dimension: 4 }
        ]
    };

    let result = client.createCollection(schema);
    console.log('Create:', result.success, result.error);

    result = client.createIndex('dense', { indexType: 'FLAT', metricType: 'L2' }, 'hybrid_debug');
    console.log('Index dense:', result.success, result.error);

    result = client.createIndex('sparse', { indexType: 'FLAT', metricType: 'IP' }, 'hybrid_debug');
    console.log('Index sparse:', result.success, result.error);

    result = client.loadCollection('hybrid_debug');
    console.log('Load:', result.success, result.error);

    result = client.insert({
        dense: [[0.1, 0.2, 0.3, 0.4], [0.5, 0.6, 0.7, 0.8]],
        sparse: [[0.9, 0.8, 0.7, 0.6], [0.5, 0.4, 0.3, 0.2]]
    }, 'hybrid_debug');
    console.log('Insert:', result.success, result.error, 'count:', result.result ? result.result.insert_count : 0);

    sleep(1);

    const hybridResult = client.hybridSearch(
        [
            {
                vectors: [[0.1, 0.2, 0.3, 0.4]],
                vectorField: 'dense',
                limit: 2,
                params: { metricType: 'L2' }
            },
            {
                vectors: [[0.9, 0.8, 0.7, 0.6]],
                vectorField: 'sparse',
                limit: 2,
                params: { metricType: 'IP' }
            }
        ],
        { type: 'rrf', params: { k: 60 } },
        2,
        ['id'],
        'hybrid_debug'
    );

    console.log('Hybrid search - success:', hybridResult.success);
    console.log('Hybrid search - empty:', hybridResult.empty);
    console.log('Hybrid search - error:', hybridResult.error);
    console.log('Hybrid search - result count:', hybridResult.result ? hybridResult.result.length : 0);
    if (hybridResult.result && hybridResult.result.length > 0) {
        console.log('First result:', JSON.stringify(hybridResult.result[0]));
    }

    client.dropCollection('hybrid_debug');
    client.close();
}
