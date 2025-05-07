import type { ReactNode } from "react";
import { Component } from "react";

function ErrorMessage({ error }: { error: unknown }) {
  const message =
    error && typeof error === "object" && "message" in error
      ? (error.message as string)
      : `${error as any}`; // eslint-disable-line

  return (
    <div className="bg-rose-600 text-white text-sm font-mono leading-tight p-4 rounded-lg">
      <h2 className="font-bold mb-2">An error occurred:</h2>
      <p>{message}</p>
    </div>
  );
}

export type Props = { children: ReactNode };
export type State = { error: unknown | null };

class ErrorBoundary extends Component<Props, State> {
  constructor(props: Props) {
    super(props);
    this.state = { error: null };
  }

  static getDerivedStateFromError(error: unknown) {
    return { error };
  }

  componentDidCatch(error: unknown, info: object) {
    console.error(error, info); // eslint-disable-line no-console
  }

  render() {
    const { error } = this.state;

    if (error) {
      return <ErrorMessage error={error} />;
    }

    const { children } = this.props;

    return children;
  }
}

export default ErrorBoundary;
