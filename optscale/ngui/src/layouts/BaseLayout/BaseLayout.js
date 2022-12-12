import React, { useState, Children } from "react";
import MenuIcon from "@mui/icons-material/Menu";
import AppBar from "@mui/material/AppBar";
import Box from "@mui/material/Box";
import Container from "@mui/material/Container";
import Drawer from "@mui/material/Drawer";
import Toolbar from "@mui/material/Toolbar";
import Typography from "@mui/material/Typography";
import PropTypes from "prop-types";
import { FormattedMessage } from "react-intl";
import { useNavigate } from "react-router-dom";
import { GAEvent, GA_EVENT_CATEGORIES } from "components/ActivityListener";
import Button from "components/Button";
import ErrorBoundary from "components/ErrorBoundary";
import HeaderButtons from "components/HeaderButtons";
import Hidden from "components/Hidden";
import IconButton from "components/IconButton";
import Logo from "components/Logo";
import TopAlertWrapper from "components/TopAlertWrapper";
import MainLayoutContainer from "containers/MainLayoutContainer";
import MainMenu from "containers/MainMenuContainer";
import OrganizationSelectorContainer from "containers/OrganizationSelectorContainer";
import { useIsDownMediaQuery } from "hooks/useMediaQueries";
import { useOrganizationInfo } from "hooks/useOrganizationInfo";
import { REGISTER } from "urls";
import { LOGO_SIZE } from "utils/constants";
import useStyles from "./BaseLayout.styles";

const logoHeight = 45;

const getLogoSize = (isDemo, isDownMd, isDownSm) => {
  if (isDemo) {
    return isDownMd ? LOGO_SIZE.SHORT : LOGO_SIZE.FULL;
  }
  return isDownSm ? LOGO_SIZE.SHORT : LOGO_SIZE.FULL;
};

const AppToolbar = ({ showMainMenu = false, onMenuIconClick, showOrganizationSelector = false }) => {
  const { classes, cx } = useStyles();

  const navigate = useNavigate();
  const isDownMd = useIsDownMediaQuery("md");
  const isDownSm = useIsDownMediaQuery("sm");

  const { isDemo } = useOrganizationInfo();

  const onLiveDemoRegisterClick = () => {
    navigate(REGISTER);
    GAEvent({ category: GA_EVENT_CATEGORIES.LIVE_DEMO, action: "Try register" });
  };

  return (
    <Toolbar className={classes.toolbar}>
      {showMainMenu && (
        <IconButton
          sx={{ display: { xs: "inherit", md: "none" } }}
          customClass={classes.marginRight1}
          icon={<MenuIcon />}
          color="inherit"
          onClick={onMenuIconClick}
          aria-label="open drawer"
        />
      )}
      <div style={{ height: logoHeight }} className={classes.logo}>
        <Logo
          size={getLogoSize(isDemo, isDownMd, isDownSm)}
          dataTestId="img_logo"
          height={logoHeight}
          demo={isDemo}
          white
          active
        />
      </div>
      {isDemo ? (
        <Box display="flex" alignItems="center">
          <Typography data-test-id="p_live_demo_mode" sx={{ display: { xs: "none", md: "inherit" } }}>
            <FormattedMessage id="liveDemoMode" />
          </Typography>
          <Button
            customClass={cx(classes.marginLeft1, classes.marginRight1)}
            disableElevation
            dataTestId="btn_register"
            messageId="register"
            variant="contained"
            size={isDownSm ? "small" : "medium"}
            color="success"
            onClick={onLiveDemoRegisterClick}
          />
        </Box>
      ) : null}
      <Box display="flex" alignItems="center">
        {showOrganizationSelector && <OrganizationSelectorContainer />}
        <HeaderButtons />
      </Box>
    </Toolbar>
  );
};

const BaseLayout = ({ children, showMainMenu = false, showOrganizationSelector = false }) => {
  const { organizationId } = useOrganizationInfo();

  const { classes } = useStyles();

  const [mobileOpen, setMobileOpen] = useState(false);

  const handleDrawerToggle = () => {
    setMobileOpen(!mobileOpen);
  };

  return (
    <>
      <TopAlertWrapper />
      <AppBar position="static" className={classes.appBar}>
        <AppToolbar
          showMainMenu={showMainMenu}
          onMenuIconClick={handleDrawerToggle}
          showOrganizationSelector={showOrganizationSelector}
        />
      </AppBar>
      <Box display="flex" flexGrow={1}>
        {showMainMenu && (
          <>
            <Hidden mode="down" breakpoint="md">
              <Drawer
                variant="permanent"
                classes={{
                  paper: classes.drawerPaper
                }}
                open
              >
                <MainMenu />
              </Drawer>
            </Hidden>
            <Hidden mode="up" breakpoint="md">
              <Drawer
                variant="temporary"
                classes={{
                  paper: classes.drawerPaper
                }}
                onClose={handleDrawerToggle}
                open={mobileOpen}
                ModalProps={{
                  keepMounted: true
                }}
              >
                <MainMenu />
              </Drawer>
            </Hidden>
          </>
        )}
        <Container key={organizationId} component="main" className={classes.content}>
          <ErrorBoundary>
            <MainLayoutContainer>{Children.only(children)}</MainLayoutContainer>
          </ErrorBoundary>
        </Container>
      </Box>
    </>
  );
};

AppToolbar.propTypes = {
  showMainMenu: PropTypes.bool.isRequired,
  onMenuIconClick: PropTypes.func.isRequired,
  showOrganizationSelector: PropTypes.bool.isRequired
};

BaseLayout.propTypes = {
  children: PropTypes.node.isRequired,
  showMainMenu: PropTypes.bool,
  showOrganizationSelector: PropTypes.bool
};

export default BaseLayout;
