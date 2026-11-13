package vast

import (
	"encoding/xml"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func readFixture(t testing.TB, name string) []byte {
	t.Helper()

	raw, err := os.ReadFile("testdata/" + name)
	require.NoError(t, err)

	return raw
}

func marshalFull(t *testing.T, raw []byte) string {
	t.Helper()

	v, err := FromXMLFull(raw)
	require.NoError(t, err)

	b, err := xml.Marshal(v)
	require.NoError(t, err)

	return string(b)
}

func marshalLegacy(t *testing.T, raw []byte) string {
	t.Helper()

	v, err := FromXML(raw)
	require.NoError(t, err)

	b, err := xml.Marshal(v)
	require.NoError(t, err)

	return string(b)
}

// TestFromXMLFull_KeepsWhatFromXMLDrops covers the elements the audit found missing from rebuilt VASTs.
func TestFromXMLFull_KeepsWhatFromXMLDrops(t *testing.T) {
	tests := []struct {
		name    string
		fixture string
		want    string
	}{
		{name: "Ad adType", fixture: "vast4_inline_full.xml", want: `adType="video"`},
		{name: "Ad conditionalAd", fixture: "vast4_inline_full.xml", want: `conditionalAd="false"`},
		{name: "AdServingId", fixture: "vast4_inline_full.xml", want: `<AdServingId>serving-1</AdServingId>`},
		{name: "Category", fixture: "vast4_inline_full.xml", want: `<Category authority="https://iabtechlab.com/categoryauthority">IAB1-1</Category>`},
		{name: "ViewableImpression", fixture: "vast4_inline_full.xml", want: `<ViewableImpression id="vi-1"><Viewable><![CDATA[https://dsp.example/viewable]]></Viewable>`},
		{name: "Pricing attributes", fixture: "vast4_inline_full.xml", want: `<Pricing model="cpm" currency="USD">1.25</Pricing>`},
		{name: "Expires", fixture: "vast4_inline_full.xml", want: `<Expires>3600</Expires>`},
		{name: "UniversalAdId", fixture: "vast4_inline_full.xml", want: `<UniversalAdId idRegistry="ad-id.org">UAI-123</UniversalAdId>`},
		{name: "Creative adId", fixture: "vast4_inline_full.xml", want: `adId="adid-1"`},
		{name: "MediaFile fileSize and mediaType", fixture: "vast4_inline_full.xml", want: `fileSize="1048576" mediaType="2D"`},
		{name: "Mezzanine", fixture: "vast4_inline_full.xml", want: `<Mezzanine id="mz-1" delivery="progressive" type="video/mp4" width="1920" height="1080"`},
		{name: "InteractiveCreativeFile", fixture: "vast4_inline_full.xml", want: `<InteractiveCreativeFile type="text/html" apiFramework="SIMID" variableDuration="true">`},
		{name: "ClosedCaptionFiles", fixture: "vast4_inline_full.xml", want: `<ClosedCaptionFile type="application/ttml+xml" language="fr">`},
		{name: "first of two JS resources", fixture: "vast4_inline_full.xml", want: `<JavaScriptResource apiFramework="omid" browserOptional="true"><![CDATA[https://v.example/web.js]]>`},
		{name: "AdParameters xmlEncoded as escaped text", fixture: "vast4_inline_full.xml", want: `<AdParameters xmlEncoded="true">&lt;p&gt;&#xA;&#x9;keep&lt;/p&gt;</AdParameters>`},
		{name: "newline in CDATA", fixture: "vast4_inline_full.xml", want: "var a = 1; // comment\nvar b = 2;"},
		{name: "newline and tab in VPAID AdParameters", fixture: "vast3_vpaid_pricing_text.xml", want: "\"a // b\nc\", \"tab\": \"x\ty\""},
		{name: "wrapper attributes", fixture: "vast4_wrapper_full.xml", want: `<Wrapper followAdditionalWrappers="false" allowMultipleAds="true" fallbackOnNoAd="1">`},
		{name: "wrapper Pricing", fixture: "vast4_wrapper_full.xml", want: `<Pricing model="cpm" currency="EUR">2.5</Pricing>`},
		{name: "wrapper ViewableImpression", fixture: "vast4_wrapper_full.xml", want: `<ViewableImpression><Viewable><![CDATA[https://dsp.example/wview]]></Viewable></ViewableImpression>`},
		{name: "BlockedAdCategories", fixture: "vast4_wrapper_full.xml", want: `<BlockedAdCategories authority="https://iabtechlab.com/categoryauthority">IAB26</BlockedAdCategories>`},
		{name: "wrapper UniversalAdId", fixture: "vast4_wrapper_full.xml", want: `<UniversalAdId idRegistry="clearcast.co.uk">CC-1</UniversalAdId>`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			raw := readFixture(t, tt.fixture)

			assert.Contains(t, marshalFull(t, raw), tt.want, "kept by FromXMLFull")
			assert.NotContains(t, marshalLegacy(t, raw), tt.want, "dropped by FromXML")
		})
	}
}

