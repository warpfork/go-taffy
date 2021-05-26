package scanner

import "io"

func NewScanner(r io.Reader) *scanner {
	return &scanner{r: r}
}

type scanner struct {
	r            io.Reader
	buf          []byte // buffer for content hunks.
	buf2         []byte // buffer for potential section header.
	one          [1]byte
	foundSection bool   // if true, we've encountered at least one section header, ever.  Used to determine if we should yield an empty content token before subsequent section headers.
	title        []byte // if non-nil, we just scanned a section body... and also detected the next section header, which must be returned next.
	err          error  // if non-nil, we encountered an error, and should return that next.  (title takes precidence.)
}

// Token really only comes in two kinds:
// section headers (which have a title)
// and content (which is either body for the previous section header,
// or a leading comment, depending on context of the token stream).
// If content is nil, then it's a section header.
type Token struct {
	Title   string
	Content []byte
}

func (t Token) IsSectionHeader() bool {
	return t.Content == nil
}
func (t Token) IsContent() bool {
	return t.Content != nil
}

func (s *scanner) init() {
	if s.buf != nil {
		return
	}
	s.buf = []byte{}
	s.buf2 = []byte{'-'}
}

// Scan will consume data from the reader and return the next complete token or an error.
// Scan will never return a token and an error from the same call.
//
// Scan will return a content token after every section header, even if the content is empty.
// Scan will never return two content tokens in a row, because without a section header to separate it,
// naturally, all that data is just one piece of content and is just one token.
//
// The very first token returned from a scanner can be either a section header or a content token.
// If it's a content token, this is sometimes called "leading comment" data,
// and taffy files with a leading comment data section are considered noncanonical.
//
// The byte slice returned for Token.Content may be reused between subsequent calls to Scan!
// The caller should copy the content to a new buffer before the next call to Scan if continued access to that data is needed.
// Modifying the buffer, up until the time of the next Scan call, is acceptable, and will have no effect.
func (s *scanner) Scan() (Token, error) {
	s.init()

	// First: return a section header token if we had detected one during the previous scan.
	if s.title != nil {
		title := string(s.title)
		s.title = nil
		return Token{Title: title}, nil
	}

	// Second: return an error if we had detected one during the previous scan.
	if s.err != nil {
		return Token{}, s.err
	}

	// Okay, scan away.
	s.buf = s.buf[0:0]
line:
	// Start of line state.
	//  We're either on the very first line of the file;
	//  or, we're at the start of a new line because we've just consumed a linebreak.
	// Immediately, one fun corner case: if we have read exactly one empty line, and yet we're starting a new one already,
	//  then count two.  Normally, the newline that ends a section header doesn't count, nor does the one that starts a section header,
	//   but if there's no other content on that line, then it feels natural for that linebreak to mean something.
	if len(s.buf) == 1 && s.buf[len(s.buf)-1] == '\n' {
		s.buf = append(s.buf, '\n')
	}
	byt, err := s.read1()
	if err != nil {
		s.err = err
		return Token{Content: s.buf}, nil
	}
	switch byt {
	case '\t': // Do not keep a tab if its at the start of a new line.
		goto content
	case '\n': // If we immediately get another new line... so be it.
		s.buf = append(s.buf, '\n')
		goto line
	case '-': // Start recieving possible section header.
		// Read ahead until an end-of-line, and then we'll decide.
		s.buf2 = s.buf2[0:1]
		for {
			byt, err := s.read1()
			if err != nil {
				s.buf = append(s.buf, s.buf2...)
				s.err = err
				return Token{Content: s.buf}, nil
			}
			switch byt {
			default:
				s.buf2 = append(s.buf2, byt)
			case '\n':
				// Alright, now check if it has all the markers.
				// 123456
				// --  --
				end := len(s.buf2)
				endOffset := end - 3
				if end < 6 ||
					string(s.buf2[0:3]) != "-- " ||
					string(s.buf2[endOffset:end]) != " --" {
					// If we didn't get a section header:
					//  flush all the buffered content we considered into body.
					s.buf = append(s.buf, s.buf2...)
					s.buf = append(s.buf, '\n')
					goto line
				}
				// Okay, we have a section header!
				//  If this is the first content ever, we can return it immediately.
				if !s.foundSection && len(s.buf) == 0 {
					return Token{Title: string(s.buf2[3:endOffset])}, nil
				}
				s.foundSection = true
				// If it's not the first content ever...
				//  Actually, we have to hang onto it for a round:
				//   first we'll have to return a token with the body for the previous section,
				//    since detecting a section header is also the first time we know that the previous section ended.
				s.title = s.buf2[3:endOffset]
				// Within this, there's also one more fiddly bit.  The linebreak at the end of the previous content...
				//  doesn't actually belong to that content hunk; it belongs to the section header.
				//  (This is important, so that it's possible to define content that doesn't have a trailing linebreak.)
				//  We'll have to chomp off one byte before returning the content.
				if len(s.buf) == 0 {
					return Token{Content: s.buf}, nil
				}
				return Token{Content: s.buf[0 : len(s.buf)-1]}, nil
				// FIXME it's also fairly awful that if your file has a trailing linebreak, it ends up in the content.  We should... not do that, probably.

			}
		}
	default: // Content.  It should've started with a tab, for clarity, but we'll be forgiving.
		s.buf = append(s.buf, byt)
		goto content
	}
content:
	// Slurp content into buf until a newline.
	for {
		byt, err := s.read1()
		if err != nil {
			s.err = err
			return Token{Content: s.buf}, nil
		}
		switch byt {
		case '\n':
			s.buf = append(s.buf, '\n')
			goto line
		default:
			s.buf = append(s.buf, byt)
		}
	}
}
func (s *scanner) read1() (byte, error) {
	_, err := s.r.Read(s.one[:])
	return s.one[0], err
}
