import React, { useState } from "react";
import PropTypes from "prop-types";
import { AttachEnvironmentDropdownButton } from "components/EnvironmentAttachmentDropdownButtons";
import makeRequest from "utils/makeRequest";

const AttachEnvironmentDropdownButtonContainer = ({
  environmentId,
  onSuccess,
  availableIssueStatusesForAutomaticUnlinking
}) => {
  const [isLoading, setIsLoading] = useState(false);

  const onAttach = ({ detachOnIssueStatus }) => {
    setIsLoading(true);
    makeRequest({
      url: `/jira_bus/v2/shareable_resource/${environmentId}/issue_attachment`,
      options: {
        method: "POST",
        body: JSON.stringify({
          auto_detach_status: detachOnIssueStatus
        })
      }
    }).then(({ error }) => {
      setIsLoading(false);

      if (!error && typeof onSuccess === "function") {
        onSuccess();
      }
    });
  };

  return (
    <AttachEnvironmentDropdownButton
      onAttach={onAttach}
      isLoading={isLoading}
      availableIssueStatusesForAutomaticUnlinking={availableIssueStatusesForAutomaticUnlinking}
    />
  );
};

AttachEnvironmentDropdownButtonContainer.propTypes = {
  environmentId: PropTypes.string.isRequired,
  onSuccess: PropTypes.func.isRequired,
  availableIssueStatusesForAutomaticUnlinking: PropTypes.array.isRequired
};

export default AttachEnvironmentDropdownButtonContainer;
