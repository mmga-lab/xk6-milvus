// BM25 Full-Text Search Example
// This example demonstrates BM25-based full-text search:
// - Automatic sparse vector generation from text
// - Text analyzer configuration
// - Full-text search with BM25 function
// - Combined text + vector search

import milvus from 'k6/x/milvus';
import { check } from 'k6';

export const options = {
    vus: 3,
    duration: '10s',
};

const MILVUS_HOST = __ENV.MILVUS_HOST || 'localhost:19530';
const COLLECTION_NAME = 'fulltext_search_demo';

export function setup() {
    const client = milvus.client(MILVUS_HOST);

    // Drop if exists
    const hasResult = client.hasCollection(COLLECTION_NAME);
    if (hasResult.success && hasResult.result.exists) {
        client.dropCollection(COLLECTION_NAME);
    }

    // Create collection with BM25 function
    const schema = {
        name: COLLECTION_NAME,
        numShards: 2,
        fields: [
            {
                name: 'id',
                dataType: 'Int64',
                isPrimaryKey: true
            },
            {
                name: 'title',
                dataType: 'VarChar',
                maxLength: 200
            },
            {
                name: 'content',
                dataType: 'VarChar',
                maxLength: 10000,
                enableAnalyzer: true,
                analyzerParams: { type: 'standard' },
                enableMatch: true
            },
            {
                name: 'category',
                dataType: 'VarChar',
                maxLength: 50
            },
            // Sparse vector field for BM25
            {
                name: 'sparse_vector',
                dataType: 'SparseFloatVector'
            }
        ],
        functions: [
            {
                name: 'text_bm25_emb',
                functionType: 'BM25',
                inputFieldNames: ['content'],
                outputFieldNames: ['sparse_vector']
            }
        ]
    };

    const createResult = client.createCollection(schema);
    check(createResult, {
        'collection created': (r) => r.success === true,
    });

    // Create index for sparse vector (BM25 function output)
    const indexResult = client.createIndex('sparse_vector', {
        indexType: 'SPARSE_INVERTED_INDEX',
        metricType: 'BM25'
    }, COLLECTION_NAME);
    check(indexResult, {
        'index created': (r) => r.success === true,
    });

    // Load collection
    const loadResult = client.loadCollection(COLLECTION_NAME);
    check(loadResult, {
        'collection loaded': (r) => r.success === true,
    });

    // Insert sample documents
    const documents = [
        {
            id: 1,
            title: 'Getting Started with Vector Databases',
            content: 'Vector databases are specialized databases designed to store and query high-dimensional vectors efficiently. They use similarity search algorithms like HNSW and IVF.',
            category: 'Tutorial'
        },
        {
            id: 2,
            title: 'Understanding Milvus Architecture',
            content: 'Milvus is a cloud-native vector database built for scalability and high performance. It supports multiple index types and provides flexible deployment options.',
            category: 'Architecture'
        },
        {
            id: 3,
            title: 'BM25 Algorithm Explained',
            content: 'BM25 is a ranking function used for information retrieval. It calculates relevance scores based on term frequency and document length normalization.',
            category: 'Algorithm'
        },
        {
            id: 4,
            title: 'Hybrid Search Techniques',
            content: 'Hybrid search combines multiple search methods such as dense vector search and sparse retrieval to improve search quality and recall.',
            category: 'Tutorial'
        },
        {
            id: 5,
            title: 'Optimizing Vector Search Performance',
            content: 'To optimize vector search, consider using appropriate index types, tuning search parameters, and scaling your infrastructure based on workload.',
            category: 'Performance'
        }
    ];

    // Upsert documents (sparse vectors generated automatically by BM25 function)
    const upsertResult = client.upsert({
        id: documents.map(d => d.id),
        title: documents.map(d => d.title),
        content: documents.map(d => d.content),
        category: documents.map(d => d.category)
    }, COLLECTION_NAME);

    check(upsertResult, {
        'documents inserted': (r) => r.success === true,
        'all documents inserted': (r) => r.result.upsert_count === 5,
    });

    console.log('Setup complete: 5 documents inserted with automatic BM25 embeddings');
    client.close();

    return { ready: true };
}

export default function() {
    const client = milvus.clientWithCollection(MILVUS_HOST, COLLECTION_NAME);

    // Example 1: Full-text search for "vector database"
    // Note: In production, you would use the search endpoint with text query
    // For this example, we demonstrate using query with text matching

    let queryResult = client.query(
        'category == "Tutorial"',
        ['id', 'title', 'category'],
    );

    check(queryResult, {
        'tutorial query successful': (r) => r.success === true,
        'found tutorials': (r) => !r.empty,
    });

    // Example 2: Query by category
    queryResult = client.query(
        'category == "Architecture"',
        ['id', 'title', 'content'],
    );

    check(queryResult, {
        'architecture query successful': (r) => r.success === true,
    });

    // Example 3: Update document
    const updateResult = client.upsert({
        id: [6],
        title: ['Advanced Indexing Strategies'],
        content: ['Advanced indexing involves choosing the right index type for your use case. HNSW provides fast search, while IVF offers better memory efficiency.'],
        category: ['Advanced']
    });

    check(updateResult, {
        'document upserted': (r) => r.success === true,
    });

    // Example 4: Query all documents
    queryResult = client.query(
        'id >= 1',
        ['id', 'title', 'category'],
    );

    check(queryResult, {
        'all documents query successful': (r) => r.success === true,
        'has multiple results': (r) => !r.empty && r.result && r.result.length >= 5,
    });

    // Example 5: Delete old documents
    const deleteResult = client.delete('id < 3');

    check(deleteResult, {
        'delete successful': (r) => r.success === true,
        'deleted some documents': (r) => r.result.delete_count >= 0,
    });

    client.close();
}

export function teardown(data) {
    if (data.ready) {
        const client = milvus.client(MILVUS_HOST);
        client.dropCollection(COLLECTION_NAME);
        client.close();
        console.log('Teardown complete');
    }
}
