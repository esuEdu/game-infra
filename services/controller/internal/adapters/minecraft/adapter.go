package minecraft

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/esuEdu/game-infra/controller/internal/adapters/awsruntime"
	"github.com/esuEdu/game-infra/controller/internal/domain"
)

type Adapter struct {
	log        *slog.Logger
	mu         sync.Mutex
	running    bool
	lastBackup string
	lastSource string

	awsRegion string
	cluster   string
	service   string
	bucket    string

	backupPrefix string
	dataDir      string

	aws *awsruntime.Client

	gitUserName  string
	gitUserEmail string
	gitToken     string
}

func NewAdapter(log *slog.Logger) *Adapter {
	return &Adapter{
		log:          log,
		awsRegion:    envOrDefault("AWS_REGION", "us-east-1"),
		cluster:      strings.TrimSpace(os.Getenv("ECS_CLUSTER_NAME")),
		service:      strings.TrimSpace(os.Getenv("ECS_SERVICE_MINECRAFT")),
		bucket:       strings.TrimSpace(os.Getenv("BACKUP_BUCKET")),
		backupPrefix: strings.Trim(strings.TrimSpace(envOrDefault("BACKUP_PREFIX", "backups")), "/"),
		dataDir:      envOrDefault("MC_DATA_DIR", "/srv/minecraft-data"),
		gitUserName:  envOrDefault("GIT_USER_NAME", "GameStack Bot"),
		gitUserEmail: envOrDefault("GIT_USER_EMAIL", "gamestack-bot@example.com"),
		gitToken:     strings.TrimSpace(os.Getenv("GIT_AUTH_TOKEN")),
	}
}

func (a *Adapter) Type() domain.GameType { return domain.GameMinecraft }

func (a *Adapter) Start(ctx context.Context) error {
	if a.ecsConfigured() {
		awsClient, err := a.awsClient(ctx)
		if err != nil {
			return err
		}
		if err := awsClient.SetServiceDesiredCount(ctx, a.cluster, a.service, 1, true); err != nil {
			return err
		}
		if err := awsClient.WaitServiceStable(ctx, a.cluster, a.service, 10*time.Minute); err != nil {
			return err
		}
	}

	a.mu.Lock()
	a.running = true
	a.mu.Unlock()
	a.log.Info("minecraft start", "cluster", a.cluster, "service", a.service)
	return nil
}

func (a *Adapter) Stop(ctx context.Context) error {
	if a.ecsConfigured() {
		awsClient, err := a.awsClient(ctx)
		if err != nil {
			return err
		}
		if err := awsClient.SetServiceDesiredCount(ctx, a.cluster, a.service, 0, false); err != nil {
			return err
		}
		if err := awsClient.WaitServiceStable(ctx, a.cluster, a.service, 10*time.Minute); err != nil {
			return err
		}
	}

	a.mu.Lock()
	a.running = false
	a.mu.Unlock()
	a.log.Info("minecraft stop", "cluster", a.cluster, "service", a.service)
	return nil
}

func (a *Adapter) Backup(ctx context.Context) (string, error) {
	if !a.s3Configured() {
		return "", errors.New("s3 backup not configured")
	}

	if err := os.MkdirAll(a.dataDir, 0o755); err != nil {
		return "", fmt.Errorf("prepare data dir: %w", err)
	}

	tmpZip, err := os.CreateTemp("", "minecraft-backup-*.zip")
	if err != nil {
		return "", fmt.Errorf("create temp backup: %w", err)
	}
	tmpZipPath := tmpZip.Name()
	_ = tmpZip.Close()
	defer os.Remove(tmpZipPath)

	if err := zipDirectory(a.dataDir, tmpZipPath); err != nil {
		return "", err
	}

	key := a.backupKey()
	uri := fmt.Sprintf("s3://%s/%s", a.bucket, key)
	awsClient, err := a.awsClient(ctx)
	if err != nil {
		return "", err
	}
	if err := awsClient.UploadFile(ctx, a.bucket, key, tmpZipPath); err != nil {
		return "", fmt.Errorf("upload backup to s3: %w", err)
	}

	if err := awsClient.PutString(ctx, a.bucket, a.latestBackupKey(), key); err != nil {
		return "", fmt.Errorf("upload latest marker: %w", err)
	}

	a.mu.Lock()
	a.lastBackup = uri
	backup := a.lastBackup
	a.mu.Unlock()

	a.log.Info("minecraft backup complete", "backup", backup)
	return backup, nil
}