func TestFromXMLFull_LayoutWhitespaceIsRemoved(t *testing.T) {
	got := marshalFull(t, readFixture(t, "vast4_inline_full.xml"))

	assert.Contains(t, got, `<Impression id="dsp"><![CDATA[https://dsp.example/imp]]></Impression>`, "indentation around CDATA isn't part of the URL")
	assert.Contains(t, got, `<Description>Line one&#xA;line two</Description>`, "a newline inside text is kept")
}

func TestFromXMLFull_RoundTripIsStable(t *testing.T) {
	for _, fixture := range []string{"vast4_inline_full.xml", "vast4_wrapper_full.xml", "vast3_vpaid_pricing_text.xml", "vast_inline_linear.xml"} {
		t.Run(fixture, func(t *testing.T) {
			once := marshalFull(t, readFixture(t, fixture))
			assert.Equal(t, once, marshalFull(t, []byte(once)))
		})
	}
}

func TestFromXMLFull_Malformed(t *testing.T) {
	_, err := FromXMLFull([]byte(`<VAST version="4.0"><Ad>`))
	assert.Error(t, err)
}

func TestFromXML_KeepsLastJavaScriptResource(t *testing.T) {
	v, err := FromXML(readFixture(t, "vast4_inline_full.xml"))
	require.NoError(t, err)

	resources := v.Ads[0].InLine.AdVerifications.AdVerifications[0].JavaScriptResources
	require.Len(t, resources, 1)
	assert.Equal(t, "https://v.example/app.js", resources[0].URI)
}

func TestAdParameters_MarshalXML(t *testing.T) {
	tests := []struct {
		name   string
		params AdParameters
		want   string
	}{
		{
			name:   "legacy, xmlEncoded — CDATA as before",
			params: AdParameters{XMLEncoded: true, Parameters: []byte("<p>")},
			want:   `<AdParameters xmlEncoded="true"><![CDATA[<p>]]></AdParameters>`,
		},
		{
			name:   "full, xmlEncoded — escaped text",
			params: AdParameters{XMLEncoded: true, Parameters: []byte("<p>"), keepEncoding: true},
			want:   `<AdParameters xmlEncoded="true">&lt;p&gt;</AdParameters>`,
		},
		{
			name:   "full, xmlEncoded value sent already escaped inside CDATA — CDATA, not escaped twice",
			params: AdParameters{XMLEncoded: true, Parameters: []byte("&lt;p&gt;"), keepEncoding: true},
			want:   `<AdParameters xmlEncoded="true"><![CDATA[&lt;p&gt;]]></AdParameters>`,
		},
		{
			name:   "full, not encoded — CDATA",
			params: AdParameters{Parameters: []byte(`{"a":1}`), keepEncoding: true},
			want:   `<AdParameters><![CDATA[{"a":1}]]></AdParameters>`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b, err := xml.Marshal(tt.params)
			require.NoError(t, err)
			assert.Equal(t, tt.want, string(b))
		})
	}
}

