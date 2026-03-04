package s3

import (
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// bucketsLoadedMsg is sent when ListBuckets succeeds.
type bucketsLoadedMsg struct {
	buckets []types.Bucket
}

// bucketsErrMsg is sent when ListBuckets fails.
type bucketsErrMsg struct {
	err error
}

// prefixesLoadedMsg is sent when ListObjectsV2 for a prefix succeeds.
// prefixes are virtual folders (CommonPrefixes with trailing /),
// objects are real objects at the current level only.
type prefixesLoadedMsg struct {
	prefixes []string
	objects  []types.Object
}

// prefixesErrMsg is sent when ListObjectsV2 fails.
type prefixesErrMsg struct {
	err error
}

// bucketClientReadyMsg is sent when HeadBucket succeeds and the per-bucket
// regional client has been constructed. The panel model stores this client
// for all subsequent object operations in this bucket.
type bucketClientReadyMsg struct {
	bucket string
	region string
}

// bucketClientErrMsg is sent when HeadBucket fails during regional client construction.
type bucketClientErrMsg struct {
	bucket string
	err    error
}

// s3RefreshTickMsg is the 30-second S3-specific refresh ticker message.
// Distinct from the global 1-second tickMsg in internal/ui/messages.go.
type s3RefreshTickMsg struct{}
