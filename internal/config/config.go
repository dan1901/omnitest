package config

import (
	"fmt"
	"os"
	"regexp"

	"gopkg.in/yaml.v3"

	"github.com/omnitest/omnitest/pkg/model"
)

var envVarRe = regexp.MustCompile(`\$\{(\w+)\}`)

// Load는 YAML 파일을 읽고 파싱하여 TestConfig를 반환한다.
// 환경변수 ${VAR} 치환을 수행한 후 스키마 검증을 실행한다.
func Load(path string) (*model.TestConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("✗ Error: failed to read scenario file\n  → %s: %w\n  → Check the file path and permissions", path, err)
	}

	expanded := expandEnvVars(string(data))

	var cfg model.TestConfig
	if err := yaml.Unmarshal([]byte(expanded), &cfg); err != nil {
		return nil, fmt.Errorf("✗ Error: failed to parse scenario file\n  → %s: %w\n  → Check YAML syntax", path, err)
	}

	if err := validate(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// validate는 TestConfig의 필수 필드와 참조 무결성을 검증한다.
func validate(cfg *model.TestConfig) error {
	if len(cfg.Targets) == 0 {
		return fmt.Errorf("✗ Error: invalid scenario configuration\n  → at least one target is required\n  → Add a target with name and base_url")
	}
	if len(cfg.Scenarios) == 0 {
		return fmt.Errorf("✗ Error: invalid scenario configuration\n  → at least one scenario is required\n  → Add a scenario with name, target, vusers, duration, and requests")
	}

	targetNames := make(map[string]bool, len(cfg.Targets))
	for i, t := range cfg.Targets {
		if t.Name == "" {
			return fmt.Errorf("✗ Error: invalid scenario configuration\n  → target[%d]: name is required\n  → Set a unique name for each target", i)
		}
		if t.BaseURL == "" {
			return fmt.Errorf("✗ Error: invalid scenario configuration\n  → target[%d] %q: base_url is required\n  → Set the base URL (e.g., \"https://api.example.com\")", i, t.Name)
		}
		if targetNames[t.Name] {
			return fmt.Errorf("✗ Error: invalid scenario configuration\n  → target[%d]: duplicate target name %q\n  → Each target must have a unique name", i, t.Name)
		}
		targetNames[t.Name] = true
	}

	for i, s := range cfg.Scenarios {
		if s.Name == "" {
			return fmt.Errorf("✗ Error: invalid scenario configuration\n  → scenario[%d]: name is required", i)
		}
		if s.Target == "" {
			return fmt.Errorf("✗ Error: invalid scenario configuration\n  → scenario[%d] %q: target is required\n  → Reference a target name defined in targets section", i, s.Name)
		}
		if !targetNames[s.Target] {
			return fmt.Errorf("✗ Error: invalid scenario configuration\n  → scenario[%d] %q: target %q not found in targets\n  → Available targets: check your targets section", i, s.Name, s.Target)
		}
		if s.VUsers <= 0 {
			return fmt.Errorf("✗ Error: invalid scenario configuration\n  → scenario[%d] %q: vusers must be > 0 (got %d)\n  → Set a positive integer for vusers", i, s.Name, s.VUsers)
		}
		if s.Duration <= 0 {
			return fmt.Errorf("✗ Error: invalid scenario configuration\n  → scenario[%d] %q: duration must be > 0\n  → Set a valid duration (e.g., \"30s\", \"5m\")", i, s.Name)
		}
		if len(s.Requests) == 0 {
			return fmt.Errorf("✗ Error: invalid scenario configuration\n  → scenario[%d] %q: at least one request is required\n  → Add requests with method and path", i, s.Name)
		}
		for j, r := range s.Requests {
			if r.Method == "" {
				return fmt.Errorf("✗ Error: invalid scenario configuration\n  → scenario[%d] %q, request[%d]: method is required\n  → Use GET, POST, PUT, or DELETE", i, s.Name, j)
			}
			if r.Path == "" {
				return fmt.Errorf("✗ Error: invalid scenario configuration\n  → scenario[%d] %q, request[%d]: path is required\n  → Set the request path (e.g., \"/users\")", i, s.Name, j)
			}
		}
	}

	return nil
}

// LoadFromString은 YAML 문자열을 직접 파싱하여 TestConfig를 반환한다.
// Agent 모드에서 Controller로부터 받은 scenario_yaml을 파싱할 때 사용한다.
func LoadFromString(data string) (*model.TestConfig, error) {
	expanded := expandEnvVars(data)

	var cfg model.TestConfig
	if err := yaml.Unmarshal([]byte(expanded), &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse scenario YAML: %w", err)
	}

	if err := validate(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// expandEnvVars는 문자열 내 ${VAR}를 os.Getenv로 치환한다.
func expandEnvVars(s string) string {
	return envVarRe.ReplaceAllStringFunc(s, func(match string) string {
		key := envVarRe.FindStringSubmatch(match)[1]
		if val, ok := os.LookupEnv(key); ok {
			return val
		}
		return match
	})
}
