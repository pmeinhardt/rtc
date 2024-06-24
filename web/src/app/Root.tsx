import App from "./App";
import Wrapper from "./Wrapper";

export type Props = Record<string, never>;

function Root(/* _: Props */) {
  return (
    <Wrapper>
      <App />
    </Wrapper>
  );
}

export default Root;
