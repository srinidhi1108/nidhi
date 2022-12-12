import React, { useState } from "react";
import AccountCircleIcon from "@mui/icons-material/AccountCircle";
import LiveHelpOutlinedIcon from "@mui/icons-material/LiveHelpOutlined";
import MenuBookOutlinedIcon from "@mui/icons-material/MenuBookOutlined";
import MoreVertIcon from "@mui/icons-material/MoreVert";
import Box from "@mui/material/Box";
import Menu from "@mui/material/Menu";
import { FormattedMessage } from "react-intl";
import { useDispatch } from "react-redux";
import IconButton from "components/IconButton";
import Popover from "components/Popover";
import { startTour, PRODUCT_TOUR } from "components/ProductTour";
import ProfileMenuContainer from "containers/ProfileMenuContainer";
import { DOCS_HYSTAX_OPTSCALE } from "urls";
import useStyles from "./HeaderButtons.styles";

const HeaderButtons = () => {
  const { classes } = useStyles();
  const dispatch = useDispatch();

  const [anchorEl, setAnchorEl] = useState(null);
  const openMobileMenu = (event) => {
    setAnchorEl(event.currentTarget);
  };

  const closeMobileMenu = () => {
    setAnchorEl(null);
  };

  const startProductTour = () => {
    dispatch(startTour(PRODUCT_TOUR));
  };

  return (
    <>
      <Box component="div" className={classes.sectionDesktop}>
        {/* Was commented out due to the NGUI-1039 task
      <LatestEventsContainer /> */}
        <IconButton
          dataTestId="btn_doc"
          color="inherit"
          href={DOCS_HYSTAX_OPTSCALE}
          icon={<MenuBookOutlinedIcon />}
          tooltip={{
            show: true,
            value: <FormattedMessage id="documentation" />
          }}
        />
        <IconButton
          dataTestId="btn_product_tour"
          color="inherit"
          icon={<LiveHelpOutlinedIcon />}
          onClick={startProductTour}
          tooltip={{
            show: true,
            value: <FormattedMessage id="productTour" />
          }}
        />
        <Popover
          label={
            <IconButton
              dataTestId="btn_profile"
              icon={<AccountCircleIcon />}
              color="inherit"
              tooltip={{
                show: true,
                value: <FormattedMessage id="profile" />
              }}
            />
          }
          menu={<ProfileMenuContainer />}
        />
      </Box>
      {/* TODO: Maybe we can make the Popup component more universal and include the case below */}
      {/* TODO: https://datatrendstech.atlassian.net/browse/NGUI-2808 to handle dynamic header buttons, product tour is hidden on mdDown (when hamburger menu is activated) */}
      <Box component="div" className={classes.sectionMobile}>
        <IconButton icon={<MoreVertIcon />} color="inherit" onClick={openMobileMenu} />
        <Menu anchorEl={anchorEl} open={Boolean(anchorEl)} onClose={closeMobileMenu}>
          {/* Was commented out due to the NGUI-1039 task
        <Box className={classes.customMenuItem}>
          <LatestEventsContainer />
        </Box> */}
          <Box className={classes.customMenuItem}>
            <IconButton
              href={DOCS_HYSTAX_OPTSCALE}
              icon={<MenuBookOutlinedIcon />}
              size="medium"
              color="primary"
              tooltip={{
                show: true,
                value: <FormattedMessage id="documentation" />
              }}
            />
          </Box>
          <Box className={classes.customMenuItem}>
            <Popover
              label={
                <IconButton
                  icon={<AccountCircleIcon />}
                  size="medium"
                  color="primary"
                  tooltip={{
                    show: true,
                    value: <FormattedMessage id="profile" />
                  }}
                />
              }
              menu={<ProfileMenuContainer />}
            />
          </Box>
        </Menu>
      </Box>
    </>
  );
};

export default HeaderButtons;
