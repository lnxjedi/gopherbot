#!/bin/bash

# mkdocs.sh - generate updated mdbook docs

mv gopherbot-doc/.git gopherbot-doc.git
cd doc/
mdbook build -d ../gopherbot-doc/
cd -
mv gopherbot-doc.git gopherbot-doc/.git