// Package icons is the glyph vocabulary belp apps draw with.
//
// Octicons throughout, so the stroke weight and optical size agree: mixing the
// families is what makes a row of icons look ransom-noted.
//
// A terminal only shows these if its own font is a patched one. A symbols-only
// fallback does not help — a private-use codepoint carries no script, so the
// system has nothing to infer a fallback font from.
package icons

// Written as escapes, never as the character: a private-use glyph does not
// survive every editor, clipboard and diff it passes through. The comment is the
// font's own name for it, read out of its name table.
const (
	Commit            = "\uf417" // oct-git_commit
	Push              = "\uf403" // oct-repo_push
	Branch            = "\uf418" // oct-git_branch
	Trash             = "\uf48e" // oct-trash
	PullRequest       = "\uf407" // oct-git_pull_request
	Merge             = "\uf419" // oct-git_merge
	PullRequestClosed = "\uf4dc" // oct-git_pull_request_closed
	Eye               = "\uf441" // oct-eye
	Comment           = "\uf41f" // oct-comment
)

// All is every glyph with its name, for a preview to draw and a test to check.
var All = []struct {
	Glyph string
	Name  string
}{
	{Commit, "commit"},
	{Push, "push"},
	{Branch, "branch"},
	{Trash, "trash"},
	{PullRequest, "pull request"},
	{Merge, "merge"},
	{PullRequestClosed, "pull request closed"},
	{Eye, "eye"},
	{Comment, "comment"},
}
