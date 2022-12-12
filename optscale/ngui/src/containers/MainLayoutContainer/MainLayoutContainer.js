import React, { useEffect } from "react";
import CircularProgress from "@mui/material/CircularProgress";
import PropTypes from "prop-types";
import { useDispatch } from "react-redux";
import {
  getOrganizations,
  getCurrentEmployee,
  getCloudAccounts,
  getOrganizationAllowedActions,
  getOrganizationFeatures,
  getOrganizationThemeSettings
} from "api";
import { GET_ORGANIZATION_ALLOWED_ACTIONS } from "api/auth/actionTypes";
import {
  GET_ORGANIZATIONS,
  GET_CURRENT_EMPLOYEE,
  GET_CLOUD_ACCOUNTS,
  GET_ORGANIZATION_FEATURES,
  GET_ORGANIZATION_THEME_SETTINGS
} from "api/restapi/actionTypes";
import Backdrop from "components/Backdrop";
import { useApiState } from "hooks/useApiState";
import { useInitialMount } from "hooks/useInitialMount";
import { useIsLoading } from "hooks/useIsLoading";
import { useOrganizationInfo } from "hooks/useOrganizationInfo";
import { useShouldRenderLoader } from "hooks/useShouldRenderLoader";

const MainLayoutContainer = ({ children }) => {
  const dispatch = useDispatch();

  const { isInitialMount, setIsInitialMount } = useInitialMount();

  useEffect(() => {
    if (isInitialMount) {
      setIsInitialMount(false);
    }
  }, [isInitialMount, setIsInitialMount]);

  const { organizationId } = useOrganizationInfo();

  const { shouldInvoke: shouldInvokeGetOrganizations } = useApiState(GET_ORGANIZATIONS);
  const { shouldInvoke: shouldInvokeGetAllowedActions } = useApiState(GET_ORGANIZATION_ALLOWED_ACTIONS, organizationId);
  const { shouldInvoke: shouldInvokeGetCurrentEmployee } = useApiState(GET_CURRENT_EMPLOYEE, organizationId);
  const { shouldInvoke: shouldInvokeCloudAccounts } = useApiState(GET_CLOUD_ACCOUNTS, organizationId);
  const { shouldInvoke: shouldInvokeGetOrganizationsFeatures } = useApiState(GET_ORGANIZATION_FEATURES, organizationId);
  const { shouldInvoke: shouldInvokeGetOrganizationThemeSettings } = useApiState(
    GET_ORGANIZATION_THEME_SETTINGS,
    organizationId
  );

  useEffect(() => {
    if (shouldInvokeGetOrganizations) {
      dispatch(getOrganizations());
    }
  }, [shouldInvokeGetOrganizations, dispatch]);

  useEffect(() => {
    if (organizationId && shouldInvokeGetAllowedActions) {
      dispatch(getOrganizationAllowedActions(organizationId));
    }
  }, [dispatch, organizationId, shouldInvokeGetAllowedActions]);

  useEffect(() => {
    if (organizationId && shouldInvokeGetCurrentEmployee) {
      dispatch(getCurrentEmployee(organizationId));
    }
  }, [dispatch, organizationId, shouldInvokeGetCurrentEmployee]);

  useEffect(() => {
    if (organizationId && shouldInvokeCloudAccounts) {
      dispatch(getCloudAccounts(organizationId));
    }
  }, [dispatch, organizationId, shouldInvokeCloudAccounts]);

  useEffect(() => {
    if (organizationId && shouldInvokeGetOrganizationsFeatures) {
      dispatch(getOrganizationFeatures(organizationId));
    }
  }, [dispatch, organizationId, shouldInvokeGetOrganizationsFeatures]);

  useEffect(() => {
    if (organizationId && shouldInvokeGetOrganizationThemeSettings) {
      dispatch(getOrganizationThemeSettings(organizationId));
    }
  }, [dispatch, organizationId, shouldInvokeGetOrganizationThemeSettings]);

  const apiIsLoading = useIsLoading([
    GET_ORGANIZATIONS,
    GET_ORGANIZATION_ALLOWED_ACTIONS,
    GET_CURRENT_EMPLOYEE,
    GET_CLOUD_ACCOUNTS,
    GET_ORGANIZATION_FEATURES
  ]);
  const isLoading = useShouldRenderLoader(isInitialMount, [apiIsLoading]);

  return isLoading ? (
    <Backdrop>
      <CircularProgress />
    </Backdrop>
  ) : (
    children
  );
};

MainLayoutContainer.propTypes = {
  children: PropTypes.node.isRequired
};

export default MainLayoutContainer;
