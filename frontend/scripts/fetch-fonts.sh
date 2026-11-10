#!/usr/bin/env bash
# Refreshes static/fonts from Google Fonts: every weight the app uses, every subset Google serves.
set -euo pipefail
out=${1:-"$(dirname "$0")/../static/fonts"}
families='family=IBM+Plex+Mono:wght@400;500;600&family=IBM+Plex+Sans:wght@400;500;600;700&display=swap'
# Plex draws these, but none of Google's unicode subsets carries them: ←→↶↷↻⇄✓
symbols='%E2%86%90%E2%86%92%E2%86%B6%E2%86%B7%E2%86%BB%E2%87%84%E2%9C%93'
links="https://fonts.googleapis.com/css2?$families https://fonts.googleapis.com/css2?$families&text=$symbols"
ua='Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/140.0.0.0 Safari/537.36'
cd "$out"
css=$(for l in $links; do curl -fsS -A "$ua" "$l"; done)
rm -f ./*.woff2
for url in $(grep -o 'https://fonts.gstatic.com/[^)]*' <<<"$css" | sort -u); do
	family=$(sed -nE 's#https://fonts.gstatic.com/s/([^/]+)/.*#\1#p' <<<"$url")
	name=${family:-ibmplexsymbols}-$(sha256sum <<<"$url" | cut -c1-12).woff2
	curl -fsS -o "$name" "$url"
	css=${css//"$url"/"$name"}
done
printf '/* from: %s */\n%s\n' "$links" "$css" >fonts.css
npx prettier --write --log-level warn fonts.css
