import type { RouteRecordRaw } from 'vue-router'

export const routes: RouteRecordRaw[] = [
  {
    path: '/',
    name: 'home',
    component: () => import('@/views/HomeView.vue'),
    meta: { title: '首页', icon: 'LayoutDashboard' },
  },
  {
    path: '/rank',
    name: 'rank',
    component: () => import('@/views/RankView.vue'),
    meta: { title: '全A排名', icon: 'Trophy' },
  },
  {
    path: '/scan',
    name: 'scan',
    component: () => import('@/views/ScanView.vue'),
    meta: { title: '扫描', icon: 'Search' },
  },
  {
    path: '/position',
    name: 'position',
    component: () => import('@/views/PositionView.vue'),
    meta: { title: '持仓', icon: 'Briefcase' },
  },
  {
    path: '/settings',
    name: 'settings',
    component: () => import('@/views/SettingsView.vue'),
    meta: { title: '设置', icon: 'Settings' },
  },
]
