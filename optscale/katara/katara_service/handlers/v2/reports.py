import json

from katara_service.controllers.report import ReportAsyncController
from katara_service.handlers.v2.base import (
    BaseAsyncCollectionHandler, BaseAsyncItemHandler)
from katara_service.utils import ModelEncoder


class ReportAsyncCollectionHandler(BaseAsyncCollectionHandler):
    def _get_controller_class(self):
        return ReportAsyncController

    async def post(self, **kwargs):
        self.raise405()

    async def get(self, **kwargs):
        """
        ---
        tags: [reports]
        summary: List reports
        description: >
            Gets a list of reports \n\n
            Required permission: CLUSTER_SECRET
        responses:
            200:
                description: Success
                schema:
                    type: object
                    properties:
                        reports:
                            type: array
                            items:
                                type: object
                                properties:
                                    id: {type: string,
                                        description: "Unique report id"}
                                    created_at: {type: integer,
                                        description: "Created timestamp (service field)"}
                                    name: {type: string,
                                        description: "Report name"}
                                    module_name: {type: string,
                                        description: "Report module name"}
                                    report_format: {type: string,
                                        description: "Report format"}
                                    template: {type: string,
                                        description: "Report template"}
                                    description: {type: string,
                                        description: "Report description"}
            401:
                description: |
                    Unauthorized:
                    - OKA0011: This resource requires authorization
            403:
                description: |
                    Forbidden:
                    - OKA0010: Bad secret
        security:
        - secret: []
        """
        res = await self.controller.list(**kwargs)
        report_dict = {'reports': [
            report.to_dict() for report in res]}
        self.write(json.dumps(report_dict, cls=ModelEncoder))


class ReportAsyncItemHandler(BaseAsyncItemHandler):
    def _get_controller_class(self):
        return ReportAsyncController

    async def delete(self, id, **kwargs):
        await self.raise405()

    async def patch(self, id, **kwargs):
        self.raise405()

    async def get(self, id):
        """
        ---
        description: >
            Gets report with specified ID \n\n
            Required permission: CLUSTER_SECRET
        tags: [reports]
        summary: Get report
        parameters:
        -   name: id
            in: path
            description: Report ID
            required: true
            type: string
        responses:
            200:
                description: Report data
                schema:
                    type: object
                    properties:
                        id: {type: string,
                            description: "Unique report id"}
                        created_at: {type: integer,
                            description: "Created timestamp (service field)"}
                        name: {type: string,
                            description: "Report name"}
                        module_name: {type: string,
                            description: "Report module name"}
                        report_format: {type: string,
                            description: "Report format"}
                        template: {type: string,
                            description: "Report template"}
                        description: {type: string,
                            description: "Report description"}
            401:
                description: |
                    Unauthorized:
                    - OKA0011: This resource requires authorization
            403:
                description: |
                    Forbidden:
                    - OKA0010: Bad secret
            404:
                description: |
                    Not found:
                    - OKA0013: Report not found
        security:
        - secret: []
        """
        await super().get(id)
