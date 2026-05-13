# Technical Design: Gopherbot Microsoft Teams Connector

## 1. Overview

This design specifies a native Microsoft Teams connector for Gopherbot that functions behind a firewall using only Microsoft-native resources. It bypasses the traditional "Inbound Webhook" model in favor of a pull-based change notification architecture using Microsoft Graph and Azure Event Hubs (AEH).

## 2. Architecture

The connector consists of an outbound-only Go routine that establishes a persistent connection to Azure.

### Components

- Azure Event Hubs (Basic Tier): Acts as the event queue (`$0.015/hr`).
- Microsoft Graph API: The source of Teams events.
- Entra ID (App Registration): Provides the bot identity and OAuth2 credentials.
- Gopherbot Connector (Go): The implementation of the `robot.Connector` interface.

### The "Pull" Data Flow

1. Subscription: On startup, Gopherbot ensures a Graph subscription exists via `POST /subscriptions`.
2. Event Delivery: Microsoft Graph pushes change notifications, such as new messages, to the Event Hub.
3. Ingress: Gopherbot pulls these messages via an outbound AMQP connection using `github.com/Azure/azure-sdk-for-go/sdk/messaging/azeventhubs`.
4. Ack/Cursor: Gopherbot manages its own offset cursor in the Event Hub stream to ensure no messages are missed.

## 3. Contract Mapping (`connector_defs.go`)

### Inbound: `ConnectorMessage` Mapping

| Gopherbot Field | Teams Mapping / Implementation |
| --- | --- |
| `Protocol` | `"Teams"` |
| `UserName` | `from.user.displayName` |
| `UserID` | `from.user.id` (Entra Object ID) |
| `ChannelID` | `channelIdentity.channelId` |
| `ThreadID` | `replyToId` if reply, otherwise `id` for a new thread |
| `MessageText` | Sanitized `body.content` with HTML-to-text conversion |
| `DirectMessage` | `true` if `chatId` exists without `channelId` |
| `BotMessage` | `true` if the message includes the bot mention or is a DM |
| `HiddenMessage` | `true` for targeted messages (ephemeral replies) |

### Outbound: Connector Interface Implementation

- `SendProtocolChannelThreadMessage`: Sends a `POST` to `/teams/{id}/channels/{id}/messages`. If a `ThreadID` is provided, the message is sent as a reply.
- `SendProtocolUserMessage`: Sends a `POST` to `/chats/{id}/messages`. If a chat does not exist, the connector first calls `/chats` to create a 1:1 conversation with the `UserID`.
- `GetProtocolUserAttribute`: Queries the Graph API with `GET /users/{id}` to retrieve metadata such as email and job title.

## 4. Advanced Feature Implementation

### Reading All Messages (RSC)

To see messages without `@mentions`, the bot uses Resource-Specific Consent (RSC).

- Manifest requirement: Include `ChannelMessage.Read.Group` in the `authorization.permissions.resourceSpecific` section.
- User experience: When added to a Team, the bot can immediately process all channel traffic.

### Hidden Messages and Private Replies

Gopherbot's "Private Reply in Channel" requirement is handled via the 2026 Targeted Messages for Agents API:

```json
{
  "isTargetedActivity": true,
  "targetedUserIds": ["{UserID}"]
}
```

These messages are ephemeral and disappear after 24 hours.

## 5. Deployment and Cost (4 Robots)

By using a single Azure Event Hubs namespace (Basic Tier), multiple robots can share the same infrastructure.

| Resource | Quantity | Monthly Est. |
| --- | --- | --- |
| AEH Namespace | 1 | `~$11.00` |
| Event Hubs | 4 (1 per robot) | Included |
| Graph API | Unlimited | `$0.00` |
| Total |  | `~$11.00` |

## 6. Implementation Milestones

1. Azure Setup: Configure the AEH namespace and Entra app registration.
2. Go SDK Integration: Implement `github.com/Azure/azure-sdk-for-go/sdk/messaging/azeventhubs`.
3. HTML Sanitizer: Build a robust routine to convert Teams HTML message bodies into the clean strings expected by Gopherbot.
4. Subscription Manager: Add a background task in the Go routine to renew the Graph subscription every 60 minutes, since the Graph default expiry is short.