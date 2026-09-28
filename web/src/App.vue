<template>
  <el-config-provider :locale="epLocale">
    <el-container class="layout">
      <el-aside width="220px" class="aside">
        <div class="brand">
          <div class="brand-title">{{ t('app.title') }}</div>
          <div class="brand-sub">{{ t('app.sub') }}</div>
        </div>
        <el-menu :default-active="active" router background-color="#0e1726" text-color="#aeb9cc"
                 active-text-color="#ffffff" class="menu">
          <el-menu-item index="/"><el-icon><Odometer /></el-icon>{{ t('menu.dashboard') }}</el-menu-item>
          <el-menu-item index="/acceptance"><el-icon><CircleCheck /></el-icon>{{ t('menu.acceptance') }}</el-menu-item>
          <el-menu-item index="/installers"><el-icon><User /></el-icon>{{ t('menu.installers') }}</el-menu-item>
          <el-menu-item index="/deployment"><el-icon><Box /></el-icon>{{ t('menu.deployment') }}</el-menu-item>
          <el-menu-item index="/stress/http"><el-icon><DataLine /></el-icon>{{ t('menu.stressHttp') }}</el-menu-item>
          <el-menu-item index="/stress/burnin"><el-icon><Monitor /></el-icon>{{ t('menu.stressBurnin') }}</el-menu-item>
          <el-menu-item index="/spec"><el-icon><Document /></el-icon>{{ t('menu.spec') }}</el-menu-item>
        </el-menu>
        <div class="lang-switch">
          <el-radio-group :model-value="locale" size="small" @update:model-value="switchLang">
            <el-radio-button value="zh">中文</el-radio-button>
            <el-radio-button value="en">EN</el-radio-button>
          </el-radio-group>
        </div>
      </el-aside>
      <el-main class="main">
        <router-view />
      </el-main>
    </el-container>
  </el-config-provider>
</template>

<script setup>
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import { ElConfigProvider } from 'element-plus'
import zhCn from 'element-plus/es/locale/lang/zh-cn'
import en from 'element-plus/es/locale/lang/en'
import { locale, setLocale, t } from './i18n'

const route = useRoute()
const active = computed(() => route.path)
const epLocale = computed(() => (locale.value === 'en' ? en : zhCn))

function switchLang(l) {
  setLocale(l)
  document.title = l === 'en'
    ? 'Deployment & Stress-test Platform · Installer Requirements & Device Acceptance'
    : '部署管理压测平台 · 安装人员要求与设备验收'
}
</script>

<style>
* { margin: 0; padding: 0; box-sizing: border-box; }
html, body, #app { height: 100%; }
body { font-family: 'Segoe UI', 'Microsoft YaHei', sans-serif; background: #f0f2f5; }

.layout { height: 100%; }
.aside { background: #0e1726; display: flex; flex-direction: column; }
.brand { padding: 18px 16px 14px; border-bottom: 1px solid rgba(255, 255, 255, 0.08); }
.brand-title { color: #fff; font-size: 16px; font-weight: 600; letter-spacing: 1px; }
.brand-sub { color: #7a8aa5; font-size: 12px; margin-top: 6px; }
.menu { border-right: none; flex: 1; }
.menu .el-menu-item.is-active { background: #1f7cff !important; }
.lang-switch { padding: 12px; border-top: 1px solid rgba(255, 255, 255, 0.08); }
.main { padding: 16px 20px; overflow-y: auto; }

.page-title { font-size: 18px; font-weight: 600; color: #1f2d3d; margin-bottom: 4px; }
.page-desc { color: #8492a6; font-size: 13px; margin-bottom: 16px; }
.card-block { background: #fff; border-radius: 8px; padding: 16px; box-shadow: 0 1px 3px rgba(0, 0, 0, 0.06); margin-bottom: 16px; }
.block-title { font-size: 15px; font-weight: 600; color: #1f2d3d; margin-bottom: 12px; }
</style>
