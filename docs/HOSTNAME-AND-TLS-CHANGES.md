# Hostname and TLS Flow Changes

This document summarizes the install-to-DNS flow changes made in the `fix-hostname-tls-flow` branch.

## What changed

1. **All generated Hostbox URLs now come from one shared hostname helper**
   - production: `project.domain`
   - branch-stable: `project-branch.domain`
   - preview: `project-suffix.domain`
   - The worker, Caddy route builder, and deployment services now use the same logic, so preview URLs match the hosts Caddy actually serves.

2. **Preview URLs now derive from the deployment ID path, not the commit SHA path**
   - This removes the mismatch where the API could report one preview hostname while Caddy was routing another.

3. **Project slugs are normalized for DNS safety**
   - Project slugs are capped to fit inside a single DNS label even after the preview suffix is added.
   - If truncation is needed, Hostbox adds a short hash to keep the result stable.
   - Hostbox now rejects project names that would claim the reserved dashboard subdomain (for example `hostbox.example.com` when the dashboard is configured there).

4. **The platform apex now serves the dashboard too**
   - The installer already asked users to point `domain` at the server.
   - Hostbox now makes that record useful by serving the dashboard on both `domain` and `hostbox.domain`.

5. **Wildcard TLS configuration is now consistent**
   - `DNS_PROVIDER=none` explicitly means “do not configure wildcard DNS challenge”.
   - When a supported provider is selected, Hostbox derives the matching Caddy DNS-challenge config automatically.
   - Cloudflare now uses `CF_API_TOKEN` consistently.
   - Route53 now includes `AWS_HOSTED_ZONE_ID` in the documented/install flow.

6. **Initial setup now creates a real refresh-token session**
   - The first admin setup flow now persists the same kind of session record that normal login/register flows create.
   - That keeps refresh-token behavior consistent immediately after setup.

## Operator impact

- Existing DNS guidance still works: point `domain` and `*.domain` to the server IP.
- Wildcard TLS is optional. Without a DNS provider, Hostbox relies on per-host certificates.
- New deployments will use the corrected preview URL format automatically.
