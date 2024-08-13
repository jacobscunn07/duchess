package s3

import "time"

type Bucket struct {
}

type BucketObject struct {
	ETag         string
	Key          string
	LastModified time.Time
	Owner        string
	Size         int64
}
