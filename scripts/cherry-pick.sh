#!/usr/bin/env bash
set -e
safecommit=$1

git add .
git commit -m "temp file commit" 

currentcommit="$(git rev-parse HEAD)"

git checkout -b go-rewrite-convert $safecommit

echo $currentcommit

git cherry-pick $currentcommit --no-commit

files=`git diff --name-only --diff-filter=A --cached`
for file in $files; do
  mv $file ${file%".temp"}
done