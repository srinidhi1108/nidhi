import React, { useCallback, useEffect, useMemo } from "react";
import AddOutlinedIcon from "@mui/icons-material/AddOutlined";
import BarChartOutlinedIcon from "@mui/icons-material/BarChartOutlined";
import EditOutlinedIcon from "@mui/icons-material/EditOutlined";
import IosShareOutlinedIcon from "@mui/icons-material/IosShareOutlined";
import LinkOutlinedIcon from "@mui/icons-material/LinkOutlined";
import StorageOutlinedIcon from "@mui/icons-material/StorageOutlined";
import PropTypes from "prop-types";
import { FormattedMessage } from "react-intl";
import { useDispatch } from "react-redux";
import { useNavigate } from "react-router-dom";
import FormattedMoney from "components/FormattedMoney";
import Icon from "components/Icon";
import PoolLabel from "components/PoolLabel";
import ProgressBar from "components/ProgressBar";
import { ShareSettingsModal } from "components/SideModalManager/SideModals";
import Table from "components/Table";
import Expander from "components/Table/Expander";
import TableCellActions from "components/TableCellActions";
import TableLoader from "components/TableLoader";
import TextWithDataTestId from "components/TextWithDataTestId";
import { useIsDownMediaQuery } from "hooks/useMediaQueries";
import { useOpenSideModal } from "hooks/useOpenSideModal";
import { useRootData } from "hooks/useRootData";
import { getCreatePoolUrl, getEditPoolUrl, getThisMonthPoolExpensesUrl, getThisMonthResourcesByPoolUrl } from "urls";
import { getRecursiveParent } from "utils/arrays";
import { SCOPE_TYPES, FORMATTED_MONEY_TYPES } from "utils/constants";
import { getPoolColorStatus } from "utils/layouts";
import { getPercentageChange, round } from "utils/math";
import { setExpandedRow, setExpandedRows } from "./actionCreators";
import { EXPANDED_POOL_ROWS } from "./reducer";

const isOverspent = (value, pool) => value > pool;

const isExpensesCloseToLimit = (value, pool) => value !== 0 && getPercentageChange(value, pool) < 10;

const getTextByPercent = ({ percent, value, textForLess, textForEqual, textForMore }) => {
  if (percent < 100) {
    return (
      <FormattedMessage
        id={textForLess}
        values={{
          value: Math.trunc(percent)
        }}
      />
    );
  }
  if (percent === 100) {
    return <FormattedMessage id={textForEqual} />;
  }
  return (
    <FormattedMessage
      id={textForMore}
      values={{
        value
      }}
    />
  );
};

