# NeKiro Samples RepoWiki

这里是独立 sample Agent Runtime 的中英文 RepoWiki 入口。仓库 README 是
canonical source，MkDocs 在 CI 中从这些文档生成页面，不复制 Core 或 SDK 源码。

## 从这里开始

- [源文档](source-docs/index.md)：仓库总览、Runtime A 和 Runtime B 指南。
- [GitHub 仓库](https://github.com/NeKiro-project/NeKiro-Samples)：源码、Issue 和 Release。
- [Core RepoWiki](https://nekiro-project.github.io/NeKiro/zh/)：平台契约与架构。

Runtime A 使用 `trpc-agent-go`，Runtime B 使用直接的 `a2a-go` server；两者
都只能通过 Router 调用其他 Agent，并使用公共 Go SDK。
