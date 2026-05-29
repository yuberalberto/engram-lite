# Add IDE Workspace Configs — Issues

## Dependency Graph

┌──────────┐
│ 001-01   │  IDEConfigWriter: detection and file operations
└──────────┘
     │
     ├─→ ┌──────────┐
     │   │ 001-02   │  IDEConfigWriter: interactive prompt
     │   └──────────┘
     │        │
     └────────┴─→ ┌──────────┐
                  │ 001-03   │  cmdInit: wire and fix re-run
                  └──────────┘
                       │
                       └─→ ┌──────────┐
                           │ 001-04   │  README: replace global config instructions
                           └──────────┘

## Status Table

| Issue  | Title | Type | Status | Blocked By |
|--------|-------|------|--------|------------|
| 001-01 | IDEConfigWriter: detection and file operations | AFK | done | - |
| 001-02 | IDEConfigWriter: interactive prompt | AFK | done | 001-01 |
| 001-03 | cmdInit: wire and fix re-run | AFK | done | 001-01, 001-02 |
| 001-04 | README: replace global config instructions | AFK | done | 001-03 |
