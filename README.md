# ```webhookd```

## What this project is about

This is a daemon that enables you to run your own webhook service.
It provides a route for generating a webhook which serves static content.
In the future one should be able to upload signed Plugins that add dynamic capabilities.

## How do I install it ?

```
git clone git@github.com:4thel00z/webhookd.git
make build
```

## Ok, but how do I run this cringelord ? :S

In one terminal you can do:

```
➜  webhookd git:(master) make run
2020/10/29 02:58:38

              ___.   .__                   __       .___
__  _  __ ____\_ |__ |  |__   ____   ____ |  | __ __| _/
\ \/ \/ // __ \| __ \|  |  \ /  _ \ /  _ \|  |/ // __ |
 \     /\  ___/| \_\ \   Y  (  <_> |  <_> )    </ /_/ |
  \/\_/  \___  >___  /___|  /\____/ \____/|__|_ \____ |
             \/    \/     \/                   \/    \/

2020/10/29 02:58:38 👩  Version: 0.0.1
2020/10/29 02:58:38 🏁  Listening on [::]:1337
2020/10/29 02:58:38 👠  The routes 🛣️  are:
2020/10/29 02:58:38     http://[::]:1337/v1/debug/routes with method: GET
2020/10/29 02:58:38     Query this endpoint like this:
                curl http://0.0.0.0:1337/v1/debug/routes
2020/10/29 02:58:38     http://[::]:1337/v1/debug/private with method: GET
2020/10/29 02:58:38     Query this endpoint like this:
                curl http://0.0.0.0:1337/v1/debug/private
2020/10/29 02:58:38     http://[::]:1337/v1/webhook/generate with method: POST
2020/10/29 02:58:38     Query this endpoint like this:
                curl -X POST http://0.0.0.0:1337/v1/webhook/generate
2020/10/29 02:58:38     http://[::]:1337/v1/webhook/unregister with method: POST
2020/10/29 02:58:38     Query this endpoint like this:
                curl -X POST http://0.0.0.0:1337/v1/webhook/unregister

```

Using your favourite http client you can now send a request to the webhookd like this:

```
➜  webhookd git:(master) http POST http://0.0.0.0:1337/v1/webhook/generate < examples/generate_webhook.json
HTTP/1.1 200 OK
Content-Length: 60
Content-Type: application/json
Date: Thu, 29 Oct 2020 01:58:43 GMT

{
    "path": "/v1/webhook/4122493a-fd9b-4ec8-7861-bcafc1e4d5c4"
}

```

It will return the path under which you can find your webhook.
The webhook is defined in an (exemplary) file called `examples/generate_webhook.json` which can also be found in this repo.
We produce it here for brevity:

```
{
"method" : "get",
"body" : "4thel00z is the real deal",
"headers" : {}
}
```

We can then simply call the webhook and it will send us back the specified body and headers:

```
➜  webhookd git:(master) ✗ http GET http://0.0.0.0:1337/v1/webhook/4122493a-fd9b-4ec8-7861-bcafc1e4d5c4
HTTP/1.1 200 OK
Content-Length: 28
Content-Type: application/json
Date: Thu, 29 Oct 2020 02:03:12 GMT

"4thel00z is the real deal"

```

## Acknowledgements

We used my [serviced_template](https://github.com/4thel00z/service_templated) in the process, if the code strikes you as familiar it's probably because you spend too much time on my Github page.
## License

This project is licensed under the GPL-3 license.
