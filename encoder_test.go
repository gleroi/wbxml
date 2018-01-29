package wbxml

import (
	"bytes"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEncoderEncodeTag(t *testing.T) {
	tests := []struct {
		tag      StartElement
		expected []byte
	}{
		{
			tag:      StartElement{Name: "SyncML"},
			expected: []byte{0x2D},
		},
		{
			tag:      StartElement{Name: "SyncML", Content: true},
			expected: []byte{0x6D},
		},
		{
			tag: StartElement{Name: "SyncML", Content: true,
				Attr: []Attr{Attr{Name: "A"}}},
			expected: []byte{0xED, 05, 01},
		},
		{
			tag:      StartElement{Name: "CS"},
			expected: []byte{0x00, 0x08, 0x05},
		},
	}

	for testID, test := range tests {
		buf := bytes.NewBuffer(nil)
		{
			e := NewEncoder(buf, SyncMLTags, CodeSpace{0: CodePage{5: "A"}})
			err := e.encodeTag(test.tag)
			if err != nil {
				t.Errorf("case %d: unexpected error: %s", testID, err)
				continue
			}
			assert.Equal(t, test.expected, buf.Bytes(), "case %d", testID)
		}
	}
}

func TestEncoderToken(t *testing.T) {
	for testID, tokens := range tokensExamples {
		expected := encodingExamples[testID]
		space := tagSpaceExamples[testID]

		w := bytes.NewBuffer(nil)
		e := NewEncoder(w, space.tags, space.attrs)

		err := e.EncodeHeader(headerExamples[testID])
		if err != nil {
			t.Fatalf("case %d: encoding header: %s", testID, err)
		}

		for _, tok := range tokens {
			if tok == nil {
				break
			}
			err := e.EncodeToken(tok)
			if err != nil {
				t.Errorf("error: token %v: %s", tok, err)
			}
		}

		assert.Equal(t, expected, w.Bytes(), "case %d", testID)
	}
}

type fullmsg struct {
	SyncHdr  fullheader
	SyncBody body
}

type fullheader struct {
	VerDTD    string
	VerProto  string
	SessionID string
	MsgID     uint32
	Source    endpoint
	Target    endpoint
	Meta      meta
}

func TestEncoderEncode(t *testing.T) {
	var syncMlMsg1 = fullmsg{
		SyncHdr: fullheader{
			VerDTD:    "1.2",
			VerProto:  "m2m/1.2",
			SessionID: "S7eNe",
			MsgID:     94,
			Source: endpoint{
				LocURI: "tcp://Accueil.NocId.amm.fr",
			},
			Target: endpoint{
				LocURI: "gdo:99005Z1338-21178",
			},
			Meta: meta{
				EMI: emi{
					Sign: []byte{0x30, 0x46, 0x02, 0x21, 0x00, 0x9a, 0x9f, 0x72, 0x4f, 0x51, 0x46, 0xb6, 0xe2, 0x6a, 0x35, 0x7b, 0x4b, 0x53, 0x22, 0x13, 0x88, 0xbe, 0xef, 0x1a, 0x95, 0xc6, 0xf4, 0xba, 0x9f, 0x05, 0x72, 0xd5, 0x85, 0x4f, 0x02, 0x3e, 0x54, 0x02, 0x21, 0x00, 0x8d, 0xd8, 0x85, 0xe0, 0x88, 0x28, 0x43, 0x6c, 0x6e, 0x2b, 0x08, 0xfb, 0xb8, 0x16, 0xd3, 0x59, 0x79, 0x1b, 0x9d, 0x8c, 0xb1, 0xca, 0x63, 0x34, 0xf8, 0x20, 0x1f, 0xee, 0x13, 0x09, 0x09, 0xa9},
				},
			},
		},
		SyncBody: body{
			Status: status{
				CmdID:  1,
				MsgRef: 93,
				CmdRef: 1,
				Cmd:    "Put",
				Data:   500,
			},
			Final: true,
		},
	}

	w := bytes.NewBuffer(nil)
	d := NewEncoder(w, SyncMLTags, CodeSpace{})
	err := d.EncodeHeader(Header{
		Version:     3,
		PublicID:    0,
		Charset:     3,
		StringTable: []byte{0x12, 0x01},
	})
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}
	err = d.EncodeElement(syncMlMsg1, StartElement{Name: "SyncML"})
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}

	if !assert.Equal(t, syncMLInput, w.Bytes()) {
		XML(os.Stdout, NewDecoder(w, SyncMLTags, CodeSpace{}), " ")
	}
}
