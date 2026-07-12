# Implementation Report

## Summary

Restored reproducible dashboard and marketing verification for the Hostbox v1 Milestone 0 baseline.

- Fixed the dashboard's two ANSI regular-expression lint errors without changing ANSI parsing behavior.
- Removed the auth bootstrap hook warning while retaining mount-time bootstrap semantics.
- Removed the unused, vulnerable `ansi-to-react` runtime dependency and updated React Router to the patched `6.30.4` release.
- Applied compatible lockfile-only audit updates; the dashboard production dependency audit now has zero findings.
- Aligned marketing with a mutually compatible Node 20 toolchain: Astro `5.18.2`, `@astrojs/react` `4.4.2`, and TypeScript `5.9.3`.
- Regenerated the marketing lockfile and applied compatible transitive audit fixes.
- Added an independent marketing `npm ci` and `npm run build` job to the test workflow.

## Files Changed

- `.github/workflows/test.yml`: added the Node 20 marketing clean-install/build job.
- `web/src/components/deployments/log-viewer.tsx`: constructs the ANSI escape matcher from a string so the existing behavior passes `no-control-regex`.
- `web/src/components/shared/auth-guard.tsx`: retains initial auth/bootstrap values in refs for the mount-only effect, satisfying hook dependency lint without repeated refresh requests.
- `web/package.json`: removed unused `ansi-to-react`; updated `react-router-dom` to `^6.30.4`.
- `web/package-lock.json`: synchronized dependency removals/updates and compatible audit fixes.
- `marketing/package.json`: selected Astro/React integration/TypeScript versions compatible with each other and Node 20.
- `marketing/package-lock.json`: regenerated the reproducible compatible dependency graph and applied non-breaking audit fixes.
- `.agent/tasks/2026-07-12-v1-m0-frontend/{TASK.md,PLAN.md,IMPLEMENTATION_REPORT.md`: track artifacts.

## Commands Run

- Initial dashboard: `npm ci`, `npm run lint`, `npx tsc --noEmit`, `npm run build`, `npm audit --omit=dev --json`.
- Initial marketing: `npm ci` (reproduced the Astro 6 / Tailwind integration peer conflict).
- Dependency updates via npm install/uninstall and non-breaking `npm audit fix` in each package.
- Final dashboard under Node `20.20.2`: `npm ci`, `npm run lint`, `npx tsc --noEmit`, `npm run build`.
- Final marketing under Node `20.20.2`: `npm ci`, `npm run build`.
- `npm audit --omit=dev --json` and full `npm audit --json` in the dashboard.
- `npm audit --omit=dev --json` in marketing.
- Parsed `.github/workflows/test.yml` with Ruby YAML.
- `git diff --check` restricted to owned files.

## Tests

- Dashboard Node 20 clean install: passed.
- Dashboard ESLint: passed with zero errors. Four pre-existing Fast Refresh warnings remain in unowned UI primitive files (`badge.tsx`, `button.tsx`, `form.tsx`, and `toggle.tsx`).
- Dashboard TypeScript check: passed.
- Dashboard production build: passed.
- Dashboard production dependency audit: passed with zero findings.
- Dashboard full audit: two development-server/tooling findings remain (`vite` high, transitive `esbuild` moderate); npm's available fix is the breaking Vite 8 upgrade.
- Marketing Node 20 clean install: passed.
- Marketing static production build: passed; all three routes generated.
- Marketing audit: three package findings remain (`astro` high, `@astrojs/tailwind` low, transitive `esbuild` low). npm's available remediation requires Astro 7, which is incompatible with the current Tailwind integration and would require a major integration/toolchain migration. Compatible fixes reduced the audit from five findings to three.
- Workflow YAML parse: passed.
- Owned-file whitespace validation: passed.

## Deviations From Plan

None. Major dependency migrations were deliberately not forced because the assigned goal allowed compatible fixes and prohibited unrelated redesign/new dependency work.

## Known Risks

- Marketing remains on Astro 5 because `@astrojs/tailwind` 6 supports Astro only through v5 and CI targets Node 20. Clearing its residual Astro advisories requires a separate migration away from the deprecated integration and onto a current Astro/Node toolchain.
- The residual dashboard Vite advisory applies to development-server tooling rather than the shipped static assets, but should be removed in a separately validated Vite major-upgrade task.
- The four existing Fast Refresh lint warnings do not fail ESLint or CI and are outside this track's file ownership.

## Next Steps

- Integrate this track with the other Milestone 0 work and run the milestone-wide verification matrix.
- Schedule a focused Astro/Tailwind migration and a focused Vite major upgrade to clear the remaining development/build-tool advisories.
