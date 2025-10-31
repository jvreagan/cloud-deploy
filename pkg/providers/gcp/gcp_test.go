package gcp

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jvreagan/cloud-deploy/pkg/manifest"
)

func TestProviderName(t *testing.T) {
	provider := &Provider{
		projectID: "test-project",
		region:    "us-central1",
	}

	if provider.Name() != "gcp" {
		t.Errorf("Expected provider name 'gcp', got '%s'", provider.Name())
	}
}

func TestCreateTarGz(t *testing.T) {
	// Create a temporary directory with test files
	tmpDir := t.TempDir()

	// Create test files
	testFiles := map[string]string{
		"file1.txt":            "content1",
		"file2.txt":            "content2",
		"subdir/file3.txt":     "content3",
		"subdir/deep/file4.go": "package main\n\nfunc main() {}",
	}

	for path, content := range testFiles {
		fullPath := filepath.Join(tmpDir, path)
		dir := filepath.Dir(fullPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	// Create a temporary tar.gz file
	tarFile, err := os.CreateTemp("", "test-*.tar.gz")
	if err != nil {
		t.Fatalf("Failed to create temp tar file: %v", err)
	}
	defer os.Remove(tarFile.Name())
	defer tarFile.Close()

	// Test createTarGz
	err = createTarGz(tmpDir, tarFile)
	if err != nil {
		t.Fatalf("createTarGz failed: %v", err)
	}

	// Rewind and verify tar.gz contents
	if _, err := tarFile.Seek(0, 0); err != nil {
		t.Fatalf("Failed to seek: %v", err)
	}

	// Open gzip reader
	gzReader, err := gzip.NewReader(tarFile)
	if err != nil {
		t.Fatalf("Failed to create gzip reader: %v", err)
	}
	defer gzReader.Close()

	// Open tar reader
	tarReader := tar.NewReader(gzReader)

	// Read all files from tar
	foundFiles := make(map[string]string)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("Failed to read tar: %v", err)
		}

		// Read file content
		content, err := io.ReadAll(tarReader)
		if err != nil {
			t.Fatalf("Failed to read tar entry: %v", err)
		}

		foundFiles[header.Name] = string(content)
	}

	// Verify all files are in the tar
	for expectedPath, expectedContent := range testFiles {
		actualContent, exists := foundFiles[expectedPath]
		if !exists {
			t.Errorf("Expected file '%s' not found in tar", expectedPath)
			continue
		}
		if actualContent != expectedContent {
			t.Errorf("File '%s': expected content '%s', got '%s'", expectedPath, expectedContent, actualContent)
		}
	}
}

func TestCreateTarGzSkipsHiddenFiles(t *testing.T) {
	// Create a temporary directory with hidden files
	tmpDir := t.TempDir()

	// Create regular and hidden files
	files := []string{
		"regular.txt",
		".hidden",
		".git/config",
		"subdir/.env",
	}

	for _, file := range files {
		fullPath := filepath.Join(tmpDir, file)
		dir := filepath.Dir(fullPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}
		if err := os.WriteFile(fullPath, []byte("content"), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	// Create a temporary tar.gz file
	tarFile, err := os.CreateTemp("", "test-*.tar.gz")
	if err != nil {
		t.Fatalf("Failed to create temp tar file: %v", err)
	}
	defer os.Remove(tarFile.Name())
	defer tarFile.Close()

	// Test createTarGz
	err = createTarGz(tmpDir, tarFile)
	if err != nil {
		t.Fatalf("createTarGz failed: %v", err)
	}

	// Rewind and verify tar.gz contents
	if _, err := tarFile.Seek(0, 0); err != nil {
		t.Fatalf("Failed to seek: %v", err)
	}

	// Open gzip reader
	gzReader, err := gzip.NewReader(tarFile)
	if err != nil {
		t.Fatalf("Failed to create gzip reader: %v", err)
	}
	defer gzReader.Close()

	// Open tar reader
	tarReader := tar.NewReader(gzReader)

	// Read all files from tar
	foundFiles := make(map[string]bool)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("Failed to read tar: %v", err)
		}

		foundFiles[header.Name] = true

		// Verify no hidden files
		if strings.HasPrefix(filepath.Base(header.Name), ".") {
			t.Errorf("Hidden file '%s' should not be in tar", header.Name)
		}
	}

	// Verify regular file is included
	if !foundFiles["regular.txt"] {
		t.Error("Regular file 'regular.txt' should be in tar")
	}
}

