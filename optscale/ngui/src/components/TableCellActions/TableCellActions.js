import React from "react";
import PropTypes from "prop-types";
import { FormattedMessage } from "react-intl";
import Dropdown from "components/Dropdown";
import IconButton from "components/IconButton";
import { useAllowedItems } from "hooks/useAllowedActions";
import { useApiState } from "hooks/useApiState";
import { useIsUpMediaQuery } from "hooks/useMediaQueries";
import { isEmpty } from "utils/arrays";
import { SCOPE_TYPES } from "utils/constants";

const renderActions = (items, allowedItems, isGetAllowedActionsLoading) => {
  const targetItems = isGetAllowedActionsLoading ? items : allowedItems;
  return targetItems.map((item) => (
    <IconButton
      key={item.key}
      color={item.color}
      dataTestId={item.dataTestId}
      icon={item.icon}
      onClick={item.action}
      disabled={item.disabled}
      isLoading={(isGetAllowedActionsLoading && !isEmpty(item.requiredActions)) || item.isLoading}
      tooltip={{
        show: true,
        value: <FormattedMessage id={item.messageId} />
      }}
    />
  ));
};

const renderDropdown = (allowedItems) => <Dropdown popupId="mobilePopup" items={allowedItems} trigger="iconButton" isMobile />;

const TableCellActions = ({ items, entityId, entityType }) => {
  const isUpMd = useIsUpMediaQuery("md");
  const allowedItems = useAllowedItems({ entityId, entityType, items });

  const allowedActionsLabel = Object.values(SCOPE_TYPES).includes(entityType)
    ? `GET_${entityType.toUpperCase()}_ALLOWED_ACTIONS`
    : undefined;

  const { isLoading: isGetAllowedActionsLoading } = useApiState(allowedActionsLabel);

  return (
    <div
      style={{
        display: "flex"
      }}
    >
      {isUpMd ? renderActions(items, allowedItems, isGetAllowedActionsLoading) : renderDropdown(allowedItems)}
    </div>
  );
};

TableCellActions.propTypes = {
  items: PropTypes.arrayOf(PropTypes.object).isRequired,
  entityId: PropTypes.string,
  entityType: PropTypes.string
};

export default TableCellActions;
