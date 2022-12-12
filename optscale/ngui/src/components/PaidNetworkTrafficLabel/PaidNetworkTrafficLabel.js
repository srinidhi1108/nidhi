import React from "react";
import PropTypes from "prop-types";
import { FormattedMessage } from "react-intl";
import FormattedDigitalUnit, { SI_UNITS } from "components/FormattedDigitalUnit";
import FormattedMoney from "components/FormattedMoney";
import { FORMATTED_MONEY_TYPES } from "utils/constants";

const PaidNetworkTrafficLabel = ({ cost, usage }) => (
  <FormattedMessage
    id="value / value"
    values={{
      value1: <FormattedMoney type={FORMATTED_MONEY_TYPES.COMMON} value={cost} />,
      value2: <FormattedDigitalUnit value={usage} baseUnit={SI_UNITS.GIGABYTE} />
    }}
  />
);

PaidNetworkTrafficLabel.propTypes = {
  cost: PropTypes.number.isRequired,
  usage: PropTypes.number.isRequired
};

export default PaidNetworkTrafficLabel;
