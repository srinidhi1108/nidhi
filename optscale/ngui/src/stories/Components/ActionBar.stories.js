import React from "react";
import AddOutlinedIcon from "@mui/icons-material/AddOutlined";
import ActionBar from "components/ActionBar";
import AttachMoneyOutlinedIcon from "@mui/icons-material/AttachMoneyOutlined";
import EditOutlinedIcon from "@mui/icons-material/EditOutlined";
import { boolean } from "@storybook/addon-knobs";
import { KINDS } from "stories";

export default {
  title: `${KINDS.COMPONENTS}/ActionBar`
};

const actionBarDefinition = {
  title: {
    text: "Title (text)"
  },
  items: [
    {
      key: "pool",
      icon: <AddOutlinedIcon fontSize="small" />,
      messageId: "add",
      type: "button",
      link: "#"
    },
    {
      key: "employees",
      icon: <AddOutlinedIcon fontSize="small" />,
      messageId: "add",
      type: "button",
      link: "#"
    },
    {
      key: "overview-assignPool",
      icon: <AttachMoneyOutlinedIcon fontSize="small" />,
      messageId: "assignPool",
      link: "#",
      type: "button"
    },
    {
      key: "overview-edit",
      icon: <EditOutlinedIcon fontSize="small" />,
      messageId: "edit",
      link: "#",
      type: "button"
    }
  ]
};

const actionBarDefinitionWithPopup = {
  title: {
    messageId: "organization"
  },
  items: [
    {
      key: "clouds-add",
      icon: <AddOutlinedIcon fontSize="small" />,
      messageId: "add",
      type: "dropdown",
      menu: {
        items: [
          {
            key: "pool",
            messageId: "pool",
            link: "#"
          },
          {
            key: "employees",
            messageId: "employees",
            link: "#"
          }
        ]
      }
    },
    {
      key: "overview-edit",
      icon: <EditOutlinedIcon fontSize="small" />,
      messageId: "edit",
      link: "#",
      type: "button"
    }
  ]
};

const withMobileMessageId = {
  title: {
    messageId: "organization"
  },
  items: [
    {
      key: "clouds-add",
      icon: <AddOutlinedIcon fontSize="small" />,
      messageId: "add",
      type: "dropdown",
      menu: {
        items: [
          {
            key: "pool",
            messageId: "pool",
            mobileMessageId: "add",
            link: "#"
          },
          {
            key: "employees",
            messageId: "employees",
            mobileMessageId: "add",
            link: "#"
          }
        ]
      }
    },
    {
      key: "overview-edit",
      icon: <EditOutlinedIcon fontSize="small" />,
      messageId: "edit",
      link: "#",
      type: "button"
    }
  ]
};

export const basic = () => <ActionBar data={actionBarDefinition} />;

export const withDropdown = () => <ActionBar data={actionBarDefinitionWithPopup} />;

export const withKnobs = () => (
  <ActionBar data={boolean("with mobileMessageId:", true) ? withMobileMessageId : actionBarDefinitionWithPopup} />
);
