import React from "react";
import Skeleton from "@mui/material/Skeleton";
import { useTheme } from "@mui/material/styles";

const ActionBarHeaderLoader = () => {
  const theme = useTheme();
  return <Skeleton height={theme.spacing(6)} width={theme.spacing(30)} />;
};

export default ActionBarHeaderLoader;
