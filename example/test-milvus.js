import milvus from 'k6/x/milvus';
import { check } from 'k6';

export const options = {
    vus: 10,
    duration: '30s',
};

const MILVUS_HOST = __ENV.MILVUS_HOST || 'localhost:19530';
const COLLECTION_NAME = 'test_collection';
const VECTOR_DIM = 128;
const INSERT_BATCH_SIZE = 100;
const SEARCH_TOP_K = 10;

function generateRandomVectors(count, dim) {
    const vectors = [];
    for (let i = 0; i < count; i++) {
        const vector = [];
        for (let j = 0; j < dim; j++) {
            vector.push(Math.random());
        }
        vectors.push(vector);
    }
    return vectors;
}

export function setup() {
    const client = milvus.client(MILVUS_HOST);
    
    const hasCollection = client.hasCollection(COLLECTION_NAME);
    if (hasCollection) {
        client.dropCollection(COLLECTION_NAME);
    }
    
    client.createCollectionSimple(COLLECTION_NAME, VECTOR_DIM);
    
    client.createIndexSimple(COLLECTION_NAME, "vector");
    
    client.loadCollection(COLLECTION_NAME);
    
    client.close();
    
    return { collectionCreated: true };
}

export default function() {
    const client = milvus.client(MILVUS_HOST);
    
    const insertVectors = generateRandomVectors(INSERT_BATCH_SIZE, VECTOR_DIM);
    const insertResult = client.insertVectors(COLLECTION_NAME, insertVectors);
    check(insertResult, {
        'insert successful': (r) => r && r.length === INSERT_BATCH_SIZE,
    });
    
    const searchVectors = generateRandomVectors(1, VECTOR_DIM);
    const searchResult = client.searchSimple(COLLECTION_NAME, searchVectors, SEARCH_TOP_K);
    check(searchResult, {
        'search successful': (r) => r && r.length > 0,
        'search returns expected results': (r) => r && r.length <= SEARCH_TOP_K,
    });
    
    client.close();
}

export function teardown(data) {
    if (data.collectionCreated) {
        const client = milvus.client(MILVUS_HOST);
        client.dropCollection(COLLECTION_NAME);
        client.close();
    }
}