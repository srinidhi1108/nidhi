import React from "react";
import ApartmentIcon from "@mui/icons-material/Apartment";
import Link from "@mui/material/Link";
import PropTypes from "prop-types";
import { Link as RouterLink } from "react-router-dom";
import Icon from "components/Icon";
import { getHomeUrl } from "urls";

const OrganizationLabel = ({ name, id, dataTestId, disableLink = false }) => (
  <>
    <Icon icon={ApartmentIcon} hasRightMargin />
    {!disableLink ? (
      <Link data-test-id={dataTestId} color="primary" to={getHomeUrl(id)} component={RouterLink}>
        {name}
      </Link>
    ) : (
      <span data-test-id={dataTestId}>{name}</span>
    )}
  </>
);

OrganizationLabel.propTypes = {
  name: PropTypes.string.isRequired,
  id: PropTypes.string,
  disableLink: PropTypes.bool,
  dataTestId: PropTypes.string
};

export default OrganizationLabel;
