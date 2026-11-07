# AGENTS.md

本文件约束在 RClick 仓库中工作的代理。RClick 是 macOS FinderSync 扩展的 Go/Swift 桌面应用；目标是在不破坏跨进程配置契约和本机文件行为的前提下完成变更。

## 阅读顺序

1. [README.md](README.md)：构建、安装、配置和运行限制。
2. [docs/README.md](docs/README.md)：文档导航与编号规则。
3. [功能清单](docs/功能/02-功能清单.md) 与 [项目架构](docs/功能/01-项目架构与模块.md)。
4. [领域术语表](docs/功能/03-领域术语表.md) 和相关流程文档。
5. 修改目标附近的 Go/Swift 源码与测试。

## 仓库地图

| 位置 | 职责 |
| --- | --- |
| `cmd/rclick/` | Go 应用入口 |
| `internal/app/` | 生命周期、UI 与消息路由 |
| `internal/config/` | 配置契约、校验、持久化 |
| `internal/ipc/` | Go/helper IPC |
| `internal/services/` | 应用打开、文件操作、命令执行 |
| `native/finder/` | FinderSync 扩展 |
| `native/helper/` | macOS 原生 helper |
| `docs/` | 架构、功能与流程文档 |

## 常用命令

```bash
make build
make test
make vet
make fmt
```

`make helper`、`make finder`、`make package`、`make install` 会编译/打包或改变本机安装状态，需要 macOS/Xcode；执行安装、注册或其他系统级操作前应确认影响范围。

## 变更约束

- 默认不引入与需求无关的重构；文档任务不改业务代码。
- `internal/config/types.go` 中的 JSON 键与 Swift `SharedConfigStore` 是跨语言契约，变更时同步检查两侧和现有配置兼容性。
- 快捷命令在本机 shell 执行；保留目标路径校验和危险命令确认，不把确认机制描述为沙盒。
- 文件删除、隐藏、AirDrop 和权限目录行为须依据实现与系统验证，不凭界面名称推断。
- 测试应使用 `go test ./...`；补充功能/缺陷修复时，按改动范围增加回归测试。不要通过跳过测试或放宽断言来获得绿色结果。
- 不记录或提交真实配置、令牌、密钥、用户路径、敏感命令内容。
- 只有明确要求时才提交或推送；不得执行未经确认的安装、扩展注册或破坏性本机操作。

## 文档与完成标准

- 功能清单按源码统计入口，明确统计分母；未观察到的行为标“待确认”。
- 新流程使用 `docs/流程/{两位编号}-{中文名称}-v{N}.md`；只有流程语义改变才升版本。
- 文档变更同步更新 `docs/README.md`；相对链接、表格和 Mermaid 应验证。
- 完成说明仅报告实际执行的构建、测试与渲染结果，并列明未执行项。