func TestHTMLResource_MarshalXML(t *testing.T) {
	legacy, err := xml.Marshal(HTMLResource{XMLEncoded: true, HTML: []byte("<b>")})
	require.NoError(t, err)
	assert.Equal(t, `<HTMLResource xmlEncoded="true"><![CDATA[<b>]]></HTMLResource>`, string(legacy))

	full, err := xml.Marshal(HTMLResource{XMLEncoded: true, HTML: []byte("<b>"), keepEncoding: true})
	require.NoError(t, err)
	assert.Equal(t, `<HTMLResource xmlEncoded="true">&lt;b&gt;</HTMLResource>`, string(full))

	escapedInCDATA, err := xml.Marshal(HTMLResource{XMLEncoded: true, HTML: []byte("&lt;b&gt;"), keepEncoding: true})
	require.NoError(t, err)
	assert.Equal(t, `<HTMLResource xmlEncoded="true"><![CDATA[&lt;b&gt;]]></HTMLResource>`, string(escapedInCDATA))
}

// TestFromXMLFull_XMLEncodedRoundTrip covers both ways an ad server sends XML-encoded content:
// each comes back in the form it was sent, so the player decodes it once to the same value.
func TestFromXMLFull_XMLEncodedRoundTrip(t *testing.T) {
	tests := []struct {
		name string
		in   string
	}{
		{name: "escaped text", in: `<AdParameters xmlEncoded="true">&lt;p&gt;x&lt;/p&gt;</AdParameters>`},
		{name: "escaped inside CDATA", in: `<AdParameters xmlEncoded="true"><![CDATA[&lt;p&gt;x&lt;/p&gt;]]></AdParameters>`},
		{name: "companion HTML, escaped text", in: `<HTMLResource xmlEncoded="true">&lt;b&gt;x&lt;/b&gt;</HTMLResource>`},
		{name: "companion HTML, escaped inside CDATA", in: `<HTMLResource xmlEncoded="true"><![CDATA[&lt;b&gt;x&lt;/b&gt;]]></HTMLResource>`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var creative string
			if strings.HasPrefix(tt.in, "<AdParameters") {
				creative = `<Linear><Duration>00:00:10</Duration>` + tt.in + `</Linear>`
			} else {
				creative = `<CompanionAds><Companion width="1" height="1">` + tt.in + `</Companion></CompanionAds>`
			}

			doc := `<VAST version="4.1"><Ad><InLine><AdSystem>D</AdSystem><Creatives><Creative>` + creative +
				`</Creative></Creatives></InLine></Ad></VAST>`

			assert.Contains(t, marshalFull(t, []byte(doc)), tt.in)
		})
	}
}

// TestFromXMLFull_CDATAURLsAreTrimmed covers URLs a DSP puts on their own line inside CDATA:
// FromXML removed the newlines, and FromXMLFull must not pass them on to the player.
func TestFromXMLFull_CDATAURLsAreTrimmed(t *testing.T) {
	const doc = "<VAST version=\"4.1\"><Ad><Wrapper><AdSystem>D</AdSystem>" +
		"<Impression><![CDATA[\n  https://dsp/imp\n]]></Impression>" +
		"<VASTAdTagURI><![CDATA[\n\thttps://dsp/vast \n]]></VASTAdTagURI>" +
		"<Extensions><Extension><![CDATA[\nvar a = 1;\nvar b = 2;\n]]></Extension></Extensions>" +
		"</Wrapper></Ad></VAST>"

	got := marshalFull(t, []byte(doc))

	assert.Contains(t, got, `<Impression><![CDATA[https://dsp/imp]]></Impression>`)
	assert.Contains(t, got, `<VASTAdTagURI><![CDATA[https://dsp/vast]]></VASTAdTagURI>`)
	assert.Contains(t, got, "var a = 1;\nvar b = 2;", "newlines inside a script are kept")
}

func TestMarshalXMLFull(t *testing.T) {
	v, err := FromXMLFull(readFixture(t, "vast3_vpaid_pricing_text.xml"))
	require.NoError(t, err)

	b, err := v.MarshalXMLFull()
	require.NoError(t, err)

	assert.True(t, strings.HasPrefix(string(b), xml.Header+"<VAST"))
	assert.Contains(t, string(b), "a // b\nc", "newlines kept, unlike MarshalXML")
}

