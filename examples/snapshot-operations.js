/**
 * Snapshot Operations Example for xk6-milvus
 *
 * This example demonstrates how to use Milvus snapshot functionality
 * for backup and restore operations during stability testing.
 *
 * Usage:
 *   ./k6 run examples/snapshot-operations.js
 *
 * Environment:
 *   MILVUS_HOST - Milvus server address (default: localhost:19530)
 *
 * Note: Snapshot functionality requires Milvus version that supports snapshots.
 */

import milvus from "k6/x/milvus";
import { check, sleep } from "k6";

const MILVUS_HOST = __ENV.MILVUS_HOST || "localhost:19530";
const COLLECTION_NAME = `snapshot_test_${Date.now()}`;
const SNAPSHOT_NAME = `backup_${Date.now()}`;
const RESTORED_COLLECTION = `${COLLECTION_NAME}_restored`;
const DIMENSION = 128;

export const options = {
  vus: 1,
  iterations: 1,
  setupTimeout: "120s",
  teardownTimeout: "60s",
};

// Generate random vector
function generateVector(dim) {
  const vector = [];
  for (let i = 0; i < dim; i++) {
    vector.push(Math.random());
  }
  return vector;
}

// Generate test data
function generateTestData(count) {
  const ids = [];
  const titles = [];
  const prices = [];
  const vectors = [];

  for (let i = 0; i < count; i++) {
    ids.push(i);
    titles.push(`Product ${i}`);
    prices.push(Math.random() * 100);
    vectors.push(generateVector(DIMENSION));
  }

  return { id: ids, title: titles, price: prices, vector: vectors };
}

export function setup() {
  console.log(`Setting up test with Milvus at ${MILVUS_HOST}`);

  const client = milvus.client(MILVUS_HOST);

  // Create collection
  const schema = {
    name: COLLECTION_NAME,
    fields: [
      { name: "id", dataType: "Int64", isPrimaryKey: true },
      { name: "title", dataType: "VarChar", maxLength: 200 },
      { name: "price", dataType: "Float" },
      { name: "vector", dataType: "FloatVector", dimension: DIMENSION },
    ],
  };

  const createResult = client.createCollection(schema);
  check(createResult, {
    "collection created": (r) => r.success === true,
  });

  if (!createResult.success) {
    console.error(`Failed to create collection: ${createResult.error}`);
    client.close();
    return { error: createResult.error };
  }

  // Insert test data
  const data = generateTestData(100);
  const insertResult = client.insert(data, COLLECTION_NAME);
  check(insertResult, {
    "data inserted": (r) => r.success === true,
    "inserted 100 records": (r) =>
      r.result && r.result.insert_count === 100,
  });

  console.log(`Inserted ${insertResult.result?.insert_count || 0} records`);

  client.close();
  return { collectionName: COLLECTION_NAME };
}

