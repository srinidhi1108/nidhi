import React from "react";
import EditOutlinedIcon from "@mui/icons-material/EditOutlined";
import Box from "@mui/material/Box";
import Skeleton from "@mui/material/Skeleton";
import Typography from "@mui/material/Typography";
import PropTypes from "prop-types";
import CostModelFormattedMoney from "components/CostModelFormattedMoney";
import IconButton from "components/IconButton";
import KeyValueLabel from "components/KeyValueLabel";
import { EnvironmentCostModelModal } from "components/SideModalManager/SideModals";
import { useIsAllowed } from "hooks/useAllowedActions";
import { useOpenSideModal } from "hooks/useOpenSideModal";
import { SCOPE_TYPES } from "utils/constants";

const EnvironmentCostModel = ({ resourceId, hourlyPrice, isLoadingProps = {} }) => {
  const isManageResourcesAllowed = useIsAllowed({
    entityType: SCOPE_TYPES.RESOURCE,
    entityId: resourceId,
    requiredActions: ["MANAGE_RESOURCES"]
  });

  const openSideModal = useOpenSideModal();

  const Body = () => (
    <Box display="flex" alignItems="center">
      <Typography>
        <KeyValueLabel messageId="hourlyPrice" value={<CostModelFormattedMoney value={hourlyPrice} />} />
      </Typography>
      {isManageResourcesAllowed && (
        <IconButton
          key="edit"
          icon={<EditOutlinedIcon />}
          onClick={() => openSideModal(EnvironmentCostModelModal, { resourceId, hourlyPrice })}
          tooltip={{
            show: true,
            messageId: "edit"
          }}
        />
      )}
    </Box>
  );

  const { isGetResourceCostModelLoading = false, isGetPermissionsLoading = false } = isLoadingProps;

  return isGetResourceCostModelLoading || isGetPermissionsLoading ? (
    <Skeleton>
      <Body />
    </Skeleton>
  ) : (
    <Body />
  );
};

EnvironmentCostModel.propTypes = {
  resourceId: PropTypes.string.isRequired,
  hourlyPrice: PropTypes.number.isRequired,
  isLoadingProps: PropTypes.shape({
    isGetResourceCostModelLoading: PropTypes.bool,
    isGetPermissionsLoading: PropTypes.bool
  })
};

export default EnvironmentCostModel;
