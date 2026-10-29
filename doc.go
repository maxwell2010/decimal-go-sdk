// Package decimal implements Decimal EVM transaction construction, signing,
// contract calls and API access. Building a Call never contacts the network.
// Sending requires an explicit Client.Send or Client.Broadcast invocation.
// Cosmos protobuf messages are a separate protocol and are not EVM calls.
package decimal
