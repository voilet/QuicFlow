import { defineStore } from 'pinia'
import { api, getToken, setToken, removeToken } from '@/api'

// 从 localStorage 获取用户信息
function getStoredUserInfo() {
  try {
    const userStr = localStorage.getItem('user')
    if (userStr) {
      return JSON.parse(userStr)
    }
  } catch (e) {
    console.error('Failed to parse stored user info:', e)
  }
  return null
}

export const useUserStore = defineStore('user', {
  state: () => ({
    token: getToken(),
    userInfo: getStoredUserInfo(), // 从 localStorage 读取用户信息
    authorities: [], // 用户权限列表
    menus: [] // 用户菜单
  }),

  getters: {
    // 是否已登录（只检查 token）
    isLoggedIn: (state) => !!state.token,

    // 用户显示名称
    displayName: (state) => state.userInfo?.nick_name || state.userInfo?.username || '',

    // 是否是管理员
    isAdmin: (state) => state.userInfo?.authority_id === 1
  },

  actions: {
    // 设置Token
    SET_TOKEN(token) {
      this.token = token
      setToken(token)
    },

    // 设置用户信息
    SET_USER_INFO(userInfo) {
      this.userInfo = userInfo
      // 同时保存到 localStorage
      if (userInfo) {
        localStorage.setItem('user', JSON.stringify(userInfo))
      } else {
        localStorage.removeItem('user')
      }
    },

    // 设置权限列表
    SET_AUTHORITIES(authorities) {
      this.authorities = authorities
    },

    // 设置菜单
    SET_MENUS(menus) {
      this.menus = menus
    },

    // 登录
    async login(loginData) {
      try {
        const res = await api.login(loginData)
        if (res.code === 0 && res.data) {
          this.SET_TOKEN(res.data.token)
          // 存储基本信息
          if (res.data.user) {
            this.SET_USER_INFO(res.data.user)
          }
          return res
        }
        throw new Error(res.msg || '登录失败')
      } catch (error) {
        this.logout()
        throw error
      }
    },

    // 获取用户信息
    async getUserInfo() {
      if (!this.token) {
        throw new Error('未登录')
      }

      try {
        const res = await api.getUserInfo()
        if (res.code === 0 && res.data) {
          const userInfo = res.data.user || res.data

          // 合并用户信息
          this.SET_USER_INFO({
            ...this.userInfo,
            ...userInfo
          })

          // 存储权限和菜单（如果后端返回了）
          if (res.data.menus) {
            this.SET_MENUS(res.data.menus)
          }

          return res.data
        }
        throw new Error(res.msg || '获取用户信息失败')
      } catch (error) {
        // 获取用户信息失败，不自动清除状态，允许继续使用
        console.warn('Failed to get user info, using cached info if available:', error)
        if (!this.userInfo) {
          throw error
        }
        return this.userInfo
      }
    },

    // 登出
    async logout() {
      try {
        if (this.token) {
          await api.logout()
        }
      } catch (error) {
        console.error('Logout error:', error)
      } finally {
        this.resetState()
      }
    },

    // 重置状态
    resetState() {
      this.token = ''
      this.userInfo = null
      this.authorities = []
      this.menus = []
      removeToken()
      localStorage.removeItem('user')
    },

    // 检查是否有权限
    hasPermission(value) {
      if (!value) return true
      if (!this.authorities || this.authorities.length === 0) {
        // 如果没有权限列表，检查是否是管理员
        return this.isAdmin
      }
      return this.authorities.includes(value)
    },

    // 检查是否有任一权限
    hasAnyPermission(values) {
      if (!values || values.length === 0) return true
      if (!this.authorities || this.authorities.length === 0) {
        return this.isAdmin
      }
      return values.some(v => this.authorities.includes(v))
    },

    // 检查是否有所有权限
    hasAllPermissions(values) {
      if (!values || values.length === 0) return true
      if (!this.authorities || this.authorities.length === 0) {
        return this.isAdmin
      }
      return values.every(v => this.authorities.includes(v))
    }
  }
})
