export type Props = {
  desc: RTCSessionDescription;
};

function Answer({ desc }: Props) {
  const share = () => {}; // TODO

  return (
    <div className="flex flex-col max-w-xl mb-12 drop-shadow-2xl bg-white rounded-2xl">
      <div className="flex gap-6 items-center justify-between border-b border-neutral-100 p-4">
        <p className="text-sm font-light">
          Share the session description and send it to your peer.
        </p>
        <button
          type="button"
          className="flex items-center justify-center font-sm font-semibold bg-blue-100 disabled:bg-neutral-50 enabled:hover:bg-blue-200 enabled:active:bg-blue-600 text-blue-700 disabled:text-neutral-300 enabled:active:text-white border-0 rounded-full px-4 py-2"
          onClick={share}
        >
          Share
        </button>
      </div>
      <div>
        <textarea
          className="w-full h-48 text-slate-600 resize-none outline outline-0 rounded-b-2xl p-4"
          placeholder="..."
          value={desc ? JSON.stringify(desc) : ""}
          readOnly
        />
      </div>
    </div>
  );
}

export default Answer;
