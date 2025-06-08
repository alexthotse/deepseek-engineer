package fileops

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Helper function to create a temporary directory for testing
func setupTestDir(t *testing.T) (string, func()) {
	// Base temp directory within /app for consistency with NormalizePath checks
	baseAppDir := "/app/tmp"
	if err := os.MkdirAll(baseAppDir, 0755); err != nil {
		// Fallback to os.TempDir() if /app/tmp cannot be created (e.g. permission issues in some environments)
		baseAppDir = os.TempDir()
	}

	tmpDir, err := os.MkdirTemp(baseAppDir, "fileops_test_")
	if err != nil {
		t.Fatalf("Failed to create temp dir for testing: %v", err)
	}

	// Ensure the created tmpDir is "within" /app for NormalizePath if baseAppDir was /app/tmp
	// If baseAppDir defaulted to os.TempDir(), this might not align with NormalizePath's current strictness.
	// This highlights a potential area for improvement in NormalizePath or test setup for broader compatibility.
	// For now, we proceed assuming tmpDir is acceptable to NormalizePath or tests will adapt.

	return tmpDir, func() {
		os.RemoveAll(tmpDir)
	}
}


func TestNormalizePath(t *testing.T) {
	// Assuming /app is the root for these tests, as per NormalizePath's current logic.
	// Create a dummy structure that NormalizePath expects.
	// This is a bit of a workaround for NormalizePath's strictness.
	// A more flexible NormalizePath would make testing easier.
	if err := os.MkdirAll("/app/testdir/subdir", 0755); err != nil {
		t.Fatalf("Failed to create dummy /app/testdir/subdir: %v", err)
	}
	defer os.RemoveAll("/app/testdir")


	validPath := "/app/testdir/./subdir/../file.txt" // Should resolve to /app/testdir/file.txt
	expectedValidPath := "/app/testdir/file.txt"

	// Create the actual file so Abs can work without issues on some systems if it resolves symlinks etc.
	// For NormalizePath, it primarily works with string manipulation after Abs, but good practice.
	if err := os.WriteFile(expectedValidPath, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create dummy file for NormalizePath test: %v", err)
	}
	defer os.Remove(expectedValidPath)


	normalized, err := NormalizePath(validPath)
	if err != nil {
		t.Errorf("NormalizePath(%q) returned error: %v", validPath, err)
	}
	if normalized != expectedValidPath {
		t.Errorf("NormalizePath(%q) = %q, want %q", validPath, normalized, expectedValidPath)
	}

	// Test relative path from within a simulated "/app" context
	// This requires the test to be run with a working directory that makes sense for "/app"
	// Or, more robustly, make NormalizePath aware of a base project directory.
	// For now, let's assume tests run where "/app/..." makes sense.
	// Note: This part of the test might be flaky depending on execution environment
	// if /app isn't the true root or accessible as such.
	// The function itself uses filepath.Abs which resolves based on CWD.
	// Let's test with a path that should be resolvable relative to /app.

	// Create a file in a known sub-path of /app for this test
	relTestPathDir := "/app/reltest"
	relTestFile := filepath.Join(relTestPathDir, "test_relative.txt")
	if err := os.MkdirAll(relTestPathDir, 0755); err != nil {
		t.Fatalf("Failed to create dir for relative path test: %v", err)
	}
	if err := os.WriteFile(relTestFile, []byte("content"), 0644); err != nil {
		t.Fatalf("Failed to create file for relative path test: %v", err)
	}
	defer os.RemoveAll(relTestPathDir)

	// Test a relative path that, when Abs is called, should resolve under /app
	// This relies on Abs behavior. If CWD is /app, then "reltest/test_relative.txt" is fine.
	// If CWD is /app/deepseek-go, then "../reltest/test_relative.txt" could be used.
	// Let's assume CWD is /app for this conceptual test.
	// A better way is to pass a "projectRoot" to NormalizePath.
	// Given the current implementation, we test with an absolute-like path that we know is in /app.

	normalizedRel, err := NormalizePath(relTestFile) // Using the absolute path we know is good
	if err != nil {
		t.Errorf("NormalizePath for relative-like path %q returned error: %v", relTestFile, err)
	}
	if normalizedRel != relTestFile { // Expecting it to be cleaned to itself
		t.Errorf("NormalizePath for relative-like path %q = %q, want %q", relTestFile, normalizedRel, relTestFile)
	}


	// Test path traversal attempts
	// These paths are designed to try and escape an assumed root, even if Abs makes them absolute first.
	// NormalizePath's explicit ".." check after Clean is what we're testing here.
	// The key is that filepath.Clean might resolve "/app/testdir/../../../../etc/passwd" to "/etc/passwd",
	// and then the HasPrefix("/app") and ".." checks in NormalizePath kick in.

	// This path, after Abs and Clean, might become /app/some/other/../../../../etc/passwd -> /etc/passwd
	// This will be caught by !strings.HasPrefix(cleanedPath, "/app")
	travPath1 := "/app/some/other/../../../../etc/passwd"
	_, err = NormalizePath(travPath1)
	if err == nil {
		t.Errorf("NormalizePath(%q) did not return an error for traversal attempt", travPath1)
	} else if !strings.Contains(err.Error(), "is outside the allowed project directory") && !strings.Contains(err.Error(), "path traversal attempt detected") {
		t.Errorf("NormalizePath(%q) error %q, want error containing 'is outside the allowed project directory' or 'path traversal attempt detected'", travPath1, err.Error())
	}

	// This path attempts to use ".." after being within /app.
	// e.g. /app/../otherdir. filepath.Clean makes this /otherdir.
	// This is caught by !strings.HasPrefix(cleanedPath, "/app")
	travPath2 := "/app/mydir/../../otherdir/file" // Should become /otherdir/file
	_, err = NormalizePath(travPath2)
	if err == nil {
		t.Errorf("NormalizePath(%q) did not return an error for traversal attempt", travPath2)
	} else if !strings.Contains(err.Error(), "is outside the allowed project directory") {
         // Check if the error is the one we expect from the HasPrefix check
		t.Errorf("NormalizePath(%q) error %q, want error containing 'is outside the allowed project directory'", travPath2, err.Error())
	}


	// This path might be cleaned to something like "/app/../file" -> "/file"
	// which would then be caught by the prefix check.
	travPath3 := "../../../etc/passwd"
	// If CWD is /app/deepseek-go, Abs makes it /app/deepseek-go/../../../etc/passwd -> /etc/passwd
	// This will be caught by !strings.HasPrefix(cleanedPath, "/app")
	_, err = NormalizePath(travPath3)
	if err == nil {
		t.Errorf("NormalizePath(%q) did not return an error for traversal attempt from relative", travPath3)
	} else if !strings.Contains(err.Error(), "is outside the allowed project directory") {
		t.Errorf("NormalizePath(%q) error %q, want error containing 'is outside the allowed project directory'", travPath3, err.Error())
	}

	// Test path that results in ".." after cleaning (edge case, Clean should handle this mostly)
	// This specific test might be tricky as `filepath.Clean` is quite good.
	// The custom ".." check in NormalizePath is for additional safety layer or specific patterns.
	// Example: "/app/..something/file" -> cleaned might be "/app/..something/file" if "..something" is a valid dir name
	// But "/app/../file" -> "/file" (caught by prefix check)
	// And "/app/./../file" -> "/file" (caught by prefix check)
	// The specific ".." segment check seems more relevant if `Clean` somehow misses a logical ".."
	// or if a path like "/app/legit/../nefarious../file" was constructed where "nefarious.." itself is the issue.
	// For now, the existing HasPrefix check covers most direct traversal out of /app.
}


