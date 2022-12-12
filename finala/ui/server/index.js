const { resolve } = require("path");
const express = require("express");
const middleware = require("./middleware.base");

const app = express();

middleware(app, {
  outputPath: resolve(process.cwd(), "build"),
  publicPath: "/"
});

app.listen(8080, err => {
  if (err) {
    return console.error(err); // eslint-disable-line no-console
  }
  console.log("Listening at http://localhost:8080"); // eslint-disable-line no-console
});
