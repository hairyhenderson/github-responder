# github-responder

[![Build Status][circleci-image]][circleci-url]
[![hairyhenderson/github-responder on DockerHub][dockerhub-image]][dockerhub-url]

A library & CLI tool that automatically sets up GitHub WebHooks and listens for events, with automatic TLS.

For example, if you want to run a command every time someone stars your repo (the `watch` event):

```console
$ github-responder --repo my/repo --domain hydrogen.hairyhenderson.ca -e watch ./ring-the-bell.sh
...
```

This will do a bunch of things:
1. Register a new Webhook at the named repo (`--repo`)
2. Start a web server to serve webhook events
3. Run the command `ring-the-bell.sh` every time a `watch` event is received

A few more details:
- github-responder is reasonably secure:
  - the webhook server is automatically protected by TLS, configured with a free automatically-renewing certificate from [Let's Encrypt][]
  - the webhook listens at a randomly-generated URL - all other traffic is rejected
  - incoming events must be signed by a randomly-generated secret key - every event is verified
- the command is provided with all event details:
  - the event type is provided as the first flag on the command line
  - the unique delivery ID is provided as the second flag on the command line (this can be used to de-duplicate events, which may be re-delivered in some cases)
  - the event payload is sent to the command as standard input (in JSON format)
- logs are output as structured JSON, or in a slightly easier-to-read format when run in an interactive terminal
- github-responder can be used as a library in other Go programs


## License

[The MIT License](http://opensource.org/licenses/MIT)

Copyright (c) 2018 Dave Henderson

[circleci-image]: https://circleci.com/gh/hairyhenderson/github-responder/tree/master.svg?style=shield
[circleci-url]: https://circleci.com/gh/hairyhenderson/github-responder/tree/master
[dockerhub-image]: https://img.shields.io/badge/docker-ready-blue.svg
[dockerhub-url]: https://hub.docker.com/r/hairyhenderson/github-responder

[Let's Encrypt]: http://letsencrypt.org