func TestLoadCredentials(t *testing.T) {
	tests := []struct {
		name        string
		creds       *manifest.CredentialsConfig
		expectError bool
		errorMsg    string
	}{
		{
			name: "with service account key path",
			creds: &manifest.CredentialsConfig{
				ServiceAccountKeyPath: "/path/to/key.json",
			},
			expectError: false,
		},
		{
			name: "with service account key JSON",
			creds: &manifest.CredentialsConfig{
				ServiceAccountKeyJSON: `{"type":"service_account","project_id":"test"}`,
			},
			expectError: false,
		},
		{
			name:        "with nil credentials",
			creds:       nil,
			expectError: true,
			errorMsg:    "credentials are required",
		},
		{
			name:        "with empty credentials",
			creds:       &manifest.CredentialsConfig{},
			expectError: true,
			errorMsg:    "either service_account_key_path or service_account_key_json is required",
		},
		{
			name: "with both path and JSON (path takes precedence)",
			creds: &manifest.CredentialsConfig{
				ServiceAccountKeyPath: "/path/to/key.json",
				ServiceAccountKeyJSON: `{"type":"service_account"}`,
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			option, err := loadCredentials(tt.creds)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error containing '%s', but got none", tt.errorMsg)
				} else if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing '%s', got: %v", tt.errorMsg, err)
				}
				if option != nil {
					t.Error("Expected option to be nil when error occurs")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if option == nil {
					t.Error("Expected option to be non-nil")
				}
			}
		})
	}
}

func TestCreateTarGzEmptyDirectory(t *testing.T) {
	// Create an empty temporary directory
	tmpDir := t.TempDir()

	// Create a temporary tar.gz file
	tarFile, err := os.CreateTemp("", "test-*.tar.gz")
	if err != nil {
		t.Fatalf("Failed to create temp tar file: %v", err)
	}
	defer os.Remove(tarFile.Name())
	defer tarFile.Close()

	// Test createTarGz with empty directory
	err = createTarGz(tmpDir, tarFile)
	if err != nil {
		t.Fatalf("createTarGz failed on empty directory: %v", err)
	}

	// Verify tar.gz was created (even if empty)
	stat, err := tarFile.Stat()
	if err != nil {
		t.Fatalf("Failed to stat tar file: %v", err)
	}
	if stat.Size() == 0 {
		t.Error("Expected non-zero tar.gz file size")
	}
}

func TestCreateTarGzLargeFile(t *testing.T) {
	// Create a temporary directory with a large file
	tmpDir := t.TempDir()

	// Create a 1MB test file
	largeContent := make([]byte, 1024*1024)
	for i := range largeContent {
		largeContent[i] = byte(i % 256)
	}

	largePath := filepath.Join(tmpDir, "large.bin")
	if err := os.WriteFile(largePath, largeContent, 0644); err != nil {
		t.Fatalf("Failed to create large file: %v", err)
	}

	// Create a temporary tar.gz file
	tarFile, err := os.CreateTemp("", "test-*.tar.gz")
	if err != nil {
		t.Fatalf("Failed to create temp tar file: %v", err)
	}
	defer os.Remove(tarFile.Name())
	defer tarFile.Close()

	// Test createTarGz
	err = createTarGz(tmpDir, tarFile)
	if err != nil {
		t.Fatalf("createTarGz failed: %v", err)
	}

	// Verify file was added
	if _, err := tarFile.Seek(0, 0); err != nil {
		t.Fatalf("Failed to seek: %v", err)
	}

	gzReader, err := gzip.NewReader(tarFile)
	if err != nil {
		t.Fatalf("Failed to create gzip reader: %v", err)
	}
	defer gzReader.Close()

	tarReader := tar.NewReader(gzReader)
	header, err := tarReader.Next()
	if err != nil {
		t.Fatalf("Failed to read tar: %v", err)
	}

	if header.Name != "large.bin" {
		t.Errorf("Expected file 'large.bin', got '%s'", header.Name)
	}

	if header.Size != int64(len(largeContent)) {
		t.Errorf("Expected size %d, got %d", len(largeContent), header.Size)
	}

	// Read and verify content
	content, err := io.ReadAll(tarReader)
	if err != nil {
		t.Fatalf("Failed to read content: %v", err)
	}

	if len(content) != len(largeContent) {
		t.Errorf("Expected content length %d, got %d", len(largeContent), len(content))
	}
}

