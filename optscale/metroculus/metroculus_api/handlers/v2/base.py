import json
import logging
import traceback
from json.decoder import JSONDecodeError
import tornado.web

from metroculus_api.exceptions import Err
from metroculus_api.utils import ModelEncoder

from optscale_exceptions.common_exc import UnauthorizedException
from optscale_exceptions.http_exc import OptHTTPError


LOG = logging.getLogger(__name__)


class DefaultHandler(tornado.web.RequestHandler):
    def write_error(self, status_code, **kwargs):
        self.set_header('Content-Type', 'application/json')
        self.set_status(404)
        self.finish(json.dumps({
            'error': {
                'status_code': 404,
                'error_code': Err.OM0002.name,
                'reason': self._reason,
                'params': [],
            }
        }))


class BaseHandler(tornado.web.RequestHandler):
    def initialize(self, config):
        self._config = config
        self._controller = None

    def raise405(self):
        raise OptHTTPError(405, Err.OM0003, [self.request.method])

    def head(self, *args, **kwargs):
        self.raise405()

    def get(self, *args, **kwargs):
        self.raise405()

    def post(self, *args, **kwargs):
        self.raise405()

    def delete(self, *args, **kwargs):
        self.raise405()

    def patch(self, *args, **kwargs):
        self.raise405()

    def put(self, *args, **kwargs):
        self.raise405()

    def options(self, *args, **kwargs):
        self.raise405()

    def _get_request(self):
        return self.request

    def prepare(self):
        self.set_content_type()

    def set_content_type(self,
                         content_type='application/json; charset="utf-8"'):
        self.set_header('Content-Type', content_type)

    @property
    def controller(self):
        if not self._controller:
            self._controller = self._get_controller_class()(self._config)
        return self._controller

    def _get_controller_class(self):
        raise NotImplementedError

    def write_error(self, status_code, **kwargs):
        exc = kwargs.get('exc_info')[1]
        res = {
            'error': {
                'status_code': status_code,
                'error_code': getattr(exc, 'error_code', 'U0%s' % status_code),
                'reason': self._reason,
                'params': getattr(exc, 'params', []),
            }
        }
        self.set_content_type('application/json; charset="utf-8"')
        self.finish(json.dumps(res, cls=ModelEncoder))

    def _request_body(self):
        try:
            return json.loads(self.request.body.decode('utf-8'))
        except JSONDecodeError:
            raise OptHTTPError(400, Err.OM0004, [])

    def _request_arguments(self):
        return self.request.arguments

    def log_exception(self, typ, value, tb):
        out_list = traceback.format_exception(typ, value, tb)
        if isinstance(value, tornado.web.HTTPError):
            if value.log_message:
                format = "%d %s: " + value.log_message + "\\n%s"
                args = ([value.status_code, self._request_summary()] +
                        list(value.args) + [repr(''.join(out_list))])
            else:
                format = "%d %s:\\n%s"
                args = ([value.status_code, self._request_summary()] +
                        [repr(''.join(out_list))])
            LOG.warning(format, *args)
        else:
            LOG.error("Uncaught exception %s\\n%r\\n %s",
                      self._request_summary(), self.request,
                      repr(''.join(out_list)))

    def get_request_data(self):
        raise NotImplementedError

    def get_request_arguments(self):
        return self._request_arguments()

    def get_arg(self, name, type, default=None, repeated=False):
        try:
            if repeated:
                result = [type(a) for a in self.get_arguments(name)]
                if not result and default:
                    result = default
                return result
            else:
                arg = self.get_argument(name, default=default)
                if arg:
                    if type == bool and isinstance(arg, str):
                        lowered = arg.lower()
                        if lowered not in ['true', 'false']:
                            raise ValueError('%s should be true or false' % arg)
                        return lowered == 'true'
                    return type(arg)
                else:
                    return arg
        except (ValueError, TypeError):
            raise OptHTTPError(400, Err.OM0006, [name])


class SecretHandler(BaseHandler):

    def initialize(self, config):
        super().initialize(config)
        self.cluster_secret = config.cluster_secret()

    @property
    def secret(self):
        return self.request.headers.get('Secret')

    def check_cluster_secret(self, **kwargs):
        return self._check_secret(self.cluster_secret, **kwargs)

    def _check_secret(self, secret, raises=True):
        if raises and not self.secret == secret:
            raise OptHTTPError(403, Err.OM0005, [])
        else:
            return self.secret == secret

    def prepare(self):
        try:
            if not self.secret:
                raise OptHTTPError(401, Err.OM0007, [])
        except UnauthorizedException as exc:
            self.set_status(401)
            raise OptHTTPError.from_opt_exception(401, exc)
        super().prepare()
