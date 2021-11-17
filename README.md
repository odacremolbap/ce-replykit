# ce-replykit

Utility that receives CloudEvents containing instructions on how to reply.

The `ce-replykit` helps testing scenarios where replies are involved in different ways:

- ACK.
- NACK.
- Timeouts/Delays.
- Retries (based on CloudEvent ID).
- A combination of all of the above.

## Deploy

`ce-replykit` should be deployed using [ko](https://github.com/google/ko).

```console
ko apply -f deploy.yaml
```

Building the binary is `go` straightforward.

```console
mkdir -p output && go build -o output/ce-replykit
```

## Configuration

 `ce-replykit` contains an in-memory request store that aggregates the number of requests received per event ID, which enables the creation of conditional instructions based on retries.

 This in-memory store will consider stale and remove any request after a  configurable TTL. You can find examples of the retry usage cases [below](#examples).

 The TTL for received request stats is set by default to 300 and can be customized via the `CE_REPLY_KIT_STORAGE_TTL_SECONDS` environment variable at the deployment manifest.

## Usage

CloudEvents Reply Kit expect accepts Events containing a JSON array of instructions. Each instruction might contain:

- Condition: determines if the instruction needs to be executed.
- Action: defines the action to execute.
- Reply: JSON element that will be used by actions when a reply payload is needed.

Instruction example:

```json
{
       "condition":"always",
       "action":"ack+event",
       "reply": {"hello":"world"}
}
```

Condition element is optional and when non existing defaults to`always`. CloudEvents toolkit will iterate instructions in order until one condition is true, in which case it will execute its action and return. Possible values are:

- `always` condition will always be true.
- `retrycount_gt: <retries>`, will be true if the same Event (with same Event ID) is sent more than a number of times.
- `retrycount_lt: <retries>`, will be true if the same Event (with same Event ID) has been sent less than a number of times.

Action element possible values are:
- `delay-ack: <seconds>`, waits a number of seconds and then returns ACK.
- `ack`, returns ACK.
- `nack`, returns NACK.
- `ack+event`, returns ACK along with a reply payload.
- `nack+event`, returns NACK along with a reply payload.
- `reset`, deletes cached request retry count at server.

## Examples

To try this examples run a local instance of the `ce-replykit`:

```console
go run .
```

Return ACK

```console
curl -v "http://localhost:8080" \
       -X POST \
       -H "Ce-Id: 536808d3-88be-4077-9d7a-a3f162705f70" \
       -H "Ce-Specversion: 1.0" \
       -H "Ce-Type: replykit.instructions" \
       -H "Ce-Source: curl.shell" \
       -H "Content-Type: application/json" \
       -d '[{"condition":"always","action":"ack"}]'

```

Return NACK the first 2 times the request is sent, return ACK after third retry. To test this case you need to issue the `curl` command 3 times using the same `Ce-Id` header.


```sh
curl -v "http://localhost:8080" \
       -X POST \
       -H "Ce-Id: 536808d3-88be-4077-9d7a-a3f162705f71" \
       -H "Ce-Specversion: 1.0" \
       -H "Ce-Type: replykit.instructions" \
       -H "Ce-Source: curl.shell" \
       -H "Content-Type: application/json" \
       -d '[
              {"condition":"retrycount_lt: 2","action":"nack"},
              {"condition":"always","action":"ack"}
       ]'

```

Return ACK plus a reply. In this case the reply's payload contains also an instruction which could be consumed by a different instance of the `ce-replykit` to return an ACK.

```sh
curl -v "http://localhost:8080" \
       -X POST \
       -H "Ce-Id: 536808d3-88be-4077-9d7a-a3f162705f71" \
       -H "Ce-Specversion: 1.0" \
       -H "Ce-Type: replykit.instructions" \
       -H "Ce-Source: curl.shell" \
       -H "Content-Type: application/json" \
       -d '[{"action":"ack+event","reply":[{"action":"ack"}]}]'
```

