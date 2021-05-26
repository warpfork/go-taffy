package taffy

// Archive is a collection of Sections.
// It may also have a leading comment.
type Archive struct {
	Comment  []byte
	Sections []Section
}

// Section is a single hunk in an Archive;
// it has a title and it has body content.
type Section struct {
	Title string
	Body  []byte
}

// IndexedArchive is constructed over an Archive
// and provides map-like access its sections as keyed by the section titles.
type IndexedArchive struct {
	Archive Archive
	Index   map[string]*Section
}
