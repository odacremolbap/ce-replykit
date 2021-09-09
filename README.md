# ce-replykit

WIP CloudEvents Reply Kit.

## Usage

Conditions:
- always
- requestcount_gt, requires integer value
- requestcount_lt, requires integer value

```sh
curl -v "http://localhost:8080" \
       -X POST \
       -H "Ce-Id: 536808d3-88be-4077-9d7a-a3f162705f79" \
       -H "Ce-Specversion: 1.0" \
       -H "Ce-Type: test.event" \
       -H "Ce-Source: curl.shell" \
       -H "Content-Type: application/json" \
       -d '[{"condition":"requestcount_gt: 3"}]'
```

Actions:
- delay-ack, requires integer value
- ack
- nack
- ack+payload
- nack+payload

```sh
curl -v "http://localhost:8080" \
       -X POST \
       -H "Ce-Id: 536808d3-88be-4077-9d7a-a3f162705f79" \
       -H "Ce-Specversion: 1.0" \
       -H "Ce-Type: test.event" \
       -H "Ce-Source: curl.shell" \
       -H "Content-Type: application/json" \
       -d '[{"condition":"requestcount_lt: 4", "action":"delay-ack: 3"},
            {"condition":"requestcount_lt: 7", "action":"nack"},
            {"action":"ack"}]'
```

```sh
curl -v "http://localhost:8080" \
       -X POST \
       -H "Ce-Id: 536808d3-88be-4077-9d7a-a3f162705f79" \
       -H "Ce-Specversion: 1.0" \
       -H "Ce-Type: test.event" \
       -H "Ce-Source: curl.shell" \
       -H "Content-Type: application/json" \
       -d '[{"condition":"requestcount_lt: 3", "action":"ack"},{"action":"nack"}]'
```

