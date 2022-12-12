from collections import OrderedDict

from bumi_worker.modules.base import ModuleBase

DEFAULT_DAYS_THRESHOLD = 1


class VolumesNotAttachedForALongTime(ModuleBase):
    def __init__(self, organization_id, config_client, created_at):
        super().__init__(organization_id, config_client, created_at)
        self.option_ordered_map = OrderedDict({
            'days_threshold': {'default': DEFAULT_DAYS_THRESHOLD},
            'excluded_pools': {
                'default': {},
                'clean_func': self.clean_excluded_pools,
            },
            'skip_cloud_accounts': {'default': []}
        })

    def _get(self):
        (days_threshold, excluded_pools,
         skip_cloud_accounts) = self.get_options_values()
        ca_type_map = self.get_cloud_accounts(
            skip_cloud_accounts=skip_cloud_accounts, only_type=True)
        date_field_name = 'last_attached'
        detached_volumes = self.get_resources_stuck_in_state(
            resource_type='volume',
            status_field_name='attached',
            date_field_name=date_field_name,
            resource_stuck_condition=False,
            cloud_account_ids=list(ca_type_map.keys()),
            delta_days=days_threshold,
        )
        result = [
            {
                'cloud_resource_id': volume['cloud_resource_id'],
                'resource_name': volume.get('name'),
                'resource_id': volume['resource_id'],
                'cloud_account_id': volume['cloud_account_id'],
                'cloud_type': ca_type_map.get(volume['cloud_account_id']),
                'cost_in_detached_state': volume['cost_in_resource_state'],
                'saving': volume['savings'],
                'last_seen_in_attached_state': volume['meta'][date_field_name],
                'region': volume['region'],
                'is_excluded': volume.get('pool_id') in excluded_pools,
            } for volume in detached_volumes.values()
            if volume.get('savings', 0) > 0
        ]
        return result


def main(organization_id, config_client, created_at, **kwargs):
    return VolumesNotAttachedForALongTime(
        organization_id, config_client, created_at).get()


def get_module_email_name():
    return 'Not attached Volume'
