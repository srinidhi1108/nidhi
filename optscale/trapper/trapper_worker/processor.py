import requests
from datetime import datetime
from pymongo import MongoClient
from kombu.log import get_logger
from clickhouse_driver import Client as ClickHouseClient
from cloud_adapter.cloud import Cloud as CloudAdapter
from rest_api_client.client_v2 import Client as RestClient

LOG = get_logger(__name__)
CHUNK_SIZE = 10000


class BaseTrafficExpenseProcessor:
    TRAFFIC_IDENTIFIER = {}
    TRAFFIC_FIELDS = []

    def __init__(self, config_cl):
        self.config_cl = config_cl
        self._mongo_cl = None
        self._clickhouse_cl = None
        self._rest_cl = None

    @property
    def rest_cl(self):
        if self._rest_cl is None:
            self._rest_cl = RestClient(
                url=self.config_cl.restapi_url(), verify=False)
            self._rest_cl.secret = self.config_cl.cluster_secret()
        return self._rest_cl

    @property
    def mongo_cl(self):
        if not self._mongo_cl:
            mongo_params = self.config_cl.mongo_params()
            mongo_conn_string = "mongodb://%s:%s@%s:%s" % mongo_params[:-1]
            self._mongo_cl = MongoClient(mongo_conn_string)
        return self._mongo_cl

    @property
    def clickhouse_cl(self):
        if not self._clickhouse_cl:
            user, password, host, db_name = self.config_cl.clickhouse_params()
            self._clickhouse_cl = ClickHouseClient(
                host=host, password=password, database=db_name, user=user)
        return self._clickhouse_cl

    @staticmethod
    def extract_locations_and_usage(e):
        raise NotImplementedError

    @staticmethod
    def get_region_names_map(cloud_adapter):
        res = cloud_adapter.get_regions_coordinates()
        region_names_map = {}
        for k, v in res.items():
            for key in ['name', 'alias']:
                value = v.get(key)
                if not value:
                    continue
                region_names_map[value.lower()] = k
        return region_names_map

    def get_existing_expenses(self, cloud_account_id, tasks):
        date_intervals = []
        for task in tasks:
            date_intervals.append(f"(date >= {task.get('start_date')} "
                                  f"AND date <= {task.get('end_date')})")
        date_intervals = " OR ".join(date_intervals)
        expenses = self.clickhouse_cl.execute(f"""
            SELECT resource_id, date, from, to, SUM(cost*sign), SUM(usage*sign)
            FROM traffic_expenses
            WHERE cloud_account_id = %(ca_id)s AND ({date_intervals})
            GROUP BY resource_id, date, from, to
        """, params={
            'ca_id': cloud_account_id
        })
        return {
            (e[0], e[1], e[2], e[3]): {'cost': e[4], 'usage': e[5]}
            for e in expenses
        }

    def get_cloud_adapter(self, cloud_account_id):
        _, cloud_account = self.rest_cl.cloud_account_get(cloud_account_id)
        config = {
            'type': cloud_account['type'],
            **cloud_account['config'],
        }
        return CloudAdapter.get_adapter(config)

    def process(self, cloud_account_id, tasks):
        try:
            cloud_adapter = self.get_cloud_adapter(cloud_account_id)
        except requests.exceptions.HTTPError as exc:
            if exc.response.status_code == 404:
                return
            raise
        region_names_map = self.get_region_names_map(cloud_adapter)
        res = self.mongo_cl.restapi.raw_expenses.find({
            'cloud_account_id': cloud_account_id,
            '$or': [
                {
                    'start_date': {
                        '$gte': datetime.utcfromtimestamp(t['start_date']),
                        '$lte': datetime.utcfromtimestamp(t['end_date'])
                    }
                }
                for t in tasks
            ],
            **self.TRAFFIC_IDENTIFIER,
            'cost': {'$ne': 0},
        }, self.TRAFFIC_FIELDS + ['resource_id', 'start_date', 'cost'])
        expenses_map = {}
        for r in res:
            resource_id = r.get('resource_id', 'Unknown')
            date = r['start_date'].replace(
                hour=0, minute=0, second=0, microsecond=0)
            _from, _to, _usage = self.extract_locations_and_usage(r)
            from_region = region_names_map.get(_from.lower()) or _from
            to_region = region_names_map.get(_to.lower()) or _to
            key = (resource_id, date, from_region, to_region)
            if key not in expenses_map:
                expenses_map[key] = {
                    'cost': 0,
                    'usage': 0,
                }
            expenses_map[key]['cost'] += r.get('cost', 0)
            expenses_map[key]['usage'] += _usage
        existing_expenses = self.get_existing_expenses(cloud_account_id, tasks)
        collapse_expenses = {}
        for k, v in existing_expenses.items():
            updated_expense = expenses_map.get(k)
            if not updated_expense:
                continue
            if updated_expense == v:
                expenses_map.pop(k, None)
            else:
                collapse_expenses[k] = v

        keys = list(expenses_map.keys())
        for i in range(0, len(keys), CHUNK_SIZE):
            keys_chunk = keys[i:i + CHUNK_SIZE]
            chunk = []
            collapsed_cnt = 0
            for k in keys_chunk:
                new_expense = {
                    'cloud_account_id': cloud_account_id,
                    'resource_id': k[0],
                    'date': k[1],
                    'type': 1,
                    'from': k[2],
                    'to': k[3],
                }
                collapse_expense = collapse_expenses.get(k)
                if collapse_expense:
                    coll_ex = new_expense.copy()
                    coll_ex.update({'sign': -1, **collapse_expense})
                    chunk.append(coll_ex)
                    collapsed_cnt += 1
                new_expense.update({**expenses_map[k], 'sign': 1})
                chunk.append(new_expense)
            if chunk:
                cnt = self.clickhouse_cl.execute(
                    'INSERT INTO traffic_expenses VALUES', chunk)
                LOG.info(
                    f'Inserted %s traffic expenses for '
                    f'cloud_account %s (%s collapsed)' % (
                        cnt, cloud_account_id, collapsed_cnt))


