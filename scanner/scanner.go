package scanner

import "io"

func NewScanner(r io.Reader) *scanner {
	return &scanner{r: r}
}

type scanner struct {
	r     io.Reader
	buf   []byte // buffer for content hunks.
	buf2  []byte // buffer for potential section header.
	one   [1]byte
	title []byte // if non-nil, we just scanned a section body... and also detected the next section header, which must be returned next.
	err   error  // if non-nil, we encountered an error, and should return that next.  (title takes precidence.)
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
// The byte slice returned for Token.Content may be reused between subsequent calls to Scan!
// The caller should copy the content to a new buffer before the next call to Scan if continued access to that data is needed.
// Modifying the buffer, up until the time of the next Scan call, is acceptable, and will have no effect.
func (s *scanner) Scan() (Token, error) {
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
	s.init()
	s.buf = s.buf[0:0]
line:
	// Start of line state.
	byt, err := s.read1()
	if err != nil {
		s.err = err
		return Token{Content: s.buf}, nil
	}
	switch byt {
	case '\t': // Do not keep.
		goto content
	case '\n':
		s.buf = append(s.buf, byt)
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
					s.buf = append(s.buf, s.buf2...)
					s.buf = append(s.buf, '\n')
					goto line
				}
				// Okay, we have a section header!
				//  If this is the first content ever, we can return it immediately.
				//  Otherwise, we actaully have to hang onto it for a round:
				//   first we'll have to return a token with the body for the previous section,
				//    since detecting a section header is also the first time we know that the previous section ended.
				if len(s.buf) == 0 {
					return Token{Title: string(s.buf2[3:endOffset])}, nil // FIXME: this is wrong, you should still yield an empty body token or it's incredibly annoying.
					// FIXME it's also fairly awful that if your file has a trailing linebreak, it ends up in the content.  We should... not do that, probably.
				}
				// One more fiddly bit.  The linebreak at the end of the previous content...
				//  doesn't actually belong to that content hunk; it belongs to the section header.
				//  (This is important, so that it's possible to define content that doesn't have a trailing linebreak.)
				//  We'll have to chomp off one byte before returning the content.
				s.title = s.buf2[3:endOffset]
				return Token{Content: s.buf[0 : len(s.buf)-1]}, nil
			}
		}
	default: // Content.
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
			s.buf = append(s.buf, byt)
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
