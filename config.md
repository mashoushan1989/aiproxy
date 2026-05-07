# YAML Configuration

Priority: **Env Vars** > **YAML** (`config.yaml` or `CONFIG_FILE_PATH`) > **Database**

```yaml
channels:
  - name: "openai-primary"
    type_name: "openai"    # see core/model/yaml_integration.go for all types
    key: "sk-xxx"
    base_url: "https://api.openai.com"
    models: ["gpt-4"]
    sets: ["default"]

modelconfigs:
  - model: "gpt-4"
    owner: "openai"
    type_name: "chat"      # see core/model/yaml_integration.go for all types
    rpm: 3500
    tpm: 80000
    price: { input: 0.03, output: 0.06 }

options:                    # all values must be strings; see core/common/config/env.go
  LogStorageHours: "168"
  RetryTimes: "3"
```

YAML loaded at startup, overrides DB. Web UI changes stored in DB, overridden by YAML on restart.