export default function (setupData) {
  if (setupData.error) {
    console.error(`Setup failed: ${setupData.error}`);
    return;
  }

  const client = milvus.clientWithCollection(MILVUS_HOST, COLLECTION_NAME);

  // Step 1: Create a snapshot
  console.log("\n=== Creating Snapshot ===");
  const createSnapshotResult = client.createSnapshot(SNAPSHOT_NAME, null, {
    description: "Daily backup for stability testing",
  });

  check(createSnapshotResult, {
    "snapshot created": (r) => r.success === true,
    "snapshot creation fast": (r) => r.response_time_ms < 5000,
  });

  if (!createSnapshotResult.success) {
    console.error(`Failed to create snapshot: ${createSnapshotResult.error}`);
    client.close();
    return;
  }
  console.log(
    `Created snapshot '${SNAPSHOT_NAME}' in ${createSnapshotResult.response_time_ms}ms`,
  );

  // Step 2: List all snapshots
  console.log("\n=== Listing Snapshots ===");
  const listResult = client.listSnapshots({
    collectionName: COLLECTION_NAME,
  });

  check(listResult, {
    "list snapshots success": (r) => r.success === true,
    "has snapshots": (r) => !r.empty,
  });

  if (listResult.success) {
    console.log(`Found ${listResult.result.length} snapshot(s):`);
    listResult.result.forEach((snap) => console.log(`  - ${snap}`));
  }

  // Step 3: Describe snapshot details
  console.log("\n=== Describing Snapshot ===");
  const describeResult = client.describeSnapshot(SNAPSHOT_NAME);

  check(describeResult, {
    "describe snapshot success": (r) => r.success === true,
    "snapshot has correct name": (r) =>
      r.result && r.result.name === SNAPSHOT_NAME,
  });

  if (describeResult.success) {
    console.log("Snapshot details:");
    console.log(`  Name: ${describeResult.result.name}`);
    console.log(`  Description: ${describeResult.result.description}`);
    console.log(`  Collection: ${describeResult.result.collectionName}`);
    console.log(`  Create Time: ${describeResult.result.createTs}`);
  }

  // Step 4: Restore snapshot to a new collection
  console.log("\n=== Restoring Snapshot ===");
  const restoreResult = client.restoreSnapshot(SNAPSHOT_NAME, RESTORED_COLLECTION);

  check(restoreResult, {
    "restore initiated": (r) => r.success === true,
    "got job ID": (r) => r.result && r.result.jobId > 0,
  });

  if (!restoreResult.success) {
    console.error(`Failed to initiate restore: ${restoreResult.error}`);
    // Continue to cleanup
  } else {
    const jobId = restoreResult.result.jobId;
    console.log(`Restore job started with ID: ${jobId}`);

    // Step 5: Poll restore status
    console.log("\n=== Polling Restore Status ===");
    let attempts = 0;
    const maxAttempts = 30;

    while (attempts < maxAttempts) {
      const stateResult = client.getRestoreSnapshotState(jobId);

      if (!stateResult.success) {
        console.error(`Failed to get state: ${stateResult.error}`);
        break;
      }

      const state = stateResult.result.state;
      const progress = stateResult.result.progress;

      console.log(`  Attempt ${attempts + 1}: State=${state}, Progress=${progress}%`);

      if (state === "RestoreSnapshotCompleted") {
        console.log("Restore completed successfully!");
        check(stateResult, {
          "restore completed": (r) =>
            r.result.state === "RestoreSnapshotCompleted",
        });
        break;
      }

      if (state === "RestoreSnapshotFailed") {
        console.error(`Restore failed: ${stateResult.result.reason}`);
        break;
      }

      sleep(1);
      attempts++;
    }

    if (attempts >= maxAttempts) {
      console.warn("Restore polling timed out");
    }
  }

  // Step 6: List all restore jobs
  console.log("\n=== Listing Restore Jobs ===");
  const jobsResult = client.listRestoreSnapshotJobs();

  check(jobsResult, {
    "list jobs success": (r) => r.success === true,
  });

  if (jobsResult.success && !jobsResult.empty) {
    console.log(`Found ${jobsResult.result.length} restore job(s):`);
    jobsResult.result.forEach((job) => {
      console.log(
        `  - Job ${job.jobId}: ${job.snapshotName} -> ${job.collectionName} (${job.state})`,
      );
    });
  }

  // Step 7: Clean up - Drop snapshot
  console.log("\n=== Cleanup: Dropping Snapshot ===");
  const dropSnapshotResult = client.dropSnapshot(SNAPSHOT_NAME);

  check(dropSnapshotResult, {
    "snapshot dropped": (r) => r.success === true,
  });

  if (dropSnapshotResult.success) {
    console.log(`Dropped snapshot '${SNAPSHOT_NAME}'`);
  }

  client.close();
}

export function teardown(setupData) {
  if (setupData.error) {
    return;
  }

  console.log("\n=== Teardown: Cleaning up collections ===");

  const client = milvus.client(MILVUS_HOST);

  // Drop original collection
  const dropOriginal = client.dropCollection(COLLECTION_NAME);
  if (dropOriginal.success) {
    console.log(`Dropped collection '${COLLECTION_NAME}'`);
  }

  // Drop restored collection (if it exists)
  const dropRestored = client.dropCollection(RESTORED_COLLECTION);
  if (dropRestored.success) {
    console.log(`Dropped collection '${RESTORED_COLLECTION}'`);
  }

  client.close();
  console.log("Teardown complete");
}
