import { createApp } from 'vue'
import { createPinia } from 'pinia'
import ElementPlus from 'element-plus'
import 'element-plus/dist/index.css'
import './styles/tech-theme.css'
import * as ElementPlusIconsVue from '@element-plus/icons-vue'
import App from './App.vue'
import router from './router'
import authDirective, { auths, authsAll } from './directives/auth'

const app = createApp(App)
const pinia = createPinia()

// 注册所有图标
for (const [key, component] of Object.entries(ElementPlusIconsVue)) {
  app.component(key, component)
}

// 注册权限指令
app.directive('auth', authDirective)
app.directive('auths', auths)
app.directive('auths-all', authsAll)

app.use(pinia)
app.use(ElementPlus)
app.use(router)
app.mount('#app')
