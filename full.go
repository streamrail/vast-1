package vast

import (
	"bytes"
	"encoding/xml"
)

// FromXMLFull parses a VAST document keeping the VAST 4 fields that FromXML drops, and keeping
// newlines and tabs inside CDATA and text. Only layout whitespace is removed: whitespace-only
// text between tags and whitespace at the edges of text next to a tag.
func FromXMLFull(xmlStr []byte) (*VAST, error) {
	var v VAST
	if err := xml.Unmarshal(trimLayout(xmlStr), &v); err != nil {
		return nil, err
	}

	v.walkEncoded(func(keep *bool) { *keep = true })

	return &v, nil
}

// MarshalXMLFull writes the document with an XML header. Unlike MarshalXML it doesn't trim
// strings or remove newlines and tabs.
func (v *VAST) MarshalXMLFull() ([]byte, error) {
	data, err := xml.Marshal(v)
	if err != nil {
		return nil, err
	}

	return append([]byte(xml.Header), data...), nil
}

// MarshalXML writes XML-encoded parameters back as escaped text when parsed by FromXMLFull,
// so the player decodes the same value the ad server sent.
func (a AdParameters) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	if !writeAsEscapedText(a.XMLEncoded, a.keepEncoding, a.Parameters) {
		type plain AdParameters
		return e.EncodeElement(plain(a), start)
	}

	return e.EncodeElement(encodedText{XMLEncoded: true, Text: a.Parameters}, start)
}

// MarshalXML writes XML-encoded HTML back as escaped text when parsed by FromXMLFull.
func (h HTMLResource) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	if !writeAsEscapedText(h.XMLEncoded, h.keepEncoding, h.HTML) {
		type plain HTMLResource
		return e.EncodeElement(plain(h), start)
	}

	return e.EncodeElement(encodedText{XMLEncoded: true, Text: h.HTML}, start)
}

// writeAsEscapedText reports whether an XML-encoded value parsed by FromXMLFull is written back as
// escaped text. encoding/xml can't tell CDATA from escaped text, so the decoded value decides: with
// a '<' it was sent as escaped text; without one it was sent already escaped inside CDATA, and goes
// back as CDATA so it isn't escaped twice.
func writeAsEscapedText(xmlEncoded, keepEncoding bool, value []byte) bool {
	return xmlEncoded && keepEncoding && bytes.IndexByte(value, '<') >= 0
}

type encodedText struct {
	XMLEncoded bool   `xml:"xmlEncoded,attr,omitempty"`
	Text       []byte `xml:",chardata"`
}

// stripVAST4 clears everything FromXML didn't keep before the VAST 4 fields were added, so
// FromXML output stays byte-identical for existing callers.
func (v *VAST) stripVAST4() {
	for i := range v.Ads {
		ad := &v.Ads[i]
		ad.AdType, ad.ConditionalAd = "", ""

		if in := ad.InLine; in != nil {
			in.AdServingID, in.Categories, in.ViewableImpression, in.Expires = "", nil, nil, ""
			in.Pricing = legacyPricing(in.Pricing)
			stripVerifications(in.AdVerifications)

			for j := range in.Creatives {
				c := &in.Creatives[j]
				c.SpecAdID, c.UniversalAdIDs, c.CreativeExtensions = "", nil, nil

				if l := c.Linear; l != nil {
					l.Mezzanines, l.InteractiveCreativeFiles, l.ClosedCaptionFiles = nil, nil, nil
					for k := range l.MediaFiles {
						l.MediaFiles[k].FileSize, l.MediaFiles[k].MediaType = "", ""
					}
				}
			}
		}

		if w := ad.Wrapper; w != nil {
			w.FollowAdditionalWrappers, w.AllowMultipleAds, w.FallbackOnNoAd = "", "", ""
			w.Pricing, w.ViewableImpression, w.BlockedAdCategories = nil, nil, nil
			stripVerifications(w.AdVerifications)

			for j := range w.Creatives {
				c := &w.Creatives[j]
				c.SpecAdID, c.UniversalAdIDs, c.CreativeExtensions = "", nil, nil
			}
		}
	}
}

// legacyPricing mirrors the old string field: value only, dropped when empty.
func legacyPricing(p *Pricing) *Pricing {
	if p == nil || p.Value == "" {
		return nil
	}

	return &Pricing{Value: p.Value}
}

// stripVerifications keeps only the last resource of each kind, which is what decoding repeated
// elements into the old single-pointer fields did.
func stripVerifications(av *AdVerifications) {
	if av == nil {
		return
	}

	for i := range av.AdVerifications {
		ver := &av.AdVerifications[i]
		if n := len(ver.JavaScriptResources); n > 1 {
			ver.JavaScriptResources = ver.JavaScriptResources[n-1:]
		}

		if n := len(ver.ExecutableResources); n > 1 {
			ver.ExecutableResources = ver.ExecutableResources[n-1:]
		}
	}
}

