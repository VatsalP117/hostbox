# Implementation Plan

## 1. Task Summary

Restore a truthful green dashboard and marketing build baseline for the first Hostbox v1 milestone.

## 2. Current System Understanding

The dashboard has known lint failures and production dependency advisories. The marketing package manifest and lockfile are not currently proven to support a reproducible clean install. CI verifies the dashboard but does not verify marketing.

## 3. Scope

### In Scope

- Correct lint failures in the assigned dashboard components without changing behavior.
- Update existing dashboard dependency versions and lockfile when compatible fixes exist.
- Repair marketing package versions, lockfile, or configuration needed for clean install/build.
- Add a marketing clean install and production build job to the test workflow.
- Run and report focused verification.

### Out of Scope

- Dashboard feature changes or broad refactors.
- New dependencies or disabled lint rules.
- Release packaging, CLI, backend, infrastructure, or deployment behavior.

## 4. Proposed Technical Approach

Reproduce clean-install, lint, build, and audit failures first. Apply the smallest source and dependency corrections supported by the existing toolchain. Add an independent CI job for marketing so its lockfile and build remain enforced.

## 5. Step-by-Step Execution Plan

1. Run dashboard clean install, lint, type-check, build, and production audit.
2. Run marketing clean install and build.
3. Fix assigned lint targets while preserving behavior.
4. Update compatible existing dependency versions/lockfiles and marketing configuration as required.
5. Add marketing installation/build verification to CI.
6. Re-run focused checks and document exact results and residual audit findings.

## 6. Test Plan

- Dashboard: `npm ci`, `npm run lint`, `npx tsc --noEmit`, `npm run build`, `npm audit --omit=dev`.
- Marketing: `npm ci`, `npm run build`, `npm audit --omit=dev`.
- Repository: workflow syntax inspection and `git diff --check` for owned files.

## 7. Acceptance Criteria

- Dashboard clean install, lint, type-check, and build pass.
- No lint rule is disabled.
- Compatible production advisory fixes are applied and remaining findings are reported.
- Marketing clean install/build pass from the committed lockfile.
- CI independently verifies marketing clean install/build.

## 8. Risks and Guardrails

- Preserve dashboard runtime behavior while correcting hook/lint structure.
- Avoid dependency major upgrades unless required for compatibility and validated by builds.
- Do not edit files outside the assigned ownership set.

## 9. Executor Instructions

Keep changes minimal, run focused checks, do not commit, and write the implementation report for orchestrator review.
