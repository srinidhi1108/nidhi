import React from "react";
import PropTypes from "prop-types";
import OrganizationOptionForm from "components/OrganizationOptionForm";
import OrganizationOptionsService from "services/OrganizationOptionsService";

const CreateOrganizationOptionContainer = ({ onCancel }) => {
  const { useCreateOption } = OrganizationOptionsService();
  const { isCreateOrganizationOptionLoading, createOption } = useCreateOption();

  const submit = (optionName, optionValue) => {
    // optionValue comes from JsonView only if it is valid
    if (optionValue) {
      createOption(optionName, optionValue, onCancel);
    }
  };

  return <OrganizationOptionForm onCancel={onCancel} onSubmit={submit} isLoading={isCreateOrganizationOptionLoading} />;
};

CreateOrganizationOptionContainer.propTypes = {
  onCancel: PropTypes.func.isRequired
};

export default CreateOrganizationOptionContainer;
