# belp-ui

The design system shared by belp and the apps it launches, so they look like
one product rather than several programs that happen to be in a terminal.

## Releasing

Commit subjects carry a Conventional Commits type — `feat(theme): Add the Cursor
Style` — because the release reads the bump out of them. A `feat` moves the
minor, anything else the patch, and a `!` or a `BREAKING CHANGE:` footer moves
the minor too while the major is 0.

    make release-plan          # the tag it would create, and why
    make release               # check, then tag and push
    make release LEVEL=minor   # when the subjects cannot say
    make release NOTE="the footer question and the search caret"

Nothing tags on a push to main: a release is a decision, not a consequence of
merging. A subject with no type stops the release rather than defaulting to a
patch, which is how a feature ships without anyone noticing it did.
