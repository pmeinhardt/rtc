# rtc

```txt
      ╭────────────╮ ╔═══════|~|══════╗ ╭────────────╮
      │ stdin      │ ║    signaling   ║ │      stdin │
      ▼            │ ▽                ▽ │            ▼
  client-1        rtc ◁══════════════▷ rtc       client-2
      │            ▲                    ▲            │
      │ stdout+err │   RTCDataChannel   │ stdout+err │
      ╰────────────╯                    ╰────────────╯
```

```shell
rtc init [command [arg ...]]
rtc join [command [arg ...]]
rtc web

stdin (0) -> rtc -> stdout (1)
                 -> stderr (2)

signal send < sdp
signal recv > sdp
```

```shell
rtc init nchat
rtc join nchat
```

Ideas:

- [ ] Raw version with session info (encoded as text) exchanged over separate channel (SMS, Signal, Slack, email, …)
- [ ] Adapter for exchanging session info based on GitHub user names (encrypt SDP via SSH public key + signing)
- [ ] Other SDP exchange channels via add-on tools
- [ ] Web-client interface (static HTML + JS) for non-technical users (send and or receive)
- [ ] Demo apps?
- [ ] Chat
- [ ] Game

## References

- https://developer.mozilla.org/en-US/docs/Web/API/WebRTC_API/Signaling_and_video_calling
- https://en.wikipedia.org/wiki/WebRTC
- https://en.wikipedia.org/wiki/Stream_Control_Transmission_Protocol
- https://en.wikipedia.org/wiki/Datagram_Transport_Layer_Security
- https://github.com/pion
