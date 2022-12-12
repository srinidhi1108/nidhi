import React from "react";
import ExitToAppOutlinedIcon from "@mui/icons-material/ExitToAppOutlined";
import MenuItem from "components/MenuItem";
import { KINDS } from "stories";

export default {
  title: `${KINDS.COMPONENTS}/MenuItem`
};

export const basic = () => (
  <MenuItem messageId="hystax">
    <ExitToAppOutlinedIcon />
  </MenuItem>
);

export const withOnclick = () => (
  <MenuItem messageId="hystax" onClick={() => console.log("IconMenuItem clicked")}>
    <ExitToAppOutlinedIcon />
  </MenuItem>
);
