/**
 * main.ts
 *
 * Bootstraps Vuetify and other plugins then mounts the App`
 */

// Plugins
import { registerPlugins } from '@/plugins'
import 'vuetify/styles';
import vuetify from './plugins/vuetify';
import '@mdi/font/css/materialdesignicons.css';

// Components
import App from './App.vue'
import Overlay from './components/Overlay.vue';

// Composables
import { createApp } from 'vue'
import router from './router'

const app = createApp(App)
registerPlugins(app)

app.use(vuetify);
app.use(router)
app.mount('#app')
app.component('Overlay', Overlay);