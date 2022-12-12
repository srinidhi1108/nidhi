import { IRouter } from '../index';
import { lazy } from 'react';
import { ChartIcon } from 'tdesign-icons-react';
import { useTranslation } from 'react-i18next';

export const useCostRouteConfig = (): IRouter[] => {
  const { t } = useTranslation();
  return [
    {
      path: '/cost',
      meta: {
        title: t('成本洞察'),
        Icon: ChartIcon,
      },
      children: [
        {
          path: 'insight',
          Component: lazy(() => import('pages/Cost/insight/InsightPanel')),
          meta: {
            title: t('成本分布'),
          },
        },
        {
          path: 'carbon',
          Component: lazy(() => import('pages/Cost/CarbonInsight/Index')),
          meta: {
            title: t('碳排放分析'),
          },
        },
      ],
    },
  ];
};
