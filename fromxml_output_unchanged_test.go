package vast

import (
	"encoding/xml"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var updateFromXMLOutput = flag.Bool("update-fromxml-output", false,
	"rewrite testdata/fromxml_output from the current code; only for an intended FromXML output change")

// TestFromXML_OutputUnchanged checks that FromXML output is byte-identical to the output saved in
// testdata/fromxml_output. Those files were written before the VAST 4 fields were added, so this
// proves existing callers get exactly what they got before. Two outputs are saved per fixture:
// <name>.xml_marshal.xml (xml.Marshal) and <name>.vast_marshalxml.xml (VAST.MarshalXML).
func TestFromXML_OutputUnchanged(t *testing.T) {
	fixtures, err := filepath.Glob("testdata/*.xml")
	require.NoError(t, err)
	require.NotEmpty(t, fixtures)

	for _, fixture := range fixtures {
		name := strings.TrimSuffix(filepath.Base(fixture), ".xml")

		t.Run(name, func(t *testing.T) {
			raw, err := os.ReadFile(fixture)
			require.NoError(t, err)

			v, err := FromXML(raw)
			require.NoError(t, err)

			marshaled, err := xml.Marshal(v)
			require.NoError(t, err)
			assertSameAsSaved(t, filepath.Join("testdata", "fromxml_output", name+".xml_marshal.xml"), marshaled)

			v, err = FromXML(raw)
			require.NoError(t, err)

			custom, err := v.MarshalXML()
			require.NoError(t, err)
			assertSameAsSaved(t, filepath.Join("testdata", "fromxml_output", name+".vast_marshalxml.xml"), custom)
		})
	}
}

func assertSameAsSaved(t *testing.T, path string, got []byte) {
	t.Helper()

	if *updateFromXMLOutput {
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
		require.NoError(t, os.WriteFile(path, got, 0o644))

		return
	}

	want, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, string(want), string(got))
}
