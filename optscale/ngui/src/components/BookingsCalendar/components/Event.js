import React from "react";
import Typography from "@mui/material/Typography";
import "react-big-calendar/lib/css/react-big-calendar.css";
import { useIntl } from "react-intl";
import CloudResourceId from "components/CloudResourceId";
import KeyValueLabel from "components/KeyValueLabel";
import PoolLabel from "components/PoolLabel";
import ResourceLink from "components/ResourceLink";
import { RESOURCE_PAGE_TABS } from "utils/constants";
import { EN_FULL_FORMAT, format, MAX_UTC_DATE_TIMESTAMP } from "utils/datetime";
import useStyles from "../BookingsCalendar.styles";
import Popover from "./Popover";

const Event = ({ event }) => {
  const intl = useIntl();
  const { classes } = useStyles();

  const {
    environment,
    booking,
    props: { linkedTitle = true }
  } = event;
  const {
    name: environmentName,
    cloud_resource_id: cloudResourceId,
    resource_type: resourceType,
    pool_id: poolId,
    pool_name: poolName,
    pool_purpose: poolPurpose,
    id: environmentId
  } = environment;
  const {
    acquired_by: { name: employeeName }
  } = booking;

  const getTimeLabel = () => {
    const startTime = format(event.start, EN_FULL_FORMAT);
    const endTime =
      +event.end === MAX_UTC_DATE_TIMESTAMP ? intl.formatMessage({ id: "notLimited" }) : format(event.end, EN_FULL_FORMAT);

    return `${startTime} — ${endTime}`;
  };

  const renderLinkedTitle = () =>
    environmentName !== undefined || environmentName !== null ? (
      <ResourceLink tabName={RESOURCE_PAGE_TABS.DETAILS} resourceId={environmentId}>
        {environmentName}
      </ResourceLink>
    ) : (
      <CloudResourceId cloudResourceId={cloudResourceId} resourceId={environmentId} />
    );

  return (
    <Popover
      popoverId="event-popover"
      popoverContent={
        <div className={classes.eventPopupWrapper}>
          <Typography paragraph component="div">
            <div>{linkedTitle ? renderLinkedTitle() : <strong>{environmentName ?? cloudResourceId}</strong>}</div>
            <div>{getTimeLabel()}</div>
          </Typography>
          <KeyValueLabel value={employeeName} messageId="user" />
          <KeyValueLabel value={resourceType} messageId="resourceType" />
          <KeyValueLabel value={<PoolLabel id={poolId} name={poolName} type={poolPurpose} />} messageId="pool" />
        </div>
      }
    >
      {event.title}
    </Popover>
  );
};

export default Event;
