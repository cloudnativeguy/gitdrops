#!/bin/bash

gitdrops="gitdrops.yaml"
gitdropsupdate="gitdrops-update.yaml"

if cmp -s "$gitdrops" "$gitdropsupdate"; then
	printf 'The file "%s" is the same as "%s" - no commit necessary\n' "$gitdrops" "$gitdropsupdate"
else
	printf 'The file "%s" is different from "%s" - commit changes\n' "$gitdrops" "$gitdropsupdate"
	cp -f "$gitdropsupdate" "$gitdrops"
	git config --local user.email "41898282+github-actions[bot]@users.noreply.github.com"
	git config --local user.name "github-actions[bot]"
	git add gitdrops.yaml
	git commit -m "Copy gitdrops-update.yaml to gitdrops.yaml" -a
fi
