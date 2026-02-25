# Pitfalls & Lessons Learned

> **Agents: append to this file** when you discover a non-obvious bug, surprising behavior, or a pattern that future work should be aware of. Include what went wrong, why, and the fix.

## SDK Pitfalls

| Pitfall | What went wrong | Fix |
|---------|----------------|-----|
| Goroutine leak in LogService.Follow | Follow adapter didn't select on context cancellation alongside channel reads. Goroutine lived forever after caller cancelled. | `nextLogCmd` must select on both the entries channel and `ctx.Done()`. |
| PITR duration truncation | Used `Truncate(1)` instead of `Truncate(time.Second)`. `Truncate(1)` truncates to 1 nanosecond, which is a no-op. | Always use `time.Duration` constants with `Truncate`. |
| Stale timeline cursor | Compared pointers (`cursor == &item`) instead of values. After data refresh, pointer identity changes even if the item is logically the same. | Track selection by value identity (name, key), not pointer or index. |
| ConfigName empty string | PBM uses `""` for main config. If SDK doesn't normalize, consumers see empty strings and write broken comparisons. | SDK normalizes to `MainConfig` constant. Never check for `""` in TUI code. |
| PITR restore base backup | SDK requires `BackupName` for `StartPITRRestore` (PBM CLI auto-selects). Decision to add SDK auto-selection is postponed. | TUI handles it via `findBaseBackup()` — selects latest completed backup before target time from cached data. |

## TUI Pitfalls

| Pitfall | What went wrong | Fix |
|---------|----------------|-----|
| Flash error persistence | Error flash messages were immediately cleared by the next poll cycle's success response. Users never saw the error. | Action errors (`flashErr`) survive across poll cycles. Only cleared by the next successful *action*, not by data fetches. |
| Config apply with no main config | File picker crashed when applying config if no main config existed yet. Cursor mapping also broke. | Handle nil main config as a valid state in both file picker and cursor resolution. |
| Panel overflow | Lipgloss width calculations didn't account for border width, causing content to overflow panels. | Subtract border width from available width before rendering inner content. |
| Log jump on follow toggle | Toggling follow mode reset viewport position, causing visible jump. | Preserve viewport position when switching between poll and follow modes. |
| Cursor jump on refresh | Selection tracked by list index. When data refreshed and list order changed, cursor pointed to wrong item. | Track selection by item identity (backup name, agent node), not index. |
| Epoch dates in backup list | Derived timestamps (`LastWriteTS` -> `StartTS` fallback) show 1970 dates for freshly started backups where both are zero. | Display backup `Name` (always an RFC 3339 timestamp, always set) instead of derived timestamps. |
| huh theme mismatch | Used `huh.ThemeCatppuccin()` which is adaptive and ignores the chosen flavor. Forms looked wrong on named Catppuccin flavors. | Build per-flavor huh themes from catppuccin-go color values. Note: `ThemeCatppuccin()` is still correct for the adaptive/default theme — the pitfall only applies to named flavors (Mocha, Latte, etc.). |

## General Pitfalls

| Pitfall | Lesson |
|---------|--------|
| Adding to sealed interfaces | When adding a new variant, the compiler catches all missing type switch cases via `default: panic("unreachable")`. Let compile errors guide you. |
| Conversion test coverage | Always test PBM -> SDK conversion for new fields. The conversion boundary is where most bugs hide. |
| `require` vs `assert` in tests | Use `require` only for preconditions (setup that must succeed). Using `require` for test assertions hides subsequent failures. |
| PBM workarounds | Always mark with `TODO(pbm-fix)`. Isolate workarounds to minimize blast radius when PBM fixes the issue upstream. |
| Unknown enum values | Log `slog.Warn`, don't crash. SDK pins to a specific PBM version; unknown enums appear only on version mismatch, which is recoverable. |
| `TestType_Method` underscore naming | The `TestType_Method` pattern is NOT a Go stdlib convention. The stdlib uses concatenated names (`TestReaderLenSize`, `TestBufferGrowth`). Use `TestTypeMethod` — no underscores. |
| Non-table-driven tests | When testing multiple inputs against the same logic, individual test functions are harder to maintain and extend. Use table-driven tests with `t.Run` subtests instead. |
