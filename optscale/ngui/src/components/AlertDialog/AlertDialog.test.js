import React from "react";
import Link from "@mui/material/Link";
import ReactDOM from "react-dom";
import { FormattedMessage } from "react-intl";
import MailTo from "components/MailTo";
import TestProvider from "tests/TestProvider";
import { EMAIL_SUPPORT, DOCS_HYSTAX_OPTSCALE } from "urls";
import AlertDialog from "./AlertDialog";

it("renders without crashing", () => {
  const div = document.createElement("div");
  ReactDOM.render(
    <TestProvider>
      <AlertDialog
        show
        message={
          <FormattedMessage
            id="privacyWarning"
            values={{
              email: <MailTo email={EMAIL_SUPPORT} text={EMAIL_SUPPORT} />,
              docs: (chunks) => (
                <Link target="_blank" href={DOCS_HYSTAX_OPTSCALE}>
                  {chunks}
                </Link>
              ),
              p: (chunks) => <p>{chunks}</p>,
              ul: (chunks) => <ul>{chunks}</ul>,
              li: (chunks) => <li>{chunks}</li>,
              strong: (chunks) => <strong>{chunks}</strong>,
              br: <br />
            }}
          />
        }
        buttonMessageId="proceedToOptScale"
      />
    </TestProvider>,
    div
  );
  ReactDOM.unmountComponentAtNode(div);
});
