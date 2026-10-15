#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")"
ua='Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/140.0.0.0 Safari/537.36'
links=$(grep -o 'https://fonts.googleapis.com/css2?[^"]*' ../../frontend/src/app.html)
css=$(for l in $links; do curl -fsS -A "$ua" "${l//&amp;/&}"; done)
rm -f ./*.woff2
for url in $(grep -o 'https://fonts.gstatic.com/[^)]*\.woff2' <<<"$css" | sort -u); do
	name=$(sed -E 's#https://fonts.gstatic.com/s/([^/]+)/.*#\1#' <<<"$url")-$(sha256sum <<<"$url" | cut -c1-12).woff2
	curl -fsS -o "$name" "$url"
	css=${css//"$url"/"$name"}
done
printf '/* from: %s */\n%s\n' "$(echo $links)" "$css" >fonts.css
