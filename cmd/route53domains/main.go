package main

import (
	"github.com/pedrokiefer/route53copy/cmd"
	"github.com/pedrokiefer/route53copy/cmd/route53domains/app"
)

func main() {
	cmd.Run(app.NewCommand())
}