func TestReadLocalFile(t *testing.T) {
	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	filePath := filepath.Join(tmpDir, "testfile.txt")
	content := "Hello, World!"

	// Create a dummy file in /app as NormalizePath expects paths to be there.
	// This is a bit of a hack due to NormalizePath's strictness.
	// We'll use the tmpDir which is now created under /app/tmp or os.TempDir()
	// If it's under os.TempDir() and not /app, NormalizePath will fail.
	// This test relies on setupTestDir placing tmpDir in a location NormalizePath considers valid.
	// Let's assume setupTestDir correctly places tmpDir under /app/tmp for this test.

	// Adjust filePath to be "within" /app for NormalizePath
	// This assumes tmpDir is something like /app/tmp/fileops_test_XXXX
	// If tmpDir is /tmp/fileops_test_XXXX, NormalizePath will reject it.
	// This is a known limitation of current NormalizePath or test setup.
	// For this test to pass with current NormalizePath, tmpDir *must* be under /app.

	// If tmpDir is not under /app, this test will fail at NormalizePath.
	// We can proceed and see, or make NormalizePath more flexible, or ensure tmpDir is always under /app.
	// The setupTestDir tries to use /app/tmp.

	err := os.WriteFile(filePath, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	readContent, err := ReadLocalFile(filePath)
	if err != nil {
		t.Fatalf("ReadLocalFile failed for existing file: %v", err)
	}
	if readContent != content {
		t.Errorf("ReadLocalFile content = %q, want %q", readContent, content)
	}

	_, err = ReadLocalFile(filepath.Join(tmpDir, "nonexistent.txt"))
	if err == nil {
		t.Errorf("ReadLocalFile succeeded for non-existing file, want error")
	}
}

func TestCreateFile(t *testing.T) {
	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	filePath := filepath.Join(tmpDir, "newfile.txt")
	content := "This is a new file."

	err := CreateFile(filePath, content)
	if err != nil {
		t.Fatalf("CreateFile failed: %v", err)
	}

	readBytes, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Failed to read created file: %v", err)
	}
	if string(readBytes) != content {
		t.Errorf("CreateFile content = %q, want %q", string(readBytes), content)
	}

	// Test creating in a subdirectory
	subDirPath := filepath.Join(tmpDir, "subdir", "newfile.txt")
	err = CreateFile(subDirPath, content)
	if err != nil {
		t.Fatalf("CreateFile in subdir failed: %v", err)
	}
	readBytes, err = os.ReadFile(subDirPath)
	if err != nil {
		t.Fatalf("Failed to read created file in subdir: %v", err)
	}
	if string(readBytes) != content {
		t.Errorf("CreateFile in subdir content = %q, want %q", string(readBytes), content)
	}

	// Test file size limit
	largeContent := strings.Repeat("a", MaxFileSize+1)
	err = CreateFile(filepath.Join(tmpDir, "largefile.txt"), largeContent)
	if err == nil {
		t.Errorf("CreateFile succeeded for oversized file, want error")
	} else if !strings.Contains(err.Error(), "exceeds maximum file size") {
		t.Errorf("CreateFile error for oversized file was %q, want 'exceeds maximum file size'", err.Error())
	}
}

