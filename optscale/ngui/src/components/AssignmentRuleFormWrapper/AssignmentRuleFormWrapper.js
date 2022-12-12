import React from "react";
import PropTypes from "prop-types";
import { FormattedMessage } from "react-intl";
import ActionBar from "components/ActionBar";
import AssignmentRuleForm from "components/AssignmentRuleForm";
import PageContentWrapper from "components/PageContentWrapper";
import WrapperCard from "components/WrapperCard";
import { isEmpty } from "utils/arrays";

const getCreateActionBarConfig = (poolId, pools) => ({
  text: !poolId ? (
    <FormattedMessage id="addAssignmentRuleTitle" />
  ) : (
    <FormattedMessage
      id="addAssignmentRuleToTitle"
      values={{ poolName: isEmpty(pools) ? "..." : pools.find(({ id }) => id === poolId).name }}
    />
  ),
  dataTestId: "lbl_add_rule"
});

const getEditActionBarConfig = (assignmentRuleName) => ({
  text: <FormattedMessage id="edit{}" values={{ value: assignmentRuleName }} />,
  dataTestId: "lbl_edit_rule"
});

const AssignmentRuleFormWrapper = ({
  poolId,
  pools,
  onSubmit,
  onCancel,
  onPoolChange,
  cloudAccounts,
  poolOwners,
  defaultValues,
  readOnlyProps = {},
  isLoadingProps = {},
  assignmentRuleName,
  assignmentRuleId
}) => {
  const { isActionBarLoading = false, ...restIsLoadingProps } = isLoadingProps;

  const isEdit = !!assignmentRuleId;

  const actionBarConfig = isEdit ? getEditActionBarConfig(assignmentRuleName) : getCreateActionBarConfig(poolId, pools);

  const actionBarDefinition = {
    goBack: true,
    title: {
      ...actionBarConfig,
      isLoading: isActionBarLoading
    }
  };

  return (
    <>
      <ActionBar data={actionBarDefinition} />
      <PageContentWrapper>
        <WrapperCard className="halfWidth">
          <AssignmentRuleForm
            onSubmit={onSubmit}
            onCancel={onCancel}
            cloudAccounts={cloudAccounts}
            pools={pools}
            onPoolChange={onPoolChange}
            poolOwners={poolOwners}
            defaultValues={defaultValues}
            readOnlyProps={readOnlyProps}
            isLoadingProps={restIsLoadingProps}
            isEdit={isEdit}
          />
        </WrapperCard>
      </PageContentWrapper>
    </>
  );
};

AssignmentRuleFormWrapper.propTypes = {
  cloudAccounts: PropTypes.array.isRequired,
  pools: PropTypes.array.isRequired,
  onSubmit: PropTypes.func.isRequired,
  onCancel: PropTypes.func.isRequired,
  onPoolChange: PropTypes.func.isRequired,
  poolOwners: PropTypes.array.isRequired,
  defaultValues: PropTypes.object.isRequired,
  assignmentRuleName: PropTypes.string,
  assignmentRuleId: PropTypes.string,
  poolId: PropTypes.string,
  readOnlyProps: PropTypes.object,
  isLoadingProps: PropTypes.shape({
    isActionBarLoading: PropTypes.bool
  })
};

export default AssignmentRuleFormWrapper;
