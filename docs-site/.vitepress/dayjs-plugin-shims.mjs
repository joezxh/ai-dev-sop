import { existsSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { createRequire } from 'node:module'

const require = createRequire(import.meta.url)
const dayjsRoot = dirname(require.resolve('dayjs'))

const SHIM_PREFIX = 'dayjs-plugin-shims:'
const SHIM_BODY = 'export default (option, Dayjs, dayjs) => {}'

export const dayjsPluginShimsPlugin = () => {
  return {
    name: 'dayjs-plugin-shims',
    enforce: 'pre',
    resolveId(source) {
      if (typeof source !== 'string') return null
      if (!source.startsWith('dayjs/plugin/')) return null
      const filePath = join(dayjsRoot, source.replace('dayjs/', ''))
      if (existsSync(filePath)) {
        return filePath
      }
      const sub = source.slice('dayjs/plugin/'.length)
      return SHIM_PREFIX + sub
    },
    load(id) {
      if (typeof id !== 'string') return null
      if (!id.startsWith(SHIM_PREFIX)) return null
      return SHIM_BODY
    },
  }
}