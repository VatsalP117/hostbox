# Hostbox Marketing Site

Standalone static marketing website generated from Stitch and kept separate from the dashboard app in `web/`.

## Source

- Stitch project: `projects/4710930812260054651`
- Screens used:
  - `5af9f2041e044eacb53534720dfaeb9d` -> `index.html`
  - `09d2fc7c12814956b665c9b82507795e` -> `features.html`
  - `8b1bf398026647408c28345e8a5588ba` -> `cli.html`

## Files

- `index.html`: landing page
- `features.html`: product/features page
- `cli.html`: CLI showcase page

## Local Preview

Run from repo root:

```bash
python3 -m http.server 4173 -d marketing
```

Then open `http://localhost:4173`.

## Deploy Separately

Deploy this folder as its own static project by pointing your deployment to the `marketing/` directory.
