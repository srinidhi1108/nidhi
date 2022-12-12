import json
from rest_api_server.controllers.traffic_processing_task import (
    TrafficProcessingTaskAsyncController)
from rest_api_server.handlers.v1.base_async import (BaseAsyncCollectionHandler,
                                                    BaseAsyncItemHandler)
from rest_api_server.handlers.v1.base import BaseAuthHandler
from rest_api_server.utils import run_task, ModelEncoder


class TrafficProcessingTaskAsyncCollectionHandler(BaseAsyncCollectionHandler,
                                                  BaseAuthHandler):
    def _get_controller_class(self):
        return TrafficProcessingTaskAsyncController

    async def post(self, cloud_account_id, **url_params):
        """
        ---
        description: |
            Create traffic processing task
            Required permission: CLUSTER_SECRET
        tags: [traffic_processing_tasks]
        summary: Create traffic processing task
        parameters:
        -   in: path
            name: cloud_account_id
            description: cloud_account id
            required: true
        -   in: body
            name: body
            description: Task info
            required: true
            schema:
                type: object
                properties:
                    -   name: start_date
                        in: query
                        description: Start date (timestamp in seconds)
                        required: true
                        type: integer
                    -   name: end_date
                        in: query
                        description: End date (timestamp in seconds)
                        required: true
                        type: integer
        responses:
            201:
                description: Created (returns created task object)
                schema:
                    type: object
                    example:
                        id: e1a3bb04-d513-42d2-b9b7-019d24097dec
                        created_at: 1632829593
                        deleted_at: 0
                        cloud_account_id: aea929ab-3103-4932-a77c-ca0a758a992b
                        start_date: 1632812345
                        end_date: 1632824312
            400:
                description: |
                    Wrong arguments:
                    - OE0211: Parameter is immutable
                    - OE0212: Parameter is unexpected
                    - OE0224: Wrong integer value
                    - OE0216: Argument not provided
                    - OE0223: Argument should be integer
                    - OE0233: Incorrect body received
                    - OE0446: "end_date" should be greater than "start_date"
                    - OE0456: Duplicate path parameters in the request body
            401:
                description: |
                    Unauthorized:
                    - OE0237: This resource requires authorization
            403:
                description: |
                    Forbidden:
                    - OE0236: Bad secret
            404:
                description: |
                    Not found:
                    - OE0002: Cloud account not found
            409:
                description: |
                    Conflict:
                    - OE0519: Task with this range already exists
        security:
        - secret: []
        """
        self.check_cluster_secret()
        await super().post(cloud_account_id=cloud_account_id, **url_params)

    async def get(self, cloud_account_id):
        """
        ---
        description: |
            Get list of cloud_account's traffic processing tasks
            Required permission: CLUSTER_SECRET
        tags: [traffic_processing_tasks]
        summary: List of traffic processing tasks for cloud account
        responses:
            200:
                description: Traffic processing tasks list
                schema:
                    type: object
                    properties:
                        traffic_processing_tasks:
                            type: array
                            items:
                                type: object
                            example:
                                -   id: 17cb0d9f-2f42-4f26-beeb-220ef946274c
                                    created_at: 1621428662,
                                    deleted_at: 0,
                                    cloud_account_id:
                                        07d0bb6b-442e-4b20-bc01-df2c6ca24f9d,
                                    start_date: 1621321200,
                                    end_date: 1621324800
                                -   id: 17cb0d9f-2f42-4f26-beeb-220ef946274c
                                    created_at: 1621428662,
                                    deleted_at: 0,
                                    cloud_account_id:
                                        07d0bb6b-442e-4b20-bc01-df2c6ca24f9d,
                                    start_date: 1648339200,
                                    end_date: 1648342800
            401:
                description: |
                    Unauthorized:
                    - OE0237: This resource requires authorization
            403:
                description: |
                    Forbidden:
                    - OE0236: Bad secret
            404:
                description: |
                    Not found:
                    - OE0002: Cloud account not found
        security:
        - secret: []
        """
        self.check_cluster_secret()
        res = await run_task(self.controller.list, cloud_account_id)
        tasks = {'traffic_processing_tasks': [
            task.to_dict() for task in res]}
        self.write(json.dumps(tasks, cls=ModelEncoder))


class TrafficProcessingTaskAsyncItemHandler(BaseAsyncItemHandler, BaseAuthHandler):
    def _get_controller_class(self):
        return TrafficProcessingTaskAsyncController

    async def delete(self, id, **kwargs):
        """
        ---
        description: |
            Deletes an existing traffic processing task
            Required permission: CLUSTER_SECRET
        tags: [traffic_processing_tasks]
        summary: Delete traffic processing task
        parameters:
        -   name: id
            in: path
            description: traffic_processing_tasks id
            required: true
            type: string
        responses:
            204:
                description: Success
            401:
                description: |
                    Unauthorized:
                    - OE0237: This resource requires authorization
            403:
                description: |
                    Forbidden:
                    - OE0236: Bad secret
            404:
                description: |
                    Not found:
                    - OE0002: Traffic processing task not found
        security:
        - secret: []
        """
        self.check_cluster_secret()
        await super().delete(id, **kwargs)