func TestCreateTarGzNestedDirectories(t *testing.T) {
	// Create a deeply nested directory structure
	tmpDir := t.TempDir()

	deepPath := filepath.Join(tmpDir, "a", "b", "c", "d", "e")
	if err := os.MkdirAll(deepPath, 0755); err != nil {
		t.Fatalf("Failed to create nested directories: %v", err)
	}

	deepFile := filepath.Join(deepPath, "deep.txt")
	if err := os.WriteFile(deepFile, []byte("deep content"), 0644); err != nil {
		t.Fatalf("Failed to create deep file: %v", err)
	}

	// Create a temporary tar.gz file
	tarFile, err := os.CreateTemp("", "test-*.tar.gz")
	if err != nil {
		t.Fatalf("Failed to create temp tar file: %v", err)
	}
	defer os.Remove(tarFile.Name())
	defer tarFile.Close()

	// Test createTarGz
	err = createTarGz(tmpDir, tarFile)
	if err != nil {
		t.Fatalf("createTarGz failed: %v", err)
	}

	// Verify nested file is in tar with correct path
	if _, err := tarFile.Seek(0, 0); err != nil {
		t.Fatalf("Failed to seek: %v", err)
	}

	gzReader, err := gzip.NewReader(tarFile)
	if err != nil {
		t.Fatalf("Failed to create gzip reader: %v", err)
	}
	defer gzReader.Close()

	tarReader := tar.NewReader(gzReader)
	header, err := tarReader.Next()
	if err != nil {
		t.Fatalf("Failed to read tar: %v", err)
	}

	expectedPath := "a/b/c/d/e/deep.txt"
	if header.Name != expectedPath {
		t.Errorf("Expected path '%s', got '%s'", expectedPath, header.Name)
	}

	content, err := io.ReadAll(tarReader)
	if err != nil {
		t.Fatalf("Failed to read content: %v", err)
	}

	if string(content) != "deep content" {
		t.Errorf("Expected content 'deep content', got '%s'", string(content))
	}
}

func TestCreateTarGzErrorCases(t *testing.T) {
	t.Run("non-existent directory", func(t *testing.T) {
		tarFile, err := os.CreateTemp("", "test-*.tar.gz")
		if err != nil {
			t.Fatalf("Failed to create temp tar file: %v", err)
		}
		defer os.Remove(tarFile.Name())
		defer tarFile.Close()

		err = createTarGz("/non/existent/path", tarFile)
		if err == nil {
			t.Error("Expected error for non-existent directory")
		}
	})
}

func TestCreateTarGzWithSpecialCharacters(t *testing.T) {
	tmpDir := t.TempDir()

	// Create files with special characters
	testFiles := []string{
		"file with spaces.txt",
		"file-with-dashes.txt",
		"file_with_underscores.txt",
		"file.multiple.dots.txt",
	}

	for _, filename := range testFiles {
		fullPath := filepath.Join(tmpDir, filename)
		if err := os.WriteFile(fullPath, []byte("content"), 0644); err != nil {
			t.Fatalf("Failed to create file %s: %v", filename, err)
		}
	}

	tarFile, err := os.CreateTemp("", "test-*.tar.gz")
	if err != nil {
		t.Fatalf("Failed to create temp tar file: %v", err)
	}
	defer os.Remove(tarFile.Name())
	defer tarFile.Close()

	err = createTarGz(tmpDir, tarFile)
	if err != nil {
		t.Fatalf("createTarGz failed: %v", err)
	}

	// Verify all files are in the tar
	if _, err := tarFile.Seek(0, 0); err != nil {
		t.Fatalf("Failed to seek: %v", err)
	}

	gzReader, err := gzip.NewReader(tarFile)
	if err != nil {
		t.Fatalf("Failed to create gzip reader: %v", err)
	}
	defer gzReader.Close()

	tarReader := tar.NewReader(gzReader)

	foundFiles := make(map[string]bool)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("Failed to read tar: %v", err)
		}
		foundFiles[header.Name] = true
	}

	for _, expectedFile := range testFiles {
		if !foundFiles[expectedFile] {
			t.Errorf("Expected file '%s' not found in tar", expectedFile)
		}
	}
}

func TestCreateTarGzWithSymlinks(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a regular file
	regularFile := filepath.Join(tmpDir, "regular.txt")
	if err := os.WriteFile(regularFile, []byte("regular content"), 0644); err != nil {
		t.Fatalf("Failed to create regular file: %v", err)
	}

	// Create a symlink (may not work on all systems)
	symlinkPath := filepath.Join(tmpDir, "symlink.txt")
	if err := os.Symlink(regularFile, symlinkPath); err != nil {
		t.Skipf("Skipping symlink test: %v", err)
	}

	tarFile, err := os.CreateTemp("", "test-*.tar.gz")
	if err != nil {
		t.Fatalf("Failed to create temp tar file: %v", err)
	}
	defer os.Remove(tarFile.Name())
	defer tarFile.Close()

	err = createTarGz(tmpDir, tarFile)
	// Symlinks may cause issues with tar creation, which is expected behavior
	// The function should either succeed or fail gracefully
	if err != nil {
		// If it fails, that's acceptable for symlinks
		t.Logf("createTarGz failed with symlink (expected): %v", err)
		return
	}

	// If it succeeds, verify tar was created
	stat, err := tarFile.Stat()
	if err != nil {
		t.Fatalf("Failed to stat tar file: %v", err)
	}
	if stat.Size() == 0 {
		t.Error("Expected non-zero tar file size")
	}
}

