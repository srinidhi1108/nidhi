import yaml
import os.path
import re
import rest_api_server.server as server
from apispec import APISpec, utils

# Spec reference:
# https://github.com/OAI/OpenAPI-Specification/blob/master/versions/2.0.md

OPENAPI_SPEC = """
swagger: '2.0'
info:
    description: >
        OptScale Rest API. Call `POST` `/auth/v2/tokens` with `{"email": "<email>",
        "password": "<password>"}` body to receive an authorization token and
        use `Bearer <token>` string in `Authorization` header to authorize.\n\n
        Permission definitions: \n\n
        -   `ACTION`: (INFO_ORGANIZATION, MANAGE_INVITES etc) you must use
        an authorization token and have a assignment on resource with
        the appropriate action\n\n
        -   `TOKEN`: To use this API you only need a token\n\n
        -   `NOT_PROTECTED`: open api\n\n
        -   `CLUSTER_SECRET`: Private API, used inside the cluster by services
        with special secret key\n\n
        `Important`: Be careful, all APIs uses only one method of auth at the
        same time. Using both methods `TOKEN` and `CLUSTER_SECRET` can
        return unexpected response data.
    title: Rest API
    version: 1.0.0
securityDefinitions:
    token:
        in: header
        name: 'Authorization'
        type: apiKey
    secret:
        in: header
        name: 'Secret'
        type: apiKey
"""


def main():
    settings = yaml.load(OPENAPI_SPEC)
    title = settings['info'].pop('title')
    spec_version = settings['info'].pop('version')
    openapi_version = settings.pop('swagger')
    for version in ['v2']:
        spec = APISpec(
            title=title,
            version=spec_version,
            openapi_version=openapi_version,
            plugins=(),
            **settings
        )

        handlers = server.get_handlers(dict(), version)
        for urlspec in handlers:
            path = re.sub(r"\(.*?<(.*?)>.*?\)", r"{\1}", urlspec[0])
            operations = dict()
            for method_name in utils.PATH_KEYS:
                method = getattr(urlspec[1], method_name)
                operation_data = utils.load_yaml_from_docstring(method.__doc__)
                if operation_data:
                    operations[method_name] = operation_data
            if len(operations) > 0:
                spec.add_path(path=path, operations=operations)
            else:
                print("Warning: docstrings for '" + urlspec[0] + "' are not found")

        # Api spec file
        with open(os.path.join(server.SWAGGER_PATH, "spec_%s.yaml" % version
                               ), "w") as file:
            file.write(spec.to_yaml())


if __name__ == "__main__":
    main()