func (a *Adapter) Restore(ctx context.Context, backupKey string) error {
	if !a.s3Configured() {
		return errors.New("s3 backup not configured")
	}
	if strings.TrimSpace(backupKey) == "" {
		return errors.New("empty backup key")
	}

	bucket, key, err := parseBackupRef(a.bucket, backupKey)
	if err != nil {
		return err
	}

	tmpZip, err := os.CreateTemp("", "minecraft-restore-*.zip")
	if err != nil {
		return fmt.Errorf("create temp restore file: %w", err)
	}
	tmpZipPath := tmpZip.Name()
	_ = tmpZip.Close()
	defer os.Remove(tmpZipPath)

	awsClient, err := a.awsClient(ctx)
	if err != nil {
		return err
	}
	if err := awsClient.DownloadFile(ctx, bucket, key, tmpZipPath); err != nil {
		return fmt.Errorf("download backup from s3: %w", err)
	}

	if err := resetDirectory(a.dataDir); err != nil {
		return err
	}
	if err := unzipToDirectory(tmpZipPath, a.dataDir); err != nil {
		return err
	}

	a.mu.Lock()
	a.lastBackup = fmt.Sprintf("s3://%s/%s", bucket, key)
	a.mu.Unlock()
	a.log.Info("minecraft restore complete", "backup", a.lastBackup)
	return nil
}

