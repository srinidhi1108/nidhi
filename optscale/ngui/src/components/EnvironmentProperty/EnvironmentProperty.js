import React, { useState } from "react";
import CreateOutlinedIcon from "@mui/icons-material/CreateOutlined";
import DeleteOutlinedIcon from "@mui/icons-material/DeleteOutlined";
import { Typography } from "@mui/material";
import PropTypes from "prop-types";
import IconButton from "components/IconButton";
import Markdown from "components/Markdown";
import PropertyGridLayout from "components/PropertyGridLayout";
import { DeleteEnvironmentPropertyModal } from "components/SideModalManager/SideModals";
import UpdateEnvironmentPropertiesFormContainer from "containers/UpdateEnvironmentPropertiesFormContainer";
import { useIsAllowedToCUDEnvironmentProperties } from "hooks/useIsAllowedToCUDEnvironmentProperties";
import { useOpenSideModal } from "hooks/useOpenSideModal";

const EnvironmentProperty = ({ environmentId, propertyName, propertyValue, existingProperties }) => {
  const [editMode, setEditMode] = useState(false);

  const openSideModal = useOpenSideModal();

  const isAllowedToCUDEnvironmentProperties = useIsAllowedToCUDEnvironmentProperties();

  return editMode ? (
    <UpdateEnvironmentPropertiesFormContainer
      environmentId={environmentId}
      defaultPropertyName={propertyName}
      defaultPropertyValue={propertyValue}
      onSuccess={() => setEditMode(false)}
      onCancel={() => setEditMode(false)}
      existingProperties={existingProperties}
      isEdit
    />
  ) : (
    <PropertyGridLayout
      propertyName={
        <Typography component="span" style={{ wordBreak: "break-all" }}>
          {propertyName}
        </Typography>
      }
      propertyValue={<Markdown>{propertyValue}</Markdown>}
      iconButtons={
        isAllowedToCUDEnvironmentProperties && (
          <>
            <IconButton
              icon={<CreateOutlinedIcon />}
              onClick={() => setEditMode(true)}
              tooltip={{ show: true, messageId: "edit" }}
            />
            <IconButton
              color="error"
              icon={<DeleteOutlinedIcon />}
              onClick={() => openSideModal(DeleteEnvironmentPropertyModal, { environmentId, propertyName })}
            />
          </>
        )
      }
    />
  );
};

EnvironmentProperty.propTypes = {
  environmentId: PropTypes.string.isRequired,
  existingProperties: PropTypes.object.isRequired,
  propertyName: PropTypes.string.isRequired,
  propertyValue: PropTypes.string.isRequired
};

export default EnvironmentProperty;
