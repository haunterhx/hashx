package ioc

import (
	"bufio"
	"os"
	"strings"
)

func LoadIOCs(path string) (map[string]struct{}, error) {
	set := make(map[string]struct{})

	if path == "" {
		return set, nil
	}

	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	defer f.Close()

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.ToLower(strings.TrimSpace(sc.Text()))
		if line == "" || strings.HasPrefix(line, "#cl") {
			continue
		}
		if len(line) == 64 {
			set[line] = struct{}{}
		}
	}

	return set, sc.Err()
}
