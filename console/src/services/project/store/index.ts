import type { ProjectPageState } from '@/services/project/store/type';

import * as actions from './actions';
import * as getters from './getters';
import * as mutations from './mutations';

const state: ProjectPageState = {
    isInitiated: false,

    searchText: undefined,

    rootNode: null,
    selectedItem: {},
    treeEditMode: false,
    permissionInfo: {},

    hasProjectGroup: undefined,
    projectCount: undefined,

    actionTargetItem: {},
    projectGroupFormVisible: false,
    projectGroupFormUpdateMode: false,
    projectGroupDeleteCheckModalVisible: false,
    projectFormVisible: false,

    shouldUpdateProjectList: false,
};

export default {
    namespaced: true,
    state: () => ({ ...state }),
    getters,
    actions,
    mutations,
};
