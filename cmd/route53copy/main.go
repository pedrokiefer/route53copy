package main

import (
	"github.com/pedrokiefer/route53copy/cmd"
	"github.com/pedrokiefer/route53copy/cmd/route53copy/app"
)

func main() {
	cmd.Run(app.NewCommand())
}
