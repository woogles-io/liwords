package main

import (
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

	msg := flag.String("msg", "", "the message, in hexadecimal")
	flag.Parse()

	var pbmsg proto.Message
	raw, err := hex.DecodeString(*msg)
	if err != nil {
		panic(err)
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
		default:
			panic("message " + *messageName + " not handled")
		}
	} else {
		panic("protofile " + *protofile + " not handled")
	}

	err = proto.Unmarshal(raw, pbmsg)
	if err != nil {
		panic(err)
	}
	b, err := json.Marshal(pbmsg)
	if err != nil {
		panic(err)
	}
	fmt.Println(string(b))
}
