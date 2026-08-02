# Monorepo architecture

```text
stats/
├─ collector/                 SourceMod collector source
│  ├─ src/                    single .sp entry point
│  └─ include/                compile-time .inc modules
├─ config/                    public configuration examples
├─ contracts/                 confirmed gameplay/data contracts
├─ database/
│  ├─ migrations/             canonical SQL migrations per driver
│  └─ schema.md               logical schema contract
├─ dashboard/                 future Go + embedded web application
├─ docs/                      operator and development documentation
├─ scripts/                   build, validation, deploy, package tools
└─ dist/                      generated artifacts (ignored)
```

The collector source is modular but compiles into one plugin binary. Runtime
SQL migrations remain external files so the SQL stored in the repository is the
same SQL executed by SourceMod. Build artifacts never become source of truth.

Local machine paths belong in ignored `scripts/config.local.ps1`. Database
credentials belong only in the server's SourceMod `databases.cfg`.