// walkEncoded calls f with the keepEncoding flag of every AdParameters and HTMLResource.
func (v *VAST) walkEncoded(f func(keep *bool)) {
	params := func(p *AdParameters) {
		if p != nil {
			f(&p.keepEncoding)
		}
	}
	html := func(h *HTMLResource) {
		if h != nil {
			f(&h.keepEncoding)
		}
	}
	icons := func(icons []Icon) {
		for i := range icons {
			html(icons[i].HTMLResource)
		}
	}

	for i := range v.Ads {
		if in := v.Ads[i].InLine; in != nil {
			for j := range in.Creatives {
				c := &in.Creatives[j]
				if l := c.Linear; l != nil {
					params(l.AdParameters)
					icons(l.Icons)
				}

				if ca := c.CompanionAds; ca != nil {
					for k := range ca.Companions {
						params(ca.Companions[k].AdParameters)
						html(ca.Companions[k].HTMLResource)
					}
				}

				if na := c.NonLinearAds; na != nil {
					for k := range na.NonLinears {
						params(na.NonLinears[k].AdParameters)
						html(na.NonLinears[k].HTMLResource)
					}
				}
			}
		}

		if w := v.Ads[i].Wrapper; w != nil {
			for j := range w.Creatives {
				c := &w.Creatives[j]
				if l := c.Linear; l != nil {
					icons(l.Icons)
				}

				if ca := c.CompanionAds; ca != nil {
					for k := range ca.Companions {
						params(ca.Companions[k].AdParameters)
						html(ca.Companions[k].HTMLResource)
					}
				}
			}
		}
	}
}

var (
	cdataOpen  = []byte("<![CDATA[")
	cdataClose = []byte("]]>")
	layoutWS   = " \t\r\n"
)

// trimLayout removes indentation: whitespace-only text between tags is dropped, and text next
// to a tag loses its leading/trailing whitespace. CDATA sections lose whitespace at their edges,
// so URLs come out clean while newlines inside scripts are kept. Comments, processing
// instructions and attribute values are copied unchanged.
func trimLayout(b []byte) []byte {
	out := make([]byte, 0, len(b))
	afterCDATA := false

	for i := 0; i < len(b); {
		if b[i] != '<' {
			j := i + bytes.IndexByte(b[i:], '<')
			if j < i {
				j = len(b)
			}

			text := b[i:j]
			if !afterCDATA {
				text = bytes.TrimLeft(text, layoutWS)
			}

			if !bytes.HasPrefix(b[j:], cdataOpen) {
				text = bytes.TrimRight(text, layoutWS)
			}

			out = append(out, text...)
			i, afterCDATA = j, false

			continue
		}

		end, isCDATA := markupEnd(b, i)
		if isCDATA {
			out = appendTrimmedCDATA(out, b[i:end])
		} else {
			out = append(out, b[i:end]...)
		}

		i, afterCDATA = end, isCDATA
	}

	return out
}

// appendTrimmedCDATA appends a CDATA section without the whitespace at the edges of its content.
// An unterminated section is copied unchanged; xml.Unmarshal then reports the error.
func appendTrimmedCDATA(out, section []byte) []byte {
	if len(section) < len(cdataOpen)+len(cdataClose) || !bytes.HasSuffix(section, cdataClose) {
		return append(out, section...)
	}

	content := bytes.Trim(section[len(cdataOpen):len(section)-len(cdataClose)], layoutWS)

	out = append(out, cdataOpen...)
	out = append(out, content...)

	return append(out, cdataClose...)
}

// markupEnd returns the offset just after the markup starting at b[i] == '<'. Unterminated
// markup runs to the end; xml.Unmarshal then reports the error.
func markupEnd(b []byte, i int) (int, bool) {
	rest := b[i:]

	switch {
	case bytes.HasPrefix(rest, cdataOpen):
		return closeAfter(b, i, cdataClose), true
	case bytes.HasPrefix(rest, []byte("<!--")):
		return closeAfter(b, i, []byte("-->")), false
	case bytes.HasPrefix(rest, []byte("<?")):
		return closeAfter(b, i, []byte("?>")), false
	}

	// Tags and declarations: stop at the first '>' outside quotes and brackets.
	var quote byte

	depth := 0

	for j := i + 1; j < len(b); j++ {
		c := b[j]

		switch {
		case quote != 0:
			if c == quote {
				quote = 0
			}
		case c == '"' || c == '\'':
			quote = c
		case c == '[':
			depth++
		case c == ']':
			depth--
		case c == '>' && depth <= 0:
			return j + 1, false
		}
	}

	return len(b), false
}

func closeAfter(b []byte, i int, closer []byte) int {
	if k := bytes.Index(b[i:], closer); k >= 0 {
		return i + k + len(closer)
	}

	return len(b)
}
