# NeKiro Samples RepoWiki

This is the bilingual reading surface for the independent sample Agent
Runtimes. The repository README files remain canonical; MkDocs renders them
without copying Core or SDK source into this repository.

## Start here

- [Source documentation](source-docs/index.md) — repository and runtime guides.
- [GitHub repository](https://github.com/NeKiro-project/NeKiro-Samples) — source, issues, and releases.
- [Core RepoWiki](https://nekiro-project.github.io/NeKiro/) — platform contracts and architecture.

Runtime A uses `trpc-agent-go`; Runtime B uses a direct `a2a-go` server. Both
call other Agents only through the Router and use the public Go SDK.
