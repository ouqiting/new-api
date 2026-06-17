/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/

// This is a utility file that exports getLobeIcon; the internal component is
// intentionally kept here to keep the icon helper self-contained. Fast refresh
// is less critical for a library helper than for a page component.
/* eslint-disable react-refresh/only-export-components */

import {
  memo,
  useEffect,
  useState,
  type ComponentType,
  type ReactNode,
} from 'react'
/**
 * LobeHub Icon Loader
 *
 * Supports icon names from @lobehub/icons, e.g.:
 * - "OpenAI" (renders the Mono SVG variant)
 * - "Claude" (renders Mono)
 * - "Claude.Color" (renders the Color SVG variant)
 * - "OpenAI.Text" (renders the Text SVG variant)
 *
 * Why dynamic imports with a fixed variant list?
 * @lobehub/icons bundles the icon components as a "compounded" export that also
 * pulls in Avatar/Combine wrappers. Those wrappers depend on @lobehub/ui and
 * antd-style/peer deps that this project does not install, which breaks the
 * production build. Importing only the safe SVG sub-components (Mono/Color/Text)
 * avoids the heavy peer dependencies entirely while keeping tree-shaking for
 * the icons that are actually used.
 */

function renderFallback(iconName: string, size: number): ReactNode {
  return (
    <div
      className='bg-muted text-muted-foreground flex items-center justify-center rounded-full text-xs font-medium'
      style={{ width: size, height: size }}
    >
      {iconName.charAt(0).toUpperCase()}
    </div>
  )
}

const LobeIconAsync = memo(function LobeIconAsync({
  iconName,
  size,
}: {
  iconName: string
  size: number
}) {
  const [Icon, setIcon] = useState<ComponentType<Record<string, unknown>> | null>(
    null
  )
  const [error, setError] = useState(false)

  useEffect(() => {
    let cancelled = false
    const trimmedName = iconName.trim()
    const segments = trimmedName.split('.')
    const baseKey = segments[0]
    const variant = segments[1] || 'Mono'

    const loadIcon = async () => {
      setIcon(null)
      setError(false)
      try {
        let module: { default?: ComponentType<Record<string, unknown>> }
        switch (variant) {
          case 'Color':
            module = await import(
              /* webpackChunkName: "lobe-icons-color" */
              `@lobehub/icons/es/${baseKey}/components/Color`
            )
            break
          case 'Text':
            module = await import(
              /* webpackChunkName: "lobe-icons-text" */
              `@lobehub/icons/es/${baseKey}/components/Text`
            )
            break
          case 'Mono':
          default:
            module = await import(
              /* webpackChunkName: "lobe-icons-mono" */
              `@lobehub/icons/es/${baseKey}/components/Mono`
            )
            break
        }
        const Comp = module?.default
        if (!cancelled) {
          if (Comp && typeof Comp === 'function') {
            setIcon(() => Comp)
          } else {
            setError(true)
          }
        }
      } catch {
        if (!cancelled) setError(true)
      }
    }

    loadIcon()
    return () => {
      cancelled = true
    }
  }, [iconName])

  if (error || !Icon) {
    return renderFallback(iconName, size)
  }

  return <Icon size={size} />
})

/**
 * Get LobeHub icon component by name.
 * Returns a React element that loads the SVG icon asynchronously.
 */
export function getLobeIcon(
  iconName: string | undefined | null,
  size: number = 20
): ReactNode {
  if (!iconName || typeof iconName !== 'string') {
    return renderFallback('?', size)
  }

  const trimmedName = iconName.trim()
  if (!trimmedName) {
    return renderFallback('?', size)
  }

  return <LobeIconAsync iconName={trimmedName} size={size} />
}
