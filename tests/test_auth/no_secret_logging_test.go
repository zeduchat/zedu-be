package test_auth

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// Logger calls must never include credential fields such as a bcrypt hash.
func TestNoCredentialFieldsInLogCalls(t *testing.T) {
	re := regexp.MustCompile(`[Ll]ogger\.(Error|Info|Warn|Warning|Debug|Fatal)\([^)]*\.(Password|Token|Hash|Secret)\b`)

	for _, root := range []string{"../../services", "../../pkg", "../../internal", "../../utility"} {
		err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") {
				return err
			}
			f, err := os.Open(path)
			if err != nil {
				return err
			}
			defer f.Close()

			sc := bufio.NewScanner(f)
			sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
			for line := 1; sc.Scan(); line++ {
				if re.MatchString(sc.Text()) {
					t.Errorf("%s:%d logs a credential field: %s", path, line, strings.TrimSpace(sc.Text()))
				}
			}
			return sc.Err()
		})
		if err != nil {
			t.Fatalf("walk %s: %v", root, err)
		}
	}
}