func (a *Adapter) SeedFromSource(ctx context.Context, sourceURL string) error {
	sourceURL = strings.TrimSpace(sourceURL)
	if sourceURL == "" {
		return errors.New("source url is required")
	}

	repoURL, repoRef, repoPath := parseSourceURL(sourceURL)
	authURL, err := a.withGitToken(repoURL)
	if err != nil {
		return err
	}

	tmpDir, err := os.MkdirTemp("", "minecraft-seed-*")
	if err != nil {
		return fmt.Errorf("create temp seed dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	repoDir := filepath.Join(tmpDir, "repo")
	if _, err := a.run(ctx, "git", "clone", "--depth", "1", "--branch", repoRef, authURL, repoDir); err != nil {
		return fmt.Errorf("git clone source: %w", err)
	}

	srcDir := repoDir
	if repoPath != "" {
		srcDir = filepath.Join(repoDir, repoPath)
	}
	if _, err := os.Stat(srcDir); err != nil {
		return fmt.Errorf("source path not found in repo: %s", repoPath)
	}

	if err := resetDirectory(a.dataDir); err != nil {
		return err
	}
	if err := copyDirectoryContents(srcDir, a.dataDir); err != nil {
		return err
	}

	a.mu.Lock()
	a.lastSource = sourceURL
	a.mu.Unlock()
	a.log.Info("minecraft seed from source complete", "source", sourceURL)
	return nil
}

func (a *Adapter) SyncToSource(ctx context.Context, sourceURL string) error {
	sourceURL = strings.TrimSpace(sourceURL)
	if sourceURL == "" {
		return errors.New("source url is required")
	}

	repoURL, repoRef, repoPath := parseSourceURL(sourceURL)
	authURL, err := a.withGitToken(repoURL)
	if err != nil {
		return err
	}

	tmpDir, err := os.MkdirTemp("", "minecraft-sync-*")
	if err != nil {
		return fmt.Errorf("create temp sync dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	repoDir := filepath.Join(tmpDir, "repo")
	if _, err := a.run(ctx, "git", "clone", authURL, repoDir); err != nil {
		return fmt.Errorf("git clone for sync: %w", err)
	}

	if repoRef != "" {
		if _, err := a.run(ctx, "git", "-C", repoDir, "checkout", repoRef); err != nil {
			if _, err := a.run(ctx, "git", "-C", repoDir, "checkout", "-b", repoRef); err != nil {
				return fmt.Errorf("checkout branch for sync: %w", err)
			}
		}
	}

	targetDir := repoDir
	if repoPath != "" {
		targetDir = filepath.Join(repoDir, repoPath)
		if err := os.MkdirAll(targetDir, 0o755); err != nil {
			return fmt.Errorf("create repo path for sync: %w", err)
		}
	}

	if err := clearDirectory(targetDir); err != nil {
		return err
	}
	if err := copyDirectoryContents(a.dataDir, targetDir); err != nil {
		return err
	}

	if _, err := a.run(ctx, "git", "-C", repoDir, "config", "user.name", a.gitUserName); err != nil {
		return fmt.Errorf("git config user.name: %w", err)
	}
	if _, err := a.run(ctx, "git", "-C", repoDir, "config", "user.email", a.gitUserEmail); err != nil {
		return fmt.Errorf("git config user.email: %w", err)
	}

	if _, err := a.run(ctx, "git", "-C", repoDir, "add", "-A"); err != nil {
		return fmt.Errorf("git add: %w", err)
	}

	statusOut, err := a.run(ctx, "git", "-C", repoDir, "status", "--porcelain")
	if err != nil {
		return fmt.Errorf("git status: %w", err)
	}
	if strings.TrimSpace(statusOut) == "" {
		a.log.Info("minecraft sync skipped (no changes)", "source", sourceURL)
		return nil
	}

	msg := fmt.Sprintf("chore: sync minecraft data %s", time.Now().UTC().Format(time.RFC3339))
	if _, err := a.run(ctx, "git", "-C", repoDir, "commit", "-m", msg); err != nil {
		return fmt.Errorf("git commit: %w", err)
	}

	pushRef := "HEAD"
	if repoRef != "" {
		pushRef = fmt.Sprintf("HEAD:refs/heads/%s", repoRef)
	}
	if _, err := a.run(ctx, "git", "-C", repoDir, "push", "origin", pushRef); err != nil {
		return fmt.Errorf("git push: %w", err)
	}

	a.mu.Lock()
	a.lastSource = sourceURL
	a.mu.Unlock()
	a.log.Info("minecraft sync to source complete", "source", sourceURL)
	return nil
}

func (a *Adapter) SendCommand(ctx context.Context, command string) error {
	a.log.Info("minecraft command (stub)", "cmd", command)
	return nil
}

func (a *Adapter) Status(ctx context.Context) (map[string]any, error) {
	a.mu.Lock()
	running := a.running
	lastBackup := a.lastBackup
	lastSource := a.lastSource
	a.mu.Unlock()

	return map[string]any{
		"adapter":     "minecraft",
		"ready":       true,
		"running":     running,
		"last_backup": lastBackup,
		"last_source": lastSource,
		"cluster":     a.cluster,
		"service":     a.service,
		"bucket":      a.bucket,
	}, nil
}

func (a *Adapter) LatestBackup(ctx context.Context) (string, error) {
	a.mu.Lock()
	if strings.TrimSpace(a.lastBackup) != "" {
		backup := a.lastBackup
		a.mu.Unlock()
		return backup, nil
	}
	a.mu.Unlock()

	if !a.s3Configured() {
		return "", errors.New("s3 backup not configured")
	}

	awsClient, err := a.awsClient(ctx)
	if err != nil {
		return "", err
	}
	latestValue, err := awsClient.GetString(ctx, a.bucket, a.latestBackupKey())
	if err != nil {
		return "", fmt.Errorf("read latest backup marker: %w", err)
	}

	bucket, key, err := parseBackupRef(a.bucket, strings.TrimSpace(latestValue))
	if err != nil {
		return "", fmt.Errorf("parse latest backup marker: %w", err)
	}
	if key == "" {
		return "", errors.New("latest backup marker is empty")
	}

	backup := fmt.Sprintf("s3://%s/%s", bucket, key)
	a.mu.Lock()
	a.lastBackup = backup
	a.mu.Unlock()
	return backup, nil
}

func (a *Adapter) ecsConfigured() bool {
	return a.cluster != "" && a.service != "" && a.awsRegion != ""
}

func (a *Adapter) s3Configured() bool {
	return a.bucket != "" && a.awsRegion != ""
}

func (a *Adapter) run(ctx context.Context, cmd string, args ...string) (string, error) {
	c := exec.CommandContext(ctx, cmd, args...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	c.Stdout = &stdout
	c.Stderr = &stderr
	if err := c.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return stdout.String(), fmt.Errorf("%s failed: %s", cmd, msg)
	}
	return stdout.String(), nil
}

func (a *Adapter) backupKey() string {
	base := fmt.Sprintf("minecraft/%s.zip", time.Now().UTC().Format("20060102-150405"))
	if a.backupPrefix == "" {
		return base
	}
	return a.backupPrefix + "/" + base
}

func (a *Adapter) latestBackupKey() string {
	key := "minecraft/latest.txt"
	if a.backupPrefix != "" {
		key = a.backupPrefix + "/" + key
	}
	return key
}

func (a *Adapter) awsClient(ctx context.Context) (*awsruntime.Client, error) {
	a.mu.Lock()
	existing := a.aws
	a.mu.Unlock()
	if existing != nil {
		return existing, nil
	}

	client, err := awsruntime.New(ctx, a.awsRegion)
	if err != nil {
		return nil, err
	}

	a.mu.Lock()
	if a.aws == nil {
		a.aws = client
	} else {
		client = a.aws
	}
	a.mu.Unlock()
	return client, nil
}

func envOrDefault(key, fallback string) string {
	val := strings.TrimSpace(os.Getenv(key))
	if val == "" {
		return fallback
	}
	return val
}

func parseSourceURL(raw string) (repoURL, ref, path string) {
	repoURL = strings.TrimSpace(raw)
	ref = "main"
	path = ""

	parts := strings.SplitN(repoURL, "#", 2)
	repoURL = parts[0]
	if len(parts) == 1 {
		return repoURL, ref, path
	}

	refSpec := parts[1]
	refParts := strings.SplitN(refSpec, ":", 2)
	if strings.TrimSpace(refParts[0]) != "" {
		ref = strings.TrimSpace(refParts[0])
	}
	if len(refParts) == 2 {
		path = strings.TrimPrefix(strings.TrimSpace(refParts[1]), "/")
	}

	return repoURL, ref, path
}

func (a *Adapter) withGitToken(repoURL string) (string, error) {
	if a.gitToken == "" {
		return repoURL, nil
	}
	parsed, err := url.Parse(repoURL)
	if err != nil {
		return "", fmt.Errorf("invalid repository url: %w", err)
	}
	if parsed.Scheme != "https" {
		return repoURL, nil
	}
	parsed.User = url.UserPassword("x-access-token", a.gitToken)
	return parsed.String(), nil
}

func parseBackupRef(defaultBucket, backupRef string) (bucket, key string, err error) {
	ref := strings.TrimSpace(backupRef)
	if ref == "" {
		return "", "", errors.New("empty backup ref")
	}
	if strings.HasPrefix(ref, "s3://") {
		trimmed := strings.TrimPrefix(ref, "s3://")
		parts := strings.SplitN(trimmed, "/", 2)
		if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
			return "", "", fmt.Errorf("invalid s3 backup ref: %s", backupRef)
		}
		return parts[0], parts[1], nil
	}
	if defaultBucket == "" {
		return "", "", errors.New("default bucket is empty")
	}
	return defaultBucket, ref, nil
}

func resetDirectory(dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create directory %s: %w", dir, err)
	}
	if err := clearDirectory(dir); err != nil {
		return err
	}
	return nil
}

func clearDirectory(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("read directory %s: %w", dir, err)
	}
	for _, entry := range entries {
		name := entry.Name()
		if name == ".git" {
			continue
		}
		if err := os.RemoveAll(filepath.Join(dir, name)); err != nil {
			return fmt.Errorf("remove %s: %w", filepath.Join(dir, name), err)
		}
	}
	return nil
}

func copyDirectoryContents(srcDir, dstDir string) error {
	if err := os.MkdirAll(dstDir, 0o755); err != nil {
		return fmt.Errorf("create destination dir %s: %w", dstDir, err)
	}

	entries, err := os.ReadDir(srcDir)
	if err != nil {
		return fmt.Errorf("read source dir %s: %w", srcDir, err)
	}

	for _, entry := range entries {
		if entry.Name() == ".git" {
			continue
		}
		srcPath := filepath.Join(srcDir, entry.Name())
		dstPath := filepath.Join(dstDir, entry.Name())
		if err := copyPath(srcPath, dstPath); err != nil {
			return err
		}
	}
	return nil
}

func copyPath(srcPath, dstPath string) error {
	info, err := os.Lstat(srcPath)
	if err != nil {
		return fmt.Errorf("stat %s: %w", srcPath, err)
	}

	switch mode := info.Mode(); {
	case mode.IsRegular():
		return copyFile(srcPath, dstPath, mode.Perm())
	case mode.IsDir():
		if err := os.MkdirAll(dstPath, mode.Perm()); err != nil {
			return fmt.Errorf("mkdir %s: %w", dstPath, err)
		}
		children, err := os.ReadDir(srcPath)
		if err != nil {
			return fmt.Errorf("readdir %s: %w", srcPath, err)
		}
		for _, child := range children {
			if err := copyPath(filepath.Join(srcPath, child.Name()), filepath.Join(dstPath, child.Name())); err != nil {
				return err
			}
		}
		return nil
	default:
		// Skip special files for now (symlinks/devices/sockets).
		return nil
	}
}

func copyFile(srcPath, dstPath string, perm fs.FileMode) error {
	src, err := os.Open(srcPath)
	if err != nil {
		return fmt.Errorf("open source file %s: %w", srcPath, err)
	}
	defer src.Close()

	dst, err := os.OpenFile(dstPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, perm)
	if err != nil {
		return fmt.Errorf("open destination file %s: %w", dstPath, err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return fmt.Errorf("copy file %s -> %s: %w", srcPath, dstPath, err)
	}
	return nil
}

func zipDirectory(srcDir, dstZip string) error {
	out, err := os.Create(dstZip)
	if err != nil {
		return fmt.Errorf("create zip %s: %w", dstZip, err)
	}
	defer out.Close()

	zw := zip.NewWriter(out)
	defer zw.Close()

	if err := filepath.WalkDir(srcDir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == srcDir {
			return nil
		}

		relPath, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}
		relPath = filepath.ToSlash(relPath)

		if d.IsDir() {
			_, err := zw.Create(relPath + "/")
			return err
		}

		info, err := d.Info()
		if err != nil {
			return err
		}

		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}
		header.Name = relPath
		header.Method = zip.Deflate

		w, err := zw.CreateHeader(header)
		if err != nil {
			return err
		}

		f, err := os.Open(path)
		if err != nil {
			return err
		}
		_, err = io.Copy(w, f)
		closeErr := f.Close()
		if err != nil {
			return err
		}
		if closeErr != nil {
			return closeErr
		}
		return nil
	}); err != nil {
		return fmt.Errorf("walk source dir for zip: %w", err)
	}

	return nil
}

