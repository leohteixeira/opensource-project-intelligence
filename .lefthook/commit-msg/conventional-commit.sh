#!/usr/bin/env sh
# Validates the commit subject against the Conventional Commits specification.
# POSIX shell on purpose: the hook must not depend on the JavaScript toolchain.
set -eu

MESSAGE_FILE="${1:?commit message file is required}"
MAX_SUBJECT_LENGTH=100

SUBJECT="$(sed -n '/^[^#]/{p;q;}' "${MESSAGE_FILE}")"

if [ -z "${SUBJECT}" ]; then
  echo "commit-msg: the commit message is empty." >&2
  exit 1
fi

# Git generates these subjects itself; they are not authored by a developer.
case "${SUBJECT}" in
  "Merge "* | "Revert "* | "fixup! "* | "squash! "*) exit 0 ;;
esac

TYPES='build|chore|ci|docs|feat|fix|perf|refactor|revert|style|test'

if ! printf '%s' "${SUBJECT}" \
  | grep -Eq "^(${TYPES})(\([a-z0-9./-]+\))?!?: .+"; then
  cat >&2 <<USAGE
commit-msg: the subject does not follow Conventional Commits.

  found:    ${SUBJECT}
  expected: <type>(<optional scope>): <description>
  types:    ${TYPES}
  example:  feat(findings): add advisory matching

Write commit messages in English.
USAGE
  exit 1
fi

if [ "${#SUBJECT}" -gt "${MAX_SUBJECT_LENGTH}" ]; then
  echo "commit-msg: the subject is ${#SUBJECT} characters, the limit is ${MAX_SUBJECT_LENGTH}." >&2
  exit 1
fi