func TestTrimLayout(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "indentation between tags", in: "<A>\n\t<B>x</B>\n</A>", want: "<A><B>x</B></A>"},
		{name: "whitespace around CDATA", in: "<A>\n  <![CDATA[u]]>\n</A>", want: "<A><![CDATA[u]]></A>"},
		{name: "CDATA edges trimmed, inner newline and tab kept", in: "<A><![CDATA[ a\n\tb ]]></A>", want: "<A><![CDATA[a\n\tb]]></A>"},
		{name: "CDATA URL on its own lines", in: "<A><![CDATA[\n  https://u\n]]></A>", want: "<A><![CDATA[https://u]]></A>"},
		{name: "whitespace-only CDATA", in: "<A><![CDATA[ \n ]]></A>", want: "<A><![CDATA[]]></A>"},
		{name: "empty CDATA", in: "<A><![CDATA[]]></A>", want: "<A><![CDATA[]]></A>"},
		{name: "text edges trimmed, inner newline kept", in: "<A>\n x\ny \n</A>", want: "<A>x\ny</A>"},
		{name: "text before CDATA keeps its trailing space", in: "<A>x <![CDATA[y]]></A>", want: "<A>x <![CDATA[y]]></A>"},
		{name: "text after CDATA keeps its leading space", in: "<A><![CDATA[y]]> x</A>", want: "<A><![CDATA[y]]> x</A>"},
		{name: "space between two CDATA sections kept", in: "<A><![CDATA[a]]> <![CDATA[b]]></A>", want: "<A><![CDATA[a]]> <![CDATA[b]]></A>"},
		{name: "attribute with > and newline untouched", in: "<A b=\"1>\n2\" c='>'>\n</A>", want: "<A b=\"1>\n2\" c='>'></A>"},
		{name: "comment and declaration untouched", in: "<?xml version=\"1.0\"?>\n<!-- a\n b -->\n<A/>", want: "<?xml version=\"1.0\"?><!-- a\n b --><A/>"},
		{name: "doctype with internal subset", in: "<!DOCTYPE A [<!ENTITY e \"x\">]>\n<A/>", want: "<!DOCTYPE A [<!ENTITY e \"x\">]><A/>"},
		{name: "unterminated CDATA copied to the end", in: "<A><![CDATA[x", want: "<A><![CDATA[x"},
		{name: "empty", in: "", want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, string(trimLayout([]byte(tt.in))))
		})
	}
}

func BenchmarkFromXML(b *testing.B) {
	raw := readFixture(b, "vast4_inline_full.xml")

	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		if _, err := FromXML(raw); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkFromXMLFull(b *testing.B) {
	raw := readFixture(b, "vast4_inline_full.xml")

	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		if _, err := FromXMLFull(raw); err != nil {
			b.Fatal(err)
		}
	}
}

// TestFromXMLFull_AcceptsWhatFromXMLAccepts covers the only values the two parsers treated
// differently: duration and offset attributes holding newlines or tabs.
func TestFromXMLFull_AcceptsWhatFromXMLAccepts(t *testing.T) {
	const doc = "<VAST version=\"3.0\"><Ad><InLine><AdSystem>D</AdSystem><Creatives><Creative>" +
		"<Linear skipoffset=\"\n00:00:05\t\"><Duration>00:00:10</Duration><TrackingEvents>" +
		"<Tracking event=\"progress\" offset=\"\t25%\n\"><![CDATA[https://t]]></Tracking></TrackingEvents></Linear></Creative>" +
		"<Creative><NonLinearAds><NonLinear width=\"1\" height=\"1\" minSuggestedDuration=\"00:00:\n03\"></NonLinear></NonLinearAds></Creative>" +
		"</Creatives></InLine></Ad></VAST>"

	legacy, err := FromXML([]byte(doc))
	require.NoError(t, err)

	full, err := FromXMLFull([]byte(doc))
	require.NoError(t, err)

	legacyXML, err := xml.Marshal(legacy)
	require.NoError(t, err)

	fullXML, err := xml.Marshal(full)
	require.NoError(t, err)

	assert.Equal(t, string(legacyXML), string(fullXML))
	assert.Contains(t, string(fullXML), `skipoffset="00:00:05"`)
	assert.Contains(t, string(fullXML), `offset="25%"`)
	assert.Contains(t, string(fullXML), `minSuggestedDuration="00:00:03"`)
}