func unzipToDirectory(srcZip, dstDir string) error {
	r, err := zip.OpenReader(srcZip)
	if err != nil {
		return fmt.Errorf("open zip %s: %w", srcZip, err)
	}
	defer r.Close()

	for _, f := range r.File {
		cleanName := filepath.Clean(f.Name)
		if filepath.IsAbs(cleanName) || strings.HasPrefix(cleanName, "..") {
			return fmt.Errorf("zip contains invalid path: %s", f.Name)
		}
		outPath := filepath.Join(dstDir, cleanName)

		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(outPath, f.Mode()); err != nil {
				return fmt.Errorf("mkdir %s: %w", outPath, err)
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
			return fmt.Errorf("mkdir parent for %s: %w", outPath, err)
		}

		in, err := f.Open()
		if err != nil {
			return fmt.Errorf("open zip entry %s: %w", f.Name, err)
		}

		out, err := os.OpenFile(outPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, f.Mode())
		if err != nil {
			in.Close()
			return fmt.Errorf("open output file %s: %w", outPath, err)
		}

		if _, err := io.Copy(out, in); err != nil {
			in.Close()
			out.Close()
			return fmt.Errorf("extract %s: %w", f.Name, err)
		}
		in.Close()
		out.Close()
	}

	return nil
}
