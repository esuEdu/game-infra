package awsruntime

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
)

const (
	ecsTargetPrefix = "AmazonEC2ContainerServiceV20141113."
	defaultWaitPoll = 5 * time.Second
)

type Client struct {
	region      string
	cfg         aws.Config
	signer      *v4.Signer
	httpClient  aws.HTTPClient
	s3          *s3.Client
	ecsEndpoint string
}

func New(ctx context.Context, region string) (*Client, error) {
	region = strings.TrimSpace(region)
	if region == "" {
		return nil, errors.New("aws region is required")
	}

	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		return nil, fmt.Errorf("load aws config: %w", err)
	}

	httpClient := cfg.HTTPClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	return &Client{
		region:      region,
		cfg:         cfg,
		signer:      v4.NewSigner(),
		httpClient:  httpClient,
		s3:          s3.NewFromConfig(cfg),
		ecsEndpoint: strings.TrimSpace(os.Getenv("ECS_ENDPOINT_URL")),
	}, nil
}

func (c *Client) SetServiceDesiredCount(ctx context.Context, cluster, service string, desired int32, forceNewDeployment bool) error {
	cluster = strings.TrimSpace(cluster)
	service = strings.TrimSpace(service)
	if cluster == "" || service == "" {
		return errors.New("cluster and service are required")
	}

	payload := map[string]any{
		"cluster":      cluster,
		"service":      service,
		"desiredCount": desired,
	}
	if forceNewDeployment {
		payload["forceNewDeployment"] = true
	}

	if err := c.ecsJSONRPC(ctx, "UpdateService", payload, nil); err != nil {
		return err
	}
	return nil
}

func (c *Client) WaitServiceStable(ctx context.Context, cluster, service string, timeout time.Duration) error {
	cluster = strings.TrimSpace(cluster)
	service = strings.TrimSpace(service)
	if cluster == "" || service == "" {
		return errors.New("cluster and service are required")
	}
	if timeout <= 0 {
		timeout = 10 * time.Minute
	}

	deadlineCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(defaultWaitPoll)
	defer ticker.Stop()

	for {
		st, err := c.DescribeService(deadlineCtx, cluster, service)
		if err != nil {
			return err
		}
		if st.isStable() {
			return nil
		}

		select {
		case <-deadlineCtx.Done():
			return fmt.Errorf("wait for ecs service stable: %w", deadlineCtx.Err())
		case <-ticker.C:
		}
	}
}

func (c *Client) DescribeService(ctx context.Context, cluster, service string) (ECSServiceState, error) {
	cluster = strings.TrimSpace(cluster)
	service = strings.TrimSpace(service)
	if cluster == "" || service == "" {
		return ECSServiceState{}, errors.New("cluster and service are required")
	}

	payload := map[string]any{
		"cluster":  cluster,
		"services": []string{service},
	}
	var out ecsDescribeServicesOutput
	if err := c.ecsJSONRPC(ctx, "DescribeServices", payload, &out); err != nil {
		return ECSServiceState{}, err
	}

	if len(out.Failures) > 0 {
		fail := out.Failures[0]
		msg := strings.TrimSpace(fail.Reason)
		if msg == "" {
			msg = "unknown ecs describe failure"
		}
		return ECSServiceState{}, fmt.Errorf("ecs describe service failure: %s", msg)
	}

	if len(out.Services) == 0 {
		return ECSServiceState{}, fmt.Errorf("ecs service not found: %s", service)
	}

	return out.Services[0], nil
}

func (c *Client) UploadFile(ctx context.Context, bucket, key, path string) error {
	bucket = strings.TrimSpace(bucket)
	key = strings.Trim(strings.TrimSpace(key), "/")
	if bucket == "" || key == "" {
		return errors.New("bucket and key are required")
	}

	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open upload file %s: %w", path, err)
	}
	defer f.Close()

	if _, err := c.s3.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
		Body:   f,
	}); err != nil {
		return fmt.Errorf("s3 put object s3://%s/%s: %w", bucket, key, err)
	}

	return nil
}

func (c *Client) DownloadFile(ctx context.Context, bucket, key, path string) error {
	bucket = strings.TrimSpace(bucket)
	key = strings.Trim(strings.TrimSpace(key), "/")
	if bucket == "" || key == "" {
		return errors.New("bucket and key are required")
	}

	out, err := c.s3.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("s3 get object s3://%s/%s: %w", bucket, key, err)
	}
	defer out.Body.Close()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create parent dir for %s: %w", path, err)
	}

	f, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("open destination file %s: %w", path, err)
	}
	defer f.Close()

	if _, err := io.Copy(f, out.Body); err != nil {
		return fmt.Errorf("write destination file %s: %w", path, err)
	}

	return nil
}

