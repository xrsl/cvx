package main

import "github.com/xrsl/cvx/cmd"

func main() {
	// Pass embedded agent FS to cmd package
	cmd.SetAgentFS(&AgentFS)
	cmd.Execute()
}
