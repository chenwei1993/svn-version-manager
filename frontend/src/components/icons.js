import { h } from 'vue'
import AppIcon from './AppIcon.vue'

function make(name, size = 14) {
  return (props) => h(AppIcon, { name, size, ...props })
}

export const IconPlus = make('plus')
export const IconSearch = make('search')
export const IconFolder = make('folder')
export const IconRefresh = make('refresh')
export const IconEdit = make('edit')
export const IconTrash = make('trash')
export const IconDatabase = make('database')
export const IconChevronRight = make('chevronRight')
export const IconChevronDown = make('chevronDown')
export const IconArrowLeft = make('arrowLeft', 13)
export const IconClock = make('clock')
export const IconFile = make('file')
export const IconDownload = make('download')
export const IconCheck = make('check', 13)
export const IconClose = make('close', 13)
export const IconLink = make('link', 13)
export const IconConnection = make('plug')
export const IconInfo = make('info', 14)
export const IconCopy = make('copy', 14)
export const IconFilter = make('filter', 14)
