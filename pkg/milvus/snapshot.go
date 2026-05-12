package milvus

import (
	"fmt"
	"time"

	"github.com/milvus-io/milvus/client/v2/milvusclient"
)

// CreateSnapshot creates a snapshot for a collection
// Parameters:
//   - name: snapshot name
//   - collectionName: optional collection name (uses default if bound)
//   - options: optional map with "description" and "dbName" keys
func (c *Client) CreateSnapshot(name string, collectionName interface{}, options ...map[string]interface{}) interface{} {
	start := time.Now()

	// Handle collectionName which can be string or nil
	coll := c.resolveCollectionName(collectionName)
	if coll == "" {
		return toMap(&OperationResult{
			Success:      false,
			ResponseTime: float64(time.Since(start).Milliseconds()),
			Error:        ErrCollectionNameRequired.Error(),
		})
	}

	opt := milvusclient.NewCreateSnapshotOption(name, coll)

	// Parse optional parameters
	if len(options) > 0 && options[0] != nil {
		if desc, ok := options[0]["description"].(string); ok && desc != "" {
			opt = opt.WithDescription(desc)
		}
		if dbName, ok := options[0]["dbName"].(string); ok && dbName != "" {
			opt = opt.WithDbName(dbName)
		}
	}

	err := c.client.CreateSnapshot(c.context(), opt)
	if err != nil {
		return toMap(&OperationResult{
			Success:      false,
			ResponseTime: float64(time.Since(start).Milliseconds()),
			Error:        fmt.Sprintf("failed to create snapshot: %v", err),
		})
	}

	return toMap(&OperationResult{
		Success:      true,
		ResponseTime: float64(time.Since(start).Milliseconds()),
		Result: map[string]interface{}{
			"name": name,
		},
	})
}

// DropSnapshot drops a snapshot by name
// Parameters:
//   - name: snapshot name
//   - options: optional map with "collectionName" and "dbName" keys
func (c *Client) DropSnapshot(name string, options ...map[string]interface{}) interface{} {
	start := time.Now()

	collectionName := c.defaultCollection
	var dbName string
	if len(options) > 0 && options[0] != nil {
		if coll, ok := options[0]["collectionName"].(string); ok && coll != "" {
			collectionName = coll
		}
		if db, ok := options[0]["dbName"].(string); ok && db != "" {
			dbName = db
		}
	}
	if collectionName == "" {
		return toMap(&OperationResult{
			Success:      false,
			ResponseTime: float64(time.Since(start).Milliseconds()),
			Error:        ErrCollectionNameRequired.Error(),
		})
	}

	opt := milvusclient.NewDropSnapshotOption(name, collectionName)
	if dbName != "" {
		opt = opt.WithDbName(dbName)
	}

	err := c.client.DropSnapshot(c.context(), opt)
	if err != nil {
		return toMap(&OperationResult{
			Success:      false,
			ResponseTime: float64(time.Since(start).Milliseconds()),
			Error:        fmt.Sprintf("failed to drop snapshot: %v", err),
		})
	}

	return toMap(&OperationResult{
		Success:      true,
		ResponseTime: float64(time.Since(start).Milliseconds()),
	})
}

// ListSnapshots lists all snapshots
// Parameters:
//   - options: optional map with "collectionName" and "dbName" keys
func (c *Client) ListSnapshots(options ...map[string]interface{}) interface{} {
	start := time.Now()

	collectionName := c.defaultCollection
	var dbName string
	if len(options) > 0 && options[0] != nil {
		if coll, ok := options[0]["collectionName"].(string); ok && coll != "" {
			collectionName = coll
		}
		if db, ok := options[0]["dbName"].(string); ok && db != "" {
			dbName = db
		}
	}
	if collectionName == "" {
		return toMap(&OperationResult{
			Success:      false,
			ResponseTime: float64(time.Since(start).Milliseconds()),
			Error:        ErrCollectionNameRequired.Error(),
		})
	}

	opt := milvusclient.NewListSnapshotsOption(collectionName)
	if dbName != "" {
		opt = opt.WithDbName(dbName)
	}

	snapshots, err := c.client.ListSnapshots(c.context(), opt)
	if err != nil {
		return toMap(&OperationResult{
			Success:      false,
			ResponseTime: float64(time.Since(start).Milliseconds()),
			Error:        fmt.Sprintf("failed to list snapshots: %v", err),
		})
	}

	return toMap(&OperationResult{
		Success:      true,
		ResponseTime: float64(time.Since(start).Milliseconds()),
		Result:       snapshots,
		Empty:        len(snapshots) == 0,
	})
}

