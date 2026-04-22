---
name: hostbox-vm-update
description: Use when the user asks to publish Hostbox changes to GitHub and update the test VM at root@204.168.243.54 from the latest main branch.
---

# Hostbox VM Update

Use this skill when updating the user's Hostbox test VM after local changes are ready to publish.

## Context

- Local repo: `/Users/vatsalpatel/Desktop/Projects/hostbox`
- GitHub repo: `https://github.com/VatsalP117/hostbox.git`
- VM SSH target: `root@204.168.243.54`
- VM checkout: `/opt/hostbox`
- Normal VM update command: `cd /opt/hostbox && bash scripts/update.sh`

## Workflow

1. Inspect local changes:
   - Run `git status --short --branch`.
   - Review `git diff --stat` and any risky diffs.
   - Do not include unrelated untracked files unless the user asks.

2. Validate before publishing:
   - Run `npm run build` in `web/` for dashboard changes.
   - Run `go test ./...` from the repo root for backend changes.
   - If Go cannot use the build cache because of sandboxing, rerun with approval.
   - Note unrelated lint failures instead of fixing them during a deploy-only task.

3. Commit and push:
   - Stage only intended files with explicit `git add` paths.
   - Commit on `main` if the user asked to push the latest version directly.
   - Push with `git push origin main`.
   - If push fails because auth is missing, stop and report the blocker.

4. Update the VM:
   - SSH to the VM: `ssh root@204.168.243.54`.
   - Run `cd /opt/hostbox && bash scripts/update.sh`.
   - Do not run `bash scripts/update.sh --fresh` unless the user explicitly asks to wipe runtime data.

5. Verify the VM:
   - Run `cd /opt/hostbox && git rev-parse --short HEAD`.
   - Run `cd /opt/hostbox && docker compose ps`.
   - Run `cd /opt/hostbox && docker compose exec -T hostbox wget --no-verbose --tries=1 -O- http://127.0.0.1:8080/api/v1/health`.
   - If the update fails, collect `docker compose logs --tail=120 hostbox` and `docker compose logs --tail=120 caddy`.

## Safety

- Preserve `/opt/hostbox/.env` and runtime directories.
- Avoid destructive commands such as `rm`, `git reset --hard`, or `update.sh --fresh` without explicit user approval.
- Keep the final response short: commit SHA, push result, VM update result, and health status.
