package blog_utility

import (
	"fmt"
	"strings"

	"github.com/russross/blackfriday/v2"
	"gopkg.in/yaml.v2"
)

type FrontMatter struct {
	Title        string `yaml:"title"`
	PublishedAt  string `yaml:"publishedAt"`
	Summary      string `yaml:"summary"`
	Image        string `yaml:"image"`
	PreviewImage string `yaml:"previewImage"`
}

func ParseFrontMatter(content string) (FrontMatter, string, error) {
	var fm FrontMatter

	parts := strings.SplitN(content, "---", 3)
	if len(parts) < 3 {
		return fm, "", fmt.Errorf("invalid front matter")
	}

	if err := yaml.Unmarshal([]byte(parts[1]), &fm); err != nil {
		return fm, "", err
	}

	return fm, strings.TrimSpace(parts[2]), nil
}

func ConvertMarkdownToHTML(markdown string) string {
	output := blackfriday.Run([]byte(markdown))
	return string(output)
}
