import React, { useState } from "react";
import Box from "@mui/material/Box";
import Checkbox from "@mui/material/Checkbox";
import FormControlLabel from "@mui/material/FormControlLabel";
import Typography from "@mui/material/Typography";
import PropTypes from "prop-types";
import { Controller, useForm } from "react-hook-form";
import { FormattedMessage, useIntl } from "react-intl";
import Button from "components/Button";
import ButtonGroup from "components/ButtonGroup";
import ButtonLoader from "components/ButtonLoader";
import FormButtonsWrapper from "components/FormButtonsWrapper";
import PoolLabel from "components/PoolLabel";
import PoolTypeIcon from "components/PoolTypeIcon";
import Selector from "components/Selector";
import SelectorLoader from "components/SelectorLoader";
import { EMPTY_UUID } from "utils/constants";

const POOL_ID = "poolId";
const INCLUDE_CHILDREN = "includeChildren";
const WHOLE_ORGANIZATION_ID = 0;
const SPECIFIC_POOL_ID = 1;

const buildPoolSelectorData = (pools) => ({
  items: pools.map(({ id, name, pool_purpose: poolPurpose }) => ({
    name,
    value: id,
    type: poolPurpose
  }))
});

const getChildren = (array, parentId) => {
  const children = array.filter((x) => x.parent_id === parentId);
  return children.length !== 0 ? [...children, ...children.flatMap((child) => getChildren(array, child.id))] : children;
};

const ReapplyRulesetForm = ({ onSubmit, pools, closeSideModal, isFormLoading, isSubmitLoading }) => {
  const {
    handleSubmit,
    control,
    watch,
    formState: { errors }
  } = useForm({
    defaultValues: {
      [POOL_ID]: "",
      [INCLUDE_CHILDREN]: false
    }
  });
  const intl = useIntl();
  const [isSpecificPool, setIsSpecificPool] = useState(false);
  const [poolId, includeChildren] = watch([POOL_ID, INCLUDE_CHILDREN]);
  const submit = (formData) => {
    if (isSpecificPool) {
      onSubmit(formData[POOL_ID] === EMPTY_UUID ? undefined : formData[POOL_ID], formData[INCLUDE_CHILDREN]);
    } else {
      const rootPoolId = pools.find((i) => i.parent_id === null)?.id;
      onSubmit(rootPoolId, true);
    }
  };

  const buttonsGroup = [
    {
      id: WHOLE_ORGANIZATION_ID,
      messageId: "wholeOrganization",
      dataTestId: "btn_apply_rule_type_whole_organization",
      action: () => {
        setIsSpecificPool(false);
      }
    },
    {
      id: SPECIFIC_POOL_ID,
      messageId: "specificPool",
      dataTestId: "btn_apply_rule_type_specific_pool",
      action: () => {
        setIsSpecificPool(true);
      }
    }
  ];

  return (
    <form onSubmit={handleSubmit(submit)} noValidate>
      <Box display="flex" flexDirection="column" mb={2}>
        <Typography data-test-id="reapply_description">
          <FormattedMessage id="reapplyRulesetDescription" />
        </Typography>
        <FormButtonsWrapper withTopMargin alignItems="center">
          <Typography component="span">
            <FormattedMessage id="reapplyRulesetTo" />{" "}
          </Typography>
          <ButtonGroup buttons={buttonsGroup} activeButtonIndex={isSpecificPool ? SPECIFIC_POOL_ID : WHOLE_ORGANIZATION_ID} />
        </FormButtonsWrapper>
        {isFormLoading
          ? isSpecificPool && <SelectorLoader readOnly fullWidth labelId="pools" isRequired />
          : isSpecificPool && (
              <Controller
                name={POOL_ID}
                control={control}
                rules={{
                  required: {
                    value: true,
                    message: intl.formatMessage({ id: "thisFieldIsRequired" })
                  }
                }}
                render={({ field: { onChange, ...rest } }) => (
                  <Selector
                    dataTestId="pools_selector"
                    required
                    menuItemIcon={{
                      component: PoolTypeIcon,
                      getComponentProps: ({ type }) => ({
                        type
                      })
                    }}
                    error={!!errors[POOL_ID]}
                    helperText={errors?.[POOL_ID]?.message}
                    data={buildPoolSelectorData(pools)}
                    labelId="pools"
                    onChange={(id) => onChange(id)}
                    {...rest}
                  />
                )}
              />
            )}
        {poolId !== EMPTY_UUID && isSpecificPool && (
          <FormControlLabel
            control={
              <Controller
                name={INCLUDE_CHILDREN}
                control={control}
                render={({ field: { value, onChange, ...rest } }) => (
                  <Checkbox
                    data-test-id="with_sub_pools_checkbox"
                    checked={value}
                    {...rest}
                    onChange={(event) => onChange(event.target.checked)}
                  />
                )}
              />
            }
            label={
              <Typography>
                <FormattedMessage id="withSubPools" />
              </Typography>
            }
          />
        )}
        {includeChildren && isSpecificPool && (
          <Box display="flex" flexWrap="wrap">
            {getChildren(pools, poolId).map((pool, index) => (
              <Box mr={1} key={pool.id}>
                <PoolLabel
                  iconProps={{ dataTestId: `pool_icon_${index}` }}
                  dataTestId={`pool_name_${index}`}
                  disableLink
                  id={pool.id}
                  name={pool.name}
                  type={pool.pool_purpose}
                />
              </Box>
            ))}
          </Box>
        )}
        <FormButtonsWrapper>
          <ButtonLoader
            dataTestId="run_btn"
            messageId="run"
            color="primary"
            variant="contained"
            isLoading={isSubmitLoading}
            type="submit"
          />
          <Button dataTestId="cancel_btn" messageId="cancel" variant="outlined" onClick={closeSideModal} />
        </FormButtonsWrapper>
      </Box>
    </form>
  );
};

ReapplyRulesetForm.propTypes = {
  onSubmit: PropTypes.func.isRequired,
  pools: PropTypes.array.isRequired,
  closeSideModal: PropTypes.func.isRequired,
  isFormLoading: PropTypes.bool,
  isSubmitLoading: PropTypes.bool
};

export default ReapplyRulesetForm;