// DescribeSnapshot describes a snapshot by name
// Parameters:
//   - name: snapshot name
//   - options: optional map with "collectionName" and "dbName" keys
func (c *Client) DescribeSnapshot(name string, options ...map[string]interface{}) interface{} {
	start := time.Now()

	collectionName := c.defaultCollection
	var dbName string
	if len(options) > 0 && options[0] != nil {
		if coll, ok := options[0]["collectionName"].(string); ok && coll != "" {
			collectionName = coll
		}
		if db, ok := options[0]["dbName"].(string); ok && db != "" {
			dbName = db
		}
	}
	if collectionName == "" {
		return toMap(&OperationResult{
			Success:      false,
			ResponseTime: float64(time.Since(start).Milliseconds()),
			Error:        ErrCollectionNameRequired.Error(),
		})
	}

	opt := milvusclient.NewDescribeSnapshotOption(name, collectionName)
	if dbName != "" {
		opt = opt.WithDbName(dbName)
	}

	resp, err := c.client.DescribeSnapshot(c.context(), opt)
	if err != nil {
		return toMap(&OperationResult{
			Success:      false,
			ResponseTime: float64(time.Since(start).Milliseconds()),
			Error:        fmt.Sprintf("failed to describe snapshot: %v", err),
		})
	}

	return toMap(&OperationResult{
		Success:      true,
		ResponseTime: float64(time.Since(start).Milliseconds()),
		Result: map[string]interface{}{
			"name":           resp.GetName(),
			"description":    resp.GetDescription(),
			"collectionName": resp.GetCollectionName(),
			"partitionNames": resp.GetPartitionNames(),
			"createTs":       resp.GetCreateTs(),
			"s3Location":     resp.GetS3Location(),
		},
	})
}

// RestoreSnapshot restores a snapshot to a target collection (async operation)
// Returns the job ID for tracking the restore progress
// Parameters:
//   - name: snapshot name
//   - collectionName: target collection name to restore to
//   - options: optional map with "collectionName"/"sourceCollectionName", "dbName", and "targetDbName" keys
func (c *Client) RestoreSnapshot(name string, collectionName string, options ...map[string]interface{}) interface{} {
	start := time.Now()

	if collectionName == "" {
		return toMap(&OperationResult{
			Success:      false,
			ResponseTime: float64(time.Since(start).Milliseconds()),
			Error:        "target collection name required for restore",
		})
	}

	sourceCollectionName := c.defaultCollection
	var dbName, targetDbName string
	if len(options) > 0 && options[0] != nil {
		if coll, ok := options[0]["sourceCollectionName"].(string); ok && coll != "" {
			sourceCollectionName = coll
		} else if coll, ok := options[0]["collectionName"].(string); ok && coll != "" {
			sourceCollectionName = coll
		}
		if db, ok := options[0]["dbName"].(string); ok && db != "" {
			dbName = db
		}
		if targetDB, ok := options[0]["targetDbName"].(string); ok && targetDB != "" {
			targetDbName = targetDB
		}
	}
	if sourceCollectionName == "" {
		return toMap(&OperationResult{
			Success:      false,
			ResponseTime: float64(time.Since(start).Milliseconds()),
			Error:        ErrCollectionNameRequired.Error(),
		})
	}

	opt := milvusclient.NewRestoreSnapshotOption(name, sourceCollectionName, collectionName)
	if dbName != "" {
		opt = opt.WithDbName(dbName)
	}
	if targetDbName != "" {
		opt = opt.WithTargetDbName(targetDbName)
	}

	jobID, err := c.client.RestoreSnapshot(c.context(), opt)
	if err != nil {
		return toMap(&OperationResult{
			Success:      false,
			ResponseTime: float64(time.Since(start).Milliseconds()),
			Error:        fmt.Sprintf("failed to restore snapshot: %v", err),
		})
	}

	return toMap(&OperationResult{
		Success:      true,
		ResponseTime: float64(time.Since(start).Milliseconds()),
		Result: map[string]interface{}{
			"jobId": jobID,
		},
	})
}

