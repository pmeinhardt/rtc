import type { ChangeEvent, FormEvent } from "react";
import { useCallback, useState } from "react";

import type { Message } from "./types";

export type Props = {
  messages: Message[];
  send: (msg: string) => void;
};

function Connected({ messages, send }: Props) {
  const [text, setText] = useState("");

  const update = useCallback(
    (event: ChangeEvent<HTMLInputElement>) => setText(event.target.value),
    []
  );

  const submit = useCallback(
    (event: FormEvent) => {
      event.preventDefault();
      send(`${text}\n`);
      setText("");
    },
    [send, text]
  );

  return (
    <div className="flex flex-col max-w-xl mb-12 drop-shadow-2xl bg-white rounded-2xl">
      <form
        className="flex gap-6 items-center justify-between border-b border-neutral-100 p-4"
        onSubmit={submit}
      >
        <input
          type="text"
          placeholder="Type here…"
          value={text}
          onChange={update}
          className="flex-grow outline outline-0 text-sm font-light"
        />
        <button
          className="flex items-center justify-center font-sm font-semibold bg-blue-100 disabled:bg-neutral-50 enabled:hover:bg-blue-200 enabled:active:bg-blue-600 text-blue-700 disabled:text-neutral-300 enabled:active:text-white border-0 rounded-full px-4 py-2"
          disabled={text.length === 0}
        >
          Send
        </button>
      </form>
      <div className="text-sm text-slate-500 p-4 font-light">
        <ul>
          {messages.length === 0 && (
            <li key="empty" className="flex justify-center p-6">
              No messages yet.
            </li>
          )}
          {messages.map((msg, i) => (
            <li
              key={i}
              className={`py-2 ${msg.local ? "text-right" : "text-left"}`}
            >
              {msg.text}
            </li>
          ))}
        </ul>
      </div>
    </div>
  );
}

export default Connected;
