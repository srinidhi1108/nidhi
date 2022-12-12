import React from "react";
import { useTheme } from "@mui/material";
import Typography from "@mui/material/Typography";
import PropTypes from "prop-types";
import { useIntl } from "react-intl";
import BreakdownLabel from "components/BreakdownLabel";
import CircleLabel from "components/CircleLabel";
import HighlightsLayer from "components/HighlightsLayer";
import KeyValueLabel from "components/KeyValueLabel";
import LineChart from "components/LineChart";
import { useIsOrganizationWeekend } from "hooks/useIsOrganizationWeekend";
import { useShowWeekends } from "hooks/useShowWeekends";
import { getLength } from "utils/arrays";

const TYPOGRAPHY_PROPS = {
  gutterBottom: true
};

const ChartTooltip = ({ points: allPoints, isOrganizationWeekend, breakdownBy }) => {
  const intl = useIntl();
  const theme = useTheme();

  const stringDate = allPoints[0]?.data?.x;
  const totalCount = allPoints[0]?.data?.yStacked;

  if (stringDate === undefined || totalCount === undefined) {
    return null;
  }

  const getTitleText = () => {
    const total = intl.formatMessage({ id: "total{date}" }, { date: stringDate });
    const weekend = isOrganizationWeekend(new Date(stringDate))
      ? `(${intl.formatMessage({ id: "weekend" }).toLowerCase()})`
      : "";

    return [total, weekend].filter(Boolean).join(" ");
  };

  const titleText = getTitleText();

  const renderTotalLabel = () =>
    getLength(allPoints) > 1 ? (
      <KeyValueLabel text={titleText} value={totalCount} typographyProps={TYPOGRAPHY_PROPS} />
    ) : (
      <Typography {...TYPOGRAPHY_PROPS}>{titleText}</Typography>
    );

  return (
    <>
      {renderTotalLabel()}
      {allPoints.map(({ id: pointId, serieColor, data: pointData = {} }) => (
        <KeyValueLabel
          key={pointId}
          typographyProps={{
            gutterBottom: true
          }}
          renderKey={() => (
            <CircleLabel
              figureColor={serieColor}
              label={pointData.translatedSerieId ?? <BreakdownLabel breakdownBy={breakdownBy} details={pointData.details} />}
              textFirst={false}
            />
          )}
          value={
            <>
              <span
                style={{
                  marginRight: theme.spacing(0.5)
                }}
              >
                {pointData.y}
              </span>
              {pointData.details.created !== 0 && (
                <span
                  style={{
                    color: theme.palette.success.main,
                    marginRight: pointData.details.deletedDayBefore !== 0 ? theme.spacing(0.5) : 0
                  }}
                >
                  +{pointData.details.created}
                </span>
              )}
              {pointData.details.deletedDayBefore !== 0 && (
                <span style={{ color: theme.palette.error.main }}>-{pointData.details.deletedDayBefore}</span>
              )}
            </>
          }
        />
      ))}
    </>
  );
};

const ResourceCountBreakdownLineChart = ({ data, colors, isLoading, style, breakdownBy }) => {
  const isOrganizationWeekend = useIsOrganizationWeekend();

  const { showWeekends } = useShowWeekends();

  return (
    <LineChart
      dataTestId="resource_count_breakdown_chart"
      data={data}
      stacked
      style={style}
      colors={({ id }) => colors[id]}
      isLoading={isLoading}
      renderTooltipBody={({ slice = {} }) => {
        const { points: allPoints = [] } = slice;
        return <ChartTooltip points={allPoints} isOrganizationWeekend={isOrganizationWeekend} breakdownBy={breakdownBy} />;
      }}
      highlightsLayer={
        showWeekends &&
        ((chartOptions) => (
          <HighlightsLayer
            chartOptions={chartOptions}
            shouldHighlight={(x) => {
              const date = new Date(x);
              return isOrganizationWeekend(date);
            }}
          />
        ))
      }
    />
  );
};

ChartTooltip.propTypes = {
  points: PropTypes.array.isRequired,
  isOrganizationWeekend: PropTypes.func.isRequired,
  breakdownBy: PropTypes.string
};

ResourceCountBreakdownLineChart.propTypes = {
  data: PropTypes.array.isRequired,
  breakdownBy: PropTypes.string,
  emptyMessageId: PropTypes.string,
  isLoading: PropTypes.bool,
  colors: PropTypes.object,
  style: PropTypes.object
};

export default ResourceCountBreakdownLineChart;
