import type { ChangeEvent } from "react";
import { useCallback, useMemo, useState } from "react";

function isValidOffer(value: unknown): value is RTCSessionDescription {
  if (typeof value !== "object" || value === null) return false;
  if (!("sdp" in value) || !("type" in value)) return false;
  return typeof value.sdp === "string" && value.type === "offer";
}

export type Props = {
  onSubmit: (sd: RTCSessionDescription) => void;
};

function Start({ onSubmit }: Props) {
  const [value, setValue] = useState("");

  const description = useMemo(() => {
    try {
      const parsed = JSON.parse(value) as unknown;
      if (isValidOffer(parsed)) return parsed;
      return null;
    } catch {
      return null;
    }
  }, [value]);

  const submit = useCallback(
    () => description && onSubmit(description),
    [description, onSubmit]
  );

  const update = useCallback(
    (event: ChangeEvent<HTMLTextAreaElement>) => setValue(event.target.value),
    [setValue]
  );

  return (
    <div className="flex flex-col max-w-xl mb-12 drop-shadow-2xl bg-white rounded-2xl">
      <div className="flex gap-6 items-center justify-between border-b border-neutral-100 p-4">
        <p className="text-sm font-light">
          Paste the session description sent by your peer.
        </p>
        <button
          type="button"
          className="flex items-center justify-center font-sm font-semibold bg-blue-100 disabled:bg-neutral-50 enabled:hover:bg-blue-200 enabled:active:bg-blue-600 text-blue-700 disabled:text-neutral-300 enabled:active:text-white border-0 rounded-full px-4 py-2"
          onClick={submit}
          disabled={description === null}
        >
          Join…
        </button>
      </div>
      <div>
        <textarea
          className="w-full h-48 text-slate-600 resize-none outline outline-0 rounded-b-2xl p-4"
          placeholder="..."
          value={value}
          onChange={update}
        />
      </div>
    </div>
  );
}

export default Start;
