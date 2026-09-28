<template>
  <div ref="el" class="chart" :style="{ height, width }"></div>
</template>

<script setup>
import * as echarts from 'echarts'
import { onBeforeUnmount, onMounted, ref, watch } from 'vue'

const props = defineProps({
  option: { type: Object, required: true },
  height: { type: String, default: '320px' },
  width: { type: String, default: '100%' },
})

const el = ref(null)
let chart = null

onMounted(() => {
  chart = echarts.init(el.value)
  chart.setOption(props.option)
  window.addEventListener('resize', onResize)
})

onBeforeUnmount(() => {
  window.removeEventListener('resize', onResize)
  if (chart) chart.dispose()
})

watch(
  () => props.option,
  (opt) => chart && chart.setOption(opt, { notMerge: false }),
  { deep: true },
)

function onResize() {
  chart && chart.resize()
}
</script>

<style scoped>
.chart { min-height: 120px; }
</style>