class AwsTrafficExpenseProcessor(BaseTrafficExpenseProcessor):
    TRAFFIC_IDENTIFIER = {
        'product/servicecode': 'AWSDataTransfer',
        'pricing/term': 'OnDemand'
    }
    TRAFFIC_FIELDS = [
        'product/fromRegionCode', 'product/fromLocation',
        'product/toRegionCode', 'product/toLocation', 'lineItem/UsageAmount',
        'product_from_region_code', 'product_to_region_code'
    ]

    @staticmethod
    def extract_locations_and_usage(e):
        _from = e.get(
            'product/fromRegionCode') or e.get(
            'product_from_region_code') or e.get(
            'product/fromLocation') or 'Unknown'
        _to = e.get(
            'product/toRegionCode') or e.get(
            'product_to_region_code') or e.get(
            'product/toLocation') or 'Unknown'
        _usage = float(e.get('lineItem/UsageAmount') or 0)
        return _from, _to, _usage


class AzureTrafficExpenseProcessor(BaseTrafficExpenseProcessor):
    TRAFFIC_IDENTIFIER = {
        'meter_details.meter_category': 'Bandwidth'
    }
    TRAFFIC_FIELDS = [
        'meter_details', 'resource_location', 'usage_quantity'
    ]

    @staticmethod
    def extract_locations_and_usage(e):
        _from = e.get('resource_location') or 'Unknown'
        meter_details = e.get('meter_details', {})
        _to = meter_details.get('meter_location') or meter_details.get(
            'meter_sub_category')
        if not _to or _to == 'Zone 1':
            _to = 'External'
        _usage = float(e.get('usage_quantity') or 0)
        return _from, _to, _usage


class AlibabaTrafficExpenseProcessor(BaseTrafficExpenseProcessor):
    TRAFFIC_IDENTIFIER = {
        'BillingItemCode': 'NetworkOut'
    }
    TRAFFIC_FIELDS = [
        'Usage', 'Zone', 'Region', 'InternetIP'
    ]

    @staticmethod
    def extract_locations_and_usage(e):
        _from = e.get('Zone') or e.get('Region') or 'Unknown'
        _to = 'Unknown' if e.get('InternetIP') else 'External'
        _usage = float(e.get('Usage') or 0)
        return _from, _to, _usage


class ProcessorFactory:
    __modules__ = {
        'aws_cnr': AwsTrafficExpenseProcessor,
        'azure_cnr': AzureTrafficExpenseProcessor,
        'alibaba_cnr': AlibabaTrafficExpenseProcessor,
    }

    @staticmethod
    def get_module(cloud_type, config_cl):
        type_ = cloud_type.lower()
        if type_ not in ProcessorFactory.__modules__:
            return
        return ProcessorFactory.__modules__.get(type_)(config_cl)