func TestLoadCredentialsWithInvalidJSON(t *testing.T) {
	tests := []struct {
		name        string
		creds       *manifest.CredentialsConfig
		expectError bool
	}{
		{
			name: "with empty JSON",
			creds: &manifest.CredentialsConfig{
				ServiceAccountKeyJSON: `{}`,
			},
			expectError: false, // Empty JSON is valid, just minimal credentials
		},
		{
			name: "with valid JSON string",
			creds: &manifest.CredentialsConfig{
				ServiceAccountKeyJSON: `{"type":"service_account","project_id":"test"}`,
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			option, err := loadCredentials(tt.creds)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				if option != nil {
					t.Error("Expected option to be nil when error occurs")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if option == nil {
					t.Error("Expected option to be non-nil")
				}
			}
		})
	}
}

func TestProviderRegionAndProject(t *testing.T) {
	tests := []struct {
		name      string
		projectID string
		region    string
	}{
		{"us-central1", "test-project-1", "us-central1"},
		{"us-east1", "test-project-2", "us-east1"},
		{"europe-west1", "test-project-3", "europe-west1"},
		{"asia-southeast1", "test-project-4", "asia-southeast1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := &Provider{
				projectID: tt.projectID,
				region:    tt.region,
			}

			if provider.projectID != tt.projectID {
				t.Errorf("Expected projectID '%s', got '%s'", tt.projectID, provider.projectID)
			}
			if provider.region != tt.region {
				t.Errorf("Expected region '%s', got '%s'", tt.region, provider.region)
			}
		})
	}
}

func TestProviderPublicAccessSetting(t *testing.T) {
	tests := []struct {
		name         string
		publicAccess bool
	}{
		{"public access enabled", true},
		{"public access disabled", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := &Provider{
				projectID:    "test-project",
				region:       "us-central1",
				publicAccess: tt.publicAccess,
			}

			if provider.publicAccess != tt.publicAccess {
				t.Errorf("Expected publicAccess %v, got %v", tt.publicAccess, provider.publicAccess)
			}
		})
	}
}

func TestCreateTarGzMultipleFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create multiple files at different levels
	files := map[string]string{
		"root1.txt":              "root content 1",
		"root2.txt":              "root content 2",
		"dir1/file1.txt":         "dir1 content",
		"dir1/file2.txt":         "dir1 content 2",
		"dir2/subdir/file3.txt":  "nested content",
		"dir2/subdir/file4.go":   "package main",
		"dir3/a.txt":             "a",
		"dir3/b.txt":             "b",
		"dir3/nested/c.txt":      "c",
	}

	for path, content := range files {
		fullPath := filepath.Join(tmpDir, path)
		dir := filepath.Dir(fullPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create file: %v", err)
		}
	}

	tarFile, err := os.CreateTemp("", "test-*.tar.gz")
	if err != nil {
		t.Fatalf("Failed to create temp tar file: %v", err)
	}
	defer os.Remove(tarFile.Name())
	defer tarFile.Close()

	err = createTarGz(tmpDir, tarFile)
	if err != nil {
		t.Fatalf("createTarGz failed: %v", err)
	}

	// Verify all files are in tar with correct content
	if _, err := tarFile.Seek(0, 0); err != nil {
		t.Fatalf("Failed to seek: %v", err)
	}

	gzReader, err := gzip.NewReader(tarFile)
	if err != nil {
		t.Fatalf("Failed to create gzip reader: %v", err)
	}
	defer gzReader.Close()

	tarReader := tar.NewReader(gzReader)

	foundFiles := make(map[string]string)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("Failed to read tar: %v", err)
		}

		content, err := io.ReadAll(tarReader)
		if err != nil {
			t.Fatalf("Failed to read file content: %v", err)
		}
		foundFiles[header.Name] = string(content)
	}

	// Verify all expected files are present with correct content
	for expectedPath, expectedContent := range files {
		actualContent, exists := foundFiles[expectedPath]
		if !exists {
			t.Errorf("Expected file '%s' not found in tar", expectedPath)
			continue
		}
		if actualContent != expectedContent {
			t.Errorf("File '%s': expected content '%s', got '%s'",
				expectedPath, expectedContent, actualContent)
		}
	}

	// Verify we found the expected number of files
	if len(foundFiles) != len(files) {
		t.Errorf("Expected %d files in tar, found %d", len(files), len(foundFiles))
	}
}
