# Pitfalls & Lessons Learned

> **Agents: append to this file** when you discover a non-obvious bug, surprising behavior, or a pattern that future work should be aware of. Include what went wrong, why, and the fix.

## SDK Pitfalls

| Pitfall | What went wrong | Fix |
|---------|----------------|-----|
| Goroutine leak in LogService.Follow | Follow adapter didn't select on context cancellation alongside channel reads. Goroutine lived forever after caller cancelled. | `nextLogCmd` must select on both the entries channel and `ctx.Done()`. |
| PITR duration truncation | Used `Truncate(1)` instead of `Truncate(time.Second)`. `Truncate(1)` truncates to 1 nanosecond, which is a no-op. | Always use `time.Duration` constants with `Truncate`. |
| Stale timeline cursor | Compared pointers (`cursor == &item`) instead of values. After data refresh, pointer identity changes even if the item is logically the same. | Track selection by value identity (name, key), not pointer or index. |
| ConfigName empty string | PBM uses `""` for main config. If SDK doesn't normalize, consumers see empty strings and write broken comparisons. | SDK normalizes to `MainConfig` constant. Never check for `""` in TUI code. |
| PITR restore base backup | SDK requires `BackupName` for `StartPITRRestore` (PBM CLI auto-selects). Decision to add SDK auto-selection is postponed. | SDK exports `FilterPITRBases` for validation; TUI shows a base backup selector in PITR restore forms via `pitrBaseGroup` helper. |
| NumParallelColls on incremental | `StartIncrementalBackup` had a `NumParallelColls` field, but PBM's `doPhysical` never reads it — only `doLogical` uses parallel collections. The field was dead. | Removed from SDK. TUI only shows "Parallel Collections" for logical backups. |
| Physical/incremental restore shuts down mongod | PBM's physical restore (`PhysRestore.Snapshot`) shuts down mongod on every node, wipes the data directory, copies WiredTiger files, and does multiple mongod restarts. The TUI loses its connection. | TUI shows a warning confirmation overlay before dispatch, then exits cleanly with a farewell message. |
| Timestamp comparisons ignored ordinal | PITR filtering compared only `.T` (seconds), ignoring `.I` (ordinal increment). Two events in the same second could be misordered. | Added `Timestamp.Before()`/`After()` methods that compare T first, then I as tiebreaker. Used throughout PITR filtering. |
| ConfigName "all profiles" doc mismatch | `DeleteBackupsBefore.ConfigName` doc said "zero value means all profiles" but `configNameToPBM` maps zero to `""`, which PBM interprets as "main config only". PBM does not support cross-profile deletion in a single command. | Fixed doc to say "zero value means main config". TUI bulk delete form shows a concrete profile selector (Main + named profiles), no "All" option. |
| MaskedString breaks YAML roundtrip | PBM's `storage.MaskedString` has `MarshalYAML()` that unconditionally returns `"***"` for non-empty values. `GetYAML`/`GetProfileYAML` used `yaml.Marshal` which triggered masking — editing and re-applying config destroyed credentials. The Percona Operator hit the same issue. This is a PBM design flaw: masking is a presentation concern (CLI output safety) baked into the serialization layer. | BSON roundtrip in `unmaskYAML` (`config_unmask.go`): `bson.Marshal` (no `MarshalBSON` exists on `MaskedString`, so real values are preserved) -> `bson.D` -> `yaml.MapSlice` -> `yaml.Marshal`. Filters PBM metadata keys (`epoch`, `name`, `profile`) that `yaml:"-"` and `yaml:",omitempty"` would normally exclude. Callers opt in via `WithUnmasked()` functional option; the default is masked (safe for display). Minor `omitempty` discrepancy: some zero-value fields in `BackupConf`/`RestoreConf`/`Azure` may appear that `yaml.Marshal` would omit — upstream PBM tag fix pending. |
| GetConfig doesn't return ErrMissedConfig | PBM's `config.GetConfig` wraps `mongo.ErrNoDocuments` with `"get"` but does NOT translate it to `config.ErrMissedConfig` (unlike `GetProfile` which returns `ErrMissedConfigProfile`). SDK only checked for `ErrMissedConfig`, so `Config.Get` on an empty database returned a wrapped error instead of `ErrNotFound`. | Also check `mongo.ErrNoDocuments` in `Get` and `GetYAML`. Marked `TODO(pbm-fix)`. |

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
| Follow context canceled flash | Pressing `f` twice quickly (start then stop follow) showed `"follow: context canceled"` because `waitForLogEntry` returns `ctx.Err()` when cancelled. | Suppress `context.Canceled` in both `logFollowMsg` and `logFollowDoneMsg` handlers — it's a normal shutdown, not an error. |
| Double action error prefixes | `setFlash("start", err)` produced `"start: start backup: ..."` because SDK already wraps errors with the operation name. | Show `msg.err.Error()` directly for action results instead of prepending a TUI prefix. |
| Config ErrNotFound as flash error | `Config.Get` returns `ErrNotFound` when no main config exists (valid state). If it raced first in `firstErrCollector`, user saw `"fetch: not found"`. | Skip `errs.set()` when the error is `ErrNotFound` for config fetch goroutines. |
| Nil context panic on follow before connect | Pressing `f` before connection succeeded panicked with "cannot create context from nil parent" because `overviewModel.ctx` is nil until `connectMsg` arrives. | Guard sub-model key dispatch in `updateKeys` with `m.client == nil` check — no input to sub-models before connection. |
| parseNamespaces returns `[""]` for empty input | `strings.Split("", ",")` returns `[""]`, not `nil`. Selective restore with no namespaces entered would send `[""]` to PBM. | Filter out empty strings after trimming. Return nil when no non-empty entries remain. |
| latestTimeline compared only `.T` | `latestTimeline()` used `.End.T > best.End.T`, ignoring the ordinal — same class of bug as the SDK Timestamp comparison pitfall. | Use `Timestamp.After()` for all timestamp comparisons, including TUI helpers. |
| Bubble Tea background color is best-effort | `tea.RequestBackgroundColor` only triggers `BackgroundColorMsg` if the terminal answers the ANSI query. Some environments (notably some Docker/PTY setups) may never reply, leaving the app on its startup fallback. | Treat background detection as opportunistic, not guaranteed. Keep a reasonable startup fallback and don't assume a follow-up `BackgroundColorMsg` will always arrive. |
| Lip Gloss v2 `Width()` applies to the final rendered block | Overlay/file-picker width math initially assumed `Width()` meant inner content width plus padding, as if border width was external. In Lip Gloss v2 the frame participates in the final width, which caused off-by-two border and separator glitches. | When sizing bordered/padded overlays, add the full frame size explicitly or measure the already-rendered block instead of relying on old v1 width assumptions. |

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
| YAML comment loss on save | `yaml.Marshal` rewrites the file from scratch. If a user hand-edits the YAML and adds comments, they're lost on next `config set`/`config unset`/`context add`/etc. This is a fundamental limitation of `yaml.Marshal` — fixing it would require a comment-preserving YAML library. Accepted trade-off. |