// GetRestoreSnapshotState gets the state of a restore snapshot job
func (c *Client) GetRestoreSnapshotState(jobID int64) interface{} {
	start := time.Now()

	opt := milvusclient.NewGetRestoreSnapshotStateOption(jobID)

	info, err := c.client.GetRestoreSnapshotState(c.context(), opt)
	if err != nil {
		return toMap(&OperationResult{
			Success:      false,
			ResponseTime: float64(time.Since(start).Milliseconds()),
			Error:        fmt.Sprintf("failed to get restore snapshot state: %v", err),
		})
	}

	return toMap(&OperationResult{
		Success:      true,
		ResponseTime: float64(time.Since(start).Milliseconds()),
		Result: map[string]interface{}{
			"jobId":          info.GetJobId(),
			"snapshotName":   info.GetSnapshotName(),
			"dbName":         info.GetDbName(),
			"collectionName": info.GetCollectionName(),
			"state":          info.GetState().String(),
			"progress":       info.GetProgress(),
			"reason":         info.GetReason(),
			"startTime":      info.GetStartTime(),
			"timeCost":       info.GetTimeCost(),
		},
	})
}

// ListRestoreSnapshotJobs lists all restore snapshot jobs
// Parameters:
//   - options: optional map with "collectionName" key
func (c *Client) ListRestoreSnapshotJobs(options ...map[string]interface{}) interface{} {
	start := time.Now()

	opt := milvusclient.NewListRestoreSnapshotJobsOption()

	// Parse optional parameters
	if len(options) > 0 && options[0] != nil {
		if coll, ok := options[0]["collectionName"].(string); ok && coll != "" {
			opt = opt.WithCollectionName(coll)
		}
	}

	jobs, err := c.client.ListRestoreSnapshotJobs(c.context(), opt)
	if err != nil {
		return toMap(&OperationResult{
			Success:      false,
			ResponseTime: float64(time.Since(start).Milliseconds()),
			Error:        fmt.Sprintf("failed to list restore snapshot jobs: %v", err),
		})
	}

	// Convert jobs to a serializable format
	jobList := make([]map[string]interface{}, 0, len(jobs))
	for _, job := range jobs {
		jobList = append(jobList, map[string]interface{}{
			"jobId":          job.GetJobId(),
			"snapshotName":   job.GetSnapshotName(),
			"dbName":         job.GetDbName(),
			"collectionName": job.GetCollectionName(),
			"state":          job.GetState().String(),
			"progress":       job.GetProgress(),
			"reason":         job.GetReason(),
			"startTime":      job.GetStartTime(),
			"timeCost":       job.GetTimeCost(),
		})
	}

	return toMap(&OperationResult{
		Success:      true,
		ResponseTime: float64(time.Since(start).Milliseconds()),
		Result:       jobList,
		Empty:        len(jobList) == 0,
	})
}

// resolveCollectionName resolves collection name from interface{}
// This handles JavaScript passing null/undefined as collectionName
func (c *Client) resolveCollectionName(collectionName interface{}) string {
	if collectionName == nil {
		return c.defaultCollection
	}
	if name, ok := collectionName.(string); ok && name != "" {
		return name
	}
	return c.defaultCollection
}
