import React from "react";
import { Stack } from "@mui/material";
import { FormattedMessage, FormattedNumber } from "react-intl";
import FormattedMoney from "components/FormattedMoney";
import RightsizingCpuUsageHeaderCell from "components/RightsizingCpuUsageHeaderCell";
import TextWithDataTestId from "components/TextWithDataTestId";
import { detectedAt, recommendedRightsizingSize, resource, resourceLocation, rightsizingSize } from "utils/columns";
import { FORMATTED_MONEY_TYPES, RECOMMENDATION_RIGHTSIZING_INSTANCES, RIGHTSIZING_INSTANCES_TYPE } from "utils/constants";
import RecommendationFactory from "../RecommendationFactory";

const RightsizingCpuUsageCell = ({ currentUsage, projectedUsage }) => (
  <Stack spacing={1}>
    <div>
      <FormattedMessage
        id="rightsizingCPUUsageProperties"
        values={{
          type: "current",
          average: <FormattedNumber value={currentUsage.cpuUsage / 100} format="percentage2" />,
          max: <FormattedNumber value={currentUsage.cpuPeak / 100} format="percentage2" />,
          q50: <FormattedNumber value={currentUsage.cpuQuantile50 / 100} format="percentage2" />,
          q99: <FormattedNumber value={currentUsage.cpuQuantile99 / 100} format="percentage2" />
        }}
      />
    </div>
    <div>
      <FormattedMessage
        id="rightsizingCPUUsageProperties"
        values={{
          type: "projected",
          average: <FormattedNumber value={projectedUsage.cpuUsage / 100} format="percentage2" />,
          max: <FormattedNumber value={projectedUsage.cpuPeak / 100} format="percentage2" />,
          q50: <FormattedNumber value={projectedUsage.cpuQuantile50 / 100} format="percentage2" />,
          q99: <FormattedNumber value={projectedUsage.cpuQuantile99 / 100} format="percentage2" />
        }}
      />
    </div>
  </Stack>
);

const RightsizingPossibleSavingCell = ({ possibleSaving, possibleSavingPercent }) => {
  const formattedPossibleSaving = <FormattedMoney value={possibleSaving} type={FORMATTED_MONEY_TYPES.COMMON} />;
  const formattedPossibleSavingPercent = <FormattedNumber value={possibleSavingPercent / 100} format="percentage" />;

  return (
    <>
      {formattedPossibleSaving} ({formattedPossibleSavingPercent})
    </>
  );
};

class RightsizingInstancesRecommendation extends RecommendationFactory {
  type = RECOMMENDATION_RIGHTSIZING_INSTANCES;

  moduleName = RIGHTSIZING_INSTANCES_TYPE;

  withExclusions = true;

  withRightsizingStrategy = true;

  messageId = "rightsizingInstancesRecommendationTitle";

  configure() {
    return {
      type: this.type,
      moduleName: this.moduleName,
      withExclusions: this.withExclusions,
      withRightsizingStrategy: this.withRightsizingStrategy,
      descriptionMessageId: "rightsizingInstancesRecommendationDescription",
      emptyMessageId: "noRightsizingInstances",
      dataTestIds: {
        listTestId: "sp_rightsizing_instance",
        textTestId: "p_rightsizing_instance",
        buttonTestIds: ["btn_rightsizing_instance_download"]
      }
    };
  }

  static configureColumns(optimization) {
    return [
      resource({
        headerDataTestId: "lbl_rightsizing_resource",
        accessor: "cloud_resource_id"
      }),
      resourceLocation({
        headerDataTestId: "lbl_rightsizing_instance_cloud"
      }),
      rightsizingSize({
        messageId: "size",
        headerDataTestId: "lbl_rightsizing_instance_size"
      }),
      recommendedRightsizingSize({
        messageId: "recommendedSize",
        headerDataTestId: "lbl_rightsizing_instance_recommended_flavor"
      }),
      {
        Header: <RightsizingCpuUsageHeaderCell options={optimization.options} />,
        id: "cpuUsage",
        disableSortBy: true,
        Cell: ({ row: { original } }) => (
          <RightsizingCpuUsageCell
            currentUsage={{
              cpuUsage: original.cpu_usage,
              cpuPeak: original.cpu_peak,
              cpuQuantile50: original.cpu_quantile_50,
              cpuQuantile99: original.cpu_quantile_99
            }}
            projectedUsage={{
              cpuUsage: original.project_cpu_avg,
              cpuPeak: original.project_cpu_peak,
              cpuQuantile50: original.projected_cpu_qtl_50,
              cpuQuantile99: original.projected_cpu_qtl_99
            }}
          />
        )
      },
      detectedAt({ headerDataTestId: "lbl_rightsizing_instance_detected_at" }),
      {
        Header: (
          <TextWithDataTestId dataTestId="lbl_rightsizing_instance_possible_savings">
            <FormattedMessage id="possibleMonthlySavings" />
          </TextWithDataTestId>
        ),
        accessor: "saving",
        defaultSort: "desc",
        Cell: ({ row: { original } }) => (
          <RightsizingPossibleSavingCell possibleSaving={original.saving} possibleSavingPercent={original.saving_percent} />
        )
      }
    ];
  }
}

export default new RightsizingInstancesRecommendation();