func (c *Client) PutString(ctx context.Context, bucket, key, value string) error {
	bucket = strings.TrimSpace(bucket)
	key = strings.Trim(strings.TrimSpace(key), "/")
	if bucket == "" || key == "" {
		return errors.New("bucket and key are required")
	}

	if _, err := c.s3.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
		Body:   strings.NewReader(value),
	}); err != nil {
		return fmt.Errorf("s3 put object s3://%s/%s: %w", bucket, key, err)
	}
	return nil
}

func (c *Client) GetString(ctx context.Context, bucket, key string) (string, error) {
	bucket = strings.TrimSpace(bucket)
	key = strings.Trim(strings.TrimSpace(key), "/")
	if bucket == "" || key == "" {
		return "", errors.New("bucket and key are required")
	}

	out, err := c.s3.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return "", fmt.Errorf("s3 get object s3://%s/%s: %w", bucket, key, err)
	}
	defer out.Body.Close()

	body, err := io.ReadAll(out.Body)
	if err != nil {
		return "", fmt.Errorf("read s3 object s3://%s/%s: %w", bucket, key, err)
	}

	return string(body), nil
}

func (c *Client) IsObjectNotFound(err error) bool {
	if err == nil {
		return false
	}
	var noSuchKey *s3types.NoSuchKey
	if errors.As(err, &noSuchKey) {
		return true
	}
	var apiErr smithy.APIError
	if errors.As(err, &apiErr) {
		code := strings.TrimSpace(apiErr.ErrorCode())
		return code == "NoSuchKey" || code == "NotFound"
	}
	return false
}

func (c *Client) ecsJSONRPC(ctx context.Context, operation string, payload any, out any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal ecs payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.ecsEndpointURL(), bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create ecs request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-amz-json-1.1")
	req.Header.Set("X-Amz-Target", ecsTargetPrefix+operation)

	payloadHash := hashSHA256Hex(body)
	req.Header.Set("X-Amz-Content-Sha256", payloadHash)

	cred, err := c.cfg.Credentials.Retrieve(ctx)
	if err != nil {
		return fmt.Errorf("retrieve aws credentials: %w", err)
	}

	if err := c.signer.SignHTTP(ctx, cred, req, payloadHash, "ecs", c.region, time.Now().UTC()); err != nil {
		return fmt.Errorf("sign ecs request %s: %w", operation, err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("do ecs request %s: %w", operation, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read ecs response %s: %w", operation, err)
	}

	if resp.StatusCode >= 300 {
		msg := strings.TrimSpace(string(respBody))
		if msg == "" {
			msg = http.StatusText(resp.StatusCode)
		}
		return fmt.Errorf("ecs %s failed (%d): %s", operation, resp.StatusCode, msg)
	}

	if out != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, out); err != nil {
			return fmt.Errorf("decode ecs response %s: %w", operation, err)
		}
	}

	return nil
}

func (c *Client) ecsEndpointURL() string {
	if c.ecsEndpoint != "" {
		return strings.TrimRight(c.ecsEndpoint, "/") + "/"
	}
	return fmt.Sprintf("https://ecs.%s.amazonaws.com/", c.region)
}

func hashSHA256Hex(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

type ecsDescribeServicesOutput struct {
	Services []ECSServiceState   `json:"services"`
	Failures []ecsServiceFailure `json:"failures"`
}

type ecsServiceFailure struct {
	Reason string `json:"reason"`
}

type ECSServiceState struct {
	ServiceName  string          `json:"serviceName"`
	Status       string          `json:"status"`
	DesiredCount int32           `json:"desiredCount"`
	RunningCount int32           `json:"runningCount"`
	PendingCount int32           `json:"pendingCount"`
	Deployments  []ecsDeployment `json:"deployments"`
}

type ecsDeployment struct {
	Status       string `json:"status"`
	RolloutState string `json:"rolloutState"`
	DesiredCount int32  `json:"desiredCount"`
	RunningCount int32  `json:"runningCount"`
	PendingCount int32  `json:"pendingCount"`
}

func (s ECSServiceState) isStable() bool {
	if strings.EqualFold(strings.TrimSpace(s.Status), "DRAINING") {
		return false
	}
	if s.PendingCount != 0 {
		return false
	}
	if s.RunningCount != s.DesiredCount {
		return false
	}
	if len(s.Deployments) == 0 {
		return false
	}
	if len(s.Deployments) > 1 {
		return false
	}

	d := s.Deployments[0]
	if d.PendingCount != 0 {
		return false
	}
	if d.RunningCount != d.DesiredCount {
		return false
	}
	if d.RunningCount != s.DesiredCount {
		return false
	}
	if strings.TrimSpace(d.RolloutState) != "" && !strings.EqualFold(strings.TrimSpace(d.RolloutState), "COMPLETED") {
		return false
	}
	return true
}
