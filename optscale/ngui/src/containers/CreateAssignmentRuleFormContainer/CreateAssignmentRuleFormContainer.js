import React, { useEffect, useState } from "react";
import PropTypes from "prop-types";
import { useDispatch } from "react-redux";
import { useNavigate } from "react-router-dom";
import { createAssignmentRule, getPoolOwners, RESTAPI, getAvailablePools } from "api";
import { CREATE_ASSIGNMENT_RULE, GET_CLOUD_ACCOUNTS, GET_POOL_OWNERS, GET_AVAILABLE_POOLS } from "api/restapi/actionTypes";
import AssignmentRuleFormWrapper from "components/AssignmentRuleFormWrapper";
import { useApiData } from "hooks/useApiData";
import { useApiState } from "hooks/useApiState";
import { useOrganizationInfo } from "hooks/useOrganizationInfo";
import { getPoolUrl, ASSIGNMENT_RULES } from "urls";
import { isError } from "utils/api";
import { isEmpty as isEmptyArray } from "utils/arrays";
import {
  DEFAULT_CONDITIONS,
  TAB_QUERY_PARAM_NAME,
  ORGANIZATION_OVERVIEW_TABS,
  CONDITION,
  TAG_CONDITION,
  CONDITION_TYPES,
  TAG_IS,
  ASSIGNMENT_RULE_CONDITIONS_QUERY_PARAMETER
} from "utils/constants";
import { getQueryParams } from "utils/network";
import { parseJSON } from "utils/strings";

const { META_INFO, TYPE } = CONDITION;
const { KEY: TAG_KEY, VALUE: TAG_VALUE } = TAG_CONDITION;

export const getDefaultConditionsFromQueryParams = (conditionsQueryParam) => {
  const conditions = conditionsQueryParam
    .map((condition) => {
      const parsedCondition = parseJSON(condition, undefined);
      if (parsedCondition) {
        const { type, value } = JSON.parse(condition);

        if (!Object.keys(CONDITION_TYPES).includes(type)) {
          return undefined;
        }

        if (type === TAG_IS) {
          return {
            [TYPE]: type,
            [`${META_INFO}_${TAG_KEY}`]: value?.tagKey,
            [`${META_INFO}_${TAG_VALUE}`]: value?.tagValue
          };
        }
        return {
          [TYPE]: type,
          [META_INFO]: value
        };
      }
      return undefined;
    })
    .filter(Boolean);

  return isEmptyArray(conditions) ? DEFAULT_CONDITIONS : conditions;
};

const CreateAssignmentRuleFormContainer = ({ poolId }) => {
  const dispatch = useDispatch();
  const navigate = useNavigate();
  const { organizationId, organizationPoolId } = useOrganizationInfo();

  const [isFormDataLoading, setIsFormDataLoading] = useState(false);

  const { [ASSIGNMENT_RULE_CONDITIONS_QUERY_PARAMETER]: conditionsQueryParam } = getQueryParams(true, {
    arrayFormat: "bracket"
  });

  const [defaultValues, setDefaultValues] = useState({
    name: "",
    active: true,
    conditions: conditionsQueryParam ? getDefaultConditionsFromQueryParams(conditionsQueryParam) : DEFAULT_CONDITIONS,
    poolId: "",
    ownerId: ""
  });

  const redirect = () => {
    if (!poolId) {
      navigate(ASSIGNMENT_RULES);
    } else {
      navigate(`${getPoolUrl(poolId)}?${TAB_QUERY_PARAM_NAME}=${ORGANIZATION_OVERVIEW_TABS.ASSIGNMENT_RULES}`);
    }
  };

  const { isLoading: isCreateAssignmentRuleLoading } = useApiState(CREATE_ASSIGNMENT_RULE);

  // Get available pools
  const {
    apiData: { pools = [] }
  } = useApiData(GET_AVAILABLE_POOLS);

  // Get owners
  const {
    apiData: { poolOwners = [] }
  } = useApiData(GET_POOL_OWNERS);

  useEffect(() => {
    dispatch((_, getState) => {
      setIsFormDataLoading(true);
      dispatch(getAvailablePools(organizationId))
        .then(() => {
          const { pools: availablePools } = getState()?.[RESTAPI]?.[GET_AVAILABLE_POOLS] ?? {};

          // poolId - if creating from a particular pool
          // organizationPoolId - from assignment rules page
          const defaultPoolId = poolId || organizationPoolId;

          const { default_owner_id: defaultOwnerId = "" } =
            availablePools.find((availablePool) => availablePool.id === defaultPoolId) ?? {};

          // There is no need to wait for getPoolOwners to be loaded since the default owner depends only on the pool
          setDefaultValues((currentDefaultValues) => ({
            ...currentDefaultValues,
            poolId: defaultPoolId,
            ownerId: defaultOwnerId
          }));

          return dispatch(getPoolOwners(defaultPoolId));
        })
        .finally(() => setIsFormDataLoading(false));
    });
  }, [poolId, dispatch, organizationPoolId, organizationId]);

  // get cloud accounts
  // Attention: we don't request cloud account here as they are included in the initial loader
  // and we assume that they are up-to-date
  const { apiData: { cloudAccounts = [] } = {} } = useApiData(GET_CLOUD_ACCOUNTS);

  return (
    <AssignmentRuleFormWrapper
      defaultValues={defaultValues}
      pools={pools}
      poolOwners={poolOwners}
      cloudAccounts={cloudAccounts}
      poolId={poolId}
      onPoolChange={(newPoolId, callback) => {
        dispatch((_, getState) => {
          dispatch(getPoolOwners(newPoolId)).then(() => {
            const { poolOwners: owners = [] } = getState()?.[RESTAPI]?.[GET_POOL_OWNERS] ?? {};
            callback(owners);
          });
        });
      }}
      onSubmit={(params) => {
        dispatch((_, getState) => {
          dispatch(createAssignmentRule(organizationId, params)).then(() => {
            if (!isError(CREATE_ASSIGNMENT_RULE, getState())) {
              redirect();
            }
            return undefined;
          });
        });
      }}
      onCancel={redirect}
      readOnlyProps={{
        poolSelector: !!poolId,
        ownerSelector: false
      }}
      isLoadingProps={{
        isActionBarLoading: poolId && isFormDataLoading,
        isActiveCheckboxLoading: false,
        isNameInputLoading: false,
        isConditionsFieldLoading: false,
        isPoolSelectorLoading: isFormDataLoading,
        isOwnerSelectorLoading: isFormDataLoading,
        isSubmitButtonLoading: isFormDataLoading || isCreateAssignmentRuleLoading
      }}
    />
  );
};

CreateAssignmentRuleFormContainer.propTypes = {
  poolId: PropTypes.string
};

export default CreateAssignmentRuleFormContainer;
