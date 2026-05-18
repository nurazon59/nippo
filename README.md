# go-template

`go-template` is a small Go CLI template with a thin `kong`-based entrypoint, XDG-backed config loading, and release automation for GitHub Releases.

## Usage

```bash
go run ./cmd/template/main.go --help
go run ./cmd/template/main.go --version
```

### Config

The CLI loads config from one of these locations:

1. `--config /path/to/config.yaml`
2. `GO_TEMPLATE_CONFIG=/path/to/config.yaml`
3. `~/.config/go-template/config.yaml`

Config format:

```yaml
version: 1
```

If the file is missing, the CLI starts with defaults.

### シェル補完

`nippo completion <shell>` で bash/zsh/fish 用の補完スクリプトを生成する。

```bash
# bash (現在のシェルで読み込む)
source <(nippo completion bash)

# bash (永続化)
nippo completion bash > /etc/bash_completion.d/nippo

# zsh (fpath 上のディレクトリに配置)
nippo completion zsh > "${fpath[1]}/_nippo"

# fish
nippo completion fish > ~/.config/fish/completions/nippo.fish
```

## Development

```bash
task fmt
task test
task lint
task vet
task ci
task snapshot
```
