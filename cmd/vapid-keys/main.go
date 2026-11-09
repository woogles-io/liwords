// vapid-keys prints a new VAPID key pair for web push, as the two settings
// liwords-api reads: VAPID_PUBLIC_KEY and VAPID_PRIVATE_KEY.
//
// Make the pair once. Every push subscription is bound to the public key, so
// a new pair breaks every subscription the old one made. The private key is a
// secret and goes only into the api's secrets.
package main

import (
	"fmt"
	"os"

	webpush "github.com/SherClockHolmes/webpush-go"
)

func main() {
	privateKey, publicKey, err := webpush.GenerateVAPIDKeys()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("VAPID_PUBLIC_KEY=%s\nVAPID_PRIVATE_KEY=%s\n", publicKey, privateKey)
}
