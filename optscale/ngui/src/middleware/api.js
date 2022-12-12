import axios from "axios";
import queryString from "query-string";
import { v4 as uuidv4 } from "uuid";
import { apiEnd, apiError, apiStart, apiSuccess, resetTtl } from "api";
import { API } from "api/actionTypes";
import { GET_TOKEN } from "api/auth/actionTypes";
import { getUrlWithNextQueryParam, SIGNOUT } from "urls";
import { ALERT_SEVERITY } from "utils/constants";
import { getFullPath } from "utils/network";
import requestManager from "utils/requestManager";
import { getSuccessAlertSettingsByLabel } from "utils/successCodes";
import history from "../history";

axios.interceptors.request.use(
  (config) => {
    requestManager.addAffectedRequest(config.label, config.affectedRequests);
    return config;
  },
  (error) => Promise.reject(error)
);

const apiMiddleware =
  ({ dispatch, getState }) =>
  (next) =>
  (action) => {
    if (typeof action === "function") {
      return action(dispatch, getState);
    }

    next(action);

    if (action.type !== API) return undefined;

    const {
      url,
      method,
      params,
      onSuccess,
      label,
      ttl,
      hash,
      entityId,
      errorHandlerType,
      successHandlerType,
      successHandlerPayload,
      headersOverride: headers,
      affectedRequests,
      allowMultipleRequests
    } = action.payload;

    if (!label) return undefined;

    const state = getState();

    const accessToken = state?.auth?.[GET_TOKEN]?.token;

    const dataOrParams = ["GET", "DELETE"].includes(method) ? "params" : "data";

    axios.defaults.baseURL = process.env.REACT_APP_BASE_URL || "";
    axios.defaults.headers.common["Content-Type"] = "application/json";
    axios.defaults.headers.common.Authorization = `Bearer ${accessToken}`;

    const requestId = uuidv4();

    if (!allowMultipleRequests && requestManager.hasPendingRequest(label)) {
      requestManager.cancelPreviousPendingRequests(label);
    }

    dispatch(apiStart(label, hash, entityId));

    const requestSignal = requestManager.addPendingRequest(requestId, label);

    return axios
      .request({
        url,
        method,
        headers,
        label,
        [dataOrParams]: params,
        signal: requestSignal,
        affectedRequests,
        // ?foo[]=bar1&foo[]=bar2 -> ?foo=bar1&foo=bar2
        paramsSerializer: (queryParams) => queryString.stringify(queryParams)
      })
      .then((response) => {
        if (typeof onSuccess === "function") {
          dispatch(onSuccess(response.data, label));
        }
        const { code: successCode, getMessageParams, getSeverity } = getSuccessAlertSettingsByLabel(label);

        // resetting affected requests
        requestManager.getAffectedRequests(label).forEach((apiLabel) => dispatch(resetTtl(apiLabel)));

        dispatch(
          apiSuccess({
            label,
            response,
            ttl,
            successHandlerType,
            messageParams:
              typeof getMessageParams === "function" ? getMessageParams(params, response, successHandlerPayload) : [],
            alertSeverity:
              typeof getSeverity === "function" ? getSeverity(params, response, successHandlerPayload) : ALERT_SEVERITY.SUCCESS,
            code: successCode
          })
        );
      })
      .catch((error) => {
        let errorResponse;
        if (error.response) {
          // The request was made and the server responded with a status code
          // that falls out of the range of 2xx
          errorResponse = error.response;
          // Specific error codes handling
          // TODO - investigate error codes handling instead of using Protector/Error page wrappers everywhere
          if (error.response.status === 401) {
            requestManager.cancelAllPendingRequests();
            const to = getUrlWithNextQueryParam(SIGNOUT, getFullPath());
            history.push(to);
          }
        } else if (error.request) {
          // The request was made but no response was received
          // `error.request` is an instance of XMLHttpRequest in the browser and an instance of
          // http.ClientRequest in node.js
          console.log(error.request);
          errorResponse = { config: { url: error.request, params: {} } };
        } else {
          // Something happened in setting up the request that triggered an error
          console.log("error: ", error.message);
          errorResponse = { config: { url: error.message, params: {} } };
        }
        dispatch(apiError(label, errorResponse, errorHandlerType));
      })
      .finally(() => {
        requestManager.removePendingRequest(requestId);
        if (!requestManager.hasPendingRequest(label)) {
          dispatch(apiEnd(label));
        }
      });
  };

export default apiMiddleware;
