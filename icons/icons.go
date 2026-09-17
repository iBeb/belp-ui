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

	// Where a row hands off to: a browser, or a terminal with a session in it.
	Link     = "\uf465" // oct-link_external
	Terminal = "\uf489" // oct-terminal

	// Window chrome: the way out of one, and the way into what it does not
	// have room for.
	// Not the octicon x, which is drawn at a third of the cell: this one fills
	// it, so the way out of a window is as plain as the title beside it.
	Close = "\uf057" // fa-times-circle
	Info  = "\uf449" // oct-info
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
	{Link, "link"},
	{Terminal, "terminal"},
	{Close, "close"},
	{Info, "info"},
}
