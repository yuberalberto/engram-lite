# add-cascade-workspace-binding Issues

## Dependency Graph

```
┌────────┐ ✅  ┌────────┐ ✅
│ 002-01 │     │ 002-03 │
└────────┘     └────────┘
    │
    └─→ ┌────────┐ ✅
        │ 002-02 │
        └────────┘
```

## Status Table

| Issue  | Title                              | Type | Status  | Blocked By |
|--------|------------------------------------|------|---------|------------|
| 002-01 | mem_use_workspace Tool — Core      | AFK  | done    | -          |
| 002-02 | Write Tool Enforcement             | AFK  | done    | 002-01     |
| 002-03 | Rule Injection in engram-lite init | AFK  | done    | -          |
