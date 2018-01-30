package wbxml

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"io"
	"os"
)

var syncMLTags = CodeSpace{
	0: CodePage{
		0x05: "Add",
		0x06: "Alert",
		0x07: "Archive",
		0x08: "Atomic",
		0x09: "Chal",
		0x0a: "Cmd",
		0x0b: "CmdID",
		0x0c: "CmdRef",
		0x0d: "Copy",
		0x0e: "Cred",
		0x0f: "Data",
		0x10: "Delete",
		0x11: "Exec",
		0x12: "Final",
		0x13: "Get",
		0x14: "Item",
		0x15: "Lang",
		0x16: "LocName",
		0x17: "LocURI",
		0x18: "Map",
		0x19: "MapItem",
		0x1a: "Meta",
		0x1b: "MsgID",
		0x1c: "MsgRef",
		0x1d: "NoResp",
		0x1e: "NoResults",
		0x1f: "Put",
		0x20: "Replace",
		0x21: "RespURI",
		0x22: "Results",
		0x23: "Search",
		0x24: "Sequence",
		0x25: "SessionID",
		0x26: "SftDel",
		0x27: "Source",
		0x28: "SourceRef",
		0x29: "Status",
		0x2a: "Sync",
		0x2b: "SyncBody",
		0x2c: "SyncHdr",
		0x2d: "SyncML",
		0x2e: "Target",
		0x2f: "TargetRef",
		0x30: "Reserved , future use",
		0x31: "VerDTD",
		0x32: "VerProto",
		0x33: "NumberOfChanged",
		0x34: "MoreData",
		0x35: "Field",
		0x36: "Filter",
		0x37: "Record",
		0x38: "FilterType",
		0x39: "SourceParent",
		0x3a: "TargetParent",
		0x3b: "Move",
		0x3c: "Correlator",
	},
	1: CodePage{
		0x05: "Anchor",
		0x06: "EMI",
		0x07: "Format",
		0x08: "FreeID",
		0x09: "FreeMem",
		0x0a: "Last",
		0x0b: "Mark",
		0x0c: "MaxMsgSize",
		0x0d: "Mem",
		0x0e: "MetInf",
		0x0f: "Next",
		0x10: "NextNonce",
		0x11: "SharedMem",
		0x12: "Size",
		0x13: "Type",
		0x14: "Version",
		0x15: "MaxObjSize",
		0x16: "FieldLevel",
	},
	8: CodePage{
		0x05: "CS",
		0x06: "HorRecv",
		0x07: "HorSend",
		0x08: "CertSign",
		0x09: "Sign",
		0x0A: "Start",
		0x0B: "Stop",
	},
}

func ExampleXML() {
	input := "030000030212016d6c7103312e32000172036d326d2f312e32000165035337654e6500015b025e016757037463703a2f2f4163637565696c2e4e6f6349642e616d6d2e66720001016e570367646f3a39393030355a313333382d32313137380001015a000146000849c34830460221009a9f724f5146b6e26a357b4b53221388beef1a95c6f4ba9f0572d5854f023e540221008dd885e08828436c6e2b08fbb816d359791b9d8cb1ca6334f8201fee130909a901010001010000016b694b0201015c025d014c0201014a0350757400014f028374010152010101"

	data, err := hex.DecodeString(input)
	if err != nil {
		panic(err)
	}
	r := bytes.NewReader(data)
	d := NewDecoder(r, syncMLTags, CodeSpace{})
	w := bytes.NewBuffer(nil)

	err = XML(w, d, "  ")
	if err != nil && err != io.EOF {
		panic(err)
	}
	fmt.Fprintf(os.Stdout, w.String())
	// Output:
	// 	<SyncML>
	//   <SyncHdr>
	//     <VerDTD>1.2</VerDTD>
	//     <VerProto>m2m/1.2</VerProto>
	//     <SessionID>S7eNe</SessionID>
	//     <MsgID>94</MsgID>
	//     <Source>
	//       <LocURI>tcp://Accueil.NocId.amm.fr</LocURI>
	//     </Source>
	//     <Target>
	//       <LocURI>gdo:99005Z1338-21178</LocURI>
	//     </Target>
	//     <Meta>
	//       <EMI>
	//         <Sign>30460221009a9f724f5146b6e26a357b4b53221388beef1a95c6f4ba9f0572d5854f023e540221008dd885e08828436c6e2b08fbb816d359791b9d8cb1ca6334f8201fee130909a9</Sign>
	//       </EMI>
	//     </Meta>
	//   </SyncHdr>
	//   <SyncBody>
	//     <Status>
	//       <CmdID>1</CmdID>
	//       <MsgRef>93</MsgRef>
	//       <CmdRef>1</CmdRef>
	//       <Cmd>Put</Cmd>
	//       <Data>500</Data>
	//     </Status>
	//     <Final></Final>
	//   </SyncBody>
	// </SyncML>
}
