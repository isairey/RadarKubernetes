# Visual Test

Visually test Radar changes against a real cluster using Playwright MCP. Takes screenshots, organizes them, and reports what looks right or wrong.

## Step 1: Verify Playwright MCP

Check if Playwright MCP tools are available (e.g., `mcp__playwright__browser_navigate`). If NOT available, tell the user "Playwright MCP is not available -- please add it to your MCP config and restart" and **stop**.

## Step 2: Understand What to Test

Determine what changed and what needs visual verification:

```bash
git status
git branch --show-current
git log main..HEAD --oneline
git diff main..HEAD --name-only
```

If on main with no branch commits, check unstaged changes instead. Summarize what changed and what UI areas are affected (which renderers, which views, which components).

## Step 3: Verify Cluster Access

**This is critical -- Radar is useless without a live cluster.** Check in order:

```bash
# 1. Check current kubeconfig context
kubectl config current-context

# 2. Verify actual connectivity (not just config)
kubectl cluster-info --request-timeout=5s
```

**If cluster is not reachable**, try to diagnose:
```bash
# GKE:
gcloud container clusters list 2>/dev/null | head -5
# EKS:
aws eks list-clusters 2>/dev/null | head -5
```

**If auth is expired or cluster is unreachable:**
- Tell the user which context is configured and that it's not reachable
- Suggest the auth command (e.g., `gcloud container clusters get-credentials ...` or `aws eks update-kubeconfig ...`)
- Ask them to run `! <auth-command>` in the prompt to authenticate
- **Stop and wait** -- don't proceed without a working cluster

## Step 4: Check Required Resources Exist

Based on what changed, verify the cluster has the CRD resources needed to test:

```bash
# Check if the CRD exists
kubectl api-resources --verbs=list 2>/dev/null | grep -i <kind>

# Check if instances exist
kubectl get <kind> -A --no-headers 2>/dev/null | head -5
```

**If CRDs exist but no instances:**
- Tell the user: "The cluster has the <Kind> CRD but no instances. Want me to create a test resource?"
- **Wait for confirmation before creating anything**
- If creating: use a clearly labeled test namespace/name (e.g., `radar-test/test-<kind>`)

**If CRDs don't exist at all:**
- Tell the user which CRDs are missing
- Some CRDs can be installed easily (e.g., `kubectl apply -f` a CRD manifest), others require full operator installation
- Ask the user how they want to proceed

**IMPORTANT: Never modify cluster state (create/delete/update resources) without explicit user confirmation. Read-only operations (get, list, describe) are always safe.**

## Step 5: Build and Launch Radar

Use the helper script — it handles build, port selection, launch, health check, and state file creation in a single command:

```bash
./scripts/visual-test-start.sh
```

If the binary is already built and you just want to relaunch:
```bash
./scripts/visual-test-start.sh --skip-build
```

The script outputs the URL, PID, screenshot dir, and writes state to `.playwright-mcp/visual-test-state.env`. Source it to get the variables:
```bash
source .playwright-mcp/visual-test-state.env
# Now $RADAR_URL, $SCREENSHOT_DIR, $RADAR_PID, $RADAR_LOG are set
```

**When to use `--dev` mode instead:** For rapid CSS/layout iteration, you can skip the helper and run the Vite dev server + Go backend separately (Vite on port+1 proxies /api to the backend). But for testing actual renderer changes, the full `make build` + binary is the right approach — it tests the real embedded frontend.

## Step 6: Visual Testing with Playwright

Navigate to `$RADAR_URL` and systematically test the changed areas.

### Screenshot Strategy

**Choose the right scope for each screenshot:**
- **Full page**: For layout issues, overall view verification, or when context matters
- **Element/region**: For specific component rendering (renderers, badges, sections). Often more useful -- the reader can see detail without noise. Use Playwright's element screenshot or crop to the relevant area.

**Naming convention**: `<kind>-<what>.png` (e.g., `nodepool-resource-usage.png`, `prometheusrule-alert-details.png`, `helmrelease-values-section.png`)

**Screenshots MUST be saved under `.playwright-mcp/`** — Playwright MCP blocks writes outside this dir and the repo root. The start script already creates the right directory.

### Testing flow

For each changed renderer/component:

1. **Navigate to the resource list**: Navigate directly to `$RADAR_URL/resources/<plural-kind>`
2. **Screenshot the list view** (if table columns changed)
3. **Click into a resource** to open the detail drawer
4. **Use `browser_snapshot`** to find elements in the drawer — don't rely on keyboard scrolling. Drawers have their own scroll container, so `End`/`PageDown` keys scroll the page, not the drawer.
5. **Screenshot the specific section** that was added/changed (element screenshot preferred over full page)
6. **Test interactions** if applicable: expand/collapse, search/filter, click links
7. **Check the browser console** for errors: `mcp__playwright__browser_console_messages`
8. **Check dark mode** if color/styling changes were made (toggle in settings)

### What to look for

- Sections render with data (not empty/undefined/null)
- Badges have correct colors and text
- Links are clickable and navigate to the right resource
- Empty states show appropriate text (not blank space)
- Layout doesn't break (no overflow, no clipping, proper spacing)
- Responsive behavior in the drawer (text truncation, wrapping)
- Long text doesn't overflow into adjacent table columns

## Step 7: Report

List screenshot paths as plain text in a code block — do NOT use markdown links (they break when clicked in the terminal). At the end, run `open $SCREENSHOT_DIR` to open the folder in Finder.

```
## Visual Test Results

Screenshots saved to:
  .playwright-mcp/visual-test/<timestamp>/

### Passed
- [kind]: what looks correct (screenshot: filename.png)

### Issues Found
- [kind]: what's wrong (screenshot: filename.png)

### Could Not Test
- [kind]: why (e.g., no instances in cluster, CRD not installed)

### Console Errors
- (list any JS errors, or "None")
```

## Step 8: Cleanup

Use the stop script:
```bash
./scripts/visual-test-stop.sh
```

This kills the Radar process, opens the screenshot folder, and cleans up the state file.

Don't delete the screenshots or logs -- the user may want to reference them or attach to a PR.