const PoolsTable = ({ rootPool, isLoadingProps = {} }) => {
  const isDownLg = useIsDownMediaQuery("lg");

  const dispatch = useDispatch();
  const navigate = useNavigate();

  const { id: rootPoolId, children: rootPoolChildren = [] } = rootPool;

  const openSideModal = useOpenSideModal();

  const openShareSideModal = (selectedPoolId) => {
    const {
      name,
      purpose,
      expenses_export_link: link
    } = [rootPool, ...rootPoolChildren].find(({ id }) => id === selectedPoolId);

    openSideModal(ShareSettingsModal, {
      poolId: selectedPoolId,
      poolName: name,
      poolPurpose: purpose,
      initialLink: link
    });
  };

  useEffect(() => {
    const pools = [rootPool, ...rootPoolChildren];

    const expandArray = pools.reduce((result, pool) => {
      const isOverspentCost = isOverspent(pool.cost, pool.limit);
      const isOverspentForecast = isOverspent(pool.forecast, pool.limit);
      const isExpensesCloseToLimitValue = isExpensesCloseToLimit(pool.forecast, pool.limit);
      if (isOverspentCost || isOverspentForecast || isExpensesCloseToLimitValue) {
        return [...result, ...getRecursiveParent(pool, pools, "id")];
      }
      return result;
    }, []);
    dispatch(setExpandedRows(expandArray));
  }, [rootPool, dispatch, rootPoolChildren]);

  const { rootData = [] } = useRootData(EXPANDED_POOL_ROWS);
  const onExpand = (rowData, isExpanded) => dispatch(setExpandedRow(rowData.id, isExpanded));

  const { isGetPoolLoading = false, isGetPoolAllowedActionsLoading = false } = isLoadingProps;

  const actionBarDefinition = {
    items: [
      {
        key: "add",
        icon: <AddOutlinedIcon fontSize="small" />,
        messageId: "add",
        color: "success",
        variant: "contained",
        type: "button",
        link: getCreatePoolUrl(rootPoolId),
        requiredActions: ["MANAGE_POOLS"],
        isLoading: isGetPoolAllowedActionsLoading,
        dataTestId: "btn_add"
      }
    ],
    poolId: rootPoolId
  };

  const tableActions = [
    {
      key: "edit",
      messageId: "edit",
      icon: <EditOutlinedIcon />,
      action: (rowDataId) => navigate(getEditPoolUrl(rowDataId)),
      dataTestId: "btn_edit",
      requiredActions: ["MANAGE_POOLS"]
    },
    {
      key: "seeInCostExplorer",
      messageId: "seeInCostExplorer",
      icon: <BarChartOutlinedIcon />,
      action: (rowDataId) => navigate(getThisMonthPoolExpensesUrl(rowDataId)),
      dataTestId: "btn_see_in_ce"
    },
    {
      key: "seeResourceList",
      messageId: "seeResourceList",
      icon: <StorageOutlinedIcon />,
      action: (rowDataId) => navigate(getThisMonthResourcesByPoolUrl(rowDataId)),
      dataTestId: "btn_see_rl"
    },
    {
      key: "openShareSettings",
      messageId: "openShareSettings",
      icon: <IosShareOutlinedIcon />,
      action: (rowDataId) => {
        openShareSideModal(rowDataId);
      },
      dataTestId: "btn_share"
    }
  ];

  const columns = [
    {
      Header: <TextWithDataTestId dataTestId="lbl_name" messageId="name" />,
      accessor: "name",
      Cell: ({ row }) => {
        const { original, id: rowId } = row;
        const { purpose: type, id, name, expenses_export_link: expensesExportLink } = original;

        return (
          <div style={{ display: "flex", alignItems: "center" }}>
            <Expander row={row} onExpand={onExpand} />
            <PoolLabel
              disableLink={id === rootPoolId}
              type={type}
              id={id}
              name={name}
              endAdornment={
                !!expensesExportLink && (
                  <span
                    onClick={() => {
                      openShareSideModal(id);
                    }}
                    style={{ cursor: "pointer" }}
                  >
                    <Icon icon={LinkOutlinedIcon} hasLeftMargin tooltip={{ show: true, messageId: "thisPoolIsShared" }} />
                  </span>
                )
              }
              dataTestId={`link_pool_${rowId}`}
            />
          </div>
        );
      },
      style: {
        whiteSpace: "nowrap"
      }
    },
    {
      Header: <TextWithDataTestId dataTestId="lbl_limit" messageId="limit" />,
      accessor: "limit",
      Cell: ({ cell: { value } }) => <FormattedMoney type={FORMATTED_MONEY_TYPES.COMMON} value={value || 0} />
    },
    {
      Header: <TextWithDataTestId dataTestId="lbl_expenses" messageId="expensesThisMonth" />,
      accessor: "cost",
      Cell: ({ row: { original } }) => {
        const cost = original.cost || 0;
        const xDividedByY = cost / original.limit;
        const percent = xDividedByY * 100;
        return (
          <ProgressBar
            width={isDownLg ? "100%" : "50%"}
            tooltip={{
              show: cost !== 0 && original.limit !== 0,
              value: getTextByPercent({
                percent,
                value: round(xDividedByY, 1),
                textForLess: "thisMonthExpensesIsOfPool",
                textForEqual: "thisMonthExpensesEqualPool",
                textForMore: "thisMonthExpensesAreTimesMoreThanPool"
              })
            }}
            color={getPoolColorStatus(percent)}
            value={percent}
          >
            <FormattedMoney type={FORMATTED_MONEY_TYPES.COMMON} value={cost} />
          </ProgressBar>
        );
      },
      defaultSort: "desc"
    },
    {
      Header: <TextWithDataTestId dataTestId="lbl_forecast" messageId="forecastThisMonth" />,
      accessor: "forecast",
      Cell: ({ row: { original } }) => {
        const forecast = original.forecast || 0;
        const xDividedByY = forecast / original.limit;
        const percent = xDividedByY * 100;
        return (
          <ProgressBar
            width={isDownLg ? "100%" : "50%"}
            tooltip={{
              show: forecast !== 0 && original.limit !== 0,
              value: getTextByPercent({
                percent,
                value: round(xDividedByY, 1),
                textForLess: "thisMonthForecastIsOfPool",
                textForEqual: "thisMonthForecastEqualsPool",
                textForMore: "thisMonthForecastIsTimesMoreThanPool"
              })
            }}
            color={getPoolColorStatus(percent)}
            value={percent}
          >
            <FormattedMoney type={FORMATTED_MONEY_TYPES.COMMON} value={forecast} />
          </ProgressBar>
        );
      }
    },
    {
      Header: <TextWithDataTestId dataTestId="lbl_actions" messageId="actions" />,
      id: "actions",
      disableSortBy: true,
      Cell: ({ row }) => (
        <TableCellActions
          entityType={SCOPE_TYPES.POOL}
          entityId={row.original.id}
          items={tableActions.map((item) => ({
            ...item,
            dataTestId: `${item.dataTestId}_${row.id}`,
            action: () => item.action(row.original.id)
          }))}
        />
      )
    }
  ];

  const root = useMemo(() => [rootPool], [rootPool]);

  const getSubRows = useCallback(
    (parentObject) => [rootPool, ...rootPoolChildren].filter((otherObject) => otherObject.parent_id === parentObject.id) || [],
    [rootPool, rootPoolChildren]
  );

  return (
    <>
      {isGetPoolLoading ? (
        <TableLoader columnsCounter={columns.length} showHeader />
      ) : (
        <Table
          data={root}
          columns={columns}
          actionBar={{
            show: true,
            definition: actionBarDefinition
          }}
          withSubRows
          getSubRows={getSubRows}
          expandedByDefault={(rowData) => rootData.includes(rowData.id)}
          localization={{
            emptyMessageId: "noPools"
          }}
        />
      )}
    </>
  );
};

PoolsTable.propTypes = {
  rootPool: PropTypes.object.isRequired,
  isLoadingProps: PropTypes.shape({
    isGetPoolLoading: PropTypes.bool,
    isGetPoolAllowedActionsLoading: PropTypes.bool
  })
};

export default PoolsTable;
