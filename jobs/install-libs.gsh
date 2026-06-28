#!/bin/sh

set -e

cd custom

if [ -e "requirements.txt" ]
then
	if [ -z "${PYTHONUSERBASE:-}" ]
	then
		echo "PYTHONUSERBASE must be set" >&2
		exit 1
	fi
	mkdir -p "${PYTHONUSERBASE}"
	chmod 0755 "${PYTHONUSERBASE}"
	python3 -m pip install --user -r requirements.txt
fi

if [ -e "Gemfile" ]
then
	if [ -z "${GEM_HOME:-}" ]
	then
		echo "GEM_HOME must be set" >&2
		exit 1
	fi
	if [ -z "${HOME:-}" ]
	then
		echo "HOME must be set" >&2
		exit 1
	fi
	mkdir -p "${GEM_HOME}"
	chmod 0755 "${GEM_HOME}"

	if ! "${HOME}/bin/bundle" --version >/dev/null 2>&1
	then
		gem install --no-user-install --no-document \
			--install-dir "${GEM_HOME}" \
			--bindir "${HOME}/bin" \
			bundler
	fi

	export BUNDLE_PATH__SYSTEM=true
	"${HOME}/bin/bundle" install
fi
