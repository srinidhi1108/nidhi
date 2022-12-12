import React, { useMemo } from "react";
import Link from "@mui/material/Link";
import { alpha, useTheme } from "@mui/material/styles";
import PropTypes from "prop-types";
import { Link as RouterLink } from "react-router-dom";
import FormattedMoney from "components/FormattedMoney";
import KeyValueLabel from "components/KeyValueLabel";
import OrganizationLabel from "components/OrganizationLabel";
import PoolLabel from "components/PoolLabel";
import Table from "components/Table";
import TableLoader from "components/TableLoader";
import { intl } from "translations/react-intl-config";
import { RECOMMENDATIONS } from "urls";
import { FORMATTED_MONEY_TYPES } from "utils/constants";

const ALPHA = 0.08;

const getExceedingLabel = (limitName, pool, rowData) => {
  if (pool.organization_id === rowData.id) {
    const { id, [limitName]: limit = 0, name, purpose: type } = pool;
    return (
      <KeyValueLabel
        key={id}
        text={<PoolLabel id={id} type={type} name={name} organizationId={rowData.id} />}
        value={<FormattedMoney format={rowData.currency} type={FORMATTED_MONEY_TYPES.COMMON} value={limit} />}
      />
    );
  }
  return null;
};

const getExceedingLimits = (categoryName, sourceData) => sourceData[categoryName];

const OrganizationsOverviewTable = ({ data, total = data.length, isLoading = false }) => {
  const theme = useTheme();

  const tableData = useMemo(() => data, [data]);

  const columns = useMemo(
    () => [
      {
        Header: <span data-test-id="lbl_name">{intl.formatMessage({ id: "name" })}</span>,
        accessor: "name",
        Cell: ({
          row: {
            original: { name, id: organizationId }
          }
        }) => <OrganizationLabel name={name} id={organizationId} dataTestId={`link_org_${organizationId}`} />
      },
      {
        Header: <span data-test-id="lbl_pool">{intl.formatMessage({ id: "limit" })}</span>,
        accessor: "limit",
        Cell: ({
          row: {
            original: { currency }
          },
          cell: { value }
        }) => <FormattedMoney format={currency} type={FORMATTED_MONEY_TYPES.COMMON} value={value || 0} />
      },
      {
        Header: <span data-test-id="lbl_expenses">{intl.formatMessage({ id: "expenses" })}</span>,
        accessor: "cost",
        Cell: ({
          row: {
            original: { last_month_cost: lastMonthCost = 0, cost = 0, forecast = 0, currency }
          }
        }) => (
          <>
            <KeyValueLabel
              messageId="lastMonth"
              value={<FormattedMoney format={currency} type={FORMATTED_MONEY_TYPES.COMMON} value={lastMonthCost} />}
            />
            <KeyValueLabel
              messageId="thisMonth"
              value={<FormattedMoney format={currency} type={FORMATTED_MONEY_TYPES.COMMON} value={cost} />}
            />
            <KeyValueLabel
              messageId="thisMonthForecast"
              value={<FormattedMoney format={currency} type={FORMATTED_MONEY_TYPES.COMMON} value={forecast} />}
            />
          </>
        ),
        defaultSort: "desc"
      },
      {
        Header: <span data-test-id="lbl_savings">{intl.formatMessage({ id: "possibleMonthlySavings" })}</span>,
        accessor: "saving",
        Cell: ({
          row: {
            original: { saving, id: organizationId, currency }
          }
        }) =>
          saving ? (
            <Link to={`${RECOMMENDATIONS}?organizationId=${organizationId}`} component={RouterLink}>
              <FormattedMoney format={currency} value={saving} type={FORMATTED_MONEY_TYPES.COMMON} />
            </Link>
          ) : null
      },
      {
        Header: <span data-test-id="lbl_pools_exceeding_limits">{intl.formatMessage({ id: "poolsExceedingLimit" })}</span>,
        accessor: "exceededPools",
        disableSortBy: true,
        Cell: ({ row: { original } }) =>
          getExceedingLimits("exceededPools", original).map((pool) => getExceedingLabel("cost", pool, original))
      },
      {
        Header: (
          <span data-test-id="lbl_forecasts_exceeding_limits">{intl.formatMessage({ id: "forecastsExceedingLimit" })}</span>
        ),
        disableSortBy: true,
        accessor: "exceededForecasts",
        Cell: ({ row: { original } }) =>
          getExceedingLimits("exceededForecasts", original).map((pool) => getExceedingLabel("forecast", pool, original))
      }
    ],
    []
  );

  return isLoading ? (
    <TableLoader columnsCounter={columns.length} showHeader />
  ) : (
    <Table
      data={tableData}
      columns={columns}
      localization={{
        emptyMessageId: "noOrganizations"
      }}
      pageSize={50}
      totalNumberOverride={total}
      counters={{ showCounters: true, hideTotal: false }}
      getConditionalRowStyle={(rowData) => ({
        backgroundColor: rowData.exceededOrganizationIds.has(rowData.id)
          ? alpha(theme.palette.error.main, ALPHA)
          : alpha(theme.palette.success.main, ALPHA)
      })}
      dataTestIds={{
        infoArea: {
          total: "counter_total",
          displayed: "counter_displayed"
        }
      }}
    />
  );
};

OrganizationsOverviewTable.propTypes = {
  data: PropTypes.array.isRequired,
  total: PropTypes.number,
  isLoading: PropTypes.bool
};

export default OrganizationsOverviewTable;
