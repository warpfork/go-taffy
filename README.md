go-taffy
========

`go-taffy` is a library implementing the "taffy" file format.

"taffy" stands for "Test Anything Files -- For You".
It's a simple, human-editable file format which is intended to be useful for writing test fixtures,
and for being easily parsable in any language with a minimal amount of parser code.


Example taffy content
---------------------

The taffy format is meant to be easy to work with:
editable as a human,
and also easily editable programmatically.

```text
-- hunk-1 --
	This is a taffy file.
-- like/files --
	Taffy files contain a bunch of hunks of data.
	They're meant to feel roughly like an archive of files.
-- like/filesystems --
	Many taffy libraries will provide ways to read a taffy file the same way they read a tar archive,
	or even provide functions to unpack them to a filesystem.
-- safe-for/most-content --
	The indentation in a taffy files means they're safe for all content.
	For example, this next line:
	-- this-isnt-a-problem --
	... did not start a new hunk.  We know because of the indentation on start of line.
	This means you can emit taffy files for user content.
-- safe-for/whitespace --
	 This hunk starts with a space.
	That's fine.
		So are tabs.
-- safe-for/binary-but-dont-push-it --
	Binary data should still be b64 encoded, or packed in some other way.
	The taffy format is technically safe for binary content,
	but because the files are intended to be humanely editable, putting binary content with
	nonprintable characters is not recommended.
-- non-recursive --
	The taffy format isn't really recursive.
	Test fixtures tend not to need this, so, the format follows the ancient adage of "KISS".
	But you've noticed the "dir1/dir2/filename" convention for hunks?  Use it.
-- comment --
	The taffy format doesn't technically have comments.
	However, it's easy to create a section titled "comment",
	or "foobar/comment" when there's more than one comment, or it relates to "foobar/data".
```

The parsing rules are simple:

- Each line which starts with "`-- `" and ends with "` --`" is a section header.
- The content between the "`-- `" and "` --`" sequence is the section title.
- All of the bytes until the next section header are that section's body.
- Exactly one tab (`\t`) character will be stripped from the start of each line of the section body, if present.


Taffy data is usually thought of "like a filesystem" (or, more precisely, "like a tar archive").
Each section is treated roughly like a file.
Section names are treated roughly like a filesystem path.

The order of sections is treated as stable (like a tar archive).

Repeated section names are usually treated as an error
(because it would not be possible to map this into a filesystem),
but most taffy libraries can be configured to ignore this if desired.


Loading taffy files in golang
-----------------------------

```go
// example todo
```


Writing taffy files in golang
-----------------------------

```go
// example todo
```

Taffy format in Detail
----------------------

The taffy format is designed to be simple to parse,
simple to edit as a human,
simple to emit as a machine,
and be relatively "stable" to round-trip through parse-edit-emit cycles.
It's also designed to have as few "invalid" states as possible
(most forms of typo will result in more data becoming "content",
rather than the file becoming "corrupt" and unparsable).

Nonetheless, we end up with a few interesting corner cases to mind
when parsing and emitting taffy documents:

- The tab characters in section bodies are technically optional.
  However, all taffy writers are encouraged to use them
  because it makes the format safe for all body content
  without requiring other (uglier) forms of escaping,
  and taffy parsers are encouraged to require the leading tabs
  unless explicitly configured by the user to be tolerant,
  in order to encourage good behavior throughout the ecosystem.
	- The tab characters on empty lines should always be
	  considered optional, and not be emitted by taffy writers,
	  as a concession to the prevalence of human text editors
	  which will strip "trailing whitespace" from lines.
	- Multiple leading tab characters should *never* be munched;
	  if there are two tab characters on a line, one of them is real content.
	- In general, the rules here are aiming for "Postel's Principle",
	  but the resounding expectation of users should be that if documents
	  are written without indentation, the ecosystem's tooling will
	  normalize those documents to be written with indentation.
- Content which comes before the first section header is possible.
  We call this "leading comment" and discourage its use.
  Most taffy parsers should reject files starting with leading comments by default,
  in order to encourage good behavior throughout the ecosystem.
- Note that there's no such thing as "trailing comment" content;
  such bytes are still part of the body of the last section header,
  or, if there's absolutely no section headers in the entire file,
  such bytes would still be "leading comment" content.

Taffy was inspired by [txtar](https://pkg.go.dev/golang.org/x/tools/txtar),
and also by [wishfix](https://github.com/warpfork/go-wish/blob/master/wishfix/format.md),
but adds slightly more safety for user-defined-content than txtar,
and significantly simplifies from wishfix.


License
-------

SPDX-License-Identifier: Apache-2.0 OR MIT
