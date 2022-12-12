import React from "react";
import InfoOutlinedIcon from "@mui/icons-material/InfoOutlined";
import { Grid } from "@mui/material";
import Typography from "@mui/material/Typography";
import PropTypes from "prop-types";
import { FormattedMessage, useIntl } from "react-intl";
import { useDispatch } from "react-redux";
import { getCleanExpenses, REST_API_URL } from "api";
import CleanExpensesTable from "components/CleanExpensesTable";
import CleanExpensesTableGroup from "components/CleanExpensesTableGroup";
import ExpensesDailyBreakdown from "components/ExpensesDailyBreakdown";
import { ExpensesDailyBreakdownByMockup } from "components/ExpensesDailyBreakdownBy";
import Icon from "components/Icon";
import TableLoader from "components/TableLoader";
import ExpensesDailyBreakdownByContainer from "containers/ExpensesDailyBreakdownByContainer";
import { useFetchAndDownload } from "hooks/useFetchAndDownload";
import { useOrganizationInfo } from "hooks/useOrganizationInfo";
import CleanExpensesService, { mapCleanExpensesRequestParamsToApiParams } from "services/CleanExpensesService";
import { getLength, isEmpty } from "utils/arrays";
import { formQueryString } from "utils/network";

const shouldRenderLimitWarning = (limit, expenses) => getLength(expenses) === limit;

const LimitWarning = ({ limit }) => {
  const intl = useIntl();

  return (
    <div
      style={{
        display: "flex",
        marginBottom: "8px"
      }}
    >
      <Icon icon={InfoOutlinedIcon} hasRightMargin color="warning" />
      <Typography>
        <FormattedMessage
          id="rowsLimitWarning"
          values={{
            entities: intl.formatMessage({ id: "resources" }).toLocaleLowerCase(),
            count: limit
          }}
        />{" "}
        <FormattedMessage id="adjustFiltersWarning" />
      </Typography>
    </div>
  );
};

const CleanExpensesBreakdownContainer = ({ requestParams }) => {
  const dispatch = useDispatch();

  const { organizationId } = useOrganizationInfo();
  const { isFileDownloading, fetchAndDownload } = useFetchAndDownload();

  const downloadResources = (format) => {
    // Intentionally keep limit in requestParams
    const params = { ...requestParams, format };

    fetchAndDownload({
      url: `${REST_API_URL}/organizations/${organizationId}/clean_expenses?${formQueryString(
        mapCleanExpensesRequestParamsToApiParams(params)
      )}`,
      fallbackFilename: `resources_list.${format}`
    });
  };

  const { useGet } = CleanExpensesService();
  const { isLoading, data: apiData } = useGet({ params: requestParams });
  const { clean_expenses: expenses = [] } = apiData;

  const startDateTimestamp = Number(requestParams.startDate);
  const endDateTimestamp = Number(requestParams.endDate);

  const renderExpensesBreakdownTable = () => {
    if (isLoading) {
      return <TableLoader columnsCounter={1} showHeader />;
    }
    if (isEmpty(expenses)) {
      return <CleanExpensesTable expenses={expenses} />;
    }

    return (
      <>
        {shouldRenderLimitWarning(requestParams.limit, expenses) && <LimitWarning limit={requestParams.limit} />}
        <CleanExpensesTableGroup
          startDateTimestamp={startDateTimestamp}
          endDateTimestamp={endDateTimestamp}
          expenses={expenses}
          onSideModalClose={() =>
            dispatch(getCleanExpenses(organizationId, mapCleanExpensesRequestParamsToApiParams(requestParams)))
          }
          downloadResources={downloadResources}
          isDownloadingResources={isFileDownloading}
        />
      </>
    );
  };

  return (
    <Grid container spacing={2}>
      <Grid item xs={12}>
        <ExpensesDailyBreakdown
          container={<ExpensesDailyBreakdownByContainer cleanExpensesRequestParams={requestParams} />}
          mockup={<ExpensesDailyBreakdownByMockup startDateTimestamp={startDateTimestamp} />}
        />
      </Grid>
      <Grid item xs={12}>
        {renderExpensesBreakdownTable()}
      </Grid>
    </Grid>
  );
};

CleanExpensesBreakdownContainer.propTypes = {
  requestParams: PropTypes.object
};

export default CleanExpensesBreakdownContainer;
