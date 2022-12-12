import React from "react";
import ReactDOM from "react-dom";
import { EXPENSES_BREAKDOWN_PERIOD_TYPE } from "components/ExpensesBreakdown/BreakdownByPeriodWidget/reducer";
import TestProvider from "tests/TestProvider";
import { EXPENSES_BREAKDOWN_PERIODS } from "utils/constants";
import { millisecondsToSeconds } from "utils/datetime";
import CostExplorer from "./CostExplorer";

const firstDateRangePoint = millisecondsToSeconds(+new Date());
const lastDateRangePoint = millisecondsToSeconds(+new Date());

it("renders without crashing", () => {
  const div = document.createElement("div");
  ReactDOM.render(
    <TestProvider
      state={{
        rangeDates: {},
        [EXPENSES_BREAKDOWN_PERIOD_TYPE]: EXPENSES_BREAKDOWN_PERIODS.DAILY
      }}
    >
      <CostExplorer
        total={1000}
        previousTotal={500}
        breakdown={{}}
        organizationName="org"
        isLoading={false}
        onApply={jest.fn}
        startDateTimestamp={firstDateRangePoint}
        endDateTimestamp={lastDateRangePoint}
      />
    </TestProvider>,
    div
  );
  ReactDOM.unmountComponentAtNode(div);
});
