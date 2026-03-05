package s3

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	tea "github.com/charmbracelet/bubbletea"
)

// ListAllBuckets returns all S3 buckets visible to the client using pagination.
// It uses the ListBuckets paginator to handle accounts with many buckets without truncation.
func ListAllBuckets(ctx context.Context, client *s3.Client) ([]types.Bucket, error) {
	paginator := s3.NewListBucketsPaginator(client, &s3.ListBucketsInput{})
	var buckets []types.Bucket
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		buckets = append(buckets, page.Buckets...)
	}
	return buckets, nil
}

// BucketRegion returns the AWS region for the given bucket using HeadBucket.
// HeadBucket is used instead of GetBucketLocation because GetBucketLocation
// returns null LocationConstraint for us-east-1 buckets (documented AWS API quirk).
// HeadBucket also returns an empty BucketRegion for us-east-1, which we normalize
// to the "us-east-1" string here.
func BucketRegion(ctx context.Context, client *s3.Client, bucket string) (string, error) {
	out, err := client.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(bucket),
	})
	if err != nil {
		return "", err
	}
	region := aws.ToString(out.BucketRegion)
	if region == "" {
		// AWS quirk: HeadBucket returns empty BucketRegion for us-east-1 buckets,
		// same convention as GetBucketLocation returning null LocationConstraint.
		return "us-east-1", nil
	}
	return region, nil
}

// NewBucketClient constructs a per-bucket regional S3 client by first calling
// HeadBucket to determine the bucket's actual region, then overriding the region
// in the base config. This is required because cross-region S3 operations fail
// with a redirect error if the client region does not match the bucket region.
func NewBucketClient(ctx context.Context, baseCfg aws.Config, baseClient *s3.Client, bucket string) (*s3.Client, error) {
	region, err := BucketRegion(ctx, baseClient, bucket)
	if err != nil {
		return nil, fmt.Errorf("failed to determine region for bucket %q: %w", bucket, err)
	}
	return s3.NewFromConfig(baseCfg, func(o *s3.Options) {
		o.Region = region
	}), nil
}

// ListPrefix lists objects and virtual folders at the given prefix within a bucket.
// It uses Delimiter="/" so that only the current level is returned — virtual folders
// appear in CommonPrefixes and real objects appear in Contents.
// IMPORTANT: The Delimiter is mandatory. Without it, S3 returns all objects recursively,
// which could be millions of objects for large buckets.
// For filter support, callers pass currentPrefix+filterValue as the prefix string
// to perform a server-side key prefix filter without changing navigation depth.
func ListPrefix(ctx context.Context, client *s3.Client, bucket, prefix string) (prefixes []string, objects []types.Object, err error) {
	paginator := s3.NewListObjectsV2Paginator(client, &s3.ListObjectsV2Input{
		Bucket:    aws.String(bucket),
		Prefix:    aws.String(prefix),    // "" for bucket root; "logs/" for subdirectory; "logs/2024" for filtered
		Delimiter: aws.String("/"),        // REQUIRED: without delimiter, all objects returned recursively (potentially millions)
	})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, nil, err
		}
		for _, cp := range page.CommonPrefixes {
			prefixes = append(prefixes, aws.ToString(cp.Prefix))
		}
		objects = append(objects, page.Contents...)
	}
	return prefixes, objects, nil
}

// FetchBucketsCmd returns a tea.Cmd that lists all S3 buckets asynchronously.
// On success it returns bucketsLoadedMsg; on failure it returns bucketsErrMsg.
func FetchBucketsCmd(ctx context.Context, client *s3.Client) tea.Cmd {
	return func() tea.Msg {
		buckets, err := ListAllBuckets(ctx, client)
		if err != nil {
			return bucketsErrMsg{err: err}
		}
		return bucketsLoadedMsg{buckets: buckets}
	}
}

// FetchBucketClientCmd returns a tea.Cmd that calls HeadBucket to determine the
// bucket region and sends bucketClientReadyMsg on success, or bucketClientErrMsg
// on failure. The panel model uses the region from bucketClientReadyMsg to construct
// a regional client for all subsequent object operations in this bucket.
func FetchBucketClientCmd(ctx context.Context, baseCfg aws.Config, baseClient *s3.Client, bucket string) tea.Cmd {
	return func() tea.Msg {
		region, err := BucketRegion(ctx, baseClient, bucket)
		if err != nil {
			return bucketClientErrMsg{bucket: bucket, err: err}
		}
		return bucketClientReadyMsg{bucket: bucket, region: region}
	}
}

// FetchPrefixCmd returns a tea.Cmd that lists objects and virtual folders for a given
// prefix path within a bucket asynchronously. On success it returns prefixesLoadedMsg;
// on failure it returns prefixesErrMsg.
func FetchPrefixCmd(ctx context.Context, client *s3.Client, bucket, prefix string) tea.Cmd {
	return func() tea.Msg {
		prefixes, objects, err := ListPrefix(ctx, client, bucket, prefix)
		if err != nil {
			return prefixesErrMsg{err: err}
		}
		return prefixesLoadedMsg{prefixes: prefixes, objects: objects}
	}
}

// S3RefreshTickCmd returns a tea.Cmd that fires s3RefreshTickMsg after d.
// This is distinct from the global 1-second tickMsg in internal/ui/messages.go —
// it drives S3 data refresh at the interval configured by the user.
func S3RefreshTickCmd(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(t time.Time) tea.Msg {
		return s3RefreshTickMsg{}
	})
}
