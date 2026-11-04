<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Calendar, CirclePlus, CopyDocument, Key as KeyIcon, Refresh, Search, User } from '@element-plus/icons-vue'
import { keyApi, type Key, type UsagePoint } from '../api/keys'
import UsageChart from '../components/UsageChart.vue'
import AppLayout from '../components/AppLayout.vue'

const keys = ref<Key[]>([])
const loading = ref(false)
const creating = ref(false)
const syncing = ref(false)
const dialog = ref(false)
const selected = ref<Key | null>(null)
const usageHistory = ref<UsagePoint[]>([])
const usageLoading = ref(false)
const drawerWidth = ref(680)
const resizing = ref(false)
const query = ref('')
const filter = ref('all')
const form = reactive({ name: '', days: 30, quota_gb: 0 })
const now = () => Date.now() / 1000
const filtered = computed(() => keys.value.filter(item => {
  const matches = !query.value || item.name.toLowerCase().includes(query.value.trim().toLowerCase()) || item.id.includes(query.value.trim())
  const state = filter.value === 'all' || filter.value === 'active' && item.active || filter.value === 'disabled' && !item.active || filter.value === 'issue' && item.enabled && !item.xray_synced
  return matches && state
}))
const reason = (error: any) => error?.response?.data?.error || error?.message || '操作失败'
const state = (item: Key) => item.expires_at <= now() ? '已过期' : !item.enabled ? '已停用' : !item.xray_synced ? '同步异常' : item.active ? '正常' : '额度用尽'
const stateClass = (item: Key) => item.active && item.xray_synced ? 'good' : item.enabled && !item.xray_synced ? 'issue' : 'muted'
const date = (value: number) => new Date(value * 1000).toLocaleString('zh-CN', { year: 'numeric', month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' })
const size = (value: number) => value >= 1024 ** 3 ? `${(value / 1024 ** 3).toFixed(2)} GB` : `${(value / 1024 ** 2).toFixed(1)} MB`
const usage = (item: Key) => item.quota_bytes ? Math.min(100, item.used_bytes / item.quota_bytes * 100) : 0
async function load() {
  loading.value = true
  try { keys.value = await keyApi.list(); if (selected.value) selected.value = keys.value.find(item => item.id === selected.value?.id) || null }
  catch (error) { ElMessage.error(reason(error)) }
  finally { loading.value = false }
}
async function create() {
  if (!form.name.trim()) return ElMessage.warning('请输入用户名称')
  creating.value = true
  try {
    const row = await keyApi.create({ ...form, name: form.name.trim() })
    dialog.value = false
    Object.assign(form, { name: '', days: 30, quota_gb: 0 })
    await load()
    selected.value = row
    ElMessage.success(row.xray_synced ? '用户已创建' : '已创建，请检查 Xray 同步状态')
  } catch (error) { ElMessage.error(reason(error)) }
  finally { creating.value = false }
}
async function action(row: Key, operation: 'revoke' | 'enable' | 'renew') {
  try {
    let days: number | undefined
    if (operation === 'revoke') await ElMessageBox.confirm(`停用后“${row.name}”将无法建立新连接，确定继续吗？`, '停用用户', { type: 'warning', confirmButtonText: '确认停用' })
    if (operation === 'renew') {
      const result = await ElMessageBox.prompt('增加有效天数', `为“${row.name}”续期`, { inputValue: '30', inputPattern: /^[1-9]\d{0,3}$/, inputErrorMessage: '请输入 1 到 3650 天', confirmButtonText: '确认续期' })
      days = Number(result.value)
    }
    const updated = await keyApi.change(row.id, operation, days)
    await load()
    if (selected.value?.id === row.id) selected.value = updated
    ElMessage.success(operation === 'renew' ? '续期成功' : operation === 'enable' ? '用户已启用' : '用户已停用')
  } catch (error: any) { if (error !== 'cancel' && error !== 'close') ElMessage.error(reason(error)) }
}
async function reconcile() {
  syncing.value = true
  try { await keyApi.reconcile(); await load(); ElMessage.success('同步任务已完成') }
  catch (error) { ElMessage.error(reason(error)) }
  finally { syncing.value = false }
}
async function copy(value?: string) {
  if (!value) return
  try {
    if (!navigator.clipboard?.writeText) throw new Error('clipboard unavailable')
    await navigator.clipboard.writeText(value)
    ElMessage.success('订阅地址已复制')
    return
  } catch {
    // HTTP 页面或浏览器拒绝 Clipboard API 时，使用兼容方式复制。
    const field = document.createElement('textarea')
    field.value = value
    field.setAttribute('readonly', '')
    field.style.position = 'fixed'
    field.style.opacity = '0'
    document.body.appendChild(field)
    field.select()
    const copied = document.execCommand('copy')
    field.remove()
    if (copied) ElMessage.success('订阅地址已复制')
    else ElMessage.error('复制失败，请手动选中地址')
  }
}
watch(() => selected.value?.id, async id => {
  usageHistory.value = []
  if (!id) return
  usageLoading.value = true
  try {
    const points = await keyApi.usage(id)
    if (selected.value?.id === id) usageHistory.value = points
  } catch (error) { ElMessage.error(reason(error)) }
  finally { usageLoading.value = false }
})
function startResize(event: PointerEvent) {
  if (window.innerWidth <= 760) return
  resizing.value = true
  ;(event.currentTarget as HTMLElement).setPointerCapture(event.pointerId)
  event.preventDefault()
}
function moveResize(event: PointerEvent) {
  if (!resizing.value) return
  drawerWidth.value = Math.max(440, Math.min(window.innerWidth - 40, window.innerWidth - event.clientX))
}
function stopResize(event: PointerEvent) {
  resizing.value = false
  const target = event.currentTarget as HTMLElement
  if (target.hasPointerCapture(event.pointerId)) target.releasePointerCapture(event.pointerId)
}
onMounted(load)
</script>

<template>
  <AppLayout>
    <div class="keys-page">
      <div class="page-top"><h1>用户列表</h1><div class="top-actions"><el-button :icon="Refresh" :loading="syncing" @click="reconcile">立即同步</el-button><el-button type="primary" :icon="CirclePlus" @click="dialog = true">创建用户</el-button></div></div>
      <section class="table-panel"><div class="table-toolbar"><div class="toolbar-controls"><el-input v-model="query" :prefix-icon="Search" clearable placeholder="搜索名称或 key" class="search"/><el-select v-model="filter" class="filter"><el-option label="全部状态" value="all"/><el-option label="正常" value="active"/><el-option label="已停用 / 已过期" value="disabled"/><el-option label="同步异常" value="issue"/></el-select></div></div>
        <el-table :data="filtered" v-loading="loading" empty-text="暂无符合条件的用户" class="user-table" row-key="id"><el-table-column label="用户" min-width="175"><template #default="{ row }"><div class="name-cell"><span class="avatar">{{ row.name.slice(0, 1) }}</span><div><strong>{{ row.name }}</strong><small>{{ row.id.slice(0, 8) }}…</small></div></div></template></el-table-column><el-table-column label="状态" min-width="115"><template #default="{ row }"><span class="state" :class="stateClass(row)"><i></i>{{ state(row) }}</span></template></el-table-column><el-table-column label="到期时间" min-width="175"><template #default="{ row }"><span class="date">{{ date(row.expires_at) }}</span></template></el-table-column><el-table-column label="流量使用" min-width="190"><template #default="{ row }"><div class="usage"><span>{{ size(row.used_bytes) }} <small>/ {{ row.quota_bytes ? size(row.quota_bytes) : '不限额' }}</small></span><div v-if="row.quota_bytes" class="usage-track"><span :style="{ width: `${usage(row)}%` }"></span></div></div></template></el-table-column><el-table-column label="操作" min-width="215" align="right"><template #default="{ row }"><div class="row-actions"><el-button link type="primary" @click="selected = row">详情</el-button><el-button link @click="action(row, 'renew')">续期</el-button><el-button v-if="row.enabled" link type="danger" @click="action(row, 'revoke')">停用</el-button><el-button v-else link type="success" @click="action(row, 'enable')">启用</el-button></div></template></el-table-column></el-table>
      </section>
      <el-dialog v-model="dialog" width="500px" class="create-dialog" :show-close="true">
        <template #header>
          <div class="create-heading"><div class="create-heading-icon"><el-icon><User /></el-icon></div><div><h2>创建用户</h2><p>设置访问期限与流量额度</p></div></div>
        </template>
        <el-form label-position="top" class="create-form" @submit.prevent="create">
          <el-form-item label="用户名称" class="create-field">
            <el-input v-model="form.name" size="large" maxlength="80" placeholder="输入用户名称" :prefix-icon="User" @keyup.enter="create" />
          </el-form-item>
          <div class="create-field"><div class="field-label"><el-icon><Calendar /></el-icon><span>有效天数</span></div><div class="duration-options"><button v-for="days in [7, 30, 90, 365]" :key="days" type="button" :class="{ selected: form.days === days }" @click="form.days = days">{{ days }} 天</button></div><el-input-number v-model="form.days" :min="1" :max="3650" controls-position="right" class="wide-number" /><small>也可以输入 1–3650 天</small></div>
          <div class="create-field"><div class="field-label"><el-icon><KeyIcon /></el-icon><span>流量额度</span></div><div class="quota-control"><el-input-number v-model="form.quota_gb" :min="0" :max="100000" controls-position="right" class="wide-number" /><span>GB</span></div><small>0 表示不限流量</small></div>
        </el-form>
        <template #footer><el-button size="large" @click="dialog = false">取消</el-button><el-button type="primary" size="large" :loading="creating" @click="create">创建用户</el-button></template>
      </el-dialog>
      <el-drawer :model-value="!!selected" @update:model-value="(value: boolean) => { if (!value) selected = null }" :title="selected?.name || '用户详情'" :size="`${drawerWidth}px`" class="resizable-detail-drawer"><div class="drawer-resize-handle" role="separator" aria-label="拖动调整详情宽度" aria-orientation="vertical" @pointerdown="startResize" @pointermove="moveResize" @pointerup="stopResize" @pointercancel="stopResize"><span></span></div><template v-if="selected"><div class="drawer-content"><div class="detail-banner"><span class="avatar large">{{ selected.name.slice(0, 1) }}</span><div><strong>{{ selected.name }}</strong><span class="state" :class="stateClass(selected)"><i></i>{{ state(selected) }}</span></div></div><div v-loading="usageLoading"><UsageChart :points="usageHistory" /></div><div class="detail-section"><h3>账户信息</h3><div class="detail-row"><span>到期时间</span><strong>{{ date(selected.expires_at) }}</strong></div><div class="detail-row"><span>已用流量</span><strong>{{ size(selected.used_bytes) }} / {{ selected.quota_bytes ? size(selected.quota_bytes) : '不限额' }}</strong></div><div class="detail-row"><span>Xray 同步</span><strong>{{ selected.xray_synced ? '正常' : selected.sync_error || '等待同步' }}</strong></div></div><div class="detail-section"><h3>订阅地址</h3><p>将此地址交给用户，在客户端添加订阅即可。</p><div class="subscription-box"><el-input :model-value="selected.subscription_url || '请先设置服务域名'" readonly/><el-button type="primary" :icon="CopyDocument" :disabled="!selected.subscription_url" @click="copy(selected.subscription_url)">复制</el-button></div><small>停用、过期或超额后，订阅地址无法再次下载配置。</small></div><div class="detail-section"><h3>连接凭据</h3><div class="key-value">{{ selected.id }}</div><small>订阅配置内包含此 key，请仅分享给对应用户。</small></div><div class="drawer-actions"><el-button @click="action(selected!, 'renew')">续期</el-button><el-button v-if="selected.enabled" type="danger" plain @click="action(selected!, 'revoke')">停用用户</el-button><el-button v-else type="success" plain @click="action(selected!, 'enable')">启用用户</el-button></div></div></template></el-drawer>
    </div>
  </AppLayout>
</template>

<style scoped>
.keys-page{width:100%}.page-top{display:flex;justify-content:space-between;align-items:flex-end;gap:24px;margin-bottom:28px}.top-actions{display:flex;gap:10px}.table-panel{background:white;border:1px solid #e8edf5;border-radius:19px;box-shadow:0 12px 36px rgba(25,55,96,.045);overflow:hidden}.table-toolbar{display:flex;align-items:center;justify-content:flex-end;gap:20px;padding:16px 22px;border-bottom:1px solid #edf1f6}.toolbar-controls{display:flex;gap:10px}.search{width:230px}.filter{width:150px}.user-table{--el-table-header-bg-color:#f8fafd;--el-table-row-hover-bg-color:#f6f9fe}.user-table :deep(th.el-table__cell){height:48px;color:#8393a9;font-size:12px;font-weight:700}.user-table :deep(td.el-table__cell){padding:13px 0}.name-cell{display:flex;align-items:center;gap:11px}.avatar{height:37px;width:37px;flex:0 0 37px;border-radius:11px;display:grid;place-items:center;background:#eaf2ff;color:#4274bf;font-size:15px;font-weight:700}.name-cell>div{display:flex;flex-direction:column;gap:2px}.name-cell strong{font-size:13px;color:#223653}.name-cell small{font-size:11px;color:#a1aec0;font-family:ui-monospace,monospace}.state{display:inline-flex;align-items:center;gap:6px;border-radius:30px;padding:5px 10px;font-size:11px;font-weight:700;white-space:nowrap}.state i{height:6px;width:6px;background:currentColor;border-radius:50%}.state.good{background:#e9f8f1;color:#219b73}.state.issue{background:#fff4e6;color:#c38a2e}.state.muted{background:#f1f3f6;color:#8794a6}.date{font-size:12px;color:#52647c}.usage{font-size:12px;color:#314963}.usage small{color:#a1adbb}.usage-track{height:4px;width:135px;max-width:90%;background:#edf1f7;border-radius:5px;margin-top:7px;overflow:hidden}.usage-track span{display:block;height:100%;background:#4c8ee4;border-radius:5px}.row-actions{display:flex;justify-content:flex-end;gap:2px;padding-right:6px}.dialog-subtitle{font-size:13px;color:#8e9db0;margin:-4px 0 22px}.form-row{display:grid;grid-template-columns:1fr 1fr;gap:16px}.form-row :deep(.el-input-number){width:100%}.form-hint{font-size:12px;color:#a1adbc;margin:0}.drawer-content{padding:6px 4px 20px}.detail-banner{display:flex;align-items:center;gap:13px;background:#f4f8ff;border-radius:15px;padding:18px;margin-bottom:25px}.avatar.large{height:48px;width:48px;font-size:19px}.detail-banner>div{display:flex;flex-direction:column;align-items:flex-start;gap:6px}.detail-banner strong{font-size:16px}.detail-section{padding:0 0 23px;margin-bottom:22px;border-bottom:1px solid #edf1f5}.detail-section h3{font-size:14px;color:#223957;margin:0 0 14px}.detail-section p,.detail-section small{font-size:12px;color:#94a3b7;line-height:1.6}.detail-row{display:flex;align-items:center;justify-content:space-between;gap:20px;margin:13px 0;font-size:12px;color:#8fa0b4}.detail-row strong{font-weight:600;color:#334762;text-align:right;word-break:break-all}.subscription-box{display:flex;gap:8px;margin:14px 0 9px}.subscription-box .el-input{min-width:0}.key-value{padding:10px 12px;background:#f5f7fa;border-radius:8px;color:#566c88;font-size:12px;font-family:ui-monospace,monospace;overflow-wrap:anywhere;margin-bottom:7px}.drawer-actions{display:flex;gap:8px}@media(max-width:760px){.page-top{align-items:flex-start;flex-direction:column}.top-actions{width:100%}.top-actions .el-button{flex:1}.table-toolbar{align-items:stretch;flex-direction:column}.toolbar-controls{width:100%}.search{flex:1;width:auto}.form-row{grid-template-columns:1fr}}@media(max-width:480px){.filter{width:118px}.table-toolbar{padding:18px}.row-actions{flex-wrap:wrap}.subscription-box{flex-direction:column}}
</style>
