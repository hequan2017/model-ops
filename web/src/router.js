import { createRouter, createWebHistory } from 'vue-router'

const routes = [
  { path: '/', name: 'dashboard', meta: { title: '总览' }, component: () => import('./views/Dashboard.vue') },
  { path: '/acceptance', name: 'acceptance', meta: { title: '设备验收' }, component: () => import('./views/Acceptance.vue') },
  { path: '/installers', name: 'installers', meta: { title: '安装人员' }, component: () => import('./views/Installers.vue') },
  { path: '/deployment', name: 'deployment', meta: { title: '部署管理' }, component: () => import('./views/Deployment.vue') },
  { path: '/stress/http', name: 'stress-http', meta: { title: '并发压测' }, component: () => import('./views/StressHttp.vue') },
  { path: '/stress/burnin', name: 'stress-burnin', meta: { title: '满载烤机' }, component: () => import('./views/StressBurnin.vue') },
  { path: '/spec', name: 'spec', meta: { title: '验收规范' }, component: () => import('./views/Spec.vue') },
]

export const router = createRouter({ history: createWebHistory(), routes })
