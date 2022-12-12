import React, { forwardRef } from "react";
import PropTypes from "prop-types";
import { useIntl } from "react-intl";
import PoolTypeIcon from "components/PoolTypeIcon";
import Selector from "components/Selector";
import { POOL_TYPES } from "utils/constants";

const buildPurposeSelectorData = (selected, intl) => ({
  selected,
  items: Object.entries(POOL_TYPES).map(([type, messageId]) => ({
    name: intl.formatMessage({ id: messageId }),
    value: type,
    type
  }))
});

const PoolTypeSelector = forwardRef(({ error, helperText, dataTestId, onChange = () => {}, value, ...rest }, ref) => {
  const intl = useIntl();

  return (
    <Selector
      data-test-id={dataTestId}
      error={error}
      helperText={helperText}
      data={buildPurposeSelectorData(value, intl)}
      labelId="type"
      onChange={onChange}
      fullWidth
      menuItemIcon={{
        component: PoolTypeIcon,
        getComponentProps: (itemInfo) => ({
          type: itemInfo.type
        })
      }}
      ref={ref}
      {...rest}
    />
  );
});

PoolTypeSelector.propTypes = {
  value: PropTypes.string,
  dataTestId: PropTypes.string,
  error: PropTypes.bool,
  helperText: PropTypes.string,
  onChange: PropTypes.func
};

export default PoolTypeSelector;
