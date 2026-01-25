package files

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func ReadFile(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("error reading file: %s %s", path, err)
	}
	return data, nil
}

func LoadEnv(fileName string) error {
	file, err := os.Open(fileName)
	if err != nil {
		return fmt.Errorf("error while opening: %s error: %s", fileName, err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		// Check if its comments
		if strings.HasPrefix(line, "#") || strings.TrimSpace(line) == "" {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		value = strings.Trim(value, "\"")
		err := os.Setenv(key, value)
		if err != nil {
			return err
		}
		if err := scanner.Err(); err != nil {
			return err
		}
	}
	return nil
}
