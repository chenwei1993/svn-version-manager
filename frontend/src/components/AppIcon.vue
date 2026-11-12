<script setup>
import { computed } from 'vue'

const props = defineProps({
  name: { type: String, required: true },
  size: { type: [Number, String], default: 16 },
})

const ICONS = {
  grid: '<rect x="3" y="3" width="7.5" height="7.5" rx="2"/><rect x="13.5" y="3" width="7.5" height="7.5" rx="2"/><rect x="3" y="13.5" width="7.5" height="7.5" rx="2"/><rect x="13.5" y="13.5" width="7.5" height="7.5" rx="2"/>',
  list: '<path d="M8 6h13M8 12h13M8 18h13"/><circle cx="3.6" cy="6" r="1.2" fill="currentColor" stroke="none"/><circle cx="3.6" cy="12" r="1.2" fill="currentColor" stroke="none"/><circle cx="3.6" cy="18" r="1.2" fill="currentColor" stroke="none"/>',
  branch: '<circle cx="6" cy="6" r="2.6"/><circle cx="6" cy="18" r="2.6"/><circle cx="18" cy="8" r="2.6"/><path d="M6 8.6v6.8M18 10.6c0 3.2-3.4 4.6-6.6 5.4"/>',
  download: '<path d="M12 3v12m0 0l-4.2-4.2M12 15l4.2-4.2M5 20.5h14"/>',
  settings: '<circle cx="12" cy="12" r="3.1"/><path d="M19.4 15a1.7 1.7 0 00.34 1.87l.06.06a2 2 0 11-2.83 2.83l-.06-.06a1.7 1.7 0 00-1.87-.34 1.7 1.7 0 00-1 1.55V21a2 2 0 11-4 0v-.09a1.7 1.7 0 00-1.11-1.55 1.7 1.7 0 00-1.87.34l-.06.06a2 2 0 11-2.83-2.83l.06-.06a1.7 1.7 0 00.34-1.87 1.7 1.7 0 00-1.55-1H3a2 2 0 110-4h.09a1.7 1.7 0 001.55-1.11 1.7 1.7 0 00-.34-1.87l-.06-.06a2 2 0 112.83-2.83l.06.06a1.7 1.7 0 001.87.34H9a1.7 1.7 0 001-1.55V3a2 2 0 114 0v.09a1.7 1.7 0 001 1.55 1.7 1.7 0 001.87-.34l.06-.06a2 2 0 112.83 2.83l-.06.06a1.7 1.7 0 00-.34 1.87V9a1.7 1.7 0 001.55 1H21a2 2 0 110 4h-.09a1.7 1.7 0 00-1.55 1z"/>',
  building: '<path d="M4 20V6.5A1.5 1.5 0 015.5 5H10v15M10 10h5.5A1.5 1.5 0 0117 11.5V20M4 20h16M13 14h1M13 17h1M6.5 8.5h1M6.5 12h1M6.5 15.5h1"/>',
  search: '<circle cx="10.8" cy="10.8" r="6.8"/><path d="M19.5 19.5l-3.6-3.6"/>',
  plus: '<path d="M12 5v14M5 12h14"/>',
  refresh: '<path d="M20 11a8 8 0 10-2.34 5.66M20 4.5V11h-6.5"/>',
  folder: '<path d="M3.5 7.5A2.5 2.5 0 016 5h4l2 2.5h6a2.5 2.5 0 012.5 2.5v7A2.5 2.5 0 0116 19.5H6A2.5 2.5 0 013.5 17V7.5z"/>',
  edit: '<path d="M4 20h4l10.5-10.5a2.1 2.1 0 00-3-3L5 17v3z"/>',
  trash: '<path d="M4 7h16M9 7V5a1 1 0 011-1h4a1 1 0 011 1v2M6 7l1 13h10l1-13M10 11v5M14 11v5"/>',
  database: '<ellipse cx="12" cy="5.8" rx="7" ry="2.8"/><path d="M5 5.8v6c0 1.6 3.1 2.8 7 2.8s7-1.2 7-2.8v-6M5 11.8v6c0 1.6 3.1 2.8 7 2.8s7-1.2 7-2.8v-6"/>',
  more: '<circle cx="6" cy="12" r="1.4" fill="currentColor" stroke="none"/><circle cx="12" cy="12" r="1.4" fill="currentColor" stroke="none"/><circle cx="18" cy="12" r="1.4" fill="currentColor" stroke="none"/>',
  chevronDown: '<path d="M6 9l6 6 6-6"/>',
  chevronRight: '<path d="M9 6l6 6-6 6"/>',
  arrowLeft: '<path d="M19 12H5m0 0l6-6m-6 6l6 6"/>',
  arrowRight: '<path d="M5 12h14m0 0l-6-6m6 6l-6 6"/>',
  check: '<path d="M5 12.5l4.2 4.2L19 6.5"/>',
  close: '<path d="M6 6l12 12M18 6L6 18"/>',
  clock: '<circle cx="12" cy="12" r="8.5"/><path d="M12 7.5V12l3 2"/>',
  file: '<path d="M14 3H7a2 2 0 00-2 2v14a2 2 0 002 2h10a2 2 0 002-2V8z"/><path d="M14 3v5h5"/>',
  link: '<path d="M10 13a5 5 0 007 0l2-2a5 5 0 00-7-7l-1 1M14 11a5 5 0 00-7 0l-2 2a5 5 0 007 7l1-1"/>',
  info: '<circle cx="12" cy="12" r="8.5"/><path d="M12 11v5.2M12 7.8h.01"/>',
  copy: '<rect x="8" y="8" width="11" height="12" rx="2"/><path d="M16 8V6a2 2 0 00-2-2H6a2 2 0 00-2 2v10a2 2 0 002 2h2"/>',
  filter: '<path d="M4 5h16M7 12h10M10 19h4"/>',
  sliders: '<path d="M4 7h9M17 7h3M4 17h3M11 17h9"/><circle cx="15" cy="7" r="2"/><circle cx="9" cy="17" r="2"/>',
  plug: '<path d="M9 3v6M15 3v6M6 9h12v3a6 6 0 01-12 0V9zM12 18v3"/>',
}

const inner = computed(() => ICONS[props.name] || ICONS.info)
</script>

<template>
  <svg
    class="app-icon"
    :width="size"
    :height="size"
    viewBox="0 0 24 24"
    fill="none"
    xmlns="http://www.w3.org/2000/svg"
    stroke="currentColor"
    stroke-width="1.7"
    stroke-linecap="round"
    stroke-linejoin="round"
    aria-hidden="true"
  >
    <g v-html="inner" />
  </svg>
</template>
