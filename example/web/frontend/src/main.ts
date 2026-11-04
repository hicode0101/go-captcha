import { createApp, h } from 'vue'
import naive from 'naive-ui'
import { NMessageProvider } from 'naive-ui'
import App from './App.vue'
import './highlight'
import './style.css'

// MessageProvider 需要包裹在使用 useMessage 的组件外层
createApp({
  render: () => h(NMessageProvider, null, { default: () => h(App) }),
})
  .use(naive)
  .mount('#app')
