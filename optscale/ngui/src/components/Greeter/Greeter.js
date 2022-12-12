import React from "react";
import LanguageOutlinedIcon from "@mui/icons-material/LanguageOutlined";
import Grid from "@mui/material/Grid";
import Link from "@mui/material/Link";
import { useTheme } from "@mui/material/styles";
import Typography from "@mui/material/Typography";
import PropTypes from "prop-types";
import { FormattedMessage, useIntl } from "react-intl";
import { useNavigate } from "react-router-dom";
import certifiedFinopsSolution from "assets/welcome/certified-finops-solution.svg";
import costAnomalyDetection from "assets/welcome/cost-anomaly-detection.svg";
import finopsReadinessMaturityAssessment from "assets/welcome/finops-readiness-maturity-assessment.svg";
import freeCloudK8sCostOptimization from "assets/welcome/free-cloud-K8s-cost-optimization.svg";
import geoNetworkTrafficCostMap from "assets/welcome/geo-network-traffic-cost-map.svg";
import itEnvironmentManagement from "assets/welcome/it-environment-management.svg";
import Button from "components/Button";
import CustomersGallery from "components/CustomersGallery";

import IntegrationsGallery from "components/IntegrationsGallery";
import Logo from "components/Logo";
import OAuthSignIn from "components/OAuthSignIn";
import SubTitle from "components/SubTitle";
import { useIsDownMediaQuery, useIsUpMediaQuery } from "hooks/useMediaQueries";
import { HYSTAX, LIVE_DEMO, HYSTAX_FORRESTER } from "urls";
import { tag as tagHotjar } from "utils/hotjar";
import { SPACING_4, SPACING_2 } from "utils/layouts";
import useStyles from "./Greeter.styles";

const OptScaleLink = () => {
  const { classes, cx } = useStyles();
  const intl = useIntl();

  return (
    <SubTitle className={classes.rightTextColor}>
      <LanguageOutlinedIcon className={cx(classes.icon, classes.rightTextColor)} />
      <Link
        data-test-id="link_optscale_site"
        href={HYSTAX}
        onClick={() => {
          tagHotjar(["went_optscale_website"]);
        }}
        color="inherit"
        target="_blank"
        rel="noopener"
      >
        {intl.formatMessage({ id: "hystaxDotCom" })}
      </Link>
    </SubTitle>
  );
};

const ImagesWithCaptions = () => {
  const intl = useIntl();
  const { classes, cx } = useStyles();

  const data = [
    { caption: "welcome.caption1", src: certifiedFinopsSolution },
    { caption: "welcome.caption2", src: finopsReadinessMaturityAssessment },
    { caption: "welcome.caption3", src: freeCloudK8sCostOptimization },
    { caption: "welcome.caption4", src: costAnomalyDetection },
    { caption: "welcome.caption5", src: itEnvironmentManagement },
    { caption: "welcome.caption6", src: geoNetworkTrafficCostMap }
  ];

  const isUpLg = useIsUpMediaQuery("lg");

  return (
    <Grid container spacing={isUpLg ? SPACING_4 : SPACING_2} className={classes.imagesWithCaptions}>
      {data.map(({ caption, src }, index) => (
        <Grid item lg={4} md={4} sm={6} key={caption} className={classes.imageWithCaptionWrapper}>
          <img
            src={src}
            alt={intl.formatMessage({ id: caption })}
            data-test-id={`img_banner_${index}`}
            className={classes.image}
          />
          <SubTitle dataTestId={`img_banner_caption_${index}`} className={cx(classes.rightTextColor, classes.caption)}>
            <FormattedMessage id={caption} />
          </SubTitle>
        </Grid>
      ))}
    </Grid>
  );
};

const LiveDemoButton = ({ onClick }) => (
  <Button
    dataTestId="btn_live_demo"
    color="lightYellow"
    variant="contained"
    messageId="liveDemo"
    size="large"
    onClick={onClick}
  />
);

const defaultOrder = [0, 1, 2, 3, 4, 5];

const getVerticalOrder = () => {
  const [evenNumbers, oddNumbers] = defaultOrder.reduce(
    ([even, odd], curr) => (curr % 2 === 0 ? [[...even, curr], [...odd]] : [[...even], [...odd, curr]]),
    [[], []]
  );
  return [...evenNumbers, ...oddNumbers];
};

const Greeter = ({ form, withOAuth = true }) => {
  const { classes, cx } = useStyles();
  const theme = useTheme();
  const navigate = useNavigate();

  const isInVerticalOrder = useIsDownMediaQuery("md");

  // default order
  //    order0 - empty
  //    order1 - link/button
  //    order2 - form
  //    order3 - map and text
  //    order4 - customers
  //    order5 - empty
  // ------------------------------
  // vertical order
  //    order0 - empty
  //    order1 - form
  //    order2 - customers
  //    order3 - link/button
  //    order4 - map and text
  //    order5 - empty
  const order = isInVerticalOrder ? getVerticalOrder() : defaultOrder;

  const gridDefinition = [
    {
      key: "empty"
    },
    {
      key: "link",
      children: (
        <div className={classes.linkWrapper}>
          <LiveDemoButton onClick={() => navigate(LIVE_DEMO)} />
          <OptScaleLink />
        </div>
      )
    },
    {
      key: "form",
      children: (
        <Grid container alignItems="center" justifyContent="center" className={classes.wrapper} spacing={SPACING_4}>
          <Grid item xs={12}>
            <Logo width={200} dataTestId="img_logo" />
            {form}
          </Grid>
          {withOAuth && (
            <Grid item xs={12}>
              <OAuthSignIn />
            </Grid>
          )}
        </Grid>
      ),
      className: classes.centeredFlexColumnDirection
    },
    {
      key: "bannerAndText",
      className: classes.centeredFlexColumnDirection,
      children: <ImagesWithCaptions />
    },
    {
      key: "customers",
      className: classes.centeredFlexColumnDirection,
      children: (
        <>
          <CustomersGallery />
          <Typography>
            <Link data-test-id="link_recognized_by_forrester" href={HYSTAX_FORRESTER} target="_blank" rel="noopener">
              <FormattedMessage id="recognizedByForrester" />
            </Link>
          </Typography>
        </>
      )
    },
    {
      key: "integrations",
      className: classes.centeredFlexColumnDirection,
      children: <IntegrationsGallery />
    }
  ];

  const spacing = SPACING_2;
  const halfSpacing = spacing / 2;

  /**
   * TODO: Remove custom padding and margin when mui-v5 fixes will released
   *    https://gitlab.com/hystax/ngui/-/merge_requests/2495
   *    https://github.com/mui-org/material-ui/issues/29266
   *    https://github.com/mui-org/material-ui/pull/30333
   */
  return (
    <div
      style={{
        padding: theme.spacing(halfSpacing)
      }}
    >
      <Grid
        sx={{
          m: -halfSpacing
        }}
        spacing={spacing}
        container
        className={classes.root}
      >
        {order.map((gridIndex) => {
          const { key, children = null, className = "" } = gridDefinition[gridIndex];
          return (
            <Grid
              sx={{
                p: spacing
              }}
              key={key}
              md={6}
              xs={12}
              item
              className={cx(gridIndex % 2 === 0 ? classes.leftSideGrid : classes.rightSideGrid, className)}
            >
              {children}
            </Grid>
          );
        })}
      </Grid>
    </div>
  );
};

LiveDemoButton.propTypes = {
  onClick: PropTypes.func.isRequired
};

Greeter.propTypes = {
  form: PropTypes.node.isRequired,
  withOAuth: PropTypes.bool
};

export default Greeter;
