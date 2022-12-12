import React from "react";
import ReactDOM from "react-dom";
import TestProvider from "tests/TestProvider";
import Circle from "./Circle";

it("renders without crashing", () => {
  const div = document.createElement("div");
  ReactDOM.render(
    <TestProvider>
      <Circle />
    </TestProvider>,
    div
  );
  ReactDOM.unmountComponentAtNode(div);
});
