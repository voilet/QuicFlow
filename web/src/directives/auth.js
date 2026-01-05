import { useUserStore } from '@/stores/user'

/**
 * 权限指令
 * 用法: v-auth="'user:create'"
 * 用法: v-auths="['user:create', 'user:update']" (满足其一即可)
 * 用法: v-auths-all="['user:create', 'user:update']" (全部满足)
 */

// 检查是否有权限
function hasPermission(value, authorities) {
  if (!value) return false
  if (!authorities || authorities.length === 0) return true
  return authorities.includes(value)
}

// 检查是否有任一权限
function hasAnyPermission(values, authorities) {
  if (!values || values.length === 0) return true
  if (!authorities || authorities.length === 0) return true
  return values.some(v => authorities.includes(v))
}

// 检查是否有所有权限
function hasAllPermissions(values, authorities) {
  if (!values || values.length === 0) return true
  if (!authorities || authorities.length === 0) return true
  return values.every(v => authorities.includes(v))
}

export default {
  mounted(el, binding) {
    const { value } = binding
    // 在组件挂载后获取 store
    const checkAuth = () => {
      const userStore = useUserStore()
      const authorities = userStore.authorities
      // 如果没有权限列表，管理员拥有所有权限
      if (value && !userStore.hasPermission(value)) {
        el.parentNode?.removeChild(el)
      }
    }
    // 延迟检查，确保 store 已初始化
    setTimeout(checkAuth, 0)
  }
}

// v-auths 指令（满足其一）
export const auths = {
  mounted(el, binding) {
    const { value } = binding
    const checkAuth = () => {
      const userStore = useUserStore()
      if (value && value.length > 0 && !userStore.hasAnyPermission(value)) {
        el.parentNode?.removeChild(el)
      }
    }
    setTimeout(checkAuth, 0)
  }
}

// v-auths-all 指令（全部满足）
export const authsAll = {
  mounted(el, binding) {
    const { value } = binding
    const checkAuth = () => {
      const userStore = useUserStore()
      if (value && value.length > 0 && !userStore.hasAllPermissions(value)) {
        el.parentNode?.removeChild(el)
      }
    }
    setTimeout(checkAuth, 0)
  }
}

export { hasPermission, hasAnyPermission, hasAllPermissions }
