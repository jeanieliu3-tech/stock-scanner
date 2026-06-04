<script setup lang="ts">
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { routes } from '@/router'
import {
  LayoutDashboard,
  Search,
  Briefcase,
  Settings,
  Activity,
  Trophy,
} from 'lucide-vue-next'

const route = useRoute()
const router = useRouter()

const iconMap: Record<string, typeof LayoutDashboard> = {
  LayoutDashboard,
  Search,
  Briefcase,
  Settings,
  Trophy,
}

const navItems = routes.map(r => ({
  path: r.path,
  name: r.name as string,
  title: (r.meta?.title as string) || r.name,
  icon: iconMap[r.meta?.icon as string] || LayoutDashboard,
}))

const activeIndex = computed(() => navItems.findIndex(item => item.path === route.path))
</script>

<template>
  <div class="min-h-screen flex flex-col bg-slate-50">
    <!-- Top Header -->
    <header class="sticky top-0 z-50 bg-white border-b border-slate-200 shadow-sm">
      <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div class="flex items-center justify-between h-14">
          <div class="flex items-center gap-3">
            <Activity class="w-6 h-6 text-blue-600" />
            <h1 class="text-lg font-bold text-slate-900">波段趋势共振助手</h1>
          </div>
          <nav class="hidden md:flex items-center gap-1">
            <button
              v-for="item in navItems"
              :key="item.path"
              @click="router.push(item.path)"
              :class="[
                'flex items-center gap-2 px-4 py-2 rounded-lg text-sm font-medium transition-colors',
                route.path === item.path
                  ? 'bg-blue-50 text-blue-700'
                  : 'text-slate-600 hover:bg-slate-100 hover:text-slate-900'
              ]"
            >
              <component :is="item.icon" class="w-4 h-4" />
              {{ item.title }}
            </button>
          </nav>
        </div>
      </div>
    </header>

    <!-- Main Content -->
    <main class="flex-1 max-w-7xl w-full mx-auto px-4 sm:px-6 lg:px-8 py-6">
      <slot />
    </main>

    <!-- Bottom Navigation (mobile) -->
    <nav class="md:hidden fixed bottom-0 left-0 right-0 bg-white border-t border-slate-200 z-50">
      <div class="flex items-center justify-around h-16">
        <button
          v-for="item in navItems"
          :key="item.path"
          @click="router.push(item.path)"
          :class="[
            'flex flex-col items-center justify-center gap-1 flex-1 py-2 transition-colors',
            route.path === item.path ? 'text-blue-600' : 'text-slate-400'
          ]"
        >
          <component :is="item.icon" class="w-5 h-5" />
          <span class="text-xs font-medium">{{ item.title }}</span>
        </button>
      </div>
    </nav>

    <!-- Bottom spacer for mobile nav -->
    <div class="md:hidden h-16" />
  </div>
</template>
