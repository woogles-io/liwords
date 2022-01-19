// Package ipc implements the inter-process communication between the different
// services of this repo.
package ipc

import "github.com/nats-io/nats.go"


func Request(conn *nats.Conn) 