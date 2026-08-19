#!/bin/sh
# Tags the next version, read out of the commits since the last tag.
#
# The bump is inferred rather than typed. A Conventional Commit already says
# whether it added something or repaired something, so asking again on the
# command line is a second place to get it wrong — and the answer that gets typed
# is the one nobody checks against the log.
#
# While the major is 0 a breaking change moves the minor: there is nothing
# coarser to move, and 0.x is the promise that it can. Past 1.0 the same marker
# moves the major instead, which is the one line of this that has to change then.
#
# A subject with no recognisable type stops the release. The fallback would be a
# patch, and a patch is how a feature ships without anyone noticing it shipped.
#
# Usage: release.sh plan            what a release would do, and why
#        release.sh tag [level]     do it, level overriding the inference
set -eu

mode=${1:?usage: release.sh plan|tag [patch|minor|major]}
level=${2:-}

# The types Conventional Commits allows here. Anything else is a typo.
types='feat|fix|docs|style|refactor|perf|test|build|ci|chore|revert'
prefix="^($types)(\([^)]+\))?!?: "

last=$(git tag --sort=-v:refname | head -1)
base=${last:-v0.0.0}
range=${last:+$last..HEAD}

# shellcheck disable=SC2086  # an empty range means "every commit", deliberately
subjects=$(git log --format=%s $range)
bodies=$(git log --format=%B $range)
count=$(printf '%s' "$subjects" | grep -c . || true)

unknown=$(printf '%s\n' "$subjects" | grep -Ev "$prefix" || true)
breaking=$(printf '%s\n' "$subjects" | grep -E "^($types)(\([^)]+\))?!: " || true)
breaking="$breaking$(printf '%s\n' "$bodies" | grep -E '^BREAKING[ -]CHANGE:' || true)"
feats=$(printf '%s\n' "$subjects" | grep -E '^feat(\([^)]+\))?!?: ' || true)

major=$(echo "${base#v}" | cut -d. -f1)
minor=$(echo "${base#v}" | cut -d. -f2)
patch=$(echo "${base#v}" | cut -d. -f3)

if [ -n "$level" ]; then
	why="level given on the command line"
elif [ -n "$breaking" ]; then
	if [ "$major" -eq 0 ]; then level=minor; else level=major; fi
	why="a breaking marker"
elif [ -n "$feats" ]; then
	level=minor
	why="$(printf '%s\n' "$feats" | grep -c .) feat commit(s)"
else
	level=patch
	why="no feat, no breaking marker"
fi

case $level in
major) major=$((major + 1)); minor=0; patch=0 ;;
minor) minor=$((minor + 1)); patch=0 ;;
patch) patch=$((patch + 1)) ;;
*) echo "release: $level is not a level" >&2; exit 2 ;;
esac
next="v$major.$minor.$patch"

report() {
	echo "last tag   $base"
	echo "commits    $count since it"
	echo "level      $level — $why"
	echo "next tag   $next"
	if [ -n "$subjects" ]; then
		echo
		printf '%s\n' "$subjects" | sed 's/^/  /'
	fi
	if [ -n "$unknown" ]; then
		echo
		echo "no type on:"
		printf '%s\n' "$unknown" | sed 's/^/  /'
	fi
}

if [ "$mode" = plan ]; then
	report
	exit 0
fi
[ "$mode" = tag ] || { echo "release: $mode is not plan or tag" >&2; exit 2; }

[ "$count" -gt 0 ] || { echo "release: nothing since $base" >&2; exit 1; }

# Refuse rather than warn, and only where a check is cheaper than the mistake: a
# tag is a name other repos pin, so it has to name something that exists exactly
# once, on the branch, with nothing local left out of it.
if [ -n "$unknown" ] && [ -z "${2:-}" ]; then
	echo "release: these subjects carry no type, so the bump cannot be read:" >&2
	printf '%s\n' "$unknown" | sed 's/^/  /' >&2
	echo "fix them, or say the level outright: make release LEVEL=minor" >&2
	exit 1
fi
git diff --quiet && git diff --cached --quiet ||
	{ echo "release: the working tree is not clean" >&2; exit 1; }
branch=$(git rev-parse --abbrev-ref HEAD)
[ "$branch" = main ] || { echo "release: on $branch, not main" >&2; exit 1; }
git fetch -q origin main
[ "$(git rev-parse HEAD)" = "$(git rev-parse origin/main)" ] ||
	{ echo "release: main and origin/main have diverged" >&2; exit 1; }

# The subject carries the number, the body the log. NOTE adds the phrase the
# tags before this one all have — a release is easier to place from a few words
# than from a version.
{
	if [ -n "${NOTE:-}" ]; then
		printf '%s — %s\n\n' "$next" "$NOTE"
	else
		printf '%s\n\n' "$next"
	fi
	printf '%s\n' "$subjects"
} | git tag -a "$next" -F -
echo "tagged $next"

if git push -q origin "$next"; then
	echo "pushed $next"
else
	echo "release: $next is tagged locally but was not pushed. Retry with:" >&2
	echo "  GH_TOKEN=\"\$(gh auth token --user $(git config user.name))\" git push origin $next" >&2
	exit 1
fi
