# rtc

*Note: This is still in a very early stage ⚠️*

```txt
  ╭────────────╮ ╔═══════|~|══════╗ ╭────────────╮
  │ stdin      │ ║    signaling   ║ │      stdin │
  ▼            │ ▽                ▽ │            ▼
app-1         rtc ◁══════════════▷ rtc         app-2
  │            ▲                    ▲            │
  │ stdout     │   RTCDataChannel   │     stdout │
  ╰────────────╯                    ╰────────────╯
```

TODO: What about app stderr?

## Usage

```shell
rtc init [flags] COMMAND [ARG ...]
rtc join [flags] COMMAND [ARG ...]
rtc web [flags]
```

## Apps

### Chat

Build:

```shell
(cd apps/chat && go build .)
```

Use:

```shell
rtc init ./apps/chat/chat # initiate connection
rtc join ./apps/chat/chat # join on the other side
```

## Signaling plugins

For exchanging SDP offer and answer through different channels…

Commands should support `send` and `recv` subcommands.
Commands should block until done.

```shell
signal send < sdp # send SDP received on stdin
signal recv > sdp # receive SDP and print to stdout
```

Example usage with `rtc` command:

```shell
rtc init -s ./signaling-plugins/apple-mail -s somebody@somewhere ./apps/chat/chat

# on the other end
rtc join -s ./signaling-plugins/apple-mail -s sender@elsewhere ./apps/chat/chat
```

## Ideas

- [ ] Adapter for exchanging session info based on GitHub user names (encrypt SDP via SSH public key + signing)
- [ ] Keybase looks like it would be a great fit: Holds pub keys and encrypts messages for specific user, identified by name and with proof of GitHub, Twitter… accounts
- [ ] Web-client interface (static HTML + JS) for non-technical users (send and or receive)

## References

- https://developer.mozilla.org/en-US/docs/Web/API/WebRTC_API/Signaling_and_video_calling
- https://en.wikipedia.org/wiki/WebRTC
- https://en.wikipedia.org/wiki/Stream_Control_Transmission_Protocol
- https://en.wikipedia.org/wiki/Datagram_Transport_Layer_Security
- https://github.com/pion
