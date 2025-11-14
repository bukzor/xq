package utils

import (
	"bytes"
	"encoding/json"
	"os"
	"path"
	"strings"
	"testing"

	"github.com/antchfx/xmlquery"
	"github.com/stretchr/testify/assert"
)

func TestXmlToJSON(t *testing.T) {
	tests := []struct {
		unformattedFile string
		expectedFile    string
		depth           int
	}{
		{"unformatted.xml", "formatted.json", -1},
		{"unformatted2.xml", "formatted2.json", -1},
		{"unformatted3.xml", "formatted3.json", -1},
		{"unformatted4.xml", "formatted4.json", 1},
	}

	for _, testCase := range tests {
		inputFileName := path.Join("..", "..", "test", "data", "xml2json", testCase.unformattedFile)
		unformattedXmlReader := getFileReader(inputFileName)

		outputFileName := path.Join("..", "..", "test", "data", "xml2json", testCase.expectedFile)
		data, jsonReadErr := os.ReadFile(outputFileName)
		assert.Nil(t, jsonReadErr)
		expectedJson := string(data)

		node, parseErr := xmlquery.Parse(unformattedXmlReader)
		assert.Nil(t, parseErr)
		result := NodeToJSON(node, testCase.depth)
		jsonData, jsonMarshalErr := json.Marshal(result)
		assert.Nil(t, jsonMarshalErr)

		output := new(strings.Builder)
		formatErr := FormatJson(bytes.NewReader(jsonData), output, "  ", ColorsDisabled)
		assert.Nil(t, formatErr)
		assert.Equal(t, expectedJson, output.String())
	}
}

func TestExhaustiveNodeTypeHandling(t *testing.T) {
	// Test that all xmlquery node types are handled without panicking
	// This verifies our exhaustive switch statements work correctly

	xmlInput := `<?xml version="1.0"?>
<!DOCTYPE root>
<!-- This is a comment -->
<root>
	<element>text content</element>
	<cdata><![CDATA[raw & unescaped < > content]]></cdata>
	<!-- another comment inside -->
	<?processing-instruction data?>
	<mixed>text<child>more</child>tail</mixed>
</root>`

	node, err := xmlquery.Parse(strings.NewReader(xmlInput))
	assert.NoError(t, err)

	// Should not panic - this exercises all the node types
	result := NodeToJSON(node, -1)
	assert.NotNil(t, result)

	// Verify the result is an OrderedMap
	resultMap, ok := result.(*OrderedMap)
	assert.True(t, ok)

	// Verify root element exists
	root, ok := resultMap.Get("root")
	assert.True(t, ok)

	rootMap, ok := root.(*OrderedMap)
	assert.True(t, ok)

	// Verify CDATA is preserved as text
	cdataElem, ok := rootMap.Get("cdata")
	assert.True(t, ok)
	assert.Contains(t, cdataElem, "raw & unescaped")
}

func TestSelfClosingTagsHaveNullContent(t *testing.T) {
	// Self-closing XML elements should have null content in JSON
	xmlInput := `<root><self-closing/></root>`

	node, err := xmlquery.Parse(strings.NewReader(xmlInput))
	assert.NoError(t, err)

	result := NodeToJSON(node, -1)
	assert.NotNil(t, result)

	resultMap, ok := result.(*OrderedMap)
	assert.True(t, ok)

	rootVal, ok := resultMap.Get("root")
	assert.True(t, ok)
	rootMap, ok := rootVal.(*OrderedMap)
	assert.True(t, ok)

	selfClosing, _ := rootMap.Get("self-closing")
	assert.Nil(t, selfClosing, "Self-closing tag should have null content")
}

func TestWhitespaceOnlyTagsAreSelfClosing(t *testing.T) {
	// Establishes that the XML formatter treats whitespace-only elements as self-closing
	xmlInput := `<root><spaces>  </spaces><newlines>
</newlines><empty></empty><immediate-closing></immediate-closing></root>`

	node, err := xmlquery.Parse(strings.NewReader(xmlInput))
	assert.NoError(t, err)

	result := NodeToJSON(node, -1)
	assert.NotNil(t, result)

	resultMap, ok := result.(*OrderedMap)
	assert.True(t, ok)

	rootVal, ok := resultMap.Get("root")
	assert.True(t, ok)
	rootMap, ok := rootVal.(*OrderedMap)
	assert.True(t, ok)

	spaces, _ := rootMap.Get("spaces")
	assert.Nil(t, spaces, "Spaces should have null content")
	newlines, _ := rootMap.Get("newlines")
	assert.Nil(t, newlines, "Newlines should have null content")
	empty, _ := rootMap.Get("empty")
	assert.Nil(t, empty, "Empty should have null content")
	immediateClosing, _ := rootMap.Get("immediate-closing")
	assert.Nil(t, immediateClosing, "Immediate-closing should have null content")
}

func TestElementOrderInJSON(t *testing.T) {
	// Test that element order is preserved in JSON output
	xmlInput := `<root>
		<command-name>/status</command-name>
		<command-message>status</command-message>
		<command-args></command-args>
	</root>`

	node, err := xmlquery.Parse(strings.NewReader(xmlInput))
	assert.NoError(t, err)

	result := NodeToJSON(node, -1)
	assert.NotNil(t, result)

	// Marshal to JSON and check order
	jsonData, err := json.Marshal(result)
	assert.NoError(t, err)

	jsonStr := string(jsonData)
	t.Logf("JSON output: %s", jsonStr)

	// Find positions of each key in the JSON string
	namePos := strings.Index(jsonStr, "command-name")
	messagePos := strings.Index(jsonStr, "command-message")
	argsPos := strings.Index(jsonStr, "command-args")

	// Check that they appear in the original order
	assert.True(t, namePos < messagePos, "command-name should appear before command-message")
	assert.True(t, messagePos < argsPos, "command-message should appear before command-args")
}
