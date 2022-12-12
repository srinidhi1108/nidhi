import { MENU_ID } from '@/lib/menu/config';

export const ASSET_INVENTORY_ROUTE = Object.freeze({
    _NAME: MENU_ID.ASSET_INVENTORY,
    SERVER: { _NAME: MENU_ID.ASSET_INVENTORY_SERVER },
    CLOUD_SERVICE: {
        _NAME: MENU_ID.ASSET_INVENTORY_CLOUD_SERVICE,
        SEARCH: { _NAME: `${MENU_ID.ASSET_INVENTORY_CLOUD_SERVICE}.search` },
        TYPE_SEARCH: { _NAME: `${MENU_ID.ASSET_INVENTORY_CLOUD_SERVICE}.type_search` },
        NO_RESOURCE: { _NAME: `${MENU_ID.ASSET_INVENTORY_CLOUD_SERVICE}.no_resource` },
        DETAIL: { _NAME: `${MENU_ID.ASSET_INVENTORY_CLOUD_SERVICE}.detail` },
    },
    COLLECTOR: {
        _NAME: MENU_ID.ASSET_INVENTORY_COLLECTOR,
        CREATE: {
            _NAME: `${MENU_ID.ASSET_INVENTORY_COLLECTOR}.create`,
            STEPS: { _NAME: `${MENU_ID.ASSET_INVENTORY_COLLECTOR}.create.steps` },
        },
        HISTORY: {
            _NAME: `${MENU_ID.ASSET_INVENTORY_COLLECTOR}.history`,
            JOB: { _NAME: `${MENU_ID.ASSET_INVENTORY_COLLECTOR}.history.job` },
        },
    },
    SERVICE_ACCOUNT: {
        _NAME: MENU_ID.ASSET_INVENTORY_SERVICE_ACCOUNT,
        DETAIL: { _NAME: `${MENU_ID.ASSET_INVENTORY_SERVICE_ACCOUNT}.detail` },
        SEARCH: { _NAME: `${MENU_ID.ASSET_INVENTORY_SERVICE_ACCOUNT}.search` },
        ADD: { _NAME: `${MENU_ID.ASSET_INVENTORY_SERVICE_ACCOUNT}.add` },
        NO_RESOURCE: { _NAME: `${MENU_ID.ASSET_INVENTORY_SERVICE_ACCOUNT}.no_resource` },
    },
});
