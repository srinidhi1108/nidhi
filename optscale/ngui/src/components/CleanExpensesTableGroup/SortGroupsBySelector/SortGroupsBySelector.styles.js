import { makeStyles } from "tss-react/mui";

const useStyles = makeStyles()((theme) => ({
  selector: {
    minWidth: "initial",
    "& .MuiInputBase-root": {
      fontSize: "inherit"
    }
  },
  titleText: {
    marginRight: theme.spacing(0.5)
  },
  sortGroupsByWrapper: {
    marginLeft: theme.spacing(3)
  }
}));

export default useStyles;
