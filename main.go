// @title           go-webhook API
// @version         0.1.0
// @description     Configurable webhook forwarding engine. Receives JSON webhooks, transforms payloads using expr rules, and forwards to target services.
// @host            localhost:8080
// @BasePath        /
package main

import "github.com/shidaxi/go-webhook/cmd"

func main() {
	cmd.Execute()
}
