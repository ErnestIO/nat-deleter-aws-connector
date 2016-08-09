/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at http://mozilla.org/MPL/2.0/. */

package main

import (
	"fmt"
	"os"
	"runtime"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	ecc "github.com/ernestio/ernest-config-client"
	"github.com/nats-io/nats"
)

var nc *nats.Conn
var natsErr error

func eventHandler(m *nats.Msg) {
	var n Event

	err := n.Process(m.Data)
	if err != nil {
		nc.Publish("nat.delete.aws.error", m.Data)
		return
	}

	if err = n.Validate(); err != nil {
		n.Error(err)
		return
	}

	err = deleteNat(&n)
	if err != nil {
		n.Error(err)
		return
	}

	n.Complete()
}

func deleteNat(ev *Event) error {
	creds := credentials.NewStaticCredentials(ev.DatacenterAccessKey, ev.DatacenterAccessToken, "")
	svc := ec2.New(session.New(), &aws.Config{
		Region:      aws.String(ev.DatacenterRegion),
		Credentials: creds,
	})

	req := ec2.DeleteNatGatewayInput{
		NatGatewayId: aws.String(ev.NatGatewayAWSID),
	}

	_, err := svc.DeleteNatGateway(&req)
	if err != nil {
		return err
	}

	return nil
}

func main() {
	nc = ecc.NewConfig(os.Getenv("NATS_URI")).Nats()

	fmt.Println("listening for nat.delete.aws")
	nc.Subscribe("nat.delete.aws", eventHandler)

	runtime.Goexit()
}
