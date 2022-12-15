package main

import (
	"encoding/base64"
	"encoding/hex"
	"flag"
	"fmt"
	"io/ioutil"
	"os"

	ipc "github.com/domino14/liwords/rpc/api/proto/ipc"
	macondopb "github.com/domino14/macondo/gen/api/proto/macondo"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// A script to convert protobuf from binary to JSON

func main() {
	protofile := flag.String("protofile", "macondo", "the name of the protofile: macondo or ipc")
	messageName := flag.String("messagename", "", "the name of the pb message")
	conversion := flag.String("conversion", "hex2json", "hex2json, b642json, json2hex, binfile2hex, json2binfile")
	// pb packets that get sent through the socket have two bytes for length and one for msg type; skip these in this case.
	skipheader := flag.Bool("skipheader", false, "if the format is a binary one, and this flag is enabled, skip the first three bytes of the packet")

	msg := flag.String("msg", "", "the message, in hexadecimal, base64, json, or a filename with binary")
	flag.Parse()

	var pbmsg proto.Message
	var raw []byte
	var err error
	if *conversion == "hex2json" {
		raw, err = hex.DecodeString(*msg)
		if err != nil {
			panic(err)
		}
	} else if *conversion == "b642json" {
		raw, err = base64.StdEncoding.DecodeString(*msg)
		if err != nil {
			panic(err)
		}
	} else if *conversion == "binfile2hex" {
		raw, err = ioutil.ReadFile(*msg)
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

	} else if *protofile == "ipc" {
		switch *messageName {
		case "GameRequest":
			pbmsg = &ipc.GameRequest{}
		case "GameDocument":
			pbmsg = &ipc.GameDocument{}
		case "GameHistoryRefresher":
			pbmsg = &ipc.GameHistoryRefresher{}
		case "ServerGameplayEvent":
			pbmsg = &ipc.ServerGameplayEvent{}
		case "FullTournamentDivisions":
			pbmsg = &ipc.FullTournamentDivisions{}
		default:
			panic("message " + *messageName + " not handled")
		}
	} else {
		panic("protofile " + *protofile + " not handled")
	}
	var b []byte
	var bstr string

	if *conversion == "hex2json" || *conversion == "b642json" || *conversion == "binfile2json" {
		if *skipheader {
			raw = raw[3:]
		}
		err = proto.Unmarshal(raw, pbmsg)
		if err != nil {
			panic(err)
		}
		b, err = protojson.Marshal(pbmsg)
		if err != nil {
			panic(err)
		}
		bstr = string(b)
	} else if *conversion == "json2hex" || *conversion == "json2binfile" {
		err = protojson.Unmarshal([]byte(*msg), pbmsg)
		if err != nil {
			panic(err)
		}
		b, err = proto.Marshal(pbmsg)
		if err != nil {
			panic(err)
		}
		if *conversion == "json2hex" {
			bstr = hex.EncodeToString(b)
		}
	}
	if *conversion != "json2binfile" {
		fmt.Println(bstr)
	} else {
		f, err := os.Create("/tmp/out.pb")
		if err != nil {
			panic(err)
		}
		_, err = f.Write(b)
		if err != nil {
			panic(err)
		}
		fmt.Println("wrote to /tmp/out.pb")
	}
}
