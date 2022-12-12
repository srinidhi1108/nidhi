import React from "react";
import { useTheme } from "@emotion/react";
import { FormControlLabel, Typography } from "@mui/material";
import { alpha } from "@mui/material/styles";
import Switch from "@mui/material/Switch";
import PropTypes from "prop-types";
import { FormattedMessage } from "react-intl";
import FormattedMoney from "components/FormattedMoney";
import { useShowLessThanValue } from "hooks/useShowLessThanValue";

// TODO: that is copy of Show weekends stuff
const ShowLessThanValueSwitch = ({ disabled = false }) => {
  const { showLessThanValue, onChange } = useShowLessThanValue();
  const theme = useTheme();

  return (
    <FormControlLabel
      control={<Switch disabled={disabled} onChange={(e) => onChange(e.target.checked)} checked={showLessThanValue} />}
      label={
        <Typography style={{ color: alpha(theme.palette.text.primary, disabled ? 0.5 : 1) }}>
          <FormattedMessage id="showDirectionsWithExpenses" values={{ value: <FormattedMoney value="1" /> }} />
        </Typography>
      }
      labelPlacement="start"
      sx={{
        float: "right",
        marginRight: "0"
      }}
    />
  );
};

ShowLessThanValueSwitch.propTypes = {
  disabled: PropTypes.bool
};

export default ShowLessThanValueSwitch;