func TestIsBinaryFile(t *testing.T) {
	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	textFile := filepath.Join(tmpDir, "text.txt")
	binaryFile := filepath.Join(tmpDir, "binary.bin")

	os.WriteFile(textFile, []byte("This is a text file."), 0644)
	os.WriteFile(binaryFile, []byte{'b', 'i', 'n', '\x00', 'a', 'r', 'y'}, 0644)

	isBin, err := IsBinaryFile(textFile)
	if err != nil {
		t.Fatalf("IsBinaryFile failed for text file: %v", err)
	}
	if isBin {
		t.Errorf("IsBinaryFile(textFile) = true, want false")
	}

	isBin, err = IsBinaryFile(binaryFile)
	if err != nil {
		t.Fatalf("IsBinaryFile failed for binary file: %v", err)
	}
	if !isBin {
		t.Errorf("IsBinaryFile(binaryFile) = false, want true")
	}

	_, err = IsBinaryFile(filepath.Join(tmpDir, "nonexistent.txt"))
	if err == nil {
		t.Errorf("IsBinaryFile succeeded for non-existing file, want error")
	}
}

func TestApplyDiffEdit(t *testing.T) {
	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	filePath := filepath.Join(tmpDir, "editme.txt")
	originalContent := "Hello, old world!\nThis is a test.\nOld snippet here."
	originalSnippet := "old world"
	newSnippet := "new world"
	expectedContent := "Hello, new world!\nThis is a test.\nOld snippet here."

	err := CreateFile(filePath, originalContent) // Use CreateFile to ensure path normalization
	if err != nil {
		t.Fatalf("Failed to create file for ApplyDiffEdit test: %v", err)
	}

	err = ApplyDiffEdit(filePath, originalSnippet, newSnippet)
	if err != nil {
		t.Fatalf("ApplyDiffEdit failed: %v", err)
	}

	readBytes, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Failed to read file after ApplyDiffEdit: %v", err)
	}
	if string(readBytes) != expectedContent {
		t.Errorf("ApplyDiffEdit content = %q, want %q", string(readBytes), expectedContent)
	}

	// Test snippet not found
	err = ApplyDiffEdit(filePath, "nonexistent snippet", "irrelevant")
	if err == nil {
		t.Errorf("ApplyDiffEdit succeeded with nonexistent snippet, want error")
	} else if !strings.Contains(err.Error(), "original snippet not found") {
		t.Errorf("ApplyDiffEdit error %q, want 'original snippet not found'", err.Error())
	}

	// Test multiple snippets (should replace first, current behavior)
	multiPath := filepath.Join(tmpDir, "multiedit.txt")
	multiOriginal := "abc 123 abc 456"
	multiExpected := "xyz 123 abc 456"
	CreateFile(multiPath, multiOriginal)

	err = ApplyDiffEdit(multiPath, "abc", "xyz")
	if err != nil {
		t.Fatalf("ApplyDiffEdit failed for multiple snippets: %v", err)
	}
	readBytes, err = os.ReadFile(multiPath)
	if err != nil {
		t.Fatalf("Failed to read file after multi-snippet ApplyDiffEdit: %v", err)
	}
	if string(readBytes) != multiExpected {
		t.Errorf("ApplyDiffEdit for multi-snippet content = %q, want %q", string(readBytes), multiExpected)
	}

	// Test ApplyDiffEdit on a non-existent file
	err = ApplyDiffEdit(filepath.Join(tmpDir,"nonexistent_edit.txt"), "a", "b")
	if err == nil {
		t.Errorf("ApplyDiffEdit succeeded on non-existent file, want error")
	}
}
