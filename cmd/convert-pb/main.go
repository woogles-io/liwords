package main

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"

	realtime "github.com/domino14/liwords/rpc/api/proto/realtime"
	macondopb "github.com/domino14/macondo/gen/api/proto/macondo"
	"google.golang.org/protobuf/proto"
)

// A script to convert protobuf from binary to JSON

func main() {
	protofile := flag.String("protofile", "macondo", "the name of the protofile: macondo or realtime")
	messageName := flag.String("messagename", "", "the name of the pb message")
	convertfrom := flag.String("convertfrom", "hex", "hex: hex->json,  b64: b64->json,  json: json->hex")
	// pb packets that get sent through the socket have two bytes for length and one for msg type; skip these in this case.
	skipheader := flag.Bool("skipheader", false, "if the format is a binary one, and this flag is enabled, skip the first three bytes of the packet")

	msg := flag.String("msg", "", "the message, in hexadecimal, base64, or json")
	flag.Parse()

	var pbmsg proto.Message
	var raw []byte
	var err error
	if *convertfrom == "hex" {
		raw, err = hex.DecodeString(*msg)
		if err != nil {
			panic(err)
		}
	} else if *convertfrom == "b64" {
		raw, err = base64.StdEncoding.DecodeString(*msg)
		if err != nil {
			panic(err)
		}
	}

	if *protofile == "macondo" {

		switch *messageName {
		case "GameHistory":
			pbmsg = &macondopb.GameHistory{}
		default:
			panic("message " + *messageName + " not handled")
		}

	} else if *protofile == "realtime" {
		switch *messageName {
		case "GameRequest":
			pbmsg = &realtime.GameRequest{}
		case "GameHistoryRefresher":
			pbmsg = &realtime.GameHistoryRefresher{}
		default:
			panic("message " + *messageName + " not handled")
		}
	} else {
		panic("protofile " + *protofile + " not handled")
	}
	var b []byte
	var bstr string

	if *convertfrom == "hex" || *convertfrom == "b64" {
		if *skipheader {
			raw = raw[3:]
		}
		err = proto.Unmarshal(raw, pbmsg)
		if err != nil {
			panic(err)
		}
		b, err = json.Marshal(pbmsg)
		if err != nil {
			panic(err)
		}
		bstr = string(b)
	} else if *convertfrom == "json" {
		err = json.Unmarshal([]byte(*msg), pbmsg)
		if err != nil {
			panic(err)
		}
		b, err = proto.Marshal(pbmsg)
		if err != nil {
			panic(err)
		}
		bstr = hex.EncodeToString(b)
	}
	fmt.Println(bstr)
}
