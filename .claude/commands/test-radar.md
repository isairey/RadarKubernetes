# Test Radar

Build and test Radar changes using a real running instance. Combines build verification, API smoke tests, and optional Playwright E2E tests.

## Prerequisites

**Playwright MCP**: Check if the Playwright MCP tools are available (e.g., `mcp__playwright__browser_navigate`). If NOT available, tell the user "Playwright MCP is not available — please add it to your MCP config and restart" and **stop**. Do not attempt workarounds.

## Setup

1. **Pick a random port** (e.g., 9300-9399 range) for the backend to avoid conflicting with other running Radar instances.
2. **Build**: Run `make build` from the repo root. If it fails, diagnose and report — do not proceed.
3. **Start server**: Run the built binary with `--port <random-port> --no-open` in the background. Wait for it to be ready (curl the health endpoint or `/api/cluster-info`).

## Test Plan

Create a test plan based on the current changes (check `git diff main..HEAD --name-only` to scope what changed). Then execute what you can autonomously.

### What to test yourself:
- **API smoke tests**: Hit key API endpoints via curl/fetch and verify they return valid responses (200, correct JSON structure). Good candidates: `/api/cluster-info`, `/api/resource-counts`, `/api/resources/pods`, `/api/dashboard`, `/api/events/stream` (SSE — connect briefly, verify event format).
- **Frontend loads**: Use Playwright to navigate to `http://localhost:<port>`, verify the page loads without console errors, key UI elements render.
- **Feature-specific tests**: Based on the diff, test the specific feature that changed. For SSE-related changes, connect to the SSE stream and verify events arrive. For UI changes, use Playwright to verify rendering. For API changes, hit the relevant endpoints.

### What to flag for manual testing:
- Anything requiring `kubectl` commands against a live cluster (you don't know if one is connected)
- Destructive operations (creating/deleting K8s resources)
- Visual/UX verification that can't be checked programmatically
- Multi-browser or performance testing

## On Adding Tests

Consider whether unit/smoke/E2E tests should be added to the codebase. Be smart about it:
- **Quality over quantity** — don't add tests just for coverage numbers
- **No brittle tests** — avoid tests that break on minor refactors, depend on timing, or test implementation details
- **Test behavior, not wiring** — a test that verifies "SSE events trigger query invalidation" is valuable; a test that verifies "setTimeout was called with 3000" is not
- **Prefer integration over unit for UI** — testing that a component re-renders after an SSE event is more valuable than testing the debounce function in isolation
- If the change is small/mechanical and the risk is low, it's fine to say "no new tests needed" — explain why

## Cleanup

When done testing, **kill the Radar process** you started. Don't leave orphan processes.

## Output

Summarize:
1. Build result (pass/fail)
2. Test results (what passed, what failed, what was skipped)
3. Manual testing needed (what the user should verify themselves)
4. Test recommendation (should we add automated tests? which ones? why or why not?)
