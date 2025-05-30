'use client'

import { useTheme } from 'next-themes'
import { LineShadowText } from './line-shadow-text'

export default function ShadowText() {
  const theme = useTheme()
  const shadowColor = theme.resolvedTheme === "dark" ? "white" : "black"
  return (
    <h1 className="text-balance text-5xl font-semibold leading-none tracking-tighter sm:text-6xl md:text-7xl lg:text-8xl">
      Ship
      <LineShadowText className="italic" shadowColor={shadowColor}>
        Fast
      </LineShadowText>
    </h1>
  )
}