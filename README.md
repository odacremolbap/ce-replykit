# ce-replykit

WIP CloudEvents Reply Kit.

Utility that receives CloudEvents containing instructions on how to reply.

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

# Examples

Return ACK

```sh
curl -v "http://localhost:8080" \
       -X POST \
       -H "Ce-Id: 536808d3-88be-4077-9d7a-a3f162705f70" \
       -H "Ce-Specversion: 1.0" \
       -H "Ce-Type: replykit.instructions" \
       -H "Ce-Source: curl.shell" \
       -H "Content-Type: application/json" \
       -d '[{"condition":"always","action":"ack"}]'

```

Return NACK the first 2 times the request is sent, return ACK after third retry.

```sh
curl -v "http://localhost:8080" \
       -X POST \
       -H "Ce-Id: 536808d3-88be-4077-9d7a-a3f162705f71" \
       -H "Ce-Specversion: 1.0" \
       -H "Ce-Type: replykit.instructions" \
       -H "Ce-Source: curl.shell" \
       -H "Content-Type: application/json" \
       -d '[
              {"condition":"retrycount_lt: 3","action":"nack"},
              {"condition":"always","action":"ack"}
       ]'

```

## Deploy

```sh
ko apply -f .local/conformance/01-prepare
```
