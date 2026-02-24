---
description: Review uncommitted changes
agent: plan
subtask: true
---
Review all uncommitted changes in this repository.

!`git diff --stat`
!`git diff`

Check for:
- Convention violations (see docs/agents/agent-rules.md)
- Known pitfalls (see docs/agents/pitfalls.md)
- Missing tests for new logic
- Magic constants or leaked PBM types

Provide a concise summary of findings.
