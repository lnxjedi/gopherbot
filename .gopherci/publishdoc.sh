#!/bin/bash

# publishdoc.sh - check for updated docs and push if needed

COMMIT=$(git rev-parse --short HEAD)

cd gopherbot-doc
if [ -z "$(git status --porcelain)" ]
then
    echo "No updates to documentation"
    exit 0
fi

git add .
git commit -m "Updates from commit $COMMIT"
git remote add update git@github.com:lnxjedi/gopherbot-doc.git
git push -u update master
