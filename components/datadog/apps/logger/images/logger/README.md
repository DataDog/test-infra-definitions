#  Logger app

This is a simple service that allows you to log ondemand. The logger support writing to `stdout` and `stderr`. It can also be configured to write to a TCP or UDP endpoint.

## Config options

| Config    | Value     | Default | Description                                                             |
|-----------|-----------|---------|-------------------------------------------------------------------------|
| `port`    | int       | `3333`  | port to listen on                                                       |
| `udp`     | bool      | `false` | if `true` sends logs via UDP to address set in `target`                 |
| `tcp`     | bool      | `false` | if `true` sends logs via TCP to address set in `target`                 |
| `target`  | string    | `false` | if `udp` or `tcp` set then `target` is required (e.g. `127.0.0.1:8080`) |


## Paylod to generate logs

The following is the payload that the service accepts. The service will walk through each item in `data` and log the contetns of `message`.

```
{
  "data": [
    {
      "message": "some text"
    },
    {
      "message": "c29tZSB0ZXh0",
      "encoded": true
    },
    {
      "message": "some text",
      "output": "stderr"
    }
  ]
}
```

* If `encoded` set to `true` then the service will assume it `base64` encoded and will decode it before logging it.
* If `output` set to `stderr` then the service will write it to `stderr`.
