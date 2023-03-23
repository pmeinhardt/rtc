import { useCallback, useEffect, useMemo, useState } from "react";

import Answer from "./Answer";
import Connected from "./Connected";
import Start from "./Start";
import type { Message } from "./types";

const config = {
  iceServers: [{ urls: "stun:stun.l.google.com:19302" }],
};

export type Props = Record<string, never>;

function App(/* _: Props */) {
  const pc = useMemo(() => new RTCPeerConnection(config), []);
  const [dc, use] = useState<RTCDataChannel | null>(null);

  const [offer, setOffer] = useState<RTCSessionDescription | null>(null);
  const [answer, setAnswer] = useState<RTCSessionDescription | null>(null);

  const [messages, setMessages] = useState<Message[]>([]);
  const send = useCallback(
    (text: string) => {
      if (dc === null) return;

      dc.send(text);

      const msg = { local: true, text };
      setMessages((messages) => [...messages, msg]);
    },
    [dc]
  );

  useEffect(() => {
    const onconnectionstatechange = () => {
      console.debug("connection state change:", pc.connectionState); // eslint-disable-line no-console
    };

    const onicegatheringstatechange = (event: Event) => {
      console.debug("ice gathering state change:", event); // eslint-disable-line no-console

      if (pc.iceGatheringState === "complete") {
        setAnswer(pc.localDescription);
      }
    };

    const onsignalingstatechange = (event: Event) => {
      console.debug("signaling state change:", event); // eslint-disable-line no-console
    };

    const ondatachannel = ({ channel }: RTCDataChannelEvent) => {
      console.debug("data channel:", channel); // eslint-disable-line no-console
      if (channel.label === "data") use(channel);
    };

    pc.addEventListener("connectionstatechange", onconnectionstatechange);
    pc.addEventListener("icegatheringstatechange", onicegatheringstatechange);
    pc.addEventListener("signalingstatechange", onsignalingstatechange);
    pc.addEventListener("datachannel", ondatachannel);

    return () => {
      pc.removeEventListener("connectionstatechange", onconnectionstatechange);
      pc.removeEventListener(
        "icegatheringstatechange",
        onicegatheringstatechange
      );
      pc.removeEventListener("signalingstatechange", onsignalingstatechange);
      pc.removeEventListener("datachannel", ondatachannel);
    };
  }, [pc]);

  useEffect(() => {
    if (offer === null) return;

    (async () => {
      await pc.setRemoteDescription(offer);
      const prelim = await pc.createAnswer();
      await pc.setLocalDescription(prelim);
    })();
  }, [pc, offer]);

  useEffect(() => {
    if (dc === null) return () => undefined;

    const onmessage = async ({ data }: MessageEvent<Blob>) => {
      const text = await data.text();
      const msg = { local: false, text };
      setMessages((messages) => [...messages, msg]);
    };

    dc.addEventListener("message", onmessage);

    return () => {
      dc.removeEventListener("message", onmessage);
    };
  }, [dc]);

  const state = dc ? "connected" : answer ? "connecting" : "new";

  return (
    <>
      {state === "new" && <Start onSubmit={setOffer} />}
      {/* eslint-disable-next-line @typescript-eslint/no-non-null-assertion */}
      {state === "connecting" && <Answer desc={answer!} />}
      {state === "connected" && <Connected messages={messages} send={send} />}
    </>
  );
}

export default App;
