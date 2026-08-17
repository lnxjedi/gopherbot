#!/bin/bash
set -e

DOCS_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
exec mdbook serve "$DOCS_DIR" -p 8888
